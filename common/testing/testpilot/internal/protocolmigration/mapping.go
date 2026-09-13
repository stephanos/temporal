package protocolmigration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Object is one JSON object of a fixture while the Mapping runs. Message is the snapshot message
// the object encoded when the baseline was validated, and it stays that name after steps rename
// the object's fields or the message itself, so every step addresses messages by their baseline
// names. Objects that encode no snapshot message (map fields, Any payloads of other packages,
// correlated fixture entries, objects a step builds) carry an empty Message.
type Object struct {
	Message protoreflect.FullName
	Fields  map[string]any
}

func (o *Object) MarshalJSON() ([]byte, error) {
	if o.Fields == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(o.Fields)
}

// ApplyFunc transforms one fixture's JSON tree. fixture is the repository-relative path the
// baseline and regenerated trees share, so a step can confine itself to the fixtures it names.
// Scalars are json.Number, string, bool or nil; arrays are []any; objects are *Object.
type ApplyFunc func(fixture string, tree any) (any, error)

// Step is one declared difference between the baseline protocol and the current one.
type Step struct {
	Name string
	// Requirement is the fn-87 R-ID the step implements.
	Requirement string
	Apply       ApplyFunc
}

// Mapping is the ordered list of Steps from the frozen baseline to the current protocol.
type Mapping []Step

// Declared is the mapping from the baseline snapshot to the current protocol. Every structural
// task appends the steps its change declares; a difference no step declares fails the
// equivalence test. Steps split the names the migration retires, so the retired-vocabulary scan
// that reads this file keeps holding those names everywhere else.
var Declared = Mapping{
	{
		Name: "Run" + "Status becomes RunDisposition", Requirement: "R1",
		Apply: sequence(
			RenameField(protocol+"Run", "status", "disposition"),
			RenameEnumLiteral(protocol+"Run", "disposition", "RUN_"+"STATUS_UNSPECIFIED", "RUN_DISPOSITION_UNSPECIFIED"),
			RenameEnumLiteral(protocol+"Run", "disposition", "RUN_"+"STATUS_COMPLETED", "RUN_DISPOSITION_COMPLETED"),
			RenameEnumLiteral(protocol+"Run", "disposition", "RUN_"+"STATUS_STOPPED_BY_MONITOR", "RUN_DISPOSITION_STOPPED_BY_MONITOR"),
			RenameEnumLiteral(protocol+"Run", "disposition", "RUN_"+"STATUS_INCOMPLETE", "RUN_DISPOSITION_INCOMPLETE"),
		),
	},
	{
		Name: "CorrelatedContract.clauses becomes rules", Requirement: "R1",
		Apply: RenameField(protocol+"CorrelatedContract", "clauses", "rules"),
	},
	{
		Name: "CorrelatedRule.clause" + "_id becomes rule_id", Requirement: "R1",
		Apply: RenameField(protocol+"CorrelatedRule", "clauseId", "ruleId"),
	},
	{
		Name: "the provenance correlated rule key clauseId becomes ruleId", Requirement: "R1",
		Apply: RewriteMessages(protocol+"CaseProvenance", renameCorrelatedRuleKey("clauseId", "ruleId")),
	},
	{
		Name: "CONTRACT_STATE_STATUS_" + "NONTERMINAL becomes CONTRACT_STATE_STATUS_PENDING", Requirement: "R1",
		Apply: RenameEnumLiteral(protocol+"ContractState"+"Definition", "status", "CONTRACT_STATE_STATUS_"+"NONTERMINAL", "CONTRACT_STATE_STATUS_PENDING"),
	},
	{
		Name: "Correlated" + "Value becomes ModelValue", Requirement: "R1",
		Apply: RenameMessage(protocol+"Correlated"+"Value", protocol+"ModelValue"),
	},
	{
		Name: "INSTRUCTION_OUTCOME_STATUS_PROTOCOL_" + "NON_SUCCESS becomes INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE", Requirement: "R1",
		Apply: RenameEnumLiteral(protocol+"InstructionOutcome", "status", "INSTRUCTION_OUTCOME_STATUS_PROTOCOL_"+"NON_SUCCESS", "INSTRUCTION_OUTCOME_STATUS_PROTOCOL_FAILURE"),
	},
	{
		Name: "ContractRule" + "Definition becomes ContractRule", Requirement: "R1",
		Apply: RenameMessage(protocol+"ContractRule"+"Definition", protocol+"ContractRule"),
	},
	{
		Name: "ContractState" + "Definition becomes ContractState", Requirement: "R1",
		Apply: RenameMessage(protocol+"ContractState"+"Definition", protocol+"ContractState"),
	},
	{
		Name: "ContractTransition" + "Definition becomes ContractTransition", Requirement: "R1",
		Apply: RenameMessage(protocol+"ContractTransition"+"Definition", protocol+"ContractTransition"),
	},
	{
		Name: "ContractCapture" + "Definition becomes ContractCapture", Requirement: "R1",
		Apply: RenameMessage(protocol+"ContractCapture"+"Definition", protocol+"ContractCapture"),
	},
	{
		Name: "CorrelatedEvidenceRule.source becomes evidence_source", Requirement: "R1",
		Apply: RenameField(protocol+"CorrelatedEvidenceRule", "source", "evidenceSource"),
	},
	{
		Name: "CorrelatedIdentity.source becomes evidence_source", Requirement: "R1",
		Apply: RenameField(protocol+"CorrelatedIdentity", "source", "evidenceSource"),
	},
	{
		Name: "Response" + "Projection becomes ResponseRead", Requirement: "R1",
		Apply: RenameMessage(protocol+"Response"+"Projection", protocol+"ResponseRead"),
	},
	{
		Name: "InvokeRPC.response" + "_projections becomes response_reads", Requirement: "R1",
		Apply: RenameField(protocol+"InvokeRPC", "response"+"Projections", "responseReads"),
	},
	{
		Name: "Response" + "Projection.source becomes path", Requirement: "R1",
		Apply: RenameField(protocol+"Response"+"Projection", "source", "path"),
	},
	{
		Name: "Projection" + "Target becomes ReadTarget", Requirement: "R1",
		Apply: RenameMessage(protocol+"Projection"+"Target", protocol+"ReadTarget"),
	},
	{
		Name: "Projection" + "Kind becomes ReadCardinality", Requirement: "R1",
		Apply: sequence(
			RenameEnumLiteral(protocol+"Response"+"Projection", "kind", "PROJECTION_"+"KIND_UNSPECIFIED", "READ_CARDINALITY_UNSPECIFIED"),
			RenameEnumLiteral(protocol+"Response"+"Projection", "kind", "PROJECTION_"+"KIND_ONE", "READ_CARDINALITY_ONE"),
			RenameEnumLiteral(protocol+"Response"+"Projection", "kind", "PROJECTION_"+"KIND_EMIT_EACH", "READ_CARDINALITY_EMIT_EACH"),
		),
	},
	{
		Name: "OpaqueCapability" + "Type becomes OpaqueHandleType", Requirement: "R1",
		Apply: RenameMessage(protocol+"OpaqueCapability"+"Type", protocol+"OpaqueHandleType"),
	},
	{
		Name: "the opaque_capability fields become opaque_handle", Requirement: "R1",
		Apply: sequence(
			RenameField(protocol+"SingularType", "opaqueCapability", "opaqueHandle"),
			RenameField(protocol+"Slot"+"Definition", "opaqueCapability", "opaqueHandle"),
		),
	},
	{
		Name: "capability" + "_slot_id becomes handle_slot_id", Requirement: "R1",
		Apply: sequence(
			RenameField(protocol+"CompleteNexusOperation", "capability"+"SlotId", "handleSlotId"),
			RenameField(protocol+"RespondNexus", "capability"+"SlotId", "handleSlotId"),
		),
	},
	{
		Name: "InvokeRPC becomes InvokeRpc", Requirement: "R1",
		Apply: RenameMessage(protocol+"InvokeRPC", protocol+"InvokeRpc"),
	},
	valueArm("text", "textValue"),
	valueArm("natural", "natural"+"Value"),
	valueArm("signedInteger", "signedIntegerValue"),
	valueArm("unsignedInteger", "unsignedIntegerValue"),
	valueArm("floatingPoint", "floatingPointValue"),
	{
		Name: "Role" + "Definition becomes Role", Requirement: "R1",
		Apply: RenameMessage(protocol+"Role"+"Definition", protocol+"Role"),
	},
	{
		Name: "Slot" + "Definition becomes Slot", Requirement: "R1",
		Apply: RenameMessage(protocol+"Slot"+"Definition", protocol+"Slot"),
	},
	{
		Name: "Observation" + "Definition becomes Observation", Requirement: "R1",
		Apply: RenameMessage(protocol+"Observation"+"Definition", protocol+"Observation"),
	},
	{
		Name: "Entrypoint" + "Definition becomes Entrypoint", Requirement: "R1",
		Apply: RenameMessage(protocol+"Entrypoint"+"Definition", protocol+"Entrypoint"),
	},
	{
		Name: "Cleanup" + "Definition becomes Cleanup", Requirement: "R1",
		Apply: RenameMessage(protocol+"Cleanup"+"Definition", protocol+"Cleanup"),
	},
	{
		Name: "Instruction" + "Definition becomes InstructionNode", Requirement: "R1",
		Apply: RenameMessage(protocol+"Instruction"+"Definition", protocol+"InstructionNode"),
	},
	{
		Name: "Instruction" + "Ref becomes InstructionReference", Requirement: "R1",
		Apply: RenameMessage(protocol+"Instruction"+"Ref", protocol+"InstructionReference"),
	},
	{
		Name: "ComparisonOperator gains EQUAL and NOT_EQUAL before the ordering operators", Requirement: "R3",
		Apply: sequence(renumberComparisons(protocol+"Program"+"CompareExpression"), renumberComparisons(protocol+"Contract"+"CompareExpression")),
	},
	{
		Name: "Program" + "Expression and Contract" + "Expression become Expression over one Reference", Requirement: "R3",
		Apply: sequence(
			RewriteMessages(protocol+"Program"+"Expression", rewriteExpression),
			RewriteMessages(protocol+"Contract"+"Expression", rewriteExpression),
			RenameField(protocol+"Program"+"PathExpression", "source", "operand"),
			RenameField(protocol+"Contract"+"PathExpression", "source", "operand"),
		),
	},
	{
		Name: "ContractCaptureAssignment.observation becomes observation_id", Requirement: "R3",
		Apply: RewriteMessages(protocol+"ContractCaptureAssignment", captureAssignmentObservation),
	},
	{
		Name: "CorrelatedCapture" + "Ref becomes CorrelatedCaptureReference", Requirement: "R3",
		Apply: RenameMessage(protocol+"CorrelatedCapture"+"Ref", protocol+"CorrelatedCaptureReference"),
	},
	{
		Name: "ResponseRead.kind becomes cardinality", Requirement: "R1",
		Apply: RenameField(protocol+"Response"+"Projection", "kind", "cardinality"),
	},
	{
		Name: "Correlated" + "Predicate becomes a present or EQUAL step condition over a correlated step reference", Requirement: "R3, R5",
		Apply: RewriteMessages(protocol+"Correlated"+"Predicate", rewriteStepCondition),
	},
	{
		Name: "Correlated" + "Operand becomes a literal, evidence field or correlated capture Expression", Requirement: "R3",
		Apply: RewriteMessages(protocol+"Correlated"+"Operand", rewriteCorrelatedOperand),
	},
	{
		Name: "Correlated" + "Comparison becomes CompareExpression and Correlated" + "Correlation becomes Expression", Requirement: "R3",
		Apply: sequence(
			RenameEnumLiteral(protocol+"Correlated"+"Comparison", "operator", "CORRELATED_"+"COMPARISON_OPERATOR_EQUAL", "COMPARISON_OPERATOR_EQUAL"),
			RenameEnumLiteral(protocol+"Correlated"+"Comparison", "operator", "CORRELATED_"+"COMPARISON_OPERATOR_NOT_EQUAL", "COMPARISON_OPERATOR_NOT_EQUAL"),
			RewriteMessages(protocol+"Correlated"+"Correlation", rewriteCorrelation),
		),
	},
	{
		Name: "CorrelatedEvidenceRule.guard becomes an Expression over the projected value and guard_" + "equals_text folds into EQUAL", Requirement: "R3, R6",
		Apply: RewriteMessages(protocol+"CorrelatedEvidenceRule", rewriteEvidenceGuard),
	},
	{
		Name: "the Run Event fault coordinates become paths from the payload reference into fault_injected", Requirement: "R4",
		Apply: RewriteMessages(protocol+"Contract"+"Expression", rewriteFaultCoordinate),
	},
	{
		Name: "Contract" + "Deadline becomes Deadline, its one positive bound under the bound oneof", Requirement: "R5",
		Apply: sequence(
			RewriteMessages(protocol+"Contract"+"Deadline", rewriteDeadlineBound),
			RenameMessage(protocol+"Contract"+"Deadline", protocol+"Deadline"),
		),
	},
	{
		Name: "Contract" + "CaptureType becomes SingularType", Requirement: "R6",
		Apply: RenameMessage(protocol+"Contract"+"CaptureType", protocol+"SingularType"),
	},
	{
		Name: "CorrelatedContract.version is removed", Requirement: "R6",
		Apply: DropField(protocol+"CorrelatedContract", "version", func(_ string, object *Object) error {
			if literalText(object.Fields["version"]) != "1" {
				return fmt.Errorf("version is %v, want 1", object.Fields["version"])
			}
			return nil
		}),
	},
	{
		Name: "Correlated" + "Binding and CorrelatedEvidence" + "Field become NamedValue", Requirement: "R6",
		Apply: sequence(
			RewriteMessages(protocol+"Correlated"+"Binding", rewriteScopeValue),
			RenameMessage(protocol+"Correlated"+"Binding", protocol+"NamedValue"),
			RenameMessage(protocol+"CorrelatedEvidence"+"Field", protocol+"NamedValue"),
		),
	},
	{
		Name: "CorrelatedEvidence" + "Binding becomes NamedExpression over a text literal or a projected path", Requirement: "R6",
		Apply: sequence(
			RewriteMessages(protocol+"CorrelatedEvidence"+"Binding", rewriteEvidenceBinding),
			RenameMessage(protocol+"CorrelatedEvidence"+"Binding", protocol+"NamedExpression"),
		),
	},
	{
		Name: "Value.natural" + "_value becomes unsigned_integer_value and SCALAR_KIND_" + "NATURAL becomes SCALAR_KIND_UINT64", Requirement: "R6",
		Apply: sequence(
			RewriteMessages(protocol+"Value", rewriteNaturalValue),
			RewriteMessages(protocol+"ScalarType", rewriteScalarKind),
		),
	},
	{
		Name: "SingularType.opaque_handle is removed, so only a Slot holds an opaque handle", Requirement: "R6",
		Apply: RewriteMessages(protocol+"SingularType", func(_ string, object *Object) (any, error) {
			if _, opaque := object.Fields["opaqueHandle"]; opaque {
				return nil, errors.New("a singular type carries an opaque handle, which only a Slot may hold")
			}
			return object, nil
		}),
	},
	{
		Name: "Program, Contract and correlated limits move to the Profile, and InstructionLimits keeps only its timeout and attempts", Requirement: "R12",
		Apply: sequence(
			DropField(protocol+"Program", "limits", checkMovedLimits),
			DropField(protocol+"Contract", "limits", checkMovedLimits),
			DropField(protocol+"CorrelatedContract", "limits", checkMovedLimits),
			DropField(protocol+"InstructionLimits", "maxEmittedEvents", checkMovedBound("maxEmittedEvents")),
			DropField(protocol+"InstructionLimits", "maxResponseBytes", checkMovedBound("maxResponseBytes")),
		),
	},
	{
		Name: "Instruction" + "Definition.dependencies becomes after, written only where it is not the previous instruction, and the success guard becomes the default", Requirement: "R9",
		Apply: sequence(
			RewriteMessages(protocol+"Entrypoint"+"Definition", defaultInstructionOrder),
			RewriteMessages(protocol+"Cleanup"+"Definition", defaultInstructionOrder),
		),
	},
	{
		Name: "Program.environment, activation reservations and instruction outcomes are derived, and instruction limits equal to the Profile defaults are omitted", Requirement: "R10",
		Apply: RewriteMessages(protocol+"Program", deriveProgramDeclarations),
	},
	{
		Name: "the opaque provenance payload becomes typed Definition, source, Known Gap and correlated rule rows, and its CASE_ kind literals take their enums' prefixes", Requirement: "R13",
		Apply: RewriteMessages(protocol+"CaseProvenance", liftProvenanceRows),
	},
}

// defaultInstructionOrder rewrites one entrypoint's instructions against the default order: an
// instruction runs after the one before it, and only when every instruction it runs after succeeded.
// A dependency list equal to that predecessor is dropped and any other list becomes after, an empty
// one included, since an absent after now means the predecessor. A guard equal to the success of
// every dependency is dropped; a node that had dependencies and no guard ran regardless of their
// outcomes, so it gains a true guard; any other guard stays.
func defaultInstructionOrder(_ string, object *Object) (any, error) {
	entrypointID, _ := object.Fields["entrypointId"].(string)
	instructions, _ := object.Fields["instructions"].([]any)
	previous := ""
	for index, element := range instructions {
		node, ok := element.(*Object)
		if !ok {
			return nil, fmt.Errorf("instruction %d is not an object", index)
		}
		dependencies, err := dependencyIDs(entrypointID, node.Fields["dependencies"])
		if err != nil {
			return nil, fmt.Errorf("instruction %d: %w", index, err)
		}
		delete(node.Fields, "dependencies")
		guard, guarded := node.Fields["guard"]
		switch {
		case guarded:
			succeeded, err := isSuccessGuard(entrypointID, dependencies, guard)
			if err != nil {
				return nil, fmt.Errorf("instruction %d guard: %w", index, err)
			}
			if succeeded {
				delete(node.Fields, "guard")
			}
		case len(dependencies) > 0:
			node.Fields["guard"] = &Object{Fields: map[string]any{"literal": &Object{Fields: map[string]any{"boolValue": true}}}}
		default:
		}
		if index == 0 && len(dependencies) != 0 || index > 0 && !slices.Equal(dependencies, []string{previous}) {
			references := make([]any, len(dependencies))
			for k, id := range dependencies {
				references[k] = &Object{Fields: map[string]any{"entrypointId": entrypointID, "instructionId": id}}
			}
			node.Fields["after"] = &Object{Fields: map[string]any{"instructions": references}}
		}
		previous, _ = node.Fields["instructionId"].(string)
	}
	return object, nil
}

// dependencyIDs reads a baseline dependency list in order. Baseline preparation admitted only
// references into the entrypoint that declares them.
func dependencyIDs(entrypointID string, value any) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	list, ok := value.([]any)
	if !ok {
		return nil, errors.New("dependencies is not a list")
	}
	ids := make([]string, len(list))
	for k, element := range list {
		reference, isObject := element.(*Object)
		if !isObject {
			return nil, fmt.Errorf("dependency %d is not an object", k)
		}
		for key := range reference.Fields {
			if key != "entrypointId" && key != "instructionId" {
				return nil, fmt.Errorf("dependency %d carries %q", k, key)
			}
		}
		if reference.Fields["entrypointId"] != entrypointID {
			return nil, fmt.Errorf("dependency %d names entrypoint %v, not %q", k, reference.Fields["entrypointId"], entrypointID)
		}
		ids[k], _ = reference.Fields["instructionId"].(string)
	}
	return ids, nil
}

// isSuccessGuard reports whether guard is the default the current protocol derives for dependencies:
// for one dependency all[present(status), status == SUCCEEDED], and for several the all of those.
func isSuccessGuard(entrypointID string, dependencies []string, guard any) (bool, error) {
	if len(dependencies) == 0 {
		return false, nil
	}
	encoded, err := json.Marshal(guard)
	if err != nil {
		return false, err
	}
	var decoded testpilotspb.Expression
	if err := protojson.Unmarshal(encoded, &decoded); err != nil {
		return false, err
	}
	succeeded := func(id string) *testpilotspb.Expression {
		status := func() *testpilotspb.Expression {
			return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_Outcome{Outcome: &testpilotspb.InstructionOutcomeReference{
				Instruction: &testpilotspb.InstructionReference{EntrypointId: entrypointID, InstructionId: id},
				Field:       testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS,
			}}}}}
		}
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{
			{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: status()}}},
			{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
				Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL,
				Left:     status(),
				Right:    &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Number: int32(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)}}}}},
			}}},
		}}}}
	}
	want := succeeded(dependencies[0])
	if len(dependencies) > 1 {
		operands := make([]*testpilotspb.Expression, len(dependencies))
		for k, id := range dependencies {
			operands[k] = succeeded(id)
		}
		want = &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: operands}}}
	}
	return proto.Equal(&decoded, want), nil
}

// rewriteNaturalValue moves a baseline natural into the unsigned integer arm, which spells it with
// the same canonical base-10 text. The baseline natural was unbounded, so a value above the unsigned
// 64-bit range fails rather than being carried into an arm that cannot hold it.
func rewriteNaturalValue(_ string, object *Object) (any, error) {
	value, ok := object.Fields["natural"+"Value"]
	if !ok {
		return object, nil
	}
	text, isText := value.(string)
	parsed, err := strconv.ParseUint(text, 10, 64)
	if !isText || err != nil || strconv.FormatUint(parsed, 10) != text {
		return nil, fmt.Errorf("natural value %v is not a canonical unsigned 64-bit integer", value)
	}
	if _, clashes := object.Fields["unsignedIntegerValue"]; clashes {
		return nil, errors.New("value carries a natural and an unsigned integer")
	}
	delete(object.Fields, "natural"+"Value")
	object.Fields["unsignedIntegerValue"] = text
	return object, nil
}

// rewriteScalarKind renames a baseline NATURAL kind, by name or number, to UINT64 and moves every
// later baseline kind number one down, the numbering the removal leaves dense. A kind spelled by any
// other name keeps its name.
func rewriteScalarKind(_ string, object *Object) (any, error) {
	kind, ok := object.Fields["kind"]
	if !ok {
		return object, nil
	}
	text := literalText(kind)
	if text == "SCALAR_KIND_"+"NATURAL" || text == "2" {
		object.Fields["kind"] = "SCALAR_KIND_UINT64"
		return object, nil
	}
	if number, err := strconv.Atoi(text); err == nil && number > 2 {
		object.Fields["kind"] = json.Number(strconv.Itoa(number - 1))
	}
	return object, nil
}

// rewriteDeadlineBound checks one baseline deadline against the bound oneof. The baseline admitted
// exactly one positive bound, which ProtoJSON already spells as the oneof arm of the same name, so the
// step drops a zero-valued bound and refuses a deadline with no positive bound or two.
func rewriteDeadlineBound(_ string, object *Object) (any, error) {
	bounds := 0
	for key, value := range object.Fields {
		switch key {
		case "violationStateId":
		case "ruleEvents", "elapsedMilliseconds":
			bound, err := strconv.ParseInt(literalText(value), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("deadline %s is %v: %w", key, value, err)
			}
			switch {
			case bound == 0:
				delete(object.Fields, key)
			case bound > 0:
				bounds++
			default:
				return nil, fmt.Errorf("deadline %s is negative", key)
			}
		default:
			return nil, fmt.Errorf("deadline carries %q", key)
		}
	}
	if bounds != 1 {
		return nil, fmt.Errorf("deadline carries %d positive bounds, want 1", bounds)
	}
	return object, nil
}

// rewriteScopeValue turns the text of one baseline scope binding into a text Value. The baseline
// admitted only a non-empty text, so an absent or non-string value fails rather than becoming an
// absent Value.
func rewriteScopeValue(_ string, object *Object) (any, error) {
	for key := range object.Fields {
		if key != "fieldId" && key != "value" {
			return nil, fmt.Errorf("scope binding carries %q", key)
		}
	}
	text, isText := object.Fields["value"].(string)
	if !isText || text == "" {
		return nil, fmt.Errorf("scope binding value is %v, want a non-empty string", object.Fields["value"])
	}
	object.Fields["value"] = &Object{Fields: map[string]any{"textValue": text}}
	return object, nil
}

// rewriteEvidenceBinding turns one baseline lift binding into a NamedExpression: a literal becomes a
// text literal and a path a path over the projected value. It requires exactly one supply.
func rewriteEvidenceBinding(_ string, object *Object) (any, error) {
	literal, hasLiteral := object.Fields["literal"]
	path, hasPath := object.Fields["path"]
	for key := range object.Fields {
		if key != "fieldId" && key != "literal" && key != "path" {
			return nil, fmt.Errorf("evidence binding carries %q", key)
		}
	}
	var value *Object
	switch {
	case hasLiteral && !hasPath:
		text, isText := literal.(string)
		if !isText {
			return nil, errors.New("evidence binding literal is not a string")
		}
		value = textLiteral(text)
	case hasPath && !hasLiteral:
		fieldPath, isObject := path.(*Object)
		if !isObject {
			return nil, errors.New("evidence binding path is not an object")
		}
		read, err := projectedPath(fieldPath)
		if err != nil {
			return nil, err
		}
		value = read
	default:
		return nil, errors.New("evidence binding carries no single supply")
	}
	delete(object.Fields, "literal")
	delete(object.Fields, "path")
	object.Fields["value"] = value
	return object, nil
}

// faultCoordinates maps each baseline fault coordinate literal, by name or number, to the
// FaultInjected field a payload path reads instead.
var faultCoordinates = map[string]string{
	"RUN_EVENT_FIELD_" + "FAULT_ROLE_ID": "role_id", "10": "role_id",
	"RUN_EVENT_FIELD_" + "FAULT_KIND": "kind", "11": "kind",
}

// rewriteFaultCoordinate turns a Contract expression reading a fault coordinate of the Run Event
// into a path from the payload reference through fault_injected. It runs after the expression step,
// so the object is already a Reference; a reference to any other coordinate is left as it is.
func rewriteFaultCoordinate(_ string, object *Object) (any, error) {
	reference, isReference := object.Fields["reference"].(*Object)
	if !isReference {
		return object, nil
	}
	runEvent, isRunEvent := reference.Fields["runEvent"].(*Object)
	if !isRunEvent {
		return object, nil
	}
	field, fault := faultCoordinates[literalText(runEvent.Fields["field"])]
	if !fault {
		return object, nil
	}
	if len(runEvent.Fields) != 1 {
		return nil, fmt.Errorf("fault coordinate reference carries %d keys, want 1", len(runEvent.Fields))
	}
	segment := func(name string) *Object { return &Object{Fields: map[string]any{"field": name}} }
	object.Fields = map[string]any{"path": &Object{Fields: map[string]any{
		"operand": &Object{Fields: map[string]any{"reference": &Object{Fields: map[string]any{"runEvent": &Object{Fields: map[string]any{"payload": &Object{Fields: map[string]any{}}}}}}}},
		"path":    &Object{Fields: map[string]any{"segments": []any{segment("fault_injected"), segment(field)}}},
	}}}
	return object, nil
}

// stepFields maps each baseline predicate field literal, by name or number, to its step field.
var stepFields = map[string]string{
	"CORRELATED_" + "PREDICATE_FIELD_ACTION": "CORRELATED_STEP_FIELD_ACTION", "1": "CORRELATED_STEP_FIELD_ACTION",
	"CORRELATED_" + "PREDICATE_FIELD_OUTCOME": "CORRELATED_STEP_FIELD_OUTCOME", "2": "CORRELATED_STEP_FIELD_OUTCOME",
	"CORRELATED_" + "PREDICATE_FIELD_STATE": "CORRELATED_STEP_FIELD_STATE", "3": "CORRELATED_STEP_FIELD_STATE",
	"CORRELATED_" + "PREDICATE_FIELD_FACT": "CORRELATED_STEP_FIELD_FACT", "4": "CORRELATED_STEP_FIELD_FACT",
}

// rewriteStepCondition rewrites one baseline predicate into a step condition. The baseline admitted
// only a true presence constraint or a text equality, so a false presence, both constraints or a key
// outside the predicate's three fails rather than being dropped.
func rewriteStepCondition(_ string, object *Object) (any, error) {
	step := &Object{Fields: map[string]any{}}
	if id, ok := object.Fields["definitionId"]; ok {
		step.Fields["definitionId"] = id
	}
	if field, ok := object.Fields["field"]; ok {
		name, known := stepFields[literalText(field)]
		if !known {
			return nil, fmt.Errorf("predicate field %v has no step field", field)
		}
		step.Fields["field"] = name
	}
	reference := &Object{Fields: map[string]any{"reference": &Object{Fields: map[string]any{"correlatedStep": step}}}}
	present, hasPresent := object.Fields["present"]
	text, hasText := object.Fields["equalsText"]
	for key := range object.Fields {
		if key != "definitionId" && key != "field" && key != "present" && key != "equalsText" {
			return nil, fmt.Errorf("predicate carries %q", key)
		}
	}
	switch {
	case hasPresent && !hasText:
		if present != true {
			return nil, fmt.Errorf("predicate presence is %v, want true", present)
		}
		return &Object{Fields: map[string]any{"present": &Object{Fields: map[string]any{"operand": reference}}}}, nil
	case hasText && !hasPresent:
		literal, isText := text.(string)
		if !isText {
			return nil, errors.New("predicate equalsText is not a string")
		}
		return &Object{Fields: map[string]any{"compare": &Object{Fields: map[string]any{
			"operator": "COMPARISON_OPERATOR_EQUAL",
			"left":     reference,
			"right":    textLiteral(literal),
		}}}}, nil
	default:
		return nil, errors.New("predicate carries no single constraint")
	}
}

// rewriteCorrelatedOperand rewrites one baseline operand into the Expression it reads.
func rewriteCorrelatedOperand(_ string, object *Object) (any, error) {
	if len(object.Fields) != 1 {
		return nil, fmt.Errorf("operand carries %d arms, want 1", len(object.Fields))
	}
	for name, value := range object.Fields {
		switch name {
		case "literal":
			return &Object{Fields: map[string]any{"literal": value}}, nil
		case "fieldId":
			return &Object{Fields: map[string]any{"reference": &Object{Fields: map[string]any{"evidenceFieldId": value}}}}, nil
		case "capture":
			return &Object{Fields: map[string]any{"reference": &Object{Fields: map[string]any{"correlatedCapture": value}}}}, nil
		default:
			return nil, fmt.Errorf("operand arm %q is not in the baseline vocabulary", name)
		}
	}
	return nil, errors.New("unreachable")
}

// rewriteCorrelation rewrites one baseline correlation into Expression: a predicate is already the
// step condition an earlier step built, a comparison is a compare, and a group keeps its operands.
func rewriteCorrelation(_ string, object *Object) (any, error) {
	if len(object.Fields) != 1 {
		return nil, fmt.Errorf("correlation carries %d arms, want 1", len(object.Fields))
	}
	for name, value := range object.Fields {
		switch name {
		case "predicate":
			return value, nil
		case "comparison":
			return &Object{Fields: map[string]any{"compare": value}}, nil
		case "all", "any":
			return object, nil
		default:
			return nil, fmt.Errorf("correlation arm %q is not in the baseline vocabulary", name)
		}
	}
	return nil, errors.New("unreachable")
}

// rewriteEvidenceGuard turns a lift guard path into present(path(projected_value, guard)) and, when
// the rule also required a text, conjoins an EQUAL comparison of the same path with that text.
func rewriteEvidenceGuard(_ string, object *Object) (any, error) {
	guard, ok := object.Fields["guard"].(*Object)
	if !ok {
		return nil, errors.New("evidence rule carries no guard path")
	}
	read, err := projectedPath(guard)
	if err != nil {
		return nil, err
	}
	resolves := &Object{Fields: map[string]any{"present": &Object{Fields: map[string]any{"operand": read}}}}
	object.Fields["guard"] = resolves
	equals, hasEquals := object.Fields["guard"+"EqualsText"]
	if !hasEquals {
		return object, nil
	}
	text, isText := equals.(string)
	if !isText {
		return nil, errors.New("evidence rule guard text is not a string")
	}
	compared, err := projectedPath(guard)
	if err != nil {
		return nil, err
	}
	delete(object.Fields, "guard"+"EqualsText")
	object.Fields["guard"] = &Object{Fields: map[string]any{"all": &Object{Fields: map[string]any{"operands": []any{
		resolves,
		&Object{Fields: map[string]any{"compare": &Object{Fields: map[string]any{
			"operator": "COMPARISON_OPERATOR_EQUAL",
			"left":     compared,
			"right":    textLiteral(text),
		}}}},
	}}}}}
	return object, nil
}

// projectedPath reads an independent copy of path out of the projected value.
func projectedPath(path *Object) (*Object, error) {
	copied, err := cloneTree(path)
	if err != nil {
		return nil, err
	}
	return &Object{Fields: map[string]any{"path": &Object{Fields: map[string]any{
		"operand": &Object{Fields: map[string]any{"reference": &Object{Fields: map[string]any{"projectedValue": &Object{Fields: map[string]any{}}}}}},
		"path":    copied,
	}}}}, nil
}

func textLiteral(text string) *Object {
	return &Object{Fields: map[string]any{"literal": &Object{Fields: map[string]any{"textValue": text}}}}
}

// literalText spells an enum literal the way RenameEnumLiteral compares it.
func literalText(value any) string {
	switch literal := value.(type) {
	case json.Number:
		return string(literal)
	case string:
		return literal
	default:
		return ""
	}
}

// cloneTree deep-copies a mapped JSON tree, so one baseline value can appear twice.
func cloneTree(value any) (any, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return decodeJSON(encoded)
}

// renumberComparisons moves the ordering operators two numbers up, highest first so no literal is
// renumbered twice. A literal spelled by name keeps its name.
func renumberComparisons(message protoreflect.FullName) ApplyFunc {
	return sequence(
		RenameEnumLiteral(message, "operator", "4", "6"),
		RenameEnumLiteral(message, "operator", "3", "5"),
		RenameEnumLiteral(message, "operator", "2", "4"),
		RenameEnumLiteral(message, "operator", "1", "3"),
	)
}

// expressionReferences maps each baseline reference arm to its Reference arm, and names the one key
// the baseline reference message may carry when its arm holds that key's value alone.
var expressionReferences = map[string]struct{ arm, key string }{
	"slot":        {arm: "slotId", key: "slotId"},
	"observation": {arm: "observationId", key: "observationId"},
	"capture":     {arm: "captureId", key: "captureId"},
	"environment": {arm: "environmentBindingId", key: "bindingId"},
	"outcome":     {arm: "outcome"},
	"run":         {arm: "run"},
	"runEvent":    {arm: "runEvent"},
}

// rewriteExpression rewrites one baseline Program or Contract expression into Expression. It
// requires exactly one arm, and a reference whose value is one identifier must carry nothing else,
// so no field is dropped silently.
func rewriteExpression(_ string, object *Object) (any, error) {
	if len(object.Fields) != 1 {
		return nil, fmt.Errorf("expression carries %d arms, want 1", len(object.Fields))
	}
	for name, value := range object.Fields {
		switch name {
		case "literal", "path", "present", "compare", "all", "any":
			return object, nil
		case "negation":
			object.Fields = map[string]any{"not": value}
			return object, nil
		case "equals":
			operands, ok := value.(*Object)
			if !ok {
				return nil, errors.New("equals is not an object")
			}
			if _, clashes := operands.Fields["operator"]; clashes {
				return nil, errors.New("equals carries an operator")
			}
			operands.Fields["operator"] = "COMPARISON_OPERATOR_EQUAL"
			object.Fields = map[string]any{"compare": operands}
			return object, nil
		}
		reference, known := expressionReferences[name]
		if !known {
			return nil, fmt.Errorf("expression arm %q is not in the baseline vocabulary", name)
		}
		referenced, ok := value.(*Object)
		if !ok {
			return nil, fmt.Errorf("%s is not an object", name)
		}
		if reference.key == "" {
			object.Fields = map[string]any{"reference": &Object{Fields: map[string]any{reference.arm: referenced}}}
			return object, nil
		}
		identifier, err := soleString(name, referenced, reference.key)
		if err != nil {
			return nil, err
		}
		object.Fields = map[string]any{"reference": &Object{Fields: map[string]any{reference.arm: identifier}}}
		return object, nil
	}
	return nil, errors.New("unreachable")
}

// captureAssignmentObservation moves the observation reference of a capture assignment to its id.
func captureAssignmentObservation(_ string, object *Object) (any, error) {
	value, ok := object.Fields["observation"]
	if !ok {
		return object, nil
	}
	referenced, isObject := value.(*Object)
	if !isObject {
		return nil, errors.New("observation is not an object")
	}
	identifier, err := soleString("observation", referenced, "observationId")
	if err != nil {
		return nil, err
	}
	if identifier == "" {
		return nil, errors.New("observation names no Observation")
	}
	delete(object.Fields, "observation")
	object.Fields["observationId"] = identifier
	return object, nil
}

// soleString reads the one string key a reference object may carry; an omitted key is the empty
// identifier ProtoJSON leaves out.
func soleString(name string, object *Object, key string) (string, error) {
	for field := range object.Fields {
		if field != key {
			return "", fmt.Errorf("%s carries %q beside %q", name, field, key)
		}
	}
	value, present := object.Fields[key]
	if !present {
		return "", nil
	}
	text, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("%s.%s is not a string", name, key)
	}
	return text, nil
}

// valueArm renames one Value oneof arm. The arm names are ordinary words other messages may spell
// as field names, so the step is scoped to objects that encoded a baseline Value.
func valueArm(from, to string) Step {
	return Step{
		Name: "Value." + from + " becomes " + to, Requirement: "R1",
		Apply: RenameField(protocol+"Value", from, to),
	}
}

const protocol = protoreflect.FullName("temporal.server.api.testpilot.v1.")

func sequence(applies ...ApplyFunc) ApplyFunc {
	return func(fixture string, tree any) (any, error) {
		for _, apply := range applies {
			mapped, err := apply(fixture, tree)
			if err != nil {
				return nil, err
			}
			tree = mapped
		}
		return tree, nil
	}
}

// renameCorrelatedRuleKey renames one key of every correlatedRules entry in the opaque Umpire
// payload a CaseProvenance carries. The comparison pins the payload's bytes, so the key is renamed
// in place, and the result must decode to the original payload with only that key moved.
func renameCorrelatedRuleKey(from, to string) func(fixture string, object *Object) (any, error) {
	return func(_ string, object *Object) (any, error) {
		encoded, ok := object.Fields[opaqueBytesKey].(string)
		if !ok {
			return object, nil
		}
		payload, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", opaqueBytesKey, err)
		}
		key, renamedKey := []byte(`"`+from+`":`), []byte(`"`+to+`":`)
		// Producers other than Umpire write payloads that need not be JSON.
		if !bytes.Contains(payload, key) {
			return object, nil
		}
		var want map[string]any
		if err := json.Unmarshal(payload, &want); err != nil {
			return nil, fmt.Errorf("%s: %w", opaqueBytesKey, err)
		}
		rules, _ := want["correlatedRules"].([]any)
		for index, entry := range rules {
			rule, isObject := entry.(map[string]any)
			value, found := rule[from]
			if _, clashes := rule[to]; !isObject || !found || clashes {
				return nil, fmt.Errorf("%s correlatedRules[%d] does not carry %q alone", opaqueBytesKey, index, from)
			}
			delete(rule, from)
			rule[to] = value
		}
		renamed := bytes.ReplaceAll(payload, key, renamedKey)
		var got map[string]any
		if err := json.Unmarshal(renamed, &got); err != nil {
			return nil, fmt.Errorf("%s after renaming %q: %w", opaqueBytesKey, from, err)
		}
		if !reflect.DeepEqual(want, got) {
			return nil, fmt.Errorf("%s spells %q outside its correlatedRules entries", opaqueBytesKey, from)
		}
		object.Fields[opaqueBytesKey] = base64.StdEncoding.EncodeToString(renamed)
		return object, nil
	}
}

func (m Mapping) apply(fixture string, tree any) (any, error) {
	for _, step := range m {
		mapped, err := step.Apply(fixture, tree)
		if err != nil {
			return nil, fmt.Errorf("fixture %s: step %q (%s): %w", fixture, step.Name, step.Requirement, err)
		}
		tree = mapped
	}
	return tree, nil
}

// RewriteMessages replaces every object that encoded message with rewrite's result. Nested
// messages are rewritten before the objects that contain them.
func RewriteMessages(message protoreflect.FullName, rewrite func(fixture string, object *Object) (any, error)) ApplyFunc {
	return func(fixture string, tree any) (any, error) {
		return rewriteTree(tree, func(value any) (any, error) {
			object, ok := value.(*Object)
			if !ok || object.Message != message {
				return value, nil
			}
			return rewrite(fixture, object)
		})
	}
}

// RenameField moves field from to field to in every object that encoded message.
func RenameField(message protoreflect.FullName, from, to string) ApplyFunc {
	return RewriteMessages(message, func(_ string, object *Object) (any, error) {
		value, ok := object.Fields[from]
		if !ok {
			return object, nil
		}
		if _, exists := object.Fields[to]; exists {
			return nil, fmt.Errorf("%s carries both %q and %q", message, from, to)
		}
		delete(object.Fields, from)
		object.Fields[to] = value
		return object, nil
	})
}

// RenameMessage renames message from to to where a fixture spells a message name: the type URL of
// a google.protobuf.Any payload. Objects keep the baseline Message they were annotated with.
func RenameMessage(from, to protoreflect.FullName) ApplyFunc {
	return RewriteMessages(anyMessage, func(_ string, object *Object) (any, error) {
		url, ok := object.Fields["@type"].(string)
		if !ok {
			return object, nil
		}
		separator := strings.LastIndex(url, "/")
		if protoreflect.FullName(url[separator+1:]) == from {
			object.Fields["@type"] = url[:separator+1] + string(to)
		}
		return object, nil
	})
}

// RenameEnumLiteral replaces the enum literal from with to in field of every object that encoded
// message, whether the field holds one literal or a list. A literal is its number or its value
// name; to is written as a JSON number when it is one.
func RenameEnumLiteral(message protoreflect.FullName, field, from, to string) ApplyFunc {
	return RewriteMessages(message, func(_ string, object *Object) (any, error) {
		value, ok := object.Fields[field]
		if !ok {
			return object, nil
		}
		if list, isList := value.([]any); isList {
			for index, element := range list {
				list[index] = renameLiteral(element, from, to)
			}
			return object, nil
		}
		object.Fields[field] = renameLiteral(value, from, to)
		return object, nil
	})
}

func renameLiteral(value any, from, to string) any {
	var text string
	switch literal := value.(type) {
	case json.Number:
		text = string(literal)
	case string:
		text = literal
	default:
		return value
	}
	if text != from {
		return value
	}
	if _, err := strconv.ParseInt(to, 10, 32); err == nil {
		return json.Number(to)
	}
	return to
}

// DropField removes field from every object that encoded message. check runs on each object
// that still carries the field and must prove the value is what the current protocol derives or
// defaults, so a drop never discards data silently.
func DropField(message protoreflect.FullName, field string, check func(fixture string, object *Object) error) ApplyFunc {
	return RewriteMessages(message, func(fixture string, object *Object) (any, error) {
		if _, ok := object.Fields[field]; !ok {
			return object, nil
		}
		if check == nil {
			return nil, errors.New("dropping " + string(message) + "." + field + " declares no check")
		}
		if err := check(fixture, object); err != nil {
			return nil, fmt.Errorf("%s.%s: %w", message, field, err)
		}
		delete(object.Fields, field)
		return object, nil
	})
}

func rewriteTree(value any, visit func(any) (any, error)) (any, error) {
	switch node := value.(type) {
	case *Object:
		for key, field := range node.Fields {
			rewritten, err := rewriteTree(field, visit)
			if err != nil {
				return nil, err
			}
			node.Fields[key] = rewritten
		}
	case []any:
		for index, element := range node {
			rewritten, err := rewriteTree(element, visit)
			if err != nil {
				return nil, err
			}
			node[index] = rewritten
		}
	default:
		// Scalars have no children.
	}
	return visit(value)
}
