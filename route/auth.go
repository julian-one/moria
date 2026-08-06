package route

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jmoiron/sqlx"

	"moria/internal/database"
	"moria/internal/session"
	"moria/internal/user"
)

func Login(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		identifier, password, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid basic auth credentials"})
			return
		}

		u, err := user.Authenticate(r.Context(), db, identifier, password)
		if errors.Is(err, user.ErrInvalidCredentials) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}
		if err != nil {
			slog.Error("failed to authenticate", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication error"})
			return
		}

		if err := session.DeleteExpired(r.Context(), db); err != nil {
			slog.Error("failed to delete expired sessions", "error", err)
		}

		s, token, err := session.Mint(r.Context(), db, u.UserID)
		if err != nil {
			slog.Error("failed to create session", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Token     session.Token    `json:"token"`
			ExpiresAt database.UTCTime `json:"expires_at"`
		}{token, s.ExpiresAt})
	}
}

func Logout(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(session.CookieName)
		if err == nil && cookie.Value != "" {
			if err := session.Delete(r.Context(), db, session.Token(cookie.Value).ID()); err != nil {
				slog.Error("failed to delete session", "error", err)
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
