//go:build test_dep

package tagged

import (
	"os"
	"slices"
	"testing"
)

// The seeded runtime replaces the environment with TZ=UTC, so seeing exactly
// that proves exec.sh turned GOMAD3_CHILD_SEED into an active GOMADSEED.
func TestTargetRequiresTestDep(t *testing.T) {
	if environment := os.Environ(); !slices.Equal(environment, []string{"TZ=UTC"}) {
		t.Fatalf("environment = %v, want the seeded runtime's TZ=UTC only", environment)
	}
}
