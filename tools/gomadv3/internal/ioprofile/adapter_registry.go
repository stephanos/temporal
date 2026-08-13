package ioprofile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/record"
	"go.temporal.io/server/tools/gomadv3/internal/target"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

type BuildAdapter struct {
	Module            string
	Version           string
	Sum               string
	BuildModFile      string
	Source            string
	Replacement       string
	SourceSHA256      string
	ReplacementSHA256 string
}

type InvalidBuildAdapterConfigurationError struct {
	Err error
}

func (err *InvalidBuildAdapterConfigurationError) Error() string {
	return err.Err.Error()
}

func (err *InvalidBuildAdapterConfigurationError) Unwrap() error {
	return err.Err
}

func IsInvalidBuildAdapterConfiguration(err error) bool {
	var invalid *InvalidBuildAdapterConfigurationError
	return errors.As(err, &invalid)
}

func invalidBuildAdapterConfiguration(err error) error {
	return &InvalidBuildAdapterConfigurationError{Err: err}
}

type adapterPreparation struct {
	replacement string
	evidence    BuildAdapter
}

type adapterImplementation struct {
	module    string
	inventory inventoryEntry
	prepare   func(string, string, gomadversion.AdapterIdentity) (adapterPreparation, error)
}

type adapterDefinition struct {
	identity       gomadversion.AdapterIdentity
	implementation adapterImplementation
}

type adapterRegistry struct {
	definitions []adapterDefinition
}

func newAdapterRegistry(identities []gomadversion.AdapterIdentity, implementations []adapterImplementation) (adapterRegistry, error) {
	byModule := make(map[string]adapterImplementation, len(implementations))
	for _, implementation := range implementations {
		if implementation.module == "" {
			return adapterRegistry{}, errors.New("adapter implementation has no module identity")
		}
		if _, found := byModule[implementation.module]; found {
			return adapterRegistry{}, fmt.Errorf("adapter implementation is duplicated: %s", implementation.module)
		}
		byModule[implementation.module] = implementation
	}
	definitions := make([]adapterDefinition, len(identities))
	for index, identity := range identities {
		implementation, found := byModule[identity.Module]
		if !found {
			return adapterRegistry{}, fmt.Errorf("adapter %s has no built-in implementation", identity.Module)
		}
		definitions[index] = adapterDefinition{identity: identity, implementation: implementation}
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].identity.Module < definitions[j].identity.Module })
	return adapterRegistry{definitions: definitions}, nil
}

func (registry adapterRegistry) inventory() []inventoryEntry {
	entries := make([]inventoryEntry, len(registry.definitions))
	for index, definition := range registry.definitions {
		entries[index] = definition.implementation.inventory
	}
	return entries
}

func (profile ProfileSpec) Adapters() []gomadversion.AdapterIdentity {
	if profile.definition == nil {
		return nil
	}
	identities := make([]gomadversion.AdapterIdentity, len(profile.definition.adapters.definitions))
	for index, definition := range profile.definition.adapters.definitions {
		identities[index] = definition.identity
	}
	return identities
}

func (profile ProfileSpec) VerifyAdapters(adapters []record.TargetAdapter) error {
	definition, err := profile.validated()
	if err != nil {
		return err
	}
	if adapters == nil {
		return errors.New("selected adapter identities are missing")
	}
	available := make(map[string]gomadversion.AdapterIdentity, len(definition.adapters.definitions))
	for _, adapter := range definition.adapters.definitions {
		available[adapter.identity.Module] = adapter.identity
	}
	for index, adapter := range adapters {
		if index > 0 && adapters[index-1].Module >= adapter.Module {
			return errors.New("selected adapter identities are not sorted and unique")
		}
		identity, found := available[adapter.Module]
		if !found || identity.Version != adapter.Version || identity.Sum != adapter.Sum {
			return fmt.Errorf("selected adapter %s is unavailable or modified", adapter.Module)
		}
	}
	return nil
}

func RecordAdapters(adapters []BuildAdapter) []record.TargetAdapter {
	result := make([]record.TargetAdapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = record.TargetAdapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

func (registry adapterRegistry) prepare(spec target.Spec, moduleCache string) (target.Spec, []BuildAdapter, error) {
	if len(registry.definitions) == 0 {
		return spec, []BuildAdapter{}, nil
	}
	workingDirectory, err := filepath.Abs(spec.WorkingDir)
	if err != nil {
		return target.Spec{}, nil, fmt.Errorf("resolve target working directory: %w", err)
	}
	moduleFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.mod"))
	if err != nil {
		return target.Spec{}, nil, invalidBuildAdapterConfiguration(fmt.Errorf("read target module file: %w", err))
	}
	selected := make([]adapterDefinition, 0, len(registry.definitions))
	for _, definition := range registry.definitions {
		version, detectErr := detectModuleVersion(moduleFile, definition.identity.Module)
		if detectErr != nil {
			return target.Spec{}, nil, invalidBuildAdapterConfiguration(detectErr)
		}
		if version == "" {
			continue
		}
		if version != definition.identity.Version {
			return target.Spec{}, nil, invalidBuildAdapterConfiguration(fmt.Errorf("unsupported %s version %q", definition.identity.Module, version))
		}
		selected = append(selected, definition)
	}
	if len(selected) == 0 {
		return spec, []BuildAdapter{}, nil
	}
	if moduleCache == "" || spec.PreparationRoot == "" {
		return target.Spec{}, nil, errors.New("deterministic I/O build adapter requires module cache and preparation root")
	}
	if spec.BuildModFile != "" {
		return target.Spec{}, nil, invalidBuildAdapterConfiguration(errors.New("deterministic I/O build adapters cannot replace an existing build modfile"))
	}
	preparationRoot, err := filepath.Abs(spec.PreparationRoot)
	if err != nil {
		return target.Spec{}, nil, fmt.Errorf("resolve deterministic I/O preparation root: %w", err)
	}
	root := filepath.Join(preparationRoot, ".io-adapter")
	if err := os.Mkdir(root, 0o700); err != nil {
		return target.Spec{}, nil, fmt.Errorf("create deterministic I/O adapter directory: %w", err)
	}
	evidence := make([]BuildAdapter, 0, len(selected))
	for _, definition := range selected {
		prepared, prepareErr := definition.implementation.prepare(moduleCache, root, definition.identity)
		if prepareErr != nil {
			return target.Spec{}, nil, prepareErr
		}
		moduleFile = append(moduleFile, []byte("\nreplace "+definition.identity.Module+" => "+prepared.replacement+"\n")...)
		evidence = append(evidence, prepared.evidence)
	}
	modFilePath := filepath.Join(root, "gomad.mod")
	if err := writeExclusive(modFilePath, moduleFile); err != nil {
		return target.Spec{}, nil, err
	}
	sumFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.sum"))
	if err != nil {
		return target.Spec{}, nil, invalidBuildAdapterConfiguration(fmt.Errorf("read target module sums: %w", err))
	}
	if err := writeExclusive(filepath.Join(root, "gomad.sum"), sumFile); err != nil {
		return target.Spec{}, nil, err
	}
	for index := range evidence {
		evidence[index].BuildModFile = modFilePath
	}
	spec.BuildModFile = modFilePath
	return spec, evidence, nil
}

func (profile ProfileSpec) PrepareBuildAdapters(spec target.Spec, moduleCache string) (target.Spec, []BuildAdapter, error) {
	definition, err := profile.validated()
	if err != nil {
		return target.Spec{}, nil, err
	}
	return definition.adapters.prepare(spec, moduleCache)
}

func detectModuleVersion(contents []byte, module string) (string, error) {
	inRequireBlock := false
	for _, line := range strings.Split(string(contents), "\n") {
		line = strings.TrimSpace(strings.SplitN(line, "//", 2)[0])
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		if fields[0] == "replace" && len(fields) > 1 && fields[1] == module {
			return "", fmt.Errorf("target module already replaces %s", module)
		}
		if fields[0] == "require" {
			if len(fields) == 2 && fields[1] == "(" {
				inRequireBlock = true
				continue
			}
			if len(fields) >= 3 && fields[1] == module {
				return fields[2], nil
			}
		}
		if inRequireBlock {
			if fields[0] == ")" {
				inRequireBlock = false
				continue
			}
			if len(fields) >= 2 && fields[0] == module {
				return fields[1], nil
			}
		}
	}
	return "", nil
}

func mustAdapterRegistry(identities []gomadversion.AdapterIdentity, implementations []adapterImplementation) adapterRegistry {
	registry, err := newAdapterRegistry(identities, implementations)
	if err != nil {
		panic(err) //nolint:forbidigo // A generated adapter without an implementation makes the package unusable.
	}
	return registry
}
