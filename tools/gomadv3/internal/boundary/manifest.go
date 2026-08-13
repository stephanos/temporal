package boundary

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomadv3/internal/safefile"
)

const manifestPath = "boundary/manifest.json"

const compilerTestManifestPath = "boundary/compiler-tests.json"

const maximumSemanticProbes = 256

var (
	goVersionPattern = regexp.MustCompile(`^go([1-9][0-9]*)\.([0-9]+)\.[0-9]+$`)
	probePattern     = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)
	sha256Pattern    = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
)

type manifest struct {
	SchemaVersion      uint                `json:"schema_version"`
	ManifestVersion    string              `json:"manifest_version"`
	GoVersion          string              `json:"go_version"`
	Platforms          []string            `json:"platforms"`
	HookPolicies       []hookPolicy        `json:"hook_policies,omitempty"`
	Intercepts         []intercept         `json:"intercepts"`
	ReviewedCandidates []reviewedCandidate `json:"reviewed_candidates"`
}

type compilerTestManifest struct {
	SchemaVersion uint           `json:"schema_version"`
	GoVersion     string         `json:"go_version"`
	Tests         []compilerTest `json:"tests"`
}

type reviewedCandidate struct {
	Target      string   `json:"target"`
	Disposition string   `json:"disposition"`
	Boundaries  []string `json:"boundaries"`
}

type receiver struct {
	Name    string `json:"name"`
	Pointer bool   `json:"pointer"`
}

type hookPolicy struct {
	ID               string       `json:"id"`
	Package          string       `json:"package"`
	Output           string       `json:"output"`
	Imports          []hookImport `json:"imports"`
	Enabled          string       `json:"enabled"`
	DisabledFallback string       `json:"disabled_fallback"`
	Transcript       string       `json:"transcript"`
	ResultValues     string       `json:"result_values"`
	UnsupportedError string       `json:"unsupported_error"`
	ErrorWrapping    string       `json:"error_wrapping"`
}

type hookImport struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type intercept struct {
	Package             string    `json:"package"`
	Receiver            *receiver `json:"receiver,omitempty"`
	Symbol              string    `json:"symbol"`
	Signature           string    `json:"signature"`
	Source              string    `json:"source"`
	DeclarationSHA256   string    `json:"declaration_sha256"`
	PackageSHA256       string    `json:"package_sha256"`
	Operation           string    `json:"operation"`
	Probe               string    `json:"probe"`
	Disposition         string    `json:"disposition"`
	Hook                string    `json:"hook"`
	HookPolicy          string    `json:"hook_policy,omitempty"`
	DelegatedBoundary   string    `json:"delegated_boundary"`
	Adapters            []string  `json:"adapters"`
	ConformanceFixtures []string  `json:"conformance_fixtures"`
	NegativeFixtures    []string  `json:"negative_fixtures"`
	EscapeFixtures      []string  `json:"escape_fixtures"`
}

type compilerTest struct {
	Case              string    `json:"case"`
	Package           string    `json:"package"`
	Receiver          *receiver `json:"receiver,omitempty"`
	Symbol            string    `json:"symbol"`
	Hook              string    `json:"hook"`
	DeclarationSHA256 string    `json:"declaration_sha256,omitempty"`
	Diagnostic        string    `json:"diagnostic,omitempty"`
}

type CompilerTestCase struct {
	Case       string
	Package    string
	Diagnostic string
}

type artifact struct {
	path    string
	content []byte
}

// Generate validates the canonical boundary manifest and writes or checks all
// derived interception artifacts.
func Generate(root string, check bool) error {
	definition, err := load(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return err
	}
	artifacts, err := render(definition)
	if err != nil {
		return err
	}
	for _, generated := range artifacts {
		path := filepath.Join(root, filepath.FromSlash(generated.path))
		if check {
			current, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(current, generated.content) {
				return fmt.Errorf("generated boundary artifact is stale: %s", generated.path)
			}
			continue
		}
		if err := safefile.Replace(path, generated.content, 0o644); err != nil {
			return fmt.Errorf("write boundary artifact: %w", err)
		}
	}
	return nil
}

// CheckCompilerTests validates the conformance-only compiler interception
// manifest against the production boundary's pinned Go version.
func CheckCompilerTests(root string) error {
	definition, err := load(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return err
	}
	_, err = loadCompilerTests(filepath.Join(root, filepath.FromSlash(compilerTestManifestPath)), definition.GoVersion)
	return err
}

func CompilerTestCases(root string) ([]CompilerTestCase, error) {
	definition, err := load(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return nil, err
	}
	tests, err := loadCompilerTests(filepath.Join(root, filepath.FromSlash(compilerTestManifestPath)), definition.GoVersion)
	if err != nil {
		return nil, err
	}
	result := make([]CompilerTestCase, len(tests))
	for index, test := range tests {
		result[index] = CompilerTestCase{Case: test.Case, Package: test.Package, Diagnostic: test.Diagnostic}
	}
	return result, nil
}

// GenerateCompilerTestOverlay emits an overlay for building a compiler that
// contains conformance-only interception entries. The installed production
// spec must match the current boundary manifest before it can be replaced.
func GenerateCompilerTestOverlay(root, goroot, output string) error {
	definition, err := load(filepath.Join(root, filepath.FromSlash(manifestPath)))
	if err != nil {
		return err
	}
	tests, err := loadCompilerTests(filepath.Join(root, filepath.FromSlash(compilerTestManifestPath)), definition.GoVersion)
	if err != nil {
		return err
	}
	identity, err := manifestIdentity(definition)
	if err != nil {
		return err
	}
	productionSpec, err := renderCompilerSpec(definition, nil, identity)
	if err != nil {
		return err
	}
	relativeSpecPath, err := compilerSpecPath(definition.GoVersion)
	if err != nil {
		return err
	}
	installedSpecPath := filepath.Join(goroot, filepath.FromSlash(strings.TrimPrefix(relativeSpecPath, "overlay/")))
	installedSpec, err := os.ReadFile(installedSpecPath)
	if err != nil {
		return fmt.Errorf("read installed production compiler spec: %w", err)
	}
	if !bytes.Equal(installedSpec, productionSpec) {
		return errors.New("installed production compiler spec does not match the boundary manifest")
	}
	testSpec, err := renderCompilerSpec(definition, tests, identity)
	if err != nil {
		return err
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return fmt.Errorf("resolve compiler test overlay directory: %w", err)
	}
	if output == string(filepath.Separator) {
		return errors.New("compiler test overlay directory cannot be the filesystem root")
	}
	if err := os.MkdirAll(output, 0o700); err != nil {
		return fmt.Errorf("create compiler test overlay directory: %w", err)
	}
	info, err := os.Lstat(output)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("compiler test overlay path is not a directory: %s", output)
	}
	testSpecPath := filepath.Join(output, filepath.Base(relativeSpecPath))
	if err := safefile.Replace(testSpecPath, testSpec, 0o644); err != nil {
		return fmt.Errorf("write compiler test spec: %w", err)
	}
	overlay, err := json.MarshalIndent(struct {
		Replace map[string]string `json:"Replace"`
	}{Replace: map[string]string{installedSpecPath: testSpecPath}}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode compiler test overlay: %w", err)
	}
	overlay = append(overlay, '\n')
	if err := safefile.Replace(filepath.Join(output, "overlay.json"), overlay, 0o644); err != nil {
		return fmt.Errorf("write compiler test overlay: %w", err)
	}
	return nil
}

func load(path string) (manifest, error) {
	definition, err := decode(path)
	if err != nil {
		return manifest{}, err
	}
	if err := validate(definition); err != nil {
		return manifest{}, err
	}
	return definition, nil
}

func decode(path string) (manifest, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, fmt.Errorf("read boundary manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition manifest
	if err := decoder.Decode(&definition); err != nil {
		return manifest{}, fmt.Errorf("decode boundary manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return manifest{}, errors.New("boundary manifest has trailing data")
	}
	return definition, nil
}

func loadCompilerTests(path, goVersion string) ([]compilerTest, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read compiler test manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var definition compilerTestManifest
	if err := decoder.Decode(&definition); err != nil {
		return nil, fmt.Errorf("decode compiler test manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, errors.New("compiler test manifest has trailing data")
	}
	if definition.SchemaVersion != 1 {
		return nil, fmt.Errorf("compiler test manifest schema version %d is unsupported", definition.SchemaVersion)
	}
	if definition.GoVersion != goVersion {
		return nil, fmt.Errorf("compiler test manifest Go version %q does not match %q", definition.GoVersion, goVersion)
	}
	if len(definition.Tests) == 0 {
		return nil, errors.New("compiler test manifest has no tests")
	}
	for index, fixture := range definition.Tests {
		if fixture.Case == "" || fixture.Package == "" || fixture.Symbol == "" || fixture.Hook == "" {
			return nil, fmt.Errorf("compiler test manifest entry %d is incomplete", index+1)
		}
		if err := validateReceiver(fixture.Receiver); err != nil {
			return nil, fmt.Errorf("compiler test manifest entry %d: %w", index+1, err)
		}
		if fixture.DeclarationSHA256 != "" && !sha256Pattern.MatchString(fixture.DeclarationSHA256) {
			return nil, fmt.Errorf("compiler test manifest entry %d has an invalid declaration fingerprint", index+1)
		}
	}
	return append([]compilerTest(nil), definition.Tests...), nil
}

func validate(definition manifest) error {
	if definition.SchemaVersion != 1 {
		return fmt.Errorf("boundary manifest schema version %d is unsupported", definition.SchemaVersion)
	}
	if definition.ManifestVersion == "" {
		return errors.New("boundary manifest version is empty")
	}
	if goVersionPattern.FindStringSubmatch(definition.GoVersion) == nil {
		return fmt.Errorf("boundary manifest Go version is invalid: %q", definition.GoVersion)
	}
	if len(definition.Platforms) == 0 {
		return errors.New("boundary manifest has no platforms")
	}
	seenPlatforms := make(map[string]struct{}, len(definition.Platforms))
	for _, platform := range definition.Platforms {
		if !strings.Contains(platform, "/") {
			return fmt.Errorf("boundary manifest platform is invalid: %q", platform)
		}
		if _, duplicate := seenPlatforms[platform]; duplicate {
			return fmt.Errorf("boundary manifest platform is duplicated: %s", platform)
		}
		seenPlatforms[platform] = struct{}{}
	}
	if len(definition.Intercepts) == 0 {
		return errors.New("boundary manifest has no interceptions")
	}
	if len(definition.Intercepts) > maximumSemanticProbes {
		return fmt.Errorf("boundary manifest has %d interceptions, maximum is %d", len(definition.Intercepts), maximumSemanticProbes)
	}
	seenTargets := make(map[string]struct{}, len(definition.Intercepts))
	policiesByID := make(map[string]hookPolicy, len(definition.HookPolicies))
	policyUses := make(map[string]int, len(definition.HookPolicies))
	seenOutputs := make(map[string]struct{}, len(definition.HookPolicies))
	for index, policy := range definition.HookPolicies {
		if err := validateHookPolicy(policy); err != nil {
			return fmt.Errorf("boundary manifest hook policy %d: %w", index+1, err)
		}
		if _, duplicate := policiesByID[policy.ID]; duplicate {
			return fmt.Errorf("boundary manifest hook policy is duplicated: %s", policy.ID)
		}
		if _, duplicate := seenOutputs[policy.Output]; duplicate {
			return fmt.Errorf("boundary manifest hook policy output is duplicated: %s", policy.Output)
		}
		policiesByID[policy.ID] = policy
		seenOutputs[policy.Output] = struct{}{}
	}
	seenProbes := make(map[string]struct{}, len(definition.Intercepts))
	seenProbeIDs := make(map[uint64]string, len(definition.Intercepts))
	seenHooks := make(map[string]struct{}, len(definition.Intercepts))
	entriesByProbe := make(map[string]intercept, len(definition.Intercepts))
	for index, entry := range definition.Intercepts {
		if err := validateIntercept(entry); err != nil {
			return fmt.Errorf("boundary manifest interception %d: %w", index+1, err)
		}
		target := entry.Package + "." + targetName(entry.Receiver, entry.Symbol)
		if _, duplicate := seenTargets[target]; duplicate {
			return fmt.Errorf("boundary manifest target is duplicated: %s", target)
		}
		seenTargets[target] = struct{}{}
		if _, duplicate := seenHooks[entry.Hook]; duplicate {
			return fmt.Errorf("boundary manifest hook is duplicated: %s", entry.Hook)
		}
		seenHooks[entry.Hook] = struct{}{}
		if entry.HookPolicy != "" {
			policy, found := policiesByID[entry.HookPolicy]
			if !found {
				return fmt.Errorf("boundary manifest interception %s has an unknown hook policy %q", target, entry.HookPolicy)
			}
			if entry.Disposition != "deny" || entry.Package != policy.Package {
				return fmt.Errorf("boundary manifest interception %s is incompatible with hook policy %q", target, entry.HookPolicy)
			}
			if _, _, err := hookSignature(entry); err != nil {
				return fmt.Errorf("boundary manifest interception %s cannot generate its hook: %w", target, err)
			}
			policyUses[entry.HookPolicy]++
		}
		if _, duplicate := seenProbes[entry.Probe]; duplicate {
			return fmt.Errorf("boundary manifest probe is duplicated: %s", entry.Probe)
		}
		seenProbes[entry.Probe] = struct{}{}
		id := boundaryProbeID(entry.Probe)
		if id == 0 {
			return fmt.Errorf("boundary manifest probe %s has reserved semantic ID zero", entry.Probe)
		}
		if previous, collision := seenProbeIDs[id]; collision {
			return fmt.Errorf("boundary manifest probes %s and %s have the same semantic ID", previous, entry.Probe)
		}
		seenProbeIDs[id] = entry.Probe
		entriesByProbe[entry.Probe] = entry
	}
	for _, policy := range definition.HookPolicies {
		if policyUses[policy.ID] == 0 {
			return fmt.Errorf("boundary manifest hook policy %q is unused", policy.ID)
		}
	}
	for _, entry := range definition.Intercepts {
		if err := validateDelegateChain(entry, entriesByProbe); err != nil {
			return err
		}
	}
	seenCandidates := make(map[string]struct{}, len(definition.ReviewedCandidates))
	for index, candidate := range definition.ReviewedCandidates {
		if candidate.Target == "" {
			return fmt.Errorf("boundary manifest reviewed candidate %d has no target", index+1)
		}
		if _, intercepted := seenTargets[candidate.Target]; intercepted {
			return fmt.Errorf("boundary manifest reviewed candidate is already intercepted: %s", candidate.Target)
		}
		if _, duplicate := seenCandidates[candidate.Target]; duplicate {
			return fmt.Errorf("boundary manifest reviewed candidate is duplicated: %s", candidate.Target)
		}
		seenCandidates[candidate.Target] = struct{}{}
		switch candidate.Disposition {
		case "delegate", "dynamic", "unreachable":
			if len(candidate.Boundaries) == 0 {
				return fmt.Errorf("boundary manifest reviewed candidate %s has no controlling boundary", candidate.Target)
			}
			for _, probe := range candidate.Boundaries {
				if _, found := entriesByProbe[probe]; !found {
					return fmt.Errorf("boundary manifest reviewed candidate %s has an unresolved boundary: %s", candidate.Target, probe)
				}
			}
		case "patch", "upstream":
			if len(candidate.Boundaries) != 0 {
				return fmt.Errorf("boundary manifest reviewed candidate %s has controlling boundaries", candidate.Target)
			}
		default:
			return fmt.Errorf("boundary manifest reviewed candidate %s has invalid disposition %q", candidate.Target, candidate.Disposition)
		}
	}
	return nil
}

func validateHookPolicy(policy hookPolicy) error {
	if policy.ID == "" || policy.Package == "" || policy.Output == "" {
		return errors.New("ID, package, and output are required")
	}
	prefix := "overlay/src/" + policy.Package + "/"
	if !strings.HasPrefix(policy.Output, prefix) || filepath.ToSlash(filepath.Clean(policy.Output)) != policy.Output || !strings.HasSuffix(policy.Output, ".go") {
		return fmt.Errorf("output must be a Go file below %s", prefix)
	}
	previous := ""
	for _, imported := range policy.Imports {
		if imported.Name == "" || imported.Path == "" || imported.Name <= previous {
			return errors.New("imports must have names and paths and be sorted by unique name")
		}
		previous = imported.Name
	}
	for name, expression := range map[string]string{"enabled expression": policy.Enabled, "unsupported error": policy.UnsupportedError} {
		if _, err := parser.ParseExpr(expression); err != nil {
			return fmt.Errorf("invalid %s: %w", name, err)
		}
	}
	if policy.DisabledFallback != "upstream" || policy.Transcript != "compiler-probe" || policy.ResultValues != "zero" || policy.ErrorWrapping != "none" {
		return errors.New("only upstream fallback, compiler-probe transcripts, zero results, and unwrapped errors can be generated")
	}
	return nil
}

func validateDelegateChain(entry intercept, entriesByProbe map[string]intercept) error {
	seen := make(map[string]struct{})
	for entry.Disposition == "delegate" {
		if _, cycle := seen[entry.Probe]; cycle {
			return fmt.Errorf("boundary delegate chain contains a cycle at %s", entry.Probe)
		}
		seen[entry.Probe] = struct{}{}
		next, found := entriesByProbe[entry.DelegatedBoundary]
		if !found {
			return fmt.Errorf("boundary delegate %s is unresolved: %s", entry.Probe, entry.DelegatedBoundary)
		}
		entry = next
	}
	return nil
}

func validateIntercept(entry intercept) error {
	if entry.Package == "" || entry.Symbol == "" || entry.Hook == "" {
		return errors.New("package, symbol, and hook are required")
	}
	if err := validateReceiver(entry.Receiver); err != nil {
		return err
	}
	if !strings.HasPrefix(entry.Signature, "func(") {
		return errors.New("exact signature is required")
	}
	if !strings.HasPrefix(entry.Source, entry.Package+"/") {
		return errors.New("qualified source path is required")
	}
	if !sha256Pattern.MatchString(entry.DeclarationSHA256) || !sha256Pattern.MatchString(entry.PackageSHA256) {
		return errors.New("qualified source fingerprints are required")
	}
	if entry.Operation == "" {
		return errors.New("semantic operation is required")
	}
	if !probePattern.MatchString(entry.Probe) {
		return fmt.Errorf("probe ID is invalid: %q", entry.Probe)
	}
	switch entry.Disposition {
	case "model", "deny":
		if entry.DelegatedBoundary != "" {
			return fmt.Errorf("%s interception has a delegated boundary", entry.Disposition)
		}
	case "delegate":
		if entry.DelegatedBoundary == "" {
			return errors.New("delegate interception has no delegated boundary")
		}
	default:
		return fmt.Errorf("disposition is invalid: %q", entry.Disposition)
	}
	if len(entry.Adapters) == 0 {
		return errors.New("permitted adapters are required")
	}
	if len(entry.ConformanceFixtures) == 0 {
		return errors.New("a conformance fixture is required")
	}
	return nil
}

func validateReceiver(value *receiver) error {
	if value != nil && value.Name == "" {
		return errors.New("receiver name is empty")
	}
	return nil
}

func render(definition manifest) ([]artifact, error) {
	specPath, err := compilerSpecPath(definition.GoVersion)
	if err != nil {
		return nil, err
	}
	identity, err := manifestIdentity(definition)
	if err != nil {
		return nil, err
	}
	spec, err := renderCompilerSpec(definition, nil, identity)
	if err != nil {
		return nil, err
	}
	hostIdentity, err := renderHostIdentity(definition, identity)
	if err != nil {
		return nil, err
	}
	platformName := strings.ReplaceAll(strings.Join(definition.Platforms, "+"), "/", "-")
	artifacts := []artifact{
		{path: specPath, content: spec},
		{path: "expected-intercepts-" + definition.GoVersion + ".txt", content: renderExpectedReport(definition)},
		{path: filepath.ToSlash(filepath.Join("boundary", definition.GoVersion+"-"+platformName+".md")), content: renderInventory(definition, identity)},
		{path: "internal/ioprofile/boundary_generated.go", content: hostIdentity},
	}
	for _, policy := range definition.HookPolicies {
		generated, renderErr := renderHookPolicy(policy, definition.Intercepts)
		if renderErr != nil {
			return nil, renderErr
		}
		artifacts = append(artifacts, artifact{path: policy.Output, content: generated})
	}
	return artifacts, nil
}

func renderHookPolicy(policy hookPolicy, intercepts []intercept) ([]byte, error) {
	var source strings.Builder
	source.WriteString("// Copyright 2026 The Go Authors. All rights reserved.\n")
	source.WriteString("// Use of this source code is governed by a BSD-style\n")
	source.WriteString("// license that can be found in the LICENSE file.\n\n")
	source.WriteString("// Code generated by internal/boundarygen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&source, "package %s\n\n", policy.Package)
	if len(policy.Imports) != 0 {
		source.WriteString("import (\n")
		for _, imported := range policy.Imports {
			if imported.Name == path.Base(imported.Path) {
				fmt.Fprintf(&source, "\t%q\n", imported.Path)
			} else {
				fmt.Fprintf(&source, "\t%s %q\n", imported.Name, imported.Path)
			}
		}
		source.WriteString(")\n\n")
	}
	for _, entry := range intercepts {
		if entry.HookPolicy != policy.ID {
			continue
		}
		parameters, results, err := hookSignature(entry)
		if err != nil {
			return nil, err
		}
		fmt.Fprintf(&source, "func %s(%s) (%s, bool) {\n", entry.Hook, strings.Join(parameters, ", "), strings.Join(results, ", "))
		for index, result := range results[:len(results)-1] {
			fmt.Fprintf(&source, "\tvar result%d %s\n", index, result)
		}
		disabled := make([]string, 0, len(results)+1)
		unsupported := make([]string, 0, len(results)+1)
		for index := range results[:len(results)-1] {
			disabled = append(disabled, fmt.Sprintf("result%d", index))
			unsupported = append(unsupported, fmt.Sprintf("result%d", index))
		}
		disabled = append(disabled, "nil", "false")
		unsupported = append(unsupported, policy.UnsupportedError, "true")
		fmt.Fprintf(&source, "\tif !%s {\n\t\treturn %s\n\t}\n", policy.Enabled, strings.Join(disabled, ", "))
		fmt.Fprintf(&source, "\treturn %s\n}\n\n", strings.Join(unsupported, ", "))
	}
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated hook policy %s: %w", policy.ID, err)
	}
	return formatted, nil
}

func hookSignature(entry intercept) ([]string, []string, error) {
	expression, err := parser.ParseExpr(entry.Signature)
	if err != nil {
		return nil, nil, fmt.Errorf("parse signature: %w", err)
	}
	function, ok := expression.(*ast.FuncType)
	if !ok {
		return nil, nil, errors.New("signature is not a function")
	}
	parameters, err := fieldTypes(function.Params)
	if err != nil {
		return nil, nil, err
	}
	if entry.Receiver != nil {
		receiverType := entry.Receiver.Name
		if entry.Receiver.Pointer {
			receiverType = "*" + receiverType
		}
		parameters = append([]string{receiverType}, parameters...)
	}
	for index := range parameters {
		parameters[index] = "_ " + parameters[index]
	}
	results, err := fieldTypes(function.Results)
	if err != nil {
		return nil, nil, err
	}
	if len(results) == 0 || results[len(results)-1] != "error" {
		return nil, nil, errors.New("generated denial hook must return one final error")
	}
	for _, result := range results[:len(results)-1] {
		if result == "error" {
			return nil, nil, errors.New("generated denial hook has more than one error result")
		}
	}
	return parameters, results, nil
}

func fieldTypes(fields *ast.FieldList) ([]string, error) {
	if fields == nil {
		return nil, nil
	}
	types := make([]string, 0, fields.NumFields())
	for _, field := range fields.List {
		var encoded bytes.Buffer
		if err := format.Node(&encoded, token.NewFileSet(), field.Type); err != nil {
			return nil, fmt.Errorf("format signature type: %w", err)
		}
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for range count {
			types = append(types, encoded.String())
		}
	}
	return types, nil
}

func compilerSpecPath(goVersion string) (string, error) {
	version := goVersionPattern.FindStringSubmatch(goVersion)
	if version == nil {
		return "", fmt.Errorf("cannot derive compiler spec path from Go version %q", goVersion)
	}
	return fmt.Sprintf("overlay/src/cmd/compile/internal/gomadintercept/spec_go%s%s.go", version[1], version[2]), nil
}

func renderCompilerSpec(definition manifest, tests []compilerTest, identity string) ([]byte, error) {
	var source strings.Builder
	source.WriteString("// Copyright 2026 The Go Authors. All rights reserved.\n")
	source.WriteString("// Use of this source code is governed by a BSD-style\n")
	source.WriteString("// license that can be found in the LICENSE file.\n\n")
	source.WriteString("// Code generated by internal/boundarygen. DO NOT EDIT.\n\n")
	source.WriteString("package gomadintercept\n\n")
	fmt.Fprintf(&source, "const boundaryManifestVersion = %q\n", definition.ManifestVersion)
	fmt.Fprintf(&source, "const boundaryManifestSHA256 = %q\n\n", identity)
	source.WriteString("func qualifiedPlatform(goos, goarch string) bool {\n")
	for _, platform := range definition.Platforms {
		goos, goarch, _ := strings.Cut(platform, "/")
		fmt.Fprintf(&source, "\tif goos == %q && goarch == %q { return true }\n", goos, goarch)
	}
	source.WriteString("\treturn false\n}\n\n")
	source.WriteString("var specs = []spec{\n")
	for _, entry := range definition.Intercepts {
		writeSpec(&source, entry.Package, entry.Receiver, entry.Symbol, entry.Hook, entry.DeclarationSHA256, boundaryProbeID(entry.Probe))
	}
	for _, fixture := range tests {
		writeSpec(&source, fixture.Package, fixture.Receiver, fixture.Symbol, fixture.Hook, fixture.DeclarationSHA256, 0)
	}
	source.WriteString("}\n")
	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return nil, fmt.Errorf("format generated compiler interception spec: %w", err)
	}
	return formatted, nil
}

func writeSpec(output *strings.Builder, packagePath string, receiverValue *receiver, symbol, hook, declarationSHA256 string, probeID uint64) {
	fmt.Fprintf(output, "\t{PackagePath: %q, ", packagePath)
	if receiverValue != nil {
		fmt.Fprintf(output, "Receiver: &receiverSpec{Name: %q, Pointer: %t}, ", receiverValue.Name, receiverValue.Pointer)
	}
	fmt.Fprintf(output, "Function: %q, Hook: %q", symbol, hook)
	if declarationSHA256 != "" {
		fmt.Fprintf(output, ", DeclarationSHA256: %q", declarationSHA256)
	}
	if probeID != 0 {
		fmt.Fprintf(output, ", ProbeID: %d", probeID)
	}
	output.WriteString("},\n")
}

func renderExpectedReport(definition manifest) []byte {
	lines := make([]string, 0, len(definition.Intercepts))
	for _, entry := range definition.Intercepts {
		lines = append(lines, fmt.Sprintf("%s.%s -> %s.%s", entry.Package, targetName(entry.Receiver, entry.Symbol), entry.Package, entry.Hook))
	}
	slices.Sort(lines)
	return []byte(strings.Join(lines, "\n") + "\n")
}

func renderInventory(definition manifest, identity string) []byte {
	var output strings.Builder
	fmt.Fprintf(&output, "# Gomad deterministic boundary: %s\n\n", definition.ManifestVersion)
	fmt.Fprintf(&output, "Generated from [`manifest.json`](manifest.json) for Go %s on %s. Manifest identity: `%s`. Do not edit this inventory directly.\n\n", definition.GoVersion, strings.Join(definition.Platforms, ", "), identity)
	output.WriteString("| Target | Signature | Operation | Probe | Disposition | Hook | Hook policy | Adapters | Conformance | Negative | Escape |\n")
	output.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, entry := range definition.Intercepts {
		values := []string{
			entry.Package + "." + targetName(entry.Receiver, entry.Symbol), entry.Signature,
			entry.Operation, entry.Probe, entry.Disposition, entry.Hook, entry.HookPolicy,
			strings.Join(entry.Adapters, "<br>"), strings.Join(entry.ConformanceFixtures, "<br>"),
			strings.Join(entry.NegativeFixtures, "<br>"), strings.Join(entry.EscapeFixtures, "<br>"),
		}
		for index := range values {
			values[index] = strings.ReplaceAll(values[index], "|", "\\|")
		}
		fmt.Fprintf(&output, "| %s |\n", strings.Join(values, " | "))
	}
	if len(definition.ReviewedCandidates) != 0 {
		output.WriteString("\n## Reviewed transitive candidates\n\n")
		output.WriteString("These source-discovered entry points are controlled without an additional compiler prologue.\n\n")
		output.WriteString("| Target | Disposition | Controlling boundaries |\n")
		output.WriteString("| --- | --- | --- |\n")
		for _, candidate := range definition.ReviewedCandidates {
			fmt.Fprintf(&output, "| %s | %s | %s |\n", candidate.Target, candidate.Disposition, strings.Join(candidate.Boundaries, "<br>"))
		}
	}
	return []byte(output.String())
}

func renderHostIdentity(definition manifest, identity string) ([]byte, error) {
	platform := strings.Split(definition.Platforms[0], "/")
	if len(definition.Platforms) != 1 || len(platform) != 2 {
		return nil, errors.New("host identity generation requires exactly one GOOS/GOARCH platform")
	}
	source := fmt.Sprintf(`// Code generated by internal/boundarygen. DO NOT EDIT.

package ioprofile

const (
	generatedBoundaryManifestVersion = %q
	generatedBoundaryManifestSHA256  = %q
	generatedBoundaryGoVersion       = %q
	generatedBoundaryGOOS            = %q
	generatedBoundaryGOARCH          = %q
)
`, definition.ManifestVersion, identity, definition.GoVersion, platform[0], platform[1])
	var probes strings.Builder
	probes.WriteString("\nvar generatedBoundaryProbes = []struct {\n\tID uint64\n\tName string\n}{\n")
	for _, entry := range definition.Intercepts {
		fmt.Fprintf(&probes, "\t{ID: %d, Name: %q},\n", boundaryProbeID(entry.Probe), entry.Probe)
	}
	probes.WriteString("}\n")
	source += probes.String()
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return nil, fmt.Errorf("format generated boundary identity: %w", err)
	}
	return formatted, nil
}

func boundaryProbeID(probe string) uint64 {
	digest := sha256.Sum256([]byte("gomadv3-boundary-probe/v1\x00" + probe))
	return binary.BigEndian.Uint64(digest[:8]) & (1<<63 - 1)
}

func manifestIdentity(definition manifest) (string, error) {
	// Preserve the legacy null compiler_tests projection so moving the test
	// corpus out of the production manifest does not rotate production identity.
	identity := struct {
		SchemaVersion      uint                `json:"schema_version"`
		ManifestVersion    string              `json:"manifest_version"`
		GoVersion          string              `json:"go_version"`
		Platforms          []string            `json:"platforms"`
		HookPolicies       []hookPolicy        `json:"hook_policies,omitempty"`
		Intercepts         []intercept         `json:"intercepts"`
		ReviewedCandidates []reviewedCandidate `json:"reviewed_candidates"`
		CompilerTests      []compilerTest      `json:"compiler_tests"`
	}{
		SchemaVersion: definition.SchemaVersion, ManifestVersion: definition.ManifestVersion, GoVersion: definition.GoVersion,
		Platforms: definition.Platforms, HookPolicies: definition.HookPolicies, Intercepts: definition.Intercepts, ReviewedCandidates: definition.ReviewedCandidates,
	}
	canonical, err := json.Marshal(identity)
	if err != nil {
		return "", fmt.Errorf("encode boundary manifest identity: %w", err)
	}
	digest := sha256.Sum256(canonical)
	return fmt.Sprintf("sha256:%x", digest), nil
}

func targetName(receiverValue *receiver, symbol string) string {
	if receiverValue == nil {
		return symbol
	}
	prefix := ""
	if receiverValue.Pointer {
		prefix = "*"
	}
	return fmt.Sprintf("(%s%s).%s", prefix, receiverValue.Name, symbol)
}
