package user

import "context"

type contextKey string

const userKey contextKey = "user"

func NewContext(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func FromContext(ctx context.Context) *User {
	u, ok := ctx.Value(userKey).(*User)
	if !ok {
		return nil
	}
	return u
}
