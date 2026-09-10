package regression

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGeneratedViewLoaderAcceptsCanonicalV2Artifact(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", "..", ".."))
	reference := Reference{
		FormatVersion: "umpire-experiment/v2",
		Identity:      "switch.query.exact-action",
		FixturePath:   "model/Umpire/Examples/testdata/switch-experiment-spec.json",
		Sources:       []string{"Umpire/Examples/Switch.lean"},
		Properties:    []string{"switch.property.flip-turns-on"},
		ObservationRequirements: []string{
			"switch.observation.power",
		},
		ArtifactChecksum: "sha256:0b745cd509a28f18f9d721d5d151892cd85cb688a4f9e025fe90055ea0f56f5a",
	}

	view, err := loadGeneratedView(repositoryRoot, reference)
	require.NoError(t, err)
	require.Equal(t, reference.ArtifactChecksum, view.ArtifactChecksum)
}
