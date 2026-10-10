package target

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
	"go.temporal.io/server/tools/gomad3/target/internal/livecap"
	targetprovenance "go.temporal.io/server/tools/gomad3/target/internal/provenance"
	"go.temporal.io/server/tools/gomad3/toolchain/installation"
)

type Kind string

const (
	KindExec   Kind = "exec"
	KindGoRun  Kind = "go-run"
	KindGoTest Kind = "go-test"
)

type CapabilityMode string

const (
	CapabilityModeClosure CapabilityMode = "closure"
	CapabilityModeLinked  CapabilityMode = "linked"
	CapabilityModeGuarded CapabilityMode = "guarded"
)

const provenanceSchema = "gomad3.exec-provenance/v3"

const maximumProvenanceBytes = 16 << 20
const maximumGoEnvironmentBytes = 64 << 10
const maximumGoModuleDownloadBytes = 4 << 20
const maximumGoBuildDiagnosticBytes = 4 << 20

type Spec struct {
	Backend             string `json:",omitempty"`
	Kind                Kind
	Source              string
	Provenance          string
	Args                []string
	BuildTags           []string
	WorkingDir          string
	PreparationRoot     string
	ToolchainRoot       string
	BuildOverlay        string
	BuildModFile        string
	AdapterReplacements []AdapterReplacement
	CapabilityMode      CapabilityMode
}

type ModuleIdentity struct {
	Path    string `json:"path"`
	Version string `json:"version"`
	Sum     string `json:"sum"`
}

type AdapterReplacement struct {
	Original                         ModuleIdentity
	ReplacementPath                  string
	PreparedPackage                  string
	ProfileName                      string
	ProfileImplementationSHA256      string
	Adapter                          ModuleIdentity
	OriginalSourceInventorySHA256    string
	ReplacementSourceInventorySHA256 string
	PreparedSourceSetSHA256          string
}

type ToolchainIdentity struct {
	GoVersion    string
	BuildKey     string
	TargetGOOS   string
	TargetGOARCH string
}

type Prepared struct {
	Backend            *record.BackendMetadata `json:",omitempty"`
	BackendPayloads    []BackendPayload        `json:",omitempty"`
	Path               string
	Kind               Kind
	Source             string
	SHA256             string
	Size               uint64
	Argv               []string
	BuildTags          []string
	Adapters           []record.TargetAdapter
	Compatibility      []record.CompatibilityPack
	BuildInfo          record.BuildInfo
	GoVersion          string
	BuildKey           string
	TargetGOOS         string
	TargetGOARCH       string
	CapabilityMode     CapabilityMode
	CapabilityManifest *CapabilityManifest
}

type CapabilityManifest struct {
	Schema                       string        `json:"schema"`
	SHA256                       record.SHA256 `json:"sha256"`
	Bytes                        uint64        `json:"bytes"`
	Facts                        uint64        `json:"facts"`
	ProducerImplementationSHA256 string        `json:"producer_implementation_sha256"`
	GuardImplementationSHA256    string        `json:"guard_implementation_sha256"`
	CapabilityUniverseSHA256     string        `json:"capability_universe_sha256"`
	Payload                      []byte        `json:"-"`
}

func (prepared Prepared) RecordTarget() record.Target {
	buildInfo := prepared.BuildInfo
	buildInfo.Settings = append([]record.BuildSetting(nil), prepared.BuildInfo.Settings...)
	recorded := record.Target{
		Backend: record.CloneBackendMetadata(prepared.Backend),
		Kind:    string(prepared.Kind), Source: prepared.Source, SHA256: record.SHA256(prepared.SHA256), Size: record.Uint64String(prepared.Size),
		Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...),
		Adapters: append([]record.TargetAdapter{}, prepared.Adapters...), Compatibility: append([]record.CompatibilityPack{}, prepared.Compatibility...), BuildInfo: buildInfo,
		CapabilityMode: string(prepared.CapabilityMode),
	}
	if recorded.CapabilityMode == "" {
		recorded.CapabilityMode = string(CapabilityModeClosure)
	}
	if manifest := prepared.CapabilityManifest; manifest != nil {
		recorded.CapabilityManifest = manifest.Record()
	}
	return recorded
}

func (manifest CapabilityManifest) Record() *record.TargetCapabilityManifest {
	return &record.TargetCapabilityManifest{
		Schema: manifest.Schema, File: "target-capabilities.json", SHA256: manifest.SHA256,
		Bytes: record.Uint64String(manifest.Bytes), Facts: record.Uint64String(manifest.Facts),
		ProducerImplementationSHA256: record.SHA256(manifest.ProducerImplementationSHA256),
		GuardImplementationSHA256:    record.SHA256(manifest.GuardImplementationSHA256),
		CapabilityUniverseSHA256:     record.SHA256(manifest.CapabilityUniverseSHA256),
	}
}

func CapabilityManifestFromRecord(manifest *record.TargetCapabilityManifest) *CapabilityManifest {
	if manifest == nil {
		return nil
	}
	return &CapabilityManifest{
		Schema: manifest.Schema, SHA256: manifest.SHA256, Bytes: uint64(manifest.Bytes), Facts: uint64(manifest.Facts),
		ProducerImplementationSHA256: string(manifest.ProducerImplementationSHA256),
		GuardImplementationSHA256:    string(manifest.GuardImplementationSHA256),
		CapabilityUniverseSHA256:     string(manifest.CapabilityUniverseSHA256),
	}
}

func (prepared Prepared) RecordToolchain() record.Toolchain {
	return record.Toolchain{
		GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey,
		TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH,
	}
}

type preparation struct {
	buildInfo     record.BuildInfo
	compatibility []record.CompatibilityPack
	review        CapabilityReview
	manifest      *CapabilityManifest
}

type unsupportedPolicy uint8

const (
	allowUnsupported unsupportedPolicy = iota
	rejectUnsupported
)

type Provenance struct {
	SchemaVersion      int
	GoVersion          string
	BuildKey           string
	TargetGOOS         string
	TargetGOARCH       string
	BinarySHA256       string
	BinarySize         uint64
	BuildInfo          record.BuildInfo
	CapabilityClosure  CapabilityClosure
	CapabilityMode     CapabilityMode
	CapabilityManifest *CapabilityManifest
}

type provenanceWire struct {
	Schema             string              `json:"schema"`
	SchemaVersion      int                 `json:"schema_version"`
	GoVersion          string              `json:"go_version"`
	BuildKey           string              `json:"build_key"`
	TargetGOOS         string              `json:"target_goos"`
	TargetGOARCH       string              `json:"target_goarch"`
	BinarySHA256       string              `json:"binary_sha256"`
	BinarySize         record.Uint64String `json:"binary_size"`
	BuildInfo          record.BuildInfo    `json:"build_info"`
	CapabilityClosure  CapabilityClosure   `json:"capability_closure"`
	CapabilityMode     CapabilityMode      `json:"capability_mode,omitempty"`
	CapabilityManifest *CapabilityManifest `json:"capability_manifest,omitempty"`
}

func Prepare(ctx context.Context, spec Spec) (Prepared, error) {
	return prepareWith(ctx, spec, gocommand.Default())
}

func prepareWith(ctx context.Context, spec Spec, runner gocommand.Runner) (prepared Prepared, retErr error) {
	tags, err := targetbuild.NormalizeTags(spec.BuildTags)
	if err != nil {
		return Prepared{}, err
	}
	mode, err := normalizeCapabilityMode(spec.CapabilityMode)
	if err != nil {
		return Prepared{}, err
	}
	spec.CapabilityMode = mode
	identity, err := readPinnedToolchainWith(context.Background(), spec.ToolchainRoot, runner)
	if err != nil {
		return Prepared{}, err
	}
	if spec.PreparationRoot == "" {
		return Prepared{}, fmt.Errorf("preparation root is required")
	}
	if err := os.MkdirAll(spec.PreparationRoot, 0o700); err != nil {
		return Prepared{}, fmt.Errorf("create preparation root: %w", err)
	}
	if err := os.Chmod(spec.PreparationRoot, 0o700); err != nil {
		return Prepared{}, fmt.Errorf("make preparation root private: %w", err)
	}
	preparationDir, err := os.MkdirTemp(spec.PreparationRoot, ".prepare-")
	if err != nil {
		return Prepared{}, fmt.Errorf("create private preparation directory: %w", err)
	}
	keep := false
	defer func() {
		if !keep {
			if cleanupErr := os.RemoveAll(preparationDir); cleanupErr != nil {
				retErr = errors.Join(retErr, fmt.Errorf("remove failed preparation: %w", cleanupErr))
			}
		}
	}()
	if err := os.Chmod(preparationDir, 0o700); err != nil {
		return Prepared{}, fmt.Errorf("make preparation directory private: %w", err)
	}

	targetPath := filepath.Join(preparationDir, "target")
	preparedTarget := preparation{}
	switch spec.Kind {
	case KindExec:
		preparedTarget, err = prepareExec(ctx, spec, identity, targetPath)
	case KindGoRun, KindGoTest:
		preparedTarget, err = prepareGo(ctx, spec, tags, identity, targetPath, runner)
	default:
		err = fmt.Errorf("unsupported target kind %q", spec.Kind)
	}
	if err != nil {
		return Prepared{}, err
	}
	if err := os.Chmod(targetPath, 0o500); err != nil {
		return Prepared{}, fmt.Errorf("make prepared target read-only: %w", err)
	}
	hash, size, err := hashRegularFile(targetPath)
	if err != nil {
		return Prepared{}, fmt.Errorf("hash prepared target: %w", err)
	}
	if spec.Kind != KindExec {
		info, infoErr := buildinfo.ReadFile(targetPath)
		if infoErr != nil {
			return Prepared{}, fmt.Errorf("read prepared target build info: %w", infoErr)
		}
		preparedTarget.buildInfo = ProjectBuildInfo(info)
	}
	prepared = Prepared{
		Path:               targetPath,
		Kind:               spec.Kind,
		Source:             spec.Source,
		SHA256:             hash,
		Size:               size,
		Argv:               append([]string{"gomad3-target"}, spec.Args...),
		BuildTags:          tags,
		Adapters:           []record.TargetAdapter{},
		Compatibility:      preparedTarget.compatibility,
		BuildInfo:          preparedTarget.buildInfo,
		GoVersion:          identity.GoVersion,
		BuildKey:           identity.BuildKey,
		TargetGOOS:         identity.TargetGOOS,
		TargetGOARCH:       identity.TargetGOARCH,
		CapabilityMode:     mode,
		CapabilityManifest: cloneCapabilityManifest(preparedTarget.manifest),
	}
	keep = true
	return prepared, nil
}

func (prepared Prepared) Verify() error {
	mode := prepared.CapabilityMode
	if mode == "" {
		mode = CapabilityModeClosure
	}
	if err := VerifyCompatibility(prepared.Compatibility); err != nil {
		return fmt.Errorf("verify prepared target compatibility: %w", err)
	}
	hash, size, err := hashRegularFile(prepared.Path)
	if err != nil {
		return fmt.Errorf("verify prepared target: %w", err)
	}
	if hash != prepared.SHA256 || size != prepared.Size {
		return fmt.Errorf("prepared target changed after preparation")
	}
	if mode != CapabilityModeClosure {
		if prepared.CapabilityManifest == nil {
			return errors.New("verify prepared target capability manifest: missing linked manifest")
		}
		actual, err := ReadCapabilityManifest(prepared.Path, ToolchainIdentity{
			GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey, TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH,
		})
		if err != nil {
			return fmt.Errorf("verify prepared target capability manifest: %w", err)
		}
		if !sameCapabilityManifest(prepared.CapabilityManifest, actual) {
			return errors.New("verify prepared target capability manifest: embedded record changed after preparation")
		}
	} else if mode != CapabilityModeClosure || prepared.CapabilityManifest != nil {
		return errors.New("verify prepared target capability mode is invalid")
	}
	return nil
}

func ReadCapabilityManifest(path string, identity ToolchainIdentity) (*CapabilityManifest, error) {
	record, err := readLinkedCapabilityRecord(path, identity)
	if err != nil {
		return nil, err
	}
	return capabilityManifest(record), nil
}

func ReadCapabilityManifestFile(file *os.File, identity ToolchainIdentity) (*CapabilityManifest, error) {
	record, err := livecap.ReadFile(file, livecap.Expectation{
		GoVersion: identity.GoVersion, ToolchainBuildKey: identity.BuildKey, GOOS: identity.TargetGOOS, GOARCH: identity.TargetGOARCH,
	})
	if err != nil {
		return nil, linkedCapabilityError(err)
	}
	return capabilityManifest(record), nil
}

func ReadToolchainIdentity(root string) (ToolchainIdentity, error) {
	toolchain, err := readPinnedToolchainWith(context.Background(), root, gocommand.Default())
	return toolchain.ToolchainIdentity, err
}

// pinnedToolchain is a validated installation together with the identity its
// pinned Go command reports; preparation reads every installation location
// from it.
type pinnedToolchain struct {
	ToolchainIdentity
	installation installation.Description
}

func readPinnedToolchainWith(ctx context.Context, root string, runner gocommand.Runner) (pinnedToolchain, error) {
	description, err := installation.Describe(root)
	if err != nil {
		return pinnedToolchain{}, err
	}
	result, err := runner.StructuredCommand(ctx, gocommand.Request{
		Command: []string{description.GoCommand(), "env", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED"},
		Dir:     description.Root(), Env: targetbuild.Environment(), OutputLimit: maximumGoEnvironmentBytes,
	})
	if err == nil {
		err = result.OutputError()
	}
	if err != nil {
		return pinnedToolchain{}, fmt.Errorf("query pinned Go command: %w", err)
	}
	output := result.Stdout
	fields := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(fields) != 4 || fields[0] == "" || fields[1] == "" || fields[2] == "" || fields[3] != "0" {
		return pinnedToolchain{}, fmt.Errorf("pinned Go command returned invalid identity %q", output)
	}
	if fields[1] != runtime.GOOS || fields[2] != runtime.GOARCH {
		return pinnedToolchain{}, fmt.Errorf("pinned Go target %s/%s does not match host %s/%s", fields[1], fields[2], runtime.GOOS, runtime.GOARCH)
	}
	return pinnedToolchain{ToolchainIdentity: ToolchainIdentity{
		GoVersion:    fields[0],
		BuildKey:     description.BuildKey(),
		TargetGOOS:   fields[1],
		TargetGOARCH: fields[2],
	}, installation: description}, nil
}

func ReadModuleCache(ctx context.Context, root string) (string, error) {
	layout, err := installation.At(root)
	if err != nil {
		return "", fmt.Errorf("resolve pinned Go command: %w", err)
	}
	result, err := gocommand.Default().StructuredCommand(ctx, gocommand.Request{
		Command: []string{layout.GoCommand(), "env", "GOMODCACHE"},
		Dir:     layout.Root(), Env: targetbuild.Environment(), OutputLimit: maximumGoEnvironmentBytes,
	})
	if err == nil {
		err = result.OutputError()
	}
	if err != nil {
		return "", fmt.Errorf("query pinned module cache: %w", err)
	}
	output := result.Stdout
	path := strings.TrimSuffix(string(output), "\n")
	if path == "" || strings.Contains(path, "\n") || !filepath.IsAbs(path) {
		return "", fmt.Errorf("pinned Go command returned invalid module cache %q", output)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve pinned module cache: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return "", errors.New("pinned module cache is not a directory")
	}
	return path, nil
}

// DownloadModule places module in the pinned Go command's module cache, so a
// build input read from the cache does not depend on an earlier build having
// fetched it, and fails unless the downloaded module has the pinned checksum.
// It runs outside every module, because inside one the go command records
// the downloaded module's sums in that module's go.sum.
func DownloadModule(ctx context.Context, root string, module ModuleIdentity) (retErr error) {
	layout, err := installation.At(root)
	if err != nil {
		return fmt.Errorf("resolve pinned Go command: %w", err)
	}
	goCommand := layout.GoCommand()
	outside, err := os.MkdirTemp("", "gomad3-module-download-")
	if err != nil {
		return fmt.Errorf("create module download directory: %w", err)
	}
	defer func() { retErr = errors.Join(retErr, os.RemoveAll(outside)) }()
	query := module.Path + "@" + module.Version
	result, operationErr := gocommand.Default().StructuredCommand(ctx, gocommand.Request{
		Command: []string{goCommand, "mod", "download", "-json", query},
		Dir:     outside, Env: targetbuild.Environment(), OutputLimit: maximumGoModuleDownloadBytes,
	})
	if operationErr != nil {
		return fmt.Errorf("download pinned module %s: %w", query, operationErr)
	}
	runErr := result.CommandError
	output, stderr := result.Stdout, result.Stderr
	var downloaded struct {
		Sum   string
		Error string
	}
	if err := json.Unmarshal(output, &downloaded); err != nil {
		return fmt.Errorf("download pinned module %s: %w: %s", query, errors.Join(runErr, err), strings.TrimSpace(string(stderr)))
	}
	if downloaded.Error != "" {
		return fmt.Errorf("download pinned module %s: %s", query, downloaded.Error)
	}
	if runErr != nil {
		return fmt.Errorf("download pinned module %s: %w: %s", query, runErr, strings.TrimSpace(string(stderr)))
	}
	if downloaded.Sum != module.Sum {
		return fmt.Errorf("pinned module %s checksum mismatch: got %q, want %q", query, downloaded.Sum, module.Sum)
	}
	return nil
}

func WriteProvenance(path string, provenance Provenance) error {
	wire := provenanceWire{
		Schema:             provenanceSchema,
		SchemaVersion:      provenance.SchemaVersion,
		GoVersion:          provenance.GoVersion,
		BuildKey:           provenance.BuildKey,
		TargetGOOS:         provenance.TargetGOOS,
		TargetGOARCH:       provenance.TargetGOARCH,
		BinarySHA256:       provenance.BinarySHA256,
		BinarySize:         record.Uint64String(provenance.BinarySize),
		BuildInfo:          provenance.BuildInfo,
		CapabilityClosure:  provenance.CapabilityClosure,
		CapabilityMode:     provenance.CapabilityMode,
		CapabilityManifest: cloneCapabilityManifest(provenance.CapabilityManifest),
	}
	if err := validateProvenance(wire); err != nil {
		return err
	}
	_, err := targetprovenance.Store(path, wire)
	return err
}

func ReadProvenance(path string) (Provenance, error) {
	provenance, _, err := readProvenance(path)
	return provenance, err
}

func readProvenance(path string) (Provenance, []byte, error) {
	var wire provenanceWire
	encoded, err := targetprovenance.Load(path, maximumProvenanceBytes, &wire)
	if err != nil {
		return Provenance{}, nil, fmt.Errorf("read provenance: %w", err)
	}
	if err := validateProvenance(wire); err != nil {
		return Provenance{}, nil, err
	}
	return Provenance{
		SchemaVersion: wire.SchemaVersion, GoVersion: wire.GoVersion, BuildKey: wire.BuildKey,
		TargetGOOS: wire.TargetGOOS, TargetGOARCH: wire.TargetGOARCH,
		BinarySHA256: wire.BinarySHA256, BinarySize: uint64(wire.BinarySize),
		BuildInfo: wire.BuildInfo, CapabilityClosure: wire.CapabilityClosure,
		CapabilityMode: wire.CapabilityMode, CapabilityManifest: cloneCapabilityManifest(wire.CapabilityManifest),
	}, encoded, nil
}

func prepareExec(ctx context.Context, spec Spec, identity pinnedToolchain, targetPath string) (preparation, error) {
	if spec.Source == "" || spec.Provenance == "" {
		return preparation{}, errors.New("exec target and provenance are required")
	}
	provenance, provenanceBytes, err := readProvenance(spec.Provenance)
	if err != nil {
		return preparation{}, fmt.Errorf("read exec provenance: %w", err)
	}
	if provenance.GoVersion != identity.GoVersion || provenance.BuildKey != identity.BuildKey || provenance.TargetGOOS != identity.TargetGOOS || provenance.TargetGOARCH != identity.TargetGOARCH {
		return preparation{}, errors.New("exec provenance does not match pinned toolchain")
	}
	if provenance.CapabilityMode != spec.CapabilityMode {
		return preparation{}, errors.New("exec provenance capability mode does not match the requested mode")
	}
	if err := validateExecStandardPackages(ctx, identity.installation.GoCommand(), provenance.CapabilityClosure); err != nil {
		return preparation{}, err
	}
	if err := copyRegularFile(spec.Source, targetPath); err != nil {
		return preparation{}, err
	}
	hash, size, err := hashRegularFile(targetPath)
	if err != nil {
		return preparation{}, fmt.Errorf("hash prepared provenance binary: %w", err)
	}
	if hash != provenance.BinarySHA256 || size != provenance.BinarySize {
		return preparation{}, errors.New("provenance binary identity does not match prepared target")
	}
	info, err := buildinfo.ReadFile(targetPath)
	if err != nil {
		return preparation{}, fmt.Errorf("read prepared exec target build info: %w", err)
	}
	if err := validateExecCapabilityModules(info, provenance.CapabilityClosure); err != nil {
		return preparation{}, err
	}
	actualBuildInfo := ProjectBuildInfo(info)
	recordedBuildInfo, err := canonicaljson.CanonicalJSON(provenance.BuildInfo)
	if err != nil {
		return preparation{}, fmt.Errorf("encode provenance build info: %w", err)
	}
	actualBuildInfoBytes, err := canonicaljson.CanonicalJSON(actualBuildInfo)
	if err != nil {
		return preparation{}, fmt.Errorf("encode exec target build info: %w", err)
	}
	if actualBuildInfo.GoVersion != targetbuild.BinaryGoVersion(provenance.GoVersion) || string(actualBuildInfoBytes) != string(recordedBuildInfo) {
		return preparation{}, errors.New("exec target build info does not match provenance")
	}
	if err := writePreparedFile(filepath.Join(filepath.Dir(targetPath), "provenance.json"), provenanceBytes, 0o400); err != nil {
		return preparation{}, fmt.Errorf("snapshot exec provenance: %w", err)
	}
	prepared := preparation{buildInfo: provenance.BuildInfo, compatibility: recordCompatibility(provenance.CapabilityClosure.Compatibility)}
	if provenance.CapabilityMode != CapabilityModeClosure {
		record, err := readLinkedCapabilityRecord(targetPath, identity.ToolchainIdentity)
		if err != nil {
			return preparation{}, fmt.Errorf("extract exec target capability manifest: %w", err)
		}
		actual := capabilityManifest(record)
		if !sameCapabilityManifest(provenance.CapabilityManifest, actual) {
			return preparation{}, errors.New("exec target capability manifest does not match provenance")
		}
		review, err := reviewRecordedClosure(provenance.CapabilityClosure, nil)
		if err != nil {
			return preparation{}, fmt.Errorf("exec provenance capability closure: %w", err)
		}
		review = projectLinkedCapabilityReview(review, record, provenance.CapabilityMode)
		if len(review.Findings) != 0 {
			return preparation{}, unsupportedFinding(review.Findings[0])
		}
		prepared.manifest = actual
	}
	return prepared, nil
}

func validateExecCapabilityModules(info *debug.BuildInfo, closure CapabilityClosure) error {
	mainModules := make(map[string]struct{})
	reviewed := make(map[string]struct{})
	for _, pkg := range closure.Packages {
		if pkg.Module == nil {
			continue
		}
		if pkg.Module.Main {
			mainModules[pkg.Module.Path] = struct{}{}
			continue
		}
		reviewed[capabilityModuleIdentity(pkg.Module)] = struct{}{}
	}
	if len(mainModules) != 1 {
		return fmt.Errorf("exec provenance capability closure must identify one main module")
	}
	if _, found := mainModules[info.Main.Path]; !found {
		return fmt.Errorf("exec target main module does not match capability closure")
	}
	actual := make(map[string]struct{}, len(info.Deps))
	for _, module := range info.Deps {
		actual[debugModuleIdentity(module)] = struct{}{}
	}
	if !sameStringSet(reviewed, actual) {
		return fmt.Errorf("exec target module dependencies do not match capability closure")
	}
	return nil
}

func capabilityModuleIdentity(module *CapabilityModule) string {
	replacement := ""
	if module.Replacement != nil {
		if module.Replacement.Local {
			replacement = "local"
		} else {
			replacement = module.Replacement.Path + "\x00" + module.Replacement.Version + "\x00" + module.Replacement.Sum
		}
	}
	return module.Path + "\x00" + module.Version + "\x00" + module.Sum + "\x00" + replacement
}

func debugModuleIdentity(module *debug.Module) string {
	replacement := ""
	if module.Replace != nil {
		if module.Replace.Version == "" && module.Replace.Sum == "" {
			replacement = "local"
		} else {
			replacement = module.Replace.Path + "\x00" + module.Replace.Version + "\x00" + module.Replace.Sum
		}
	}
	return module.Path + "\x00" + module.Version + "\x00" + module.Sum + "\x00" + replacement
}

func sameStringSet(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, found := right[value]; !found {
			return false
		}
	}
	return true
}

func prepareGo(ctx context.Context, spec Spec, tags []string, identity pinnedToolchain, targetPath string, runner gocommand.Runner) (preparation, error) {
	if spec.Source == "" || spec.WorkingDir == "" {
		return preparation{}, errors.New("go target source and working directory are required")
	}
	if strings.HasPrefix(spec.Source, "-") || strings.Contains(spec.Source, "...") || strings.IndexFunc(spec.Source, unicode.IsSpace) >= 0 || strings.IndexByte(spec.Source, 0) >= 0 {
		return preparation{}, fmt.Errorf("go target package argument %q must select exactly one package", spec.Source)
	}
	goCommand := identity.installation.GoCommand()
	buildContext, err := targetbuild.Resolve(spec.WorkingDir, spec.Source, tags)
	if err != nil {
		return preparation{}, err
	}
	review, packages, err := reviewGoCapabilityPackagesWith(ctx, goCommand, spec, buildContext.Tags, buildContext.Directory, buildContext.Package, runner)
	if err != nil {
		return preparation{}, err
	}
	cache, err := openPreparedTargetCache(spec, buildContext.Tags, identity, buildContext.Directory, buildContext.Package, review, packages)
	if err != nil {
		return preparation{}, err
	}
	return buildGoTargetWith(ctx, spec, buildContext.Tags, identity, targetPath, goCommand, buildContext.Directory, buildContext.Package, review, rejectUnsupported, cache, runner)
}

func buildGoTarget(
	ctx context.Context,
	spec Spec,
	tags []string,
	identity pinnedToolchain,
	targetPath string,
	goCommand string,
	commandDirectory string,
	packageArgument string,
	review CapabilityReview,
	policy unsupportedPolicy,
	cache *preparedTargetCache,
) (preparation, error) {
	return buildGoTargetWith(ctx, spec, tags, identity, targetPath, goCommand, commandDirectory, packageArgument, review, policy, cache, gocommand.Default())
}

func buildGoTargetWith(
	ctx context.Context, spec Spec, tags []string, identity pinnedToolchain, targetPath string,
	goCommand, commandDirectory, packageArgument string, review CapabilityReview,
	policy unsupportedPolicy, cache *preparedTargetCache, runner gocommand.Runner,
) (preparation, error) {
	if policy == rejectUnsupported && spec.CapabilityMode == CapabilityModeClosure && len(review.Findings) != 0 {
		return preparation{}, unsupportedFinding(review.Findings[0])
	}
	if cache != nil {
		reused, err := cache.restore(targetPath)
		if err != nil {
			return preparation{}, err
		}
		if reused {
			return finishGoTarget(spec, identity.ToolchainIdentity, targetPath, review, policy)
		}
	}
	arguments := []string{}
	if spec.Kind == KindGoRun {
		arguments = append(arguments, "build")
	} else {
		arguments = append(arguments, "test", "-c")
	}
	// VCS stamping would record repository state no prepared-target identity
	// binds, so a restored binary could carry another commit's build info.
	arguments = append(arguments, "-trimpath", "-buildvcs=false", "-o", targetPath)
	if spec.CapabilityMode != CapabilityModeClosure {
		gcflags := "-gcflags=all=-gomadcap"
		if spec.CapabilityMode == CapabilityModeGuarded {
			gcflags += " -gomadguard"
		}
		arguments = append(arguments, gcflags, "-ldflags=-linkmode=internal -gomadcap="+identity.BuildKey)
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
	buildCache, err := targetbuild.PrepareCache(identity.installation.PinnedBuild().TargetCache())
	if err != nil {
		return preparation{}, err
	}
	cacheUse, err := targetbuild.UseCache(buildCache)
	if err != nil {
		return preparation{}, err
	}
	result, err := runner.DiagnosticCommand(ctx, gocommand.Request{
		Command: append([]string{goCommand}, arguments...), Dir: commandDirectory,
		Env: append(targetbuild.Environment(), "GOCACHE="+buildCache), OutputLimit: maximumGoBuildDiagnosticBytes,
	})
	output := result.Output.Bytes
	if releaseErr := cacheUse.Release(); releaseErr != nil {
		return preparation{}, fmt.Errorf("release target build cache: %w", releaseErr)
	}
	if err == nil {
		err = result.CommandError
	}
	if err != nil {
		if spec.CapabilityMode != CapabilityModeClosure {
			return preparation{}, fmt.Errorf("prepare %s target: %w", spec.Kind, linkedCapabilityBuildError(err, output))
		}
		return preparation{}, fmt.Errorf("prepare %s target: %w: %s", spec.Kind, err, output)
	}
	if cache != nil {
		if err := cache.publish(targetPath); err != nil {
			return preparation{}, err
		}
	}
	if err := targetbuild.TrimCache(buildCache, targetbuild.MaximumCacheBytes); err != nil {
		return preparation{}, err
	}
	return finishGoTarget(spec, identity.ToolchainIdentity, targetPath, review, policy)
}

// finishGoTarget projects the capability evidence of a built or restored go
// target; a restored binary gets the same linked-mode extraction as a fresh one.
func finishGoTarget(spec Spec, identity ToolchainIdentity, targetPath string, review CapabilityReview, policy unsupportedPolicy) (preparation, error) {
	prepared := preparation{compatibility: recordCompatibility(review.Closure.Compatibility), review: review}
	if spec.CapabilityMode != CapabilityModeClosure {
		record, err := readLinkedCapabilityRecord(targetPath, identity)
		if err != nil {
			return preparation{}, fmt.Errorf("extract linked target capability manifest: %w", err)
		}
		prepared.review = projectLinkedCapabilityReview(review, record, spec.CapabilityMode)
		prepared.manifest = capabilityManifest(record)
		if policy == rejectUnsupported && len(prepared.review.Findings) != 0 {
			return preparation{}, unsupportedFinding(prepared.review.Findings[0])
		}
	}
	return prepared, nil
}

func normalizeCapabilityMode(mode CapabilityMode) (CapabilityMode, error) {
	if mode == "" {
		return CapabilityModeClosure, nil
	}
	switch mode {
	case CapabilityModeClosure, CapabilityModeLinked, CapabilityModeGuarded:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported capability mode %q", mode)
	}
}

func validateProvenance(provenance provenanceWire) error {
	if provenance.SchemaVersion != 3 || provenance.Schema != provenanceSchema {
		return fmt.Errorf("unsupported exec provenance schema")
	}
	recorded := record.Target{CapabilityMode: string(provenance.CapabilityMode)}
	if provenance.CapabilityManifest != nil {
		recorded.CapabilityManifest = provenance.CapabilityManifest.Record()
	}
	if err := record.ValidateCurrentTargetCapability(recorded); err != nil {
		return fmt.Errorf("exec provenance capability evidence: %w", err)
	}
	if provenance.GoVersion == "" || provenance.BuildKey == "" || provenance.TargetGOOS == "" || provenance.TargetGOARCH == "" || provenance.BuildInfo.GoVersion == "" || provenance.BuildInfo.Path == "" {
		return fmt.Errorf("exec provenance has an empty identity field")
	}
	if len(provenance.BuildKey) != sha256.Size*2 || !isLowerHex(provenance.BuildKey) {
		return fmt.Errorf("exec provenance build key is malformed")
	}
	if _, err := record.ParseSHA256(provenance.BinarySHA256); err != nil {
		return fmt.Errorf("exec provenance binary hash is malformed")
	}
	if err := validateDeterministicBuildInfo(provenance.BuildInfo); err != nil {
		return err
	}
	review, err := reviewRecordedClosure(provenance.CapabilityClosure, nil)
	if err != nil {
		return fmt.Errorf("exec provenance capability closure: %w", err)
	}
	if provenance.CapabilityMode == CapabilityModeClosure && len(review.Findings) != 0 {
		return fmt.Errorf("exec provenance capability closure: %w", unsupportedFinding(review.Findings[0]))
	}
	return nil
}

func validateDeterministicBuildInfo(info record.BuildInfo) error {
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["CGO_ENABLED"] != "0" {
		return fmt.Errorf("exec provenance requires CGO_ENABLED=0")
	}
	if HasCoverageInstrumentation(info) {
		return fmt.Errorf("exec provenance uses unsupported coverage instrumentation")
	}
	if settings["-race"] == "true" {
		return fmt.Errorf("exec provenance uses the unsupported race detector")
	}
	if buildMode := settings["-buildmode"]; buildMode != "" && buildMode != "exe" {
		return fmt.Errorf("exec provenance uses unsupported build mode %q", buildMode)
	}
	if settings["-linkshared"] == "true" {
		return fmt.Errorf("exec provenance uses unsupported shared-library linking")
	}
	ldflags := settings["-ldflags"]
	if strings.Contains(ldflags, "-linkmode=external") || strings.Contains(ldflags, "-linkmode external") || strings.Contains(ldflags, "-buildmode=plugin") {
		return fmt.Errorf("exec provenance uses unsupported external or plugin linking")
	}
	return nil
}

func HasCoverageInstrumentation(info record.BuildInfo) bool {
	for _, setting := range info.Settings {
		if setting.Key == "-cover" && setting.Value == "true" {
			return true
		}
	}
	return false
}

func ProjectBuildInfo(info *debug.BuildInfo) record.BuildInfo {
	settings := make([]record.BuildSetting, len(info.Settings))
	for index, setting := range info.Settings {
		settings[index] = record.BuildSetting{Key: setting.Key, Value: setting.Value}
	}
	sort.Slice(settings, func(i, j int) bool { return settings[i].Key < settings[j].Key })
	mainModule := info.Main.Path
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		mainModule += "@" + info.Main.Version
	}
	return record.BuildInfo{GoVersion: info.GoVersion, Path: info.Path, MainModule: mainModule, Settings: settings}
}

func hashRegularFile(path string) (digest string, fileSize uint64, retErr error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		if errors.Is(err, hostfs.ErrSymbolicLink) {
			return "", 0, fmt.Errorf("%s is not a regular executable", path)
		}
		return "", 0, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			digest, fileSize = "", 0
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", 0, fmt.Errorf("%s is not a regular executable", path)
	}
	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}
	if size < 0 {
		return "", 0, fmt.Errorf("negative target size")
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), uint64(size), nil
}

func copyRegularFile(source, destination string) (retErr error) {
	input, info, err := hostfs.OpenPath(source)
	if err != nil {
		if errors.Is(err, hostfs.ErrSymbolicLink) {
			return fmt.Errorf("exec target is not a regular executable")
		}
		return fmt.Errorf("stat exec target: %w", err)
	}
	defer func() {
		if closeErr := input.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("exec target is not a regular executable")
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return fmt.Errorf("create prepared exec target: %w", err)
	}
	if err := output.Chmod(0o700); err != nil {
		if closeErr := output.Close(); closeErr != nil {
			return errors.Join(fmt.Errorf("set prepared exec target mode: %w", err), closeErr)
		}
		return fmt.Errorf("set prepared exec target mode: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		if closeErr := output.Close(); closeErr != nil {
			return errors.Join(fmt.Errorf("copy exec target: %w", err), closeErr)
		}
		return fmt.Errorf("copy exec target: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close prepared exec target: %w", err)
	}
	return nil
}

func writePreparedFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}
	if _, err := file.Write(data); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}
	if err := file.Sync(); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}
	return file.Close()
}

func isLowerHex(value string) bool {
	for _, character := range value {
		if character < '0' || character > '9' && character < 'a' || character > 'f' {
			return false
		}
	}
	return true
}
