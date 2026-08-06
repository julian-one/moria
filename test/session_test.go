package test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"moria/internal/session"
	"moria/internal/user"
	"moria/test/seeding"
)

func TestSessionRoutesRequireAuthentication(t *testing.T) {
	routes := []struct {
		method, path string
	}{
		{http.MethodGet, "/sessions/no-such-session"},
		{http.MethodDelete, "/sessions/no-such-session"},
		{http.MethodGet, "/users/no-such-id/sessions"},
		{http.MethodDelete, "/users/no-such-id/sessions"},
	}
	for _, tt := range routes {
		req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tt.method, tt.path, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf(
				"%s %s: status = %d, want %d",
				tt.method,
				tt.path,
				resp.StatusCode,
				http.StatusUnauthorized,
			)
		}
		if body["error"] != "Authentication required" {
			t.Errorf(
				"%s %s: error = %q, want %q",
				tt.method,
				tt.path,
				body["error"],
				"Authentication required",
			)
		}
	}
}

func TestSessionAdminRoutesForbidNonAdmin(t *testing.T) {
	routes := []struct {
		method, path string
	}{
		{http.MethodGet, "/sessions/no-such-session"},
		{http.MethodDelete, "/sessions/no-such-session"},
		{http.MethodGet, "/users/" + seeding.AliceID + "/sessions"},
		{http.MethodDelete, "/users/no-such-id/sessions"},
	}
	for _, tt := range routes {
		req, err := http.NewRequest(tt.method, srv.URL+tt.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AliceToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", tt.method, tt.path, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf(
				"%s %s: status = %d, want %d",
				tt.method,
				tt.path,
				resp.StatusCode,
				http.StatusForbidden,
			)
		}
		if body["error"] != "Forbidden: admin access required" {
			t.Errorf(
				"%s %s: error = %q, want %q",
				tt.method,
				tt.path,
				body["error"],
				"Forbidden: admin access required",
			)
		}
	}
}

func TestGetSession(t *testing.T) {
	cases := []struct {
		token, userID string
	}{
		{seeding.AliceToken, seeding.AliceID},
		{seeding.AdminToken, seeding.AdminID},
	}
	for _, tt := range cases {
		sessionID := string(session.Token(tt.token).ID())
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/sessions/"+sessionID, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.token, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("%s: status = %d, want %d", tt.token, resp.StatusCode, http.StatusOK)
		}
		var got struct {
			SessionID string `json:"session_id"`
			UserID    string `json:"user_id"`
			ExpiresAt string `json:"expires_at"`
			CreatedAt string `json:"created_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("%s: %v", tt.token, err)
		}
		resp.Body.Close()

		if got.SessionID != sessionID {
			t.Errorf("%s: session_id = %q, want %q", tt.token, got.SessionID, sessionID)
		}
		if got.UserID != tt.userID {
			t.Errorf("%s: user_id = %q, want %q", tt.token, got.UserID, tt.userID)
		}
		if got.ExpiresAt == "" || got.CreatedAt == "" {
			t.Errorf(
				"%s: timestamps = %q/%q, want non-empty",
				tt.token,
				got.ExpiresAt,
				got.CreatedAt,
			)
		}
	}
}

func TestGetSessionUnknownIs404(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/sessions/no-such-session", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "Session not found" {
		t.Errorf("error = %q, want %q", body["error"], "Session not found")
	}
}

func TestGetSessionExpiredIs404(t *testing.T) {
	if err := seeding.ExpiredSession(db, "sess-expired-token", seeding.AliceID); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(
		http.MethodGet,
		srv.URL+"/sessions/"+string(session.Token("sess-expired-token").ID()),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "Session not found" {
		t.Errorf("error = %q, want %q", body["error"], "Session not found")
	}
}

func TestListSessions(t *testing.T) {
	if err := seeding.User(db, "sess-list-id", "sess-list", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "sess-list-live-token", "sess-list-id"); err != nil {
		t.Fatal(err)
	}
	if err := seeding.ExpiredSession(db, "sess-list-expired-token", "sess-list-id"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/users/sess-list-id/sessions", nil)
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
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	var body struct {
		Items []struct {
			SessionID string `json:"session_id"`
			UserID    string `json:"user_id"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if body.Total != 2 || len(body.Items) != 2 {
		t.Fatalf("total/len = %d/%d, want 2/2", body.Total, len(body.Items))
	}
	if body.Items[0].SessionID != string(session.Token("sess-list-live-token").ID()) {
		t.Errorf(
			"items[0].session_id = %q, want %q",
			body.Items[0].SessionID,
			string(session.Token("sess-list-live-token").ID()),
		)
	}
	if body.Items[1].SessionID != string(session.Token("sess-list-expired-token").ID()) {
		t.Errorf(
			"items[1].session_id = %q, want %q",
			body.Items[1].SessionID,
			string(session.Token("sess-list-expired-token").ID()),
		)
	}
	for _, item := range body.Items {
		if item.UserID != "sess-list-id" {
			t.Errorf("%s: user_id = %q, want %q", item.SessionID, item.UserID, "sess-list-id")
		}
	}
}

func TestListSessionsUnknownUserIsEmpty(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/users/no-such-id/sessions", nil)
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
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
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

func TestDeleteSession(t *testing.T) {
	if err := seeding.User(db, "sess-del-id", "sess-del", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "sess-del-token", "sess-del-id"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(
		http.MethodDelete,
		srv.URL+"/sessions/"+string(session.Token("sess-del-token").ID()),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
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

	req, err = http.NewRequest(
		http.MethodGet,
		srv.URL+"/sessions/"+string(session.Token("sess-del-token").ID()),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("GET deleted session status = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	req, err = http.NewRequest(http.MethodGet, srv.URL+"/users/sess-del-id", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "sess-del-token"})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var errBody map[string]string
	json.NewDecoder(resp.Body).Decode(&errBody)
	resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("revoked token status = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
	if errBody["error"] != "Invalid or expired session" {
		t.Errorf(
			"revoked token error = %q, want %q",
			errBody["error"],
			"Invalid or expired session",
		)
	}
}

func TestDeleteSessionUnknownIs204(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/sessions/no-such-session", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}

func TestDeleteAllSessions(t *testing.T) {
	if err := seeding.User(db, "sess-purge-id", "sess-purge", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "sess-purge-token-1", "sess-purge-id"); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "sess-purge-token-2", "sess-purge-id"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/users/sess-purge-id/sessions", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
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

	for _, token := range []string{"sess-purge-token-1", "sess-purge-token-2"} {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/sessions/"+string(session.Token(token).ID()), nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", token, err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status = %d, want %d", token, resp.StatusCode, http.StatusNotFound)
		}
	}
}

func TestDeleteAllSessionsUnknownUserIs204(t *testing.T) {
	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/users/no-such-id/sessions", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}
}
