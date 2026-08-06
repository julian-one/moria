package test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"moria/internal/session"
	"moria/internal/user"
	"moria/test/seeding"
)

// Unrouted paths and wrong methods never reach the middleware chains — they
// fall through to ServeMux itself: plain text, not the JSON error envelope.
func TestUnroutedAndWrongMethodArePlainText(t *testing.T) {
	cases := []struct {
		method, path string
		status       int
		body         string
	}{
		{http.MethodGet, "/nope", http.StatusNotFound, "404 page not found\n"},
		{
			http.MethodDelete,
			"/users/" + seeding.AliceID,
			http.StatusMethodNotAllowed,
			"Method Not Allowed\n",
		},
		{http.MethodGet, "/login", http.StatusMethodNotAllowed, "Method Not Allowed\n"},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tt.method, tt.path, err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != tt.status {
			t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, resp.StatusCode, tt.status)
		}
		if string(raw) != tt.body {
			t.Errorf("%s %s: body = %q, want %q", tt.method, tt.path, raw, tt.body)
		}
		if ct := resp.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
			t.Errorf(
				"%s %s: content type = %q, want %q",
				tt.method,
				tt.path,
				ct,
				"text/plain; charset=utf-8",
			)
		}
	}
}

// Every body — resource or {"error": ...} — is application/json; the 204
// routes carry no body and no content type.
func TestContentTypeHeader(t *testing.T) {
	if err := seeding.User(db, "ct-id", "ct-user", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "ct-token", "ct-id"); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name, method, path, token string
		login                     bool
		status                    int
		contentType               string
	}{
		{"health", http.MethodGet, "/health", "", false, http.StatusOK, "application/json"},
		{"login", http.MethodPost, "/login", "", true, http.StatusOK, "application/json"},
		{
			"me",
			http.MethodGet,
			"/me",
			"ct-token",
			false,
			http.StatusOK,
			"application/json",
		},
		{
			"error body",
			http.MethodGet,
			"/users/ct-id",
			"",
			false,
			http.StatusUnauthorized,
			"application/json",
		},
		{"logout 204", http.MethodPost, "/logout", "", false, http.StatusNoContent, ""},
		{
			"delete 204",
			http.MethodDelete,
			"/sessions/no-such-session",
			seeding.AdminToken,
			false,
			http.StatusNoContent,
			"",
		},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(tt.method, srv.URL+tt.path, strings.NewReader(""))
		if err != nil {
			t.Fatal(err)
		}
		if tt.login {
			req.SetBasicAuth("ct-user", seeding.DefaultPassword)
		}
		if tt.token != "" {
			req.AddCookie(&http.Cookie{Name: session.CookieName, Value: tt.token})
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		resp.Body.Close()

		if resp.StatusCode != tt.status {
			t.Errorf("%s: status = %d, want %d", tt.name, resp.StatusCode, tt.status)
		}
		if ct := resp.Header.Get("Content-Type"); ct != tt.contentType {
			t.Errorf("%s: content type = %q, want %q", tt.name, ct, tt.contentType)
		}
	}
}
