package main

import (
	"cmp"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type schemaNodeProjection struct {
	ValueShape  string
	FileContext string
	Name        string
	Syntax      string
	Descriptor  string
	References  []string
}

type operationSchemaProjection struct {
	Nodes    []schemaNodeProjection
	Closures map[string][]string
	Inputs   []schemaNodeProjection
}

func projectOperationSchemas(files []protoreflect.FileDescriptor,
	messages map[protoreflect.FullName]protoreflect.MessageDescriptor,
	enums map[protoreflect.FullName]protoreflect.EnumDescriptor,
) (*operationSchemaProjection, error) {
	result := &operationSchemaProjection{Closures: make(map[string][]string)}
	for _, name := range sortedNames(messages) {
		node, err := projectMessageSchema(messages[name])
		if err != nil {
			return nil, err
		}
		result.Nodes = append(result.Nodes, node)
	}

	for _, name := range sortedNames(enums) {
		enum := enums[name]
		descriptor := protodesc.ToEnumDescriptorProto(enum)
		normalizeEnumDescriptor(descriptor)
		node, err := schemaNode(string(name), enum.ParentFile().Syntax().String(), descriptor)
		if err != nil {
			return nil, err
		}
		node.ValueShape = concreteEnumShape(enum)
		node.FileContext, err = schemaFileContext(enum.ParentFile())
		if err != nil {
			return nil, err
		}
		result.Nodes = append(result.Nodes, node)
	}
	slices.SortFunc(result.Nodes, func(a, b schemaNodeProjection) int { return strings.Compare(a.Name, b.Name) })
	var err error
	result.Closures, err = schemaClosures(result.Nodes)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		descriptor := protodesc.ToFileDescriptorProto(file)
		descriptor.SourceCodeInfo = nil
		normalizeFileDescriptor(descriptor)
		node, err := schemaNode(file.Path(), file.Syntax().String(), descriptor)
		if err != nil {
			return nil, err
		}
		node.References = slices.Clone(descriptor.Dependency)
		result.Inputs = append(result.Inputs, node)
	}
	return result, nil
}

func projectMessageSchema(message protoreflect.MessageDescriptor) (schemaNodeProjection, error) {
	descriptor := protodesc.ToDescriptorProto(message)
	normalizeMessageDescriptor(descriptor)
	node, err := schemaNode(string(message.FullName()), message.ParentFile().Syntax().String(), descriptor)
	if err != nil {
		return schemaNodeProjection{}, err
	}
	node.ValueShape = concreteMessageShape(message)
	fields := message.Fields()
	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		if field.Message() != nil {
			node.References = append(node.References, string(field.Message().FullName()))
		}
		if field.Enum() != nil {
			node.References = append(node.References, string(field.Enum().FullName()))
		}
	}
	for i := 0; i < message.Messages().Len(); i++ {
		node.References = append(node.References, string(message.Messages().Get(i).FullName()))
	}
	for i := 0; i < message.Enums().Len(); i++ {
		node.References = append(node.References, string(message.Enums().Get(i).FullName()))
	}
	for i := 0; i < message.Extensions().Len(); i++ {
		extension := message.Extensions().Get(i)
		if extension.Message() != nil {
			node.References = append(node.References, string(extension.Message().FullName()))
		}
		if extension.Enum() != nil {
			node.References = append(node.References, string(extension.Enum().FullName()))
		}
	}
	slices.Sort(node.References)
	node.References = slices.Compact(node.References)
	node.FileContext, err = schemaFileContext(message.ParentFile())
	if err != nil {
		return schemaNodeProjection{}, err
	}
	return node, nil
}

func schemaClosures(entries []schemaNodeProjection) (map[string][]string, error) {
	closures := make(map[string][]string)
	nodes := make(map[string]schemaNodeProjection, len(entries))
	for _, node := range entries {
		nodes[node.Name] = node
	}
	for _, node := range entries {
		visited := make(map[string]bool)
		pending := []string{node.Name}
		for len(pending) > 0 {
			name := pending[len(pending)-1]
			pending = pending[:len(pending)-1]
			if visited[name] {
				continue
			}
			dependency, ok := nodes[name]
			if !ok {
				return nil, fmt.Errorf("schema %q has unresolved dependency %q", node.Name, name)
			}
			visited[name] = true
			pending = append(pending, dependency.References...)
		}
		for name := range visited {
			closures[node.Name] = append(closures[node.Name], name)
		}
		slices.Sort(closures[node.Name])
	}
	return closures, nil
}

func schemaNode(name, syntax string, descriptor proto.Message) (schemaNodeProjection, error) {
	encoded, err := (proto.MarshalOptions{Deterministic: true}).Marshal(descriptor)
	if err != nil {
		return schemaNodeProjection{}, fmt.Errorf("encode structural schema %q: %w", name, err)
	}
	return schemaNodeProjection{Name: name, Syntax: syntax, Descriptor: hex.EncodeToString(encoded)}, nil
}

func schemaFileContext(file protoreflect.FileDescriptor) (string, error) {
	descriptor := protodesc.ToFileDescriptorProto(file)
	context := &descriptorpb.FileDescriptorProto{Name: descriptor.Name, Package: descriptor.Package,
		Syntax: descriptor.Syntax, Edition: descriptor.Edition, Options: descriptor.Options}
	node, err := schemaNode(file.Path(), file.Syntax().String(), context)
	return node.Descriptor, err
}

func normalizeFileDescriptor(file *descriptorpb.FileDescriptorProto) {
	old := slices.Clone(file.Dependency)
	slices.Sort(file.Dependency)
	for _, indices := range [][]int32{file.PublicDependency, file.WeakDependency} {
		for i, index := range indices {
			indices[i] = int32(slices.Index(file.Dependency, old[index]))
		}
		slices.Sort(indices)
	}
	normalizeDeclarations(file.MessageType, file.EnumType, file.Extension)
	slices.SortFunc(file.Service, func(a, b *descriptorpb.ServiceDescriptorProto) int { return strings.Compare(a.GetName(), b.GetName()) })
	for _, service := range file.Service {
		slices.SortFunc(service.Method, func(a, b *descriptorpb.MethodDescriptorProto) int { return strings.Compare(a.GetName(), b.GetName()) })
	}
}

func normalizeDeclarations(messages []*descriptorpb.DescriptorProto, enums []*descriptorpb.EnumDescriptorProto, extensions []*descriptorpb.FieldDescriptorProto) {
	slices.SortFunc(messages, func(a, b *descriptorpb.DescriptorProto) int { return strings.Compare(a.GetName(), b.GetName()) })
	slices.SortFunc(enums, func(a, b *descriptorpb.EnumDescriptorProto) int { return strings.Compare(a.GetName(), b.GetName()) })
	slices.SortFunc(extensions, func(a, b *descriptorpb.FieldDescriptorProto) int { return cmp.Compare(a.GetNumber(), b.GetNumber()) })
	for _, message := range messages {
		normalizeMessageDescriptor(message)
	}
	for _, enum := range enums {
		normalizeEnumDescriptor(enum)
	}
}

func normalizeMessageDescriptor(message *descriptorpb.DescriptorProto) {
	old := slices.Clone(message.OneofDecl)
	synthetic := make(map[string]bool)
	for _, field := range message.Field {
		if field.GetProto3Optional() {
			synthetic[old[field.GetOneofIndex()].GetName()] = true
		}
	}
	slices.SortFunc(message.OneofDecl, func(a, b *descriptorpb.OneofDescriptorProto) int {
		if synthetic[a.GetName()] != synthetic[b.GetName()] {
			if synthetic[a.GetName()] {
				return 1
			}
			return -1
		}
		return strings.Compare(a.GetName(), b.GetName())
	})
	for _, field := range message.Field {
		if field.OneofIndex != nil {
			field.OneofIndex = proto.Int32(int32(slices.Index(message.OneofDecl, old[field.GetOneofIndex()])))
		}
	}
	slices.SortFunc(message.Field, func(a, b *descriptorpb.FieldDescriptorProto) int { return cmp.Compare(a.GetNumber(), b.GetNumber()) })
	slices.Sort(message.ReservedName)
	slices.SortFunc(message.ReservedRange, func(a, b *descriptorpb.DescriptorProto_ReservedRange) int {
		return cmp.Compare(a.GetStart(), b.GetStart())
	})
	slices.SortFunc(message.ExtensionRange, func(a, b *descriptorpb.DescriptorProto_ExtensionRange) int {
		return cmp.Compare(a.GetStart(), b.GetStart())
	})
	normalizeDeclarations(message.NestedType, message.EnumType, message.Extension)
}

func normalizeEnumDescriptor(enum *descriptorpb.EnumDescriptorProto) {
	// The first value supplies the implicit enum default, so its position is semantic.
	if len(enum.Value) > 1 {
		slices.SortFunc(enum.Value[1:], func(a, b *descriptorpb.EnumValueDescriptorProto) int {
			if order := cmp.Compare(a.GetNumber(), b.GetNumber()); order != 0 {
				return order
			}
			return strings.Compare(a.GetName(), b.GetName())
		})
	}
	slices.Sort(enum.ReservedName)
	slices.SortFunc(enum.ReservedRange, func(a, b *descriptorpb.EnumDescriptorProto_EnumReservedRange) int {
		return cmp.Compare(a.GetStart(), b.GetStart())
	})
}
