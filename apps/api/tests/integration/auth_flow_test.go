//go:build integration

package integration_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"catch/apps/api/internal/app/bootstrap"
	"catch/apps/api/internal/app/composition"
	"catch/apps/api/internal/app/config"
	"catch/apps/api/internal/platform/db"
	"catch/apps/api/internal/platform/logger"
	"catch/apps/api/internal/platform/mail"
	"catch/apps/api/internal/platform/outbox"
	"catch/apps/api/internal/platform/search"
)

func TestDevLoginCurrentUserAndLogout(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	cfg := testConfig(databaseURL)
	log := logger.New(config.EnvTest)

	container, err := composition.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	defer container.Close()

	if err := db.ApplyMigrations(ctx, container.DB, migrationsDir(t)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	cleanupIdentityTables(t, container)
	defer cleanupIdentityTables(t, container)

	server := httptest.NewServer(bootstrap.NewRouter(container))
	defer server.Close()

	client := server.Client()
	devLoginResponse := doJSON(t, client, server.URL+"/api/v1/dev/auth/login", http.MethodPost, `{"email":"integration-dev@catch.local"}`, http.StatusOK, nil)
	defer devLoginResponse.Body.Close()
	sessionCookie := findCookie(t, devLoginResponse.Cookies(), cfg.Auth.SessionCookieName)
	csrfCookie := findCookie(t, devLoginResponse.Cookies(), cfg.Auth.CSRFCookieName)

	meRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/me", nil)
	if err != nil {
		t.Fatalf("create me request: %v", err)
	}
	meRequest.AddCookie(sessionCookie)

	meResponse, err := client.Do(meRequest)
	if err != nil {
		t.Fatalf("call /auth/me: %v", err)
	}
	defer meResponse.Body.Close()
	if meResponse.StatusCode != http.StatusOK {
		t.Fatalf("/auth/me status = %d, want %d", meResponse.StatusCode, http.StatusOK)
	}

	var currentUser struct {
		User struct {
			Email string `json:"email"`
		} `json:"user"`
		Capabilities struct {
			CanCreateArticle bool `json:"can_create_article"`
		} `json:"capabilities"`
	}
	if err := json.NewDecoder(meResponse.Body).Decode(&currentUser); err != nil {
		t.Fatalf("decode /auth/me response: %v", err)
	}
	if currentUser.User.Email != "integration-dev@catch.local" {
		t.Fatalf("email = %q, want %q", currentUser.User.Email, "integration-dev@catch.local")
	}
	if !currentUser.Capabilities.CanCreateArticle {
		t.Fatal("new dev user must be able to create articles at rating 0")
	}

	logoutRequest, err := http.NewRequest(http.MethodPost, server.URL+"/api/v1/auth/logout", nil)
	if err != nil {
		t.Fatalf("create logout request: %v", err)
	}
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.Header.Set(cfg.Auth.CSRFHeaderName, csrfCookie.Value)

	logoutResponse, err := client.Do(logoutRequest)
	if err != nil {
		t.Fatalf("call /auth/logout: %v", err)
	}
	defer logoutResponse.Body.Close()
	if logoutResponse.StatusCode != http.StatusNoContent {
		t.Fatalf("/auth/logout status = %d, want %d", logoutResponse.StatusCode, http.StatusNoContent)
	}
}

func TestNotificationStreamSendsUnreadCounter(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	cfg := testConfig(databaseURL)
	log := logger.New(config.EnvTest)

	container, err := composition.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	defer container.Close()

	if err := db.ApplyMigrations(ctx, container.DB, migrationsDir(t)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	cleanupIdentityTables(t, container)
	defer cleanupIdentityTables(t, container)

	server := httptest.NewServer(bootstrap.NewRouter(container))
	defer server.Close()

	client := server.Client()
	devLoginResponse := doJSON(t, client, server.URL+"/api/v1/dev/auth/login", http.MethodPost, `{"email":"stream-dev@catch.local"}`, http.StatusOK, nil)
	defer devLoginResponse.Body.Close()
	sessionCookie := findCookie(t, devLoginResponse.Cookies(), cfg.Auth.SessionCookieName)

	streamCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(streamCtx, http.MethodGet, server.URL+"/api/v1/notifications/stream", nil)
	if err != nil {
		t.Fatalf("create stream request: %v", err)
	}
	request.AddCookie(sessionCookie)

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("call notification stream: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("notification stream status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := response.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("content-type = %q, want text/event-stream", got)
	}

	reader := bufio.NewReader(response.Body)
	eventLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read stream event: %v", err)
	}
	dataLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read stream data: %v", err)
	}
	if strings.TrimSpace(eventLine) != "event: unread-count" {
		t.Fatalf("event line = %q, want unread-count event", eventLine)
	}
	if !strings.Contains(dataLine, `"unread_total":0`) {
		t.Fatalf("data line = %q, want unread_total 0", dataLine)
	}
}

func TestEmailCodeRequestIsDeliveredThroughOutbox(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	cfg := testConfig(databaseURL)
	log := logger.New(config.EnvTest)

	container, err := composition.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	defer container.Close()

	if err := db.ApplyMigrations(ctx, container.DB, migrationsDir(t)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	cleanupIdentityTables(t, container)
	defer cleanupIdentityTables(t, container)

	server := httptest.NewServer(bootstrap.NewRouter(container))
	defer server.Close()

	email := "email-code@catch.local"
	response := doJSON(t, server.Client(), server.URL+"/api/v1/auth/email/request-code", http.MethodPost, `{"email":"`+email+`"}`, http.StatusAccepted, nil)
	defer response.Body.Close()
	var payload struct {
		DevCode string `json:"dev_code"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode email code response: %v", err)
	}
	if payload.DevCode == "" {
		t.Fatal("dev code is empty")
	}

	sender := &recordingMailSender{}
	worker := outbox.NewWorker(container.DB, outbox.NewNotificationHandler(container.DB, search.NoopArticleIndexer{}, sender, log), log, "auth-flow-test")
	if _, err := worker.ProcessOnce(context.Background()); err != nil {
		t.Fatalf("process outbox: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("sent messages = %+v, want one email", sender.messages)
	}
	if sender.messages[0].To != email || !strings.Contains(sender.messages[0].Text, payload.DevCode) {
		t.Fatalf("sent message = %+v, want email code delivery", sender.messages[0])
	}
}

func TestOAuthStartAndCallbackCreateSession(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse token form: %v", err)
			}
			if r.Form.Get("code") != "oauth-code" || r.Form.Get("code_verifier") == "" {
				t.Fatalf("token form = %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "provider-token"})
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer provider-token" {
				t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"sub":   "google-user-1",
				"email": "oauth-user@catch.local",
				"name":  "OAuth User",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer provider.Close()

	ctx := context.Background()
	cfg := testConfig(databaseURL)
	cfg.Auth.OAuth = config.OAuthConfig{
		StateCookieName: "catch_oauth_state",
		StateTTL:        time.Minute,
		Providers: map[string]config.OAuthProviderConfig{
			"google": {
				ClientID:     "google-client-id",
				ClientSecret: "google-client-secret",
				AuthURL:      provider.URL + "/authorize",
				TokenURL:     provider.URL + "/token",
				UserInfoURL:  provider.URL + "/userinfo",
				RedirectURL:  "http://api.test/api/v1/auth/oauth/google/callback",
				Scopes:       []string{"openid", "email", "profile"},
			},
		},
	}
	log := logger.New(config.EnvTest)

	container, err := composition.New(ctx, cfg, log)
	if err != nil {
		t.Fatalf("create container: %v", err)
	}
	defer container.Close()

	if err := db.ApplyMigrations(ctx, container.DB, migrationsDir(t)); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	cleanupIdentityTables(t, container)
	defer cleanupIdentityTables(t, container)

	server := httptest.NewServer(bootstrap.NewRouter(container))
	defer server.Close()

	client := noRedirectClient(server.Client())
	startResponse, err := client.Get(server.URL + "/api/v1/auth/oauth/google/start?return_to=/profile")
	if err != nil {
		t.Fatalf("start OAuth: %v", err)
	}
	defer startResponse.Body.Close()
	if startResponse.StatusCode != http.StatusFound {
		t.Fatalf("start status = %d, want %d", startResponse.StatusCode, http.StatusFound)
	}
	stateCookie := findCookie(t, startResponse.Cookies(), cfg.Auth.OAuth.StateCookieName)
	location, err := url.Parse(startResponse.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse OAuth location: %v", err)
	}
	if location.Query().Get("client_id") != "google-client-id" || location.Query().Get("code_challenge_method") != "S256" {
		t.Fatalf("OAuth redirect location = %s", location.String())
	}
	state := location.Query().Get("state")
	if state == "" {
		t.Fatal("OAuth state is empty")
	}

	callbackRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/oauth/google/callback?code=oauth-code&state="+url.QueryEscape(state), nil)
	if err != nil {
		t.Fatalf("create callback request: %v", err)
	}
	callbackRequest.AddCookie(stateCookie)
	callbackResponse, err := client.Do(callbackRequest)
	if err != nil {
		t.Fatalf("OAuth callback: %v", err)
	}
	defer callbackResponse.Body.Close()
	if callbackResponse.StatusCode != http.StatusFound {
		t.Fatalf("callback status = %d, want %d", callbackResponse.StatusCode, http.StatusFound)
	}
	if callbackResponse.Header.Get("Location") != "/profile" {
		t.Fatalf("callback location = %q, want /profile", callbackResponse.Header.Get("Location"))
	}
	sessionCookie := findCookie(t, callbackResponse.Cookies(), cfg.Auth.SessionCookieName)

	meRequest, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/auth/me", nil)
	if err != nil {
		t.Fatalf("create me request: %v", err)
	}
	meRequest.AddCookie(sessionCookie)
	meResponse, err := server.Client().Do(meRequest)
	if err != nil {
		t.Fatalf("call /auth/me: %v", err)
	}
	defer meResponse.Body.Close()
	if meResponse.StatusCode != http.StatusOK {
		t.Fatalf("/auth/me status = %d, want %d", meResponse.StatusCode, http.StatusOK)
	}
}

type recordingMailSender struct {
	messages []mail.Message
}

func (s *recordingMailSender) Send(_ context.Context, message mail.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

func testConfig(databaseURL string) config.Config {
	return config.Config{
		AppName: "catch-api-test",
		Env:     config.EnvTest,
		HTTP: config.HTTPConfig{
			Addr:            ":0",
			ReadTimeout:     time.Second,
			WriteTimeout:    time.Second,
			IdleTimeout:     time.Second,
			ShutdownTimeout: time.Second,
		},
		Database: config.DatabaseConfig{
			URL:           databaseURL,
			MinConns:      0,
			MaxConns:      4,
			MigrationsDir: migrationsDirMust(),
		},
		Auth: config.AuthConfig{
			SessionCookieName:      "catch_session",
			CSRFCookieName:         "catch_csrf",
			CSRFHeaderName:         "X-CSRF-Token",
			Secret:                 "integration-secret",
			SessionTTL:             time.Hour,
			EmailCodeTTL:           10 * time.Minute,
			EmailCodeLength:        6,
			DevLoginEnabled:        true,
			DevLoginEmail:          "dev@catch.local",
			DevEmailCodeInResponse: true,
		},
		Storage: config.StorageConfig{
			Provider:       "local",
			LocalPath:      filepath.Join(os.TempDir(), "catch-api-integration-storage"),
			PublicBaseURL:  "/api/v1/media/files",
			MaxUploadBytes: 10 * 1024 * 1024,
		},
	}
}

func doJSON(t *testing.T, client *http.Client, url, method, body string, wantStatus int, cookies []*http.Cookie) *http.Response {
	t.Helper()

	request, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("call %s: %v", url, err)
	}
	if response.StatusCode != wantStatus {
		defer response.Body.Close()
		t.Fatalf("%s status = %d, want %d", url, response.StatusCode, wantStatus)
	}
	return response
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()

	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	t.Fatalf("cookie %s not found", name)
	return nil
}

func noRedirectClient(base *http.Client) *http.Client {
	client := *base
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &client
}

func cleanupIdentityTables(t *testing.T, container *composition.Container) {
	t.Helper()

	_, err := container.DB.Exec(context.Background(), `
		truncate table audit_log, outbox_events, users restart identity cascade
	`)
	if err != nil {
		t.Fatalf("cleanup identity tables: %v", err)
	}
}

func migrationsDir(t *testing.T) string {
	t.Helper()
	return migrationsDirMust()
}

func migrationsDirMust() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "../../migrations"
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "migrations"))
}
