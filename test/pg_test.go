package test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"moria/internal/database"
)

func createTestDatabase() (string, func() error, error) {
	base := os.Getenv("MORIA_TEST_DATABASE_URL")
	if base == "" {
		return "", nil, fmt.Errorf("MORIA_TEST_DATABASE_URL is not set")
	}
	admin, err := sqlx.Open("pgx", base)
	if err != nil {
		return "", nil, err
	}
	name := "moria_test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	if _, err := admin.Exec("CREATE DATABASE " + name); err != nil {
		admin.Close()
		return "", nil, err
	}
	u, err := url.Parse(base)
	if err != nil {
		admin.Close()
		return "", nil, err
	}
	u.Path = "/" + name
	drop := func() error {
		defer admin.Close()
		if _, err := admin.Exec("DROP DATABASE " + name + " WITH (FORCE)"); err != nil {
			return fmt.Errorf("drop %s: %w", name, err)
		}
		return nil
	}
	return u.String(), drop, nil
}

func newTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	dbURL, drop, err := createTestDatabase()
	if err != nil {
		t.Fatal(err)
	}
	tdb, err := database.New(context.Background(), dbURL)
	if err != nil {
		_ = drop()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		tdb.Close()
		if err := drop(); err != nil {
			t.Error(err)
		}
	})
	return tdb
}

func newTestDatabaseURL(t *testing.T) string {
	t.Helper()
	dbURL, drop, err := createTestDatabase()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := drop(); err != nil {
			t.Error(err)
		}
	})
	return dbURL
}
