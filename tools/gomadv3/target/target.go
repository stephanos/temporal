package target

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"unicode"

	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/internal/hostfs"
	"go.temporal.io/server/tools/gomadv3/target/internal/livecap"
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
)

const (
	provenanceSchema      = "gomadv3.exec-provenance/v3"
	priorProvenanceSchema = "gomadv3.exec-provenance/v2"
)

const maximumProvenanceBytes = 16 << 20

type Spec struct {
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
	Path               string
	Kind               Kind
	Source             string
	SHA256             string
	Size               uint64
	Argv               []string
	BuildTags          []string
	Adapters           []evidence.TargetAdapter
	Compatibility      []evidence.CompatibilityPack
	BuildInfo          evidence.BuildInfo
	GoVersion          string
	BuildKey           string
	TargetGOOS         string
	TargetGOARCH       string
	CapabilityMode     CapabilityMode
	CapabilityManifest *CapabilityManifest
}

type CapabilityManifest struct {
	Schema                       string          `json:"schema"`
	SHA256                       evidence.SHA256 `json:"sha256"`
	Bytes                        uint64          `json:"bytes"`
	Facts                        uint64          `json:"facts"`
	ProducerImplementationSHA256 string          `json:"producer_implementation_sha256"`
	CapabilityUniverseSHA256     string          `json:"capability_universe_sha256"`
	Payload                      []byte          `json:"-"`
}

func (prepared Prepared) RecordTarget() evidence.Target {
	buildInfo := prepared.BuildInfo
	buildInfo.Settings = append([]evidence.BuildSetting(nil), prepared.BuildInfo.Settings...)
	recorded := evidence.Target{
		Kind: string(prepared.Kind), Source: prepared.Source, SHA256: evidence.SHA256(prepared.SHA256), Size: evidence.Uint64String(prepared.Size),
		Argv: append([]string{}, prepared.Argv...), BuildTags: append([]string{}, prepared.BuildTags...),
		Adapters: append([]evidence.TargetAdapter{}, prepared.Adapters...), Compatibility: append([]evidence.CompatibilityPack{}, prepared.Compatibility...), BuildInfo: buildInfo,
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

func (manifest CapabilityManifest) Record() *evidence.TargetCapabilityManifest {
	return &evidence.TargetCapabilityManifest{
		Schema: manifest.Schema, File: "target-capabilities.json", SHA256: manifest.SHA256,
		Bytes: evidence.Uint64String(manifest.Bytes), Facts: evidence.Uint64String(manifest.Facts),
		ProducerImplementationSHA256: evidence.SHA256(manifest.ProducerImplementationSHA256),
		CapabilityUniverseSHA256:     evidence.SHA256(manifest.CapabilityUniverseSHA256),
	}
}

func CapabilityManifestFromRecord(manifest *evidence.TargetCapabilityManifest) *CapabilityManifest {
	if manifest == nil {
		return nil
	}
	return &CapabilityManifest{
		Schema: manifest.Schema, SHA256: manifest.SHA256, Bytes: uint64(manifest.Bytes), Facts: uint64(manifest.Facts),
		ProducerImplementationSHA256: string(manifest.ProducerImplementationSHA256),
		CapabilityUniverseSHA256:     string(manifest.CapabilityUniverseSHA256),
	}
}

func (prepared Prepared) RecordToolchain() evidence.Toolchain {
	return evidence.Toolchain{
		GoVersion: prepared.GoVersion, BuildKey: prepared.BuildKey,
		TargetGOOS: prepared.TargetGOOS, TargetGOARCH: prepared.TargetGOARCH,
	}
}

type preparation struct {
	buildInfo     evidence.BuildInfo
	compatibility []evidence.CompatibilityPack
	review        CapabilityReview
	manifest      *CapabilityManifest
}

type Provenance struct {
	SchemaVersion      int
	GoVersion          string
	BuildKey           string
	TargetGOOS         string
	TargetGOARCH       string
	BinarySHA256       string
	BinarySize         uint64
	BuildInfo          evidence.BuildInfo
	CapabilityClosure  CapabilityClosure
	CapabilityMode     CapabilityMode
	CapabilityManifest *CapabilityManifest
}

type provenanceWire struct {
	Schema             string                `json:"schema"`
	SchemaVersion      int                   `json:"schema_version"`
	GoVersion          string                `json:"go_version"`
	BuildKey           string                `json:"build_key"`
	TargetGOOS         string                `json:"target_goos"`
	TargetGOARCH       string                `json:"target_goarch"`
	BinarySHA256       string                `json:"binary_sha256"`
	BinarySize         evidence.Uint64String `json:"binary_size"`
	BuildInfo          evidence.BuildInfo    `json:"build_info"`
	CapabilityClosure  CapabilityClosure     `json:"capability_closure"`
	CapabilityMode     CapabilityMode        `json:"capability_mode,omitempty"`
	CapabilityManifest *CapabilityManifest   `json:"capability_manifest,omitempty"`
}

func Prepare(ctx context.Context, spec Spec) (prepared Prepared, retErr error) {
	tags, err := normalizeBuildTags(spec.BuildTags)
	if err != nil {
		return Prepared{}, err
	}
	mode, err := normalizeCapabilityMode(spec.CapabilityMode)
	if err != nil {
		return Prepared{}, err
	}
	spec.CapabilityMode = mode
	identity, err := ReadToolchainIdentity(spec.ToolchainRoot)
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
		preparedTarget, err = prepareGo(ctx, spec, tags, identity, targetPath)
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
		Argv:               append([]string{"gomadv3-target"}, spec.Args...),
		BuildTags:          tags,
		Adapters:           []evidence.TargetAdapter{},
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
	if mode == CapabilityModeLinked {
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
	record, err := livecap.Read(path, livecap.Expectation{
		GoVersion: identity.GoVersion, ToolchainBuildKey: identity.BuildKey, GOOS: identity.TargetGOOS, GOARCH: identity.TargetGOARCH,
	})
	if err != nil {
		return nil, linkedCapabilityError(err)
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
	if root == "" {
		return ToolchainIdentity{}, fmt.Errorf("toolchain root is required")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return ToolchainIdentity{}, fmt.Errorf("resolve toolchain root: %w", err)
	}
	goCommand := filepath.Join(root, "bin", "go")
	info, err := os.Lstat(goCommand)
	if err != nil {
		return ToolchainIdentity{}, fmt.Errorf("stat pinned Go command in %s: %w; set --toolchain-root or GOMADV3_TOOLCHAIN_DIR to a complete Gomad installation", root, err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return ToolchainIdentity{}, fmt.Errorf("pinned Go command is not a regular executable")
	}
	buildKeyBytes, err := os.ReadFile(filepath.Join(root, "build-key"))
	if err != nil {
		return ToolchainIdentity{}, fmt.Errorf("read toolchain build key in %s: %w; set --toolchain-root or GOMADV3_TOOLCHAIN_DIR to a complete Gomad installation", root, err)
	}
	buildKey := strings.TrimSuffix(string(buildKeyBytes), "\n")
	if len(buildKey) != sha256.Size*2 || !isLowerHex(buildKey) || string(buildKeyBytes) != buildKey+"\n" {
		return ToolchainIdentity{}, fmt.Errorf("toolchain build key is malformed")
	}
	builtGo := filepath.Join(root, "builds", buildKey, "bin", "go")
	if builtInfo, statErr := os.Stat(builtGo); statErr != nil || !builtInfo.Mode().IsRegular() || builtInfo.Mode()&0o111 == 0 {
		return ToolchainIdentity{}, fmt.Errorf("toolchain build %s is missing or stale in %s; set --toolchain-root or GOMADV3_TOOLCHAIN_DIR to a complete Gomad installation", buildKey, root)
	}
	command := exec.Command(goCommand, "env", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED")
	command.Env = preparationEnvironment()
	output, err := command.Output()
	if err != nil {
		return ToolchainIdentity{}, fmt.Errorf("query pinned Go command: %w", err)
	}
	fields := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
	if len(fields) != 4 || fields[0] == "" || fields[1] == "" || fields[2] == "" || fields[3] != "0" {
		return ToolchainIdentity{}, fmt.Errorf("pinned Go command returned invalid identity %q", output)
	}
	if fields[1] != runtime.GOOS || fields[2] != runtime.GOARCH {
		return ToolchainIdentity{}, fmt.Errorf("pinned Go target %s/%s does not match host %s/%s", fields[1], fields[2], runtime.GOOS, runtime.GOARCH)
	}
	return ToolchainIdentity{
		GoVersion:    fields[0],
		BuildKey:     buildKey,
		TargetGOOS:   fields[1],
		TargetGOARCH: fields[2],
	}, nil
}

func ReadModuleCache(ctx context.Context, root string) (string, error) {
	goCommand, err := filepath.Abs(filepath.Join(root, "bin", "go"))
	if err != nil {
		return "", fmt.Errorf("resolve pinned Go command: %w", err)
	}
	command := exec.CommandContext(ctx, goCommand, "env", "GOMODCACHE")
	command.Env = preparationEnvironment()
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("query pinned module cache: %w", err)
	}
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

func WriteProvenance(path string, provenance Provenance) error {
	schema := provenanceSchema
	if provenance.SchemaVersion == 2 {
		schema = priorProvenanceSchema
	}
	wire := provenanceWire{
		Schema:             schema,
		SchemaVersion:      provenance.SchemaVersion,
		GoVersion:          provenance.GoVersion,
		BuildKey:           provenance.BuildKey,
		TargetGOOS:         provenance.TargetGOOS,
		TargetGOARCH:       provenance.TargetGOARCH,
		BinarySHA256:       provenance.BinarySHA256,
		BinarySize:         evidence.Uint64String(provenance.BinarySize),
		BuildInfo:          provenance.BuildInfo,
		CapabilityClosure:  provenance.CapabilityClosure,
		CapabilityMode:     provenance.CapabilityMode,
		CapabilityManifest: cloneCapabilityManifest(provenance.CapabilityManifest),
	}
	if err := validateProvenance(wire); err != nil {
		return err
	}
	encoded, err := evidence.CanonicalJSON(wire)
	if err != nil {
		return fmt.Errorf("encode provenance: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write provenance: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		return errors.Join(fmt.Errorf("set provenance mode: %w", err), file.Close())
	}
	if _, err := file.Write(encoded); err != nil {
		return errors.Join(fmt.Errorf("write provenance: %w", err), file.Close())
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close provenance: %w", err)
	}
	return nil
}

func ReadProvenance(path string) (Provenance, error) {
	provenance, _, err := readProvenance(path)
	return provenance, err
}

func readProvenance(path string) (Provenance, []byte, error) {
	encoded, err := readBoundedRegularFile(path, maximumProvenanceBytes)
	if err != nil {
		return Provenance{}, nil, fmt.Errorf("read provenance: %w", err)
	}
	var wire provenanceWire
	if err := evidence.DecodeCanonicalJSON(encoded, &wire); err != nil {
		return Provenance{}, nil, fmt.Errorf("decode provenance: %w", err)
	}
	if err := validateProvenance(wire); err != nil {
		return Provenance{}, nil, err
	}
	if wire.SchemaVersion == 2 {
		wire.CapabilityMode = CapabilityModeClosure
	}
	return Provenance{
		SchemaVersion: wire.SchemaVersion, GoVersion: wire.GoVersion, BuildKey: wire.BuildKey,
		TargetGOOS: wire.TargetGOOS, TargetGOARCH: wire.TargetGOARCH,
		BinarySHA256: wire.BinarySHA256, BinarySize: uint64(wire.BinarySize),
		BuildInfo: wire.BuildInfo, CapabilityClosure: wire.CapabilityClosure,
		CapabilityMode: wire.CapabilityMode, CapabilityManifest: cloneCapabilityManifest(wire.CapabilityManifest),
	}, encoded, nil
}

func prepareExec(ctx context.Context, spec Spec, identity ToolchainIdentity, targetPath string) (preparation, error) {
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
	if err := validateExecStandardPackages(ctx, filepath.Join(spec.ToolchainRoot, "bin", "go"), provenance.CapabilityClosure); err != nil {
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
	recordedBuildInfo, err := evidence.CanonicalJSON(provenance.BuildInfo)
	if err != nil {
		return preparation{}, fmt.Errorf("encode provenance build info: %w", err)
	}
	actualBuildInfoBytes, err := evidence.CanonicalJSON(actualBuildInfo)
	if err != nil {
		return preparation{}, fmt.Errorf("encode exec target build info: %w", err)
	}
	if actualBuildInfo.GoVersion != provenance.GoVersion || string(actualBuildInfoBytes) != string(recordedBuildInfo) {
		return preparation{}, errors.New("exec target build info does not match provenance")
	}
	if err := writePreparedFile(filepath.Join(filepath.Dir(targetPath), "provenance.json"), provenanceBytes, 0o400); err != nil {
		return preparation{}, fmt.Errorf("snapshot exec provenance: %w", err)
	}
	prepared := preparation{buildInfo: provenance.BuildInfo, compatibility: recordCompatibility(provenance.CapabilityClosure.Compatibility)}
	if provenance.CapabilityMode == CapabilityModeLinked {
		record, err := livecap.Read(targetPath, livecap.Expectation{
			GoVersion: identity.GoVersion, ToolchainBuildKey: identity.BuildKey, GOOS: identity.TargetGOOS, GOARCH: identity.TargetGOARCH,
		})
		if err != nil {
			return preparation{}, fmt.Errorf("extract exec target capability manifest: %w", linkedCapabilityError(err))
		}
		actual := capabilityManifest(record)
		if !sameCapabilityManifest(provenance.CapabilityManifest, actual) {
			return preparation{}, errors.New("exec target capability manifest does not match provenance")
		}
		selection, err := validateCapabilityReviewStructure(provenance.CapabilityClosure)
		if err != nil {
			return preparation{}, fmt.Errorf("exec provenance capability closure: %w", err)
		}
		review := projectLinkedCapabilityReview(capabilityReviewFromClosure(provenance.CapabilityClosure, nil, selection), record)
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

func prepareGo(ctx context.Context, spec Spec, tags []string, identity ToolchainIdentity, targetPath string) (preparation, error) {
	if spec.Source == "" || spec.WorkingDir == "" {
		return preparation{}, errors.New("go target source and working directory are required")
	}
	if strings.HasPrefix(spec.Source, "-") || strings.Contains(spec.Source, "...") || strings.IndexFunc(spec.Source, unicode.IsSpace) >= 0 || strings.IndexByte(spec.Source, 0) >= 0 {
		return preparation{}, fmt.Errorf("go target package argument %q must select exactly one package", spec.Source)
	}
	goCommand, err := filepath.Abs(filepath.Join(spec.ToolchainRoot, "bin", "go"))
	if err != nil {
		return preparation{}, fmt.Errorf("resolve pinned Go command: %w", err)
	}
	commandDirectory, packageArgument, err := resolveBuildContext(spec.WorkingDir, spec.Source)
	if err != nil {
		return preparation{}, err
	}
	review, err := reviewGoCapabilityReview(ctx, goCommand, spec, tags, commandDirectory, packageArgument)
	if err != nil {
		return preparation{}, err
	}
	return buildGoTarget(ctx, spec, tags, identity, targetPath, goCommand, commandDirectory, packageArgument, review, true)
}

func buildGoTarget(
	ctx context.Context,
	spec Spec,
	tags []string,
	identity ToolchainIdentity,
	targetPath string,
	goCommand string,
	commandDirectory string,
	packageArgument string,
	review CapabilityReview,
	rejectUnsupported bool,
) (preparation, error) {
	if rejectUnsupported && spec.CapabilityMode == CapabilityModeClosure && len(review.Findings) != 0 {
		return preparation{}, unsupportedFinding(review.Findings[0])
	}
	arguments := []string{}
	if spec.Kind == KindGoRun {
		arguments = append(arguments, "build")
	} else {
		arguments = append(arguments, "test", "-c")
	}
	arguments = append(arguments, "-trimpath", "-o", targetPath)
	if spec.CapabilityMode == CapabilityModeLinked {
		arguments = append(arguments, "-gcflags=all=-gomadcap", "-ldflags=-linkmode=internal -gomadcap="+identity.BuildKey)
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
	output, err := command.CombinedOutput()
	if err != nil {
		if spec.CapabilityMode == CapabilityModeLinked {
			return preparation{}, fmt.Errorf("prepare %s target: %w", spec.Kind, linkedCapabilityBuildError(err, output))
		}
		return preparation{}, fmt.Errorf("prepare %s target: %w: %s", spec.Kind, err, output)
	}
	prepared := preparation{compatibility: recordCompatibility(review.Closure.Compatibility), review: review}
	if spec.CapabilityMode == CapabilityModeLinked {
		record, err := livecap.Read(targetPath, livecap.Expectation{
			GoVersion: identity.GoVersion, ToolchainBuildKey: identity.BuildKey, GOOS: identity.TargetGOOS, GOARCH: identity.TargetGOARCH,
		})
		if err != nil {
			return preparation{}, fmt.Errorf("extract linked target capability manifest: %w", linkedCapabilityError(err))
		}
		prepared.review = projectLinkedCapabilityReview(review, record)
		prepared.manifest = capabilityManifest(record)
		if rejectUnsupported && len(prepared.review.Findings) != 0 {
			return preparation{}, unsupportedFinding(prepared.review.Findings[0])
		}
	}
	return prepared, nil
}

func resolveBuildContext(workingDirectory, source string) (string, string, error) {
	if !filepath.IsAbs(source) && !strings.HasPrefix(source, ".") {
		return workingDirectory, source, nil
	}
	packagePath := source
	if !filepath.IsAbs(packagePath) {
		packagePath = filepath.Join(workingDirectory, packagePath)
	}
	packagePath, err := filepath.Abs(packagePath)
	if err != nil {
		return "", "", fmt.Errorf("resolve Go target package: %w", err)
	}
	info, err := os.Stat(packagePath)
	if err != nil {
		return "", "", fmt.Errorf("stat Go target package: %w", err)
	}
	if !info.IsDir() {
		return "", "", fmt.Errorf("Go target package %s is not a directory", source)
	}
	for directory := packagePath; ; directory = filepath.Dir(directory) {
		moduleFile := filepath.Join(directory, "go.mod")
		if moduleInfo, statErr := os.Stat(moduleFile); statErr == nil && moduleInfo.Mode().IsRegular() {
			relative, relErr := filepath.Rel(directory, packagePath)
			if relErr != nil {
				return "", "", fmt.Errorf("resolve Go target within module: %w", relErr)
			}
			if relative == "." {
				return directory, ".", nil
			}
			return directory, "./" + filepath.ToSlash(relative), nil
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			break
		}
	}
	return "", "", fmt.Errorf("Go target package %s has no owning go.mod", source)
}

func normalizeBuildTags(supplied []string) ([]string, error) {
	set := make(map[string]struct{}, len(supplied))
	for _, tag := range supplied {
		if tag == "" || tag == "race" || strings.Contains(tag, ",") || strings.IndexFunc(tag, unicode.IsSpace) >= 0 {
			return nil, fmt.Errorf("unsupported build tag %q", tag)
		}
		set[tag] = struct{}{}
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

func normalizeCapabilityMode(mode CapabilityMode) (CapabilityMode, error) {
	if mode == "" {
		return CapabilityModeClosure, nil
	}
	switch mode {
	case CapabilityModeClosure, CapabilityModeLinked:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported capability mode %q", mode)
	}
}

func preparationEnvironment() []string {
	reserved := map[string]struct{}{
		"CGO_ENABLED": {}, "GOMADSEED": {}, "GOMADV3_CHILD_SEED": {},
		"GOENV": {}, "GOEXPERIMENT": {}, "GOFLAGS": {}, "GOROOT": {}, "GOTOOLCHAIN": {}, "GOWORK": {}, "TZ": {},
	}
	environment := make([]string, 0, len(os.Environ())+5)
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if _, found := reserved[name]; !found {
			environment = append(environment, entry)
		}
	}
	environment = append(environment, "CGO_ENABLED=0", "GOENV=off", "GOEXPERIMENT=", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "TZ=UTC")
	return environment
}

func validateProvenance(provenance provenanceWire) error {
	if provenance.SchemaVersion != 2 && provenance.SchemaVersion != 3 || provenance.SchemaVersion == 2 && provenance.Schema != priorProvenanceSchema || provenance.SchemaVersion == 3 && provenance.Schema != provenanceSchema {
		return fmt.Errorf("unsupported exec provenance schema")
	}
	if provenance.SchemaVersion == 2 {
		if provenance.CapabilityMode != "" || provenance.CapabilityManifest != nil {
			return fmt.Errorf("historical exec provenance contains linked capability evidence")
		}
	} else {
		recorded := evidence.Target{CapabilityMode: string(provenance.CapabilityMode)}
		if provenance.CapabilityManifest != nil {
			recorded.CapabilityManifest = provenance.CapabilityManifest.Record()
		}
		if err := evidence.ValidateCurrentTargetCapability(recorded); err != nil {
			return fmt.Errorf("exec provenance capability evidence: %w", err)
		}
	}
	if provenance.GoVersion == "" || provenance.BuildKey == "" || provenance.TargetGOOS == "" || provenance.TargetGOARCH == "" || provenance.BuildInfo.GoVersion == "" || provenance.BuildInfo.Path == "" {
		return fmt.Errorf("exec provenance has an empty identity field")
	}
	if len(provenance.BuildKey) != sha256.Size*2 || !isLowerHex(provenance.BuildKey) {
		return fmt.Errorf("exec provenance build key is malformed")
	}
	if _, err := evidence.ParseSHA256(provenance.BinarySHA256); err != nil {
		return fmt.Errorf("exec provenance binary hash is malformed")
	}
	if err := validateDeterministicBuildInfo(provenance.BuildInfo); err != nil {
		return err
	}
	selection, err := validateCapabilityReviewStructure(provenance.CapabilityClosure)
	if err != nil {
		return fmt.Errorf("exec provenance capability closure: %w", err)
	}
	if provenance.SchemaVersion == 2 || provenance.CapabilityMode == CapabilityModeClosure {
		review := capabilityReviewFromClosure(provenance.CapabilityClosure, nil, selection)
		if len(review.Findings) != 0 {
			return fmt.Errorf("exec provenance capability closure: %w", unsupportedFinding(review.Findings[0]))
		}
	}
	return nil
}

func validateDeterministicBuildInfo(info evidence.BuildInfo) error {
	settings := make(map[string]string, len(info.Settings))
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	if settings["CGO_ENABLED"] != "0" {
		return fmt.Errorf("exec provenance requires CGO_ENABLED=0")
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

func ProjectBuildInfo(info *debug.BuildInfo) evidence.BuildInfo {
	settings := make([]evidence.BuildSetting, len(info.Settings))
	for index, setting := range info.Settings {
		settings[index] = evidence.BuildSetting{Key: setting.Key, Value: setting.Value}
	}
	sort.Slice(settings, func(i, j int) bool { return settings[i].Key < settings[j].Key })
	mainModule := info.Main.Path
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		mainModule += "@" + info.Main.Version
	}
	return evidence.BuildInfo{GoVersion: info.GoVersion, Path: info.Path, MainModule: mainModule, Settings: settings}
}

func hashRegularFile(path string) (string, uint64, error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		if errors.Is(err, hostfs.ErrSymbolicLink) {
			return "", 0, fmt.Errorf("%s is not a regular executable", path)
		}
		return "", 0, err
	}
	defer file.Close()
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

func copyRegularFile(source, destination string) error {
	input, info, err := hostfs.OpenPath(source)
	if err != nil {
		if errors.Is(err, hostfs.ErrSymbolicLink) {
			return fmt.Errorf("exec target is not a regular executable")
		}
		return fmt.Errorf("stat exec target: %w", err)
	}
	defer input.Close()
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("exec target is not a regular executable")
	}
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o700)
	if err != nil {
		return fmt.Errorf("create prepared exec target: %w", err)
	}
	if err := output.Chmod(0o700); err != nil {
		output.Close()
		return fmt.Errorf("set prepared exec target mode: %w", err)
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return fmt.Errorf("copy exec target: %w", err)
	}
	if err := output.Close(); err != nil {
		return fmt.Errorf("close prepared exec target: %w", err)
	}
	return nil
}

func readBoundedRegularFile(path string, maximum uint64) (_ []byte, retErr error) {
	file, info, err := hostfs.OpenPath(path)
	if err != nil {
		if errors.Is(err, hostfs.ErrSymbolicLink) {
			return nil, fmt.Errorf("%s is not a regular file", path)
		}
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, file.Close()) }()
	if info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, fmt.Errorf("%s exceeds its size bound", path)
	}
	data, err := io.ReadAll(io.LimitReader(file, int64(maximum)+1))
	if err != nil {
		return nil, err
	}
	if uint64(len(data)) > maximum {
		return nil, fmt.Errorf("%s exceeds its size bound", path)
	}
	return data, nil
}

func writePreparedFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
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
