package identityhttp

import (
	"log/slog"
	"net/http"

	identityapp "catch/apps/api/internal/modules/identity/app"
	httpx "catch/apps/api/internal/platform/http"
)

func (h *Handler) RequireAuth(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := h.auth.ReadSessionToken(r)
			if !ok {
				httpx.WriteError(w, r, log, httpx.Unauthorized("Требуется авторизация"))
				return
			}

			sessionUser, err := h.service.Authenticate(r.Context(), token)
			if err != nil {
				httpx.WriteError(w, r, log, err)
				return
			}

			ctx := httpx.WithAuthenticatedUser(r.Context(), identityapp.ToHTTPAuthenticatedUser(sessionUser.User))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (h *Handler) RequireCSRF(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := h.auth.ReadSessionToken(r)
			if !ok {
				httpx.WriteError(w, r, log, httpx.Unauthorized("Требуется авторизация"))
				return
			}

			csrfToken, ok := h.auth.ReadCSRFToken(r)
			if !ok {
				httpx.WriteError(w, r, log, httpx.Forbidden("CSRF token обязателен"))
				return
			}

			if err := h.service.ValidateCSRF(r.Context(), token, csrfToken); err != nil {
				httpx.WriteError(w, r, log, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
