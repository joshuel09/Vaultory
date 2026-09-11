package unit

import (
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// FR-016: exact amounts, no rounding or precision loss. FR-017: zero accepted, negative rejected.
func TestParseMoneyAccepted(t *testing.T) {
	cases := map[string]string{
		"0":             "0.00",
		"0.00":          "0.00",
		"0.0":           "0.00",
		"0.5":           "0.50",
		"0.05":          "0.05",
		"1250":          "1250.00",
		"1250.00":       "1250.00",
		"249.99":        "249.99",
		"9999999999.99": "9999999999.99", // the largest numeric(12,2) permits
	}
	for in, want := range cases {
		m, err := collectible.ParseMoney(in)
		if err != nil {
			t.Fatalf("ParseMoney(%q) returned error: %v", in, err)
		}
		if got := m.String(); got != want {
			t.Errorf("ParseMoney(%q).String() = %q, want %q", in, got, want)
		}
	}
}

func TestParseMoneyRejected(t *testing.T) {
	// Each of these is an edge case named in spec.md: negatives, excess precision, separators,
	// symbols, and amounts beyond the representable range.
	rejected := []string{
		"-1", "-0.01", "-0.00",
		"10.005", "1.234", "0.001",
		"1,250.00", "1 250", "$249.99", "249.99 USD", "€10",
		"", "   ", "abc", "1.2.3", ".", ".50", "1.", "+1.00",
		"10000000000.00", "99999999999999999999",
		"01.00", "007",
	}
	for _, in := range rejected {
		if m, err := collectible.ParseMoney(in); err == nil {
			t.Errorf("ParseMoney(%q) was accepted as %q; it must be rejected, never silently rounded", in, m)
		}
	}
}

// FR-007: a recorded zero is not the same as no amount recorded. Absence is a nil *Money on the
// collectible; this only checks that zero is representable and distinguishable.
func TestZeroIsARecordedAmount(t *testing.T) {
	m, err := collectible.ParseMoney("0.00")
	if err != nil {
		t.Fatalf("zero should be accepted (a gift or giveaway win): %v", err)
	}
	if !m.IsZero() {
		t.Error("0.00 should report IsZero")
	}
	if m.String() != "0.00" {
		t.Errorf("zero rendered as %q, want \"0.00\"", m.String())
	}
}

// Round-tripping through storage must not change the amount (FR-016, SC-009).
func TestMoneyRoundTripsThroughMinorUnits(t *testing.T) {
	for _, in := range []string{"0.00", "0.01", "0.99", "1.00", "249.99", "1250.00", "9999999999.99"} {
		m, err := collectible.ParseMoney(in)
		if err != nil {
			t.Fatalf("ParseMoney(%q): %v", in, err)
		}
		back, err := collectible.MoneyFromMinorUnits(m.MinorUnits())
		if err != nil {
			t.Fatalf("MoneyFromMinorUnits(%d): %v", m.MinorUnits(), err)
		}
		if back.String() != m.String() {
			t.Errorf("%q round-tripped to %q", m, back)
		}
	}
}

func TestMoneyFromMinorUnitsRejectsOutOfRange(t *testing.T) {
	if _, err := collectible.MoneyFromMinorUnits(-1); err == nil {
		t.Error("negative minor units must be rejected")
	}
	if _, err := collectible.MoneyFromMinorUnits(1_000_000_000_000); err == nil {
		t.Error("minor units beyond numeric(12,2) must be rejected")
	}
}
