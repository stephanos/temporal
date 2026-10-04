package producer

import (
	"strconv"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
)

// Builders for the Program and Contract messages a realization is written with.

// Text is a text value.
func Text(value string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: value}}
}

// Bool is a boolean value.
func Bool(value bool) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_BoolValue{BoolValue: value}}
}

// SignedInteger is a signed integer value in the protocol's decimal spelling.
func SignedInteger(value int64) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: strconv.FormatInt(value, 10)}}
}

// Enum is an enum value by its name.
func Enum(name string) *testpilotspb.Value {
	return &testpilotspb.Value{Value: &testpilotspb.Value_EnumValue{EnumValue: &testpilotspb.EnumValue{Name: name}}}
}

// Literal is a literal expression.
func Literal(v *testpilotspb.Value) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: v}}
}

func reference(r *testpilotspb.Reference) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: r}}
}

// Environment refers to one symbolic environment resource.
func Environment(bindingID string) *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: bindingID}})
}

// Run refers to the current Run.
func Run() *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Run{Run: &testpilotspb.RunReference{}}})
}

// ProjectedValue refers to the value an evidence lift or poll is projecting.
func ProjectedValue() *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}})
}

func correlatedStep(field testpilotspb.CorrelatedStepField, definitionID string) *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{
		Field: field, DefinitionId: definitionID}}})
}

// Path reads the value at a path out of an operand.
func Path(operand *testpilotspb.Expression, path string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: operand, Path: path}}}
}

// Present tests an operand for presence.
func Present(operand *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Present{Present: &testpilotspb.PresentExpression{Operand: operand}}}
}

// Equal compares two operands for equality.
func Equal(left, right *testpilotspb.Expression) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
		Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL, Left: left, Right: right}}}
}

// Assign is one request assignment.
func Assign(target string, value *testpilotspb.Expression) *testpilotspb.RequestAssignment {
	return &testpilotspb.RequestAssignment{Target: target, Value: value}
}

// InvokeRPC invokes one unary method on an endpoint role.
func InvokeRPC(endpointRoleID, method string, assignments []*testpilotspb.RequestAssignment, reads []*testpilotspb.ResponseRead) *testpilotspb.Instruction {
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{
		EndpointRoleId: endpointRoleID, Method: method, RequestAssignments: assignments, ResponseReads: reads}}}
}

// ReadEvidence polls the read an evidence declaration names until the condition holds.
func ReadEvidence(evidenceID, endpointRoleID string, assignments []*testpilotspb.RequestAssignment, until *testpilotspb.Expression,
	pollIntervalMilliseconds int64) *testpilotspb.Instruction {
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{
		EvidenceId: evidenceID, EndpointRoleId: endpointRoleID, RequestAssignments: assignments, Until: until,
		PollIntervalMilliseconds: pollIntervalMilliseconds}}}
}

// NodeOption sets one optional part of an instruction node.
type NodeOption func(*testpilotspb.InstructionNode)

// TimeoutMilliseconds writes the node's dispatch timeout.
func TimeoutMilliseconds(ms int64) NodeOption {
	return func(n *testpilotspb.InstructionNode) {
		n.Limits = &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: ms}}
	}
}

// Guard sets the node's guard.
func Guard(e *testpilotspb.Expression) NodeOption {
	return func(n *testpilotspb.InstructionNode) { n.Guard = e }
}

// Node defines one instruction node.
func Node(id string, instruction *testpilotspb.Instruction, opts ...NodeOption) *testpilotspb.InstructionNode {
	n := &testpilotspb.InstructionNode{InstructionId: id, Instruction: instruction}
	for _, o := range opts {
		o(n)
	}
	return n
}

// ResponseRead reads a response path into targets.
func ResponseRead(path string, cardinality testpilotspb.ReadCardinality, targets ...*testpilotspb.ReadTarget) *testpilotspb.ResponseRead {
	return &testpilotspb.ResponseRead{Path: path, Cardinality: cardinality, Targets: targets}
}

// ObservationTarget writes into a declared Observation.
func ObservationTarget(id string) *testpilotspb.ReadTarget {
	return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: id}}
}

// EvidenceTarget is a history read's lift target: the history kinds among the resolved rules, each
// a rule naming its declaration, in the order the rules name them (`Temporal.Case.Evidence.target`).
func EvidenceTarget(observationID string, rules []EvidenceRule) *testpilotspb.ReadTarget {
	var lifted []*testpilotspb.CorrelatedEvidenceRule
	for _, r := range rules {
		if r.ReadsHistory() {
			lifted = append(lifted, &testpilotspb.CorrelatedEvidenceRule{EvidenceId: r.Source.KindID})
		}
	}
	return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{
		ObservationId: observationID, Rules: lifted}}}
}

// Role declares one logical role.
func Role(id string, kind testpilotspb.RoleKind, namespaceBinding, resourceBinding string) *testpilotspb.Role {
	return &testpilotspb.Role{RoleId: id, Kind: kind, NamespaceBindingId: namespaceBinding, ResourceBindingId: resourceBinding}
}

// HandleSlot declares an opaque handle slot.
func HandleSlot(id string) *testpilotspb.Slot {
	return &testpilotspb.Slot{SlotId: id, Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &testpilotspb.OpaqueHandleType{}}}
}

// MessageObservation declares an Observation of one protobuf message type.
func MessageObservation(id, protobufType string) *testpilotspb.Observation {
	return &testpilotspb.Observation{ObservationId: id, Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{
		Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: protobufType}}}}}}
}
