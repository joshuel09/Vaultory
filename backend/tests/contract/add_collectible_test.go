//go:build integration

package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return out
}

func postJSON(t *testing.T, h *harness, body string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, h.server.URL+"/api/collectibles", strings.NewReader(body))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	return h.do(t, req, cookie)
}

// The 201 body must match the Collectible schema.
func TestAddCollectibleResponseShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	resp := postJSON(t, h,
		`{"submissionKey":"k1","name":"Kaiju Sentinel","collectionStatus":"owned"}`, cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	body := decode(t, resp)

	// Required properties.
	for _, key := range []string{"id", "name", "collectionStatus", "createdAt"} {
		if body[key] == nil {
			t.Errorf("response is missing required property %q", key)
		}
	}
	// Nullable properties must be present and null, not absent — the generated TS type declares
	// them.
	for _, key := range []string{
		"character", "series", "manufacturer", "category", "scale", "edition",
		"purchasePrice", "purchaseDate", "releaseDate", "notes", "image",
	} {
		v, present := body[key]
		if !present {
			t.Errorf("response omits nullable property %q", key)
		}
		if v != nil {
			t.Errorf("property %q = %v, want null when unrecorded", key, v)
		}
	}
	if body["collectionStatus"] != "owned" {
		t.Errorf("collectionStatus = %v", body["collectionStatus"])
	}
}

// Money crosses as a string, never a JSON number (Constitution IV).
func TestPurchasePriceIsAString(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	resp := postJSON(t, h,
		`{"submissionKey":"k2","name":"Priced","collectionStatus":"owned","purchasePrice":"1250.00"}`, cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	body := decode(t, resp)
	price, ok := body["purchasePrice"].(string)
	if !ok {
		t.Fatalf("purchasePrice is %T, want string — a JSON number invites a float", body["purchasePrice"])
	}
	if price != "1250.00" {
		t.Errorf("purchasePrice = %q, want \"1250.00\" exactly", price)
	}

	// And it survives the round trip through the raw bytes without becoming a number.
	raw := postJSON(t, h,
		`{"submissionKey":"k3","name":"Priced2","collectionStatus":"owned","purchasePrice":"0.00"}`, cookie)
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(raw.Body)
	_ = raw.Body.Close()
	if !strings.Contains(buf.String(), `"purchasePrice":"0.00"`) {
		t.Errorf("zero should serialise as the string \"0.00\": %s", buf.String())
	}
}

// FR-020: every problem in one response, matching the ValidationFailed schema.
func TestValidationFailureShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	resp := postJSON(t, h,
		`{"submissionKey":"k4","name":"   ","collectionStatus":"borrowed","purchasePrice":"-5.00","purchaseDate":"2030-01-01"}`,
		cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	body := decode(t, resp)

	errObj, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("response has no error object: %v", body)
	}
	if errObj["code"] != "validation_failed" {
		t.Errorf("code = %v, want validation_failed", errObj["code"])
	}
	if errObj["message"] == nil {
		t.Error("error has no message")
	}
	fields, ok := errObj["fields"].([]any)
	if !ok {
		t.Fatalf("error has no fields array: %v", errObj)
	}
	if len(fields) < 4 {
		t.Errorf("%d field errors, want at least 4 reported together (FR-020)", len(fields))
	}
	seen := map[string]string{}
	for _, f := range fields {
		fe, ok := f.(map[string]any)
		if !ok {
			t.Fatalf("field entry is not an object: %v", f)
		}
		for _, key := range []string{"field", "code", "message"} {
			if fe[key] == nil {
				t.Errorf("field entry missing %q: %v", key, fe)
			}
		}
		seen[fe["field"].(string)] = fe["code"].(string)
	}
	for field, code := range map[string]string{
		"name":             "required",
		"collectionStatus": "invalid_value",
		"purchasePrice":    "negative_amount",
		"purchaseDate":     "date_in_future",
	} {
		if seen[field] != code {
			t.Errorf("field %q has code %q, want %q", field, seen[field], code)
		}
	}
}

// FR-047: the contract declares submissionKey required.
func TestSubmissionKeyIsRequired(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	resp := postJSON(t, h, `{"name":"No Key","collectionStatus":"owned"}`, cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 when submissionKey is absent", resp.StatusCode)
	}
	body := decode(t, resp)
	errObj := body["error"].(map[string]any)
	fields, _ := errObj["fields"].([]any)
	found := false
	for _, f := range fields {
		if fe, ok := f.(map[string]any); ok && fe["field"] == "submissionKey" {
			found = true
		}
	}
	if !found {
		t.Errorf("no field error for submissionKey: %v", errObj)
	}
}

// additionalProperties: false in the contract must be real, not decorative.
func TestUnknownFieldsAreRefused(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")
	resp := postJSON(t, h,
		`{"submissionKey":"k5","name":"X","collectionStatus":"owned","collectorId":"22222222-2222-4222-8222-222222222222"}`,
		cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 for an unknown field", resp.StatusCode)
	}
	// Note what was attempted above: supplying a collector id in the body. FR-028 forbids honouring
	// it, and refusing unknown fields means it cannot even be misread.
}

// FR-029: no session, no content.
func TestUnauthenticatedShape(t *testing.T) {
	h := newHarness(t)
	resp := postJSON(t, h, `{"submissionKey":"k6","name":"X","collectionStatus":"owned"}`, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
	body := decode(t, resp)
	errObj := body["error"].(map[string]any)
	if errObj["code"] != "unauthenticated" {
		t.Errorf("code = %v, want unauthenticated", errObj["code"])
	}
	if body["items"] != nil {
		t.Error("an unauthenticated response must carry no collection content")
	}
}

// FR-047 over HTTP: a replayed key returns the same collectible, with the same 201.
func TestReplayedKeyReturnsSameCollectible(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")
	body := `{"submissionKey":"replay-1","name":"Retry","collectionStatus":"owned"}`

	first := decode(t, postJSON(t, h, body, cookie))
	second := decode(t, postJSON(t, h, body, cookie))

	if first["id"] != second["id"] {
		t.Errorf("replay produced a different collectible: %v vs %v", first["id"], second["id"])
	}
}
