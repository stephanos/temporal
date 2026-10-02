package lower_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/umpire/internal/golden"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
)

const (
	jobFamily   = "fixture.job"
	jobEvidence = jobFamily + ".evidence."
	jobSource   = jobFamily + ".source."
)

// jobTable is the job's machine. A submitted job is listed. A worker takes it, which records that it
// was taken and its attempt count; a failed attempt lists it again with the count; a finished one
// closes it. A job may also be dropped while it is queued, which records that it was dropped. A
// running job may be checked, which records nothing, and closed, which finishes it or drops it.
func jobTable(t *testing.T) *umpiremodel.Table {
	t.Helper()
	model := golden.JobModel("once", jobOnce...)
	require.NoError(t, umpiremodel.Validate(model))
	built, err := umpiremodel.Build(model)
	require.NoError(t, err)
	require.Len(t, built, 1)
	return built["job"].Table
}

// jobQuery finds the job finished by the last step of a path of these classes.
func jobQuery(t *testing.T, name string, actions ...string) *umpiremodel.Query {
	t.Helper()
	model := golden.JobModel(name, actions...)
	table := jobTable(t)
	realizer, err := umpiremodel.NewRealizer(model, umpiremodel.DefaultScope)
	require.NoError(t, err)
	require.Equal(t, table.IDs(), realizer.Machine("job").Table.IDs())
	query, err := realizer.Find(umpiremodel.ClaimKey{Family: jobFamily, Owner: "job", Name: name})
	require.NoError(t, err)
	return query
}

var (
	jobOnce    = []string{"submit", "take", "finish"}
	jobRetried = []string{"submit", "take", "fail", "settle", "take", "finish"}
)

func projected(path string) *testpilotspb.Expression { return cp.Path(cp.ProjectedValue(), path) }

// attemptIs is the guard that takes the record of one numbered attempt.
func attemptIs(number int64) *testpilotspb.Expression {
	return cp.Equal(projected("activity_attempt.sdk_attempt"), cp.Literal(cp.SignedInteger(number)))
}

func attemptRecord(guard *testpilotspb.Expression) cp.Recorded {
	return cp.Recorded{RunEvent: &cp.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, EntrypointID: "controller",
		InstructionID: "submit-job", RunKeyed: true, Guard: guard}}
}

var attemptFields = []cp.EvidenceField{
	{ID: "attempt", Path: "activity_attempt.sdk_attempt", Type: testpilotspb.SCALAR_KIND_UINT64, Role: "attempt"},
	{ID: "delivery", Path: "activity_attempt.delivery_id", Type: testpilotspb.SCALAR_KIND_TEXT, Role: "delivery"},
}

func jobKind(records string, recorded cp.Recorded, key string) *cp.EvidenceSource {
	return &cp.EvidenceSource{EventKind: records, Recorded: recorded, OperationKeyPath: key, KindID: jobEvidence + records,
		SourceID: jobSource + records}
}

// jobSources is the job's evidence. The listing and the closing status are read back. That a worker
// took the job is the Run's record of its first attempt; the record of its second attempt is what
// shows the failed attempt listed the job again and a worker took it again, so that one kind confirms
// both of those steps, and each of the two takes has a kind of its own.
func jobSources() []*cp.EvidenceSource {
	taken := jobKind("taken", attemptRecord(attemptIs(1)), "")
	taken.Fields, taken.Confirms = attemptFields, []cp.Taking{{Key: "take", Occurrence: 1}}
	again := jobKind("count", attemptRecord(attemptIs(2)), "")
	again.Fields, again.Confirms = attemptFields, []cp.Taking{{Key: "fail", Occurrence: 1}, {Key: "take", Occurrence: 2}}
	return []*cp.EvidenceSource{
		jobKind("listed", cp.Recorded{Method: "/fixture.Jobs/List", Path: "jobs"}, "job_id"),
		taken, again,
		jobKind("finished", cp.Recorded{Method: "/fixture.Jobs/Describe", Path: "job", Single: true}, "job_id"),
		jobKind("wasDropped", cp.Recorded{HistoryAttributes: "job_dropped_event_attributes"}, "attributes<job_dropped_event_attributes>.job_id"),
	}
}

func jobRealization(t *testing.T, sources []*cp.EvidenceSource) *cp.Realization {
	t.Helper()
	submit := jobTable(t).ActionAtom("submit").ID
	return &cp.Realization{ProducerID: "fixture.job.testpilot", ProducerVersion: "1", ProjectionID: jobFamily + ".projection",
		ScopeField: jobFamily + ".scope.run", OperationKey: jobFamily + ".scope.job", CorrelatedObservation: "correlated-evidence",
		Sources: sources, ProjectionLimits: cp.ProjectionLimits{Events: 8, Buffered: 8, Keys: 2, Support: 16, Work: 1000, EventSize: 512},
		Actions: []cp.ActionBinding{{Action: submit, Key: "submit", InstructionID: "submit-job",
			Node: func(_ cp.Placement, id string) *testpilotspb.InstructionNode {
				return cp.Node(id, cp.InvokeRPC("jobs", "/fixture.Jobs/Submit", nil, nil))
			}}},
		Plan: cp.ProgramPlan{
			Observations: []*testpilotspb.Observation{cp.MessageObservation("correlated-evidence", "temporal.server.api.testpilot.v1.CorrelatedEvidence")},
			Entrypoints: []cp.EntrypointPlan{{
				Items: []cp.Item{cp.Actions{Classes: []string{submit}}, cp.Fixed{Node: func(_ cp.Placement, rules []cp.EvidenceRule) *testpilotspb.InstructionNode {
					return cp.Node("history", cp.InvokeRPC("jobs", "/fixture.Jobs/History", nil, []*testpilotspb.ResponseRead{
						cp.ResponseRead("events[*]", testpilotspb.READ_CARDINALITY_EMIT_EACH, cp.EvidenceTarget("correlated-evidence", rules))}))
				}}},
				Activate: func(_ cp.Placement, nodes []*testpilotspb.InstructionNode) *testpilotspb.Entrypoint {
					return &testpilotspb.Entrypoint{EntrypointId: "controller", Instructions: nodes,
						Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}
				}}}}}
}

func jobIdentity(name string) cp.Identity { return cp.IdentityFor("fixture.case", "jobs", name) }

func produced(t *testing.T, name string, sources []*cp.EvidenceSource, actions ...string) *testpilotspb.Case {
	t.Helper()
	c, err := cp.Produce(jobQuery(t, name, actions...), jobIdentity(name), jobRealization(t, sources), cp.Source{Path: "job.go", Provenance: "test"})
	require.NoError(t, err)
	return c
}
