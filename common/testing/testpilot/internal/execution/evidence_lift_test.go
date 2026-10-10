package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/casefile"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"go.temporal.io/server/common/testing/testpilot/internal/testsupport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
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
	descriptors := testsupport.DescriptorClosure(testpilotspb.File_temporal_server_api_testpilot_v1_run_proto)
	descriptors.File = append(descriptors.File, source)
	catalog, err := ir.NewCatalog(descriptors)
	require.NoError(t, err)

	limits := testsupport.ProgramLimits()
	policy := Profile{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []contract.RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/lift.Source/Read"}}}, Opcodes: []contract.Opcode{contract.InvokeRPC}, Limits: proto.CloneOf(limits)}
	node := &testpilotspb.InstructionNode{InstructionId: "read", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRpc{EndpointRoleId: "endpoint", Method: "/lift.Source/Read"}}}, Limits: &testpilotspb.InstructionLimits{Timeout: durationpb.New(time.Duration(1000) * time.Millisecond), MaxAttempts: proto.Int64(1)}}
	artifact := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: casefile.CurrentMajor}, CaseId: "lift", Program: &testpilotspb.Program{
		ProgramId: "program", Roles: []*testpilotspb.Role{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}},
		Observations: []*testpilotspb.Observation{{ObservationId: "evidence", Type: messageValueType("temporal.server.api.testpilot.v1.CorrelatedEvidence")}, {ObservationId: "other", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}},
		Entrypoints:  []*testpilotspb.Entrypoint{{EntrypointId: "controller", Activation: &testpilotspb.Entrypoint_Controller{Controller: &emptypb.Empty{}}, Instructions: []*testpilotspb.InstructionNode{node}}},
		Cleanup:      &testpilotspb.Cleanup{EntrypointId: "cleanup"}}, Contract: &testpilotspb.Contract{ContractId: "contract"}}
	node.Instruction.GetInvokeRpc().ResponseReads = []*testpilotspb.ResponseRead{{Targets: []*testpilotspb.ReadTarget{{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{CorrelatedEvidence: liftProjection()}}}}}
	artifact.Program.Evidence = liftDeclarations()
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
	return cel.Path(cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_ProjectedValue{ProjectedValue: &emptypb.Empty{}}}), path)
}

// resolves is the guard that fires where path resolves on the projected value.
func resolves(path string) *testpilotspb.Expression {
	return present(projected(path))
}

// readsText is the guard that fires where path reads exactly text. It needs no presence conjunct: a
// comparison with an absent path is false.
func readsText(path, text string) *testpilotspb.Expression {
	return cel.All(resolves(path), cel.Compare("_==_", projected(path), cel.Literal(&celpb.Value{Kind: &celpb.Value_StringValue{StringValue: text}})))
}

// literalBinding supplies field with a declared text.
func literalBinding(field, text string) *testpilotspb.NamedExpression {
	return &testpilotspb.NamedExpression{FieldId: field, Value: cel.Literal(&celpb.Value{Kind: &celpb.Value_StringValue{StringValue: text}})}
}

// pathBinding supplies field with the value path reads from the projected value.
func pathBinding(field, path string) *testpilotspb.NamedExpression {
	return &testpilotspb.NamedExpression{FieldId: field, Value: projected(path)}
}

func liftProjection() *testpilotspb.CorrelatedEvidenceProjection {
	return &testpilotspb.CorrelatedEvidenceProjection{ObservationId: "evidence", EvidenceIds: []string{"scheduled-first", "scheduled-other", "completed"}}
}

func liftDeclarations() []*testpilotspb.EvidenceDeclaration {
	projectedSource := func() *testpilotspb.EvidenceDeclaration_Projected {
		return &testpilotspb.EvidenceDeclaration_Projected{Projected: &testpilotspb.ProjectedSource{Type: messageValueType("lift.Record")}}
	}
	return []*testpilotspb.EvidenceDeclaration{
		{EvidenceId: "scheduled-first", EvidenceSource: "source", Kind: "scheduled.first", Source: projectedSource(), Guard: readsText("scheduled.operation", "first"), Operation: projected("scheduled.operation"), Scope: []*testpilotspb.NamedExpression{literalBinding("run", "one")}, Fields: []*testpilotspb.NamedExpression{pathBinding("identity", "scheduled.operation")}},
		{EvidenceId: "scheduled-other", EvidenceSource: "source", Kind: "scheduled.other", Source: projectedSource(), Guard: resolves("scheduled"), Operation: projected("scheduled.operation"), Scope: []*testpilotspb.NamedExpression{literalBinding("run", "one")}},
		{EvidenceId: "completed", EvidenceSource: "source", Kind: "completed", Source: projectedSource(), Guard: resolves("completed"), Operation: projected("completed.referenced"), Scope: []*testpilotspb.NamedExpression{literalBinding("run", "one")}},
	}
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
			require.NoError(t, observation.Value.GetObjectValue().UnmarshalTo(evidence))
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
	require.Equal(t, []*testpilotspb.NamedValue{{FieldId: "run", Value: &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: "one"}}}}, first[0].GetIdentity().GetScope())
	require.Equal(t, []*testpilotspb.NamedValue{{FieldId: "identity", Value: &celpb.Value{Kind: &celpb.Value_StringValue{StringValue: "first"}}}}, first[0].GetFields())

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
		operator string
		holds    bool
	}{
		{"_==_", true},
		{"_!=_", false},
		{"_<_", false},
		{"_<=_", true},
		{"_>_", false},
		{"_>=_", true},
	} {
		t.Run(tc.operator, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			artifact.Program.Evidence[0].Guard = cel.All(resolves("completed.referenced"), cel.Compare(tc.operator, projected(nestedPath("completed", "referenced")), cel.Literal(&celpb.Value{Kind: &celpb.Value_Int64Value{Int64Value: 7}})))
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
	for name, mutate := range map[string]func(*testpilotspb.Case){
		"wrong observation": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence().ObservationId = "other"
		},
		"unknown declaration": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence().EvidenceIds[0] = "absent"
		},
		"no declarations": func(c *testpilotspb.Case) {
			c.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence().EvidenceIds = nil
		},
		"missing kind":       func(c *testpilotspb.Case) { c.Program.Evidence[0].Kind = "" },
		"missing source":     func(c *testpilotspb.Case) { c.Program.Evidence[0].EvidenceSource = "" },
		"message operation":  func(c *testpilotspb.Case) { c.Program.Evidence[0].Operation = projected("scheduled") },
		"message field":      func(c *testpilotspb.Case) { c.Program.Evidence[0].Fields[0] = pathBinding("identity", "nested") },
		"unknown coordinate": func(c *testpilotspb.Case) { c.Program.Evidence[0].Operation = projected("scheduled.absent") },
		"unsupplied binding": func(c *testpilotspb.Case) { c.Program.Evidence[0].Scope[0].Value = nil },
		"duplicate field": func(c *testpilotspb.Case) {
			c.Program.Evidence[0].Fields = append(c.Program.Evidence[0].Fields, proto.CloneOf(c.Program.Evidence[0].Fields[0]))
		},
		"nonboolean guard": func(c *testpilotspb.Case) { c.Program.Evidence[0].Guard = projected("scheduled.operation") },
	} {
		t.Run(name, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			mutate(artifact)
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
		"run":                {Reference: &testpilotspb.Reference_Run{Run: &emptypb.Empty{}}},
		"observation_id":     {Reference: &testpilotspb.Reference_ObservationId{ObservationId: "other"}},
		"evidence_field_id":  {Reference: &testpilotspb.Reference_EvidenceFieldId{EvidenceFieldId: "field"}},
		"correlated_capture": {Reference: &testpilotspb.Reference_CorrelatedCapture{CorrelatedCapture: &testpilotspb.CorrelatedCaptureReference{CaptureId: "capture"}}},
		"correlated_step":    {Reference: &testpilotspb.Reference_CorrelatedStep{CorrelatedStep: &testpilotspb.CorrelatedStepReference{Field: testpilotspb.CORRELATED_STEP_FIELD_ACTION, DefinitionId: "action"}}},
	} {
		t.Run(name, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			artifact.Program.Evidence[1].Guard = present(cel.Ref(value))
			_, err := Prepare(artifact, catalog, policy)
			var diagnostic *ir.Error
			require.ErrorAs(t, err, &diagnostic)
			require.Equal(t, &ir.Error{
				Category: ir.Unknown,
				Path:     "program.evidence[1].guard.bindings[0].reference." + name,
				Detail:   "reference is not admitted in this expression context",
			}, diagnostic)
		})
	}
}

// A literal binding supplies a text: an empty text and any other value reject with their own detail.
func TestEvidenceLiftLiteralBindingRequiresText(t *testing.T) {
	for detail, value := range map[string]*celpb.Value{
		"evidence literal binding requires a value":    {Kind: &celpb.Value_StringValue{}},
		"evidence binding reads an unsupported scalar": {Kind: &celpb.Value_BoolValue{BoolValue: true}},
	} {
		t.Run(detail, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			artifact.Program.Evidence[0].Scope[0].Value = cel.Literal(value)
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
	artifact.Program.Evidence[0].Fields[0].Value = cel.Ref(&testpilotspb.Reference{Reference: &testpilotspb.Reference_SlotId{SlotId: "slot"}})
	_, err := Prepare(artifact, catalog, policy)
	var diagnostic *ir.Error
	require.ErrorAs(t, err, &diagnostic)
	require.Equal(t, &ir.Error{
		Category: ir.Unknown,
		Path:     "program.evidence[0].fields[0].value.bindings[0].reference.slot_id",
		Detail:   "reference is not admitted in this expression context",
	}, diagnostic)
}

// TestEvidenceLiftRejectsPartialEvidence pins the runtime error: a rule that fired but cannot read
// one of its own declared coordinates fails rather than recording evidence with a hole in it.
func TestEvidenceLiftRejectsPartialEvidence(t *testing.T) {
	artifact, catalog, policy := liftFixture(t)
	projection := artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseReads[0].Targets[0].GetCorrelatedEvidence()
	projection.EvidenceIds = projection.EvidenceIds[2:]
	artifact.Program.Evidence[2].Fields = []*testpilotspb.NamedExpression{pathBinding("identity", nestedPath("scheduled", "operation"))}
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
