package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// TrashedRow is one collectible in the Trash, with when it was deleted.
type TrashedRow struct {
	Row
	DeletedAt time.Time
}

// TrashPage is one page of a collector's Trash.
type TrashPage struct {
	Rows       []TrashedRow
	NextCursor string
}

// ListTrash returns one page of this collector's Trash: collectibles deleted within retention, most
// recently deleted first (feature 010, FR-006).
//
// Keyset pagination on (deleted_at, id), for the same reason the gallery pages on (created_at, id):
// page cost stays constant, and the id makes the order total. The cursor encodes deleted_at, so a
// Trash cursor and a gallery cursor are not interchangeable — passing one to the other yields a page
// that is merely wrong, never another collector's data, since collector_id is in every predicate.
//
// The retention cutoff is the one Restore and PurgeExpiredTrash use (research.md Decision 3).
func (s *Store) ListTrash(
	ctx context.Context, collectorID uuid.UUID, retention time.Duration, cursor string, limit int,
) (TrashPage, error) {
	var (
		afterTime *time.Time
		afterID   *uuid.UUID
	)
	if cursor != "" {
		t, id, err := decodeCursor(cursor)
		if err != nil {
			return TrashPage{}, err
		}
		afterTime, afterID = &t, &id
	}

	rows, err := s.pool.Query(ctx, `
		SELECT `+collectibleColumns+`, c.deleted_at
		FROM collectibles c
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE c.collector_id = $1
		  AND c.deleted_at IS NOT NULL
		  AND c.deleted_at > now() - make_interval(secs => $2)
		  AND ($3::timestamptz IS NULL OR (c.deleted_at, c.id) < ($3, $4))
		ORDER BY c.deleted_at DESC, c.id DESC
		LIMIT $5`,
		collectorID, retention.Seconds(), afterTime, afterID, limit+1)
	if err != nil {
		return TrashPage{}, fmt.Errorf("list trash: %w", err)
	}
	defer rows.Close()

	var out []TrashedRow
	for rows.Next() {
		var deletedAt time.Time
		r, err := scanRow(trailing{rows, &deletedAt})
		if err != nil {
			return TrashPage{}, fmt.Errorf("scan trashed collectible: %w", err)
		}
		out = append(out, TrashedRow{Row: r, DeletedAt: deletedAt})
	}
	if err := rows.Err(); err != nil {
		return TrashPage{}, fmt.Errorf("read trash: %w", err)
	}

	page := TrashPage{}
	if len(out) > limit {
		last := out[limit-1]
		page.NextCursor = encodeCursor(last.DeletedAt, last.Collectible.ID)
		out = out[:limit]
	}
	page.Rows = out
	return page, nil
}

// trailing lets scanRow read the shared projection from a row that carries one more column after it.
type trailing struct {
	row   pgx.Row
	extra any
}

func (t trailing) Scan(dest ...any) error { return t.row.Scan(append(dest, t.extra)...) }

// Purged identifies a collectible the purge destroyed, for the record the caller keeps (FR-015).
type Purged struct {
	CollectorID   uuid.UUID
	CollectibleID uuid.UUID
}

// PurgeExpiredTrash destroys up to limit collectibles that have been in the Trash longer than
// retention, oldest deletion first, with their photographs (feature 010, FR-010).
//
// Each goes the way feature 006 deleted things: the collectible row and its image row in one
// transaction, the files queued for the drain. So a photograph is never fetchable after its
// collectible has gone, and a storage fault is retried rather than leaked (006 FR-020, FR-020a).
//
// Across every collector, deliberately — like the deletion drain, it reads nothing out to anyone,
// and it returns each collector with each collectible so the record names both.
//
// FOR UPDATE SKIP LOCKED against a concurrent restore (FR-012). A restore that has locked a row is
// passed over, and if it commits, its row no longer qualifies. A restore arriving after this has the
// lock waits, then finds nothing to restore. Either way exactly one outcome — and a collector's
// deletion, which runs this, never waits on someone else's restore.
func (s *Store) PurgeExpiredTrash(ctx context.Context, retention time.Duration, limit int) ([]Purged, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, collector_id, image_id
		FROM collectibles
		WHERE deleted_at IS NOT NULL
		  AND deleted_at <= now() - make_interval(secs => $1)
		ORDER BY deleted_at
		LIMIT $2
		FOR UPDATE SKIP LOCKED`, retention.Seconds(), limit)
	if err != nil {
		return nil, fmt.Errorf("find expired trash: %w", err)
	}
	type expired struct {
		id, collectorID uuid.UUID
		imageID         *uuid.UUID
	}
	var found []expired
	for rows.Next() {
		var e expired
		if err := rows.Scan(&e.id, &e.collectorID, &e.imageID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan expired trash: %w", err)
		}
		found = append(found, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read expired trash: %w", err)
	}

	purged := make([]Purged, 0, len(found))
	for _, e := range found {
		if _, err := tx.Exec(ctx,
			`DELETE FROM collectibles WHERE id = $1 AND collector_id = $2`, e.id, e.collectorID); err != nil {
			return nil, fmt.Errorf("purge collectible: %w", err)
		}
		if e.imageID != nil {
			if err := releaseImage(ctx, tx, e.collectorID, *e.imageID); err != nil {
				return nil, err
			}
		}
		purged = append(purged, Purged{CollectorID: e.collectorID, CollectibleID: e.id})
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit purge: %w", err)
	}
	return purged, nil
}
