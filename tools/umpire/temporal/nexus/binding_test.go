package nexus

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestCheckRequestBindsTheExactCallerClosureProgram(t *testing.T) {
	request, err := CheckRequest(
		admitCallerClosureSet(t),
		"umpire.local.caller-closure.binding-1",
	)
	require.NoError(t, err)

	program := request.Program()
	require.Equal(t, callerClosureProgramDefinitionID, program.DefinitionID())
	require.EqualValues(t, 1, program.Version())
	require.Equal(t, callerClosureProgramBehaviorFingerprint, program.BehaviorFingerprint())
	require.Equal(t, []string{callerClosureTargetDefinitionID}, program.TargetDefinitionIDs())
	require.Equal(t, []string{forceCloseActionDefinitionID}, program.ActionDefinitionIDs())
	require.Equal(t, callerClosureCapabilities, program.CapabilityDefinitionIDs())
	require.Equal(t, []umpireruntime.CommandKind{
		umpireruntime.CommandPrepare,
		umpireruntime.CommandRealize,
		umpireruntime.CommandObserve,
		umpireruntime.CommandCleanup,
	}, program.CommandKinds())

	occurrences := program.Occurrences()
	require.Len(t, occurrences, 1)
	require.Equal(t, forceCloseOccurrenceDefinitionID, occurrences[0].DefinitionID())
	require.Equal(t, forceCloseActionDefinitionID, occurrences[0].ActionDefinitionID())
	require.EqualValues(t, 1, occurrences[0].Position())
	require.EqualValues(t, 0, request.Seed())
	require.EqualValues(t, 1, request.Attempt())
}

func TestCallerClosureProgramVersionMatchesTheSystemModel(t *testing.T) {
	model, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "..", "model", "Temporal", "System", "Execution", "Nexus.lean",
	))
	require.NoError(t, err)
	version := regexp.MustCompile(
		`(?s)private def canonicalParticipantProgramDraft.*?reference := \{.*?version := ([0-9]+).*?def canonicalParticipantProgramDefinition.*?canonicalParticipantProgramDraft\.reference with`,
	).FindSubmatch(model)
	require.Len(t, version, 2)
	require.Equal(t, strconv.FormatUint(callerClosureProgramVersion, 10), string(version[1]))
}

func TestCheckRequestRejectsAnUnsupportedSetBeforeExecution(t *testing.T) {
	_, err := CheckRequest(artifact.AdmittedSet{}, "umpire.local.caller-closure.unsupported-1")
	require.Error(t, err)

	var preflight *umpireruntime.PreflightError
	require.True(t, errors.As(err, &preflight))
	require.Equal(t, umpireruntime.PreflightInputSet, preflight.Kind())
}

func admitCallerClosureSet(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	root := filepath.Join("testdata", "caller-closure-input-set")
	files := make(map[string][]byte, 3)
	for _, path := range []string{
		"artifacts/experiment.json",
		"artifacts/runtime-configuration.json",
		"manifest.json",
	} {
		encoded, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		require.NoError(t, err)
		files[path] = encoded
	}
	admitted, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	return admitted
}
