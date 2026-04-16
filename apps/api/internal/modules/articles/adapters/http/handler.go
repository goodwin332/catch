package articlehttp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/articles/app"
	"catch/apps/api/internal/modules/articles/app/dto"
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
	r.Get("/feed", httpx.Wrap(log, h.feed))
	r.Get("/feed/popular", httpx.Wrap(log, h.popularFeed))
	r.Get("/search", httpx.Wrap(log, h.search))
	r.Get("/articles/{articleID}", httpx.Wrap(log, h.publicArticle))

	r.Route("/articles", func(r chi.Router) {
		r.Use(requireAuth)
		r.Get("/feed", httpx.Wrap(log, h.myFeed))
		r.Get("/my", httpx.Wrap(log, h.myArticles))
		r.Get("/drafts/{articleID}", httpx.Wrap(log, h.getDraft))

		r.Group(func(r chi.Router) {
			r.Use(requireCSRF)
			r.Post("/drafts", httpx.Wrap(log, h.createDraft))
			r.Patch("/drafts/{articleID}", httpx.Wrap(log, h.updateDraft))
			r.Post("/drafts/{articleID}/submit", httpx.Wrap(log, h.submitDraft))
		})
	})
}

func (h *Handler) publicArticle(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.GetPublishedArticle(r.Context(), chi.URLParam(r, "articleID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) feed(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.ListFeed(r.Context(), limitFromRequest(r), r.URL.Query().Get("cursor"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) popularFeed(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.ListPopularFeed(r.Context(), limitFromRequest(r))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) myFeed(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	response, err := h.service.ListMyFeed(r.Context(), actor, limitFromRequest(r), r.URL.Query().Get("cursor"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.Search(r.Context(), r.URL.Query().Get("q"), limitFromRequest(r), r.URL.Query().Get("cursor"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) createDraft(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	var request dto.CreateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	response, err := h.service.CreateDraft(r.Context(), actor, request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func limitFromRequest(r *http.Request) int {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		return 10
	}
	return limit
}

func (h *Handler) getDraft(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	response, err := h.service.GetMyDraft(r.Context(), actor, chi.URLParam(r, "articleID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) myArticles(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	response, err := h.service.ListMyArticles(r.Context(), actor, limitFromRequest(r))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) updateDraft(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	var request dto.UpdateDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
	}

	response, err := h.service.UpdateDraft(r.Context(), actor, chi.URLParam(r, "articleID"), request)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) submitDraft(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}

	var request dto.SubmitDraftRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			return httpx.NewError(http.StatusBadRequest, httpx.CodeInvalidRequest, "Некорректный JSON")
		}
	}

	response, err := h.service.SubmitDraft(r.Context(), actor, chi.URLParam(r, "articleID"), request)
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
