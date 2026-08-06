package test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"moria/internal/session"
	"moria/internal/user"
	"moria/route"
	"moria/test/seeding"
)

// Renaming users breaks the requester join, so a users-table fault fails
// closed at the Authentication seam: every protected and admin route answers
// 500 before reaching its handler — distinct from the 401 a bad token gets,
// so shire keeps the cookie — and only login (base tier) reaches its own 500.
func TestUserTableFailuresFailClosed(t *testing.T) {
	fdb := newTestDB(t)
	if err := seeding.User(fdb, "fr-admin-id", "fr-admin", user.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := seeding.User(fdb, "fr-user-id", "fr-user", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(fdb, "fr-admin-token", "fr-admin-id"); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(fdb, "fr-user-token", "fr-user-id"); err != nil {
		t.Fatal(err)
	}
	fsrv := httptest.NewServer(route.Initialize(route.Config{DB: fdb}))
	defer fsrv.Close()

	if _, err := fdb.Exec(`ALTER TABLE users RENAME TO users_gone`); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, method, path, token, body string
		login                           bool
		status                          int
		wantError                       string
	}{
		{
			"login", http.MethodPost, "/login", "", "", true,
			http.StatusInternalServerError, "Authentication error",
		},
		{
			"me", http.MethodGet, "/me", "fr-user-token", "", false,
			http.StatusInternalServerError, "Authentication error",
		},
		{
			"rename self", http.MethodPatch, "/users/fr-user-id", "fr-user-token", `{"username":"fr-user-2"}`, false,
			http.StatusInternalServerError, "Authentication error",
		},
		{
			"update password", http.MethodPatch, "/users/fr-user-id/password", "fr-user-token", `{"new_password":"x"}`, false,
			http.StatusInternalServerError, "Authentication error",
		},
		{
			"rename other", http.MethodPatch, "/users/fr-admin-id", "fr-user-token", `{"username":"hijack"}`, false,
			http.StatusInternalServerError, "Authentication error",
		},
		{
			"admin route", http.MethodGet, "/users", "fr-admin-token", "", false,
			http.StatusInternalServerError, "Authentication error",
		},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(tt.method, fsrv.URL+tt.path, strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		if tt.login {
			req.SetBasicAuth("fr-user", seeding.DefaultPassword)
		}
		if tt.token != "" {
			req.AddCookie(&http.Cookie{Name: session.CookieName, Value: tt.token})
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != tt.status {
			t.Errorf("%s: status = %d, want %d", tt.name, resp.StatusCode, tt.status)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.name, body["error"], tt.wantError)
		}
	}
}

// Renaming sessions makes every session query fail while users stay intact:
// Authentication answers a database failure with 500 (fail closed, but
// distinct from a bad token's 401 so clients keep their cookie), login fails
// only at session creation, and logout's best-effort 204 survives a real
// delete failure — not just a missing cookie.
func TestSessionFailuresFailClosed(t *testing.T) {
	fdb := newTestDB(t)
	if err := seeding.User(fdb, "fs-user-id", "fs-user", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(fdb, "fs-user-token", "fs-user-id"); err != nil {
		t.Fatal(err)
	}
	fsrv := httptest.NewServer(route.Initialize(route.Config{DB: fdb}))
	defer fsrv.Close()

	if _, err := fdb.Exec(`ALTER TABLE sessions RENAME TO sessions_gone`); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, fsrv.URL+"/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "fs-user-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf(
			"auth on broken sessions: status = %d, want %d",
			resp.StatusCode,
			http.StatusInternalServerError,
		)
	}
	if body["error"] != "Authentication error" {
		t.Errorf(
			"auth on broken sessions: error = %q, want %q",
			body["error"],
			"Authentication error",
		)
	}

	req, err = http.NewRequest(http.MethodPost, fsrv.URL+"/login", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.SetBasicAuth("fs-user", seeding.DefaultPassword)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body = nil
	json.NewDecoder(resp.Body).Decode(&body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("login: status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if body["error"] != "Failed to create session" {
		t.Errorf("login: error = %q, want %q", body["error"], "Failed to create session")
	}

	req, err = http.NewRequest(http.MethodPost, fsrv.URL+"/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "fs-user-token"})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("logout: status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if len(raw) != 0 {
		t.Errorf("logout: body = %q, want empty", raw)
	}
}

// Widening a table breaks every SELECT * struct-scan (sqlx rejects the
// unmapped column) while the requester join's explicit columns keep
// authentication alive — and /me with it, since /me is served entirely from
// that join. The only black-box fault that separates handler reads from
// middleware reads, so the read-only 500s stay pinned.
func TestWidenedTableFailsHandlerReads(t *testing.T) {
	fdb := newTestDB(t)
	if err := seeding.User(fdb, "fj-admin-id", "fj-admin", user.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(fdb, "fj-admin-token", "fj-admin-id"); err != nil {
		t.Fatal(err)
	}
	fsrv := httptest.NewServer(route.Initialize(route.Config{DB: fdb}))
	defer fsrv.Close()

	if _, err := fdb.Exec(`ALTER TABLE users ADD COLUMN junk TEXT`); err != nil {
		t.Fatal(err)
	}
	if _, err := fdb.Exec(`ALTER TABLE sessions ADD COLUMN junk TEXT`); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, path, wantError string
	}{
		{"get user", "/users/fj-admin-id", "Failed to get user"},
		{"list users", "/users", "Failed to list users"},
		{"get session", "/sessions/" + string(session.Token("fj-admin-token").ID()), "Failed to get session"},
		{"list sessions", "/users/fj-admin-id/sessions", "Failed to list sessions"},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodGet, fsrv.URL+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "fj-admin-token"})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf(
				"%s: status = %d, want %d",
				tt.name,
				resp.StatusCode,
				http.StatusInternalServerError,
			)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.name, body["error"], tt.wantError)
		}
	}

	req, err := http.NewRequest(http.MethodGet, fsrv.URL+"/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "fj-admin-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var me struct {
		User struct {
			UserID string `json:"user_id"`
		} `json:"user"`
	}
	json.NewDecoder(resp.Body).Decode(&me)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("me on widened tables: status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if me.User.UserID != "fj-admin-id" {
		t.Errorf("me on widened tables: user_id = %q, want %q", me.User.UserID, "fj-admin-id")
	}
}

func TestRevokedWritesSurfaceAs500(t *testing.T) {
	fdb := newTestDB(t)
	if err := seeding.User(fdb, "fw-admin-id", "fw-admin", user.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	if err := seeding.User(fdb, "fw-user-id", "fw-user", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(fdb, "fw-admin-token", "fw-admin-id"); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(fdb, "fw-user-token", "fw-user-id"); err != nil {
		t.Fatal(err)
	}
	fsrv := httptest.NewServer(route.Initialize(route.Config{DB: fdb}))
	defer fsrv.Close()

	if _, err := fdb.Exec(
		`REVOKE INSERT, UPDATE, DELETE ON users, sessions FROM CURRENT_USER`,
	); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, method, path, token, body, wantError string
	}{
		{
			"create user", http.MethodPost, "/users", "fw-admin-token",
			`{"username":"fw-new","email":"fw-new@example.com","password":"p"}`, "Failed to create user",
		},
		{
			"rename", http.MethodPatch, "/users/fw-user-id", "fw-admin-token",
			`{"username":"fw-user-2"}`, "Failed to update user",
		},
		{
			"password", http.MethodPatch, "/users/fw-user-id/password", "fw-user-token",
			`{"new_password":"x"}`, "Failed to update password",
		},
		{
			"role", http.MethodPatch, "/users/fw-user-id/role", "fw-admin-token",
			`{"role":"admin"}`, "Failed to update role",
		},
		{
			"delete session",
			http.MethodDelete,
			"/sessions/" + string(session.Token("fw-user-token").ID()),
			"fw-admin-token",
			"",
			"Failed to delete session",
		},
		{
			"delete all sessions",
			http.MethodDelete,
			"/users/fw-user-id/sessions",
			"fw-admin-token",
			"",
			"Failed to delete all sessions",
		},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(tt.method, fsrv.URL+tt.path, strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: tt.token})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf(
				"%s: status = %d, want %d",
				tt.name,
				resp.StatusCode,
				http.StatusInternalServerError,
			)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.name, body["error"], tt.wantError)
		}
	}
}
