package unit

import (
	"errors"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// T010: a total may exceed what any single recorded amount could have been.
//
// This is the whole reason ParseTotal exists. A vault of a thousand collectibles can easily total
// more than numeric(12,2) holds, and rejecting that would be rejecting a correct answer (FR-013).
func TestParseTotalAcceptsMoreThanOneAmountCanHold(t *testing.T) {
	// One more than the ten-integer-digit ceiling a stored amount is bound by.
	const beyondOneAmount = "10000000000.00"

	if _, err := collectible.ParseMoney(beyondOneAmount); err == nil {
		t.Fatalf("ParseMoney(%q) succeeded; the storage column's ceiling is meant to reject it, and "+
			"if that has changed then this test is no longer testing anything", beyondOneAmount)
	}

	got, err := collectible.ParseTotal(beyondOneAmount)
	if err != nil {
		t.Fatalf("ParseTotal(%q) = %v; a total is not bound by a single amount's ceiling", beyondOneAmount, err)
	}
	if got.String() != beyondOneAmount {
		t.Errorf("ParseTotal(%q).String() = %q, want the value back unchanged", beyondOneAmount, got.String())
	}
}

// T011: the representation's own ceiling is guarded, and the guard is reachable.
//
// Nothing realistic provokes this — a vault would need millions of maximum-value collectibles. It is
// tested because the alternative failure is silent: an unguarded int64 accumulation wraps to a
// negative total, which nobody reads as a bug.
func TestTotalBeyondTheRepresentationIsRefusedByName(t *testing.T) {
	// Sixteen integer digits is the ceiling; seventeen is past it.
	const beyondRepresentation = "10000000000000000.00"

	_, err := collectible.ParseTotal(beyondRepresentation)
	if !errors.Is(err, collectible.ErrTotalTooLarge) {
		t.Errorf("ParseTotal(%q) = %v, want ErrTotalTooLarge; a caller has to be able to tell this "+
			"apart from a malformed string", beyondRepresentation, err)
	}
}

// T011: addition refuses to exceed the ceiling, and returns nothing when it refuses.
//
// Precisely what this proves is worth stating, because it is easy to overclaim. maxTotalUnits sits
// an order of magnitude below math.MaxInt64, so a genuine int64 wrap is unreachable through this
// path — the guard stops the accumulation long before the representation could overflow. That is the
// design, and it is stronger than guarding at MaxInt64 would be.
//
// Mutation-checked: deleting the guard makes this test fail. The assertion that carries the weight
// is not "an error came back" but "no partially computed value came back with it".
func TestAddTotalRefusesToWrap(t *testing.T) {
	ceiling, err := collectible.ParseTotal("9999999999999999.99")
	if err != nil {
		t.Fatalf("the ceiling itself must parse: %v", err)
	}
	penny, err := collectible.ParseMoney("0.01")
	if err != nil {
		t.Fatalf("parse penny: %v", err)
	}

	sum, err := ceiling.AddTotal(penny)
	if !errors.Is(err, collectible.ErrTotalTooLarge) {
		t.Fatalf("ceiling + 0.01 = (%v, %v), want ErrTotalTooLarge", sum.String(), err)
	}
	if sum.MinorUnits() != 0 {
		t.Errorf("a refused addition returned %d minor units; it must return the zero value, never a "+
			"partially computed or wrapped figure", sum.MinorUnits())
	}

	// And the ceiling itself is reachable rather than being one short — otherwise the guard would be
	// rejecting a legitimate total and this test would still pass.
	nothing, err := collectible.ParseMoney("0.00")
	if err != nil {
		t.Fatalf("parse zero: %v", err)
	}
	if _, err := ceiling.AddTotal(nothing); err != nil {
		t.Errorf("ceiling + 0.00 = %v; the ceiling must be an attainable total, not one past the limit", err)
	}
}

// T012: exact decimal round-trips, including the cases that expose floating point.
//
// 0.10 + 0.20 is the canonical float failure. The expected total here is written as a literal rather
// than computed, so this test cannot agree with a bug by reproducing it.
func TestTotalsAreExactAcrossAwkwardValues(t *testing.T) {
	for _, tc := range []struct {
		name   string
		addend []string
		want   string
	}{
		{"the canonical float failure", []string{"0.10", "0.20"}, "0.30"},
		{"three tenths", []string{"0.10", "0.20", "0.30"}, "0.60"},
		{"repeated pennies", []string{"0.01", "0.01", "0.01", "0.01", "0.01", "0.01", "0.01"}, "0.07"},
		{"carries across the decimal point", []string{"0.99", "0.01"}, "1.00"},
		{"trailing zeros are preserved", []string{"10.50", "9.50"}, "20.00"},
		{"a recorded zero contributes nothing", []string{"0.00", "5.00"}, "5.00"},
		{"large but legitimate", []string{"9999999999.99", "9999999999.99"}, "19999999999.98"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var sum collectible.Money
			for _, raw := range tc.addend {
				m, err := collectible.ParseMoney(raw)
				if err != nil {
					t.Fatalf("parse %q: %v", raw, err)
				}
				sum, err = sum.AddTotal(m)
				if err != nil {
					t.Fatalf("add %q: %v", raw, err)
				}
			}
			if sum.String() != tc.want {
				t.Errorf("sum(%v) = %q, want %q", tc.addend, sum.String(), tc.want)
			}
		})
	}
}

// T012: a total read back from the database is bound by the representation, not the column.
func TestTotalFromMinorUnitsMatchesTheParseCeiling(t *testing.T) {
	const ceilingUnits int64 = 999999999999999999

	got, err := collectible.TotalFromMinorUnits(ceilingUnits)
	if err != nil {
		t.Fatalf("TotalFromMinorUnits(%d) = %v, want the ceiling to be representable", ceilingUnits, err)
	}
	if got.String() != "9999999999999999.99" {
		t.Errorf("TotalFromMinorUnits(%d).String() = %q", ceilingUnits, got.String())
	}

	if _, err := collectible.TotalFromMinorUnits(ceilingUnits + 1); !errors.Is(err, collectible.ErrTotalTooLarge) {
		t.Errorf("one minor unit past the ceiling = %v, want ErrTotalTooLarge", err)
	}
	if _, err := collectible.TotalFromMinorUnits(-1); err == nil {
		t.Error("a negative total was accepted; a wrapped accumulation arriving here must be refused")
	}
}
