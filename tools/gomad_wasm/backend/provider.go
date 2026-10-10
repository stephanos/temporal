package backend

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad_wasm/toolchain"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

const Name = "wasm"
const provenanceSchema = "gomad-wasm.provenance/v1"
const maximumProvenanceBytes = 16 << 20

type Options struct {
	CompilerPath, CompilerSHA256, HelperPath, HelperSHA256, ModelSHA256, CacheRoot string
	GoVersion                                                                      string
	RuntimeRoot                                                                    string
	Engine                                                                         wasi.EngineIdentity
	MemoryBytes, Fuel                                                              uint64
	Config                                                                         wasi.Config
}
type Provider struct{ options Options }
type sourceIdentity struct {
	Path   string
	SHA256 record.SHA256
}
type provenance struct {
	BuildCachePolicy                                                                                          string
	BuildFlags, BuildEnvironment                                                                              []string
	Schema, Profile, ReplayMode, CompilerSHA256, GoVersion, HelperSHA256, ModelSHA256, ModuleSHA256, BuildKey string
	Engine                                                                                                    wasi.EngineIdentity
	Sources                                                                                                   []sourceIdentity
	Kind                                                                                                      target.Kind
	Source                                                                                                    string
	Args, BuildTags                                                                                           []string
	Imports                                                                                                   []wasi.Import
	InitialMemoryPages, MemoryBytes, Fuel, ModuleBytes                                                        uint64
	Config                                                                                                    wasi.Config
	RuntimeSHA256                                                                                             string `json:",omitempty"`
	RuntimeImplementationSHA256                                                                               string `json:",omitempty"`
}

func New(options Options) (*Provider, error) {
	stock := options.Config.Profile == wasi.StockProfile && (options.GoVersion == "go1.27.1" || options.GoVersion == "go1.27.0")
	cooperative := options.Config.Profile == wasi.CooperativeProfile && options.GoVersion == "go1.27.1"
	if (!stock && !cooperative) || options.Engine != wasi.PinnedEngine() || options.MemoryBytes == 0 || options.Fuel == 0 || options.ModelSHA256 != wasi.ImplementationSHA256() {
		return nil, errors.New("explicit pinned WASM profile is required")
	}
	for _, digest := range []string{options.CompilerSHA256, options.HelperSHA256, options.ModelSHA256} {
		if _, err := record.ParseSHA256(digest); err != nil {
			return nil, err
		}
	}
	encoded, err := json.Marshal(options.Config)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(encoded, &options.Config); err != nil {
		return nil, err
	}
	if _, err := wasi.NewEnvironment(options.Config); err != nil {
		return nil, err
	}
	return &Provider{options: options}, nil
}
func (p *Provider) Prepare(ctx context.Context, spec target.Spec) (preparedResult target.Prepared, retErr error) {
	if spec.Backend != Name || spec.Kind != target.KindGoRun && spec.Kind != target.KindGoTest || spec.Source == "" || spec.BuildOverlay != "" || spec.BuildModFile != "" || len(spec.AdapterReplacements) != 0 || spec.Provenance != "" || spec.CapabilityMode != "" && spec.CapabilityMode != target.CapabilityModeClosure {
		return target.Prepared{}, errors.New("unsupported WASM source preparation request")
	}
	if err := checkFile(p.options.CompilerPath, p.options.CompilerSHA256, 512<<20); err != nil {
		return target.Prepared{}, err
	}
	if err := checkFile(p.options.HelperPath, p.options.HelperSHA256, 512<<20); err != nil {
		return target.Prepared{}, err
	}
	dir := spec.WorkingDir
	if dir == "" {
		dir = "."
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return target.Prepared{}, err
	}
	dir = absolute
	environment := buildEnvironment()
	version, err := command(ctx, p.options.CompilerPath, []string{"version"}, dir, environment, 64<<10)
	if err != nil {
		return target.Prepared{}, err
	}
	if !strings.Contains(string(version), p.options.GoVersion+" ") {
		return target.Prepared{}, errors.New("compiler version does not match declared profile")
	}
	tags := append([]string{}, spec.BuildTags...)
	sort.Strings(tags)
	compilerSources, includeDirectory, err := compilerInventory(ctx, p.options.CompilerPath, dir, environment)
	if err != nil {
		return target.Prepared{}, err
	}
	overlayPath, runtimeIdentity := "", ""
	var replacements map[string]string
	if p.options.Config.Profile == wasi.CooperativeProfile {
		overlayPath, runtimeIdentity, replacements, err = p.runtimeOverlay(ctx, dir, environment, includeDirectory)
		if err != nil {
			return target.Prepared{}, err
		}
	}
	listArgs := []string{"list", "-mod=readonly", "-pgo=off", "-deps", "-json"}
	if overlayPath != "" {
		listArgs = append(listArgs, "-overlay", overlayPath)
	}
	if spec.Kind == target.KindGoTest {
		listArgs = append(listArgs, "-test")
	}
	if len(tags) != 0 {
		listArgs = append(listArgs, "-tags", strings.Join(tags, ","))
	}
	listArgs = append(listArgs, spec.Source)
	listed, err := command(ctx, p.options.CompilerPath, listArgs, dir, environment, 128<<20)
	if err != nil {
		return target.Prepared{}, err
	}
	sources, err := sourceInventory(listed, includeDirectory, replacements)
	if err != nil {
		return target.Prepared{}, err
	}
	sources = append(sources, compilerSources...)
	if overlayPath != "" {
		inputs, err := p.runtimeInputs(overlayPath)
		if err != nil {
			return target.Prepared{}, err
		}
		sources = append(sources, inputs...)
	}
	sort.Slice(sources, func(i, j int) bool { return sources[i].Path < sources[j].Path })
	pr := provenance{BuildCachePolicy: "private-build-key/v1", Schema: provenanceSchema, Profile: wasi.StockProfile, ReplayMode: record.ReplayObserved, CompilerSHA256: p.options.CompilerSHA256, GoVersion: p.options.GoVersion, HelperSHA256: p.options.HelperSHA256, ModelSHA256: p.options.ModelSHA256, Engine: p.options.Engine, Sources: sources, Kind: spec.Kind, Source: spec.Source, Args: append([]string{}, spec.Args...), BuildTags: tags, MemoryBytes: p.options.MemoryBytes, Fuel: p.options.Fuel, Config: p.options.Config, BuildFlags: []string{"-mod=readonly", "-trimpath", "-buildvcs=false", "-pgo=off"}, BuildEnvironment: []string{"GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOEXPERIMENT=nogreenteagc", "GOCACHEPROG="}}
	if overlayPath != "" {
		pr.Profile, pr.ReplayMode = wasi.CooperativeProfile, record.ReplayExact
		pr.RuntimeSHA256 = "sha256:" + runtimeIdentity
		pr.RuntimeImplementationSHA256 = toolchain.ImplementationSHA256()
		pr.BuildFlags = append(pr.BuildFlags, "-overlay", overlayPath)
	}
	pr.BuildKey, err = buildKey(pr)
	if err != nil {
		return target.Prepared{}, err
	}
	if !filepath.IsAbs(p.options.CacheRoot) || !filepath.IsAbs(spec.PreparationRoot) {
		return target.Prepared{}, errors.New("absolute WASM cache and preparation roots are required")
	}
	cache := filepath.Join(p.options.CacheRoot, pr.BuildKey)
	if err := os.MkdirAll(cache, 0700); err != nil {
		return target.Prepared{}, err
	}
	modulePath := filepath.Join(cache, "module.wasm")
	receiptPath := filepath.Join(cache, "receipt.json")
	module, readErr := hostfs.ReadBounded(modulePath, 512<<20)
	if readErr == nil {
		receipt, err := hostfs.ReadBounded(receiptPath, maximumProvenanceBytes)
		if err != nil {
			return target.Prepared{}, err
		}
		cached, err := decodeProvenance(receipt)
		if err != nil || cached.BuildKey != pr.BuildKey || cached.ModuleSHA256 != string(record.HashBytes(module)) {
			return target.Prepared{}, errors.New("WASM compilation cache identity changed")
		}
		if err := p.validateProvenance(cached, preparedFrom(cached, receipt)); err != nil {
			return target.Prepared{}, err
		}
		imports, pages, err := inventory(module)
		if err != nil || !reflect.DeepEqual(imports, cached.Imports) || pages != cached.InitialMemoryPages {
			return target.Prepared{}, errors.New("cached WASM import or memory identity changed")
		}
		pr = cached
	} else if errors.Is(readErr, os.ErrNotExist) {
		temporary, err := os.MkdirTemp(cache, ".build-")
		if err != nil {
			return target.Prepared{}, err
		}
		defer func() { retErr = errors.Join(retErr, os.RemoveAll(temporary)) }()
		output := filepath.Join(temporary, "module.wasm")
		args := []string{"build", "-mod=readonly", "-trimpath", "-buildvcs=false", "-pgo=off", "-o", output}
		if spec.Kind == target.KindGoTest {
			args = []string{"test", "-c", "-mod=readonly", "-trimpath", "-buildvcs=false", "-pgo=off", "-o", output}
		}
		if overlayPath != "" {
			args = append(args, "-overlay", overlayPath)
		}
		if len(tags) != 0 {
			args = append(args, "-tags", strings.Join(tags, ","))
		}
		args = append(args, spec.Source)
		buildEnvironment := append(append([]string(nil), environment...), "GOCACHE="+filepath.Join(cache, "objects"))
		if _, err := command(ctx, p.options.CompilerPath, args, dir, buildEnvironment, 4<<20); err != nil {
			return target.Prepared{}, err
		}
		module, err = hostfs.ReadBounded(output, 512<<20)
		if err != nil {
			return target.Prepared{}, err
		}
		pr.ModuleSHA256 = string(record.HashBytes(module))
		pr.ModuleBytes = uint64(len(module))
		pr.Imports, pr.InitialMemoryPages, err = inventory(module)
		if err != nil {
			return target.Prepared{}, err
		}
		afterList, err := command(ctx, p.options.CompilerPath, listArgs, dir, environment, 128<<20)
		if err != nil {
			return target.Prepared{}, err
		}
		afterSources, err := sourceInventory(afterList, includeDirectory, replacements)
		if err != nil {
			return target.Prepared{}, err
		}
		afterCompiler, afterIncludeDirectory, err := compilerInventory(ctx, p.options.CompilerPath, dir, environment)
		if err != nil {
			return target.Prepared{}, err
		}
		afterSources = append(afterSources, afterCompiler...)
		if overlayPath != "" {
			afterOverlay, afterIdentity, _, err := p.runtimeOverlay(ctx, dir, environment, afterIncludeDirectory)
			if err != nil {
				return target.Prepared{}, err
			}
			if afterOverlay != overlayPath || afterIdentity != runtimeIdentity {
				return target.Prepared{}, errors.New("WASM runtime overlay changed during compilation")
			}
			inputs, err := p.runtimeInputs(afterOverlay)
			if err != nil {
				return target.Prepared{}, err
			}
			afterSources = append(afterSources, inputs...)
		}
		sort.Slice(afterSources, func(i, j int) bool { return afterSources[i].Path < afterSources[j].Path })
		if afterIncludeDirectory != includeDirectory || !reflect.DeepEqual(sources, afterSources) {
			return target.Prepared{}, errors.New("selected WASM source inventory changed during compilation")
		}
		if err := verifySources(sources); err != nil {
			return target.Prepared{}, err
		}
		if err := checkFile(p.options.CompilerPath, p.options.CompilerSHA256, 512<<20); err != nil {
			return target.Prepared{}, err
		}
		receipt, err := json.Marshal(pr)
		if err != nil {
			return target.Prepared{}, err
		}
		if err := os.WriteFile(filepath.Join(temporary, "receipt.json"), receipt, 0600); err != nil {
			return target.Prepared{}, err
		}
		if err := os.Rename(output, modulePath); err != nil {
			return target.Prepared{}, err
		}
		if err := os.Rename(filepath.Join(temporary, "receipt.json"), receiptPath); err != nil {
			return target.Prepared{}, err
		}
	} else {
		return target.Prepared{}, readErr
	}
	if err := os.MkdirAll(spec.PreparationRoot, 0700); err != nil {
		return target.Prepared{}, err
	}
	data, err := json.Marshal(pr)
	if err != nil {
		return target.Prepared{}, err
	}
	prepared := preparedFrom(pr, data)
	prepared.Path = filepath.Join(spec.PreparationRoot, "module.wasm")
	if err := os.WriteFile(prepared.Path, module, 0500); err != nil {
		return target.Prepared{}, err
	}
	for _, payload := range prepared.BackendPayloads {
		path := filepath.Join(spec.PreparationRoot, filepath.FromSlash(payload.Reference.File))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return target.Prepared{}, err
		}
		if err := os.WriteFile(path, payload.Data, 0600); err != nil {
			return target.Prepared{}, err
		}
	}
	return prepared, nil
}
func preparedFrom(pr provenance, data []byte) target.Prepared {
	reference := record.BackendPayload{Schema: provenanceSchema, File: "backend/provenance.json", SHA256: record.HashBytes(data), Bytes: record.Uint64String(len(data))}
	return target.Prepared{Kind: pr.Kind, Source: pr.Source, SHA256: pr.ModuleSHA256, Size: pr.ModuleBytes, Argv: append([]string{"gomad3-target"}, pr.Args...), BuildTags: pr.BuildTags, Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, GoVersion: pr.GoVersion, BuildKey: pr.BuildKey, TargetGOOS: "wasip1", TargetGOARCH: "wasm", CapabilityMode: target.CapabilityModeClosure, BuildInfo: record.BuildInfo{GoVersion: pr.GoVersion, Path: pr.Source, Settings: []record.BuildSetting{}}, Backend: &record.BackendMetadata{Name: Name, ReplayMode: pr.ReplayMode, Provenance: reference}, BackendPayloads: []target.BackendPayload{{Reference: reference, Data: append([]byte(nil), data...)}}}
}
func (p *Provider) ValidatePrepared(spec target.Spec, prepared target.Prepared, _ []string) error {
	if prepared.Backend == nil || prepared.Backend.Name != Name || spec.Backend != Name || prepared.Kind != spec.Kind || prepared.Source != spec.Source || !reflect.DeepEqual(prepared.Argv, append([]string{"gomad3-target"}, spec.Args...)) {
		return errors.New("WASM prepared specification mismatch")
	}
	if err := prepared.ValidateBackendPayloads(); err != nil {
		return err
	}
	pr, err := decodeProvenance(prepared.BackendPayloads[0].Data)
	if err != nil {
		return err
	}
	if err := p.validateProvenance(pr, prepared); err != nil {
		return err
	}
	return prepared.Verify()
}
func (p *Provider) ValidateReplay(_ context.Context, opened *artifact.Opened) (target.Prepared, error) {
	manifest := opened.Manifest()
	if manifest.Target.Backend == nil || manifest.Target.Backend.Name != Name || manifest.Target.Backend.Evidence == nil || manifest.SimulationProfile != nil || manifest.IOProfile.Transcript != nil || manifest.IOProfile.ReadOnlyMounts != nil {
		return target.Prepared{}, errors.New("unsupported or incomplete WASM artifact")
	}
	if p.options.Config.Profile == wasi.StockProfile && (manifest.ReplayMode != record.ReplayObserved || manifest.ChoiceProfile != nil) || p.options.Config.Profile == wasi.CooperativeProfile && (manifest.ReplayMode != record.ReplayExact || manifest.ChoiceProfile == nil) {
		return target.Prepared{}, errors.New("WASM artifact replay profile mismatch")
	}
	reference := manifest.Target.Backend.Provenance
	data, err := opened.ReadPayload(reference.File, maximumProvenanceBytes)
	if err != nil {
		return target.Prepared{}, err
	}
	pr, err := decodeProvenance(data)
	if err != nil {
		return target.Prepared{}, err
	}
	prepared := preparedFrom(pr, data)
	prepared.Size = uint64(manifest.Target.Size)
	if pr.ModuleBytes != uint64(manifest.Target.Size) || pr.ModuleSHA256 != string(manifest.Target.SHA256) || !reflect.DeepEqual(pr.BuildTags, manifest.Target.BuildTags) || len(manifest.Target.Adapters) != 0 || len(manifest.Target.Compatibility) != 0 || manifest.Target.CapabilityMode != "closure" || !reflect.DeepEqual(prepared.RecordToolchain(), manifest.Toolchain) || !reflect.DeepEqual(prepared.Argv, manifest.Target.Argv) || prepared.Kind != target.Kind(manifest.Target.Kind) || prepared.Source != manifest.Target.Source || !reflect.DeepEqual(prepared.RecordTarget().BuildInfo, manifest.Target.BuildInfo) || !reflect.DeepEqual(record.BackendIOProfile(*prepared.Backend), manifest.IOProfile) {
		return target.Prepared{}, errors.New("WASM artifact provenance does not match retained record")
	}
	if err := p.validateProvenance(pr, prepared); err != nil {
		return target.Prepared{}, err
	}
	if profile := manifest.ChoiceProfile; profile != nil {
		implementation, err := choice.ImplementationIdentity(pr.BuildKey)
		if err != nil || profile.Name != choice.Profile || string(profile.ImplementationSHA256) != fmt.Sprintf("sha256:%x", implementation) || profile.Trace.TapeSHA256 == "" || profile.Trace.TerminalState != "complete" {
			return target.Prepared{}, errors.New("WASM artifact exact choice identity mismatch")
		}
	}
	module, err := opened.ReadPayload(manifest.Target.File, uint64(manifest.Target.Size))
	if err != nil {
		return target.Prepared{}, err
	}
	imports, pages, err := inventory(module)
	if err != nil || !reflect.DeepEqual(imports, pr.Imports) || pages != pr.InitialMemoryPages {
		return target.Prepared{}, errors.New("retained WASM import or memory identity mismatch")
	}
	if err := checkFile(p.options.HelperPath, p.options.HelperSHA256, 512<<20); err != nil {
		return target.Prepared{}, err
	}
	return prepared, nil
}
func (p *Provider) validateProvenance(pr provenance, prepared target.Prepared) error {
	replayMode := record.ReplayObserved
	if p.options.Config.Profile == wasi.CooperativeProfile {
		replayMode = record.ReplayExact
		if _, err := record.ParseSHA256(pr.RuntimeSHA256); err != nil || pr.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() {
			return errors.New("WASM runtime implementation identity mismatch")
		}
	} else if pr.RuntimeSHA256 != "" || pr.RuntimeImplementationSHA256 != "" {
		return errors.New("stock WASM provenance has a cooperative runtime")
	}
	if pr.BuildCachePolicy != "private-build-key/v1" || pr.Schema != provenanceSchema || pr.Profile != p.options.Config.Profile || pr.ReplayMode != replayMode || prepared.Backend == nil || prepared.Backend.ReplayMode != replayMode || pr.CompilerSHA256 != p.options.CompilerSHA256 || pr.GoVersion != p.options.GoVersion || pr.HelperSHA256 != p.options.HelperSHA256 || pr.ModelSHA256 != p.options.ModelSHA256 || pr.Engine != p.options.Engine || pr.MemoryBytes != p.options.MemoryBytes || pr.Fuel != p.options.Fuel || pr.ModuleSHA256 != prepared.SHA256 || pr.BuildKey != prepared.BuildKey || prepared.TargetGOOS != "wasip1" || prepared.TargetGOARCH != "wasm" {
		return errors.New("WASM compiler/helper/engine/model/limits identity mismatch")
	}
	if !reflect.DeepEqual(pr.Config, p.options.Config) {
		return errors.New("WASM immutable configuration identity mismatch")
	}
	key, err := buildKey(pr)
	if err != nil || key != pr.BuildKey {
		return errors.New("WASM source build identity mismatch")
	}
	if pr.ModuleBytes != prepared.Size || !reflect.DeepEqual(pr.Args, prepared.Argv[1:]) || !reflect.DeepEqual(pr.BuildTags, prepared.BuildTags) || pr.Kind != prepared.Kind || pr.Source != prepared.Source {
		return errors.New("WASM prepared source/argv/tag/size identity mismatch")
	}
	if _, err := wasi.NewEnvironment(pr.Config); err != nil {
		return err
	}
	return nil
}
func (p *Provider) Run(ctx context.Context, request runnerbackend.Request) (runnerbackend.Result, error) {
	if request.Target.Backend == nil || request.Target.Backend.Name != Name {
		return runnerbackend.Result{}, errors.New("WASM target metadata is required")
	}
	if err := request.Target.ValidateBackendPayloads(); err != nil {
		return runnerbackend.Result{}, err
	}
	pr, err := decodeProvenance(request.Target.BackendPayloads[0].Data)
	if err != nil {
		return runnerbackend.Result{}, err
	}
	if err := p.validateProvenance(pr, request.Target); err != nil {
		return runnerbackend.Result{}, err
	}
	config := pr.Config
	config.Args = append([]string{}, request.Target.Argv...)
	config.Environment = append([]string{}, request.Environment...)
	config.Environment = append(config.Environment, "PWD="+config.WorkingDirectory)
	var runtimeControl *wasi.RuntimeControl
	if pr.Profile == wasi.CooperativeProfile {
		if request.Choice == nil {
			return runnerbackend.Result{}, errors.New("cooperative WASM execution requires choice recording or replay")
		}
		implementation, err := choice.ImplementationIdentity(pr.BuildKey)
		if err != nil {
			return runnerbackend.Result{}, err
		}
		identity := choice.ExecutionIdentity{ToolchainBuildKey: pr.BuildKey, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: implementation}
		moduleIdentity, err := hex.DecodeString(strings.TrimPrefix(pr.ModuleSHA256, "sha256:"))
		if err != nil || len(moduleIdentity) != sha256.Size {
			return runnerbackend.Result{}, errors.New("cooperative WASM module identity is invalid")
		}
		copy(identity.TargetSHA256[:], moduleIdentity)
		if request.Choice.ExecutionIdentity != identity {
			return runnerbackend.Result{}, errors.New("cooperative WASM choice execution identity mismatch")
		}
		runtimeControl = &wasi.RuntimeControl{Seed: request.Seed, Choice: request.Choice, Diagnostics: request.Diagnostics}
	} else {
		if request.Choice != nil || request.Diagnostics {
			return runnerbackend.Result{}, errors.New("stock WASM execution does not support runtime choices or diagnostics")
		}
		config.EntropyKey = sha256.Sum256([]byte(fmt.Sprintf("%s:%d", pr.BuildKey, request.Seed)))
	}
	config.Limits.OutputBytes = request.OutputBytes
	config.Limits.TranscriptBytes = request.TranscriptBytes
	if !filepath.IsAbs(request.Target.Path) || !filepath.IsAbs(p.options.HelperPath) || len(pr.ModuleSHA256) != 71 || len(pr.HelperSHA256) != 71 {
		return runnerbackend.Result{}, fmt.Errorf("invalid WASM launch paths or identities: target=%q helper=%q module-digest-bytes=%d helper-digest-bytes=%d", request.Target.Path, p.options.HelperPath, len(pr.ModuleSHA256), len(pr.HelperSHA256))
	}
	result, err := wasi.Run(ctx, wasi.Request{HelperPath: p.options.HelperPath, HelperSHA256: strings.TrimPrefix(pr.HelperSHA256, "sha256:"), ModulePath: request.Target.Path, ModuleSHA256: strings.TrimPrefix(pr.ModuleSHA256, "sha256:"), Engine: pr.Engine, Imports: pr.Imports, InitialMemoryPages: pr.InitialMemoryPages, MemoryBytes: pr.MemoryBytes, Fuel: pr.Fuel, Config: config, Timeout: request.Timeout, Runtime: runtimeControl})
	if err != nil {
		return runnerbackend.Result{}, err
	}
	observed := runnerbackend.Result{Termination: runnerbackend.Termination(result.Termination), Stdout: result.Stdout, Stderr: result.Stderr, Evidence: result.Transcript, Reaped: result.Reaped, Cancelled: result.Cancelled, WatchdogTimeout: result.WatchdogTimeout, ChoiceTrace: result.ChoiceTrace, DiagnosticTrace: result.DiagnosticTrace, ChoiceDivergence: result.ChoiceDivergence}
	if request.Choice != nil {
		observed.ImplementationSHA256 = request.Choice.ExecutionIdentity.ImplementationSHA256
	}
	if result.ExitCode != nil {
		observed.ExitCode = int(*result.ExitCode)
	}
	return observed, nil
}
func decodeProvenance(data []byte) (provenance, error) {
	var pr provenance
	if len(data) > maximumProvenanceBytes {
		return pr, errors.New("WASM provenance exceeds bound")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&pr); err != nil {
		return pr, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return pr, errors.New("trailing WASM provenance")
	}
	canonical, err := json.Marshal(pr)
	if err != nil || !bytes.Equal(data, canonical) {
		return pr, errors.New("noncanonical WASM provenance")
	}
	return pr, nil
}
func checkFile(name, digest string, limit uint64) error {
	data, err := hostfs.ReadBounded(name, limit)
	if err != nil {
		return err
	}
	if string(record.HashBytes(data)) != digest {
		return errors.New("pinned WASM infrastructure bytes changed")
	}
	return nil
}
func command(ctx context.Context, compiler string, args []string, dir string, env []string, limit uint64) ([]byte, error) {
	result, err := hostexec.Run(ctx, hostexec.Request{Command: append([]string{compiler}, args...), Dir: dir, Env: env, Timeout: 10 * time.Minute, TerminateGrace: time.Second, OutputLimit: limit})
	if err != nil {
		return nil, err
	}
	if !result.GroupGone || result.Cancelled || result.WatchdogTimeout || result.Termination != hostexec.TerminationExit || result.ExitCode != 0 {
		return nil, fmt.Errorf("WASM compiler command failed: %s", result.Stderr)
	}
	return result.Stdout, nil
}
func buildEnvironment() []string {
	var env []string
	for _, value := range os.Environ() {
		name, _, _ := strings.Cut(value, "=")
		switch name {
		case "GOROOT", "GOENV", "GOFLAGS", "GOTOOLCHAIN", "GOWORK", "GOOS", "GOARCH", "CGO_ENABLED", "GOEXPERIMENT", "GOMADSEED", "GOMAD3_CHILD_SEED", "GOCACHE", "GOCACHEPROG":
			continue
		}
		env = append(env, value)
	}
	return append(env, "GOENV=off", "GOFLAGS=", "GOTOOLCHAIN=local", "GOWORK=off", "GOOS=wasip1", "GOARCH=wasm", "CGO_ENABLED=0", "GOEXPERIMENT=nogreenteagc", "GOCACHEPROG=")
}
func sourceInventory(data []byte, includeDirectory string, overlays ...map[string]string) ([]sourceIdentity, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	files := map[string]bool{}
	replacements := map[string]string{}
	for _, overlay := range overlays {
		for original, effective := range overlay {
			replacements[original] = effective
		}
	}
	for {
		var pkg struct {
			Dir                                                                                                        string
			GoFiles, EmbedFiles, TestGoFiles, XTestGoFiles, TestEmbedFiles, XTestEmbedFiles, SFiles, SysoFiles, HFiles []string
			Standard                                                                                                   bool
			Module                                                                                                     *struct {
				GoMod   string
				Replace *struct{ Path string }
			}
		}
		err := decoder.Decode(&pkg)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if pkg.Module != nil {
			if pkg.Module.Replace != nil {
				return nil, errors.New("ambient module replacement is unsupported")
			}
			if pkg.Module.GoMod != "" {
				files[pkg.Module.GoMod] = true
				if _, err := os.Stat(filepath.Join(filepath.Dir(pkg.Module.GoMod), "go.sum")); err == nil {
					files[filepath.Join(filepath.Dir(pkg.Module.GoMod), "go.sum")] = true
				} else if !errors.Is(err, os.ErrNotExist) {
					return nil, err
				}
			}
		}
		for _, selected := range [][]string{pkg.GoFiles, pkg.EmbedFiles, pkg.TestGoFiles, pkg.XTestGoFiles, pkg.TestEmbedFiles, pkg.XTestEmbedFiles, pkg.SFiles, pkg.SysoFiles, pkg.HFiles} {
			for _, file := range selected {
				if !filepath.IsAbs(file) {
					file = filepath.Join(pkg.Dir, file)
				}
				if effective, replaced := replacements[file]; replaced {
					if effective == "" {
						return nil, errors.New("runtime overlay removed a selected build input")
					}
					files[effective] = true
					if _, err := os.Stat(file); errors.Is(err, os.ErrNotExist) {
						continue
					} else if err != nil {
						return nil, err
					}
				}
				files[file] = true
			}
		}
		for _, file := range pkg.SFiles {
			if !filepath.IsAbs(file) {
				file = filepath.Join(pkg.Dir, file)
			}
			if err := assemblyIncludes(file, pkg.Dir, filepath.Dir(file), includeDirectory, files, map[string]bool{}); err != nil {
				return nil, err
			}
		}
	}
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	sources := make([]sourceIdentity, 0, len(paths))
	for _, path := range paths {
		data, err := hostfs.ReadBounded(path, 512<<20)
		if err != nil {
			return nil, err
		}
		sources = append(sources, sourceIdentity{path, record.HashBytes(data)})
	}
	return sources, nil
}
func verifySources(sources []sourceIdentity) error {
	for _, source := range sources {
		if err := checkFile(source.Path, string(source.SHA256), 512<<20); err != nil {
			return fmt.Errorf("WASM build source changed: %w", err)
		}
	}
	return nil
}

func buildKey(pr provenance) (string, error) {
	pr.BuildKey = ""
	pr.ModuleSHA256 = ""
	pr.ModuleBytes = 0
	pr.Imports = nil
	pr.InitialMemoryPages = 0
	data, err := json.Marshal(pr)
	if err != nil {
		return "", err
	}
	return strings.TrimPrefix(string(record.HashBytes(data)), "sha256:"), nil
}

func compilerInventory(ctx context.Context, compiler, dir string, environment []string) ([]sourceIdentity, string, error) {
	data, err := command(ctx, compiler, []string{"env", "-json", "GOTOOLDIR", "GOROOT"}, dir, environment, 64<<10)
	if err != nil {
		return nil, "", err
	}
	var identity struct{ GOTOOLDIR, GOROOT string }
	if err := json.Unmarshal(data, &identity); err != nil {
		return nil, "", err
	}
	if !filepath.IsAbs(identity.GOTOOLDIR) || !filepath.IsAbs(identity.GOROOT) {
		return nil, "", errors.New("absolute compiler tool directory is required")
	}
	sources := []sourceIdentity{}
	for _, name := range []string{"asm", "compile", "link"} {
		path := filepath.Join(identity.GOTOOLDIR, name)
		data, err := hostfs.ReadBounded(path, 512<<20)
		if err != nil {
			return nil, "", err
		}
		sources = append(sources, sourceIdentity{Path: path, SHA256: record.HashBytes(data)})
	}
	includeDirectory := filepath.Join(identity.GOROOT, "pkg", "include")
	headers, err := filepath.Glob(filepath.Join(includeDirectory, "*.h"))
	if err != nil {
		return nil, "", err
	}
	if len(headers) == 0 {
		return nil, "", errors.New("compiler assembler headers are missing")
	}
	for _, path := range headers {
		data, err := hostfs.ReadBounded(path, 512<<20)
		if err != nil {
			return nil, "", err
		}
		sources = append(sources, sourceIdentity{Path: path, SHA256: record.HashBytes(data)})
	}
	return sources, includeDirectory, nil
}
