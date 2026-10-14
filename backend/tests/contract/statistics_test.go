//go:build integration

package contract

import (
	"net/http"
	"regexp"
	"testing"
)

// exactDecimal is the MonetaryTotal pattern from the contract: up to sixteen integer digits, with at
// most two fractional ones. Sixteen rather than MonetaryAmount's ten, because a total is derived and
// is not bound by what the storage column holds.
var exactDecimal = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})(\.[0-9]{1,2})?$`)

func getStatistics(t *testing.T, h *harness, cookie *http.Cookie) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles/statistics", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp := h.do(t, req, cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET statistics: status %d, want 200", resp.StatusCode)
	}
	return decode(t, resp)
}

// T029: the response carries every field the contract requires, in the shapes it specifies.
func TestStatisticsResponseShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	decode(t, postJSON(t, h,
		`{"submissionKey":"st-1","name":"Saber","collectionStatus":"owned",`+
			`"series":"Fate","manufacturer":"Good Smile Company","purchasePrice":"1250.00"}`, cookie))
	decode(t, postJSON(t, h,
		`{"submissionKey":"st-2","name":"Archer","collectionStatus":"sold",`+
			`"series":"Fate","manufacturer":"Alter","purchasePrice":"300.00"}`, cookie))

	body := getStatistics(t, h, cookie)

	for _, field := range []string{"totalCount", "pricedCount", "byStatus", "bySeries", "byManufacturer"} {
		if _, ok := body[field]; !ok {
			t.Errorf("response is missing required field %q", field)
		}
	}

	byStatus, ok := body["byStatus"].([]any)
	if !ok {
		t.Fatalf("byStatus is %T, want an array", body["byStatus"])
	}
	// minItems and maxItems are both 4 in the contract: a client never has to decide what an absent
	// status means.
	if len(byStatus) != 4 {
		t.Fatalf("byStatus has %d entries, want exactly 4", len(byStatus))
	}

	seen := map[string]bool{}
	for _, raw := range byStatus {
		row, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("byStatus entry is %T, want an object", raw)
		}
		for _, field := range []string{"status", "count", "pricedCount", "total"} {
			if _, ok := row[field]; !ok {
				t.Errorf("status entry is missing required field %q: %+v", field, row)
			}
		}
		status, _ := row["status"].(string)
		seen[status] = true

		total, ok := row["total"].(string)
		if !ok {
			t.Errorf("total for %q is %T, want a string — a JSON number invites a float on the "+
				"other side, which is precisely what Principle IV forbids", status, row["total"])
			continue
		}
		if !exactDecimal.MatchString(total) {
			t.Errorf("total for %q is %q, which does not match the MonetaryTotal pattern", status, total)
		}
	}
	for _, want := range []string{"owned", "preordered", "wishlist", "sold"} {
		if !seen[want] {
			t.Errorf("byStatus does not include %q; all four are always present", want)
		}
	}
}

// T029: breakdown entries carry a kind, so a client never infers one from a label.
func TestStatisticsBreakdownShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	decode(t, postJSON(t, h,
		`{"submissionKey":"bd-1","name":"With series","collectionStatus":"owned",`+
			`"series":"Monogatari","purchasePrice":"10.00"}`, cookie))
	decode(t, postJSON(t, h,
		`{"submissionKey":"bd-2","name":"Without series","collectionStatus":"owned",`+
			`"purchasePrice":"20.00"}`, cookie))

	body := getStatistics(t, h, cookie)

	bySeries, ok := body["bySeries"].([]any)
	if !ok {
		t.Fatalf("bySeries is %T, want an array", body["bySeries"])
	}
	if len(bySeries) == 0 {
		t.Fatal("bySeries is empty for a vault holding two collectibles")
	}

	kinds := map[string]bool{}
	for _, raw := range bySeries {
		row, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("bySeries entry is %T, want an object", raw)
		}
		for _, field := range []string{"label", "kind", "count", "pricedCount", "total"} {
			if _, ok := row[field]; !ok {
				t.Errorf("group entry is missing required field %q: %+v", field, row)
			}
		}
		kind, _ := row["kind"].(string)
		switch kind {
		case "named", "unspecified", "remainder":
			kinds[kind] = true
		default:
			t.Errorf("group kind %q is not one of the three the contract allows", kind)
		}
		if total, ok := row["total"].(string); !ok || !exactDecimal.MatchString(total) {
			t.Errorf("group total is %v (%T), want an exact decimal string", row["total"], row["total"])
		}
	}

	if !kinds["named"] {
		t.Error("no named group, though one collectible records a series")
	}
	if !kinds["unspecified"] {
		t.Error("no unspecified group, though one collectible records no series — those collectibles " +
			"must be accounted for, not dropped from a breakdown that claims to cover the collection")
	}
}

// T029: the operation requires a session, and offers no way to name a collector.
//
// There is no parameter to probe with, which is the point: FR-003 is satisfied by the shape of the
// operation rather than by a check inside it. The most this can assert is that the session is
// required and that another collector's data never appears.
func TestStatisticsRefusesWithoutASession(t *testing.T) {
	h := newHarness(t)

	req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles/statistics", nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp := h.do(t, req, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("statistics without a session: status %d, want 401", resp.StatusCode)
	}
}
