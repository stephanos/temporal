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
		ArtifactChecksum: "sha256:9fa327849c3d0a48290bb16fec73a00be4cc1b6234862ee506a547f29b6d3b12",
	}

	view, err := loadGeneratedView(repositoryRoot, reference)
	require.NoError(t, err)
	require.Equal(t, reference.ArtifactChecksum, view.ArtifactChecksum)
}
