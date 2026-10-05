package model

import (
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/common/testing/protorequire"
	"go.temporal.io/server/tools/umpire/internal/golden"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

// The IR schema was renamed once, from the package below to umpire/v1. testdata/schema/before-rename
// holds what the schema generated before that: its file descriptor, and the wire bytes of every
// frozen migration input and of schemaSupplement. Nothing regenerates it.
const (
	schemaPackageBefore = "temporal.server.api.modelir.v1"
	schemaPackage       = "temporal.server.api.umpire.v1"
	// The fields no frozen migration input sets.
	schemaSupplement = `{"version":1,"functions":[{"name":"f","params":[{"name":"p","type":{"intRange":{"low":"-3","high":"4"}}}],"body":{"match":{"scrutinee":{"literal":{"record":{"type":"r","fields":[{"list":{"items":[{"int":"-1"},{"bool":true}]}}]}}},"cases":[{"pattern":{"wildcard":{}},"guard":{"literal":{"bool":true}},"body":{"literal":{"text":"x"}}}]}}}]}`
	// The fields schemaAdded lists, each set. It is current, not captured: no historical bytes have them.
	schemaAddedSupplement = `{"queries":[{"name":"q","total":"48"}],"functions":[{"name":"g","body":{"construct":{"type":"umpire.Step","choice":"committed"}}}],"realizations":[{"requiredSettings":[{"key":"k","value":"v"}],` +
		`"behavior":{"visibility":[{"id":"v","position":{"file":"f"},"method":"/s/W","read":"/s/R","eventuallyWithin":{"position":{"file":"f"},"intervalMs":"1","atMostMs":"2"}},{"cause":"CAUSE_KIND_TIMER"}],` +
		`"causes":[{"id":"c","position":{"file":"f"},"kind":"CAUSE_KIND_TIMER","bound":{"intervalMs":"1"}}],` +
		`"attemptNumbering":{"position":{"file":"f"},"first":"1","oneRun":true},` +
		`"instructionDefaults":{"position":{"file":"f"},"timeoutMs":"1","attempts":"1"},"runOrderIsCausal":true},` +
		`"serverSteps":[{"position":{"file":"f"},"step":{"action":"a"},"kind":"CAUSE_KIND_TIMER","deadlineMs":"2"}]}]}`
)

// schemaAddedField is a field the schema gained after the capture: the message it was added to, by its
// name in the file (dotted when nested), and its descriptor.
type schemaAddedField struct {
	message string
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
		}}},
	}
)

func schemaBeforeRename(t *testing.T) map[string][]byte {
	t.Helper()
	captured, err := golden.Read(filepath.Join("testdata", "schema", "before-rename"))
	require.NoError(t, err)
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
	for _, added := range schemaAddedFields {
		messages := file.GetMessageType()
		var message *descriptorpb.DescriptorProto
		for _, name := range strings.Split(added.message, ".") {
			i := slices.IndexFunc(messages, func(m *descriptorpb.DescriptorProto) bool { return m.GetName() == name })
			require.NotEqual(t, -1, i, "no message %s", added.message)
			message, messages = messages[i], messages[i].GetNestedType()
		}
		for _, field := range message.GetField() {
			require.NotEqual(t, added.field.GetName(), field.GetName(), "%s.%s was captured", added.message, field.GetName())
			require.NotEqual(t, added.field.GetNumber(), field.GetNumber(), "%s.%s was captured", added.message, field.GetName())
		}
		message.Field = append(message.Field, proto.CloneOf(added.field))
	}
}

// The captured wire bytes still decode to what their sources say and re-encode byte for byte, and set
// every field the schema had at the capture: every field but those added since. schemaAddedSupplement
// sets those, and round-trips through wire bytes too, so together they set every field of the schema.
func TestSchemaRenameKeepsTheWireBytes(t *testing.T) {
	inputs, err := golden.Read(filepath.Join("testdata", "migration", "inputs"))
	require.NoError(t, err)
	sources := map[string][]byte{"wire/supplement.binpb": []byte(schemaSupplement)}
	for name, encoded := range inputs {
		sources["wire/"+strings.TrimSuffix(name, ".json")+".binpb"] = encoded
	}
	captured := schemaBeforeRename(t)
	delete(captured, "descriptor.binpb")
	require.Equal(t, slices.Sorted(maps.Keys(sources)), slices.Sorted(maps.Keys(captured)))

	historical := map[protoreflect.FullName]bool{}
	for name, wire := range captured {
		expected := new(umpirespb.Model)
		require.NoError(t, protojson.Unmarshal(sources[name], expected), name)
		decoded := new(umpirespb.Model)
		require.NoError(t, proto.Unmarshal(wire, decoded), name)
		protorequire.ProtoEqual(t, expected, decoded)
		encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(expected)
		require.NoError(t, err, name)
		require.Equal(t, golden.Digest(wire), golden.Digest(encoded), name)
		schemaFieldsSet(decoded.ProtoReflect(), historical)
	}
	var added []protoreflect.FullName
	for _, a := range schemaAddedFields {
		added = append(added, protoreflect.FullName(schemaPackage+"."+a.message+"."+a.field.GetName()))
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
