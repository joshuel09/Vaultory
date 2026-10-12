package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// PendingImageDeletion is an image whose database row is already gone and whose files are not.
//
// It exists because the two cannot be removed atomically: the row deletion is transactional, the
// file deletion is not, and a transaction that spans a filesystem has no sensible rollback. The
// queue is what makes the second half survive a crash, so a failed file removal is retried rather
// than forgotten (FR-020a).
type PendingImageDeletion struct {
	ImageID      uuid.UUID
	CollectorID  uuid.UUID
	OriginalKey  string
	RenditionKey string
}

// PendingImageDeletions reads a bounded batch, oldest first.
//
// Bounded because this runs inside a collector's edit or delete request, and an unbounded queue
// drain would turn one deletion into arbitrarily much work. Oldest first so an entry that keeps
// failing cannot starve the rest.
func (s *Store) PendingImageDeletions(ctx context.Context, limit int) ([]PendingImageDeletion, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT image_id, collector_id, original_key, rendition_key
		FROM pending_image_deletions
		ORDER BY created_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("read pending image deletions: %w", err)
	}
	defer rows.Close()

	var out []PendingImageDeletion
	for rows.Next() {
		var p PendingImageDeletion
		if err := rows.Scan(&p.ImageID, &p.CollectorID, &p.OriginalKey, &p.RenditionKey); err != nil {
			return nil, fmt.Errorf("scan pending image deletion: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read pending image deletions: %w", err)
	}
	return out, nil
}

// ForgetImageDeletion removes a queue entry once its files are gone.
//
// Called only after the files have actually been deleted. Removing the row first would turn a
// storage failure into a permanent leak, which is the one thing the queue exists to prevent.
func (s *Store) ForgetImageDeletion(ctx context.Context, imageID uuid.UUID) error {
	if _, err := s.pool.Exec(ctx,
		`DELETE FROM pending_image_deletions WHERE image_id = $1`, imageID); err != nil {
		return fmt.Errorf("forget image deletion: %w", err)
	}
	return nil
}
