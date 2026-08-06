package route

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/jmoiron/sqlx"

	"moria/internal/session"
)

func GetSession(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s, err := session.ByID(r.Context(), db, session.ID(r.PathValue("id")))
		if errors.Is(err, session.ErrNotFound) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "Session not found"})
			return
		}
		if err != nil {
			slog.Error("failed to get session", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get session"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s)
	}
}

func ListSessions(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessions, err := session.List(r.Context(), db, r.PathValue("id"))
		if err != nil {
			slog.Error("failed to list sessions", "error", err)
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

func DeleteSession(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := session.Delete(r.Context(), db, session.ID(r.PathValue("id"))); err != nil {
			slog.Error("failed to delete session", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete session"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func DeleteAllSessions(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := session.DeleteAll(r.Context(), db, r.PathValue("id")); err != nil {
			slog.Error("failed to delete all sessions", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to delete all sessions"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
