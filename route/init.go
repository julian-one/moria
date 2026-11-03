package route

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

func Initialize(db *pgxpool.Pool) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", Health())
	mux.HandleFunc("POST /users", CreateUser(db))

	return mux
}

func Health() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.InfoContext(r.Context(), "health check", "remote_addr", r.RemoteAddr)
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			slog.ErrorContext(r.Context(), "write health response", "error", err)
		}
	}
}
