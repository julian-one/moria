package database

import (
	"database/sql"
	"fmt"

	"moria/schema"

	_ "modernc.org/sqlite"
)

func New(path string) (*sql.DB, error) {
	// Pragmas ride on the DSN so they apply to every pooled connection; a
	// bare PRAGMA Exec would reach only the one connection serving it.
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_time_format=sqlite",
		path,
	)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open the database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping the database: %w", err)
	}
	if _, err := db.Exec(schema.Model); err != nil {
		return nil, fmt.Errorf("failed to apply the schema: %w", err)
	}
	return db, nil
}
