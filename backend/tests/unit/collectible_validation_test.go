package unit

import (
	"strings"
	"testing"
	"time"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

var today = time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)

// minimalSubmitted is the smallest valid set of attributes: a name and a status, nothing else
// (FR-005). Both entry points take it, which is the point.
func minimalSubmitted() collectible.Submitted {
	return collectible.Submitted{Name: "Kaiju Sentinel", Status: "owned"}
}

func minimalDraft() collectible.Draft {
	return collectible.Draft{SubmissionKey: "key-1", Submitted: minimalSubmitted()}
}

func minimalEdit() collectible.EditDraft {
	return collectible.EditDraft{ExpectedVersion: 1, Submitted: minimalSubmitted()}
}

func ptr(s string) *string { return &s }

func fieldsOf(v []collectible.Violation) map[string]string {
	m := make(map[string]string, len(v))
	for _, x := range v {
		m[x.Field] = x.Code
	}
	return m
}

// FR-005: a name and a status alone constitute a valid collectible.
func TestNameAndStatusAloneAreValid(t *testing.T) {
	got, v := minimalDraft().Validate(today)
	if len(v) != 0 {
		t.Fatalf("expected no violations, got %+v", v)
	}
	if got.Name != "Kaiju Sentinel" || got.Status != collectible.StatusOwned {
		t.Errorf("unexpected result: %+v", got)
	}
	// FR-007: nothing recorded stays absent rather than becoming empty strings or zeros.
	if got.Character != nil || got.PurchasePrice != nil || got.PurchaseDate != nil || got.ImageID != nil {
		t.Error("unsupplied optional values should be absent, not zero")
	}
}

// FR-002: both required values are reported when both are missing.
func TestNameAndStatusAreRequired(t *testing.T) {
	d := collectible.Draft{SubmissionKey: "k"}
	_, v := d.Validate(today)
	f := fieldsOf(v)
	if f["name"] != collectible.CodeRequired {
		t.Error("a missing name must be reported as required (FR-002)")
	}
	if f["collectionStatus"] != collectible.CodeRequired {
		t.Error("a missing status must be reported as required (FR-002)")
	}
}

// FR-003: whitespace alone is not a name.
func TestWhitespaceOnlyNameIsMissing(t *testing.T) {
	for _, name := range []string{" ", "   ", "\t", "\n", " \t\n "} {
		d := minimalDraft()
		d.Name = name
		_, v := d.Validate(today)
		if fieldsOf(v)["name"] != collectible.CodeRequired {
			t.Errorf("name %q should be treated as missing (FR-003)", name)
		}
	}
}

func TestNameIsTrimmed(t *testing.T) {
	d := minimalDraft()
	d.Name = "  Kaiju Sentinel  "
	got, v := d.Validate(today)
	if len(v) != 0 {
		t.Fatalf("unexpected violations: %+v", v)
	}
	if got.Name != "Kaiju Sentinel" {
		t.Errorf("name = %q, want trimmed", got.Name)
	}
}

// Spec edge case: a very long name is refused with a length message rather than silently truncated.
func TestNameLengthCap(t *testing.T) {
	d := minimalDraft()
	d.Name = strings.Repeat("a", collectible.MaxNameLength+1)
	_, v := d.Validate(today)
	if fieldsOf(v)["name"] != collectible.CodeTooLong {
		t.Error("an over-length name must be refused as too long")
	}

	d.Name = strings.Repeat("a", collectible.MaxNameLength)
	if _, v := d.Validate(today); len(v) != 0 {
		t.Errorf("a name at exactly the limit must be accepted, got %+v", v)
	}
}

// FR-004: exactly four statuses.
func TestStatusMustBeOneOfFour(t *testing.T) {
	for _, s := range []string{"owned", "preordered", "wishlist", "sold"} {
		d := minimalDraft()
		d.Status = s
		if _, v := d.Validate(today); len(v) != 0 {
			t.Errorf("status %q should be accepted, got %+v", s, v)
		}
	}
	for _, s := range []string{"borrowed", "OWNED", "Owned", "traded", "0"} {
		d := minimalDraft()
		d.Status = s
		if fieldsOf(v(d))["collectionStatus"] != collectible.CodeInvalidValue {
			t.Errorf("status %q must be rejected (FR-004)", s)
		}
	}
}

// v runs validation and returns just the violations, keeping the tables above readable.
func v(d collectible.Draft) []collectible.Violation {
	_, vs := d.Validate(today)
	return vs
}

// FR-020: every problem is reported together, not one at a time.
func TestAllViolationsReportedTogether(t *testing.T) {
	d := collectible.Draft{SubmissionKey: "k", Submitted: collectible.Submitted{
		Name:          "   ",
		Status:        "borrowed",
		PurchasePrice: ptr("-5.00"),
		PurchaseDate:  ptr("2030-01-01"),
	}}
	_, vs := d.Validate(today)
	f := fieldsOf(vs)
	for field, want := range map[string]string{
		"name":             collectible.CodeRequired,
		"collectionStatus": collectible.CodeInvalidValue,
		"purchasePrice":    collectible.CodeNegative,
		"purchaseDate":     collectible.CodeDateInFuture,
	} {
		if f[field] != want {
			t.Errorf("field %q: code %q, want %q", field, f[field], want)
		}
	}
	if len(vs) < 4 {
		t.Errorf("got %d violations, want at least 4 reported together (FR-020)", len(vs))
	}
}

// FR-007: an optional value that is empty once trimmed is recorded as absent, not as "".
func TestEmptyOptionalsBecomeAbsent(t *testing.T) {
	d := minimalDraft()
	d.Character = ptr("   ")
	d.Notes = ptr("")
	d.PurchasePrice = ptr("  ")
	got, vs := d.Validate(today)
	if len(vs) != 0 {
		t.Fatalf("unexpected violations: %+v", vs)
	}
	if got.Character != nil || got.Notes != nil || got.PurchasePrice != nil {
		t.Error("blank optional values must be absent rather than empty (FR-007)")
	}
}

// FR-006: every optional attribute is recorded when supplied.
func TestAllOptionalAttributesRecorded(t *testing.T) {
	d := minimalDraft()
	d.Character = ptr("Sentinel Prime")
	d.Series = ptr("Kaiju Wars")
	d.Manufacturer = ptr("Apex Studio")
	d.Category = ptr("Statue")
	d.Scale = ptr("1/4")
	d.Edition = ptr("Deluxe Exclusive")
	d.PurchasePrice = ptr("1250.00")
	d.PurchaseDate = ptr("2026-08-14")
	d.ReleaseDate = ptr("2026-11-30")
	d.Notes = ptr("Box has a small dent.")

	got, vs := d.Validate(today)
	if len(vs) != 0 {
		t.Fatalf("unexpected violations: %+v", vs)
	}
	if got.Character == nil || *got.Character != "Sentinel Prime" {
		t.Error("character not recorded")
	}
	if got.PurchasePrice == nil || got.PurchasePrice.String() != "1250.00" {
		t.Error("purchase price not recorded exactly")
	}
	if got.PurchaseDate == nil || got.PurchaseDate.Format(time.DateOnly) != "2026-08-14" {
		t.Error("purchase date not recorded")
	}
	if got.ReleaseDate == nil || got.ReleaseDate.Format(time.DateOnly) != "2026-11-30" {
		t.Error("release date not recorded")
	}
}

// FR-018: a purchase date later than the collector's today is refused; today itself is fine.
func TestPurchaseDateNotInFuture(t *testing.T) {
	d := minimalDraft()
	d.PurchaseDate = ptr("2026-09-11")
	if fieldsOf(v(d))["purchaseDate"] != collectible.CodeDateInFuture {
		t.Error("a future purchase date must be refused (FR-018)")
	}

	d.PurchaseDate = ptr("2026-09-10") // today
	if vs := v(d); len(vs) != 0 {
		t.Errorf("today must be accepted as a purchase date, got %+v", vs)
	}

	d.PurchaseDate = ptr("1999-01-01")
	if vs := v(d); len(vs) != 0 {
		t.Errorf("a past purchase date must be accepted, got %+v", vs)
	}
}

// FR-019: a release date is accepted in the past or the future, in any combination.
func TestReleaseDateUnbounded(t *testing.T) {
	cases := []struct{ status, release, purchase string }{
		{"preordered", "1999-01-01", ""},      // preorder whose release already passed
		{"owned", "2099-12-31", "2026-01-01"}, // owned, not yet released
		{"sold", "2030-06-01", "2026-05-05"},  // sold before release
		{"wishlist", "2026-09-10", ""},        // released today
	}
	for _, c := range cases {
		d := minimalDraft()
		d.Status = c.status
		d.ReleaseDate = ptr(c.release)
		if c.purchase != "" {
			d.PurchaseDate = ptr(c.purchase)
		}
		if vs := v(d); len(vs) != 0 {
			t.Errorf("status=%s release=%s purchase=%s should be accepted (FR-019), got %+v",
				c.status, c.release, c.purchase, vs)
		}
	}
}

// FR-047: the submission key is required and length-bounded.
func TestSubmissionKeyRequired(t *testing.T) {
	d := minimalDraft()
	d.SubmissionKey = ""
	if fieldsOf(v(d))["submissionKey"] != collectible.CodeRequired {
		t.Error("a submission key is required (FR-047)")
	}

	d.SubmissionKey = strings.Repeat("k", collectible.MaxSubmissionKeyLen+1)
	if fieldsOf(v(d))["submissionKey"] != collectible.CodeTooLong {
		t.Error("an over-length submission key must be refused")
	}
}

// Spec edge case: non-Latin scripts, accents, and emoji are stored faithfully.
func TestTextFidelity(t *testing.T) {
	for _, name := range []string{"怪獣センチネル", "Ámbar Guardián", "Sentinel 🦖", "Кайдзю"} {
		d := minimalDraft()
		d.Name = name
		got, vs := d.Validate(today)
		if len(vs) != 0 {
			t.Fatalf("name %q should be accepted: %+v", name, vs)
		}
		if got.Name != name {
			t.Errorf("name %q came back as %q", name, got.Name)
		}
	}
}

// Length limits count characters, not bytes, so a name of multi-byte characters is not penalised.
func TestLengthLimitsCountCharacters(t *testing.T) {
	d := minimalDraft()
	d.Name = strings.Repeat("怪", collectible.MaxNameLength)
	if vs := v(d); len(vs) != 0 {
		t.Errorf("200 multi-byte characters must be accepted, got %+v", vs)
	}
}

func TestInvalidImageReference(t *testing.T) {
	d := minimalDraft()
	d.ImageID = ptr("not-a-uuid")
	if fieldsOf(v(d))["imageId"] != collectible.CodeInvalidValue {
		t.Error("a malformed image reference must be refused")
	}
}

// ---------------------------------------------------------------------------
// T011 — adding and editing apply one rule set
// ---------------------------------------------------------------------------

// Every case runs through both entry points and must produce the same verdict.
//
// This is SC-008 expressed as a test rather than as a promise: "every validation rule that rejects
// a value when adding rejects the same value when editing, with no rule applying in only one of
// the two". A rule added to one path and forgotten on the other fails here, which is the whole
// reason the two share a validator (FR-010).
func TestAddAndEditApplyTheSameRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*collectible.Submitted)
		want   map[string]string // field -> code; empty means the submission must be accepted
	}{
		{"minimal is valid", func(s *collectible.Submitted) {}, nil},
		{"name required", func(s *collectible.Submitted) { s.Name = "" },
			map[string]string{"name": collectible.CodeRequired}},
		{"whitespace name is missing", func(s *collectible.Submitted) { s.Name = " \t\n " },
			map[string]string{"name": collectible.CodeRequired}},
		{"name length cap", func(s *collectible.Submitted) { s.Name = strings.Repeat("a", collectible.MaxNameLength+1) },
			map[string]string{"name": collectible.CodeTooLong}},
		{"status required", func(s *collectible.Submitted) { s.Status = "" },
			map[string]string{"collectionStatus": collectible.CodeRequired}},
		{"status closed to four", func(s *collectible.Submitted) { s.Status = "borrowed" },
			map[string]string{"collectionStatus": collectible.CodeInvalidValue}},
		{"status is case sensitive", func(s *collectible.Submitted) { s.Status = "Owned" },
			map[string]string{"collectionStatus": collectible.CodeInvalidValue}},
		{"negative price", func(s *collectible.Submitted) { s.PurchasePrice = ptr("-0.01") },
			map[string]string{"purchasePrice": collectible.CodeNegative}},
		{"zero price is a recorded amount", func(s *collectible.Submitted) { s.PurchasePrice = ptr("0.00") }, nil},
		{"excess precision refused not rounded", func(s *collectible.Submitted) { s.PurchasePrice = ptr("1.005") },
			map[string]string{"purchasePrice": collectible.CodeInvalidValue}},
		{"future purchase date", func(s *collectible.Submitted) { s.PurchaseDate = ptr("2030-01-01") },
			map[string]string{"purchaseDate": collectible.CodeDateInFuture}},
		{"purchase date today is fine", func(s *collectible.Submitted) { s.PurchaseDate = ptr("2026-09-10") }, nil},
		{"malformed purchase date", func(s *collectible.Submitted) { s.PurchaseDate = ptr("14/08/2026") },
			map[string]string{"purchaseDate": collectible.CodeInvalidValue}},
		{"future release date is fine", func(s *collectible.Submitted) { s.ReleaseDate = ptr("2099-12-31") }, nil},
		{"notes length cap", func(s *collectible.Submitted) { s.Notes = ptr(strings.Repeat("n", collectible.MaxNotesLength+1)) },
			map[string]string{"notes": collectible.CodeTooLong}},
		{"category length cap", func(s *collectible.Submitted) { s.Category = ptr(strings.Repeat("c", collectible.MaxCategoryLength+1)) },
			map[string]string{"category": collectible.CodeTooLong}},
		{"scale length cap", func(s *collectible.Submitted) { s.Scale = ptr(strings.Repeat("s", collectible.MaxScaleLength+1)) },
			map[string]string{"scale": collectible.CodeTooLong}},
		{"malformed image reference", func(s *collectible.Submitted) { s.ImageID = ptr("not-a-uuid") },
			map[string]string{"imageId": collectible.CodeInvalidValue}},
		{"blank optionals become absent", func(s *collectible.Submitted) {
			s.Character = ptr("  ")
			s.Notes = ptr("")
		}, nil},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			added := minimalSubmitted()
			c.mutate(&added)
			_, addViolations := collectible.Draft{SubmissionKey: "key-1", Submitted: added}.Validate(today)

			edited := minimalSubmitted()
			c.mutate(&edited)
			_, editViolations := collectible.EditDraft{ExpectedVersion: 1, Submitted: edited}.Validate(today)

			addFields, editFields := fieldsOf(addViolations), fieldsOf(editViolations)

			for field, code := range c.want {
				if addFields[field] != code {
					t.Errorf("adding: field %q gave code %q, want %q", field, addFields[field], code)
				}
			}
			if len(c.want) == 0 && len(addViolations) != 0 {
				t.Errorf("adding: expected no violations, got %+v", addViolations)
			}

			// The real assertion. Not "editing also rejects this" but "editing reaches exactly the
			// same verdict", so a rule cannot be stricter or looser on one path.
			if len(addFields) != len(editFields) {
				t.Fatalf("adding reported %v, editing reported %v — the two paths disagree (SC-008)",
					addFields, editFields)
			}
			for field, code := range addFields {
				if editFields[field] != code {
					t.Errorf("field %q: adding says %q, editing says %q (SC-008)",
						field, code, editFields[field])
				}
			}
		})
	}
}

// FR-006: any status to any other, in any order, with no transition forbidden.
//
// This holds by construction today — the domain validates the new status and knows nothing of the
// old one — which is exactly the kind of property a later change removes without noticing.
func TestEveryStatusTransitionIsAllowed(t *testing.T) {
	statuses := []string{"owned", "preordered", "wishlist", "sold"}
	for _, from := range statuses {
		for _, to := range statuses {
			e := minimalEdit()
			e.Status = to
			if _, vs := e.Validate(today); len(vs) != 0 {
				t.Errorf("%s -> %s was refused: %+v — no transition is forbidden (FR-006)",
					from, to, vs)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// T012 — the one field each path has that the other does not
// ---------------------------------------------------------------------------

// FR-027a: an edit that names no version cannot be checked for staleness, and accepting it would
// mean overwriting whatever changed since the collector opened the collectible.
func TestEditRequiresTheVersionItWasBasedOn(t *testing.T) {
	for _, version := range []int{0, -1} {
		e := minimalEdit()
		e.ExpectedVersion = version
		if fieldsOf(violationsOfEdit(e))["expectedVersion"] != collectible.CodeRequired {
			t.Errorf("expectedVersion %d must be refused (FR-027a)", version)
		}
	}

	e := minimalEdit()
	e.ExpectedVersion = 7
	got, vs := e.Validate(today)
	if len(vs) != 0 {
		t.Fatalf("a valid edit was refused: %+v", vs)
	}
	if got.ExpectedVersion != 7 {
		t.Errorf("ExpectedVersion = %d, want 7", got.ExpectedVersion)
	}
}

// The mirror of the above: adding still requires its submission key, and editing must not have
// acquired one. Sharing a validator must not have loosened FR-047.
func TestSubmissionKeyBelongsToAddingOnly(t *testing.T) {
	d := minimalDraft()
	d.SubmissionKey = ""
	if fieldsOf(v(d))["submissionKey"] != collectible.CodeRequired {
		t.Error("adding still requires a submission key (FR-047)")
	}

	// An edit carries no key and must not be asked for one.
	if f := fieldsOf(violationsOfEdit(minimalEdit())); f["submissionKey"] != "" {
		t.Errorf("editing asked for a submission key (%q); it has nothing to de-duplicate", f["submissionKey"])
	}
}

func violationsOfEdit(e collectible.EditDraft) []collectible.Violation {
	_, vs := e.Validate(today)
	return vs
}
