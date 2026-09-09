package execution

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// liftFixture is one RPC whose response carries two alternative recorded shapes, plus the Testpilot
// protocol closure a declared ScopedEvidence Observation names.
func liftFixture(t *testing.T) (*testpilotspb.Case, *ir.Catalog, Policy) {
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

	limits := &testpilotspb.ProgramLimits{MaxEntrypoints: 8, MaxNodes: 32, MaxEdges: 64, MaxActivations: 64, MaxAttempts: 32, MaxRunEvents: 256, MaxExpressionDepth: 16, MaxPathFanout: 128, MaxRequestBytes: 4096, MaxResponseBytes: 4096, MaxTotalDurationMilliseconds: 30000, MaxCleanupDurationMilliseconds: 5000}
	policy := Policy{Identity: "host", CatalogIdentity: catalog.Identity(), Roles: []RolePolicy{{ID: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT, Methods: []string{"/lift.Source/Read"}}}, Capabilities: []Opcode{InvokeRPC}, Limits: proto.CloneOf(limits)}
	node := &testpilotspb.InstructionDefinition{InstructionId: "read", Instruction: &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InvokeRpc{InvokeRpc: &testpilotspb.InvokeRPC{EndpointRoleId: "endpoint", Method: "/lift.Source/Read"}}}, Outcome: statusSchema(), Limits: &testpilotspb.InstructionLimits{TimeoutMilliseconds: 1000, MaxAttempts: 1, MaxEmittedEvents: 8, MaxResponseBytes: 4096}}
	artifact := &testpilotspb.Case{Version: &testpilotspb.FormatVersion{Major: 1}, CaseId: "lift", Program: &testpilotspb.Program{
		ProgramId: "program", Roles: []*testpilotspb.RoleDefinition{{RoleId: "endpoint", Kind: testpilotspb.ROLE_KIND_ENDPOINT}},
		Observations: []*testpilotspb.ObservationDefinition{{ObservationId: "evidence", Type: messageValueType("temporal.server.api.testpilot.v1.ScopedEvidence")}, {ObservationId: "other", Type: scalar(testpilotspb.SCALAR_KIND_TEXT)}},
		Entrypoints:  []*testpilotspb.EntrypointDefinition{{EntrypointId: "controller", Activation: &testpilotspb.EntrypointDefinition_Controller{Controller: &testpilotspb.ControllerActivation{}}, Instructions: []*testpilotspb.InstructionDefinition{node}}},
		Cleanup:      &testpilotspb.CleanupDefinition{EntrypointId: "cleanup"}, Limits: limits}, Contract: &testpilotspb.Contract{ContractId: "contract"}}
	node.Instruction.GetInvokeRpc().ResponseProjections = []*testpilotspb.ResponseProjection{{
		Source: &testpilotspb.FieldPath{}, Kind: testpilotspb.PROJECTION_KIND_ONE,
		Targets: []*testpilotspb.ProjectionTarget{{Target: &testpilotspb.ProjectionTarget_ScopedEvidence{ScopedEvidence: liftProjection()}}}}}
	return artifact, catalog, policy
}

func messageValueType(name string) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Message{Message: &testpilotspb.NamedType{ProtobufType: name}}}}}
}
func nestedPath(names ...string) *testpilotspb.FieldPath {
	path := &testpilotspb.FieldPath{}
	for _, name := range names {
		path.Segments = append(path.Segments, &testpilotspb.FieldPathSegment{Field: name})
	}
	return path
}
func liftProjection() *testpilotspb.ScopedEvidenceProjection {
	return &testpilotspb.ScopedEvidenceProjection{ObservationId: "evidence", Rules: []*testpilotspb.ScopedEvidenceRule{
		{
			Guard: nestedPath("scheduled", "operation"), GuardEqualsText: "first", Source: "source", Kind: "scheduled.first",
			Operation: nestedPath("scheduled", "operation"),
			Scope:     []*testpilotspb.ScopedEvidenceBinding{{FieldId: "run", Value: &testpilotspb.ScopedEvidenceBinding_Literal{Literal: "one"}}},
			Fields:    []*testpilotspb.ScopedEvidenceBinding{{FieldId: "identity", Value: &testpilotspb.ScopedEvidenceBinding_Path{Path: nestedPath("scheduled", "operation")}}},
		},
		{
			Guard: nestedPath("scheduled"), Source: "source", Kind: "scheduled.other",
			Operation: nestedPath("scheduled", "operation"),
			Scope:     []*testpilotspb.ScopedEvidenceBinding{{FieldId: "run", Value: &testpilotspb.ScopedEvidenceBinding_Literal{Literal: "one"}}},
		},
		{
			Guard: nestedPath("completed"), Source: "source", Kind: "completed",
			Operation: nestedPath("completed", "referenced"),
			Scope:     []*testpilotspb.ScopedEvidenceBinding{{FieldId: "run", Value: &testpilotspb.ScopedEvidenceBinding_Literal{Literal: "one"}}},
		},
	}}
}

func liftResponse(t *testing.T, prepared *PreparedProgram, mutate func(protoreflect.Message)) EffectResult {
	t.Helper()
	response := dynamicpb.NewMessage(prepared.graphs[0].nodes[0].method.Output())
	mutate(response)
	return EffectResult{Outcome: &testpilotspb.InstructionOutcome{Status: testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED}, Response: response}
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

func stagedEvidence(t *testing.T, values *activationValues, prepared *PreparedProgram, mutate func(protoreflect.Message)) []*testpilotspb.ScopedEvidence {
	t.Helper()
	coordinate := Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "read", Attempt: 1}
	batch, _, err := values.stage(context.Background(), coordinate, liftResponse(t, prepared, mutate), values.workLimit())
	require.NoError(t, err)
	staged := []*testpilotspb.ScopedEvidence{}
	for _, fact := range batch.facts {
		for _, observation := range fact.observations {
			require.Equal(t, "evidence", observation.ObservationId)
			evidence := &testpilotspb.ScopedEvidence{}
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
	require.Equal(t, "source", first[0].GetIdentity().GetSource())
	require.EqualValues(t, 0, first[0].GetIdentity().GetOrdinal())
	require.Equal(t, []*testpilotspb.ScopedBinding{{FieldId: "run", Value: "one"}}, first[0].GetIdentity().GetScope())
	require.Equal(t, []*testpilotspb.ScopedEvidenceField{{FieldId: "identity", Value: &testpilotspb.Value{Value: &testpilotspb.Value_Text{Text: "first"}}}}, first[0].GetFields())

	// The guard equality is what separates the two scheduled rules, so a different identity falls
	// through to the unguarded one.
	other := stagedEvidence(t, values, prepared, setScheduled("second"))
	require.Len(t, other, 1)
	require.Equal(t, "scheduled.other", other[0].GetKind())
	require.Empty(t, other[0].GetFields())

	// A completion names its operation by the record it references, narrowed to a natural key.
	completed := stagedEvidence(t, values, prepared, setCompleted(7))
	require.Len(t, completed, 1)
	require.Equal(t, "completed", completed[0].GetKind())
	require.Equal(t, "7", completed[0].GetOperation())

	require.Empty(t, stagedEvidence(t, values, prepared, func(protoreflect.Message) {}))
}

// TestEvidenceLiftRejectsUndeclarableRules pins the admission errors: the sink must be the exact
// declared ScopedEvidence Observation, a rule must exist, and every bound coordinate must read a
// scalar the portable evidence domain admits.
func TestEvidenceLiftRejectsUndeclarableRules(t *testing.T) {
	for name, mutate := range map[string]func(*testpilotspb.ScopedEvidenceProjection){
		"wrong observation": func(p *testpilotspb.ScopedEvidenceProjection) { p.ObservationId = "other" },
		"unknown observation": func(p *testpilotspb.ScopedEvidenceProjection) {
			p.ObservationId = "absent"
		},
		"no rules":       func(p *testpilotspb.ScopedEvidenceProjection) { p.Rules = nil },
		"missing kind":   func(p *testpilotspb.ScopedEvidenceProjection) { p.Rules[0].Kind = "" },
		"missing source": func(p *testpilotspb.ScopedEvidenceProjection) { p.Rules[0].Source = "" },
		"message operation": func(p *testpilotspb.ScopedEvidenceProjection) {
			p.Rules[0].Operation = nestedPath("scheduled")
		},
		"message field": func(p *testpilotspb.ScopedEvidenceProjection) {
			p.Rules[0].Fields[0].Value = &testpilotspb.ScopedEvidenceBinding_Path{Path: nestedPath("nested")}
		},
		"unknown coordinate": func(p *testpilotspb.ScopedEvidenceProjection) {
			p.Rules[0].Operation = nestedPath("scheduled", "absent")
		},
		"nontext guard equality": func(p *testpilotspb.ScopedEvidenceProjection) {
			p.Rules[0].Guard = nestedPath("completed", "referenced")
		},
		"empty literal": func(p *testpilotspb.ScopedEvidenceProjection) {
			p.Rules[0].Scope[0].Value = &testpilotspb.ScopedEvidenceBinding_Literal{Literal: ""}
		},
		"unsupplied binding": func(p *testpilotspb.ScopedEvidenceProjection) { p.Rules[0].Scope[0].Value = nil },
		"duplicate field": func(p *testpilotspb.ScopedEvidenceProjection) {
			p.Rules[0].Fields = append(p.Rules[0].Fields, proto.CloneOf(p.Rules[0].Fields[0]))
		},
	} {
		t.Run(name, func(t *testing.T) {
			artifact, catalog, policy := liftFixture(t)
			mutate(artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections[0].Targets[0].GetScopedEvidence())
			_, err := Prepare(artifact, catalog, policy)
			require.Error(t, err)
		})
	}
}

// TestEvidenceLiftRejectsPartialEvidence pins the runtime error: a rule that fired but cannot read
// one of its own declared coordinates fails rather than recording evidence with a hole in it.
func TestEvidenceLiftRejectsPartialEvidence(t *testing.T) {
	artifact, catalog, policy := liftFixture(t)
	projection := artifact.Program.Entrypoints[0].Instructions[0].Instruction.GetInvokeRpc().ResponseProjections[0].Targets[0].GetScopedEvidence()
	projection.Rules = projection.Rules[2:]
	projection.Rules[0].Fields = []*testpilotspb.ScopedEvidenceBinding{{FieldId: "identity", Value: &testpilotspb.ScopedEvidenceBinding_Path{Path: nestedPath("scheduled", "operation")}}}
	prepared, err := Prepare(artifact, catalog, policy)
	require.NoError(t, err)
	store, err := newValueStore(prepared, "run")
	require.NoError(t, err)
	values, err := store.activate("controller", "activation")
	require.NoError(t, err)
	coordinate := Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "read", Attempt: 1}
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
	coordinate := Coordinate{RunID: "run", EntrypointID: "controller", ActivationID: "activation", InstructionID: "read", Attempt: 1}
	_, _, err = values.stage(context.Background(), coordinate, liftResponse(t, prepared, setCompleted(-1)), values.workLimit())
	require.Error(t, err)
}
