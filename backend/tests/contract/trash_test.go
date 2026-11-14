//go:build integration

package contract

import (
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
)

// Feature 010: the Trash endpoints answer as backend/api/openapi.yaml says.

func call(t *testing.T, h *harness, method, path string, cookie *http.Cookie) *http.Response {
	t.Helper()
	req, err := http.NewRequest(method, h.server.URL+path, nil)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	return h.do(t, req, cookie)
}

func addOne(t *testing.T, h *harness, key string, cookie *http.Cookie) string {
	t.Helper()
	body := decode(t, postJSON(t, h,
		`{"submissionKey":"`+key+`","name":"Kaiju Sentinel","collectionStatus":"owned"}`, cookie))
	id, _ := body["id"].(string)
	if id == "" {
		t.Fatalf("no id in the add response: %+v", body)
	}
	return id
}

func status(t *testing.T, resp *http.Response) int {
	t.Helper()
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return resp.StatusCode
}

// TrashPage and TrashedCollectible: required fields, the nested Collectible, and the vault no
// longer serving the deleted collectible.
func TestTrashListingShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")
	id := addOne(t, h, "k-trash-shape", cookie)

	if got := status(t, call(t, h, http.MethodDelete, "/api/collectibles/"+id, cookie)); got != http.StatusNoContent {
		t.Fatalf("DELETE collectible: %d, want 204", got)
	}
	if got := status(t, call(t, h, http.MethodGet, "/api/collectibles/"+id, cookie)); got != http.StatusNotFound {
		t.Errorf("GET a trashed collectible: %d, want 404", got)
	}

	resp := call(t, h, http.MethodGet, "/api/trash", cookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/trash: %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)
	items, ok := body["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %#v, want one entry", body["items"])
	}
	item, _ := items[0].(map[string]any)
	for _, field := range []string{"collectible", "deletedAt", "purgeAt"} {
		if _, ok := item[field]; !ok {
			t.Errorf("TrashedCollectible is missing required %q", field)
		}
	}
	if len(item) != 3 {
		t.Errorf("TrashedCollectible has %d fields, want exactly 3 (additionalProperties: false): %v", len(item), item)
	}
	c, _ := item["collectible"].(map[string]any)
	for _, field := range []string{"id", "version", "name", "collectionStatus", "createdAt"} {
		if _, ok := c[field]; !ok {
			t.Errorf("nested Collectible is missing required %q", field)
		}
	}
	if c["id"] != id {
		t.Errorf("collectible.id = %v, want %s", c["id"], id)
	}

	if got := status(t, call(t, h, http.MethodGet, "/api/trash?limit=49", cookie)); got != http.StatusBadRequest {
		t.Errorf("limit 49: %d, want 400", got)
	}
}

// restoreFromTrash: 200 with a Collectible; 404 for everything else, all alike.
func TestRestoreStatusCodes(t *testing.T) {
	h := newHarness(t)
	a := h.sessionFor(t, "")
	b := h.sessionFor(t, "second")
	id := addOne(t, h, "k-restore", a)

	if got := status(t, call(t, h, http.MethodPost, "/api/trash/"+id+"/restore", a)); got != http.StatusNotFound {
		t.Errorf("restore of a collectible in the vault: %d, want 404", got)
	}
	_ = status(t, call(t, h, http.MethodDelete, "/api/collectibles/"+id, a))

	// Another collector, an absent id, and a malformed id: indistinguishable (FR-013, SC-006).
	for name, path := range map[string]string{
		"another collector's": "/api/trash/" + id + "/restore",
		"absent":              "/api/trash/" + uuid.NewString() + "/restore",
		"malformed":           "/api/trash/not-a-uuid/restore",
	} {
		if got := status(t, call(t, h, http.MethodPost, path, b)); got != http.StatusNotFound {
			t.Errorf("restore of %s: %d, want 404", name, got)
		}
	}

	resp := call(t, h, http.MethodPost, "/api/trash/"+id+"/restore", a)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore: %d, want 200", resp.StatusCode)
	}
	body := decode(t, resp)
	if body["id"] != id {
		t.Errorf("restored id = %v, want %s", body["id"], id)
	}
	if _, ok := body["deletedAt"]; ok {
		t.Error("the restored Collectible carries deletedAt, which the Collectible schema does not allow")
	}
	if got := status(t, call(t, h, http.MethodGet, "/api/collectibles/"+id, a)); got != http.StatusOK {
		t.Errorf("GET after restore: %d, want 200", got)
	}
	if got := status(t, call(t, h, http.MethodPost, "/api/trash/"+id+"/restore", a)); got != http.StatusNotFound {
		t.Errorf("second restore: %d, want 404", got)
	}
}

// deleteFromTrashPermanently: 204 always; only ever destroys the acting collector's Trash.
func TestDeletePermanentlyStatusCodes(t *testing.T) {
	h := newHarness(t)
	a := h.sessionFor(t, "")
	b := h.sessionFor(t, "second")
	id := addOne(t, h, "k-perm", a)
	_ = status(t, call(t, h, http.MethodDelete, "/api/collectibles/"+id, a))

	for name, path := range map[string]string{
		"another collector's": "/api/trash/" + id,
		"absent":              "/api/trash/" + uuid.NewString(),
		"malformed":           "/api/trash/not-a-uuid",
	} {
		if got := status(t, call(t, h, http.MethodDelete, path, b)); got != http.StatusNoContent {
			t.Errorf("permanent delete of %s: %d, want 204", name, got)
		}
	}
	if got := status(t, call(t, h, http.MethodPost, "/api/trash/"+id+"/restore", a)); got != http.StatusOK {
		t.Fatalf("another collector's permanent delete reached A's Trash: restore answered %d", got)
	}
	_ = status(t, call(t, h, http.MethodDelete, "/api/collectibles/"+id, a))

	for i := 0; i < 2; i++ {
		if got := status(t, call(t, h, http.MethodDelete, "/api/trash/"+id, a)); got != http.StatusNoContent {
			t.Errorf("permanent delete %d: %d, want 204", i+1, got)
		}
	}
	if got := status(t, call(t, h, http.MethodPost, "/api/trash/"+id+"/restore", a)); got != http.StatusNotFound {
		t.Errorf("restore after permanent deletion: %d, want 404", got)
	}
}

// The Trash requires a session, like every collector endpoint.
func TestTrashRequiresASession(t *testing.T) {
	h := newHarness(t)
	for _, c := range []struct{ method, path string }{
		{http.MethodGet, "/api/trash"},
		{http.MethodPost, "/api/trash/" + uuid.NewString() + "/restore"},
		{http.MethodDelete, "/api/trash/" + uuid.NewString()},
	} {
		if got := status(t, call(t, h, c.method, c.path, nil)); got != http.StatusUnauthorized {
			t.Errorf("%s %s without a session: %d, want 401", c.method, c.path, got)
		}
	}
}
