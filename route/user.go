package route

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"moria/internal/session"
	"moria/internal/user"

	"github.com/jmoiron/sqlx"
)

func ListUsers(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		opts, err := user.ParseListOptions(r)
		if err != nil {
			logger.Warn("failed to parse user list options", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request"})
			return
		}

		users, err := user.List(ctx, db, opts)
		if err != nil {
			logger.Error("failed to list users", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list users"})
			return
		}

		total, err := user.Count(ctx, db, opts)
		if err != nil {
			logger.Error("failed to count users", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to list users"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			Items []user.User `json:"items"`
			Total int         `json:"total"`
		}{users, total})
	}
}

func GetUser(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		userID := r.PathValue("id")
		u, err := user.ByID(ctx, db, userID)
		if err != nil {
			logger.Error("failed to get user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to get user"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(u)
	}
}

func UpdateUser(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	type Request struct {
		Username *string `json:"username,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		ctx := r.Context()
		taken, err := user.IsUsernameTaken(ctx, db, *req.Username)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check username"})
			return
		}
		if taken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Username is already taken"})
			return
		}

		s, ok := ctx.Value(session.ContextKey).(*session.Session)
		if !ok || s == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
			return
		}

		id := r.PathValue("id")
		if s.User != id {
			var u *user.User
			u, err = user.ByID(ctx, db, s.User)
			if err != nil || u.Role != user.RoleAdmin {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).
					Encode(map[string]string{"error": "You can only update your own username"})
				return
			}
		}
		u, err := user.Update(ctx, db, id, req.Username, nil)
		if err != nil {
			logger.Error("failed to update user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update user role"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(u)
	}
}

func UpdateUserRole(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	type Request struct {
		Role string `json:"role"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		if !user.Role(req.Role).Valid() {
			logger.Warn("invalid role provided", "role", req.Role)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role provided"})
			return
		}

		ctx := r.Context()
		id := r.PathValue("id")
		_, err := user.Update(ctx, db, id, nil, &req.Role)
		if err != nil {
			logger.Error("failed to update user role", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update user role"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func UpdatePassword(logger *slog.Logger, db *sqlx.DB) http.HandlerFunc {
	type Request struct {
		NewPassword string `json:"new_password"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.NewPassword == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		ctx := r.Context()
		s, ok := ctx.Value(session.ContextKey).(*session.Session)
		if !ok || s == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
			return
		}

		id := r.PathValue("id")
		if s.User != id {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).
				Encode(map[string]string{"error": "You can only update your own password"})
			return
		}

		if err := user.UpdatePassword(ctx, db, id, req.NewPassword); err != nil {
			logger.Error("failed to update password", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update password"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
