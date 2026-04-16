package notificationhttp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/notifications/app"
	"catch/apps/api/internal/modules/notifications/app/dto"
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
		r.Get("/notifications", httpx.Wrap(log, h.list))
		r.Get("/notifications/unread-count", httpx.Wrap(log, h.unreadCount))
		r.Get("/notifications/stream", httpx.Wrap(log, h.stream))

		r.Group(func(r chi.Router) {
			r.Use(requireCSRF)
			r.Post("/notifications/{notificationID}/read", httpx.Wrap(log, h.markRead))
			r.Post("/notifications/read-target", httpx.Wrap(log, h.markTargetRead))
		})
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	limit, err := intQuery(r, "limit")
	if err != nil {
		return err
	}
	response, err := h.service.List(r.Context(), actor, limit)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) unreadCount(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.UnreadCount(r.Context(), actor)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) stream(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		return httpx.ServiceUnavailable("Streaming недоступен", nil)
	}

	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	sendUnreadCount := func() error {
		response, err := h.service.UnreadCount(r.Context(), actor)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(response)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "event: unread-count\ndata: %s\n\n", payload)
		flusher.Flush()
		return err
	}

	if err := sendUnreadCount(); err != nil {
		return err
	}

	ticker := time.NewTicker(10 * time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	deadline := time.NewTimer(25 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()
	defer deadline.Stop()

	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-deadline.C:
			return nil
		case <-ticker.C:
			if err := sendUnreadCount(); err != nil {
				return nil
			}
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	if err := h.service.MarkRead(r.Context(), actor, chi.URLParam(r, "notificationID")); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (h *Handler) markTargetRead(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.MarkTargetReadRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	if err := h.service.MarkTargetRead(r.Context(), actor, request.TargetType, request.TargetID); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func intQuery(r *http.Request, name string) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, httpx.ValidationError("Некорректные параметры запроса", map[string]any{name: "Должно быть неотрицательное число"})
	}
	return value, nil
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
