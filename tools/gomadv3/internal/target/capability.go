package target

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
	"slices"
	"sort"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/compatibility"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const capabilityClosureSchema = "gomadv3.target-capability-closure/v2"

type CapabilityClosure struct {
	Schema        string                   `json:"schema"`
	Compatibility []compatibility.Identity `json:"compatibility"`
	Packages      []CapabilityPackage      `json:"packages"`
}

type CapabilityPackage struct {
	ImportPath        string             `json:"import_path"`
	Name              string             `json:"name"`
	Standard          bool               `json:"standard"`
	Imports           []string           `json:"imports"`
	Module            *CapabilityModule  `json:"module,omitempty"`
	Sources           []CapabilitySource `json:"sources"`
	ForeignSources    []string           `json:"foreign_sources"`
	GeneratedTestMain bool               `json:"generated_test_main,omitempty"`
}

type CapabilityModule struct {
	Path        string            `json:"path"`
	Version     string            `json:"version"`
	Sum         string            `json:"sum"`
	Main        bool              `json:"main"`
	Local       bool              `json:"local"`
	Replacement *CapabilityModule `json:"replacement,omitempty"`
}

type CapabilitySource struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type UnsupportedCapabilityError struct {
	ImportPath string
	Capability string
}

func (err *UnsupportedCapabilityError) Error() string {
	return fmt.Sprintf("unsupported target capability: package %s %s", err.ImportPath, err.Capability)
}

func validateGoCapabilityClosure(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string) (CapabilityClosure, error) {
	closure, err := reviewGoCapabilityClosure(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	if err != nil {
		return CapabilityClosure{}, err
	}
	if err := validateCapabilityReview(closure); err != nil {
		return CapabilityClosure{}, err
	}
	return closure, nil
}

func ReviewCapabilityClosure(ctx context.Context, spec Spec) (CapabilityClosure, error) {
	if spec.Kind != KindGoRun && spec.Kind != KindGoTest {
		return CapabilityClosure{}, fmt.Errorf("capability review requires a go-run or go-test target")
	}
	tags, err := normalizeBuildTags(spec.BuildTags)
	if err != nil {
		return CapabilityClosure{}, err
	}
	if spec.Source == "" || spec.WorkingDir == "" || spec.ToolchainRoot == "" {
		return CapabilityClosure{}, fmt.Errorf("capability review requires source, working directory, and toolchain root")
	}
	goCommand, err := filepath.Abs(filepath.Join(spec.ToolchainRoot, "bin", "go"))
	if err != nil {
		return CapabilityClosure{}, fmt.Errorf("resolve pinned Go command: %w", err)
	}
	commandDirectory, packageArgument, err := resolveBuildContext(spec.WorkingDir, spec.Source)
	if err != nil {
		return CapabilityClosure{}, err
	}
	closure, err := reviewGoCapabilityClosure(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	if err != nil {
		return CapabilityClosure{}, err
	}
	if err := validateCapabilityReview(closure); err != nil {
		return CapabilityClosure{}, err
	}
	return closure, nil
}

func reviewGoCapabilityClosure(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string) (CapabilityClosure, error) {
	arguments := []string{"list", "-deps", "-json"}
	if spec.Kind == KindGoTest {
		arguments = append(arguments, "-test")
	}
	if spec.BuildOverlay != "" {
		arguments = append(arguments, "-overlay", spec.BuildOverlay)
	}
	if spec.BuildModFile != "" {
		arguments = append(arguments, "-modfile", spec.BuildModFile)
	}
	if len(tags) > 0 {
		arguments = append(arguments, "-tags", strings.Join(tags, ","))
	}
	arguments = append(arguments, packageArgument)
	command := exec.CommandContext(ctx, goCommand, arguments...)
	command.Dir = commandDirectory
	command.Env = preparationEnvironment()
	output, err := command.Output()
	if err != nil {
		var stderr []byte
		if exit, ok := err.(*exec.ExitError); ok {
			stderr = exit.Stderr
		}
		return CapabilityClosure{}, fmt.Errorf("inspect target capability closure: %w: %s", err, stderr)
	}
	decoder := json.NewDecoder(bytes.NewReader(output))
	var packages []listedPackage
	for {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return CapabilityClosure{}, fmt.Errorf("decode target capability closure: %w", err)
		}
		packages = append(packages, pkg)
	}
	overlay, err := loadBuildOverlay(spec.BuildOverlay, commandDirectory)
	if err != nil {
		return CapabilityClosure{}, err
	}
	return projectCapabilityClosure(packages, overlay)
}

type listedModule struct {
	Path    string        `json:"Path"`
	Version string        `json:"Version"`
	Sum     string        `json:"Sum"`
	Main    bool          `json:"Main"`
	Dir     string        `json:"Dir"`
	Replace *listedModule `json:"Replace"`
}

type listedPackage struct {
	ImportPath   string        `json:"ImportPath"`
	ForTest      string        `json:"ForTest"`
	Name         string        `json:"Name"`
	Standard     bool          `json:"Standard"`
	Dir          string        `json:"Dir"`
	GoFiles      []string      `json:"GoFiles"`
	TestGoFiles  []string      `json:"TestGoFiles"`
	XTestGoFiles []string      `json:"XTestGoFiles"`
	CgoFiles     []string      `json:"CgoFiles"`
	CFiles       []string      `json:"CFiles"`
	CXXFiles     []string      `json:"CXXFiles"`
	MFiles       []string      `json:"MFiles"`
	FFiles       []string      `json:"FFiles"`
	SFiles       []string      `json:"SFiles"`
	SwigFiles    []string      `json:"SwigFiles"`
	SwigCXXFiles []string      `json:"SwigCXXFiles"`
	SysoFiles    []string      `json:"SysoFiles"`
	Imports      []string      `json:"Imports"`
	TestImports  []string      `json:"TestImports"`
	XTestImports []string      `json:"XTestImports"`
	Module       *listedModule `json:"Module"`
}

func validateCapabilityClosure(packages []listedPackage) error {
	closure, err := projectCapabilityClosure(packages, nil)
	if err != nil {
		return err
	}
	return validateCapabilityReview(closure)
}

func projectCapabilityClosure(packages []listedPackage, overlay map[string]string) (CapabilityClosure, error) {
	compatibilityPackages := make([]compatibility.Package, 0, len(packages))
	for _, pkg := range packages {
		compatibilityPackages = append(compatibilityPackages, listedCompatibilityPackage(pkg))
	}
	selection, err := compatibility.Select(compatibilityPackages)
	if err != nil {
		return CapabilityClosure{}, fmt.Errorf("select target compatibility packs: %w", err)
	}
	closure := CapabilityClosure{
		Schema:        capabilityClosureSchema,
		Compatibility: selection.Identities(),
		Packages:      make([]CapabilityPackage, 0, len(packages)),
	}
	for _, pkg := range packages {
		sourceFiles := packageSourceFiles(pkg)
		projected := CapabilityPackage{
			ImportPath:        pkg.ImportPath,
			Name:              pkg.Name,
			Standard:          pkg.Standard,
			Imports:           sortedSetCopy(packageImports(pkg)),
			Module:            projectCapabilityModule(pkg.Module),
			Sources:           []CapabilitySource{},
			ForeignSources:    projectForeignSources(pkg),
			GeneratedTestMain: generatedTestMain(pkg),
		}
		if pkg.Standard || projected.GeneratedTestMain {
			closure.Packages = append(closure.Packages, projected)
			continue
		}
		if pkg.ForTest == "" && len(sourceFiles) == 0 && len(projected.ForeignSources) == 0 && (len(pkg.TestGoFiles) != 0 || len(pkg.XTestGoFiles) != 0) {
			continue
		}
		linknames := make(map[string][]string)
		for _, name := range sourceFiles {
			if filepath.Base(name) != name || pkg.Dir == "" {
				return CapabilityClosure{}, unsupportedCapability(pkg.ImportPath, "has an invalid source path")
			}
			path := filepath.Join(pkg.Dir, name)
			if replacement, found := overlay[filepath.Clean(path)]; found {
				path = replacement
			}
			info, err := os.Lstat(path)
			if err != nil || !info.Mode().IsRegular() {
				return CapabilityClosure{}, unsupportedCapability(pkg.ImportPath, "has an unreadable source file "+name)
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return CapabilityClosure{}, fmt.Errorf("inspect target capability source %s: %w", pkg.ImportPath, err)
			}
			hash := sha256.Sum256(contents)
			digest := fmt.Sprintf("sha256:%x", hash)
			if bytes.Contains(contents, []byte("//go:linkname")) {
				directives, valid := linknameDirectives(contents)
				if !valid {
					return CapabilityClosure{}, unsupportedCapability(pkg.ImportPath, "uses go:linkname in "+name)
				}
				linknames[name] = directives
			}
			projected.Sources = append(projected.Sources, CapabilitySource{Name: name, SHA256: digest})
		}
		sort.Slice(projected.Sources, func(i, j int) bool { return projected.Sources[i].Name < projected.Sources[j].Name })
		compatibilityPackage := capabilityCompatibilityPackage(projected)
		for _, source := range projected.Sources {
			directives, found := linknames[source.Name]
			if found && !selection.AllowsLinkname(compatibilityPackage, source.Name, source.SHA256, directives) {
				return CapabilityClosure{}, unsupportedCapability(pkg.ImportPath, "uses go:linkname in "+source.Name)
			}
		}
		closure.Packages = append(closure.Packages, projected)
	}
	sort.Slice(closure.Packages, func(i, j int) bool {
		if closure.Packages[i].ImportPath != closure.Packages[j].ImportPath {
			return closure.Packages[i].ImportPath < closure.Packages[j].ImportPath
		}
		return closure.Packages[i].Name < closure.Packages[j].Name
	})
	return closure, nil
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
	contents, err := readBoundedRegularFile(path, 4<<20)
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
		return nil, fmt.Errorf("decode target build overlay: trailing data")
	}
	result := make(map[string]string, len(wire.Replace))
	for original, replacement := range wire.Replace {
		if !filepath.IsAbs(original) || !filepath.IsAbs(replacement) {
			return nil, fmt.Errorf("unsupported target capability: build overlay paths must be absolute")
		}
		result[filepath.Clean(original)] = filepath.Clean(replacement)
	}
	return result, nil
}

func validateCapabilityReview(closure CapabilityClosure) error {
	if closure.Schema != capabilityClosureSchema || closure.Compatibility == nil || len(closure.Packages) == 0 {
		return fmt.Errorf("unsupported or empty target capability closure")
	}
	if !sortedUniqueCompatibility(closure.Compatibility) {
		return errors.New("target capability closure compatibility packs are not canonical")
	}
	compatibilityPackages := make([]compatibility.Package, 0, len(closure.Packages))
	for _, pkg := range closure.Packages {
		compatibilityPackages = append(compatibilityPackages, capabilityCompatibilityPackage(pkg))
	}
	selection, err := compatibility.Select(compatibilityPackages)
	if err != nil {
		return fmt.Errorf("select target compatibility packs: %w", err)
	}
	if !slices.Equal(selection.Identities(), closure.Compatibility) {
		return errors.New("target capability closure compatibility pack identity does not match its package closure")
	}
	mainPackage := false
	for index, pkg := range closure.Packages {
		if pkg.ImportPath == "" || pkg.Name == "" {
			return fmt.Errorf("target capability closure has an empty package identity")
		}
		if pkg.Imports == nil || pkg.Sources == nil || pkg.ForeignSources == nil {
			return fmt.Errorf("target capability closure package %s has non-canonical null fields", pkg.ImportPath)
		}
		if index > 0 {
			previous := closure.Packages[index-1]
			if previous.ImportPath > pkg.ImportPath || previous.ImportPath == pkg.ImportPath && previous.Name >= pkg.Name {
				return fmt.Errorf("target capability closure packages are not sorted and unique")
			}
		}
		if !sortedUnique(pkg.Imports) || !sortedUnique(pkg.ForeignSources) || !sortedUniqueSources(pkg.Sources) {
			return fmt.Errorf("target capability closure package %s is not canonical", pkg.ImportPath)
		}
		if err := validateCapabilityModule(pkg.Module); err != nil {
			return fmt.Errorf("target capability closure package %s: %w", pkg.ImportPath, err)
		}
		for _, source := range pkg.Sources {
			_, digestErr := record.ParseSHA256(source.SHA256)
			if filepath.Base(source.Name) != source.Name || source.Name == "" || digestErr != nil {
				return fmt.Errorf("target capability closure package %s has invalid source evidence", pkg.ImportPath)
			}
		}
		if pkg.Name == "main" {
			mainPackage = true
		}
		if pkg.GeneratedTestMain && (pkg.Name != "main" || !strings.HasSuffix(pkg.ImportPath, ".test") || pkg.Standard || pkg.Module != nil && !pkg.Module.Main || len(pkg.Sources) != 0 || len(pkg.ForeignSources) != 0) {
			return fmt.Errorf("target capability closure package %s has invalid generated test-main evidence", pkg.ImportPath)
		}
		if pkg.Standard {
			continue
		}
		compatibilityPackage := capabilityCompatibilityPackage(pkg)
		for _, imported := range pkg.Imports {
			if forbiddenImport(imported) && !selection.AllowsCapability(compatibilityPackage, "import:"+imported) {
				return unsupportedCapability(pkg.ImportPath, "imports "+imported)
			}
		}
		for _, source := range pkg.ForeignSources {
			if !selection.AllowsCapability(compatibilityPackage, "foreign:"+source) {
				return unsupportedCapability(pkg.ImportPath, "contains foreign or assembly source "+source)
			}
		}
		if pkg.GeneratedTestMain {
			continue
		}
		if len(pkg.Sources) == 0 {
			return unsupportedCapability(pkg.ImportPath, "has no reviewed Go source")
		}
	}
	if !mainPackage {
		return fmt.Errorf("target capability closure has no main package")
	}
	return nil
}

func forbiddenImport(importPath string) bool {
	return importPath == "syscall" || importPath == "os/exec" || importPath == "os/signal" || importPath == "os/user" || importPath == "plugin" || importPath == "runtime/cgo" || strings.HasPrefix(importPath, "golang.org/x/sys/")
}

func validateExecStandardPackages(ctx context.Context, goCommand string, closure CapabilityClosure) error {
	command := exec.CommandContext(ctx, goCommand, "list", "std")
	command.Env = preparationEnvironment()
	output, err := command.Output()
	if err != nil {
		var stderr []byte
		if exit, ok := err.(*exec.ExitError); ok {
			stderr = exit.Stderr
		}
		return fmt.Errorf("inspect pinned standard packages: %w: %s", err, stderr)
	}
	standard := make(map[string]struct{})
	for _, importPath := range strings.Fields(string(output)) {
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

func projectCapabilityModule(module *listedModule) *CapabilityModule {
	if module == nil {
		return nil
	}
	projected := &CapabilityModule{Path: module.Path, Version: module.Version, Sum: module.Sum, Main: module.Main}
	if module.Replace != nil {
		projected.Replacement = projectCapabilityModule(module.Replace)
		projected.Replacement.Local = module.Replace.Dir != ""
		if projected.Replacement.Local {
			projected.Replacement.Path = ""
			projected.Replacement.Version = ""
			projected.Replacement.Sum = ""
		}
	}
	return projected
}

func projectForeignSources(pkg listedPackage) []string {
	projected := []string{}
	groups := []struct {
		kind  string
		files []string
	}{
		{kind: "cgo", files: pkg.CgoFiles},
		{kind: "c", files: pkg.CFiles},
		{kind: "cxx", files: pkg.CXXFiles},
		{kind: "objc", files: pkg.MFiles},
		{kind: "fortran", files: pkg.FFiles},
		{kind: "assembly", files: pkg.SFiles},
		{kind: "swig", files: pkg.SwigFiles},
		{kind: "swig-cxx", files: pkg.SwigCXXFiles},
		{kind: "object", files: pkg.SysoFiles},
	}
	for _, group := range groups {
		for _, name := range group.files {
			projected = append(projected, group.kind+":"+name)
		}
	}
	sort.Strings(projected)
	return projected
}

func validateCapabilityModule(module *CapabilityModule) error {
	if module == nil {
		return nil
	}
	if module.Path == "" && !module.Local {
		return fmt.Errorf("module identity is empty")
	}
	if module.Main && module.Local {
		return fmt.Errorf("main module cannot be a local replacement")
	}
	if module.Replacement != nil {
		if module.Replacement.Main || module.Replacement.Replacement != nil {
			return fmt.Errorf("module replacement is malformed")
		}
		if err := validateCapabilityModule(module.Replacement); err != nil {
			return err
		}
	}
	return nil
}

func listedCompatibilityPackage(pkg listedPackage) compatibility.Package {
	return compatibility.Package{ImportPath: pkg.ImportPath, Module: listedCompatibilityModule(pkg.Module)}
}

func listedCompatibilityModule(module *listedModule) compatibility.Module {
	if module == nil {
		return compatibility.Module{}
	}
	return compatibility.Module{
		Path:             module.Path,
		Version:          module.Version,
		Sum:              module.Sum,
		Replaced:         module.Replace != nil,
		LocalReplacement: module.Replace != nil && module.Replace.Dir != "",
	}
}

func capabilityCompatibilityPackage(pkg CapabilityPackage) compatibility.Package {
	sources := make([]compatibility.Source, len(pkg.Sources))
	for index, source := range pkg.Sources {
		sources[index] = compatibility.Source{Name: source.Name, SHA256: source.SHA256}
	}
	return compatibility.Package{
		ImportPath: pkg.ImportPath, Module: capabilityCompatibilityModule(pkg.Module), SourceSetSHA256: compatibility.DigestSources(sources),
	}
}

func capabilityCompatibilityModule(module *CapabilityModule) compatibility.Module {
	if module == nil {
		return compatibility.Module{}
	}
	return compatibility.Module{
		Path:             module.Path,
		Version:          module.Version,
		Sum:              module.Sum,
		Replaced:         module.Replacement != nil,
		LocalReplacement: module.Replacement != nil && module.Replacement.Local,
	}
}

func packageImports(pkg listedPackage) []string {
	return append([]string(nil), pkg.Imports...)
}

func packageSourceFiles(pkg listedPackage) []string {
	files := append([]string(nil), pkg.GoFiles...)
	files = append(files, pkg.CgoFiles...)
	return sortedSetCopy(files)
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

func sortedUnique(values []string) bool {
	for index, value := range values {
		if value == "" || index > 0 && values[index-1] >= value {
			return false
		}
	}
	return true
}

func sortedUniqueSources(sources []CapabilitySource) bool {
	for index, source := range sources {
		if index > 0 && sources[index-1].Name >= source.Name {
			return false
		}
	}
	return true
}

func sortedUniqueCompatibility(identities []compatibility.Identity) bool {
	for index, identity := range identities {
		_, digestErr := record.ParseSHA256(identity.SHA256)
		if identity.ID == "" || digestErr != nil || index > 0 && identities[index-1].ID >= identity.ID {
			return false
		}
	}
	return true
}

func recordCompatibility(identities []compatibility.Identity) []record.CompatibilityPack {
	result := make([]record.CompatibilityPack, len(identities))
	for index, identity := range identities {
		result[index] = record.CompatibilityPack{ID: identity.ID, SHA256: record.SHA256(identity.SHA256)}
	}
	return result
}

func VerifyCompatibility(packs []record.CompatibilityPack) error {
	if packs == nil {
		return errors.New("compatibility pack identity is missing")
	}
	identities := make([]compatibility.Identity, len(packs))
	for index, pack := range packs {
		identities[index] = compatibility.Identity{ID: pack.ID, SHA256: string(pack.SHA256)}
	}
	return compatibility.VerifyIdentities(identities)
}

func generatedTestMain(pkg listedPackage) bool {
	return pkg.Name == "main" && strings.HasSuffix(pkg.ImportPath, ".test") && len(pkg.GoFiles) == 1 && filepath.IsAbs(pkg.GoFiles[0])
}

func unsupportedCapability(importPath, capability string) error {
	return &UnsupportedCapabilityError{ImportPath: importPath, Capability: capability}
}
