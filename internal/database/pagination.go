package database

import (
	"fmt"
	"strconv"
)

const (
	// DefaultLimit is the number of rows returned when no limit is specified.
	DefaultLimit = 20
	// MaxLimit is the largest limit a caller may request.
	MaxLimit = 100
)

// Pagination holds the limit/offset values derived from query parameters.
type Pagination struct {
	Limit  int
	Offset int
}

// ParsePagination parses the "limit" and "offset" query parameter strings into
// a Pagination value. Missing parameters are replaced with DefaultLimit and 0
// respectively. Limit is capped at MaxLimit.
func ParsePagination(limit, offset string) (Pagination, error) {
	p := Pagination{Limit: DefaultLimit, Offset: 0}

	if limit != "" {
		v, err := strconv.Atoi(limit)
		if err != nil || v < 1 {
			return Pagination{}, fmt.Errorf("invalid limit: must be a positive integer")
		}
		if v > MaxLimit {
			v = MaxLimit
		}
		p.Limit = v
	}

	if offset != "" {
		v, err := strconv.Atoi(offset)
		if err != nil || v < 0 {
			return Pagination{}, fmt.Errorf("invalid offset: must be a non-negative integer")
		}
		p.Offset = v
	}

	return p, nil
}
