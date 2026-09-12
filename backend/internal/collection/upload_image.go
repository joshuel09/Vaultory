package collection

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/imaging"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Upload failures a collector can act on. Everything else is an internal error.
var (
	ErrImageTooLarge    = errors.New("image is too large")
	ErrImageUnsupported = errors.New("image format is not supported")
)

// UploadImage validates an upload, derives the gallery rendition, stores both, and records the
// image against the uploading collector.
//
// Uploading is separate from adding a collectible precisely so a refused image never blocks saving
// the rest of it (FR-013, research.md Decision 3). A refusal here leaves nothing behind: no bytes
// in the store and no row in the database.
func (s *Service) UploadImage(
	ctx context.Context, collectorID uuid.UUID, r io.Reader,
) (postgres.Image, error) {
	decoded, err := imaging.Decode(r)
	switch {
	case errors.Is(err, imaging.ErrTooLarge):
		return postgres.Image{}, ErrImageTooLarge
	case errors.Is(err, imaging.ErrUnsupportedFormat):
		return postgres.Image{}, ErrImageUnsupported
	case err != nil:
		return postgres.Image{}, fmt.Errorf("decode upload: %w", err)
	}

	rendition, err := imaging.DeriveRendition(decoded.Image)
	if err != nil {
		return postgres.Image{}, fmt.Errorf("derive rendition: %w", err)
	}

	// Keys are generated here, never derived from anything a collector supplied, so no upload can
	// name a location.
	id := uuid.New()
	originalKey := fmt.Sprintf("%s/%s/original", collectorID, id)
	renditionKey := fmt.Sprintf("%s/%s/rendition.jpg", collectorID, id)

	if err := s.images.Put(ctx, originalKey, decoded.Bytes); err != nil {
		return postgres.Image{}, fmt.Errorf("store original: %w", err)
	}
	if err := s.images.Put(ctx, renditionKey, rendition); err != nil {
		// Do not leave the original behind for a rendition that never existed.
		_ = s.images.Delete(ctx, originalKey)
		return postgres.Image{}, fmt.Errorf("store rendition: %w", err)
	}

	img, err := s.store.InsertImage(ctx, postgres.Image{
		CollectorID:  collectorID,
		OriginalKey:  originalKey,
		RenditionKey: renditionKey,
		ContentType:  decoded.ContentType,
		ByteSize:     int64(len(decoded.Bytes)),
		Width:        decoded.Width,
		Height:       decoded.Height,
	})
	if err != nil {
		// The row is the record of truth; without it the bytes are unreachable and unowned.
		_ = s.images.Delete(ctx, originalKey)
		_ = s.images.Delete(ctx, renditionKey)
		return postgres.Image{}, err
	}
	return img, nil
}

// OpenRendition returns the gallery rendition of one image, for this collector only.
//
// Authorization happens on every request, in the query itself: holding the address grants nothing
// (FR-015). A rendition belonging to someone else is reported as not found, never as forbidden
// (FR-027).
func (s *Service) OpenRendition(
	ctx context.Context, collectorID, imageID uuid.UUID,
) (io.ReadCloser, error) {
	img, err := s.store.GetImage(ctx, collectorID, imageID)
	if err != nil {
		return nil, err // postgres.ErrNotFound flows through unchanged
	}
	rc, err := s.images.Open(ctx, img.RenditionKey)
	if err != nil {
		return nil, fmt.Errorf("open rendition: %w", err)
	}
	return rc, nil
}
