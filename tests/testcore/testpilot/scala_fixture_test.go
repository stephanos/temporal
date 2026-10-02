package testpilot

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/model/scalav2/goir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func TestScalaCasesLowerForExistingConsumers(t *testing.T) {
	for _, item := range []struct{ model, family, owner, set, query string }{
		{"activity", "temporal.activity.standalone", "activityProtocol", "standaloneActivityTests", "completion"},
		{"activity", "temporal.activity.standalone", "activityProtocol", "standaloneActivityTests", "retry"},
		{"activity", "temporal.activity.standalone", "activityProtocol", "standaloneActivityTests", "pauseResume"},
		{"nexus-caller", "temporal.nexus.caller", "nexusProtocol", "nexusCallerTests", "syncCompletion"},
		{"nexus-caller", "temporal.nexus.caller", "nexusProtocol", "nexusCallerTests", "asyncCompletion"},
	} {
		t.Run(item.query, func(t *testing.T) {
			path := filepath.Join("..", "..", "..", "model", "scalav2", "ir", item.model+".json")
			key := goir.ClaimKey{Family: item.family, Owner: item.owner, Name: item.query}
			fixture, err := LoadScalaCase(path, key, item.set)
			require.NoError(t, err)
			again, err := LoadScalaCase(path, key, item.set)
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

// The held race lowers only for an environment that supplies a delivery control, names the claim a
// Run of it is assessed for, and names its durable kinds, so a harness can take the commit evidence
// out of a recorded Run without knowing the Case.
func TestTheHeldRaceIsLoadedWithItsClaimAndItsDurableKinds(t *testing.T) {
	fixture, err := LoadScalaCase(filepath.Join("..", "..", "..", "model", "scalav2", "ir", "activity-race.json"),
		goir.ClaimKey{Family: "temporal.activity.standalone.system", Owner: "heldAdmission", Name: "heldAdmission.staleDelivery"}, "standaloneActivityRace")
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
	model := &modelirspb.Model{Realizations: []*modelirspb.Realization{
		{Machine: "reported", Evidence: []*modelirspb.Evidence{{Id: "shared", Commitment: modelirspb.Evidence_COMMITMENT_REPORTED}}},
		{Machine: "durable", Evidence: []*modelirspb.Evidence{{Id: "shared", Commitment: modelirspb.Evidence_COMMITMENT_DURABLE}}},
	}}
	source := &testpilotspb.Case{
		Provenance: &testpilotspb.CaseProvenance{LocalNames: []*testpilotspb.LocalName{{DefinitionId: "shared", LocalName: "local"}}},
		Program:    &testpilotspb.Program{Evidence: []*testpilotspb.EvidenceDeclaration{{EvidenceId: "local"}}},
	}
	require.Empty(t, durableEvidence(model, "reported", source))
	require.Equal(t, []string{"local"}, durableEvidence(model, "durable", source))
}
