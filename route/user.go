package route

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"moria/internal/session"
	"moria/internal/user"
)

func ListUsers(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := user.List(r.Context(), db)
		if err != nil {
			slog.Error("failed to list users", "error", err)
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
		}{
			users,
			len(users),
		})
	}
}

func CreateUser(db *sql.DB) http.HandlerFunc {
	type Request struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if json.NewDecoder(r.Body).Decode(&req) != nil ||
			req.Username == "" || req.Email == "" || req.Password == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}
		if req.Role == "" {
			req.Role = string(user.RoleUser)
		}
		if !user.Role(req.Role).Valid() {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role provided"})
			return
		}

		taken, err := user.IsUsernameTaken(r.Context(), db, req.Username, "")
		if err != nil {
			slog.Error("failed to check username", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check username"})
			return
		}
		if taken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Username is already taken"})
			return
		}

		taken, err = user.IsEmailTaken(r.Context(), db, req.Email, "")
		if err != nil {
			slog.Error("failed to check email", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to check email"})
			return
		}
		if taken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"error": "Email is already taken"})
			return
		}

		u, err := user.Create(r.Context(), db, req.Username, req.Email, req.Password, user.Role(req.Role))
		if err != nil {
			slog.Error("failed to create user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to create user"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)
	}
}

func GetUser(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := user.ByID(r.Context(), db, r.PathValue("id"))
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}
		if err != nil {
			slog.Error("failed to get user", "error", err)
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

func UpdateUser(db *sql.DB) http.HandlerFunc {
	type Request struct {
		Username string `json:"username"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Username == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
			return
		}

		id := r.PathValue("id")
		s, ok := r.Context().Value(session.ContextKey).(*session.Session)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Authentication required"})
			return
		}
		if s.UserID != id {
			requester, err := user.ByID(r.Context(), db, s.UserID)
			if err != nil || requester.Role != user.RoleAdmin {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]string{"error": "You can only update your own username"})
				return
			}
		}

		taken, err := user.IsUsernameTaken(r.Context(), db, req.Username, id)
		if err != nil {
			slog.Error("failed to check username", "error", err)
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

		u, err := user.UpdateUsername(r.Context(), db, id, req.Username)
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}
		if err != nil {
			slog.Error("failed to update user", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update user"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(u)
	}
}

func UpdatePassword(db *sql.DB) http.HandlerFunc {
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

		id := r.PathValue("id")
		s, ok := r.Context().Value(session.ContextKey).(*session.Session)
		if !ok || s.UserID != id {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"error": "You can only update your own password"})
			return
		}

		err := user.UpdatePassword(r.Context(), db, id, req.NewPassword)
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}
		if err != nil {
			slog.Error("failed to update password", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update password"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func UpdateUserRole(db *sql.DB) http.HandlerFunc {
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
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid role provided"})
			return
		}

		err := user.UpdateRole(r.Context(), db, r.PathValue("id"), user.Role(req.Role))
		if errors.Is(err, sql.ErrNoRows) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "User not found"})
			return
		}
		if err != nil {
			slog.Error("failed to update role", "error", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "Failed to update role"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
