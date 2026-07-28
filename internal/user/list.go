package user

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"moria/internal/database"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

// ListOptions contains options for listing users
type ListOptions struct {
	Search     string
	Role       *Role
	OrderBy    []database.Order
	Pagination database.Pagination
}

func ParseListOptions(r *http.Request) (ListOptions, error) {
	query := r.URL.Query()

	var opts ListOptions

	// Parse search parameter
	if search := query.Get("search"); search != "" {
		opts.Search = search
	}

	// Parse role filter
	if roleStr := query.Get("role"); roleStr != "" {
		role := Role(roleStr)
		if !role.Valid() {
			return opts, fmt.Errorf("invalid role: %s (must be 'admin' or 'user')", roleStr)
		}
		opts.Role = &role
	}

	parsed, err := database.ParseOrder(query.Get("order_by"))
	if err != nil {
		return opts, err
	}
	opts.OrderBy = parsed

	pagination, err := database.ParsePagination(query.Get("limit"), query.Get("offset"))
	if err != nil {
		return opts, err
	}
	opts.Pagination = pagination

	return opts, opts.validate()
}

func (opts ListOptions) validate() error {
	if len(opts.OrderBy) == 0 {
		return nil
	}

	t := reflect.TypeFor[User]()
	validColumns := make(map[string]bool, t.NumField())
	for field := range t.Fields() {
		if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
			validColumns[tag] = true
		}
	}

	for _, o := range opts.OrderBy {
		if !validColumns[o.Column] {
			return fmt.Errorf("invalid column name: %s", o.Column)
		}
	}
	return nil
}

func applyFilters(q sq.SelectBuilder, opts ListOptions) sq.SelectBuilder {
	if opts.Search != "" {
		searchPattern := "%" + strings.ToLower(opts.Search) + "%"
		q = q.Where(sq.Or{
			sq.Expr("LOWER(u.username) LIKE ?", searchPattern),
			sq.Expr("LOWER(u.email) LIKE ?", searchPattern),
		})
	}
	if opts.Role != nil {
		q = q.Where(sq.Eq{"u.role": *opts.Role})
	}
	return q
}

func Count(ctx context.Context, db sqlx.QueryerContext, opts ListOptions) (int, error) {
	q := applyFilters(database.QB.Select("COUNT(*)").From("users u"), opts)

	sql, args, err := q.ToSql()
	if err != nil {
		return 0, fmt.Errorf("failed to build count query: %w", err)
	}

	var total int
	if err = db.QueryRowxContext(ctx, sql, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("failed to count users: %w", err)
	}

	return total, nil
}

func List(ctx context.Context, db sqlx.QueryerContext, opts ListOptions) ([]User, error) {
	if opts.Role != nil && !opts.Role.Valid() {
		return nil, fmt.Errorf("invalid role: %s", *opts.Role)
	}

	query := applyFilters(database.QB.Select("u.*").From("users u"), opts)

	if len(opts.OrderBy) > 0 {
		for _, o := range opts.OrderBy {
			query = query.OrderBy(fmt.Sprintf("u.%s %s", o.Column, o.Direction))
		}
	} else {
		query = query.OrderBy("u.created_at DESC")
	}

	query = query.Limit(uint64(opts.Pagination.Limit)).Offset(uint64(opts.Pagination.Offset))

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	users := []User{}
	err = sqlx.SelectContext(ctx, db, &users, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}

	return users, nil
}
