package conformance

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	enumspb "go.temporal.io/api/enums/v1"
	historypb "go.temporal.io/api/history/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/common/testing/testpilot/temporal"
	"go.temporal.io/server/tools/umpire/check"
	"go.temporal.io/server/tools/umpire/ir"
	"go.temporal.io/server/tools/umpire/lower"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	nexusFamily   = "temporal.features.nexuscaller.system"
	nexusMachine  = "nexusSystem"
	nexusEvidence = "temporal.features.nexuscaller.evidence."
)

func nexusModel(t testing.TB) *umpirespb.Model {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", "nexus-caller.json"))
	require.NoError(t, err)
	return m
}

// loweredNexus is one functional Query of the Nexus caller Model lowered to its Case, prepared as a
// black-box consumer prepares it, with the Query's assessment bound to it.
func loweredNexus(t testing.TB, m *umpirespb.Model, query string, limits Limits) *bound {
	t.Helper()
	producer, err := lower.NewProducer(m)
	require.NoError(t, err)
	lowering, err := producer.Lower(query, lower.IdentityFor("temporal.case", "nexusCallerTests", query))
	require.NoError(t, err)
	require.Equal(t, lower.Lowered, lowering.Standing)
	catalog, err := temporal.NewWorkflowServiceCatalog()
	require.NoError(t, err)
	profile, err := temporal.DeriveProfile(lowering.Case, catalog, temporal.Environment{Identity: query + "-profile", Namespace: "namespace",
		TaskQueue: "task-queue", HandlerTaskQueue: "task-queue-handler", NexusEndpoint: "nexus-endpoint"})
	require.NoError(t, err)
	plain, err := testpilot.Prepare(lowering.Case, profile)
	require.NoError(t, err)
	factory, err := Prepare(m, check.ClaimKey{Family: nexusFamily, Owner: nexusMachine, Name: query}, lowering.Case, limits)
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
		// A kind the Case names by no name of its own is named by its Definition ID.
		name, renamed := local[nexusEvidence+kind]
		if !renamed {
			name = nexusEvidence + kind
		}
		var declared *testpilotspb.EvidenceDeclaration
		for _, e := range source.GetProgram().GetEvidence() {
			if e.GetEvidenceId() == name {
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
	return constructedRunOf(t, source, evidence, failure, false)
}

// constructedRunOf is constructedRun, with, when historyRead is set, the history read's own completion
// recorded as a Run records it: one event that carries the instruction's outcome, before the events
// its response read emits.
func constructedRunOf(t testing.TB, source *testpilotspb.Case, evidence []*testpilotspb.CorrelatedEvidence, failure int64, historyRead bool) *testpilotspb.Run {
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
	if historyRead {
		record(&testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, SourceId: "history.completed",
			Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", ActivationId: "controller", InstructionId: "history", Attempt: 1},
			Payload:     &testpilotspb.RunEvent_Outcome{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}}})
	}
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

// nexusWitness is the witness Run of one lowered Nexus caller Case: the kinds of evidence it records,
// and why its Property stays open on it while nothing says what its history does not hold.
type nexusWitness struct {
	query, property string
	kinds           []string
	why             reason
}

// Each expectation is read off model/temporal/features/nexuscaller/system/System.scala (the protocol
// machine's effects, rules and properties), which the comment beside it cites.
var nexusWitnesses = []nexusWitness{
	// A completed event is also what a completion of an operation the handler never answered
	// synchronously records (NexusSystem.effects.complete). Such a completion records a started event too, which only a
	// closed history shows to be absent: without the closing read, on that execution no synchronous
	// reply is taken.
	{"syncCompletion", "syncSucceeds", []string{"scheduled", "completed"}, whyNeverRead},
	// A completion that arrives after the operation is over is not found and records nothing
	// (NexusSystem.rules, effects.notFound), so no evidence excludes one, and the claim, which is about every
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
}

// Each of the seven lowered Cases is replayed on a constructed Run that records exactly its
// witness's evidence. The Contract reads that evidence as the witness, step by step, and is
// satisfied. The assessment keeps every execution of nexusSystem that explains it.
func TestAWitnessRunConformsWhileItsPropertyStaysOpen(t *testing.T) {
	m := nexusModel(t)
	for _, test := range nexusWitnesses {
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
					Reason: ir.ExpectationID(test.why), Detail: nexusMachine + ", " + nexusInstance(test.query) + ": " + wording[test.why]}},
			}, evaluation.Assessment)
		})
	}
}

// The realization declares the history kinds exhaustive and the history read their closing read
// (Realization.scala), every Case carries all five of them, and each witness Run here records that
// read's success. A closed source rules out an unobserved fact of a kind the Case carries, so a
// history that holds no started, failed, canceled or timed-out event rules out every execution that
// records one.
//
// That settles syncCompletion: an operation completed with no started event was answered
// synchronously (the kernel's handlerReplyStep and completeStep both record a started event on every
// other way to a completion), so on every execution left the completed event is the synchronous
// reply's and `syncSucceeds` holds.
//
// The other six stay open, each for a reason no closed history touches, and none was narrowed:
//
//   - handlerError: a failed reply records the same failed event as a non-retryable handler error, so
//     no kind of evidence tells the two classes apart, and on the failed reply the claim is never read.
//   - asyncCompletion, asyncFailure: a completion that arrives after the operation is over is not
//     found and records no fact, and a source reports facts. No kind of evidence, exhaustive or not,
//     reports a step that records nothing, and the claim, which is about every completion, fails on it.
//   - retry: the pending-attempts poll stops at the first count it sees and is no exhaustive source,
//     and the claim also fixes the deadlines, which the scheduled event's one fact does not tell apart.
//   - scheduleToStartTimeout, startToCloseTimeout: the three deadlines record one fact name, and the
//     timeout type is an enum, which evidence cannot keep as a field.
func TestAClosedHistorySettlesSyncCompletionAndLeavesTheOtherSixOpen(t *testing.T) {
	m := nexusModel(t)
	for _, test := range nexusWitnesses {
		t.Run(test.query, func(t *testing.T) {
			b := loweredNexus(t, m, test.query, generous)
			run := constructedRunOf(t, b.source, nexusEvidenceOf(t, b.source, test.kinds...), 0, true)
			var support []int64
			for i := range test.kinds {
				support = append(support, int64(i)+3)
			}
			property := testpilot.PropertyAssessment{ID: test.property, Status: testpilot.PropertyInconclusive,
				Reason: ir.ExpectationID(test.why), Detail: nexusMachine + ", " + nexusInstance(test.query) + ": " + wording[test.why]}
			if test.query == "syncCompletion" {
				property = testpilot.PropertyAssessment{ID: test.property, Status: testpilot.PropertySatisfied, SupportingEventSequences: support}
			}
			verdict, evaluation, err := b.assessed.Evaluate(t.Context(), run, nil)
			require.NoError(t, err)
			require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())
			binding := b.factory.Binding()
			require.Equal(t, &testpilot.Assessment{Model: binding.Model, Query: binding.Query,
				Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, SupportingEventSequences: support},
				Properties:  []testpilot.PropertyAssessment{property},
			}, evaluation.Assessment)
		})
	}
}

// historyDriver plays a lowered Nexus caller Case against no server, through the public facade alone.
// It answers every call of the controller, hands each read of the workflow's history, the poll for the
// scheduled event and the closing read alike, the history it was given, and settles every worker
// activation as one that ran. The evidence of the Run is then what the Case's own declarations lift
// from that history, by Testpilot's executor and recorder.
type historyDriver struct {
	identity testpilot.DriverIdentity
	history  []*historypb.HistoryEvent
}

func (d *historyDriver) Identity(context.Context) (testpilot.DriverIdentity, error) {
	return d.identity, nil
}
func (d *historyDriver) Validate(context.Context, testpilot.PreparedProgram) error { return nil }
func (d *historyDriver) Open(context.Context, string, testpilot.PreparedProgram) (testpilot.Session, error) {
	return &historySession{history: d.history}, nil
}

type historySession struct {
	history []*historypb.HistoryEvent
	// reserved counts the activations reserved, which names each one.
	reserved atomic.Int64
}

func (s *historySession) response(method protoreflect.MethodDescriptor) proto.Message {
	if method.Name() == "GetWorkflowExecutionHistory" {
		return &workflowservice.GetWorkflowExecutionHistoryResponse{History: &historypb.History{Events: s.history}}
	}
	return dynamicpb.NewMessage(method.Output())
}

func (s *historySession) Reserve(_ context.Context, request testpilot.ReservationRequest) ([]testpilot.ReservationHandle, error) {
	var out []testpilot.ReservationHandle
	for ordinal := range request.Count {
		out = append(out, activation{testpilot.ReservationIdentity{Origin: request.Origin, EntrypointID: request.EntrypointID, Ordinal: ordinal,
			ID: "activation-" + strconv.FormatInt(s.reserved.Add(1), 10)}})
	}
	return out, nil
}
func (s *historySession) InvokeRPC(_ context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message) (testpilot.EffectHandle, error) {
	return effect{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: s.response(method)}, nil
}
func (s *historySession) PollRPC(ctx context.Context, _ testpilot.Coordinate, _ string, method protoreflect.MethodDescriptor, _ proto.Message, _ time.Duration,
	accepts testpilot.PollPredicate) (testpilot.EffectHandle, error) {
	response := s.response(method)
	if accepted, err := accepts(ctx, response); err != nil || !accepted {
		return nil, fmt.Errorf("the history played never ends the poll of %s: %w", method.Name(), err)
	}
	return effect{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}, nil
}
func (*historySession) InvokeHandle(context.Context, testpilot.Coordinate, testpilot.OpaqueHandle, proto.Message) (testpilot.EffectHandle, error) {
	return nil, errUnscripted
}
func (*historySession) InjectFault(context.Context, testpilot.Coordinate, string, testpilotspb.FaultKind) (testpilot.EffectHandle, error) {
	return nil, errUnscripted
}
func (*historySession) Bridge(context.Context) (testpilot.HandleBridge, error)   { return nil, nil }
func (*historySession) Quarantine(context.Context, testpilot.EffectHandle) error { return nil }
func (*historySession) Close(context.Context) error                              { return nil }
func (*historySession) Diagnose(context.Context, string, *testpilotspb.RunDiagnostic) error {
	return nil
}

// activation is a worker activation that ran: what a Driver reports of a workflow or a Nexus handler
// its worker served.
type activation struct{ identity testpilot.ReservationIdentity }

func (a activation) Identity() testpilot.ReservationIdentity { return a.identity }
func (a activation) Consume(context.Context) (testpilot.Coordinate, error) {
	return a.identity.Origin, nil
}
func (activation) Wait(context.Context) (testpilot.EffectResult, error) {
	return testpilot.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}}, nil
}
func (activation) Cancel(context.Context) error { return nil }
func (activation) Drain(context.Context) error  { return nil }

// scheduledEventID is the id of the scheduled event of the one operation these histories hold, which
// keys every event of the operation.
const scheduledEventID = 5

func historyEvent(id int64, attributes any) *historypb.HistoryEvent {
	event := &historypb.HistoryEvent{EventId: id}
	switch attributes := attributes.(type) {
	case *historypb.NexusOperationScheduledEventAttributes:
		event.EventType = enumspb.EVENT_TYPE_NEXUS_OPERATION_SCHEDULED
		event.Attributes = &historypb.HistoryEvent_NexusOperationScheduledEventAttributes{NexusOperationScheduledEventAttributes: attributes}
	case *historypb.NexusOperationStartedEventAttributes:
		event.EventType = enumspb.EVENT_TYPE_NEXUS_OPERATION_STARTED
		event.Attributes = &historypb.HistoryEvent_NexusOperationStartedEventAttributes{NexusOperationStartedEventAttributes: attributes}
	case *historypb.NexusOperationCompletedEventAttributes:
		event.EventType = enumspb.EVENT_TYPE_NEXUS_OPERATION_COMPLETED
		event.Attributes = &historypb.HistoryEvent_NexusOperationCompletedEventAttributes{NexusOperationCompletedEventAttributes: attributes}
	default:
		event.EventType = enumspb.EVENT_TYPE_WORKFLOW_TASK_COMPLETED
	}
	return event
}

// syncCompletion concludes on its witness Run. The lowered Case runs here, live, through Testpilot's
// own executor and recorder against a Driver that plays a history, and the Run it records is then
// replayed: the two assessments are one value.
//
// The witness's history holds the scheduled event and the completed event of one operation. The Case
// carries the started event though its path records none, the history read closes it, and no started
// event is lifted, so every execution that starts the operation asynchronously, or completes one never
// answered, is ruled out: both record that event (the kernel's handlerReplyStep and completeStep). The
// completed event is then the synchronous reply's on every execution left, and `syncSucceeds` is
// satisfied.
//
// A history that also holds a started event is the history of an operation completed another way. The
// Contract gives the started event no meaning and reads the Run as the witness, satisfied; the
// assessment reads an operation whose completion no synchronous reply made, on which the claim is
// never read, and leaves it inconclusive.
func TestSyncCompletionConcludesOnItsWitnessRunLiveAndReplayed(t *testing.T) {
	m := nexusModel(t)
	scheduled := historyEvent(scheduledEventID, &historypb.NexusOperationScheduledEventAttributes{})
	started := historyEvent(scheduledEventID+1, &historypb.NexusOperationStartedEventAttributes{ScheduledEventId: scheduledEventID})
	completed := func(id int64) *historypb.HistoryEvent {
		return historyEvent(id, &historypb.NexusOperationCompletedEventAttributes{ScheduledEventId: scheduledEventID})
	}
	// Events of the workflow that are none of the operation's surround it, as in any history.
	other := func(id int64) *historypb.HistoryEvent { return historyEvent(id, nil) }
	for name, test := range map[string]struct {
		history  []*historypb.HistoryEvent
		kinds    []string
		property func(support []int64) testpilot.PropertyAssessment
	}{
		"the witness": {[]*historypb.HistoryEvent{other(4), scheduled, completed(6), other(7)}, []string{"scheduled", "completed"},
			func(support []int64) testpilot.PropertyAssessment {
				return testpilot.PropertyAssessment{ID: "syncSucceeds", Status: testpilot.PropertySatisfied, SupportingEventSequences: support}
			}},
		"a history that holds a started event": {[]*historypb.HistoryEvent{other(4), scheduled, started, completed(7), other(8)},
			[]string{"scheduled", "started", "completed"},
			func([]int64) testpilot.PropertyAssessment {
				return testpilot.PropertyAssessment{ID: "syncSucceeds", Status: testpilot.PropertyInconclusive,
					Reason: "never_evaluated", Detail: nexusMachine + ", " + nexusInstance("syncCompletion") + ": " + wording[whyNeverRead]}
			}},
	} {
		t.Run(name, func(t *testing.T) {
			b := loweredNexus(t, m, "syncCompletion", generous)
			run, verdict, live, err := b.assessed.Run(t.Context(), &historyDriver{identity: b.plain.Identity(), history: test.history})
			require.NoError(t, err)
			require.Equal(t, testpilotspb.RUN_DISPOSITION_COMPLETED, run.GetDisposition(), "%v", run.GetDiagnostics())
			require.Equal(t, testpilotspb.VERDICT_STATUS_SATISFIED, verdict.GetStatus())

			// The Run carries what the Case's declarations lift from the history, and nothing else.
			support, kinds := playedKinds(t, b.source, run, nexusEvidence)
			require.Equal(t, test.kinds, kinds)
			binding := b.factory.Binding()
			require.Equal(t, &testpilot.Assessment{Model: binding.Model, Query: binding.Query,
				Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceConformant, SupportingEventSequences: support},
				Properties:  []testpilot.PropertyAssessment{test.property(support)}}, live)

			plainVerdict, _, err := b.plain.Evaluate(t.Context(), run)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, plainVerdict, verdict)
			replayed, evaluation, err := b.assessed.Evaluate(t.Context(), run, live)
			require.NoError(t, err)
			protorequire.ProtoEqual(t, verdict, replayed)
			require.Equal(t, live, evaluation.Assessment)
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
	why := nexusMachine + ", " + nexusInstance("syncCompletion") + ": " + wording[whyUnexplained]
	require.Equal(t, &testpilot.Assessment{Model: binding.Model, Query: binding.Query,
		Conformance: testpilot.ConformanceAssessment{Status: testpilot.ConformanceNonconformant, SupportingEventSequences: []int64{2, 3, 4}, Reason: "unexplained", Detail: why},
		Properties:  []testpilot.PropertyAssessment{{ID: "syncSucceeds", Status: testpilot.PropertyInconclusive, Reason: "unexplained", Detail: why}},
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

func mustPrepare(t testing.TB, m *umpirespb.Model, query string, source *testpilotspb.Case, limits Limits) *Factory {
	t.Helper()
	factory, err := Prepare(m, check.ClaimKey{Family: nexusFamily, Owner: nexusMachine, Name: query}, source, limits)
	require.NoError(t, err)
	return factory
}
