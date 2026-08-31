package nexus

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/runner"
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

func TestCheckRequestBindsTheExactDuplicateDeliveryProgram(t *testing.T) {
	request, err := CheckRequest(
		admitCallerClosureDuplicateDeliverySet(t),
		"umpire.local.caller-closure.duplicate-delivery-binding-1",
	)
	require.NoError(t, err)

	program := request.Program()
	require.Equal(t, duplicateDeliveryProgramDefinitionID, program.DefinitionID())
	require.EqualValues(t, 1, program.Version())
	require.Equal(t, duplicateDeliveryProgramBehaviorFingerprint, program.BehaviorFingerprint())
	require.Equal(t, []string{callerClosureTargetDefinitionID}, program.TargetDefinitionIDs())
	require.Equal(t, []string{forceCloseActionDefinitionID}, program.ActionDefinitionIDs())
	require.Equal(t, callerClosureCapabilities, program.CapabilityDefinitionIDs())
	require.Equal(t, duplicateDeliveryConfigurationDefinitionID,
		request.RuntimeConfiguration().ConfigurationDefinitionID)
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
	require.ErrorAs(t, err, &preflight)
	require.Equal(t, umpireruntime.PreflightInputSet, preflight.Kind())
}

func admitCallerClosureSet(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	return admitCallerClosureSetAt(t, "caller-closure-input-set")
}

func admitCallerClosureDuplicateDeliverySet(t *testing.T) artifact.AdmittedSet {
	t.Helper()
	return admitCallerClosureSetAt(t, "caller-closure-duplicate-delivery-input-set")
}

func admitCallerClosureSetAt(t *testing.T, name string) artifact.AdmittedSet {
	t.Helper()
	root := filepath.Join("testdata", name)
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

func callerClosureInputBinding() runner.InputBinding {
	return runner.InputBinding{
		ArtifactSetIdentity:                     "umpire.artifact-set.ed3605976ba999ec8e166d4309247e2b711fee18f4a421cfb8c6dc037344f1a2",
		ArtifactSetChecksum:                     "sha256:074356889cda0296b13152f87e57d7b980d76125329a9014ceb5321c3f5bda7b",
		ManifestSHA256:                          "sha256:f381da231395b8fec738837535a8bb8da0dd227a08e3d60bf9c2bda620c46b14",
		ExperimentArtifactChecksum:              "sha256:dde2fb35891dcc0020dbedf301805feda1b5136ec8622dd67fdc47a3d00fb1a8",
		ExperimentBehaviorFingerprint:           "sha256:d393ae60847c8524f3a57de6769478f95fd4a6a90a0fefcad6af118206d458af",
		RuntimeConfigurationArtifactChecksum:    "sha256:21b4f7d0db2f68f939df901c2c5d146b1be3e45e55ad6cc171445fda5f29c1d5",
		RuntimeConfigurationBehaviorFingerprint: "sha256:7c4c35a8031d07ff55ef5e83b90c64e63cbc6b196642c379ed75b5fc461f3a67",
		AuthorityRequiredCapabilityDefinitionIDs: []string{
			"umpire.runtime.capability.complete-workflow-history-read",
			"umpire.runtime.capability.ephemeral-server-lifecycle",
			"umpire.runtime.capability.sdk-worker-lifecycle",
		},
	}
}

func callerClosureDuplicateDeliveryInputBinding() runner.InputBinding {
	return runner.InputBinding{
		ArtifactSetIdentity:                     "umpire.artifact-set.2a6c3ef5fbd3b7dfba1acbe2c9ffc5ec3072b19daf50d3d63bd16b122fc2bd68",
		ArtifactSetChecksum:                     "sha256:3ddabf041e499ee0b7e970cac3900b8d6306ec9009e92924ef7b9ea0f584a5f8",
		ManifestSHA256:                          "sha256:96cf1869d444e1db25f9999ea3d3928f5c07308b8c7f387b570027f5f69b5f4b",
		ExperimentArtifactChecksum:              "sha256:09091758defd5ce50cc9acbba23a5c8499da4eef9b6e36878ac989ddea87fedf",
		ExperimentBehaviorFingerprint:           "sha256:eb6c9391f0bbd82effc5793d4b0650c3b01f2471b5f05838cdec7377a5931a91",
		RuntimeConfigurationArtifactChecksum:    "sha256:440c0632b911571e4efb34c96fb4c4c7096fbd52f23900ed4784e037370063cf",
		RuntimeConfigurationBehaviorFingerprint: "sha256:d88670a6766c2ef9037c82183f00c1c42179a7578c3c4c07714eadb5540750c0",
		AuthorityRequiredCapabilityDefinitionIDs: []string{
			"umpire.runtime.capability.complete-workflow-history-read",
			"umpire.runtime.capability.ephemeral-server-lifecycle",
			"umpire.runtime.capability.sdk-worker-lifecycle",
		},
	}
}
