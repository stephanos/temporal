package conformance

import (
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/scalav2/goir"
	goirtestpilot "go.temporal.io/server/model/scalav2/goir/testpilot"
	"google.golang.org/protobuf/proto"
)

const (
	nexusFamily   = "temporal.nexus.caller"
	nexusMachine  = "nexusProtocol"
	nexusEvidence = "temporal.nexus.caller.evidence."
)

func nexusModel(t testing.TB) *modelirspb.Model {
	t.Helper()
	m, err := goir.Load(filepath.Join("..", "..", "ir", "nexus-caller.json"))
	require.NoError(t, err)
	return m
}

// loweredNexus is one functional Query of the Nexus caller Model lowered to its Case, prepared as a
// black-box consumer prepares it, with the Query's assessment bound to it.
func loweredNexus(t testing.TB, m *modelirspb.Model, query string, limits Limits) *bound {
	t.Helper()
	producer, err := goirtestpilot.NewProducer(m)
	require.NoError(t, err)
	lowering, err := producer.Lower(query, cp.IdentityFor("temporal.case", "nexusCallerTests", query))
	require.NoError(t, err)
	require.Equal(t, goirtestpilot.Lowered, lowering.Standing)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(lowering.Case, catalog, temporal.Environment{Identity: query + "-profile", Namespace: "namespace",
		TaskQueue: "task-queue", HandlerTaskQueue: "task-queue-handler", NexusEndpoint: "nexus-endpoint"})
	require.NoError(t, err)
	plain, err := testpilot.Prepare(lowering.Case, profile)
	require.NoError(t, err)
	factory, err := Prepare(m, goir.ClaimKey{Family: nexusFamily, Owner: nexusMachine, Name: query}, lowering.Case, limits)
	require.NoError(t, err)
	assessed, err := plain.WithAssessment(factory)
	require.NoError(t, err)
	return &bound{source: lowering.Case, plain: plain, assessed: assessed, factory: factory}
}

// nexusEvidenceOf is the evidence a Run of a lowered Case would lift for one operation, one piece per
// named kind in order: each under the Case's own name, source and scope for the kind, at the next
// ordinal of its source, and after the piece before it when that came from another source, which is
// how Testpilot's executor chains the evidence one operation's instructions lift.
func nexusEvidenceOf(t testing.TB, source *testpilotspb.Case, kinds ...string) []*testpilotspb.CorrelatedEvidence {
	t.Helper()
	local := map[string]string{}
	for _, name := range source.GetProvenance().GetLocalNames() {
		local[name.GetDefinitionId()] = name.GetLocalName()
	}
	ordinals := map[string]int64{}
	var out []*testpilotspb.CorrelatedEvidence
	for _, kind := range kinds {
		var declared *testpilotspb.EvidenceDeclaration
		for _, e := range source.GetProgram().GetEvidence() {
			if e.GetEvidenceId() == local[nexusEvidence+kind] {
				declared = e
			}
		}
		require.NotNil(t, declared, "the Case carries %s", kind)
		evidence := &testpilotspb.CorrelatedEvidence{Kind: declared.GetEvidenceId(), Operation: "5", Identity: &testpilotspb.CorrelatedIdentity{
			EvidenceSource: declared.GetEvidenceSource(), Ordinal: ordinals[declared.GetEvidenceSource()], Scope: declared.GetScope()}}
		ordinals[declared.GetEvidenceSource()]++
		if len(out) > 0 && out[len(out)-1].GetIdentity().GetEvidenceSource() != declared.GetEvidenceSource() {
			evidence.Parents = append(evidence.Parents, proto.CloneOf(out[len(out)-1].GetIdentity()))
		}
		out = append(out, evidence)
	}
	return out
}

// constructedRun writes out a closed Run of a Case that recorded these pieces of evidence, one per
// Run Event, as the recorder leaves it: complete, or, when the Contract's own evaluation fails on
// the event at failure, with that coordinate recorded and the Run incomplete from the next event on.
func constructedRun(t testing.TB, source *testpilotspb.Case, evidence []*testpilotspb.CorrelatedEvidence, failure int64) *testpilotspb.Run {
	t.Helper()
	run := &testpilotspb.Run{RunId: testpilot.RunIDPrefix + "00000000-0000-4000-8000-000000000001", CaseId: source.GetCaseId(),
		ProgramId: source.GetProgram().GetProgramId(), Disposition: testpilotspb.RUN_DISPOSITION_COMPLETED,
		Cleanup: &testpilotspb.CleanupOutcome{Status: testpilotspb.CLEANUP_STATUS_SUCCEEDED}}
	record := func(event *testpilotspb.RunEvent) {
		event.Sequence = int64(len(run.GetEvents())) + 1
		event.ElapsedMilliseconds = event.GetSequence() - 1
		event.ExecutionIncomplete = failure > 0 && event.GetSequence() > failure
		run.Events = append(run.Events, event)
	}
	record(&testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_RUN_OPENED, SourceId: "opened"})
	for i, piece := range evidence {
		record(&testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, SourceId: "history.read." + strconv.Itoa(i),
			Coordinates:  &testpilotspb.RunEventCoordinates{EntrypointId: "controller", ActivationId: "controller", InstructionId: "history", Attempt: 1, EmittedIndex: int64(i)},
			Observations: []*testpilotspb.ObservationResult{{ObservationId: evidenceObservation, Value: evidenceValue(t, piece)}}})
	}
	record(&testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_RUN_CLOSED, SourceId: "closed"})
	if failure > 0 {
		run.Disposition = testpilotspb.RUN_DISPOSITION_INCOMPLETE
		run.EvaluationFailure = &testpilotspb.Run_EvaluationFailureSequence{EvaluationFailureSequence: failure}
	}
	return run
}

func nexusInstance(query string) string { return `run="nexusCallerTests-` + query + `";5` }

// Each of the seven lowered Cases is replayed on a constructed Run that records exactly its
// witness's evidence. The Contract reads that evidence as the witness, step by step, and is
// satisfied. The assessment keeps every execution of nexusProtocol that explains it, and each
// expectation below is read off model/scalav2/scala/temporal/nexuscaller (Claims.scala and the
// kernel's step functions), which the comment beside it cites.
func TestAWitnessRunConformsWhileItsPropertyStaysOpen(t *testing.T) {
	m := nexusModel(t)
	for _, test := range []struct {
		query, property string
		kinds           []string
		why             string
	}{
		// A completed event is also what a completion of an operation the handler never answered
		// synchronously records (completeStep), and the Case does not carry the started event that would
		// tell them apart: on that execution no synchronous reply is taken.
		{"syncCompletion", "syncSucceeds", []string{"scheduled", "completed"}, whyNeverRead},
		// A completion that arrives after the operation is over is not found and records nothing
		// (completeStep, terminalPhase), so no evidence excludes one, and the claim, which is about every
		// succeeded completion, fails on it.
		{"asyncCompletion", "completionSucceeds", []string{"scheduled", "started", "completed"}, whyDisagreement},
		{"asyncFailure", "completionFails", []string{"scheduled", "started", "failed"}, whyDisagreement},
		// A failed event is also what a failed reply and a failed completion record.
		{"handlerError", "handlerErrorFails", []string{"scheduled", "failed"}, whyNeverRead},
		// The history-sensitive one. The claim fixes the attempt count at one. The pending-attempts
		// evidence says a retryable failure happened and not how many: a second one, unreported, leaves
		// the count at two, and the same reply then lands in another state.
		{"retry", "retrySucceeds", []string{"scheduled", "pendingAttempts", "completed"}, whyDisagreement},
		// A timed-out event is recorded by each of the three deadlines, and the scheduled event does not
		// say which of them the command set.
		{"scheduleToStartTimeout", "scheduleToStartFires", []string{"scheduled", "timedOut"}, whyNeverRead},
		{"startToCloseTimeout", "startToCloseFires", []string{"scheduled", "started", "timedOut"}, whyNeverRead},
	} {
		t.Run(test.query, func(t *testing.T) {
			b := loweredNexus(t, m, test.query, generous)
			run := constructedRun(t, b.source, nexusEvidenceOf(t, b.source, test.kinds...), 0)
			var support []int64
			for i := range test.kinds {
				support = append(support, int64(i)+2)
			}

			plainVerdict, _, err := b.plain.Evaluate(t.Context(), run)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, plainVerdict.GetStatus())

			verdict, evaluation, err := b.assessed.Evaluate(t.Context(), run, nil)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, plainVerdict, verdict)
			binding := b.factory.Binding()
			require.Equal(t, &testpilot.Assessment{Model: binding.Model, Query: binding.Query,
				Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, SupportingEventSequences: support},
				Properties: []testpilot.PropertyAssessment{{ID: test.property, Status: testpilot.PropertyInconclusive,
					Detail: nexusMachine + ", " + nexusInstance(test.query) + ": " + test.why}},
			}, evaluation.Assessment)
		})
	}
}

// A second completed event is a visible mismatch: once an operation is over no step records anything
// (terminalIsFinal, and the kernel's handlerReplyStep and completeStep). The Contract cannot place the
// event either, so its own evaluation fails there and its Verdict is inconclusive. The assessment
// reads that event, which the Run recorded whole, and reports the nonconformance beside the Verdict.
func TestAMismatchTheContractFailsOnIsANonconformance(t *testing.T) {
	b := loweredNexus(t, nexusModel(t), "syncCompletion", generous)
	evidence := nexusEvidenceOf(t, b.source, "scheduled", "completed", "completed")

	_, _, err := b.plain.Evaluate(t.Context(), constructedRun(t, b.source, evidence, 0))
	require.ErrorContains(t, err, "event 4:", "the Contract's evaluation fails on the second completed event")

	run := constructedRun(t, b.source, evidence, 4)
	plainVerdict, _, err := b.plain.Evaluate(t.Context(), run)
	require.NoError(t, err)
	require.Equal(t, testpilotspb.VERDICT_STATUS_INCONCLUSIVE, plainVerdict.GetStatus())

	verdict, evaluation, err := b.assessed.Evaluate(t.Context(), run, nil)
	require.NoError(t, err)
	protorequire.ProtoEqual(t, plainVerdict, verdict)
	binding := b.factory.Binding()
	why := nexusMachine + ", " + nexusInstance("syncCompletion") + ": " + whyUnexplained
	require.Equal(t, &testpilot.Assessment{Model: binding.Model, Query: binding.Query,
		Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceNonconformant, SupportingEventSequences: []int64{2, 3, 4}, Detail: why},
		Properties:  []testpilot.PropertyAssessment{{ID: "syncSucceeds", Status: testpilot.PropertyInconclusive, Detail: why}},
	}, evaluation.Assessment)
}

// A recorded Assessment is replayed only under the binding it was made under: another Model, another
// Query, and the same Query under other ceilings are each another identity, refused before anything
// is evaluated. A factory is attached only to the Case it was prepared for.
func TestAnAssessmentIsReplayedOnlyUnderItsOwnBinding(t *testing.T) {
	m := nexusModel(t)
	b := loweredNexus(t, m, "retry", generous)
	run := constructedRun(t, b.source, nexusEvidenceOf(t, b.source, "scheduled", "pendingAttempts", "completed"), 0)
	_, evaluation, err := b.assessed.Evaluate(t.Context(), run, nil)
	require.NoError(t, err)
	recorded := evaluation.Assessment
	_, _, err = b.assessed.Evaluate(t.Context(), run, recorded)
	require.NoError(t, err)

	changed := proto.CloneOf(m)
	changed.GetQueries()[0].GetLimits().Search++
	smaller := generous
	smaller.MaxCandidates--
	for name, other := range map[string]*Factory{
		"another Model":  mustPrepare(t, changed, "retry", b.source, generous),
		"other ceilings": mustPrepare(t, m, "retry", b.source, smaller),
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, b.factory.Binding(), other.Binding())
			assessed, err := b.plain.WithAssessment(other)
			require.NoError(t, err)
			_, _, err = assessed.Evaluate(t.Context(), run, recorded)
			require.ErrorIs(t, err, testpilot.ErrForeignAssessment)
		})
	}

	t.Run("another Query's Case", func(t *testing.T) {
		other := loweredNexus(t, m, "syncCompletion", generous)
		require.NotEqual(t, b.factory.Binding().Query, other.factory.Binding().Query)
		_, err := other.plain.WithAssessment(b.factory)
		var refused *testpilot.PreparationError
		require.ErrorAs(t, err, &refused)
		require.Equal(t, "assessment.binding.case", refused.Path)
	})

	// Moving a declaration in its file is no other Model.
	moved := proto.CloneOf(m)
	moved.GetMachines()[0].GetPosition().Line += 100
	require.Equal(t, b.factory.Binding(), mustPrepare(t, moved, "retry", b.source, generous).Binding())
}

func mustPrepare(t testing.TB, m *modelirspb.Model, query string, source *testpilotspb.Case, limits Limits) *Factory {
	t.Helper()
	factory, err := Prepare(m, goir.ClaimKey{Family: nexusFamily, Owner: nexusMachine, Name: query}, source, limits)
	require.NoError(t, err)
	return factory
}
