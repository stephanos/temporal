package testpilot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/lower"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestModelCasesLowerForExistingConsumers(t *testing.T) {
	for _, item := range []struct{ model, family, owner, set, query string }{
		{"activity", "temporal.features.standaloneactivity.system", "activityProtocol", "standaloneActivityTests", "completion"},
		{"activity", "temporal.features.standaloneactivity.system", "activityProtocol", "standaloneActivityTests", "retry"},
		{"activity", "temporal.features.standaloneactivity.system", "activityProtocol", "standaloneActivityTests", "pauseResume"},
		{"nexus-caller", "temporal.features.nexuscaller.system", "nexusProtocol", "nexusCallerTests", "syncCompletion"},
		{"nexus-caller", "temporal.features.nexuscaller.system", "nexusProtocol", "nexusCallerTests", "asyncCompletion"},
	} {
		t.Run(item.query, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "model", "ir", item.model+".json")
			key := umpiremodel.ClaimKey{Family: item.family, Owner: item.owner, Name: item.query}
			fixture, err := LoadModelCase(path, key, item.set)
			require.NoError(t, err)
			again, err := LoadModelCase(path, key, item.set)
			require.NoError(t, err)
			require.Equal(t, fixture.Bytes, again.Bytes)
			decoded, err := testpilot.DecodeCaseProtoJSON(fixture.Bytes)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, fixture.Source, decoded)
			catalog, err := temporal.NewWorkflowServiceCatalog()
			require.NoError(t, err)
			profile, err := temporal.DeriveProfile(fixture.Source, catalog, temporal.Environment{Identity: "scala", Namespace: "ns", TaskQueue: "q", HandlerTaskQueue: "h", NexusEndpoint: "e"})
			require.NoError(t, err)
			prepared, err := testpilot.Prepare(fixture.Source, profile)
			require.NoError(t, err)
			_, err = prepared.WithAssessment(fixture.Assessment)
			require.NoError(t, err)
		})
	}
}

// The standalone Nexus operation's generated Cases require the feature's flag of the environment, so
// a Profile derived from a server without it is refused at preparation, naming the setting, the
// value it must take and that it is unset; one derived from a server that sets it admits them.
func TestGeneratedNexusOperationCasesRequireTheStandaloneFlag(t *testing.T) {
	directory := filepath.Join("..", "..", "..", "model", "cases")
	entries, err := GeneratedCases(directory)
	require.NoError(t, err)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	checked := 0
	for _, entry := range entries {
		if entry.Model != "nexus-operation.json" || entry.Standing != lower.Lowered {
			continue
		}
		checked++
		t.Run(entry.File, func(t *testing.T) {
			fixture, err := LoadGeneratedCase(directory, entry)
			require.NoError(t, err)
			settings := fixture.Source.GetProgram().GetRequiredSettings()
			require.Len(t, settings, 1, "the Case carries its realization's required settings")
			protorequire.ProtoEqual(t, &testpilotspb.RequiredSetting{Key: "nexusoperation.enableStandalone", Value: "true"}, settings[0])
			environment := temporal.Environment{Identity: "scala", Namespace: "ns", TaskQueue: "q", NexusEndpoint: "e",
				DynamicConfig: map[string]string{"history.enableChasm": "true"}}
			if temporal.HandlerTaskQueueBindingID(fixture.Source.GetProgram()) != "" {
				environment.HandlerTaskQueue = "h"
			}
			profile, err := temporal.DeriveProfile(fixture.Source, catalog, environment)
			require.NoError(t, err)
			_, err = testpilot.Prepare(fixture.Source, profile)
			var refusal *testpilot.PreparationError
			require.ErrorAs(t, err, &refusal)
			require.Equal(t, testpilot.PreparationUnavailable, refusal.Category)
			require.Equal(t, "program.required_settings[0]", refusal.Path)
			require.Contains(t, refusal.Detail, `"nexusoperation.enableStandalone" must be "true"`)
			require.Contains(t, refusal.Detail, "leaves it unset")

			environment.DynamicConfig["nexusoperation.enableStandalone"] = "false"
			profile, err = temporal.DeriveProfile(fixture.Source, catalog, environment)
			require.NoError(t, err)
			_, err = testpilot.Prepare(fixture.Source, profile)
			require.ErrorAs(t, err, &refusal)
			require.Contains(t, refusal.Detail, `sets it to "false"`)

			environment.DynamicConfig["nexusoperation.enableStandalone"] = "true"
			profile, err = temporal.DeriveProfile(fixture.Source, catalog, environment)
			require.NoError(t, err)
			prepared, err := testpilot.Prepare(fixture.Source, profile)
			require.NoError(t, err)
			_, err = prepared.WithAssessment(fixture.Assessment)
			require.NoError(t, err)
		})
	}
	require.Equal(t, 2, checked, "both standalone Nexus operation Cases are generated")
}

// The held race lowers only for an environment that supplies a delivery control, names the claim a
// Run of it is assessed for, and names its durable kinds, so a harness can take the commit evidence
// out of a recorded Run without knowing the Case.
func TestTheHeldRaceIsLoadedWithItsClaimAndItsDurableKinds(t *testing.T) {
	fixture, err := LoadModelCase(filepath.Join("..", "..", "..", "model", "ir", "activity-race.json"),
		umpiremodel.ClaimKey{Family: "temporal.features.standaloneactivity.system", Owner: "heldAdmission", Name: "heldAdmission.staleDelivery"}, "standaloneActivityRace")
	require.NoError(t, err)
	require.Equal(t, "staleDeliveryRejected", fixture.Property)
	require.Equal(t, []string{"evidence.admissionRejected", "evidence.attemptAdmitted"}, fixture.Durable)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	environment := temporal.Environment{Identity: "scala", Namespace: "ns", TaskQueue: "q"}
	profile, err := temporal.DeriveProfile(fixture.Source, catalog, environment)
	require.NoError(t, err)
	_, err = testpilot.Prepare(fixture.Source, profile)
	require.ErrorContains(t, err, "unsupported at controller.hold-dispatch")
	environment.DeliveryControl = true
	profile, err = temporal.DeriveProfile(fixture.Source, catalog, environment)
	require.NoError(t, err)
	prepared, err := testpilot.Prepare(fixture.Source, profile)
	require.NoError(t, err)
	_, err = prepared.WithAssessment(fixture.Assessment)
	require.NoError(t, err)

	evidence := func(kind string) *testpilotspb.ObservationResult {
		packed, err := anypb.New(&testpilotspb.CorrelatedEvidence{Kind: kind})
		require.NoError(t, err)
		return &testpilotspb.ObservationResult{ObservationId: "correlated-evidence", Value: &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: packed}}}
	}
	admission := &testpilotspb.DeliveryAdmission{ActivityId: "a", DeliveryId: "1", Decision: testpilotspb.DELIVERY_ADMISSION_DECISION_REJECTED}
	run := &testpilotspb.Run{Events: []*testpilotspb.RunEvent{
		{Sequence: 1, Observations: []*testpilotspb.ObservationResult{evidence("evidence.statusPaused")}},
		{Sequence: 2, Observations: []*testpilotspb.ObservationResult{evidence("evidence.admissionRejected")},
			Payload: &testpilotspb.RunEvent_Outcome{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, DeliveryAdmission: admission}}},
	}}
	recorded := proto.CloneOf(run)
	stripped, err := fixture.WithoutDurableEvidence(run)
	require.NoError(t, err)
	protorequire.ProtoEqual(t, recorded, run)
	protorequire.ProtoEqual(t, &testpilotspb.Run{Events: []*testpilotspb.RunEvent{
		recorded.GetEvents()[0],
		{Sequence: 2, Payload: &testpilotspb.RunEvent_Outcome{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}},
	}}, stripped)
}

func TestDurableEvidenceIsScopedToTheSelectedRealization(t *testing.T) {
	model := &umpirespb.Model{Realizations: []*umpirespb.Realization{
		{Machine: "reported", Evidence: []*umpirespb.Evidence{{Id: "shared", Commitment: umpirespb.Evidence_COMMITMENT_REPORTED}}},
		{Machine: "durable", Evidence: []*umpirespb.Evidence{{Id: "shared", Commitment: umpirespb.Evidence_COMMITMENT_DURABLE}}},
	}}
	source := &testpilotspb.Case{
		Provenance: &testpilotspb.CaseProvenance{LocalNames: []*testpilotspb.LocalName{{DefinitionId: "shared", LocalName: "local"}}},
		Program:    &testpilotspb.Program{Evidence: []*testpilotspb.EvidenceDeclaration{{EvidenceId: "local"}}},
	}
	require.Empty(t, durableEvidence(model, "reported", source))
	require.Equal(t, []string{"local"}, durableEvidence(model, "durable", source))
}
