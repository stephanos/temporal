package testpilot

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// Generated Go descriptors carry no source info, so the comment check reads a descriptor set that
// `make umpire-check-testpilot-protocol` builds with --include_source_info.
const protocolDescriptorSetVariable = "TESTPILOT_PROTOCOL_DESCRIPTOR_SET"

var apiLinterDirective = regexp.MustCompile(`(?s)\(--.*?--\)`)

func TestProtocolMessagesCarryLeadingComments(t *testing.T) {
	path := os.Getenv(protocolDescriptorSetVariable)
	if path == "" {
		t.Skipf("%s is unset; make umpire-check-testpilot-protocol runs this check", protocolDescriptorSetVariable)
	}
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	set := new(descriptorpb.FileDescriptorSet)
	require.NoError(t, proto.Unmarshal(encoded, set))
	var names, problems []string
	for _, source := range set.GetFile() {
		names = append(names, source.GetName())
		file, err := protodesc.NewFile(source, protoregistry.GlobalFiles)
		require.NoError(t, err, source.GetName())
		problems = append(problems, protocolDocumentationProblems(file)...)
	}
	require.ElementsMatch(t, protocolFiles, names)
	require.Empty(t, problems)
}

func TestProtocolDocumentationProblemsNameEachViolation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		messages []syntheticMessage
		enums    []syntheticEnum
		want     []string
	}{
		{
			name:     "documented dense messages and enums",
			messages: []syntheticMessage{{name: "Documented", comment: " Documented is described.\n", numbers: []int32{1, 2}}},
			enums:    []syntheticEnum{{name: "Kind", comment: " Kind is described.\n"}},
		},
		{
			name:     "message without a leading comment",
			messages: []syntheticMessage{{name: "Undocumented", numbers: []int32{1}}},
			want:     []string{"test.Undocumented has no leading comment"},
		},
		{
			name:  "enum whose only comment is an api-linter directive",
			enums: []syntheticEnum{{name: "Kind", comment: " (-- api-linter: core::0191::file-layout=disabled --)\n"}},
			want:  []string{"test.Kind has no leading comment"},
		},
		{
			name:     "field numbers that skip",
			messages: []syntheticMessage{{name: "Sparse", comment: " Sparse is described.\n", numbers: []int32{1, 3}}},
			want:     []string{"test.Sparse field numbers [1 3] are not dense from 1"},
		},
		{
			name:     "declaration order that differs from number order",
			messages: []syntheticMessage{{name: "Swapped", comment: " Swapped is described.\n", numbers: []int32{2, 1}}},
			want:     []string{"test.Swapped declares field numbers [2 1] out of number order"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			file, err := protodesc.NewFile(syntheticProtocolFile(test.messages, test.enums), new(protoregistry.Files))
			require.NoError(t, err)
			require.Equal(t, test.want, protocolDocumentationProblems(file))
		})
	}
}

// protocolDocumentationProblems names each message or enum in file without a leading comment,
// ignoring api-linter directives, and each message whose field numbers are not 1..n in
// declaration order.
func protocolDocumentationProblems(file protoreflect.FileDescriptor) []string {
	var problems []string
	locations := file.SourceLocations()
	requireComment := func(descriptor protoreflect.Descriptor) {
		comment := apiLinterDirective.ReplaceAllString(locations.ByDescriptor(descriptor).LeadingComments, "")
		if strings.TrimSpace(comment) == "" {
			problems = append(problems, fmt.Sprintf("%s has no leading comment", descriptor.FullName()))
		}
	}
	var visit func(messages protoreflect.MessageDescriptors, enums protoreflect.EnumDescriptors)
	visit = func(messages protoreflect.MessageDescriptors, enums protoreflect.EnumDescriptors) {
		for index := range enums.Len() {
			requireComment(enums.Get(index))
		}
		for index := range messages.Len() {
			message := messages.Get(index)
			requireComment(message)
			fields := message.Fields()
			numbers := make([]int, fields.Len())
			for field := range fields.Len() {
				numbers[field] = int(fields.Get(field).Number())
			}
			sorted := slices.Sorted(slices.Values(numbers))
			dense := true
			for position, number := range sorted {
				dense = dense && number == position+1
			}
			if !dense {
				problems = append(problems, fmt.Sprintf("%s field numbers %v are not dense from 1", message.FullName(), numbers))
			} else if !slices.Equal(numbers, sorted) {
				problems = append(problems, fmt.Sprintf("%s declares field numbers %v out of number order", message.FullName(), numbers))
			}
			visit(message.Messages(), message.Enums())
		}
	}
	visit(file.Messages(), file.Enums())
	return problems
}

type syntheticMessage struct {
	name    string
	comment string
	numbers []int32
}

type syntheticEnum struct {
	name    string
	comment string
}

// syntheticProtocolFile builds a descriptor with the source locations protoc records for leading
// comments: path [4, i] for the i-th message and [5, i] for the i-th enum.
func syntheticProtocolFile(messages []syntheticMessage, enums []syntheticEnum) *descriptorpb.FileDescriptorProto {
	file := &descriptorpb.FileDescriptorProto{
		Name: proto.String("test/protocol.proto"), Package: proto.String("test"), Syntax: proto.String("proto3"),
		SourceCodeInfo: &descriptorpb.SourceCodeInfo{},
	}
	for index, message := range messages {
		descriptor := &descriptorpb.DescriptorProto{Name: proto.String(message.name)}
		for _, number := range message.numbers {
			descriptor.Field = append(descriptor.Field, &descriptorpb.FieldDescriptorProto{
				Name: proto.String(fmt.Sprintf("field_%d", number)), Number: proto.Int32(number),
				Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(), Type: descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
			})
		}
		file.MessageType = append(file.MessageType, descriptor)
		file.SourceCodeInfo.Location = append(file.SourceCodeInfo.Location, syntheticLocation([]int32{4, int32(index)}, message.comment))
	}
	for index, enum := range enums {
		file.EnumType = append(file.EnumType, &descriptorpb.EnumDescriptorProto{
			Name:  proto.String(enum.name),
			Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String(strings.ToUpper(enum.name) + "_UNSPECIFIED"), Number: proto.Int32(0)}},
		})
		file.SourceCodeInfo.Location = append(file.SourceCodeInfo.Location, syntheticLocation([]int32{5, int32(index)}, enum.comment))
	}
	return file
}

func syntheticLocation(path []int32, comment string) *descriptorpb.SourceCodeInfo_Location {
	location := &descriptorpb.SourceCodeInfo_Location{Path: path, Span: []int32{0, 0, 1}}
	if comment != "" {
		location.LeadingComments = proto.String(comment)
	}
	return location
}
