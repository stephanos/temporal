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
	// A Model that sets each field of schemaAdditions, which no captured wire bytes can hold.
	currentSupplement = `{"functions":[{"name":"g","body":{"construct":{"type":"umpire.Step","choice":"committed"}}}]}`
)

// schemaAdditions are the fields the schema gained after the rename, by full name. The captured
// descriptor and wire bytes stay as they were captured; each addition is checked on its own instead.
var schemaAdditions = []protoreflect.FullName{
	"temporal.server.api.umpire.v1.Construct.choice",
}

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
	current := protodesc.ToFileDescriptorProto(umpirespb.File_temporal_server_api_umpire_v1_ir_proto)
	added(t, expected, current)
	protorequire.ProtoEqual(t, expected, current)
}

// added appends each of schemaAdditions to the captured descriptor as the current schema declares
// it, with the synthetic oneof of a field with presence, so that the descriptors compare equal only
// when nothing else changed.
func added(t *testing.T, captured, current *descriptorpb.FileDescriptorProto) {
	t.Helper()
	for _, name := range schemaAdditions {
		message, field := string(name.Parent().Name()), string(name.Name())
		in := func(file *descriptorpb.FileDescriptorProto) *descriptorpb.DescriptorProto {
			i := slices.IndexFunc(file.GetMessageType(), func(m *descriptorpb.DescriptorProto) bool { return m.GetName() == message })
			require.GreaterOrEqual(t, i, 0, "%s names no top-level message", name)
			return file.GetMessageType()[i]
		}
		was, is := in(captured), in(current)
		require.False(t, slices.ContainsFunc(was.GetField(), func(f *descriptorpb.FieldDescriptorProto) bool { return f.GetName() == field }),
			"%s is in the captured descriptor already", name)
		i := slices.IndexFunc(is.GetField(), func(f *descriptorpb.FieldDescriptorProto) bool { return f.GetName() == field })
		require.GreaterOrEqual(t, i, 0, "the schema has no field %s", name)
		f := proto.CloneOf(is.GetField()[i])
		if f.OneofIndex != nil {
			require.True(t, f.GetProto3Optional(), "%s is added to a oneof, which a later addition cannot do", name)
			f.OneofIndex = proto.Int32(int32(len(was.GetOneofDecl())))
			was.OneofDecl = append(was.OneofDecl, proto.CloneOf(is.GetOneofDecl()[is.GetField()[i].GetOneofIndex()]))
		}
		was.Field = append(was.Field, f)
	}
}

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

	set := map[protoreflect.FullName]bool{}
	for name, wire := range captured {
		expected := new(umpirespb.Model)
		require.NoError(t, protojson.Unmarshal(sources[name], expected), name)
		decoded := new(umpirespb.Model)
		require.NoError(t, proto.Unmarshal(wire, decoded), name)
		protorequire.ProtoEqual(t, expected, decoded)
		encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(expected)
		require.NoError(t, err, name)
		require.Equal(t, golden.Digest(wire), golden.Digest(encoded), name)
		schemaFieldsSet(decoded.ProtoReflect(), set)
	}
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
	require.ElementsMatch(t, schemaAdditions, unset, "the captured wire bytes set every field of the schema but its additions")
}

// TestSchemaAdditionsRoundTrip sets every field the schema gained after the rename and reads it back
// from its wire bytes.
func TestSchemaAdditionsRoundTrip(t *testing.T) {
	m := new(umpirespb.Model)
	require.NoError(t, protojson.Unmarshal([]byte(currentSupplement), m))
	set := map[protoreflect.FullName]bool{}
	schemaFieldsSet(m.ProtoReflect(), set)
	for _, name := range schemaAdditions {
		require.True(t, set[name], "currentSupplement sets no %s", name)
	}
	wire, err := proto.MarshalOptions{Deterministic: true}.Marshal(m)
	require.NoError(t, err)
	decoded := new(umpirespb.Model)
	require.NoError(t, proto.Unmarshal(wire, decoded))
	protorequire.ProtoEqual(t, m, decoded)
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
