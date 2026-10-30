package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/identity"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// The Trash (feature 010). Its own collection under /api/trash, rather than a parameter on
// /api/collectibles, so no vault endpoint can return a trashed collectible because a parameter was
// missing (research.md Decision 6).

// trashedCollectibleResponse nests the collectible rather than adding fields to it: Collectible is a
// closed schema in the contract (additionalProperties: false), and one shape for a collectible
// everywhere is worth more than a flat object here.
type trashedCollectibleResponse struct {
	Collectible collectibleResponse `json:"collectible"`
	DeletedAt   string              `json:"deletedAt"`
	// PurgeAt is when it will be destroyed. Sent, not computed in the browser: retention is a rule
	// of the system (Principle II).
	PurgeAt string `json:"purgeAt"`
}

type trashPageResponse struct {
	Items      []trashedCollectibleResponse `json:"items"`
	NextCursor *string                      `json:"nextCursor"`
}

// handleListTrash returns one page of the collector's Trash, most recently deleted first (FR-006).
func (s *Server) handleListTrash(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	q := r.URL.Query()
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

	page, err := s.service.ListTrash(r.Context(), collector, q.Get("cursor"), limit)
	if err != nil {
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

	items := make([]trashedCollectibleResponse, 0, len(page.Items))
	for _, it := range page.Items {
		items = append(items, trashedCollectibleResponse{
			Collectible: toCollectibleResponse(it.Row),
			DeletedAt:   it.DeletedAt.UTC().Format(time.RFC3339),
			PurgeAt:     it.PurgeAt.UTC().Format(time.RFC3339),
		})
	}
	resp := trashPageResponse{Items: items}
	if page.NextCursor != "" {
		resp.NextCursor = &page.NextCursor
	}
	WriteJSON(w, http.StatusOK, resp)
}

// handleRestoreFromTrash returns a collectible from the Trash to the vault (FR-007).
//
// 404 for anything not in this collector's Trash within the window, and for a malformed id, all the
// same (FR-008, FR-013). Not 204-on-absent like deletion: the collector must learn whether it came
// back.
func (s *Server) handleRestoreFromTrash(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	id, err := uuid.Parse(r.PathValue("collectibleId"))
	if err != nil {
		WriteNotFound(w)
		return
	}
	row, err := s.service.Restore(r.Context(), collector, id)
	if errors.Is(err, postgres.ErrNotFound) {
		WriteNotFound(w)
		return
	}
	if err != nil {
		WriteInternal(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toCollectibleResponse(row))
}

// handleDeletePermanently destroys a collectible in the Trash, and its photograph (FR-011).
//
// 204 whatever was there, as for any deletion (006 FR-025).
func (s *Server) handleDeletePermanently(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	id, err := uuid.Parse(r.PathValue("collectibleId"))
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := s.service.DeletePermanently(r.Context(), collector, id); err != nil {
		WriteInternal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
