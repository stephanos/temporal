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
		Identity:      "umpire.switch.query.exactAction",
		FixturePath:   "model/Umpire/Examples/testdata/switch-experiment-spec.json",
		Sources:       []string{"Umpire/Examples/Switch.lean"},
		Properties:    []string{"umpire.switch.property.flipTurnsOn"},
		ObservationRequirements: []string{
			"umpire.switch.fact.twoState.off",
			"umpire.switch.fact.twoState.on",
		},
		ArtifactChecksum: "sha256:91c596811d96a246842d90c0cd55374cdbec3a0a054292276062958dbec6793a",
	}

	view, err := loadGeneratedView(repositoryRoot, reference)
	require.NoError(t, err)
	require.Equal(t, reference.ArtifactChecksum, view.ArtifactChecksum)
}
