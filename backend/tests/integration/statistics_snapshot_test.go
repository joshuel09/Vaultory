//go:build integration

package integration

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/joshuel09/vaultory/backend/internal/domain/collectible"
)

// statisticsInterleave commits a write on another connection after the status
// aggregate is consumed and before the first breakdown starts. No timing sleeps.
type statisticsInterleave struct {
	queryCountTracer
	once  sync.Once
	write func()
}

func (s *statisticsInterleave) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	ctx = s.queryCountTracer.TraceQueryStart(ctx, conn, data)
	if strings.Contains(data.SQL, "WITH grouped AS") {
		s.once.Do(s.write)
	}
	return ctx
}

// FR-017a: a committed change between aggregates must not split one response
// across two moments. A subsequent request must still see the committed change.
func TestStatisticsKeepOneSnapshotDuringConcurrentWrite(t *testing.T) {
	_, writer := freshStore(t)
	ctx := context.Background()
	if _, violations, err := writer.Add(ctx, collectorA, priced("snapshot-before", "Before", "owned", "10.00", "Before", "Before")); err != nil || len(violations) != 0 {
		t.Fatalf("seed before: %v %+v", err, violations)
	}
	wrote := false
	cfg := pool.Config()
	cfg.ConnConfig.Tracer = &statisticsInterleave{write: func() {
		if _, violations, err := writer.Add(ctx, collectorA, priced("snapshot-after", "After", "preordered", "20.00", "After", "After")); err != nil || len(violations) != 0 {
			t.Fatalf("commit concurrent write: %v %+v", err, violations)
		}
		wrote = true
	}}
	readerPool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(readerPool.Close)
	_, reader := serviceForPool(t, readerPool)
	var stats collectible.CollectionStatistics
	queries := countQueries(ctx, func(ctx context.Context) {
		var err error
		stats, err = reader.Statistics(ctx, collectorA)
		if err != nil {
			t.Fatalf("statistics: %v", err)
		}
	})
	if !wrote {
		t.Fatal("concurrent write never ran between aggregates")
	}
	if queries != 3 {
		t.Errorf("statistics used %d data queries, want 3", queries)
	}
	if stats.TotalCount != 1 || stats.PricedCount != 1 {
		t.Errorf("in-flight response counts = %d/%d, want 1/1", stats.TotalCount, stats.PricedCount)
	}
	if got := statusOf(t, stats, collectible.StatusOwned).Total.String(); got != "10.00" {
		t.Errorf("in-flight owned total = %s, want 10.00", got)
	}
	for dimension, groups := range map[string][]collectible.GroupSummary{"series": stats.BySeries, "manufacturer": stats.ByManufacturer} {
		if len(groups) != 1 || groups[0].Label != "Before" || groups[0].Count != 1 || groups[0].Total.String() != "10.00" {
			t.Errorf("%s observed the later commit: %+v; want only Before, count 1, total 10.00", dimension, groups)
		}
	}
	latest, err := writer.Statistics(ctx, collectorA)
	if err != nil {
		t.Fatal(err)
	}
	if latest.TotalCount != 2 || latest.PricedCount != 2 {
		t.Errorf("next request counts = %d/%d, want 2/2", latest.TotalCount, latest.PricedCount)
	}
	if got := statusOf(t, latest, collectible.StatusPreordered).Total.String(); got != "20.00" {
		t.Errorf("next request preorder total = %s, want 20.00", got)
	}
}
