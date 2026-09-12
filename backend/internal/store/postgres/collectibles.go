package postgres

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// ErrNotFound means no such row exists *for this collector*. Callers turn it into a 404 without
// distinguishing "absent" from "someone else's" — that distinction is exactly what FR-027 forbids
// disclosing.
var ErrNotFound = errors.New("not found")

// uniqueViolation is PostgreSQL's SQLSTATE for a unique constraint breach.
const uniqueViolation = "23505"

// Store holds every query Vaultory runs. Each one carries collector_id as a predicate, so there is
// no code path that can hold the wrong collector's row (research.md Decision 5).
type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// collectibleColumns is the projection the gallery and the add response share.
//
// "character" is quoted because CHARACTER is a PostgreSQL keyword. purchase_price is cast to text
// so the amount arrives as an exact decimal string and never passes through a float on the way out
// (Constitution IV).
const collectibleColumns = `
	c.id, c.collector_id, c.name, c.collection_status,
	c."character", c.series, c.manufacturer, c.category, c.scale, c.edition,
	c.purchase_price::text, c.purchase_date, c.release_date, c.notes,
	c.image_id, i.rendition_key, c.created_at`

// Row is one collectible as stored, with just enough of its image to render a card. The rendition
// arrives through a join in the same statement — there is no per-entry image lookup (no N+1).
type Row struct {
	Collectible  collectible.Collectible
	RenditionKey *string
}

func scanRow(s pgx.Row) (Row, error) {
	var (
		r            Row
		c            collectible.Collectible
		priceText    *string
		purchaseDate *time.Time
		releaseDate  *time.Time
		renditionKey *string
	)
	err := s.Scan(
		&c.ID, &c.CollectorID, &c.Name, &c.Status,
		&c.Character, &c.Series, &c.Manufacturer, &c.Category, &c.Scale, &c.Edition,
		&priceText, &purchaseDate, &releaseDate, &c.Notes,
		&c.ImageID, &renditionKey, &c.CreatedAt,
	)
	if err != nil {
		return Row{}, err
	}
	if priceText != nil {
		money, err := collectible.ParseMoney(strings.TrimSpace(*priceText))
		if err != nil {
			// The database holds numeric(12,2) with a non-negative check, so this cannot happen
			// without the schema having drifted. Fail loudly rather than show a wrong amount.
			return Row{}, fmt.Errorf("stored purchase price %q is not a valid amount: %w", *priceText, err)
		}
		c.PurchasePrice = &money
	}
	c.PurchaseDate = purchaseDate
	c.ReleaseDate = releaseDate
	r.Collectible = c
	r.RenditionKey = renditionKey
	return r, nil
}

// AddResult reports what an add did. Existing is true when a replayed submission key returned the
// collectible that key already created, rather than creating another (FR-047).
type AddResult struct {
	Row      Row
	Existing bool
}

// Add inserts one collectible and records its submission key, in a single transaction.
//
// If the key has been seen within the window, the collectible it created is returned instead and
// nothing is inserted. Two concurrent requests carrying one key cannot both succeed: the primary
// key on (collector_id, submission_key) rejects the second, and that path then reads back the
// first one's collectible. A check-then-insert would let both through.
func (s *Store) Add(
	ctx context.Context,
	collectorID uuid.UUID,
	v collectible.Validated,
	window time.Duration,
) (AddResult, error) {
	// Fast path: an already-known key, still inside the window.
	if row, ok, err := s.findBySubmissionKey(ctx, collectorID, v.SubmissionKey, window); err != nil {
		return AddResult{}, err
	} else if ok {
		return AddResult{Row: row, Existing: true}, nil
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return AddResult{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var price *string
	if v.PurchasePrice != nil {
		text := v.PurchasePrice.String()
		price = &text
	}

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO collectibles (
			collector_id, name, collection_status,
			"character", series, manufacturer, category, scale, edition,
			purchase_price, purchase_date, release_date, notes, image_id
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8, $9,
			$10::numeric, $11, $12, $13, $14
		) RETURNING id`,
		collectorID, v.Name, string(v.Status),
		v.Character, v.Series, v.Manufacturer, v.Category, v.Scale, v.Edition,
		price, v.PurchaseDate, v.ReleaseDate, v.Notes, v.ImageID,
	).Scan(&id)
	if err != nil {
		// The composite foreign key refuses an image belonging to another collector. That is the
		// database enforcing FR-015, and it reaches the collector as an unknown-image violation
		// rather than an internal error.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return AddResult{}, ErrUnknownImage
		}
		return AddResult{}, fmt.Errorf("insert collectible: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO collectible_submissions (collector_id, submission_key, collectible_id)
		VALUES ($1, $2, $3)`,
		collectorID, v.SubmissionKey, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			// A concurrent request carrying the same key won the race. Abandon this insert and
			// return theirs, so exactly one collectible exists for the key.
			_ = tx.Rollback(ctx)
			row, ok, findErr := s.findBySubmissionKey(ctx, collectorID, v.SubmissionKey, window)
			if findErr != nil {
				return AddResult{}, findErr
			}
			if !ok {
				return AddResult{}, fmt.Errorf("submission key conflicted but no collectible was found")
			}
			return AddResult{Row: row, Existing: true}, nil
		}
		return AddResult{}, fmt.Errorf("record submission: %w", err)
	}

	row, err := scanRow(tx.QueryRow(ctx, `
		SELECT `+collectibleColumns+`
		FROM collectibles c
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE c.id = $1 AND c.collector_id = $2`, id, collectorID))
	if err != nil {
		return AddResult{}, fmt.Errorf("read back collectible: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return AddResult{}, fmt.Errorf("commit: %w", err)
	}
	return AddResult{Row: row}, nil
}

// ErrUnknownImage means the referenced image does not exist for this collector. Deliberately the
// same answer whether it never existed or belongs to someone else (FR-027).
var ErrUnknownImage = errors.New("unknown image for this collector")

func (s *Store) findBySubmissionKey(
	ctx context.Context, collectorID uuid.UUID, key string, window time.Duration,
) (Row, bool, error) {
	row, err := scanRow(s.pool.QueryRow(ctx, `
		SELECT `+collectibleColumns+`
		FROM collectible_submissions s
		JOIN collectibles c ON c.id = s.collectible_id AND c.collector_id = s.collector_id
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE s.collector_id = $1 AND s.submission_key = $2 AND s.created_at > now() - $3::interval`,
		collectorID, key, window.String()))
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, false, nil
	}
	if err != nil {
		return Row{}, false, fmt.Errorf("look up submission key: %w", err)
	}
	return row, true, nil
}

// Page is one page of a collection.
type Page struct {
	Rows            []Row
	NextCursor      string
	TotalUnfiltered int
}

// List returns one page in creation order, newest first, optionally narrowed to one status.
//
// Keyset pagination on (created_at, id): page cost stays constant as a collection grows, and the
// id tiebreaker makes the order total, so entries created in the same instant cannot be skipped or
// repeated across pages (research.md Decision 7).
func (s *Store) List(
	ctx context.Context,
	collectorID uuid.UUID,
	status *collectible.CollectionStatus,
	cursor string,
	limit int,
) (Page, error) {
	var (
		afterTime *time.Time
		afterID   *uuid.UUID
	)
	if cursor != "" {
		t, id, err := decodeCursor(cursor)
		if err != nil {
			return Page{}, err
		}
		afterTime, afterID = &t, &id
	}

	var statusArg *string
	if status != nil {
		v := string(*status)
		statusArg = &v
	}

	// One extra row tells us whether a further page exists, without a second count.
	rows, err := s.pool.Query(ctx, `
		SELECT `+collectibleColumns+`
		FROM collectibles c
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE c.collector_id = $1
		  AND ($2::text IS NULL OR c.collection_status = $2)
		  AND ($3::timestamptz IS NULL OR (c.created_at, c.id) < ($3, $4))
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT $5`,
		collectorID, statusArg, afterTime, afterID, limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("list collectibles: %w", err)
	}
	defer rows.Close()

	var out []Row
	for rows.Next() {
		r, err := scanRow(rows)
		if err != nil {
			return Page{}, fmt.Errorf("scan collectible: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("read collectibles: %w", err)
	}

	page := Page{}
	if len(out) > limit {
		last := out[limit-1]
		page.NextCursor = encodeCursor(last.Collectible.CreatedAt, last.Collectible.ID)
		out = out[:limit]
	}
	page.Rows = out

	// The second of the page's two statements. It is what lets the frontend tell an empty vault
	// (FR-041) from a filter matching nothing (FR-040) without another request. Constant work per
	// page load, not per entry.
	if err := s.pool.QueryRow(ctx,
		`SELECT count(*) FROM collectibles WHERE collector_id = $1`, collectorID,
	).Scan(&page.TotalUnfiltered); err != nil {
		return Page{}, fmt.Errorf("count collection: %w", err)
	}
	return page, nil
}

// Cursors are opaque to the client: an encoding detail, not an API contract.
func encodeCursor(t time.Time, id uuid.UUID) string {
	return base64.RawURLEncoding.EncodeToString(
		[]byte(strconv.FormatInt(t.UTC().UnixNano(), 10) + "|" + id.String()))
}

func decodeCursor(raw string) (time.Time, uuid.UUID, error) {
	invalid := errors.New("cursor is not valid")
	data, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return time.Time{}, uuid.Nil, invalid
	}
	nanos, idPart, ok := strings.Cut(string(data), "|")
	if !ok {
		return time.Time{}, uuid.Nil, invalid
	}
	n, err := strconv.ParseInt(nanos, 10, 64)
	if err != nil {
		return time.Time{}, uuid.Nil, invalid
	}
	id, err := uuid.Parse(idPart)
	if err != nil {
		return time.Time{}, uuid.Nil, invalid
	}
	return time.Unix(0, n).UTC(), id, nil
}
