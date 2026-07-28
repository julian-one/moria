package route

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"moria/internal/session"

	"github.com/jmoiron/sqlx"
)

func GetSession(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.PathValue("id")
		s, err := session.ByID(ctx, db, id)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve session"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s)
	}
}

func ListSessions(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := r.PathValue("id")
		sessions, err := session.List(ctx, db, id)
		if err != nil {
			logger.Error("failed to list sessions", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list sessions"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Items []session.Session `json:"items"`
			Total int               `json:"total"`
		}{sessions, len(sessions)})
	}
}

func DeleteSession(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sessionID := r.PathValue("id")
		err := session.Delete(ctx, db, sessionID)
		if err != nil {
			logger.Error("failed to delete session", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete session"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DeleteAllSessions(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		userID := r.PathValue("id")
		err := session.DeleteAll(ctx, db, userID)
		if err != nil {
			logger.Error("failed to delete all sessions", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete all sessions"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
