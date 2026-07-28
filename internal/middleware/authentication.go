package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"moria/internal/session"

	"github.com/jmoiron/sqlx"
)

// Authentication validates the session token from either the TOKEN cookie
// (used by server-side SvelteKit requests) or the Authorization: Bearer header
// (used by client-side browser requests where the cookie can't be sent cross-origin).
func Authentication(db *sqlx.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			cookie, err := r.Cookie(session.CookieName)
			if err == nil && cookie.Value != "" {
				token = cookie.Value
			}

			if token == "" {
				if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
					token = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if token == "" {
				token = r.URL.Query().Get("token")
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

// OptionalAuthentication attempts to extract a session from the request.
// If a valid token is found, the session is added to the context.
// If no token is present or the token is invalid, the request proceeds without a session.
func OptionalAuthentication(db *sqlx.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			cookie, err := r.Cookie(session.CookieName)
			if err == nil && cookie.Value != "" {
				token = cookie.Value
			}

			if token == "" {
				if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
					token = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if token != "" {
				ctx := r.Context()
				s, err := session.ByID(ctx, db, token)
				if err == nil {
					if logCtx, ok := ctx.Value(LogContextKey).(*LogContext); ok {
						logCtx.UserID = s.User
						logCtx.SessionID = s.SessionID
					}
					ctx = context.WithValue(ctx, session.ContextKey, s)
					r = r.WithContext(ctx)
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
