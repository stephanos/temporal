package boundary

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const manifestPath = "boundary/manifest.json"

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
	Intercepts         []intercept         `json:"intercepts"`
	ReviewedCandidates []reviewedCandidate `json:"reviewed_candidates"`
	CompilerTests      []compilerTest      `json:"compiler_tests"`
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
		if err := writeAtomic(path, generated.content); err != nil {
			return err
		}
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
	seenProbes := make(map[string]struct{}, len(definition.Intercepts))
	seenProbeIDs := make(map[uint64]string, len(definition.Intercepts))
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
	for index, fixture := range definition.CompilerTests {
		if fixture.Case == "" || fixture.Package == "" || fixture.Symbol == "" || fixture.Hook == "" {
			return fmt.Errorf("boundary manifest compiler test %d is incomplete", index+1)
		}
		if err := validateReceiver(fixture.Receiver); err != nil {
			return fmt.Errorf("boundary manifest compiler test %d: %w", index+1, err)
		}
		if fixture.DeclarationSHA256 != "" && !sha256Pattern.MatchString(fixture.DeclarationSHA256) {
			return fmt.Errorf("boundary manifest compiler test %d has an invalid declaration fingerprint", index+1)
		}
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
	version := goVersionPattern.FindStringSubmatch(definition.GoVersion)
	specPath := fmt.Sprintf("overlay/src/cmd/compile/internal/gomadintercept/spec_go%s%s.go", version[1], version[2])
	identity, err := manifestIdentity(definition)
	if err != nil {
		return nil, err
	}
	spec, err := renderCompilerSpec(definition, identity)
	if err != nil {
		return nil, err
	}
	hostIdentity, err := renderHostIdentity(definition, identity)
	if err != nil {
		return nil, err
	}
	platformName := strings.ReplaceAll(strings.Join(definition.Platforms, "+"), "/", "-")
	return []artifact{
		{path: specPath, content: spec},
		{path: "expected-intercepts-" + definition.GoVersion + ".txt", content: renderExpectedReport(definition)},
		{path: filepath.ToSlash(filepath.Join("boundary", definition.GoVersion+"-"+platformName+".md")), content: renderInventory(definition, identity)},
		{path: "internal/ioprofile/boundary_generated.go", content: hostIdentity},
	}, nil
}

func renderCompilerSpec(definition manifest, identity string) ([]byte, error) {
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
	for _, fixture := range definition.CompilerTests {
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
	output.WriteString("| Target | Signature | Operation | Probe | Disposition | Hook | Adapters | Conformance | Negative | Escape |\n")
	output.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, entry := range definition.Intercepts {
		values := []string{
			entry.Package + "." + targetName(entry.Receiver, entry.Symbol), entry.Signature,
			entry.Operation, entry.Probe, entry.Disposition, entry.Hook,
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
	identity := definition
	identity.CompilerTests = nil
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

func writeAtomic(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create boundary artifact directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".boundary-*")
	if err != nil {
		return fmt.Errorf("create boundary artifact: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() { _ = os.Remove(temporaryPath) }()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write boundary artifact: %w", err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set boundary artifact mode: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close boundary artifact: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish boundary artifact: %w", err)
	}
	return nil
}
