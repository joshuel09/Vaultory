//go:build integration

package contract

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/imaging"
)

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 60, G: 80, B: 110, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// upload posts a multipart body, optionally lying about the content type — which the server must
// ignore in favour of decoding the bytes (FR-009).
func upload(t *testing.T, h *harness, filename, declaredType string, data []byte, cookie *http.Cookie) *http.Response {
	t.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	part, err := mw.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := mw.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, h.server.URL+"/api/images", &body)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if declaredType != "" {
		req.Header.Set("X-Declared-Type", declaredType)
	}
	return h.do(t, req, cookie)
}

func TestUploadImageResponseShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	resp := upload(t, h, "sentinel.jpg", "", jpegBytes(t, 600, 800), cookie)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	body := decode(t, resp)

	for _, key := range []string{"id", "renditionUrl", "contentType", "byteSize", "width", "height"} {
		if body[key] == nil {
			t.Errorf("response is missing required property %q", key)
		}
	}
	if body["contentType"] != "image/jpeg" {
		t.Errorf("contentType = %v, want image/jpeg", body["contentType"])
	}
	if url, _ := body["renditionUrl"].(string); url != "/api/images/"+body["id"].(string)+"/rendition" {
		t.Errorf("renditionUrl = %v, does not match the documented shape", body["renditionUrl"])
	}
}

// FR-010: the 413 names the limit.
func TestOversizedUploadShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	oversized := bytes.Repeat([]byte{0x5A}, int(imaging.MaxUploadBytes)+1024)
	resp := upload(t, h, "huge.jpg", "", oversized, cookie)
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413", resp.StatusCode)
	}
	errObj := decode(t, resp)["error"].(map[string]any)
	if errObj["code"] != "image_too_large" {
		t.Errorf("code = %v, want image_too_large", errObj["code"])
	}
	if msg, _ := errObj["message"].(string); msg == "" {
		t.Error("the refusal must state the limit")
	}
}

// FR-009: a non-image is refused, whatever its name or declared type claims.
func TestUnsupportedFormatShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	pdf := []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n")
	resp := upload(t, h, "document.jpg", "image/jpeg", pdf, cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 — the filename and declared type must not be trusted", resp.StatusCode)
	}
	errObj := decode(t, resp)["error"].(map[string]any)
	if errObj["code"] != "unsupported_image_format" {
		t.Errorf("code = %v, want unsupported_image_format", errObj["code"])
	}
	fields, _ := errObj["fields"].([]any)
	if len(fields) == 0 {
		t.Error("the refusal should name the offending field")
	}
}

// FR-015, FR-027: the owner gets the bytes; anyone else gets 404, never 403.
func TestRenditionAuthorization(t *testing.T) {
	h := newHarness(t)
	ownerCookie := h.sessionFor(t, "")

	created := decode(t, upload(t, h, "s.jpg", "", jpegBytes(t, 500, 600), ownerCookie))
	renditionURL := h.server.URL + created["renditionUrl"].(string)

	// Owner: 200 and a JPEG at exactly the gallery geometry.
	req, _ := http.NewRequest(http.MethodGet, renditionURL, nil)
	resp := h.do(t, req, ownerCookie)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("owner status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "image/jpeg" {
		t.Errorf("content type = %q, want image/jpeg", ct)
	}
	if cc := resp.Header.Get("Cache-Control"); !bytes.Contains([]byte(cc), []byte("private")) {
		t.Errorf("Cache-Control = %q, must be private so no shared cache holds one collector's image", cc)
	}
	cfg, _, err := image.DecodeConfig(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("rendition did not decode: %v", err)
	}
	if cfg.Width != imaging.RenditionWidth || cfg.Height != imaging.RenditionHeight {
		t.Errorf("rendition = %dx%d, want %dx%d", cfg.Width, cfg.Height, imaging.RenditionWidth, imaging.RenditionHeight)
	}

	// Another collector: 404, indistinguishable from an image that never existed.
	otherCookie := h.sessionFor(t, "second")
	req, _ = http.NewRequest(http.MethodGet, renditionURL, nil)
	resp = h.do(t, req, otherCookie)
	if resp.StatusCode == http.StatusForbidden {
		t.Fatal("403 confirms the image exists; FR-027 requires 404")
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("other collector status = %d, want 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// A nonexistent image: the same 404.
	req, _ = http.NewRequest(http.MethodGet, h.server.URL+"/api/images/"+uuid.New().String()+"/rendition", nil)
	resp = h.do(t, req, otherCookie)
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("nonexistent image status = %d, want the identical 404", resp.StatusCode)
	}
	_ = resp.Body.Close()

	// No session at all: 401.
	req, _ = http.NewRequest(http.MethodGet, renditionURL, nil)
	resp = h.do(t, req, nil)
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("unauthenticated status = %d, want 401", resp.StatusCode)
	}
	_ = resp.Body.Close()
}

// The listing shape, including what distinguishes an empty vault from a filter that matched
// nothing (FR-040, FR-041).
func TestListCollectiblesShape(t *testing.T) {
	h := newHarness(t)
	cookie := h.sessionFor(t, "")

	req, _ := http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles", nil)
	body := decode(t, h.do(t, req, cookie))
	items, ok := body["items"].([]any)
	if !ok {
		t.Fatalf("items is %T, want an array even when empty", body["items"])
	}
	if len(items) != 0 {
		t.Errorf("%d items in an empty vault", len(items))
	}
	if body["totalUnfiltered"] == nil {
		t.Error("totalUnfiltered is required on every page")
	}

	postJSON(t, h, `{"submissionKey":"L1","name":"One","collectionStatus":"owned"}`, cookie)
	req, _ = http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?status=sold", nil)
	body = decode(t, h.do(t, req, cookie))
	if n := len(body["items"].([]any)); n != 0 {
		t.Errorf("%d sold items, want 0", n)
	}
	if total, _ := body["totalUnfiltered"].(float64); total != 1 {
		t.Errorf("totalUnfiltered = %v, want 1 — this is what separates a no-results filter from an empty vault", body["totalUnfiltered"])
	}

	// An unrecognised status is refused rather than quietly ignored.
	req, _ = http.NewRequest(http.MethodGet, h.server.URL+"/api/collectibles?status=borrowed", nil)
	resp := h.do(t, req, cookie)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status=borrowed returned %d, want 400", resp.StatusCode)
	}
	_ = resp.Body.Close()
}
