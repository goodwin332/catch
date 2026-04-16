package identityhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	identityapp "catch/apps/api/internal/modules/identity/app"
	"catch/apps/api/internal/modules/identity/app/dto"
	platformauth "catch/apps/api/internal/platform/auth"
	httpx "catch/apps/api/internal/platform/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	devLoginEnabled bool
	service         *identityapp.Service
	auth            *platformauth.Manager
}

func NewHandler(devLoginEnabled bool, service *identityapp.Service, authManager *platformauth.Manager) *Handler {
	return &Handler{
		devLoginEnabled: devLoginEnabled,
		service:         service,
		auth:            authManager,
	}
}

func (h *Handler) RegisterRoutes(r chi.Router, log *slog.Logger) {
	emailCodeLimiter := httpx.NewRateLimiter(10, time.Minute).Middleware()
	emailVerifyLimiter := httpx.NewRateLimiter(20, time.Minute).Middleware()
	devLoginLimiter := httpx.NewRateLimiter(30, time.Minute).Middleware()

	r.Route("/auth", func(r chi.Router) {
		r.With(emailCodeLimiter).Post("/email/request-code", httpx.Wrap(log, h.requestEmailCode))
		r.With(emailVerifyLimiter).Post("/email/verify", httpx.Wrap(log, h.verifyEmailCode))
		r.Get("/oauth/{provider}/start", httpx.Wrap(log, h.oauthStart))
		r.Get("/oauth/{provider}/callback", httpx.Wrap(log, h.oauthCallback))

		r.Group(func(r chi.Router) {
			r.Use(h.RequireAuth(log))
			r.Get("/me", httpx.Wrap(log, h.currentUser))

			r.Group(func(r chi.Router) {
				r.Use(h.RequireCSRF(log))
				r.Post("/logout", httpx.Wrap(log, h.logout))
			})
		})
	})

	if h.devLoginEnabled {
		r.Route("/dev/auth", func(r chi.Router) {
			r.With(devLoginLimiter).Post("/login", httpx.Wrap(log, h.devLogin))
		})
	}
}

func (h *Handler) requestEmailCode(w http.ResponseWriter, r *http.Request) error {
	var request dto.RequestEmailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	response, err := h.service.RequestEmailCode(r.Context(), request, identityapp.ClientIP(r.RemoteAddr))
	if err != nil {
		return err
	}

	return httpx.JSON(w, http.StatusAccepted, response)
}

func (h *Handler) verifyEmailCode(w http.ResponseWriter, r *http.Request) error {
	var request dto.VerifyEmailCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	result, err := h.service.VerifyEmailCode(r.Context(), request, r.UserAgent(), identityapp.ClientIP(r.RemoteAddr))
	if err != nil {
		return err
	}
	h.auth.SetSessionCookies(w, result.Tokens, result.ExpiresAt)

	return httpx.JSON(w, http.StatusOK, result.Response)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) error {
	token, ok := h.auth.ReadSessionToken(r)
	if !ok {
		return httpx.Unauthorized("Требуется авторизация")
	}

	response, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		return err
	}

	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) error {
	token, _ := h.auth.ReadSessionToken(r)
	if err := h.service.Logout(r.Context(), token); err != nil {
		return err
	}
	h.auth.ClearSessionCookies(w)

	httpx.NoContent(w)
	return nil
}

func (h *Handler) oauthStart(w http.ResponseWriter, r *http.Request) error {
	provider := chi.URLParam(r, "provider")
	result, err := h.service.StartOAuth(provider, r.URL.Query().Get("return_to"))
	if err != nil {
		return err
	}
	h.auth.SetOAuthStateCookie(w, result.StateCookie)
	http.Redirect(w, r, result.AuthorizationURL, http.StatusFound)
	return nil
}

func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) error {
	provider := chi.URLParam(r, "provider")
	stateCookie, err := r.Cookie(h.service.OAuthStateCookieName())
	if err != nil {
		return httpx.Forbidden("OAuth state cookie обязателен")
	}

	result, returnTo, err := h.service.CompleteOAuth(
		r.Context(),
		provider,
		r.URL.Query().Get("code"),
		r.URL.Query().Get("state"),
		stateCookie.Value,
		r.UserAgent(),
		identityapp.ClientIP(r.RemoteAddr),
	)
	if err != nil {
		return err
	}
	h.auth.ClearOAuthStateCookie(w, h.service.OAuthStateCookieName())
	h.auth.SetSessionCookies(w, result.Tokens, result.ExpiresAt)
	http.Redirect(w, r, returnTo, http.StatusFound)
	return nil
}

func (h *Handler) devLogin(w http.ResponseWriter, r *http.Request) error {
	var request dto.DevLoginRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
		}
	}

	result, err := h.service.DevLogin(r.Context(), request, r.UserAgent(), identityapp.ClientIP(r.RemoteAddr))
	if err != nil {
		return err
	}
	h.auth.SetSessionCookies(w, result.Tokens, result.ExpiresAt)

	return httpx.JSON(w, http.StatusOK, result.Response)
}
