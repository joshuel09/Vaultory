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
	c.image_id, i.rendition_key, c.created_at, c.version`

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
		&c.ImageID, &renditionKey, &c.CreatedAt, &c.Version,
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
		if isImageInUse(err) {
			return AddResult{}, ErrImageInUse
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

// ErrImageInUse means the referenced image already belongs to another collectible — possibly one in
// the Trash. A photograph belongs to at most one collectible, because destroying a collectible
// deletes its image row, and a second collectible sharing it would silently lose its photograph
// (feature 010, FR-018).
var ErrImageInUse = errors.New("image is already used by another collectible")

// imageUniqueConstraint is the index that enforces ErrImageInUse (migration 000012).
const imageUniqueConstraint = "collectibles_image_unique"

func isImageInUse(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation && pgErr.ConstraintName == imageUniqueConstraint
}

// findBySubmissionKey looks for a collectible already created under this key, inside the window.
//
// The window is passed as seconds through make_interval rather than as a duration string: Go
// renders 24h as "24h0m0s", and relying on PostgreSQL's interval tokenizer to read that is a
// dependency worth not having when the alternative is exact.
func (s *Store) findBySubmissionKey(
	ctx context.Context, collectorID uuid.UUID, key string, window time.Duration,
) (Row, bool, error) {
	row, err := scanRow(s.pool.QueryRow(ctx, `
		SELECT `+collectibleColumns+`
		FROM collectible_submissions s
		JOIN collectibles c ON c.id = s.collectible_id AND c.collector_id = s.collector_id
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE s.collector_id = $1 AND s.submission_key = $2
		  AND s.created_at > now() - make_interval(secs => $3)
		  AND c.deleted_at IS NULL`,
		collectorID, key, window.Seconds()))
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, false, nil
	}
	if err != nil {
		return Row{}, false, fmt.Errorf("look up submission key: %w", err)
	}
	return row, true, nil
}

// Get reads one collectible belonging to this collector.
//
// collector_id is part of the WHERE clause rather than something checked afterwards, so there is
// no moment at which this code holds another collector's row and has yet to decide what to do with
// it. Absent and someone-else's are the same answer (FR-030, FR-031).
//
// A collectible in the Trash is not in the vault, so it is absent here too (feature 010, FR-002).
func (s *Store) Get(ctx context.Context, collectorID, collectibleID uuid.UUID) (Row, error) {
	row, err := scanRow(s.pool.QueryRow(ctx, `
		SELECT `+collectibleColumns+`
		FROM collectibles c
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE c.id = $1 AND c.collector_id = $2 AND c.deleted_at IS NULL`, collectibleID, collectorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Row{}, ErrNotFound
	}
	if err != nil {
		return Row{}, fmt.Errorf("get collectible: %w", err)
	}
	return row, nil
}

// releaseImage deletes an image row and queues its files for removal, inside the caller's
// transaction.
//
// Deleting the row is what makes the image unfetchable, and it is the whole of the privacy
// guarantee (FR-020). A rendition is authorized against collectible_images.collector_id on its
// own — it has to be, because the upload preview fetches one before any collectible references it
// — so merely unlinking an image would leave it readable by its owner indefinitely.
//
// The files are a separate matter. They are queued rather than deleted here because a transaction
// cannot roll back a filesystem, and because a storage fault must never be able to stop a
// collector deleting something (FR-020a, research.md Decisions 4 and 5).
func releaseImage(ctx context.Context, tx pgx.Tx, collectorID, imageID uuid.UUID) error {
	var originalKey, renditionKey string
	err := tx.QueryRow(ctx, `
		DELETE FROM collectible_images
		WHERE id = $1 AND collector_id = $2
		RETURNING original_key, rendition_key`, imageID, collectorID,
	).Scan(&originalKey, &renditionKey)
	if errors.Is(err, pgx.ErrNoRows) {
		// Already gone. Nothing to queue, and nothing wrong — a concurrent edit may have released
		// it first, and the end state is the one we wanted.
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete image row: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO pending_image_deletions (image_id, collector_id, original_key, rendition_key)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (image_id) DO NOTHING`,
		imageID, collectorID, originalKey, renditionKey)
	if err != nil {
		return fmt.Errorf("queue image deletion: %w", err)
	}
	return nil
}

// VersionConflictError means the collectible changed after the collector opened it.
//
// It carries the collectible as it now stands, because the collector has to be shown what it
// actually says, and making them fetch it again would open a second window in which it changes
// (FR-027).
type VersionConflictError struct {
	Current Row
}

func (e *VersionConflictError) Error() string {
	return fmt.Sprintf("collectible %s is at version %d",
		e.Current.Collectible.ID, e.Current.Collectible.Version)
}

// Edit replaces every attribute of one collectible, if nobody has changed it in the meantime.
//
// The shape here is deliberate. The obvious implementation —
//
//	UPDATE collectibles SET … WHERE id = $1 AND collector_id = $2 AND version = $3
//
// reports zero rows affected for three different situations: no such collectible, somebody else's
// collectible, and a version that has moved on. The first two must answer 404 and the third 409,
// so collapsing them loses the distinction the requirements depend on (research.md Decision 6).
//
// SELECT … FOR UPDATE also serialises concurrent edits of the same row. Without it, two
// transactions both read version 3, both find it current, and both write — one collector's work
// disappearing with nobody told, which is the thing Principle IV forbids.
func (s *Store) Edit(
	ctx context.Context,
	collectorID, collectibleID uuid.UUID,
	v collectible.ValidatedEdit,
) (Row, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Row{}, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Lock the row and read what it currently holds. The image id comes back too, which is what
	// tells us whether the photograph is being replaced or removed.
	var (
		currentVersion int
		currentImageID *uuid.UUID
	)
	err = tx.QueryRow(ctx, `
		SELECT version, image_id FROM collectibles
		WHERE id = $1 AND collector_id = $2 AND deleted_at IS NULL
		FOR UPDATE`, collectibleID, collectorID).Scan(&currentVersion, &currentImageID)
	if errors.Is(err, pgx.ErrNoRows) {
		// Absent, another collector's, or in the Trash. Deliberately the same answer (FR-031). An
		// edit form left open across a deletion is refused here rather than editing — or
		// resurrecting — something the collector deleted (feature 010).
		return Row{}, ErrNotFound
	}
	if err != nil {
		return Row{}, fmt.Errorf("lock collectible: %w", err)
	}

	if currentVersion != v.ExpectedVersion {
		// Read the current state through the same transaction so what the collector is shown is
		// the row we just locked, not a third version that arrived in between.
		current, readErr := scanRow(tx.QueryRow(ctx, `
			SELECT `+collectibleColumns+`
			FROM collectibles c
			LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
			WHERE c.id = $1 AND c.collector_id = $2`, collectibleID, collectorID))
		if readErr != nil {
			return Row{}, fmt.Errorf("read the current collectible: %w", readErr)
		}
		return Row{}, &VersionConflictError{Current: current}
	}

	var price *string
	if v.PurchasePrice != nil {
		text := v.PurchasePrice.String()
		price = &text
	}

	_, err = tx.Exec(ctx, `
		UPDATE collectibles SET
			name = $3, collection_status = $4,
			"character" = $5, series = $6, manufacturer = $7, category = $8,
			scale = $9, edition = $10,
			purchase_price = $11::numeric, purchase_date = $12, release_date = $13,
			notes = $14, image_id = $15,
			version = version + 1
		WHERE id = $1 AND collector_id = $2`,
		collectibleID, collectorID,
		v.Name, string(v.Status),
		v.Character, v.Series, v.Manufacturer, v.Category,
		v.Scale, v.Edition,
		price, v.PurchaseDate, v.ReleaseDate,
		v.Notes, v.ImageID,
	)
	if err != nil {
		// The composite foreign key refusing an image that is not this collector's. Reaches the
		// collector as an unknown-image violation rather than an internal error (FR-021).
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return Row{}, ErrUnknownImage
		}
		if isImageInUse(err) {
			return Row{}, ErrImageInUse
		}
		return Row{}, fmt.Errorf("update collectible: %w", err)
	}

	// The photograph is gone from this collectible — replaced or removed. Deleting its row is what
	// makes it unfetchable, and it happens here, inside the same transaction as the edit, so the
	// two cannot disagree (FR-020).
	if currentImageID != nil && (v.ImageID == nil || *v.ImageID != *currentImageID) {
		if err := releaseImage(ctx, tx, collectorID, *currentImageID); err != nil {
			return Row{}, err
		}
	}

	row, err := scanRow(tx.QueryRow(ctx, `
		SELECT `+collectibleColumns+`
		FROM collectibles c
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE c.id = $1 AND c.collector_id = $2`, collectibleID, collectorID))
	if err != nil {
		return Row{}, fmt.Errorf("read back collectible: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Row{}, fmt.Errorf("commit: %w", err)
	}
	return row, nil
}

// Trash moves one collectible to the Trash (feature 010, FR-001).
//
// Nothing about the row changes except deleted_at — not its id, its created_at, its version, its
// attributes or its photograph — which is what lets a restore return it exactly as it was, in its
// original gallery position (research.md Decisions 1 and 2).
//
// Reports whether it moved. Something absent, another collector's, or already in the Trash is left
// alone, and its 30 days are not restarted (FR-004). The caller answers success either way (006
// FR-025) but records only a real move (FR-015).
//
// Its add's idempotency record goes in the same transaction. Otherwise a retried add carrying the
// same submission key would find this collectible and hand back something the collector has deleted
// (research.md Decision 5).
func (s *Store) Trash(ctx context.Context, collectorID, collectibleID uuid.UUID) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		UPDATE collectibles SET deleted_at = now()
		WHERE id = $1 AND collector_id = $2 AND deleted_at IS NULL`, collectibleID, collectorID)
	if err != nil {
		return false, fmt.Errorf("move to trash: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM collectible_submissions
		WHERE collectible_id = $1 AND collector_id = $2`, collectibleID, collectorID); err != nil {
		return false, fmt.Errorf("forget submission: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	return true, nil
}

// Restore returns one collectible from the Trash to the vault, if it was deleted within retention
// (feature 010, FR-007).
//
// ErrNotFound for anything else — absent, another collector's, never deleted, already restored,
// destroyed, or older than retention — without saying which (FR-008). The cutoff is the same one
// the listing and the purge use, so nothing is listed but unrestorable or the reverse (research.md
// Decision 3).
//
// Against the purge: if the purge locked the row first, this UPDATE waits, then finds it gone. If
// this commits first, the purge's re-check sees deleted_at NULL and passes over it (FR-012).
func (s *Store) Restore(
	ctx context.Context, collectorID, collectibleID uuid.UUID, retention time.Duration,
) (Row, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE collectibles SET deleted_at = NULL
		WHERE id = $1 AND collector_id = $2
		  AND deleted_at IS NOT NULL
		  AND deleted_at > now() - make_interval(secs => $3)`,
		collectibleID, collectorID, retention.Seconds())
	if err != nil {
		return Row{}, fmt.Errorf("restore: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return Row{}, ErrNotFound
	}
	return s.Get(ctx, collectorID, collectibleID)
}

// DeletePermanently destroys one collectible that is in the Trash, and its photograph, now
// (feature 010, FR-011).
//
// This is feature 006's deletion, moved: the image row goes in the same transaction, so nothing can
// observe the collectible destroyed and its photograph still fetchable (006 FR-020). The files are
// queued and drained afterwards (006 FR-020a).
//
// Only from the Trash. A collectible in the vault has to be moved there first, so there is one way
// to delete and one confirmation for it. Anything not in this collector's Trash is left alone, and
// the caller answers success either way, as for any deletion (006 FR-025).
func (s *Store) DeletePermanently(ctx context.Context, collectorID, collectibleID uuid.UUID) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var imageID *uuid.UUID
	err = tx.QueryRow(ctx, `
		DELETE FROM collectibles
		WHERE id = $1 AND collector_id = $2 AND deleted_at IS NOT NULL
		RETURNING image_id`, collectibleID, collectorID).Scan(&imageID)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("delete collectible: %w", err)
	}
	if imageID != nil {
		if err := releaseImage(ctx, tx, collectorID, *imageID); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit: %w", err)
	}
	return true, nil
}

// Page is one page of a collection.
type Page struct {
	Rows            []Row
	NextCursor      string
	TotalUnfiltered int
}

// ListFilter narrows a page of a collection. A nil field is not a filter.
//
// Series and Manufacturer are pointers for a reason that is easy to miss: nil and a pointer to the
// empty string mean different things. nil is "every series"; empty is "the collectibles recording no
// series at all", which is the group the statistics label unspecified. Collapsing the two — which is
// what a plain string does, because Go cannot tell an absent query parameter from a blank one —
// makes the unspecified group's link return the whole vault (008 FR-031, research.md Decision 8).
type ListFilter struct {
	Status       *collectible.CollectionStatus
	Series       *string
	Manufacturer *string
}

// List returns one page in creation order, newest first, optionally narrowed.
//
// Keyset pagination on (created_at, id): page cost stays constant as a collection grows, and the
// id tiebreaker makes the order total, so entries created in the same instant cannot be skipped or
// repeated across pages (research.md Decision 7).
//
// The series and manufacturer predicates normalise both sides through PostgreSQL, in exactly the
// form the statistics breakdown groups by: coalesce first, then btrim, then lower. Normalising the
// parameter in Go instead would be a second implementation of the same rule, and Go's ToLower and
// PostgreSQL's collation-dependent lower() do not have to agree on every input. When they disagree,
// a breakdown group and the gallery it links to hold different collectibles (008 SC-010).
func (s *Store) List(
	ctx context.Context,
	collectorID uuid.UUID,
	filter ListFilter,
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
	if filter.Status != nil {
		v := string(*filter.Status)
		statusArg = &v
	}

	// One extra row tells us whether a further page exists, without a second count.
	rows, err := s.pool.Query(ctx, `
		SELECT `+collectibleColumns+`
		FROM collectibles c
		LEFT JOIN collectible_images i ON i.id = c.image_id AND i.collector_id = c.collector_id
		WHERE c.collector_id = $1
		  AND c.deleted_at IS NULL
		  AND ($2::text IS NULL OR c.collection_status = $2)
		  AND ($3::timestamptz IS NULL OR (c.created_at, c.id) < ($3, $4))
		  AND ($6::text IS NULL OR lower(btrim(coalesce(c.series, ''))) = lower(btrim($6)))
		  AND ($7::text IS NULL OR lower(btrim(coalesce(c.manufacturer, ''))) = lower(btrim($7)))
		ORDER BY c.created_at DESC, c.id DESC
		LIMIT $5`,
		collectorID, statusArg, afterTime, afterID, limit+1,
		filter.Series, filter.Manufacturer)
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
		`SELECT count(*) FROM collectibles WHERE collector_id = $1 AND deleted_at IS NULL`, collectorID,
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
