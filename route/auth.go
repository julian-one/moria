package route

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julian-one/moria/internal/session"
	"github.com/julian-one/moria/internal/user"
)

func Login(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, "invalid request")
			return
		}

		u, err := user.Authenticate(r.Context(), db, user.LoginRequest{
			Identifier: identifier,
			Password:   password,
		})
		if errors.Is(err, user.ErrInvalidIdentifier) ||
			errors.Is(err, user.ErrInvalidPassword) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, err.Error())
			return
		}
		if errors.Is(err, user.ErrInvalidCredentials) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, err.Error())
			return
		}
		if err != nil {
			slog.ErrorContext(r.Context(), "authenticate user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, "internal error")
			return
		}

		s, err := session.Create(r.Context(), db, u.UserID)
		if err != nil {
			slog.ErrorContext(r.Context(), "create session", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, "internal error")
			return
		}

		session.SetCookie(w, s)
		slog.InfoContext(r.Context(), "session created", "user_id", u.UserID)
		w.WriteHeader(http.StatusNoContent)
	}
}

func Logout(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token := session.Token(r); token != "" {
			if err := session.Delete(r.Context(), db, token); err != nil {
				slog.ErrorContext(r.Context(), "delete session", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = fmt.Fprintf(w, `{"error": %q}`, "internal error")
				return
			}
		}

		session.ClearCookie(w)
		w.WriteHeader(http.StatusNoContent)
	}
}
