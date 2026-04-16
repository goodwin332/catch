package app

import (
	"context"
	"crypto/hmac"
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"catch/apps/api/internal/app/config"
	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/identity/app/dto"
	"catch/apps/api/internal/modules/identity/domain"
	"catch/apps/api/internal/modules/identity/ports"
	platformauth "catch/apps/api/internal/platform/auth"
	"catch/apps/api/internal/platform/db"
	"catch/apps/api/internal/platform/events"
	httpx "catch/apps/api/internal/platform/http"
	platformoauth "catch/apps/api/internal/platform/oauth"
)

type Service struct {
	cfg      config.AuthConfig
	tx       *db.TxManager
	users    ports.UserRepository
	sessions ports.SessionRepository
	codes    ports.EmailCodeRepository
	auth     *platformauth.Manager
	oauth    platformoauth.Client
	now      func() time.Time
}

type AuthResult struct {
	Response  dto.CurrentUserResponse
	Tokens    platformauth.SessionTokens
	ExpiresAt time.Time
}

func NewService(
	cfg config.AuthConfig,
	tx *db.TxManager,
	users ports.UserRepository,
	sessions ports.SessionRepository,
	codes ports.EmailCodeRepository,
	authManager *platformauth.Manager,
	oauthClient platformoauth.Client,
) *Service {
	if oauthClient == nil {
		oauthClient = platformoauth.NewHTTPClient(5 * time.Second)
	}
	return &Service{
		cfg:      cfg,
		tx:       tx,
		users:    users,
		sessions: sessions,
		codes:    codes,
		auth:     authManager,
		oauth:    oauthClient,
		now:      time.Now,
	}
}

type OAuthStartResult struct {
	AuthorizationURL string
	StateCookie      platformauth.OAuthStateCookie
}

func (s *Service) StartOAuth(provider, returnTo string) (OAuthStartResult, error) {
	providerCfg, err := s.oauthProviderConfig(provider)
	if err != nil {
		return OAuthStartResult{}, err
	}
	if !providerCfg.Enabled() {
		return OAuthStartResult{}, httpx.ServiceUnavailable("OAuth-провайдер не настроен", nil)
	}

	state, err := s.auth.NewOAuthState(provider, sanitizeReturnTo(returnTo), s.cfg.OAuth.StateTTL, s.now())
	if err != nil {
		return OAuthStartResult{}, err
	}
	cookieValue, err := s.auth.SignOAuthState(state)
	if err != nil {
		return OAuthStartResult{}, err
	}

	authURL, err := buildOAuthAuthorizationURL(providerCfg, state)
	if err != nil {
		return OAuthStartResult{}, err
	}

	return OAuthStartResult{
		AuthorizationURL: authURL,
		StateCookie: platformauth.OAuthStateCookie{
			Name:      s.cfg.OAuth.StateCookieName,
			Value:     cookieValue,
			ExpiresAt: state.ExpiresAt,
		},
	}, nil
}

func (s *Service) CompleteOAuth(ctx context.Context, provider, code, stateToken, stateCookie, userAgent, requestIP string) (AuthResult, string, error) {
	providerCfg, err := s.oauthProviderConfig(provider)
	if err != nil {
		return AuthResult{}, "", err
	}
	if !providerCfg.Enabled() {
		return AuthResult{}, "", httpx.ServiceUnavailable("OAuth-провайдер не настроен", nil)
	}
	if strings.TrimSpace(code) == "" {
		return AuthResult{}, "", httpx.ValidationError("OAuth code обязателен", map[string]any{"code": "required"})
	}
	if strings.TrimSpace(stateToken) == "" {
		return AuthResult{}, "", httpx.ValidationError("OAuth state обязателен", map[string]any{"state": "required"})
	}
	if stateCookie == "" {
		return AuthResult{}, "", httpx.Forbidden("OAuth state cookie обязателен")
	}

	state, err := s.auth.VerifyOAuthState(stateCookie, provider, stateToken, s.now())
	if err != nil {
		return AuthResult{}, "", httpx.Forbidden("OAuth state недействителен")
	}

	profile, err := s.oauth.Exchange(ctx, provider, providerCfg, code, state.CodeVerifier)
	if err != nil {
		return AuthResult{}, "", httpx.ServiceUnavailable("OAuth-провайдер временно недоступен", err)
	}
	email, err := domain.NewEmail(profile.Email)
	if err != nil {
		return AuthResult{}, "", httpx.ServiceUnavailable("OAuth-провайдер вернул некорректный email", err)
	}

	var result AuthResult
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		user, err := s.users.FindByOAuthAccount(ctx, provider, profile.ProviderAccountID)
		if errors.Is(err, domain.ErrNotFound) {
			user, err = s.users.FindByEmail(ctx, email)
			if errors.Is(err, domain.ErrNotFound) {
				user, err = s.users.CreateEmailUser(ctx, ports.CreateEmailUserInput{Email: email, DisplayName: profile.DisplayName})
			}
			if err != nil {
				return err
			}
			if err := s.users.LinkOAuthAccount(ctx, ports.LinkOAuthAccountInput{
				UserID:            user.ID,
				Provider:          provider,
				ProviderAccountID: profile.ProviderAccountID,
				Email:             email,
			}); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		if err := ensureActiveUser(user); err != nil {
			return err
		}

		authResult, err := s.createSession(ctx, user, userAgent, requestIP)
		if err != nil {
			return err
		}
		result = authResult
		return nil
	})
	if err != nil {
		return AuthResult{}, "", err
	}

	return result, state.ReturnTo, nil
}

func (s *Service) OAuthStateCookieName() string {
	return s.cfg.OAuth.StateCookieName
}

func (s *Service) RequestEmailCode(ctx context.Context, request dto.RequestEmailCodeRequest, requestIP string) (dto.RequestEmailCodeResponse, error) {
	email, err := domain.NewEmail(request.Email)
	if err != nil {
		return dto.RequestEmailCodeResponse{}, httpx.ValidationError("Email указан некорректно", map[string]any{"email": "invalid"})
	}

	code, err := s.auth.NewEmailCode(s.cfg.EmailCodeLength)
	if err != nil {
		return dto.RequestEmailCodeResponse{}, err
	}
	codeHash := s.auth.HashEmailCode(email.String(), code)
	expiresAt := s.now().Add(s.cfg.EmailCodeTTL)

	var response dto.RequestEmailCodeResponse
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		purpose := ports.EmailCodePurposeRegistration
		if _, err := s.users.FindByEmail(ctx, email); err == nil {
			purpose = ports.EmailCodePurposeLogin
		} else if !errors.Is(err, domain.ErrNotFound) {
			return err
		}

		if err := s.codes.Create(ctx, ports.CreateEmailCodeInput{
			Email:     email,
			CodeHash:  codeHash,
			Purpose:   purpose,
			RequestIP: requestIP,
			ExpiresAt: expiresAt,
		}); err != nil {
			return err
		}
		if err := events.AddOutbox(ctx, s.tx.Querier(ctx), "email_login_code", email.String(), "auth.email_code.requested", map[string]any{
			"email":      email.String(),
			"code":       code,
			"purpose":    string(purpose),
			"expires_at": expiresAt.Format(time.RFC3339),
		}); err != nil {
			return err
		}

		response = dto.RequestEmailCodeResponse{Status: "accepted"}
		if s.cfg.DevEmailCodeInResponse {
			response.DevCode = code
		}
		return nil
	})
	if err != nil {
		return dto.RequestEmailCodeResponse{}, err
	}

	return response, nil
}

func (s *Service) VerifyEmailCode(ctx context.Context, request dto.VerifyEmailCodeRequest, userAgent, requestIP string) (AuthResult, error) {
	email, err := domain.NewEmail(request.Email)
	if err != nil {
		return AuthResult{}, httpx.ValidationError("Email указан некорректно", map[string]any{"email": "invalid"})
	}

	code := strings.TrimSpace(request.Code)
	if code == "" {
		return AuthResult{}, httpx.ValidationError("Код подтверждения обязателен", map[string]any{"code": "required"})
	}

	var result AuthResult
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		now := s.now()
		codeHash := s.auth.HashEmailCode(email.String(), code)
		if _, err := s.codes.Consume(ctx, ports.ConsumeEmailCodeInput{Email: email, CodeHash: codeHash, Now: now}); err != nil {
			if errors.Is(err, domain.ErrInvalidCode) {
				_ = s.codes.IncrementAttempts(ctx, email, now)
				return httpx.Unauthorized("Код подтверждения недействителен или истёк")
			}
			return err
		}

		user, err := s.users.FindByEmail(ctx, email)
		if errors.Is(err, domain.ErrNotFound) {
			user, err = s.users.CreateEmailUser(ctx, ports.CreateEmailUserInput{Email: email})
		}
		if err != nil {
			return err
		}
		if err := ensureActiveUser(user); err != nil {
			return err
		}

		authResult, err := s.createSession(ctx, user, userAgent, requestIP)
		if err != nil {
			return err
		}
		result = authResult
		return nil
	})
	if err != nil {
		return AuthResult{}, err
	}

	return result, nil
}

func (s *Service) CurrentUser(ctx context.Context, sessionToken string) (dto.CurrentUserResponse, error) {
	sessionUser, err := s.Authenticate(ctx, sessionToken)
	if err != nil {
		return dto.CurrentUserResponse{}, err
	}

	return mapCurrentUser(sessionUser.User), nil
}

func (s *Service) Authenticate(ctx context.Context, sessionToken string) (domain.SessionUser, error) {
	if sessionToken == "" {
		return domain.SessionUser{}, httpx.Unauthorized("Требуется авторизация")
	}

	sessionUser, err := s.sessions.FindUserByTokenHash(ctx, s.auth.HashSessionToken(sessionToken), s.now())
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return domain.SessionUser{}, httpx.Unauthorized("Сессия недействительна")
		}
		return domain.SessionUser{}, err
	}

	return sessionUser, nil
}

func (s *Service) Logout(ctx context.Context, sessionToken string) error {
	if sessionToken == "" {
		return nil
	}
	return s.sessions.RevokeByTokenHash(ctx, s.auth.HashSessionToken(sessionToken), s.now())
}

func (s *Service) ValidateCSRF(ctx context.Context, sessionToken, csrfToken string) error {
	if sessionToken == "" || csrfToken == "" {
		return httpx.Forbidden("CSRF token обязателен")
	}

	sessionUser, err := s.sessions.FindUserByTokenHash(ctx, s.auth.HashSessionToken(sessionToken), s.now())
	if err != nil {
		if errors.Is(err, domain.ErrSessionNotFound) {
			return httpx.Unauthorized("Сессия недействительна")
		}
		return err
	}

	if !hmac.Equal(sessionUser.CSRFTokenHash, s.auth.HashCSRFToken(csrfToken)) {
		return httpx.Forbidden("CSRF token недействителен")
	}

	return nil
}

func (s *Service) DevLogin(ctx context.Context, request dto.DevLoginRequest, userAgent, requestIP string) (AuthResult, error) {
	if !s.cfg.DevLoginEnabled {
		return AuthResult{}, httpx.NewError(404, httpx.CodeNotFound, "Dev login недоступен")
	}

	emailValue := strings.TrimSpace(request.Email)
	if emailValue == "" {
		emailValue = s.cfg.DevLoginEmail
	}
	email, err := domain.NewEmail(emailValue)
	if err != nil {
		return AuthResult{}, httpx.ValidationError("Email указан некорректно", map[string]any{"email": "invalid"})
	}

	var result AuthResult
	err = s.tx.WithinTx(ctx, func(ctx context.Context) error {
		user, err := s.users.EnsureDevUser(ctx, email)
		if err != nil {
			return err
		}
		if err := ensureActiveUser(user); err != nil {
			return err
		}

		authResult, err := s.createSession(ctx, user, userAgent, requestIP)
		if err != nil {
			return err
		}
		result = authResult
		return nil
	})
	if err != nil {
		return AuthResult{}, err
	}

	return result, nil
}

func (s *Service) createSession(ctx context.Context, user domain.User, userAgent, requestIP string) (AuthResult, error) {
	tokens, err := s.auth.NewSessionTokens()
	if err != nil {
		return AuthResult{}, err
	}

	expiresAt := s.now().Add(s.cfg.SessionTTL)
	if err := s.sessions.Create(ctx, ports.CreateSessionInput{
		UserID:        user.ID,
		TokenHash:     tokens.SessionTokenHash,
		CSRFTokenHash: tokens.CSRFTokenHash,
		UserAgent:     userAgent,
		IP:            requestIP,
		ExpiresAt:     expiresAt,
	}); err != nil {
		return AuthResult{}, err
	}

	return AuthResult{
		Response:  mapCurrentUser(user),
		Tokens:    tokens,
		ExpiresAt: expiresAt,
	}, nil
}

func mapCurrentUser(user domain.User) dto.CurrentUserResponse {
	principal := accessdomain.Principal{
		UserID: user.ID,
		Role:   accessdomain.Role(user.Role),
		Rating: user.Rating,
	}

	return dto.CurrentUserResponse{
		User: dto.UserDTO{
			ID:          user.ID,
			Email:       user.Email.String(),
			Username:    user.Username,
			DisplayName: user.DisplayName,
			AvatarURL:   user.AvatarURL,
			Rating:      user.Rating,
			Role:        user.Role,
		},
		Capabilities: dto.CapabilitiesDTO{
			CanCreateArticle:   principal.CanCreateArticle(),
			CanComment:         principal.CanComment(),
			CanChat:            principal.CanChat(),
			CanReport:          principal.CanReport(),
			CanPublishDirectly: principal.CanPublishDirectly(),
			CanModerate:        principal.CanModerate(),
			CanChatWithDevLead: principal.CanChatWithDevLead(),
		},
	}
}

func ensureActiveUser(user domain.User) error {
	if user.Status != "active" {
		return httpx.Forbidden("Аккаунт недоступен")
	}
	return nil
}

func (s *Service) oauthProviderConfig(provider string) (config.OAuthProviderConfig, error) {
	if provider != "google" && provider != "vk" && provider != "yandex" {
		return config.OAuthProviderConfig{}, httpx.NewError(404, httpx.CodeNotFound, "OAuth-провайдер не найден")
	}
	providerCfg, ok := s.cfg.OAuth.Providers[provider]
	if !ok {
		return config.OAuthProviderConfig{}, httpx.ServiceUnavailable("OAuth-провайдер не настроен", nil)
	}
	return providerCfg, nil
}

func buildOAuthAuthorizationURL(providerCfg config.OAuthProviderConfig, state platformauth.OAuthState) (string, error) {
	parsed, err := url.Parse(providerCfg.AuthURL)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	query.Set("response_type", "code")
	query.Set("client_id", providerCfg.ClientID)
	query.Set("redirect_uri", providerCfg.RedirectURL)
	query.Set("state", state.State)
	query.Set("code_challenge", state.CodeChallenge)
	query.Set("code_challenge_method", "S256")
	if len(providerCfg.Scopes) > 0 {
		query.Set("scope", strings.Join(providerCfg.Scopes, " "))
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func sanitizeReturnTo(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "/"
	}
	if len(value) > 512 {
		return "/"
	}
	if strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//") {
		return value
	}
	return "/"
}

func ClientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return host
	}
	return remoteAddr
}

func ToHTTPAuthenticatedUser(user domain.User) httpx.AuthenticatedUser {
	return httpx.AuthenticatedUser{
		ID:          user.ID,
		Email:       user.Email.String(),
		Username:    user.Username,
		DisplayName: user.DisplayName,
		AvatarURL:   user.AvatarURL,
		Rating:      user.Rating,
		Role:        user.Role,
	}
}
