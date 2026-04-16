package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"catch/apps/api/internal/app/config"
)

type Profile struct {
	ProviderAccountID string
	Email             string
	DisplayName       string
	AvatarURL         string
}

type Client interface {
	Exchange(ctx context.Context, provider string, cfg config.OAuthProviderConfig, code, codeVerifier string) (Profile, error)
}

type HTTPClient struct {
	httpClient *http.Client
}

func NewHTTPClient(timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &HTTPClient{
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *HTTPClient) Exchange(ctx context.Context, provider string, cfg config.OAuthProviderConfig, code, codeVerifier string) (Profile, error) {
	token, err := c.exchangeToken(ctx, cfg, code, codeVerifier)
	if err != nil {
		return Profile{}, err
	}
	userInfo, err := c.fetchUserInfo(ctx, cfg.UserInfoURL, token.AccessToken)
	if err != nil {
		return Profile{}, err
	}

	profile := Profile{
		ProviderAccountID: firstString(userInfo, "sub", "id", "uid", "user_id"),
		Email:             firstString(userInfo, "email", "default_email"),
		DisplayName:       firstString(userInfo, "name", "display_name", "real_name", "login"),
		AvatarURL:         firstString(userInfo, "picture", "avatar_url", "photo_200"),
	}
	if profile.Email == "" {
		profile.Email = token.Email
	}
	if profile.DisplayName == "" {
		profile.DisplayName = strings.TrimSpace(firstString(userInfo, "first_name") + " " + firstString(userInfo, "last_name"))
	}
	if profile.ProviderAccountID == "" {
		return Profile{}, fmt.Errorf("%s OAuth profile does not contain account id", provider)
	}
	if profile.Email == "" {
		return Profile{}, fmt.Errorf("%s OAuth profile does not contain email", provider)
	}

	return profile, nil
}

func (c *HTTPClient) exchangeToken(ctx context.Context, cfg config.OAuthProviderConfig, code, codeVerifier string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURL)
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code_verifier", codeVerifier)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return tokenResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return tokenResponse{}, fmt.Errorf("OAuth token exchange failed with status %d: %s", response.StatusCode, string(body))
	}

	var token tokenResponse
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil {
		return tokenResponse{}, err
	}
	if token.AccessToken == "" {
		return tokenResponse{}, errors.New("OAuth token response does not contain access_token")
	}
	return token, nil
}

func (c *HTTPClient) fetchUserInfo(ctx context.Context, userInfoURL, accessToken string) (map[string]any, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return nil, fmt.Errorf("OAuth userinfo request failed with status %d: %s", response.StatusCode, string(body))
	}

	contentType, _, _ := mime.ParseMediaType(response.Header.Get("Content-Type"))
	body, err := io.ReadAll(io.LimitReader(response.Body, 1024*1024))
	if err != nil {
		return nil, err
	}
	if contentType == "application/x-www-form-urlencoded" {
		values, err := url.ParseQuery(string(body))
		if err != nil {
			return nil, err
		}
		result := make(map[string]any, len(values))
		for key := range values {
			result[key] = values.Get(key)
		}
		return result, nil
	}

	var result map[string]any
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	Email       string `json:"email"`
}

func firstString(values map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := values[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) != "" {
				return strings.TrimSpace(typed)
			}
		case float64:
			return strconv.FormatInt(int64(typed), 10)
		case json.Number:
			return typed.String()
		}
	}
	return ""
}
