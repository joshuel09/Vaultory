//go:build integration

package contract

import (
	"net/http"
	"strconv"
	"strings"
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

// putJSON sends an edit.
func putJSON(t *testing.T, h *harness, id, body string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPut, h.server.URL+"/api/collectibles/"+id, strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return h.do(t, req, cookie)
}

// T032 — the editCollectible operation, against the contract.
func TestEditCollectibleResponseShapes(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	add := func(key, name string) (string, int) {
		t.Helper()
		body := decode(t, postJSON(t, h,
			`{"submissionKey":"`+key+`","name":"`+name+`","collectionStatus":"owned"}`, cookie))
		v, _ := body["version"].(float64)
		id, _ := body["id"].(string)
		return id, int(v)
	}

	t.Run("200 carries the collectible and its new version", func(t *testing.T) {
		id, version := add("ed-ok", "Kaiju Sentinel")
		resp := putJSON(t, h, id, `{"expectedVersion":`+itoa(version)+
			`,"name":"Kaiju Sentinel MkII","collectionStatus":"sold"}`, cookie)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d, want 200", resp.StatusCode)
		}
		body := decode(t, resp)
		if body["name"] != "Kaiju Sentinel MkII" || body["collectionStatus"] != "sold" {
			t.Errorf("the response does not reflect the edit: %+v", body)
		}
		if v, _ := body["version"].(float64); int(v) != version+1 {
			t.Errorf("version = %v, want %d", body["version"], version+1)
		}
	})

	t.Run("400 reports every violation together", func(t *testing.T) {
		id, version := add("ed-bad", "Kaiju Sentinel")
		resp := putJSON(t, h, id, `{"expectedVersion":`+itoa(version)+
			`,"name":"   ","collectionStatus":"borrowed","purchasePrice":"-5.00","purchaseDate":"2030-01-01"}`,
			cookie)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status %d, want 400", resp.StatusCode)
		}
		fields, _ := decode(t, resp)["error"].(map[string]any)["fields"].([]any)
		if len(fields) < 4 {
			t.Errorf("%d field errors, want at least 4 reported together (FR-014)", len(fields))
		}
	})

	t.Run("409 carries the collectible as it now stands", func(t *testing.T) {
		id, version := add("ed-stale", "Kaiju Sentinel")
		// Somebody edits first.
		if resp := putJSON(t, h, id, `{"expectedVersion":`+itoa(version)+
			`,"name":"Won the race","collectionStatus":"owned"}`, cookie); resp.StatusCode != http.StatusOK {
			t.Fatalf("the first edit failed with %d", resp.StatusCode)
		}
		// The stale save.
		resp := putJSON(t, h, id, `{"expectedVersion":`+itoa(version)+
			`,"name":"Lost the race","collectionStatus":"owned"}`, cookie)
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status %d, want 409 — accepting it would silently discard the first edit", resp.StatusCode)
		}
		body := decode(t, resp)
		if body["error"].(map[string]any)["code"] != "version_conflict" {
			t.Errorf("code = %v, want version_conflict", body["error"])
		}
		current, ok := body["current"].(map[string]any)
		if !ok {
			t.Fatalf("the 409 carried no current collectible: %+v", body)
		}
		if current["name"] != "Won the race" {
			t.Errorf("current.name = %v, want the winning edit's", current["name"])
		}
	})

	t.Run("404 for another collector's, identical to a fictional one", func(t *testing.T) {
		id, version := add("ed-theirs", "Private Statue")
		theirs := h.sessionFor(t, "second")
		body := `{"expectedVersion":` + itoa(version) + `,"name":"Taken","collectionStatus":"owned"}`

		mine := decode(t, putJSON(t, h, id, body, theirs))
		fiction := decode(t, putJSON(t, h, uuid.NewString(), body, theirs))

		// Both must be 404 with the same envelope. A distinct refusal would confirm the id is real.
		if mine["error"].(map[string]any)["code"] != "not_found" ||
			fiction["error"].(map[string]any)["code"] != "not_found" {
			t.Errorf("not both not_found: %+v / %+v", mine, fiction)
		}
		if mine["error"].(map[string]any)["message"] != fiction["error"].(map[string]any)["message"] {
			t.Error("another collector's collectible answered differently from a fictional one (FR-031)")
		}
	})

	t.Run("401 without a session, and an asserted identity is ignored", func(t *testing.T) {
		id, version := add("ed-auth", "Kaiju Sentinel")
		body := `{"expectedVersion":` + itoa(version) + `,"name":"Hijacked","collectionStatus":"owned"}`

		req, err := http.NewRequest(http.MethodPut,
			h.server.URL+"/api/collectibles/"+id+"?collectorId="+h.collectorA.String(),
			strings.NewReader(body))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		// A collector named in a header or a query parameter counts for nothing (FR-032).
		req.Header.Set("X-Collector-Id", h.collectorA.String())
		if resp := h.do(t, req, nil); resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status %d, want 401", resp.StatusCode)
		}

		// And the collectible is untouched.
		getReq, _ := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles/"+id, nil)
		after := decode(t, h.do(t, getReq, cookie))
		if after["name"] == "Hijacked" {
			t.Error("an unauthenticated request changed a collectible")
		}
	})
}

func itoa(n int) string { return strconv.Itoa(n) }
