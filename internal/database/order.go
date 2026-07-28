package database

import (
	"fmt"
	"strings"
)

// Direction is the sort direction for a query order clause.
type Direction string

const (
	Descending Direction = "DESC"
	Ascending  Direction = "ASC"
)

// Valid reports whether d is a recognised sort direction.
func (o Direction) Valid() bool {
	switch o {
	case Descending, Ascending:
		return true
	default:
		return false
	}
}

// Order represents a single column ordering for database queries
type Order struct {
	Column    string
	Direction Direction
}

// ParseOrder parses a comma-separated order_by string (e.g. "column:asc,column:desc")
// into a slice of Order values.
func ParseOrder(raw string) ([]Order, error) {
	if raw == "" {
		return nil, nil
	}

	var result []Order

	for order := range strings.SplitSeq(raw, ",") {
		parts := strings.Split(strings.TrimSpace(order), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf(
				"invalid order_by format: %s (expected 'column:order')",
				order,
			)
		}

		dir := Direction(strings.ToUpper(strings.TrimSpace(parts[1])))
		if !dir.Valid() {
			return nil, fmt.Errorf(
				"invalid order direction: %s (must be 'asc' or 'desc')",
				parts[1],
			)
		}

		result = append(result, Order{
			Column:    strings.TrimSpace(parts[0]),
			Direction: dir,
		})
	}

	return result, nil
}
