package httpapi

import (
	"fmt"
	"time"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/imaging"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// Wire types. These mirror contracts/openapi.yaml exactly; the contract is the source of truth and
// the frontend's types are generated from it (Constitution Principle III).

type addCollectibleRequest struct {
	SubmissionKey string  `json:"submissionKey"`
	Name          string  `json:"name"`
	Status        string  `json:"collectionStatus"`
	Character     *string `json:"character"`
	Series        *string `json:"series"`
	Manufacturer  *string `json:"manufacturer"`
	Category      *string `json:"category"`
	Scale         *string `json:"scale"`
	Edition       *string `json:"edition"`
	PurchasePrice *string `json:"purchasePrice"`
	PurchaseDate  *string `json:"purchaseDate"`
	ReleaseDate   *string `json:"releaseDate"`
	Notes         *string `json:"notes"`
	ImageID       *string `json:"imageId"`
}

func (r addCollectibleRequest) toDraft() collectible.Draft {
	return collectible.Draft{
		SubmissionKey: r.SubmissionKey,
		Name:          r.Name,
		Status:        r.Status,
		Character:     r.Character,
		Series:        r.Series,
		Manufacturer:  r.Manufacturer,
		Category:      r.Category,
		Scale:         r.Scale,
		Edition:       r.Edition,
		PurchasePrice: r.PurchasePrice,
		PurchaseDate:  r.PurchaseDate,
		ReleaseDate:   r.ReleaseDate,
		Notes:         r.Notes,
		ImageID:       r.ImageID,
	}
}

type imageRefResponse struct {
	ID           string `json:"id"`
	RenditionURL string `json:"renditionUrl"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

type collectibleResponse struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Status        string            `json:"collectionStatus"`
	Character     *string           `json:"character"`
	Series        *string           `json:"series"`
	Manufacturer  *string           `json:"manufacturer"`
	Category      *string           `json:"category"`
	Scale         *string           `json:"scale"`
	Edition       *string           `json:"edition"`
	PurchasePrice *string           `json:"purchasePrice"`
	PurchaseDate  *string           `json:"purchaseDate"`
	ReleaseDate   *string           `json:"releaseDate"`
	Notes         *string           `json:"notes"`
	Image         *imageRefResponse `json:"image"`
	CreatedAt     string            `json:"createdAt"`
}

type collectionPageResponse struct {
	Items           []collectibleResponse `json:"items"`
	NextCursor      *string               `json:"nextCursor"`
	TotalUnfiltered int                   `json:"totalUnfiltered"`
}

type uploadedImageResponse struct {
	ID           string `json:"id"`
	RenditionURL string `json:"renditionUrl"`
	ContentType  string `json:"contentType"`
	ByteSize     int64  `json:"byteSize"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
}

// renditionPath is the address the gallery's <img> elements use. Same-origin through the Next.js
// rewrite, so the session cookie accompanies the request and FR-015's per-request check can run.
func renditionPath(imageID fmt.Stringer) string {
	return "/api/images/" + imageID.String() + "/rendition"
}

func toCollectibleResponse(row postgres.Row) collectibleResponse {
	c := row.Collectible
	out := collectibleResponse{
		ID:           c.ID.String(),
		Name:         c.Name,
		Status:       string(c.Status),
		Character:    c.Character,
		Series:       c.Series,
		Manufacturer: c.Manufacturer,
		Category:     c.Category,
		Scale:        c.Scale,
		Edition:      c.Edition,
		Notes:        c.Notes,
		CreatedAt:    c.CreatedAt.UTC().Format(time.RFC3339),
	}
	// An amount crosses as a decimal string, never a JSON number, so nothing on the far side can
	// turn it into a float (Constitution IV).
	if c.PurchasePrice != nil {
		s := c.PurchasePrice.String()
		out.PurchasePrice = &s
	}
	if c.PurchaseDate != nil {
		s := c.PurchaseDate.Format(time.DateOnly)
		out.PurchaseDate = &s
	}
	if c.ReleaseDate != nil {
		s := c.ReleaseDate.Format(time.DateOnly)
		out.ReleaseDate = &s
	}
	// Absent when the collector supplied no image; the gallery shows its designed placeholder
	// instead (FR-033).
	if c.ImageID != nil && row.RenditionKey != nil {
		out.Image = &imageRefResponse{
			ID:           c.ImageID.String(),
			RenditionURL: renditionPath(c.ImageID),
			Width:        imaging.RenditionWidth,
			Height:       imaging.RenditionHeight,
		}
	}
	return out
}

func toFieldErrors(violations []collectible.Violation) []FieldError {
	out := make([]FieldError, 0, len(violations))
	for _, v := range violations {
		out = append(out, FieldError{Field: v.Field, Code: v.Code, Message: v.Message})
	}
	return out
}
