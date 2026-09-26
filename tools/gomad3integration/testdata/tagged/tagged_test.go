//go:build test_dep

package tagged

import (
	"os"
	"testing"
)

func TestTargetRequiresTestDep(t *testing.T) {
	if _, set := os.LookupEnv("GOMAD3_CHILD_SEED"); set {
		t.Fatal("GOMAD3_CHILD_SEED leaked into the test binary; exec.sh must consume it")
	}
	if _, set := os.LookupEnv("GOMADSEED"); !set {
		t.Fatal("GOMADSEED was not delivered to the test binary")
	}
}
