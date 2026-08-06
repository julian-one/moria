package test

import (
	"context"
	"testing"

	"moria/internal/session"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurgeExpired(t *testing.T) {
	db.MustExec(
		`INSERT INTO sessions (session_id, user_id, expires_at)
		 VALUES (?, ?, datetime('now', '-1 hour'))`,
		"expired-session", td.User.ID,
	)

	n, err := session.PurgeExpired(context.Background(), db)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, n, int64(1))

	var count int
	require.NoError(t,
		db.Get(&count, `SELECT COUNT(*) FROM sessions WHERE session_id = ?`, "expired-session"))
	assert.Zero(t, count)

	// Live sessions survive the purge.
	require.NoError(t,
		db.Get(&count, `SELECT COUNT(*) FROM sessions WHERE session_id = ?`, td.User.Session))
	assert.Equal(t, 1, count)
}
