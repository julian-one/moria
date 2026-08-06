package middleware

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"moria/internal/session"
	"moria/internal/user"
)

type Middleware func(http.Handler) http.Handler

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

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

func Authentication(db *sqlx.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(session.CookieName)
			if err != nil || cookie.Value == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
				return
			}

			rq, err := session.RequesterByID(r.Context(), db, session.Token(cookie.Value).ID())
			if errors.Is(err, session.ErrNotFound) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired session"})
				return
			}
			if err != nil {
				slog.Error("failed to authenticate", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication error"})
				return
			}

			next.ServeHTTP(w, r.WithContext(session.WithRequester(r.Context(), rq)))
		})
	}
}

func Admin(db *sqlx.DB) Middleware {
	auth := Authentication(db)
	return func(next http.Handler) http.Handler {
		return auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rq := session.RequesterFrom(r.Context()); rq.User.Role != user.RoleAdmin {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).
					Encode(map[string]string{"error": "Forbidden: admin access required"})
				return
			}

			next.ServeHTTP(w, r)
		}))
	}
}
