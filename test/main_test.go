package test

import (
	"bytes"
	"context"
	"log"
	"log/slog"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/jmoiron/sqlx"

	"moria/internal/database"
	"moria/route"
	"moria/test/seeding"
)

var (
	srv  *httptest.Server
	db   *sqlx.DB
	logs lockedBuffer
)

// The server logs from request goroutines after the response is already on
// the wire, so the capture must be safe for concurrent writes.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

func TestMain(m *testing.M) {
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))

	boot := log.New(os.Stderr, "", log.LstdFlags)

	dbURL, drop, err := createTestDatabase()
	if err != nil {
		boot.Fatal(err)
	}
	db, err = database.New(context.Background(), dbURL)
	if err != nil {
		boot.Fatal(err)
	}
	if err := seeding.Defaults(db); err != nil {
		boot.Fatal(err)
	}
	srv = httptest.NewServer(route.Initialize(route.Config{DB: db}))

	code := m.Run()

	srv.Close()
	db.Close()
	if err := drop(); err != nil {
		boot.Print(err)
	}
	os.Exit(code)
}
