package collection

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// AddResult is what an add produced.
//
// Existing is true when a replayed submission key returned the collectible it had already created.
// The transport does not vary its response on this — the client cannot tell the two apart and has
// no need to; in both cases exactly one collectible exists for that key (FR-047).
type AddResult struct {
	Row      postgres.Row
	Existing bool
}

// Add validates a draft and records the collectible.
//
// Violations come back as a slice rather than an error, because there is usually more than one and
// the collector should see them together (FR-020).
func (s *Service) Add(
	ctx context.Context, collectorID uuid.UUID, draft collectible.Draft,
) (AddResult, []collectible.Violation, error) {
	validated, violations := draft.Validate(s.now())
	if len(violations) > 0 {
		return AddResult{}, violations, nil
	}

	// A referenced image must be this collector's own (FR-015). The composite foreign key is the
	// real guarantee; checking first is what turns a constraint breach into a field-level message
	// the collector can act on.
	if validated.ImageID != nil {
		ok, err := s.store.ImageExistsForCollector(ctx, collectorID, *validated.ImageID)
		if err != nil {
			return AddResult{}, nil, err
		}
		if !ok {
			// Deliberately the same message whether the image never existed or belongs to someone
			// else: FR-027 forbids revealing which.
			return AddResult{}, []collectible.Violation{{
				Field:   "imageId",
				Code:    collectible.CodeInvalidValue,
				Message: "That image could not be found.",
			}}, nil
		}
	}

	res, err := s.store.Add(ctx, collectorID, validated, s.idempotencyWindow)
	if errors.Is(err, postgres.ErrUnknownImage) {
		// The window between the check above and the insert. Rare, but the database caught it.
		return AddResult{}, []collectible.Violation{{
			Field:   "imageId",
			Code:    collectible.CodeInvalidValue,
			Message: "That image could not be found.",
		}}, nil
	}
	if err != nil {
		return AddResult{}, nil, err
	}
	return AddResult{Row: res.Row, Existing: res.Existing}, nil, nil
}
