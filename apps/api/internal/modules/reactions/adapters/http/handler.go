package reactionhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/reactions/app"
	"catch/apps/api/internal/modules/reactions/app/dto"
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
	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Use(requireCSRF)
		r.Post("/reactions", httpx.Wrap(log, h.set))
	})
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	var request dto.SetReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	response, err := h.service.SetReaction(r.Context(), actor, request)
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
