package route

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"moria/internal/session"
	"moria/internal/user"

	"github.com/jmoiron/sqlx"
)

// ValidateSession serves the cluster-internal session validation endpoint.
// It is unauthenticated and must never be exposed through an ingress.
func ValidateSession(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.PathValue("id")

		v, err := session.Validate(r.Context(), db, sessionID)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or expired session"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(v)
	}
}

// ListUsersInternal serves batch user lookups for the cluster-internal
// listener, so other services can hydrate usernames without owning the
// users table. Missing ids are simply absent from the result.
func ListUsersInternal(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var ids []string
		for _, id := range strings.Split(r.URL.Query().Get("ids"), ",") {
			if id != "" {
				ids = append(ids, id)
			}
		}

		users, err := user.ByIDs(r.Context(), db, ids)
		if err != nil {
			logger.Error("failed to list users by ids", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list users"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(users)
	}
}
