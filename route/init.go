package route

import (
	"database/sql"
	"net/http"

	"moria/internal/middleware"
)

type Config struct {
	DB *sql.DB
}

func Initialize(config Config) http.Handler {
	baseChain := middleware.New(
		middleware.Logger(),
	)
	protectedChain := baseChain.Append(
		middleware.Authentication(config.DB),
	)
	adminChain := protectedChain.Append(
		middleware.Admin(config.DB),
	)

	mux := http.NewServeMux()

	mux.Handle("GET /health", baseChain.Wrap(Health()))

	// Login is rate limited at Traefik; moria only ever sees shire's IP.
	mux.Handle("POST /login", baseChain.Wrap(Login(config.DB)))
	mux.Handle("POST /logout", baseChain.Wrap(Logout(config.DB)))

	// Admin-tier because accounts are provisioned, never self-registered.
	mux.Handle("POST /users", adminChain.Wrap(CreateUser(config.DB)))
	mux.Handle("GET /users", adminChain.Wrap(ListUsers(config.DB)))
	mux.Handle("GET /users/{id}", protectedChain.Wrap(GetUser(config.DB)))
	mux.Handle("PATCH /users/{id}", protectedChain.Wrap(UpdateUser(config.DB)))
	mux.Handle("PATCH /users/{id}/password", protectedChain.Wrap(UpdatePassword(config.DB)))
	mux.Handle("PATCH /users/{id}/role", adminChain.Wrap(UpdateUserRole(config.DB)))

	mux.Handle("GET /sessions/{id}", protectedChain.Wrap(GetSession(config.DB)))
	mux.Handle("DELETE /sessions/{id}", adminChain.Wrap(DeleteSession(config.DB)))
	mux.Handle("GET /users/{id}/sessions", adminChain.Wrap(ListSessions(config.DB)))
	mux.Handle("DELETE /users/{id}/sessions", adminChain.Wrap(DeleteAllSessions(config.DB)))

	return mux
}
