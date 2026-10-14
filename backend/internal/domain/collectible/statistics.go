package collectible

import "sort"

// GroupKind says what a breakdown entry represents.
//
// It exists so that nothing has to infer a group's nature from its label. An unspecified group whose
// label happened to read "Unspecified" would otherwise be indistinguishable from a collector who
// typed that word as a manufacturer.
type GroupKind string

const (
	GroupNamed       GroupKind = "named"
	GroupUnspecified GroupKind = "unspecified"
	GroupRemainder   GroupKind = "remainder"
)

// MaxBreakdownGroups bounds how many named groups a breakdown returns before the rest are folded
// into a single remainder entry (FR-025).
const MaxBreakdownGroups = 8

// StatusSummary is the collectibles holding one collection status, and what was paid for them.
type StatusSummary struct {
	Status      CollectionStatus
	Count       int
	PricedCount int
	Total       Money
}

// GroupSummary is one entry in a breakdown — a series, a manufacturer, the unspecified group, or
// the remainder.
type GroupSummary struct {
	Label       string
	Kind        GroupKind
	Count       int
	PricedCount int
	Total       Money
}

// CollectionStatistics is the whole-vault summary.
//
// There is deliberately no headline field. The four status summaries are peers; which one a
// collector sees largest is a presentation decision, and encoding it here would make a change of
// emphasis a change to the domain.
type CollectionStatistics struct {
	TotalCount     int
	PricedCount    int
	ByStatus       []StatusSummary
	BySeries       []GroupSummary
	ByManufacturer []GroupSummary
}

// FillMissingStatuses returns exactly one summary per status, in the order AllStatuses gives.
//
// It reuses the closed set declared in status.go rather than restating it. A second list of the four
// statuses is a second place to forget when a fifth is added — exactly the drift that makes a view
// silently omit something.
//
// A status holding nothing produces no row in the database, and FR-015 requires it to appear anyway:
// a client should never have to decide what an absent status means, and "it was omitted" and "it
// holds nothing" are not distinguishable once one of them is missing.
func FillMissingStatuses(found []StatusSummary) []StatusSummary {
	byStatus := make(map[CollectionStatus]StatusSummary, len(found))
	for _, s := range found {
		byStatus[s.Status] = s
	}

	statuses := AllStatuses()
	out := make([]StatusSummary, 0, len(statuses))
	for _, status := range statuses {
		if s, ok := byStatus[status]; ok {
			out = append(out, s)
			continue
		}
		out = append(out, StatusSummary{Status: status})
	}
	return out
}

// SortGroups orders a breakdown: named groups first by size, then the unspecified group, then the
// remainder (FR-022).
//
// The two special groups sort below every named group *whatever their own size*. Ordering by count
// alone puts "everything else" above a real group as soon as the tail is larger than the smallest
// named one, which reads as though the collection is mostly nothing in particular. That was found by
// running the query against real data during planning, not by reasoning about it.
//
// Ties between named groups resolve on the label, so repeated calls return the same order (FR-023).
func SortGroups(groups []GroupSummary) {
	rank := func(k GroupKind) int {
		switch k {
		case GroupUnspecified:
			return 1
		case GroupRemainder:
			return 2
		default:
			return 0
		}
	}

	sort.SliceStable(groups, func(i, j int) bool {
		a, b := groups[i], groups[j]
		if ra, rb := rank(a.Kind), rank(b.Kind); ra != rb {
			return ra < rb
		}
		if a.Count != b.Count {
			return a.Count > b.Count
		}
		return a.Label < b.Label
	})
}

// Totals derives the whole-vault figures from the per-status summaries.
//
// Deriving rather than querying separately is what lets User Story 1 and User Story 2 share one
// statement: the four rows already contain everything the headline needs. The arithmetic lives here,
// in the domain, which is where the constitution puts business logic — and specifically not in the
// browser, which FR-001 forbids.
func Totals(byStatus []StatusSummary) (count int, priced int, total Money, err error) {
	for _, s := range byStatus {
		count += s.Count
		priced += s.PricedCount
		total, err = total.AddTotal(s.Total)
		if err != nil {
			return 0, 0, Money{}, err
		}
	}
	return count, priced, total, nil
}
