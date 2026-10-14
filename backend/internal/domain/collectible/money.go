package collectible

import (
	"errors"
	"fmt"
	"strings"
)

// maxMoneyUnits bounds a single stored amount to what numeric(12,2) can hold: ten integer digits.
const maxMoneyUnits int64 = 9_999_999_999_99

// maxTotalUnits bounds a derived total — sixteen integer digits.
//
// A total is not a stored amount and has no reason to inherit a column's limit. What it must respect
// is the limit of the representation: int64 minor units. This constant sits an order of magnitude
// below math.MaxInt64 so that accumulation has headroom, and it is seven orders of magnitude above
// maxMoneyUnits, which is the point — a vault would need millions of maximum-value collectibles to
// reach it. See specs/008-collection-value-stats/research.md Decision 3.
const maxTotalUnits int64 = 9_999_999_999_999_999_99

// ErrTotalTooLarge is returned when a total exceeds what can be represented exactly.
//
// It is named, rather than being one more fmt.Errorf, because the alternative failure is silent: an
// unguarded accumulation wraps to a negative total, and a negative total is not obviously a bug to
// anyone reading it. Nothing realistic provokes this, which is exactly why it needs a name a test
// can assert on.
var ErrTotalTooLarge = errors.New("total is too large to represent exactly")

// Money is an exact monetary amount, held as a whole number of minor units (cents).
//
// It is deliberately not a float. Principle IV forbids floating-point arithmetic for financial
// values, and this type makes that impossible rather than merely discouraged: there is no
// conversion to or from float64 anywhere in its API. Amounts cross the API as decimal strings for
// the same reason — a JSON number would invite a float on the other side.
type Money struct {
	units int64 // minor units, e.g. 124999 for 1249.99
}

// ParseMoney reads an exact decimal amount with at most two fractional digits.
//
// It rejects rather than rounds. A collector who types three decimal places has made a mistake
// worth telling them about; silently turning 10.005 into 10.01 changes what they paid.
func ParseMoney(raw string) (Money, error) {
	return parseWithin(raw, maxMoneyUnits)
}

// ParseTotal reads a derived total, bound by the representation rather than by the storage column.
//
// Identical to ParseMoney in every respect but the ceiling. Summing purchase prices across a vault
// can legitimately exceed what any one of them could have been, so applying the column's limit here
// would reject a correct answer (FR-013).
func ParseTotal(raw string) (Money, error) {
	return parseWithin(raw, maxTotalUnits)
}

func parseWithin(raw string, max int64) (Money, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return Money{}, fmt.Errorf("amount is empty")
	}
	if strings.HasPrefix(s, "-") {
		return Money{}, fmt.Errorf("amount is negative")
	}
	// No currency symbols, thousands separators, or signs: the contract specifies a plain decimal
	// string, and guessing at a collector's formatting risks changing the amount.
	if strings.ContainsAny(s, "+,_ $€£¥") {
		return Money{}, fmt.Errorf("amount must be a plain decimal, without symbols or separators")
	}

	intPart, fracPart, hasFrac := strings.Cut(s, ".")
	if hasFrac {
		if len(fracPart) == 0 || len(fracPart) > 2 {
			return Money{}, fmt.Errorf("amount must have one or two decimal places")
		}
	}
	if intPart == "" {
		return Money{}, fmt.Errorf("amount must have at least one digit before the decimal point")
	}
	if len(intPart) > 1 && intPart[0] == '0' {
		return Money{}, fmt.Errorf("amount must not have leading zeros")
	}

	units, err := digitsToInt(intPart, max)
	if err != nil {
		return Money{}, err
	}
	if units > max/100 {
		return Money{}, tooLarge(max)
	}
	units *= 100

	if hasFrac {
		padded := fracPart
		if len(padded) == 1 {
			padded += "0"
		}
		cents, err := digitsToInt(padded, max)
		if err != nil {
			return Money{}, err
		}
		units += cents
	}
	if units > max {
		return Money{}, tooLarge(max)
	}
	return Money{units: units}, nil
}

func digitsToInt(s string, max int64) (int64, error) {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("amount contains a non-digit character")
		}
		// Guard before multiplying, so a long input cannot wrap.
		if n > (max-int64(r-'0'))/10 {
			return 0, tooLarge(max)
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
}

// tooLarge keeps the existing wording for a single amount — collectors see that one — while giving
// a total the named error a caller can match on.
func tooLarge(max int64) error {
	if max == maxTotalUnits {
		return ErrTotalTooLarge
	}
	return fmt.Errorf("amount is too large")
}

// String renders the amount with exactly two decimal places, which is what the API carries.
// Zero renders as "0.00": a recorded amount of nothing, distinct from no amount recorded at all.
func (m Money) String() string {
	return fmt.Sprintf("%d.%02d", m.units/100, m.units%100)
}

// MinorUnits exposes the amount for storage. The store scales it back to a decimal for
// numeric(12,2); nothing converts through a float on the way.
func (m Money) MinorUnits() int64 { return m.units }

// MoneyFromMinorUnits rebuilds an amount read back from storage.
func MoneyFromMinorUnits(units int64) (Money, error) {
	if units < 0 {
		return Money{}, fmt.Errorf("amount is negative")
	}
	if units > maxMoneyUnits {
		return Money{}, fmt.Errorf("amount is too large")
	}
	return Money{units: units}, nil
}

// IsZero distinguishes a recorded zero from an absent amount. Absence is represented by a nil
// *Money on the collectible, never by a zero value (FR-007).
func (m Money) IsZero() bool { return m.units == 0 }

// AddTotal sums two figures, refusing to wrap.
//
// This is where an overflow would actually occur — parsing checks one value, but a total is built by
// repeated addition, and int64 addition wraps silently to a negative number. Every accumulation of
// money in this codebase goes through here so that the guard cannot be forgotten at one call site.
//
// The result is bound by maxTotalUnits rather than maxMoneyUnits: adding two legitimate amounts can
// exceed what either of them could have been, which is the entire point of a total.
func (m Money) AddTotal(other Money) (Money, error) {
	// Checked before the addition rather than after. Testing the sum for negativity afterwards
	// would rely on the very wrap this is meant to prevent.
	if m.units > maxTotalUnits-other.units {
		return Money{}, ErrTotalTooLarge
	}
	return Money{units: m.units + other.units}, nil
}

// TotalFromMinorUnits rebuilds a total read back from the database, bound by the representation
// rather than by the storage column (FR-013).
func TotalFromMinorUnits(units int64) (Money, error) {
	if units < 0 {
		return Money{}, fmt.Errorf("total is negative")
	}
	if units > maxTotalUnits {
		return Money{}, ErrTotalTooLarge
	}
	return Money{units: units}, nil
}
