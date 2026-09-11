package collectible

import (
	"fmt"
	"strings"
)

// maxMoneyUnits bounds an amount to what numeric(12,2) can hold: ten integer digits.
const maxMoneyUnits int64 = 9_999_999_999_99

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

	units, err := digitsToInt(intPart)
	if err != nil {
		return Money{}, err
	}
	if units > maxMoneyUnits/100 {
		return Money{}, fmt.Errorf("amount is too large")
	}
	units *= 100

	if hasFrac {
		padded := fracPart
		if len(padded) == 1 {
			padded += "0"
		}
		cents, err := digitsToInt(padded)
		if err != nil {
			return Money{}, err
		}
		units += cents
	}
	if units > maxMoneyUnits {
		return Money{}, fmt.Errorf("amount is too large")
	}
	return Money{units: units}, nil
}

func digitsToInt(s string) (int64, error) {
	var n int64
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("amount contains a non-digit character")
		}
		// Guard before multiplying, so a long input cannot wrap.
		if n > (maxMoneyUnits-int64(r-'0'))/10 {
			return 0, fmt.Errorf("amount is too large")
		}
		n = n*10 + int64(r-'0')
	}
	return n, nil
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
