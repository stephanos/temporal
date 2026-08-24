package api

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"go.temporal.io/server/tools/umpire/internal/protofile"
)

type sourceGroup string

type descriptorSpec struct {
	Name    string
	Locator string
	Path    string
}

type sourceRule struct {
	Group  sourceGroup `json:"group"`
	Prefix string      `json:"prefix"`
}

type outputLayout struct {
	RootModule    string
	CorePath      string
	UmbrellaPath  string
	GeneratedPath string
	TypesPath     string
	SchemaPath    string
	ManifestPath  string
}

type generationConfig struct {
	Operation     string
	Descriptors   []descriptorSpec
	Sources       []sourceRule
	Groups        []sourceGroup
	DefaultSource sourceGroup
	OutputRoot    string
	Layout        outputLayout
}

func (configuration generationConfig) Classify(filePath string) sourceGroup {
	for _, rule := range configuration.Sources {
		if strings.HasPrefix(filePath, rule.Prefix) {
			return rule.Group
		}
	}
	return configuration.DefaultSource
}

type descriptorValues []descriptorSpec

func (values *descriptorValues) String() string {
	parts := make([]string, len(*values))
	for index, value := range *values {
		parts[index] = value.Name + "=" + value.Locator
	}
	return strings.Join(parts, ",")
}

func (values *descriptorValues) Set(value string) error {
	name, locator, found := strings.Cut(value, "=")
	if !found {
		return errors.New("descriptor must have the form NAME=PATH")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("descriptor name is required")
	}
	if locator == "" {
		return errors.New("descriptor path is required")
	}
	locator = filepath.ToSlash(filepath.Clean(locator))
	*values = append(*values, descriptorSpec{Name: name, Locator: locator, Path: filepath.FromSlash(locator)})
	return nil
}

type sourceValues []sourceRule

func (values *sourceValues) String() string {
	parts := make([]string, len(*values))
	for index, value := range *values {
		parts[index] = string(value.Group) + "=" + value.Prefix
	}
	return strings.Join(parts, ",")
}

func (values *sourceValues) Set(value string) error {
	group, prefix, found := strings.Cut(value, "=")
	if !found {
		return errors.New("source must have the form GROUP=PREFIX")
	}
	group = strings.TrimSpace(group)
	if err := validateModuleSegment(group); err != nil {
		return fmt.Errorf("invalid source group %q: %w", group, err)
	}
	normalized, err := protofile.NormalizePrefix(prefix)
	if err != nil {
		return fmt.Errorf("invalid source prefix %q: %w", prefix, err)
	}
	*values = append(*values, sourceRule{Group: sourceGroup(group), Prefix: normalized})
	return nil
}

func parseGenerationConfig(arguments []string) (generationConfig, error) {
	if len(arguments) == 0 || strings.HasPrefix(arguments[0], "-") {
		return generationConfig{}, errors.New("operation is required: generate, check, or inspect")
	}
	configuration := generationConfig{Operation: arguments[0]}
	switch configuration.Operation {
	case "generate", "check", "inspect":
	default:
		return generationConfig{}, fmt.Errorf("unknown operation %q", configuration.Operation)
	}

	var descriptors descriptorValues
	var sources sourceValues
	flags := flag.NewFlagSet("umpire-gen-api "+configuration.Operation, flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Var(&descriptors, "descriptor", "named descriptor set NAME=PATH (repeatable)")
	flags.Var(&sources, "source", "source classification GROUP=PREFIX (repeatable)")
	defaultSource := flags.String("default-source", "", "source group for unmatched protobuf files")
	leanRoot := flags.String("lean-root", "", "root Lean module")
	outputRoot := flags.String("output-root", "", "generated artifact root")
	if err := flags.Parse(arguments[1:]); err != nil {
		return generationConfig{}, err
	}
	if flags.NArg() != 0 {
		return generationConfig{}, errors.New("unexpected positional arguments")
	}
	if len(descriptors) == 0 {
		return generationConfig{}, errors.New("at least one --descriptor is required")
	}
	seenDescriptors := make(map[string]bool, len(descriptors))
	for _, descriptor := range descriptors {
		if seenDescriptors[descriptor.Name] {
			return generationConfig{}, fmt.Errorf("duplicate descriptor name %q", descriptor.Name)
		}
		seenDescriptors[descriptor.Name] = true
	}
	if *defaultSource == "" {
		return generationConfig{}, errors.New("--default-source is required")
	}
	if err := validateModuleSegment(*defaultSource); err != nil {
		return generationConfig{}, fmt.Errorf("invalid default source %q: %w", *defaultSource, err)
	}
	if *leanRoot == "" {
		return generationConfig{}, errors.New("--lean-root is required")
	}
	if err := validateLeanRoot(*leanRoot); err != nil {
		return generationConfig{}, err
	}
	if configuration.Operation != "inspect" && *outputRoot == "" {
		return generationConfig{}, fmt.Errorf("--output-root is required for %s", configuration.Operation)
	}

	seenPrefixes := make(map[string]sourceGroup, len(sources))
	groups := map[sourceGroup]bool{sourceGroup(*defaultSource): true}
	for _, rule := range sources {
		if previous, exists := seenPrefixes[rule.Prefix]; exists {
			if previous == rule.Group {
				return generationConfig{}, fmt.Errorf("duplicate source rule %s=%s", rule.Group, rule.Prefix)
			}
			return generationConfig{}, fmt.Errorf("source prefix %q is assigned to both %q and %q", rule.Prefix, previous, rule.Group)
		}
		seenPrefixes[rule.Prefix] = rule.Group
		groups[rule.Group] = true
	}

	configuration.Descriptors = append([]descriptorSpec(nil), descriptors...)
	slices.SortFunc(configuration.Descriptors, func(left, right descriptorSpec) int {
		if order := strings.Compare(left.Name, right.Name); order != 0 {
			return order
		}
		return strings.Compare(left.Locator, right.Locator)
	})
	configuration.Sources = append([]sourceRule(nil), sources...)
	slices.SortFunc(configuration.Sources, func(left, right sourceRule) int {
		if order := len(right.Prefix) - len(left.Prefix); order != 0 {
			return order
		}
		if order := strings.Compare(string(left.Group), string(right.Group)); order != 0 {
			return order
		}
		return strings.Compare(left.Prefix, right.Prefix)
	})
	for group := range groups {
		configuration.Groups = append(configuration.Groups, group)
	}
	slices.SortFunc(configuration.Groups, func(left, right sourceGroup) int {
		return strings.Compare(string(left), string(right))
	})
	configuration.DefaultSource = sourceGroup(*defaultSource)
	configuration.OutputRoot = *outputRoot
	configuration.Layout = newOutputLayout(*leanRoot)
	return configuration, nil
}

func newOutputLayout(root string) outputLayout {
	rootPath := strings.ReplaceAll(root, ".", "/")
	generatedPath := rootPath + "/Generated"
	return outputLayout{
		RootModule: root,
		CorePath:   rootPath + "/Proto/Core.lean", UmbrellaPath: generatedPath + ".lean",
		GeneratedPath: generatedPath, TypesPath: generatedPath + "/Types.lean",
		SchemaPath: generatedPath + "/schema.json", ManifestPath: generatedPath + "/manifest.json",
	}
}

func validateLeanRoot(value string) error {
	parts := strings.Split(value, ".")
	if parts[0] == "_" || parts[0] == "Type" || parts[0] == "Prop" || parts[0] == "Sort" {
		return fmt.Errorf("invalid Lean root %q: first module segment %q is reserved", value, parts[0])
	}
	for _, part := range parts {
		if err := validateModuleSegment(part); err != nil {
			return fmt.Errorf("invalid Lean root %q: %w", value, err)
		}
	}
	return nil
}

func validateModuleSegment(value string) error {
	if value == "" {
		return errors.New("module segment is empty")
	}
	for index, character := range value {
		if index == 0 {
			if character != '_' && !unicode.IsLetter(character) {
				return errors.New("module segment must begin with a letter or underscore")
			}
			continue
		}
		if character != '_' && character != '\'' && !unicode.IsLetter(character) && !unicode.IsNumber(character) {
			return fmt.Errorf("module segment contains invalid character %q", character)
		}
	}
	if leanReserved[value] {
		return fmt.Errorf("module segment %q is reserved", value)
	}
	return nil
}
