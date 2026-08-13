package version

import (
	"bytes"
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

const descriptorPath = "version.json"

var (
	goVersionPattern = regexp.MustCompile(`^go[1-9][0-9]*\.[0-9]+\.[0-9]+$`)
	sha256Pattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
	versionPattern   = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`)
)

type Descriptor struct {
	SchemaVersion           uint      `json:"schema_version"`
	GoVersion               string    `json:"go_version"`
	Archive                 Archive   `json:"archive"`
	SupportedPlatforms      []string  `json:"supported_platforms"`
	BoundaryManifestVersion string    `json:"boundary_manifest_version"`
	Patch                   string    `json:"patch"`
	Adapters                []Adapter `json:"adapters"`
	PatchAllowlist          []string  `json:"patch_allowlist"`
	OverlayAllowlist        []string  `json:"overlay_allowlist"`
}

type Archive struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type Adapter struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Sum     string `json:"sum"`
}

type boundaryIdentity struct {
	ManifestVersion string   `json:"manifest_version"`
	GoVersion       string   `json:"go_version"`
	Platforms       []string `json:"platforms"`
}

type artifact struct {
	path    string
	content []byte
}

func Load(root string) (Descriptor, error) {
	path := filepath.Join(root, descriptorPath)
	contents, err := os.ReadFile(path)
	if err != nil {
		return Descriptor{}, fmt.Errorf("read version descriptor: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var descriptor Descriptor
	if err := decoder.Decode(&descriptor); err != nil {
		return Descriptor{}, fmt.Errorf("decode version descriptor: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return Descriptor{}, errors.New("version descriptor has trailing data")
	}
	if err := validate(descriptor); err != nil {
		return Descriptor{}, err
	}
	return descriptor, nil
}

func Generate(root string, check bool) error {
	descriptor, err := Load(root)
	if err != nil {
		return err
	}
	if err := validateBoundaryIdentity(root, descriptor); err != nil {
		return err
	}
	if err := validateSourceAllowlists(root, descriptor); err != nil {
		return err
	}
	artifacts, err := render(descriptor)
	if err != nil {
		return err
	}
	for _, generated := range artifacts {
		path := filepath.Join(root, filepath.FromSlash(generated.path))
		if check {
			current, readErr := os.ReadFile(path)
			if readErr != nil || !bytes.Equal(current, generated.content) {
				return fmt.Errorf("generated version artifact is stale: %s", generated.path)
			}
			continue
		}
		if err := writeAtomic(path, generated.content); err != nil {
			return err
		}
	}
	return nil
}

func validateSourceAllowlists(root string, descriptor Descriptor) error {
	patch, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(descriptor.Patch)))
	if err != nil {
		return fmt.Errorf("read versioned patch: %w", err)
	}
	var patchPaths []string
	for _, line := range strings.Split(string(patch), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 4 && fields[0] == "diff" && fields[1] == "--git" {
			if !strings.HasPrefix(fields[2], "a/") || !strings.HasPrefix(fields[3], "b/") || fields[2][2:] != fields[3][2:] {
				return errors.New("versioned patch has a non-canonical file header")
			}
			patchPaths = append(patchPaths, fields[2][2:])
		}
	}
	slices.Sort(patchPaths)
	patchPaths = slices.Compact(patchPaths)
	if !slices.Equal(patchPaths, descriptor.PatchAllowlist) {
		return fmt.Errorf("patch allowlist does not match patch paths: got %v, want %v", descriptor.PatchAllowlist, patchPaths)
	}

	overlayRoot := filepath.Join(root, "overlay")
	var overlayPaths []string
	err = filepath.WalkDir(overlayRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("versioned overlay has a non-regular entry: %s", path)
		}
		relative, relativeErr := filepath.Rel(overlayRoot, path)
		if relativeErr != nil {
			return relativeErr
		}
		overlayPaths = append(overlayPaths, filepath.ToSlash(relative))
		return nil
	})
	if err != nil {
		return fmt.Errorf("walk versioned overlay: %w", err)
	}
	slices.Sort(overlayPaths)
	if !slices.Equal(overlayPaths, descriptor.OverlayAllowlist) {
		return fmt.Errorf("overlay allowlist does not match overlay paths: got %v, want %v", descriptor.OverlayAllowlist, overlayPaths)
	}
	return nil
}

func validate(descriptor Descriptor) error {
	if descriptor.SchemaVersion != 1 {
		return fmt.Errorf("version descriptor schema version %d is unsupported", descriptor.SchemaVersion)
	}
	if !goVersionPattern.MatchString(descriptor.GoVersion) {
		return fmt.Errorf("version descriptor Go version is invalid: %q", descriptor.GoVersion)
	}
	wantArchive := descriptor.GoVersion + ".src.tar.gz"
	if descriptor.Archive.Name != wantArchive {
		return fmt.Errorf("version descriptor archive name is %q, want %q", descriptor.Archive.Name, wantArchive)
	}
	if descriptor.Archive.URL != "https://go.dev/dl/"+descriptor.Archive.Name {
		return errors.New("version descriptor archive URL does not match its name")
	}
	if !sha256Pattern.MatchString(descriptor.Archive.SHA256) {
		return errors.New("version descriptor archive SHA-256 is invalid")
	}
	if descriptor.BoundaryManifestVersion == "" {
		return errors.New("version descriptor boundary manifest version is empty")
	}
	if err := validateRelativeFile("patch", descriptor.Patch); err != nil {
		return err
	}
	if len(descriptor.SupportedPlatforms) == 0 {
		return errors.New("version descriptor has no supported platforms")
	}
	if err := validateSortedUnique("supported platform", descriptor.SupportedPlatforms, validatePlatform); err != nil {
		return err
	}
	if len(descriptor.Adapters) == 0 {
		return errors.New("version descriptor has no adapters")
	}
	for index, adapter := range descriptor.Adapters {
		if adapter.Module == "" || !versionPattern.MatchString(adapter.Version) || !strings.HasPrefix(adapter.Sum, "h1:") {
			return fmt.Errorf("version descriptor adapter %d is invalid", index+1)
		}
		if index > 0 && adapter.Module <= descriptor.Adapters[index-1].Module {
			return fmt.Errorf("version descriptor adapters are not sorted and unique at %s", adapter.Module)
		}
	}
	if _, found := adapterByModule(descriptor, "modernc.org/libc"); !found {
		return errors.New("version descriptor omits the modernc.org/libc adapter")
	}
	if err := validateSortedUnique("patch allowlist path", descriptor.PatchAllowlist, func(path string) error {
		return validateRelativeFile("patch allowlist", path)
	}); err != nil {
		return err
	}
	if err := validateSortedUnique("overlay allowlist path", descriptor.OverlayAllowlist, func(path string) error {
		return validateRelativeFile("overlay allowlist", path)
	}); err != nil {
		return err
	}
	return nil
}

func validateBoundaryIdentity(root string, descriptor Descriptor) error {
	path := filepath.Join(root, "boundary", "manifest.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read boundary manifest identity: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	var identity boundaryIdentity
	if err := decoder.Decode(&identity); err != nil {
		return fmt.Errorf("decode boundary manifest identity: %w", err)
	}
	if identity.ManifestVersion != descriptor.BoundaryManifestVersion {
		return fmt.Errorf("boundary manifest version is %q, descriptor requires %q", identity.ManifestVersion, descriptor.BoundaryManifestVersion)
	}
	if identity.GoVersion != descriptor.GoVersion {
		return fmt.Errorf("boundary manifest Go version is %q, descriptor requires %q", identity.GoVersion, descriptor.GoVersion)
	}
	if !slices.Equal(identity.Platforms, descriptor.SupportedPlatforms) {
		return fmt.Errorf("boundary manifest platforms are %v, descriptor requires %v", identity.Platforms, descriptor.SupportedPlatforms)
	}
	return nil
}

func validateSortedUnique(name string, values []string, validateValue func(string) error) error {
	if len(values) == 0 {
		return fmt.Errorf("version descriptor has no %ss", name)
	}
	for index, value := range values {
		if err := validateValue(value); err != nil {
			return err
		}
		if index > 0 && value <= values[index-1] {
			if value == values[index-1] {
				return fmt.Errorf("version descriptor %s is duplicated: %s", name, value)
			}
			return fmt.Errorf("version descriptor %ss are not sorted at %s", name, value)
		}
	}
	return nil
}

func validatePlatform(value string) error {
	parts := strings.Split(value, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("version descriptor supported platform is invalid: %q", value)
	}
	return nil
}

func validateRelativeFile(name, value string) error {
	if value == "" || filepath.IsAbs(value) || filepath.Clean(value) != value || value == "." || strings.HasPrefix(value, ".."+string(filepath.Separator)) || strings.Contains(value, "\\") {
		return fmt.Errorf("version descriptor %s path is invalid: %q", name, value)
	}
	return nil
}

func render(descriptor Descriptor) ([]artifact, error) {
	shell := renderShell(descriptor)
	makefile := renderMake(descriptor)
	goSource, err := renderGo(descriptor)
	if err != nil {
		return nil, err
	}
	return []artifact{
		{path: "toolchain-version.sh", content: shell},
		{path: "version_generated.mk", content: makefile},
		{path: "internal/version/generated.go", content: goSource},
		{path: upgradeGuideName(descriptor), content: renderUpgradeGuide(descriptor)},
	}, nil
}

func renderShell(descriptor Descriptor) []byte {
	var output strings.Builder
	output.WriteString("#!/usr/bin/env bash\n\n# Code generated by internal/versiongen. DO NOT EDIT.\n\n")
	fmt.Fprintf(&output, "go_version=%s\narchive_name=%s\narchive_url=%s\narchive_sha256=%s\n", descriptor.GoVersion, descriptor.Archive.Name, descriptor.Archive.URL, descriptor.Archive.SHA256)
	fmt.Fprintf(&output, "boundary_manifest_version=%s\npatch_name=%s\n", descriptor.BoundaryManifestVersion, descriptor.Patch)
	fmt.Fprintf(&output, "expected_intercepts_name=%s\nboundary_report_name=%s\n", expectedInterceptsName(descriptor), boundaryReportName(descriptor))
	output.WriteString("qualified_platforms=(\n")
	for _, platform := range descriptor.SupportedPlatforms {
		fmt.Fprintf(&output, "\t'%s'\n", platform)
	}
	output.WriteString(")\n")
	for _, adapter := range descriptor.Adapters {
		identifier := shellIdentifier(adapter.Module)
		fmt.Fprintf(&output, "adapter_%s_version=%s\nadapter_%s_sum='%s'\n", identifier, adapter.Version, identifier, adapter.Sum)
	}
	renderShellArray(&output, "patch_allowed_paths", descriptor.PatchAllowlist)
	renderShellArray(&output, "overlay_allowed_paths", descriptor.OverlayAllowlist)
	return []byte(output.String())
}

func renderShellArray(output *strings.Builder, name string, values []string) {
	fmt.Fprintf(output, "%s=(\n", name)
	for _, value := range values {
		fmt.Fprintf(output, "\t'%s'\n", value)
	}
	output.WriteString(")\n")
}

func renderMake(descriptor Descriptor) []byte {
	return []byte(fmt.Sprintf("# Code generated by internal/versiongen. DO NOT EDIT.\n\nGOMADV3_GO_VERSION := %s\nGOMADV3_PATCH_FILE := %s\nGOMADV3_EXPECTED_INTERCEPTS := %s\nGOMADV3_BOUNDARY_REPORT := %s\nGOMADV3_COMPILER_SPEC := %s\nGOMADV3_UPGRADE_GUIDE := %s\n", descriptor.GoVersion, descriptor.Patch, expectedInterceptsName(descriptor), boundaryReportName(descriptor), compilerSpecName(descriptor), upgradeGuideName(descriptor)))
}

func renderUpgradeGuide(descriptor Descriptor) []byte {
	var output strings.Builder
	fmt.Fprintf(&output, "# Gomad v3 upgrade qualification: %s\n\n", descriptor.BoundaryManifestVersion)
	output.WriteString("Generated from [`../version.json`](../version.json). Do not edit this guide directly.\n\n")
	output.WriteString("## Pinned inputs\n\n")
	fmt.Fprintf(&output, "- Go release: `%s`\n", descriptor.GoVersion)
	fmt.Fprintf(&output, "- source archive SHA-256: `%s`\n", descriptor.Archive.SHA256)
	fmt.Fprintf(&output, "- supported platforms: `%s`\n", strings.Join(descriptor.SupportedPlatforms, "`, `"))
	fmt.Fprintf(&output, "- boundary manifest: `%s`\n", descriptor.BoundaryManifestVersion)
	fmt.Fprintf(&output, "- patch: [`../%s`](../%s)\n", descriptor.Patch, descriptor.Patch)
	for _, adapter := range descriptor.Adapters {
		fmt.Fprintf(&output, "- adapter: `%s@%s` (`%s`)\n", adapter.Module, adapter.Version, adapter.Sum)
	}
	output.WriteString("\n## Qualification command\n\n")
	output.WriteString("Run from `tools/gomadv3` after updating `version.json`, the boundary manifest, patch, and overlays:\n\n")
	output.WriteString("```sh\nmake generate\nmake upgrade-dossier GOMADV3_BASELINE_REF=<previous-commit>\n```\n\n")
	output.WriteString("The command publishes `.toolchain/upgrade-dossier.json`, even when a behavioral gate fails. The dossier contains the complete upstream patch diff, semantic boundary-manifest diff, expected and applied interception evidence, archive-based overlay collision results, disabled-mode upstream results, mandatory-probe gates, host-clock escape audit, optional retained-corpus report, and platform qualification. CI uploads this file on every run.\n")
	return []byte(output.String())
}

func renderGo(descriptor Descriptor) ([]byte, error) {
	libc, _ := adapterByModule(descriptor, "modernc.org/libc")
	source := fmt.Sprintf(`// Code generated by internal/versiongen. DO NOT EDIT.

package version

const (
	GoVersion = %q
	BoundaryManifestVersion = %q
	PatchFile = %q
	ModerncLibcVersion = %q
	ModerncLibcSum = %q
)

var SupportedPlatforms = [...]string{%s}
`, descriptor.GoVersion, descriptor.BoundaryManifestVersion, descriptor.Patch, libc.Version, libc.Sum, quotedGoStrings(descriptor.SupportedPlatforms))
	formatted, err := format.Source([]byte(source))
	if err != nil {
		return nil, fmt.Errorf("format generated version Go source: %w", err)
	}
	return formatted, nil
}

func adapterByModule(descriptor Descriptor, module string) (Adapter, bool) {
	for _, adapter := range descriptor.Adapters {
		if adapter.Module == module {
			return adapter, true
		}
	}
	return Adapter{}, false
}

func expectedInterceptsName(descriptor Descriptor) string {
	return "expected-intercepts-" + descriptor.GoVersion + ".txt"
}

func boundaryReportName(descriptor Descriptor) string {
	return "boundary/" + descriptor.BoundaryManifestVersion[:strings.LastIndex(descriptor.BoundaryManifestVersion, "-v")] + ".md"
}

func compilerSpecName(descriptor Descriptor) string {
	parts := strings.Split(strings.TrimPrefix(descriptor.GoVersion, "go"), ".")
	return "overlay/src/cmd/compile/internal/gomadintercept/spec_go" + parts[0] + parts[1] + ".go"
}

func upgradeGuideName(descriptor Descriptor) string {
	return "boundary/upgrade-" + descriptor.BoundaryManifestVersion[:strings.LastIndex(descriptor.BoundaryManifestVersion, "-v")] + ".md"
}

func shellIdentifier(value string) string {
	return strings.NewReplacer(".", "_", "/", "_", "-", "_").Replace(value)
}

func quotedGoStrings(values []string) string {
	quoted := make([]string, len(values))
	for index, value := range values {
		quoted[index] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}

func writeAtomic(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create generated version directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".gomadv3-version-*")
	if err != nil {
		return fmt.Errorf("create generated version artifact: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("chmod generated version artifact: %w", err)
	}
	if _, err := temporary.Write(contents); err != nil {
		temporary.Close()
		return fmt.Errorf("write generated version artifact: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close generated version artifact: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("publish generated version artifact: %w", err)
	}
	return nil
}
