package reporthttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/reports/app"
	"catch/apps/api/internal/modules/reports/app/dto"
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
	reportLimiter := httpx.NewRateLimiter(10, time.Minute).Middleware()

	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/reports", httpx.Wrap(log, h.listPending))
		r.Group(func(r chi.Router) {
			r.Use(requireCSRF)
			r.With(reportLimiter).Post("/reports", httpx.Wrap(log, h.create))
			r.Post("/reports/{reportID}/decisions", httpx.Wrap(log, h.decide))
		})
	})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.CreateReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.Create(r.Context(), actor, request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func (h *Handler) listPending(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	response, err := h.service.ListPending(r.Context(), actor, limit)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) decide(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.DecideReportRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.Decide(r.Context(), actor, chi.URLParam(r, "reportID"), request)
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
	return accessdomain.Principal{UserID: user.ID, Role: accessdomain.Role(user.Role), Rating: user.Rating}, nil
}
