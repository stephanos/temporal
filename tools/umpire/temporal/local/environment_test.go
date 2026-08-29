package local

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

func TestFactoryHasNoCallerSelectedAuthorityInput(t *testing.T) {
	constructor := reflect.TypeOf(NewFactory)
	require.Equal(t, 0, constructor.NumIn())
	require.Equal(t, 1, constructor.NumOut())
}

func TestRealFactoriesOwnDistinctLoopbackEnvironments(t *testing.T) {
	requests := []umpireruntime.CheckedRunRequest{
		testRequest(t, "umpire.local.environment.one"),
		testRequest(t, "umpire.local.environment.two"),
	}
	type started struct {
		environment Environment
		receipt     umpireruntime.Receipt
	}
	startedEnvironments := make([]started, len(requests))
	var group sync.WaitGroup
	for index := range requests {
		group.Add(1)
		go func() {
			defer group.Done()
			command, ok := requests[index].Command(umpireruntime.CommandPrepare)
			require.True(t, ok)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			runtimeEnvironment, receipt := NewFactory().Prepare(ctx, requests[index], command)
			require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
			environment, ok := AsEnvironment(runtimeEnvironment)
			require.True(t, ok)
			startedEnvironments[index] = started{environment: environment, receipt: receipt}
		}()
	}
	group.Wait()
	t.Cleanup(func() {
		for index, startedEnvironment := range startedEnvironments {
			if startedEnvironment.environment == nil {
				continue
			}
			command, ok := requests[index].Command(umpireruntime.CommandCleanup)
			require.True(t, ok)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			receipt := startedEnvironment.environment.Cleanup(ctx, command)
			cancel()
			require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
		}
	})

	first := startedEnvironments[0].environment.Identities()
	second := startedEnvironments[1].environment.Identities()
	require.NotEqual(t, first.Namespace, second.Namespace)
	require.NotEqual(t, first.Endpoint, second.Endpoint)
	require.NotEqual(t, first.TaskQueue, second.TaskQueue)
	for _, identity := range []string{
		first.Namespace, first.Endpoint, first.TaskQueue,
		second.Namespace, second.Endpoint, second.TaskQueue,
	} {
		require.Regexp(t, `^sha256:[0-9a-f]{64}$`, identity)
	}
	require.NotNil(t, startedEnvironments[0].environment.Client())
	require.NotNil(t, startedEnvironments[1].environment.Client())
	require.NotEqual(t,
		startedEnvironments[0].environment.Client(),
		startedEnvironments[1].environment.Client(),
	)

	for index, startedEnvironment := range startedEnvironments {
		command, ok := requests[index].Command(umpireruntime.CommandRealize)
		require.True(t, ok)
		options, ok := startedEnvironment.environment.WorkflowOptions(command)
		require.True(t, ok)
		require.NotEmpty(t, options.ID)
		require.NotEmpty(t, options.TaskQueue)
		require.NotNil(t, options.RetryPolicy)
		require.EqualValues(t, 1, options.RetryPolicy.MaximumAttempts)
	}
}

func TestFactoryCancellationDoesNotStartAuthority(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.canceled")
	command, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	starter := &recordingStarter{}
	factory := newFactory(starter)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	environment, receipt := factory.Prepare(ctx, request, command)
	require.Nil(t, environment)
	require.Equal(t, umpireruntime.ReceiptCanceled, receipt.Status())
	require.Equal(t, 0, starter.calls)
	require.Equal(t, "umpire.runtime.code.canceled", errorCode(receipt))
}

func TestFactorySanitizesPartialStartupFailures(t *testing.T) {
	request := testRequest(t, "umpire.local.environment.partial")
	command, ok := request.Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	backend := &recordingAuthority{
		resources: []ownedResource{{kind: ownedServer}},
		clientErr: errors.New("secret endpoint and credential"),
	}
	factory := newFactory(&recordingStarter{authority: backend})

	runtimeEnvironment, receipt := factory.Prepare(context.Background(), request, command)
	require.Equal(t, umpireruntime.ReceiptFailed, receipt.Status())
	require.Equal(t, "umpire.runtime.code.failed", errorCode(receipt))
	require.NotContains(t, receiptText(receipt), "secret")
	require.NotContains(t, receiptText(receipt), "credential")
	environment, ok := AsEnvironment(runtimeEnvironment)
	require.True(t, ok)

	cleanupCommand, ok := request.Command(umpireruntime.CommandCleanup)
	require.True(t, ok)
	cleanupReceipt := environment.Cleanup(context.Background(), cleanupCommand)
	require.Equal(t, umpireruntime.ReceiptAccepted, cleanupReceipt.Status())
	require.Equal(t, []string{"server"}, backend.releaseOrder)
}

func testRequest(t *testing.T, runIdentity string) umpireruntime.CheckedRunRequest {
	t.Helper()
	experimentBytes, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "..", "model", "Umpire", "Artifact", "Tests", "Fixtures", "SwitchExperimentSpecV2.json",
	))
	require.NoError(t, err)
	experiment, err := artifact.DecodeExperimentV2(experimentBytes)
	require.NoError(t, err)
	experiment.Plan.CapabilityRequirementDefinitionIDs = []string{"switch.capability.state"}
	experiment, err = artifactv2.SealExperiment(experiment)
	require.NoError(t, err)

	configurationBytes, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "..", "model", "Umpire", "Artifact", "Tests", "Fixtures", "RuntimeConfigurationV2.json",
	))
	require.NoError(t, err)
	configuration, err := artifact.DecodeRuntimeConfigurationV2(configurationBytes)
	require.NoError(t, err)
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)
	configuration.Experiment = experimentBinding
	configuration.AuthorityProfile = artifactv2.AuthorityProfile{
		DefinitionID:                    ProfileDefinitionID,
		Version:                         artifactv2.NaturalFromUint64(ProfileVersion),
		BehaviorFingerprint:             ProfileBehaviorFingerprint,
		RequiredCapabilityDefinitionIDs: []string{},
	}
	configuration, err = artifactv2.SealRuntimeConfiguration(configuration)
	require.NoError(t, err)
	experimentBytes, err = artifact.EncodeExperimentV2(experiment)
	require.NoError(t, err)
	configurationBytes, err = artifact.EncodeRuntimeConfigurationV2(configuration)
	require.NoError(t, err)
	set, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experimentBytes},
		{Path: "artifacts/runtime-configuration.json", Encoded: configurationBytes},
	})
	require.NoError(t, err)

	program := testProgram(t)
	authority, err := NewAuthority(
		configuration.ConfigurationDefinitionID,
		configuration.BehaviorFingerprint,
		configuration.ParticipantBindings[0].ParticipantDefinitionID,
		configuration.ParticipantBindings[0].ProtocolDefinitionID,
		program,
	)
	require.NoError(t, err)
	request, err := umpireruntime.CheckRequest(set, authority, runIdentity, 0, 1)
	require.NoError(t, err)
	return request
}

func testProgram(t *testing.T) umpireruntime.Program {
	t.Helper()
	occurrence, err := umpireruntime.NewOccurrence("switch.occurrence.flip", "switch.action.flip", 1)
	require.NoError(t, err)
	program, err := umpireruntime.NewProgram(
		"switch.participant.program",
		2,
		"sha256:92489e8192608a0a88e319591737e528194b9c1239ac3d08f92bdc692aca3d31",
		[]string{"switch.target.two-state"},
		[]string{"switch.action.flip"},
		[]umpireruntime.Occurrence{occurrence},
		[]string{"switch.capability.state"},
	)
	require.NoError(t, err)
	return program
}

func errorCode(receipt umpireruntime.Receipt) string {
	for _, fact := range receipt.Facts() {
		for _, field := range fact.Fields() {
			if field.DefinitionID() == umpireruntime.EvidenceFieldErrorCode {
				return field.Value()
			}
		}
	}
	return ""
}

func receiptText(receipt umpireruntime.Receipt) string {
	values := []string{}
	for _, fact := range receipt.Facts() {
		values = append(values, fact.DefinitionID(), fact.SourceDefinitionID(), fact.KindDefinitionID())
		for _, field := range fact.Fields() {
			values = append(values, field.DefinitionID(), field.Value())
		}
	}
	slices.Sort(values)
	return strings.Join(values, "\n")
}

var _ client.Client
