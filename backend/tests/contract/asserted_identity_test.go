//go:build integration

package contract

import (
	"encoding/json"
	"net/http"
	"testing"
)

// FR-013, SC-005 — the test that would catch the mistake this whole architecture exists to prevent.
//
// Better Auth issues sessions in Next.js. If Go ever accepted an identity that Next.js merely
// asserted — a header, a query parameter, a body field — then a bug anywhere in the frontend could
// hand one collector another's vault, and every careful thing in the backend would be decoration.
// The constitution states it directly: client-provided user IDs must not be accepted without
// verification.
func TestAnAssertedIdentityIsNeverAccepted(t *testing.T) {
	h := newHarness(t)
	victim := h.collectorA.String()

	assertions := map[string]func(*http.Request){
		"X-Collector-Id header":   func(r *http.Request) { r.Header.Set("X-Collector-Id", victim) },
		"X-User-Id header":        func(r *http.Request) { r.Header.Set("X-User-Id", h.userA) },
		"Authorization bearer":    func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+victim) },
		"collector_id query":      func(r *http.Request) { q := r.URL.Query(); q.Set("collector_id", victim); r.URL.RawQuery = q.Encode() },
		"collectorId query":       func(r *http.Request) { q := r.URL.Query(); q.Set("collectorId", victim); r.URL.RawQuery = q.Encode() },
		"X-Forwarded-User header": func(r *http.Request) { r.Header.Set("X-Forwarded-User", h.userA) },
	}

	for name, assert := range assertions {
		t.Run(name+" with no session", func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles", nil)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			assert(req)
			resp := h.do(t, req, nil)
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("status %d — an asserted identity was honoured without a session",
					resp.StatusCode)
			}
		})
	}

	// The sharper case: a genuine session for one collector, plus an assertion naming another.
	// The assertion must be ignored entirely rather than treated as an override.
	t.Run("a valid session for B cannot be redirected at A", func(t *testing.T) {
		cookie := h.sessionFor(t, "second")
		req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles", nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("X-Collector-Id", victim)
		req.Header.Set("X-User-Id", h.userA)
		resp := h.do(t, req, cookie)
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d — B's own session should still work", resp.StatusCode)
		}
		// B's collection is empty; A's is not. Seeing anything here means the header won.
		if n := decodeTotal(t, resp); n != 0 {
			t.Fatalf("B saw %d collectibles — the asserted header overrode the verified session", n)
		}
	})
}

// decodeTotal reads totalUnfiltered from a collection page.
func decodeTotal(t *testing.T, resp *http.Response) int {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	total, _ := body["totalUnfiltered"].(float64)
	return int(total)
}
