package user

import (
	"context"
	"database/sql"
	"fmt"
)

func List(ctx context.Context, db *sql.DB) ([]User, error) {
	fmt.Println("asdf")
	rows, err := db.QueryContext(ctx,
		`SELECT * FROM users ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var u User
		if err := u.scan(rows); err != nil {
			return nil, fmt.Errorf("failed to list users: %w", err)
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	return users, nil
}
