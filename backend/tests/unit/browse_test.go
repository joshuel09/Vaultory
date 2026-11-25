package unit

import (
	"strings"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// Feature 011: what a search and a sort order mean.

func TestParseSearchSplitsOnWhitespace(t *testing.T) {
	words, v := collectible.ParseSearch("  kaiju \t prime\n1/4  ")
	if v != nil {
		t.Fatalf("violation: %+v", v)
	}
	if strings.Join(words, "|") != "kaiju|prime|1/4" {
		t.Errorf("words = %q", words)
	}
}

func TestParseSearchEmptyIsNoSearch(t *testing.T) {
	for _, raw := range []string{"", "   ", "\t\n"} {
		words, v := collectible.ParseSearch(raw)
		if v != nil || words != nil {
			t.Errorf("ParseSearch(%q) = %q, %+v; want no search", raw, words, v)
		}
	}
}

func TestParseSearchKeepsWildcardsForTheStoreToEscape(t *testing.T) {
	// The domain does not know how matching works, so it must not alter these.
	words, _ := collectible.ParseSearch(`100% a_b c\d`)
	if strings.Join(words, "|") != `100%|a_b|c\d` {
		t.Errorf("words = %q", words)
	}
}

func TestParseSearchLimits(t *testing.T) {
	if _, v := collectible.ParseSearch(strings.Repeat("a", 200)); v != nil {
		t.Errorf("200 characters refused: %+v", v)
	}
	if _, v := collectible.ParseSearch(strings.Repeat("a", 201)); v == nil || v.Field != "q" {
		t.Errorf("201 characters accepted: %+v", v)
	}
	// Counted in characters, not bytes: 200 Japanese characters are one search, not 600 bytes of one.
	if _, v := collectible.ParseSearch(strings.Repeat("フ", 200)); v != nil {
		t.Errorf("200 multibyte characters refused: %+v", v)
	}
	if _, v := collectible.ParseSearch(strings.Repeat("a ", 21)); v == nil {
		t.Error("21 words accepted")
	}
}

func TestParseSortOrder(t *testing.T) {
	if o, v := collectible.ParseSortOrder(""); v != nil || o != collectible.DefaultSortOrder {
		t.Errorf("empty = %q, %+v; want the default", o, v)
	}
	for _, o := range collectible.AllSortOrders() {
		if got, v := collectible.ParseSortOrder(string(o)); v != nil || got != o {
			t.Errorf("ParseSortOrder(%q) = %q, %+v", o, got, v)
		}
	}
	if len(collectible.AllSortOrders()) != 10 {
		t.Errorf("%d orders, want the ten the spec names", len(collectible.AllSortOrders()))
	}
	if _, v := collectible.ParseSortOrder("price"); v == nil || v.Field != "sort" {
		t.Errorf("an unknown order was accepted: %+v", v)
	}
}
