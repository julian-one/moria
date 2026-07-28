package user

import "time"

type User struct {
	ID        string    `db:"user_id"       json:"user_id"`
	Username  string    `db:"username"      json:"username"`
	Email     string    `db:"email"         json:"email"`
	Hash      string    `db:"password_hash" json:"-"`
	Salt      []byte    `db:"salt"          json:"-"`
	Role      Role      `db:"role"          json:"role"`
	CreatedAt time.Time `db:"created_at"    json:"created_at"`
	UpdatedAt time.Time `db:"updated_at"    json:"updated_at"`
}

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

func (r Role) Valid() bool {
	switch r {
	case RoleAdmin, RoleUser:
		return true
	default:
		return false
	}
}
