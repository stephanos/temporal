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
	"go.temporal.io/server/tools/umpire/temporal/local"
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

func TestDuplicateDeliveryInputSetIsCanonicalAndPreflightClosed(t *testing.T) {
	const faultDefinitionID = "temporal.nexus.caller-closure.fault.duplicate-delivery-observation"
	fixture := newNexusFixture(
		t,
		"caller-closure-duplicate-delivery-input-set",
		faultDefinitionID,
	)
	factory := &countingFactory{}

	require.Equal(t,
		"umpire.artifact-set.2a6c3ef5fbd3b7dfba1acbe2c9ffc5ec3072b19daf50d3d63bd16b122fc2bd68",
		fixture.set.Identity(),
	)
	require.Equal(t,
		"sha256:3ddabf041e499ee0b7e970cac3900b8d6306ec9009e92924ef7b9ea0f584a5f8",
		fixture.set.Checksum(),
	)
	require.Equal(t,
		"sha256:09091758defd5ce50cc9acbba23a5c8499da4eef9b6e36878ac989ddea87fedf",
		fixture.experiment.ArtifactChecksum,
	)
	require.Equal(t,
		"sha256:440c0632b911571e4efb34c96fb4c4c7096fbd52f23900ed4784e037370063cf",
		fixture.runtimeConfiguration.ArtifactChecksum,
	)
	require.Equal(t,
		"temporal.system.nexus.caller-closure.duplicate-delivery.profile",
		fixture.runtimeConfiguration.Observation.ProfileDefinitionID,
	)
	require.Equal(t,
		"sha256:02517311485c8f87f13581d9381447ae34cb159526bdc865c1054efe2067acb8",
		fixture.runtimeConfiguration.Observation.ProfileBehaviorFingerprint,
	)
	require.Equal(t,
		"temporal.system.nexus.caller-closure.duplicate-delivery.observation-program",
		fixture.runtimeConfiguration.Observation.ProgramDefinitionID,
	)
	require.Equal(t,
		"sha256:7226f7762d3a21e7a66d460a4bf6b9d9a1d244bca847e4919cc0bc7debf432bd",
		fixture.runtimeConfiguration.Observation.ProgramBehaviorFingerprint,
	)
	require.Equal(t,
		"temporal.system.nexus.caller-closure.duplicate-delivery.mapping",
		fixture.runtimeConfiguration.Observation.MappingDefinitionID,
	)
	require.Equal(t,
		"sha256:cc5910e77e3d43f4cad56de88a68f099eea8b25bbbe0fde451a02b2afda01438",
		fixture.runtimeConfiguration.Observation.MappingBehaviorFingerprint,
	)

	request, err := umpireruntime.CheckRequest(
		fixture.set, fixture.authority, "runtime.run.duplicate-delivery", 0, 1,
	)
	require.NoError(t, err)
	require.Equal(t, fixture.set.Identity(), request.AdmittedSet().Identity())
	require.Equal(t, 0, factory.calls)

	faultDrifts := []struct {
		name   string
		faults func(artifactv2.Experiment) []artifactv2.ModelValue
	}{
		{
			name: "missing",
			faults: func(artifactv2.Experiment) []artifactv2.ModelValue {
				return []artifactv2.ModelValue{}
			},
		},
		{
			name: "wrong identity",
			faults: func(experiment artifactv2.Experiment) []artifactv2.ModelValue {
				return []artifactv2.ModelValue{{
					DefinitionID: "temporal.nexus.caller-closure.fault.other",
					Value:        experiment.Plan.LinearExtension[0].DefinitionID,
				}}
			},
		},
		{
			name: "wrong occurrence",
			faults: func(artifactv2.Experiment) []artifactv2.ModelValue {
				return []artifactv2.ModelValue{{
					DefinitionID: faultDefinitionID,
					Value:        "workflow-nexus.occurrence.other",
				}}
			},
		},
		{
			name: "duplicate",
			faults: func(experiment artifactv2.Experiment) []artifactv2.ModelValue {
				return append(
					append([]artifactv2.ModelValue{}, experiment.Plan.RequestedFaults...),
					experiment.Plan.RequestedFaults...,
				)
			},
		},
	}
	for _, drift := range faultDrifts {
		t.Run("fault "+drift.name, func(t *testing.T) {
			set := mutateExperiment(t, fixture, func(experiment *artifactv2.Experiment) {
				experiment.Plan.RequestedFaults = drift.faults(*experiment)
			})
			request, err := umpireruntime.CheckRequest(
				set,
				fixture.authority,
				"runtime.run.duplicate-delivery-fault-drift",
				0,
				1,
			)
			requirePreflightKind(t, err, umpireruntime.PreflightFault)
			require.Empty(t, request.RunIdentity())
			require.Equal(t, 0, factory.calls)
		})
	}

	normalFixture := newNexusFixture(t, "caller-closure-input-set", "")
	normalWithFault := mutateExperiment(t, normalFixture, func(experiment *artifactv2.Experiment) {
		experiment.Plan.RequestedFaults = append(
			[]artifactv2.ModelValue{},
			fixture.experiment.Plan.RequestedFaults...,
		)
	})
	observationDrifts := []struct {
		name   string
		mutate func(*artifactv2.ObservationConfiguration)
	}{
		{
			name: "profile identity",
			mutate: func(observation *artifactv2.ObservationConfiguration) {
				observation.ProfileDefinitionID =
					"temporal.system.nexus.caller-closure.profile.other"
			},
		},
		{
			name: "profile fingerprint",
			mutate: func(observation *artifactv2.ObservationConfiguration) {
				observation.ProfileBehaviorFingerprint =
					"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			},
		},
		{
			name: "program identity",
			mutate: func(observation *artifactv2.ObservationConfiguration) {
				observation.ProgramDefinitionID =
					"temporal.system.nexus.caller-closure.observation-program.other"
			},
		},
		{
			name: "program fingerprint",
			mutate: func(observation *artifactv2.ObservationConfiguration) {
				observation.ProgramBehaviorFingerprint =
					"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			},
		},
		{
			name: "mapping identity",
			mutate: func(observation *artifactv2.ObservationConfiguration) {
				observation.MappingDefinitionID =
					"temporal.system.nexus.caller-closure.mapping.other"
			},
		},
		{
			name: "mapping fingerprint",
			mutate: func(observation *artifactv2.ObservationConfiguration) {
				observation.MappingBehaviorFingerprint =
					"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
			},
		},
	}
	for _, drift := range observationDrifts {
		t.Run("observation "+drift.name+" drift", func(t *testing.T) {
			set := mutateConfiguration(t, fixture, func(configuration *artifactv2.RuntimeConfiguration) {
				drift.mutate(&configuration.Observation)
			})
			request, err := umpireruntime.CheckRequest(
				set,
				fixture.authority,
				"runtime.run.duplicate-delivery-observation-drift",
				0,
				1,
			)
			requirePreflightKind(t, err, umpireruntime.PreflightConfiguration)
			require.Empty(t, request.RunIdentity())
			require.Equal(t, 0, factory.calls)
		})
	}
	budgetDrift := mutateConfiguration(t, fixture, func(configuration *artifactv2.RuntimeConfiguration) {
		configuration.PhaseLimits[0].DurationMilliseconds = artifactv2.Natural("30001")
	})
	budgetAuthority := authorityForSet(t, budgetDrift, fixture.program)
	observationProgram := newObservationProgram(t, fixture.runtimeConfiguration.Observation)
	capabilityProgram, err := umpireruntime.NewProgramWithRequestedFault(
		fixture.program.DefinitionID(),
		fixture.program.Version(),
		fixture.program.BehaviorFingerprint(),
		fixture.program.TargetDefinitionIDs(),
		fixture.program.ActionDefinitionIDs(),
		fixture.program.Occurrences(),
		observationProgram,
		faultDefinitionID,
		fixture.program.CapabilityDefinitionIDs()[1:],
	)
	require.NoError(t, err)
	capabilityAuthority := authorityForSet(t, fixture.set, capabilityProgram)
	programDrift, err := umpireruntime.NewProgramWithRequestedFault(
		"temporal.nexus.participant-program.caller-closure-other",
		fixture.program.Version(),
		fixture.program.BehaviorFingerprint(),
		fixture.program.TargetDefinitionIDs(),
		fixture.program.ActionDefinitionIDs(),
		fixture.program.Occurrences(),
		observationProgram,
		faultDefinitionID,
		fixture.program.CapabilityDefinitionIDs(),
	)
	require.NoError(t, err)
	programAuthority := authorityForSet(t, fixture.set, programDrift)
	protocolAuthority, err := local.NewAuthority(
		fixture.runtimeConfiguration.ConfigurationDefinitionID,
		fixture.runtimeConfiguration.BehaviorFingerprint,
		fixture.runtimeConfiguration.ParticipantBindings[0].ParticipantDefinitionID,
		"umpire.participant-protocol.other",
		fixture.program,
	)
	require.NoError(t, err)
	profileAuthority, err := umpireruntime.NewAuthority(
		"temporal.runtime-profile.other",
		local.ProfileVersion,
		local.ProfileBehaviorFingerprint,
		fixture.runtimeConfiguration.ConfigurationDefinitionID,
		fixture.runtimeConfiguration.BehaviorFingerprint,
		local.RequiredCapabilityDefinitionIDs(),
		[]string{},
		umpireruntime.CanonicalPhaseLimits(),
		0,
		1,
		fixture.runtimeConfiguration.ParticipantBindings[0].ParticipantDefinitionID,
		fixture.runtimeConfiguration.ParticipantBindings[0].ProtocolDefinitionID,
		2,
		1,
		1,
		fixture.program,
	)
	require.NoError(t, err)

	tests := []struct {
		name      string
		kind      umpireruntime.PreflightErrorKind
		set       artifact.AdmittedSet
		authority umpireruntime.Authority
		seed      uint64
		attempt   uint64
	}{
		{
			name: "normal program rejects fault", kind: umpireruntime.PreflightFault,
			set: normalWithFault, authority: normalFixture.authority, attempt: 1,
		},
		{
			name: "normal configuration crossing", kind: umpireruntime.PreflightConfiguration,
			set: normalFixture.set, authority: fixture.authority, attempt: 1,
		},
		{
			name: "faulted configuration crossing", kind: umpireruntime.PreflightConfiguration,
			set: fixture.set, authority: normalFixture.authority, attempt: 1,
		},
		{
			name: "profile drift", kind: umpireruntime.PreflightProfile,
			set: fixture.set, authority: profileAuthority, attempt: 1,
		},
		{
			name: "program drift", kind: umpireruntime.PreflightParticipant,
			set: fixture.set, authority: programAuthority, attempt: 1,
		},
		{
			name: "protocol drift", kind: umpireruntime.PreflightProtocol,
			set: fixture.set, authority: protocolAuthority, attempt: 1,
		},
		{
			name: "capability drift", kind: umpireruntime.PreflightCapability,
			set: fixture.set, authority: capabilityAuthority, attempt: 1,
		},
		{
			name: "budget drift", kind: umpireruntime.PreflightBudget,
			set: budgetDrift, authority: budgetAuthority, attempt: 1,
		},
		{
			name: "seed drift", kind: umpireruntime.PreflightSeed,
			set: fixture.set, authority: fixture.authority, seed: 1, attempt: 1,
		},
		{
			name: "attempt drift", kind: umpireruntime.PreflightAttempt,
			set: fixture.set, authority: fixture.authority, attempt: 2,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := factory.calls
			request, err := umpireruntime.CheckRequest(
				test.set,
				test.authority,
				"runtime.run.duplicate-delivery-rejected",
				test.seed,
				test.attempt,
			)
			requirePreflightKind(t, err, test.kind)
			require.Empty(t, request.RunIdentity())
			require.Equal(t, before, factory.calls)
		})
	}
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
			name: "fault",
			kind: umpireruntime.PreflightFault,
			run: func(t *testing.T) (umpireruntime.CheckedRunRequest, error) {
				set := mutateExperiment(t, fixture, func(experiment *artifactv2.Experiment) {
					experiment.Plan.RequestedFaults = []artifactv2.ModelValue{{
						DefinitionID: "switch.fault.requested",
						Value:        "enabled",
					}}
				})
				return umpireruntime.CheckRequest(
					set, fixture.authority, "runtime.run.fault-requested", 0, 1,
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
	request := mustCheckedRequest(t, fixture)
	command, ok := request.Command(umpireruntime.CommandPrepare)
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
	require.False(t, receipt.ControlAttempted())

	realizationCommand, ok := request.Command(umpireruntime.CommandRealize)
	require.True(t, ok)
	controlReceipt, err := umpireruntime.NewControlReceipt(
		realizationCommand,
		umpireruntime.ReceiptAccepted,
		[]umpireruntime.Fact{},
		[]umpireruntime.Resource{},
		[]umpireruntime.Resource{},
	)
	require.NoError(t, err)
	require.True(t, controlReceipt.ControlAttempted())
	_, err = umpireruntime.NewControlReceipt(
		command,
		umpireruntime.ReceiptAccepted,
		[]umpireruntime.Fact{},
		[]umpireruntime.Resource{},
		[]umpireruntime.Resource{},
	)
	require.Error(t, err)

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

func newNexusFixture(
	t *testing.T,
	directory string,
	requestedFaultDefinitionID string,
) checkedFixture {
	t.Helper()
	root := filepath.Join("..", "temporal", "nexus", "testdata", directory)
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
	set, err := artifact.AdmitSetFiles(files)
	require.NoError(t, err)
	executable, ok := set.Executable()
	require.True(t, ok)
	experiment := executable.Experiment()
	configuration := executable.RuntimeConfiguration()
	require.Len(t, configuration.ParticipantBindings, 1)
	binding := configuration.ParticipantBindings[0]
	occurrence, err := umpireruntime.NewOccurrence(
		experiment.Plan.LinearExtension[0].DefinitionID,
		experiment.Plan.LinearExtension[0].ActionDefinitionID,
		1,
	)
	require.NoError(t, err)
	var program umpireruntime.Program
	if requestedFaultDefinitionID == "" {
		program, err = umpireruntime.NewProgram(
			binding.ProgramDefinitionID,
			1,
			binding.ProgramBehaviorFingerprint,
			[]string{experiment.Plan.TargetDefinitionID},
			[]string{experiment.Plan.RequestedActions[0].DefinitionID},
			[]umpireruntime.Occurrence{occurrence},
			binding.CapabilityDefinitionIDs,
		)
	} else {
		observationProgram := newObservationProgram(t, configuration.Observation)
		program, err = umpireruntime.NewProgramWithRequestedFault(
			binding.ProgramDefinitionID,
			1,
			binding.ProgramBehaviorFingerprint,
			[]string{experiment.Plan.TargetDefinitionID},
			[]string{experiment.Plan.RequestedActions[0].DefinitionID},
			[]umpireruntime.Occurrence{occurrence},
			observationProgram,
			requestedFaultDefinitionID,
			binding.CapabilityDefinitionIDs,
		)
	}
	require.NoError(t, err)
	authority := authorityForSet(t, set, program)
	return checkedFixture{
		set:                  set,
		experiment:           experiment,
		runtimeConfiguration: configuration,
		program:              program,
		authority:            authority,
	}
}

func newObservationProgram(
	t *testing.T,
	configuration artifactv2.ObservationConfiguration,
) umpireruntime.ObservationProgram {
	t.Helper()
	program, err := umpireruntime.NewObservationProgram(
		configuration.ProfileDefinitionID,
		configuration.ProfileBehaviorFingerprint,
		configuration.ProgramDefinitionID,
		configuration.ProgramBehaviorFingerprint,
		configuration.MappingDefinitionID,
		configuration.MappingBehaviorFingerprint,
	)
	require.NoError(t, err)
	return program
}

func authorityForSet(
	t *testing.T,
	set artifact.AdmittedSet,
	program umpireruntime.Program,
) umpireruntime.Authority {
	t.Helper()
	executable, ok := set.Executable()
	require.True(t, ok)
	configuration := executable.RuntimeConfiguration()
	require.Len(t, configuration.ParticipantBindings, 1)
	binding := configuration.ParticipantBindings[0]
	authority, err := local.NewAuthority(
		configuration.ConfigurationDefinitionID,
		configuration.BehaviorFingerprint,
		binding.ParticipantDefinitionID,
		binding.ProtocolDefinitionID,
		program,
	)
	require.NoError(t, err)
	return authority
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

func mutateExperiment(
	t *testing.T,
	fixture checkedFixture,
	mutate func(*artifactv2.Experiment),
) artifact.AdmittedSet {
	t.Helper()
	experiment := fixture.experiment
	experiment.Plan.RequestedFaults = append([]artifactv2.ModelValue{}, experiment.Plan.RequestedFaults...)
	mutate(&experiment)
	experiment, err := artifactv2.SealExperiment(experiment)
	require.NoError(t, err)
	experimentBytes, err := artifactv2.CanonicalExperimentBytes(experiment)
	require.NoError(t, err)
	experimentBinding, err := artifactv2.ExperimentArtifactBinding(experiment)
	require.NoError(t, err)

	configuration := fixture.runtimeConfiguration
	configuration.Experiment = experimentBinding
	configuration, err = artifactv2.SealRuntimeConfiguration(configuration)
	require.NoError(t, err)
	configurationBytes, err := artifactv2.CanonicalRuntimeConfigurationBytes(configuration)
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
