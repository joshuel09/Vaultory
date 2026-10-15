package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/identity"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
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

	// 008 FR-031: the filters behind a breakdown group's link.
	//
	// q.Has, not q.Get. An absent parameter and a blank one are different requests here: absent is
	// every series, blank is the collectibles recording no series at all — the unspecified group.
	// q.Get returns "" for both, so reading these the obvious way makes that group's link quietly
	// return the entire vault.
	series, bad := groupFilter(q, "series")
	if bad != nil {
		WriteValidationFailed(w, []FieldError{*bad})
		return
	}
	manufacturer, bad := groupFilter(q, "manufacturer")
	if bad != nil {
		WriteValidationFailed(w, []FieldError{*bad})
		return
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

	filter := postgres.ListFilter{Status: status, Series: series, Manufacturer: manufacturer}
	page, err := s.service.List(r.Context(), collector, filter, q.Get("cursor"), limit)
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

// groupFilter reads one of the breakdown filters, preserving the difference between absent and
// blank.
//
// The length bound is the contract's, and matches the column: a longer value cannot match anything,
// so refusing it says so rather than returning an empty gallery that looks like an empty vault.
func groupFilter(q url.Values, name string) (*string, *FieldError) {
	if !q.Has(name) {
		return nil, nil
	}
	v := q.Get(name)
	if len(v) > maxGroupFilterLength {
		return nil, &FieldError{
			Field:   name,
			Code:    FieldTooLong,
			Message: "That filter is longer than any value a collectible can hold.",
		}
	}
	return &v, nil
}

// maxGroupFilterLength matches the contract's maxLength for the series and manufacturer parameters,
// which in turn matches what those fields accept.
const maxGroupFilterLength = 200

// handleGetCollectible reads one of the acting collector's collectibles, which is what the edit
// screen is filled from (FR-001, FR-002).
func (s *Server) handleGetCollectible(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	id, err := uuid.Parse(r.PathValue("collectibleId"))
	if err != nil {
		// A malformed identifier gets the same answer as one that does not exist, so probing with
		// rubbish learns nothing either (FR-031).
		WriteNotFound(w)
		return
	}

	row, err := s.service.Get(r.Context(), collector, id)
	if errors.Is(err, postgres.ErrNotFound) {
		// 404, never 403. A 403 would confirm the collectible exists and belongs to someone, which
		// is exactly what a collector's private vault must not disclose (FR-031).
		WriteNotFound(w)
		return
	}
	if err != nil {
		WriteInternal(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toCollectibleResponse(row))
}

// handleEditCollectible replaces every attribute of one collectible (FR-004).
func (s *Server) handleEditCollectible(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	if ct := r.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "application/json") {
		WriteError(w, http.StatusUnsupportedMediaType, CodeUnsupportedMedia,
			"Send this request as JSON.")
		return
	}

	id, err := uuid.Parse(r.PathValue("collectibleId"))
	if err != nil {
		WriteNotFound(w)
		return
	}

	var req editCollectibleRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxJSONBody))
	// The contract declares additionalProperties: false, and refusing unknown fields is what makes
	// that real rather than aspirational.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			WriteError(w, http.StatusBadRequest, CodeValidationFailed, "That submission is too large.")
			return
		}
		WriteError(w, http.StatusBadRequest, CodeValidationFailed,
			"That request could not be read as JSON.")
		return
	}
	if err := dec.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		WriteError(w, http.StatusBadRequest, CodeValidationFailed,
			"That request body contained more than one JSON value.")
		return
	}

	row, violations, err := s.service.Edit(r.Context(), collector, id, req.toEditDraft())
	if len(violations) > 0 {
		WriteValidationFailed(w, toFieldErrors(violations))
		return
	}
	var conflict *postgres.VersionConflictError
	switch {
	case errors.As(err, &conflict):
		// Somebody changed it first. The collector is shown what it now says rather than having
		// their edit applied over it (FR-027).
		WriteVersionConflict(w, toCollectibleResponse(conflict.Current))
		return
	case errors.Is(err, postgres.ErrNotFound):
		// Gone, or never theirs. 404, never 403 (FR-031).
		WriteNotFound(w)
		return
	case err != nil:
		WriteInternal(w, r, err)
		return
	}

	WriteJSON(w, http.StatusOK, toCollectibleResponse(row))
}

// handleDeleteCollectible removes one collectible permanently (FR-022).
//
// Answers 204 in every case for an authenticated collector. There is no 404 here, deliberately:
// a collectible that is not in this collector's vault — already deleted, never existed, or
// somebody else's — means the state they asked for already holds (FR-025), and answering
// differently for an identifier that happens to be real would disclose that it is (FR-031).
func (s *Server) handleDeleteCollectible(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	id, err := uuid.Parse(r.PathValue("collectibleId"))
	if err != nil {
		// Even a malformed identifier. It names nothing in anybody's vault, which is the condition
		// this operation reports success for.
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if err := s.service.Delete(r.Context(), collector, id); err != nil {
		WriteInternal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
