package target

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target/internal/compatibility"
	"go.temporal.io/server/tools/gomadv3/target/internal/livecap"
)

const CapabilityClosureSchema = "gomadv3.target-capability-closure/v3"
const CapabilityReviewSchema = "gomadv3.target-capability-review/v3"
const maximumCapabilityReviewOutputBytes = 64 << 20
const maximumCapabilityReviewPackages = 100000
const maximumCapabilitySourceBytes = 16 << 20

type AdapterCapacityError struct {
	Resource string
	Limit    uint64
}

func (err *AdapterCapacityError) Error() string {
	return fmt.Sprintf("adapter module exceeds %s limit %d", err.Resource, err.Limit)
}

type InvalidCapabilityReviewError struct {
	Err error
}

func (err *InvalidCapabilityReviewError) Error() string {
	return err.Err.Error()
}

func (err *InvalidCapabilityReviewError) Unwrap() error {
	return err.Err
}

func IsInvalidCapabilityReview(err error) bool {
	var invalid *InvalidCapabilityReviewError
	return errors.As(err, &invalid)
}

func invalidCapabilityReview(err error) error {
	return &InvalidCapabilityReviewError{Err: err}
}

type CapabilityClosure struct {
	Schema        string                   `json:"schema"`
	Compatibility []compatibility.Identity `json:"compatibility"`
	Packages      []CapabilityPackage      `json:"packages"`
}

type CapabilityPackage struct {
	ImportPath        string                    `json:"import_path"`
	ForTest           string                    `json:"for_test,omitempty"`
	Name              string                    `json:"name"`
	Root              bool                      `json:"root,omitempty"`
	Standard          bool                      `json:"standard"`
	Imports           []string                  `json:"imports"`
	Module            *CapabilityModule         `json:"module,omitempty"`
	Sources           []CapabilitySource        `json:"sources"`
	ForeignSources    []CapabilityForeignSource `json:"foreign_sources"`
	GeneratedTestMain bool                      `json:"generated_test_main,omitempty"`
}

type CapabilityModule struct {
	Path        string                        `json:"path"`
	Version     string                        `json:"version"`
	Sum         string                        `json:"sum"`
	Main        bool                          `json:"main"`
	Local       bool                          `json:"local"`
	Replacement *CapabilityModule             `json:"replacement,omitempty"`
	Adapter     *CapabilityAdapterReplacement `json:"adapter,omitempty"`
}

type CapabilityAdapterReplacement struct {
	ProfileName                      string         `json:"profile_name"`
	ProfileImplementationSHA256      string         `json:"profile_implementation_sha256"`
	Adapter                          ModuleIdentity `json:"adapter"`
	OriginalSourceInventorySHA256    string         `json:"original_source_inventory_sha256"`
	ReplacementSourceInventorySHA256 string         `json:"replacement_source_inventory_sha256"`
	PreparedSourceSetSHA256          string         `json:"prepared_source_set_sha256"`
}

type CapabilitySource struct {
	Name               string   `json:"name"`
	SHA256             string   `json:"sha256"`
	LinknameDirectives []string `json:"linkname_directives,omitempty"`
	MalformedLinkname  bool     `json:"malformed_linkname,omitempty"`
}

type CapabilityForeignSource struct {
	Kind   string `json:"kind"`
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
}

type CapabilityPackageReference struct {
	ImportPath string `json:"import_path"`
	ForTest    string `json:"for_test,omitempty"`
	Name       string `json:"name"`
}

type CompatibilityIdentity = compatibility.Identity
type CompatibilityPackEvidence = compatibility.PackEvidence
type CompatibilityDisposition = compatibility.Disposition
type CompatibilityRemediation = compatibility.RemediationCategory

const (
	DispositionAllowedExactPack = compatibility.DispositionAllowedExactPack
	DispositionDenied           = compatibility.DispositionDenied

	RemediationAddExactPack      = compatibility.RemediationAddExactPack
	RemediationAddAdapter        = compatibility.RemediationAddAdapter
	RemediationModelOperation    = compatibility.RemediationModelOperation
	RemediationRemoveDependency  = compatibility.RemediationRemoveDependency
	RemediationRemainUnsupported = compatibility.RemediationRemainUnsupported
)

type CapabilityFindingKind string

const (
	FindingForbiddenImport    CapabilityFindingKind = "forbidden_import"
	FindingForeignSource      CapabilityFindingKind = "foreign_source"
	FindingUnapprovedLinkname CapabilityFindingKind = "unapproved_linkname"
	FindingMalformedLinkname  CapabilityFindingKind = "malformed_linkname"
	FindingNoReviewedGoSource CapabilityFindingKind = "no_reviewed_go_source"
	FindingDeniedBoundary     CapabilityFindingKind = "denied_boundary"
)

type CapabilityFinding struct {
	Kind              CapabilityFindingKind             `json:"kind"`
	Package           CapabilityPackageReference        `json:"package"`
	Module            *CapabilityModule                 `json:"module,omitempty"`
	SourceSetSHA256   string                            `json:"source_set_sha256"`
	SourceName        string                            `json:"source_name,omitempty"`
	SourceSHA256      string                            `json:"source_sha256,omitempty"`
	Directives        []string                          `json:"directives"`
	Capability        string                            `json:"capability"`
	PolicyDisposition compatibility.Disposition         `json:"policy_disposition"`
	Remediation       compatibility.RemediationCategory `json:"remediation"`
	PackID            string                            `json:"pack_id,omitempty"`
}

type CapabilityReview struct {
	Schema             string                       `json:"schema"`
	BuildTags          []string                     `json:"build_tags"`
	Roots              []CapabilityPackageReference `json:"roots"`
	Closure            CapabilityClosure            `json:"closure"`
	Packs              []compatibility.PackEvidence `json:"packs"`
	CapabilityMode     CapabilityMode               `json:"capability_mode"`
	CapabilityManifest *CapabilityManifest          `json:"capability_manifest,omitempty"`
	Findings           []CapabilityFinding          `json:"findings"`
	EliminatedFindings []CapabilityFinding          `json:"eliminated_findings"`
}

type UnsupportedCapabilityError struct {
	ImportPath string
	Capability string
	Finding    CapabilityFinding
}

func (err *UnsupportedCapabilityError) Error() string {
	return fmt.Sprintf("unsupported target capability: package %s %s", err.ImportPath, err.Capability)
}

func validateGoCapabilityClosure(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string) (CapabilityClosure, error) {
	review, err := reviewGoCapabilityReview(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	if err != nil {
		return CapabilityClosure{}, err
	}
	if err := validateCapabilityReview(review.Closure); err != nil {
		return CapabilityClosure{}, err
	}
	return review.Closure, nil
}

func ReviewCapabilityClosure(ctx context.Context, spec Spec) (CapabilityClosure, error) {
	review, err := ReviewCapabilities(ctx, spec)
	if err != nil {
		return CapabilityClosure{}, err
	}
	if len(review.Findings) != 0 {
		return CapabilityClosure{}, unsupportedFinding(review.Findings[0])
	}
	return review.Closure, nil
}

func ReviewCapabilities(ctx context.Context, spec Spec) (CapabilityReview, error) {
	if spec.Kind != KindGoRun && spec.Kind != KindGoTest {
		return CapabilityReview{}, invalidCapabilityReview(errors.New("capability review requires a go-run or go-test target"))
	}
	tags, err := normalizeBuildTags(spec.BuildTags)
	if err != nil {
		return CapabilityReview{}, invalidCapabilityReview(err)
	}
	mode, err := normalizeCapabilityMode(spec.CapabilityMode)
	if err != nil {
		return CapabilityReview{}, invalidCapabilityReview(err)
	}
	spec.CapabilityMode = mode
	if spec.Source == "" || spec.WorkingDir == "" || spec.ToolchainRoot == "" {
		return CapabilityReview{}, invalidCapabilityReview(errors.New("capability review requires source, working directory, and toolchain root"))
	}
	if strings.HasPrefix(spec.Source, "-") || strings.Contains(spec.Source, "...") || strings.IndexFunc(spec.Source, unicode.IsSpace) >= 0 || strings.IndexByte(spec.Source, 0) >= 0 {
		return CapabilityReview{}, invalidCapabilityReview(fmt.Errorf("go target package argument %q must select exactly one package", spec.Source))
	}
	goCommand, err := filepath.Abs(filepath.Join(spec.ToolchainRoot, "bin", "go"))
	if err != nil {
		return CapabilityReview{}, fmt.Errorf("resolve pinned Go command: %w", err)
	}
	commandDirectory, packageArgument, err := resolveBuildContext(spec.WorkingDir, spec.Source)
	if err != nil {
		return CapabilityReview{}, invalidCapabilityReview(err)
	}
	review, err := reviewGoCapabilityReview(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	if err != nil {
		return CapabilityReview{}, err
	}
	if mode == CapabilityModeClosure {
		return review, nil
	}
	if spec.PreparationRoot == "" {
		return CapabilityReview{}, invalidCapabilityReview(errors.New("linked capability review requires a preparation root"))
	}
	identity, err := ReadToolchainIdentity(spec.ToolchainRoot)
	if err != nil {
		return CapabilityReview{}, err
	}
	workspace, err := os.MkdirTemp(spec.PreparationRoot, ".linked-review-")
	if err != nil {
		return CapabilityReview{}, fmt.Errorf("create linked capability review workspace: %w", err)
	}
	prepared, buildErr := buildGoTarget(ctx, spec, tags, identity, filepath.Join(workspace, "target"), goCommand, commandDirectory, packageArgument, review, false)
	cleanupErr := os.RemoveAll(workspace)
	if buildErr != nil || cleanupErr != nil {
		return CapabilityReview{}, errors.Join(buildErr, cleanupErr)
	}
	return prepared.review, nil
}

func reviewGoCapabilityReview(ctx context.Context, goCommand string, spec Spec, tags []string, commandDirectory, packageArgument string) (CapabilityReview, error) {
	arguments := []string{"list", "-deps", "-json", "-mod=readonly"}
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
	stdout, stderr, err := runBoundedCapabilityCommand(command, maximumCapabilityReviewOutputBytes)
	if err != nil {
		if ctx.Err() != nil {
			return CapabilityReview{}, fmt.Errorf("inspect target capability closure: %w", ctx.Err())
		}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			failure := fmt.Errorf("inspect target capability closure: %w: %s", err, stderr)
			if invalidGoListDiagnostic(stderr) {
				return CapabilityReview{}, invalidCapabilityReview(failure)
			}
			return CapabilityReview{}, failure
		}
		return CapabilityReview{}, fmt.Errorf("inspect target capability closure: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(stdout))
	var packages []listedPackage
	for {
		var pkg listedPackage
		if err := decoder.Decode(&pkg); err == io.EOF {
			break
		} else if err != nil {
			return CapabilityReview{}, fmt.Errorf("decode target capability closure: %w", err)
		}
		packages = append(packages, pkg)
		if len(packages) > maximumCapabilityReviewPackages {
			return CapabilityReview{}, fmt.Errorf("target capability closure package count exceeds %d", maximumCapabilityReviewPackages)
		}
	}
	overlay, err := loadBuildOverlay(spec.BuildOverlay, commandDirectory)
	if err != nil {
		return CapabilityReview{}, err
	}
	if err := validateAdapterReplacementInputs(spec); err != nil {
		return CapabilityReview{}, err
	}
	return projectCapabilityReview(packages, overlay, tags, spec.AdapterReplacements)
}

func invalidGoListDiagnostic(stderr []byte) bool {
	message := string(stderr)
	fragments := []string{
		"build constraints exclude all Go files",
		"cannot find main module",
		"directory prefix ",
		"is not in std",
		"import lookup disabled by -mod=readonly",
		"malformed import path",
		"missing go.sum entry",
		"no Go files in",
		"no required module provides package",
		"outside main module or its selected dependencies",
		"package without type was imported",
		"updates to go.mod needed",
	}
	for _, fragment := range fragments {
		if strings.Contains(message, fragment) {
			return true
		}
	}
	return false
}

func runBoundedCapabilityCommand(command *exec.Cmd, limit uint64) (stdoutBytes, stderrBytes []byte, retErr error) {
	stdout, err := newBoundedCommandBuffer(limit)
	if err != nil {
		return nil, nil, err
	}
	stderr, err := newBoundedCommandBuffer(limit)
	if err != nil {
		return nil, nil, err
	}
	command.Stdout = stdout
	command.Stderr = stderr
	runErr := command.Run()
	if stdout.overflow || stderr.overflow {
		return nil, nil, fmt.Errorf("target capability closure output exceeds %d bytes", limit)
	}
	if runErr != nil {
		return nil, stderr.bytes, runErr
	}
	return stdout.bytes, stderr.bytes, nil
}

type boundedCommandBuffer struct {
	bytes    []byte
	limit    uint64
	overflow bool
}

func newBoundedCommandBuffer(limit uint64) (*boundedCommandBuffer, error) {
	if limit == 0 || limit > uint64(^uint(0)>>1) {
		return nil, fmt.Errorf("invalid command output limit %d", limit)
	}
	return &boundedCommandBuffer{bytes: make([]byte, 0, int(limit)), limit: limit}, nil
}

func (buffer *boundedCommandBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - uint64(len(buffer.bytes))
	if uint64(len(data)) > remaining {
		buffer.bytes = append(buffer.bytes, data[:int(remaining)]...)
		buffer.overflow = true
		return len(data), nil
	}
	buffer.bytes = append(buffer.bytes, data...)
	return len(data), nil
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
	DepOnly      bool          `json:"DepOnly"`
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
	HFiles       []string      `json:"HFiles"`
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
	review, err := projectCapabilityReview(packages, nil, nil)
	if err != nil {
		return err
	}
	return validateCapabilityReview(review.Closure)
}

func projectCapabilityClosure(packages []listedPackage, overlay map[string]string) (CapabilityClosure, error) {
	review, err := projectCapabilityReview(packages, overlay, nil)
	if err != nil {
		return CapabilityClosure{}, err
	}
	return review.Closure, nil
}

func projectCapabilityReview(packages []listedPackage, overlay map[string]string, tags []string, replacementSets ...[]AdapterReplacement) (CapabilityReview, error) {
	replacements := []AdapterReplacement{}
	if len(replacementSets) > 1 {
		return CapabilityReview{}, errors.New("target capability review has duplicate adapter replacement inputs")
	}
	if len(replacementSets) == 1 {
		replacements = replacementSets[0]
	}
	replacementsByModule, err := indexAdapterReplacements(replacements)
	if err != nil {
		return CapabilityReview{}, err
	}
	closure := CapabilityClosure{
		Schema:        CapabilityClosureSchema,
		Compatibility: []compatibility.Identity{},
		Packages:      make([]CapabilityPackage, 0, len(packages)),
	}
	for _, pkg := range packages {
		projected, include, err := projectCapabilityPackage(pkg, overlay, replacementsByModule)
		if err != nil {
			return CapabilityReview{}, err
		}
		if !include {
			continue
		}
		closure.Packages = append(closure.Packages, projected)
	}
	if err := validatePreparedAdapterPackages(closure.Packages, replacements); err != nil {
		return CapabilityReview{}, err
	}
	sort.Slice(closure.Packages, func(i, j int) bool {
		if closure.Packages[i].ImportPath != closure.Packages[j].ImportPath {
			return closure.Packages[i].ImportPath < closure.Packages[j].ImportPath
		}
		if closure.Packages[i].ForTest != closure.Packages[j].ForTest {
			return closure.Packages[i].ForTest < closure.Packages[j].ForTest
		}
		return closure.Packages[i].Name < closure.Packages[j].Name
	})
	compatibilityPackages := make([]compatibility.Package, len(closure.Packages))
	for index, pkg := range closure.Packages {
		compatibilityPackages[index] = capabilityCompatibilityPackage(pkg)
	}
	selection, err := compatibility.Select(compatibilityPackages)
	if err != nil {
		return CapabilityReview{}, fmt.Errorf("select target compatibility packs: %w", err)
	}
	closure.Compatibility = selection.Identities()
	selection, err = validateCapabilityReviewStructure(closure)
	if err != nil {
		return CapabilityReview{}, err
	}
	return capabilityReviewFromClosure(closure, tags, selection), nil
}

func validatePreparedAdapterPackages(packages []CapabilityPackage, replacements []AdapterReplacement) error {
	reviewed := make(map[string]bool, len(replacements))
	prepared := make(map[string]bool, len(replacements))
	for _, pkg := range packages {
		if pkg.Module == nil || pkg.Module.Adapter == nil {
			continue
		}
		reviewed[pkg.Module.Path] = true
		for _, replacement := range replacements {
			if replacement.Original.Path == pkg.Module.Path && replacement.PreparedPackage == pkg.ImportPath {
				prepared[pkg.Module.Path] = true
				break
			}
		}
	}
	for _, replacement := range replacements {
		if reviewed[replacement.Original.Path] && !prepared[replacement.Original.Path] {
			return fmt.Errorf("inspect target capability source: adapter prepared package %s is absent", replacement.PreparedPackage)
		}
	}
	return nil
}

func projectCapabilityPackage(pkg listedPackage, overlay map[string]string, replacements map[string]AdapterReplacement) (CapabilityPackage, bool, error) {
	sourceFiles := packageSourceFiles(pkg)
	replacement, hasReplacement, err := matchAdapterReplacement(pkg.Module, replacements)
	if err != nil {
		return CapabilityPackage{}, false, err
	}
	projected := CapabilityPackage{
		ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name, Root: !pkg.DepOnly, Standard: pkg.Standard,
		Imports: sortedSetCopy(packageImports(pkg)), Module: projectCapabilityModule(pkg.Module, replacement, hasReplacement), Sources: []CapabilitySource{},
		ForeignSources: []CapabilityForeignSource{}, GeneratedTestMain: generatedTestMain(pkg),
	}
	if pkg.Standard || projected.GeneratedTestMain {
		return projected, true, nil
	}
	foreignSources, err := projectForeignSources(pkg, overlay)
	if err != nil {
		return CapabilityPackage{}, false, err
	}
	projected.ForeignSources = foreignSources
	if pkg.ForTest == "" && len(sourceFiles) == 0 && len(projected.ForeignSources) == 0 && (len(pkg.TestGoFiles) != 0 || len(pkg.XTestGoFiles) != 0) {
		return CapabilityPackage{}, false, nil
	}
	for _, name := range sourceFiles {
		source, err := projectCapabilitySource(pkg, overlay, name)
		if err != nil {
			return CapabilityPackage{}, false, err
		}
		projected.Sources = append(projected.Sources, source)
	}
	sort.Slice(projected.Sources, func(i, j int) bool { return projected.Sources[i].Name < projected.Sources[j].Name })
	if hasReplacement && pkg.ImportPath == replacement.PreparedPackage {
		sourceSetSHA256 := capabilityCompatibilityPackage(projected).SourceSetSHA256
		if sourceSetSHA256 != replacement.PreparedSourceSetSHA256 {
			return CapabilityPackage{}, false, fmt.Errorf("inspect target capability source %s: adapter prepared source-set identity mismatch: got %s, want %s", pkg.ImportPath, sourceSetSHA256, replacement.PreparedSourceSetSHA256)
		}
	}
	return projected, true, nil
}

func projectCapabilitySource(pkg listedPackage, overlay map[string]string, name string) (CapabilitySource, error) {
	if filepath.Base(name) != name || pkg.Dir == "" {
		return CapabilitySource{}, fmt.Errorf("inspect target capability source %s: invalid source path %q", pkg.ImportPath, name)
	}
	path := filepath.Join(pkg.Dir, name)
	if replacement, found := overlay[filepath.Clean(path)]; found {
		path = replacement
	}
	contents, err := readBoundedRegularFile(path, maximumCapabilitySourceBytes)
	if err != nil {
		return CapabilitySource{}, fmt.Errorf("inspect target capability source %s: unreadable source %s: %w", pkg.ImportPath, name, err)
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
	return CapabilitySource{Name: name, SHA256: fmt.Sprintf("sha256:%x", hash), LinknameDirectives: directives, MalformedLinkname: malformedLinkname}, nil
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

func validateCapabilityReview(closure CapabilityClosure) error {
	selection, err := validateCapabilityReviewStructure(closure)
	if err != nil {
		return err
	}
	review := capabilityReviewFromClosure(closure, nil, selection)
	if len(review.Findings) != 0 {
		return unsupportedFinding(review.Findings[0])
	}
	return nil
}

func validateCapabilityReviewStructure(closure CapabilityClosure) (compatibility.Selection, error) {
	if closure.Schema != CapabilityClosureSchema || closure.Compatibility == nil || len(closure.Packages) == 0 {
		return compatibility.Selection{}, errors.New("unsupported or empty target capability closure")
	}
	if !sortedUniqueCompatibility(closure.Compatibility) {
		return compatibility.Selection{}, errors.New("target capability closure compatibility packs are not canonical")
	}
	compatibilityPackages := make([]compatibility.Package, 0, len(closure.Packages))
	for _, pkg := range closure.Packages {
		compatibilityPackages = append(compatibilityPackages, capabilityCompatibilityPackage(pkg))
	}
	selection, err := compatibility.Select(compatibilityPackages)
	if err != nil {
		return compatibility.Selection{}, fmt.Errorf("select target compatibility packs: %w", err)
	}
	if !slices.Equal(selection.Identities(), closure.Compatibility) {
		return compatibility.Selection{}, errors.New("target capability closure compatibility pack identity does not match its package closure")
	}
	mainPackage := false
	for index, pkg := range closure.Packages {
		if err := validateCapabilityPackageStructure(closure.Packages, index); err != nil {
			return compatibility.Selection{}, err
		}
		if pkg.Name == "main" {
			mainPackage = true
		}
	}
	if !mainPackage {
		return compatibility.Selection{}, errors.New("target capability closure has no main package")
	}
	return selection, nil
}

func validateCapabilityPackageStructure(packages []CapabilityPackage, index int) error {
	pkg := packages[index]
	if pkg.ImportPath == "" || pkg.Name == "" {
		return errors.New("target capability closure has an empty package identity")
	}
	if pkg.Imports == nil || pkg.Sources == nil || pkg.ForeignSources == nil {
		return fmt.Errorf("target capability closure package %s has non-canonical null fields", pkg.ImportPath)
	}
	if index > 0 && compareCapabilityPackage(packages[index-1], pkg) >= 0 {
		return errors.New("target capability closure packages are not sorted and unique")
	}
	if !sortedUnique(pkg.Imports) || !sortedUniqueForeignSources(pkg.ForeignSources) || !sortedUniqueSources(pkg.Sources) {
		return fmt.Errorf("target capability closure package %s is not canonical", pkg.ImportPath)
	}
	if err := validateCapabilityModule(pkg.Module); err != nil {
		return fmt.Errorf("target capability closure package %s: %w", pkg.ImportPath, err)
	}
	for _, source := range pkg.Sources {
		if err := validateCapabilitySource(source); err != nil {
			return fmt.Errorf("target capability closure package %s: %w", pkg.ImportPath, err)
		}
	}
	for _, source := range pkg.ForeignSources {
		if source.Kind == "" || filepath.Base(source.Name) != source.Name || source.Name == "" {
			return fmt.Errorf("target capability closure package %s has invalid foreign source evidence", pkg.ImportPath)
		}
		if _, err := evidence.ParseSHA256(source.SHA256); err != nil {
			return fmt.Errorf("target capability closure package %s has invalid foreign source evidence", pkg.ImportPath)
		}
	}
	if pkg.GeneratedTestMain && (pkg.Name != "main" || !strings.HasSuffix(pkg.ImportPath, ".test") || pkg.Standard || pkg.Module != nil && !pkg.Module.Main || len(pkg.Sources) != 0 || len(pkg.ForeignSources) != 0) {
		return fmt.Errorf("target capability closure package %s has invalid generated test-main evidence", pkg.ImportPath)
	}
	return nil
}

func validateCapabilitySource(source CapabilitySource) error {
	_, digestErr := evidence.ParseSHA256(source.SHA256)
	if filepath.Base(source.Name) != source.Name || source.Name == "" || digestErr != nil {
		return errors.New("has invalid source evidence")
	}
	if source.MalformedLinkname && len(source.LinknameDirectives) != 0 {
		return errors.New("has invalid linkname evidence")
	}
	return nil
}

func capabilityReviewFromClosure(closure CapabilityClosure, tags []string, selection compatibility.Selection) CapabilityReview {
	roots := []CapabilityPackageReference{}
	for _, pkg := range closure.Packages {
		if pkg.Root {
			roots = append(roots, capabilityPackageReference(pkg))
		}
	}
	return CapabilityReview{
		Schema: CapabilityReviewSchema, BuildTags: append([]string{}, tags...), Roots: roots, Closure: closure,
		Packs: selection.Evidence(), CapabilityMode: CapabilityModeClosure,
		Findings: collectCapabilityFindings(closure, selection), EliminatedFindings: []CapabilityFinding{},
	}
}

func projectLinkedCapabilityReview(review CapabilityReview, record livecap.Record) CapabilityReview {
	packages := make([]livecap.ClosurePackage, len(review.Closure.Packages))
	for index, pkg := range review.Closure.Packages {
		packages[index] = livecap.ClosurePackage{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Root: pkg.Root, Standard: pkg.Standard}
	}
	findings := make([]livecap.ClosureFinding, len(review.Findings))
	for index, finding := range review.Findings {
		findings[index] = livecap.ClosureFinding{
			Kind: string(finding.Kind), Package: finding.Package.ImportPath, ForTest: finding.Package.ForTest, Capability: finding.Capability,
		}
	}
	projection := livecap.ProjectFindings(record.Manifest, packages, findings)
	active := make([]CapabilityFinding, 0, len(review.Findings)-len(projection.Eliminated))
	eliminated := make([]CapabilityFinding, 0, len(projection.Eliminated))
	for index, finding := range review.Findings {
		finding.Directives = append([]string{}, finding.Directives...)
		if projection.Active[index] {
			active = append(active, finding)
		} else {
			eliminated = append(eliminated, finding)
		}
	}
	active = append(active, projectDeniedBoundaryFindings(review.Closure.Packages, projection.Denied)...)
	sort.Slice(active, func(i, j int) bool { return compareCapabilityFinding(active[i], active[j]) < 0 })
	review.CapabilityMode = CapabilityModeLinked
	review.CapabilityManifest = capabilityManifest(record)
	review.Findings = active
	review.EliminatedFindings = eliminated
	return review
}

func projectDeniedBoundaryFindings(packages []CapabilityPackage, facts []livecap.Fact) []CapabilityFinding {
	result := []CapabilityFinding{}
	seen := make(map[string]struct{})
	for _, fact := range facts {
		pkg, found := capabilityOwnerPackage(packages, fact.OwnerPackage, fact.ForTest)
		if !found {
			continue
		}
		key := pkg.ImportPath + "\x00" + pkg.ForTest + "\x00" + fact.Capability
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, CapabilityFinding{
			Kind: FindingDeniedBoundary, Package: capabilityPackageReference(pkg), Module: copyCapabilityModule(pkg.Module),
			SourceSetSHA256: capabilityCompatibilityPackage(pkg).SourceSetSHA256, Directives: []string{},
			Capability: fact.Capability, PolicyDisposition: compatibility.DispositionDenied, Remediation: compatibility.RemediationModelOperation,
		})
	}
	return result
}

func capabilityOwnerPackage(packages []CapabilityPackage, owner, forTest string) (CapabilityPackage, bool) {
	for _, pkg := range packages {
		if pkg.ImportPath == owner && pkg.ForTest == forTest {
			return pkg, true
		}
	}
	for _, pkg := range packages {
		if pkg.Root && !pkg.Standard {
			return pkg, true
		}
	}
	return CapabilityPackage{}, false
}

func capabilityManifest(record livecap.Record) *CapabilityManifest {
	return &CapabilityManifest{
		Schema: record.Manifest.Schema, SHA256: record.SHA256, Bytes: uint64(len(record.Payload)), Facts: uint64(len(record.Manifest.Facts)),
		ProducerImplementationSHA256: record.Manifest.ProducerImplementationSHA256,
		CapabilityUniverseSHA256:     record.Manifest.CapabilityUniverseSHA256,
		Payload:                      append([]byte(nil), record.Payload...),
	}
}

func cloneCapabilityManifest(manifest *CapabilityManifest) *CapabilityManifest {
	if manifest == nil {
		return nil
	}
	cloned := *manifest
	cloned.Payload = append([]byte(nil), manifest.Payload...)
	return &cloned
}

func sameCapabilityManifest(left, right *CapabilityManifest) bool {
	if left == nil || right == nil {
		return left == right
	}
	return left.Schema == right.Schema && left.SHA256 == right.SHA256 && left.Bytes == right.Bytes && left.Facts == right.Facts &&
		left.ProducerImplementationSHA256 == right.ProducerImplementationSHA256 && left.CapabilityUniverseSHA256 == right.CapabilityUniverseSHA256 &&
		(len(left.Payload) == 0 || bytes.Equal(left.Payload, right.Payload))
}

func collectCapabilityFindings(closure CapabilityClosure, selection compatibility.Selection) []CapabilityFinding {
	findings := []CapabilityFinding{}
	for _, pkg := range closure.Packages {
		if pkg.Standard {
			continue
		}
		findings = append(findings, collectCapabilityPackageFindings(pkg, selection)...)
	}
	sort.Slice(findings, func(i, j int) bool { return compareCapabilityFinding(findings[i], findings[j]) < 0 })
	return findings
}

func collectCapabilityPackageFindings(pkg CapabilityPackage, selection compatibility.Selection) []CapabilityFinding {
	compatibilityPackage := capabilityCompatibilityPackage(pkg)
	findings := collectImportFindings(pkg, compatibilityPackage, selection)
	for _, source := range pkg.ForeignSources {
		// Headers remain source-set evidence, but cannot execute without a separately reviewed compiled foreign input.
		if source.Kind == "header" {
			continue
		}
		fact := compatibility.Fact{Kind: compatibility.FactCapability, Capability: "foreign:" + source.Kind + ":" + source.Name}
		if decision := selection.Evaluate(compatibilityPackage, fact); !decision.Allowed {
			findings = append(findings, capabilityFinding(pkg, FindingForeignSource, fact, CapabilitySource{Name: source.Name, SHA256: source.SHA256}, decision))
		}
	}
	findings = append(findings, collectLinknameFindings(pkg, compatibilityPackage, selection)...)
	if !pkg.GeneratedTestMain && len(pkg.Sources) == 0 {
		fact := compatibility.Fact{Kind: compatibility.FactNoReviewedGoSource, Capability: "source:go"}
		findings = append(findings, capabilityFinding(pkg, FindingNoReviewedGoSource, fact, CapabilitySource{}, selection.Evaluate(compatibilityPackage, fact)))
	}
	return findings
}

func collectImportFindings(pkg CapabilityPackage, compatibilityPackage compatibility.Package, selection compatibility.Selection) []CapabilityFinding {
	findings := []CapabilityFinding{}
	for _, imported := range pkg.Imports {
		if !forbiddenImport(imported) {
			continue
		}
		fact := compatibility.Fact{Kind: compatibility.FactCapability, Capability: "import:" + imported}
		if decision := selection.Evaluate(compatibilityPackage, fact); !decision.Allowed {
			findings = append(findings, capabilityFinding(pkg, FindingForbiddenImport, fact, CapabilitySource{}, decision))
		}
	}
	return findings
}

func collectLinknameFindings(pkg CapabilityPackage, compatibilityPackage compatibility.Package, selection compatibility.Selection) []CapabilityFinding {
	findings := []CapabilityFinding{}
	for _, source := range pkg.Sources {
		fact, kind, present := linknameFinding(source)
		if !present {
			continue
		}
		if builtInSimulationLinknameAllowed(pkg, source) {
			continue
		}
		if decision := selection.Evaluate(compatibilityPackage, fact); !decision.Allowed {
			findings = append(findings, capabilityFinding(pkg, kind, fact, source, decision))
		}
	}
	return findings
}

var builtInSimulationLinknames = map[string]CapabilitySource{
	"runtime_domain.go": {
		Name: "runtime_domain.go", SHA256: "sha256:8d7f3d9d4fa4f3ad939364e2dc26110fc0cbdff4698ba8f234927e792d5f57af",
		LinknameDirectives: []string{"gomadSimulationEnabled runtime.gomadDeterministicEnabled", "gomadSimulationBegin internal/gomadsim.Begin", "gomadSimulationRegister internal/gomadsim.Register", "gomadSimulationEnter internal/gomadsim.Enter", "gomadSimulationLeave internal/gomadsim.Leave", "gomadSimulationRevoke internal/gomadsim.Revoke", "gomadSimulationFinish internal/gomadsim.Finish"},
	},
	"runtime_network.go": {
		Name: "runtime_network.go", SHA256: "sha256:2f4c6f740dfe9d10fac2e75e07db096d6e6adef99f1370bc385ccfe7efd77223",
		LinknameDirectives: []string{"gomadNetworkBegin internal/gomadio.BeginSimulation", "gomadNetworkPartition internal/gomadio.PartitionSimulation", "gomadNetworkHeal internal/gomadio.HealSimulation", "gomadNetworkDelay internal/gomadio.DelaySimulation", "gomadNetworkGroup internal/gomadio.ChangeSimulationGroup", "gomadNetworkRevoke internal/gomadio.RevokeSimulation", "gomadNetworkFinish internal/gomadio.FinishSimulation"},
	},
	"runtime_process.go": {
		Name: "runtime_process.go", SHA256: "sha256:3f5cc7f97fe13f8699503ca2b9654ec3cfaea4c0865699838e2c4adfee507124",
		LinknameDirectives: []string{"gomadProcessAvailable internal/gomadsim.ProcessAvailable", "gomadProcessRole internal/gomadsim.ProcessRole", "gomadProcessBootstrap internal/gomadsim.ProcessBootstrap", "gomadProcessExchange internal/gomadsim.ProcessExchange", "gomadProcessWaitStop internal/gomadsim.ProcessWaitStop", "gomadProcessServeModel internal/gomadsim.ProcessServeModel"},
	},
	"runtime_process_model.go": {
		Name: "runtime_process_model.go", SHA256: "sha256:d42aab0768800d79393eb23cd8f3e29c47663afcb5694bfc61b2285ddb24e7a4",
		LinknameDirectives: []string{"gomadProcessNetworkOperation internal/gomadio.ProcessSimulationNetworkOperation"},
	},
	"runtime_volume.go": {
		Name: "runtime_volume.go", SHA256: "sha256:08072b86675fead340e8266d7c1c6d7c027159c3c30573e33b318a68b75de2de",
		LinknameDirectives: []string{"gomadVolumeBegin internal/gomadfs.BeginSimulationVolumes", "gomadInitializeVolumeFilesystem os.gomadInitializeSimulationFilesystem", "gomadVolumeRegister internal/gomadfs.RegisterSimulationVolumes", "gomadVolumeRevoke internal/gomadfs.RevokeSimulationVolumes", "gomadVolumeEnumerate internal/gomadfs.EnumerateSimulationVolume", "gomadVolumeFinish internal/gomadfs.FinishSimulationVolumes"},
	},
}

func builtInSimulationLinknameAllowed(pkg CapabilityPackage, source CapabilitySource) bool {
	const importPath = "go.temporal.io/server/tools/gomadv3sim"
	exactPackage := pkg.ImportPath == importPath || pkg.ForTest == importPath && strings.HasPrefix(pkg.ImportPath, importPath+" [")
	if !exactPackage || pkg.Module == nil || pkg.Module.Path != "go.temporal.io/server" || !pkg.Module.Main || pkg.Module.Replacement != nil || source.MalformedLinkname {
		return false
	}
	want, ok := builtInSimulationLinknames[source.Name]
	return ok && source.SHA256 == want.SHA256 && slices.Equal(source.LinknameDirectives, want.LinknameDirectives)
}

func linknameFinding(source CapabilitySource) (compatibility.Fact, CapabilityFindingKind, bool) {
	if source.MalformedLinkname {
		return compatibility.Fact{Kind: compatibility.FactMalformedLinkname, Capability: "linkname:malformed", Source: source.Name, SHA256: source.SHA256, Directives: []string{}}, FindingMalformedLinkname, true
	}
	if len(source.LinknameDirectives) == 0 {
		return compatibility.Fact{}, "", false
	}
	return compatibility.Fact{
		Kind: compatibility.FactLinkname, Capability: "linkname:" + source.Name, Source: source.Name,
		SHA256: source.SHA256, Directives: source.LinknameDirectives,
	}, FindingUnapprovedLinkname, true
}

func capabilityFinding(pkg CapabilityPackage, kind CapabilityFindingKind, fact compatibility.Fact, source CapabilitySource, decision compatibility.Decision) CapabilityFinding {
	return CapabilityFinding{
		Kind: kind, Package: capabilityPackageReference(pkg), Module: copyCapabilityModule(pkg.Module),
		SourceSetSHA256: capabilityCompatibilityPackage(pkg).SourceSetSHA256,
		SourceName:      source.Name, SourceSHA256: source.SHA256, Directives: append([]string{}, fact.Directives...),
		Capability: fact.Capability, PolicyDisposition: decision.Disposition, Remediation: decision.Remediation, PackID: decision.PackID,
	}
}

func capabilityPackageReference(pkg CapabilityPackage) CapabilityPackageReference {
	return CapabilityPackageReference{ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name}
}

func compareCapabilityPackage(left, right CapabilityPackage) int {
	return compareCapabilityPackageReference(capabilityPackageReference(left), capabilityPackageReference(right))
}

func compareCapabilityPackageReference(left, right CapabilityPackageReference) int {
	if comparison := strings.Compare(left.ImportPath, right.ImportPath); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.ForTest, right.ForTest); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.Name, right.Name)
}

func compareCapabilityFinding(left, right CapabilityFinding) int {
	if comparison := compareCapabilityPackageReference(left.Package, right.Package); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(string(left.Kind), string(right.Kind)); comparison != 0 {
		return comparison
	}
	if comparison := strings.Compare(left.Capability, right.Capability); comparison != 0 {
		return comparison
	}
	return strings.Compare(left.SourceName, right.SourceName)
}

func copyCapabilityModule(module *CapabilityModule) *CapabilityModule {
	if module == nil {
		return nil
	}
	result := *module
	result.Replacement = copyCapabilityModule(module.Replacement)
	if module.Adapter != nil {
		adapter := *module.Adapter
		result.Adapter = &adapter
	}
	return &result
}

func unsupportedFinding(finding CapabilityFinding) error {
	description := "requires an unsupported capability"
	switch finding.Kind {
	case FindingForbiddenImport:
		description = "imports " + strings.TrimPrefix(finding.Capability, "import:")
	case FindingForeignSource:
		description = "contains foreign or assembly source " + strings.TrimPrefix(finding.Capability, "foreign:")
	case FindingUnapprovedLinkname, FindingMalformedLinkname:
		description = "uses go:linkname in " + finding.SourceName
	case FindingNoReviewedGoSource:
		description = "has no reviewed Go source"
	case FindingDeniedBoundary:
		description = "reaches denied deterministic boundary " + finding.Capability
	default:
	}
	return &UnsupportedCapabilityError{ImportPath: finding.Package.ImportPath, Capability: description, Finding: finding}
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

func projectCapabilityModule(module *listedModule, adapter AdapterReplacement, hasAdapter bool) *CapabilityModule {
	if module == nil {
		return nil
	}
	projected := &CapabilityModule{Path: module.Path, Version: module.Version, Sum: module.Sum, Main: module.Main}
	if hasAdapter {
		projected.Path = adapter.Original.Path
		projected.Version = adapter.Original.Version
		projected.Sum = adapter.Original.Sum
	}
	if module.Replace != nil {
		projected.Replacement = projectCapabilityModule(module.Replace, AdapterReplacement{}, false)
		projected.Replacement.Local = module.Replace.Dir != ""
		if projected.Replacement.Local {
			projected.Replacement.Path = ""
			projected.Replacement.Version = ""
			projected.Replacement.Sum = ""
		}
	}
	if hasAdapter {
		projected.Adapter = &CapabilityAdapterReplacement{
			ProfileName: adapter.ProfileName, ProfileImplementationSHA256: adapter.ProfileImplementationSHA256,
			Adapter: adapter.Adapter, OriginalSourceInventorySHA256: adapter.OriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: adapter.ReplacementSourceInventorySHA256,
			PreparedSourceSetSHA256:          adapter.PreparedSourceSetSHA256,
		}
	}
	return projected
}

func projectForeignSources(pkg listedPackage, overlay map[string]string) ([]CapabilityForeignSource, error) {
	projected := []CapabilityForeignSource{}
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
			contents, err := readBoundedRegularFile(path, maximumCapabilitySourceBytes)
			if err != nil {
				return nil, fmt.Errorf("inspect target capability source %s: unreadable source %s: %w", pkg.ImportPath, name, err)
			}
			digest := sha256.Sum256(contents)
			projected = append(projected, CapabilityForeignSource{Kind: group.kind, Name: name, SHA256: fmt.Sprintf("sha256:%x", digest)})
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

func validateCapabilityModule(module *CapabilityModule) error {
	if module == nil {
		return nil
	}
	if module.Path == "" && !module.Local {
		return errors.New("module identity is empty")
	}
	if module.Main && module.Local {
		return errors.New("main module cannot be a local replacement")
	}
	if module.Replacement != nil {
		if module.Replacement.Main || module.Replacement.Replacement != nil {
			return errors.New("module replacement is malformed")
		}
		if err := validateCapabilityModule(module.Replacement); err != nil {
			return err
		}
	}
	if module.Adapter != nil {
		if module.Replacement == nil || !module.Replacement.Local {
			return errors.New("adapter evidence requires a local replacement")
		}
		if err := validateCapabilityAdapter(*module.Adapter); err != nil {
			return err
		}
	}
	return nil
}

func validateCapabilityAdapter(adapter CapabilityAdapterReplacement) error {
	if adapter.ProfileName == "" || adapter.Adapter.Path == "" || adapter.Adapter.Version == "" || adapter.Adapter.Sum == "" {
		return errors.New("adapter replacement identity is incomplete")
	}
	for _, digest := range []string{
		adapter.ProfileImplementationSHA256, adapter.OriginalSourceInventorySHA256,
		adapter.ReplacementSourceInventorySHA256, adapter.PreparedSourceSetSHA256,
	} {
		if _, err := evidence.ParseSHA256(digest); err != nil {
			return errors.New("adapter replacement digest is invalid")
		}
	}
	return nil
}

func capabilityCompatibilityPackage(pkg CapabilityPackage) compatibility.Package {
	goSources := make([]compatibility.Source, len(pkg.Sources))
	sources := make([]compatibility.Source, 0, len(pkg.Sources)+len(pkg.ForeignSources))
	for index, source := range pkg.Sources {
		goSources[index] = compatibility.Source{Name: source.Name, SHA256: source.SHA256}
		sources = append(sources, goSources[index])
	}
	foreignSources := make([]compatibility.ForeignSource, len(pkg.ForeignSources))
	for index, source := range pkg.ForeignSources {
		foreignSources[index] = compatibility.ForeignSource{Kind: source.Kind, Name: source.Name, SHA256: source.SHA256}
		sources = append(sources, compatibility.Source{Name: source.Kind + ":" + source.Name, SHA256: source.SHA256})
	}
	return compatibility.Package{
		ImportPath: pkg.ImportPath, Module: capabilityCompatibilityModule(pkg.Module), SourceSetSHA256: compatibility.DigestSources(sources),
		GoSources: goSources, ForeignSources: foreignSources,
	}
}

func capabilityCompatibilityModule(module *CapabilityModule) compatibility.Module {
	if module == nil {
		return compatibility.Module{}
	}
	projected := compatibility.Module{
		Path:             module.Path,
		Version:          module.Version,
		Sum:              module.Sum,
		Replaced:         module.Replacement != nil,
		LocalReplacement: module.Replacement != nil && module.Replacement.Local,
	}
	if module.Adapter != nil {
		projected.Adapter = &compatibility.AdapterEvidence{
			ProfileName: module.Adapter.ProfileName, ProfileImplementationSHA256: module.Adapter.ProfileImplementationSHA256,
			Module: module.Adapter.Adapter.Path, Version: module.Adapter.Adapter.Version, Sum: module.Adapter.Adapter.Sum,
			OriginalSourceInventorySHA256:    module.Adapter.OriginalSourceInventorySHA256,
			ReplacementSourceInventorySHA256: module.Adapter.ReplacementSourceInventorySHA256,
			PreparedSourceSetSHA256:          module.Adapter.PreparedSourceSetSHA256,
		}
	}
	return projected
}

func validateAdapterReplacementInputs(spec Spec) error {
	if len(spec.AdapterReplacements) == 0 {
		return nil
	}
	root, err := filepath.EvalSymlinks(spec.PreparationRoot)
	if err != nil {
		return fmt.Errorf("resolve adapter preparation root: %w", err)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve adapter preparation root: %w", err)
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
		relative, err := filepath.Rel(root, path)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			return errors.New("adapter replacement path is outside the private preparation root")
		}
	}
	return nil
}

func indexAdapterReplacements(replacements []AdapterReplacement) (map[string]AdapterReplacement, error) {
	result := make(map[string]AdapterReplacement, len(replacements))
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

func matchAdapterReplacement(module *listedModule, replacements map[string]AdapterReplacement) (AdapterReplacement, bool, error) {
	if module == nil || module.Replace == nil || module.Replace.Dir == "" {
		return AdapterReplacement{}, false, nil
	}
	replacement, found := replacements[module.Path]
	if !found {
		return AdapterReplacement{}, false, nil
	}
	if replacement.Original.Path != module.Path || replacement.Original.Version != module.Version ||
		module.Sum != "" && replacement.Original.Sum != module.Sum {
		return AdapterReplacement{}, false, fmt.Errorf(
			"adapter replacement module identity mismatch: got %s@%s %q, want %s@%s %q",
			module.Path, module.Version, module.Sum,
			replacement.Original.Path, replacement.Original.Version, replacement.Original.Sum,
		)
	}
	wantPath, err := filepath.EvalSymlinks(replacement.ReplacementPath)
	if err != nil {
		return AdapterReplacement{}, false, fmt.Errorf("resolve adapter replacement evidence: %w", err)
	}
	actualPath, err := filepath.EvalSymlinks(module.Replace.Dir)
	if err != nil {
		return AdapterReplacement{}, false, fmt.Errorf("resolve target adapter replacement: %w", err)
	}
	wantPath, err = filepath.Abs(wantPath)
	if err != nil {
		return AdapterReplacement{}, false, err
	}
	actualPath, err = filepath.Abs(actualPath)
	if err != nil {
		return AdapterReplacement{}, false, err
	}
	if wantPath != actualPath {
		return AdapterReplacement{}, false, errors.New("adapter replacement operational path mismatch")
	}
	digest, err := DigestAdapterSourceInventory(actualPath)
	if err != nil {
		return AdapterReplacement{}, false, fmt.Errorf("inspect adapter replacement source inventory: %w", err)
	}
	if digest != replacement.ReplacementSourceInventorySHA256 {
		return AdapterReplacement{}, false, errors.New("adapter replacement source inventory mismatch")
	}
	portable := CapabilityAdapterReplacement{
		ProfileName: replacement.ProfileName, ProfileImplementationSHA256: replacement.ProfileImplementationSHA256,
		Adapter: replacement.Adapter, OriginalSourceInventorySHA256: replacement.OriginalSourceInventorySHA256,
		ReplacementSourceInventorySHA256: replacement.ReplacementSourceInventorySHA256,
		PreparedSourceSetSHA256:          replacement.PreparedSourceSetSHA256,
	}
	if err := validateCapabilityAdapter(portable); err != nil {
		return AdapterReplacement{}, false, err
	}
	return replacement, true, nil
}

func DigestAdapterSourceInventory(root string) (string, error) {
	const maximumFiles = 5000
	const maximumBytes = uint64(512 << 20)
	return digestAdapterSourceInventory(root, maximumFiles, maximumBytes)
}

func digestAdapterSourceInventory(root string, maximumFiles int, maximumBytes uint64) (string, error) {
	if maximumFiles <= 0 || maximumBytes == 0 {
		return "", errors.New("adapter source inventory limits must be positive")
	}
	hasher := sha256.New()
	_, _ = hasher.Write([]byte("gomadv3.adapter-source-inventory/v1\x00"))
	files := 0
	total := uint64(0)
	err := filepath.WalkDir(root, func(filePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return errors.New("adapter source inventory contains a symbolic link")
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() || info.Size() < 0 {
			return errors.New("adapter source inventory contains a non-regular file")
		}
		files++
		if files > maximumFiles {
			return &AdapterCapacityError{Resource: "files", Limit: uint64(maximumFiles)}
		}
		size := uint64(info.Size())
		if size > maximumBytes-total {
			return &AdapterCapacityError{Resource: "bytes", Limit: maximumBytes}
		}
		contents, err := readBoundedRegularFile(filePath, maximumBytes-total)
		if err != nil {
			return err
		}
		total += uint64(len(contents))
		relative, err := filepath.Rel(root, filePath)
		if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return errors.New("adapter source inventory path is invalid")
		}
		digest := sha256.Sum256(contents)
		_, _ = hasher.Write([]byte(filepath.ToSlash(relative)))
		_, _ = hasher.Write([]byte{0})
		_, _ = hasher.Write([]byte(fmt.Sprintf("sha256:%x", digest)))
		_, _ = hasher.Write([]byte{0})
		return nil
	})
	if err != nil {
		return "", err
	}
	if files == 0 {
		return "", errors.New("adapter source inventory is empty")
	}
	return fmt.Sprintf("sha256:%x", hasher.Sum(nil)), nil
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

func sortedUniqueForeignSources(sources []CapabilityForeignSource) bool {
	for index, source := range sources {
		if index == 0 {
			continue
		}
		previous := sources[index-1]
		if previous.Kind > source.Kind || previous.Kind == source.Kind && previous.Name >= source.Name {
			return false
		}
	}
	return true
}

func sortedUniqueCompatibility(identities []compatibility.Identity) bool {
	for index, identity := range identities {
		_, digestErr := evidence.ParseSHA256(identity.SHA256)
		if identity.ID == "" || digestErr != nil || index > 0 && identities[index-1].ID >= identity.ID {
			return false
		}
	}
	return true
}

func recordCompatibility(identities []compatibility.Identity) []evidence.CompatibilityPack {
	result := make([]evidence.CompatibilityPack, len(identities))
	for index, identity := range identities {
		result[index] = evidence.CompatibilityPack{ID: identity.ID, SHA256: evidence.SHA256(identity.SHA256)}
	}
	return result
}

func VerifyCompatibility(packs []evidence.CompatibilityPack) error {
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
