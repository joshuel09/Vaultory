//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"
)

// Feature 013: sign-in is locked for an email address after five consecutive failures, for fifteen
// minutes. The rule is three PostgreSQL functions, which the frontend calls from Better Auth's
// hooks; these tests drive the functions directly. The frontend's part — which response counts as
// a failure — is covered by its own unit tests and the browser suite.

func resetLockouts(t *testing.T) {
	t.Helper()
	freshStore(t)
	if _, err := pool.Exec(context.Background(), `DELETE FROM sign_in_lockouts`); err != nil {
		t.Fatalf("reset lockouts: %v", err)
	}
}

func fail(t *testing.T, key string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `SELECT sign_in_record_failure($1)`, key); err != nil {
		t.Fatalf("record failure: %v", err)
	}
}

func locked(t *testing.T, key string) bool {
	t.Helper()
	var l bool
	if err := pool.QueryRow(context.Background(), `SELECT sign_in_is_locked($1)`, key).Scan(&l); err != nil {
		t.Fatalf("is locked: %v", err)
	}
	return l
}

func failures(t *testing.T, key string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT coalesce((SELECT failures FROM sign_in_lockouts WHERE address_key = $1), 0)`, key).Scan(&n)
	if err != nil {
		t.Fatalf("failures: %v", err)
	}
	return n
}

func clearKey(t *testing.T, key string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `SELECT sign_in_clear($1)`, key); err != nil {
		t.Fatalf("clear: %v", err)
	}
}

// T004 — FR-002, SC-001: the fifth consecutive failure locks; the first four do not.
func TestSignInLockoutAfterFiveFailures(t *testing.T) {
	resetLockouts(t)
	key := "k-five"
	for i := 1; i <= 4; i++ {
		fail(t, key)
		if locked(t, key) {
			t.Fatalf("locked after %d failures, want 5", i)
		}
	}
	fail(t, key)
	if !locked(t, key) {
		t.Fatal("not locked after 5 failures")
	}
	// Another address is untouched.
	if locked(t, "k-other") {
		t.Error("a lock on one address applied to another")
	}
}

// T004 — spec edge cases: an attempt during the lock neither counts nor extends it.
func TestSignInLockoutIsNotExtendedByAttemptsDuringIt(t *testing.T) {
	resetLockouts(t)
	key := "k-during"
	for i := 0; i < 5; i++ {
		fail(t, key)
	}
	var before string
	if err := pool.QueryRow(context.Background(),
		`SELECT locked_until::text FROM sign_in_lockouts WHERE address_key = $1`, key).Scan(&before); err != nil {
		t.Fatalf("read lock: %v", err)
	}
	fail(t, key)
	fail(t, key)
	var after string
	if err := pool.QueryRow(context.Background(),
		`SELECT locked_until::text FROM sign_in_lockouts WHERE address_key = $1`, key).Scan(&after); err != nil {
		t.Fatalf("read lock: %v", err)
	}
	if after != before || failures(t, key) != 5 {
		t.Errorf("a failure during the lock changed it: until %s → %s, failures %d", before, after, failures(t, key))
	}
}

// T004 — FR-003, SC-004: the lock lifts on its own, and the next failure starts the count at one.
func TestSignInLockoutLiftsAndStartsAgain(t *testing.T) {
	resetLockouts(t)
	key := "k-lift"
	for i := 0; i < 5; i++ {
		fail(t, key)
	}
	// Fifteen minutes later, without waiting fifteen minutes.
	if _, err := pool.Exec(context.Background(),
		`UPDATE sign_in_lockouts SET locked_until = now() - interval '1 second' WHERE address_key = $1`, key); err != nil {
		t.Fatalf("expire lock: %v", err)
	}
	if locked(t, key) {
		t.Fatal("still locked after the lock expired")
	}
	fail(t, key)
	if locked(t, key) || failures(t, key) != 1 {
		t.Errorf("after expiry one failure gave locked=%v failures=%d, want false and 1", locked(t, key), failures(t, key))
	}
}

// T004 — FR-004, FR-005: success or a completed reset clears the count and any lock.
func TestSignInLockoutIsClearedBySuccess(t *testing.T) {
	resetLockouts(t)
	key := "k-clear"
	for i := 0; i < 4; i++ {
		fail(t, key)
	}
	clearKey(t, key)
	for i := 0; i < 4; i++ {
		fail(t, key)
	}
	if locked(t, key) {
		t.Error("four failures after a clear locked: the count was not reset")
	}

	for i := 0; i < 5; i++ {
		fail(t, "k-locked")
	}
	clearKey(t, "k-locked")
	if locked(t, "k-locked") {
		t.Error("a clear did not end the lock")
	}
}

// T004 — SC-005: concurrent failures for one address are all counted.
func TestSignInLockoutCountsConcurrentFailures(t *testing.T) {
	resetLockouts(t)
	key := "k-concurrent"
	// Four at a time, so none reaches five and the lock does not freeze the count mid-test.
	for round := 0; round < 5; round++ {
		clearKey(t, key)
		var wg sync.WaitGroup
		errs := make(chan error, 4)
		for i := 0; i < 4; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := pool.Exec(context.Background(), `SELECT sign_in_record_failure($1)`, key); err != nil {
					errs <- err
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Fatalf("concurrent failure: %v", err)
		}
		if n := failures(t, key); n != 4 {
			t.Fatalf("round %d: 4 concurrent failures counted as %d", round, n)
		}
	}

	// And twenty at once lock it, with the count stopping at five.
	clearKey(t, key)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = pool.Exec(context.Background(), `SELECT sign_in_record_failure($1)`, key)
		}()
	}
	wg.Wait()
	if !locked(t, key) || failures(t, key) != 5 {
		t.Errorf("20 concurrent failures: locked=%v failures=%d, want true and 5", locked(t, key), failures(t, key))
	}
}

// Housekeeping: a count untouched for 30 days is forgotten.
func TestSignInLockoutForgetsOldCounts(t *testing.T) {
	resetLockouts(t)
	for i := 0; i < 3; i++ {
		fail(t, "k-old")
	}
	if _, err := pool.Exec(context.Background(),
		`UPDATE sign_in_lockouts SET updated_at = now() - interval '31 days' WHERE address_key = 'k-old'`); err != nil {
		t.Fatalf("age row: %v", err)
	}
	fail(t, "k-new")
	if n := failures(t, "k-old"); n != 0 {
		t.Errorf("a 31-day-old count survived: %d", n)
	}
	if n := failures(t, "k-new"); n != 1 {
		t.Errorf("new count = %d, want 1", n)
	}
}
