package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"moria/internal/session"

	"github.com/jmoiron/sqlx"
)

// Authentication validates the session id from the TOKEN cookie — the only
// accepted credential (docs/security.md).
func Authentication(db *sqlx.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			cookie, err := r.Cookie(session.CookieName)
			if err == nil && cookie.Value != "" {
				token = cookie.Value
			}

			if token == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
				return
			}

			ctx := r.Context()
			s, err := session.ByID(ctx, db, token)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired session"})
				return
			}

			if logCtx, ok := ctx.Value(LogContextKey).(*LogContext); ok {
				logCtx.UserID = s.User
				logCtx.SessionID = s.SessionID
			}

			ctx = context.WithValue(ctx, session.ContextKey, s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
