// Package imagestore stores and retrieves image bytes by locator.
//
// It knows nothing about image formats, dimensions, or collectibles: deciding what is a valid
// image belongs to internal/imaging, and deciding who may read one belongs to the transport and
// the store. This package only moves bytes.
package imagestore

import (
	"context"
	"errors"
	"io"
)

// ErrNotFound means no object exists at that locator.
var ErrNotFound = errors.New("image not found in store")

// Store is the seam between Vaultory and wherever image bytes actually live. The filesystem
// implementation is for development; a production deployment swaps in an object store without
// changing anything above this interface.
type Store interface {
	// Put writes the bytes and returns nothing: the caller chose the locator, so it already knows
	// where the object is.
	Put(ctx context.Context, key string, data []byte) error

	// Open returns a reader for the object. The caller closes it.
	Open(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes the object. Deleting something absent is not an error, so cleaning up a
	// half-finished upload never needs to check first.
	Delete(ctx context.Context, key string) error
}
