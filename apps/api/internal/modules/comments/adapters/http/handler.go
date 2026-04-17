package commenthttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/comments/app"
	"catch/apps/api/internal/modules/comments/app/dto"
	httpx "catch/apps/api/internal/platform/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *app.Service
}

func NewHandler(service *app.Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(r chi.Router, log *slog.Logger, requireAuth func(http.Handler) http.Handler, requireCSRF func(http.Handler) http.Handler) {
	commentLimiter := httpx.NewRateLimiter(5, time.Minute).Middleware()

	r.Get("/articles/{articleID}/comments", httpx.Wrap(log, h.list))
	r.Get("/comments/{commentID}", httpx.Wrap(log, h.get))
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Use(requireCSRF)
		r.With(commentLimiter).Post("/articles/{articleID}/comments", httpx.Wrap(log, h.create))
		r.Patch("/comments/{commentID}", httpx.Wrap(log, h.update))
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.ListByArticle(r.Context(), chi.URLParam(r, "articleID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.Get(r.Context(), chi.URLParam(r, "commentID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	var request dto.CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	response, err := h.service.Create(r.Context(), actor, chi.URLParam(r, "articleID"), request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	var request dto.UpdateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	response, err := h.service.Update(r.Context(), actor, chi.URLParam(r, "commentID"), request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func actorFromRequest(r *http.Request) (accessdomain.Principal, error) {
	user, ok := httpx.AuthenticatedUserFromContext(r.Context())
	if !ok {
		return accessdomain.Principal{}, httpx.Unauthorized("Требуется авторизация")
	}
	return accessdomain.Principal{
		UserID: user.ID,
		Role:   accessdomain.Role(user.Role),
		Rating: user.Rating,
	}, nil
}
