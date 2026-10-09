package ir

import (
	"compress/gzip"
	"crypto/sha256"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// The IR schema was renamed once, from the package below to umpire/v1. testdata/schema/before-rename
// holds what the schema generated before that: its file descriptor, and the wire bytes of the Models
// lifted then and of schemaSupplement. Nothing regenerates it.
const (
	schemaPackageBefore = "temporal.server.api.modelir.v1"
	schemaPackage       = "temporal.server.api.umpire.v1"
	// The fields none of the Models lifted at the capture sets.
	schemaSupplement = `{"version":1,"functions":[{"name":"f","params":[{"name":"p","type":{"intRange":{"low":"-3","high":"4"}}}],"body":{"match":{"scrutinee":{"literal":{"record":{"type":"r","fields":[{"list":{"items":[{"int":"-1"},{"bool":true}]}}]}}},"cases":[{"pattern":{"wildcard":{}},"guard":{"literal":{"bool":true}},"body":{"literal":{"text":"x"}}}]}}}]}`
	// The fields schemaAdded lists, each set. It is current, not captured: no historical bytes have them.
	schemaAddedSupplement = `{"properties":[{"name":"p","origin":{"name":"o","position":{"file":"f"}}}],"queries":[{"name":"q","total":"48","expectedRun":{"reason":"REASON_HOLE","conformanceReason":"REASON_INCOMPLETE","disposition":"DISPOSITION_COMPLETED","cleanup":"CLEANUP_SUCCEEDED","monitors":[{"reason":"REASON_HOLE"}]}}],"functions":[{"name":"g","body":{"construct":{"type":"framework.Step","choice":"committed"}}}],"realizations":[{"requiredSettings":[{"key":"k","value":"v"}],` +
		`"behavior":{"visibility":[{"id":"v","position":{"file":"f"},"method":"/s/W","read":"/s/R","eventuallyWithin":{"position":{"file":"f"},"intervalMs":"1","atMostMs":"2"}},{"cause":"CAUSE_KIND_TIMER"}],` +
		`"causes":[{"id":"c","position":{"file":"f"},"kind":"CAUSE_KIND_TIMER","bound":{"intervalMs":"1"}}],` +
		`"attemptNumbering":{"position":{"file":"f"},"first":"1","oneRun":true},` +
		`"instructionDefaults":{"position":{"file":"f"},"timeoutMs":"1","attempts":"1"},"runOrderIsCausal":true},` +
		`"serverSteps":[{"position":{"file":"f"},"step":{"action":"a"},"kind":"CAUSE_KIND_TIMER","deadlineMs":"2","timeoutBasis":"TIMEOUT_BASIS_HEARTBEAT"}],` +
		`"rejectionCodes":[{"rejection":"REJECTION_NOT_FOUND","grpcCode":"NOT_FOUND"}],` +
		`"externalSettlements":[{"position":{"file":"f"},"carrier":"c","activity":"a","attempt":"1","pending":"p","held":"h","answer":"a","requestCancel":"r",` +
		`"settlement":"s","cleanup":{"attemptWithheld":{"mode":"WITHHOLDING_MODE_SDK_PENDING","externalSettlement":"s"}}},` +
		`{"cleanup":{"attemptHeartbeat":{"details":{"fields":[{"name":"d","value":{"messages":{"values":[{"message":"m"}]}}}]}}}},` +
		`{"cleanup":{"awaitActivityPublication":"p"}}],` +
		`"resetSettlements":[{"position":{"file":"f"},"carrier":"c","activity":"a","attempt":"1","pending":"p","held":"h","resetRequest":"r",` +
		`"timer":{"action":"t"},"freshAttempt":"1","settlement":"s","cleanup":{"awaitActivityPublication":"p"}}]}]}`
)

// schemaAddedField is a field the schema gained after the capture: the message it was added to, by its
// name in the file (dotted when nested), and its descriptor. It is declared last, or right before the
// captured field named before.
type schemaAddedField struct {
	message string
	before  string
	field   *descriptorpb.FieldDescriptorProto
}

// schemaAddedMessage is a top-level message the schema gained after the capture, declared in the file
// right after the message named after.
type schemaAddedMessage struct {
	after   string
	message *descriptorpb.DescriptorProto
}

// schemaAddedEnum is a top-level enum the schema gained after the capture, declared in the file right
// after the enum named after, or first where after is empty.
type schemaAddedEnum struct {
	after string
	enum  *descriptorpb.EnumDescriptorProto
}

// schemaAddedNestedEnum is an enum the schema gained after the capture inside a captured message, by
// the message's name in the file, declared after the enums it had.
type schemaAddedNestedEnum struct {
	message string
	enum    *descriptorpb.EnumDescriptorProto
}

// schemaReplacedField is a captured field the schema replaced after the capture by a field of another
// type: the captured field's number is reserved, and the replacement takes its place in the message.
type schemaReplacedField struct {
	message     string
	replaced    string
	replacement *descriptorpb.FieldDescriptorProto
}

// schemaRenamedField is a captured field the schema renamed after the capture, keeping its number
// and type: the wire bytes are unchanged, and the ProtoJSON key is the new name.
type schemaRenamedField struct {
	message string
	from    string
	to      string
}

// schemaEnumOf is the descriptor protoc gives an enum, its values numbered from 0 in order.
func schemaEnumOf(name string, values ...string) *descriptorpb.EnumDescriptorProto {
	enum := &descriptorpb.EnumDescriptorProto{Name: proto.String(name)}
	for i, value := range values {
		enum.Value = append(enum.Value, &descriptorpb.EnumValueDescriptorProto{Name: proto.String(value), Number: proto.Int32(int32(i))})
	}
	return enum
}

// schemaFieldOf is the descriptor protoc gives a field added since the capture. A message's or an
// enum's type is named within the package; a field of a oneof names the oneof's index.
func schemaFieldOf(name string, number int32, label descriptorpb.FieldDescriptorProto_Label, typ descriptorpb.FieldDescriptorProto_Type,
	typeName, jsonName string, oneof ...int32) *descriptorpb.FieldDescriptorProto {
	field := &descriptorpb.FieldDescriptorProto{Name: proto.String(name), Number: proto.Int32(number), Label: label.Enum(), Type: typ.Enum(),
		JsonName: proto.String(jsonName)}
	if typeName != "" {
		field.TypeName = proto.String("." + schemaPackage + "." + typeName)
	}
	for _, index := range oneof {
		field.OneofIndex = proto.Int32(index)
	}
	return field
}

const (
	schemaOptional = descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	schemaRepeated = descriptorpb.FieldDescriptorProto_LABEL_REPEATED
	schemaString   = descriptorpb.FieldDescriptorProto_TYPE_STRING
	schemaInt64    = descriptorpb.FieldDescriptorProto_TYPE_INT64
	schemaBool     = descriptorpb.FieldDescriptorProto_TYPE_BOOL
	schemaEnum     = descriptorpb.FieldDescriptorProto_TYPE_ENUM
	schemaMessage  = descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
)

// What the schema gained after the capture, in the order it was added: the files it imports, the
// messages, the enums and the fields. The lists are closed: a field added to the schema, a message
// or an enum, fails these tests until it is listed here and schemaAddedSupplement sets each field.
var (
	schemaAddedDependencies = []string{
		// Query.total's wrapper.
		"google/protobuf/wrappers.proto",
	}
	schemaAddedFields = []schemaAddedField{
		// The author's static combination count (model/SEMANTICS.md, Query totals).
		{message: "Query", field: &descriptorpb.FieldDescriptorProto{Name: proto.String("total"), Number: proto.Int32(10),
			Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
			TypeName: proto.String(".google.protobuf.Int64Value"), JsonName: proto.String("total")}},
		// The name of a named choice's alternative on its step record (model/SEMANTICS.md, Named choices).
		{message: "Construct", field: &descriptorpb.FieldDescriptorProto{Name: proto.String("choice"), Number: proto.Int32(4),
			Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			JsonName: proto.String("choice")}},
		// The dynamic-configuration settings a realization requires of the system it runs (fn-122.4).
		{message: "Realization", field: &descriptorpb.FieldDescriptorProto{Name: proto.String("required_settings"), Number: proto.Int32(15),
			Label: descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
			TypeName: proto.String("." + schemaPackage + ".RequiredSetting"), JsonName: proto.String("requiredSettings")}},
		// How the APIs a realization calls behave between calls, and its server steps (fn-118.2).
		{message: "Realization", field: schemaFieldOf("behavior", 16, schemaOptional, schemaMessage, "ApiBehavior", "behavior")},
		{message: "Realization", field: schemaFieldOf("server_steps", 17, schemaRepeated, schemaMessage, "ServerStep", "serverSteps")},
		// The shared rejection reasons and the gRPC status codes a realization returns (fn-139.8).
		{message: "Realization", field: schemaFieldOf("rejection_codes", 18, schemaRepeated, schemaMessage, "RejectionCode", "rejectionCodes")},
		// How an expected Run ends and how its cleanup ends, declared (fn-124.5).
		{message: "RunExpectation", field: schemaFieldOf("disposition", 6, schemaOptional, schemaEnum, "RunExpectation.Disposition", "disposition")},
		{message: "RunExpectation", field: schemaFieldOf("cleanup", 7, schemaOptional, schemaEnum, "RunExpectation.Cleanup", "cleanup")},
		// Why an expected Run's conformance is not conformant, by the judge's id (fn-124.6).
		{message: "RunExpectation", field: schemaFieldOf("conformance_reason", 9, schemaOptional, schemaEnum, "RunExpectation.Reason", "conformanceReason")},
		// The capability Property a generated Property was expanded from, inert (fn-134.1).
		{message: "Property", field: schemaFieldOf("origin", 8, schemaOptional, schemaMessage, "PropertyOrigin", "origin")},
		// The typed request field whose timeout supplies a timer server step (fn-129.1).
		{message: "ServerStep", field: schemaFieldOf("timeout_basis", 5, schemaOptional, schemaEnum, "TimeoutBasis", "timeoutBasis")},
		// An activity attempt's answer withheld, and its heartbeat (fn-128.3, fn-129.1).
		{message: "Command", before: "closes", field: schemaFieldOf("attempt_withheld", 20, schemaOptional, schemaMessage, "AttemptWithheld", "attemptWithheld", 0)},
		{message: "Command", before: "closes", field: schemaFieldOf("attempt_heartbeat", 21, schemaOptional, schemaMessage, "AttemptHeartbeat", "attemptHeartbeat", 0)},
		// An ordered list of typed protobuf messages (fn-129.1).
		{message: "ProtoValue", field: schemaFieldOf("messages", 10, schemaOptional, schemaMessage, "ProtoMessages", "messages", 0)},
		// The activity settlements a controller answers independently, by ID, and the wait for their
		// pending publication (fn-129.2).
		{message: "Realization", field: schemaFieldOf("external_settlements", 19, schemaRepeated, schemaMessage, "ActivityExternalSettlement",
			"externalSettlements")},
		{message: "Command", before: "closes", field: schemaFieldOf("await_activity_publication", 22, schemaOptional, schemaString, "", "awaitActivityPublication", 0)},
		// The activity resets a controller defers until the held attempt's pending record is published
		// (fn-129.3).
		{message: "Realization", field: schemaFieldOf("reset_settlements", 20, schemaRepeated, schemaMessage, "ActivityResetSettlement",
			"resetSettlements")},
	}
	// An expected Run's reasons, prose at the capture, are the judge's ids since fn-124.5, so the
	// captured wire bytes no longer encode their expected Runs as the schema now reads them.
	schemaReplacedFields = []schemaReplacedField{
		{message: "RunExpectation", replaced: "reason", replacement: schemaFieldOf("reason", 8, schemaOptional, schemaEnum, "RunExpectation.Reason", "reason")},
		{message: "MonitorExpectation", replaced: "reason", replacement: schemaFieldOf("reason", 4, schemaOptional, schemaEnum, "RunExpectation.Reason", "reason")},
	}
	schemaRenamedFields = []schemaRenamedField{
		// fn-126 decision 26: who takes an action is its actor.
		{message: "Action", from: "party", to: "actor"},
	}
	schemaAddedNestedEnums = []schemaAddedNestedEnum{
		{message: "RunExpectation", enum: schemaEnumOf("Disposition", "DISPOSITION_UNSPECIFIED", "DISPOSITION_COMPLETED", "DISPOSITION_STOPPED_BY_MONITOR",
			"DISPOSITION_INCOMPLETE")},
		{message: "RunExpectation", enum: schemaEnumOf("Cleanup", "CLEANUP_UNSPECIFIED", "CLEANUP_SUCCEEDED", "CLEANUP_FAILED", "CLEANUP_TIMED_OUT")},
		{message: "RunExpectation", enum: schemaEnumOf("Reason", "REASON_UNSPECIFIED", "REASON_NO_EVIDENCE", "REASON_INCOMPLETE", "REASON_HOLE",
			"REASON_UNEXPLAINED", "REASON_EXPLANATIONS_DISAGREE", "REASON_NEVER_EVALUATED", "REASON_UNREADABLE", "REASON_EVERY_EXPLANATION_VIOLATES")},
	}
	schemaAddedMessages = []schemaAddedMessage{
		// Realization.required_settings' entry.
		{after: "Realization", message: &descriptorpb.DescriptorProto{Name: proto.String("RequiredSetting"), Field: []*descriptorpb.FieldDescriptorProto{
			{Name: proto.String("key"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: proto.String("key")},
			{Name: proto.String("value"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: proto.String("value")},
		}}},
		// Realization.behavior and Realization.server_steps' messages.
		{after: "RequiredSetting", message: &descriptorpb.DescriptorProto{Name: proto.String("ApiBehavior"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("visibility", 1, schemaRepeated, schemaMessage, "Visibility", "visibility"),
			schemaFieldOf("causes", 2, schemaRepeated, schemaMessage, "CauseBound", "causes"),
			// How attempts are numbered, an instruction's default limits and causal run order (fn-124.3).
			schemaFieldOf("attempt_numbering", 3, schemaOptional, schemaMessage, "AttemptNumbering", "attemptNumbering"),
			schemaFieldOf("instruction_defaults", 4, schemaOptional, schemaMessage, "InstructionLimit", "instructionDefaults"),
			schemaFieldOf("run_order_is_causal", 5, schemaOptional, schemaBool, "", "runOrderIsCausal"),
		}}},
		{after: "ApiBehavior", message: &descriptorpb.DescriptorProto{Name: proto.String("AttemptNumbering"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("position", 1, schemaOptional, schemaMessage, "Position", "position"),
			schemaFieldOf("first", 2, schemaOptional, schemaInt64, "", "first"),
			schemaFieldOf("one_run", 3, schemaOptional, schemaBool, "", "oneRun"),
		}}},
		{after: "AttemptNumbering", message: &descriptorpb.DescriptorProto{Name: proto.String("InstructionLimit"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("position", 1, schemaOptional, schemaMessage, "Position", "position"),
			schemaFieldOf("timeout_ms", 2, schemaOptional, schemaInt64, "", "timeoutMs"),
			schemaFieldOf("attempts", 3, schemaOptional, schemaInt64, "", "attempts"),
		}}},
		{after: "InstructionLimit", message: &descriptorpb.DescriptorProto{Name: proto.String("Visibility"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("id", 1, schemaOptional, schemaString, "", "id"),
			schemaFieldOf("position", 2, schemaOptional, schemaMessage, "Position", "position"),
			schemaFieldOf("method", 3, schemaOptional, schemaString, "", "method", 0),
			schemaFieldOf("cause", 4, schemaOptional, schemaEnum, "CauseKind", "cause", 0),
			schemaFieldOf("read", 5, schemaOptional, schemaString, "", "read"),
			schemaFieldOf("eventually_within", 6, schemaOptional, schemaMessage, "WaitBound", "eventuallyWithin"),
		}, OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: proto.String("write")}}}},
		{after: "Visibility", message: &descriptorpb.DescriptorProto{Name: proto.String("WaitBound"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("position", 1, schemaOptional, schemaMessage, "Position", "position"),
			schemaFieldOf("interval_ms", 2, schemaOptional, schemaInt64, "", "intervalMs"),
			schemaFieldOf("at_most_ms", 3, schemaOptional, schemaInt64, "", "atMostMs"),
		}}},
		{after: "WaitBound", message: &descriptorpb.DescriptorProto{Name: proto.String("CauseBound"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("id", 1, schemaOptional, schemaString, "", "id"),
			schemaFieldOf("position", 2, schemaOptional, schemaMessage, "Position", "position"),
			schemaFieldOf("kind", 3, schemaOptional, schemaEnum, "CauseKind", "kind"),
			schemaFieldOf("bound", 4, schemaOptional, schemaMessage, "WaitBound", "bound"),
		}}},
		{after: "CauseBound", message: &descriptorpb.DescriptorProto{Name: proto.String("ServerStep"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("position", 1, schemaOptional, schemaMessage, "Position", "position"),
			schemaFieldOf("step", 2, schemaOptional, schemaMessage, "ActionClass", "step"),
			schemaFieldOf("kind", 3, schemaOptional, schemaEnum, "CauseKind", "kind"),
			schemaFieldOf("deadline_ms", 4, schemaOptional, schemaInt64, "", "deadlineMs"),
		}}},
		// Property.origin's message (fn-134.1).
		{after: "Property", message: &descriptorpb.DescriptorProto{Name: proto.String("PropertyOrigin"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("name", 1, schemaOptional, schemaString, "", "name"),
			schemaFieldOf("position", 2, schemaOptional, schemaMessage, "Position", "position"),
		}}},
		// Realization.rejection_codes' entry (fn-139.8).
		{after: "ProtoEntry", message: &descriptorpb.DescriptorProto{Name: proto.String("RejectionCode"),
			Field: []*descriptorpb.FieldDescriptorProto{
				schemaFieldOf("rejection", 1, schemaOptional, schemaEnum, "RejectionCode.Rejection", "rejection"),
				schemaFieldOf("grpc_code", 2, schemaOptional, schemaString, "", "grpcCode"),
			},
			EnumType: []*descriptorpb.EnumDescriptorProto{schemaEnumOf("Rejection", "REJECTION_UNSPECIFIED", "REJECTION_NOT_FOUND",
				"REJECTION_ALREADY_EXISTS", "REJECTION_FAILED_PRECONDITION", "REJECTION_INVALID_ARGUMENT")},
		}},
		// Command.attempt_heartbeat's and Command.attempt_withheld's messages (fn-128.3, fn-129.1, fn-129.2).
		{after: "Command", message: &descriptorpb.DescriptorProto{Name: proto.String("AttemptHeartbeat"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("details", 1, schemaOptional, schemaMessage, "Proto", "details"),
		}}},
		{after: "AttemptHeartbeat", message: &descriptorpb.DescriptorProto{Name: proto.String("AttemptWithheld"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("mode", 1, schemaOptional, schemaEnum, "WithholdingMode", "mode"),
			schemaFieldOf("external_settlement", 2, schemaOptional, schemaString, "", "externalSettlement"),
		}}},
		// ProtoValue.messages' message (fn-129.1).
		{after: "ProtoValue", message: &descriptorpb.DescriptorProto{Name: proto.String("ProtoMessages"), Field: []*descriptorpb.FieldDescriptorProto{
			schemaFieldOf("values", 1, schemaRepeated, schemaMessage, "Proto", "values"),
		}}},
		// Realization.external_settlements' entry (fn-129.2).
		{after: "Realization", message: &descriptorpb.DescriptorProto{Name: proto.String("ActivityExternalSettlement"),
			Field: []*descriptorpb.FieldDescriptorProto{
				schemaFieldOf("position", 1, schemaOptional, schemaMessage, "Position", "position"),
				schemaFieldOf("carrier", 2, schemaOptional, schemaString, "", "carrier"),
				schemaFieldOf("activity", 3, schemaOptional, schemaString, "", "activity"),
				schemaFieldOf("attempt", 4, schemaOptional, schemaInt64, "", "attempt"),
				schemaFieldOf("pending", 5, schemaOptional, schemaString, "", "pending"),
				schemaFieldOf("held", 6, schemaOptional, schemaString, "", "held"),
				schemaFieldOf("answer", 7, schemaOptional, schemaString, "", "answer"),
				schemaFieldOf("request_cancel", 8, schemaOptional, schemaString, "", "requestCancel"),
				schemaFieldOf("settlement", 9, schemaOptional, schemaString, "", "settlement"),
				schemaFieldOf("cleanup", 10, schemaOptional, schemaMessage, "Command", "cleanup"),
			},
		}},
		// Realization.reset_settlements' entry (fn-129.3).
		{after: "ActivityExternalSettlement", message: &descriptorpb.DescriptorProto{Name: proto.String("ActivityResetSettlement"),
			Field: []*descriptorpb.FieldDescriptorProto{
				schemaFieldOf("position", 1, schemaOptional, schemaMessage, "Position", "position"),
				schemaFieldOf("carrier", 2, schemaOptional, schemaString, "", "carrier"),
				schemaFieldOf("activity", 3, schemaOptional, schemaString, "", "activity"),
				schemaFieldOf("attempt", 4, schemaOptional, schemaInt64, "", "attempt"),
				schemaFieldOf("pending", 5, schemaOptional, schemaString, "", "pending"),
				schemaFieldOf("held", 6, schemaOptional, schemaString, "", "held"),
				schemaFieldOf("reset_request", 7, schemaOptional, schemaString, "", "resetRequest"),
				schemaFieldOf("timer", 8, schemaOptional, schemaMessage, "ActionClass", "timer"),
				schemaFieldOf("fresh_attempt", 9, schemaOptional, schemaInt64, "", "freshAttempt"),
				schemaFieldOf("settlement", 10, schemaOptional, schemaString, "", "settlement"),
				schemaFieldOf("cleanup", 11, schemaOptional, schemaMessage, "Command", "cleanup"),
			},
		}},
	}
	schemaAddedEnums = []schemaAddedEnum{
		// The kinds of asynchronous cause a CauseBound, a Visibility and a ServerStep name (fn-118.2).
		{enum: &descriptorpb.EnumDescriptorProto{Name: proto.String("CauseKind"), Value: []*descriptorpb.EnumValueDescriptorProto{
			{Name: proto.String("CAUSE_KIND_UNSPECIFIED"), Number: proto.Int32(0)},
			{Name: proto.String("CAUSE_KIND_ACTIVITY_ANSWER"), Number: proto.Int32(1)},
			{Name: proto.String("CAUSE_KIND_WORKFLOW_TASK"), Number: proto.Int32(2)},
			{Name: proto.String("CAUSE_KIND_HANDLER_REPLY"), Number: proto.Int32(3)},
			{Name: proto.String("CAUSE_KIND_DELIVERY"), Number: proto.Int32(4)},
			{Name: proto.String("CAUSE_KIND_TIMER"), Number: proto.Int32(5)},
			// An SDK activity heartbeat (fn-129.1).
			{Name: proto.String("CAUSE_KIND_ACTIVITY_HEARTBEAT"), Number: proto.Int32(6)},
		}}},
		// The typed request field whose timeout supplies a timer server step (fn-129.1).
		{after: "CauseKind", enum: schemaEnumOf("TimeoutBasis", "TIMEOUT_BASIS_UNSPECIFIED", "TIMEOUT_BASIS_SCHEDULE_TO_CLOSE",
			"TIMEOUT_BASIS_START_TO_CLOSE", "TIMEOUT_BASIS_HEARTBEAT")},
		// How an activity attempt's answer is withheld (fn-129.1).
		{after: "TimeoutBasis", enum: schemaEnumOf("WithholdingMode", "WITHHOLDING_MODE_CONTEXT", "WITHHOLDING_MODE_SDK_PENDING")},
	}
)

// schemaBeforeRename is every file under testdata/schema/before-rename, gunzipped, by its path there
// without the .gz.
func schemaBeforeRename(t *testing.T) map[string][]byte {
	t.Helper()
	dir := filepath.Join("testdata", "schema", "before-rename")
	captured := map[string][]byte{}
	require.NoError(t, filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		require.True(t, d.Type().IsRegular() && strings.HasSuffix(path, ".gz"), "unexpected capture file %s", path)
		file, err := os.Open(path)
		require.NoError(t, err)
		defer func() { require.NoError(t, file.Close()) }()
		reader, err := gzip.NewReader(file)
		require.NoError(t, err)
		data, err := io.ReadAll(reader)
		require.NoError(t, err)
		require.NoError(t, reader.Close())
		rel, err := filepath.Rel(dir, path)
		require.NoError(t, err)
		captured[filepath.ToSlash(strings.TrimSuffix(rel, ".gz"))] = data
		return nil
	}))
	return captured
}

// renamedSchema is the whole of the rename: the file's path and package, the Go and Java packages
// generated from it, and the package in every reference to one of its types.
func renamedSchema(t *testing.T, before *descriptorpb.FileDescriptorProto) *descriptorpb.FileDescriptorProto {
	t.Helper()
	file := proto.CloneOf(before)
	file.Name = proto.String("temporal/server/api/umpire/v1/ir.proto")
	file.Package = proto.String(schemaPackage)
	file.Options.GoPackage = proto.String("go.temporal.io/server/api/umpire/v1;umpire")
	file.Options.JavaPackage = proto.String("io.temporal.server.api.umpire.v1")
	var rename func([]*descriptorpb.DescriptorProto)
	rename = func(messages []*descriptorpb.DescriptorProto) {
		for _, message := range messages {
			for _, field := range message.GetField() {
				if field.TypeName == nil {
					continue
				}
				name, ok := strings.CutPrefix(field.GetTypeName(), "."+schemaPackageBefore+".")
				require.True(t, ok, "%s.%s refers to %s", message.GetName(), field.GetName(), field.GetTypeName())
				field.TypeName = proto.String("." + schemaPackage + "." + name)
			}
			rename(message.GetNestedType())
		}
	}
	rename(file.GetMessageType())
	return file
}

func TestSchemaRenameKeepsTheDescriptor(t *testing.T) {
	before := new(descriptorpb.FileDescriptorProto)
	require.NoError(t, proto.Unmarshal(schemaBeforeRename(t)["descriptor.binpb"], before))
	require.Equal(t, schemaPackageBefore, before.GetPackage())
	expected := renamedSchema(t, before)
	require.NotContains(t, prototext.Format(expected), "modelir")
	addedSinceTheCapture(t, expected)
	protorequire.ProtoEqual(t, expected, protodesc.ToFileDescriptorProto(umpirespb.File_temporal_server_api_umpire_v1_ir_proto))
}

// addedSinceTheCapture adds to the renamed capture what the schema gained after it, and nothing else:
// none of it was in the capture.
func addedSinceTheCapture(t *testing.T, file *descriptorpb.FileDescriptorProto) {
	t.Helper()
	for _, dependency := range schemaAddedDependencies {
		require.NotContains(t, file.GetDependency(), dependency)
		file.Dependency = append(file.Dependency, dependency)
	}
	for _, added := range schemaAddedMessages {
		messages := file.GetMessageType()
		require.False(t, slices.ContainsFunc(messages, func(m *descriptorpb.DescriptorProto) bool { return m.GetName() == added.message.GetName() }),
			"%s was captured", added.message.GetName())
		i := slices.IndexFunc(messages, func(m *descriptorpb.DescriptorProto) bool { return m.GetName() == added.after })
		require.NotEqual(t, -1, i, "no message %s", added.after)
		file.MessageType = slices.Insert(messages, i+1, proto.CloneOf(added.message))
	}
	for _, added := range schemaAddedEnums {
		enums := file.GetEnumType()
		require.False(t, slices.ContainsFunc(enums, func(e *descriptorpb.EnumDescriptorProto) bool { return e.GetName() == added.enum.GetName() }),
			"%s was captured", added.enum.GetName())
		i := -1
		if added.after != "" {
			i = slices.IndexFunc(enums, func(e *descriptorpb.EnumDescriptorProto) bool { return e.GetName() == added.after })
			require.NotEqual(t, -1, i, "no enum %s", added.after)
		}
		file.EnumType = slices.Insert(enums, i+1, proto.CloneOf(added.enum))
	}
	message := func(name string) *descriptorpb.DescriptorProto {
		messages := file.GetMessageType()
		var message *descriptorpb.DescriptorProto
		for _, part := range strings.Split(name, ".") {
			i := slices.IndexFunc(messages, func(m *descriptorpb.DescriptorProto) bool { return m.GetName() == part })
			require.NotEqual(t, -1, i, "no message %s", name)
			message, messages = messages[i], messages[i].GetNestedType()
		}
		return message
	}
	for _, added := range schemaAddedNestedEnums {
		m := message(added.message)
		require.False(t, slices.ContainsFunc(m.GetEnumType(), func(e *descriptorpb.EnumDescriptorProto) bool { return e.GetName() == added.enum.GetName() }),
			"%s.%s was captured", added.message, added.enum.GetName())
		m.EnumType = append(m.EnumType, proto.CloneOf(added.enum))
	}
	for _, replaced := range schemaReplacedFields {
		m := message(replaced.message)
		i := slices.IndexFunc(m.GetField(), func(f *descriptorpb.FieldDescriptorProto) bool { return f.GetName() == replaced.replaced })
		require.NotEqual(t, -1, i, "%s.%s was not captured", replaced.message, replaced.replaced)
		require.NotEqual(t, m.GetField()[i].GetType(), replaced.replacement.GetType(), "a replacement changes the type")
		number := m.GetField()[i].GetNumber()
		m.ReservedRange = append(m.ReservedRange, &descriptorpb.DescriptorProto_ReservedRange{Start: proto.Int32(number), End: proto.Int32(number + 1)})
		m.Field[i] = proto.CloneOf(replaced.replacement)
	}
	for _, renamed := range schemaRenamedFields {
		m := message(renamed.message)
		i := slices.IndexFunc(m.GetField(), func(f *descriptorpb.FieldDescriptorProto) bool { return f.GetName() == renamed.from })
		require.NotEqual(t, -1, i, "%s.%s was not captured", renamed.message, renamed.from)
		m.Field[i].Name, m.Field[i].JsonName = proto.String(renamed.to), proto.String(renamed.to)
	}
	for _, added := range schemaAddedFields {
		message := message(added.message)
		for _, field := range message.GetField() {
			require.NotEqual(t, added.field.GetName(), field.GetName(), "%s.%s was captured", added.message, field.GetName())
			require.NotEqual(t, added.field.GetNumber(), field.GetNumber(), "%s.%s was captured", added.message, field.GetName())
		}
		i := len(message.GetField())
		if added.before != "" {
			i = slices.IndexFunc(message.GetField(), func(f *descriptorpb.FieldDescriptorProto) bool { return f.GetName() == added.before })
			require.NotEqual(t, -1, i, "%s.%s was not captured", added.message, added.before)
		}
		message.Field = slices.Insert(message.Field, i, proto.CloneOf(added.field))
	}
}

// The captured wire bytes still decode under the renamed schema, re-encode byte for byte, and set
// every field the schema had at the capture: every field but those added since. The supplement decodes
// to what its source says. schemaAddedSupplement sets the fields added since, and round-trips through
// wire bytes too, so together they set every field of the schema.
func TestSchemaRenameKeepsTheWireBytes(t *testing.T) {
	captured := schemaBeforeRename(t)
	delete(captured, "descriptor.binpb")
	const lifts = "wire/lifter/testdata/lifts/expected/"
	require.ElementsMatch(t, []string{
		"wire/supplement.binpb",
		"wire/ir/activity.binpb", "wire/ir/activity-race.binpb", "wire/ir/activity-system.binpb",
		"wire/ir/nexus-caller.binpb", "wire/ir/nexus-close.binpb", "wire/ir/nexus-control.binpb",
		lifts + "admission.binpb", lifts + "channels.binpb", lifts + "closereset.binpb",
		lifts + "declarations.binpb", lifts + "presence.binpb", lifts + "realizations.binpb",
	}, slices.Collect(maps.Keys(captured)))
	historical := map[protoreflect.FullName]bool{}
	for name, wire := range captured {
		decoded := new(umpirespb.Model)
		require.NoError(t, proto.Unmarshal(wire, decoded), name)
		// Only the reasons an expected Run once wrote as text are no field of the schema now.
		require.Empty(t, schemaUnknownOutside(decoded.ProtoReflect(), "RunExpectation", "MonitorExpectation"), name)
		if name == "wire/supplement.binpb" {
			expected := new(umpirespb.Model)
			require.NoError(t, protojson.Unmarshal([]byte(schemaSupplement), expected))
			protorequire.ProtoEqual(t, expected, decoded)
		}
		// An expected Run's reasons were captured as text, a field since replaced (schemaReplacedFields),
		// so a capture with expected Runs no longer re-encodes as it was written.
		if !slices.ContainsFunc(decoded.GetQueries(), func(q *umpirespb.Query) bool { return q.GetExpectedRun() != nil }) {
			encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(decoded)
			require.NoError(t, err, name)
			require.Equal(t, sha256.Sum256(wire), sha256.Sum256(encoded), name)
		}
		schemaFieldsSet(decoded.ProtoReflect(), historical)
	}
	var added []protoreflect.FullName
	for _, a := range schemaAddedFields {
		added = append(added, protoreflect.FullName(schemaPackage+"."+a.message+"."+a.field.GetName()))
	}
	for _, r := range schemaReplacedFields {
		added = append(added, protoreflect.FullName(schemaPackage+"."+r.message+"."+r.replacement.GetName()))
	}
	for _, a := range schemaAddedMessages {
		for _, field := range a.message.GetField() {
			added = append(added, protoreflect.FullName(schemaPackage+"."+a.message.GetName()+"."+field.GetName()))
		}
	}
	require.ElementsMatch(t, added, schemaFieldsUnset(historical), "the captured wire bytes set every field but those added since")

	current := map[protoreflect.FullName]bool{}
	supplement := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal([]byte(schemaAddedSupplement), supplement))
	wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(supplement)
	require.NoError(t, err)
	decoded := new(umpirespb.Model)
	require.NoError(t, proto.Unmarshal(wire, decoded))
	protorequire.ProtoEqual(t, supplement, decoded)
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(decoded)
	require.NoError(t, err)
	require.Equal(t, wire, encoded)
	schemaFieldsSet(decoded.ProtoReflect(), current)
	for _, name := range added {
		require.True(t, current[name], "schemaAddedSupplement sets %s", name)
	}

	maps.Copy(current, historical)
	require.Empty(t, schemaFieldsUnset(current), "the captured wire bytes and schemaAddedSupplement set every field of the schema")
}

// schemaUnknownOutside is every message of m, by its full name, that carries bytes no field of the
// schema reads, but those named allowed.
func schemaUnknownOutside(m protoreflect.Message, allowed ...protoreflect.Name) []protoreflect.FullName {
	var out []protoreflect.FullName
	if len(m.GetUnknown()) > 0 && !slices.Contains(allowed, m.Descriptor().Name()) {
		out = append(out, m.Descriptor().FullName())
	}
	m.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case field.Message() == nil, field.IsMap():
		case field.IsList():
			for i := range value.List().Len() {
				out = append(out, schemaUnknownOutside(value.List().Get(i).Message(), allowed...)...)
			}
		default:
			out = append(out, schemaUnknownOutside(value.Message(), allowed...)...)
		}
		return true
	})
	return out
}

// schemaFieldsUnset is every field of the schema not in set.
func schemaFieldsUnset(set map[protoreflect.FullName]bool) []protoreflect.FullName {
	var unset []protoreflect.FullName
	var visit func(protoreflect.MessageDescriptors)
	visit = func(messages protoreflect.MessageDescriptors) {
		for i := range messages.Len() {
			message := messages.Get(i)
			for j := range message.Fields().Len() {
				if name := message.Fields().Get(j).FullName(); !set[name] {
					unset = append(unset, name)
				}
			}
			visit(message.Messages())
		}
	}
	visit(umpirespb.File_temporal_server_api_umpire_v1_ir_proto.Messages())
	return unset
}

func schemaFieldsSet(m protoreflect.Message, set map[protoreflect.FullName]bool) {
	m.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		set[field.FullName()] = true
		switch {
		case field.Message() == nil, field.IsMap():
		case field.IsList():
			for i := range value.List().Len() {
				schemaFieldsSet(value.List().Get(i).Message(), set)
			}
		default:
			schemaFieldsSet(value.Message(), set)
		}
		return true
	})
}
