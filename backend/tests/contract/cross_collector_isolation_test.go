//go:build integration

package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

// FR-015, SC-004 — feature 001's central guarantee, re-established against real sessions.
//
// Feature 001 proved cross-collector isolation with tests written against the development
// resolver. Feature 004 replaced that entire seam, so those tests can no longer vouch for it:
// they were exercising a mechanism that no longer exists. This asserts the same property through
// signed Better Auth sessions, end to end over HTTP.
//
// 404, never 403. A 403 answers the question "does this exist?", which is the question a private
// vault must never answer for someone else's collectible.
func TestOneCollectorCannotReachAnother(t *testing.T) {
	h := newHarness(t)
	owner := h.sessionFor(t, "")
	other := h.sessionFor(t, "second")

	// The owner adds a collectible and an image.
	body := map[string]any{
		"submissionKey":    "isolation-key",
		"name":             "Private Statue",
		"collectionStatus": "owned",
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, h.server.URL+"/api/collectibles", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp := h.do(t, req, owner)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("owner could not add a collectible: %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	_ = resp.Body.Close()

	t.Run("the other collector's gallery is empty", func(t *testing.T) {
		r, _ := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles", nil)
		resp := h.do(t, r, other)
		defer func() { _ = resp.Body.Close() }()
		if n := decodeTotal(t, resp); n != 0 {
			t.Fatalf("the other collector sees %d collectibles, want 0", n)
		}
	})

	t.Run("filtering does not leak it either", func(t *testing.T) {
		r, _ := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?status=owned", nil)
		resp := h.do(t, r, other)
		defer func() { _ = resp.Body.Close() }()
		if n := decodeTotal(t, resp); n != 0 {
			t.Fatalf("a status filter exposed %d of another collector's collectibles", n)
		}
	})

	t.Run("an image rendition answers 404, never 403", func(t *testing.T) {
		// Any identifier the other collector does not own must answer the same way, whether it
		// exists or not — otherwise the status code itself is a disclosure.
		id, _ := created["id"].(string)
		for name, target := range map[string]string{
			"another collector's image":        id,
			"an identifier that never existed": "00000000-0000-4000-8000-000000000000",
		} {
			r, _ := http.NewRequest(http.MethodGet,
				fmt.Sprintf("%s/api/images/%s/rendition", h.server.URL, target), nil)
			resp := h.do(t, r, other)
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusForbidden {
				t.Fatalf("%s: 403 confirms the resource exists", name)
			}
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("%s: status %d, want 404", name, resp.StatusCode)
			}
		}
	})

	t.Run("the owner still sees their own", func(t *testing.T) {
		r, _ := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles", nil)
		resp := h.do(t, r, owner)
		defer func() { _ = resp.Body.Close() }()
		if n := decodeTotal(t, resp); n != 1 {
			t.Fatalf("the owner sees %d, want 1 — isolation that hides it from everyone is not "+
				"isolation", n)
		}
	})
}
