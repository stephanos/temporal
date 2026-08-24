package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type sourceKind string

const (
	sourcePublic   sourceKind = "public"
	sourceInternal sourceKind = "internal"
	sourceCHASM    sourceKind = "chasm"
	sourceExternal sourceKind = "external"
)

type enumValueProjection struct {
	FullName   string `json:"fullName"`
	Name       string `json:"name"`
	Number     int32  `json:"number"`
	Deprecated bool   `json:"deprecated"`
}

type enumProjection struct {
	FullName     string                `json:"fullName"`
	Name         string                `json:"name"`
	Package      string                `json:"package"`
	Parent       string                `json:"parent,omitempty"`
	Source       sourceKind            `json:"source"`
	Values       []enumValueProjection `json:"values"`
	AllowAliases bool                  `json:"allowAliases"`
	Deprecated   bool                  `json:"deprecated"`
}

type fieldProjection struct {
	FullName   string `json:"fullName"`
	Name       string `json:"name"`
	JSONName   string `json:"jsonName"`
	Number     int32  `json:"number"`
	Kind       string `json:"kind"`
	TypeName   string `json:"typeName,omitempty"`
	MapKey     string `json:"mapKey,omitempty"`
	MapValue   string `json:"mapValue,omitempty"`
	Presence   bool   `json:"presence"`
	Required   bool   `json:"required"`
	HasDefault bool   `json:"hasDefault"`
	Default    string `json:"defaultValue,omitempty"`
	Oneof      string `json:"oneof,omitempty"`
	Repeated   bool   `json:"repeated"`
	Map        bool   `json:"map"`
	Packed     bool   `json:"packed"`
	Deprecated bool   `json:"deprecated"`
}

type oneofProjection struct {
	FullName   string   `json:"fullName"`
	Name       string   `json:"name"`
	FieldNames []string `json:"fieldNames"`
}

type messageProjection struct {
	FullName   string            `json:"fullName"`
	Name       string            `json:"name"`
	Package    string            `json:"package"`
	Parent     string            `json:"parent,omitempty"`
	Source     sourceKind        `json:"source"`
	Fields     []fieldProjection `json:"fields"`
	Oneofs     []oneofProjection `json:"oneofs"`
	Deprecated bool              `json:"deprecated"`
}

type methodProjection struct {
	FullName        string `json:"fullName"`
	Name            string `json:"name"`
	InputType       string `json:"inputType"`
	OutputType      string `json:"outputType"`
	ClientStreaming bool   `json:"clientStreaming"`
	ServerStreaming bool   `json:"serverStreaming"`
	Deprecated      bool   `json:"deprecated"`
}

type serviceProjection struct {
	FullName   string             `json:"fullName"`
	Name       string             `json:"name"`
	Package    string             `json:"package"`
	Source     sourceKind         `json:"source"`
	Methods    []methodProjection `json:"methods"`
	Deprecated bool               `json:"deprecated"`
}

type fileProjection struct {
	Path         string     `json:"path"`
	Package      string     `json:"package"`
	Syntax       string     `json:"syntax"`
	Source       sourceKind `json:"source"`
	Dependencies []string   `json:"dependencies"`
	Services     []string   `json:"services"`
}

type projection struct {
	DescriptorDigest string              `json:"descriptorDigest"`
	Files            []fileProjection    `json:"files"`
	Enums            []enumProjection    `json:"enums"`
	Messages         []messageProjection `json:"messages"`
	Services         []serviceProjection `json:"services"`
}

func buildProjection(set *descriptorpb.FileDescriptorSet) (projection, error) {
	files, messageDescriptors, enumDescriptors, serviceDescriptors, err := indexDescriptors(set)
	if err != nil {
		return projection{}, err
	}
	result := projection{
		Files: projectFiles(files), Enums: projectEnums(enumDescriptors),
		Messages: projectMessages(messageDescriptors), Services: projectServices(serviceDescriptors),
	}
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(set)
	if err != nil {
		return projection{}, fmt.Errorf("marshal normalized descriptors: %w", err)
	}
	digest := sha256.Sum256(encoded)
	result.DescriptorDigest = "sha256:" + hex.EncodeToString(digest[:])
	return result, nil
}

func indexDescriptors(set *descriptorpb.FileDescriptorSet) (
	[]protoreflect.FileDescriptor,
	map[protoreflect.FullName]protoreflect.MessageDescriptor,
	map[protoreflect.FullName]protoreflect.EnumDescriptor,
	map[protoreflect.FullName]protoreflect.ServiceDescriptor,
	error,
) {
	registry, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("resolve descriptor graph: %w", err)
	}
	messageDescriptors := make(map[protoreflect.FullName]protoreflect.MessageDescriptor)
	enumDescriptors := make(map[protoreflect.FullName]protoreflect.EnumDescriptor)
	serviceDescriptors := make(map[protoreflect.FullName]protoreflect.ServiceDescriptor)
	var files []protoreflect.FileDescriptor
	registry.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		files = append(files, file)
		collectMessages(file.Messages(), messageDescriptors, enumDescriptors)
		collectEnums(file.Enums(), enumDescriptors)
		services := file.Services()
		for index := 0; index < services.Len(); index++ {
			service := services.Get(index)
			serviceDescriptors[service.FullName()] = service
		}
		return true
	})
	slices.SortFunc(files, func(left, right protoreflect.FileDescriptor) int {
		return strings.Compare(left.Path(), right.Path())
	})
	return files, messageDescriptors, enumDescriptors, serviceDescriptors, nil
}

func projectFiles(files []protoreflect.FileDescriptor) []fileProjection {
	result := make([]fileProjection, 0, len(files))
	for _, file := range files {
		item := fileProjection{
			Path: file.Path(), Package: string(file.Package()), Syntax: file.Syntax().String(),
			Source: classifySource(file.Path()), Dependencies: []string{}, Services: []string{},
		}
		imports := file.Imports()
		for index := 0; index < imports.Len(); index++ {
			item.Dependencies = append(item.Dependencies, imports.Get(index).Path())
		}
		slices.Sort(item.Dependencies)
		services := file.Services()
		for index := 0; index < services.Len(); index++ {
			item.Services = append(item.Services, string(services.Get(index).FullName()))
		}
		result = append(result, item)
	}
	return result
}

func projectEnums(descriptors map[protoreflect.FullName]protoreflect.EnumDescriptor) []enumProjection {
	result := make([]enumProjection, 0, len(descriptors))
	for _, name := range sortedNames(descriptors) {
		descriptor := descriptors[name]
		if parent, ok := descriptor.Parent().(protoreflect.MessageDescriptor); ok && parent.IsMapEntry() {
			continue
		}
		item := enumProjection{
			FullName: string(name), Name: string(descriptor.Name()), Package: string(descriptor.ParentFile().Package()),
			Parent: descriptorParent(descriptor), Source: classifySource(descriptor.ParentFile().Path()),
			Values: []enumValueProjection{}, AllowAliases: descriptor.Options().(*descriptorpb.EnumOptions).GetAllowAlias(),
			Deprecated: descriptor.Options().(*descriptorpb.EnumOptions).GetDeprecated(),
		}
		values := descriptor.Values()
		for index := 0; index < values.Len(); index++ {
			value := values.Get(index)
			item.Values = append(item.Values, enumValueProjection{
				FullName: string(value.FullName()), Name: string(value.Name()), Number: int32(value.Number()),
				Deprecated: value.Options().(*descriptorpb.EnumValueOptions).GetDeprecated(),
			})
		}
		result = append(result, item)
	}
	return result
}

func projectMessages(descriptors map[protoreflect.FullName]protoreflect.MessageDescriptor) []messageProjection {
	projected := make([]messageProjection, 0, len(descriptors))
	for _, name := range sortedNames(descriptors) {
		descriptor := descriptors[name]
		if descriptor.IsMapEntry() {
			continue
		}
		message := messageProjection{
			FullName: string(name), Name: string(descriptor.Name()), Package: string(descriptor.ParentFile().Package()),
			Parent: descriptorParent(descriptor), Source: classifySource(descriptor.ParentFile().Path()),
			Fields: []fieldProjection{}, Oneofs: []oneofProjection{},
			Deprecated: descriptor.Options().(*descriptorpb.MessageOptions).GetDeprecated(),
		}
		oneofs := make(map[protoreflect.Name]*oneofProjection)
		fields := descriptor.Fields()
		for index := 0; index < fields.Len(); index++ {
			field := projectField(fields.Get(index))
			message.Fields = append(message.Fields, field)
			if containing := fields.Get(index).ContainingOneof(); containing != nil && !containing.IsSynthetic() {
				oneof := oneofs[containing.Name()]
				if oneof == nil {
					oneof = &oneofProjection{
						FullName: string(containing.FullName()), Name: string(containing.Name()), FieldNames: []string{},
					}
					oneofs[containing.Name()] = oneof
				}
				oneof.FieldNames = append(oneof.FieldNames, field.FullName)
			}
		}
		oneofNames := make([]string, 0, len(oneofs))
		for name := range oneofs {
			oneofNames = append(oneofNames, string(name))
		}
		slices.Sort(oneofNames)
		for _, oneofName := range oneofNames {
			message.Oneofs = append(message.Oneofs, *oneofs[protoreflect.Name(oneofName)])
		}
		projected = append(projected, message)
	}
	return projected
}

func projectServices(descriptors map[protoreflect.FullName]protoreflect.ServiceDescriptor) []serviceProjection {
	result := make([]serviceProjection, 0, len(descriptors))
	for _, name := range sortedNames(descriptors) {
		descriptor := descriptors[name]
		service := serviceProjection{
			FullName: string(name), Name: string(descriptor.Name()), Package: string(descriptor.ParentFile().Package()),
			Source:  classifySource(descriptor.ParentFile().Path()),
			Methods: []methodProjection{}, Deprecated: descriptor.Options().(*descriptorpb.ServiceOptions).GetDeprecated(),
		}
		methods := descriptor.Methods()
		for index := 0; index < methods.Len(); index++ {
			method := methods.Get(index)
			service.Methods = append(service.Methods, methodProjection{
				FullName: string(method.FullName()), Name: string(method.Name()),
				InputType: string(method.Input().FullName()), OutputType: string(method.Output().FullName()),
				ClientStreaming: method.IsStreamingClient(), ServerStreaming: method.IsStreamingServer(),
				Deprecated: method.Options().(*descriptorpb.MethodOptions).GetDeprecated(),
			})
		}
		result = append(result, service)
	}
	return result
}

func collectMessages(
	descriptors protoreflect.MessageDescriptors,
	messages map[protoreflect.FullName]protoreflect.MessageDescriptor,
	enums map[protoreflect.FullName]protoreflect.EnumDescriptor,
) {
	for index := 0; index < descriptors.Len(); index++ {
		descriptor := descriptors.Get(index)
		messages[descriptor.FullName()] = descriptor
		collectMessages(descriptor.Messages(), messages, enums)
		collectEnums(descriptor.Enums(), enums)
	}
}

func collectEnums(descriptors protoreflect.EnumDescriptors, enums map[protoreflect.FullName]protoreflect.EnumDescriptor) {
	for index := 0; index < descriptors.Len(); index++ {
		descriptor := descriptors.Get(index)
		enums[descriptor.FullName()] = descriptor
	}
}

func projectField(field protoreflect.FieldDescriptor) fieldProjection {
	result := fieldProjection{
		FullName: string(field.FullName()), Name: string(field.Name()), JSONName: field.JSONName(),
		Number: int32(field.Number()), Kind: field.Kind().String(),
		Presence: field.HasPresence(), Required: field.Cardinality() == protoreflect.Required,
		HasDefault: field.HasDefault(), Repeated: field.Cardinality() == protoreflect.Repeated,
		Map: field.IsMap(), Packed: field.IsPacked(),
		Deprecated: field.Options().(*descriptorpb.FieldOptions).GetDeprecated(),
	}
	if result.HasDefault {
		result.Default = protodesc.ToFieldDescriptorProto(field).GetDefaultValue()
	}
	if oneof := field.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() {
		result.Oneof = string(oneof.Name())
	}
	if field.IsMap() {
		result.Repeated = false
		result.MapKey = descriptorTypeName(field.MapKey())
		result.MapValue = descriptorTypeName(field.MapValue())
		if value := field.MapValue(); value.Message() != nil || value.Enum() != nil {
			result.TypeName = descriptorTypeName(value)
		}
		return result
	}
	if message := field.Message(); message != nil {
		result.TypeName = string(message.FullName())
	} else if enum := field.Enum(); enum != nil {
		result.TypeName = string(enum.FullName())
	}
	return result
}

func descriptorTypeName(field protoreflect.FieldDescriptor) string {
	if message := field.Message(); message != nil {
		return string(message.FullName())
	}
	if enum := field.Enum(); enum != nil {
		return string(enum.FullName())
	}
	return field.Kind().String()
}

func sortedNames[T any](values map[protoreflect.FullName]T) []protoreflect.FullName {
	result := make([]protoreflect.FullName, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
}

func descriptorParent(descriptor protoreflect.Descriptor) string {
	if parent, ok := descriptor.Parent().(protoreflect.MessageDescriptor); ok {
		return string(parent.FullName())
	}
	return ""
}

func classifySource(path string) sourceKind {
	switch {
	case strings.HasPrefix(path, "temporal/api/"):
		return sourcePublic
	case strings.HasPrefix(path, "temporal/server/api/"):
		return sourceInternal
	case strings.HasPrefix(path, "chasm/lib/"):
		return sourceCHASM
	default:
		return sourceExternal
	}
}
