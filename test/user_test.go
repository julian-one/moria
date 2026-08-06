package test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"moria/internal/session"
	"moria/internal/user"
	"moria/test/seeding"
)

func TestUserRoutesRequireAuthentication(t *testing.T) {
	routes := []struct {
		method, path string
	}{
		{http.MethodPost, "/users"},
		{http.MethodGet, "/users"},
		{http.MethodGet, "/users/" + seeding.AliceID},
		{http.MethodPatch, "/users/" + seeding.AliceID},
		{http.MethodPatch, "/users/" + seeding.AliceID + "/password"},
		{http.MethodPatch, "/users/" + seeding.AliceID + "/role"},
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

func TestExpiredOrUnknownSessionIs401(t *testing.T) {
	if err := seeding.ExpiredSession(db, "stale-token", seeding.AliceID); err != nil {
		t.Fatal(err)
	}
	for _, token := range []string{"stale-token", "no-such-token"} {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/users/"+seeding.AliceID, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: token})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", token, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s: status = %d, want %d", token, resp.StatusCode, http.StatusUnauthorized)
		}
		if body["error"] != "Invalid or expired session" {
			t.Errorf("%s: error = %q, want %q", token, body["error"], "Invalid or expired session")
		}
	}
}

func TestAdminRoutesForbidNonAdmin(t *testing.T) {
	routes := []struct {
		method, path string
	}{
		{http.MethodPost, "/users"},
		{http.MethodGet, "/users"},
		{http.MethodGet, "/users/" + seeding.AliceID},
		{http.MethodPatch, "/users/" + seeding.AliceID + "/role"},
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

func TestCreateUser(t *testing.T) {
	cases := []struct {
		body, username, email, role string
	}{
		{
			`{"username":"bob","email":"bob@example.com","password":"bob-password","role":"admin"}`,
			"bob",
			"bob@example.com",
			"admin",
		},
		{
			`{"username":"carol","email":"carol@example.com","password":"carol-password"}`,
			"carol",
			"carol@example.com",
			"user",
		},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/users", strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.username, err)
		}
		if resp.StatusCode != http.StatusCreated {
			resp.Body.Close()
			t.Fatalf("%s: status = %d, want %d", tt.username, resp.StatusCode, http.StatusCreated)
		}
		var created struct {
			UserID    string `json:"user_id"`
			Username  string `json:"username"`
			Email     string `json:"email"`
			Role      string `json:"role"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
			t.Fatalf("%s: %v", tt.username, err)
		}
		resp.Body.Close()

		if created.UserID == "" {
			t.Errorf("%s: user_id is empty", tt.username)
		}
		if created.Username != tt.username {
			t.Errorf("%s: username = %q, want %q", tt.username, created.Username, tt.username)
		}
		if created.Email != tt.email {
			t.Errorf("%s: email = %q, want %q", tt.username, created.Email, tt.email)
		}
		if created.Role != tt.role {
			t.Errorf("%s: role = %q, want %q", tt.username, created.Role, tt.role)
		}
		if created.CreatedAt == "" || created.UpdatedAt == "" {
			t.Errorf(
				"%s: timestamps = %q/%q, want non-empty",
				tt.username,
				created.CreatedAt,
				created.UpdatedAt,
			)
		}

		req, err = http.NewRequest(http.MethodGet, srv.URL+"/users/"+created.UserID, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.username, err)
		}
		var fetched struct {
			Username string `json:"username"`
		}
		json.NewDecoder(resp.Body).Decode(&fetched)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf(
				"%s: follow-up GET status = %d, want %d",
				tt.username,
				resp.StatusCode,
				http.StatusOK,
			)
		}
		if fetched.Username != tt.username {
			t.Errorf(
				"%s: follow-up GET username = %q, want %q",
				tt.username,
				fetched.Username,
				tt.username,
			)
		}
	}
}

func TestCreateUserInvalidBodyIs400(t *testing.T) {
	cases := []struct {
		body, wantError string
	}{
		{`not json`, "Invalid request body"},
		{`{"email":"missing-username@example.com","password":"p"}`, "Invalid request body"},
		{`{"username":"missing-email","password":"p"}`, "Invalid request body"},
		{
			`{"username":"missing-password","email":"missing-password@example.com"}`,
			"Invalid request body",
		},
		{
			`{"username":"bad-role","email":"bad-role@example.com","password":"p","role":"superuser"}`,
			"Invalid role provided",
		},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/users", strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.body, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", tt.body, resp.StatusCode, http.StatusBadRequest)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.body, body["error"], tt.wantError)
		}
	}
}

func TestCreateUserDuplicateIs409(t *testing.T) {
	cases := []struct {
		body, wantError string
	}{
		{
			`{"username":"alice","email":"dupe-username@example.com","password":"p"}`,
			"Username is already taken",
		},
		{
			`{"username":"dupe-email","email":"alice@example.com","password":"p"}`,
			"Email is already taken",
		},
		{
			`{"username":"alice","email":"alice@example.com","password":"p"}`,
			"Username is already taken",
		},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/users", strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.body, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusConflict {
			t.Errorf("%s: status = %d, want %d", tt.body, resp.StatusCode, http.StatusConflict)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.body, body["error"], tt.wantError)
		}
	}
}

func TestListUsers(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/users", nil)
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
			Username string `json:"username"`
		} `json:"items"`
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}

	if body.Total != len(body.Items) {
		t.Errorf("total = %d, want %d (len of items)", body.Total, len(body.Items))
	}
	for _, want := range []string{"admin", "alice"} {
		found := false
		for _, item := range body.Items {
			if item.Username == want {
				found = true
			}
		}
		if !found {
			t.Errorf("items missing username %q", want)
		}
	}
}

func TestGetUser(t *testing.T) {
	cases := []struct {
		id, username, email, role string
	}{
		{seeding.AliceID, "alice", "alice@example.com", "user"},
		{seeding.AdminID, "admin", "admin@example.com", "admin"},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodGet, srv.URL+"/users/"+tt.id, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.username, err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			t.Fatalf("%s: status = %d, want %d", tt.username, resp.StatusCode, http.StatusOK)
		}
		var got struct {
			UserID    string `json:"user_id"`
			Username  string `json:"username"`
			Email     string `json:"email"`
			Role      string `json:"role"`
			CreatedAt string `json:"created_at"`
			UpdatedAt string `json:"updated_at"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("%s: %v", tt.username, err)
		}
		resp.Body.Close()

		if got.UserID != tt.id {
			t.Errorf("%s: user_id = %q, want %q", tt.username, got.UserID, tt.id)
		}
		if got.Username != tt.username {
			t.Errorf("%s: username = %q, want %q", tt.username, got.Username, tt.username)
		}
		if got.Email != tt.email {
			t.Errorf("%s: email = %q, want %q", tt.username, got.Email, tt.email)
		}
		if got.Role != tt.role {
			t.Errorf("%s: role = %q, want %q", tt.username, got.Role, tt.role)
		}
		if got.CreatedAt == "" || got.UpdatedAt == "" {
			t.Errorf(
				"%s: timestamps = %q/%q, want non-empty",
				tt.username,
				got.CreatedAt,
				got.UpdatedAt,
			)
		}
	}
}

func TestGetUserUnknownIs404(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/users/no-such-id", nil)
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
	if body["error"] != "User not found" {
		t.Errorf("error = %q, want %q", body["error"], "User not found")
	}
}

func TestUpdateUsernameSelf(t *testing.T) {
	if err := seeding.User(db, "rename-self-id", "rename-self", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "rename-self-token", "rename-self-id"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/rename-self-id",
		strings.NewReader(`{"username":"rename-self-2"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "rename-self-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var updated struct {
		UserID   string `json:"user_id"`
		Username string `json:"username"`
	}
	json.NewDecoder(resp.Body).Decode(&updated)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
	if updated.UserID != "rename-self-id" || updated.Username != "rename-self-2" {
		t.Errorf(
			"updated = %q/%q, want %q/%q",
			updated.UserID,
			updated.Username,
			"rename-self-id",
			"rename-self-2",
		)
	}

	req, err = http.NewRequest(http.MethodGet, srv.URL+"/me", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "rename-self-token"})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var fetched struct {
		User struct {
			Username string `json:"username"`
		} `json:"user"`
	}
	json.NewDecoder(resp.Body).Decode(&fetched)
	resp.Body.Close()

	if fetched.User.Username != "rename-self-2" {
		t.Errorf("persisted username = %q, want %q", fetched.User.Username, "rename-self-2")
	}
}

func TestUpdateUsernameAsAdmin(t *testing.T) {
	if err := seeding.User(db, "rename-target-id", "rename-target", user.RoleUser); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/rename-target-id",
		strings.NewReader(`{"username":"rename-target-2"}`))
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
	var updated struct {
		Username string `json:"username"`
	}
	json.NewDecoder(resp.Body).Decode(&updated)
	if updated.Username != "rename-target-2" {
		t.Errorf("username = %q, want %q", updated.Username, "rename-target-2")
	}
}

func TestUpdateUsernameOfOtherUserIs403(t *testing.T) {
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/"+seeding.AdminID,
		strings.NewReader(`{"username":"hijack"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AliceToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "You can only update your own username" {
		t.Errorf("error = %q, want %q", body["error"], "You can only update your own username")
	}
}

func TestUpdateUsernameTakenIs409(t *testing.T) {
	if err := seeding.User(db, "rename-taken-id", "rename-taken", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "rename-taken-token", "rename-taken-id"); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		body, wantError string
		status          int
	}{
		{`{"username":"admin"}`, "Username is already taken", http.StatusConflict},
		{`{"username":""}`, "Invalid request body", http.StatusBadRequest},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/rename-taken-id",
			strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "rename-taken-token"})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.body, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != tt.status {
			t.Errorf("%s: status = %d, want %d", tt.body, resp.StatusCode, tt.status)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.body, body["error"], tt.wantError)
		}
	}
}

func TestUpdateUsernameUnknownUserIs404(t *testing.T) {
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/no-such-id",
		strings.NewReader(`{"username":"ghost"}`))
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
	if body["error"] != "User not found" {
		t.Errorf("error = %q, want %q", body["error"], "User not found")
	}
}

func TestUpdatePasswordSelf(t *testing.T) {
	if err := seeding.User(db, "pw-self-id", "pw-self", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "pw-self-token", "pw-self-id"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/pw-self-id/password",
		strings.NewReader(`{"new_password":"pw-self-new"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "pw-self-token"})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusNoContent)
	}

	logins := []struct {
		password string
		status   int
	}{
		{"pw-self-new", http.StatusOK},
		{seeding.DefaultPassword, http.StatusUnauthorized},
	}
	for _, tt := range logins {
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/login", nil)
		if err != nil {
			t.Fatal(err)
		}
		req.SetBasicAuth("pw-self", tt.password)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("login %s: %v", tt.password, err)
		}
		resp.Body.Close()

		if resp.StatusCode != tt.status {
			t.Errorf("login %s: status = %d, want %d", tt.password, resp.StatusCode, tt.status)
		}
	}
}

func TestUpdatePasswordByAdminIsForbidden(t *testing.T) {
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/"+seeding.AliceID+"/password",
		strings.NewReader(`{"new_password":"hijack"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusForbidden)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "You can only update your own password" {
		t.Errorf("error = %q, want %q", body["error"], "You can only update your own password")
	}
}

func TestUpdatePasswordEmptyIs400(t *testing.T) {
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/"+seeding.AliceID+"/password",
		strings.NewReader(`{"new_password":""}`))
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AliceToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
	var body map[string]string
	json.NewDecoder(resp.Body).Decode(&body)
	if body["error"] != "Invalid request body" {
		t.Errorf("error = %q, want %q", body["error"], "Invalid request body")
	}
}

func TestUpdateRole(t *testing.T) {
	if err := seeding.User(db, "role-promote-id", "role-promote", user.RoleUser); err != nil {
		t.Fatal(err)
	}
	if err := seeding.Session(db, "role-promote-token", "role-promote-id"); err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/role-promote-id/role",
		strings.NewReader(`{"role":"admin"}`))
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

	req, err = http.NewRequest(http.MethodGet, srv.URL+"/users", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: "role-promote-token"})
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("promoted GET /users status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestUpdateRoleInvalidIs400(t *testing.T) {
	cases := []struct {
		body, wantError string
	}{
		{`{"role":"superuser"}`, "Invalid role provided"},
		{`{"role":""}`, "Invalid role provided"},
		{`not json`, "Invalid request body"},
	}
	for _, tt := range cases {
		req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/"+seeding.AliceID+"/role",
			strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.body, err)
		}
		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want %d", tt.body, resp.StatusCode, http.StatusBadRequest)
		}
		if body["error"] != tt.wantError {
			t.Errorf("%s: error = %q, want %q", tt.body, body["error"], tt.wantError)
		}
	}
}

func TestUpdateRoleUnknownUserIs404(t *testing.T) {
	req, err := http.NewRequest(http.MethodPatch, srv.URL+"/users/no-such-id/role",
		strings.NewReader(`{"role":"user"}`))
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
	if body["error"] != "User not found" {
		t.Errorf("error = %q, want %q", body["error"], "User not found")
	}
}

// The user package reads password_hash and salt on every SELECT * — this is
// the guard that they never reach a response body.
func TestResponsesOmitPasswordHashAndSalt(t *testing.T) {
	check := func(name string, body []byte) {
		t.Helper()
		if strings.Contains(string(body), `"password_hash"`) ||
			strings.Contains(string(body), `"salt"`) {
			t.Errorf("%s: body leaks credentials: %s", name, body)
		}
	}

	req, err := http.NewRequest(
		http.MethodPost,
		srv.URL+"/users",
		strings.NewReader(
			`{"username":"raw-create","email":"raw-create@example.com","password":"raw-create-password"}`,
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: seeding.AdminToken})
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /users: status = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	check("POST /users", raw)

	var created struct {
		UserID string `json:"user_id"`
	}
	if err := json.Unmarshal(raw, &created); err != nil {
		t.Fatal(err)
	}

	reads := []struct {
		name, method, path, token, body string
		status                          int
	}{
		{"GET /users", http.MethodGet, "/users", seeding.AdminToken, "", http.StatusOK},
		{
			"GET /users/{id}",
			http.MethodGet,
			"/users/" + seeding.AliceID,
			seeding.AdminToken,
			"",
			http.StatusOK,
		},
		{"GET /me", http.MethodGet, "/me", seeding.AliceToken, "", http.StatusOK},
		{
			"PATCH /users/{id}",
			http.MethodPatch,
			"/users/" + created.UserID,
			seeding.AdminToken,
			`{"username":"raw-create-2"}`,
			http.StatusOK,
		},
	}
	for _, tt := range reads {
		req, err := http.NewRequest(tt.method, srv.URL+tt.path, strings.NewReader(tt.body))
		if err != nil {
			t.Fatal(err)
		}
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: tt.token})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != tt.status {
			t.Errorf("%s: status = %d, want %d", tt.name, resp.StatusCode, tt.status)
		}
		check(tt.name, raw)
	}
}
