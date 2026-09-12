// Package collection is Vaultory's application layer: the use cases a collector performs.
//
// It composes the domain, the store, and the image store. It holds no SQL and no HTTP — the
// domain decides what is valid, the store decides how it is written, and this package decides the
// order things happen in (Constitution Principle II).
package collection

import (
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

// SetClock replaces the clock. Tests use it so date rules are deterministic rather than dependent
// on when the suite happens to run.
func (s *Service) SetClock(now func() time.Time) { s.now = now }
