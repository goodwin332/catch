package chathttp

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/chat/app"
	"catch/apps/api/internal/modules/chat/app/dto"
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
	messageLimiter := httpx.NewRateLimiter(60, time.Minute).Middleware()

	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/chat/conversations", httpx.Wrap(log, h.listConversations))
		r.Get("/chat/conversations/{conversationID}/messages", httpx.Wrap(log, h.listMessages))
		r.Get("/chat/conversations/{conversationID}/messages/stream", httpx.Wrap(log, h.streamMessages))

		r.Group(func(r chi.Router) {
			r.Use(requireCSRF)
			r.Post("/chat/conversations", httpx.Wrap(log, h.startConversation))
			r.With(messageLimiter).Post("/chat/conversations/{conversationID}/messages", httpx.Wrap(log, h.sendMessage))
			r.Post("/chat/conversations/{conversationID}/read", httpx.Wrap(log, h.markRead))
		})
	})
}

func (h *Handler) startConversation(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.StartConversationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.StartConversation(r.Context(), actor, request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func (h *Handler) listConversations(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	limit, err := intQuery(r, "limit")
	if err != nil {
		return err
	}
	response, err := h.service.ListConversations(r.Context(), actor, limit)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) listMessages(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	limit, err := intQuery(r, "limit")
	if err != nil {
		return err
	}
	response, err := h.service.ListMessages(r.Context(), actor, chi.URLParam(r, "conversationID"), r.URL.Query().Get("after_id"), limit)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) streamMessages(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		return httpx.ServiceUnavailable("Streaming недоступен", nil)
	}

	conversationID := chi.URLParam(r, "conversationID")
	afterID := r.URL.Query().Get("after_id")
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	if _, err := fmt.Fprint(w, "event: ready\ndata: {}\n\n"); err != nil {
		return nil
	}
	flusher.Flush()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-ticker.C:
			response, err := h.service.ListMessagesAfter(r.Context(), actor, conversationID, afterID, 50)
			if err != nil {
				return err
			}
			if len(response.Items) == 0 {
				if _, err := fmt.Fprint(w, "event: ping\ndata: {}\n\n"); err != nil {
					return nil
				}
				flusher.Flush()
				continue
			}
			for _, message := range response.Items {
				payload, err := json.Marshal(message)
				if err != nil {
					return err
				}
				if _, err := fmt.Fprintf(w, "event: message\ndata: %s\n\n", payload); err != nil {
					return nil
				}
				afterID = message.ID
			}
			flusher.Flush()
		}
	}
}

func (h *Handler) sendMessage(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.SendMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.SendMessage(r.Context(), actor, chi.URLParam(r, "conversationID"), request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	if err := h.service.MarkRead(r.Context(), actor, chi.URLParam(r, "conversationID")); err != nil {
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
