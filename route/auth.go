package route

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"moria/internal/email"
	"moria/internal/session"
	"moria/internal/token"
	"moria/internal/user"

	"github.com/jmoiron/sqlx"
)

func Register(
	logger *slog.Logger,
	db *sqlx.DB,
	emailClient *email.Client,
	signingKey string,
) http.HandlerFunc {
	type Request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			logger.Warn("failed to decode register request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		logger = logger.With("username", request.Username, "email", request.Email)
		if request.Username == "" || request.Email == "" {
			logger.Warn("username and email are required")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Username and email are required"})
			return
		}

		// Validate username uniqueness
		usernameTaken, err := user.IsUsernameTaken(ctx, db, request.Username)
		if err != nil {
			logger.Error("failed to check username", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Failed to check username availability"})
			return
		}
		if usernameTaken {
			logger.Warn("username is already taken")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Username is already taken"})
			return
		}

		// Validate email uniqueness
		emailTaken, err := user.IsEmailTaken(ctx, db, request.Email)
		if err != nil {
			logger.Error("failed to check email", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Failed to check email availability"})
			return
		}
		if emailTaken {
			logger.Warn("email is already taken")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email is already taken"})
			return
		}

		// Create signed verification token
		t, err := token.Create(signingKey, request.Username, request.Email)
		if err != nil {
			logger.Error("failed to create verification token", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process registration"})
			return
		}

		// Send verification email
		err = emailClient.SendVerification(
			request.Email,
			request.Username,
			emailClient.VerificationURL(t),
		)
		if err != nil {
			logger.Error("failed to send verification email", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Failed to send verification email"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"email":   request.Email,
			"message": "Verification email sent",
		})
	}
}

func VerifyRegistration(logger *slog.Logger, signingKey string) http.HandlerFunc {
	type Request struct {
		Token string `json:"token"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var request Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			logger.Warn("failed to decode verify-registration request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}
		if request.Token == "" {
			logger.Warn("token is required")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Token is required"})
			return
		}

		// Validate token
		claims, err := token.Verify(signingKey, request.Token)
		if err != nil {
			logger.Warn("verification failed", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Invalid or expired verification token"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"valid":    true,
			"token":    request.Token,
			"username": claims.Username,
		})
	}
}

func CompleteRegistration(logger *slog.Logger, db *sqlx.DB, signingKey string) http.HandlerFunc {
	type Request struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			logger.Warn("failed to decode complete-registration request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}
		if request.Token == "" || request.Password == "" {
			logger.Warn("token and password are required")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Token and password are required"})
			return
		}

		// Validate token
		claims, err := token.Verify(signingKey, request.Token)
		if err != nil {
			logger.Warn("complete-registration token verification failed", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Invalid or expired verification token"})
			return
		}

		userID, err := user.Create(ctx, db, user.CreateRequest{
			Username: claims.Username,
			Email:    claims.Email,
			Password: request.Password,
		})
		if err != nil {
			logger.Error("failed to create user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Username or email is already taken"})
			return
		}

		s, err := session.Create(ctx, db, userID)
		if err != nil {
			logger.Error("failed to create session after verification", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Account created but failed to create session"})
			return
		}
		session.SetSessionCookie(w, s.SessionID)

		logger.Info("email verified and user created", "user_id", userID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s)
	}
}

func Login(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		identifier, password, ok := r.BasicAuth()
		if !ok {
			logger.Warn("invalid basic auth credentials")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid basic auth credentials"})
			return
		}

		// Get user by email or username in order to compare the provided password
		u, err := user.ByEmailOrUsername(ctx, db, identifier)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}

		match, err := user.Verify(password, u.Hash, u.Salt)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication error"})
			return
		}
		if !match {
			logger.Warn("invalid password attempt")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid credentials"})
			return
		}

		s, err := session.Create(ctx, db, u.ID)
		if err != nil {
			logger.Error("failed to create session", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create session"})
			return
		}
		session.SetSessionCookie(w, s.SessionID)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(s)
	}
}

func Logout(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(session.CookieName)
		if err != nil || cookie.Value == "" {
			// Don't return an error if the cookie is missing or empty, just treat it as a successful logout
			logger.Info("no session cookie found, treating as successful logout")
		} else {
			err = session.Delete(r.Context(), db, cookie.Value)
			if err != nil {
				// Don't return an error if session deletion fails, just log it and continue with clearing the cookie
				logger.Error("failed to delete session", "error", err)
			}
		}

		session.ClearSessionCookie(w)
		w.WriteHeader(http.StatusNoContent)
	}
}

func ForgotPassword(
	logger *slog.Logger,
	db *sqlx.DB,
	emailClient *email.Client,
	signingKey string,
) http.HandlerFunc {
	type Request struct {
		Email string `json:"email"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			logger.Warn("failed to decode forgot-password request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}
		if request.Email == "" {
			logger.Warn("email is required")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email is required"})
			return
		}

		u, err := user.ByEmailOrUsername(ctx, db, request.Email)
		if err != nil {
			// Do not leak whether the email exists or not
			logger.Info("forgot password requested for non-existent email", "email", request.Email)
		} else {
			// Generate token
			t, err := token.Create(signingKey, u.Username, u.Email)
			if err != nil {
				logger.Error("failed to create forgot-password token", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "Failed to process request"})
				return
			}

			// Send email
			err = emailClient.SendPasswordReset(
				u.Email,
				u.Username,
				emailClient.PasswordResetURL(t),
			)
			if err != nil {
				logger.Error("failed to send password reset email", "error", err)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).
					Encode(map[string]string{"error": "Failed to send password reset email"})
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "If an account with that email exists, we have sent a password reset link.",
		})
	}
}

func ResetPassword(logger *slog.Logger, db *sqlx.DB, signingKey string) http.HandlerFunc {
	type Request struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var request Request
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&request)
		if err != nil {
			logger.Warn("failed to decode reset-password request body", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}
		if request.Token == "" || request.Password == "" {
			logger.Warn("token and password are required")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Token and password are required"})
			return
		}

		// Validate token
		claims, err := token.Verify(signingKey, request.Token)
		if err != nil {
			logger.Warn("reset-password token verification failed", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Invalid or expired reset token"})
			return
		}

		u, err := user.ByEmailOrUsername(ctx, db, claims.Email)
		if err != nil {
			logger.Warn("user not found from token claims", "email", claims.Email)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Invalid token payload"})
			return
		}

		err = user.UpdatePassword(ctx, db, u.ID, request.Password)
		if err != nil {
			logger.Error("failed to update password", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "Failed to update password"})
			return
		}

		logger.Info("password reset successfully", "user_id", u.ID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Password updated successfully"})
	}
}
