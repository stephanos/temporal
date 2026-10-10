package producer

import (
	"math"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/duration"
	"go.temporal.io/server/tools/umpire/internal/runtimecel"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Builders for the Program and Contract messages a realization is written with.

// Text is a text value.
func Text(value string) *celpb.Value {
	return &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: value}}
}

// Bool is a boolean value.
func Bool(value bool) *celpb.Value {
	return &celpb.Value{Kind: &celpb.Value_BoolValue{BoolValue: value}}
}

// SignedInteger is a native CEL signed integer.
func SignedInteger(value int64) *celpb.Value {
	return &celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: value}}
}

// Enum is an enum value by its name.
func Enum(name string) *celpb.Value {
	value, err := runtimecel.Enum(name)
	if err != nil {
		panic(err)
	}
	return value
}

// Literal is a literal expression.
func Literal(v *celpb.Value) *testpilotspb.Expression {
	return cel.Literal(v)
}

func reference(r *testpilotspb.Reference) *testpilotspb.Expression {
	return cel.Ref(r)
}

// Environment refers to one symbolic environment resource.
func Environment(bindingID string) *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_EnvironmentBindingId{EnvironmentBindingId: bindingID}})
}

// Run refers to the current Run.
func Run() *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_Run{Run: &emptypb.Empty{}}})
}

// ProjectedValue refers to the value an evidence lift or poll is projecting.
func ProjectedValue() *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &emptypb.Empty{}}})
}

func correlatedStep(field testpilotspb.CorrelatedStepField, definitionID string) *testpilotspb.Expression {
	return reference(&testpilotspb.Reference{Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{
		Field: field, DefinitionId: definitionID}}})
}

// Path reads the value at a path out of an operand.
func Path(operand *testpilotspb.Expression, path string) *testpilotspb.Expression {
	return cel.Path(operand, path)
}

// Present tests an operand for presence.
func Present(operand *testpilotspb.Expression) *testpilotspb.Expression {
	return cel.Present(operand)
}

// Equal compares two operands for equality.
func Equal(left, right *testpilotspb.Expression) *testpilotspb.Expression {
	return cel.Compare("_==_", left, right)
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
	var interval *durationpb.Duration
	if pollIntervalMilliseconds != 0 {
		interval = duration.FromMilliseconds(pollIntervalMilliseconds)
	}
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ReadEvidence{ReadEvidence: &testpilotspb.ReadEvidence{
		EvidenceId: evidenceID, EndpointRoleId: endpointRoleID, RequestAssignments: assignments, Until: until,
		Interval: interval}}}
}

// NodeOption sets one optional part of an instruction node.
type NodeOption func(*testpilotspb.InstructionNode)

// TimeoutMilliseconds writes the node's dispatch timeout.
func TimeoutMilliseconds(ms int64) NodeOption {
	return func(n *testpilotspb.InstructionNode) {
		n.Limits = &testpilotspb.InstructionLimits{Timeout: duration.FromMilliseconds(ms)}
	}
}

// ReadOnce makes a node's ReadEvidence read once: its condition is checked once, with no interval.
func ReadOnce() NodeOption {
	return func(n *testpilotspb.InstructionNode) {
		read := n.GetInstruction().GetReadEvidence()
		read.Interval = nil
	}
}

// WaitWithin makes a node's ReadEvidence poll every interval within the bounds of the hints it
// names: the node writes its own timeout, their sum, and no Profile default applies to it.
func WaitWithin(intervalMilliseconds int64, hints ...*testpilotspb.WaitHint) NodeOption {
	return func(n *testpilotspb.InstructionNode) {
		var sum int64
		for _, h := range hints {
			ms, err := duration.Milliseconds("wait hint at_most", h.GetAtMost())
			if err != nil {
				panic(err)
			}
			if ms > math.MaxInt64-sum {
				panic("wait hint duration sum overflows milliseconds")
			}
			sum += ms
		}
		n.GetInstruction().GetReadEvidence().Interval = duration.FromMilliseconds(intervalMilliseconds)
		n.WaitHints = hints
		TimeoutMilliseconds(sum)(n)
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
func ResponseRead(path string, targets ...*testpilotspb.ReadTarget) *testpilotspb.ResponseRead {
	return &testpilotspb.ResponseRead{Path: path, Targets: targets}
}

// ObservationTarget writes into a declared Observation.
func ObservationTarget(id string) *testpilotspb.ReadTarget {
	return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_ObservationId{ObservationId: id}}
}

// EvidenceTarget is a history read's lift target: the history kinds among the resolved rules, each
// a rule naming its declaration, in the order the rules name them (`Temporal.Case.Evidence.target`).
func EvidenceTarget(observationID string, rules []EvidenceRule) *testpilotspb.ReadTarget {
	var lifted []string
	for _, r := range rules {
		if r.ReadsHistory() {
			lifted = append(lifted, r.Source.KindID)
		}
	}
	return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{
		ObservationId: observationID, EvidenceIds: lifted}}}
}

// Role declares one logical role.
func Role(id string, kind testpilotspb.RoleKind, namespaceBinding, resourceBinding string) *testpilotspb.Role {
	return &testpilotspb.Role{RoleId: id, Kind: kind, NamespaceBindingId: namespaceBinding, ResourceBindingId: resourceBinding}
}

// HandleSlot declares an opaque handle slot.
func HandleSlot(id string) *testpilotspb.Slot {
	return &testpilotspb.Slot{SlotId: id, Content: &testpilotspb.Slot_OpaqueHandle{OpaqueHandle: &emptypb.Empty{}}}
}

// MessageObservation declares an Observation of one protobuf message type.
func MessageObservation(id, protobufType string) *testpilotspb.Observation {
	return &testpilotspb.Observation{ObservationId: id, Type: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{
		Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: protobufType}}}}}}
}
