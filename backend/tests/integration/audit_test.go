//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Feature 014: the audit log. These drive the Go side — collection changes and the database's
// enforcement. Account events are written by the frontend and are covered by its unit tests and by
// driving the real server.

type auditEntry struct{ action, outcome, account, subject string }

func auditFor(t *testing.T, subject string) []auditEntry {
	t.Helper()
	rows, err := pool.Query(context.Background(), `
		SELECT action, outcome, coalesce(account_id, ''), coalesce(subject_id, '')
		FROM audit_events WHERE subject_id = $1 ORDER BY id`, subject)
	if err != nil {
		t.Fatalf("read audit: %v", err)
	}
	defer rows.Close()
	var out []auditEntry
	for rows.Next() {
		var e auditEntry
		if err := rows.Scan(&e.action, &e.outcome, &e.account, &e.subject); err != nil {
			t.Fatalf("scan audit: %v", err)
		}
		out = append(out, e)
	}
	return out
}

func accountOf(t *testing.T, collector uuid.UUID) string {
	t.Helper()
	var id string
	if err := pool.QueryRow(context.Background(), `SELECT user_id FROM collectors WHERE id = $1`, collector).Scan(&id); err != nil {
		t.Fatalf("account of collector: %v", err)
	}
	return id
}

// T005 — FR-001, FR-002, FR-003, SC-001: each change records one entry, with the account and the
// collectible, and nothing of what the collector typed.
func TestAuditRecordsCollectionChanges(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	d := draft("audit-create", "Canary Gundam 0079")
	notes := "canary-notes-do-not-log"
	d.Notes = &notes
	added, v, err := svc.Add(ctx, collectorA, d)
	if err != nil || len(v) > 0 {
		t.Fatalf("add: %v %v", err, v)
	}
	c := added.Row.Collectible
	changed := edit(c, c.Version)
	changed.Name = "Canary Gundam 0080"
	if _, v, err := svc.Edit(ctx, collectorA, c.ID, changed); err != nil || len(v) > 0 {
		t.Fatalf("edit: %v %v", err, v)
	}
	if err := svc.Delete(ctx, collectorA, c.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	got := auditFor(t, c.ID.String())
	account := accountOf(t, collectorA)
	want := []auditEntry{
		{"collectible.created", "succeeded", account, c.ID.String()},
		{"collectible.edited", "succeeded", account, c.ID.String()},
		{"collectible.trashed", "succeeded", account, c.ID.String()},
	}
	if len(got) != len(want) {
		t.Fatalf("entries = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// SC-002: nothing the collector typed is anywhere in the log.
	var leaks int
	if err := pool.QueryRow(ctx, `
		SELECT count(*) FROM audit_events
		WHERE concat_ws(' ', action, outcome, account_id, subject_id) ILIKE '%canary%'`).Scan(&leaks); err != nil {
		t.Fatalf("scan for leaks: %v", err)
	}
	if leaks != 0 {
		t.Errorf("%d audit entries contain the collectible's text", leaks)
	}
}

// T005 — US2/AC2: a refused change records nothing.
func TestAuditRecordsNothingForARefusedChange(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("audit-refused", "Kaiju Sentinel"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	c := added.Row.Collectible

	stale := edit(c, c.Version+5) // a version the collector was never shown
	stale.Name = "Kaiju Sentinel II"
	if _, _, err := svc.Edit(ctx, collectorA, c.ID, stale); err == nil {
		t.Fatal("an edit at a stale version was accepted")
	}
	// Someone else's collectible: answered as success, and not a deletion.
	if err := svc.Delete(ctx, collectorB, c.ID); err != nil {
		t.Fatalf("delete as another collector: %v", err)
	}

	got := auditFor(t, c.ID.String())
	if len(got) != 1 || got[0].action != "collectible.created" {
		t.Errorf("entries = %+v, want only the creation", got)
	}
}

// T005 — FR-004, SC-004: if the entry cannot be written, the change does not happen.
func TestAuditAChangeWithoutItsEntryIsNotApplied(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `ALTER TABLE audit_events RENAME TO audit_events_hidden`); err != nil {
		t.Fatalf("hide the log: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `ALTER TABLE audit_events_hidden RENAME TO audit_events`); err != nil {
			t.Fatalf("restore the log: %v", err)
		}
	})

	if _, _, err := svc.Add(ctx, collectorA, draft("audit-unrecorded", "Unrecorded")); err == nil {
		t.Fatal("a collectible was added although its audit entry could not be written")
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM collectibles WHERE name = 'Unrecorded'`).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("%d collectibles exist without their audit entry", n)
	}
}

// T006 — FR-006, SC-003: entries cannot be changed, or removed before a year, by the service that
// owns the table; the retention step removes year-old entries and only those.
func TestAuditEntriesCannotBeRewritten(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	added, _, err := svc.Add(ctx, collectorA, draft("audit-immutable", "Immutable"))
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	subject := added.Row.Collectible.ID.String()

	for _, stmt := range []string{
		`UPDATE audit_events SET outcome = 'failed' WHERE subject_id = '` + subject + `'`,
		`DELETE FROM audit_events WHERE subject_id = '` + subject + `'`,
		`TRUNCATE audit_events`,
	} {
		_, err := pool.Exec(ctx, stmt)
		if err == nil || !strings.Contains(err.Error(), "audit entries cannot be") {
			t.Errorf("%s: got %v, want a refusal", strings.Fields(stmt)[0], err)
		}
	}
	if got := auditFor(t, subject); len(got) != 1 || got[0].outcome != "succeeded" {
		t.Errorf("the entry changed: %+v", got)
	}

	// A year-old entry, made the only way one can be: inserted with an old timestamp.
	old := "audit-old-" + uuid.NewString()
	if _, err := pool.Exec(ctx, `
		INSERT INTO audit_events (occurred_at, action, outcome, subject_id)
		VALUES (now() - interval '1 year 1 day', 'collectible.created', 'succeeded', $1)`, old); err != nil {
		t.Fatalf("insert old entry: %v", err)
	}
	if n := svc.PurgeExpiredAudit(ctx); n < 1 {
		t.Errorf("purge removed %d entries, want the year-old one", n)
	}
	if len(auditFor(t, old)) != 0 {
		t.Error("a year-old entry survived the purge")
	}
	if len(auditFor(t, subject)) != 1 {
		t.Error("the purge removed a recent entry")
	}
}

// The Trash lifecycle records distinct successful transitions, never owner refusals or no-ops.
func TestAuditTrashLifecycle(t *testing.T) {
	st, svc := freshStore(t)
	ctx := context.Background()
	added, v, err := svc.Add(ctx, collectorA, draft("audit-trash", "Lifecycle"))
	if err != nil || len(v) != 0 {
		t.Fatalf("add: %v %v", err, v)
	}
	id := added.Row.Collectible.ID
	if _, err := st.Trash(ctx, collectorB, id); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Trash(ctx, collectorA, id); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Trash(ctx, collectorA, id); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Restore(ctx, collectorA, id, 30*24*time.Hour); err != nil {
		t.Fatal(err)
	}
	if _, err := st.Trash(ctx, collectorA, id); err != nil {
		t.Fatal(err)
	}
	if _, err := st.DeletePermanently(ctx, collectorA, id); err != nil {
		t.Fatal(err)
	}
	got := auditFor(t, id.String())
	want := []string{"collectible.created", "collectible.trashed", "collectible.restored", "collectible.trashed", "collectible.permanently_deleted"}
	if len(got) != len(want) {
		t.Fatalf("events=%+v, want %v", got, want)
	}
	for i, action := range want {
		if got[i].action != action || got[i].account != accountOf(t, collectorA) {
			t.Errorf("event %d=%+v", i, got[i])
		}
	}
}

func TestAuditTrashMutationsRollbackWithoutTheirEntry(t *testing.T) {
	for _, operation := range []string{"trash", "restore", "permanent", "purge"} {
		t.Run(operation, func(t *testing.T) {
			st, svc := freshStore(t)
			ctx := context.Background()
			added, v, err := svc.Add(ctx, collectorA, draft("audit-rollback-"+operation, "Rollback"))
			if err != nil || len(v) != 0 {
				t.Fatalf("seed: %v %v", err, v)
			}
			id := added.Row.Collectible.ID
			if operation != "trash" {
				if _, err := st.Trash(ctx, collectorA, id); err != nil {
					t.Fatal(err)
				}
			}
			if operation == "purge" {
				if _, err := pool.Exec(ctx, `UPDATE collectibles SET deleted_at=now()-interval '31 days' WHERE id=$1`, id); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := pool.Exec(ctx, `ALTER TABLE audit_events RENAME TO audit_events_hidden`); err != nil {
				t.Fatal(err)
			}
			defer func() {
				if _, err := pool.Exec(ctx, `ALTER TABLE audit_events_hidden RENAME TO audit_events`); err != nil {
					t.Fatal(err)
				}
			}()
			switch operation {
			case "trash":
				_, err = st.Trash(ctx, collectorA, id)
			case "restore":
				_, err = st.Restore(ctx, collectorA, id, 30*24*time.Hour)
			case "permanent":
				_, err = st.DeletePermanently(ctx, collectorA, id)
			case "purge":
				_, err = st.PurgeExpiredTrash(ctx, 30*24*time.Hour, 20)
			}
			if err == nil {
				t.Fatal("mutation succeeded without its audit entry")
			}
			var trashed bool
			if err := pool.QueryRow(ctx, `SELECT deleted_at IS NOT NULL FROM collectibles WHERE id=$1`, id).Scan(&trashed); err != nil {
				t.Fatal(err)
			}
			if trashed != (operation != "trash") {
				t.Fatalf("mutation escaped rollback: trashed=%v", trashed)
			}
		})
	}
}

func TestAuditRecordsExpiryPurge(t *testing.T) {
	st, svc := freshStore(t)
	ctx := context.Background()
	added, v, err := svc.Add(ctx, collectorA, draft("audit-purge", "Expired"))
	if err != nil || len(v) != 0 {
		t.Fatalf("seed: %v %v", err, v)
	}
	id := added.Row.Collectible.ID
	if _, err := st.Trash(ctx, collectorA, id); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE collectibles SET deleted_at=now()-interval '31 days' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := st.PurgeExpiredTrash(ctx, 30*24*time.Hour, 20); err != nil {
		t.Fatal(err)
	}
	events := auditFor(t, id.String())
	if len(events) != 3 || events[2].action != "collectible.purged" || events[2].account != accountOf(t, collectorA) {
		t.Fatalf("events=%+v", events)
	}
}
