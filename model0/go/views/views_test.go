package views

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// Every view renders the same bytes twice and equals its checked-in golden. Regenerate with
// `go run ./model/go/views/cmd/render -out model/go/views/testdata`.
func TestViewsAreDeterministicAndMatchTheGoldens(t *testing.T) {
	first, err := All()
	require.NoError(t, err)
	second, err := All()
	require.NoError(t, err)
	require.Equal(t, first, second)
	for name, content := range first {
		golden, err := os.ReadFile(filepath.Join("testdata", name))
		require.NoError(t, err, "missing golden %s", name)
		require.Equal(t, string(golden), content, name)
	}
}
