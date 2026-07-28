package middleware

import (
	"encoding/json"
	"net/http"

	"moria/internal/session"
	"moria/internal/user"

	"github.com/jmoiron/sqlx"
)

// Admin checks that the authenticated user has the admin role.
// Must be used after Authentication middleware.
func Admin(db *sqlx.DB) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			s, ok := ctx.Value(session.ContextKey).(*session.Session)
			if !ok || s == nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
				return
			}

			u, err := user.ByID(ctx, db, s.User)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "Forbidden"})
				return
			}

			if u.Role != user.RoleAdmin {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).
					Encode(map[string]string{"error": "Forbidden: admin access required"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
