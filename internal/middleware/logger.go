package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type logContextKey string

const LogContextKey logContextKey = "log_context"

type LogContext struct {
	UserID    string
	SessionID string
}

// Logger returns middleware that logs each request after the handler completes,
// including method, path, status, duration, a generated request ID, and optional user/session info.
func Logger(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			requestID := uuid.New().String()
			logCtx := &LogContext{}

			// Add logCtx to context so inner middlewares/handlers can populate it
			ctx := context.WithValue(r.Context(), LogContextKey, logCtx)
			r = r.WithContext(ctx)

			requestLogger := logger.With(slog.String("request_id", requestID))
			w.Header().Set("X-Request-ID", requestID)

			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rw, r)

			// Read updated fields from logCtx
			fields := []any{
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", GetClientIP(r),
				"status", rw.status,
				"duration", time.Since(start),
			}
			if logCtx.UserID != "" {
				fields = append(fields, slog.String("user_id", logCtx.UserID))
			}
			if logCtx.SessionID != "" {
				fields = append(fields, slog.String("session_id", logCtx.SessionID))
			}

			requestLogger.Info("http request completed", fields...)
		})
	}
}
