package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// Statistics summarises one collector's whole vault.
//
// All three aggregates run inside a single read-only transaction. Three independent reads observe
// three independent snapshots, and a collectible added between the status query and the series query
// would be counted by one and not the other — a response whose figures are required to reconcile
// (FR-017, FR-026) then fails to, through no fault of the arithmetic. It is rare, it is silent, and
// it looks exactly like a bug in the aggregation. See 008's research.md Decision 10.
//
// The transaction takes no locks any writer waits on; these are reads of one collector's rows, held
// briefly.
func (s *Store) Statistics(ctx context.Context, collectorID uuid.UUID) (collectible.CollectionStatistics, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return collectible.CollectionStatistics{}, fmt.Errorf("begin statistics: %w", err)
	}
	// Read-only: there is nothing to commit, and rolling back is the cheapest way to release it.
	defer func() { _ = tx.Rollback(ctx) }()

	byStatus, err := statusSummaries(ctx, tx, collectorID)
	if err != nil {
		return collectible.CollectionStatistics{}, err
	}
	bySeries, err := breakdown(ctx, tx, collectorID, "series")
	if err != nil {
		return collectible.CollectionStatistics{}, err
	}
	byManufacturer, err := breakdown(ctx, tx, collectorID, "manufacturer")
	if err != nil {
		return collectible.CollectionStatistics{}, err
	}

	count, priced, _, err := collectible.Totals(byStatus)
	if err != nil {
		return collectible.CollectionStatistics{}, fmt.Errorf("total collection value: %w", err)
	}

	return collectible.CollectionStatistics{
		TotalCount:     count,
		PricedCount:    priced,
		ByStatus:       byStatus,
		BySeries:       bySeries,
		ByManufacturer: byManufacturer,
	}, nil
}

// statusSummaries is one GROUP BY returning at most four rows.
//
// count(purchase_price) counts non-null values, which is exactly pricedCount and costs nothing
// extra. sum(...)::text keeps the total an exact decimal string the whole way out — a numeric scanned
// into a float64 compiles, runs, passes every small test, and is wrong by a cent on a real vault.
func statusSummaries(ctx context.Context, tx pgx.Tx, collectorID uuid.UUID) ([]collectible.StatusSummary, error) {
	rows, err := tx.Query(ctx, `
		SELECT collection_status,
		       count(*)                              AS n,
		       count(purchase_price)                 AS priced,
		       coalesce(sum(purchase_price), 0)::text AS total
		FROM collectibles
		WHERE collector_id = $1
		GROUP BY collection_status`, collectorID)
	if err != nil {
		return nil, fmt.Errorf("summarise by status: %w", err)
	}
	defer rows.Close()

	var found []collectible.StatusSummary
	for rows.Next() {
		var (
			statusText string
			count      int
			priced     int
			totalText  string
		)
		if err := rows.Scan(&statusText, &count, &priced, &totalText); err != nil {
			return nil, fmt.Errorf("scan status summary: %w", err)
		}
		status, err := collectible.ParseCollectionStatus(statusText)
		if err != nil {
			// The column has a CHECK constraint, so this means the database holds a value the
			// domain does not know. Failing is right: silently dropping it would lose collectibles
			// from a total that claims to cover the vault.
			return nil, fmt.Errorf("unknown status in collectibles: %w", err)
		}
		total, err := collectible.ParseTotal(totalText)
		if err != nil {
			return nil, fmt.Errorf("status total %q: %w", totalText, err)
		}
		found = append(found, collectible.StatusSummary{
			Status: status, Count: count, PricedCount: priced, Total: total,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read status summaries: %w", err)
	}

	// A status holding nothing produced no row; FR-015 requires it to appear anyway.
	return collectible.FillMissingStatuses(found), nil
}

// breakdownColumns is the closed set of fields a breakdown may group on.
//
// The column name is interpolated into the statement because PostgreSQL cannot parameterise an
// identifier. This map is what makes that safe: the caller passes a key, never a column name, so no
// caller-supplied string ever reaches the SQL text.
var breakdownColumns = map[string]string{
	"series":       "series",
	"manufacturer": "manufacturer",
}

// breakdown groups the vault by one free-text field, bounded in rows as well as in statements.
//
// Three things are load-bearing, and each is a defect if changed:
//
//   - coalesce BEFORE normalising, so NULL, ” and '   ' form one group. The other way round leaves
//     two differently-named unspecified buckets, which a collector sees as an obvious defect.
//   - mode() WITHIN GROUP gives the collector's own most-used spelling back, rather than a lowercased
//     form nobody typed. Ties resolve alphabetically, which is repeatable but is not a claim about
//     preference — a tie does not record one.
//   - the outer GROUP BY folds everything past the bound into a single remainder row, so the result
//     is at most bound + 2 rows however many distinct series a vault holds. Ranking here and folding
//     in Go would transfer eight hundred rows to display eight.
//   - the PARTITION BY on the window takes the unspecified group out of the competition for those
//     slots, by giving it a partition of its own in which it is always first.
//     Without it the unspecified group is ranked among the named ones, and both ways round are
//     wrong: a large unspecified group wins a slot a real series should have had, and a small one
//     ranks past the bound and is folded into the remainder, so it disappears from a breakdown that
//     is required to report it. Nine series of two with one unnamed piece reproduces the second case
//     against PostgreSQL — the unspecified group vanished and the remainder silently grew by one.
//
// The unspecified group comes back with key ” and the remainder with key NULL. They do not collide,
// though psql renders both as blank, which makes it look as though they do.
func breakdown(
	ctx context.Context, tx pgx.Tx, collectorID uuid.UUID, field string,
) ([]collectible.GroupSummary, error) {
	column, ok := breakdownColumns[field]
	if !ok {
		return nil, fmt.Errorf("breakdown on unsupported field %q", field)
	}

	rows, err := tx.Query(ctx, fmt.Sprintf(`
		WITH grouped AS (
		    SELECT lower(btrim(coalesce(%[1]s, '')))        AS key,
		           mode() WITHIN GROUP (ORDER BY btrim(%[1]s)) AS display,
		           count(*)                                 AS n,
		           count(purchase_price)                    AS priced,
		           coalesce(sum(purchase_price), 0)         AS total
		    FROM collectibles
		    WHERE collector_id = $1
		    GROUP BY 1
		),
		ranked AS (
		    SELECT *, row_number() OVER (PARTITION BY (key = '') ORDER BY n DESC, key ASC) AS rn
		    FROM grouped
		)
		SELECT CASE WHEN rn <= $2 THEN key     ELSE NULL END AS key,
		       CASE WHEN rn <= $2 THEN display ELSE NULL END AS display,
		       sum(n)        AS n,
		       sum(priced)   AS priced,
		       sum(total)::text AS total
		FROM ranked
		GROUP BY 1, 2`, column), collectorID, collectible.MaxBreakdownGroups)
	if err != nil {
		return nil, fmt.Errorf("breakdown by %s: %w", field, err)
	}
	defer rows.Close()

	var out []collectible.GroupSummary
	for rows.Next() {
		var (
			key       *string
			display   *string
			count     int
			priced    int
			totalText string
		)
		if err := rows.Scan(&key, &display, &count, &priced, &totalText); err != nil {
			return nil, fmt.Errorf("scan %s breakdown: %w", field, err)
		}
		total, err := collectible.ParseTotal(totalText)
		if err != nil {
			return nil, fmt.Errorf("%s group total %q: %w", field, totalText, err)
		}

		group := collectible.GroupSummary{Count: count, PricedCount: priced, Total: total}
		switch {
		case key == nil:
			group.Kind = collectible.GroupRemainder
		case *key == "":
			group.Kind = collectible.GroupUnspecified
		default:
			group.Kind = collectible.GroupNamed
			if display != nil {
				group.Label = *display
			}
		}
		out = append(out, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read %s breakdown: %w", field, err)
	}

	// Labels for the two special groups are the presentation layer's business; the domain carries
	// the kind, and ordering is applied here so every caller gets the same order (FR-022).
	collectible.SortGroups(out)
	return out, nil
}
