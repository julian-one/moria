package route

import (
	"encoding/json"
	"net/http"

	"moria/internal/session"
	"moria/internal/user"
)

func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rq := session.RequesterFrom(r.Context())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(struct {
			User    *user.User       `json:"user"`
			Session *session.Session `json:"session"`
		}{&rq.User, &rq.Session})
	}
}
