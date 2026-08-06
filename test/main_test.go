package test

import (
	"flag"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"moria/route"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

var (
	server *httptest.Server
	db     *sqlx.DB
	td     *TestData
)

func TestMain(m *testing.M) {
	flag.Parse()

	db = sqlx.MustConnect("sqlite", "file::memory:?_pragma=foreign_keys(1)&_time_format=sqlite")

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

	// Initialize the router with the test database and logger
	config := route.Config{
		DB:     db,
		Logger: logger,
	}
	server = httptest.NewServer(route.Initialize(config))

	// Seed the database with test data
	td = Seed(db)

	// Run the tests
	code := m.Run()

	// NOTE: defer doesn't work here because os.Exit will terminate the program immediately
	server.Close()
	db.Close()

	os.Exit(code)
}
