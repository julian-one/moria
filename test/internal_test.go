package test

import (
	"encoding/json"
	"net/http"
	"testing"

	"moria/internal/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateSession_Valid(t *testing.T) {
	resp, err := http.Get(internalServer.URL + "/internal/sessions/" + td.User.Session)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var v session.Validation
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&v))
	assert.Equal(t, td.User.Session, v.SessionID)
	assert.Equal(t, td.User.ID, v.UserID)
	assert.Equal(t, "user", v.Role)
}

func TestValidateSession_AdminRole(t *testing.T) {
	resp, err := http.Get(internalServer.URL + "/internal/sessions/" + td.Admin.Session)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var v session.Validation
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&v))
	assert.Equal(t, td.Admin.ID, v.UserID)
	assert.Equal(t, "admin", v.Role)
}

func TestValidateSession_Unknown(t *testing.T) {
	resp, err := http.Get(internalServer.URL + "/internal/sessions/not-a-session")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestValidateSession_Expired(t *testing.T) {
	db.MustExec(
		`INSERT INTO sessions (session_id, user_id, expires_at)
		 VALUES (?, ?, datetime('now', '-1 hour'))`,
		"expired-session-id", td.User.ID,
	)

	resp, err := http.Get(internalServer.URL + "/internal/sessions/expired-session-id")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestValidateSession_NotOnPublicListener(t *testing.T) {
	resp, err := http.Get(server.URL + "/internal/sessions/" + td.User.Session)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestInternalListUsers(t *testing.T) {
	resp, err := http.Get(
		internalServer.URL + "/internal/users?ids=" + td.User.ID + "," + td.Admin.ID + ",missing",
	)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var users []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&users))
	assert.Len(t, users, 2)

	byID := make(map[string]map[string]any)
	for _, u := range users {
		byID[u["user_id"].(string)] = u
		assert.NotContains(t, u, "password_hash")
		assert.NotContains(t, u, "salt")
	}
	assert.Equal(t, "regularuser", byID[td.User.ID]["username"])
	assert.Equal(t, "adminuser", byID[td.Admin.ID]["username"])
}

func TestInternalListUsers_Empty(t *testing.T) {
	resp, err := http.Get(internalServer.URL + "/internal/users?ids=")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var users []map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&users))
	assert.Len(t, users, 0)
}
