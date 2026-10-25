package postgres

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const CredentialRetention = 7 * 24 * time.Hour
const RetentionBatchSize = 500
const RetentionInterval = 10 * time.Minute

type RetentionResult struct{ Sessions, AccessKeys, Invites int64 }

// CleanupExpiredCredentials deletes at most batch rows from each table. Active
// credentials are never removed. Multiple instances can safely share the work.
func (r Repository) CleanupExpiredCredentials(ctx context.Context, now time.Time, batch int) (RetentionResult, error) {
	var result RetentionResult
	if batch < 1 || batch > RetentionBatchSize {
		return result, fmt.Errorf("retention batch must be between 1 and %d", RetentionBatchSize)
	}
	cutoff := now.Add(-CredentialRetention)
	queries := []struct {
		sql   string
		count *int64
	}{
		{`WITH expired AS (SELECT id FROM sessions WHERE expires_at <= $1 OR revoked_at <= $1 ORDER BY expires_at,id LIMIT $2 FOR UPDATE SKIP LOCKED) DELETE FROM sessions USING expired WHERE sessions.id=expired.id`, &result.Sessions},
		{`WITH expired AS (SELECT id FROM admin_access_keys WHERE expires_at <= $1 OR used_at <= $1 ORDER BY expires_at,id LIMIT $2 FOR UPDATE SKIP LOCKED) DELETE FROM admin_access_keys USING expired WHERE admin_access_keys.id=expired.id`, &result.AccessKeys},
		{`WITH expired AS (SELECT id FROM family_invites WHERE expires_at <= $1 OR revoked_at <= $1 ORDER BY expires_at,id LIMIT $2 FOR UPDATE SKIP LOCKED) DELETE FROM family_invites USING expired WHERE family_invites.id=expired.id`, &result.Invites},
	}
	for _, query := range queries {
		tag, err := r.pool.Exec(ctx, query.sql, cutoff, batch)
		if err != nil {
			return result, normalizeError(err)
		}
		*query.count = tag.RowsAffected()
	}
	return result, nil
}

func (r Repository) RunCredentialRetention(ctx context.Context, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	ticker := time.NewTicker(RetentionInterval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		batchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		result, err := r.CleanupExpiredCredentials(batchCtx, time.Now(), RetentionBatchSize)
		cancel()
		if err != nil && ctx.Err() == nil {
			logger.Error("credential retention failed", "error", err)
		}
		if result.Sessions+result.AccessKeys+result.Invites > 0 {
			logger.Info("credential retention completed", "sessions", result.Sessions, "access_keys", result.AccessKeys, "invites", result.Invites)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
