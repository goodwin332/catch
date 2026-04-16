package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const oauthPKCEPlainSize = 48

type OAuthState struct {
	Provider      string    `json:"provider"`
	State         string    `json:"state"`
	CodeVerifier  string    `json:"code_verifier"`
	CodeChallenge string    `json:"code_challenge"`
	ReturnTo      string    `json:"return_to"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type OAuthStateCookie struct {
	Name      string
	Value     string
	ExpiresAt time.Time
}

func (m *Manager) NewOAuthState(provider, returnTo string, ttl time.Duration, now time.Time) (OAuthState, error) {
	stateToken, err := randomToken(32)
	if err != nil {
		return OAuthState{}, err
	}
	verifier, err := randomToken(oauthPKCEPlainSize)
	if err != nil {
		return OAuthState{}, err
	}

	sum := sha256.Sum256([]byte(verifier))
	return OAuthState{
		Provider:      provider,
		State:         stateToken,
		CodeVerifier:  verifier,
		CodeChallenge: base64.RawURLEncoding.EncodeToString(sum[:]),
		ReturnTo:      returnTo,
		ExpiresAt:     now.Add(ttl),
	}, nil
}

func (m *Manager) SignOAuthState(state OAuthState) (string, error) {
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := m.hmac("oauth-state:" + encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

func (m *Manager) VerifyOAuthState(value, provider, stateToken string, now time.Time) (OAuthState, error) {
	payload, signature, ok := strings.Cut(value, ".")
	if !ok || payload == "" || signature == "" {
		return OAuthState{}, errors.New("oauth state cookie is malformed")
	}

	expected := m.hmac("oauth-state:" + payload)
	actual, err := base64.RawURLEncoding.DecodeString(signature)
	if err != nil {
		return OAuthState{}, errors.New("oauth state signature is malformed")
	}
	if !hmac.Equal(actual, expected) {
		return OAuthState{}, errors.New("oauth state signature is invalid")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return OAuthState{}, errors.New("oauth state payload is malformed")
	}
	var state OAuthState
	if err := json.Unmarshal(decoded, &state); err != nil {
		return OAuthState{}, err
	}
	if state.Provider != provider {
		return OAuthState{}, errors.New("oauth state provider mismatch")
	}
	if state.State == "" || !hmac.Equal([]byte(state.State), []byte(stateToken)) {
		return OAuthState{}, errors.New("oauth state token mismatch")
	}
	if !state.ExpiresAt.After(now) {
		return OAuthState{}, errors.New("oauth state expired")
	}
	if state.CodeVerifier == "" {
		return OAuthState{}, errors.New("oauth code verifier is empty")
	}

	return state, nil
}

func (m *Manager) SetOAuthStateCookie(w http.ResponseWriter, cookie OAuthStateCookie) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookie.Name,
		Value:    cookie.Value,
		Path:     "/api/v1/auth/oauth",
		Expires:  cookie.ExpiresAt,
		MaxAge:   int(time.Until(cookie.ExpiresAt).Seconds()),
		HttpOnly: true,
		Secure:   m.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) ClearOAuthStateCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/api/v1/auth/oauth",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}
