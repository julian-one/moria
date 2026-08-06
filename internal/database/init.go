package database

import (
	"fmt"
	"os"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

// QB is the global query builder, pre-configured with SQLite's ? placeholder format.
var QB = sq.StatementBuilder.PlaceholderFormat(sq.Question)

// New opens the SQLite database at path, applies recommended pragmas, and
// initialises the schema from schemaPath. The caller is responsible for
// closing the returned *sqlx.DB.
func New(path, schemaPath string) (*sqlx.DB, error) {
	// Pragmas must ride on the DSN so they apply to every pooled connection;
	// see docs/architecture.md (Storage) for what each one does.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_time_format=sqlite",
		path,
	)

	db, err := sqlx.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database: %w", err)
	}

	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	schemaBytes, err := os.ReadFile(schemaPath) //nolint:gosec // The path comes from operator config, not request input.
	if err != nil {
		return nil, fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := db.Exec(string(schemaBytes)); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}
