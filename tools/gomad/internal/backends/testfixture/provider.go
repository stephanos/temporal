//go:build !gomad_native

package testfixture

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	transcript "go.temporal.io/server/tools/gomad/internal/evidence/record/testfixture"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

type Config struct {
	ToolchainRoot, RunnerBuild                              string
	SupervisorCommand, CoordinatorCommand, BootstrapCommand []string
}

type Provider struct{ config Config }

func New(config Config) *Provider { return &Provider{config: config} }

func (*Provider) Capabilities() backends.Capabilities {
	return backends.Capabilities{Name: "native", ReplayModes: []string{record.ReplayExact, record.ReplayDiagnostic}, ChoiceProfiles: []string{choice.Profile}, Simulation: true, ReadOnlyMounts: true, IOTranscript: true}
}

func (provider *Provider) Describe(ctx context.Context, config backends.Configuration) (backends.Description, error) {
	if err := ctx.Err(); err != nil {
		return backends.Description{}, err
	}
	if config.IOProfile != "" && config.IOProfile != transcript.Deterministic {
		return backends.Description{}, fmt.Errorf("unknown native I/O profile %q", config.IOProfile)
	}
	if config.ClockTick != "" && config.ClockTick != record.ClockTickStrict && config.ClockTick != record.ClockTickForward {
		return backends.Description{}, fmt.Errorf("clock tick policy %q must be strict or forward", config.ClockTick)
	}
	if _, err := transcript.MissingRequiredSemanticProbes(record.SemanticCoverage{}, config.RequiredSemanticProbes); err != nil {
		return backends.Description{}, err
	}
	profile := transcript.Default()
	version, boundary := transcript.BoundaryManifestIdentity()
	return backends.Description{Isolated: len(provider.config.CoordinatorCommand) != 0, RunnerBuild: provider.config.RunnerBuild, IOProfile: record.IOProfile{Name: profile.Name(), ImplementationSHA256: profile.ImplementationSHA256(), Inventory: string(profile.Inventory()), InventorySHA256: profile.InventorySHA256()}, BoundaryVersion: version, BoundarySHA256: boundary, SemanticInstrumentationSHA256: transcript.SemanticInstrumentationIdentity(), SemanticProbes: transcript.RequiredSemanticProbes()}, nil
}

func Prepare(context.Context, target.Spec) (target.Prepared, error) {
	return target.Prepared{}, errors.New("fixture preparation requires an injected preparer")
}
func (*Provider) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return Prepare(ctx, spec)
}
func (*Provider) ValidatePrepared(_ target.Spec, prepared target.Prepared, _ []string) error {
	return prepared.Verify()
}
func (*Provider) Run(context.Context, backends.Request) (backends.Result, error) {
	return backends.Result{}, errors.New("fixture execution requires an injected executor")
}
func (*Provider) Execute(_ context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	err := errors.New("fixture execution requires an injected executor")
	if request.Coordinator != nil && request.Coordinator.Stdout != nil {
		err = errors.Join(err, request.Coordinator.Stdout.Close())
	}
	return backends.ExecutionResult{}, err
}
func (*Provider) Evidence(ctx context.Context, request backends.EvidenceRequest) (backends.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return backends.Evidence{}, err
	}
	coverage := record.NewSemanticCoverage(request.Probes)
	var operations []backends.OperationOutcome
	if request.Transcript != nil {
		var err error
		coverage, err = transcript.DecodeSemanticCoverage(request.Transcript)
		if err != nil {
			return backends.Evidence{}, err
		}
		decoded, err := transcript.DecodeTranscript(request.Transcript)
		if err != nil {
			return backends.Evidence{}, err
		}
		for _, operation := range decoded {
			operations = append(operations, backends.OperationOutcome{Name: operation.Name, Result: operation.Result})
		}
	}
	missing, err := transcript.MissingRequiredSemanticProbes(coverage, request.RequiredSemanticProbes)
	return backends.Evidence{Coverage: coverage, Operations: operations, MissingRequiredProbes: missing}, err
}

func ReadToolchainIdentity(string) (target.ToolchainIdentity, error) {
	return target.ToolchainIdentity{GoVersion: runtime.Version(), BuildKey: "cbeccfefbc62a2ca026d9dded0316ecedfce33bd46b5c71b6645e86b67a0713e", TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH}, nil
}

func ProjectBuildInfo(info *debug.BuildInfo) record.BuildInfo {
	result := record.BuildInfo{GoVersion: info.GoVersion, Path: info.Path, MainModule: info.Main.Path, Settings: []record.BuildSetting{}}
	for _, setting := range info.Settings {
		result.Settings = append(result.Settings, record.BuildSetting{Key: setting.Key, Value: setting.Value})
	}
	return result
}

func (*Provider) ValidateReplay(ctx context.Context, opened *artifact.Opened) (target.Prepared, error) {
	if err := ctx.Err(); err != nil {
		return target.Prepared{}, err
	}
	manifest := opened.Manifest()
	if len(manifest.Target.Compatibility) != 0 {
		return target.Prepared{}, errors.New("fixture does not select compatibility packs")
	}
	file, err := opened.OpenPayload(manifest.Target.File, uint64(manifest.Target.Size))
	if err != nil {
		return target.Prepared{}, err
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		return target.Prepared{}, err
	}
	return target.Prepared{Path: path, Kind: target.Kind(manifest.Target.Kind), Source: manifest.Target.Source, SHA256: string(manifest.Target.SHA256), Size: uint64(manifest.Target.Size), Argv: append([]string(nil), manifest.Target.Argv...), BuildTags: append([]string(nil), manifest.Target.BuildTags...), Adapters: append([]record.TargetAdapter(nil), manifest.Target.Adapters...), Compatibility: append([]record.CompatibilityPack(nil), manifest.Target.Compatibility...), BuildInfo: manifest.Target.BuildInfo, GoVersion: manifest.Toolchain.GoVersion, BuildKey: manifest.Toolchain.BuildKey, TargetGOOS: manifest.Toolchain.TargetGOOS, TargetGOARCH: manifest.Toolchain.TargetGOARCH, CapabilityMode: target.CapabilityMode(manifest.Target.CapabilityMode)}, nil
}
