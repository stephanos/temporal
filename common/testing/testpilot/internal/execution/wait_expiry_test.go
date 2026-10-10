package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/protorequire"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/contract"
	pbduration "go.temporal.io/server/common/testing/testpilot/duration"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// withWaitHints makes the fixture's poll wait within two declared bounds, 30 and 20 ms.
func withWaitHints(source *testpilotspb.Case) {
	poll := source.Program.Entrypoints[0].Instructions[0]
	poll.Limits.Timeout = pbduration.FromMilliseconds(50)
	poll.WaitHints = []*testpilotspb.WaitHint{
		{HintId: "visibility.describe", Source: &testpilotspb.SourceLocation{Path: "model/Behavior.scala", Line: 12}, AtMost: pbduration.FromMilliseconds(30)},
		{HintId: "cause.timer", Source: &testpilotspb.SourceLocation{Path: "model/Behavior.scala", Line: 20}, AtMost: pbduration.FromMilliseconds(20)},
	}
}

// asReadOnce makes the fixture's poll a read once.
func asReadOnce(source *testpilotspb.Case) {
	read := source.Program.Entrypoints[0].Instructions[0].Instruction.GetReadEvidence()
	read.Interval = nil
}

// readOnce answers a read as the Driver contract says a zero interval does: one call, the condition
// asked once, and a response it does not accept ends the effect TIMED_OUT without the response. It
// keeps the interval each read was given.
func readOnce(intervals *[]time.Duration, response proto.Message) func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, contract.PollPredicate) (contract.EffectHandle, error) {
	return func(ctx context.Context, _ contract.Coordinate, _ string, _ protoreflect.MethodDescriptor, _ proto.Message, interval time.Duration, satisfied contract.PollPredicate) (contract.EffectHandle, error) {
		*intervals = append(*intervals, interval)
		accepted, err := satisfied(ctx, response)
		if err != nil {
			return nil, err
		}
		result := contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}}
		if accepted {
			result = contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}
		}
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) { return result, nil }}, nil
	}
}

// runReport prepares and runs the fixture against host and returns the recorded Run.
func runReport(t *testing.T, source *testpilotspb.Case, catalog *ir.Catalog, policy Profile, host *testsupport.Session) *testpilotspb.Run {
	t.Helper()
	prepared, err := Prepare(source, catalog, policy)
	require.NoError(t, err)
	s, err := newScheduler(prepared, "run", "case", host, schedulerMonitor{}, time.Now)
	require.NoError(t, err)
	require.NoError(t, s.execute(context.Background()))
	s.waits.Wait()
	require.Empty(t, diagnosticCodes(s.recorder.run))
	return s.recorder.run
}

// recordedOutcomes is every outcome the Run recorded, in order.
func recordedOutcomes(run *testpilotspb.Run) []*testpilotspb.InstructionOutcome {
	var recorded []*testpilotspb.InstructionOutcome
	for _, event := range run.GetEvents() {
		if outcome := event.GetOutcome(); outcome != nil {
			recorded = append(recorded, outcome)
		}
	}
	return recorded
}

// withoutDetail is the Run's events without what two runs of one Case may differ in: the elapsed time
// and every outcome's detail.
func withoutDetail(run *testpilotspb.Run) []*testpilotspb.RunEvent {
	var events []*testpilotspb.RunEvent
	for _, event := range run.GetEvents() {
		event = proto.CloneOf(event)
		event.Elapsed = pbduration.FromMilliseconds(0)
		if outcome := event.GetOutcome(); outcome != nil {
			outcome.Detail = ""
		}
		events = append(events, event)
	}
	return events
}

// A hinted poll that never holds times out naming its condition, the bound it ran under and each
// hint with where it is declared, whether the scheduler's deadline or the Driver ended it. Under a
// scaled Profile it also names the declared bound and the scale.
func TestAnExpiredHintedPollNamesItsConditionBoundAndHints(t *testing.T) {
	const hints = "; hints: visibility.describe (model/Behavior.scala:12) 30 ms, cause.timer (model/Behavior.scala:20) 20 ms"
	driverTimedOut := func(context.Context, contract.Coordinate, string, protoreflect.MethodDescriptor, proto.Message, time.Duration, contract.PollPredicate) (contract.EffectHandle, error) {
		return &testsupport.Effect{OnWait: func(context.Context) (contract.EffectResult, error) {
			return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, ProtocolCode: "deadline_exceeded"}}, nil
		}}, nil
	}
	for name, test := range map[string]struct {
		scale   contract.BoundScale
		driver  bool
		outcome *testpilotspb.InstructionOutcome
	}{
		"as declared": {outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT,
			Detail: "evidence itemSettled: until value.state > 1 did not hold within 50 ms" + hints}},
		"scaled": {scale: 150, outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT,
			Detail: "evidence itemSettled: until value.state > 1 did not hold within 75 ms (declared 50 ms, scaled by 150%)" + hints}},
		"timed out by the Driver": {driver: true, outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT, ProtocolCode: "deadline_exceeded",
			Detail: "evidence itemSettled: until value.state > 1 did not hold within 50 ms" + hints}},
	} {
		t.Run(name, func(t *testing.T) {
			source, catalog, policy := reportFixture(t)
			withWaitHints(source)
			policy.BoundScale = test.scale
			var answers []bool
			host := &testsupport.Session{OnPollRPC: polled(&answers, report(t, catalog, &reportItem{"a", 1}))}
			if test.driver {
				host.OnPollRPC = driverTimedOut
			}
			run := runReport(t, source, catalog, policy, host)
			require.Empty(t, liftedEvidence(t, run))
			protorequire.ProtoSliceEqual(t, []*testpilotspb.InstructionOutcome{test.outcome}, recordedOutcomes(run))
			for _, event := range run.GetEvents() {
				if event.GetOutcome() != nil {
					require.Equal(t, testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT, event.GetKind())
				}
			}
		})
	}
}

// A poll that declares no hint times out as it always has: its outcome carries no detail.
func TestAnExpiredUnhintedPollCarriesNoDetail(t *testing.T) {
	source, catalog, policy := reportFixture(t)
	source.Program.Entrypoints[0].Instructions[0].Limits.Timeout = pbduration.FromMilliseconds(50)
	var answers []bool
	run := runReport(t, source, catalog, policy, &testsupport.Session{OnPollRPC: polled(&answers, report(t, catalog, &reportItem{"a", 1}))})
	protorequire.ProtoSliceEqual(t, []*testpilotspb.InstructionOutcome{{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT}}, recordedOutcomes(run))
}

// A read once whose condition does not hold reads one time, with no interval, and records what a
// poll whose timeout ran out records, but for the detail that says it was read once: TIMED_OUT and
// no evidence.
func TestAReadOnceThatDoesNotHoldRecordsAsATimedOutPoll(t *testing.T) {
	source, catalog, policy := reportFixture(t)
	asReadOnce(source)
	var intervals []time.Duration
	host := &testsupport.Session{OnPollRPC: readOnce(&intervals, report(t, catalog, &reportItem{"a", 1}))}
	once := runReport(t, source, catalog, policy, host)
	require.Equal(t, []time.Duration{0}, intervals)
	require.Equal(t, 1, host.Calls("PollRPC"))
	require.Empty(t, liftedEvidence(t, once))
	protorequire.ProtoSliceEqual(t, []*testpilotspb.InstructionOutcome{{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_TIMED_OUT,
		Detail: "evidence itemSettled: until value.state > 1 did not hold when read once, within 1000 ms"}}, recordedOutcomes(once))

	source, catalog, policy = reportFixture(t)
	source.Program.Entrypoints[0].Instructions[0].Limits.Timeout = pbduration.FromMilliseconds(50)
	var answers []bool
	poll := runReport(t, source, catalog, policy, &testsupport.Session{OnPollRPC: polled(&answers, report(t, catalog, &reportItem{"a", 1}))})
	protorequire.ProtoSliceEqual(t, withoutDetail(poll), withoutDetail(once))
}

// A read once whose condition holds lifts and records exactly what a poll accepting the same
// response does.
func TestAReadOnceThatHoldsLiftsWhatAPollWould(t *testing.T) {
	source, catalog, policy := reportFixture(t)
	asReadOnce(source)
	var intervals []time.Duration
	once := runReport(t, source, catalog, policy, &testsupport.Session{OnPollRPC: readOnce(&intervals, report(t, catalog, &reportItem{"a", 2}))})
	require.Equal(t, []time.Duration{0}, intervals)
	protorequire.ProtoSliceEqual(t, []*testpilotspb.CorrelatedEvidence{sourced("report", 0, "itemSettled", "a", stateField("2"))}, liftedEvidence(t, once))

	source, catalog, policy = reportFixture(t)
	var answers []bool
	poll := runReport(t, source, catalog, policy, &testsupport.Session{OnPollRPC: polled(&answers, report(t, catalog, &reportItem{"a", 2}))})
	protorequire.ProtoSliceEqual(t, withoutDetail(poll), withoutDetail(once))
	require.Empty(t, recordedOutcomes(once)[0].GetDetail())
}

// A condition is named the same way every time, from every arm of the expression language, and
// within its bound however deep or long it is.
func TestAnExpiryNamesItsConditionDeterministically(t *testing.T) {
	text := func(value string) *testpilotspb.Expression {
		return cel.Literal(textValue(value))
	}
	not := func(operand *testpilotspb.Expression) *testpilotspb.Expression {
		return cel.Not(operand)
	}
	anyOf := func(operands ...*testpilotspb.Expression) *testpilotspb.Expression {
		return cel.Any(operands...)
	}
	nested := projected("state")
	for range conditionRenderDepth {
		nested = not(nested)
	}
	for name, test := range map[string]struct {
		condition *testpilotspb.Expression
		rendered  string
	}{
		"every arm": {
			all(present(projected("item.key")), not(compare("_!=_", projected("key"), textValue("a"))), anyOf(compare("_<=_", projected(""), integer("-3")), text("b"))),
			`all(present(value.item.key), all(not(value.key != "a"), any(value <= -3, "b")))`,
		},
		"too deep": {nested, "not(not(not(not(not(not(not(not(...))))))))"},
		"too long": {text(strings.Repeat("x", 2*conditionRenderBytes)), `"` + strings.Repeat("x", conditionRenderBytes-4) + "..."},
		"absent":   {nil, "<absent>"},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.rendered, renderCondition(test.condition))
			require.LessOrEqual(t, len(renderCondition(test.condition)), conditionRenderBytes)
		})
	}
}
