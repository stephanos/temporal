package deterministicio

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
	"golang.org/x/mod/modfile"
)

type BuildAdapter struct {
	Module                           string
	Version                          string
	Sum                              string
	BuildModFile                     string
	Source                           string
	ReplacementRoot                  string
	Replacement                      string
	PreparedPackage                  string
	SourceSHA256                     string
	ReplacementSHA256                string
	OriginalSourceInventorySHA256    string
	ReplacementSourceInventorySHA256 string
	PreparedSourceSetSHA256          string
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
	// rewritten is the anchored rewrite an adapter prepared by
	// prepareRewrittenModule declares, which regeneration re-derives.
	rewritten *rewrittenModule
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

func (profile Spec) Adapters() []gomadversion.AdapterIdentity {
	if profile.definition == nil {
		return nil
	}
	identities := make([]gomadversion.AdapterIdentity, len(profile.definition.adapters.definitions))
	for index, definition := range profile.definition.adapters.definitions {
		identities[index] = definition.identity
	}
	return identities
}

func (profile Spec) VerifyAdapters(adapters []Adapter) error {
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

func SelectedAdapters(adapters []BuildAdapter) []Adapter {
	result := make([]Adapter, len(adapters))
	for index, adapter := range adapters {
		result[index] = Adapter{Module: adapter.Module, Version: adapter.Version, Sum: adapter.Sum}
	}
	return result
}

// selected returns the adapters whose modules the target module at
// workingDirectory requires, with the target's module file.
func (registry adapterRegistry) selected(workingDirectory string) ([]adapterDefinition, []byte, error) {
	if len(registry.definitions) == 0 {
		return nil, nil, nil
	}
	moduleFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.mod"))
	if err != nil {
		return nil, nil, invalidBuildAdapterConfiguration(fmt.Errorf("read target module file: %w", err))
	}
	selected := make([]adapterDefinition, 0, len(registry.definitions))
	for _, definition := range registry.definitions {
		version, detectErr := detectModuleVersion(moduleFile, definition.identity.Module)
		if detectErr != nil {
			return nil, nil, invalidBuildAdapterConfiguration(detectErr)
		}
		if version == "" {
			continue
		}
		if version != definition.identity.Version {
			return nil, nil, invalidBuildAdapterConfiguration(fmt.Errorf("unsupported %s version %q", definition.identity.Module, version))
		}
		selected = append(selected, definition)
	}
	return selected, moduleFile, nil
}

// requireAdapterSums returns the target module's sums after checking that
// they pin every selected adapter's exact identity.
func requireAdapterSums(workingDirectory string, selected []adapterDefinition) ([]byte, error) {
	sumFile, err := os.ReadFile(filepath.Join(workingDirectory, "go.sum"))
	if err != nil {
		return nil, invalidBuildAdapterConfiguration(fmt.Errorf("read target module sums: %w", err))
	}
	for _, definition := range selected {
		if !hasExactModuleSum(sumFile, definition.identity) {
			return nil, invalidBuildAdapterConfiguration(fmt.Errorf("target module sum for %s@%s is missing or modified", definition.identity.Module, definition.identity.Version))
		}
	}
	return sumFile, nil
}

func (registry adapterRegistry) prepare(spec target.Spec, moduleCache string) (target.Spec, []BuildAdapter, error) {
	if len(registry.definitions) == 0 {
		return spec, []BuildAdapter{}, nil
	}
	workingDirectory, err := target.ModuleDirectory(spec)
	if err != nil {
		return target.Spec{}, nil, invalidBuildAdapterConfiguration(fmt.Errorf("resolve target module directory: %w", err))
	}
	selected, moduleFile, err := registry.selected(workingDirectory)
	if err != nil {
		return target.Spec{}, nil, err
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
	sumFile, err := requireAdapterSums(workingDirectory, selected)
	if err != nil {
		return target.Spec{}, nil, err
	}
	preparationRoot, err := filepath.Abs(spec.PreparationRoot)
	if err != nil {
		return target.Spec{}, nil, fmt.Errorf("resolve deterministic I/O preparation root: %w", err)
	}
	root := filepath.Join(preparationRoot, ".io-adapter")
	if err := os.Mkdir(root, 0o700); err != nil {
		return target.Spec{}, nil, fmt.Errorf("create deterministic I/O adapter directory: %w", err)
	}
	// The go command records a directory replacement's path in the binary's
	// module information, so the replacement must live at a path that is the
	// same for every preparation on this host or the target identity changes
	// between repetitions. The toolchain root holds one copy per adapter
	// identity and replacement inventory.
	cacheRoot := root
	if spec.ToolchainRoot != "" {
		cacheRoot, err = filepath.Abs(filepath.Join(spec.ToolchainRoot, "adapters"))
		if err != nil {
			return target.Spec{}, nil, fmt.Errorf("resolve deterministic I/O adapter cache: %w", err)
		}
		if err := os.MkdirAll(cacheRoot, 0o700); err != nil {
			return target.Spec{}, nil, fmt.Errorf("create deterministic I/O adapter cache: %w", err)
		}
	}
	evidence := make([]BuildAdapter, 0, len(selected))
	for _, definition := range selected {
		var prepared adapterPreparation
		var prepareErr error
		if cacheRoot == root {
			prepared, prepareErr = definition.implementation.prepare(moduleCache, root, definition.identity)
			if prepareErr == nil && prepared.evidence.ReplacementRoot != prepared.replacement {
				prepareErr = errors.New("deterministic I/O adapter replacement root mismatch")
			}
		} else {
			prepared, prepareErr = prepareCachedAdapter(definition, moduleCache, cacheRoot)
		}
		if prepareErr != nil {
			return target.Spec{}, nil, prepareErr
		}
		moduleFile = append(moduleFile, []byte("\nreplace "+definition.identity.Module+" "+definition.identity.Version+" => "+prepared.replacement+"\n")...)
		evidence = append(evidence, prepared.evidence)
	}
	modFilePath := filepath.Join(root, "gomad.mod")
	if err := writeExclusive(modFilePath, moduleFile); err != nil {
		return target.Spec{}, nil, err
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

// PrepareTargetBuildAdapters prepares spec's build adapters from the module
// cache of spec's pinned toolchain after downloading each selected adapter's
// pinned module into it, so a clean module cache prepares the same adapters.
func (profile Spec) PrepareTargetBuildAdapters(ctx context.Context, spec target.Spec) (target.Spec, []BuildAdapter, error) {
	definition, err := profile.validated()
	if err != nil {
		return target.Spec{}, nil, err
	}
	workingDirectory, err := target.ModuleDirectory(spec)
	if err != nil {
		return target.Spec{}, nil, invalidBuildAdapterConfiguration(fmt.Errorf("resolve target module directory: %w", err))
	}
	selected, _, err := definition.adapters.selected(workingDirectory)
	if err != nil {
		return target.Spec{}, nil, err
	}
	if len(selected) == 0 {
		return profile.PrepareBuildAdapters(spec, "")
	}
	if _, err := requireAdapterSums(workingDirectory, selected); err != nil {
		return target.Spec{}, nil, err
	}
	for _, adapter := range selected {
		identity := target.ModuleIdentity{Path: adapter.identity.Module, Version: adapter.identity.Version, Sum: adapter.identity.Sum}
		if err := target.DownloadModule(ctx, spec.ToolchainRoot, identity); err != nil {
			return target.Spec{}, nil, err
		}
	}
	moduleCache, err := target.ReadModuleCache(ctx, spec.ToolchainRoot)
	if err != nil {
		return target.Spec{}, nil, err
	}
	return profile.PrepareBuildAdapters(spec, moduleCache)
}

func (profile Spec) PrepareBuildAdapters(spec target.Spec, moduleCache string) (target.Spec, []BuildAdapter, error) {
	definition, err := profile.validated()
	if err != nil {
		return target.Spec{}, nil, err
	}
	if len(spec.AdapterReplacements) != 0 {
		return target.Spec{}, nil, invalidBuildAdapterConfiguration(errors.New("target specification already contains adapter replacement evidence"))
	}
	prepared, adapters, err := definition.adapters.prepare(spec, moduleCache)
	if err != nil {
		return target.Spec{}, nil, err
	}
	profileIdentity := profile.Identity()
	prepared.AdapterReplacements = make([]target.AdapterReplacement, len(adapters))
	for index, adapter := range adapters {
		prepared.AdapterReplacements[index] = projectAdapterReplacement(profileIdentity, adapter)
	}
	return prepared, adapters, nil
}

func projectAdapterReplacement(profileIdentity Contract, adapter BuildAdapter) target.AdapterReplacement {
	return target.AdapterReplacement{
		Original:        target.ModuleIdentity{Path: adapter.Module, Version: adapter.Version, Sum: adapter.Sum},
		ReplacementPath: adapter.ReplacementRoot,
		PreparedPackage: adapter.PreparedPackage,
		ProfileName:     profileIdentity.Name, ProfileImplementationSHA256: string(profileIdentity.ImplementationSHA256),
		Adapter:                          target.ModuleIdentity{Path: adapter.Module, Version: adapter.Version, Sum: adapter.Sum},
		OriginalSourceInventorySHA256:    adapter.OriginalSourceInventorySHA256,
		ReplacementSourceInventorySHA256: adapter.ReplacementSourceInventorySHA256,
		PreparedSourceSetSHA256:          adapter.PreparedSourceSetSHA256,
	}
}

func hasExactModuleSum(contents []byte, identity gomadversion.AdapterIdentity) bool {
	found := false
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 || fields[0] != identity.Module || fields[1] != identity.Version {
			continue
		}
		if found || fields[2] != identity.Sum {
			return false
		}
		found = true
	}
	return found
}

func detectModuleVersion(contents []byte, module string) (string, error) {
	parsed, err := modfile.Parse("go.mod", contents, nil)
	if err != nil {
		return "", fmt.Errorf("parse target module file: %w", err)
	}
	for _, replacement := range parsed.Replace {
		if replacement.Old.Path == module {
			return "", fmt.Errorf("target module already replaces %s", module)
		}
	}
	version := ""
	for _, requirement := range parsed.Require {
		if requirement.Mod.Path != module {
			continue
		}
		if version != "" {
			return "", fmt.Errorf("target module requires %s more than once", module)
		}
		version = requirement.Mod.Version
	}
	return version, nil
}

// prepareCachedAdapter prepares an adapter replacement in a private directory
// and publishes it under a name derived from its identity and replacement
// inventory, reusing a published copy whose inventory still matches.
func prepareCachedAdapter(definition adapterDefinition, moduleCache, cacheRoot string) (adapterPreparation, error) {
	work, err := os.MkdirTemp(cacheRoot, ".prepare-*")
	if err != nil {
		return adapterPreparation{}, fmt.Errorf("create deterministic I/O adapter work directory: %w", err)
	}
	defer os.RemoveAll(work)
	prepared, err := definition.implementation.prepare(moduleCache, work, definition.identity)
	if err != nil {
		return adapterPreparation{}, err
	}
	if prepared.evidence.ReplacementRoot != prepared.replacement || !strings.HasPrefix(prepared.replacement, work+string(filepath.Separator)) {
		return adapterPreparation{}, errors.New("deterministic I/O adapter replacement root mismatch")
	}
	inventory := prepared.evidence.ReplacementSourceInventorySHA256
	if len(inventory) < len("sha256:")+16 {
		return adapterPreparation{}, errors.New("deterministic I/O adapter replacement inventory is incomplete")
	}
	published := filepath.Join(cacheRoot, filepath.Base(prepared.replacement)+"@"+definition.identity.Version+"-"+inventory[len("sha256:"):len("sha256:")+16])
	if err := os.Rename(prepared.replacement, published); err != nil {
		if _, statErr := os.Lstat(published); statErr != nil {
			return adapterPreparation{}, fmt.Errorf("publish deterministic I/O adapter replacement: %w", err)
		}
		existing, digestErr := digestAdapterSourceInventory(published)
		if digestErr != nil {
			return adapterPreparation{}, fmt.Errorf("verify published deterministic I/O adapter replacement: %w", digestErr)
		}
		if existing != inventory {
			return adapterPreparation{}, fmt.Errorf("published deterministic I/O adapter replacement %s has inventory %s, want %s", published, existing, inventory)
		}
	}
	relocate := func(path string) string {
		return published + strings.TrimPrefix(path, prepared.replacement)
	}
	prepared.evidence.Replacement = relocate(prepared.evidence.Replacement)
	prepared.evidence.ReplacementRoot = published
	prepared.replacement = published
	return prepared, nil
}

// hostPin selects the identity recorded for the running platform when a
// prepared source set legitimately differs between qualified platforms.
func hostPin(pins map[string]string) string {
	return pins[runtime.GOOS+"/"+runtime.GOARCH]
}

func mustAdapterRegistry(identities []gomadversion.AdapterIdentity, implementations []adapterImplementation) adapterRegistry {
	registry, err := newAdapterRegistry(identities, implementations)
	if err != nil {
		panic(err) //nolint:forbidigo // A generated adapter without an implementation makes the package unusable.
	}
	return registry
}
