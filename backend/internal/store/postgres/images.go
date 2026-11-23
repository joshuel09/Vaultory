package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Image is a stored upload and its derived gallery rendition.
type Image struct {
	ID           uuid.UUID
	CollectorID  uuid.UUID
	OriginalKey  string
	RenditionKey string
	ContentType  string
	ByteSize     int64
	Width        int
	Height       int
}

// InsertImage records an accepted upload against the collector who uploaded it.
//
// The owner is recorded on the image itself, not inferred from a collectible, so an image can be
// authorized on its own request — including one not yet attached to anything (FR-015).
func (s *Store) InsertImage(ctx context.Context, img Image) (Image, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO collectible_images (
			collector_id, original_key, rendition_key, content_type, byte_size, width, height
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		img.CollectorID, img.OriginalKey, img.RenditionKey,
		img.ContentType, img.ByteSize, img.Width, img.Height,
	).Scan(&img.ID)
	if err != nil {
		return Image{}, fmt.Errorf("insert image: %w", err)
	}
	return img, nil
}

// GetImage reads one image *belonging to this collector*.
//
// The collector is part of the WHERE clause rather than something checked afterwards. That is what
// makes FR-015 hold by construction: there is no moment at which this code holds another
// collector's image and has yet to decide whether to return it. Absent and someone-else's are the
// same answer (FR-027).
func (s *Store) GetImage(ctx context.Context, collectorID, imageID uuid.UUID) (Image, error) {
	var img Image
	err := s.pool.QueryRow(ctx, `
		SELECT id, collector_id, original_key, rendition_key, content_type, byte_size, width, height
		FROM collectible_images
		WHERE id = $1 AND collector_id = $2`,
		imageID, collectorID,
	).Scan(&img.ID, &img.CollectorID, &img.OriginalKey, &img.RenditionKey,
		&img.ContentType, &img.ByteSize, &img.Width, &img.Height)
	if errors.Is(err, pgx.ErrNoRows) {
		return Image{}, ErrNotFound
	}
	if err != nil {
		return Image{}, fmt.Errorf("get image: %w", err)
	}
	return img, nil
}

// ImageExistsForCollector is the pre-insert check that a referenced image is the collector's own.
// The composite foreign key is the real guarantee; this exists so the collector gets a field-level
// validation message rather than a bare failure.
func (s *Store) ImageExistsForCollector(ctx context.Context, collectorID, imageID uuid.UUID) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM collectible_images WHERE id = $1 AND collector_id = $2)`,
		imageID, collectorID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check image ownership: %w", err)
	}
	return exists, nil
}

// SweepAbandonedImages removes uploads that no collectible references and that are older than
// olderThan, at most limit of them, oldest first, and queues their files for deletion. It returns
// how many it removed.
//
// An upload exists before anything references it — uploading is a separate operation (research.md
// Decision 3 of feature 001) — so a form abandoned after its upload leaves a row and two files
// behind. This is what eventually reclaims them (feature 009).
//
// The hard part is a collector saving with that very upload while it is being swept. The outcome
// must be one of two: the save keeps its photograph and the upload survives, or the save is refused
// and the upload is gone (FR-006). It is settled by row locks rather than by timing:
//
//   - A save referencing the upload makes the foreign key take FOR KEY SHARE on its row. FOR UPDATE
//     conflicts with that, so SKIP LOCKED passes over an upload a save is attaching right now.
//   - A save that committed after the first statement's snapshot is seen by the second, which takes
//     a fresh one and re-checks for a reference before deleting.
//   - A save that starts after the lock is taken waits on it, then fails its foreign-key check
//     against a row that is gone — ErrUnknownImage, which the collector sees as a refused save.
//
// Why not the obvious single DELETE ... WHERE NOT EXISTS: it waits. Run against an upload a save is
// attaching, it blocks on the save's lock until the save commits. That was tried, and PostgreSQL
// then did the right thing and kept the upload — but this runs inside a collector's upload request,
// and that collector should not wait on somebody else's save (SC-005). SKIP LOCKED passes over the
// busy upload instead; the next sweep finds it referenced. The explicit re-check in a second
// statement also means correctness does not rest on how READ COMMITTED re-evaluates a subquery
// after a lock wait (research.md Decision 2 of feature 009).
//
// No collector_id predicate, deliberately. Every collector-facing query carries one because it
// reads something out to a collector; this reads nothing out to anyone. It is maintenance, like the
// deletion drain, and each queue entry still records whose files it holds.
func (s *Store) SweepAbandonedImages(ctx context.Context, olderThan time.Duration, limit int) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT ci.id
		FROM collectible_images ci
		WHERE ci.created_at < now() - make_interval(secs => $1)
		  AND NOT EXISTS (SELECT 1 FROM collectibles c WHERE c.image_id = ci.id)
		ORDER BY ci.created_at
		LIMIT $2
		FOR UPDATE OF ci SKIP LOCKED`, olderThan.Seconds(), limit)
	if err != nil {
		return 0, fmt.Errorf("find abandoned uploads: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return 0, fmt.Errorf("find abandoned uploads: %w", err)
	}
	if len(ids) == 0 {
		return 0, nil
	}

	// A new statement, so a new snapshot: a reference committed since the one above is visible
	// here, and nothing can commit a new one while this transaction holds the rows.
	tag, err := tx.Exec(ctx, `
		WITH swept AS (
			DELETE FROM collectible_images ci
			WHERE ci.id = ANY($1)
			  AND NOT EXISTS (SELECT 1 FROM collectibles c WHERE c.image_id = ci.id)
			RETURNING ci.id, ci.collector_id, ci.original_key, ci.rendition_key
		)
		INSERT INTO pending_image_deletions (image_id, collector_id, original_key, rendition_key)
		SELECT id, collector_id, original_key, rendition_key FROM swept
		ON CONFLICT (image_id) DO NOTHING`, ids)
	if err != nil {
		return 0, fmt.Errorf("sweep abandoned uploads: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit sweep: %w", err)
	}
	return int(tag.RowsAffected()), nil
}
