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
	schemaAddedSupplement = `{"queries":[{"name":"q","total":"48"}],"functions":[{"name":"g","body":{"construct":{"type":"umpire.Step","choice":"committed"}}}],"realizations":[{"requiredSettings":[{"key":"k","value":"v"}]}]}`
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

// What the schema gained after the capture, in the order it was added: the files it imports, the
// messages and the fields. The lists are closed: a field added to the schema, or a message, fails
// these tests until it is listed here and schemaAddedSupplement sets each of its fields.
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
	}
	schemaAddedMessages = []schemaAddedMessage{
		// Realization.required_settings' entry.
		{after: "Realization", message: &descriptorpb.DescriptorProto{Name: proto.String("RequiredSetting"), Field: []*descriptorpb.FieldDescriptorProto{
			{Name: proto.String("key"), Number: proto.Int32(1), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: proto.String("key")},
			{Name: proto.String("value"), Number: proto.Int32(2), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
				Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(), JsonName: proto.String("value")},
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
