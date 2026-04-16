package httpx

import (
	"log/slog"
	"net"
	"net/http"

	"catch/apps/api/internal/platform/db"
)

func AuditLog(tx *db.TxManager, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)

			if !shouldAudit(r.Method) {
				return
			}
			userID := ""
			if user, ok := AuthenticatedUserFromContext(r.Context()); ok {
				userID = user.ID
			}
			if _, err := tx.Querier(r.Context()).Exec(r.Context(), `
				insert into audit_log (actor_user_id, method, path, status, trace_id, ip, user_agent)
				values (nullif($1, '')::uuid, $2, $3, $4, nullif($5, ''), nullif($6, '')::inet, nullif($7, ''))
			`, userID, r.Method, r.URL.Path, recorder.status, RequestIDFromContext(r.Context()), auditIP(r), r.UserAgent()); err != nil {
				log.WarnContext(r.Context(), "audit_log_write_failed", slog.String("error", err.Error()))
			}
		})
	}
}

func shouldAudit(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

func auditIP(r *http.Request) string {
	ip := clientIP(r)
	if parsed := net.ParseIP(ip); parsed != nil {
		return ip
	}
	return ""
}

func nilLogger() *slog.Logger {
	return slog.Default()
}
