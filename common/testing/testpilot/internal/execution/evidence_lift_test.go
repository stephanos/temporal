package execution

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// liftFixture is one RPC whose response carries two alternative recorded shapes, plus the Testpilot
// protocol closure a declared CorrelatedEvidence Observation names.
func liftFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Profile) {
	t.Helper()
	message := func(name string, fields ...*descriptorpb.FieldDescriptorProto) *descriptorpb.DescriptorProto {
		return &descriptorpb.DescriptorProto{Name: proto.String(name), Field: fields}
	}
	scalarField := func(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Type: kind.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}
	}
	messageField := func(name string, number int32, typeName string) *descriptorpb.FieldDescriptorProto {
		return &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), TypeName: proto.String(typeName)}
	}
	source := &descriptorpb.FileDescriptorProto{
		Name: proto.String("lift.proto"), Package: proto.String("lift"), Syntax: proto.String("proto3"),
		MessageType: []*descriptorpb.DescriptorProto{
			message("Record",
				messageField("scheduled", 1, ".lift.Scheduled"),
				messageField("completed", 2, ".lift.Completed"),
				messageField("nested", 3, ".lift.Scheduled")),
			message("Scheduled", scalarField("operation", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING)),
			message("Completed", scalarField("referenced", 1, descriptorpb.FieldDescriptorProto_TYPE_INT64)),
		},
		Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Source"), Method: []*descriptorpb.MethodDescriptorProto{{
			Name: proto.String("Read"), InputType: proto.String(".lift.Record"), OutputType: proto.String(".lift.Record"),
		}}}},
	}
	descriptors := &descriptorpb.FileDescriptorSet{}
	seen := map[string]bool{}
	var add func(protoreflect.FileDescriptor)
	add = func(file protoreflect.FileDescriptor) {
		if seen[file.Path()] {
			return
		}
		seen[file.Path()] = true
		for index := 0; index < file.Imports().Len(); index++ {
			add(file.Imports().Get(index))
		}
		descriptors.File = append(descriptors.File, protodesc.ToFileDescriptorProto(file))
	}
	add(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	descriptors.File = append(descriptors.File, source)
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)

	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000, MaxInstructionEmittedEvents: 8, MaxInstructionResponseBytes: 4096}
	policy := Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []contract.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/lift.Source/Read"}}}, Opcodes: []contract.Opcode{contract.InvokeRPC}, Limits: proto.CloneOf(limits)}
	node := &testpilotspb.InstructionNode{InstructionId: "read", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: "/lift.Source/Read"}}}, Limits: &testpilotspb.InstructionLimits{Timeout: &testpilotspb.InstructionLimits_TimeoutMilliseconds{TimeoutMilliseconds: 1000}, Attempts: &testpilotspb.InstructionLimits_MaxAttempts{MaxAttempts: 1}}}
	artifact := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "lift", Program: &testpilotspb.Program{
		ProgramId: "program", Roles: []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}},
		Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: messageValueType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}, {ObservationId: "other", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}},
		Entrypoints:  []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionNode{node}}},
		Cleanup:      &testpilotspb.Cleanup{EntrypointId: "cleanup"}}, Contract: &testpilotspb.Contract{ContractId: "contract"}}
	node.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{
		Cardinality: testpilotspb.READ_CARDINALITY_ONE,
		Targets:     []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{CorrelatedEvidence: liftProjection()}}}}}
	return artifact, catalog, policy
}

func messageValueType(name string) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}}
}
func nestedPath(names ...string) string {
	return strings.Join(names, ".")
}

// projected reads path out of the value an evidence lift is projecting.
func projected(path string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{
		Operand: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &testpilotspb.ProjectedValueReference{}}}}},
		Path:    path,
	}}}
}

// resolves is the guard that fires where path resolves on the projected value.
func resolves(path string) *testpilotspb.Expression {
	return present(projected(path))
}

// readsText is the guard that fires where path reads exactly text. It needs no presence conjunct: a
// comparison with an absent path is false.
func readsText(path, text string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
		Operator: testpilotspb.COMPARISON_OPERATOR_EQUAL,
		Left:     projected(path),
		Right:    &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: text}}}},
	}}}
}

// literalBinding supplies field with a declared text.
func literalBinding(field, text string) *testpilotspb.NamedExpression {
	return &testpilotspb.NamedExpression{FieldId: field, Value: &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: text}}}}}
}

// pathBinding supplies field with the value path reads from the projected value.
func pathBinding(field, path string) *testpilotspb.NamedExpression {
	return &testpilotspb.NamedExpression{FieldId: field, Value: projected(path)}
}

func liftProjection() *testpilotspb.CorrelatedEvidenceProjection {
	return &testpilotspb.CorrelatedEvidenceProjection{ObservationId: "evidence", Rules: []*testpilotspb.CorrelatedEvidenceRule{
		{
			Guard: readsText(nestedPath("scheduled", "operation"), "first"), EvidenceSource: "source", Kind: "scheduled.first",
			Operation: nestedPath("scheduled", "operation"),
			Scope:     []*testpilotspb.NamedExpression{literalBinding("run", "one")},
			Fields:    []*testpilotspb.NamedExpression{pathBinding("identity", nestedPath("scheduled", "operation"))},
		},
		{
			Guard: resolves(nestedPath("scheduled")), EvidenceSource: "source", Kind: "scheduled.other",
			Operation: nestedPath("scheduled", "operation"),
			Scope:     []*testpilotspb.NamedExpression{literalBinding("run", "one")},
		},
		{
			Guard: resolves(nestedPath("completed")), EvidenceSource: "source", Kind: "completed",
			Operation: nestedPath("completed", "referenced"),
			Scope:     []*testpilotspb.NamedExpression{literalBinding("run", "one")},
		},
	}}
}

func liftResponse(t *testing.T, prepared *PreparedProgram, mutate func(protoreflect.Message)) contract.EffectResult {
	t.Helper()
	response := dynamicpb.NewMessage(prepared.graphs[0].nodes[0].method.Output())
	mutate(response)
	return contract.EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}
}
func setScheduled(operation string) func(protoreflect.Message) {
	return func(m protoreflect.Message) {
		field := m.Descriptor().Fields().ByName("scheduled")
		scheduled := m.NewField(field)
		scheduled.Message().Set(scheduled.Message().Descriptor().Fields().ByName("operation"), protoreflect.ValueOfString(operation))
		m.Set(field, scheduled)
	}
}
func setCompleted(referenced int64) func(protoreflect.Message) {
	return func(m protoreflect.Message) {
		field := m.Descriptor().Fields().ByName("completed")
		completed := m.NewField(field)
		completed.Message().Set(completed.Message().Descriptor().Fields().ByName("referenced"), protoreflect.ValueOfInt64(referenced))
		m.Set(field, completed)
	}
}

func stagedEvidence(t *testing.T, values *activationValues, prepared *PreparedProgram, mutate func(protoreflect.Message)) []*testpilotspb.CorrelatedEvidence {
	t.Helper()
	coordinate := contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "read", Attempt: 1}
	batch, _, err := values.stage(context.Background(), coordinate, liftResponse(t, prepared, mutate), values.workLimit())
	require.NoError(t, err)
	staged := []*testpilotspb.CorrelatedEvidence{}
	for _, fact := range batch.facts {
		for _, observation := range fact.observations {
			require.Equal(t, "evidence", observation.ObservationId)
			evidence := &testpilotspb.CorrelatedEvidence{}
			require.NoError(t, observation.Value.GetMessageValue().UnmarshalTo(evidence))
			staged = append(staged, evidence)
		}
	}
	return staged
}

// TestEvidenceLiftSelectsOneRuleAndCountsItsOwnOrdinals drives the whole lift: the first rule whose
// guard resolves owns the value, an integer coordinate is an operation key in its decimal spelling,
// a declared literal supplies the Run coordinate the record does not carry, and a record no rule
// claims emits nothing.
func TestEvidenceLiftSelectsOneRuleAndCountsItsOwnOrdinals(t *testing.T) {
	artifact, catalog, policy := liftFixture(t)
	prepared, err := Prepare(artifact, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)

	first := stagedEvidence(t, values, prepared, setScheduled("first"))
	require.Len(t, first, 1)
	require.Equal(t, "scheduled.first", first[0].GetKind())
	require.Equal(t, "first", first[0].GetOperation())
	require.Equal(t, "source", first[0].GetIdentity().GetEvidenceSource())
	require.EqualValues(t, 0, first[0].GetIdentity().GetOrdinal())
	require.Equal(t, []*testpilotspb.NamedValue{{FieldId: "run", Value: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "one"}}}}, first[0].GetIdentity().GetScope())
	require.Equal(t, []*testpilotspb.NamedValue{{FieldId: "identity", Value: &testpilotspb.Value{Value: &testpilotspb.Value_TextValue{TextValue: "first"}}}}, first[0].GetFields())

	// The guard equality is what separates the two scheduled rules, so a different identity falls
	// through to the unguarded one.
	other := stagedEvidence(t, values, prepared, setScheduled("second"))
	require.Len(t, other, 1)
	require.Equal(t, "scheduled.other", other[0].GetKind())
	require.Empty(t, other[0].GetFields())

	// A completion names its operation by the record it references, narrowed to an unsigned integer key.
	// The first rule's guard compares the absent scheduled operation, which is false rather than an
	// evaluation failure.
	completed := stagedEvidence(t, values, prepared, setCompleted(7))
	require.Len(t, completed, 1)
	require.Equal(t, "completed", completed[0].GetKind())
	require.Equal(t, "7", completed[0].GetOperation())

	require.Empty(t, stagedEvidence(t, values, prepared, func(protoreflect.Message) {}))
}

// TestEvidenceLiftGuardComparisonsWithAnAbsentPathAreFalse pins the absent-operand rule in the
// evidence-lift context: under every operator, a guard comparing a path the record does not carry is
// false, so the next rule claims the record, while the same guard over a carried path still decides.
func TestEvidenceLiftGuardComparisonsWithAnAbsentPathAreFalse(t *testing.T) {
	for _, tc := range []struct {
		operator testpilotspb.ComparisonOperator
		holds    bool
	}{
		{testpilotspb.COMPARISON_OPERATOR_EQUAL, true},
		{testpilotspb.COMPARISON_OPERATOR_NOT_EQUAL, false},
		{testpilotspb.COMPARISON_OPERATOR_LESS_THAN, false},
		{testpilotspb.COMPARISON_OPERATOR_LESS_THAN_OR_EQUAL, true},
		{testpilotspb.COMPARISON_OPERATOR_GREATER_THAN, false},
		{testpilotspb.COMPARISON_OPERATOR_GREATER_THAN_OR_EQUAL, true},
	} {
		t.Run(tc.operator.String(), func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			lift := artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
			lift.Rules[0].Guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
				Operator: tc.operator,
				Left:     projected(nestedPath("completed", "referenced")),
				Right:    &testpilotspb.Expression{Expression: &testpilotspb.Expression_Literal{Literal: &testpilotspb.Value{Value: &testpilotspb.Value_SignedIntegerValue{SignedIntegerValue: "7"}}}},
			}}}
			prepared, err := Prepare(artifact, catalog, policy)
			require.NoError(t, err)
			store, err := newValueStore(prepared, "run")
			require.NoError(t, err)
			values, err := store.activate("controller", "activation")
			require.NoError(t, err)

			absent := stagedEvidence(t, values, prepared, setScheduled("first"))
			require.Len(t, absent, 1)
			require.Equal(t, "scheduled.other", absent[0].GetKind())

			carried := stagedEvidence(t, values, prepared, func(m protoreflect.Message) {
				setScheduled("first")(m)
				setCompleted(7)(m)
			})
			require.Len(t, carried, 1)
			require.Equal(t, map[bool]string{true: "scheduled.first", false: "scheduled.other"}[tc.holds], carried[0].GetKind())
		})
	}
}

// TestEvidenceLiftRejectsUndeclarableRules pins the admission errors: the sink must be the exact
// declared CorrelatedEvidence Observation, a rule must exist, and every bound coordinate must read a
// scalar the portable evidence domain admits.
func TestEvidenceLiftRejectsUndeclarableRules(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.CorrelatedEvidenceProjection){
		"wrong observation": func(p *testpilotspb.CorrelatedEvidenceProjection) { p.ObservationId = "other" },
		"unknown observation": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.ObservationId = "absent"
		},
		"no rules":       func(p *testpilotspb.CorrelatedEvidenceProjection) { p.Rules = nil },
		"missing kind":   func(p *testpilotspb.CorrelatedEvidenceProjection) { p.Rules[0].Kind = "" },
		"missing source": func(p *testpilotspb.CorrelatedEvidenceProjection) { p.Rules[0].EvidenceSource = "" },
		"message operation": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Operation = nestedPath("scheduled")
		},
		"message field": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Fields[0] = pathBinding("identity", nestedPath("nested"))
		},
		"unknown coordinate": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Operation = nestedPath("scheduled", "absent")
		},
		"nontext guard equality": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Guard = readsText(nestedPath("completed", "referenced"), "first")
		},
		"path over another operand": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Fields[0].Value = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Path{Path: &testpilotspb.PathExpression{Operand: projected(nestedPath("scheduled")), Path: nestedPath("operation")}}}
		},
		"other expression": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Fields[0].Value = resolves(nestedPath("scheduled", "operation"))
		},
		"unsupplied binding": func(p *testpilotspb.CorrelatedEvidenceProjection) { p.Rules[0].Scope[0].Value = nil },
		"duplicate field": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Fields = append(p.Rules[0].Fields, proto.CloneOf(p.Rules[0].Fields[0]))
		},
		"nonboolean guard": func(p *testpilotspb.CorrelatedEvidenceProjection) {
			p.Rules[0].Guard = projected(nestedPath("scheduled", "operation"))
		},
		"missing guard": func(p *testpilotspb.CorrelatedEvidenceProjection) { p.Rules[0].Guard = nil },
	} {
		t.Run(name, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			mutate(artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence())
			_, err := Prepare(artifact, catalog, policy)
			require.Error(t, err)
		})
	}
}

// A lift guard reads only the projected value, so every other reference rejects at preparation at
// the guard's located path.
func TestEvidenceLiftGuardRejectsReferencesOutsideItsContext(t *testing.T) {
	for name, value := range map[string]*testpilotspb.Reference{
		"slot_id":            {Reference: &testpilotspb.Reference_SlotId{SlotId: "slot"}},
		"run":                {Reference: &testpilotspb.Reference_Run{Run: &testpilotspb.RunReference{}}},
		"observation_id":     {Reference: &testpilotspb.Reference_ObservationId{ObservationId: "other"}},
		"evidence_field_id":  {Reference: &testpilotspb.Reference_EvidenceFieldId{EvidenceFieldId: "field"}},
		"correlated_capture": {Reference: &testpilotspb.Reference_CorrelatedCapture{CorrelatedCapture: &testpilotspb.CorrelatedCaptureReference{CaptureId: "capture"}}},
		"correlated_step":    {Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: testpilotspb.CORRELATED_STEP_FIELD_ACTION, DefinitionId: "action"}}},
	} {
		t.Run(name, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			projection := artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
			projection.Rules[1].Guard = present(&testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: value}})
			_, err := Prepare(artifact, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, &ir.Error{
				Category: ir.Unknown,
				Path:     "program.entrypoints[controller].instructions[read].instruction.invoke_rpc.response_reads[0].targets[0].correlated_evidence.rules[1].guard.present.reference." + name,
				Detail:   "reference is not admitted in this expression context",
			}, diagnostic)
		})
	}
}

// A literal binding supplies a text: an empty text and any other value reject with their own detail.
func TestEvidenceLiftLiteralBindingRequiresText(t *testing.T) {
	for detail, value := range map[string]*testpilotspb.Value{
		"evidence literal binding requires a value": {Value: &testpilotspb.Value_TextValue{}},
		"evidence literal binding requires a text":  {Value: &testpilotspb.Value_BoolValue{BoolValue: true}},
	} {
		t.Run(detail, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			projection := artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
			projection.Rules[0].Scope[0].Value.GetLiteral().Value = value.Value
			_, err := Prepare(artifact, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, detail, diagnostic.Detail)
		})
	}
}

// An evidence binding reads only the projected value, so any other reference rejects at preparation
// at the binding's located path.
func TestEvidenceLiftBindingRejectsReferencesOutsideItsContext(t *testing.T) {
	artifact, catalog, policy := liftFixture(t)
	projection := artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
	projection.Rules[0].Fields[0].Value.GetPath().Operand = &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: "slot"}}}}
	_, err := Prepare(artifact, catalog, policy)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, &ir.Error{
		Category: ir.Unknown,
		Path:     "program.entrypoints[controller].instructions[read].instruction.invoke_rpc.response_reads[0].targets[0].correlated_evidence.rules[0].fields[0].value.path.operand.reference.slot_id",
		Detail:   "reference is not admitted in this expression context",
	}, diagnostic)
}

// TestEvidenceLiftRejectsPartialEvidence pins the runtime error: a rule that fired but cannot read
// one of its own declared coordinates fails rather than recording evidence with a hole in it.
func TestEvidenceLiftRejectsPartialEvidence(t *testing.T) {
	artifact, catalog, policy := liftFixture(t)
	projection := artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
	projection.Rules = projection.Rules[2:]
	projection.Rules[0].Fields = []*testpilotspb.NamedExpression{pathBinding("identity", nestedPath("scheduled", "operation"))}
	prepared, err := Prepare(artifact, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coordinate := contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "read", Attempt: 1}
	_, _, err = values.stage(context.Background(), coordinate, liftResponse(t, prepared, setCompleted(3)), values.workLimit())
	require.Error(t, err)
}

// TestEvidenceLiftRejectsNegativeKey pins the one narrowing the portable evidence domain cannot do.
func TestEvidenceLiftRejectsNegativeKey(t *testing.T) {
	artifact, catalog, policy := liftFixture(t)
	prepared, err := Prepare(artifact, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coordinate := contract.Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "read", Attempt: 1}
	_, _, err = values.stage(context.Background(), coordinate, liftResponse(t, prepared, setCompleted(-1)), values.workLimit())
	require.Error(t, err)
}

// TestEvidenceLiftRejectsSharedSourcesAndWorkerEntrypoints pins the two shapes whose ordinals could
// not stay dense per Run: a second instruction lifting under the same declared source, and a lift on
// an entrypoint that activates more than once.
func TestEvidenceLiftRejectsSharedSourcesAndWorkerEntrypoints(t *testing.T) {
	t.Run("second instruction", func(t *testing.T) {
		artifact, catalog, policy := liftFixture(t)
		instructions := artifact.Program.Entrypoints[0].Instructions
		second := proto.CloneOf(instructions[0])
		second.InstructionId = "read-again"
		second.Guard = succeeded("controller", "read")
		artifact.Program.Entrypoints[0].Instructions = append(instructions, second)
		_, err := Prepare(artifact, catalog, policy)
		require.Error(t, err)
	})
	t.Run("worker entrypoint", func(t *testing.T) {
		artifact, catalog, policy := liftFixture(t)
		artifact.Program.Entrypoints[0].Activation = &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{WorkflowType: "flow", WorkerRoleId: "worker", TaskQueueRoleId: "queue"}}
		artifact.Program.Roles = append(artifact.Program.Roles,
			&testpilotspb.Role{RoleId: "worker", Kind: testpilotspb.ROLE_KIND_WORKER, NamespaceBindingId: "namespace"},
			&testpilotspb.Role{RoleId: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE, NamespaceBindingId: "namespace", ResourceBindingId: "queue"})
		policy.Roles = append(policy.Roles, contract.RolePolicy{ID: "worker", Kind: testpilotspb.ROLE_KIND_WORKER}, contract.RolePolicy{ID: "queue", Kind: testpilotspb.ROLE_KIND_TASK_QUEUE})
		policy.EnvironmentBindings = append(policy.EnvironmentBindings, contract.EnvironmentBinding{ID: "namespace", Value: "namespace"}, contract.EnvironmentBinding{ID: "queue", Value: "queue"})
		_, err := Prepare(artifact, catalog, policy)
		require.Error(t, err)
	})
}
