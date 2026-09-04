package main

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
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
	Projection    fieldProjection
	Name          string
	QualifiedName leanName
	Type          leanType
	BaseType      leanType
	Recursive     bool
}

type leanStructureFieldPlan struct {
	Name string
	Type leanType
}

type leanEnumValuePlan struct {
	Projection    enumValueProjection
	Name          string
	QualifiedName leanName
	Number        int32
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
	Projection      methodProjection
	Name            string
	QualifiedName   leanName
	InputType       leanType
	OutputType      leanType
	FullName        string
	ClientStreaming bool
	ServerStreaming bool
	Deprecated      bool
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

type leanPlan struct {
	ProtoModule      leanModulePlan
	TypesModule      leanModulePlan
	APIModule        leanModulePlan
	Namespaces       []leanNamespacePlan
	Enums            []leanEnumPlan
	Messages         []leanMessagePlan
	Services         []leanServicePlan
	supportNamespace string
	names            map[string]leanName
	fields           map[string]leanFieldPlan
	oneofs           map[string]leanOneofPlan
}

type declarationInfo struct {
	identity    string
	parent      string
	packageName string
	base        string
}

func protoModuleSpec(layout outputLayout) leanModulePlan {
	return leanModulePlan{Path: layout.ProtoPath, Imports: []string{}}
}

func typesModuleSpec(layout outputLayout) leanModulePlan {
	return leanModulePlan{Path: layout.TypesPath, Imports: []string{layout.RootModule + ".API.Proto"}}
}

func apiModuleSpec(layout outputLayout) leanModulePlan {
	return leanModulePlan{
		Path:    layout.APIPath,
		Imports: []string{layout.RootModule + ".API.Proto", layout.RootModule + ".API.Types"},
	}
}

func cloneLeanModulePlan(module leanModulePlan) leanModulePlan {
	return leanModulePlan{Path: module.Path, Imports: slices.Clone(module.Imports)}
}

func equalLeanModulePlan(left, right leanModulePlan) bool {
	return left.Path == right.Path && slices.Equal(left.Imports, right.Imports)
}

func buildLeanPlan(projection projection, configuration generationConfig) (leanPlan, error) {
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
	if err := validateGeneratedDeclarationCollisions(declarationNames, configuration); err != nil {
		return leanPlan{}, err
	}
	declarationPackages := buildLeanDeclarationPackages(projection)
	plan := leanPlan{
		ProtoModule:      cloneLeanModulePlan(protoModuleSpec(configuration.Layout)),
		TypesModule:      cloneLeanModulePlan(typesModuleSpec(configuration.Layout)),
		APIModule:        cloneLeanModulePlan(apiModuleSpec(configuration.Layout)),
		supportNamespace: configuration.Layout.RootModule + ".API.Proto",
		names:            declarationNames,
		fields:           make(map[string]leanFieldPlan),
		oneofs:           make(map[string]leanOneofPlan),
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
		planned, planErr := planMessage(
			message,
			packageNames[message.Package],
			declarationNames,
			declarationPackages,
			graph,
			configuration.Layout.RootModule+".API.Proto",
		)
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
		planned, planErr := planService(service, packageNames[service.Package], declarationNames, declarationPackages)
		if planErr != nil {
			return leanPlan{}, planErr
		}
		plan.Services = append(plan.Services, planned)
	}
	plan.Namespaces = buildLeanNamespacePlans(plan.Enums, plan.Messages)
	if err := validateLeanPlan(projection, plan, configuration); err != nil {
		return leanPlan{}, err
	}
	return plan, nil
}

func validateGeneratedDeclarationCollisions(
	declarationNames map[string]leanName,
	configuration generationConfig,
) error {
	supportNamespace := configuration.Layout.RootModule + ".API.Proto."
	reserved := make(map[string]string)
	for _, name := range []string{"Bytes", "MessageRef", "Method"} {
		reserved[supportNamespace+name] = "support"
	}
	for identity, name := range declarationNames {
		kind, collision := reserved[name.String()]
		if collision {
			return fmt.Errorf(
				"protobuf declaration %q collides with generated %s declaration %q",
				identity, kind, name.String(),
			)
		}
	}
	return nil
}

func buildLeanDeclarationPackages(projection projection) map[string]string {
	result := make(map[string]string, len(projection.Enums)+len(projection.Messages))
	for _, enum := range projection.Enums {
		result[enum.FullName] = enum.Package
	}
	for _, message := range projection.Messages {
		result[message.FullName] = message.Package
	}
	return result
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
			return nil, errors.New("build Lean package names: empty protobuf package")
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
	declarations := collectLeanDeclarations(projection)
	requestsByScope, err := groupLeanDeclarationRequests(declarations)
	if err != nil {
		return nil, err
	}
	localNames, err := allocateScopedLeanNames(requestsByScope, leanPackageReservations(packageNames))
	if err != nil {
		return nil, err
	}
	return qualifyLeanDeclarationNames(declarations, packageNames, localNames)
}

func collectLeanDeclarations(projection projection) []declarationInfo {
	var declarations []declarationInfo
	for _, enum := range projection.Enums {
		declarations = append(declarations, declarationInfo{
			identity: enum.FullName, parent: enum.Parent, packageName: enum.Package, base: upperIdentifier(enum.Name),
		})
	}
	for _, message := range projection.Messages {
		declarations = append(declarations, declarationInfo{
			identity: message.FullName, parent: message.Parent, packageName: message.Package, base: upperIdentifier(message.Name),
		})
		for _, oneof := range message.Oneofs {
			declarations = append(declarations, declarationInfo{
				identity: oneof.FullName, parent: message.FullName, packageName: message.Package, base: upperIdentifier(oneof.Name),
			})
		}
	}
	for _, service := range projection.Services {
		declarations = append(declarations, declarationInfo{
			identity: service.FullName, packageName: service.Package, base: upperIdentifier(service.Name),
		})
	}
	return declarations
}

func groupLeanDeclarationRequests(declarations []declarationInfo) (map[string][]nameRequest, error) {
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
		scope := "package:" + declaration.packageName
		if declaration.parent != "" {
			scope = "declaration:" + declaration.parent
		}
		requestsByScope[scope] = append(requestsByScope[scope], nameRequest{
			identity: declaration.identity,
			base:     declaration.base,
		})
	}
	return requestsByScope, nil
}

func leanPackageReservations(packageNames map[string]leanName) map[string][]string {
	reservedByScope := make(map[string][]string)
	for packageIdentity, name := range packageNames {
		parts := strings.Split(packageIdentity, ".")
		for index, segment := range name {
			parent := strings.Join(parts[:index], ".")
			reservedByScope["package:"+parent] = append(reservedByScope["package:"+parent], segment)
		}
	}
	return reservedByScope
}

func qualifyLeanDeclarationNames(
	declarations []declarationInfo,
	packageNames map[string]leanName,
	localNames map[string]string,
) (map[string]leanName, error) {
	result := make(map[string]leanName, len(declarations))
	remaining := slices.Clone(declarations)
	for len(remaining) != 0 {
		next := remaining[:0]
		progress := false
		for _, declaration := range remaining {
			if declaration.parent == "" {
				packageName, exists := packageNames[declaration.packageName]
				if !exists {
					return nil, fmt.Errorf("build Lean declaration %q: unknown package %q", declaration.identity, declaration.packageName)
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
			Projection:    value,
			Name:          allocated[value.FullName],
			QualifiedName: appendLeanName(name, allocated[value.FullName]),
			Number:        value.Number,
		})
	}
	return result, nil
}

type leanMessageMemberNames struct {
	fields       map[string]fieldProjection
	structure    map[string]string
	constructors map[string]string
	owners       map[string]leanName
}

func planMessage(
	projection messageProjection,
	packageName leanName,
	declarationNames map[string]leanName,
	declarationPackages map[string]string,
	graph messageGraph,
	supportNamespace string,
) (leanMessagePlan, error) {
	name := declarationNames[projection.FullName]
	relativeName, err := relativeLeanName(name, packageName)
	if err != nil {
		return leanMessagePlan{}, fmt.Errorf("plan message %q: %w", projection.FullName, err)
	}
	members, err := planMessageMemberNames(projection, declarationNames)
	if err != nil {
		return leanMessagePlan{}, err
	}
	result := leanMessagePlan{Projection: projection, Name: name, Namespace: packageName, RelativeName: relativeName}
	fields, err := appendLeanMessageFields(
		&result,
		members,
		packageName,
		declarationNames,
		declarationPackages,
		graph,
		supportNamespace,
	)
	if err != nil {
		return leanMessagePlan{}, err
	}
	if err := appendLeanMessageOneofs(&result, members, fields, packageName, declarationNames); err != nil {
		return leanMessagePlan{}, err
	}
	return result, nil
}

func planMessageMemberNames(
	projection messageProjection,
	declarationNames map[string]leanName,
) (leanMessageMemberNames, error) {
	result := leanMessageMemberNames{
		fields: make(map[string]fieldProjection, len(projection.Fields)),
		owners: make(map[string]leanName, len(projection.Fields)),
	}
	requests := make([]nameRequest, 0, len(projection.Fields)+len(projection.Oneofs))
	for _, field := range projection.Fields {
		if result.fields[field.FullName].FullName != "" {
			return leanMessageMemberNames{}, fmt.Errorf("plan message %q: duplicate field %q", projection.FullName, field.FullName)
		}
		result.fields[field.FullName] = field
		if field.Oneof == "" {
			requests = append(requests, nameRequest{
				identity: field.FullName, base: lowerIdentifier(field.Name), number: field.Number, hasNumber: true,
			})
		}
	}
	for _, oneof := range projection.Oneofs {
		requests = append(requests, nameRequest{identity: oneof.FullName, base: lowerIdentifier(oneof.Name)})
		owner := declarationNames[oneof.FullName]
		for _, fieldName := range oneof.FieldNames {
			result.owners[fieldName] = owner
		}
	}
	structureNames, err := allocateNames(requests, nil)
	if err != nil {
		return leanMessageMemberNames{}, fmt.Errorf("plan message %q fields: %w", projection.FullName, err)
	}
	constructorNames, err := planOneofConstructorNames(projection.Oneofs, result.fields)
	if err != nil {
		return leanMessageMemberNames{}, err
	}
	result.structure = structureNames
	result.constructors = constructorNames
	return result, nil
}

func planOneofConstructorNames(
	oneofs []oneofProjection,
	fields map[string]fieldProjection,
) (map[string]string, error) {
	result := make(map[string]string)
	for _, oneof := range oneofs {
		constructors := make([]nameRequest, 0, len(oneof.FieldNames))
		for _, fieldName := range oneof.FieldNames {
			field, exists := fields[fieldName]
			if !exists {
				return nil, fmt.Errorf("plan oneof %q: unknown field %q", oneof.FullName, fieldName)
			}
			if field.Oneof != oneof.Name {
				return nil, fmt.Errorf("plan oneof %q: field %q belongs to %q", oneof.FullName, fieldName, field.Oneof)
			}
			constructors = append(constructors, nameRequest{
				identity: field.FullName, base: lowerIdentifier(field.Name), number: field.Number, hasNumber: true,
			})
		}
		allocated, allocateErr := allocateNames(constructors, []string{"notSet"})
		if allocateErr != nil {
			return nil, fmt.Errorf("plan oneof %q constructors: %w", oneof.FullName, allocateErr)
		}
		for fieldName, constructorName := range allocated {
			result[fieldName] = constructorName
		}
	}
	return result, nil
}

func appendLeanMessageFields(
	result *leanMessagePlan,
	members leanMessageMemberNames,
	packageName leanName,
	declarationNames map[string]leanName,
	declarationPackages map[string]string,
	graph messageGraph,
	supportNamespace string,
) (map[string]leanFieldPlan, error) {
	projection := result.Projection
	fields := make(map[string]leanFieldPlan, len(projection.Fields))
	for _, field := range projection.Fields {
		baseType, recursive, typeErr := leanFieldBaseType(
			projection,
			field,
			packageName,
			declarationNames,
			declarationPackages,
			graph,
			supportNamespace,
		)
		if typeErr != nil {
			return nil, typeErr
		}
		fieldType := wrapLeanFieldType(field, baseType)
		fieldName := members.structure[field.FullName]
		if field.Oneof != "" {
			var exists bool
			fieldName, exists = members.constructors[field.FullName]
			if !exists {
				return nil, fmt.Errorf(
					"plan field %q: unresolved oneof %q",
					field.FullName,
					field.Oneof,
				)
			}
		}
		owner := result.Name
		if oneofOwner, exists := members.owners[field.FullName]; exists {
			owner = oneofOwner
		}
		planned := leanFieldPlan{
			Projection: field, Name: fieldName, QualifiedName: appendLeanName(owner, fieldName),
			Type: fieldType, BaseType: baseType, Recursive: recursive,
		}
		fields[field.FullName] = planned
		result.Fields = append(result.Fields, planned)
		if field.Oneof == "" {
			result.StructureFields = append(result.StructureFields, leanStructureFieldPlan{Name: fieldName, Type: fieldType})
		}
	}
	return fields, nil
}

func appendLeanMessageOneofs(
	result *leanMessagePlan,
	members leanMessageMemberNames,
	fields map[string]leanFieldPlan,
	packageName leanName,
	declarationNames map[string]leanName,
) error {
	for _, oneof := range result.Projection.Oneofs {
		oneofName := declarationNames[oneof.FullName]
		oneofRelativeName, relativeErr := relativeLeanName(oneofName, packageName)
		if relativeErr != nil {
			return fmt.Errorf("plan oneof %q: %w", oneof.FullName, relativeErr)
		}
		planned := leanOneofPlan{
			Projection: oneof, Name: oneofName, RelativeName: oneofRelativeName,
			SlotName: members.structure[oneof.FullName],
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
	return nil
}

func planService(
	projection serviceProjection,
	packageName leanName,
	declarationNames map[string]leanName,
	declarationPackages map[string]string,
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
		inputType, typeErr := leanNamedReference(
			method.InputType,
			projection.Package,
			packageName,
			declarationNames,
			declarationPackages,
		)
		if typeErr != nil {
			return leanServicePlan{}, fmt.Errorf("plan method %q input: %w", method.FullName, typeErr)
		}
		outputType, typeErr := leanNamedReference(
			method.OutputType,
			projection.Package,
			packageName,
			declarationNames,
			declarationPackages,
		)
		if typeErr != nil {
			return leanServicePlan{}, fmt.Errorf("plan method %q output: %w", method.FullName, typeErr)
		}
		result.Methods = append(result.Methods, leanMethodPlan{
			Projection:      method,
			Name:            methodNames[method.FullName],
			QualifiedName:   appendLeanName(name, methodNames[method.FullName]),
			InputType:       inputType,
			OutputType:      outputType,
			FullName:        method.FullName,
			ClientStreaming: method.ClientStreaming,
			ServerStreaming: method.ServerStreaming,
			Deprecated:      method.Deprecated,
		})
	}
	return result, nil
}

func leanFieldBaseType(
	message messageProjection,
	field fieldProjection,
	packageName leanName,
	declarationNames map[string]leanName,
	declarationPackages map[string]string,
	graph messageGraph,
	supportNamespace string,
) (leanType, bool, error) {
	if field.Map {
		key, err := leanScalarType(field.MapKey, supportNamespace)
		if err != nil {
			return leanType{}, false, fmt.Errorf("plan field %q map key: %w", field.FullName, err)
		}
		var value leanType
		recursive := false
		if field.TypeName != "" {
			value, err = leanNamedReference(
				field.TypeName,
				message.Package,
				packageName,
				declarationNames,
				declarationPackages,
			)
			recursive = graph.recursive(message.FullName, field.TypeName)
			if recursive {
				value = namedLeanType(supportNamespace + ".MessageRef")
			}
		} else {
			value, err = leanScalarType(field.MapValue, supportNamespace)
		}
		if err != nil {
			return leanType{}, false, fmt.Errorf("plan field %q map value: %w", field.FullName, err)
		}
		return leanType{Kind: leanTypeList, Arguments: []leanType{{
			Kind: leanTypeProduct, Arguments: []leanType{key, value},
		}}}, recursive, nil
	}
	if field.TypeName != "" {
		result, err := leanNamedReference(
			field.TypeName,
			message.Package,
			packageName,
			declarationNames,
			declarationPackages,
		)
		if err != nil {
			return leanType{}, false, fmt.Errorf("plan field %q: %w", field.FullName, err)
		}
		recursive := graph.recursive(message.FullName, field.TypeName)
		if recursive {
			result = namedLeanType(supportNamespace + ".MessageRef")
		}
		return result, recursive, nil
	}
	result, err := leanScalarType(field.Kind, supportNamespace)
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
	declarationPackages map[string]string,
) (leanType, error) {
	name, exists := declarationNames[fullName]
	if !exists {
		return leanType{}, fmt.Errorf("unknown protobuf type %q", fullName)
	}
	targetPackage, exists := declarationPackages[fullName]
	if !exists {
		return leanType{}, fmt.Errorf("protobuf declaration %q is not a message or enum", fullName)
	}
	if targetPackage == currentPackage {
		relative, err := relativeLeanName(name, packageName)
		if err != nil {
			return leanType{}, err
		}
		return namedLeanType(relative), nil
	}
	return namedLeanType(name.String()), nil
}

func leanScalarType(kind, supportNamespace string) (leanType, error) {
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
		return namedLeanType(supportNamespace + ".Bytes"), nil
	default:
		return leanType{}, fmt.Errorf("unsupported protobuf kind %q", kind)
	}
}

func namedLeanType(name string) leanType {
	return leanType{Kind: leanTypeNamed, Name: name}
}

func validateLeanPlan(projection projection, plan leanPlan, configuration generationConfig) error {
	if len(plan.Enums) != len(projection.Enums) || len(plan.Messages) != len(projection.Messages) ||
		len(plan.Services) != len(projection.Services) {
		return errors.New("validate Lean plan: declaration count mismatch")
	}
	if !equalLeanModulePlan(plan.ProtoModule, protoModuleSpec(configuration.Layout)) {
		return errors.New("validate Lean plan: Proto module is incomplete")
	}
	if !equalLeanModulePlan(plan.TypesModule, typesModuleSpec(configuration.Layout)) {
		return errors.New("validate Lean plan: Types module is incomplete")
	}
	if !equalLeanModulePlan(plan.APIModule, apiModuleSpec(configuration.Layout)) {
		return errors.New("validate Lean plan: API module is incomplete")
	}
	if plan.supportNamespace != configuration.Layout.RootModule+".API.Proto" {
		return errors.New("validate Lean plan: support namespace is incomplete")
	}
	if err := validateLeanNames(plan.names); err != nil {
		return err
	}
	if err := validateLeanMessages(plan.Messages); err != nil {
		return err
	}
	if !reflect.DeepEqual(plan.Namespaces, buildLeanNamespacePlans(plan.Enums, plan.Messages)) {
		return errors.New("validate Lean plan: namespace ownership mismatch")
	}
	if err := validateLeanEnums(plan.Enums); err != nil {
		return err
	}
	if err := validateLeanServices(plan.Services); err != nil {
		return err
	}
	for index, service := range projection.Services {
		if plan.Services[index].Projection.FullName != service.FullName {
			return errors.New("validate Lean plan: service order mismatch")
		}
	}
	return nil
}

func validateLeanNames(names map[string]leanName) error {
	seenNames := make(map[string]string, len(names))
	for fullName, name := range names {
		if name.String() == "" {
			return fmt.Errorf("validate Lean plan: %q has empty Lean name", fullName)
		}
		if previous, duplicate := seenNames[name.String()]; duplicate {
			return fmt.Errorf("validate Lean plan: %q and %q share Lean name %q", previous, fullName, name.String())
		}
		seenNames[name.String()] = fullName
	}
	return nil
}

func validateLeanMessages(messages []leanMessagePlan) error {
	messageOrder := make(map[string]int, len(messages))
	for index, message := range messages {
		messageOrder[message.Projection.FullName] = index
	}
	for index, message := range messages {
		if err := validateLeanMessage(message, index, messageOrder); err != nil {
			return err
		}
	}
	return nil
}

func validateLeanMessage(message leanMessagePlan, index int, messageOrder map[string]int) error {
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
		if field.QualifiedName.String() == "" {
			return fmt.Errorf("validate Lean plan: field %q has empty qualified name", field.Projection.FullName)
		}
		if err := validateLeanType(field.Type); err != nil {
			return fmt.Errorf("validate Lean plan: field %q: %w", field.Projection.FullName, err)
		}
		if err := validateLeanType(field.BaseType); err != nil {
			return fmt.Errorf("validate Lean plan: field %q base type: %w", field.Projection.FullName, err)
		}
		if dependency, known := messageOrder[field.Projection.TypeName]; known && !field.Recursive && dependency >= index {
			return fmt.Errorf("validate Lean plan: message %q precedes dependency %q", message.Projection.FullName, field.Projection.TypeName)
		}
	}
	return nil
}

func validateLeanEnums(enums []leanEnumPlan) error {
	for _, enum := range enums {
		for _, value := range enum.Values {
			if value.QualifiedName.String() == "" {
				return fmt.Errorf("validate Lean plan: enum value %q has empty qualified name", value.Projection.FullName)
			}
		}
	}
	return nil
}

func validateLeanServices(services []leanServicePlan) error {
	for _, service := range services {
		for _, method := range service.Methods {
			if method.QualifiedName.String() == "" {
				return fmt.Errorf("validate Lean plan: method %q has empty qualified name", method.Projection.FullName)
			}
			if err := validateLeanType(method.InputType); err != nil {
				return fmt.Errorf("validate Lean plan: method %q input: %w", method.Projection.FullName, err)
			}
			if err := validateLeanType(method.OutputType); err != nil {
				return fmt.Errorf("validate Lean plan: method %q output: %w", method.Projection.FullName, err)
			}
		}
	}
	return nil
}

func validateLeanType(value leanType) error {
	switch value.Kind {
	case leanTypeNamed:
		if value.Name == "" || len(value.Arguments) != 0 {
			return errors.New("invalid named type")
		}
		return nil
	case leanTypeOption, leanTypeList:
		if value.Name != "" || len(value.Arguments) != 1 {
			return errors.New("type constructor requires one argument")
		}
		return validateLeanType(value.Arguments[0])
	case leanTypeProduct:
		if value.Name != "" || len(value.Arguments) != 2 {
			return errors.New("product type requires two arguments")
		}
		if err := validateLeanType(value.Arguments[0]); err != nil {
			return err
		}
		return validateLeanType(value.Arguments[1])
	default:
		return fmt.Errorf("unknown type constructor %d", value.Kind)
	}
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
