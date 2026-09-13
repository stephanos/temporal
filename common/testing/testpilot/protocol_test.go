package testpilot

import (
	"fmt"
	"slices"
	"strings"
	"testing"

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
	"temporal/server/api/testpilot/v1/program.proto",
	"temporal/server/api/testpilot/v1/instruction.proto",
	"temporal/server/api/testpilot/v1/contract.proto",
	"temporal/server/api/testpilot/v1/correlated.proto",
	"temporal/server/api/testpilot/v1/event.proto",
	"temporal/server/api/testpilot/v1/run.proto",
}

// runOnlyMessages are declared for a Run and must stay outside the Case import closure.
var runOnlyMessages = []protoreflect.Name{"Run", "Verdict", "RunDiagnostic", "RunEvent", "CleanupOutcome", "FaultInjected"}

func TestProtocolEncodesExpressionAndStateScopes(t *testing.T) {
	t.Parallel()
	programExpression := messageDescriptor(t, "ProgramExpression")
	require.NotNil(t, programExpression.Oneofs().ByName("expression"))
	require.Nil(t, programExpression.Fields().ByName("observation"))
	require.Nil(t, programExpression.Fields().ByName("capture"))
	require.Nil(t, programExpression.Fields().ByName("run_event"))
	contractExpression := messageDescriptor(t, "ContractExpression")
	require.NotNil(t, contractExpression.Oneofs().ByName("expression"))
	require.Nil(t, contractExpression.Fields().ByName("slot"))
	require.Nil(t, contractExpression.Fields().ByName("outcome"))
	slot := messageDescriptor(t, "Slot")
	require.NotNil(t, slot.Oneofs().ByName("content"))
	require.Nil(t, slot.Fields().ByName("kind"))
	entrypoint := messageDescriptor(t, "Entrypoint")
	require.NotNil(t, entrypoint.Oneofs().ByName("activation"))
	require.Nil(t, entrypoint.Fields().ByName("context"))
	diagnostic := messageDescriptor(t, "RunDiagnostic")
	require.True(t, diagnostic.Fields().ByName("supporting_event_sequence").HasPresence())
	run := messageDescriptor(t, "Run")
	require.True(t, run.Fields().ByName("evaluation_failure_sequence").HasPresence())
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
		"Response" + "Projection", "Projection" + "Target", "Projection" + "Kind", "OpaqueCapability" + "Type", "InvokeRPC",
		"Role" + "Definition", "Slot" + "Definition", "Observation" + "Definition", "Entrypoint" + "Definition",
		"Cleanup" + "Definition", "Instruction" + "Definition", "Instruction" + "Ref",
		"ContractHorizon" + "Definition",
		"Scoped" + "Binding", "Scoped" + "CaptureDeclaration", "Scoped" + "CaptureRef", "Scoped" + "Clause",
		"Scoped" + "Clock", "Scoped" + "Comparison", "Scoped" + "ComparisonOperator", "Scoped" + "Contract",
		"Scoped" + "Correlation", "Scoped" + "CorrelationGroup", "Scoped" + "Endpoint", "Scoped" + "Evidence",
		"Scoped" + "EvidenceBinding", "Scoped" + "EvidenceField", "Scoped" + "EvidenceMeaning",
		"Scoped" + "EvidenceProjection", "Scoped" + "EvidenceRule", "Scoped" + "FieldDisposition",
		"Scoped" + "FieldPolicy", "Scoped" + "Identity", "Scoped" + "Limits", "Scoped" + "Operand",
		"Scoped" + "Predicate", "Scoped" + "PredicateField", "Scoped" + "ProjectionRule", "Scoped" + "Transition",
		"Scoped" + "Value",
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
		"test/case.proto imports temporal/server/api/testpilot/v1/run.proto, which declares Run-only CleanupOutcome, FaultInjected, Run, RunDiagnostic, RunEvent, Verdict",
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

func messageDescriptor(t *testing.T, name protoreflect.Name) protoreflect.MessageDescriptor {
	t.Helper()
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("temporal.server.api.testpilot.v1." + protoreflect.FullName(name))
	require.NoError(t, err)
	message, ok := descriptor.(protoreflect.MessageDescriptor)
	require.True(t, ok)
	return message
}
