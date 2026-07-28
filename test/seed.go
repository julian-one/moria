package test

import (
	"context"

	"moria/internal/session"
	"moria/internal/user"

	"github.com/jmoiron/sqlx"
)

type TestData struct {
	Admin TestUser
	User  TestUser
}

type TestUser struct {
	ID      string
	Session string
	Email   string
}

func Seed(db *sqlx.DB) *TestData {
	ctx := context.Background()

	adminID, err := user.Create(ctx, db, user.CreateRequest{
		Username: "adminuser",
		Email:    "admin@test.com",
		Password: "password123",
	})
	if err != nil {
		panic(err)
	}
	db.MustExec(`UPDATE users SET role = 'admin' WHERE user_id = ?`, adminID)

	userID, err := user.Create(ctx, db, user.CreateRequest{
		Username: "regularuser",
		Email:    "user@test.com",
		Password: "password123",
	})
	if err != nil {
		panic(err)
	}
	adminSession, err := session.Create(ctx, db, adminID)
	if err != nil {
		panic(err)
	}

	userSession, err := session.Create(ctx, db, userID)
	if err != nil {
		panic(err)
	}

	return &TestData{
		Admin: TestUser{
			ID:      adminID,
			Session: adminSession.SessionID,
			Email:   "admin@test.com",
		},
		User: TestUser{
			ID:      userID,
			Session: userSession.SessionID,
			Email:   "user@test.com",
		},
	}
}
