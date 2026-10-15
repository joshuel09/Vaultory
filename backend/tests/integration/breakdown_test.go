//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
	"github.com/joshuel09/vaultory/backend/internal/store/postgres"
)

// seriesOf writes a series value straight into the row, bypassing validation.
//
// The domain trims optional text and records a blank value as absent, so an empty or whitespace-only
// series cannot be created through the service — which is the point. These rows are what a migration,
// a direct correction, or an earlier version of the product can leave behind, and the grouping has to
// survive them rather than assume they are impossible.
func seriesOf(t *testing.T, name, value string) {
	t.Helper()
	tag, err := pool.Exec(context.Background(),
		`UPDATE collectibles SET series = $2 WHERE name = $1`, name, value)
	if err != nil {
		t.Fatalf("set series on %q: %v", name, err)
	}
	if tag.RowsAffected() != 1 {
		t.Fatalf("set series on %q changed %d rows, want 1", name, tag.RowsAffected())
	}
}

// filterFor turns a breakdown group back into the gallery filter its link carries.
//
// A named group filters on the label the collector sees. The unspecified group filters on the empty
// string — not on "no filter", which would return everything, and not on SQL NULL, which would miss
// the blank and whitespace rows that belong to it.
func filterFor(t *testing.T, g collectible.GroupSummary) postgres.ListFilter {
	t.Helper()
	switch g.Kind {
	case collectible.GroupNamed:
		label := g.Label
		return postgres.ListFilter{Series: &label}
	case collectible.GroupUnspecified:
		blank := ""
		return postgres.ListFilter{Series: &blank}
	default:
		t.Fatalf("the remainder group has no filter that reproduces it")
		return postgres.ListFilter{}
	}
}

// T053 — FR-027, FR-027a: spellings that differ only in case or surrounding space are one group.
//
// The label is the collector's own most-used spelling, not a lowercased form nobody typed.
func TestNearDuplicateSpellingsFormOneGroup(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	// Three "Nendoroid" against one each of two other spellings: same group, and the majority wins
	// the label. The majority has to be clear — an earlier version of this fixture made it two
	// against two, and then the label is decided by the tiebreak rather than by use.
	spellings := []string{"Nendoroid", " Nendoroid", "Nendoroid  ", "nendoroid", "NENDOROID"}
	for i, s := range spellings {
		d := priced(fmt.Sprintf("dup-%d", i), fmt.Sprintf("Piece %d", i), "owned", "1.00", s, "")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	named := 0
	for _, g := range stats.BySeries {
		if g.Kind == collectible.GroupNamed {
			named++
		}
	}
	if named != 1 {
		t.Fatalf("five spellings of one series produced %d named groups, want 1: %+v", named, stats.BySeries)
	}

	g := stats.BySeries[0]
	if g.Count != 5 || g.Total.String() != "5.00" {
		t.Errorf("the group holds count=%d total=%s, want 5 and 5.00", g.Count, g.Total)
	}
	if g.Label != "Nendoroid" {
		t.Errorf("the group is labelled %q, want %q — the spelling the collector used most",
			g.Label, "Nendoroid")
	}
}

// T054 — FR-021, FR-024: null, empty and whitespace-only form one unspecified group, not three.
//
// Two differently-named unspecified buckets is a defect a collector sees immediately. The grouping
// coalesces before it normalises for exactly this reason; the other order produces a NULL key and an
// empty-string key that never meet.
func TestNullEmptyAndWhitespaceFormOneUnspecifiedGroup(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for i, name := range []string{"Null series", "Empty series", "Spaces series"} {
		d := priced(fmt.Sprintf("uns-%d", i), name, "owned", "2.00", "", "")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %s: %v %+v", name, err, v)
		}
	}
	// "Null series" keeps its NULL. The other two are rewritten underneath the domain.
	seriesOf(t, "Empty series", "")
	seriesOf(t, "Spaces series", "   ")

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	unspecified := 0
	for _, g := range stats.BySeries {
		if g.Kind == collectible.GroupUnspecified {
			unspecified++
			if g.Count != 3 || g.Total.String() != "6.00" {
				t.Errorf("the unspecified group holds count=%d total=%s, want 3 and 6.00 — all "+
					"three rows belong to it", g.Count, g.Total)
			}
		}
	}
	if unspecified != 1 {
		t.Errorf("got %d unspecified groups, want exactly 1: %+v", unspecified, stats.BySeries)
	}
	if len(stats.BySeries) != 1 {
		t.Errorf("got %d series groups, want 1: %+v", len(stats.BySeries), stats.BySeries)
	}
}

// T055 — FR-026, SC-003: every collectible is in exactly one group, remainder and unspecified
// included.
//
// Deliberately shaped to produce all three kinds at once: more named series than the bound, plus
// pieces recording none. A breakdown that drops the tail, or that loses the unspecified group when
// it ranks small, fails here.
func TestBreakdownCountsReconcileIncludingUnspecifiedAndRemainder(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	// Nine named series of two, so one of them must fall past the bound of eight.
	for i := 0; i < 9; i++ {
		for j := 0; j < 2; j++ {
			d := priced(fmt.Sprintf("rec-%d-%d", i, j), fmt.Sprintf("Piece %d %d", i, j),
				"owned", "1.00", fmt.Sprintf("Series %02d", i), "")
			if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
				t.Fatalf("add: %v %+v", err, v)
			}
		}
	}
	// And one piece recording no series, which is the smallest group of all.
	if _, v, err := svc.Add(ctx, collectorA, priced("rec-none", "Unnamed", "owned", "1.00", "", "")); err != nil || len(v) > 0 {
		t.Fatalf("add unnamed: %v %+v", err, v)
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}
	if stats.TotalCount != 19 {
		t.Fatalf("the vault holds %d pieces, want 19", stats.TotalCount)
	}

	kinds := map[collectible.GroupKind]int{}
	total := 0
	for _, g := range stats.BySeries {
		kinds[g.Kind]++
		total += g.Count
	}
	if total != stats.TotalCount {
		t.Errorf("the series groups account for %d of %d collectibles", total, stats.TotalCount)
	}
	// The failure this is really watching for: the unspecified group ranking tenth by size and
	// being folded into the remainder, which makes it disappear while the arithmetic still adds up.
	if kinds[collectible.GroupUnspecified] != 1 {
		t.Errorf("got %d unspecified groups, want 1 — one piece records no series, and a group "+
			"that is small must not be swallowed by the remainder: %+v",
			kinds[collectible.GroupUnspecified], stats.BySeries)
	}
	if kinds[collectible.GroupRemainder] != 1 {
		t.Errorf("got %d remainder groups, want 1 — nine named series exceed the bound of %d",
			kinds[collectible.GroupRemainder], collectible.MaxBreakdownGroups)
	}
	if kinds[collectible.GroupNamed] != collectible.MaxBreakdownGroups {
		t.Errorf("got %d named groups, want %d", kinds[collectible.GroupNamed], collectible.MaxBreakdownGroups)
	}
}

// T056 — SC-010: a group's count equals the number of collectibles its own link returns.
//
// This is the test no unit test can stand in for, and the one that catches the whole class of
// normalisation mismatches: the breakdown groups on one rule and the gallery filters on another, so
// a group says twelve and the gallery it links to shows eleven. Nothing about either figure looks
// wrong on its own.
//
// The unspecified group is included on purpose. Its link carries an empty value, and every obvious
// shortcut on either side of the wire — a truthiness check in the client, q.Get in the handler, an
// IS NULL predicate in the store — turns that into "no filter" or "the null rows only", and makes
// this the one group that disagrees.
func TestEachGroupsCountMatchesItsOwnLink(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	fixtures := []struct{ name, series string }{
		{"A one", "Nendoroid"},
		{"A two", "nendoroid"},
		{"A three", "NENDOROID"},
		{"B one", "Figma"},
		{"B two", " Figma "},
		{"C one", "Pop Up Parade"},
		{"D none", ""},
		{"E blank", ""},
	}
	for i, f := range fixtures {
		d := priced(fmt.Sprintf("lnk-%d", i), f.name, "owned", "3.00", f.series, "")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %s: %v %+v", f.name, err, v)
		}
	}
	// One of the two unspecified rows holds whitespace rather than NULL, so the filter has to match
	// both to agree with the group.
	seriesOf(t, "E blank", "  ")

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}
	if len(stats.BySeries) == 0 {
		t.Fatal("no series groups to check")
	}

	checked := 0
	sawUnspecified := false
	for _, g := range stats.BySeries {
		if g.Kind == collectible.GroupRemainder {
			continue // nothing claims this one is reachable by a filter
		}
		if g.Kind == collectible.GroupUnspecified {
			sawUnspecified = true
		}
		page, err := svc.List(ctx, collectorA, filterFor(t, g), "", 50)
		if err != nil {
			t.Fatalf("list for group %q: %v", g.Label, err)
		}
		if len(page.Rows) != g.Count {
			t.Errorf("the %s group (kind %s) reports %d collectibles but its link returns %d",
				g.Label, g.Kind, g.Count, len(page.Rows))
		}
		checked++
	}
	if !sawUnspecified {
		t.Error("the fixture produced no unspecified group, so the case this test exists for was not exercised")
	}
	if checked < 4 {
		t.Errorf("only %d groups were checked; the fixture should produce at least four", checked)
	}
}

// T057 — FR-025: the number of entries is bounded, whatever the vault holds.
//
// Bounded in rows transferred, not merely in rows displayed. Ranking in Go instead would carry every
// distinct series out of the database to show eight of them.
func TestABreakdownIsBoundedHoweverManySeriesExist(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for i := 0; i < 40; i++ {
		d := priced(fmt.Sprintf("bnd-%d", i), fmt.Sprintf("Piece %d", i), "owned", "1.00",
			fmt.Sprintf("Series %02d", i), fmt.Sprintf("Maker %02d", i))
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}

	stats, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	// At most the bound, one unspecified group, and one remainder.
	const max = collectible.MaxBreakdownGroups + 2
	if len(stats.BySeries) > max {
		t.Errorf("40 distinct series produced %d groups, want at most %d", len(stats.BySeries), max)
	}
	if len(stats.ByManufacturer) > max {
		t.Errorf("40 distinct manufacturers produced %d groups, want at most %d", len(stats.ByManufacturer), max)
	}

	var n int
	for _, g := range stats.BySeries {
		n += g.Count
	}
	if n != 40 {
		t.Errorf("the bounded groups account for %d of 40 collectibles; the bound must not lose any", n)
	}
}

// T058 — FR-022, FR-023: the order is repeatable, and "everything else" never outranks a real group.
//
// The second half is what running the query against real data corrected during planning. Ordering by
// count alone puts the remainder above a named group the moment the tail is larger than the smallest
// named one, which reads as though the collection is mostly nothing in particular.
func TestBreakdownOrderingIsStableAndPutsSpecialGroupsLast(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	// Nine named series of one each, so the remainder holds one — then twelve pieces recording no
	// series at all, so the unspecified group is far and away the largest thing in the breakdown.
	for i := 0; i < 9; i++ {
		d := priced(fmt.Sprintf("ord-%d", i), fmt.Sprintf("Named %d", i), "owned", "1.00",
			fmt.Sprintf("Series %02d", i), "")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}
	for i := 0; i < 12; i++ {
		d := priced(fmt.Sprintf("ord-none-%d", i), fmt.Sprintf("Unnamed %d", i), "owned", "1.00", "", "")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add unnamed %d: %v %+v", i, err, v)
		}
	}

	first, err := svc.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatalf("statistics: %v", err)
	}

	// Every named group comes before both special groups, however small the named ones are.
	seenSpecial := false
	for _, g := range first.BySeries {
		if g.Kind == collectible.GroupNamed {
			if seenSpecial {
				t.Errorf("a named group (%q, %d pieces) appears after a special group; named "+
					"groups come first whatever their size (FR-022)", g.Label, g.Count)
			}
			continue
		}
		seenSpecial = true
	}
	// And the unspecified group, at twelve against named groups of one, is still below them.
	if n := len(first.BySeries); n < 2 {
		t.Fatalf("expected named and special groups, got %d", n)
	}
	last := first.BySeries[len(first.BySeries)-1]
	if last.Kind != collectible.GroupRemainder {
		t.Errorf("the last group is %s, want the remainder", last.Kind)
	}
	if unspec, ok := groupOfKind(first.BySeries, collectible.GroupUnspecified); !ok || unspec.Count != 12 {
		t.Errorf("the unspecified group holds %d (present: %v), want 12", unspec.Count, ok)
	}

	// Repeatable: ties between the nine single-piece series must not reshuffle between loads.
	shape := func(groups []collectible.GroupSummary) string {
		var b []byte
		for _, g := range groups {
			b = append(b, fmt.Sprintf("%s/%s/%d;", g.Kind, g.Label, g.Count)...)
		}
		return string(b)
	}
	want := shape(first.BySeries)
	for i := 0; i < 4; i++ {
		again, err := svc.Statistics(ctx, collectorA)
		if err != nil {
			t.Fatalf("statistics %d: %v", i, err)
		}
		if got := shape(again.BySeries); got != want {
			t.Fatalf("the breakdown reshuffled between identical calls (FR-023)\n  first: %s\n  then:  %s", want, got)
		}
	}
}

// T048 — FR-027a: a tie in the display spelling resolves the same way every time.
//
// Asserted as *repeatable*, not as correct. Two spellings used equally often record no preference,
// so there is no right answer to assert — only the requirement that the view does not alternate
// between them from one load to the next.
func TestATiedDisplaySpellingIsRepeatable(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	for i, spelling := range []string{"Figma", "Figma", "FIGMA", "FIGMA"} {
		d := priced(fmt.Sprintf("tie-%d", i), fmt.Sprintf("Piece %d", i), "owned", "1.00", spelling, "")
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}

	var label string
	for i := 0; i < 5; i++ {
		stats, err := svc.Statistics(ctx, collectorA)
		if err != nil {
			t.Fatalf("statistics %d: %v", i, err)
		}
		if len(stats.BySeries) != 1 {
			t.Fatalf("got %d series groups, want 1: %+v", len(stats.BySeries), stats.BySeries)
		}
		got := stats.BySeries[0].Label
		if i == 0 {
			label = got
			continue
		}
		if got != label {
			t.Fatalf("a tied spelling came back as %q and then %q; whichever it picks, it must "+
				"pick the same one every time (FR-027a)", label, got)
		}
	}
	// Whatever it chose, it is one of the two the collector actually typed — never a lowercased
	// form nobody used.
	if label != "Figma" && label != "FIGMA" {
		t.Errorf("the group is labelled %q, which is a spelling nobody entered", label)
	}
}
