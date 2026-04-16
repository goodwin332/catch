package mediahttp

import (
	"io"
	"log/slog"
	"net/http"

	accessdomain "catch/apps/api/internal/modules/access/domain"
	"catch/apps/api/internal/modules/media/app"
	httpx "catch/apps/api/internal/platform/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service        *app.Service
	maxUploadBytes int64
}

func NewHandler(service *app.Service, maxUploadBytes int64) *Handler {
	return &Handler{service: service, maxUploadBytes: maxUploadBytes}
}

func (h *Handler) RegisterRoutes(r chi.Router, log *slog.Logger, requireAuth func(http.Handler) http.Handler, requireCSRF func(http.Handler) http.Handler) {
	r.Get("/media/files/{fileID}", httpx.Wrap(log, h.get))
	r.Get("/media/files/{fileID}/content", httpx.Wrap(log, h.content))

	r.Group(func(r chi.Router) {
		r.Use(requireAuth)
		r.Use(requireCSRF)
		r.Post("/media/files", httpx.Wrap(log, h.upload))
	})
}

func (h *Handler) upload(w http.ResponseWriter, r *http.Request) error {
	actor, err := actorFromRequest(r)
	if err != nil {
		return err
	}
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadBytes+1024)
	if err := r.ParseMultipartForm(h.maxUploadBytes); err != nil {
		return httpx.ValidationError("Файл указан некорректно", map[string]any{"file": "multipart_required"})
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return httpx.ValidationError("Файл указан некорректно", map[string]any{"file": "required"})
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, h.maxUploadBytes+1))
	if err != nil {
		return err
	}
	response, err := h.service.Upload(r.Context(), actor, header.Filename, data)
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusCreated, response)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) error {
	response, err := h.service.Get(r.Context(), chi.URLParam(r, "fileID"))
	if err != nil {
		return err
	}
	return httpx.JSON(w, http.StatusOK, response)
}

func (h *Handler) content(w http.ResponseWriter, r *http.Request) error {
	file, data, err := h.service.Content(r.Context(), chi.URLParam(r, "fileID"))
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+file.OriginalName+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return nil
}

func actorFromRequest(r *http.Request) (accessdomain.Principal, error) {
	user, ok := httpx.AuthenticatedUserFromContext(r.Context())
	if !ok {
		return accessdomain.Principal{}, httpx.Unauthorized("Требуется авторизация")
	}
	return accessdomain.Principal{UserID: user.ID, Role: accessdomain.Role(user.Role), Rating: user.Rating}, nil
}
