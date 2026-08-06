package test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"moria/internal/session"
	"moria/test/seeding"
)

func TestMe(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AliceToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if strings.Contains(string(raw), `"password_hash"`) || strings.Contains(string(raw), `"salt"`) {
		t.Errorf("body leaks credentials: %s", raw)
	}

	var got struct {
		User struct {
			UserID   string `json:"user_id"`
			Username string `json:"username"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		} `json:"user"`
		Session struct {
			SessionID string `json:"session_id"`
			UserID    string `json:"user_id"`
			ExpiresAt string `json:"expires_at"`
		} `json:"session"`
	}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}

	if got.User.UserID != seeding.AliceID {
		t.Errorf("user.user_id = %q, want %q", got.User.UserID, seeding.AliceID)
	}
	if got.User.Username != "alice" {
		t.Errorf("user.username = %q, want %q", got.User.Username, "alice")
	}
	if got.User.Email != "alice@example.com" {
		t.Errorf("user.email = %q, want %q", got.User.Email, "alice@example.com")
	}
	if got.User.Role != "user" {
		t.Errorf("user.role = %q, want %q", got.User.Role, "user")
	}
	if got.Session.SessionID != string(session.Token(seeding.AliceToken).ID()) {
		t.Errorf(
			"session.session_id = %q, want %q",
			got.Session.SessionID,
			session.Token(seeding.AliceToken).ID(),
		)
	}
	if got.Session.UserID != seeding.AliceID {
		t.Errorf("session.user_id = %q, want %q", got.Session.UserID, seeding.AliceID)
	}
	if got.Session.ExpiresAt == "" {
		t.Error("session.expires_at is empty")
	}
}

func TestMeWithoutValidSessionIs401(t *testing.T) {
	cases := []struct {
		name, token, wantError string
	}{
		{"no cookie", "", "Authentication required"},
		{"unknown token", "no-such-token", "Invalid or expired session"},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/me", nil)
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
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want %d", tt.name, resp.StatusCode, http.StatusUnauthorized)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.name, body["error"], tt.wantError)
		}
	}
}
