package bootstrap

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"catch/apps/api/internal/app/composition"
	articlehttp "catch/apps/api/internal/modules/articles/adapters/http"
	bookmarkhttp "catch/apps/api/internal/modules/bookmarks/adapters/http"
	chathttp "catch/apps/api/internal/modules/chat/adapters/http"
	commenthttp "catch/apps/api/internal/modules/comments/adapters/http"
	identityhttp "catch/apps/api/internal/modules/identity/adapters/http"
	mediahttp "catch/apps/api/internal/modules/media/adapters/http"
	moderationhttp "catch/apps/api/internal/modules/moderation/adapters/http"
	notificationhttp "catch/apps/api/internal/modules/notifications/adapters/http"
	profilehttp "catch/apps/api/internal/modules/profiles/adapters/http"
	reactionhttp "catch/apps/api/internal/modules/reactions/adapters/http"
	reporthttp "catch/apps/api/internal/modules/reports/adapters/http"
	httpx "catch/apps/api/internal/platform/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var startedAt = time.Now()

func NewRouter(container *composition.Container) http.Handler {
	r := chi.NewRouter()
	r.Use(httpx.RequestID)
	r.Use(httpx.Recoverer(container.Logger))
	r.Use(httpx.SecurityHeaders)
	r.Use(httpx.Timeout(30 * time.Second))
	r.Use(httpx.NewRateLimiter(300, time.Minute).Middleware())
	r.Use(httpx.AccessLog(container.Logger))

	r.Get("/healthz", health)
	r.Get("/readyz", readiness(container.DB, container.Logger))
	r.Get("/metrics", metrics)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(httpx.AuditLog(container.TxManager, container.Logger))

		identityHandler := identityhttp.NewHandler(container.Config.Auth.DevLoginEnabled, container.Identity, container.Auth)
		identityHandler.RegisterRoutes(r, container.Logger)

		profileHandler := profilehttp.NewHandler(container.Profiles)
		profileHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		articleHandler := articlehttp.NewHandler(container.Articles)
		articleHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		bookmarkHandler := bookmarkhttp.NewHandler(container.Bookmarks)
		bookmarkHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		commentHandler := commenthttp.NewHandler(container.Comments)
		commentHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		reactionHandler := reactionhttp.NewHandler(container.Reactions)
		reactionHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		reportHandler := reporthttp.NewHandler(container.Reports)
		reportHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		moderationHandler := moderationhttp.NewHandler(container.Moderation)
		moderationHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		notificationHandler := notificationhttp.NewHandler(container.Notifications)
		notificationHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		chatHandler := chathttp.NewHandler(container.Chat)
		chatHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))

		mediaHandler := mediahttp.NewHandler(container.Media, container.Config.Storage.MaxUploadBytes)
		mediaHandler.RegisterRoutes(r, container.Logger, identityHandler.RequireAuth(container.Logger), identityHandler.RequireCSRF(container.Logger))
	})

	return r
}

func health(w http.ResponseWriter, r *http.Request) {
	_ = httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func metrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintf(w, "catch_api_uptime_seconds %.0f\n", time.Since(startedAt).Seconds())
}

func readiness(pool *pgxpool.Pool, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			httpx.WriteError(w, r, log, httpx.ServiceUnavailable("База данных недоступна", err))
			return
		}
		_ = httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
	}
}
