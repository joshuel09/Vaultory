package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/identity"
)

// maxJSONBody bounds a collectible submission. Generous for the largest legitimate one — a 2000
// character note plus every other field — and small enough that a runaway body is refused early.
const maxJSONBody = 64 * 1024

func (s *Server) handleAddCollectible(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		WriteError(w, http.StatusUnsupportedMediaType, CodeUnsupportedMedia,
			"Send this request as JSON.")
		return
	}

	var req addCollectibleRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	// The contract declares additionalProperties: false. Refusing unknown fields here is what
	// makes that real, and it catches a client that has drifted from the contract rather than
	// silently ignoring what it sent.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			WriteError(w, http.StatusBadRequest, CodeValidationFailed,
				"That submission is too large.")
			return
		}
		WriteError(w, http.StatusBadRequest, CodeValidationFailed,
			"That request could not be read as JSON.")
		return
	}
	// A second JSON value in the body means the client is not speaking the contract.
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		WriteError(w, http.StatusBadRequest, CodeValidationFailed,
			"That request body contained more than one JSON value.")
		return
	}

	result, violations, err := s.service.Add(r.Context(), collector, req.toDraft())
	if err != nil {
		WriteInternal(w, r, err)
		return
	}
	if len(violations) > 0 {
		WriteValidationFailed(w, toFieldErrors(violations))
		return
	}

	// 201 whether this created the collectible or replayed a submission key. The client cannot
	// distinguish the two and has no need to: in both cases exactly one collectible exists for
	// that key (FR-047).
	WriteJSON(w, http.StatusCreated, toCollectibleResponse(result.Row))
}

func (s *Server) handleListCollectibles(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	q := r.URL.Query()

	// FR-037: an optional status filter. An unrecognised value is refused rather than ignored —
	// silently returning everything would be a filter that lies.
	var status *collectible.CollectionStatus
	if raw := q.Get("status"); raw != "" {
		parsed, err := collectible.ParseCollectionStatus(raw)
		if err != nil {
			WriteValidationFailed(w, []FieldError{{
				Field:   "status",
				Code:    FieldInvalidValue,
				Message: "Filter by Owned, Preordered, Wishlist, or Sold.",
			}})
			return
		}
		status = &parsed
	}

	limit := 0
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > collection.MaxPageLimit {
			WriteValidationFailed(w, []FieldError{{
				Field:   "limit",
				Code:    FieldInvalidValue,
				Message: "Ask for between 1 and 48 collectibles.",
			}})
			return
		}
		limit = n
	}

	page, err := s.service.List(r.Context(), collector, status, q.Get("cursor"), limit)
	if err != nil {
		// A malformed cursor is the client's mistake, not a server failure.
		if strings.Contains(err.Error(), "cursor is not valid") {
			WriteValidationFailed(w, []FieldError{{
				Field:   "cursor",
				Code:    FieldInvalidValue,
				Message: "That page reference is not valid. Start from the beginning.",
			}})
			return
		}
		WriteInternal(w, r, err)
		return
	}

	items := make([]collectibleResponse, 0, len(page.Rows))
	for _, row := range page.Rows {
		items = append(items, toCollectibleResponse(row))
	}
	resp := collectionPageResponse{Items: items, TotalUnfiltered: page.TotalUnfiltered}
	if page.NextCursor != "" {
		resp.NextCursor = &page.NextCursor
	}
	WriteJSON(w, http.StatusOK, resp)
}
