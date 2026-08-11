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

	"go.temporal.io/server/tools/gomadv3/internal/record"
)

type Kind string

const (
	KindExec   Kind = "exec"
	KindGoRun  Kind = "go-run"
	KindGoTest Kind = "go-test"
)

const provenanceSchema = "gomadv3.exec-provenance/v1"

const maximumProvenanceBytes = 1 << 20

type Spec struct {
	Kind            Kind
	Source          string
	Provenance      string
	Args            []string
	BuildTags       []string
	WorkingDir      string
	PreparationRoot string
	ToolchainRoot   string
}

type ToolchainIdentity struct {
	GoVersion    string
	BuildKey     string
	TargetGOOS   string
	TargetGOARCH string
}

type Prepared struct {
	Path         string
	Kind         Kind
	Source       string
	SHA256       string
	Size         uint64
	Argv         []string
	BuildTags    []string
	BuildInfo    record.BuildInfo
	GoVersion    string
	BuildKey     string
	TargetGOOS   string
	TargetGOARCH string
}

type Provenance struct {
	SchemaVersion int
	GoVersion     string
	BuildKey      string
	TargetGOOS    string
	TargetGOARCH  string
	BinarySHA256  string
	BinarySize    uint64
	BuildInfo     record.BuildInfo
}

type provenanceWire struct {
	Schema        string              `json:"schema"`
	SchemaVersion int                 `json:"schema_version"`
	GoVersion     string              `json:"go_version"`
	BuildKey      string              `json:"build_key"`
	TargetGOOS    string              `json:"target_goos"`
	TargetGOARCH  string              `json:"target_goarch"`
	BinarySHA256  string              `json:"binary_sha256"`
	BinarySize    record.Uint64String `json:"binary_size"`
	BuildInfo     record.BuildInfo    `json:"build_info"`
}

func Prepare(ctx context.Context, spec Spec) (prepared Prepared, retErr error) {
	tags, err := normalizeBuildTags(spec.Kind, spec.BuildTags)
	if err != nil {
		return Prepared{}, err
	}
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
	buildInfo := record.BuildInfo{}
	switch spec.Kind {
	case KindExec:
		buildInfo, err = prepareExec(spec, identity, targetPath)
	case KindGoRun, KindGoTest:
		buildInfo, err = prepareGo(ctx, spec, tags, targetPath)
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
		buildInfo = projectBuildInfo(info)
	}
	prepared = Prepared{
		Path:         targetPath,
		Kind:         spec.Kind,
		Source:       spec.Source,
		SHA256:       hash,
		Size:         size,
		Argv:         append([]string{"gomadv3-target"}, spec.Args...),
		BuildTags:    tags,
		BuildInfo:    buildInfo,
		GoVersion:    identity.GoVersion,
		BuildKey:     identity.BuildKey,
		TargetGOOS:   identity.TargetGOOS,
		TargetGOARCH: identity.TargetGOARCH,
	}
	keep = true
	return prepared, nil
}

func (prepared Prepared) Verify() error {
	hash, size, err := hashRegularFile(prepared.Path)
	if err != nil {
		return fmt.Errorf("verify prepared target: %w", err)
	}
	if hash != prepared.SHA256 || size != prepared.Size {
		return fmt.Errorf("prepared target changed after preparation")
	}
	return nil
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
		return ToolchainIdentity{}, fmt.Errorf("stat pinned Go command: %w; run make -C tools/gomadv3 toolchain", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return ToolchainIdentity{}, fmt.Errorf("pinned Go command is not a regular executable")
	}
	buildKeyBytes, err := os.ReadFile(filepath.Join(root, "build-key"))
	if err != nil {
		return ToolchainIdentity{}, fmt.Errorf("read toolchain build key: %w; run make -C tools/gomadv3 toolchain", err)
	}
	buildKey := strings.TrimSuffix(string(buildKeyBytes), "\n")
	if len(buildKey) != sha256.Size*2 || !isLowerHex(buildKey) || string(buildKeyBytes) != buildKey+"\n" {
		return ToolchainIdentity{}, fmt.Errorf("toolchain build key is malformed")
	}
	builtGo := filepath.Join(root, "builds", buildKey, "bin", "go")
	if builtInfo, statErr := os.Stat(builtGo); statErr != nil || !builtInfo.Mode().IsRegular() || builtInfo.Mode()&0o111 == 0 {
		return ToolchainIdentity{}, fmt.Errorf("toolchain build %s is missing or stale; run make -C tools/gomadv3 toolchain", buildKey)
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

func WriteProvenance(path string, provenance Provenance) error {
	wire := provenanceWire{
		Schema:        provenanceSchema,
		SchemaVersion: provenance.SchemaVersion,
		GoVersion:     provenance.GoVersion,
		BuildKey:      provenance.BuildKey,
		TargetGOOS:    provenance.TargetGOOS,
		TargetGOARCH:  provenance.TargetGOARCH,
		BinarySHA256:  provenance.BinarySHA256,
		BinarySize:    record.Uint64String(provenance.BinarySize),
		BuildInfo:     provenance.BuildInfo,
	}
	if err := validateProvenance(wire); err != nil {
		return err
	}
	encoded, err := record.CanonicalJSON(wire)
	if err != nil {
		return fmt.Errorf("encode provenance: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("write provenance: %w", err)
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return fmt.Errorf("set provenance mode: %w", err)
	}
	if _, err := file.Write(encoded); err != nil {
		file.Close()
		return fmt.Errorf("write provenance: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close provenance: %w", err)
	}
	return nil
}

func prepareExec(spec Spec, identity ToolchainIdentity, targetPath string) (record.BuildInfo, error) {
	if spec.Source == "" || spec.Provenance == "" {
		return record.BuildInfo{}, fmt.Errorf("exec target and provenance are required")
	}
	provenanceBytes, err := readBoundedRegularFile(spec.Provenance, maximumProvenanceBytes)
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("read exec provenance: %w", err)
	}
	var provenance provenanceWire
	if err := record.StrictDecode(provenanceBytes, &provenance); err != nil {
		return record.BuildInfo{}, fmt.Errorf("decode exec provenance: %w", err)
	}
	canonical, err := record.CanonicalJSON(provenance)
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("canonicalize exec provenance: %w", err)
	}
	if string(canonical) != string(provenanceBytes) {
		return record.BuildInfo{}, fmt.Errorf("exec provenance is not canonical")
	}
	if err := validateProvenance(provenance); err != nil {
		return record.BuildInfo{}, err
	}
	if provenance.GoVersion != identity.GoVersion || provenance.BuildKey != identity.BuildKey || provenance.TargetGOOS != identity.TargetGOOS || provenance.TargetGOARCH != identity.TargetGOARCH {
		return record.BuildInfo{}, fmt.Errorf("exec provenance does not match pinned toolchain")
	}
	if err := copyRegularFile(spec.Source, targetPath); err != nil {
		return record.BuildInfo{}, err
	}
	hash, size, err := hashRegularFile(targetPath)
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("hash prepared provenance binary: %w", err)
	}
	if hash != provenance.BinarySHA256 || size != uint64(provenance.BinarySize) {
		return record.BuildInfo{}, fmt.Errorf("provenance binary identity does not match prepared target")
	}
	info, err := buildinfo.ReadFile(targetPath)
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("read prepared exec target build info: %w", err)
	}
	actualBuildInfo := projectBuildInfo(info)
	recordedBuildInfo, err := record.CanonicalJSON(provenance.BuildInfo)
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("encode provenance build info: %w", err)
	}
	actualBuildInfoBytes, err := record.CanonicalJSON(actualBuildInfo)
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("encode exec target build info: %w", err)
	}
	if actualBuildInfo.GoVersion != provenance.GoVersion || string(actualBuildInfoBytes) != string(recordedBuildInfo) {
		return record.BuildInfo{}, fmt.Errorf("exec target build info does not match provenance")
	}
	if err := writePreparedFile(filepath.Join(filepath.Dir(targetPath), "provenance.json"), provenanceBytes, 0o400); err != nil {
		return record.BuildInfo{}, fmt.Errorf("snapshot exec provenance: %w", err)
	}
	return provenance.BuildInfo, nil
}

func prepareGo(ctx context.Context, spec Spec, tags []string, targetPath string) (record.BuildInfo, error) {
	if spec.Source == "" || spec.WorkingDir == "" {
		return record.BuildInfo{}, fmt.Errorf("Go target source and working directory are required")
	}
	if strings.HasPrefix(spec.Source, "-") || strings.Contains(spec.Source, "...") || strings.IndexFunc(spec.Source, unicode.IsSpace) >= 0 || strings.IndexByte(spec.Source, 0) >= 0 {
		return record.BuildInfo{}, fmt.Errorf("Go target package argument %q must select exactly one package", spec.Source)
	}
	goCommand, err := filepath.Abs(filepath.Join(spec.ToolchainRoot, "bin", "go"))
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("resolve pinned Go command: %w", err)
	}
	arguments := []string{}
	if spec.Kind == KindGoRun {
		arguments = append(arguments, "build")
	} else {
		arguments = append(arguments, "test", "-c")
	}
	arguments = append(arguments, "-trimpath", "-o", targetPath)
	if len(tags) > 0 {
		arguments = append(arguments, "-tags", strings.Join(tags, ","))
	}
	commandDirectory, packageArgument, err := resolveBuildContext(spec.WorkingDir, spec.Source)
	if err != nil {
		return record.BuildInfo{}, err
	}
	arguments = append(arguments, packageArgument)
	command := exec.CommandContext(ctx, goCommand, arguments...)
	command.Dir = commandDirectory
	command.Env = preparationEnvironment()
	output, err := command.CombinedOutput()
	if err != nil {
		return record.BuildInfo{}, fmt.Errorf("prepare %s target: %w: %s", spec.Kind, err, output)
	}
	return record.BuildInfo{}, nil
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

func normalizeBuildTags(kind Kind, supplied []string) ([]string, error) {
	set := make(map[string]struct{}, len(supplied)+1)
	for _, tag := range supplied {
		if tag == "" || tag == "race" || strings.Contains(tag, ",") || strings.IndexFunc(tag, unicode.IsSpace) >= 0 {
			return nil, fmt.Errorf("unsupported build tag %q", tag)
		}
		set[tag] = struct{}{}
	}
	if kind == KindGoTest {
		set["test_dep"] = struct{}{}
	}
	tags := make([]string, 0, len(set))
	for tag := range set {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	return tags, nil
}

func preparationEnvironment() []string {
	reserved := map[string]struct{}{
		"CGO_ENABLED": {}, "GOMADSEED": {}, "GOMADV3_CHILD_SEED": {},
		"GOENV": {}, "GOEXPERIMENT": {}, "GOFLAGS": {}, "GOTOOLCHAIN": {}, "GOWORK": {}, "TZ": {},
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
	if provenance.Schema != provenanceSchema || provenance.SchemaVersion != 1 {
		return fmt.Errorf("unsupported exec provenance schema")
	}
	if provenance.GoVersion == "" || provenance.BuildKey == "" || provenance.TargetGOOS == "" || provenance.TargetGOARCH == "" || provenance.BuildInfo.GoVersion == "" || provenance.BuildInfo.Path == "" {
		return fmt.Errorf("exec provenance has an empty identity field")
	}
	if len(provenance.BuildKey) != sha256.Size*2 || !isLowerHex(provenance.BuildKey) {
		return fmt.Errorf("exec provenance build key is malformed")
	}
	if !strings.HasPrefix(provenance.BinarySHA256, "sha256:") || len(provenance.BinarySHA256) != len("sha256:")+sha256.Size*2 || !isLowerHex(strings.TrimPrefix(provenance.BinarySHA256, "sha256:")) {
		return fmt.Errorf("exec provenance binary hash is malformed")
	}
	if err := validateDeterministicBuildInfo(provenance.BuildInfo); err != nil {
		return err
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

func projectBuildInfo(info *debug.BuildInfo) record.BuildInfo {
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

func hashRegularFile(path string) (string, uint64, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return "", 0, err
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", 0, fmt.Errorf("%s is not a regular executable", path)
	}
	if err := validateLinkCount(info); err != nil {
		return "", 0, err
	}
	file, err := openNoFollow(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
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
	info, err := os.Lstat(source)
	if err != nil {
		return fmt.Errorf("stat exec target: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return fmt.Errorf("exec target is not a regular executable")
	}
	if err := validateLinkCount(info); err != nil {
		return err
	}
	input, err := openNoFollow(source)
	if err != nil {
		return fmt.Errorf("open exec target: %w", err)
	}
	defer input.Close()
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

func readBoundedRegularFile(path string, maximum uint64) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	if err := validateLinkCount(info); err != nil {
		return nil, err
	}
	if info.Size() < 0 || uint64(info.Size()) > maximum {
		return nil, fmt.Errorf("%s exceeds its size bound", path)
	}
	file, err := openNoFollow(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
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
