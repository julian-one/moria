package route

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"moria/internal/email"
	"moria/internal/middleware"

	"github.com/jmoiron/sqlx"
	"github.com/rs/cors"
	"golang.org/x/time/rate"
)

type Config struct {
	Logger     *slog.Logger
	DB         *sqlx.DB
	Email      *email.Client
	SigningKey string
}

func Initialize(ctx context.Context, config Config) http.Handler {
	baseChain := middleware.New(
		middleware.Logger(config.Logger),
	)
	protectedChain := baseChain.Append(
		middleware.Authentication(config.DB),
	)
	// NOTE: Admin middleware is used as part of the protected chain
	adminChain := protectedChain.Append(
		middleware.Admin(config.DB),
	)

	// 10 requests per minute (1 token every 6 secs), max burst of 3
	registerChain := baseChain.Append(
		middleware.NewRateLimiter(ctx, rate.Every(time.Minute/10), 3),
	)
	// 3 requests per 15 minutes (1 token every 5 mins), max burst of 1
	forgotPasswordChain := baseChain.Append(
		middleware.NewRateLimiter(ctx, rate.Every(15*time.Minute/3), 1),
	)
	// 10 requests per minute (1 token every 6 secs), max burst of 3
	loginChain := baseChain.Append(
		middleware.NewRateLimiter(ctx, rate.Every(time.Minute/10), 3),
	)

	mux := http.NewServeMux()

	// -----------------
	// Health
	// -----------------
	mux.Handle("GET /health", baseChain.Wrap(Health()))

	// -----------------
	// Auth
	// -----------------
	mux.Handle(
		"POST /register",
		registerChain.Wrap(Register(config.Logger, config.DB, config.Email, config.SigningKey)),
	)
	mux.Handle(
		"POST /register/verify",
		baseChain.Wrap(VerifyRegistration(config.Logger, config.SigningKey)),
	)
	mux.Handle(
		"POST /register/complete",
		baseChain.Wrap(CompleteRegistration(config.Logger, config.DB, config.SigningKey)),
	)
	mux.Handle("POST /login", loginChain.Wrap(Login(config.Logger, config.DB)))
	mux.Handle("POST /logout", baseChain.Wrap(Logout(config.Logger, config.DB)))
	mux.Handle(
		"POST /forgot-password",
		forgotPasswordChain.Wrap(
			ForgotPassword(config.Logger, config.DB, config.Email, config.SigningKey),
		),
	)
	mux.Handle(
		"POST /reset-password",
		baseChain.Wrap(ResetPassword(config.Logger, config.DB, config.SigningKey)),
	)

	// -----------------
	// Users
	// -----------------
	mux.Handle("GET /users", adminChain.Wrap(ListUsers(config.Logger, config.DB)))
	mux.Handle("GET /users/{id}", protectedChain.Wrap(GetUser(config.Logger, config.DB)))
	mux.Handle("PATCH /users/{id}", protectedChain.Wrap(UpdateUser(config.Logger, config.DB)))
	mux.Handle(
		"PATCH /users/{id}/password",
		protectedChain.Wrap(UpdatePassword(config.Logger, config.DB)),
	)
	mux.Handle(
		"PATCH /users/{id}/role",
		adminChain.Wrap(UpdateUserRole(config.Logger, config.DB)),
	)

	// -----------------
	// Sessions
	// -----------------
	mux.Handle("GET /sessions/{id}", protectedChain.Wrap(GetSession(config.Logger, config.DB)))
	mux.Handle("DELETE /sessions/{id}", adminChain.Wrap(DeleteSession(config.Logger, config.DB)))
	mux.Handle(
		"GET /users/{id}/sessions",
		adminChain.Wrap(ListSessions(config.Logger, config.DB)),
	)
	mux.Handle(
		"DELETE /users/{id}/sessions",
		adminChain.Wrap(DeleteAllSessions(config.Logger, config.DB)),
	)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"https://julian-one.com", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "Cache-Control"},
		AllowCredentials: true,
		MaxAge:           300,
	})

	return c.Handler(mux)
}

// InitializeInternal builds the handler for the cluster-internal listener.
// It serves only session validation and is never routed through the ingress.
func InitializeInternal(ctx context.Context, config Config) http.Handler {
	baseChain := middleware.New(
		middleware.Logger(config.Logger),
	)

	mux := http.NewServeMux()
	mux.Handle(
		"GET /internal/sessions/{id}",
		baseChain.Wrap(ValidateSession(config.Logger, config.DB)),
	)
	mux.Handle(
		"GET /internal/users",
		baseChain.Wrap(ListUsersInternal(config.Logger, config.DB)),
	)

	return mux
}
