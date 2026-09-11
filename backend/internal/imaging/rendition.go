package imaging

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"

	"golang.org/x/image/draw"
)

// Gallery rendition geometry, fixed for every collectible (FR-014, Clarifications 2026-09-10).
//
// 4:5 portrait, because figures, statues, and boxed collectibles are predominantly taller than
// wide — a square frame either trims the top off a tall statue or strands it in margins.
// 800x1000 covers a roughly 400-pixel-wide card at double density, the largest the gallery grid
// uses at any supported width.
const (
	RenditionWidth  = 800
	RenditionHeight = 1000

	// renditionQuality trades a little size for a gallery that still looks premium at 2x.
	renditionQuality = 82
)

// RenditionContentType is what every rendition is encoded as.
//
// JPEG rather than WebP: the supported Go packages decode WebP but cannot encode it, so WebP
// renditions would mean a cgo-backed dependency for no present benefit (research.md Decision 4).
const RenditionContentType = "image/jpeg"

// DeriveRendition scales and centre-crops an image to exactly RenditionWidth x RenditionHeight.
//
// The frame is filled and the overflow trimmed from the centre — never letterboxed, never
// distorted. That is what makes a panorama and a tall box sit side by side in the gallery without
// the collector touching either (FR-014, SC-013).
func DeriveRendition(src image.Image) ([]byte, error) {
	b := src.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return nil, fmt.Errorf("cannot derive a rendition from an empty image")
	}

	// Scale so the *smaller* relative dimension covers the frame, then crop the excess. Choosing
	// the larger scale factor of the two is what guarantees full coverage.
	scaleX := float64(RenditionWidth) / float64(b.Dx())
	scaleY := float64(RenditionHeight) / float64(b.Dy())
	scale := scaleX
	if scaleY > scale {
		scale = scaleY
	}

	scaledW := int(float64(b.Dx())*scale + 0.5)
	scaledH := int(float64(b.Dy())*scale + 0.5)
	// Rounding can land a pixel short on extreme ratios; never leave the frame uncovered.
	if scaledW < RenditionWidth {
		scaledW = RenditionWidth
	}
	if scaledH < RenditionHeight {
		scaledH = RenditionHeight
	}

	scaled := image.NewRGBA(image.Rect(0, 0, scaledW, scaledH))
	// CatmullRom over a faster kernel: this runs once per upload, and the result is the first
	// thing a collector sees of their own collectible.
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), src, b, draw.Src, nil)

	offsetX := (scaledW - RenditionWidth) / 2
	offsetY := (scaledH - RenditionHeight) / 2
	cropped := image.NewRGBA(image.Rect(0, 0, RenditionWidth, RenditionHeight))
	draw.Draw(cropped, cropped.Bounds(), scaled,
		image.Pt(offsetX, offsetY), draw.Src)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, cropped, &jpeg.Options{Quality: renditionQuality}); err != nil {
		return nil, fmt.Errorf("encode rendition: %w", err)
	}
	return buf.Bytes(), nil
}
