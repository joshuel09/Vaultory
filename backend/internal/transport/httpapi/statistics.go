package httpapi

import (
	"net/http"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/identity"
)

// statusSummaryResponse mirrors StatusSummary in the contract.
//
// Totals are strings. A JSON number would invite a floating-point conversion on the other side, and
// this is the figure where such a conversion would be least visible and most damaging.
type statusSummaryResponse struct {
	Status      string `json:"status"`
	Count       int    `json:"count"`
	PricedCount int    `json:"pricedCount"`
	Total       string `json:"total"`
}

type groupSummaryResponse struct {
	Label       string `json:"label"`
	Kind        string `json:"kind"`
	Count       int    `json:"count"`
	PricedCount int    `json:"pricedCount"`
	Total       string `json:"total"`
}

// collectionStatisticsResponse mirrors CollectionStatistics in the contract.
//
// The arrays are never null. An omitted breakdown and an empty one are different things to a client,
// and `null` would make a vault with nothing in it look like a server that failed to answer.
type collectionStatisticsResponse struct {
	TotalCount     int                     `json:"totalCount"`
	PricedCount    int                     `json:"pricedCount"`
	ByStatus       []statusSummaryResponse `json:"byStatus"`
	BySeries       []groupSummaryResponse  `json:"bySeries"`
	ByManufacturer []groupSummaryResponse  `json:"byManufacturer"`
}

// unspecifiedLabel and remainderLabel name the two groups the collector did not name themselves.
//
// They are filled in here rather than in the domain or the database because they are words shown to
// a person. A client that wants different wording has `kind` to switch on and never has to match on
// these strings.
const (
	unspecifiedLabel = "Not specified"
	remainderLabel   = "Everything else"
)

func toGroupResponses(groups []collectible.GroupSummary) []groupSummaryResponse {
	out := make([]groupSummaryResponse, 0, len(groups))
	for _, g := range groups {
		label := g.Label
		switch g.Kind {
		case collectible.GroupUnspecified:
			label = unspecifiedLabel
		case collectible.GroupRemainder:
			label = remainderLabel
		}
		out = append(out, groupSummaryResponse{
			Label:       label,
			Kind:        string(g.Kind),
			Count:       g.Count,
			PricedCount: g.PricedCount,
			Total:       g.Total.String(),
		})
	}
	return out
}

func toStatisticsResponse(s collectible.CollectionStatistics) collectionStatisticsResponse {
	byStatus := make([]statusSummaryResponse, 0, len(s.ByStatus))
	for _, st := range s.ByStatus {
		byStatus = append(byStatus, statusSummaryResponse{
			Status:      string(st.Status),
			Count:       st.Count,
			PricedCount: st.PricedCount,
			Total:       st.Total.String(),
		})
	}
	return collectionStatisticsResponse{
		TotalCount:     s.TotalCount,
		PricedCount:    s.PricedCount,
		ByStatus:       byStatus,
		BySeries:       toGroupResponses(s.BySeries),
		ByManufacturer: toGroupResponses(s.ByManufacturer),
	}
}

// handleCollectionStatistics answers GET /api/collectibles/statistics.
//
// It reads nothing from the request but the session. There is no path value, no query parameter and
// no body — so there is no input by which another collector's figures could be asked for, which is a
// stronger guarantee than checking an identifier and refusing it (FR-003).
//
// An empty vault is a 200 carrying zeros. A collector with nothing in their vault has a vault, and
// it is empty; 404 would be a claim that they have none.
func (s *Server) handleCollectionStatistics(
	w http.ResponseWriter, r *http.Request, collector identity.CollectorID,
) {
	stats, err := s.service.Statistics(r.Context(), collector)
	if err != nil {
		WriteInternal(w, r, err)
		return
	}
	WriteJSON(w, http.StatusOK, toStatisticsResponse(stats))
}
