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
		ArtifactChecksum: "sha256:9533fdb58edf1ef3702c9f909ea62a3546d65d0bf864e1a224706bb18925d984",
	}

	view, err := loadGeneratedView(repositoryRoot, reference)
	require.NoError(t, err)
	require.Equal(t, reference.ArtifactChecksum, view.ArtifactChecksum)
}
