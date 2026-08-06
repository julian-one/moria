package test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"moria/internal/session"
	"moria/internal/user"
	"moria/test/seeding"
)

func TestLogin(t *testing.T) {
	if err := seeding.User(db, "auth-login-id", "auth-login", user.RoleUser); err != nil {
		t.Fatal(err)
	}

	for _, identifier := range []string{"auth-login", "auth-login@example.com"} {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/login", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.SetBasicAuth(identifier, seeding.DefaultPassword)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", identifier, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("%s: status = %d, want %d", identifier, resp.StatusCode, http.StatusOK)
		}
		var got struct {
			Token     string `json:"token"`
			ExpiresAt string `json:"expires_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("%s: %v", identifier, err)
		}
		resp.Body.Close()

		if len(got.Token) != 64 {
			t.Errorf("%s: token = %q, want 64 hex chars", identifier, got.Token)
		}
		expires, err := time.Parse(time.RFC3339, got.ExpiresAt)
		if err != nil {
			t.Fatalf("%s: expires_at = %q, want RFC3339: %v", identifier, got.ExpiresAt, err)
		}
		if !strings.HasSuffix(got.ExpiresAt, "Z") {
			t.Errorf("%s: expires_at = %q, want UTC Z suffix", identifier, got.ExpiresAt)
		}
		if lifetime := time.Until(expires); lifetime < session.Duration-2*time.Second ||
			lifetime > session.Duration+2*time.Second {
			t.Errorf("%s: session lifetime = %v, want %v", identifier, lifetime, session.Duration)
		}

		req, err = http.NewRequest(http.MethodGet, srv.URL+"/me", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: got.Token})
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", identifier, err)
		}
		var me struct {
			User struct {
				UserID string `json:"user_id"`
			} `json:"user"`
			Session struct {
				SessionID string `json:"session_id"`
			} `json:"session"`
		}
		json.NewDecoder(resp.Body).Decode(&me)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf(
				"%s: GET /me with new session status = %d, want %d",
				identifier,
				resp.StatusCode,
				http.StatusOK,
			)
		}
		if me.User.UserID != "auth-login-id" {
			t.Errorf("%s: /me user_id = %q, want %q", identifier, me.User.UserID, "auth-login-id")
		}
		if me.Session.SessionID != string(session.Token(got.Token).ID()) {
			t.Errorf(
				"%s: /me session_id = %q, want the stored digest %q",
				identifier,
				me.Session.SessionID,
				session.Token(got.Token).ID(),
			)
		}
		if me.Session.SessionID == got.Token {
			t.Errorf("%s: stored session_id equals the raw token", identifier)
		}
	}
}

func TestLoginMalformedBasicAuthIs401(t *testing.T) {
	cases := []struct {
		name, header string
	}{
		{"no header", ""},
		{"bearer scheme", "Bearer " + seeding.AliceToken},
		{"invalid base64", "Basic %%%"},
		{"no colon", "Basic bm9jb2xvbg=="},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/login", nil)
		if err != nil {
			t.Fatal(err)
		}
		if tt.header != "" {
			req.Header.Set("Authorization", tt.header)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want %d", tt.name, resp.StatusCode, http.StatusUnauthorized)
		}
		if body["error"] != "Invalid basic auth credentials" {
			t.Errorf(
				"%s: error = %q, want %q",
				tt.name,
				body["error"],
				"Invalid basic auth credentials",
			)
		}
	}
}

// Unknown identifier and wrong password get one indistinguishable answer —
// the enumeration-resistance contract.
func TestLoginInvalidCredentialsIs401(t *testing.T) {
	cases := []struct {
		identifier, password string
	}{
		{"no-such-user", seeding.DefaultPassword},
		{"alice", "wrong-password"},
		{"alice@example.com", "wrong-password"},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/login", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.SetBasicAuth(tt.identifier, tt.password)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.identifier, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf(
				"%s: status = %d, want %d",
				tt.identifier,
				resp.StatusCode,
				http.StatusUnauthorized,
			)
		}
		if body["error"] != "Invalid credentials" {
			t.Errorf("%s: error = %q, want %q", tt.identifier, body["error"], "Invalid credentials")
		}
	}
}

func TestLogout(t *testing.T) {
	if err := seeding.User(db, "auth-logout-id", "auth-logout", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "auth-logout-token", "auth-logout-id"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "auth-logout-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
	if len(body) != 0 {
		t.Errorf("body = %q, want empty", body)
	}

	req, err = http.NewRequest(http.MethodGet, srv.URL+"/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "auth-logout-token"})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var errBody map[string]string
	json.NewDecoder(resp.Body).Decode(&errBody)
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("logged-out token status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if errBody["error"] != "Invalid or expired session" {
		t.Errorf(
			"logged-out token error = %q, want %q",
			errBody["error"],
			"Invalid or expired session",
		)
	}
}

// Logout is best-effort by contract: a missing or unknown cookie still
// succeeds, so shire can always shed the browser cookie.
func TestLogoutMissingOrUnknownTokenIs204(t *testing.T) {
	cases := []struct {
		name, token string
	}{
		{"no cookie", ""},
		{"unknown token", "no-such-token"},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/logout", nil)
		if err != nil {
			t.Fatal(err)
		}
		if tt.token != "" {
			req.AddCookie(&http.Cookie{Name: session.CookieName, Value: tt.token})
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("%s: status = %d, want %d", tt.name, resp.StatusCode, http.StatusNoContent)
		}
		if len(body) != 0 {
			t.Errorf("%s: body = %q, want empty", tt.name, body)
		}
	}
}

// Logout sits on the base chain, not protected — an already-expired session
// must still be able to log out. List has no expiry filter, which is what
// makes the deletion observable.
func TestLogoutExpiredSessionIs204AndDeletesRow(t *testing.T) {
	if err := seeding.User(db, "auth-logout-exp-id", "auth-logout-exp", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.ExpiredSession(
		db,
		"auth-logout-exp-token",
		"auth-logout-exp-id",
	); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/logout", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "auth-logout-exp-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	req, err = http.NewRequest(http.MethodGet, srv.URL+"/users/auth-logout-exp-id/sessions", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body struct {
		Items []struct{} `json:"items"`
		Total int        `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Total != 0 || len(body.Items) != 0 {
		t.Errorf("total/len = %d/%d, want 0/0", body.Total, len(body.Items))
	}
}
