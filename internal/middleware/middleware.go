package middleware

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julian-one/moria/internal/session"
	"github.com/julian-one/moria/internal/user"
)

type Middleware func(http.Handler) http.Handler

func Chain(mws ...Middleware) Middleware {
	return func(h http.Handler) http.Handler {
		for _, mw := range slices.Backward(mws) {
			h = mw(h)
		}
		return h
	}
}

func Authenticate(db *pgxpool.Pool) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := session.Token(r)
			u, err := session.Verify(r.Context(), db, token)
			if errors.Is(err, session.ErrInvalidSession) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = fmt.Fprintf(w, `{"error": %q}`, session.ErrInvalidSession.Error())
				return
			}
			if err != nil {
				slog.ErrorContext(r.Context(), "verify session", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, `{"error": %q}`, "internal error")
				return
			}

			next.ServeHTTP(w, r.WithContext(user.NewContext(r.Context(), u)))
		})
	}
}

func Authorize(role user.Role) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := user.FromContext(r.Context())
			if u == nil || u.Role != role {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = fmt.Fprintf(w, `{"error": %q}`, user.ErrForbidden.Error())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
