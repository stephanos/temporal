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
		ArtifactChecksum: "sha256:ac3fde668a79ff0433106e28f8ec9579a36f9f7d0ab09845d01b563289b560fd",
	}

	view, err := loadGeneratedView(repositoryRoot, reference)
	require.NoError(t, err)
	require.Equal(t, reference.ArtifactChecksum, view.ArtifactChecksum)
}
