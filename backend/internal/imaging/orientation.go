package imaging

import (
	"encoding/binary"
	"image"
	"image/draw"
)

// Orientation is the EXIF orientation tag (0x0112). 1 means "as stored".
type Orientation int

const OrientationNormal Orientation = 1

// OrientationOf finds the EXIF orientation in a JPEG's APP1 segment.
//
// Go's standard library does not parse EXIF, and the alternative was a dependency for one 16-bit
// integer. This walks the JPEG segment markers to the Exif APP1 block, reads the TIFF header to
// learn the byte order, and looks for tag 0x0112 in IFD0. Anything unexpected returns
// OrientationNormal — an image that renders slightly wrong is better than a decode that fails.
func OrientationOf(raw []byte) Orientation {
	// JPEG only. PNG has no orientation tag, and the WebP decoder here does not expose one.
	if len(raw) < 4 || raw[0] != 0xFF || raw[1] != 0xD8 {
		return OrientationNormal
	}

	i := 2
	for i+4 <= len(raw) {
		if raw[i] != 0xFF {
			return OrientationNormal
		}
		marker := raw[i+1]
		// Standalone markers carry no length.
		if marker == 0xD8 || marker == 0xD9 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		if i+4 > len(raw) {
			return OrientationNormal
		}
		segLen := int(binary.BigEndian.Uint16(raw[i+2 : i+4]))
		if segLen < 2 || i+2+segLen > len(raw) {
			return OrientationNormal
		}
		if marker == 0xE1 { // APP1
			payload := raw[i+4 : i+2+segLen]
			if len(payload) >= 6 && string(payload[:4]) == "Exif" && payload[4] == 0 {
				if o, ok := orientationFromTIFF(payload[6:]); ok {
					return o
				}
			}
		}
		// Start of scan: image data follows, no more metadata worth walking.
		if marker == 0xDA {
			return OrientationNormal
		}
		i += 2 + segLen
	}
	return OrientationNormal
}

func orientationFromTIFF(tiff []byte) (Orientation, bool) {
	if len(tiff) < 8 {
		return OrientationNormal, false
	}
	var bo binary.ByteOrder
	switch {
	case tiff[0] == 'I' && tiff[1] == 'I':
		bo = binary.LittleEndian
	case tiff[0] == 'M' && tiff[1] == 'M':
		bo = binary.BigEndian
	default:
		return OrientationNormal, false
	}
	if bo.Uint16(tiff[2:4]) != 42 {
		return OrientationNormal, false
	}
	ifdOffset := int(bo.Uint32(tiff[4:8]))
	if ifdOffset < 8 || ifdOffset+2 > len(tiff) {
		return OrientationNormal, false
	}
	count := int(bo.Uint16(tiff[ifdOffset : ifdOffset+2]))
	entry := ifdOffset + 2
	for n := 0; n < count; n++ {
		if entry+12 > len(tiff) {
			return OrientationNormal, false
		}
		tag := bo.Uint16(tiff[entry : entry+2])
		if tag == 0x0112 {
			// Type 3 (SHORT); the value sits in the first two bytes of the value field.
			value := int(bo.Uint16(tiff[entry+8 : entry+10]))
			if value >= 1 && value <= 8 {
				return Orientation(value), true
			}
			return OrientationNormal, false
		}
		entry += 12
	}
	return OrientationNormal, false
}

// ApplyOrientation returns the image as it is meant to be seen.
//
// The eight EXIF orientations combine a flip and a rotation. Rather than eight special cases, each
// is expressed as a coordinate mapping from destination back to source.
func ApplyOrientation(src image.Image, o Orientation) image.Image {
	if o == OrientationNormal || o < 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()

	// Orientations 5 through 8 involve a 90-degree turn, so the frame's axes swap.
	swapped := o >= 5
	dstW, dstH := w, h
	if swapped {
		dstW, dstH = h, w
	}

	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			var sx, sy int
			switch o {
			case 2: // mirror horizontally
				sx, sy = w-1-x, y
			case 3: // rotate 180
				sx, sy = w-1-x, h-1-y
			case 4: // mirror vertically
				sx, sy = x, h-1-y
			case 5: // transpose
				sx, sy = y, x
			case 6: // rotate 90 clockwise
				sx, sy = y, h-1-x
			case 7: // transverse
				sx, sy = w-1-y, h-1-x
			case 8: // rotate 270 clockwise
				sx, sy = w-1-y, x
			default:
				sx, sy = x, y
			}
			dst.Set(x, y, src.At(b.Min.X+sx, b.Min.Y+sy))
		}
	}
	_ = draw.Src
	return dst
}
