package collectible

import "time"

// EditDraft is a submission to change a collectible that already exists: the same attribute values
// an add carries, plus the version the collector was looking at when they started.
//
// It differs from Draft in exactly one field, and the difference is real rather than incidental. A
// submission key separates a retry from a deliberate second copy, which is meaningless here — an
// edit addresses one collectible that already exists and can only ever change that one. An
// expected version is meaningless when adding, because a new collectible has nothing to be stale
// against.
type EditDraft struct {
	// ExpectedVersion is the collectible's version as the collector was shown it. The store
	// compares it against the stored version and refuses the edit if it has moved on, rather than
	// writing over whatever changed in the meantime (FR-027, FR-027a).
	ExpectedVersion int
	Submitted
}

// ValidatedEdit is an EditDraft that has passed every rule, ready to apply.
type ValidatedEdit struct {
	ExpectedVersion int
	Values
}

// Validate checks every rule and returns all violations together (FR-014).
//
// The attribute rules are not restated here. They come from the same routine adding uses, which is
// what makes SC-008 a property of the code rather than a promise: every rule that rejects a value
// when adding rejects it when editing, because there is only one copy of each rule.
func (e EditDraft) Validate(today time.Time) (ValidatedEdit, []Violation) {
	var v []Violation
	out := ValidatedEdit{}

	// An edit that names no version cannot be checked for staleness, and accepting it would mean
	// silently overwriting whatever changed since the collector opened the collectible — which is
	// the one thing this feature exists to prevent.
	if e.ExpectedVersion < 1 {
		v = append(v, Violation{
			"expectedVersion", CodeRequired,
			"This edit is missing the version of the collectible it was based on.",
		})
	} else {
		out.ExpectedVersion = e.ExpectedVersion
	}

	values, shared := e.Submitted.validate(today)
	v = append(v, shared...)
	if len(v) > 0 {
		return ValidatedEdit{}, v
	}
	out.Values = values
	return out, nil
}
