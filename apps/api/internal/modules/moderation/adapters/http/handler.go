package moderationhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/moderation/app"
	"catch/apps/api/internal/modules/moderation/app/dto"
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
		r.Get("/moderation/submissions", httpx.Wrap(log, h.list))
		r.Get("/moderation/submissions/{submissionID}/threads", httpx.Wrap(log, h.listThreads))
		r.Group(func(r chi.Router) {
			r.Use(requireCSRF)
			r.Post("/moderation/submissions/{submissionID}/approve", httpx.Wrap(log, h.approve))
			r.Post("/moderation/submissions/{submissionID}/reject", httpx.Wrap(log, h.reject))
			r.Post("/moderation/submissions/{submissionID}/threads", httpx.Wrap(log, h.createThread))
			r.Post("/moderation/threads/{threadID}/resolve", httpx.Wrap(log, h.resolveThread))
			r.Post("/moderation/threads/{threadID}/reopen", httpx.Wrap(log, h.reopenThread))
		})
	})
}

func (h *Handler) listThreads(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.ListThreads(r.Context(), actor, chi.URLParam(r, "submissionID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
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

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.Approve(r.Context(), actor, chi.URLParam(r, "submissionID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.RejectSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.Reject(r.Context(), actor, chi.URLParam(r, "submissionID"), request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) createThread(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.CreateThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.CreateThread(r.Context(), actor, chi.URLParam(r, "submissionID"), request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func (h *Handler) resolveThread(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.ResolveThread(r.Context(), actor, chi.URLParam(r, "threadID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) reopenThread(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.ReopenThreadRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.ReopenThread(r.Context(), actor, chi.URLParam(r, "threadID"), request)
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
