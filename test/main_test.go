package test

import (
	"context"
	"flag"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"moria/route"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

var (
	server         *httptest.Server
	internalServer *httptest.Server
	db             *sqlx.DB
	td             *TestData
)

func TestMain(m *testing.M) {
	flag.Parse()

	ctx := context.Background()

	db = sqlx.MustConnect("sqlite3", ":memory:?_foreign_keys=on")

	schemaSQL, err := os.ReadFile(filepath.Join("..", "schema", "model.sql"))
	if err != nil {
		panic(err)
	}
	db.MustExec(string(schemaSQL))

	// Only log if the test is run with the -v flag
	logOutput := io.Discard
	if testing.Verbose() {
		logOutput = os.Stdout
	}
	logger := slog.New(slog.NewJSONHandler(logOutput, nil))

	// Initialize both listeners with the test database and logger
	config := route.Config{
		DB:     db,
		Logger: logger,
	}
	server = httptest.NewServer(route.Initialize(ctx, config))
	internalServer = httptest.NewServer(route.InitializeInternal(ctx, config))

	// Seed the database with test data
	td = Seed(db)

	// Run the tests
	code := m.Run()

	// NOTE: defer doesn't work here because os.Exit will terminate the program immediately
	server.Close()
	internalServer.Close()
	db.Close()

	os.Exit(code)
}
