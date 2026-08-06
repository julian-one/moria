package session

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
)

const PurgeInterval = time.Hour

// PurgeExpired deletes all sessions past their expiry.
func PurgeExpired(ctx context.Context, db sqlx.ExecerContext) (int64, error) {
	res, err := db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at <= datetime('now')`)
	if err != nil {
		return 0, fmt.Errorf("failed to purge expired sessions: %w", err)
	}
	return res.RowsAffected()
}

// PurgeLoop deletes expired sessions every PurgeInterval until ctx is
// cancelled. Validate already rejects expired sessions; this only keeps the
// table from accumulating dead rows.
func PurgeLoop(ctx context.Context, logger *slog.Logger, db sqlx.ExecerContext) {
	ticker := time.NewTicker(PurgeInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := PurgeExpired(ctx, db)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				logger.Error("failed to purge expired sessions", "error", err)
				continue
			}
			if n > 0 {
				logger.Info("purged expired sessions", "count", n)
			}
		}
	}
}
