package conformance

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	failurepb "go.temporal.io/api/failure/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

// activitySource is the Run Event source of one kind of evidence of the standalone activity Model's
// realization, by the last part of the kind's id, as the lifter emitted it
// (model/ir/activity-standalone.json).
func activitySource(t testing.TB, kind string) *umpirespb.RunEventSource {
	t.Helper()
	m, err := ir.Load(filepath.Join("..", "..", "..", "model", "ir", "activity-standalone.json"))
	require.NoError(t, err)
	for _, e := range realizationNamed(t, m, "standalone").GetEvidence() {
		if e.GetId() == activityEvidence+kind {
			require.NotNil(t, e.GetRunEvent(), "%s is the Run's own record", kind)
			return e.GetRunEvent()
		}
	}
	require.FailNow(t, "no evidence "+kind)
	return nil
}

// reported is the Run Event of one kind a command records, with an instruction outcome.
func reported(kind testpilotspb.RunEventKind, command string, outcome *testpilotspb.InstructionOutcome) *testpilotspb.RunEvent {
	return &testpilotspb.RunEvent{Sequence: 9, Kind: kind, Payload: &testpilotspb.RunEvent_Outcome{Outcome: outcome},
		Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", ActivationId: "controller", InstructionId: command, Attempt: 1}}
}

func attemptOf(sdkAttempt int32, delivery string, response testpilotspb.ActivityAttemptResponse) *testpilotspb.InstructionOutcome {
	return &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED,
		ActivityAttempt: &testpilotspb.ActivityAttempt{ActivityRunId: "run-a", SdkAttempt: sdkAttempt, DeliveryId: delivery, Response: response}}
}

// The evidence of an attempt is the worker's record of the attempt it was delivered under one number:
// the first attempt's is evidence that the activity started, and the second's that it was scheduled
// and started again (Realization.scala, `delivered`). A delivered attempt has the number the server
// counts, from 1, and a delivery (run.proto, ActivityAttempt). The record of a declared position no
// attempt was delivered for has neither, and is evidence of no attempt; nor is a record with one of
// the two. What the worker then offered does not enter into it: a delivered attempt it refused was
// delivered all the same. One record is one piece of evidence, so no record is taken by both kinds.
func TestOnlyADeliveredAttemptIsEvidenceOfAnAttemptStart(t *testing.T) {
	const diagnostic = testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC
	type taken struct{ first, second bool }
	for name, test := range map[string]struct {
		event *testpilotspb.RunEvent
		want  taken
	}{
		"a delivered attempt that completed":      {reported(diagnostic, "start-activity", attemptOf(1, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)), taken{first: true}},
		"a delivered second attempt":              {reported(diagnostic, "start-activity", attemptOf(2, "token-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE)), taken{second: true}},
		"a delivered third attempt":               {reported(diagnostic, "start-activity", attemptOf(3, "token-3", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)), taken{}},
		"a delivered attempt the worker refused":  {reported(diagnostic, "start-activity", attemptOf(1, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_REFUSED)), taken{first: true}},
		"a position no attempt was delivered for": {reported(diagnostic, "start-activity", attemptOf(0, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED)), taken{}},
		"no attempt number, with a delivery":      {reported(diagnostic, "start-activity", attemptOf(0, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)), taken{}},
		"an attempt number, with no delivery":     {reported(diagnostic, "start-activity", attemptOf(1, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)), taken{}},
		"a second attempt with no delivery":       {reported(diagnostic, "start-activity", attemptOf(2, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)), taken{}},
		"an outcome that is of no activity attempt": {reported(diagnostic, "start-activity",
			&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}), taken{}},
		"the start call's own completion": {reported(testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED, "start-activity",
			attemptOf(1, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)), taken{}},
		"an attempt another command carries": {reported(diagnostic, "pause-activity", attemptOf(1, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)), taken{}},
	} {
		t.Run(name, func(t *testing.T) {
			first, err := admits(activitySource(t, "statusStarted"), test.event)
			require.NoError(t, err)
			second, err := admits(activitySource(t, "attemptCount"), test.event)
			require.NoError(t, err)
			require.Equal(t, test.want, taken{first, second})
		})
	}
}

// A source declared the record of an attempt takes the record of the attempt of that number and of
// no other, whatever its guard: the number is read from the Run's typed record, and an outcome that
// records no attempt is the record of none.
func TestASourceDeclaredTheRecordOfAnAttemptTakesThatAttemptsRecord(t *testing.T) {
	source := &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_DIAGNOSTIC, Script: "controller", Command: "start-activity",
		Attempt: &umpirespb.AttemptOf{Script: "activity", Number: 2}}
	const diagnostic = testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC
	for name, test := range map[string]struct {
		outcome *testpilotspb.InstructionOutcome
		want    bool
	}{
		"the attempt of that number": {attemptOf(2, "token-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED), true},
		"the attempt before it":      {attemptOf(1, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_FAILED_RETRYABLE), false},
		"the attempt after it":       {attemptOf(3, "token-3", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED), false},
		"an outcome of no attempt":   {&testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, false},
	} {
		t.Run(name, func(t *testing.T) {
			admitted, err := admits(source, reported(diagnostic, "start-activity", test.outcome))
			require.NoError(t, err)
			require.Equal(t, test.want, admitted)
		})
	}
}

// The evidence that the activity was scheduled, by its start and again by the release of a pause, and
// that its cancellation was requested, is the Run's record that the call succeeded: the command's own
// completion, and no other event of it.
func TestOnlyASucceededCallIsEvidenceOfItsAnswer(t *testing.T) {
	const completed = testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED
	succeeded := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, ProtocolCode: "ok"}
	failed := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE, ProtocolCode: "already_exists"}
	for name, test := range map[string]struct {
		records string
		event   *testpilotspb.RunEvent
		want    bool
	}{
		"the start succeeded":          {"statusScheduled", reported(completed, "start-activity", succeeded), true},
		"the start failed":             {"statusScheduled", reported(completed, "start-activity", failed), false},
		"the start timed out":          {"statusScheduled", reported(testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT, "start-activity", succeeded), false},
		"an attempt the start carries": {"statusScheduled", reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", succeeded), false},
		"another call succeeded":       {"statusScheduled", reported(completed, "request-cancel-activity", succeeded), false},
		"the cancel request succeeded": {"statusCancelRequested", reported(completed, "request-cancel-activity", succeeded), true},
		"the cancel request failed":    {"statusCancelRequested", reported(completed, "request-cancel-activity", failed), false},
		"the release succeeded":        {"statusScheduledAgain", reported(completed, "unpause-activity", succeeded), true},
		"the release failed":           {"statusScheduledAgain", reported(completed, "unpause-activity", failed), false},
		"the start, for the release":   {"statusScheduledAgain", reported(completed, "start-activity", succeeded), false},
		"an event with no outcome": {"statusScheduled", &testpilotspb.RunEvent{Kind: completed,
			Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", InstructionId: "start-activity"}}, false},
	} {
		t.Run(name, func(t *testing.T) {
			admitted, err := admits(activitySource(t, test.records), test.event)
			require.NoError(t, err)
			require.Equal(t, test.want, admitted)
		})
	}
}

func guardLiteral(kind any) *umpirespb.Operand {
	value := &umpirespb.ProtoValue{}
	switch k := kind.(type) {
	case string:
		value.Kind = &umpirespb.ProtoValue_Text{Text: k}
	case int:
		value.Kind = &umpirespb.ProtoValue_Number{Number: int64(k)}
	case bool:
		value.Kind = &umpirespb.ProtoValue_Flag{Flag: k}
	case protoName:
		value.Kind = &umpirespb.ProtoValue_EnumName{EnumName: string(k)}
	default:
	}
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: value}}
}

// protoName is an enum value written out by its name.
type protoName string

func guardPath(path string) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: path,
		Of: &umpirespb.Operand{Kind: &umpirespb.Operand_Projected{Projected: &emptypb.Empty{}}}}}}
}

// guardNested is a path of a path of the payload.
func guardNested(outer, inner string) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: inner, Of: guardPath(outer)}}}
}

func guardEqual(left, right *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Equal{Equal: &umpirespb.Equal{Left: left, Right: right}}}
}

func guardGreater(left, right *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Greater{Greater: &umpirespb.Greater{Left: left, Right: right}}}
}

func guardNot(of *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Not{Not: &umpirespb.Not{Of: of}}}
}

func guardPresent(of *umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_Present{Present: &umpirespb.Present{Of: of}}}
}

func guardAll(operands ...*umpirespb.Operand) *umpirespb.Operand {
	return &umpirespb.Operand{Kind: &umpirespb.Operand_All{All: &umpirespb.All{Operands: operands}}}
}

// guarded is a source of the attempts a start call carries, under one guard.
func guarded(guard *umpirespb.Operand) *umpirespb.RunEventSource {
	return &umpirespb.RunEventSource{Kind: umpirespb.RunEventSource_KIND_DIAGNOSTIC, Script: "controller", Command: "start-activity", Guard: guard}
}

// A guard is evaluated over typed values, read as the payload types them. Presence is of a message or
// a oneof member, and says nothing of a scalar's value: an attempt number of 0 is present. A
// conjunction is read up to its first operand that does not hold, as Testpilot reads one.
func TestAGuardIsEvaluatedOverTypedValues(t *testing.T) {
	delivered := reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", attemptOf(2, "token-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))
	undelivered := reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", attemptOf(0, "", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED))
	plain := reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", &testpilotspb.InstructionOutcome{
		Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED, Value: &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}})
	for name, test := range map[string]struct {
		guard *umpirespb.Operand
		event *testpilotspb.RunEvent
		want  bool
	}{
		"no guard":                          {nil, undelivered, true},
		"a number greater":                  {guardGreater(guardPath("activity_attempt.sdk_attempt"), guardLiteral(1)), delivered, true},
		"a number not greater":              {guardGreater(guardPath("activity_attempt.sdk_attempt"), guardLiteral(2)), delivered, false},
		"a text equal":                      {guardEqual(guardPath("activity_attempt.delivery_id"), guardLiteral("token-2")), delivered, true},
		"a text not equal":                  {guardEqual(guardPath("activity_attempt.delivery_id"), guardLiteral("")), delivered, false},
		"an enum value by its name":         {guardEqual(guardPath("activity_attempt.response"), guardLiteral(protoName("ACTIVITY_ATTEMPT_RESPONSE_NOT_NEEDED"))), undelivered, true},
		"another enum value":                {guardEqual(guardLiteral(protoName("ACTIVITY_ATTEMPT_RESPONSE_REFUSED")), guardPath("activity_attempt.response")), undelivered, false},
		"two fields of one enum":            {guardEqual(guardPath("status"), guardPath("status")), delivered, true},
		"a flag":                            {guardEqual(guardPath("value.bool_value"), guardLiteral(true)), plain, true},
		"a negation":                        {guardNot(guardEqual(guardPath("activity_attempt.delivery_id"), guardLiteral(""))), undelivered, false},
		"a message that is set":             {guardPresent(guardPath("activity_attempt")), undelivered, true},
		"a message that is not set":         {guardPresent(guardPath("activity_attempt")), plain, false},
		"a field of a message not set":      {guardPresent(guardPath("activity_attempt.sdk_attempt")), plain, false},
		"a scalar at its default":           {guardPresent(guardPath("activity_attempt.sdk_attempt")), undelivered, true},
		"a oneof member that is set":        {guardPresent(guardPath("value.value<bool_value>")), plain, true},
		"a oneof member that is not":        {guardPresent(guardPath("value.value<text_value>")), plain, false},
		"a member read by its field name":   {guardPresent(guardPath("value.text_value")), plain, false},
		"a field of a path that is not set": {guardPresent(guardNested("activity_attempt", "delivery_id")), plain, false},
		"a conjunction that holds":          {guardAll(guardPresent(guardPath("activity_attempt")), guardGreater(guardPath("activity_attempt.sdk_attempt"), guardLiteral(0))), delivered, true},
		"a conjunction that stops at once":  {guardAll(guardPresent(guardPath("activity_attempt")), guardGreater(guardPath("activity_attempt.sdk_attempt"), guardLiteral(0))), plain, false},
	} {
		t.Run(name, func(t *testing.T) {
			admitted, err := admits(guarded(test.guard), test.event)
			require.NoError(t, err)
			require.Equal(t, test.want, admitted)
		})
	}

	// A source takes the events its command records in its script that carry an outcome, and no other.
	elsewhere := reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", attemptOf(1, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))
	elsewhere.Coordinates.EntrypointId = "activity"
	bare := &testpilotspb.RunEvent{Kind: testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
		Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: "controller", InstructionId: "start-activity"}}
	for _, event := range []*testpilotspb.RunEvent{elsewhere, bare} {
		admitted, err := admits(guarded(nil), event)
		require.NoError(t, err)
		require.False(t, admitted)
	}
}

// A guard that cannot be evaluated on a Run Event is an error at that event, and is never read as a
// guard that does not hold. What is wrong with a guard's types is wrong whatever the event holds: a
// field the payload's type lacks and a value of the wrong type are reported where the value is absent
// too.
func TestAGuardThatCannotBeEvaluatedIsAnError(t *testing.T) {
	delivered := reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", attemptOf(1, "token-1", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED))
	plain := reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED})
	run := &umpirespb.Operand{Kind: &umpirespb.Operand_Run{Run: &emptypb.Empty{}}}
	for name, test := range map[string]struct {
		guard *umpirespb.Operand
		event *testpilotspb.RunEvent
		says  string
	}{
		"a field the payload does not have":               {guardPresent(guardPath("activity_attempt.attempt")), delivered, "reads activity_attempt.attempt, and temporal.server.api.testpilot.v1.ActivityAttempt has no field attempt"},
		"a field of a scalar":                             {guardPresent(guardPath("detail.length")), delivered, "reads detail.length, and temporal.server.api.testpilot.v1.InstructionOutcome.detail is no message"},
		"each element of a field":                         {guardPresent(guardPath("activity_attempt[*]")), delivered, `reads "activity_attempt[*]" of the path activity_attempt[*], and a guard reads a field or oneof<member>`},
		"a oneof the message does not have":               {guardPresent(guardPath("value.value<nope>")), delivered, "reads value<nope>, and temporal.server.api.testpilot.v1.Value has no such member"},
		"an order of a text":                              {guardGreater(guardPath("activity_attempt.delivery_id"), guardLiteral(0)), delivered, "orders a text, and only numbers are ordered"},
		"a comparison of a number with a text":            {guardEqual(guardPath("activity_attempt.sdk_attempt"), guardLiteral("1")), delivered, "compares a number with a text"},
		"a comparison of a value that is absent":          {guardGreater(guardPath("activity_attempt.sdk_attempt"), guardLiteral(0)), plain, "orders an absent value"},
		"a comparison with a value that is absent":        {guardEqual(guardLiteral("token-1"), guardPath("activity_attempt.delivery_id")), plain, "compares an absent value"},
		"a comparison of a message":                       {guardEqual(guardPath("activity_attempt"), guardLiteral("x")), delivered, "compares a message"},
		"an enum name the enum does not have":             {guardEqual(guardPath("status"), guardLiteral(protoName("INSTRUCTION_OUTCOME_STATUS_NOPE"))), delivered, "compares a value of temporal.server.api.testpilot.v1.InstructionOutcomeStatus with INSTRUCTION_OUTCOME_STATUS_NOPE, which it does not have"},
		"values of two enums":                             {guardEqual(guardPath("status"), guardPath("activity_attempt.response")), delivered, "compares a value of temporal.server.api.testpilot.v1.InstructionOutcomeStatus with one of temporal.server.api.testpilot.v1.ActivityAttemptResponse"},
		"two enum values written out":                     {guardEqual(guardLiteral(protoName("A")), guardLiteral(protoName("A"))), delivered, "compares two enum values it writes out"},
		"an enum value with a text":                       {guardEqual(guardPath("status"), guardLiteral("INSTRUCTION_OUTCOME_STATUS_SUCCEEDED")), delivered, "compares an enum value with a text"},
		"a negation of a number":                          {guardNot(guardPath("activity_attempt.sdk_attempt")), delivered, "negates a number, and only a condition is negated"},
		"a conjunction over a text":                       {guardAll(guardPath("activity_attempt.delivery_id")), delivered, "joins a text, and only conditions are joined"},
		"a conjunction of nothing":                        {guardAll(), delivered, "is a conjunction of no operand"},
		"a guard that is a number":                        {guardPath("activity_attempt.sdk_attempt"), delivered, "is a number, and a guard is a condition"},
		"a guard that is a message":                       {guardPath("activity_attempt"), delivered, "is a message, and a guard is a condition"},
		"a guard that is an unset message":                {guardPath("activity_attempt"), plain, "is a message, and a guard is a condition"},
		"a guard that is an absent flag":                  {guardPath("value.bool_value"), plain, "is an absent value, and a guard is a condition"},
		"a negation of an absent flag":                    {guardNot(guardPath("value.bool_value")), plain, "negates an absent value"},
		"a conjunction over an absent flag":               {guardAll(guardPath("value.bool_value")), plain, "joins an absent value"},
		"an order by a text":                              {guardGreater(guardPath("activity_attempt.sdk_attempt"), guardLiteral("0")), delivered, "orders a text, and only numbers are ordered"},
		"an order by a value that is absent":              {guardGreater(guardLiteral(0), guardPath("activity_attempt.sdk_attempt")), plain, "orders an absent value"},
		"an order of a text that is absent":               {guardGreater(guardPath("activity_attempt.delivery_id"), guardLiteral(0)), plain, "orders a text, and only numbers are ordered"},
		"an enum name on the left the enum does not have": {guardEqual(guardLiteral(protoName("INSTRUCTION_OUTCOME_STATUS_NOPE")), guardPath("status")), delivered, "compares a value of temporal.server.api.testpilot.v1.InstructionOutcomeStatus with INSTRUCTION_OUTCOME_STATUS_NOPE, which it does not have"},
		"a field an unset message does not have":          {guardPresent(guardNested("activity_attempt", "nope")), plain, "reads nope, and temporal.server.api.testpilot.v1.ActivityAttempt has no field nope"},
		"a field of a text that is absent":                {guardPresent(guardNested("activity_attempt.delivery_id", "x")), plain, "reads x of a text, which is no message"},
		"a guard that reads the run":                      {guardEqual(run, guardLiteral("run")), delivered, "reads the run's id; a Run Event's guard reads the event's payload alone"},
		"a guard of no kind":                              {&umpirespb.Operand{}, delivered, "has an operand of no known kind"},
		"a literal that is no value of a guard":           {guardEqual(guardPath("detail"), &umpirespb.Operand{Kind: &umpirespb.Operand_Literal{Literal: &umpirespb.ProtoValue{}}}), delivered, "writes out a value that is no text, flag, number or enum value"},
		"a field of a kind no guard reads":                {guardPresent(guardPath("value.bytes_value")), delivered, "reads temporal.server.api.testpilot.v1.Value.bytes_value, which is of kind bytes"},
		"several values":                                  {guardPresent(guardPath("value.list_value.values")), delivered, "reads value.list_value.values, and temporal.server.api.testpilot.v1.ValueList.values holds several values"},
		"an enum value its enum does not name": {guardEqual(guardPath("status"), guardLiteral(protoName("INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"))),
			reported(testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC, "start-activity", &testpilotspb.InstructionOutcome{Status: 99}),
			"reads temporal.server.api.testpilot.v1.InstructionOutcome.status, whose value 99 its enum does not name"},
		"a path of what is no message": {guardPresent(&umpirespb.Operand{Kind: &umpirespb.Operand_Path{Path: &umpirespb.PathOf{Path: "x", Of: guardLiteral("y")}}}), delivered, "reads x of a text, which is no message"},
	} {
		t.Run(name, func(t *testing.T) {
			admitted, err := admits(guarded(test.guard), test.event)
			require.False(t, admitted)
			var refused *GuardError
			require.ErrorAs(t, err, &refused)
			require.Equal(t, &GuardError{Event: 9, Message: test.says}, refused)
		})
	}

	// A source of no known kind takes no event, and says so.
	_, err := admits(&umpirespb.RunEventSource{Script: "controller", Command: "start-activity"}, delivered)
	require.EqualError(t, err, "run event 9: the guard is of a Run Event source of no known kind")
}

// A message that holds itself names one field twice on a path. The value read is the one at the end
// of the path, not the one the field first reaches.
func TestAPathThroughARecursiveMessageReadsItsEnd(t *testing.T) {
	outer := &failurepb.Failure{Message: "outer", Cause: &failurepb.Failure{Message: "middle", Cause: &failurepb.Failure{Message: "inner"}}}
	message, err := valueAt(outer.ProtoReflect(), "cause.cause.message")
	require.NoError(t, err)
	require.Equal(t, value{text: "inner"}, message)
	cause, err := valueAt(outer.ProtoReflect(), "cause.cause")
	require.NoError(t, err)
	require.True(t, proto.Equal(outer.GetCause().GetCause(), cause.message.Interface()))
	absent, err := valueAt(outer.ProtoReflect(), "cause.cause.cause")
	require.NoError(t, err)
	require.Equal(t, value{absent: true}, absent)
}
