package runtime_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/tools/umpire/artifact"
	"go.temporal.io/server/tools/umpire/internal/artifactv2"
	umpireruntime "go.temporal.io/server/tools/umpire/runtime"
)

const (
	profileDefinitionID     = "runtime.profile.ephemeral-local"
	profileFingerprint      = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	programDefinitionID     = "runtime.program.single-control"
	programFingerprint      = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	participantDefinitionID = "runtime.participant.sdk"
	protocolDefinitionID    = "runtime.protocol.v2"
)

var profileCapabilities = []string{
	"runtime.capability.complete-history-read",
	"runtime.capability.ephemeral-server-lifecycle",
	"runtime.capability.worker-lifecycle",
}

type checkedFixture struct {
	set                  artifact.AdmittedSet
	experiment           artifactv2.Experiment
	runtimeConfiguration artifactv2.RuntimeConfiguration
	program              umpireruntime.Program
	authority            umpireruntime.Authority
}

func TestCheckRequestAcceptsOneImmutableExactInput(t *testing.T) {
	fixture := newCheckedFixture(t)
	request, err := umpireruntime.CheckRequest(
		fixture.set,
		fixture.authority,
		"runtime.run.accepted-1",
		0,
		1,
	)
	require.NoError(t, err)
	require.Equal(t, fixture.set.Identity(), request.AdmittedSet().Identity())
	require.Equal(t, "runtime.run.accepted-1", request.RunIdentity())
	require.EqualValues(t, 0, request.Seed())
	require.EqualValues(t, 1, request.Attempt())
	require.Equal(t, fixture.program, request.Program())
	require.Equal(t, umpireruntime.CanonicalPhaseLimits(), request.PhaseLimits())
	require.Equal(t, []umpireruntime.CommandKind{
		umpireruntime.CommandPrepare,
		umpireruntime.CommandRealize,
		umpireruntime.CommandObserve,
		umpireruntime.CommandCleanup,
	}, request.Program().CommandKinds())

	correlations := request.Correlations()
	require.Equal(t, []umpireruntime.CorrelationKind{
		umpireruntime.CorrelationWorkflow,
		umpireruntime.CorrelationOperation,
		umpireruntime.CorrelationTaskQueue,
		umpireruntime.CorrelationWorker,
		umpireruntime.CorrelationParticipant,
	}, correlationKinds(correlations))
	for _, correlation := range correlations {
		require.True(t, strings.HasPrefix(
			correlation.Identity(),
			"runtime.correlation."+string(correlation.Kind())+".",
		))
	}

	limits := request.PhaseLimits()
	limits[0] = limits[1]
	correlations[0] = correlations[1]
	require.Equal(t, umpireruntime.PhasePreparation, request.PhaseLimits()[0].Phase())
	require.Equal(t, umpireruntime.CorrelationWorkflow, request.Correlations()[0].Kind())

	command, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	require.Equal(t, umpireruntime.PhaseRealization, command.Phase())
	require.Equal(t, request.RunIdentity(), command.RunIdentity())
	require.EqualValues(t, 1, command.Attempt())
	require.Equal(t, fixture.program.DefinitionID(), command.ProgramDefinitionID())
	require.Equal(t, fixture.program.Occurrence().DefinitionID(), command.OccurrenceDefinitionID())
	require.Equal(t, umpireruntime.PhaseIsolation, request.IsolationCommand().Phase())
}

func TestCheckRequestRejectsEachPreflightMutationBeforeIO(t *testing.T) {
	fixture := newCheckedFixture(t)
	factory := &countingFactory{}

	tests := []struct {
		name string
		kind umpireruntime.PreflightErrorKind
		run  func(t *testing.T) (umpireruntime.CheckedRunRequest, error)
	}{
		{
			name: "input set",
			kind: umpireruntime.PreflightInputSet,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				return umpireruntime.CheckRequest(
					artifact.AdmittedSet{}, fixture.authority, "runtime.run.invalid-set", 0, 1,
				)
			},
		},
		{
			name: "profile",
			kind: umpireruntime.PreflightProfile,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				authority := newAuthority(t, fixture.program, authorityMutation{
					profileDefinitionID: "runtime.profile.drifted",
				})
				return umpireruntime.CheckRequest(
					fixture.set, authority, "runtime.run.profile-drift", 0, 1,
				)
			},
		},
		{
			name: "configuration",
			kind: umpireruntime.PreflightConfiguration,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				set := mutateConfiguration(t, fixture, func(configuration *artifactv2.RuntimeConfiguration) {
					configuration.KnownGaps = []artifactv2.KnownGap{{
						Kind: "input", Code: "runtime.configuration.gap",
					}}
				})
				return umpireruntime.CheckRequest(
					set, fixture.authority, "runtime.run.configuration-drift", 0, 1,
				)
			},
		},
		{
			name: "target",
			kind: umpireruntime.PreflightTarget,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				program := newProgram(t, fixture.experiment, programMutation{
					targetDefinitionID: "switch.target.drifted",
				})
				return umpireruntime.CheckRequest(
					fixture.set, newAuthority(t, program, authorityMutation{}),
					"runtime.run.target-drift", 0, 1,
				)
			},
		},
		{
			name: "action",
			kind: umpireruntime.PreflightAction,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				program := newProgram(t, fixture.experiment, programMutation{
					actionDefinitionID: "switch.action.drifted",
					occurrenceActionID: "switch.action.drifted",
				})
				return umpireruntime.CheckRequest(
					fixture.set, newAuthority(t, program, authorityMutation{}),
					"runtime.run.action-drift", 0, 1,
				)
			},
		},
		{
			name: "occurrence",
			kind: umpireruntime.PreflightOccurrence,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				program := newProgram(t, fixture.experiment, programMutation{
					occurrenceDefinitionID: "switch.occurrence.drifted",
				})
				return umpireruntime.CheckRequest(
					fixture.set, newAuthority(t, program, authorityMutation{}),
					"runtime.run.occurrence-drift", 0, 1,
				)
			},
		},
		{
			name: "participant",
			kind: umpireruntime.PreflightParticipant,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				authority := newAuthority(t, fixture.program, authorityMutation{
					participantDefinitionID: "runtime.participant.drifted",
				})
				return umpireruntime.CheckRequest(
					fixture.set, authority, "runtime.run.participant-drift", 0, 1,
				)
			},
		},
		{
			name: "protocol",
			kind: umpireruntime.PreflightProtocol,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				authority := newAuthority(t, fixture.program, authorityMutation{
					protocolDefinitionID: "runtime.protocol.drifted",
				})
				return umpireruntime.CheckRequest(
					fixture.set, authority, "runtime.run.protocol-drift", 0, 1,
				)
			},
		},
		{
			name: "capability",
			kind: umpireruntime.PreflightCapability,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				program := newProgram(t, fixture.experiment, programMutation{
					capabilityDefinitionIDs: []string{"switch.capability.other"},
				})
				return umpireruntime.CheckRequest(
					fixture.set, newAuthority(t, program, authorityMutation{}),
					"runtime.run.capability-drift", 0, 1,
				)
			},
		},
		{
			name: "profile capability ownership",
			kind: umpireruntime.PreflightCapability,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				set := mutateConfiguration(t, fixture, func(configuration *artifactv2.RuntimeConfiguration) {
					configuration.AuthorityProfile.RequiredCapabilityDefinitionIDs = append(
						append([]string{}, profileCapabilities...),
						"switch.capability.state",
					)
				})
				return umpireruntime.CheckRequest(
					set, fixture.authority, "runtime.run.profile-capability-drift", 0, 1,
				)
			},
		},
		{
			name: "budget",
			kind: umpireruntime.PreflightBudget,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				set := mutateConfiguration(t, fixture, func(configuration *artifactv2.RuntimeConfiguration) {
					configuration.PhaseLimits[0].DurationMilliseconds = artifactv2.Natural("30001")
				})
				return umpireruntime.CheckRequest(
					set, fixture.authority, "runtime.run.budget-drift", 0, 1,
				)
			},
		},
		{
			name: "run identity",
			kind: umpireruntime.PreflightRunIdentity,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				return umpireruntime.CheckRequest(fixture.set, fixture.authority, "not-namespaced", 0, 1)
			},
		},
		{
			name: "seed",
			kind: umpireruntime.PreflightSeed,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				return umpireruntime.CheckRequest(fixture.set, fixture.authority, "runtime.run.seed", 1, 1)
			},
		},
		{
			name: "attempt",
			kind: umpireruntime.PreflightAttempt,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				return umpireruntime.CheckRequest(fixture.set, fixture.authority, "runtime.run.attempt", 0, 2)
			},
		},
		{
			name: "duplicate",
			kind: umpireruntime.PreflightDuplicate,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				occurrence, err := umpireruntime.NewOccurrence(
					fixture.experiment.Plan.LinearExtension[0].DefinitionID,
					fixture.experiment.Plan.LinearExtension[0].ActionDefinitionID,
					1,
				)
				require.NoError(t, err)
				_, err = umpireruntime.NewProgram(
					programDefinitionID,
					2,
					programFingerprint,
					[]string{fixture.experiment.Plan.TargetDefinitionID},
					[]string{fixture.experiment.Plan.RequestedActions[0].DefinitionID},
					[]umpireruntime.Occurrence{occurrence},
					[]string{"switch.capability.state", "switch.capability.state"},
				)
				return umpireruntime.CheckedRunRequest{}, err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := factory.calls
			request, err := test.run(t)
			requirePreflightKind(t, err, test.kind)
			require.Empty(t, request.RunIdentity())
			require.Equal(t, before, factory.calls)
		})
	}
}

func TestCheckedContractValuesEnforceBoundsAndCanonicalOrder(t *testing.T) {
	fixture := newCheckedFixture(t)
	command, ok := mustCheckedRequest(t, fixture).Command(umpireruntime.CommandPrepare)
	require.True(t, ok)

	field, err := umpireruntime.NewFactField("runtime.field.status", "accepted")
	require.NoError(t, err)
	fact, err := umpireruntime.NewFact(
		"runtime.fact.prepared",
		"runtime.source.participant",
		"runtime.fact-kind.receipt",
		[]string{},
		[]umpireruntime.FactField{field},
	)
	require.NoError(t, err)
	require.NotNil(t, fact.CausalDefinitionIDs())
	resource, err := umpireruntime.NewResource(
		umpireruntime.ResourceParticipant,
		"runtime.resource.participant-1",
	)
	require.NoError(t, err)
	receipt, err := umpireruntime.NewReceipt(
		command,
		umpireruntime.ReceiptAccepted,
		[]umpireruntime.Fact{fact},
		[]umpireruntime.Resource{resource},
		[]umpireruntime.Resource{},
	)
	require.NoError(t, err)
	require.Equal(t, command, receipt.Command())
	require.Equal(t, umpireruntime.ReceiptAccepted, receipt.Status())
	require.Equal(t, []umpireruntime.Fact{fact}, receipt.Facts())
	require.Equal(t, []umpireruntime.Resource{resource}, receipt.AcquiredResources())

	_, err = umpireruntime.NewFactField(
		"runtime.field.too-large",
		strings.Repeat("x", umpireruntime.MaximumFactValueBytes+1),
	)
	require.Error(t, err)
	_, err = umpireruntime.NewFactField("runtime.field.raw-message", "raw error text")
	require.Error(t, err)
	_, err = umpireruntime.NewResource("unknown", "runtime.resource.invalid")
	require.Error(t, err)
	_, err = umpireruntime.NewReceipt(
		command,
		umpireruntime.ReceiptStatus("unknown"),
		[]umpireruntime.Fact{},
		[]umpireruntime.Resource{},
		[]umpireruntime.Resource{},
	)
	require.Error(t, err)
	_, err = umpireruntime.NewPhaseLimit(
		umpireruntime.PhasePreparation,
		999*time.Millisecond,
		1,
		1,
		1,
	)
	requirePreflightKind(t, err, umpireruntime.PreflightBudget)
	var preflight *umpireruntime.PreflightError
	_, err = umpireruntime.NewPhaseLimit(
		umpireruntime.Phase("invalid\nphase"),
		time.Second,
		1,
		1,
		1,
	)
	require.ErrorAs(t, err, &preflight)
	require.Equal(t, "phase-limit", preflight.Subject())
}

func TestCheckedCollectionsPreserveShapeAndEnforceLimits(t *testing.T) {
	fixture := newCheckedFixture(t)
	occurrence := fixture.program.Occurrence()
	capabilities := numberedIdentities("runtime.capability", artifact.MaximumJSONArrayItems)
	program, err := umpireruntime.NewProgram(
		programDefinitionID,
		2,
		programFingerprint,
		fixture.program.TargetDefinitionIDs(),
		fixture.program.ActionDefinitionIDs(),
		[]umpireruntime.Occurrence{occurrence},
		capabilities,
	)
	require.NoError(t, err)
	require.Equal(t, capabilities, program.CapabilityDefinitionIDs())
	_, err = umpireruntime.NewProgram(
		programDefinitionID,
		2,
		programFingerprint,
		fixture.program.TargetDefinitionIDs(),
		fixture.program.ActionDefinitionIDs(),
		[]umpireruntime.Occurrence{occurrence},
		numberedIdentities("runtime.capability", artifact.MaximumJSONArrayItems+1),
	)
	require.Error(t, err)

	causes := numberedIdentities("runtime.cause", artifact.MaximumJSONArrayItems)
	fact, err := umpireruntime.NewFact(
		"runtime.fact.maximum-causes",
		"runtime.source.participant",
		"runtime.fact-kind.receipt",
		causes,
		[]umpireruntime.FactField{},
	)
	require.NoError(t, err)
	require.Equal(t, causes, fact.CausalDefinitionIDs())
	require.NotNil(t, fact.Fields())
	_, err = umpireruntime.NewFact(
		"runtime.fact.too-many-causes",
		"runtime.source.participant",
		"runtime.fact-kind.receipt",
		numberedIdentities("runtime.cause", artifact.MaximumJSONArrayItems+1),
		[]umpireruntime.FactField{},
	)
	require.Error(t, err)

	command, ok := mustCheckedRequest(t, fixture).Command(umpireruntime.CommandPrepare)
	require.True(t, ok)
	resources := make([]umpireruntime.Resource, artifact.MaximumJSONArrayItems)
	for index, identity := range numberedIdentities("runtime.resource", len(resources)) {
		resources[index], err = umpireruntime.NewResource(umpireruntime.ResourceParticipant, identity)
		require.NoError(t, err)
	}
	receipt, err := umpireruntime.NewReceipt(
		command,
		umpireruntime.ReceiptAccepted,
		[]umpireruntime.Fact{},
		resources,
		[]umpireruntime.Resource{},
	)
	require.NoError(t, err)
	require.NotNil(t, receipt.Facts())
	require.Equal(t, resources, receipt.AcquiredResources())
	require.NotNil(t, receipt.ReleasedResources())
	extraResource, err := umpireruntime.NewResource(
		umpireruntime.ResourceParticipant,
		"runtime.resource.overflow",
	)
	require.NoError(t, err)
	_, err = umpireruntime.NewReceipt(
		command,
		umpireruntime.ReceiptAccepted,
		[]umpireruntime.Fact{},
		append(resources, extraResource),
		[]umpireruntime.Resource{},
	)
	require.Error(t, err)
}

func TestCheckRequestAcceptsEveryBoundedRunIdentity(t *testing.T) {
	fixture := newCheckedFixture(t)
	for _, length := range []int{500, 501, umpireruntime.MaximumIdentityBytes} {
		t.Run(fmt.Sprintf("%d bytes", length), func(t *testing.T) {
			runIdentity := "runtime." + strings.Repeat("x", length-len("runtime."))
			request, err := umpireruntime.CheckRequest(
				fixture.set,
				fixture.authority,
				runIdentity,
				0,
				1,
			)
			require.NoError(t, err)
			require.Equal(t, runIdentity, request.RunIdentity())
			require.Len(t, request.Correlations(), 5)
		})
	}
}

func newCheckedFixture(t *testing.T) checkedFixture {
	t.Helper()
	experimentBytes := readFixture(t, "SwitchExperimentSpecV2.json")
	runtimeConfigurationBytes := readFixture(t, "RuntimeConfigurationV2.json")
	experiment, err := artifact.DecodeExperimentV2(experimentBytes)
	require.NoError(t, err)
	runtimeConfiguration, err := artifact.DecodeRuntimeConfigurationV2(runtimeConfigurationBytes)
	require.NoError(t, err)

	experiment.Plan.CapabilityRequirementDefinitionIDs = append(
		append([]string{}, profileCapabilities...),
		"switch.capability.state",
	)
	experiment, err = artifactv2.SealExperiment(experiment)
	require.NoError(t, err)
	experimentBytes, err = artifactv2.CanonicalExperimentBytes(experiment)
	require.NoError(t, err)
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)

	runtimeConfiguration.Experiment = experimentBinding
	runtimeConfiguration.AuthorityProfile = artifactv2.AuthorityProfile{
		DefinitionID:                    profileDefinitionID,
		Version:                         artifactv2.Natural("2"),
		BehaviorFingerprint:             profileFingerprint,
		RequiredCapabilityDefinitionIDs: append([]string{}, profileCapabilities...),
	}
	runtimeConfiguration.ParticipantBindings = []artifactv2.ParticipantBinding{{
		ParticipantDefinitionID:    participantDefinitionID,
		ProtocolDefinitionID:       protocolDefinitionID,
		ProtocolVersion:            artifactv2.Natural("2"),
		ProgramDefinitionID:        programDefinitionID,
		ProgramBehaviorFingerprint: programFingerprint,
		CapabilityDefinitionIDs:    []string{"switch.capability.state"},
	}}
	runtimeConfiguration.KnownGaps = []artifactv2.KnownGap{}
	runtimeConfiguration, err = artifactv2.SealRuntimeConfiguration(runtimeConfiguration)
	require.NoError(t, err)
	runtimeConfigurationBytes, err = artifactv2.CanonicalRuntimeConfigurationBytes(runtimeConfiguration)
	require.NoError(t, err)

	set, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experimentBytes},
		{Path: "artifacts/runtime-configuration.json", Encoded: runtimeConfigurationBytes},
	})
	require.NoError(t, err)
	program := newProgram(t, experiment, programMutation{})
	authority := newAuthority(t, program, authorityMutation{})
	return checkedFixture{
		set:                  set,
		experiment:           experiment,
		runtimeConfiguration: runtimeConfiguration,
		program:              program,
		authority:            authority,
	}
}

type programMutation struct {
	targetDefinitionID      string
	actionDefinitionID      string
	occurrenceDefinitionID  string
	occurrenceActionID      string
	capabilityDefinitionIDs []string
}

func newProgram(
	t *testing.T,
	experiment artifactv2.Experiment,
	mutation programMutation,
) umpireruntime.Program {
	t.Helper()
	targetDefinitionID := valueOr(mutation.targetDefinitionID, experiment.Plan.TargetDefinitionID)
	actionDefinitionID := valueOr(
		mutation.actionDefinitionID,
		experiment.Plan.RequestedActions[0].DefinitionID,
	)
	occurrenceDefinitionID := valueOr(
		mutation.occurrenceDefinitionID,
		experiment.Plan.LinearExtension[0].DefinitionID,
	)
	occurrenceActionID := valueOr(mutation.occurrenceActionID, actionDefinitionID)
	capabilities := mutation.capabilityDefinitionIDs
	if capabilities == nil {
		capabilities = []string{"switch.capability.state"}
	}
	occurrence, err := umpireruntime.NewOccurrence(
		occurrenceDefinitionID,
		occurrenceActionID,
		1,
	)
	require.NoError(t, err)
	program, err := umpireruntime.NewProgram(
		programDefinitionID,
		2,
		programFingerprint,
		[]string{targetDefinitionID},
		[]string{actionDefinitionID},
		[]umpireruntime.Occurrence{occurrence},
		capabilities,
	)
	require.NoError(t, err)
	return program
}

type authorityMutation struct {
	profileDefinitionID     string
	participantDefinitionID string
	protocolDefinitionID    string
}

func newAuthority(
	t *testing.T,
	program umpireruntime.Program,
	mutation authorityMutation,
) umpireruntime.Authority {
	t.Helper()
	authority, err := umpireruntime.NewAuthority(
		valueOr(mutation.profileDefinitionID, profileDefinitionID),
		2,
		profileFingerprint,
		"switch.runtime.configuration",
		"sha256:6b81f3a1bc1b67f699b5f2dd7bd030e08c4bcf52c656274d4b25abb374bb87df",
		profileCapabilities,
		profileCapabilities,
		umpireruntime.CanonicalPhaseLimits(),
		0,
		1,
		valueOr(mutation.participantDefinitionID, participantDefinitionID),
		valueOr(mutation.protocolDefinitionID, protocolDefinitionID),
		2,
		1,
		1,
		program,
	)
	require.NoError(t, err)
	return authority
}

func mutateConfiguration(
	t *testing.T,
	fixture checkedFixture,
	mutate func(*artifactv2.RuntimeConfiguration),
) artifact.AdmittedSet {
	t.Helper()
	configuration := fixture.runtimeConfiguration
	configuration.PhaseLimits = append([]artifactv2.PhaseLimit{}, configuration.PhaseLimits...)
	configuration.KnownGaps = append([]artifactv2.KnownGap{}, configuration.KnownGaps...)
	mutate(&configuration)
	configuration, err := artifactv2.SealRuntimeConfiguration(configuration)
	require.NoError(t, err)
	configurationBytes, err := artifactv2.CanonicalRuntimeConfigurationBytes(configuration)
	require.NoError(t, err)
	experimentBytes, err := artifactv2.CanonicalExperimentBytes(fixture.experiment)
	require.NoError(t, err)
	set, err := artifact.AdmitSet([]artifact.SetMember{
		{Path: "artifacts/experiment.json", Encoded: experimentBytes},
		{Path: "artifacts/runtime-configuration.json", Encoded: configurationBytes},
	})
	require.NoError(t, err)
	return set
}

func mustCheckedRequest(t *testing.T, fixture checkedFixture) umpireruntime.CheckedRunRequest {
	t.Helper()
	request, err := umpireruntime.CheckRequest(
		fixture.set, fixture.authority, "runtime.run.contract", 0, 1,
	)
	require.NoError(t, err)
	return request
}

func requirePreflightKind(
	t *testing.T,
	err error,
	want umpireruntime.PreflightErrorKind,
) {
	t.Helper()
	var preflight *umpireruntime.PreflightError
	require.ErrorAs(t, err, &preflight)
	require.Equal(t, want, preflight.Kind())
}

func correlationKinds(correlations []umpireruntime.Correlation) []umpireruntime.CorrelationKind {
	kinds := make([]umpireruntime.CorrelationKind, len(correlations))
	for index, correlation := range correlations {
		kinds[index] = correlation.Kind()
	}
	return kinds
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	encoded, err := os.ReadFile(filepath.Join(
		"..", "..", "..", "model", "Umpire", "Artifact", "Tests", "Fixtures", name,
	))
	require.NoError(t, err)
	return encoded
}

func valueOr(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func numberedIdentities(prefix string, count int) []string {
	identities := make([]string, count)
	for index := range identities {
		identities[index] = fmt.Sprintf("%s.%04d", prefix, index)
	}
	return identities
}

type countingFactory struct {
	calls int
}
