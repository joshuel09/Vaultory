//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
)

// T016: the schema the migrations produce is the schema data-model.md describes.
//
// Running `migrate up` and seeing no error proves the files parse. These assertions prove the
// constraints that carry the guarantees actually exist — a migration that silently dropped the
// composite key would still apply cleanly.
func TestSchemaCarriesItsConstraints(t *testing.T) {
	ctx := context.Background()

	t.Run("composite foreign key on the image reference", func(t *testing.T) {
		var count int
		err := pool.QueryRow(ctx, `
			SELECT count(*)
			FROM information_schema.key_column_usage
			WHERE constraint_name = 'collectibles_image_same_owner'`).Scan(&count)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		// Two columns: (image_id, collector_id). One column would mean the guarantee is gone.
		if count != 2 {
			t.Errorf("the image foreign key spans %d columns, want 2 — a single-column key would "+
				"let a collectible reference another collector's image (FR-015)", count)
		}
	})

	t.Run("submission key is unique per collector", func(t *testing.T) {
		var cols []string
		rows, err := pool.Query(ctx, `
			SELECT column_name FROM information_schema.key_column_usage
			WHERE table_name = 'collectible_submissions' AND constraint_name LIKE '%pkey%'
			ORDER BY ordinal_position`)
		if err != nil {
			t.Fatalf("query: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			var c string
			if err := rows.Scan(&c); err != nil {
				t.Fatalf("scan: %v", err)
			}
			cols = append(cols, c)
		}
		joined := strings.Join(cols, ",")
		if !strings.Contains(joined, "collector_id") || !strings.Contains(joined, "submission_key") {
			t.Errorf("submissions primary key is (%s), want (collector_id, submission_key) — this "+
				"constraint is what holds under a concurrent double submit (FR-047)", joined)
		}
	})

	t.Run("status is closed to four values", func(t *testing.T) {
		for _, bad := range []string{"borrowed", "OWNED", ""} {
			_, err := pool.Exec(ctx, `
				INSERT INTO collectibles (collector_id, name, collection_status)
				VALUES ($1, 'Constraint probe', $2)`, collectorA, bad)
			if err == nil {
				t.Errorf("the database accepted status %q; FR-004 permits exactly four", bad)
				_, _ = pool.Exec(ctx, `DELETE FROM collectibles WHERE name = 'Constraint probe'`)
			}
		}
	})

	t.Run("whitespace-only name is refused at the database too", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO collectibles (collector_id, name, collection_status)
			VALUES ($1, '   ', 'owned')`, collectorA)
		if err == nil {
			t.Error("the database accepted a whitespace-only name; the CHECK is on the trimmed length (FR-003)")
			_, _ = pool.Exec(ctx, `DELETE FROM collectibles WHERE btrim(name) = ''`)
		}
	})

	t.Run("negative price is refused at the database too", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO collectibles (collector_id, name, collection_status, purchase_price)
			VALUES ($1, 'Negative probe', 'owned', -1.00)`, collectorA)
		if err == nil {
			t.Error("the database accepted a negative purchase price (FR-017)")
			_, _ = pool.Exec(ctx, `DELETE FROM collectibles WHERE name = 'Negative probe'`)
		}
	})

	t.Run("oversized image byte_size is refused", func(t *testing.T) {
		_, err := pool.Exec(ctx, `
			INSERT INTO collectible_images
				(collector_id, original_key, rendition_key, content_type, byte_size, width, height)
			VALUES ($1, 'k', 'r', 'image/jpeg', 10485761, 10, 10)`, collectorA)
		if err == nil {
			t.Error("the database accepted an image over 10 MB; the limit must hold here too (FR-010)")
			_, _ = pool.Exec(ctx, `DELETE FROM collectible_images WHERE original_key = 'k'`)
		}
	})

	t.Run("both gallery indexes exist", func(t *testing.T) {
		for _, name := range []string{"collectibles_gallery_idx", "collectibles_status_gallery_idx"} {
			var exists bool
			if err := pool.QueryRow(ctx,
				`SELECT EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = $1)`, name).Scan(&exists); err != nil {
				t.Fatalf("query: %v", err)
			}
			if !exists {
				t.Errorf("index %s is missing", name)
			}
		}
	})

	t.Run("no unique constraint blocks deliberate duplicates", func(t *testing.T) {
		// FR-023: two identical collectibles must both be storable. A well-meaning unique index on
		// (collector_id, name) would break this.
		for i := 0; i < 2; i++ {
			if _, err := pool.Exec(ctx, `
				INSERT INTO collectibles (collector_id, name, collection_status)
				VALUES ($1, 'Duplicate probe', 'owned')`, collectorA); err != nil {
				t.Fatalf("insert %d of an identical collectible failed: %v", i+1, err)
			}
		}
		_, _ = pool.Exec(ctx, `DELETE FROM collectibles WHERE name = 'Duplicate probe'`)
	})
}
