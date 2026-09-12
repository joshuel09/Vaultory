package collection

import (
	"context"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Page size bounds. The default is one screenful of gallery cards; the maximum keeps a single
// request from asking for the whole collection, which FR-035 forbids presenting at once.
const (
	DefaultPageLimit = 24
	MaxPageLimit     = 48
)

// List returns one page of the collector's own collection, newest first, optionally narrowed to a
// single status.
//
// It never returns another collector's collectibles: the collector is a predicate in the query,
// not a filter applied afterwards (FR-026).
func (s *Service) List(
	ctx context.Context,
	collectorID uuid.UUID,
	status *collectible.CollectionStatus,
	cursor string,
	limit int,
) (postgres.Page, error) {
	switch {
	case limit <= 0:
		limit = DefaultPageLimit
	case limit > MaxPageLimit:
		limit = MaxPageLimit
	}
	return s.store.List(ctx, collectorID, status, cursor, limit)
}
