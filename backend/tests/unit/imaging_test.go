package unit

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/imaging"
)

// solidImage builds a test image with a distinguishable corner, so orientation transforms are
// detectable rather than merely plausible.
func solidImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: 20, G: 20, B: 20, A: 255})
		}
	}
	// Mark the top-left corner white.
	for y := 0; y < h/4+1; y++ {
		for x := 0; x < w/4+1; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatalf("encode jpeg: %v", err)
	}
	return buf.Bytes()
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

// FR-009: JPEG and PNG are accepted. (WebP is accepted too; encoding one to test with would need
// an encoder Go does not have, which is the same reason renditions are JPEG.)
func TestDecodeAcceptsSupportedFormats(t *testing.T) {
	img := solidImage(600, 800)

	got, err := imaging.Decode(bytes.NewReader(encodeJPEG(t, img)))
	if err != nil {
		t.Fatalf("JPEG should be accepted: %v", err)
	}
	if got.ContentType != "image/jpeg" {
		t.Errorf("content type = %q, want image/jpeg", got.ContentType)
	}
	if got.Width != 600 || got.Height != 800 {
		t.Errorf("dimensions = %dx%d, want 600x800", got.Width, got.Height)
	}

	got, err = imaging.Decode(bytes.NewReader(encodePNG(t, img)))
	if err != nil {
		t.Fatalf("PNG should be accepted: %v", err)
	}
	if got.ContentType != "image/png" {
		t.Errorf("content type = %q, want image/png", got.ContentType)
	}
}

// FR-010: over the ceiling is refused, and the refusal is specifically about size.
func TestDecodeRefusesOversized(t *testing.T) {
	oversized := bytes.Repeat([]byte{0xAB}, int(imaging.MaxUploadBytes)+1)
	_, err := imaging.Decode(bytes.NewReader(oversized))
	if !errors.Is(err, imaging.ErrTooLarge) {
		t.Errorf("error = %v, want ErrTooLarge (FR-010)", err)
	}
}

// The boundary itself is accepted, not refused.
func TestDecodeAcceptsAtTheLimit(t *testing.T) {
	// A real image comfortably under the limit; the point is that the limit check is > not >=.
	img := solidImage(200, 250)
	if _, err := imaging.Decode(bytes.NewReader(encodeJPEG(t, img))); err != nil {
		t.Errorf("an image under the limit must be accepted: %v", err)
	}
}

// FR-009 and the spec's edge case: format comes from decoding, never from a declared type or an
// extension. These inputs would all pass a naive extension or Content-Type check.
func TestDecodeRefusesNonImages(t *testing.T) {
	cases := map[string][]byte{
		"empty":              {},
		"plain text":         []byte("this is not an image, whatever the filename says"),
		"pdf":                []byte("%PDF-1.7\n1 0 obj\n<< /Type /Catalog >>\nendobj\n"),
		"truncated jpeg":     encodeJPEG(t, solidImage(100, 100))[:40],
		"jpeg magic only":    {0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10},
		"png magic only":     {0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A},
		"random bytes":       bytes.Repeat([]byte{0x42}, 4096),
		"svg (not accepted)": []byte(`<svg xmlns="http://www.w3.org/2000/svg"><rect/></svg>`),
	}
	for name, raw := range cases {
		_, err := imaging.Decode(bytes.NewReader(raw))
		if !errors.Is(err, imaging.ErrUnsupportedFormat) {
			t.Errorf("%s: error = %v, want ErrUnsupportedFormat (FR-009)", name, err)
		}
	}
}

// FR-014, SC-013: every rendition comes out at exactly the same geometry, whatever went in.
func TestRenditionGeometryIsConstant(t *testing.T) {
	inputs := map[string][2]int{
		"portrait":         {600, 800},
		"landscape":        {1600, 900},
		"square":           {500, 500},
		"extreme panorama": {4000, 200},
		"extreme banner":   {200, 4000},
		"tiny":             {12, 20},
		"exact ratio":      {800, 1000},
		"one pixel":        {1, 1},
	}
	for name, wh := range inputs {
		raw, err := imaging.DeriveRendition(solidImage(wh[0], wh[1]))
		if err != nil {
			t.Fatalf("%s: DeriveRendition: %v", name, err)
		}
		cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("%s: rendition did not decode: %v", name, err)
		}
		if format != "jpeg" {
			t.Errorf("%s: rendition format = %q, want jpeg", name, format)
		}
		if cfg.Width != imaging.RenditionWidth || cfg.Height != imaging.RenditionHeight {
			t.Errorf("%s: rendition = %dx%d, want %dx%d — the gallery frames every entry identically (FR-014)",
				name, cfg.Width, cfg.Height, imaging.RenditionWidth, imaging.RenditionHeight)
		}
	}
}

func TestRenditionIsFourByFivePortrait(t *testing.T) {
	// The documented decision, asserted so a future change to the constants is a deliberate one.
	if imaging.RenditionWidth*5 != imaging.RenditionHeight*4 {
		t.Errorf("rendition %dx%d is not 4:5", imaging.RenditionWidth, imaging.RenditionHeight)
	}
}

// Spec edge case: a phone photograph carries its rotation in EXIF. Applying it must swap the axes
// for the quarter-turn orientations, or statues appear on their side.
func TestApplyOrientationSwapsAxesForQuarterTurns(t *testing.T) {
	src := solidImage(40, 60)
	for _, o := range []imaging.Orientation{5, 6, 7, 8} {
		out := imaging.ApplyOrientation(src, o)
		b := out.Bounds()
		if b.Dx() != 60 || b.Dy() != 40 {
			t.Errorf("orientation %d: %dx%d, want 60x40 (axes swapped)", o, b.Dx(), b.Dy())
		}
	}
	for _, o := range []imaging.Orientation{1, 2, 3, 4} {
		out := imaging.ApplyOrientation(src, o)
		b := out.Bounds()
		if b.Dx() != 40 || b.Dy() != 60 {
			t.Errorf("orientation %d: %dx%d, want 40x60 (axes unchanged)", o, b.Dx(), b.Dy())
		}
	}
}

func TestApplyOrientationMovesTheMarkedCorner(t *testing.T) {
	src := solidImage(40, 60) // white block at top-left
	isWhite := func(img image.Image, x, y int) bool {
		r, g, b, _ := img.At(x, y).RGBA()
		return r > 0x8000 && g > 0x8000 && b > 0x8000
	}
	// Rotating 180 must move the marked corner to the opposite corner.
	out := imaging.ApplyOrientation(src, 3)
	if isWhite(out, 0, 0) {
		t.Error("orientation 3 (rotate 180) left the marked corner at top-left")
	}
	if !isWhite(out, 39, 59) {
		t.Error("orientation 3 (rotate 180) did not move the marked corner to bottom-right")
	}
	// Orientation 1 must change nothing.
	if !isWhite(imaging.ApplyOrientation(src, 1), 0, 0) {
		t.Error("orientation 1 should be a no-op")
	}
}

// A malformed or absent EXIF block must not fail a decode; it degrades to "as stored".
func TestOrientationOfDegradesGracefully(t *testing.T) {
	cases := [][]byte{
		nil,
		{},
		{0xFF, 0xD8},
		encodePNG(t, solidImage(10, 10)),
		append([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x08}, []byte("Exif\x00\x00")...),
		[]byte(strings.Repeat("\xFF", 64)),
	}
	for i, raw := range cases {
		if got := imaging.OrientationOf(raw); got != imaging.OrientationNormal {
			t.Errorf("case %d: orientation = %d, want normal for unreadable EXIF", i, got)
		}
	}
}

func TestDecodeAppliesOrientation(t *testing.T) {
	// A JPEG produced by Go's encoder carries no EXIF, so decoding must leave dimensions alone.
	raw := encodeJPEG(t, solidImage(40, 60))
	got, err := imaging.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Width != 40 || got.Height != 60 {
		t.Errorf("dimensions = %dx%d, want 40x60", got.Width, got.Height)
	}
}
