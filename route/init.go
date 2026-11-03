package route

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"

	"github.com/julian-one/moria/internal/middleware"
	"github.com/julian-one/moria/internal/user"
)

func Initialize(db *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	authenticated := middleware.Authenticate(db)
	admin := middleware.Chain(authenticated, middleware.Authorize(user.RoleAdmin))

	mux.HandleFunc("GET /health", Health())
	mux.Handle("POST /users", admin(CreateUser(db)))

	mux.HandleFunc("POST /login", Login(db))
	mux.HandleFunc("POST /logout", Logout(db))

	mux.Handle("GET /me", authenticated(Me()))

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	})

	return c.Handler(mux)
}
