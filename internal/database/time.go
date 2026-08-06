package database

import (
	"fmt"
	"time"
)

type UTCTime struct {
	time.Time
}

func (t *UTCTime) Scan(v any) error {
	tim, ok := v.(time.Time)
	if !ok {
		return fmt.Errorf("failed to scan %T as a time", v)
	}
	t.Time = tim.UTC()
	return nil
}
