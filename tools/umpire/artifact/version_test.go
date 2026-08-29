package artifact_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
)

func TestUnsupportedFormatArtifactFamiliesPrecedeFieldValidation(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		decode  func([]byte) (any, error)
	}{
		{name: "ExperimentSpec", fixture: "experiment", decode: func(encoded []byte) (any, error) {
			return artifact.DecodeExperimentV2(encoded)
		}},
		{name: "RuntimeConfiguration", fixture: "runtime-configuration", decode: func(encoded []byte) (any, error) {
			return artifact.DecodeRuntimeConfigurationV2(encoded)
		}},
		{name: "ExperimentRun", fixture: "experiment-run", decode: func(encoded []byte) (any, error) {
			return artifact.DecodeExperimentRunV2(encoded)
		}},
		{name: "RawEvidence", fixture: "raw-evidence", decode: func(encoded []byte) (any, error) {
			return artifact.DecodeRawEvidenceV2(encoded)
		}},
		{name: "Evidence", fixture: "evidence", decode: func(encoded []byte) (any, error) {
			return artifact.DecodeEvidenceV2(encoded)
		}},
		{name: "Result", fixture: "result", decode: func(encoded []byte) (any, error) {
			return artifact.DecodeResultV2(encoded)
		}},
	}

	for _, test := range tests {
		for _, major := range []string{"v1", "v3"} {
			t.Run(test.name+"/"+major, func(t *testing.T) {
				encoded := readExperimentV2Fixture(t,
					"tools/umpire/artifact/testdata/unsupported/"+test.fixture+"-"+major+".json")
				value, err := test.decode(encoded)
				requireArtifactSetErrorCode(t, err, artifact.ErrorUnsupportedFormat)
				require.ErrorIs(t, err, artifact.ErrUnsupportedFormat)
				require.True(t, reflect.ValueOf(value).IsZero())
			})
		}
	}
}

func TestUnsupportedFormatArtifactSetManifestPrecedesMemberValidation(t *testing.T) {
	members := artifactSetFixtureMembers(t)
	corruptFirstArtifactChecksum(t, members[0].Encoded)
	before := cloneArtifactSetMembers(members)

	for _, major := range []string{"v1", "v3"} {
		t.Run(major, func(t *testing.T) {
			manifest := readExperimentV2Fixture(t,
				"tools/umpire/artifact/testdata/unsupported/artifact-set-"+major+".json")
			admitted, err := artifact.AdmitSetManifest(manifest, members)
			requireArtifactSetErrorCode(t, err, artifact.ErrorUnsupportedFormat)
			require.Empty(t, admitted.Identity())
			require.Nil(t, admitted.ManifestBytes())
			require.Equal(t, before, members)
		})
	}
}
