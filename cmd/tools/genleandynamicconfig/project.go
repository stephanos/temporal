package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	enumspb "go.temporal.io/api/enums/v1"
	enumsspb "go.temporal.io/server/api/enums/v1"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/namespace"
)

type (
	PrecedencePolicy string
	CodecClass       string
	SchemaKind       string
	ValueKind        string
	DefaultKind      string
	FixtureSource    string

	Catalog struct {
		Identity string              `json:"identity"`
		Settings []ProjectedSetting  `json:"settings"`
		Fixtures []ResolutionFixture `json:"fixtures"`
	}

	ProjectedSetting struct {
		Key         string             `json:"key"`
		Description string             `json:"description"`
		Policy      PrecedencePolicy   `json:"policy"`
		Schema      ValueSchema        `json:"schema"`
		Codec       CodecClass         `json:"codec"`
		Default     ProjectedDefault   `json:"default"`
		Provenance  []RegistrationSite `json:"provenance"`
		Identity    string             `json:"identity"`
	}

	ValueSchema struct {
		Kind     SchemaKind    `json:"kind"`
		GoType   string        `json:"go_type"`
		Length   int           `json:"length,omitempty"`
		Element  *ValueSchema  `json:"element,omitempty"`
		Fields   []SchemaField `json:"fields,omitempty"`
		Nullable bool          `json:"nullable,omitempty"`
	}

	SchemaField struct {
		Name   string      `json:"name"`
		Schema ValueSchema `json:"schema"`
	}

	ProjectedDefault struct {
		Kind        DefaultKind                   `json:"kind"`
		Value       *CanonicalValue               `json:"value,omitempty"`
		Constrained []ProjectedConstrainedDefault `json:"constrained,omitempty"`
		Opaque      *ProjectedOpaqueDefault       `json:"opaque,omitempty"`
	}

	ProjectedConstrainedDefault struct {
		Constraints ExactConstraints        `json:"constraints"`
		Value       CanonicalValue          `json:"value"`
		Opaque      *ProjectedOpaqueDefault `json:"opaque,omitempty"`
	}

	ProjectedOpaqueDefault struct {
		GoType string `json:"go_type"`
		Reason string `json:"reason"`
	}

	CanonicalValue struct {
		Kind   ValueKind        `json:"kind"`
		Scalar string           `json:"scalar,omitempty"`
		Items  []CanonicalValue `json:"items,omitempty"`
		Fields []CanonicalField `json:"fields,omitempty"`
	}

	CanonicalField struct {
		Name  string         `json:"name"`
		Value CanonicalValue `json:"value"`
	}

	ExactConstraints struct {
		Namespace     *string `json:"namespace"`
		NamespaceID   *string `json:"namespace_id"`
		TaskQueueName *string `json:"task_queue_name"`
		Destination   *string `json:"destination"`
		ChasmTaskType *string `json:"chasm_task_type"`
		TaskQueueType *int32  `json:"task_queue_type"`
		ShardID       *int32  `json:"shard_id"`
		TaskType      *int32  `json:"task_type"`
	}

	FixtureOverride struct {
		Constraints ExactConstraints `json:"constraints"`
		Value       CanonicalValue   `json:"value"`
	}

	ResolutionFixture struct {
		Name               string            `json:"name"`
		Policy             PrecedencePolicy  `json:"policy"`
		SettingKey         string            `json:"setting_key"`
		Context            ExactConstraints  `json:"context"`
		Overrides          []FixtureOverride `json:"overrides"`
		SelectedSource     FixtureSource     `json:"selected_source"`
		SelectedConstraint ExactConstraints  `json:"selected_constraint"`
		Result             CanonicalValue    `json:"result"`
	}

	productionSettings struct {
		Global        dynamicconfig.GlobalBoolSetting
		Namespace     dynamicconfig.NamespaceIntSetting
		NamespaceID   dynamicconfig.NamespaceIDBoolSetting
		TaskQueue     dynamicconfig.TaskQueueDurationConstrainedDefaultSetting
		ShardID       dynamicconfig.ShardIDIntSetting
		TaskType      dynamicconfig.TaskTypeDurationSetting
		Destination   dynamicconfig.DestinationDurationSetting
		ChasmTaskType dynamicconfig.ChasmTaskTypeDurationSetting
	}
)

const (
	PolicyGlobal        PrecedencePolicy = "global"
	PolicyNamespace     PrecedencePolicy = "namespace"
	PolicyNamespaceID   PrecedencePolicy = "namespace_id"
	PolicyTaskQueue     PrecedencePolicy = "task_queue"
	PolicyShardID       PrecedencePolicy = "shard_id"
	PolicyTaskType      PrecedencePolicy = "task_type"
	PolicyDestination   PrecedencePolicy = "destination"
	PolicyChasmTaskType PrecedencePolicy = "chasm_task_type"

	SchemaBool      SchemaKind = "bool"
	SchemaInt       SchemaKind = "int"
	SchemaUint      SchemaKind = "uint"
	SchemaFloat     SchemaKind = "float"
	SchemaString    SchemaKind = "string"
	SchemaDuration  SchemaKind = "duration"
	SchemaDynamic   SchemaKind = "dynamic"
	SchemaList      SchemaKind = "list"
	SchemaMap       SchemaKind = "map"
	SchemaStruct    SchemaKind = "struct"
	SchemaReference SchemaKind = "reference"
	SchemaOpaque    SchemaKind = "opaque"

	ValueNull     ValueKind = "null"
	ValueBool     ValueKind = "bool"
	ValueInt      ValueKind = "int"
	ValueUint     ValueKind = "uint"
	ValueFloat    ValueKind = "float"
	ValueString   ValueKind = "string"
	ValueDuration ValueKind = "duration"
	ValueList     ValueKind = "list"
	ValueObject   ValueKind = "object"

	DefaultConcrete    DefaultKind = "concrete"
	DefaultConstrained DefaultKind = "constrained"
	DefaultOpaque      DefaultKind = "opaque"

	SourceOverride           FixtureSource = "override"
	SourceConstrainedDefault FixtureSource = "constrained_default"
	SourceSimpleDefault      FixtureSource = "simple_default"
)

var durationType = reflect.TypeFor[time.Duration]()

func projectMetadata(metadata []dynamicconfig.SettingMetadata) (Catalog, error) {
	settings := make([]ProjectedSetting, 0, len(metadata))
	seen := make(map[string]struct{}, len(metadata))
	for _, registered := range metadata {
		key := strings.ToLower(registered.Key)
		if key == "" {
			return Catalog{}, fmt.Errorf("projection setting %q: empty normalized key", registered.Key)
		}
		if _, exists := seen[key]; exists {
			return Catalog{}, fmt.Errorf("projection setting %q: duplicate normalized key", key)
		}
		seen[key] = struct{}{}

		setting, err := projectSetting(registered, key)
		if err != nil {
			return Catalog{}, fmt.Errorf("projection setting %q: %w", key, err)
		}
		settings = append(settings, setting)
	}
	slices.SortFunc(settings, func(left, right ProjectedSetting) int {
		return strings.Compare(left.Key, right.Key)
	})
	catalog := Catalog{Settings: settings}
	identity, err := catalogIdentity(catalog)
	if err != nil {
		return Catalog{}, err
	}
	catalog.Identity = identity
	return catalog, nil
}

func projectSetting(metadata dynamicconfig.SettingMetadata, normalizedKey string) (ProjectedSetting, error) {
	policy, err := projectPolicy(metadata.Precedence)
	if err != nil {
		return ProjectedSetting{}, err
	}
	codec, err := projectCodec(metadata.Codec, metadata.ResultType)
	if err != nil {
		return ProjectedSetting{}, err
	}
	schema, err := projectSchema(metadata.ResultType)
	if err != nil {
		return ProjectedSetting{}, err
	}
	projectedDefault, err := projectDefault(metadata.Default, metadata.ResultType, policy)
	if err != nil {
		return ProjectedSetting{}, err
	}
	setting := ProjectedSetting{
		Key:         normalizedKey,
		Description: metadata.Description,
		Policy:      policy,
		Schema:      schema,
		Codec:       codec,
		Default:     projectedDefault,
	}
	identity, err := settingIdentity(setting)
	if err != nil {
		return ProjectedSetting{}, err
	}
	setting.Identity = identity
	return setting, nil
}

func projectPolicy(precedence dynamicconfig.Precedence) (PrecedencePolicy, error) {
	switch precedence {
	case dynamicconfig.PrecedenceGlobal:
		return PolicyGlobal, nil
	case dynamicconfig.PrecedenceNamespace:
		return PolicyNamespace, nil
	case dynamicconfig.PrecedenceNamespaceID:
		return PolicyNamespaceID, nil
	case dynamicconfig.PrecedenceTaskQueue:
		return PolicyTaskQueue, nil
	case dynamicconfig.PrecedenceShardID:
		return PolicyShardID, nil
	case dynamicconfig.PrecedenceTaskType:
		return PolicyTaskType, nil
	case dynamicconfig.PrecedenceDestination:
		return PolicyDestination, nil
	case dynamicconfig.PrecedenceChasmTaskType:
		return PolicyChasmTaskType, nil
	default:
		return "", fmt.Errorf("unknown precedence %d", precedence)
	}
}

func projectCodec(codec dynamicconfig.SettingCodec, resultType reflect.Type) (CodecClass, error) {
	if resultType == nil {
		return "", errors.New("missing result type")
	}
	validType := func(expected reflect.Type) error {
		if resultType != expected {
			return fmt.Errorf("codec %q result type %s does not match %s", codec, stableTypeName(resultType), stableTypeName(expected))
		}
		return nil
	}
	switch codec {
	case dynamicconfig.SettingCodecBool:
		return CodecClass(codec), validType(reflect.TypeFor[bool]())
	case dynamicconfig.SettingCodecInt:
		return CodecClass(codec), validType(reflect.TypeFor[int]())
	case dynamicconfig.SettingCodecFloat:
		return CodecClass(codec), validType(reflect.TypeFor[float64]())
	case dynamicconfig.SettingCodecString:
		return CodecClass(codec), validType(reflect.TypeFor[string]())
	case dynamicconfig.SettingCodecDuration:
		return CodecClass(codec), validType(durationType)
	case dynamicconfig.SettingCodecMap:
		return CodecClass(codec), validType(reflect.TypeFor[map[string]any]())
	case dynamicconfig.SettingCodecStructure, dynamicconfig.SettingCodecCustom:
		return CodecClass(codec), nil
	default:
		return "", fmt.Errorf("unknown codec %q", codec)
	}
}

func projectSchema(resultType reflect.Type) (ValueSchema, error) {
	if resultType == nil {
		return ValueSchema{}, errors.New("missing result type")
	}
	return projectSchemaAt(resultType, make(map[reflect.Type]struct{}))
}

func projectSchemaAt(resultType reflect.Type, active map[reflect.Type]struct{}) (ValueSchema, error) {
	goType := stableTypeName(resultType)
	if resultType == durationType {
		return ValueSchema{Kind: SchemaDuration, GoType: goType}, nil
	}
	if _, exists := active[resultType]; exists {
		return ValueSchema{Kind: SchemaReference, GoType: goType}, nil
	}
	switch resultType.Kind() {
	case reflect.Bool:
		return ValueSchema{Kind: SchemaBool, GoType: goType}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return ValueSchema{Kind: SchemaInt, GoType: goType}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return ValueSchema{Kind: SchemaUint, GoType: goType}, nil
	case reflect.Float32, reflect.Float64:
		return ValueSchema{Kind: SchemaFloat, GoType: goType}, nil
	case reflect.String:
		return ValueSchema{Kind: SchemaString, GoType: goType}, nil
	case reflect.Interface:
		return ValueSchema{Kind: SchemaDynamic, GoType: goType, Nullable: true}, nil
	case reflect.Array, reflect.Slice:
		active[resultType] = struct{}{}
		element, err := projectSchemaAt(resultType.Elem(), active)
		delete(active, resultType)
		if err != nil {
			return ValueSchema{}, err
		}
		return ValueSchema{
			Kind:     SchemaList,
			GoType:   goType,
			Length:   arrayLength(resultType),
			Element:  &element,
			Nullable: resultType.Kind() == reflect.Slice,
		}, nil
	case reflect.Map:
		if resultType.Key().Kind() != reflect.String {
			return ValueSchema{}, fmt.Errorf("map key type %s is not canonical", stableTypeName(resultType.Key()))
		}
		active[resultType] = struct{}{}
		element, err := projectSchemaAt(resultType.Elem(), active)
		delete(active, resultType)
		if err != nil {
			return ValueSchema{}, err
		}
		return ValueSchema{Kind: SchemaMap, GoType: goType, Element: &element, Nullable: true}, nil
	case reflect.Pointer:
		active[resultType] = struct{}{}
		element, err := projectSchemaAt(resultType.Elem(), active)
		delete(active, resultType)
		if err != nil {
			return ValueSchema{}, err
		}
		element.Nullable = true
		element.GoType = goType
		return element, nil
	case reflect.Struct:
		active[resultType] = struct{}{}
		fields := make([]SchemaField, 0, resultType.NumField())
		for index := range resultType.NumField() {
			field := resultType.Field(index)
			fieldSchema, err := projectSchemaAt(field.Type, active)
			if err != nil {
				delete(active, resultType)
				return ValueSchema{}, fmt.Errorf("field %s: %w", field.Name, err)
			}
			fields = append(fields, SchemaField{Name: field.Name, Schema: fieldSchema})
		}
		delete(active, resultType)
		slices.SortFunc(fields, func(left, right SchemaField) int {
			return strings.Compare(left.Name, right.Name)
		})
		return ValueSchema{Kind: SchemaStruct, GoType: goType, Fields: fields}, nil
	default:
		return ValueSchema{Kind: SchemaOpaque, GoType: goType, Nullable: isNilable(resultType)}, nil
	}
}

func arrayLength(resultType reflect.Type) int {
	if resultType.Kind() == reflect.Array {
		return resultType.Len()
	}
	return 0
}

func projectDefault(
	metadata dynamicconfig.SettingDefaultMetadata,
	resultType reflect.Type,
	policy PrecedencePolicy,
) (ProjectedDefault, error) {
	switch metadata.Kind {
	case dynamicconfig.SettingDefaultConcrete:
		if len(metadata.Constrained) != 0 || metadata.Opaque.ResultType != nil || metadata.Opaque.Reason != "" {
			return ProjectedDefault{}, errors.New("concrete default has conflicting metadata")
		}
		if err := validateConcreteType(metadata.Value, resultType); err != nil {
			return ProjectedDefault{}, err
		}
		value, err := canonicalValue(reflect.ValueOf(metadata.Value), resultType)
		if err != nil {
			return ProjectedDefault{}, err
		}
		return ProjectedDefault{Kind: DefaultConcrete, Value: &value}, nil
	case dynamicconfig.SettingDefaultOpaque:
		opaque, err := projectOpaqueDefault(metadata, resultType)
		if err != nil {
			return ProjectedDefault{}, err
		}
		return ProjectedDefault{Kind: DefaultOpaque, Opaque: &opaque}, nil
	case dynamicconfig.SettingDefaultConstrained:
		return projectConstrainedDefault(metadata, resultType, policy)
	default:
		return ProjectedDefault{}, fmt.Errorf("unknown default kind %q", metadata.Kind)
	}
}

func projectConstrainedDefault(
	metadata dynamicconfig.SettingDefaultMetadata,
	resultType reflect.Type,
	policy PrecedencePolicy,
) (ProjectedDefault, error) {
	if metadata.Value != nil || len(metadata.Constrained) == 0 ||
		metadata.Opaque.ResultType != nil || metadata.Opaque.Reason != "" {
		return ProjectedDefault{}, errors.New("constrained default has conflicting or empty metadata")
	}
	projected := make([]ProjectedConstrainedDefault, 0, len(metadata.Constrained))
	seen := make(map[string]struct{}, len(metadata.Constrained))
	hasFallback := false
	for _, constrained := range metadata.Constrained {
		constraints := projectConstraints(constrained.Constraints)
		if err := validateConstraintShape(policy, constraints); err != nil {
			return ProjectedDefault{}, err
		}
		key := constraintKey(constraints)
		if _, exists := seen[key]; exists {
			return ProjectedDefault{}, fmt.Errorf("duplicate exact constraint %s", constraintDisplay(constraints))
		}
		seen[key] = struct{}{}
		if constraints == (ExactConstraints{}) {
			hasFallback = true
		}
		value, opaque, err := projectConstrainedLeaf(constrained.Default, resultType)
		if err != nil {
			return ProjectedDefault{}, fmt.Errorf("constraint %s: %w", constraintDisplay(constraints), err)
		}
		projected = append(projected, ProjectedConstrainedDefault{
			Constraints: constraints,
			Value:       value,
			Opaque:      opaque,
		})
	}
	if !hasFallback {
		return ProjectedDefault{}, errors.New("constrained default has no unconstrained fallback")
	}
	slices.SortFunc(projected, func(left, right ProjectedConstrainedDefault) int {
		return strings.Compare(constraintSortKey(left.Constraints), constraintSortKey(right.Constraints))
	})
	return ProjectedDefault{Kind: DefaultConstrained, Constrained: projected}, nil
}

func projectConstrainedLeaf(
	metadata dynamicconfig.SettingDefaultMetadata,
	resultType reflect.Type,
) (CanonicalValue, *ProjectedOpaqueDefault, error) {
	switch metadata.Kind {
	case dynamicconfig.SettingDefaultConcrete:
		if len(metadata.Constrained) != 0 || metadata.Opaque.ResultType != nil || metadata.Opaque.Reason != "" {
			return CanonicalValue{}, nil, errors.New("concrete default has conflicting metadata")
		}
		if err := validateConcreteType(metadata.Value, resultType); err != nil {
			return CanonicalValue{}, nil, err
		}
		value, err := canonicalValue(reflect.ValueOf(metadata.Value), resultType)
		return value, nil, err
	case dynamicconfig.SettingDefaultOpaque:
		opaque, err := projectOpaqueDefault(metadata, resultType)
		return CanonicalValue{Kind: ValueNull}, &opaque, err
	case dynamicconfig.SettingDefaultConstrained:
		return CanonicalValue{}, nil, errors.New("nested constrained default")
	default:
		return CanonicalValue{}, nil, fmt.Errorf("unknown default kind %q", metadata.Kind)
	}
}

func projectOpaqueDefault(
	metadata dynamicconfig.SettingDefaultMetadata,
	resultType reflect.Type,
) (ProjectedOpaqueDefault, error) {
	if metadata.Value != nil || len(metadata.Constrained) != 0 ||
		metadata.Opaque.ResultType == nil || metadata.Opaque.Reason == "" {
		return ProjectedOpaqueDefault{}, errors.New("opaque default has incomplete or conflicting metadata")
	}
	if metadata.Opaque.ResultType != resultType {
		return ProjectedOpaqueDefault{}, fmt.Errorf(
			"opaque result type %s does not match %s",
			stableTypeName(metadata.Opaque.ResultType),
			stableTypeName(resultType),
		)
	}
	return ProjectedOpaqueDefault{GoType: stableTypeName(resultType), Reason: metadata.Opaque.Reason}, nil
}

func validateConcreteType(value any, resultType reflect.Type) error {
	if value == nil {
		switch resultType.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			return nil
		default:
			return fmt.Errorf("nil concrete default is not assignable to %s", stableTypeName(resultType))
		}
	}
	if !reflect.TypeOf(value).AssignableTo(resultType) {
		return fmt.Errorf(
			"concrete default type %s is not assignable to %s",
			stableTypeName(reflect.TypeOf(value)),
			stableTypeName(resultType),
		)
	}
	return nil
}

func canonicalValue(value reflect.Value, expectedType reflect.Type) (CanonicalValue, error) {
	if !value.IsValid() {
		return CanonicalValue{Kind: ValueNull}, nil
	}
	if value.Type() == durationType {
		return CanonicalValue{Kind: ValueDuration, Scalar: strconv.FormatInt(value.Int(), 10)}, nil
	}
	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return CanonicalValue{Kind: ValueNull}, nil
		}
		value = value.Elem()
		if value.Type() == durationType {
			return CanonicalValue{Kind: ValueDuration, Scalar: strconv.FormatInt(value.Int(), 10)}, nil
		}
	}
	switch value.Kind() {
	case reflect.Bool:
		return CanonicalValue{Kind: ValueBool, Scalar: strconv.FormatBool(value.Bool())}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return CanonicalValue{Kind: ValueInt, Scalar: strconv.FormatInt(value.Int(), 10)}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return CanonicalValue{Kind: ValueUint, Scalar: strconv.FormatUint(value.Uint(), 10)}, nil
	case reflect.Float32, reflect.Float64:
		floating := value.Float()
		if math.IsNaN(floating) || math.IsInf(floating, 0) {
			return CanonicalValue{}, errors.New("non-finite float is not canonical")
		}
		return CanonicalValue{
			Kind:   ValueFloat,
			Scalar: strconv.FormatFloat(floating, 'g', -1, value.Type().Bits()),
		}, nil
	case reflect.String:
		return CanonicalValue{Kind: ValueString, Scalar: value.String()}, nil
	case reflect.Array, reflect.Slice:
		return canonicalList(value)
	case reflect.Map:
		return canonicalMap(value)
	case reflect.Struct:
		return canonicalStruct(value)
	case reflect.Chan, reflect.Func, reflect.UnsafePointer:
		if value.IsNil() {
			return CanonicalValue{Kind: ValueNull}, nil
		}
		return CanonicalValue{}, fmt.Errorf("unsupported concrete %s value", value.Kind())
	default:
		return CanonicalValue{}, fmt.Errorf(
			"unsupported concrete %s value for %s",
			value.Kind(),
			stableTypeName(expectedType),
		)
	}
}

func canonicalList(value reflect.Value) (CanonicalValue, error) {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return CanonicalValue{Kind: ValueNull}, nil
	}
	items := make([]CanonicalValue, value.Len())
	for index := range value.Len() {
		item, err := canonicalValue(value.Index(index), value.Type().Elem())
		if err != nil {
			return CanonicalValue{}, fmt.Errorf("item %d: %w", index, err)
		}
		items[index] = item
	}
	return CanonicalValue{Kind: ValueList, Items: items}, nil
}

func canonicalMap(value reflect.Value) (CanonicalValue, error) {
	if value.IsNil() {
		return CanonicalValue{Kind: ValueNull}, nil
	}
	if value.Type().Key().Kind() != reflect.String {
		return CanonicalValue{}, fmt.Errorf("map key type %s is not canonical", stableTypeName(value.Type().Key()))
	}
	keys := value.MapKeys()
	slices.SortFunc(keys, func(left, right reflect.Value) int {
		return strings.Compare(left.String(), right.String())
	})
	fields := make([]CanonicalField, 0, len(keys))
	for _, key := range keys {
		item, err := canonicalValue(value.MapIndex(key), value.Type().Elem())
		if err != nil {
			return CanonicalValue{}, fmt.Errorf("map key %q: %w", key.String(), err)
		}
		fields = append(fields, CanonicalField{Name: key.String(), Value: item})
	}
	return CanonicalValue{Kind: ValueObject, Fields: fields}, nil
}

func canonicalStruct(value reflect.Value) (CanonicalValue, error) {
	fields := make([]CanonicalField, 0, value.NumField())
	for index := range value.NumField() {
		fieldInfo := value.Type().Field(index)
		fieldValue, err := canonicalValue(value.Field(index), fieldInfo.Type)
		if err != nil {
			return CanonicalValue{}, fmt.Errorf("field %s: %w", fieldInfo.Name, err)
		}
		fields = append(fields, CanonicalField{Name: fieldInfo.Name, Value: fieldValue})
	}
	slices.SortFunc(fields, func(left, right CanonicalField) int {
		return strings.Compare(left.Name, right.Name)
	})
	return CanonicalValue{Kind: ValueObject, Fields: fields}, nil
}

func projectConstraints(constraints dynamicconfig.Constraints) ExactConstraints {
	return ExactConstraints{
		Namespace:     nonemptyString(constraints.Namespace),
		NamespaceID:   nonemptyString(constraints.NamespaceID),
		TaskQueueName: nonemptyString(constraints.TaskQueueName),
		Destination:   nonemptyString(constraints.Destination),
		ChasmTaskType: nonemptyString(constraints.ChasmTaskType),
		TaskQueueType: nonzeroInt32(int32(constraints.TaskQueueType)),
		ShardID:       nonzeroInt32(constraints.ShardID),
		TaskType:      nonzeroInt32(int32(constraints.TaskType)),
	}
}

func nonemptyString(value string) *string {
	if value == "" {
		return nil
	}
	result := value
	return &result
}

func nonzeroInt32(value int32) *int32 {
	if value == 0 {
		return nil
	}
	result := value
	return &result
}

func validateConstraintShape(policy PrecedencePolicy, constraints ExactConstraints) error {
	shape := constraintShape(constraints)
	allowed := allowedConstraintShapes(policy)
	if !slices.Contains(allowed, shape) {
		return fmt.Errorf(
			"constraint %s is illegal for %s precedence",
			constraintDisplay(constraints),
			policy,
		)
	}
	return nil
}

func constraintShape(constraints ExactConstraints) string {
	var fields []string
	if constraints.Namespace != nil {
		fields = append(fields, "namespace")
	}
	if constraints.NamespaceID != nil {
		fields = append(fields, "namespace_id")
	}
	if constraints.TaskQueueName != nil {
		fields = append(fields, "task_queue_name")
	}
	if constraints.Destination != nil {
		fields = append(fields, "destination")
	}
	if constraints.ChasmTaskType != nil {
		fields = append(fields, "chasm_task_type")
	}
	if constraints.TaskQueueType != nil {
		fields = append(fields, "task_queue_type")
	}
	if constraints.ShardID != nil {
		fields = append(fields, "shard_id")
	}
	if constraints.TaskType != nil {
		fields = append(fields, "task_type")
	}
	return strings.Join(fields, "+")
}

func allowedConstraintShapes(policy PrecedencePolicy) []string {
	switch policy {
	case PolicyGlobal:
		return []string{""}
	case PolicyNamespace:
		return []string{"namespace", ""}
	case PolicyNamespaceID:
		return []string{"namespace_id", ""}
	case PolicyTaskQueue:
		return []string{
			"namespace+task_queue_name+task_queue_type",
			"namespace+task_queue_name",
			"task_queue_name",
			"namespace",
			"",
		}
	case PolicyShardID:
		return []string{"shard_id", ""}
	case PolicyTaskType:
		return []string{"task_type", ""}
	case PolicyDestination:
		return []string{"namespace+destination", "destination", "namespace", ""}
	case PolicyChasmTaskType:
		return []string{"chasm_task_type", ""}
	default:
		return nil
	}
}

func constraintKey(constraints ExactConstraints) string {
	return constraintDisplay(constraints)
}

func constraintSortKey(constraints ExactConstraints) string {
	if constraints == (ExactConstraints{}) {
		return "0"
	}
	return "1" + constraintKey(constraints)
}

func constraintDisplay(constraints ExactConstraints) string {
	var fields []string
	appendString := func(name string, value *string) {
		if value != nil {
			fields = append(fields, name+"="+strconv.Quote(*value))
		}
	}
	appendInt := func(name string, value *int32) {
		if value != nil {
			fields = append(fields, name+"="+strconv.FormatInt(int64(*value), 10))
		}
	}
	appendString("namespace", constraints.Namespace)
	appendString("namespace_id", constraints.NamespaceID)
	appendString("task_queue_name", constraints.TaskQueueName)
	appendString("destination", constraints.Destination)
	appendString("chasm_task_type", constraints.ChasmTaskType)
	appendInt("task_queue_type", constraints.TaskQueueType)
	appendInt("shard_id", constraints.ShardID)
	appendInt("task_type", constraints.TaskType)
	return "{" + strings.Join(fields, ",") + "}"
}

func stableTypeName(value reflect.Type) string {
	if value == nil {
		return "<nil>"
	}
	if value.Name() != "" && value.PkgPath() != "" {
		return value.PkgPath() + "." + value.Name()
	}
	return value.String()
}

func isNilable(value reflect.Type) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

func settingIdentity(setting ProjectedSetting) (string, error) {
	return digest(struct {
		Key     string           `json:"key"`
		Policy  PrecedencePolicy `json:"policy"`
		Schema  ValueSchema      `json:"schema"`
		Codec   CodecClass       `json:"codec"`
		Default ProjectedDefault `json:"default"`
	}{
		Key:     setting.Key,
		Policy:  setting.Policy,
		Schema:  setting.Schema,
		Codec:   setting.Codec,
		Default: setting.Default,
	})
}

func catalogIdentity(catalog Catalog) (string, error) {
	type identityEntry struct {
		Key      string `json:"key"`
		Identity string `json:"identity"`
	}
	settings := make([]identityEntry, len(catalog.Settings))
	for index, setting := range catalog.Settings {
		settings[index] = identityEntry{Key: setting.Key, Identity: setting.Identity}
	}
	return digest(struct {
		Settings []identityEntry     `json:"settings"`
		Fixtures []ResolutionFixture `json:"fixtures"`
	}{Settings: settings, Fixtures: catalog.Fixtures})
}

func digest(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("canonical identity encoding: %w", err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func allPrecedencePolicies() []PrecedencePolicy {
	return []PrecedencePolicy{
		PolicyChasmTaskType,
		PolicyDestination,
		PolicyGlobal,
		PolicyNamespace,
		PolicyNamespaceID,
		PolicyShardID,
		PolicyTaskQueue,
		PolicyTaskType,
	}
}

func validateFixtures(fixtures []ResolutionFixture) error {
	seen := make(map[string]struct{}, len(fixtures))
	for _, fixture := range fixtures {
		if fixture.Name == "" {
			return errors.New("fixture has empty name")
		}
		if _, exists := seen[fixture.Name]; exists {
			return fmt.Errorf("fixture %q is duplicated", fixture.Name)
		}
		seen[fixture.Name] = struct{}{}
		if fixture.SettingKey == "" || fixture.SettingKey != strings.ToLower(fixture.SettingKey) {
			return fmt.Errorf("fixture %q has non-canonical setting key %q", fixture.Name, fixture.SettingKey)
		}
		if !slices.Contains(allPrecedencePolicies(), fixture.Policy) {
			return fmt.Errorf("fixture %q has unknown policy %q", fixture.Name, fixture.Policy)
		}
		if err := validateConstraintShape(fixture.Policy, fixture.SelectedConstraint); err != nil {
			return fmt.Errorf("fixture %q selected constraint: %w", fixture.Name, err)
		}
		if fixture.Result.Kind == "" {
			return fmt.Errorf("fixture %q has empty result", fixture.Name)
		}
		overrides := make(map[string]struct{}, len(fixture.Overrides))
		for _, override := range fixture.Overrides {
			if err := validateConstraintShape(fixture.Policy, override.Constraints); err != nil {
				return fmt.Errorf("fixture %q override: %w", fixture.Name, err)
			}
			key := constraintKey(override.Constraints)
			if _, exists := overrides[key]; exists {
				return fmt.Errorf("fixture %q has duplicate override constraint %s", fixture.Name, constraintDisplay(override.Constraints))
			}
			overrides[key] = struct{}{}
		}
		switch fixture.SelectedSource {
		case SourceOverride, SourceConstrainedDefault, SourceSimpleDefault:
		default:
			return fmt.Errorf("fixture %q has unknown selected source %q", fixture.Name, fixture.SelectedSource)
		}
	}
	return nil
}

func productionFixtureShape() []ResolutionFixture {
	namespaceName := "fixture-namespace"
	namespaceID := "11111111-1111-1111-1111-111111111111"
	taskQueue := "temporal-sys-per-ns-tq"
	destination := "fixture-destination"
	chasmTaskType := "activity.dispatch"
	taskQueueType := int32(1)
	shardID := int32(7)
	taskType := int32(4)
	global := ExactConstraints{}
	namespaceOnly := ExactConstraints{Namespace: &namespaceName}
	taskQueueOnly := ExactConstraints{TaskQueueName: &taskQueue}
	namespaceTaskQueue := ExactConstraints{Namespace: &namespaceName, TaskQueueName: &taskQueue}
	namespaceTaskQueueType := ExactConstraints{
		Namespace:     &namespaceName,
		TaskQueueName: &taskQueue,
		TaskQueueType: &taskQueueType,
	}
	fixtures := []ResolutionFixture{
		newFixture("chasm-task-type-specific", PolicyChasmTaskType, "history.chasmstandbytaskdiscarddelay",
			ExactConstraints{ChasmTaskType: &chasmTaskType},
			[]FixtureOverride{{Constraints: ExactConstraints{ChasmTaskType: &chasmTaskType}, Value: durationValue(9 * time.Second)}},
			SourceOverride, ExactConstraints{ChasmTaskType: &chasmTaskType}, durationValue(9*time.Second)),
		newFixture("destination-namespace-fallback", PolicyDestination, "callback.request.timeout",
			ExactConstraints{Namespace: &namespaceName, Destination: &destination},
			[]FixtureOverride{{Constraints: namespaceOnly, Value: durationValue(8 * time.Second)}},
			SourceOverride, namespaceOnly, durationValue(8*time.Second)),
		newFixture("destination-specific", PolicyDestination, "callback.request.timeout",
			ExactConstraints{Namespace: &namespaceName, Destination: &destination},
			[]FixtureOverride{{Constraints: ExactConstraints{Namespace: &namespaceName, Destination: &destination}, Value: durationValue(7 * time.Second)}},
			SourceOverride, ExactConstraints{Namespace: &namespaceName, Destination: &destination}, durationValue(7*time.Second)),
		newFixture("global-simple-default", PolicyGlobal, "admin.enablelisthistorytasks",
			global, nil, SourceSimpleDefault, global, boolValue(true)),
		newFixture("global-specific", PolicyGlobal, "admin.enablelisthistorytasks",
			global, []FixtureOverride{{Constraints: global, Value: boolValue(true)}}, SourceOverride, global, boolValue(true)),
		newFixture("namespace-id-specific", PolicyNamespaceID, "history.skipreapplicationbynamespaceid",
			ExactConstraints{NamespaceID: &namespaceID},
			[]FixtureOverride{{Constraints: ExactConstraints{NamespaceID: &namespaceID}, Value: boolValue(true)}},
			SourceOverride, ExactConstraints{NamespaceID: &namespaceID}, boolValue(true)),
		newFixture("namespace-specific", PolicyNamespace, "callback.maxperexecution",
			namespaceOnly, []FixtureOverride{{Constraints: namespaceOnly, Value: intValue(37)}},
			SourceOverride, namespaceOnly, intValue(37)),
		newFixture("namespace-unconstrained-fallback", PolicyNamespace, "callback.maxperexecution",
			namespaceOnly, []FixtureOverride{{Constraints: global, Value: intValue(38)}},
			SourceOverride, global, intValue(38)),
		newFixture("shard-id-specific", PolicyShardID, "history.replicationtaskprocessorerrorretrymaxattempts",
			ExactConstraints{ShardID: &shardID},
			[]FixtureOverride{{Constraints: ExactConstraints{ShardID: &shardID}, Value: intValue(9)}},
			SourceOverride, ExactConstraints{ShardID: &shardID}, intValue(9)),
		newFixture("task-queue-constrained-default-before-namespace-override", PolicyTaskQueue, "matching.updateackinterval",
			namespaceTaskQueueType, []FixtureOverride{{Constraints: namespaceOnly, Value: durationValue(2 * time.Minute)}},
			SourceConstrainedDefault, taskQueueOnly, durationValue(5*time.Minute)),
		newFixture("task-queue-specific-override-before-constrained-default", PolicyTaskQueue, "matching.updateackinterval",
			namespaceTaskQueueType, []FixtureOverride{{Constraints: namespaceTaskQueue, Value: durationValue(2 * time.Minute)}},
			SourceOverride, namespaceTaskQueue, durationValue(2*time.Minute)),
		newFixture("task-queue-specific", PolicyTaskQueue, "matching.updateackinterval",
			namespaceTaskQueueType, []FixtureOverride{{Constraints: namespaceTaskQueueType, Value: durationValue(3 * time.Minute)}},
			SourceOverride, namespaceTaskQueueType, durationValue(3*time.Minute)),
		newFixture("task-type-specific", PolicyTaskType, "history.standbytaskmissingeventsresenddelay",
			ExactConstraints{TaskType: &taskType},
			[]FixtureOverride{{Constraints: ExactConstraints{TaskType: &taskType}, Value: durationValue(13 * time.Second)}},
			SourceOverride, ExactConstraints{TaskType: &taskType}, durationValue(13*time.Second)),
	}
	slices.SortFunc(fixtures, func(left, right ResolutionFixture) int {
		return strings.Compare(left.Name, right.Name)
	})
	return fixtures
}

func computeProductionFixtures(settings productionSettings) ([]ResolutionFixture, error) {
	fixtures := productionFixtureShape()
	for index := range fixtures {
		fixture := &fixtures[index]
		values := make([]dynamicconfig.ConstrainedValue, len(fixture.Overrides))
		for overrideIndex, override := range fixture.Overrides {
			value, err := fixtureNativeValue(override.Value)
			if err != nil {
				return nil, fmt.Errorf("fixture %q override %s: %w", fixture.Name, constraintDisplay(override.Constraints), err)
			}
			values[overrideIndex] = dynamicconfig.ConstrainedValue{
				Constraints: runtimeConstraints(override.Constraints),
				Value:       value,
			}
		}
		client := dynamicconfig.StaticClient{
			dynamicconfig.MakeKey(fixture.SettingKey): values,
		}
		collection := dynamicconfig.NewCollection(client, log.NewNoopLogger())
		observed, settingKey, err := resolveProductionFixture(*fixture, collection, settings)
		if err != nil {
			return nil, err
		}
		if settingKey != fixture.SettingKey {
			return nil, fmt.Errorf(
				"fixture %q setting key %q does not match registered setting %q",
				fixture.Name,
				fixture.SettingKey,
				settingKey,
			)
		}
		canonical, err := canonicalValue(reflect.ValueOf(observed), reflect.TypeOf(observed))
		if err != nil {
			return nil, fmt.Errorf("fixture %q result: %w", fixture.Name, err)
		}
		if !reflect.DeepEqual(canonical, fixture.Result) {
			return nil, fmt.Errorf(
				"fixture %q result mismatch: expected %+v, got %+v",
				fixture.Name,
				fixture.Result,
				canonical,
			)
		}
		fixture.Result = canonical
	}
	return fixtures, nil
}

func resolveProductionFixture(
	fixture ResolutionFixture,
	collection *dynamicconfig.Collection,
	settings productionSettings,
) (any, string, error) {
	context := fixture.Context
	switch fixture.Policy {
	case PolicyGlobal:
		return settings.Global.Get(collection)(), settings.Global.Key().String(), nil
	case PolicyNamespace:
		if context.Namespace == nil {
			return nil, "", fmt.Errorf("fixture %q is missing namespace context", fixture.Name)
		}
		return settings.Namespace.Get(collection)(*context.Namespace), settings.Namespace.Key().String(), nil
	case PolicyNamespaceID:
		if context.NamespaceID == nil {
			return nil, "", fmt.Errorf("fixture %q is missing namespace ID context", fixture.Name)
		}
		return settings.NamespaceID.Get(collection)(namespace.ID(*context.NamespaceID)), settings.NamespaceID.Key().String(), nil
	case PolicyTaskQueue:
		if context.Namespace == nil || context.TaskQueueName == nil || context.TaskQueueType == nil {
			return nil, "", fmt.Errorf("fixture %q has incomplete task queue context", fixture.Name)
		}
		return settings.TaskQueue.Get(collection)(
			*context.Namespace,
			*context.TaskQueueName,
			enumspb.TaskQueueType(*context.TaskQueueType),
		), settings.TaskQueue.Key().String(), nil
	case PolicyShardID:
		if context.ShardID == nil {
			return nil, "", fmt.Errorf("fixture %q is missing shard ID context", fixture.Name)
		}
		return settings.ShardID.Get(collection)(*context.ShardID), settings.ShardID.Key().String(), nil
	case PolicyTaskType:
		if context.TaskType == nil {
			return nil, "", fmt.Errorf("fixture %q is missing task type context", fixture.Name)
		}
		return settings.TaskType.Get(collection)(enumsspb.TaskType(*context.TaskType)), settings.TaskType.Key().String(), nil
	case PolicyDestination:
		if context.Namespace == nil || context.Destination == nil {
			return nil, "", fmt.Errorf("fixture %q has incomplete destination context", fixture.Name)
		}
		return settings.Destination.Get(collection)(
			*context.Namespace,
			*context.Destination,
		), settings.Destination.Key().String(), nil
	case PolicyChasmTaskType:
		if context.ChasmTaskType == nil {
			return nil, "", fmt.Errorf("fixture %q is missing CHASM task type context", fixture.Name)
		}
		return settings.ChasmTaskType.Get(collection)(*context.ChasmTaskType), settings.ChasmTaskType.Key().String(), nil
	default:
		return nil, "", fmt.Errorf("fixture %q has unknown policy %q", fixture.Name, fixture.Policy)
	}
}

func fixtureNativeValue(value CanonicalValue) (any, error) {
	switch value.Kind {
	case ValueBool:
		parsed, err := strconv.ParseBool(value.Scalar)
		return parsed, err
	case ValueInt:
		parsed, err := strconv.ParseInt(value.Scalar, 10, 64)
		if err != nil {
			return nil, err
		}
		return int(parsed), nil
	case ValueFloat:
		return strconv.ParseFloat(value.Scalar, 64)
	case ValueString:
		return value.Scalar, nil
	case ValueDuration:
		parsed, err := strconv.ParseInt(value.Scalar, 10, 64)
		if err != nil {
			return nil, err
		}
		return time.Duration(parsed), nil
	default:
		return nil, fmt.Errorf("unsupported fixture value kind %q", value.Kind)
	}
}

func runtimeConstraints(constraints ExactConstraints) dynamicconfig.Constraints {
	result := dynamicconfig.Constraints{}
	if constraints.Namespace != nil {
		result.Namespace = *constraints.Namespace
	}
	if constraints.NamespaceID != nil {
		result.NamespaceID = *constraints.NamespaceID
	}
	if constraints.TaskQueueName != nil {
		result.TaskQueueName = *constraints.TaskQueueName
	}
	if constraints.Destination != nil {
		result.Destination = *constraints.Destination
	}
	if constraints.ChasmTaskType != nil {
		result.ChasmTaskType = *constraints.ChasmTaskType
	}
	if constraints.TaskQueueType != nil {
		result.TaskQueueType = enumspb.TaskQueueType(*constraints.TaskQueueType)
	}
	if constraints.ShardID != nil {
		result.ShardID = *constraints.ShardID
	}
	if constraints.TaskType != nil {
		result.TaskType = enumsspb.TaskType(*constraints.TaskType)
	}
	return result
}

func newFixture(
	name string,
	policy PrecedencePolicy,
	settingKey string,
	context ExactConstraints,
	overrides []FixtureOverride,
	source FixtureSource,
	selected ExactConstraints,
	result CanonicalValue,
) ResolutionFixture {
	fixture := ResolutionFixture{
		Name:               name,
		Policy:             policy,
		SettingKey:         settingKey,
		Context:            context,
		Overrides:          overrides,
		SelectedSource:     source,
		SelectedConstraint: selected,
		Result:             result,
	}
	return fixture
}

func boolValue(value bool) CanonicalValue {
	return CanonicalValue{Kind: ValueBool, Scalar: strconv.FormatBool(value)}
}

func intValue(value int) CanonicalValue {
	return CanonicalValue{Kind: ValueInt, Scalar: strconv.Itoa(value)}
}

func durationValue(value time.Duration) CanonicalValue {
	return CanonicalValue{Kind: ValueDuration, Scalar: strconv.FormatInt(int64(value), 10)}
}
