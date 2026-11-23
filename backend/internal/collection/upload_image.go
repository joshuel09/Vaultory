package collection

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

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

	// Every upload is a potential abandoned one, so this is where abandoned ones are reclaimed —
	// a bounded batch, never this one (it is seconds old, not thirty days), and never able to turn
	// this upload into a failure (feature 009, FR-009, FR-010). After this upload is answered, not
	// inside it: see sweepAfterUpload.
	s.sweepAfterUpload(ctx)
	return img, nil
}

// sweepTimeout bounds one background sweep and its drain, and so how long shutdown waits for one.
const sweepTimeout = time.Minute

// sweepAfterUpload starts one sweep and drain in the background, unless one is already running.
//
// Not in the request, because the sweep's cost is not the batch it removes but the proof that
// nothing else qualifies: that reads every image and every reference, so it grows with the whole
// service — 93 ms at 200,000 images with nothing to sweep — and an upload was paying it (#72).
// One at a time, because a second would read the same backlog, and FOR UPDATE SKIP LOCKED would
// only make it find less. An upload arriving meanwhile starts nothing; the next one will.
//
// The context outlives the request — the response is already sent — but keeps its values, and the
// sweep's own bound applies.
func (s *Service) sweepAfterUpload(ctx context.Context) {
	if !s.sweeping.CompareAndSwap(false, true) {
		return
	}
	s.background.Add(1)
	go func() {
		defer s.background.Done()
		defer s.sweeping.Store(false)
		ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), sweepTimeout)
		defer cancel()
		if s.SweepAbandonedUploads(ctx) > 0 {
			s.DrainImageDeletions(ctx)
		}
	}()
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
