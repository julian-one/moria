package test

import (
	"encoding/base64"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogin_WithEmail(t *testing.T) {
	req, err := http.NewRequest("POST", server.URL+"/login", nil)
	require.NoError(t, err)
	req.Header.Set(
		"Authorization",
		"Basic "+base64.StdEncoding.EncodeToString([]byte("user@test.com:password123")),
	)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogin_WithUsername(t *testing.T) {
	req, err := http.NewRequest("POST", server.URL+"/login", nil)
	require.NoError(t, err)
	req.Header.Set(
		"Authorization",
		"Basic "+base64.StdEncoding.EncodeToString([]byte("regularuser:password123")),
	)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	req, err := http.NewRequest("POST", server.URL+"/login", nil)
	require.NoError(t, err)
	req.Header.Set(
		"Authorization",
		"Basic "+base64.StdEncoding.EncodeToString([]byte("nobody@test.com:wrongpass")),
	)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
