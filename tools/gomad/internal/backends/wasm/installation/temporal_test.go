package installation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTemporalConfigRefusesPackageEscape(t *testing.T) {
	t.Parallel()
	for _, pkg := range []string{"", "../tests", "/tests", "tests/...", "tests\\fixture", "a/../../tests"} {
		t.Run(pkg, func(t *testing.T) {
			t.Parallel()
			_, err := TemporalConfig("/checkout", pkg)
			require.ErrorContains(t, err, "relative package directory")
		})
	}
}
