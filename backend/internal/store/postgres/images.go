package postgres

import (
	"context"
	"errors"
	"fmt"

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
