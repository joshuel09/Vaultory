package httpapi

import (
	"errors"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/joshuel09/vaultory/backend/internal/collection"
	"github.com/joshuel09/vaultory/backend/internal/identity"
	"github.com/joshuel09/vaultory/backend/internal/imaging"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// multipartSlack allows for the multipart envelope around a 10 MB file. The file itself is bounded
// by imaging.Decode; this only stops an unbounded body from being buffered.
const multipartSlack = 1 * 1024 * 1024

func (s *Server) handleUploadImage(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	r.Body = http.MaxBytesReader(w, r.Body, imaging.MaxUploadBytes+multipartSlack)

	file, _, err := r.FormFile("file")
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeImageTooLarge(w)
			return
		}
		WriteError(w, http.StatusUnsupportedMediaType, CodeUnsupportedMedia,
			"Send the image as a multipart form field named \"file\".")
		return
	}
	defer func() { _ = file.Close() }()

	img, err := s.service.UploadImage(r.Context(), collector, file)
	switch {
	case errors.Is(err, collection.ErrImageTooLarge):
		writeImageTooLarge(w)
		return
	case errors.Is(err, collection.ErrImageUnsupported):
		// Note what is *not* consulted: the declared content type and the filename. The decode
		// decides (FR-009).
		WriteError(w, http.StatusBadRequest, CodeUnsupportedFormat,
			"That file is not a supported image. Vaultory accepts JPEG, PNG, and WebP.",
			FieldError{
				Field:   "file",
				Code:    CodeUnsupportedFormat,
				Message: "Accepted formats are JPEG, PNG, and WebP.",
			})
		return
	case err != nil:
		WriteInternal(w, r, err)
		return
	}

	WriteJSON(w, http.StatusCreated, uploadedImageResponse{
		ID:           img.ID.String(),
		RenditionURL: renditionPath(img.ID),
		ContentType:  img.ContentType,
		ByteSize:     img.ByteSize,
		Width:        img.Width,
		Height:       img.Height,
	})
}

func writeImageTooLarge(w http.ResponseWriter) {
	WriteError(w, http.StatusRequestEntityTooLarge, CodeImageTooLarge,
		"That image is larger than the 10 MB limit.",
		FieldError{
			Field:   "file",
			Code:    CodeImageTooLarge,
			Message: "Maximum image size is 10 MB.",
		})
}

func (s *Server) handleGetRendition(w http.ResponseWriter, r *http.Request, collector identity.CollectorID) {
	imageID, err := uuid.Parse(r.PathValue("imageId"))
	if err != nil {
		// A malformed identifier gets the same answer as one that does not exist, so probing
		// learns nothing (FR-027).
		WriteNotFound(w)
		return
	}

	rc, err := s.service.OpenRendition(r.Context(), collector, imageID)
	if errors.Is(err, postgres.ErrNotFound) {
		// 404, never 403: a 403 would confirm the image exists and belongs to someone (FR-027).
		WriteNotFound(w)
		return
	}
	if err != nil {
		WriteInternal(w, r, err)
		return
	}
	defer func() { _ = rc.Close() }()

	w.Header().Set("Content-Type", imaging.RenditionContentType)
	// Private only. A rendition is one collector's image; a shared cache must never hold it, or
	// the per-request check in FR-015 is bypassed by infrastructure.
	w.Header().Set("Cache-Control", "private, max-age=3600, no-store=false")
	w.Header().Set("Vary", "Cookie")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", "inline")

	if _, err := io.Copy(w, rc); err != nil {
		// Headers are already sent; the client will see a truncated image. Record it.
		WriteInternal(w, r, err)
	}
}
