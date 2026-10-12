package build

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	targetbuild "go.temporal.io/server/tools/gomad3/backends/native/build/internal/build"
	"go.temporal.io/server/tools/gomad3/backends/native/build/internal/capabilitypolicy"
	"go.temporal.io/server/tools/gomad3/backends/native/build/internal/capabilityreview"
	"go.temporal.io/server/tools/gomad3/backends/native/build/internal/gocommand"
	compatibility "go.temporal.io/server/tools/gomad3/backends/native/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/backends/native/sourceinventory"
	"go.temporal.io/server/tools/gomad3/backends/native/toolchain/installation"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/target"
)

// Capability evidence collection owns the host effects of a capability
// review: listing the closure, reading the build overlay and package sources,
// resolving and digesting adapter replacements, and loading the compatibility
// packs. Evaluation judges the collected evidence without further effects.

type listedModule = capabilityreview.Module
type listedPackage = capabilityreview.Package

// collectCapabilityListing lists the closure and reads the inputs that decide
// how its sources are collected.
func collectCapabilityListing(ctx context.Context, goCommand string, spec target.Spec, tags []string, commandDirectory, packageArgument string, runner gocommand.Runner) ([]listedPackage, map[string]string, error) {
	packages, err := capabilityreview.ListWith(ctx, capabilityreview.Request{
		GoCommand: goCommand, Directory: commandDirectory, Package: packageArgument, Tags: tags,
		Overlay: spec.BuildOverlay, ModFile: spec.BuildModFile, Environment: targetbuild.Environment(), Test: spec.Kind == target.KindGoTest,
		OutputLimit: maximumCapabilityReviewOutputBytes, PackageLimit: maximumCapabilityReviewPackages,
	}, runner)
	if err != nil {
		var commandError *capabilityreview.CommandError
		if errors.As(err, &commandError) && commandError.InvalidInput {
			return nil, nil, invalidCapabilityReview(err)
		}
		if _, ok := err.(*gocommand.OverflowError); ok {
			err = fmt.Errorf("target capability closure output exceeds %d bytes", maximumCapabilityReviewOutputBytes)
		}
		return nil, nil, fmt.Errorf("inspect target capability closure: %w", err)
	}
	overlay, err := loadBuildOverlay(spec.BuildOverlay, commandDirectory)
	if err != nil {
		return nil, nil, err
	}
	if err := validateAdapterReplacementInputs(spec); err != nil {
		return nil, nil, err
	}
	return packages, overlay, nil
}

// collectCapabilityPackages projects each listed package into canonical
// capability evidence, reading its sources and verifying adapter replacements.
func collectCapabilityPackages(packages []listedPackage, overlay map[string]string, replacementSets ...[]target.AdapterReplacement) ([]target.CapabilityPackage, error) {
	replacements := []target.AdapterReplacement{}
	if len(replacementSets) > 1 {
		return nil, errors.New("target capability review has duplicate adapter replacement inputs")
	}
	if len(replacementSets) == 1 {
		replacements = replacementSets[0]
	}
	replacementsByModule, err := indexAdapterReplacements(replacements)
	if err != nil {
		return nil, err
	}
	collected := make([]target.CapabilityPackage, 0, len(packages))
	for _, pkg := range packages {
		projected, include, err := projectCapabilityPackage(pkg, overlay, replacementsByModule)
		if err != nil {
			return nil, err
		}
		if !include {
			continue
		}
		collected = append(collected, projected)
	}
	sort.Slice(collected, func(i, j int) bool {
		if collected[i].ImportPath != collected[j].ImportPath {
			return collected[i].ImportPath < collected[j].ImportPath
		}
		if collected[i].ForTest != collected[j].ForTest {
			return collected[i].ForTest < collected[j].ForTest
		}
		return collected[i].Name < collected[j].Name
	})
	return collected, nil
}

// loadCompatibilityPolicy loads the packs this build selects from, including
// those the environment names, for the host platform.
func loadCompatibilityPolicy() (capabilitypolicy.Policy, error) {
	packs, err := compatibility.LoadPacks()
	if err != nil {
		return capabilitypolicy.Policy{}, fmt.Errorf("select target compatibility packs: %w", err)
	}
	return capabilitypolicy.Policy{Packs: packs, Platform: runtime.GOOS + "/" + runtime.GOARCH}, nil
}

func projectCapabilityPackage(pkg listedPackage, overlay map[string]string, replacements map[string]target.AdapterReplacement) (target.CapabilityPackage, bool, error) {
	sourceFiles := packageSourceFiles(pkg)
	replacement, hasReplacement, err := matchAdapterReplacement(pkg.Module, replacements)
	if err != nil {
		return target.CapabilityPackage{}, false, err
	}
	projected := target.CapabilityPackage{
		ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name, Root: !pkg.DepOnly, Standard: pkg.Standard,
		Imports: sortedSetCopy(packageImports(pkg)), Module: projectCapabilityModule(pkg.Module, replacement, hasReplacement), Sources: []target.CapabilitySource{},
		ForeignSources: []target.CapabilityForeignSource{}, GeneratedTestMain: generatedTestMain(pkg),
	}
	if pkg.Standard || projected.GeneratedTestMain {
		return projected, true, nil
	}
	foreignSources, err := projectForeignSources(pkg, overlay)
	if err != nil {
		return target.CapabilityPackage{}, false, err
	}
	projected.ForeignSources = foreignSources
	if pkg.ForTest == "" && len(sourceFiles) == 0 && len(projected.ForeignSources) == 0 && (len(pkg.TestGoFiles) != 0 || len(pkg.XTestGoFiles) != 0) {
		return target.CapabilityPackage{}, false, nil
	}
	for _, name := range sourceFiles {
		source, err := projectCapabilitySource(pkg, overlay, name)
		if err != nil {
			return target.CapabilityPackage{}, false, err
		}
		projected.Sources = append(projected.Sources, source)
	}
	sort.Slice(projected.Sources, func(i, j int) bool { return projected.Sources[i].Name < projected.Sources[j].Name })
	if hasReplacement && pkg.ImportPath == replacement.PreparedPackage {
		sourceSetSHA256 := capabilityCompatibilityPackage(projected).SourceSetSHA256
		if sourceSetSHA256 != replacement.PreparedSourceSetSHA256 {
			return target.CapabilityPackage{}, false, fmt.Errorf("inspect target capability source %s: adapter prepared source-set identity mismatch: got %s, want %s", pkg.ImportPath, sourceSetSHA256, replacement.PreparedSourceSetSHA256)
		}
	}
	return projected, true, nil
}

func projectCapabilitySource(pkg listedPackage, overlay map[string]string, name string) (target.CapabilitySource, error) {
	if filepath.Base(name) != name || pkg.Dir == "" {
		return target.CapabilitySource{}, fmt.Errorf("inspect target capability source %s: invalid source path %q", pkg.ImportPath, name)
	}
	path := filepath.Join(pkg.Dir, name)
	if replacement, found := overlay[filepath.Clean(path)]; found {
		path = replacement
	}
	contents, err := hostfs.ReadBounded(path, maximumCapabilitySourceBytes)
	if err != nil {
		return target.CapabilitySource{}, fmt.Errorf("inspect target capability source %s: unreadable source %s: %w", pkg.ImportPath, name, err)
	}
	hash := sha256.Sum256(contents)
	directives := []string{}
	malformedLinkname := false
	if bytes.Contains(contents, []byte("//go:linkname")) {
		directives, malformedLinkname = linknameDirectives(contents)
		malformedLinkname = !malformedLinkname
		if malformedLinkname {
			directives = []string{}
		}
	}
	return target.CapabilitySource{Name: name, SHA256: fmt.Sprintf("sha256:%x", hash), LinknameDirectives: directives, MalformedLinkname: malformedLinkname}, nil
}

func linknameDirectives(contents []byte) ([]string, bool) {
	marker := []byte("//go:linkname")
	directives := make([]string, 0, bytes.Count(contents, marker))
	for _, line := range bytes.Split(contents, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, marker) {
			continue
		}
		fields := strings.Fields(string(line))
		if len(fields) != 3 || fields[0] != string(marker) {
			return nil, false
		}
		directives = append(directives, fields[1]+" "+fields[2])
	}
	return directives, len(directives) == bytes.Count(contents, marker)
}

func loadBuildOverlay(path, commandDirectory string) (map[string]string, error) {
	if path == "" {
		return nil, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(commandDirectory, path)
	}
	contents, err := hostfs.ReadBounded(path, 4<<20)
	if err != nil {
		return nil, fmt.Errorf("read target build overlay: %w", err)
	}
	var wire struct {
		Replace map[string]string `json:"Replace"`
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&wire); err != nil {
		return nil, fmt.Errorf("decode target build overlay: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return nil, errors.New("decode target build overlay: trailing data")
	}
	result := make(map[string]string, len(wire.Replace))
	for original, replacement := range wire.Replace {
		if !filepath.IsAbs(original) || !filepath.IsAbs(replacement) {
			return nil, errors.New("unsupported target capability: build overlay paths must be absolute")
		}
		result[filepath.Clean(original)] = filepath.Clean(replacement)
	}
	return result, nil
}

func projectCapabilityModule(module *listedModule, adapter target.AdapterReplacement, hasAdapter bool) *target.CapabilityModule {
	if module == nil {
		return nil
	}
	projected := &target.CapabilityModule{Path: module.Path, Version: module.Version, Sum: module.Sum, Main: module.Main}
	if hasAdapter {
		projected.Path = adapter.Original.Path
		projected.Version = adapter.Original.Version
		projected.Sum = adapter.Original.Sum
	}
	if module.Replace != nil {
		projected.Replacement = projectCapabilityModule(module.Replace, target.AdapterReplacement{}, false)
		projected.Replacement.Local = module.Replace.Dir != ""
		if projected.Replacement.Local {
			projected.Replacement.Path = ""
			projected.Replacement.Version = ""
			projected.Replacement.Sum = ""
		}
	}
	if hasAdapter {
		projected.Adapter = &target.CapabilityAdapterReplacement{
			ProfileName: adapter.ProfileName, ProfileImplementationSHA256: adapter.ProfileImplementationSHA256,
			Adapter: adapter.Adapter, OriginalSourceInventorySHA256: adapter.OriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: adapter.ReplacementSourceInventorySHA256,
			PreparedSourceSetSHA256:          adapter.PreparedSourceSetSHA256,
		}
	}
	return projected
}

func projectForeignSources(pkg listedPackage, overlay map[string]string) ([]target.CapabilityForeignSource, error) {
	projected := []target.CapabilityForeignSource{}
	groups := []struct {
		kind  string
		files []string
	}{
		{kind: "cgo", files: pkg.CgoFiles},
		{kind: "c", files: pkg.CFiles},
		{kind: "cxx", files: pkg.CXXFiles},
		{kind: "objc", files: pkg.MFiles},
		{kind: "header", files: pkg.HFiles},
		{kind: "fortran", files: pkg.FFiles},
		{kind: "assembly", files: pkg.SFiles},
		{kind: "swig", files: pkg.SwigFiles},
		{kind: "swig-cxx", files: pkg.SwigCXXFiles},
		{kind: "object", files: pkg.SysoFiles},
	}
	for _, group := range groups {
		for _, name := range group.files {
			if filepath.Base(name) != name || pkg.Dir == "" {
				return nil, fmt.Errorf("inspect target capability source %s: invalid source path %q", pkg.ImportPath, name)
			}
			path := filepath.Join(pkg.Dir, name)
			if replacement, found := overlay[filepath.Clean(path)]; found {
				path = replacement
			}
			contents, err := hostfs.ReadBounded(path, maximumCapabilitySourceBytes)
			if err != nil {
				return nil, fmt.Errorf("inspect target capability source %s: unreadable source %s: %w", pkg.ImportPath, name, err)
			}
			digest := sha256.Sum256(contents)
			projected = append(projected, target.CapabilityForeignSource{Kind: group.kind, Name: name, SHA256: fmt.Sprintf("sha256:%x", digest)})
		}
	}
	sort.Slice(projected, func(i, j int) bool {
		if projected[i].Kind != projected[j].Kind {
			return projected[i].Kind < projected[j].Kind
		}
		return projected[i].Name < projected[j].Name
	})
	return projected, nil
}

func validateAdapterReplacementInputs(spec target.Spec) error {
	if len(spec.AdapterReplacements) == 0 {
		return nil
	}
	// Replacements live either in the private preparation root or in the
	// toolchain's adapter cache, which the deterministic I/O profile owns.
	roots := []string{spec.PreparationRoot}
	if spec.ToolchainRoot != "" {
		layout, err := installation.At(spec.ToolchainRoot)
		if err != nil {
			return fmt.Errorf("resolve adapter preparation root: %w", err)
		}
		roots = append(roots, layout.Adapters())
	}
	resolvedRoots := make([]string, 0, len(roots))
	for _, candidate := range roots {
		root, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			if candidate != spec.PreparationRoot && errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("resolve adapter preparation root: %w", err)
		}
		root, err = filepath.Abs(root)
		if err != nil {
			return fmt.Errorf("resolve adapter preparation root: %w", err)
		}
		resolvedRoots = append(resolvedRoots, root)
	}
	for _, replacement := range spec.AdapterReplacements {
		path, err := filepath.EvalSymlinks(replacement.ReplacementPath)
		if err != nil {
			return fmt.Errorf("resolve adapter replacement path: %w", err)
		}
		path, err = filepath.Abs(path)
		if err != nil {
			return fmt.Errorf("resolve adapter replacement path: %w", err)
		}
		inside := false
		for _, root := range resolvedRoots {
			relative, err := filepath.Rel(root, path)
			if err == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative) {
				inside = true
				break
			}
		}
		if !inside {
			return errors.New("adapter replacement path is outside the private preparation root")
		}
	}
	return nil
}

func indexAdapterReplacements(replacements []target.AdapterReplacement) (map[string]target.AdapterReplacement, error) {
	result := make(map[string]target.AdapterReplacement, len(replacements))
	for _, replacement := range replacements {
		if replacement.Original.Path == "" || replacement.Original.Version == "" || replacement.Original.Sum == "" || replacement.ReplacementPath == "" || replacement.PreparedPackage == "" {
			return nil, errors.New("adapter replacement input identity is incomplete")
		}
		if replacement.PreparedPackage != replacement.Original.Path && !strings.HasPrefix(replacement.PreparedPackage, replacement.Original.Path+"/") {
			return nil, errors.New("adapter prepared package is outside its module")
		}
		if _, duplicate := result[replacement.Original.Path]; duplicate {
			return nil, fmt.Errorf("adapter replacement input is duplicated: %s", replacement.Original.Path)
		}
		result[replacement.Original.Path] = replacement
	}
	return result, nil
}

func matchAdapterReplacement(module *listedModule, replacements map[string]target.AdapterReplacement) (target.AdapterReplacement, bool, error) {
	if module == nil || module.Replace == nil || module.Replace.Dir == "" {
		return target.AdapterReplacement{}, false, nil
	}
	replacement, found := replacements[module.Path]
	if !found {
		return target.AdapterReplacement{}, false, nil
	}
	if replacement.Original.Path != module.Path || replacement.Original.Version != module.Version ||
		module.Sum != "" && replacement.Original.Sum != module.Sum {
		return target.AdapterReplacement{}, false, fmt.Errorf(
			"adapter replacement module identity mismatch: got %s@%s %q, want %s@%s %q",
			module.Path, module.Version, module.Sum,
			replacement.Original.Path, replacement.Original.Version, replacement.Original.Sum,
		)
	}
	wantPath, err := filepath.EvalSymlinks(replacement.ReplacementPath)
	if err != nil {
		return target.AdapterReplacement{}, false, fmt.Errorf("resolve adapter replacement evidence: %w", err)
	}
	actualPath, err := filepath.EvalSymlinks(module.Replace.Dir)
	if err != nil {
		return target.AdapterReplacement{}, false, fmt.Errorf("resolve target adapter replacement: %w", err)
	}
	wantPath, err = filepath.Abs(wantPath)
	if err != nil {
		return target.AdapterReplacement{}, false, err
	}
	actualPath, err = filepath.Abs(actualPath)
	if err != nil {
		return target.AdapterReplacement{}, false, err
	}
	if wantPath != actualPath {
		return target.AdapterReplacement{}, false, errors.New("adapter replacement operational path mismatch")
	}
	digest, err := sourceinventory.Digest(actualPath)
	if err != nil {
		return target.AdapterReplacement{}, false, fmt.Errorf("inspect adapter replacement source inventory: %w", adapterInventoryError(err))
	}
	if digest != replacement.ReplacementSourceInventorySHA256 {
		return target.AdapterReplacement{}, false, errors.New("adapter replacement source inventory mismatch")
	}
	portable := target.CapabilityAdapterReplacement{
		ProfileName: replacement.ProfileName, ProfileImplementationSHA256: replacement.ProfileImplementationSHA256,
		Adapter: replacement.Adapter, OriginalSourceInventorySHA256: replacement.OriginalSourceInventorySHA256,
		ReplacementSourceInventorySHA256: replacement.ReplacementSourceInventorySHA256,
		PreparedSourceSetSHA256:          replacement.PreparedSourceSetSHA256,
	}
	if err := validateCapabilityAdapter(portable); err != nil {
		return target.AdapterReplacement{}, false, err
	}
	return replacement, true, nil
}

// adapterInventoryError keeps the target's typed capacity error for an
// adapter replacement whose inventory exceeds its limits.
func adapterInventoryError(err error) error {
	var capacity *sourceinventory.CapacityError
	if errors.As(err, &capacity) {
		return &target.AdapterCapacityError{Resource: capacity.Resource, Limit: capacity.Limit}
	}
	return err
}

func validateExecStandardPackages(ctx context.Context, goCommand string, closure target.CapabilityClosure) error {
	result, err := gocommand.Default().StructuredCommand(ctx, gocommand.Request{
		Command: []string{goCommand, "list", "std"},
		Env:     targetbuild.Environment(), OutputLimit: maximumStandardPackagesBytes,
	})
	if err != nil {
		return fmt.Errorf("inspect pinned standard packages: %w: %s", err, result.Stderr)
	}
	if err := result.OutputError(); err != nil {
		var stderr []byte
		if exit, ok := err.(*exec.ExitError); ok {
			stderr = exit.Stderr
		}
		return fmt.Errorf("inspect pinned standard packages: %w: %s", err, stderr)
	}
	standard := make(map[string]struct{})
	for _, importPath := range strings.Fields(string(result.Stdout)) {
		standard[importPath] = struct{}{}
	}
	for _, pkg := range closure.Packages {
		_, found := standard[pkg.ImportPath]
		if found != pkg.Standard {
			return fmt.Errorf("exec provenance standard package classification is invalid for %s", pkg.ImportPath)
		}
	}
	return nil
}

func packageImports(pkg listedPackage) []string {
	return append([]string(nil), pkg.Imports...)
}

func packageSourceFiles(pkg listedPackage) []string {
	return sortedSetCopy(pkg.GoFiles)
}

func sortedSetCopy(values []string) []string {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func generatedTestMain(pkg listedPackage) bool {
	return pkg.Name == "main" && strings.HasSuffix(pkg.ImportPath, ".test") && len(pkg.GoFiles) == 1 && filepath.IsAbs(pkg.GoFiles[0])
}
