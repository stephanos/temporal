package main

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

const (
	dynamicConfigFacadePath   = "Temporal/DynamicConfig.lean"
	dynamicConfigTypesPath    = "Temporal/DynamicConfig/Types.lean"
	dynamicConfigSettingsPath = "Temporal/DynamicConfig/Settings.lean"
	dynamicConfigFacadeDoc    = "/-!\n" +
		"Public facade for the generated Temporal dynamic-configuration structural catalog.\n\n" +
		"It re-exports the catalog vocabulary and setting registry. Handwritten product interpretation\n" +
		"lives in `Temporal.System.Configuration` and its owner modules.\n" +
		"-/"
	dynamicConfigTypesDoc = "/-!\n" +
		"Structural vocabulary for the generated Temporal dynamic-configuration registry.\n\n" +
		"The types retain precedence, schema, default, provenance, and resolution-fixture data without\n" +
		"assigning behavioral meaning to any setting.\n" +
		"-/"
	dynamicConfigSettingsDoc = "/-!\n" +
		"Generated Temporal dynamic-configuration registry data.\n\n" +
		"Individual definitions retain structural setting metadata. `all` exposes the complete catalog,\n" +
		"`catalogIdentity` binds it to its source projection, and `fixtures` records cross-language\n" +
		"resolution cases.\n" +
		"-/"
)

func renderCatalog(catalog Catalog) (map[string][]byte, error) {
	settings := slices.Clone(catalog.Settings)
	slices.SortFunc(settings, func(left, right ProjectedSetting) int {
		return strings.Compare(left.Key, right.Key)
	})
	identifiers := make(map[string]string, len(settings))
	for _, setting := range settings {
		identifier, err := settingIdentifier(setting.Key)
		if err != nil {
			return nil, err
		}
		if prior, exists := identifiers[identifier]; exists {
			return nil, fmt.Errorf(
				"settings %q and %q have the same Lean identifier %q",
				prior,
				setting.Key,
				identifier,
			)
		}
		identifiers[identifier] = setting.Key
	}
	fixtures := slices.Clone(catalog.Fixtures)
	slices.SortFunc(fixtures, func(left, right ResolutionFixture) int {
		return strings.Compare(left.Name, right.Name)
	})

	types := renderDynamicConfigTypes()
	settingsModule, err := renderDynamicConfigSettings(catalog.Identity, settings, fixtures)
	if err != nil {
		return nil, err
	}
	facade := renderDynamicConfigFacade()
	return map[string][]byte{
		dynamicConfigFacadePath:   facade,
		dynamicConfigTypesPath:    types,
		dynamicConfigSettingsPath: settingsModule,
	}, nil
}

func renderDynamicConfigFacade() []byte {
	imports := []string{
		"Temporal.DynamicConfig.Settings",
		"Temporal.DynamicConfig.Types",
	}
	slices.Sort(imports)
	var generated strings.Builder
	writeDynamicConfigHeader(&generated)
	for _, imported := range imports {
		fmt.Fprintf(&generated, "import %s\n", imported)
	}
	writeDynamicConfigModuleDoc(&generated, dynamicConfigFacadeDoc)
	return []byte(strings.TrimRight(generated.String(), "\n") + "\n")
}

func renderDynamicConfigTypes() []byte {
	var generated strings.Builder
	writeDynamicConfigHeader(&generated)
	writeDynamicConfigModuleDoc(&generated, dynamicConfigTypesDoc)
	generated.WriteString(`set_option linter.missingDocs false

namespace Temporal.DynamicConfig

inductive PrecedencePolicy where
  | global
  | namespace
  | namespaceId
  | taskQueue
  | shardId
  | taskType
  | destination
  | chasmTaskType
  deriving DecidableEq, Repr

inductive CodecClass where
  | bool
  | int
  | float
  | string
  | duration
  | map
  | structured
  | custom
  deriving DecidableEq, Repr

mutual
  inductive ValueSchema where
    | bool (goType : String) (nullable : Bool)
    | int (goType : String) (nullable : Bool)
    | uint (goType : String) (nullable : Bool)
    | float (goType : String) (nullable : Bool)
    | string (goType : String) (nullable : Bool)
    | duration (goType : String) (nullable : Bool)
    | dynamicValue (goType : String) (nullable : Bool)
    | list (goType : String) (length : Nat) (element : ValueSchema) (nullable : Bool)
    | map (goType : String) (element : ValueSchema) (nullable : Bool)
    | struct (goType : String) (fields : SchemaFields) (nullable : Bool)
    | reference (goType : String) (nullable : Bool)
    | opaque (goType : String) (nullable : Bool)

  inductive SchemaFields where
    | nil
    | cons (name : String) (schema : ValueSchema) (tail : SchemaFields)
end

deriving instance DecidableEq for ValueSchema, SchemaFields
deriving instance Repr for ValueSchema, SchemaFields

mutual
  inductive CanonicalValue where
    | null
    | bool (value : Bool)
    | int (value : Int)
    | uint (value : Nat)
    | float (canonical : String)
    | string (value : String)
    | duration (nanoseconds : Int)
    | list (items : CanonicalValues)
    | object (fields : CanonicalFields)

  inductive CanonicalValues where
    | nil
    | cons (value : CanonicalValue) (tail : CanonicalValues)

  inductive CanonicalFields where
    | nil
    | cons (name : String) (value : CanonicalValue) (tail : CanonicalFields)
end

deriving instance DecidableEq for CanonicalValue, CanonicalValues, CanonicalFields
deriving instance Repr for CanonicalValue, CanonicalValues, CanonicalFields

structure ExactConstraints where
  namespaceName : Option String
  namespaceId : Option String
  taskQueueName : Option String
  destination : Option String
  chasmTaskType : Option String
  taskQueueType : Option Int
  shardId : Option Int
  taskType : Option Int
  deriving DecidableEq, Repr

structure OpaqueDefault where
  goType : String
  reason : String
  deriving DecidableEq, Repr

inductive DefaultLeaf where
  | concrete (value : CanonicalValue)
  | opaque (metadata : OpaqueDefault)
  deriving DecidableEq, Repr

structure ConstrainedDefault where
  constraints : ExactConstraints
  value : DefaultLeaf
  deriving DecidableEq, Repr

inductive SettingDefault where
  | concrete (value : CanonicalValue)
  | constrained (values : List ConstrainedDefault)
  | opaque (metadata : OpaqueDefault)
  deriving DecidableEq, Repr

structure Provenance where
  packageName : String
  file : String
  line : Nat
  deriving DecidableEq, Repr

structure Setting where
  key : String
  description : String
  policy : PrecedencePolicy
  schema : ValueSchema
  codec : CodecClass
  defaultValue : SettingDefault
  provenance : List Provenance
  identity : String
  deriving DecidableEq, Repr

structure FixtureOverride where
  constraints : ExactConstraints
  value : CanonicalValue
  deriving DecidableEq, Repr

inductive FixtureSource where
  | override
  | constrainedDefault
  | simpleDefault
  deriving DecidableEq, Repr

structure ResolutionFixture where
  name : String
  policy : PrecedencePolicy
  settingKey : String
  context : ExactConstraints
  overrides : List FixtureOverride
  selectedSource : FixtureSource
  selectedConstraint : ExactConstraints
  result : CanonicalValue
  deriving DecidableEq, Repr

end Temporal.DynamicConfig
`)
	return []byte(generated.String())
}

func renderDynamicConfigSettings(
	catalogIdentity string,
	settings []ProjectedSetting,
	fixtures []ResolutionFixture,
) ([]byte, error) {
	if catalogIdentity == "" {
		return nil, errors.New("catalog identity is required")
	}
	var generated strings.Builder
	writeDynamicConfigHeader(&generated)
	generated.WriteString("import Temporal.DynamicConfig.Types\n")
	writeDynamicConfigModuleDoc(&generated, dynamicConfigSettingsDoc)
	generated.WriteString("set_option linter.missingDocs false\n")
	generated.WriteString("set_option maxRecDepth 100000\n\n")
	generated.WriteString("namespace Temporal.DynamicConfig.Settings\n\n")
	for _, setting := range settings {
		if err := renderSetting(&generated, setting); err != nil {
			return nil, fmt.Errorf("render setting %q: %w", setting.Key, err)
		}
	}
	fmt.Fprintf(&generated, "def all : List Setting :=\n  [%s]\n\n", renderSettingReferences(settings))
	fmt.Fprintf(&generated, "def catalogIdentity : String := %s\n\n", leanString(catalogIdentity))
	generated.WriteString("def fixtures : List ResolutionFixture :=\n  [")
	for index, fixture := range fixtures {
		if index > 0 {
			generated.WriteString(",\n   ")
		}
		rendered, err := renderFixture(fixture)
		if err != nil {
			return nil, fmt.Errorf("render fixture %q: %w", fixture.Name, err)
		}
		generated.WriteString(rendered)
	}
	generated.WriteString("]\n\nend Temporal.DynamicConfig.Settings\n")
	return []byte(generated.String()), nil
}

func renderSetting(generated *strings.Builder, setting ProjectedSetting) error {
	identifier, err := settingIdentifier(setting.Key)
	if err != nil {
		return err
	}
	policy, err := renderPolicy(setting.Policy)
	if err != nil {
		return err
	}
	schema, err := renderSchema(setting.Schema)
	if err != nil {
		return err
	}
	codec, err := renderCodec(setting.Codec)
	if err != nil {
		return err
	}
	defaultValue, err := renderDefault(setting.Default)
	if err != nil {
		return err
	}
	provenance := slices.Clone(setting.Provenance)
	slices.SortFunc(provenance, compareRegistrationSites)
	renderedProvenance := make([]string, len(provenance))
	for index, site := range provenance {
		if site.Line < 0 {
			return errors.New("provenance line cannot be negative")
		}
		renderedProvenance[index] = fmt.Sprintf(
			"{ packageName := %s, file := %s, line := %d }",
			leanString(site.Package),
			leanString(site.File),
			site.Line,
		)
	}
	fmt.Fprintf(generated, "def %s : Setting :=\n", identifier)
	fmt.Fprintf(generated, "  { key := %s\n", leanString(setting.Key))
	fmt.Fprintf(generated, "    description := %s\n", leanString(setting.Description))
	fmt.Fprintf(generated, "    policy := %s\n", policy)
	fmt.Fprintf(generated, "    schema := %s\n", schema)
	fmt.Fprintf(generated, "    codec := %s\n", codec)
	fmt.Fprintf(generated, "    defaultValue := %s\n", defaultValue)
	fmt.Fprintf(generated, "    provenance := [%s]\n", strings.Join(renderedProvenance, ", "))
	fmt.Fprintf(generated, "    identity := %s }\n\n", leanString(setting.Identity))
	return nil
}

func renderSettingReferences(settings []ProjectedSetting) string {
	references := make([]string, len(settings))
	for index, setting := range settings {
		references[index], _ = settingIdentifier(setting.Key)
	}
	return strings.Join(references, ",\n   ")
}

func renderFixture(fixture ResolutionFixture) (string, error) {
	policy, err := renderPolicy(fixture.Policy)
	if err != nil {
		return "", err
	}
	overrides := slices.Clone(fixture.Overrides)
	slices.SortFunc(overrides, func(left, right FixtureOverride) int {
		return strings.Compare(constraintSortKey(left.Constraints), constraintSortKey(right.Constraints))
	})
	renderedOverrides := make([]string, len(overrides))
	for index, override := range overrides {
		value, err := renderCanonicalValue(override.Value)
		if err != nil {
			return "", err
		}
		renderedOverrides[index] = fmt.Sprintf(
			"{ constraints := %s, value := %s }",
			renderConstraints(override.Constraints),
			value,
		)
	}
	source, err := renderFixtureSource(fixture.SelectedSource)
	if err != nil {
		return "", err
	}
	result, err := renderCanonicalValue(fixture.Result)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(
		"{ name := %s, policy := %s, settingKey := %s, context := %s, overrides := [%s], selectedSource := %s, selectedConstraint := %s, result := %s }",
		leanString(fixture.Name),
		policy,
		leanString(fixture.SettingKey),
		renderConstraints(fixture.Context),
		strings.Join(renderedOverrides, ", "),
		source,
		renderConstraints(fixture.SelectedConstraint),
		result,
	), nil
}

func renderPolicy(policy PrecedencePolicy) (string, error) {
	values := map[PrecedencePolicy]string{
		PolicyGlobal:        ".global",
		PolicyNamespace:     ".namespace",
		PolicyNamespaceID:   ".namespaceId",
		PolicyTaskQueue:     ".taskQueue",
		PolicyShardID:       ".shardId",
		PolicyTaskType:      ".taskType",
		PolicyDestination:   ".destination",
		PolicyChasmTaskType: ".chasmTaskType",
	}
	rendered, exists := values[policy]
	if !exists {
		return "", fmt.Errorf("unknown precedence policy %q", policy)
	}
	return rendered, nil
}

func renderCodec(codec CodecClass) (string, error) {
	values := map[CodecClass]string{
		"bool":      ".bool",
		"int":       ".int",
		"float":     ".float",
		"string":    ".string",
		"duration":  ".duration",
		"map":       ".map",
		"structure": ".structured",
		"custom":    ".custom",
	}
	rendered, exists := values[codec]
	if !exists {
		return "", fmt.Errorf("unknown codec class %q", codec)
	}
	return rendered, nil
}

func renderSchema(schema ValueSchema) (string, error) {
	base := fmt.Sprintf("%s %t", leanString(schema.GoType), schema.Nullable)
	switch schema.Kind {
	case SchemaBool:
		return ".bool " + base, nil
	case SchemaInt:
		return ".int " + base, nil
	case SchemaUint:
		return ".uint " + base, nil
	case SchemaFloat:
		return ".float " + base, nil
	case SchemaString:
		return ".string " + base, nil
	case SchemaDuration:
		return ".duration " + base, nil
	case SchemaDynamic:
		return ".dynamicValue " + base, nil
	case SchemaReference:
		return ".reference " + base, nil
	case SchemaOpaque:
		return ".opaque " + base, nil
	case SchemaList:
		if schema.Length < 0 || schema.Element == nil {
			return "", errors.New("list schema has invalid length or missing element")
		}
		element, err := renderSchema(*schema.Element)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(".list %s %d (%s) %t", leanString(schema.GoType), schema.Length, element, schema.Nullable), nil
	case SchemaMap:
		if schema.Element == nil {
			return "", errors.New("map schema has missing element")
		}
		element, err := renderSchema(*schema.Element)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf(".map %s (%s) %t", leanString(schema.GoType), element, schema.Nullable), nil
	case SchemaStruct:
		fields := slices.Clone(schema.Fields)
		slices.SortFunc(fields, func(left, right SchemaField) int {
			return strings.Compare(left.Name, right.Name)
		})
		renderedFields := make([]string, len(fields))
		for index := len(fields) - 1; index >= 0; index-- {
			field := fields[index]
			rendered, err := renderSchema(field.Schema)
			if err != nil {
				return "", fmt.Errorf("field %q: %w", field.Name, err)
			}
			renderedFields[index] = fmt.Sprintf(".cons %s (%s) (%s)", leanString(field.Name), rendered, renderedTail(renderedFields, index))
		}
		if len(renderedFields) == 0 {
			return fmt.Sprintf(".struct %s .nil %t", leanString(schema.GoType), schema.Nullable), nil
		}
		return fmt.Sprintf(".struct %s (%s) %t", leanString(schema.GoType), renderedFields[0], schema.Nullable), nil
	default:
		return "", fmt.Errorf("unknown schema kind %q", schema.Kind)
	}
}

func renderDefault(value ProjectedDefault) (string, error) {
	switch value.Kind {
	case DefaultConcrete:
		if value.Value == nil {
			return "", errors.New("concrete default has no value")
		}
		rendered, err := renderCanonicalValue(*value.Value)
		if err != nil {
			return "", err
		}
		return ".concrete (" + rendered + ")", nil
	case DefaultOpaque:
		if value.Opaque == nil {
			return "", errors.New("opaque default has no metadata")
		}
		return ".opaque " + renderOpaque(*value.Opaque), nil
	case DefaultConstrained:
		if len(value.Constrained) == 0 {
			return "", errors.New("constrained default has no values")
		}
		constrained := slices.Clone(value.Constrained)
		slices.SortFunc(constrained, func(left, right ProjectedConstrainedDefault) int {
			return strings.Compare(constraintSortKey(left.Constraints), constraintSortKey(right.Constraints))
		})
		rendered := make([]string, len(constrained))
		for index, item := range constrained {
			leaf := ""
			if item.Opaque != nil {
				leaf = ".opaque " + renderOpaque(*item.Opaque)
			} else {
				encoded, err := renderCanonicalValue(item.Value)
				if err != nil {
					return "", err
				}
				leaf = ".concrete (" + encoded + ")"
			}
			rendered[index] = fmt.Sprintf(
				"{ constraints := %s, value := %s }",
				renderConstraints(item.Constraints),
				leaf,
			)
		}
		return ".constrained [" + strings.Join(rendered, ", ") + "]", nil
	default:
		return "", fmt.Errorf("unknown default kind %q", value.Kind)
	}
}

func renderOpaque(value ProjectedOpaqueDefault) string {
	return fmt.Sprintf("{ goType := %s, reason := %s }", leanString(value.GoType), leanString(value.Reason))
}

func renderCanonicalValue(value CanonicalValue) (string, error) {
	switch value.Kind {
	case ValueNull:
		return ".null", nil
	case ValueBool:
		if value.Scalar != "true" && value.Scalar != "false" {
			return "", fmt.Errorf("invalid bool %q", value.Scalar)
		}
		parsed, _ := strconv.ParseBool(value.Scalar)
		return fmt.Sprintf(".bool %t", parsed), nil
	case ValueInt:
		parsed, err := strconv.ParseInt(value.Scalar, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid int %q", value.Scalar)
		}
		return ".int " + leanSignedLiteral(strconv.FormatInt(parsed, 10)), nil
	case ValueUint:
		if _, err := strconv.ParseUint(value.Scalar, 10, 64); err != nil {
			return "", fmt.Errorf("invalid uint %q", value.Scalar)
		}
		return ".uint " + value.Scalar, nil
	case ValueFloat:
		if _, err := strconv.ParseFloat(value.Scalar, 64); err != nil {
			return "", fmt.Errorf("invalid float %q", value.Scalar)
		}
		return ".float " + leanString(value.Scalar), nil
	case ValueString:
		return ".string " + leanString(value.Scalar), nil
	case ValueDuration:
		parsed, err := strconv.ParseInt(value.Scalar, 10, 64)
		if err != nil {
			return "", fmt.Errorf("invalid duration %q", value.Scalar)
		}
		return ".duration " + leanSignedLiteral(strconv.FormatInt(parsed, 10)), nil
	case ValueList:
		return renderCanonicalList(value.Items)
	case ValueObject:
		return renderCanonicalObject(value.Fields)
	default:
		return "", fmt.Errorf("unknown canonical value kind %q", value.Kind)
	}
}

func renderCanonicalList(values []CanonicalValue) (string, error) {
	items := make([]string, len(values))
	for index := len(values) - 1; index >= 0; index-- {
		rendered, err := renderCanonicalValue(values[index])
		if err != nil {
			return "", fmt.Errorf("item %d: %w", index, err)
		}
		items[index] = fmt.Sprintf(".cons (%s) (%s)", rendered, renderedTail(items, index))
	}
	if len(items) == 0 {
		return ".list .nil", nil
	}
	return ".list (" + items[0] + ")", nil
}

func renderCanonicalObject(values []CanonicalField) (string, error) {
	fields := slices.Clone(values)
	slices.SortFunc(fields, func(left, right CanonicalField) int {
		return strings.Compare(left.Name, right.Name)
	})
	renderedFields := make([]string, len(fields))
	for index := len(fields) - 1; index >= 0; index-- {
		field := fields[index]
		rendered, err := renderCanonicalValue(field.Value)
		if err != nil {
			return "", fmt.Errorf("field %q: %w", field.Name, err)
		}
		renderedFields[index] = fmt.Sprintf(
			".cons %s (%s) (%s)",
			leanString(field.Name),
			rendered,
			renderedTail(renderedFields, index),
		)
	}
	if len(renderedFields) == 0 {
		return ".object .nil", nil
	}
	return ".object (" + renderedFields[0] + ")", nil
}

func renderConstraints(value ExactConstraints) string {
	return fmt.Sprintf(
		"{ namespaceName := %s, namespaceId := %s, taskQueueName := %s, destination := %s, chasmTaskType := %s, taskQueueType := %s, shardId := %s, taskType := %s }",
		renderOptionalString(value.Namespace),
		renderOptionalString(value.NamespaceID),
		renderOptionalString(value.TaskQueueName),
		renderOptionalString(value.Destination),
		renderOptionalString(value.ChasmTaskType),
		renderOptionalInt(value.TaskQueueType),
		renderOptionalInt(value.ShardID),
		renderOptionalInt(value.TaskType),
	)
}

func renderOptionalString(value *string) string {
	if value == nil {
		return "none"
	}
	return "some " + leanString(*value)
}

func renderOptionalInt(value *int32) string {
	if value == nil {
		return "none"
	}
	return "some " + leanSignedLiteral(strconv.FormatInt(int64(*value), 10))
}

func renderFixtureSource(source FixtureSource) (string, error) {
	values := map[FixtureSource]string{
		SourceOverride:           ".override",
		SourceConstrainedDefault: ".constrainedDefault",
		SourceSimpleDefault:      ".simpleDefault",
	}
	rendered, exists := values[source]
	if !exists {
		return "", fmt.Errorf("unknown fixture source %q", source)
	}
	return rendered, nil
}

func settingIdentifier(key string) (string, error) {
	var identifier strings.Builder
	lastUnderscore := false
	for _, character := range key {
		valid := character == '_' || unicode.IsLetter(character) || unicode.IsDigit(character)
		if !valid {
			if !lastUnderscore {
				identifier.WriteByte('_')
				lastUnderscore = true
			}
			continue
		}
		identifier.WriteRune(character)
		lastUnderscore = character == '_'
	}
	rendered := strings.Trim(identifier.String(), "_")
	if rendered == "" {
		return "", fmt.Errorf("setting key %q has no Lean identifier", key)
	}
	first := []rune(rendered)[0]
	if unicode.IsDigit(first) {
		rendered = "setting_" + rendered
	}
	if slices.Contains([]string{"def", "end", "import", "namespace", "open", "structure"}, rendered) {
		rendered = "setting_" + rendered
	}
	return rendered, nil
}

func leanString(value string) string {
	var rendered strings.Builder
	rendered.Grow(len(value) + 2)
	rendered.WriteByte('"')
	for _, character := range value {
		switch character {
		case '"':
			rendered.WriteString(`\"`)
		case '\\':
			rendered.WriteString(`\\`)
		case '\n':
			rendered.WriteString(`\n`)
		case '\r':
			rendered.WriteString(`\r`)
		case '\t':
			rendered.WriteString(`\t`)
		default:
			if character < 0x20 || character == 0x7f {
				fmt.Fprintf(&rendered, `\u%04x`, character)
			} else {
				rendered.WriteRune(character)
			}
		}
	}
	rendered.WriteByte('"')
	return rendered.String()
}

func leanSignedLiteral(value string) string {
	if strings.HasPrefix(value, "-") {
		return "(" + value + ")"
	}
	return value
}

func writeDynamicConfigHeader(generated *strings.Builder) {
	generated.WriteString("-- Code generated by umpire-gen-lean-dynamic-config-catalog. DO NOT EDIT.\n")
	generated.WriteString("-- This is a structural registry projection, not handwritten product semantics.\n")
}

func writeDynamicConfigModuleDoc(generated *strings.Builder, moduleDoc string) {
	fmt.Fprintf(generated, "\n%s\n\n", moduleDoc)
}

func renderedTail(rendered []string, index int) string {
	if index+1 == len(rendered) {
		return ".nil"
	}
	return rendered[index+1]
}
