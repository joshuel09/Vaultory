package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The audit log (feature 014). Collection changes record their entry inside their own transaction,
// so a change and its record are committed together or not at all (FR-004). Entries carry the
// account and the collectible's id — never a name, a note, a price or an image (FR-003).
//
// The table refuses to be changed, and refuses deletion of anything under a year old; that is
// enforced by triggers in migration 000014, not here (FR-006).

const (
	auditCollectibleCreated = "collectible.created"
	auditCollectibleEdited  = "collectible.edited"
	auditCollectibleDeleted = "collectible.deleted"
)

// recordCollectibleEvent adds one entry for a collection change, inside the caller's transaction.
// The account is resolved from the collector, so the entry names the same account the frontend's
// sign-in entries do.
func recordCollectibleEvent(ctx context.Context, tx pgx.Tx, action string, collectorID, collectibleID uuid.UUID) error {
	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (action, outcome, account_id, subject_id)
		SELECT $1, 'succeeded', c.user_id, $3::text
		FROM collectors c WHERE c.id = $2`,
		action, collectorID, collectibleID.String()); err != nil {
		return fmt.Errorf("record %s: %w", action, err)
	}
	return nil
}

// PurgeExpiredAudit removes up to limit entries over a year old, oldest first, and reports how many
// (FR-007). There is no schedule: it runs at start-up and after deletions.
func (s *Store) PurgeExpiredAudit(ctx context.Context, limit int) (int, error) {
	var n int
	if err := s.pool.QueryRow(ctx, `SELECT audit_purge_expired($1)`, limit).Scan(&n); err != nil {
		return 0, fmt.Errorf("purge expired audit entries: %w", err)
	}
	return n, nil
}
