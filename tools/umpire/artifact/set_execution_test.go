package artifact

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecutableSetAdmitExecutionReusesExactInputBytes(t *testing.T) {
	members := []SetMember{
		{Path: artifactSetPaths[0], Encoded: readExecutionFixture(t, "switch-experiment-v2.json")},
		{Path: artifactSetPaths[1], Encoded: readExecutionFixture(t, "runtime-configuration-v2.json")},
	}
	original := cloneSetMembers(members)
	admitted, err := AdmitSet(members)
	require.NoError(t, err)
	executable, ok := admitted.Executable()
	require.True(t, ok)

	run, err := DecodeExperimentRunV2(readExecutionFixture(t, "experiment-run-v2.json"))
	require.NoError(t, err)
	rawEvidence, err := DecodeRawEvidenceV2(readExecutionFixture(t, "raw-evidence-v2.json"))
	require.NoError(t, err)
	execution, err := executable.AdmitExecution(run, rawEvidence)
	require.NoError(t, err)

	require.Len(t, execution.members, 4)
	require.Equal(t, artifactSetPaths[:4], []string{
		execution.members[0].Path,
		execution.members[1].Path,
		execution.members[2].Path,
		execution.members[3].Path,
	})
	require.True(t, bytes.Equal(original[0].Encoded, execution.members[0].Encoded))
	require.True(t, bytes.Equal(original[1].Encoded, execution.members[1].Encoded))
	require.True(t, bytes.Equal(original[0].Encoded, admitted.members[0].Encoded))
	require.True(t, bytes.Equal(original[1].Encoded, admitted.members[1].Encoded))
	require.Nil(t, execution.executable)
}

func readExecutionFixture(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return encoded
}
