//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
)

// T083: text arrives back exactly as entered, whatever script it is in.
func TestTextFidelityThroughStorage(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	names := []string{
		"怪獣センチネル",
		"Ámbar Guardián",
		"Sentinel 🦖 Prime",
		"Кайдзю Страж",
		"مقاتل الوحوش",
		"Ω Sentinel β",
		"O'Brien's \"Special\" Edition",
	}
	for i, name := range names {
		d := draft("text-"+name, name)
		note := name + " — note"
		d.Notes = &note
		if _, v, err := svc.Add(ctx, collectorA, d); err != nil || len(v) > 0 {
			t.Fatalf("add %d: %v %+v", i, err, v)
		}
	}

	page, err := svc.List(ctx, collectorA, nil, "", 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	got := map[string]bool{}
	for _, r := range page.Rows {
		got[r.Collectible.Name] = true
		if r.Collectible.Notes == nil || !strings.HasPrefix(*r.Collectible.Notes, r.Collectible.Name) {
			t.Errorf("note for %q did not survive intact", r.Collectible.Name)
		}
	}
	for _, name := range names {
		if !got[name] {
			t.Errorf("%q did not round-trip", name)
		}
	}
}

// T084: a failure tells the collector nothing about the inside of the system.
func TestErrorsDoNotLeakInternals(t *testing.T) {
	_, svc := freshStore(t)
	ctx := context.Background()

	// A malformed cursor is the closest thing to an internal failure a client can provoke here.
	_, err := svc.List(ctx, collectorA, nil, "!!!not-base64!!!", 10)
	if err == nil {
		t.Fatal("a malformed cursor should fail")
	}
	leaks := []string{"pgx", "SELECT", "INSERT", "collector_id", "pq:", "sql:", "goroutine", ".go:"}
	for _, leak := range leaks {
		if strings.Contains(err.Error(), leak) {
			t.Errorf("error message contains %q, which reveals internals: %v", leak, err)
		}
	}
}
