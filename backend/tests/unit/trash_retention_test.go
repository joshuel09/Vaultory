package unit

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/joshuel09/vaultory/backend/internal/collection"
)

// T036 — feature 010, FR-006: the frontend states the Trash's retention in its copy, and the
// backend decides it. The two are in different languages, so this reads the frontend's constant
// and fails when they disagree — changing one and not the other is loud, not a sentence that
// quietly lies (#73).
func TestTheFrontendStatesTheTrashRetention(t *testing.T) {
	path := filepath.Join("..", "..", "..", "frontend", "lib", "trash-retention.ts")
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	m := regexp.MustCompile(`export const TRASH_RETENTION_DAYS = (\d+)`).FindSubmatch(src)
	if m == nil {
		t.Fatalf("%s: no `export const TRASH_RETENTION_DAYS = <days>`", path)
	}
	days, _ := strconv.Atoi(string(m[1]))
	if got, want := time.Duration(days)*24*time.Hour, collection.TrashRetention(); got != want {
		t.Errorf("the frontend says %d days; the backend keeps the Trash for %v", days, want)
	}
}
