// Package collection is Vaultory's application layer: the use cases a collector performs.
//
// It composes the domain, the store, and the image store. It holds no SQL and no HTTP — the
// domain decides what is valid, the store decides how it is written, and this package decides the
// order things happen in (Constitution Principle II).
package collection

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/joshuel09/vaultory/backend/internal/imagestore"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Service carries the use cases.
type Service struct {
	store             *postgres.Store
	images            imagestore.Store
	idempotencyWindow time.Duration
	now               func() time.Time

	// Work an upload starts but does not wait for: the abandoned-upload sweep (feature 009,
	// FR-009). sweeping keeps it to one at a time; background lets shutdown and tests wait for it.
	sweeping   atomic.Bool
	background sync.WaitGroup
}

func NewService(
	store *postgres.Store,
	images imagestore.Store,
	idempotencyWindow time.Duration,
) *Service {
	return &Service{
		store:             store,
		images:            images,
		idempotencyWindow: idempotencyWindow,
		now:               time.Now,
	}
}

// WaitForBackground returns once work the service started in the background has finished. Each
// piece is bounded (see sweepAfterUpload), so this does not wait indefinitely.
func (s *Service) WaitForBackground() { s.background.Wait() }

// SetClock replaces the clock. Tests use it so date rules are deterministic rather than dependent
// on when the suite happens to run.
func (s *Service) SetClock(now func() time.Time) { s.now = now }
