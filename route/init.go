package route

import (
	"log/slog"
	"net/http"

	"moria/internal/email"
	"moria/internal/middleware"

	"github.com/jmoiron/sqlx"
)

type Config struct {
	Logger     *slog.Logger
	DB         *sqlx.DB
	Email      *email.Client
	SigningKey string
}

func Initialize(config Config) http.Handler {
	baseChain := middleware.New(
		middleware.Logger(config.Logger),
		middleware.BodyLimit(1<<20),
	)
	protectedChain := baseChain.Append(
		middleware.Authentication(config.DB),
	)
	adminChain := protectedChain.Append(
		middleware.Admin(config.DB),
	)

	mux := http.NewServeMux()

	// -----------------
	// Health
	// -----------------
	mux.Handle("GET /health", baseChain.Wrap(Health()))

	// -----------------
	// Auth
	// -----------------
	// Register, login, and forgot-password are rate limited upstream at
	// Traefik (docs/security.md).
	mux.Handle(
		"POST /register",
		baseChain.Wrap(Register(config.Logger, config.DB, config.Email, config.SigningKey)),
	)
	mux.Handle(
		"POST /register/verify",
		baseChain.Wrap(VerifyRegistration(config.Logger, config.SigningKey)),
	)
	mux.Handle(
		"POST /register/complete",
		baseChain.Wrap(CompleteRegistration(config.Logger, config.DB, config.SigningKey)),
	)
	mux.Handle("POST /login", baseChain.Wrap(Login(config.Logger, config.DB)))
	mux.Handle("POST /logout", baseChain.Wrap(Logout(config.Logger, config.DB)))
	mux.Handle(
		"POST /forgot-password",
		baseChain.Wrap(
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

	// -----------------
	// Internal (service-to-service)
	// -----------------
	// Unauthenticated by design — safe only because moria has no ingress
	// (docs/security.md).
	mux.Handle(
		"GET /internal/sessions/{id}",
		baseChain.Wrap(ValidateSession(config.Logger, config.DB)),
	)
	mux.Handle(
		"GET /internal/users",
		baseChain.Wrap(ListUsersInternal(config.Logger, config.DB)),
	)

	// No CORS: browsers never talk to moria directly (docs/security.md).
	return mux
}
