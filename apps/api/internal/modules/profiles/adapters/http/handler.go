package profilehttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"catch/apps/api/internal/modules/profiles/app"
	"catch/apps/api/internal/modules/profiles/app/dto"
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
	r.Get("/profiles/{username}", httpx.Wrap(log, h.publicProfile))
	r.Get("/search/people", httpx.Wrap(log, h.searchPeople))

	r.Route("/profile", func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/me", httpx.Wrap(log, h.myProfile))

		r.Group(func(r chi.Router) {
			r.Use(requireCSRF)
			r.Patch("/me", httpx.Wrap(log, h.updateMyProfile))
		})
	})
}

func (h *Handler) myProfile(w http.ResponseWriter, r *http.Request) error {
	user, ok := httpx.AuthenticatedUserFromContext(r.Context())
	if !ok {
		return httpx.Unauthorized("Требуется авторизация")
	}

	response, err := h.service.GetMyProfile(r.Context(), user.ID)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) updateMyProfile(w http.ResponseWriter, r *http.Request) error {
	user, ok := httpx.AuthenticatedUserFromContext(r.Context())
	if !ok {
		return httpx.Unauthorized("Требуется авторизация")
	}

	var request dto.UpdateMyProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	response, err := h.service.UpdateMyProfile(r.Context(), user.ID, request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) publicProfile(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.GetPublicProfile(r.Context(), chi.URLParam(r, "username"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) searchPeople(w http.ResponseWriter, r *http.Request) error {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	response, err := h.service.SearchPublicProfiles(r.Context(), r.URL.Query().Get("q"), limit)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}
