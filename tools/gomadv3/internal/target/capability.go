package target

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const capabilityClosureSchema = "gomadv3.target-capability-closure/v1"

type CapabilityClosure struct {
	Schema   string              `json:"schema"`
	Packages []CapabilityPackage `json:"packages"`
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

func validateGoCapabilityClosure(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string) error {
	closure, err := reviewGoCapabilityClosure(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	if err != nil {
		return err
	}
	return validateCapabilityReview(closure)
}

func ReviewCapabilityClosure(ctx context.Context, spec Spec) (CapabilityClosure, error) {
	if spec.Kind != KindGoRun && spec.Kind != KindGoTest {
		return CapabilityClosure{}, fmt.Errorf("capability review requires a go-run or go-test target")
	}
	tags, err := normalizeBuildTags(spec.Kind, spec.BuildTags)
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

type reviewedModule struct {
	version string
	sum     string
}

var reviewedAdapterModules = map[string][]reviewedModule{
	"github.com/mattn/go-isatty": {
		{version: "v0.0.20", sum: "h1:xfD0iDuEKnDkl03q4limB+vH+GxLEtL/jb4xVJSWWEY="},
		{version: "v0.0.21", sum: "h1:xYae+lCNBP7QuW4PUnNG61ffM4hVIfm+zUzDuSzYLGs="},
	},
	"github.com/remyoudompheng/bigfft": {{version: "v0.0.0-20230129092748-24d4a6f8daec", sum: "h1:W09IVJc94icq4NjY3clb7Lk8O1qJ8BdBEF8z0ibU0rE="}},
	"golang.org/x/sys": {
		{version: "v0.41.0", sum: "h1:Ivj+2Cp/ylzLiEU89QhWblYnOE9zerudt9Ftecq2C6k="},
		{version: "v0.47.0", sum: "h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs="},
	},
	"modernc.org/libc":   {{version: gomadversion.ModerncLibcVersion, sum: gomadversion.ModerncLibcSum}},
	"modernc.org/memory": {{version: "v1.11.0", sum: "h1:o4QC8aMQzmcwCK3t3Ux/ZHmwFPzE6hf2Y5LbkRs+hbI="}},
	"modernc.org/sqlite": {{version: "v1.51.0", sum: "h1:aH/MMSoayAIhozZ7uJbVTT9QO/VhzBf0J9tymmmuC/U="}},
}

var reviewedLinknameSources = map[string][]string{
	"go_above_118.go": {"mapiterinit reflect.mapiterinit"},
	"go_above_19.go":  {"resolveTypeOff reflect.resolveTypeOff", "makemap reflect.makemap"},
	"type_map.go":     {"typelinks2 reflect.typelinks"},
	"unsafe_link.go": {
		"unsafe_New reflect.unsafe_New",
		"typedmemmove reflect.typedmemmove",
		"unsafe_NewArray reflect.unsafe_NewArray",
		"typedslicecopy reflect.typedslicecopy",
		"mapassign reflect.mapassign",
		"mapaccess reflect.mapaccess",
		"mapiternext reflect.mapiternext",
		"ifaceE2I reflect.ifaceE2I",
	},
}

func validateCapabilityClosure(packages []listedPackage) error {
	closure, err := projectCapabilityClosure(packages, nil)
	if err != nil {
		return err
	}
	return validateCapabilityReview(closure)
}

func projectCapabilityClosure(packages []listedPackage, overlay map[string]string) (CapabilityClosure, error) {
	adapterEnabled := false
	for _, pkg := range packages {
		if pkg.ImportPath == "modernc.org/libc" && moduleMatches(pkg.Module, "modernc.org/libc", gomadversion.ModerncLibcVersion) && pkg.Module.Replace != nil && pkg.Module.Replace.Dir != "" {
			adapterEnabled = true
			break
		}
	}
	closure := CapabilityClosure{Schema: capabilityClosureSchema, Packages: make([]CapabilityPackage, 0, len(packages))}
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
		if pkg.Standard || reviewedAdapterPackage(pkg, adapterEnabled) || projected.GeneratedTestMain {
			closure.Packages = append(closure.Packages, projected)
			continue
		}
		if pkg.ForTest == "" && len(sourceFiles) == 0 && len(projected.ForeignSources) == 0 && (len(pkg.TestGoFiles) != 0 || len(pkg.XTestGoFiles) != 0) {
			continue
		}
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
			if bytes.Contains(contents, []byte("//go:linkname")) && !reviewedLinknameSource(pkg, name, contents) {
				return CapabilityClosure{}, unsupportedCapability(pkg.ImportPath, "uses go:linkname in "+name)
			}
			hash := sha256.Sum256(contents)
			projected.Sources = append(projected.Sources, CapabilitySource{Name: name, SHA256: fmt.Sprintf("sha256:%x", hash)})
		}
		sort.Slice(projected.Sources, func(i, j int) bool { return projected.Sources[i].Name < projected.Sources[j].Name })
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

func reviewedLinknameSource(pkg listedPackage, name string, contents []byte) bool {
	const (
		path    = "github.com/modern-go/reflect2"
		version = "v1.0.3-0.20250322232337-35a7c28c31ee"
		sum     = "h1:W5t00kpgFdJifH4BDsTlE89Zl93FEloxaWZfGcifgq8="
	)
	if pkg.ImportPath != path || !moduleMatches(pkg.Module, path, version) || pkg.Module.Sum != sum || pkg.Module.Replace != nil {
		return false
	}
	wanted, found := reviewedLinknameSources[name]
	if !found {
		return false
	}
	observed, valid := linknameDirectives(contents)
	if !valid || len(observed) != len(wanted) {
		return false
	}
	for index := range observed {
		if observed[index] != wanted[index] {
			return false
		}
	}
	return true
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
	if closure.Schema != capabilityClosureSchema || len(closure.Packages) == 0 {
		return fmt.Errorf("unsupported or empty target capability closure")
	}
	adapterEnabled := false
	for _, pkg := range closure.Packages {
		if pkg.ImportPath == "modernc.org/libc" && capabilityModuleMatches(pkg.Module, "modernc.org/libc", gomadversion.ModerncLibcVersion) && pkg.Module.Replacement != nil && pkg.Module.Replacement.Local {
			adapterEnabled = true
			break
		}
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
			if filepath.Base(source.Name) != source.Name || source.Name == "" || !validSHA256(source.SHA256) {
				return fmt.Errorf("target capability closure package %s has invalid source evidence", pkg.ImportPath)
			}
		}
		if pkg.Name == "main" {
			mainPackage = true
		}
		if pkg.GeneratedTestMain && (pkg.Name != "main" || !strings.HasSuffix(pkg.ImportPath, ".test") || pkg.Standard || pkg.Module != nil && !pkg.Module.Main || len(pkg.Sources) != 0 || len(pkg.ForeignSources) != 0) {
			return fmt.Errorf("target capability closure package %s has invalid generated test-main evidence", pkg.ImportPath)
		}
		if pkg.Standard || reviewedCapabilityPackage(pkg, adapterEnabled) {
			continue
		}
		for _, imported := range pkg.Imports {
			if imported == "syscall" || imported == "os/exec" || imported == "os/signal" || imported == "os/user" || imported == "plugin" || imported == "runtime/cgo" || strings.HasPrefix(imported, "golang.org/x/sys/") {
				return unsupportedCapability(pkg.ImportPath, "imports "+imported)
			}
		}
		if len(pkg.ForeignSources) != 0 {
			return unsupportedCapability(pkg.ImportPath, "contains foreign or assembly source "+pkg.ForeignSources[0])
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

func reviewedCapabilityPackage(pkg CapabilityPackage, enabled bool) bool {
	if !enabled || pkg.Module == nil || pkg.Module.Main {
		return false
	}
	wanted, found := reviewedAdapterModules[pkg.Module.Path]
	if !found {
		return false
	}
	for _, want := range wanted {
		if !capabilityModuleMatches(pkg.Module, pkg.Module.Path, want.version) {
			continue
		}
		if pkg.Module.Path == "modernc.org/libc" {
			return pkg.Module.Replacement != nil && pkg.Module.Replacement.Local
		}
		return pkg.Module.Sum == want.sum
	}
	return false
}

func capabilityModuleMatches(module *CapabilityModule, path, version string) bool {
	return module != nil && module.Path == path && module.Version == version
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

func validSHA256(value string) bool {
	if !strings.HasPrefix(value, "sha256:") || len(value) != len("sha256:")+sha256.Size*2 {
		return false
	}
	for _, character := range strings.TrimPrefix(value, "sha256:") {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}

func generatedTestMain(pkg listedPackage) bool {
	return pkg.Name == "main" && strings.HasSuffix(pkg.ImportPath, ".test") && len(pkg.GoFiles) == 1 && filepath.IsAbs(pkg.GoFiles[0])
}

func reviewedAdapterPackage(pkg listedPackage, enabled bool) bool {
	if !enabled || pkg.Module == nil || pkg.Module.Main {
		return false
	}
	wanted, found := reviewedAdapterModules[pkg.Module.Path]
	if !found {
		return false
	}
	for _, want := range wanted {
		if !moduleMatches(pkg.Module, pkg.Module.Path, want.version) {
			continue
		}
		if pkg.Module.Path == "modernc.org/libc" {
			return pkg.Module.Replace != nil && pkg.Module.Replace.Dir != ""
		}
		return pkg.Module.Sum == want.sum
	}
	return false
}

func moduleMatches(module *listedModule, path, version string) bool {
	return module != nil && module.Path == path && module.Version == version
}

func unsupportedCapability(importPath, capability string) error {
	return &UnsupportedCapabilityError{ImportPath: importPath, Capability: capability}
}
