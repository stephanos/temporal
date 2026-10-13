package backend

import (
	"context"
	"errors"
	"slices"

	"go.temporal.io/server/tools/gomad3/internal/backends"
	"go.temporal.io/server/tools/gomad3/internal/backends/target"
	"go.temporal.io/server/tools/gomad3/internal/evidence/record"
)

func (p *Provider) Describe(ctx context.Context, config backends.Configuration) (backends.Description, error) {
	if err := ctx.Err(); err != nil {
		return backends.Description{}, err
	}
	if config.Target.Backend != "" && config.Target.Backend != Name || config.IOProfile != "" && config.IOProfile != Name || config.ClockTick != "" && config.ClockTick != record.ClockTickStrict || len(config.RequiredSemanticProbes) != 0 {
		return backends.Description{}, errors.New("unsupported WASM execution configuration")
	}
	return backends.Description{
		IOProfile: record.BackendIOProfile(record.BackendMetadata{Name: Name, Provenance: record.BackendPayload{SHA256: record.SHA256(p.options.ModelSHA256)}}),
		Toolchain: target.ToolchainIdentity{GoVersion: p.options.GoVersion, TargetGOOS: "wasip1", TargetGOARCH: "wasm"},
	}, nil
}

func (p *Provider) Evidence(ctx context.Context, request backends.EvidenceRequest) (backends.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return backends.Evidence{}, err
	}
	if len(request.Transcript) != 0 || len(request.Probes) != 0 || len(request.RequiredSemanticProbes) != 0 {
		return backends.Evidence{}, errors.New("WASM evidence does not contain native I/O transcript probes")
	}
	return backends.Evidence{Coverage: record.NewSemanticCoverage(nil)}, nil
}

func observedRequest(request backends.ExecutionRequest) (backends.Request, error) {
	if request.Coordinator != nil {
		err := errors.New("WASM provider does not support native coordinator execution")
		if request.Coordinator.Stdout != nil {
			err = errors.Join(err, request.Coordinator.Stdout.Close())
		}
		return backends.Request{}, err
	}
	if request.Command != "" && request.Command != request.Target.Path || len(request.Args) != 0 && (len(request.Target.Argv) == 0 || !slices.Equal(request.Args, request.Target.Argv[1:])) || request.Argv0 != "" && (len(request.Target.Argv) == 0 || request.Argv0 != request.Target.Argv[0]) {
		return backends.Request{}, errors.New("WASM command does not match prepared target")
	}
	if request.Simulation != nil || len(request.World.ExpectedInitial) != 0 || len(request.World.ReplayPlan) != 0 {
		return backends.Request{}, errors.New("WASM provider does not support native simulation execution")
	}
	converted := backends.Request{Target: request.Target, Seed: request.Seed, Environment: request.Env, Timeout: request.ExecutionTimeout, OutputBytes: request.OutputLimit, Diagnostics: request.Diagnostics}
	if request.IO != nil {
		mountRequested := false
		if mount := request.IO.ReadOnlyMount; mount != nil {
			mountRequested = len(mount.Mappings) != 0
			if snapshot := mount.Replay; snapshot != nil {
				mountRequested = mountRequested || len(snapshot.Entries) != 0 || len(snapshot.NotExist) != 0 || snapshot.Requests != 0 || snapshot.TotalBytes != 0
			}
		}
		if request.IO.Profile.Name != "" && request.IO.Profile.Name != Name || mountRequested || request.IO.Transcript != nil && (request.IO.Transcript.Replay || len(request.IO.Transcript.Expected) != 0) {
			return backends.Request{}, errors.New("WASM provider does not support native I/O execution")
		}
		if request.IO.Transcript != nil {
			converted.TranscriptBytes = request.IO.Transcript.Limit
		}
	}
	if request.Choice != nil {
		converted.Choice = &backends.ChoiceRequest{Mode: request.Choice.Mode, ExecutionIdentity: request.Choice.ExecutionIdentity, Limit: request.Choice.Limit, Tape: request.Choice.ReplayPlan}
	}
	return converted, nil
}

func (p *Provider) Execute(ctx context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	converted, err := observedRequest(request)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	observed, err := p.Run(ctx, converted)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	return backends.ProjectObservedResult(converted, observed, request.StdoutHead, request.StderrHead)
}
