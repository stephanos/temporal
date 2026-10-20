package measure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/backends/target"
	wasmbackend "go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner"
	campaignoptions "go.temporal.io/server/tools/gomad/internal/runner/options"
)

func stressRunnerProvider(ctx context.Context, root string, original wasi.Request, stress wasmbackend.StressExecution, observe func(wasmbackend.ExecutionObservation)) (*wasmbackend.Provider, error) {
	compiler, _, _, err := prototypeToolchain(ctx)
	if err != nil {
		return nil, err
	}
	compilerHash, err := fileHash(ctx, compiler)
	if err != nil {
		return nil, err
	}
	stress.Diagnostics = original.Runtime != nil && original.Runtime.Diagnostics
	options := wasmbackend.Options{
		CompilerPath: compiler, CompilerSHA256: "sha256:" + compilerHash, GoVersion: "go1.27.1",
		HelperPath: original.HelperPath, HelperSHA256: "sha256:" + strings.TrimPrefix(original.HelperSHA256, "sha256:"),
		ModelSHA256: wasi.ImplementationSHA256(), Engine: original.Engine,
		MemoryBytes: original.MemoryBytes, Fuel: original.Fuel, Config: original.Config,
		CacheRoot: filepath.Join(root, "preparation-cache"), CompileCacheRoot: original.CompileCache,
		RuntimeRoot:     filepath.Join(scopeFor(ctx).options.Repo, "tools/gomad/internal/model/runtimesource"),
		StressExecution: &stress, ObserveResults: observe,
	}
	replayOptions := options
	replayOptions.CompileCacheRoot = filepath.Join(root, "replay-compile-cache")
	if err := os.MkdirAll(replayOptions.CompileCacheRoot, 0700); err != nil {
		return nil, err
	}
	manifest, err := installation.Build(replayOptions)
	if err != nil {
		return nil, err
	}
	path, err := stressRunnerInstallation(root, original)
	if err != nil {
		return nil, err
	}
	scopeFor(ctx).warm.Lock()
	defer scopeFor(ctx).warm.Unlock()
	if err := installation.Write(path, manifest); err != nil {
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		retained, err := installation.Resolve(installation.Spec{ExplicitManifest: path})
		if err != nil {
			return nil, err
		}
		want, err := json.Marshal(manifest)
		if err != nil {
			return nil, err
		}
		got, err := json.Marshal(retained.Manifest)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(want, got) {
			return nil, errors.New("retained stress installation changed declared inputs")
		}
	}
	return wasmbackend.New(options)
}

func stressRunnerInstallation(root string, original wasi.Request) (string, error) {
	encoded, err := json.Marshal(original.Config)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "stress-installation-"+digest(encoded)+".json"), nil
}

func stressRunnerCampaign(ctx context.Context, sharedRoot string, provider backends.Provider, spec target.Spec, original wasi.Request) (runner.CampaignResult, error) {
	if original.Runtime == nil || original.Runtime.Choice == nil {
		return runner.CampaignResult{}, errors.New("stress campaign requires a fresh seeded request")
	}
	if choices := original.Runtime.Choice; choices.Mode != choice.ModeRecord || choices.Trace != nil || choices.TraceSHA256 != ([32]byte{}) || choices.TapeSHA256 != ([32]byte{}) || choices.Decisions != 0 || choices.PrefixControl != nil {
		return runner.CampaignResult{}, errors.New("stress campaign requires a fresh seeded request")
	}
	choices, diagnostics := original.Runtime.Choice.Limit, original.Runtime.Diagnostics
	if len(spec.Scenario) != 0 {
		choices, diagnostics = 0, false
	}
	return runner.Explore(ctx, runner.CampaignSpec{
		Backend: provider, Strategy: runner.StrategySeed, Seeds: strconv.FormatUint(original.Runtime.Seed, 10), Parallel: 1,
		ExecutionTimeout: original.Timeout, OverallTimeout: original.Timeout + time.Hour, TerminateGrace: time.Second,
		OnFailure: runner.PolicyFirst, FailureBudget: 1,
		OutputLimit: original.Config.Limits.OutputBytes, IOTranscriptLimit: original.Config.Limits.TranscriptBytes,
		WorldTransitionLimit: 64 << 20, Diagnostics: diagnostics, ChoiceTraceLimit: choices,
		Artifacts: filepath.Join(sharedRoot, "artifacts"), Environment: stressRunnerEnvironment(original.Config.Environment),
		RunnerBuild: "gomad-wasm-determinism-stress/v1", Coverage: campaignoptions.CoverageNone,
		KeepSuccesses: runner.KeepSuccessesAll, SuccessArtifactLimit: 1, SuccessBytesLimit: 2 << 30,
		Target: spec,
	})
}

func stressRunnerEnvironment(environment []string) []string {
	var declared []string
	for _, entry := range environment {
		name, _, _ := strings.Cut(entry, "=")
		switch name {
		case "PWD", "TZ", "GOMADSEED", "GOMAD_IO_PROFILE", "GOMAD_CHOICE_PROFILE", choice.DiagnosticProfileEnvironment:
		default:
			declared = append(declared, entry)
		}
	}
	return declared
}
