// Package imaging validates uploaded images and derives the gallery rendition.
//
// Two rules drive everything here. Format is determined by decoding the bytes, never by what the
// client declared or what the filename says (FR-009, Constitution III: validate external input at
// the boundary). And every rendition comes out at one fixed geometry, so the gallery frames every
// collectible identically without the collector cropping anything (FR-014).
package imaging

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"

	// Decoders for the three accepted formats. Registering them is what makes image.Decode able to
	// recognise the bytes; the format it reports is the authoritative answer.
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

// MaxUploadBytes is the 10 MB ceiling from FR-010.
const MaxUploadBytes int64 = 10 * 1024 * 1024

var (
	// ErrTooLarge means the upload exceeded MaxUploadBytes (FR-010).
	ErrTooLarge = errors.New("image exceeds the maximum size")
	// ErrUnsupportedFormat covers anything that is not a decodable JPEG, PNG, or WebP: a wrong
	// format, a corrupt file, or a non-image wearing an image extension (FR-009).
	ErrUnsupportedFormat = errors.New("image is not a supported format")
)

// contentTypes maps the format names the decoders report onto the media types the API and the
// database accept. A format absent from this map is not accepted, whatever decoder recognised it.
var contentTypes = map[string]string{
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"webp": "image/webp",
}

// Decoded is an accepted upload.
type Decoded struct {
	Image       image.Image
	ContentType string
	Bytes       []byte
	Width       int
	Height      int
}

// Decode reads at most MaxUploadBytes+1 bytes, refuses anything larger, and then insists the bytes
// decode as one of the three accepted formats.
//
// Reading one byte past the limit is how the size is detected without trusting a Content-Length
// the client controls.
func Decode(r io.Reader) (Decoded, error) {
	raw, err := io.ReadAll(io.LimitReader(r, MaxUploadBytes+1))
	if err != nil {
		return Decoded{}, fmt.Errorf("read upload: %w", err)
	}
	if int64(len(raw)) > MaxUploadBytes {
		return Decoded{}, ErrTooLarge
	}
	if len(raw) == 0 {
		return Decoded{}, ErrUnsupportedFormat
	}

	img, format, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		// A decode failure is the same answer as a wrong format, from the collector's point of
		// view: this file is not a usable image.
		return Decoded{}, ErrUnsupportedFormat
	}
	contentType, ok := contentTypes[format]
	if !ok {
		return Decoded{}, ErrUnsupportedFormat
	}

	// Orientation is applied here, before anything measures or resizes the image. A phone
	// photograph of a figure carries its rotation in EXIF rather than in the pixels; skip this and
	// the statue appears on its side in the gallery.
	img = ApplyOrientation(img, OrientationOf(raw))

	b := img.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		return Decoded{}, ErrUnsupportedFormat
	}

	return Decoded{
		Image:       img,
		ContentType: contentType,
		Bytes:       raw,
		Width:       b.Dx(),
		Height:      b.Dy(),
	}, nil
}
