package collection

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Edit changes a collectible already in the collector's vault.
//
// Violations come back as a slice rather than an error, because there is usually more than one and
// the collector should see them together (FR-014). A stale version and a missing collectible come
// back as errors, because they are not something the collector can fix by correcting a field.
func (s *Service) Edit(
	ctx context.Context,
	collectorID, collectibleID uuid.UUID,
	draft collectible.EditDraft,
) (postgres.Row, []collectible.Violation, error) {
	validated, violations := draft.Validate(s.now())
	if len(violations) > 0 {
		return postgres.Row{}, violations, nil
	}

	// A referenced image must be this collector's own (FR-021). The composite foreign key is the
	// real guarantee; checking first is what turns a constraint breach into a field-level message
	// the collector can act on.
	if validated.ImageID != nil {
		ok, err := s.store.ImageExistsForCollector(ctx, collectorID, *validated.ImageID)
		if err != nil {
			return postgres.Row{}, nil, err
		}
		if !ok {
			// Deliberately the same message whether the image never existed or belongs to someone
			// else: FR-031 forbids revealing which.
			return postgres.Row{}, []collectible.Violation{{
				Field:   "imageId",
				Code:    collectible.CodeInvalidValue,
				Message: "That image could not be found.",
			}}, nil
		}
	}

	row, err := s.store.Edit(ctx, collectorID, collectibleID, validated)
	if errors.Is(err, postgres.ErrUnknownImage) {
		// The window between the check above and the update. Rare, but the database caught it.
		return postgres.Row{}, []collectible.Violation{{
			Field:   "imageId",
			Code:    collectible.CodeInvalidValue,
			Message: "That image could not be found.",
		}}, nil
	}
	if err != nil {
		return postgres.Row{}, nil, err
	}

	// An edit that replaced or removed a photograph has just orphaned its files. Draining here
	// works the queue exactly when it grows, which is why no background ticker is needed — and it
	// is deliberately after the commit and deliberately unable to fail the edit (FR-020a).
	s.DrainImageDeletions(ctx)

	return row, nil, nil
}
