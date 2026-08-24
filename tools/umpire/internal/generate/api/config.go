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
)

type descriptorSpec struct {
	Locator string
	Path    string
}

type outputLayout struct {
	RootModule   string
	APIPath      string
	APIDirectory string
	ProtoPath    string
	TypesPath    string
}

type generationConfig struct {
	Descriptors []descriptorSpec
	OutputRoot  string
	Layout      outputLayout
}

type descriptorValues []descriptorSpec

func (values *descriptorValues) String() string {
	locators := make([]string, len(*values))
	for index, value := range *values {
		locators[index] = value.Locator
	}
	return strings.Join(locators, ",")
}

func (values *descriptorValues) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("descriptor path is required")
	}
	locator := filepath.ToSlash(filepath.Clean(value))
	*values = append(*values, descriptorSpec{Locator: locator, Path: filepath.FromSlash(locator)})
	return nil
}

func parseGenerationConfig(arguments []string) (generationConfig, error) {
	var descriptors descriptorValues
	flags := flag.NewFlagSet("umpire-gen-api", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Var(&descriptors, "descriptor", "descriptor set path (repeatable)")
	leanRoot := flags.String("lean-root", "", "root Lean module")
	outputRoot := flags.String("output-root", "", "generated artifact root")
	if err := flags.Parse(arguments); err != nil {
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
		if seenDescriptors[descriptor.Locator] {
			return generationConfig{}, fmt.Errorf("duplicate descriptor locator %q", descriptor.Locator)
		}
		seenDescriptors[descriptor.Locator] = true
	}
	if *leanRoot == "" {
		return generationConfig{}, errors.New("--lean-root is required")
	}
	if err := validateLeanRoot(*leanRoot); err != nil {
		return generationConfig{}, err
	}
	if *outputRoot == "" {
		return generationConfig{}, errors.New("--output-root is required")
	}

	configuration := generationConfig{
		Descriptors: append([]descriptorSpec(nil), descriptors...),
		OutputRoot:  *outputRoot,
		Layout:      newOutputLayout(*leanRoot),
	}
	slices.SortFunc(configuration.Descriptors, func(left, right descriptorSpec) int {
		return strings.Compare(left.Locator, right.Locator)
	})
	return configuration, nil
}

func newOutputLayout(root string) outputLayout {
	rootPath := strings.ReplaceAll(root, ".", "/")
	apiDirectory := rootPath + "/API"
	return outputLayout{
		RootModule:   root,
		APIPath:      apiDirectory + ".lean",
		APIDirectory: apiDirectory,
		ProtoPath:    apiDirectory + "/Proto.lean",
		TypesPath:    apiDirectory + "/Types.lean",
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
