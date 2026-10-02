package conformance

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/model/scalav2/goir"
	lowering "go.temporal.io/server/model/scalav2/goir/testpilot"
	"google.golang.org/protobuf/encoding/protojson"
)

// firstReader is the first reader of a guard that can know what is wrong with it.
type firstReader int

const (
	// byNone finds nothing wrong: the guard is well formed for all three readers.
	byNone firstReader = iota
	// byAdmission is goir.Validate, which reads no descriptor.
	byAdmission
	// byLowering is the lowering to a Case, which reads the payload's descriptor.
	byLowering
)

// guardIn is a path of any value.
func guardIn(of *modelirspb.Operand, path string) *modelirspb.Operand {
	return &modelirspb.Operand{Kind: &modelirspb.Operand_Path{Path: &modelirspb.PathOf{Path: path, Of: of}}}
}

// A guard of the Run's own record has three readers: admission, which reads no descriptor; the
// lowering to a Case, which reads the payload's; and Admits, which evaluates the guard on a recorded
// Run. The table below is every form the vocabulary gives a guard, well typed and not: each kind of
// operand, a path nested to any depth, and each kind of written value. One declaration is fed to all
// three. A guard that is well formed for one is well formed for all. One that is not is refused by
// the first reader that can know, at the declaration, and Admits refuses it on every event, whatever
// the event holds, in the same words.
func TestAGuardIsWellFormedForEveryReaderOrForNone(t *testing.T) {
	const (
		pushed  = "evidence fixture.realizations.tally.evidence.pushed: its guard "
		fixture = "model/scalav2/lifter/testdata/lifts/Realizations.scala.fixture:"
		alone   = "; a Run Event's guard reads the event's payload alone"
		unknown = "writes out a value that is no text, flag, number or enum value"
		v1      = "temporal.server.api.testpilot.v1."
	)
	projected := &modelirspb.Operand{Kind: &modelirspb.Operand_Projected{Projected: &modelirspb.Empty{}}}
	run := &modelirspb.Operand{Kind: &modelirspb.Operand_Run{Run: &modelirspb.Empty{}}}
	environment := &modelirspb.Operand{Kind: &modelirspb.Operand_Environment{Environment: "namespace"}}
	learned := &modelirspb.Operand{Kind: &modelirspb.Operand_LearnedValue{LearnedValue: "handle"}}
	attempt, number, delivery := guardPath("activity_attempt"), guardPath("activity_attempt.sdk_attempt"), guardPath("activity_attempt.delivery_id")
	status := guardPath("status")
	succeeded := guardLiteral(protoName("INSTRUCTION_OUTCOME_STATUS_SUCCEEDED"))
	equalTo := func(value *modelirspb.ProtoValue) *modelirspb.Operand {
		return guardEqual(guardPath("detail"), &modelirspb.Operand{Kind: &modelirspb.Operand_Literal{Literal: value}})
	}

	for name, test := range map[string]struct {
		guard *modelirspb.Operand
		first firstReader
		// says is what is wrong, for a guard some reader refuses.
		says string
		// holds is whether a well-formed guard holds of the event that carries everything.
		holds bool
	}{
		// Well formed.
		"a written flag":                      {guardLiteral(true), byNone, "", true},
		"a written flag that is false":        {guardLiteral(false), byNone, "", false},
		"the presence of the payload":         {guardPresent(projected), byNone, "", true},
		"the presence of a message":           {guardPresent(attempt), byNone, "", true},
		"the presence of a written value":     {guardPresent(guardLiteral("x")), byNone, "", true},
		"the presence of a oneof member":      {guardPresent(guardPath("value.value<bool_value>")), byNone, "", true},
		"the presence of another member":      {guardPresent(guardPath("value.value<text_value>")), byNone, "", false},
		"the presence of a member by name":    {guardPresent(guardPath("value.text_value")), byNone, "", false},
		"a text compared":                     {guardEqual(guardPath("protocol_code"), guardLiteral("ok")), byNone, "", true},
		"a number compared":                   {guardEqual(number, guardLiteral(2)), byNone, "", true},
		"a flag compared":                     {guardEqual(guardPath("value.bool_value"), guardLiteral(true)), byNone, "", true},
		"a number compared with another":      {guardEqual(number, guardLiteral(3)), byNone, "", false},
		"a flag compared with the other flag": {guardEqual(guardPath("value.bool_value"), guardLiteral(false)), byNone, "", false},
		"a text compared with another":        {guardEqual(guardPath("protocol_code"), guardLiteral("unavailable")), byNone, "", false},
		"an enum value compared with a name":  {guardEqual(status, succeeded), byNone, "", true},
		"a name compared with an enum value":  {guardEqual(succeeded, status), byNone, "", true},
		"two values of one enum":              {guardEqual(status, status), byNone, "", true},
		"two written values of one type":      {guardEqual(guardLiteral("a"), guardLiteral("b")), byNone, "", false},
		"an order":                            {guardGreater(number, guardLiteral(1)), byNone, "", true},
		"an order the other way":              {guardGreater(guardLiteral(1), number), byNone, "", false},
		"a negation":                          {guardNot(guardEqual(delivery, guardLiteral(""))), byNone, "", true},
		"a conjunction":                       {guardAll(guardPresent(attempt), guardGreater(number, guardLiteral(0))), byNone, "", true},
		"a conjunction in a conjunction":      {guardAll(guardAll(guardLiteral(true)), guardNot(guardLiteral(false))), byNone, "", true},
		"a path of a path":                    {guardGreater(guardIn(attempt, "sdk_attempt"), guardLiteral(0)), byNone, "", true},
		"a path of a path of a path":          {guardPresent(guardIn(guardIn(guardPath("value"), "enum_value"), "name")), byNone, "", false},
		"a member of a oneof of a path":       {guardEqual(guardIn(guardPath("value"), "value<bool_value>"), guardLiteral(true)), byNone, "", true},
		"a path of the payload, of a message": {guardPresent(guardIn(projected, "activity_attempt")), byNone, "", true},

		// What admission can know: the kinds of operand a guard has, the values it writes out, and the
		// types of what it computes from them.
		"the run's id":                      {guardEqual(run, guardLiteral("r")), byAdmission, "reads the run's id" + alone, false},
		"an environment binding":            {guardEqual(environment, guardLiteral("n")), byAdmission, "reads the environment binding namespace" + alone, false},
		"a learned value":                   {guardEqual(learned, guardLiteral("h")), byAdmission, "reads the learned value handle" + alone, false},
		"an operand of no kind":             {&modelirspb.Operand{}, byAdmission, "has an operand of no known kind", false},
		"an operand of no kind, deep":       {guardNot(guardAll(guardPresent(guardIn(&modelirspb.Operand{}, "x")))), byAdmission, "has an operand of no known kind", false},
		"a conjunction of nothing":          {guardAll(), byAdmission, "is a conjunction of no operand", false},
		"a name a Case binds":               {equalTo(&modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_Named{Named: &modelirspb.Name{Prefix: "errand-", Fixture: true}}}), byAdmission, "writes out a name a Case binds" + alone, false},
		"a written value of no kind":        {equalTo(&modelirspb.ProtoValue{}), byAdmission, unknown, false},
		"written bytes":                     {equalTo(&modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_Utf8{Utf8: "x"}}), byAdmission, unknown, false},
		"a written message":                 {equalTo(&modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_Message{Message: &modelirspb.Proto{}}}), byAdmission, unknown, false},
		"a written map":                     {equalTo(&modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_Mapping{Mapping: &modelirspb.ProtoMap{}}}), byAdmission, unknown, false},
		"a written role":                    {equalTo(&modelirspb.ProtoValue{Kind: &modelirspb.ProtoValue_RoleId{RoleId: "frontend"}}), byAdmission, unknown, false},
		"a guard that is a number":          {guardLiteral(1), byAdmission, "is a number, and a guard is a condition", false},
		"a guard that is a text":            {guardLiteral("yes"), byAdmission, "is a text, and a guard is a condition", false},
		"a guard that is a name":            {succeeded, byAdmission, "is an enum value, and a guard is a condition", false},
		"a guard that is the payload":       {projected, byAdmission, "is a message, and a guard is a condition", false},
		"an order of a written text":        {guardGreater(guardLiteral("1"), guardLiteral(0)), byAdmission, "orders a text, and only numbers are ordered", false},
		"an order of a condition":           {guardGreater(guardPresent(attempt), guardLiteral(0)), byAdmission, "orders a condition, and only numbers are ordered", false},
		"an order by a written text":        {guardGreater(number, guardLiteral("0")), byAdmission, "orders a text, and only numbers are ordered", false},
		"a negation of a written number":    {guardNot(guardLiteral(1)), byAdmission, "negates a number, and only a condition is negated", false},
		"a negation of the payload":         {guardNot(projected), byAdmission, "negates a message, and only a condition is negated", false},
		"a conjunction over a written text": {guardAll(guardLiteral(true), guardLiteral("x")), byAdmission, "joins a text, and only conditions are joined", false},
		"two written values of two types":   {guardEqual(guardLiteral("1"), guardLiteral(1)), byAdmission, "compares a text with a number", false},
		"two written names":                 {guardEqual(succeeded, succeeded), byAdmission, "compares two enum values it writes out", false},
		"the payload compared":              {guardEqual(projected, guardLiteral("x")), byAdmission, "compares a message", false},
		"a path of a written text":          {guardPresent(guardIn(guardLiteral("y"), "x")), byAdmission, "reads x of a text, which is no message", false},
		"a path of a condition":             {guardPresent(guardIn(guardPresent(attempt), "x")), byAdmission, "reads x of a condition, which is no message", false},
		"each element of a field":           {guardPresent(guardPath("activity_attempt[*]")), byAdmission, `reads "activity_attempt[*]" of the path activity_attempt[*], and a guard reads a field or oneof<member>`, false},
		"each element, in a path of a path": {guardPresent(guardIn(attempt, "ids[*]")), byAdmission, `reads "ids[*]" of the path ids[*], and a guard reads a field or oneof<member>`, false},
		"an empty path":                     {guardPresent(guardPath("")), byAdmission, "reads an empty path", false},
		"a type error under a negation":     {guardNot(guardAll(guardGreater(guardLiteral("0"), number))), byAdmission, "orders a text, and only numbers are ordered", false},

		// What only the payload's descriptor tells.
		"a field the payload does not have":          {guardPresent(guardPath("activity_attempt.attempt")), byLowering, "reads activity_attempt.attempt, and " + v1 + "ActivityAttempt has no field attempt", false},
		"a path of a path, the inner misspelled":     {guardPresent(guardIn(guardPath("activity_attemtp"), "sdk_attempt")), byLowering, "reads activity_attemtp, and " + v1 + "InstructionOutcome has no field activity_attemtp", false},
		"a path of a path, the outer misspelled":     {guardPresent(guardIn(attempt, "sdk_attemtp")), byLowering, "reads sdk_attemtp, and " + v1 + "ActivityAttempt has no field sdk_attemtp", false},
		"three paths deep, the outermost misspelled": {guardPresent(guardIn(guardIn(guardPath("value"), "enum_value"), "nmae")), byLowering, "reads nmae, and " + v1 + "EnumValue has no field nmae", false},
		"three paths deep, the middle misspelled":    {guardPresent(guardIn(guardIn(guardPath("value"), "enum_vlaue"), "name")), byLowering, "reads enum_vlaue, and " + v1 + "Value has no field enum_vlaue", false},
		"a field of a scalar":                        {guardPresent(guardPath("detail.length")), byLowering, "reads detail.length, and " + v1 + "InstructionOutcome.detail is no message", false},
		"a path of a path that is a text":            {guardPresent(guardIn(guardPath("detail"), "length")), byLowering, "reads length of a text, which is no message", false},
		"a member the oneof does not have":           {guardPresent(guardPath("value.value<nope>")), byLowering, "reads value<nope>, and " + v1 + "Value has no such member", false},
		"a oneof the message does not have":          {guardPresent(guardPath("value.kind<bool_value>")), byLowering, "reads kind<bool_value>, and " + v1 + "Value has no such member", false},
		"several values":                             {guardPresent(guardPath("value.list_value.values")), byLowering, "reads value.list_value.values, and " + v1 + "ValueList.values holds several values", false},
		"bytes":                                      {guardPresent(guardPath("value.bytes_value")), byLowering, "reads " + v1 + "Value.bytes_value, which is of kind bytes", false},
		"a floating point number":                    {guardGreater(guardPath("value.floating_point_value"), guardLiteral(0)), byLowering, "reads " + v1 + "Value.floating_point_value, which is of kind double", false},
		"a guard that is a number read":              {number, byLowering, "is a number, and a guard is a condition", false},
		"a guard that is a message read":             {attempt, byLowering, "is a message, and a guard is a condition", false},
		"an order of a text read":                    {guardGreater(delivery, guardLiteral(0)), byLowering, "orders a text, and only numbers are ordered", false},
		"a negation of a number read":                {guardNot(number), byLowering, "negates a number, and only a condition is negated", false},
		"a conjunction over a text read":             {guardAll(delivery), byLowering, "joins a text, and only conditions are joined", false},
		"a number compared with a text":              {guardEqual(number, guardLiteral("1")), byLowering, "compares a number with a text", false},
		"an enum value compared with a text":         {guardEqual(status, guardLiteral("INSTRUCTION_OUTCOME_STATUS_SUCCEEDED")), byLowering, "compares an enum value with a text", false},
		"a name the enum does not have":              {guardEqual(status, guardLiteral(protoName("INSTRUCTION_OUTCOME_STATUS_NOPE"))), byLowering, "compares a value of " + v1 + "InstructionOutcomeStatus with INSTRUCTION_OUTCOME_STATUS_NOPE, which it does not have", false},
		"a name the enum does not have, first":       {guardEqual(guardLiteral(protoName("INSTRUCTION_OUTCOME_STATUS_NOPE")), status), byLowering, "compares a value of " + v1 + "InstructionOutcomeStatus with INSTRUCTION_OUTCOME_STATUS_NOPE, which it does not have", false},
		"values of two enums":                        {guardEqual(status, guardPath("activity_attempt.response")), byLowering, "compares a value of " + v1 + "InstructionOutcomeStatus with one of " + v1 + "ActivityAttemptResponse", false},
		"two messages compared":                      {guardEqual(attempt, attempt), byLowering, "compares a message", false},
	} {
		t.Run(name, func(t *testing.T) {
			// The declaration: the tally fixture's Run Event kind, under this guard.
			encoded, err := os.ReadFile(filepath.Join("..", "..", "lifter", "testdata", "lifts", "expected", "realizations.json"))
			require.NoError(t, err)
			m := &modelirspb.Model{}
			require.NoError(t, protojson.Unmarshal(encoded, m))
			var source *modelirspb.RunEventSource
			for _, r := range m.GetRealizations() {
				if r.GetName() == "tallyRealization" {
					source = r.GetEvidence()[1].GetRunEvent()
				}
			}
			require.NotNil(t, source)
			source.Guard = test.guard

			// Two events the source's command records: one that carries everything a guard here reads,
			// and one that carries a status alone.
			event := func(outcome *testpilotspb.InstructionOutcome) *testpilotspb.RunEvent {
				return &testpilotspb.RunEvent{Sequence: 9, Kind: eventKinds[source.GetKind()], Payload: &testpilotspb.RunEvent_Outcome{Outcome: outcome},
					Coordinates: &testpilotspb.RunEventCoordinates{EntrypointId: source.GetScript(), InstructionId: source.GetCommand()}}
			}
			full := attemptOf(2, "token-2", testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_OFFERED_COMPLETED)
			full.ProtocolCode, full.Value = "ok", &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: true}}
			bare := &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}

			validated := goir.Validate(m)
			var lowers error
			if validated == nil {
				p, err := lowering.NewProducer(m)
				require.NoError(t, err)
				var l *lowering.Lowering
				if l, lowers = p.Lower("tally.opens", lowering.IdentityFor("temporal.case", "fixture", "tally")); lowers == nil {
					// Testpilot lifts no such source yet, so a well-formed one is a gap and no error.
					require.Equal(t, lowering.NotSupported, l.Standing)
				}
			}

			if test.first == byNone {
				require.NoError(t, validated)
				require.NoError(t, lowers)
				held, err := Admits(source, event(full))
				require.NoError(t, err)
				require.Equal(t, test.holds, held)
				// On an event that lacks a value the guard compares, the guard cannot be evaluated, and
				// that is the one thing Admits can find wrong with a well-formed guard.
				if _, err := Admits(source, event(bare)); err != nil {
					require.ErrorContains(t, err, "an absent value")
				}
				return
			}

			if test.first == byAdmission {
				require.ErrorContains(t, validated, fixture)
				require.ErrorContains(t, validated, pushed+test.says)
			} else {
				require.NoError(t, validated)
				require.ErrorContains(t, lowers, fixture)
				require.ErrorContains(t, lowers, pushed+test.says)
			}
			for _, outcome := range []*testpilotspb.InstructionOutcome{full, bare} {
				held, err := Admits(source, event(outcome))
				require.False(t, held)
				var refused *GuardError
				require.ErrorAs(t, err, &refused)
				require.Equal(t, &GuardError{Event: 9, Message: test.says}, refused)
			}
		})
	}
}
