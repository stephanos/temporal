//go:build test_dep

package tagged

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTargetRequiresTestDep(t *testing.T) {
	require.Equal(t, time.Unix(946684800, 0).UnixNano(), time.Now().UnixNano())
	require.Equal(t, "UTC", time.Local.String())
	require.NotEmpty(t, os.Getenv("GOMADSEED"))
	require.Equal(t, 1, runtime.GOMAXPROCS(0))
}
