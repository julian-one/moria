package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"moria/internal/session"
	"moria/internal/user"
)

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// X-Forwarded-For is trusted because only in-cluster callers can reach
// moria. It feeds logging only — never key an authorization decision on it.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ip, _, _ := strings.Cut(xff, ",")
		return strings.TrimSpace(ip)
	}
	return r.RemoteAddr
}

func Logger() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rw, r)

			slog.Info("http request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", clientIP(r),
				"status", rw.status,
				"duration", time.Since(start),
			)
		})
	}
}

func Authentication(db *sql.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(session.CookieName)
			if err != nil || cookie.Value == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
				return
			}

			s, err := session.ByID(r.Context(), db, cookie.Value)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired session"})
				return
			}

			ctx := context.WithValue(r.Context(), session.ContextKey, s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// Must be appended after Authentication — it reads the session that
// middleware puts on the context.
func Admin(db *sql.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s, ok := r.Context().Value(session.ContextKey).(*session.Session)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
				return
			}

			u, err := user.ByID(r.Context(), db, s.UserID)
			if err != nil || u.Role != user.RoleAdmin {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden: admin access required"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
