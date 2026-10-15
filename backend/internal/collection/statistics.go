package collection

import (
	"context"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// Statistics summarises the acting collector's whole vault (FR-007, FR-008, FR-014).
//
// Thin, like Get, and for the same reason: the store's statements carry collector_id as a predicate,
// so there is no authorization decision left to make here. There is also no identifier to validate —
// this operation takes none, which is what makes another collector's figures unrequestable rather
// than merely refused (FR-003).
//
// Every figure is finished by the time it leaves here. Nothing downstream sums, counts, or ranks
// anything; FR-001 puts that squarely in Go.
func (s *Service) Statistics(
	ctx context.Context, collectorID uuid.UUID,
) (collectible.CollectionStatistics, error) {
	return s.store.Statistics(ctx, collectorID)
}
