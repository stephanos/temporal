package testpilot

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	celpb "cel.dev/expr"
	"github.com/stretchr/testify/require"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

var protocolFiles = []string{
	"temporal/server/api/testpilot/v1/case.proto",
	"temporal/server/api/testpilot/v1/value.proto",
	"temporal/server/api/testpilot/v1/expression.proto",
	"temporal/server/api/testpilot/v1/evidence.proto",
	"temporal/server/api/testpilot/v1/program.proto",
	"temporal/server/api/testpilot/v1/instruction.proto",
	"temporal/server/api/testpilot/v1/contract.proto",
	"temporal/server/api/testpilot/v1/correlated.proto",
	"temporal/server/api/testpilot/v1/event.proto",
	"temporal/server/api/testpilot/v1/run.proto",
	"temporal/server/api/testpilot/v1/source.proto",
}

// runOnlyMessages are declared for a Run and must stay outside the Case import closure.
// The Run Event payload arms are among them: InstructionOutcome and FaultInjected.
var runOnlyMessages = []protoreflect.Name{"Run", "Verdict", "RunDiagnostic", "RunEvent", "CleanupOutcome", "InstructionOutcome", "FaultInjected"}

func TestProtocolEncodesExpressionAndStateScopes(t *testing.T) {
	t.Parallel()
	expression := messageDescriptor(t, "Expression")
	require.Equal(t, []protoreflect.Name{"cel", "bindings"}, fieldNames(expression))
	require.Equal(t, (&celpb.ParsedExpr{}).ProtoReflect().Descriptor().FullName(), expression.Fields().ByName("cel").Message().FullName())
	binding := messageDescriptor(t, "ExpressionBinding")
	require.Equal(t, []protoreflect.Name{"variable", "reference", "literal", "path"}, fieldNames(binding))
	require.Equal(t, []protoreflect.Name{"reference", "literal"}, oneofNames(binding.Oneofs().ByName("input")))
	reference := messageDescriptor(t, "Reference")
	require.NotNil(t, reference.Oneofs().ByName("reference"))
	require.Equal(t, []protoreflect.Name{
		"slot_id", "outcome", "run", "environment_binding_id", "observation_id", "run_event", "capture_id",
		"evidence_field_id", "correlated_capture", "correlated_step", "projected_value",
		"instance_value_id",
	}, fieldNames(reference))
	rule := messageDescriptor(t, "CorrelatedRule")
	evidenceRule := messageDescriptor(t, "EvidenceDeclaration")
	for _, field := range []protoreflect.FieldDescriptor{
		rule.Fields().ByName("trigger"), rule.Fields().ByName("response"), rule.Fields().ByName("correlation"), evidenceRule.Fields().ByName("guard"),
	} {
		require.Equal(t, expression.FullName(), field.Message().FullName(), field.FullName())
	}
	slot := messageDescriptor(t, "Slot")
	require.NotNil(t, slot.Oneofs().ByName("content"))
	require.Nil(t, slot.Fields().ByName("kind"))
	require.Equal(t, []protoreflect.Name{"scalar", "enumeration", "message", "any"}, oneofNames(messageDescriptor(t, "SingularType").Oneofs().ByName("type")))
	require.Equal(t, []protoreflect.Name{
		"null_value", "bool_value", "int64_value", "uint64_value", "double_value", "string_value", "bytes_value",
		"enum_value", "object_value", "map_value", "list_value", "type_value",
	}, oneofNames((&celpb.Value{}).ProtoReflect().Descriptor().Oneofs().ByName("kind")))
	require.Nil(t, testpilotspb.ScalarKind(0).Descriptor().Values().ByName("SCALAR_KIND_"+"NATURAL"))
	entrypoint := messageDescriptor(t, "Entrypoint")
	require.NotNil(t, entrypoint.Oneofs().ByName("activation"))
	require.Nil(t, entrypoint.Fields().ByName("context"))
	diagnostic := messageDescriptor(t, "RunDiagnostic")
	require.True(t, diagnostic.Fields().ByName("supporting_event_sequence").HasPresence())
	run := messageDescriptor(t, "Run")
	require.True(t, run.Fields().ByName("evaluation_failure_sequence").HasPresence())
	// Zero is an unscaled Run, so the scale needs no presence.
	require.False(t, run.Fields().ByName("bound_scale_percent").HasPresence())
	sourceLocation := messageDescriptor(t, "SourceLocation")
	require.Equal(t, "temporal/server/api/testpilot/v1/source.proto", sourceLocation.ParentFile().Path())
	waitHint := messageDescriptor(t, "WaitHint")
	require.Equal(t, []protoreflect.Name{"hint_id", "source", "at_most"}, fieldNames(waitHint))
	require.Equal(t, protoreflect.FullName("google.protobuf.Duration"), waitHint.Fields().ByName("at_most").Message().FullName())
	require.Equal(t, sourceLocation.FullName(), waitHint.Fields().ByName("source").Message().FullName())
	require.True(t, messageDescriptor(t, "InstructionNode").Fields().ByName("wait_hints").IsList())
	require.Nil(t, messageDescriptor(t, "ReadEvidence").Fields().ByName("once"))
	require.Equal(t, protoreflect.FullName("google.protobuf.Duration"), messageDescriptor(t, "ReadEvidence").Fields().ByName("interval").Message().FullName())
	require.Nil(t, messageDescriptor(t, "ContractRule").Fields().ByName("kind"))
	require.True(t, messageDescriptor(t, "ContractTransition").Fields().ByName("supports_event").HasPresence())
	require.Equal(t, messageDescriptor(t, "LocalInstructionReference").FullName(), messageDescriptor(t, "After").Fields().ByName("instructions").Message().FullName())
	require.Equal(t, []protoreflect.Name{"observation_id", "evidence_ids"}, fieldNames(messageDescriptor(t, "CorrelatedEvidenceProjection")))
	require.Equal(t, "temporal/server/api/testpilot/v1/evidence.proto", evidenceRule.ParentFile().Path())
	require.Equal(t, []protoreflect.Name{"state_id", "atom", "fields"}, fieldNames(messageDescriptor(t, "CorrelatedState")))
	require.Equal(t, []protoreflect.Name{"result_id", "action", "state_id", "outcome", "facts"}, fieldNames(messageDescriptor(t, "CorrelatedResult")))
	require.Equal(t, []protoreflect.Name{"prior_state_id", "result_id"}, fieldNames(messageDescriptor(t, "CorrelatedTransition")))
	require.Nil(t, rule.Fields().ByName("clock"))
	require.Nil(t, messageDescriptor(t, "ReadSource").Fields().ByName("single"))
	require.Nil(t, messageDescriptor(t, "ResponseRead").Fields().ByName("cardinality"))
	for _, field := range []protoreflect.FieldDescriptor{
		reference.Fields().ByName("run"), reference.Fields().ByName("projected_value"),
		messageDescriptor(t, "RunEventReference").Fields().ByName("payload"),
	} {
		require.Equal(t, protoreflect.FullName("google.protobuf.Empty"), field.Message().FullName(), field.FullName())
	}
	for _, field := range []protoreflect.FieldDescriptor{
		messageDescriptor(t, "InstructionLimits").Fields().ByName("timeout"),
		messageDescriptor(t, "ProgramLimits").Fields().ByName("max_duration"),
		messageDescriptor(t, "ProgramLimits").Fields().ByName("cleanup_duration"),
		messageDescriptor(t, "Deadline").Fields().ByName("elapsed"),
		messageDescriptor(t, "RunEvent").Fields().ByName("elapsed"),
	} {
		require.Equal(t, protoreflect.FullName("google.protobuf.Duration"), field.Message().FullName(), field.FullName())
	}
	require.True(t, messageDescriptor(t, "InstructionLimits").Fields().ByName("max_attempts").HasPresence())
	require.True(t, messageDescriptor(t, "KnownGap").Fields().ByName("subject").HasPresence())
	require.True(t, messageDescriptor(t, "KnownGap").Fields().ByName("detail").HasPresence())
	// Kind-specific Run Event data is one payload oneof, read through a path from the payload
	// reference, so the coordinate enum names only what every event has.
	event := messageDescriptor(t, "RunEvent")
	require.Equal(t, []protoreflect.Name{"outcome", "fault_injected"}, oneofNames(event.Oneofs().ByName("payload")))
	eventReference := messageDescriptor(t, "RunEventReference")
	require.Equal(t, []protoreflect.Name{"field", "payload"}, oneofNames(eventReference.Oneofs().ByName("selection")))
	require.EqualValues(t, testpilotspb.RUN_EVENT_FIELD_RUN_ID, testpilotspb.RunEventField(0).Descriptor().Values().Len()-1)
	// Provenance is typed rows, one repeated field per kind so a later row kind is one more field,
	// rather than bytes only their Producer can read.
	require.Equal(t, []protoreflect.Name{
		"producer_id", "producer_version", "definitions", "sources", "known_gaps", "correlated_rules",
		"local_names", "model_value_fingerprints", "abstraction_claims",
	}, fieldNames(messageDescriptor(t, "CaseProvenance")))
}

func TestPrepareFormatRejectsBeforePayloadAdmission(t *testing.T) {
	for _, major := range []int32{1, 2, 3, 99} {
		t.Run(fmt.Sprint(major), func(t *testing.T) {
			source, profile := facadeFixture(t)
			source.Version.Major = major
			source.Program.ProtoReflect().SetUnknown([]byte{0x98, 0x06, 0x01})
			prepared, err := Prepare(source, profile)
			require.Nil(t, prepared)
			var preparation *PreparationError
			require.ErrorAs(t, err, &preparation)
			require.Equal(t, PreparationUnsupported, preparation.Category)
			require.Equal(t, "version", preparation.Path)
		})
	}
}

func oneofNames(oneof protoreflect.OneofDescriptor) []protoreflect.Name {
	names := make([]protoreflect.Name, 0, oneof.Fields().Len())
	for index := range oneof.Fields().Len() {
		names = append(names, oneof.Fields().Get(index).Name())
	}
	return names
}

func TestProtocolUsesCohesivePublicVocabulary(t *testing.T) {
	t.Parallel()
	for _, path := range protocolFiles {
		_, err := protoregistry.GlobalFiles.FindFileByPath(path)
		require.NoError(t, err, path)
	}
	for _, retired := range []string{
		"ValueExpression", "Case" + "Metadata", "CaseDefinition" + "Binding", "CaseDefinition" + "Kind",
		"CaseKnown" + "Gap", "CaseKnownGapK" + "ind", "OptionalString", "SlotSchema", "SlotKind",
		"ActivationBinding", "Entrypoint" + "Context", "RunEventKinds", "RunEventSequence",
		"RuleVerdictKind", "VerdictKind",
		"Run" + "Status", "Correlated" + "Value", "ContractRule" + "Definition", "ContractState" + "Definition",
		"ContractTransition" + "Definition", "ContractCapture" + "Definition",
		"Response" + "Projection", "Projection" + "Target", "Projection" + "Kind", "Opaque" + "CapabilityType", "InvokeRPC",
		"Role" + "Definition", "Slot" + "Definition", "Observation" + "Definition", "Entrypoint" + "Definition",
		"Cleanup" + "Definition", "Instruction" + "Definition", "Instruction" + "Ref",
		"ContractHorizon" + "Definition",
		"Program" + "Expression", "Contract" + "Expression", "Program" + "PathExpression", "Contract" + "PathExpression",
		"Program" + "PresentExpression", "Contract" + "PresentExpression", "ProgramEquals" + "Expression",
		"ContractEquals" + "Expression", "Program" + "CompareExpression", "Contract" + "CompareExpression",
		"Program" + "NotExpression", "Contract" + "NotExpression", "Program" + "AllExpression", "Contract" + "AllExpression",
		"Program" + "AnyExpression", "Contract" + "AnyExpression", "Equals" + "Expression",
		"Slot" + "Ref", "Run" + "Ref", "Observation" + "Ref", "Capture" + "Ref", "Environment" + "Ref",
		"InstructionOutcome" + "Ref", "RunEventField" + "Ref", "CorrelatedCapture" + "Ref",
		"Correlated" + "Predicate", "Correlated" + "PredicateField", "Correlated" + "Comparison",
		"Correlated" + "ComparisonOperator", "Correlated" + "Operand", "Correlated" + "Correlation",
		"Correlated" + "CorrelationGroup",
		"RUN_EVENT_FIELD_" + "FAULT_ROLE_ID", "RUN_EVENT_FIELD_" + "FAULT_KIND",
		"Contract" + "Deadline", "Contract" + "CaptureType", "Correlated" + "Binding", "CorrelatedEvidence" + "Field",
		"CorrelatedEvidence" + "Binding", "Entrypoint" + "Kind",
		"Scoped" + "Binding", "Scoped" + "CaptureDeclaration", "ScopedCapture" + "Ref", "Scoped" + "Clause",
		"Scoped" + "Clock", "Scoped" + "Comparison", "Scoped" + "ComparisonOperator", "Scoped" + "Contract",
		"Scoped" + "Correlation", "Scoped" + "CorrelationGroup", "Scoped" + "Endpoint", "Scoped" + "Evidence",
		"Scoped" + "EvidenceBinding", "Scoped" + "EvidenceField", "Scoped" + "EvidenceMeaning",
		"Scoped" + "EvidenceProjection", "Scoped" + "EvidenceRule", "Scoped" + "FieldDisposition",
		"Scoped" + "FieldPolicy", "Scoped" + "Identity", "Scoped" + "Limits", "Scoped" + "Operand",
		"Scoped" + "Predicate", "Scoped" + "PredicateField", "Scoped" + "ProjectionRule", "Scoped" + "Transition",
		"Scoped" + "Value",
		"StartNexus" + "Operation", "CompleteNexus" + "Operation", "Respond" + "Nexus", "NexusResponse" + "Kind",
		"Reference.model" + "_value",
		"Compare" + "Expression", "Present" + "Expression", "Path" + "Expression",
		"All" + "Expression", "Any" + "Expression", "Not" + "Expression", "Comparison" + "Operator",
		"Value", "Enum" + "Value", "List" + "Value", "Map" + "Value", "CorrelatedEvidence" + "Rule",
		"EvidenceField" + "Declaration", "EvidenceOperation" + "Key", "Correlated" + "Clock",
		"ContractRule" + "Kind", "ContractSupport" + "Kind", "Read" + "Cardinality",
		"Run" + "Reference", "ProjectedValue" + "Reference", "RunEventPayload" + "Reference",
	} {
		_, err := protoregistry.GlobalFiles.FindDescriptorByName(protoreflect.FullName("temporal.server.api.testpilot.v1." + retired))
		require.Error(t, err, retired)
	}
}

func TestCaseImportClosureExcludesRunOnlyMessages(t *testing.T) {
	t.Parallel()
	require.Empty(t, runOnlyImports(testpilotspb.File_temporal_server_api_testpilot_v1_case_proto))

	wrong, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Name: proto.String("test/case.proto"), Package: proto.String("test"), Syntax: proto.String("proto3"),
		Dependency: []string{
			testpilotspb.File_temporal_server_api_testpilot_v1_case_proto.Path(),
			testpilotspb.File_temporal_server_api_testpilot_v1_run_proto.Path(),
		},
	}, protoregistry.GlobalFiles)
	require.NoError(t, err)
	require.Equal(t, []string{
		"test/case.proto imports temporal/server/api/testpilot/v1/run.proto, which declares Run-only CleanupOutcome, FaultInjected, InstructionOutcome, Run, RunDiagnostic, RunEvent, Verdict",
	}, runOnlyImports(wrong))
}

// runOnlyImports walks root's transitive imports and names each file that pulls in a file
// declaring a Run-only message.
func runOnlyImports(root protoreflect.FileDescriptor) []string {
	var problems []string
	seen := map[string]bool{}
	var walk func(file protoreflect.FileDescriptor)
	walk = func(file protoreflect.FileDescriptor) {
		imports := file.Imports()
		for index := range imports.Len() {
			imported := imports.Get(index).FileDescriptor
			var declared []string
			for _, name := range runOnlyMessages {
				if imported.Messages().ByName(name) != nil {
					declared = append(declared, string(name))
				}
			}
			if len(declared) > 0 {
				slices.Sort(declared)
				problems = append(problems, fmt.Sprintf("%s imports %s, which declares Run-only %s", file.Path(), imported.Path(), strings.Join(declared, ", ")))
			}
			if !seen[imported.Path()] {
				seen[imported.Path()] = true
				walk(imported)
			}
		}
	}
	walk(root)
	return problems
}

func fieldNames(message protoreflect.MessageDescriptor) []protoreflect.Name {
	names := make([]protoreflect.Name, 0, message.Fields().Len())
	for index := range message.Fields().Len() {
		names = append(names, message.Fields().Get(index).Name())
	}
	return names
}

func messageDescriptor(t *testing.T, name protoreflect.Name) protoreflect.MessageDescriptor {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("temporal.server.api.testpilot.v1." + protoreflect.FullName(name))
	require.NoError(t, err)
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	require.True(t, ok)
	return message
}
