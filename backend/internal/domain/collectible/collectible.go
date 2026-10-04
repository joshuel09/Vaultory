package collectible

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Field length limits. These are the spec's edge cases on very long names and notes, expressed as
// numbers so they are testable and enforced identically in the domain and the database.
const (
	MaxNameLength         = 200
	MaxCharacterLength    = 200
	MaxSeriesLength       = 200
	MaxManufacturerLength = 200
	MaxCategoryLength     = 100
	MaxScaleLength        = 50
	MaxEditionLength      = 200
	MaxNotesLength        = 2000
	MaxSubmissionKeyLen   = 200
)

// Violation is one thing wrong with a submission: which field, a stable code, and a message fit
// for a collector to read.
type Violation struct {
	Field   string
	Code    string
	Message string
}

// Codes used by validation. The transport maps these onto the API's field codes.
const (
	CodeRequired     = "required"
	CodeInvalidValue = "invalid_value"
	CodeNegative     = "negative_amount"
	CodeDateInFuture = "date_in_future"
	CodeTooLong      = "too_long"
)

// Submitted is the attribute values a collector sent, before validation, and is the same whether
// they are adding a collectible or editing one. Optional values are pointers so that "not
// recorded" is distinguishable from an empty string or a zero (FR-007).
//
// Adding and editing share this type and the routine that validates it, so a rule cannot apply to
// one and not the other (FR-010, SC-008).
type Submitted struct {
	Name          string
	Status        string
	Character     *string
	Series        *string
	Manufacturer  *string
	Category      *string
	Scale         *string
	Edition       *string
	PurchasePrice *string
	PurchaseDate  *string
	ReleaseDate   *string
	Notes         *string
	ImageID       *string
}

// Draft is a submission to add a collectible: the shared values plus the key that separates a
// retry from a deliberate second copy (FR-047).
type Draft struct {
	SubmissionKey string
	Submitted
}

// Collectible is a validated entry in a collector's vault.
type Collectible struct {
	ID            uuid.UUID
	CollectorID   uuid.UUID
	Name          string
	Status        CollectionStatus
	Character     *string
	Series        *string
	Manufacturer  *string
	Category      *string
	Scale         *string
	Edition       *string
	PurchasePrice *Money
	PurchaseDate  *time.Time
	ReleaseDate   *time.Time
	Notes         *string
	ImageID       *uuid.UUID
	CreatedAt     time.Time
	// Version changes on every change to this collectible. An edit presents the version it was
	// based on, and one that is no longer current is refused rather than applied over the newer
	// values (FR-027, FR-027a).
	Version int
}

// Values is every attribute of a collectible once it has passed every rule.
type Values struct {
	Name          string
	Status        CollectionStatus
	Character     *string
	Series        *string
	Manufacturer  *string
	Category      *string
	Scale         *string
	Edition       *string
	PurchasePrice *Money
	PurchaseDate  *time.Time
	ReleaseDate   *time.Time
	Notes         *string
	ImageID       *uuid.UUID
}

// Validated is a Draft that has passed every rule, ready to persist.
type Validated struct {
	SubmissionKey string
	Values
}

// Validate checks every rule and returns all violations together, never stopping at the first
// (FR-020). A collector who has three things wrong should be told three things, once.
//
// today is the collector's current date, passed in rather than read from the clock so the
// future-date rule is deterministic and testable, and so it is evaluated against the collector's
// date rather than the server's timezone (FR-018).
func (d Draft) Validate(today time.Time) (Validated, []Violation) {
	var v []Violation
	out := Validated{}

	// FR-047: the submission key is what separates a retry from a deliberate second copy. It is
	// the only rule that belongs to adding alone — an edit addresses a collectible that already
	// exists, so there is nothing for a key to de-duplicate.
	key := strings.TrimSpace(d.SubmissionKey)
	switch {
	case key == "":
		v = append(v, Violation{"submissionKey", CodeRequired, "A submission key is required."})
	case len(key) > MaxSubmissionKeyLen:
		v = append(v, Violation{"submissionKey", CodeTooLong, "That submission key is too long."})
	default:
		out.SubmissionKey = key
	}

	values, shared := d.Submitted.validate(today)
	v = append(v, shared...)
	if len(v) > 0 {
		return Validated{}, v
	}
	out.Values = values
	return out, nil
}

// validate checks every rule that applies to a collectible's attributes, whether it is being added
// or edited.
//
// This routine is the whole of SC-008: adding and editing do not merely happen to agree, they run
// the same code, so a rule cannot be added to one path and forgotten on the other.
func (d Submitted) validate(today time.Time) (Values, []Violation) {
	var v []Violation
	out := Values{}

	// FR-002, FR-003: a name is required, and whitespace alone is not a name.
	name := strings.TrimSpace(d.Name)
	switch {
	case name == "":
		v = append(v, Violation{"name", CodeRequired, "A name is required."})
	case len([]rune(name)) > MaxNameLength:
		v = append(v, Violation{"name", CodeTooLong, "That name is too long. Please use 200 characters or fewer."})
	default:
		out.Name = name
	}

	// FR-002, FR-004: a status is required and must be one of exactly four.
	switch {
	case strings.TrimSpace(d.Status) == "":
		v = append(v, Violation{"collectionStatus", CodeRequired, "A collection status is required."})
	default:
		status, err := ParseCollectionStatus(d.Status)
		if err != nil {
			v = append(v, Violation{"collectionStatus", CodeInvalidValue, "Choose Owned, Preordered, Wishlist, or Sold."})
		} else {
			out.Status = status
		}
	}

	// Optional text. Trimmed, and a value that is empty once trimmed is stored as absent rather
	// than as an empty string, so "not recorded" stays distinguishable (FR-007).
	type optField struct {
		name  string
		in    *string
		out   **string
		limit int
		label string
	}
	for _, f := range []optField{
		{"character", d.Character, &out.Character, MaxCharacterLength, "character"},
		{"series", d.Series, &out.Series, MaxSeriesLength, "series or franchise"},
		{"manufacturer", d.Manufacturer, &out.Manufacturer, MaxManufacturerLength, "manufacturer"},
		{"category", d.Category, &out.Category, MaxCategoryLength, "category"},
		{"scale", d.Scale, &out.Scale, MaxScaleLength, "scale"},
		{"edition", d.Edition, &out.Edition, MaxEditionLength, "edition or variant"},
		{"notes", d.Notes, &out.Notes, MaxNotesLength, "note"},
	} {
		if f.in == nil {
			continue
		}
		trimmed := strings.TrimSpace(*f.in)
		if trimmed == "" {
			continue // recorded nothing
		}
		if len([]rune(trimmed)) > f.limit {
			v = append(v, Violation{f.name, CodeTooLong, "That " + f.label + " is too long."})
			continue
		}
		value := trimmed
		*f.out = &value
	}

	// FR-016, FR-017: exact amounts; zero accepted, negative rejected, excess precision refused
	// rather than rounded.
	if d.PurchasePrice != nil && strings.TrimSpace(*d.PurchasePrice) != "" {
		raw := strings.TrimSpace(*d.PurchasePrice)
		money, err := ParseMoney(raw)
		switch {
		case err == nil:
			out.PurchasePrice = &money
		case strings.HasPrefix(raw, "-"):
			v = append(v, Violation{"purchasePrice", CodeNegative, "A purchase price cannot be negative."})
		default:
			v = append(v, Violation{"purchasePrice", CodeInvalidValue, "Enter an amount like 249.99, with at most two decimal places."})
		}
	}

	// FR-018: not later than the collector's today.
	if d.PurchaseDate != nil && strings.TrimSpace(*d.PurchaseDate) != "" {
		date, err := parseDate(*d.PurchaseDate)
		switch {
		case err != nil:
			v = append(v, Violation{"purchaseDate", CodeInvalidValue, "Enter a date like 2026-08-14."})
		case date.After(startOfDay(today)):
			v = append(v, Violation{"purchaseDate", CodeDateInFuture, "A purchase date cannot be in the future."})
		default:
			out.PurchaseDate = &date
		}
	}

	// FR-019: a release date may be in the past or the future, in any combination with the status
	// and the purchase date. A preorder whose release has passed, and an owned collectible not yet
	// released, are both real collector situations.
	if d.ReleaseDate != nil && strings.TrimSpace(*d.ReleaseDate) != "" {
		date, err := parseDate(*d.ReleaseDate)
		if err != nil {
			v = append(v, Violation{"releaseDate", CodeInvalidValue, "Enter a date like 2026-11-30."})
		} else {
			out.ReleaseDate = &date
		}
	}

	// FR-011: at most one image. Whether it belongs to this collector is checked against storage
	// by the application service, not here — the domain has no way to know.
	if d.ImageID != nil && strings.TrimSpace(*d.ImageID) != "" {
		id, err := uuid.Parse(strings.TrimSpace(*d.ImageID))
		if err != nil {
			v = append(v, Violation{"imageId", CodeInvalidValue, "That image reference is not valid."})
		} else {
			out.ImageID = &id
		}
	}

	if len(v) > 0 {
		return Values{}, v
	}
	return out, nil
}

func parseDate(raw string) (time.Time, error) {
	return time.ParseInLocation(time.DateOnly, strings.TrimSpace(raw), time.UTC)
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
