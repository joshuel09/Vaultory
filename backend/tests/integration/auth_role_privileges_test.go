//go:build integration

package integration

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// The frontend holds a database connection, because Better Auth owns the account and session
// tables. This asserts what that connection can actually reach — against the database, not against
// the migration's intentions.
//
// It is the test that keeps an argument honest. plan.md claims a bug in the frontend cannot expose
// another collector's vault; with the backend's credentials that claim would simply be false.
func TestTheAuthRoleCannotReachACollection(t *testing.T) {
	freshStore(t)
	ctx := context.Background()

	if _, err := pool.Exec(ctx, `ALTER ROLE vaultory_auth PASSWORD 'privilege-probe'`); err != nil {
		t.Fatalf("set probe password: %v", err)
	}

	cfg := pool.Config().ConnConfig.Copy()
	cfg.User, cfg.Password = "vaultory_auth", "privilege-probe"
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect as vaultory_auth: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	// Refused. Every one of these is a collection table.
	for _, table := range []string{
		"collectors", "collectibles", "collectible_images", "collectible_submissions",
	} {
		t.Run("denied on "+table, func(t *testing.T) {
			var n int
			err := conn.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n)
			if err == nil {
				t.Fatalf("read %d rows from %s — the frontend's role can see a collection", n, table)
			}
			if !strings.Contains(err.Error(), "permission denied") {
				t.Fatalf("refused for the wrong reason: %v", err)
			}
		})
	}

	// Allowed. Without this the test would pass on a role that simply cannot connect.
	for _, table := range []string{`"user"`, `"session"`, `"account"`, `"verification"`, `"rateLimit"`} {
		t.Run("permitted on "+table, func(t *testing.T) {
			var n int
			if err := conn.QueryRow(ctx, `SELECT count(*) FROM `+table).Scan(&n); err != nil {
				t.Fatalf("%s should be readable by the auth role: %v", table, err)
			}
		})
	}

	// And registration still works, through the SECURITY DEFINER trigger: the role cannot write to
	// collectors itself, but creating an account must still produce one.
	t.Run("registration still creates a collector", func(t *testing.T) {
		id := "role-probe-user"
		if _, err := conn.Exec(ctx,
			`INSERT INTO "user" ("id","name","email","emailVerified","updatedAt")
			 VALUES ($1,'Probe',$2,false,now())`, id, id+"@example.test"); err != nil {
			t.Fatalf("the auth role could not create an account: %v", err)
		}
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM collectors WHERE user_id=$1`, id).Scan(&n); err != nil {
			t.Fatalf("check collector: %v", err)
		}
		if n != 1 {
			t.Fatalf("got %d collectors for the new account, want 1 — the trigger did not fire "+
				"for a role without privileges on collectors", n)
		}
	})
}

// A credential in a tracked migration is what the constitution forbids, and this repository is
// public. The role is created without a password and provisioned from the environment; this stops
// that decision being quietly undone.
func TestNoCredentialIsCommittedInAMigration(t *testing.T) {
	dir := filepath.Join("..", "..", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}
	password := regexp.MustCompile(`(?i)\bpassword\s+'`)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		body, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		if password.Match(body) {
			t.Errorf("%s contains a literal password; provision it from the environment instead",
				e.Name())
		}
	}
}
