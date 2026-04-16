package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"math/big"
	"net/http"
	"time"
)

type Manager struct {
	secret            []byte
	sessionCookieName string
	csrfCookieName    string
	csrfHeaderName    string
	secureCookies     bool
}

func NewManager(secret, sessionCookieName, csrfCookieName, csrfHeaderName string, secureCookies bool) *Manager {
	return &Manager{
		secret:            []byte(secret),
		sessionCookieName: sessionCookieName,
		csrfCookieName:    csrfCookieName,
		csrfHeaderName:    csrfHeaderName,
		secureCookies:     secureCookies,
	}
}

func (m *Manager) NewSessionTokens() (SessionTokens, error) {
	sessionToken, err := randomToken(32)
	if err != nil {
		return SessionTokens{}, err
	}
	csrfToken, err := randomToken(32)
	if err != nil {
		return SessionTokens{}, err
	}

	return SessionTokens{
		SessionToken:     sessionToken,
		SessionTokenHash: m.HashSessionToken(sessionToken),
		CSRFToken:        csrfToken,
		CSRFTokenHash:    m.HashCSRFToken(csrfToken),
	}, nil
}

func (m *Manager) NewEmailCode(length int) (string, error) {
	if length < 1 {
		return "", fmt.Errorf("email code length must be positive")
	}

	code := make([]byte, length)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code[i] = byte('0' + n.Int64())
	}

	return string(code), nil
}

func (m *Manager) HashSessionToken(token string) []byte {
	return m.hmac("session:" + token)
}

func (m *Manager) HashCSRFToken(token string) []byte {
	return m.hmac("csrf:" + token)
}

func (m *Manager) HashEmailCode(email, code string) []byte {
	return m.hmac("email-code:" + email + ":" + code)
}

func (m *Manager) ReadSessionToken(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(m.sessionCookieName)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	return cookie.Value, true
}

func (m *Manager) ReadCSRFToken(r *http.Request) (string, bool) {
	value := r.Header.Get(m.csrfHeaderName)
	if value == "" {
		return "", false
	}
	return value, true
}

func (m *Manager) SetSessionCookies(w http.ResponseWriter, tokens SessionTokens, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     m.sessionCookieName,
		Value:    tokens.SessionToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   m.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     m.csrfCookieName,
		Value:    tokens.CSRFToken,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: false,
		Secure:   m.secureCookies,
		SameSite: http.SameSiteLaxMode,
	})
}

func (m *Manager) ClearSessionCookies(w http.ResponseWriter) {
	for _, name := range []string{m.sessionCookieName, m.csrfCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: name == m.sessionCookieName,
			Secure:   m.secureCookies,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func (m *Manager) hmac(value string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

type SessionTokens struct {
	SessionToken     string
	SessionTokenHash []byte
	CSRFToken        string
	CSRFTokenHash    []byte
}

func randomToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
