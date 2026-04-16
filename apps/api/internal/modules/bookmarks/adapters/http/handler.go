package bookmarkhttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/bookmarks/app"
	"catch/apps/api/internal/modules/bookmarks/app/dto"
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
	bookmarkLimiter := httpx.NewRateLimiter(20, time.Minute).Middleware()
	followLimiter := httpx.NewRateLimiter(30, time.Minute).Middleware()

	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/bookmarks/lists", httpx.Wrap(log, h.lists))
		r.Get("/bookmarks/items", httpx.Wrap(log, h.articles))

		r.Group(func(r chi.Router) {
			r.Use(requireCSRF)
			r.Post("/bookmarks/lists", httpx.Wrap(log, h.createList))
			r.With(bookmarkLimiter).Post("/bookmarks/items", httpx.Wrap(log, h.addBookmark))
			r.Delete("/bookmarks/items", httpx.Wrap(log, h.removeBookmark))
			r.With(followLimiter).Post("/subscriptions/{authorID}", httpx.Wrap(log, h.follow))
			r.Delete("/subscriptions/{authorID}", httpx.Wrap(log, h.unfollow))
		})
	})
}

func (h *Handler) lists(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.Lists(r.Context(), actor)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) createList(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.CreateBookmarkListRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	response, err := h.service.CreateList(r.Context(), actor, request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func (h *Handler) articles(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	response, err := h.service.Articles(r.Context(), actor, r.URL.Query().Get("list_id"), r.URL.Query().Get("q"), limit)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) addBookmark(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.AddBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	if err := h.service.AddBookmark(r.Context(), actor, request); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (h *Handler) removeBookmark(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	var request dto.RemoveBookmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}
	if err := h.service.RemoveBookmark(r.Context(), actor, request); err != nil {
		return err
	}
	httpx.NoContent(w)
	return nil
}

func (h *Handler) follow(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.Follow(r.Context(), actor, chi.URLParam(r, "authorID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) unfollow(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.Unfollow(r.Context(), actor, chi.URLParam(r, "authorID"))
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
