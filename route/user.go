package route

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/julian-one/moria/internal/user"
)

func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).
			Encode(user.FromContext(r.Context())); err != nil {
			slog.ErrorContext(r.Context(), "write current user response", "error", err)
		}
	}
}

func CreateUser(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var request user.CreateRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			slog.InfoContext(r.Context(), "decode create user request", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, "invalid request body")
			return
		}

		u, err := user.Create(r.Context(), db, request)
		if errors.Is(err, user.ErrInvalidUsername) ||
			errors.Is(err, user.ErrInvalidEmail) ||
			errors.Is(err, user.ErrInvalidPassword) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, err.Error())
			return
		}
		if errors.Is(err, user.ErrUserExists) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, err.Error())
			return
		}
		if err != nil {
			slog.ErrorContext(r.Context(), "create user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, `{"error": %q}`, "internal error")
			return
		}

		slog.InfoContext(r.Context(), "user created", "user_id", u.UserID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(u); err != nil {
			slog.ErrorContext(r.Context(), "write create user response", "error", err)
		}
	}
}
