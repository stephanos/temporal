package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode"

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
	Name   string `json:"name"`
	Number int32  `json:"number"`
}

type enumProjection struct {
	FullName      string                `json:"fullName"`
	LeanName      string                `json:"leanName"`
	Source        sourceKind            `json:"source"`
	Values        []enumValueProjection `json:"values"`
	AllowAliases  bool                  `json:"allowAliases"`
	IsPlaceholder bool                  `json:"isPlaceholder"`
}

type fieldProjection struct {
	FullName   string `json:"fullName"`
	Name       string `json:"name"`
	JSONName   string `json:"jsonName"`
	LeanName   string `json:"leanName"`
	Number     int32  `json:"number"`
	Kind       string `json:"kind"`
	TypeName   string `json:"typeName,omitempty"`
	MapKey     string `json:"mapKey,omitempty"`
	MapValue   string `json:"mapValue,omitempty"`
	Presence   bool   `json:"presence"`
	Oneof      string `json:"oneof,omitempty"`
	Repeated   bool   `json:"repeated"`
	Map        bool   `json:"map"`
	Recursive  bool   `json:"recursive"`
	Deprecated bool   `json:"deprecated"`
}

type oneofProjection struct {
	Name     string            `json:"name"`
	LeanName string            `json:"leanName"`
	Fields   []fieldProjection `json:"fields"`
}

type messageProjection struct {
	FullName string            `json:"fullName"`
	LeanName string            `json:"leanName"`
	Source   sourceKind        `json:"source"`
	Fields   []fieldProjection `json:"fields"`
	Oneofs   []oneofProjection `json:"oneofs"`
	MapEntry bool              `json:"mapEntry"`
}

type methodProjection struct {
	FullName        string `json:"fullName"`
	Name            string `json:"name"`
	LeanName        string `json:"leanName"`
	InputType       string `json:"inputType"`
	InputLeanType   string `json:"inputLeanType"`
	OutputType      string `json:"outputType"`
	OutputLeanType  string `json:"outputLeanType"`
	ClientStreaming bool   `json:"clientStreaming"`
	ServerStreaming bool   `json:"serverStreaming"`
	Deprecated      bool   `json:"deprecated"`
}

type serviceProjection struct {
	FullName string             `json:"fullName"`
	LeanName string             `json:"leanName"`
	Source   sourceKind         `json:"source"`
	Methods  []methodProjection `json:"methods"`
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
			Source: classifySource(file.Path()),
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
	var result []enumProjection
	for _, name := range sortedNames(descriptors) {
		descriptor := descriptors[name]
		if parent, ok := descriptor.Parent().(protoreflect.MessageDescriptor); ok && parent.IsMapEntry() {
			continue
		}
		item := enumProjection{
			FullName: string(name), LeanName: leanTypeName(name), Source: classifySource(descriptor.ParentFile().Path()),
			AllowAliases: descriptor.Options().(*descriptorpb.EnumOptions).GetAllowAlias(),
		}
		values := descriptor.Values()
		for index := 0; index < values.Len(); index++ {
			value := values.Get(index)
			item.Values = append(item.Values, enumValueProjection{Name: string(value.Name()), Number: int32(value.Number())})
		}
		result = append(result, item)
	}
	return result
}

func projectMessages(descriptors map[protoreflect.FullName]protoreflect.MessageDescriptor) []messageProjection {
	projected := make(map[protoreflect.FullName]messageProjection, len(descriptors))
	for _, name := range sortedNames(descriptors) {
		descriptor := descriptors[name]
		if descriptor.IsMapEntry() {
			continue
		}
		message := messageProjection{
			FullName: string(name), LeanName: leanTypeName(name), Source: classifySource(descriptor.ParentFile().Path()),
		}
		oneofs := make(map[protoreflect.Name]*oneofProjection)
		fields := descriptor.Fields()
		usedFieldNames := make(map[string]int)
		for index := 0; index < fields.Len(); index++ {
			field := projectField(fields.Get(index), descriptor, descriptors)
			field.LeanName = uniqueName(field.LeanName, usedFieldNames, field.Number)
			message.Fields = append(message.Fields, field)
			if containing := fields.Get(index).ContainingOneof(); containing != nil && !containing.IsSynthetic() {
				oneof := oneofs[containing.Name()]
				if oneof == nil {
					oneof = &oneofProjection{Name: string(containing.Name()), LeanName: leanTypeName(name) + "_" + upperIdentifier(string(containing.Name()))}
					oneofs[containing.Name()] = oneof
				}
				oneof.Fields = append(oneof.Fields, field)
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
		projected[name] = message
	}
	return dependencyOrder(projected)
}

func projectServices(descriptors map[protoreflect.FullName]protoreflect.ServiceDescriptor) []serviceProjection {
	var result []serviceProjection
	for _, name := range sortedNames(descriptors) {
		descriptor := descriptors[name]
		service := serviceProjection{
			FullName: string(name), LeanName: leanTypeName(name), Source: classifySource(descriptor.ParentFile().Path()),
		}
		methods := descriptor.Methods()
		usedMethodNames := make(map[string]int)
		for index := 0; index < methods.Len(); index++ {
			method := methods.Get(index)
			leanName := uniqueName(lowerIdentifier(string(method.Name())), usedMethodNames, int32(index+1))
			service.Methods = append(service.Methods, methodProjection{
				FullName: string(method.FullName()), Name: string(method.Name()), LeanName: leanName,
				InputType: string(method.Input().FullName()), InputLeanType: leanTypeName(method.Input().FullName()),
				OutputType: string(method.Output().FullName()), OutputLeanType: leanTypeName(method.Output().FullName()),
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

func projectField(
	field protoreflect.FieldDescriptor,
	containing protoreflect.MessageDescriptor,
	messages map[protoreflect.FullName]protoreflect.MessageDescriptor,
) fieldProjection {
	result := fieldProjection{
		FullName: string(field.FullName()), Name: string(field.Name()), JSONName: field.JSONName(),
		LeanName: lowerIdentifier(string(field.Name())), Number: int32(field.Number()), Kind: field.Kind().String(),
		Presence: field.HasPresence(), Repeated: field.Cardinality() == protoreflect.Repeated, Map: field.IsMap(),
		Deprecated: field.Options().(*descriptorpb.FieldOptions).GetDeprecated(),
	}
	if oneof := field.ContainingOneof(); oneof != nil && !oneof.IsSynthetic() {
		result.Oneof = string(oneof.Name())
	}
	if field.IsMap() {
		result.Repeated = false
		result.MapKey = descriptorTypeName(field.MapKey())
		result.MapValue = descriptorTypeName(field.MapValue())
		if value := field.MapValue().Message(); value != nil {
			result.TypeName = string(value.FullName())
			result.Recursive = reaches(value.FullName(), containing.FullName(), messages, make(map[protoreflect.FullName]bool))
		}
		return result
	}
	if message := field.Message(); message != nil {
		result.TypeName = string(message.FullName())
		result.Recursive = reaches(message.FullName(), containing.FullName(), messages, make(map[protoreflect.FullName]bool))
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

func reaches(
	from protoreflect.FullName,
	target protoreflect.FullName,
	messages map[protoreflect.FullName]protoreflect.MessageDescriptor,
	visited map[protoreflect.FullName]bool,
) bool {
	if from == target {
		return true
	}
	if visited[from] {
		return false
	}
	visited[from] = true
	descriptor := messages[from]
	if descriptor == nil {
		return false
	}
	fields := descriptor.Fields()
	for index := 0; index < fields.Len(); index++ {
		field := fields.Get(index)
		var next protoreflect.MessageDescriptor
		if field.IsMap() {
			next = field.MapValue().Message()
		} else {
			next = field.Message()
		}
		if next != nil && reaches(next.FullName(), target, messages, visited) {
			return true
		}
	}
	return false
}

func dependencyOrder(messages map[protoreflect.FullName]messageProjection) []messageProjection {
	visited := make(map[protoreflect.FullName]bool, len(messages))
	result := make([]messageProjection, 0, len(messages))
	var visit func(protoreflect.FullName)
	visit = func(name protoreflect.FullName) {
		if visited[name] {
			return
		}
		visited[name] = true
		message := messages[name]
		var dependencies []protoreflect.FullName
		for _, field := range message.Fields {
			if field.TypeName == "" || field.Recursive {
				continue
			}
			dependency := protoreflect.FullName(field.TypeName)
			if _, ok := messages[dependency]; ok {
				dependencies = append(dependencies, dependency)
			}
		}
		slices.Sort(dependencies)
		for _, dependency := range dependencies {
			visit(dependency)
		}
		result = append(result, message)
	}
	for _, name := range sortedNames(messages) {
		visit(name)
	}
	return result
}

func sortedNames[T any](values map[protoreflect.FullName]T) []protoreflect.FullName {
	result := make([]protoreflect.FullName, 0, len(values))
	for name := range values {
		result = append(result, name)
	}
	slices.Sort(result)
	return result
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

func leanTypeName(name protoreflect.FullName) string {
	parts := strings.Split(string(name), ".")
	for index, part := range parts {
		parts[index] = upperIdentifier(part)
	}
	return strings.Join(parts, "_")
}

func upperIdentifier(value string) string {
	parts := identifierParts(value)
	var result strings.Builder
	for _, part := range parts {
		characters := []rune(part)
		if len(characters) == 0 {
			continue
		}
		result.WriteRune(unicode.ToUpper(characters[0]))
		result.WriteString(string(characters[1:]))
	}
	if result.Len() == 0 {
		return "Unnamed"
	}
	return result.String()
}

func lowerIdentifier(value string) string {
	name := upperIdentifier(value)
	characters := []rune(name)
	characters[0] = unicode.ToLower(characters[0])
	name = string(characters)
	if leanReserved[name] {
		return name + "Value"
	}
	return name
}

func identifierParts(value string) []string {
	return strings.FieldsFunc(value, func(character rune) bool {
		return !unicode.IsLetter(character) && !unicode.IsNumber(character)
	})
}

func uniqueName(name string, used map[string]int, discriminator int32) string {
	if used[name] == 0 {
		used[name] = 1
		return name
	}
	used[name]++
	return fmt.Sprintf("%s%d", name, discriminator)
}

var leanReserved = map[string]bool{
	"abbrev": true, "attribute": true, "axiom": true, "by": true, "class": true, "def": true,
	"deriving": true, "do": true, "elab": true, "else": true, "end": true, "example": true,
	"export": true, "extends": true, "for": true, "fun": true, "if": true, "import": true,
	"in": true, "include": true, "inductive": true, "infix": true, "infixl": true, "infixr": true,
	"instance": true, "let": true, "macro": true, "match": true, "meta": true, "mutual": true,
	"namespace": true, "omit": true, "opaque": true, "open": true, "partial": true, "postfix": true,
	"prefix": true, "private": true, "protected": true, "scoped": true, "structure": true,
	"syntax": true, "theorem": true, "universe": true, "variable": true, "where": true, "with": true,
}
