package test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"moria/internal/session"
	"moria/internal/user"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Read-only tests (run before mutations) ---

func TestGetUser_Authenticated(t *testing.T) {
	req, err := http.NewRequest("GET", server.URL+"/users/"+td.User.ID, nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var u user.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&u))
	assert.Equal(t, td.User.ID, u.ID)
	assert.Equal(t, "regularuser", u.Username)
	assert.Equal(t, td.User.Email, u.Email)
}

func TestGetUser_Unauthenticated(t *testing.T) {
	req, err := http.NewRequest("GET", server.URL+"/users/some-id", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestListUsers_Admin(t *testing.T) {
	req, err := http.NewRequest("GET", server.URL+"/users", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.Admin.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body struct {
		Items []user.User `json:"items"`
		Total int         `json:"total"`
	}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Items, 2)
	assert.Equal(t, 2, body.Total)
}

func TestListUsers_NonAdmin(t *testing.T) {
	req, err := http.NewRequest("GET", server.URL+"/users", nil)
	require.NoError(t, err)
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpdateUser_DuplicateUsername(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID,
		strings.NewReader(`{"username":"adminuser"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestUpdateUser_OtherUser_NonAdmin(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.Admin.ID,
		strings.NewReader(`{"username":"hacked"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpdateUserRole_NonAdmin(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID+"/role",
		strings.NewReader(`{"role":"admin"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpdateUserRole_InvalidRole(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID+"/role",
		strings.NewReader(`{"role":"superadmin"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.Admin.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// --- Mutating tests (run last) ---

func TestUpdateUser_OwnUsername(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID,
		strings.NewReader(`{"username":"newname"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var u user.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&u))
	assert.Equal(t, "newname", u.Username)
}

func TestUpdateUser_OtherUser_Admin(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID,
		strings.NewReader(`{"username":"renamed"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.Admin.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var u user.User
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&u))
	assert.Equal(t, "renamed", u.Username)
}

func TestUpdateUserRole_Admin(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID+"/role",
		strings.NewReader(`{"role":"admin"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.Admin.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestUpdatePassword_Unauthenticated(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID+"/password",
		strings.NewReader(`{"new_password":"newpass123"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

func TestUpdatePassword_OtherUser(t *testing.T) {
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.Admin.ID+"/password",
		strings.NewReader(`{"new_password":"hacked"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
}

func TestUpdatePassword_Success(t *testing.T) {
	newPassword := "updatedpass456"
	req, err := http.NewRequest(
		"PATCH",
		server.URL+"/users/"+td.User.ID+"/password",
		strings.NewReader(`{"new_password":"`+newPassword+`"}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: session.CookieName, Value: td.User.Session})

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)

	// Verify login works with the new password
	loginReq, err := http.NewRequest("POST", server.URL+"/login", nil)
	require.NoError(t, err)
	loginReq.SetBasicAuth(td.User.Email, newPassword)

	loginResp, err := http.DefaultClient.Do(loginReq)
	require.NoError(t, err)
	defer loginResp.Body.Close()

	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
}
