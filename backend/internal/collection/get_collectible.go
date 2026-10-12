package collection

import (
	"context"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Get reads one of this collector's collectibles, with every stored value and its current version
// (FR-001, FR-002).
//
// It is deliberately thin. The store's query carries collector_id as a predicate, so there is no
// authorization decision left to make here — a collectible belonging to someone else comes back as
// postgres.ErrNotFound, indistinguishable from one that never existed (FR-030, FR-031).
//
// The representation is the same one the gallery returns. There is no separate detail shape to
// drift from the list shape.
func (s *Service) Get(
	ctx context.Context, collectorID, collectibleID uuid.UUID,
) (postgres.Row, error) {
	return s.store.Get(ctx, collectorID, collectibleID)
}
