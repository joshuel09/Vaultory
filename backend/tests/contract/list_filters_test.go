//go:build integration

package contract

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"
)

// listWith issues GET /api/collectibles with a raw query string and returns the decoded page.
//
// The query is passed raw rather than built with url.Values, because the thing under test is a
// parameter that is present and empty, and a helper that normalised the query away would hide it.
func listWith(t *testing.T, h *harness, query string, cookie *http.Cookie) map[string]any {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?"+query, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp := h.do(t, req, cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/collectibles?%s: status %d, want 200", query, resp.StatusCode)
	}
	return decode(t, resp)
}

func itemCount(t *testing.T, body map[string]any) int {
	t.Helper()
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("items is %T, want an array", body["items"])
	}
	return len(items)
}

// T052: the series and manufacturer filters, across the HTTP seam.
//
// The case that matters is `?series=` — present and empty. In Go, q.Get returns the empty string for
// an absent parameter and for a blank one alike, so the natural way to read it makes the unspecified
// group's link return the whole vault. That mistake cannot be caught below this level: the store and
// the service both receive a pointer, and only the handler decides whether to build one.
func TestAnEmptySeriesFilterSelectsTheUnspecifiedGroup(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	for i, body := range []string{
		`{"submissionKey":"f-1","name":"Saber","collectionStatus":"owned","series":"Fate"}`,
		`{"submissionKey":"f-2","name":"Archer","collectionStatus":"owned","series":"fate"}`,
		`{"submissionKey":"f-3","name":"Rem","collectionStatus":"owned","series":"Re:Zero"}`,
		`{"submissionKey":"f-4","name":"Unlabelled","collectionStatus":"owned"}`,
	} {
		if resp := postJSON(t, h, body, cookie); resp.StatusCode != http.StatusCreated {
			t.Fatalf("seed %d: status %d, want 201", i, resp.StatusCode)
		}
	}

	all := listWith(t, h, "", cookie)
	if got := itemCount(t, all); got != 4 {
		t.Fatalf("the unfiltered gallery returned %d collectibles, want 4", got)
	}

	// Present and empty: the one piece recording no series, not all four.
	blank := listWith(t, h, "series=", cookie)
	if got := itemCount(t, blank); got != 1 {
		t.Errorf("?series= returned %d collectibles, want 1 — an empty filter selects the pieces "+
			"recording no series, and must not be read as no filter at all", got)
	}

	// Case folds, matching the way the breakdown groups.
	for _, spelling := range []string{"Fate", "fate", "FATE", "  Fate  "} {
		got := itemCount(t, listWith(t, h, "series="+queryEscape(spelling), cookie))
		if got != 2 {
			t.Errorf("?series=%q returned %d collectibles, want 2 — the filter folds case and "+
				"whitespace exactly as the breakdown does", spelling, got)
		}
	}

	// And the manufacturer filter behaves the same way: nothing recorded one, so every piece is in
	// the unspecified manufacturer group.
	if got := itemCount(t, listWith(t, h, "manufacturer=", cookie)); got != 4 {
		t.Errorf("?manufacturer= returned %d collectibles, want 4", got)
	}
	if got := itemCount(t, listWith(t, h, "manufacturer=Alter", cookie)); got != 0 {
		t.Errorf("?manufacturer=Alter returned %d collectibles, want 0", got)
	}
}

// T052: the filters compose with the status filter rather than replacing it.
func TestGroupFiltersComposeWithTheStatusFilter(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	for i, body := range []string{
		`{"submissionKey":"c-1","name":"One","collectionStatus":"owned","series":"Fate"}`,
		`{"submissionKey":"c-2","name":"Two","collectionStatus":"sold","series":"Fate"}`,
		`{"submissionKey":"c-3","name":"Three","collectionStatus":"owned","series":"Re:Zero"}`,
	} {
		if resp := postJSON(t, h, body, cookie); resp.StatusCode != http.StatusCreated {
			t.Fatalf("seed %d: status %d, want 201", i, resp.StatusCode)
		}
	}

	if got := itemCount(t, listWith(t, h, "series=Fate&status=owned", cookie)); got != 1 {
		t.Errorf("series=Fate&status=owned returned %d collectibles, want 1", got)
	}
	if got := itemCount(t, listWith(t, h, "series=Fate", cookie)); got != 2 {
		t.Errorf("series=Fate returned %d collectibles, want 2", got)
	}
}

// T052: a filter longer than any value a collectible can hold is refused, not silently empty.
//
// An over-long filter can match nothing, and returning an empty gallery for it is indistinguishable
// from a vault with nothing in it — the exact confusion the totalUnfiltered field exists to prevent.
func TestAnOverlongGroupFilterIsRefused(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	// 200 characters of Japanese — 600 bytes — is a series name the product stores happily, so the
	// filter that reproduces its group must be accepted. Counting bytes here refuses it, and the
	// collector following a link the product offered them gets an error page (SC-010).
	wide := ""
	for utf8.RuneCountInString(wide) < 200 {
		wide += "\u3042"
	}
	okReq, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?series="+url.QueryEscape(wide), nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp := h.do(t, okReq, cookie); resp.StatusCode != http.StatusOK {
		t.Errorf("a 200-character, %d-byte series filter answered %d, want 200 — the bound counts "+
			"characters, as the contract and the domain both do", len(wide), resp.StatusCode)
	}

	long := ""
	for utf8.RuneCountInString(long) <= 200 {
		long += "a"
	}
	req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?series="+long, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	// 400, which is what every other refused query parameter on this operation answers — the
	// status filter and the limit both do. Consistency here matters more than the choice.
	resp := h.do(t, req, cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("a %d-character series filter answered %d, want 400", utf8.RuneCountInString(long), resp.StatusCode)
	}

	body := decode(t, resp)
	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("error is %T, want an object", body["error"])
	}
	fields, ok := errObj["fields"].([]any)
	if !ok || len(fields) == 0 {
		t.Fatalf("the refusal names no field: %+v", errObj)
	}
	first, _ := fields[0].(map[string]any)
	if first["field"] != "series" {
		t.Errorf("the refusal names %v, want series", first["field"])
	}
}

func queryEscape(s string) string {
	out := ""
	for _, r := range s {
		switch {
		case r == ' ':
			out += "%20"
		case r == ':':
			out += "%3A"
		default:
			out += fmt.Sprintf("%c", r)
		}
	}
	return out
}

// Feature 011: q and sort across the HTTP seam.
func TestSearchAndSortParameters(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")
	for _, body := range []string{
		`{"submissionKey":"k-s1","name":"Kaiju Sentinel","collectionStatus":"owned","purchasePrice":"900.00"}`,
		`{"submissionKey":"k-s2","name":"Mecha Titan","collectionStatus":"owned","purchasePrice":"300.00"}`,
		`{"submissionKey":"k-s3","name":"Kaiju Brute","collectionStatus":"owned"}`,
	} {
		_ = decode(t, postJSON(t, h, body, cookie))
	}

	// A search narrows; the CollectionPage shape and its whole-vault count are unchanged.
	found := listWith(t, h, "q=kaiju", cookie)
	if n := itemCount(t, found); n != 2 {
		t.Errorf("q=kaiju returned %d, want 2", n)
	}
	if total, _ := found["totalUnfiltered"].(float64); total != 3 {
		t.Errorf("totalUnfiltered = %v, want the whole vault, 3", found["totalUnfiltered"])
	}

	// A sort orders, with missing prices last.
	sorted := listWith(t, h, "sort=price_asc", cookie)
	items, _ := sorted["items"].([]any)
	var names []string
	for _, it := range items {
		m, _ := it.(map[string]any)
		names = append(names, fmt.Sprint(m["name"]))
	}
	if strings.Join(names, ",") != "Mecha Titan,Kaiju Sentinel,Kaiju Brute" {
		t.Errorf("sort=price_asc = %v", names)
	}

	// Refusals: 400 validation_failed, naming the field.
	for query, field := range map[string]string{
		"q=" + strings.Repeat("a", 201): "q",
		"sort=price":                    "sort",
	} {
		req, _ := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?"+query, nil)
		resp := h.do(t, req, cookie)
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%s: status %d, want 400", query[:10], resp.StatusCode)
			continue
		}
		body := decode(t, resp)
		errBody, _ := body["error"].(map[string]any)
		fields, _ := errBody["fields"].([]any)
		if len(fields) != 1 || fields[0].(map[string]any)["field"] != field {
			t.Errorf("%s: fields = %v, want one on %q", query[:10], fields, field)
		}
	}

	// A cursor issued for one order is refused under another.
	first := listWith(t, h, "sort=price_desc&limit=1", cookie)
	cursor, _ := first["nextCursor"].(string)
	if cursor == "" {
		t.Fatal("no nextCursor with limit=1")
	}
	req, _ := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?sort=name_asc&cursor="+cursor, nil)
	if resp := h.do(t, req, cookie); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("cross-order cursor: status %d, want 400", resp.StatusCode)
	}
}
