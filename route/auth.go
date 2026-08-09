package route

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"moria/internal/session"
	"moria/internal/user"
)

func Login(db *sql.DB) http.HandlerFunc {
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

		s, err := session.Create(r.Context(), db, u.UserID)
		if err != nil {
			slog.Error("failed to create session", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s)
	}
}

// Best-effort by design: a missing cookie or a failed delete still returns
// 204, because shire clears the browser cookie regardless and a 500 would
// only strand the user with a cookie they cannot shed.
func Logout(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(session.CookieName)
		if err == nil && cookie.Value != "" {
			if err := session.Delete(r.Context(), db, cookie.Value); err != nil {
				slog.Error("failed to delete session", "error", err)
			}
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
