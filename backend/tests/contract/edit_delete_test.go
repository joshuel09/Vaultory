//go:build integration

package contract

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// getCollectible is what the edit screen is filled from. The representation is the one the gallery
// returns — the same projection, so a detail shape cannot drift from the list shape.
func TestGetCollectibleResponseShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	created := decode(t, postJSON(t, h,
		`{"submissionKey":"k-get","name":"Kaiju Sentinel","collectionStatus":"owned",`+
			`"series":"Kaiju Wars","purchasePrice":"1250.00","purchaseDate":"2026-08-14"}`, cookie))
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("no id in the add response: %+v", created)
	}

	req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles/"+id, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp := h.do(t, req, cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET one collectible: status %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)

	// Required by the Collectible schema.
	for _, field := range []string{"id", "version", "name", "collectionStatus", "createdAt"} {
		if _, ok := body[field]; !ok {
			t.Errorf("response is missing required field %q", field)
		}
	}

	// Every stored value comes back, because the edit screen is filled from this (FR-002).
	if body["series"] != "Kaiju Wars" {
		t.Errorf("series = %v, want Kaiju Wars", body["series"])
	}
	// An exact decimal as a string, never a JSON number (Constitution IV).
	if price, ok := body["purchasePrice"].(string); !ok || price != "1250.00" {
		t.Errorf("purchasePrice = %#v, want the string \"1250.00\"", body["purchasePrice"])
	}
	// Attributes the collector never supplied are null, not invented defaults (FR-003).
	for _, field := range []string{"character", "manufacturer", "scale", "notes", "image"} {
		if v, ok := body[field]; !ok || v != nil {
			t.Errorf("%s = %#v, want null — an unsupplied attribute must not acquire a default", field, v)
		}
	}
	// A freshly added collectible starts at version 1 and an edit will raise it (FR-027a).
	if v, ok := body["version"].(float64); !ok || v != 1 {
		t.Errorf("version = %#v, want 1", body["version"])
	}
}

// Another collector's collectible, a collectible that never existed, and a malformed identifier
// must be indistinguishable (FR-031, SC-002).
func TestGetCollectibleRevealsNothing(t *testing.T) {
	h := newHarness(t)
	mine := h.sessionFor(t, "")
	theirs := h.sessionFor(t, "second")

	created := decode(t, postJSON(t, h,
		`{"submissionKey":"k-hidden","name":"Private Statue","collectionStatus":"owned"}`, mine))
	id, _ := created["id"].(string)

	cases := []struct {
		name string
		path string
	}{
		{"another collector's collectible", "/api/collectibles/" + id},
		{"an identifier that never existed", "/api/collectibles/" + uuid.NewString()},
		{"a malformed identifier", "/api/collectibles/not-a-uuid"},
	}

	var bodies []map[string]any
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, h.server.URL+c.path, nil)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			resp := h.do(t, req, theirs)
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("status %d, want 404 — never 403, which would confirm it exists", resp.StatusCode)
			}
			bodies = append(bodies, decode(t, resp))
		})
	}

	// Not merely "all 404", but all the same 404. A difference in the body would disclose which
	// identifier is real just as surely as a different status would.
	for i := 1; i < len(bodies); i++ {
		if bodies[i]["error"].(map[string]any)["code"] != bodies[0]["error"].(map[string]any)["code"] ||
			bodies[i]["error"].(map[string]any)["message"] != bodies[0]["error"].(map[string]any)["message"] {
			t.Errorf("%s answered differently from %s: %+v vs %+v",
				cases[i].name, cases[0].name, bodies[i], bodies[0])
		}
	}
}

// FR-033: no session, no collection content. FR-032: an identity the caller asserts is ignored.
func TestGetCollectibleRefusesWithoutASession(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	created := decode(t, postJSON(t, h,
		`{"submissionKey":"k-auth","name":"Kaiju Sentinel","collectionStatus":"owned"}`, cookie))
	id, _ := created["id"].(string)

	t.Run("no session at all", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles/"+id, nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		resp := h.do(t, req, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", resp.StatusCode)
		}
		body := decode(t, resp)
		if _, leaked := body["name"]; leaked {
			t.Error("the unauthenticated response carried collection content (FR-033)")
		}
	})

	t.Run("an asserted collector id is ignored", func(t *testing.T) {
		// The identity comes from the session and nowhere else. A caller naming a collector in a
		// header or a query parameter must get no benefit from it (FR-032).
		req, err := http.NewRequest(http.MethodGet,
			h.server.URL+"/api/collectibles/"+id+"?collectorId="+h.collectorA.String(), nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("X-Collector-Id", h.collectorA.String())
		resp := h.do(t, req, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401 — an asserted identity must count for nothing", resp.StatusCode)
		}
	})

	t.Run("another collector's session does not help", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles/"+id, nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("X-Collector-Id", h.collectorA.String())
		resp := h.do(t, req, h.sessionFor(t, "second"))
		if resp.StatusCode != http.StatusNotFound {
			t.Fatalf("status %d, want 404", resp.StatusCode)
		}
	})
}
