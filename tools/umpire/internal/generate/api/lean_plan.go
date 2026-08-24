package api

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"unicode"
)

type leanName []string

func (n leanName) String() string {
	return strings.Join(n, ".")
}

type leanTypeKind uint8

const (
	leanTypeNamed leanTypeKind = iota
	leanTypeOption
	leanTypeList
	leanTypeProduct
)

type leanType struct {
	Kind      leanTypeKind
	Name      string
	Arguments []leanType
}

type leanFieldPlan struct {
	Projection fieldProjection
	Name       string
	Type       leanType
	BaseType   leanType
	Recursive  bool
}

type leanStructureFieldPlan struct {
	Name string
	Type leanType
}

type leanEnumValuePlan struct {
	Projection enumValueProjection
	Name       string
}

type leanEnumPlan struct {
	Projection   enumProjection
	Name         leanName
	Namespace    leanName
	RelativeName string
	Values       []leanEnumValuePlan
}

type leanOneofConstructorPlan struct {
	Field leanFieldPlan
}

type leanOneofPlan struct {
	Projection   oneofProjection
	Name         leanName
	RelativeName string
	SlotName     string
	Constructors []leanOneofConstructorPlan
}

type leanMessagePlan struct {
	Projection      messageProjection
	Name            leanName
	Namespace       leanName
	RelativeName    string
	Fields          []leanFieldPlan
	StructureFields []leanStructureFieldPlan
	Oneofs          []leanOneofPlan
}

type leanMethodPlan struct {
	Projection methodProjection
	Name       string
	InputType  leanType
	OutputType leanType
}

type leanServicePlan struct {
	Projection serviceProjection
	Name       leanName
	Namespace  leanName
	Methods    []leanMethodPlan
}

type leanModulePlan struct {
	Path    string
	Imports []string
}

type leanNamespacePlan struct {
	Name     leanName
	Enums    []leanEnumPlan
	Messages []leanMessagePlan
}

type leanSourcePlan struct {
	Source        sourceKind
	Name          string
	CatalogModule leanModulePlan
	GRPCModule    leanModulePlan
	Files         []fileProjection
	Enums         []leanEnumPlan
	Messages      []leanMessagePlan
	Services      []leanServicePlan
}

type leanPlan struct {
	TypesModule leanModulePlan
	Namespaces  []leanNamespacePlan
	Enums       []leanEnumPlan
	Messages    []leanMessagePlan
	Sources     []leanSourcePlan
	names       map[string]leanName
	fields      map[string]leanFieldPlan
	oneofs      map[string]leanOneofPlan
	methods     map[string]leanMethodPlan
	services    map[string]leanServicePlan
}

type nameRequest struct {
	identity  string
	base      string
	number    int32
	hasNumber bool
}

type declarationInfo struct {
	identity string
	parent   string
	package_ string
	base     string
}

var sourceModuleSpecs = []struct {
	source sourceKind
	name   string
}{
	{source: sourcePublic, name: "Public"},
	{source: sourceInternal, name: "Internal"},
	{source: sourceCHASM, name: "CHASM"},
	{source: sourceExternal, name: "External"},
}

func buildLeanPlan(projection projection) (leanPlan, error) {
	graph, err := buildMessageGraph(projection.Messages)
	if err != nil {
		return leanPlan{}, err
	}
	packageNames, err := buildLeanPackageNames(projection)
	if err != nil {
		return leanPlan{}, err
	}
	declarationNames, err := buildLeanDeclarationNames(projection, packageNames)
	if err != nil {
		return leanPlan{}, err
	}
	plan := leanPlan{
		TypesModule: leanModulePlan{Path: "Temporal/Generated/Types.lean", Imports: []string{"Temporal.Proto.Core"}},
		names:       declarationNames,
		fields:      make(map[string]leanFieldPlan),
		oneofs:      make(map[string]leanOneofPlan),
		methods:     make(map[string]leanMethodPlan),
		services:    make(map[string]leanServicePlan),
	}
	for _, enum := range projection.Enums {
		planned, planErr := planEnum(enum, packageNames[enum.Package], declarationNames[enum.FullName])
		if planErr != nil {
			return leanPlan{}, planErr
		}
		plan.Enums = append(plan.Enums, planned)
	}
	messageByName := make(map[string]messageProjection, len(projection.Messages))
	for _, message := range projection.Messages {
		messageByName[message.FullName] = message
	}
	for _, fullName := range graph.order {
		message := messageByName[fullName]
		planned, planErr := planMessage(message, packageNames[message.Package], declarationNames, graph)
		if planErr != nil {
			return leanPlan{}, planErr
		}
		plan.Messages = append(plan.Messages, planned)
		for _, field := range planned.Fields {
			plan.fields[field.Projection.FullName] = field
		}
		for _, oneof := range planned.Oneofs {
			plan.oneofs[oneof.Projection.FullName] = oneof
		}
	}
	for _, service := range projection.Services {
		planned, planErr := planService(service, packageNames[service.Package], declarationNames)
		if planErr != nil {
			return leanPlan{}, planErr
		}
		plan.services[service.FullName] = planned
		for _, method := range planned.Methods {
			plan.methods[method.Projection.FullName] = method
		}
	}
	plan.Namespaces = buildLeanNamespacePlans(plan.Enums, plan.Messages)
	plan.Sources = buildLeanSourcePlans(projection, plan)
	if err := validateLeanPlan(projection, plan); err != nil {
		return leanPlan{}, err
	}
	return plan, nil
}

func buildLeanNamespacePlans(enums []leanEnumPlan, messages []leanMessagePlan) []leanNamespacePlan {
	var result []leanNamespacePlan
	appendNamespace := func(namespace leanName) *leanNamespacePlan {
		if len(result) == 0 || !slices.Equal(result[len(result)-1].Name, namespace) {
			result = append(result, leanNamespacePlan{Name: namespace})
		}
		return &result[len(result)-1]
	}
	for _, enum := range enums {
		namespace := appendNamespace(enum.Namespace)
		namespace.Enums = append(namespace.Enums, enum)
	}
	for _, message := range messages {
		namespace := appendNamespace(message.Namespace)
		namespace.Messages = append(namespace.Messages, message)
	}
	return result
}

func buildLeanPackageNames(projection projection) (map[string]leanName, error) {
	packages := make(map[string]bool)
	for _, enum := range projection.Enums {
		packages[enum.Package] = true
	}
	for _, message := range projection.Messages {
		packages[message.Package] = true
	}
	for _, service := range projection.Services {
		packages[service.Package] = true
	}
	requestsByParent := make(map[string][]nameRequest)
	seen := make(map[string]bool)
	for packageName := range packages {
		if packageName == "" {
			return nil, fmt.Errorf("build Lean package names: empty protobuf package")
		}
		parts := strings.Split(packageName, ".")
		for index, part := range parts {
			identity := strings.Join(parts[:index+1], ".")
			if seen[identity] {
				continue
			}
			parent := strings.Join(parts[:index], ".")
			requestsByParent[parent] = append(requestsByParent[parent], nameRequest{
				identity: identity,
				base:     upperIdentifier(part),
			})
			seen[identity] = true
		}
	}
	segments := make(map[string]string, len(seen))
	parents := make([]string, 0, len(requestsByParent))
	for parent := range requestsByParent {
		parents = append(parents, parent)
	}
	slices.Sort(parents)
	for _, parent := range parents {
		allocated, err := allocateNames(requestsByParent[parent], nil)
		if err != nil {
			return nil, fmt.Errorf("build Lean package namespace %q: %w", parent, err)
		}
		for identity, name := range allocated {
			segments[identity] = name
		}
	}
	result := make(map[string]leanName, len(packages))
	for packageName := range packages {
		parts := strings.Split(packageName, ".")
		name := make(leanName, 0, len(parts))
		for index := range parts {
			name = append(name, segments[strings.Join(parts[:index+1], ".")])
		}
		result[packageName] = name
	}
	return result, nil
}

func buildLeanDeclarationNames(projection projection, packageNames map[string]leanName) (map[string]leanName, error) {
	var declarations []declarationInfo
	for _, enum := range projection.Enums {
		declarations = append(declarations, declarationInfo{
			identity: enum.FullName, parent: enum.Parent, package_: enum.Package, base: upperIdentifier(enum.Name),
		})
	}
	for _, message := range projection.Messages {
		declarations = append(declarations, declarationInfo{
			identity: message.FullName, parent: message.Parent, package_: message.Package, base: upperIdentifier(message.Name),
		})
		for _, oneof := range message.Oneofs {
			declarations = append(declarations, declarationInfo{
				identity: oneof.FullName, parent: message.FullName, package_: message.Package, base: upperIdentifier(oneof.Name),
			})
		}
	}
	for _, service := range projection.Services {
		declarations = append(declarations, declarationInfo{
			identity: service.FullName, package_: service.Package, base: upperIdentifier(service.Name),
		})
	}
	requestsByScope := make(map[string][]nameRequest)
	seen := make(map[string]bool, len(declarations))
	for _, declaration := range declarations {
		if declaration.identity == "" || declaration.base == "" {
			return nil, fmt.Errorf("build Lean declaration names: incomplete declaration %+v", declaration)
		}
		if seen[declaration.identity] {
			return nil, fmt.Errorf("build Lean declaration names: duplicate protobuf identity %q", declaration.identity)
		}
		seen[declaration.identity] = true
		scope := "package:" + declaration.package_
		if declaration.parent != "" {
			scope = "declaration:" + declaration.parent
		}
		requestsByScope[scope] = append(requestsByScope[scope], nameRequest{
			identity: declaration.identity,
			base:     declaration.base,
		})
	}
	localNames := make(map[string]string, len(declarations))
	scopes := make([]string, 0, len(requestsByScope))
	for scope := range requestsByScope {
		scopes = append(scopes, scope)
	}
	slices.Sort(scopes)
	for _, scope := range scopes {
		allocated, err := allocateNames(requestsByScope[scope], nil)
		if err != nil {
			return nil, fmt.Errorf("build Lean declaration scope %q: %w", scope, err)
		}
		for identity, name := range allocated {
			localNames[identity] = name
		}
	}

	result := make(map[string]leanName, len(declarations))
	remaining := slices.Clone(declarations)
	for len(remaining) != 0 {
		next := remaining[:0]
		progress := false
		for _, declaration := range remaining {
			if declaration.parent == "" {
				packageName, exists := packageNames[declaration.package_]
				if !exists {
					return nil, fmt.Errorf("build Lean declaration %q: unknown package %q", declaration.identity, declaration.package_)
				}
				result[declaration.identity] = appendLeanName(packageName, localNames[declaration.identity])
				progress = true
				continue
			}
			parentName, exists := result[declaration.parent]
			if !exists {
				next = append(next, declaration)
				continue
			}
			result[declaration.identity] = appendLeanName(parentName, localNames[declaration.identity])
			progress = true
		}
		if !progress {
			return nil, fmt.Errorf("build Lean declarations: unresolved parent for %q", next[0].identity)
		}
		remaining = next
	}
	return result, nil
}

func planEnum(projection enumProjection, packageName, name leanName) (leanEnumPlan, error) {
	relativeName, err := relativeLeanName(name, packageName)
	if err != nil {
		return leanEnumPlan{}, fmt.Errorf("plan enum %q: %w", projection.FullName, err)
	}
	requests := make([]nameRequest, 0, len(projection.Values))
	for _, value := range projection.Values {
		requests = append(requests, nameRequest{
			identity: value.FullName,
			base:     lowerIdentifier(value.Name), number: value.Number, hasNumber: true,
		})
	}
	allocated, err := allocateNames(requests, nil)
	if err != nil {
		return leanEnumPlan{}, fmt.Errorf("plan enum %q values: %w", projection.FullName, err)
	}
	result := leanEnumPlan{Projection: projection, Name: name, Namespace: packageName, RelativeName: relativeName}
	for _, value := range projection.Values {
		result.Values = append(result.Values, leanEnumValuePlan{
			Projection: value,
			Name:       allocated[value.FullName],
		})
	}
	return result, nil
}

func planMessage(
	projection messageProjection,
	packageName leanName,
	declarationNames map[string]leanName,
	graph messageGraph,
) (leanMessagePlan, error) {
	name := declarationNames[projection.FullName]
	relativeName, err := relativeLeanName(name, packageName)
	if err != nil {
		return leanMessagePlan{}, fmt.Errorf("plan message %q: %w", projection.FullName, err)
	}
	fieldByName := make(map[string]fieldProjection, len(projection.Fields))
	requests := make([]nameRequest, 0, len(projection.Fields)+len(projection.Oneofs))
	for _, field := range projection.Fields {
		if fieldByName[field.FullName].FullName != "" {
			return leanMessagePlan{}, fmt.Errorf("plan message %q: duplicate field %q", projection.FullName, field.FullName)
		}
		fieldByName[field.FullName] = field
		if field.Oneof == "" {
			requests = append(requests, nameRequest{
				identity: field.FullName, base: lowerIdentifier(field.Name), number: field.Number, hasNumber: true,
			})
		}
	}
	for _, oneof := range projection.Oneofs {
		requests = append(requests, nameRequest{identity: oneof.FullName, base: lowerIdentifier(oneof.Name)})
	}
	structureNames, err := allocateNames(requests, nil)
	if err != nil {
		return leanMessagePlan{}, fmt.Errorf("plan message %q fields: %w", projection.FullName, err)
	}
	constructorNames := make(map[string]string)
	for _, oneof := range projection.Oneofs {
		constructors := make([]nameRequest, 0, len(oneof.FieldNames))
		for _, fieldName := range oneof.FieldNames {
			field, exists := fieldByName[fieldName]
			if !exists {
				return leanMessagePlan{}, fmt.Errorf("plan oneof %q: unknown field %q", oneof.FullName, fieldName)
			}
			if field.Oneof != oneof.Name {
				return leanMessagePlan{}, fmt.Errorf("plan oneof %q: field %q belongs to %q", oneof.FullName, fieldName, field.Oneof)
			}
			constructors = append(constructors, nameRequest{
				identity: field.FullName, base: lowerIdentifier(field.Name), number: field.Number, hasNumber: true,
			})
		}
		allocated, allocateErr := allocateNames(constructors, []string{"notSet"})
		if allocateErr != nil {
			return leanMessagePlan{}, fmt.Errorf("plan oneof %q constructors: %w", oneof.FullName, allocateErr)
		}
		for fieldName, constructorName := range allocated {
			constructorNames[fieldName] = constructorName
		}
	}

	result := leanMessagePlan{Projection: projection, Name: name, Namespace: packageName, RelativeName: relativeName}
	fields := make(map[string]leanFieldPlan, len(projection.Fields))
	for _, field := range projection.Fields {
		baseType, recursive, typeErr := leanFieldBaseType(projection, field, packageName, declarationNames, graph)
		if typeErr != nil {
			return leanMessagePlan{}, typeErr
		}
		fieldType := wrapLeanFieldType(field, baseType)
		fieldName := structureNames[field.FullName]
		if field.Oneof != "" {
			fieldName = constructorNames[field.FullName]
		}
		planned := leanFieldPlan{
			Projection: field, Name: fieldName, Type: fieldType, BaseType: baseType, Recursive: recursive,
		}
		fields[field.FullName] = planned
		result.Fields = append(result.Fields, planned)
		if field.Oneof == "" {
			result.StructureFields = append(result.StructureFields, leanStructureFieldPlan{Name: fieldName, Type: fieldType})
		}
	}
	for _, oneof := range projection.Oneofs {
		oneofName := declarationNames[oneof.FullName]
		oneofRelativeName, relativeErr := relativeLeanName(oneofName, packageName)
		if relativeErr != nil {
			return leanMessagePlan{}, fmt.Errorf("plan oneof %q: %w", oneof.FullName, relativeErr)
		}
		planned := leanOneofPlan{
			Projection: oneof, Name: oneofName, RelativeName: oneofRelativeName,
			SlotName: structureNames[oneof.FullName],
		}
		for _, fieldName := range oneof.FieldNames {
			planned.Constructors = append(planned.Constructors, leanOneofConstructorPlan{Field: fields[fieldName]})
		}
		result.Oneofs = append(result.Oneofs, planned)
		result.StructureFields = append(result.StructureFields, leanStructureFieldPlan{
			Name: planned.SlotName,
			Type: namedLeanType(oneofRelativeName),
		})
	}
	return result, nil
}

func planService(
	projection serviceProjection,
	packageName leanName,
	declarationNames map[string]leanName,
) (leanServicePlan, error) {
	name := declarationNames[projection.FullName]
	requests := make([]nameRequest, 0, len(projection.Methods))
	for _, method := range projection.Methods {
		requests = append(requests, nameRequest{identity: method.FullName, base: lowerIdentifier(method.Name)})
	}
	methodNames, err := allocateNames(requests, nil)
	if err != nil {
		return leanServicePlan{}, fmt.Errorf("plan service %q methods: %w", projection.FullName, err)
	}
	result := leanServicePlan{Projection: projection, Name: name, Namespace: packageName}
	for _, method := range projection.Methods {
		inputType, typeErr := leanNamedReference(method.InputType, projection.Package, packageName, declarationNames)
		if typeErr != nil {
			return leanServicePlan{}, fmt.Errorf("plan method %q input: %w", method.FullName, typeErr)
		}
		outputType, typeErr := leanNamedReference(method.OutputType, projection.Package, packageName, declarationNames)
		if typeErr != nil {
			return leanServicePlan{}, fmt.Errorf("plan method %q output: %w", method.FullName, typeErr)
		}
		result.Methods = append(result.Methods, leanMethodPlan{
			Projection: method, Name: methodNames[method.FullName], InputType: inputType, OutputType: outputType,
		})
	}
	return result, nil
}

func leanFieldBaseType(
	message messageProjection,
	field fieldProjection,
	packageName leanName,
	declarationNames map[string]leanName,
	graph messageGraph,
) (leanType, bool, error) {
	if field.Map {
		key, err := leanScalarType(field.MapKey)
		if err != nil {
			return leanType{}, false, fmt.Errorf("plan field %q map key: %w", field.FullName, err)
		}
		var value leanType
		recursive := false
		if field.TypeName != "" {
			value, err = leanNamedReference(field.TypeName, message.Package, packageName, declarationNames)
			recursive = graph.recursive(message.FullName, field.TypeName)
			if recursive {
				value = namedLeanType("Temporal.Proto.MessageRef")
			}
		} else {
			value, err = leanScalarType(field.MapValue)
		}
		if err != nil {
			return leanType{}, false, fmt.Errorf("plan field %q map value: %w", field.FullName, err)
		}
		return leanType{Kind: leanTypeList, Arguments: []leanType{{
			Kind: leanTypeProduct, Arguments: []leanType{key, value},
		}}}, recursive, nil
	}
	if field.TypeName != "" {
		result, err := leanNamedReference(field.TypeName, message.Package, packageName, declarationNames)
		if err != nil {
			return leanType{}, false, fmt.Errorf("plan field %q: %w", field.FullName, err)
		}
		recursive := graph.recursive(message.FullName, field.TypeName)
		if recursive {
			result = namedLeanType("Temporal.Proto.MessageRef")
		}
		return result, recursive, nil
	}
	result, err := leanScalarType(field.Kind)
	if err != nil {
		return leanType{}, false, fmt.Errorf("plan field %q: %w", field.FullName, err)
	}
	return result, false, nil
}

func wrapLeanFieldType(field fieldProjection, base leanType) leanType {
	if field.Map {
		return base
	}
	if field.Repeated {
		return leanType{Kind: leanTypeList, Arguments: []leanType{base}}
	}
	if field.Presence && !field.Required {
		return leanType{Kind: leanTypeOption, Arguments: []leanType{base}}
	}
	return base
}

func leanNamedReference(
	fullName string,
	currentPackage string,
	packageName leanName,
	declarationNames map[string]leanName,
) (leanType, error) {
	name, exists := declarationNames[fullName]
	if !exists {
		return leanType{}, fmt.Errorf("unknown protobuf type %q", fullName)
	}
	if fullName == currentPackage || strings.HasPrefix(fullName, currentPackage+".") {
		relative, err := relativeLeanName(name, packageName)
		if err != nil {
			return leanType{}, err
		}
		return namedLeanType(relative), nil
	}
	return namedLeanType(name.String()), nil
}

func leanScalarType(kind string) (leanType, error) {
	switch kind {
	case "bool":
		return namedLeanType("Bool"), nil
	case "int32", "sint32", "sfixed32", "int64", "sint64", "sfixed64":
		return namedLeanType("Int"), nil
	case "uint32", "fixed32", "uint64", "fixed64":
		return namedLeanType("Nat"), nil
	case "float", "double":
		return namedLeanType("Float"), nil
	case "string":
		return namedLeanType("String"), nil
	case "bytes":
		return namedLeanType("Temporal.Proto.Bytes"), nil
	default:
		return leanType{}, fmt.Errorf("unsupported protobuf kind %q", kind)
	}
}

func namedLeanType(name string) leanType {
	return leanType{Kind: leanTypeNamed, Name: name}
}

func buildLeanSourcePlans(projection projection, plan leanPlan) []leanSourcePlan {
	result := make([]leanSourcePlan, 0, len(sourceModuleSpecs))
	for _, spec := range sourceModuleSpecs {
		source := leanSourcePlan{
			Source: spec.source,
			Name:   spec.name,
			CatalogModule: leanModulePlan{
				Path:    "Temporal/Generated/Catalog/" + spec.name + ".lean",
				Imports: []string{"Temporal.Proto.Core"},
			},
			GRPCModule: leanModulePlan{
				Path:    "Temporal/Generated/GRPC/" + spec.name + ".lean",
				Imports: []string{"Temporal.Proto.Core", "Temporal.Generated.Types"},
			},
		}
		for _, file := range projection.Files {
			if file.Source == spec.source {
				source.Files = append(source.Files, file)
			}
		}
		for _, enum := range plan.Enums {
			if enum.Projection.Source == spec.source {
				source.Enums = append(source.Enums, enum)
			}
		}
		for _, message := range plan.Messages {
			if message.Projection.Source == spec.source {
				source.Messages = append(source.Messages, message)
			}
		}
		for _, service := range projection.Services {
			if service.Source == spec.source {
				source.Services = append(source.Services, plan.services[service.FullName])
			}
		}
		result = append(result, source)
	}
	return result
}

func validateLeanPlan(projection projection, plan leanPlan) error {
	if len(plan.Enums) != len(projection.Enums) || len(plan.Messages) != len(projection.Messages) {
		return fmt.Errorf("validate Lean plan: declaration count mismatch")
	}
	seenNames := make(map[string]string, len(plan.names))
	for fullName, name := range plan.names {
		if name.String() == "" {
			return fmt.Errorf("validate Lean plan: %q has empty Lean name", fullName)
		}
		if previous, duplicate := seenNames[name.String()]; duplicate {
			return fmt.Errorf("validate Lean plan: %q and %q share Lean name %q", previous, fullName, name.String())
		}
		seenNames[name.String()] = fullName
	}
	messageOrder := make(map[string]int, len(plan.Messages))
	for index, message := range plan.Messages {
		messageOrder[message.Projection.FullName] = index
	}
	for index, message := range plan.Messages {
		seenFields := make(map[string]bool, len(message.StructureFields))
		for _, field := range message.StructureFields {
			if field.Name == "" || seenFields[field.Name] {
				return fmt.Errorf("validate Lean plan: message %q has invalid field %q", message.Projection.FullName, field.Name)
			}
			seenFields[field.Name] = true
			if err := validateLeanType(field.Type); err != nil {
				return fmt.Errorf("validate Lean plan: message %q field %q: %w", message.Projection.FullName, field.Name, err)
			}
		}
		for _, field := range message.Fields {
			if dependency, known := messageOrder[field.Projection.TypeName]; known && !field.Recursive && dependency >= index {
				return fmt.Errorf("validate Lean plan: message %q precedes dependency %q", message.Projection.FullName, field.Projection.TypeName)
			}
		}
	}
	files := 0
	enums := 0
	messages := 0
	services := 0
	for _, source := range plan.Sources {
		files += len(source.Files)
		enums += len(source.Enums)
		messages += len(source.Messages)
		services += len(source.Services)
		if source.CatalogModule.Path == "" || source.GRPCModule.Path == "" || len(source.GRPCModule.Imports) != 2 {
			return fmt.Errorf("validate Lean plan: source %q has incomplete modules", source.Source)
		}
	}
	if files != len(projection.Files) || enums != len(projection.Enums) || messages != len(projection.Messages) || services != len(projection.Services) {
		return fmt.Errorf("validate Lean plan: source partition count mismatch")
	}
	return nil
}

func validateLeanType(value leanType) error {
	switch value.Kind {
	case leanTypeNamed:
		if value.Name == "" || len(value.Arguments) != 0 {
			return fmt.Errorf("invalid named type")
		}
	case leanTypeOption, leanTypeList:
		if len(value.Arguments) != 1 {
			return fmt.Errorf("type constructor requires one argument")
		}
	case leanTypeProduct:
		if len(value.Arguments) != 2 {
			return fmt.Errorf("product type requires two arguments")
		}
	default:
		return fmt.Errorf("unknown type constructor %d", value.Kind)
	}
	for _, argument := range value.Arguments {
		if err := validateLeanType(argument); err != nil {
			return err
		}
	}
	return nil
}

func allocateNames(requests []nameRequest, reserved []string) (map[string]string, error) {
	result := make(map[string]string, len(requests))
	if len(requests) == 0 {
		return result, nil
	}
	groups := make(map[string][]nameRequest)
	original := make(map[string]bool, len(requests)+len(reserved))
	identities := make(map[string]bool, len(requests))
	for _, name := range reserved {
		original[name] = true
	}
	for _, request := range requests {
		if request.identity == "" || request.base == "" {
			return nil, fmt.Errorf("name identity and base are required")
		}
		if identities[request.identity] {
			return nil, fmt.Errorf("duplicate name identity %q", request.identity)
		}
		identities[request.identity] = true
		groups[request.base] = append(groups[request.base], request)
		original[request.base] = true
	}
	used := make(map[string]bool, len(original)+len(requests))
	for _, name := range reserved {
		used[name] = true
	}
	var unresolved []nameRequest
	bases := make([]string, 0, len(groups))
	for base := range groups {
		bases = append(bases, base)
	}
	slices.Sort(bases)
	for _, base := range bases {
		group := groups[base]
		slices.SortFunc(group, func(left, right nameRequest) int {
			return compareStrings(left.identity, right.identity)
		})
		if len(group) == 1 && !used[base] {
			result[group[0].identity] = base
			used[base] = true
			continue
		}
		unresolved = append(unresolved, group...)
	}
	proposals := make(map[string][]nameRequest)
	for _, request := range unresolved {
		if !request.hasNumber {
			continue
		}
		candidate := request.base + leanNumberSuffix(request.number)
		if !original[candidate] && !used[candidate] {
			proposals[candidate] = append(proposals[candidate], request)
		}
	}
	var remaining []nameRequest
	for _, request := range unresolved {
		candidate := ""
		if request.hasNumber {
			candidate = request.base + leanNumberSuffix(request.number)
		}
		if candidate != "" && len(proposals[candidate]) == 1 {
			result[request.identity] = candidate
			used[candidate] = true
			continue
		}
		remaining = append(remaining, request)
	}
	slices.SortFunc(remaining, func(left, right nameRequest) int {
		return compareStrings(left.identity, right.identity)
	})
	for _, request := range remaining {
		digest := sha256.Sum256([]byte(request.identity))
		hexDigest := hex.EncodeToString(digest[:])
		allocated := false
		for length := 8; length <= len(hexDigest); length += 2 {
			candidate := request.base + "_" + hexDigest[:length]
			if !original[candidate] && !used[candidate] {
				result[request.identity] = candidate
				used[candidate] = true
				allocated = true
				break
			}
		}
		if !allocated {
			return nil, fmt.Errorf("cannot disambiguate name %q", request.identity)
		}
	}
	return result, nil
}

func leanNumberSuffix(value int32) string {
	if value < 0 {
		return fmt.Sprintf("Neg%d", -int64(value))
	}
	return fmt.Sprint(value)
}

func relativeLeanName(name, namespace leanName) (string, error) {
	if len(name) <= len(namespace) || !slices.Equal(name[:len(namespace)], namespace) {
		return "", fmt.Errorf("Lean name %q is outside namespace %q", name.String(), namespace.String())
	}
	return leanName(name[len(namespace):]).String(), nil
}

func appendLeanName(name leanName, part string) leanName {
	result := slices.Clone(name)
	return append(result, part)
}

func upperIdentifier(value string) string {
	parts := identifierParts(value)
	var result strings.Builder
	for _, part := range parts {
		if part == strings.ToUpper(part) {
			part = strings.ToLower(part)
		}
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
