package test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"moria/internal/database"
	"moria/internal/session"
	"moria/internal/user"
	"moria/test/seeding"
)

// Deleting a user has no API route, so nothing else in the suite ever
// fires the cascade.
func TestUserDeleteCascadesSessions(t *testing.T) {
	if err := seeding.User(db, "casc-id", "casc", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "casc-token", "casc-id"); err != nil {
		t.Fatal(err)
	}

	listTotal := func(t *testing.T) int {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/users/casc-id/sessions", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("list status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		var body struct {
			Total int `json:"total"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		return body.Total
	}

	if total := listTotal(t); total != 1 {
		t.Fatalf("sessions before delete = %d, want 1", total)
	}

	if _, err := db.Exec(`DELETE FROM users WHERE user_id = $1`, "casc-id"); err != nil {
		t.Fatal(err)
	}

	if total := listTotal(t); total != 0 {
		t.Errorf("sessions after user delete = %d, want 0 (ON DELETE CASCADE)", total)
	}
}

func TestDatabaseNewRejectsBadURLs(t *testing.T) {
	if fdb, err := database.New(context.Background(), "://nope"); err == nil {
		fdb.Close()
		t.Error("malformed url: err = nil, want error")
	}

	if fdb, err := database.New(context.Background(), "postgres://moria:x@127.0.0.1:1/moria"); err == nil {
		fdb.Close()
		t.Error("unreachable server: err = nil, want error")
	}
}
