// Package manifestgen derives a qualification-set manifest from the top-level
// tests of one Go package, so coverage of that package is a property of the
// generator rather than of hand curation: every test is a workload with the
// default expectation unless a checked-in spec overrides or excludes it by name.
package manifestgen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/qualification/set"
	"go.temporal.io/server/tools/gomad3/target"
)

const SpecSchema = "gomad3.qualification-set-generator/v1"

const maximumSpecBytes = 1 << 20

// Spec is the checked-in input of a generated manifest: the package to
// enumerate, the manifest and workload defaults, and the only per-test
// deviations the generator accepts.
type Spec struct {
	Schema string `json:"schema"`
	// Package is the enumerated package, relative to the generation root.
	Package string `json:"package"`
	// BuildTags select the package's files for enumeration and are the
	// workloads' build tags, so the listed tests are the built tests.
	BuildTags []string `json:"build_tags"`
	// Platforms are the GOOS/GOARCH pairs the package must enumerate the same
	// tests on; one manifest serves every qualified host.
	Platforms  []string                `json:"platforms"`
	IDPrefix   string                  `json:"id_prefix"`
	Manifest   ManifestDefaults        `json:"manifest"`
	Workload   WorkloadDefaults        `json:"workload"`
	Tests      map[string]TestOverride `json:"tests,omitempty"`
	Exclusions map[string]Exclusion    `json:"exclusions,omitempty"`
}

type ManifestDefaults struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Module               string   `json:"module"`
	Seeds                []uint64 `json:"seeds"`
	Repeat               uint64   `json:"repeat"`
	RunTimeout           string   `json:"run_timeout"`
	OverallTimeout       string   `json:"overall_timeout"`
	TerminateGrace       string   `json:"terminate_grace"`
	OutputBytes          uint64   `json:"output_bytes"`
	WorldTransitionBytes uint64   `json:"world_transition_bytes"`
}

type WorkloadDefaults struct {
	Tier           uint64                `json:"tier"`
	CapabilityMode target.CapabilityMode `json:"capability_mode"`
	// Invariant follows the test name in each workload's invariant.
	Invariant            string                  `json:"invariant"`
	ReadOnlyMounts       []set.Mount             `json:"read_only_mounts,omitempty"`
	ChoiceBytes          uint64                  `json:"choice_bytes"`
	ReplaySuccesses      bool                    `json:"replay_successes"`
	SuccessArtifactLimit uint64                  `json:"success_artifact_limit"`
	SuccessBytesLimit    uint64                  `json:"success_bytes_limit"`
	ExecutionTimeout     string                  `json:"execution_timeout,omitempty"`
	OverallTimeout       string                  `json:"overall_timeout,omitempty"`
	Expectation          set.WorkloadExpectation `json:"expectation"`
}

// TestOverride is the per-test deviation from the workload defaults.
type TestOverride struct {
	RequiredProbes       []string                           `json:"required_probes,omitempty"`
	Expectation          *set.WorkloadExpectation           `json:"expectation,omitempty"`
	PlatformExpectations map[string]set.WorkloadExpectation `json:"platform_expectations,omitempty"`
}

// Exclusion removes a test from the manifest by name. Owner, date, and reason
// are required so no test leaves the qualified set silently.
type Exclusion struct {
	Owner  string `json:"owner"`
	Date   string `json:"date"`
	Reason string `json:"reason"`
}

var testNamePattern = regexp.MustCompile(`^Test[A-Za-z0-9_]+$`)
var workloadIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// Config names the files of one generation. Relative Spec and Output paths
// resolve against Root, as does the spec's Package.
type Config struct {
	Root   string
	Spec   string
	Output string
	// Check compares the generated manifest with Output instead of writing it.
	Check bool
}

// Run generates the manifest described by config and writes it, or, in check
// mode, fails when Output differs from what the generator produces.
func Run(config Config) error {
	specPath, outputPath := resolve(config.Root, config.Spec), resolve(config.Root, config.Output)
	spec, err := LoadSpec(specPath)
	if err != nil {
		return err
	}
	tests, err := ListTests(resolve(config.Root, spec.Package), spec.BuildTags, spec.Platforms)
	if err != nil {
		return err
	}
	manifest, err := Generate(spec, tests)
	if err != nil {
		return err
	}
	contents, err := Encode(manifest)
	if err != nil {
		return err
	}
	if !config.Check {
		if err := hostfs.Replace(outputPath, contents, 0o644); err != nil {
			return fmt.Errorf("write generated qualification manifest: %w", err)
		}
		return nil
	}
	current, err := os.ReadFile(outputPath)
	if err != nil {
		return fmt.Errorf("generated qualification manifest %s is missing: %w", config.Output, err)
	}
	if !bytes.Equal(current, contents) {
		return fmt.Errorf("generated qualification manifest %s is stale relative to %s and %s%s; regenerate it with make -C tools/gomad3 generate",
			config.Output, spec.Package, config.Spec, staleDetail(current, manifest))
	}
	return nil
}

func resolve(root, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, filepath.FromSlash(path))
}

func staleDetail(current []byte, generated set.Manifest) string {
	var previous set.Manifest
	if err := json.Unmarshal(current, &previous); err != nil {
		return ""
	}
	tests := func(manifest set.Manifest) []string {
		names := make([]string, 0, len(manifest.Suites))
		for _, workload := range manifest.Suites {
			names = append(names, workload.Test)
		}
		return names
	}
	before, after := tests(previous), tests(generated)
	var added, removed []string
	for _, name := range after {
		if !slices.Contains(before, name) {
			added = append(added, name)
		}
	}
	for _, name := range before {
		if !slices.Contains(after, name) {
			removed = append(removed, name)
		}
	}
	slices.Sort(added)
	slices.Sort(removed)
	var detail []string
	if len(added) != 0 {
		detail = append(detail, "missing tests "+strings.Join(added, ", "))
	}
	if len(removed) != 0 {
		detail = append(detail, "removed tests "+strings.Join(removed, ", "))
	}
	if len(detail) == 0 {
		return ""
	}
	return " (" + strings.Join(detail, "; ") + ")"
}

func LoadSpec(path string) (Spec, error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		return Spec{}, fmt.Errorf("read qualification manifest spec: %w", err)
	}
	if info.Size() <= 0 || info.Size() > maximumSpecBytes {
		return Spec{}, errors.Join(fmt.Errorf("qualification manifest spec must be between 1 and %d bytes", maximumSpecBytes), file.Close())
	}
	contents, readErr := io.ReadAll(io.LimitReader(file, maximumSpecBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil {
		return Spec{}, errors.Join(fmt.Errorf("read qualification manifest spec: %w", readErr), closeErr)
	}
	var spec Spec
	if err := canonicaljson.StrictDecode(contents, &spec); err != nil {
		return Spec{}, fmt.Errorf("decode qualification manifest spec: %w", err)
	}
	if err := validateSpec(spec); err != nil {
		return Spec{}, err
	}
	return spec, nil
}

func validateSpec(spec Spec) error {
	if spec.Schema != SpecSchema {
		return fmt.Errorf("qualification manifest spec schema must be %s", SpecSchema)
	}
	packagePath := strings.TrimPrefix(spec.Package, "./")
	if !strings.HasPrefix(spec.Package, "./") || packagePath == "" || filepath.ToSlash(filepath.Clean(filepath.FromSlash(packagePath))) != packagePath || strings.HasPrefix(packagePath, "../") {
		return fmt.Errorf("qualification manifest spec package %q must be a clean ./ relative path", spec.Package)
	}
	if len(spec.Platforms) == 0 {
		return errors.New("qualification manifest spec names no platforms")
	}
	for _, platform := range spec.Platforms {
		if goos, goarch, ok := strings.Cut(platform, "/"); !ok || goos == "" || goarch == "" {
			return fmt.Errorf("qualification manifest spec platform is invalid: %q", platform)
		}
	}
	if !workloadIDPattern.MatchString(spec.IDPrefix) {
		return fmt.Errorf("qualification manifest spec id prefix is invalid: %q", spec.IDPrefix)
	}
	if strings.TrimSpace(spec.Workload.Invariant) == "" {
		return errors.New("qualification manifest spec workload invariant is required")
	}
	for name, exclusion := range spec.Exclusions {
		if strings.TrimSpace(exclusion.Owner) == "" {
			return fmt.Errorf("exclusion of %s requires an owner", name)
		}
		if _, err := time.Parse(time.DateOnly, exclusion.Date); err != nil {
			return fmt.Errorf("exclusion of %s requires a YYYY-MM-DD date: %w", name, err)
		}
		if strings.TrimSpace(exclusion.Reason) == "" {
			return fmt.Errorf("exclusion of %s requires a reason", name)
		}
		if _, overridden := spec.Tests[name]; overridden {
			return fmt.Errorf("%s is both excluded and overridden", name)
		}
	}
	return nil
}

// ListTests returns the sorted top-level test names `go test -list` reports
// for the package in dir under tags, on every platform. It parses the test
// files rather than building the test binary, applying the go command's
// file-selection and test-function rules, so it needs no toolchain and runs in
// milliseconds. A package whose tests differ between platforms is refused.
func ListTests(dir string, tags []string, platforms []string) ([]string, error) {
	var listed []string
	for index, platform := range platforms {
		goos, goarch, _ := strings.Cut(platform, "/")
		tests, err := listPlatformTests(dir, tags, goos, goarch)
		if err != nil {
			return nil, err
		}
		if index > 0 && !slices.Equal(listed, tests) {
			return nil, fmt.Errorf("package %s lists different tests on %s than on %s", dir, platform, platforms[0])
		}
		listed = tests
	}
	return listed, nil
}

// baselineFeatureTags are the architecture feature tags the go command sets
// for the default GOAMD64 and GOARM64 levels, which qualification builds use.
var baselineFeatureTags = map[string]string{
	"amd64": "amd64.v1",
	"arm64": "arm64.v8.0",
}

func listPlatformTests(dir string, tags []string, goos, goarch string) ([]string, error) {
	featureTag, known := baselineFeatureTags[goarch]
	if !known {
		return nil, fmt.Errorf("no baseline architecture feature tag is known for %s/%s", goos, goarch)
	}
	buildContext := build.Default
	buildContext.GOOS, buildContext.GOARCH = goos, goarch
	buildContext.BuildTags = slices.Clone(tags)
	// build.Default carries the host architecture's feature tags; swap them for
	// the target's so files constrained on, say, amd64.v1 match as go test would.
	buildContext.ToolTags = slices.DeleteFunc(slices.Clone(build.Default.ToolTags), func(tag string) bool {
		return strings.HasPrefix(tag, build.Default.GOARCH+".")
	})
	buildContext.ToolTags = append(buildContext.ToolTags, featureTag)
	// Qualification builds targets without cgo.
	buildContext.CgoEnabled = false
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read test package: %w", err)
	}
	files := token.NewFileSet()
	seen := map[string]string{}
	var tests []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		matched, err := buildContext.MatchFile(dir, name)
		if err != nil {
			return nil, fmt.Errorf("match test file %s: %w", name, err)
		}
		if !matched {
			continue
		}
		file, err := parser.ParseFile(files, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("parse test file: %w", err)
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil {
				continue
			}
			test := function.Name.Name
			if test == "TestMain" && !takesPointerTo(function, "T") || !isTestName(test) {
				continue
			}
			if function.Type.TypeParams != nil || !takesPointerTo(function, "T") {
				return nil, fmt.Errorf("%s in %s has the wrong signature for a test", test, name)
			}
			if previous, duplicate := seen[test]; duplicate {
				return nil, fmt.Errorf("%s is declared in both %s and %s", test, previous, name)
			}
			seen[test] = name
			tests = append(tests, test)
		}
	}
	slices.Sort(tests)
	return tests, nil
}

// isTestName mirrors the go command: Test, or Test followed by a rune that is
// not lower case.
func isTestName(name string) bool {
	if !strings.HasPrefix(name, "Test") {
		return false
	}
	if len(name) == len("Test") {
		return true
	}
	next, _ := utf8.DecodeRuneInString(name[len("Test"):])
	return !unicode.IsLower(next)
}

// takesPointerTo mirrors the go command's signature check, which accepts *T
// or *pkg.T because it cannot resolve how testing was imported.
func takesPointerTo(function *ast.FuncDecl, typeName string) bool {
	parameters := function.Type.Params.List
	if function.Type.Results != nil && len(function.Type.Results.List) > 0 || len(parameters) != 1 || len(parameters[0].Names) > 1 {
		return false
	}
	pointer, ok := parameters[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	switch named := pointer.X.(type) {
	case *ast.Ident:
		return named.Name == typeName
	case *ast.SelectorExpr:
		return named.Sel.Name == typeName
	}
	return false
}

// Generate derives the manifest: one workload per listed test that is not
// excluded, ordered by workload identity. Overrides or exclusions naming a
// test that is not listed are refused, so the spec cannot drift from the
// package.
func Generate(spec Spec, tests []string) (set.Manifest, error) {
	for name := range spec.Tests {
		if !slices.Contains(tests, name) {
			return set.Manifest{}, fmt.Errorf("override names %s, which %s does not declare", name, spec.Package)
		}
	}
	for name := range spec.Exclusions {
		if !slices.Contains(tests, name) {
			return set.Manifest{}, fmt.Errorf("exclusion names %s, which %s does not declare", name, spec.Package)
		}
	}
	defaults := spec.Manifest
	manifest := set.Manifest{
		Schema: set.ManifestSchema, Name: defaults.Name, Description: defaults.Description, Module: defaults.Module,
		Seeds: slices.Clone(defaults.Seeds), Repeat: defaults.Repeat, RunTimeout: defaults.RunTimeout,
		OverallTimeout: defaults.OverallTimeout, TerminateGrace: defaults.TerminateGrace,
		OutputBytes: defaults.OutputBytes, WorldTransitionBytes: defaults.WorldTransitionBytes,
		Suites: []set.Workload{},
	}
	owners := map[string]string{}
	for _, test := range tests {
		if _, excluded := spec.Exclusions[test]; excluded {
			continue
		}
		if !testNamePattern.MatchString(test) {
			return set.Manifest{}, fmt.Errorf("test name %s cannot identify a qualification workload", test)
		}
		id := WorkloadID(spec.IDPrefix, test)
		if previous, collides := owners[id]; collides {
			return set.Manifest{}, fmt.Errorf("tests %s and %s share workload identity %s", previous, test, id)
		}
		owners[id] = test
		manifest.Suites = append(manifest.Suites, workload(spec, test, id))
	}
	slices.SortFunc(manifest.Suites, func(left, right set.Workload) int { return strings.Compare(left.ID, right.ID) })
	if len(manifest.Seeds) != 0 {
		manifest.Seed = manifest.Seeds[0]
	}
	if err := set.ValidateManifest(manifest); err != nil {
		return set.Manifest{}, fmt.Errorf("generated qualification manifest is invalid: %w", err)
	}
	return manifest, nil
}

func workload(spec Spec, test, id string) set.Workload {
	defaults := spec.Workload
	generated := set.Workload{
		ID: id, Name: test, Tier: defaults.Tier, Invariant: test + " " + defaults.Invariant,
		Package: spec.Package, Test: test, BuildTags: slices.Clone(spec.BuildTags),
		CapabilityMode: defaults.CapabilityMode, ReadOnlyMounts: slices.Clone(defaults.ReadOnlyMounts),
		ChoiceBytes: defaults.ChoiceBytes, ReplaySuccesses: defaults.ReplaySuccesses,
		SuccessArtifactLimit: defaults.SuccessArtifactLimit, SuccessBytesLimit: defaults.SuccessBytesLimit,
		ExecutionTimeout: defaults.ExecutionTimeout, OverallTimeout: defaults.OverallTimeout,
		Expectation: defaults.Expectation,
	}
	if override, found := spec.Tests[test]; found {
		generated.RequiredProbes = slices.Clone(override.RequiredProbes)
		if override.Expectation != nil {
			generated.Expectation = *override.Expectation
		}
		if len(override.PlatformExpectations) != 0 {
			generated.PlatformExpectations = maps.Clone(override.PlatformExpectations)
		}
	}
	return generated
}

// WorkloadID derives a stable workload identity from a test name: the prefix,
// then the name without Test in lower-case words, split at case changes and
// underscores.
func WorkloadID(prefix, test string) string {
	runes := []rune(strings.TrimPrefix(test, "Test"))
	var id strings.Builder
	id.WriteString(prefix)
	separate := true
	for index, current := range runes {
		if current == '_' {
			separate = true
			continue
		}
		if unicode.IsUpper(current) && index > 0 {
			previous := runes[index-1]
			nextIsLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) || unicode.IsUpper(previous) && nextIsLower {
				separate = true
			}
		}
		if separate {
			id.WriteByte('-')
			separate = false
		}
		id.WriteRune(unicode.ToLower(current))
	}
	return id.String()
}

// Encode renders the manifest as indented JSON in field order, so regeneration
// is byte-stable and reviewable.
func Encode(manifest set.Manifest) ([]byte, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return nil, fmt.Errorf("encode generated qualification manifest: %w", err)
	}
	return output.Bytes(), nil
}
