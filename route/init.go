package route

import (
	"net/http"

	"github.com/jmoiron/sqlx"

	"moria/internal/middleware"
)

type Config struct {
	DB *sqlx.DB
}

func Initialize(config Config) http.Handler {
	r := &router{mux: http.NewServeMux()}

	r.Handle("GET /health", Health())

	r.Handle("POST /login", Login(config.DB))
	r.Handle("POST /logout", Logout(config.DB))

	r.Group(func(r *router) {
		r.Use(middleware.Authentication(config.DB))

		r.Handle("GET /me", Me())
		r.Handle("PATCH /users/{id}", UpdateUser(config.DB))
		r.Handle("PATCH /users/{id}/password", UpdatePassword(config.DB))
	})

	r.Group(func(r *router) {
		r.Use(middleware.Admin(config.DB))

		r.Handle("POST /users", CreateUser(config.DB))
		r.Handle("GET /users", ListUsers(config.DB))
		r.Handle("GET /users/{id}", GetUser(config.DB))
		r.Handle("PATCH /users/{id}/role", UpdateUserRole(config.DB))
		r.Handle("GET /sessions/{id}", GetSession(config.DB))
		r.Handle("GET /users/{id}/sessions", ListSessions(config.DB))
		r.Handle("DELETE /users/{id}/sessions", DeleteAllSessions(config.DB))
		r.Handle("DELETE /sessions/{id}", DeleteSession(config.DB))
	})

	return middleware.Logger()(r.mux)
}
