package httpx

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type Problem struct {
	Type    string         `json:"type"`
	Title   string         `json:"title"`
	Status  int            `json:"status"`
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	TraceID string         `json:"trace_id,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, payload any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return nil
	}
	return json.NewEncoder(w).Encode(payload)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func WriteError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	appErr := ToAppError(err)
	traceID := RequestIDFromContext(r.Context())

	if appErr.Status >= http.StatusInternalServerError {
		log.ErrorContext(
			r.Context(),
			"http_request_failed",
			slog.String("trace_id", traceID),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("error", appErr.Error()),
		)
	}

	problem := Problem{
		Type:    "about:blank",
		Title:   http.StatusText(appErr.Status),
		Status:  appErr.Status,
		Code:    appErr.Code,
		Message: appErr.Message,
		TraceID: traceID,
		Details: appErr.Details,
	}

	w.Header().Set("Content-Type", "application/problem+json; charset=utf-8")
	w.WriteHeader(appErr.Status)
	_ = json.NewEncoder(w).Encode(problem)
}

type HandlerFunc func(http.ResponseWriter, *http.Request) error

func Wrap(log *slog.Logger, handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			WriteError(w, r, log, err)
		}
	}
}
