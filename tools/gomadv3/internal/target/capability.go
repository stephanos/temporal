package target

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode"

	"go.temporal.io/server/tools/gomadv3/internal/compatibility"
	"go.temporal.io/server/tools/gomadv3/internal/record"
)

const capabilityClosureSchema = "gomadv3.target-capability-closure/v2"
const CapabilityReviewSchema = "gomadv3.target-capability-review/v1"
const maximumCapabilityReviewOutputBytes = 64 << 20
const maximumCapabilityReviewPackages = 100000
const maximumCapabilitySourceBytes = 16 << 20

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
	ImportPath        string             `json:"import_path"`
	ForTest           string             `json:"for_test,omitempty"`
	Name              string             `json:"name"`
	Root              bool               `json:"root,omitempty"`
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
	Name               string   `json:"name"`
	SHA256             string   `json:"sha256"`
	LinknameDirectives []string `json:"linkname_directives,omitempty"`
	MalformedLinkname  bool     `json:"malformed_linkname,omitempty"`
}

type CapabilityPackageReference struct {
	ImportPath string `json:"import_path"`
	ForTest    string `json:"for_test,omitempty"`
	Name       string `json:"name"`
}

type CapabilityFindingKind string

const (
	FindingForbiddenImport    CapabilityFindingKind = "forbidden_import"
	FindingForeignSource      CapabilityFindingKind = "foreign_source"
	FindingUnapprovedLinkname CapabilityFindingKind = "unapproved_linkname"
	FindingMalformedLinkname  CapabilityFindingKind = "malformed_linkname"
	FindingNoReviewedGoSource CapabilityFindingKind = "no_reviewed_go_source"
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
	Schema    string                       `json:"schema"`
	BuildTags []string                     `json:"build_tags"`
	Roots     []CapabilityPackageReference `json:"roots"`
	Closure   CapabilityClosure            `json:"closure"`
	Packs     []compatibility.PackEvidence `json:"packs"`
	Findings  []CapabilityFinding          `json:"findings"`
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
	return review, nil
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
	return projectCapabilityReview(packages, overlay, tags)
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

func projectCapabilityReview(packages []listedPackage, overlay map[string]string, tags []string) (CapabilityReview, error) {
	closure := CapabilityClosure{
		Schema:        capabilityClosureSchema,
		Compatibility: []compatibility.Identity{},
		Packages:      make([]CapabilityPackage, 0, len(packages)),
	}
	for _, pkg := range packages {
		projected, include, err := projectCapabilityPackage(pkg, overlay)
		if err != nil {
			return CapabilityReview{}, err
		}
		if !include {
			continue
		}
		closure.Packages = append(closure.Packages, projected)
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

func projectCapabilityPackage(pkg listedPackage, overlay map[string]string) (CapabilityPackage, bool, error) {
	sourceFiles := packageSourceFiles(pkg)
	projected := CapabilityPackage{
		ImportPath: pkg.ImportPath, ForTest: pkg.ForTest, Name: pkg.Name, Root: !pkg.DepOnly, Standard: pkg.Standard,
		Imports: sortedSetCopy(packageImports(pkg)), Module: projectCapabilityModule(pkg.Module), Sources: []CapabilitySource{},
		ForeignSources: projectForeignSources(pkg), GeneratedTestMain: generatedTestMain(pkg),
	}
	if pkg.Standard || projected.GeneratedTestMain {
		return projected, true, nil
	}
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
	if closure.Schema != capabilityClosureSchema || closure.Compatibility == nil || len(closure.Packages) == 0 {
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
	if !sortedUnique(pkg.Imports) || !sortedUnique(pkg.ForeignSources) || !sortedUniqueSources(pkg.Sources) {
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
	if pkg.GeneratedTestMain && (pkg.Name != "main" || !strings.HasSuffix(pkg.ImportPath, ".test") || pkg.Standard || pkg.Module != nil && !pkg.Module.Main || len(pkg.Sources) != 0 || len(pkg.ForeignSources) != 0) {
		return fmt.Errorf("target capability closure package %s has invalid generated test-main evidence", pkg.ImportPath)
	}
	return nil
}

func validateCapabilitySource(source CapabilitySource) error {
	_, digestErr := record.ParseSHA256(source.SHA256)
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
		Packs: selection.Evidence(), Findings: collectCapabilityFindings(closure, selection),
	}
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
		fact := compatibility.Fact{Kind: compatibility.FactCapability, Capability: "foreign:" + source}
		if decision := selection.Evaluate(compatibilityPackage, fact); !decision.Allowed {
			findings = append(findings, capabilityFinding(pkg, FindingForeignSource, fact, CapabilitySource{}, decision))
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
		if decision := selection.Evaluate(compatibilityPackage, fact); !decision.Allowed {
			findings = append(findings, capabilityFinding(pkg, kind, fact, source, decision))
		}
	}
	return findings
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
	return nil
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
