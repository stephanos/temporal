package nexus_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
	"go.temporal.io/server/tools/umpire/temporal/local"
)

func TestCallerClosureInputSetIsStrictlyAdmitted(t *testing.T) {
	setPath := filepath.Join("testdata", "caller-closure-input-set")
	experimentBytes, err := os.ReadFile(filepath.Join(setPath, "artifacts", "experiment.json"))
	require.NoError(t, err)
	configurationBytes, err := os.ReadFile(filepath.Join(
		setPath, "artifacts", "runtime-configuration.json"))
	require.NoError(t, err)
	manifestBytes, err := os.ReadFile(filepath.Join(setPath, "manifest.json"))
	require.NoError(t, err)
	admitted, err := artifact.AdmitSetFiles(map[string][]byte{
		"artifacts/experiment.json":            experimentBytes,
		"artifacts/runtime-configuration.json": configurationBytes,
		"manifest.json":                        manifestBytes,
	})
	require.NoError(t, err)
	require.Equal(t, "umpire.artifact-set.ed3605976ba999ec8e166d4309247e2b711fee18f4a421cfb8c6dc037344f1a2", admitted.Identity())
	require.Equal(t, "sha256:074356889cda0296b13152f87e57d7b980d76125329a9014ceb5321c3f5bda7b", admitted.Checksum())

	executable, ok := admitted.Executable()
	require.True(t, ok)

	encodedExperiment, err := artifact.EncodeExperimentV2(executable.Experiment())
	require.NoError(t, err)
	require.Equal(t, experimentBytes, encodedExperiment)

	encodedConfiguration, err := artifact.EncodeRuntimeConfigurationV2(
		executable.RuntimeConfiguration())
	require.NoError(t, err)
	require.Equal(t, configurationBytes, encodedConfiguration)

	require.Equal(t, manifestBytes, admitted.ManifestBytes())
}

func TestCallerClosureInputSetPassesLocalRuntimePreflight(t *testing.T) {
	setPath := filepath.Join("testdata", "caller-closure-input-set")
	experimentBytes, err := os.ReadFile(filepath.Join(setPath, "artifacts", "experiment.json"))
	require.NoError(t, err)
	configurationBytes, err := os.ReadFile(filepath.Join(
		setPath, "artifacts", "runtime-configuration.json"))
	require.NoError(t, err)
	manifestBytes, err := os.ReadFile(filepath.Join(setPath, "manifest.json"))
	require.NoError(t, err)
	admitted, err := artifact.AdmitSetFiles(map[string][]byte{
		"artifacts/experiment.json":            experimentBytes,
		"artifacts/runtime-configuration.json": configurationBytes,
		"manifest.json":                        manifestBytes,
	})
	require.NoError(t, err)
	executable, ok := admitted.Executable()
	require.True(t, ok)
	configuration := executable.RuntimeConfiguration()
	require.Len(t, configuration.ParticipantBindings, 1)
	binding := configuration.ParticipantBindings[0]

	occurrence, err := umpireruntime.NewOccurrence(
		"workflow-nexus.occurrence.force-close",
		"workflow.action.force-close",
		1,
	)
	require.NoError(t, err)
	program, err := umpireruntime.NewProgram(
		binding.ProgramDefinitionID,
		2,
		binding.ProgramBehaviorFingerprint,
		[]string{"workflow-nexus.target.caller-closure"},
		[]string{"workflow.action.force-close"},
		[]umpireruntime.Occurrence{occurrence},
		binding.CapabilityDefinitionIDs,
	)
	require.NoError(t, err)
	authority, err := local.NewAuthority(
		configuration.ConfigurationDefinitionID,
		configuration.BehaviorFingerprint,
		binding.ParticipantDefinitionID,
		binding.ProtocolDefinitionID,
		program,
	)
	require.NoError(t, err)

	request, err := umpireruntime.CheckRequest(
		admitted,
		authority,
		"umpire.local.caller-closure.preflight-1",
		0,
		1,
	)
	require.NoError(t, err)
	require.Equal(t, admitted.Identity(), request.AdmittedSet().Identity())
}

func TestCallerClosureInputSetRejectsNoncanonicalMemberBytes(t *testing.T) {
	setPath := filepath.Join("testdata", "caller-closure-input-set")
	experimentBytes, err := os.ReadFile(filepath.Join(setPath, "artifacts", "experiment.json"))
	require.NoError(t, err)
	configurationBytes, err := os.ReadFile(filepath.Join(
		setPath, "artifacts", "runtime-configuration.json"))
	require.NoError(t, err)
	manifestBytes, err := os.ReadFile(filepath.Join(setPath, "manifest.json"))
	require.NoError(t, err)

	var compact bytes.Buffer
	require.NoError(t, json.Compact(&compact, configurationBytes))
	for _, test := range []struct {
		name          string
		configuration []byte
	}{
		{name: "compact", configuration: compact.Bytes()},
		{name: "extra line feed", configuration: append(bytes.Clone(configurationBytes), '\n')},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := artifact.AdmitSetFiles(map[string][]byte{
				"artifacts/experiment.json":            experimentBytes,
				"artifacts/runtime-configuration.json": test.configuration,
				"manifest.json":                        manifestBytes,
			})
			require.Error(t, err)
			code, ok := artifact.CodeOf(err)
			require.True(t, ok)
			require.Equal(t, artifact.ErrorNoncanonical, code)
		})
	}
}
