package qualification

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
	gomad3sim "go.temporal.io/server/tools/gomad/internal/model/cluster"
	"go.temporal.io/server/tools/gomad/internal/runner"
)

// RunCluster adapts real independent guests to the existing repeated workload,
// Artifact and qualification owners. Recording may call a scenario; replay only
// executes the broker owner's sealed typed lifecycle plan from retained bytes.
func RunCluster(ctx context.Context, spec Spec, cluster gomad3sim.Spec, scenario gomad3sim.Scenario, provider *simulation.Provider) (Result, error) {
	result := Result{Manifest: spec.Manifest, CompilerCommands: []backend.CompilerCommandMetrics{}, InitialPreparationWallNanos: spec.coldPreparationWallNanos, InitialPreparationCompiled: preparationCompiled(spec.initialCompilerCommands), InitialCompilerCommandCount: uint64(len(spec.initialCompilerCommands)), PreparationReusedWithinProducer: spec.preparationReused, PreparationProvenanceSHA256: preparationProvenance(spec), IntegratedSourceManifest: bytes.Clone(spec.integratedSourceManifest), OperatorPreparationBefore: spec.operatorPreparationBefore, OperatorPreparationAfter: spec.operatorPreparationAfter, OperatorExecutionBefore: hostexec.CurrentProcessResourceUsage()}
	result.CompilerCommands = append(result.CompilerCommands, spec.initialCompilerCommands...)
	if err := spec.Manifest.Validate(); err != nil {
		return result, err
	}
	if provider == nil || scenario == nil || cluster.Backend != gomad3sim.BackendWASM || cluster.Seed != spec.Manifest.Seed || spec.Manifest.Profile != "cooperative" || spec.preparedTarget.Backend == nil {
		return result, errors.New("cluster qualification requires frozen cooperative executable inputs")
	}
	if err := spec.preparedTarget.Verify(); err != nil {
		return result, err
	}
	if len(spec.integratedSourceManifest) != 0 {
		if err := validateSourceManifest(spec.integratedSourceManifest, true); err != nil {
			return result, err
		}
	}
	primaryBound := false
	for _, boot := range provider.Boots {
		if boot.ModuleSHA256 == strings.TrimPrefix(spec.preparedTarget.SHA256, "sha256:") {
			primaryBound = true
		}
		if "sha256:"+boot.HelperSHA256 != string(spec.Manifest.HelperSHA256) || boot.Engine != spec.Options.Engine || boot.Fuel != spec.Manifest.Fuel || boot.MemoryBytes != spec.Manifest.MemoryBytes {
			return result, errors.New("cluster boot differs from frozen helper/engine/limits")
		}
	}
	if !primaryBound || len(cluster.Nodes) == 0 || len(cluster.Nodes) > 2 {
		return result, errors.New("cluster primary WASM module or resident guest budget is unbound")
	}
	iteration, replayIteration := uint64(0), uint64(0)
	qualified, err := RunWorkload(ctx, WorkloadSpec{Command: spec.Command, Seed: spec.Manifest.Seed, Repeat: spec.Manifest.Repeat, ArtifactRoot: spec.ArtifactRoot, ReplaySuccesses: true,
		Explore: func(ctx context.Context, _ runner.CampaignSpec) (runner.CampaignResult, error) {
			iteration++
			started := time.Now()
			observeHostMemory(spec.observeHostMemoryStage, "record.execute.before")
			observed, err := gomad3sim.RunWASM(ctx, cluster, scenario, provider)
			observeHostMemory(spec.observeHostMemoryStage, "record.execute.after")
			elapsed := uint64(time.Since(started))
			firstSample := len(result.Samples)
			incarnations := provider.CompletedSummaries()
			result.Samples = append(result.Samples, incarnationSamples(incarnations, "execute", iteration, "", spec.Manifest.Fuel)...)
			if spec.Manifest.Workload == "sqlite" {
				if captureErr := captureNativeCrashCatalogues(&result, provider.CompletedResults(), "execute", iteration); captureErr != nil {
					return runner.CampaignResult{}, captureErr
				}
			}
			if err != nil {
				return runner.CampaignResult{}, err
			}
			observeHostMemory(spec.observeHostMemoryStage, "record.retention.before")
			payload, err := provider.RetentionPayload(cluster, observed.Record)
			observeHostMemory(spec.observeHostMemoryStage, "record.retention.after")
			if err != nil {
				return runner.CampaignResult{}, err
			}
			manifest, input, evidence, err := clusterArtifact(spec, observed, payload, incarnations, started, elapsed)
			if err != nil {
				return runner.CampaignResult{}, err
			}
			input.Manifest = manifest
			observeHostMemory(spec.observeHostMemoryStage, "artifact.publish.before")
			published, err := artifact.PublishArtifact(artifact.Store{Root: spec.ArtifactRoot}, input)
			observeHostMemory(spec.observeHostMemoryStage, "artifact.publish.after")
			if err != nil {
				return runner.CampaignResult{}, err
			}
			for i := firstSample; i < len(result.Samples); i++ {
				result.Samples[i].OutcomeSHA256 = record.HashBytes(payload.Data)
			}
			summary := runner.CampaignResult{CampaignPath: published.Path, ExecutionElapsedNanos: elapsed, ExecutionEvidence: &evidence}
			if evidence.Outcome.Domain == "success" {
				summary.RetainedSuccesses = 1
				summary.SuccessArtifacts = []string{published.Path}
			} else {
				summary.Artifacts = []string{published.Path}
			}
			return summary, nil
		},
		ReplayArtifact: func(ctx context.Context, replay runner.ReplaySpec) (runner.ReplayResult, error) {
			replayIteration++
			observed, completed, err := replayClusterArtifactObserved(ctx, replay.ArtifactPath, spec.ArtifactRoot, simulation.InstalledHelper{Path: spec.Options.HelperPath, SHA256: strings.TrimPrefix(spec.Options.HelperSHA256, "sha256:"), Engine: spec.Options.Engine}, spec.observeHostMemoryStage)
			if completed != nil {
				result.Samples = append(result.Samples, incarnationSamples(completed.CompletedSummaries(), "replay", replayIteration, "", spec.Manifest.Fuel)...)
				if spec.Manifest.Workload == "sqlite" {
					if captureErr := captureNativeCrashCatalogues(&result, completed.CompletedResults(), "replay", replayIteration); captureErr != nil {
						return runner.ReplayResult{}, captureErr
					}
				}
			}
			return observed, err
		}})
	result.Qualification, result.QualificationPath = qualified.Report, qualified.ReportPath
	if err != nil {
		return result, err
	}
	result.QualificationBytes, err = hostfs.ReadBounded(qualified.ReportPath, 16<<20)
	if err != nil {
		return result, err
	}
	for _, execution := range qualified.Report.Executions {
		if execution.Replay == nil || !execution.Replay.Match || qualified.Report.Evidence == nil || qualified.Report.Evidence.Outcome.Domain == "success" {
			continue
		}
		for i := range result.Samples {
			if result.Samples[i].Phase == "execute" {
				result.Samples[i].ReplayableBugSHA256 = result.Samples[i].OutcomeSHA256
			}
		}
	}
	finishMeasurements(&result, uint64(len(cluster.Nodes)))
	return result, nil
}

func clusterArtifact(spec Spec, observed gomad3sim.Result, payload target.BackendPayload, incarnations []simulation.IncarnationSummary, started time.Time, elapsed uint64) (record.ExecutionRecord, artifact.ArtifactInput, record.ExecutionEvidence, error) {
	world, worldPayloads := record.NoneWorld() // Global broker World is sealed in backend evidence.
	metadata := record.CloneBackendMetadata(spec.preparedTarget.Backend)
	metadata.ReplayMode = record.ReplayExact
	// Copy the reference: &payload.Reference would keep the whole payload,
	// including its Data, reachable from the retained execution evidence.
	reference := payload.Reference
	metadata.Evidence = &reference
	targetRecord := spec.preparedTarget.RecordTarget()
	targetRecord.Backend = metadata
	profile := record.BackendIOProfile(*metadata)
	stdout, stderr := []byte{}, []byte{}
	for _, output := range observed.Outputs {
		if output.Stream == gomad3sim.OutputStdout {
			stdout = append(stdout, output.Bytes...)
		} else {
			stderr = append(stderr, output.Bytes...)
		}
	}
	stream := func(data []byte) record.Stream {
		return record.Stream{FullSHA256: record.HashBytes(data), RetainedSHA256: record.HashBytes(data), TotalBytes: record.Uint64String(len(data)), RetainedBytes: record.Uint64String(len(data))}
	}
	outcome, err := clusterOutcome(observed, incarnations)
	if err != nil {
		return record.ExecutionRecord{}, artifact.ArtifactInput{}, record.ExecutionEvidence{}, err
	}
	groupGone := len(incarnations) != 0
	if observed.Record.WASM == nil {
		return record.ExecutionRecord{}, artifact.ArtifactInput{}, record.ExecutionEvidence{}, errors.New("cluster has no global clock evidence")
	}
	virtual := observed.Record.WASM.VirtualElapsedNanos
	for _, incarnation := range incarnations {
		groupGone = groupGone && incarnation.Reaped
	}
	if !groupGone {
		return record.ExecutionRecord{}, artifact.ArtifactInput{}, record.ExecutionEvidence{}, errors.New("cluster helper was not reaped")
	}
	environment := []record.Environment{{Name: "GOMAD3_IO_PROFILE", Value: profile.Name}, {Name: "GOMADSEED", Value: fmt.Sprint(spec.Manifest.Seed)}, {Name: "TZ", Value: "UTC"}}
	limits := record.Limits{ExecutionTimeoutNanos: record.Uint64String(15 * time.Minute), OverallTimeoutNanos: record.Uint64String(15 * time.Minute), OutputBytes: record.Uint64String(spec.Options.Config.Limits.OutputBytes * uint64(len(incarnations))), WorldTransitionBytes: 32 << 20}
	manifest := record.ExecutionRecord{SchemaVersion: record.SchemaVersion, ArtifactKind: record.ArtifactSuccess, CreatedAt: started.UTC().Format(time.RFC3339Nano), CampaignID: "wasm-cluster-" + strings.TrimPrefix(string(payload.Reference.SHA256), "sha256:"), Seed: record.Uint64String(spec.Manifest.Seed), ReplayMode: record.ReplayExact, Runner: record.Runner{RecordContract: record.RecordContract, RunnerBuild: spec.Campaign.RunnerBuild, HostOS: spec.Manifest.HostOS, HostArch: spec.Manifest.HostArch}, Toolchain: spec.preparedTarget.RecordToolchain(), Target: targetRecord, IOProfile: profile, Environment: environment, Limits: limits, World: world, Outcome: outcome, Streams: record.Streams{Stdout: stream(stdout), Stderr: stream(stderr)}, Host: record.Host{StartedAt: started.UTC().Format(time.RFC3339Nano), FinishedAt: time.Now().UTC().Format(time.RFC3339Nano), ElapsedNanos: record.Uint64String(elapsed)}}
	if outcome.Domain != "success" {
		manifest.ArtifactKind = record.ArtifactTargetFailure
	}
	input := artifact.ArtifactInput{Manifest: manifest, TargetPath: spec.preparedTarget.Path, Stdout: stdout, Stderr: stderr, World: worldPayloads, BackendPayloads: append(append([]target.BackendPayload{}, spec.preparedTarget.BackendPayloads...), payload)}
	evidence := record.ExecutionEvidence{Schema: record.ExecutionEvidenceSchema, Seed: manifest.Seed, RunnerBuild: manifest.Runner.RunnerBuild, Toolchain: manifest.Toolchain, Target: targetRecord, IOProfile: record.IOContract{Name: profile.Name, ImplementationSHA256: record.SHA256(profile.ImplementationSHA256), InventorySHA256: record.SHA256(profile.InventorySHA256)}, Environment: environment, Limits: record.ExecutionLimitsEvidence{ExecutionTimeoutNanos: limits.ExecutionTimeoutNanos, OutputBytes: limits.OutputBytes, WorldTransitionBytes: limits.WorldTransitionBytes}, Outcome: record.OutcomeEvidence{Domain: outcome.Domain, Reason: outcome.Reason, Termination: outcome.Termination, ExitCode: outcome.ExitCode}, GroupGone: groupGone, Stdout: manifest.Streams.Stdout, Stderr: manifest.Streams.Stderr, BackendEvidenceSHA256: payload.Reference.SHA256, IOTranscriptSHA256: record.HashBytes(nil), IOTranscriptComplete: true, VirtualTimeElapsedNanos: record.Uint64String(virtual), World: world}
	return manifest, input, evidence, nil
}

func clusterOutcome(observed gomad3sim.Result, incarnations []simulation.IncarnationSummary) (record.Outcome, error) {
	outcome := record.Outcome{Domain: "success", Reason: "cluster_completed", Termination: "none"}
	if observed.Outcome == gomad3sim.OutcomeOracleFailed {
		outcome = record.Outcome{Domain: "target", Reason: "oracle_failed", Termination: "none"}
	} else if observed.Outcome != gomad3sim.OutcomeCompleted {
		return record.Outcome{}, fmt.Errorf("cluster qualification did not complete: %s (%s)", observed.Outcome, observed.Reason)
	}
	if len(incarnations) == 0 {
		return record.Outcome{}, errors.New("cluster qualification has no completed incarnation evidence")
	}
	for _, incarnation := range incarnations {
		r := incarnation
		if !r.Reaped {
			return record.Outcome{}, errors.New("cluster helper was not reaped")
		}
		switch r.Termination {
		case "exit":
			if r.ExitCode == nil {
				return record.Outcome{}, errors.New("cluster guest exit lacks actual exit code")
			}
			if *r.ExitCode != 0 {
				code := record.Uint64String(*r.ExitCode)
				outcome = record.Outcome{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ExitCode: &code}
			}
		case "trap":
			if outcome.Domain == "success" {
				outcome = record.Outcome{Domain: "target", Reason: "wasm_trap", Termination: "none"}
			}
		case "crash", "storage_crash", "stop":
			// Typed planned interruptions are sealed and validated by RetentionPayload.
		default:
			return record.Outcome{}, fmt.Errorf("cluster qualification cannot retain helper %q as an application outcome", r.Termination)
		}
	}
	return outcome, nil
}

func incarnationSamples(incarnations []simulation.IncarnationSummary, phase string, iteration uint64, outcome record.SHA256, fuel uint64) []Sample {
	samples := make([]Sample, 0, len(incarnations))
	for _, incarnation := range incarnations {
		r := incarnation
		sample := Sample{Phase: phase, Iteration: iteration, Incarnation: fmt.Sprintf("%s/%d", incarnation.Handle.Node, incarnation.Handle.Incarnation), OutcomeSHA256: outcome, GuestPeakMemoryBytes: r.PeakMemoryBytes, VirtualTimeNanos: r.VirtualTimeElapsedNanos, WASITraceBytes: r.WASITraceBytes, ChoiceTraceBytes: r.ChoiceTraceBytes, DiagnosticTraceBytes: r.DiagnosticTraceBytes, StorageTraceBytes: r.StorageTraceBytes, NetworkTraceBytes: r.NetworkTraceBytes}
		if r.Telemetry != nil {
			sample.FuelUsed = fuel - r.FuelRemaining
		}
		sample.TraceBytes = sample.WASITraceBytes + sample.ChoiceTraceBytes + sample.DiagnosticTraceBytes + sample.StorageTraceBytes + sample.NetworkTraceBytes
		sample.HelperWallNanos = incarnation.WallElapsedNanos
		sample.AuthorizationToTerminalNanos = r.AuthorizationToTerminalNanos
		if t := r.Telemetry; t != nil {
			sample.EnginePrepareNanos = t.PrepareNanos
			sample.EngineCompileNanos = t.CompileNanos
			sample.CacheLoadNanos = t.CacheLoadNanos
			sample.CacheWriteNanos = t.CacheWriteNanos
			sample.CompileCacheHit = t.CacheHit
			sample.InstantiationNanos = t.InstantiateNanos
			sample.ExecutionWallNanos = t.ExecuteNanos
			if t.ResourceUsageAvailable {
				sample.CPUNanos = t.ProcessUserNanos + t.ProcessSystemNanos
			}
		}
		if r.HelperResourceUsage != nil {
			sample.CPUNanos = r.HelperResourceUsage.UserNanos + r.HelperResourceUsage.SystemNanos
		}
		sample.recordHelperPeakRSS(r.Telemetry, r.HelperResourceUsage)
		samples = append(samples, sample)
	}
	return samples
}

// ReplayClusterArtifact only opens verified retained Artifact inputs and the
// installed helper. It never reads the original source, compiler or runtime.
func ReplayClusterArtifact(ctx context.Context, path, directory string, installed simulation.InstalledHelper) (runner.ReplayResult, []simulation.IncarnationResult, error) {
	replay, completed, err := replayClusterArtifact(ctx, path, directory, installed)
	if completed == nil {
		return replay, nil, err
	}
	return replay, completed.CompletedResults(), err
}

func replayClusterArtifact(ctx context.Context, path, directory string, installed simulation.InstalledHelper) (replay runner.ReplayResult, completed *simulation.Provider, retErr error) {
	return replayClusterArtifactObserved(ctx, path, directory, installed, nil)
}

func replayClusterArtifactObserved(ctx context.Context, path, directory string, installed simulation.InstalledHelper, observer func(string)) (replay runner.ReplayResult, completed *simulation.Provider, retErr error) {
	observeHostMemory(observer, "replay.open.before")
	opened, err := artifact.OpenArtifact(path)
	observeHostMemory(observer, "replay.open.after")
	if err != nil {
		return runner.ReplayResult{}, nil, err
	}
	defer func() { retErr = errors.Join(retErr, opened.Close()) }()
	manifest := opened.Manifest()
	if manifest.Target.Backend == nil || manifest.Target.Backend.Evidence == nil || !simulation.IsRetainedSchema(manifest.Target.Backend.Evidence.Schema) {
		return runner.ReplayResult{}, nil, errors.New("artifact lacks retained WASM cluster evidence")
	}
	ref := *manifest.Target.Backend.Evidence
	// Older retained schemas are refused before their payload is read.
	if err := simulation.CheckRetainedSchema(ref.Schema); err != nil {
		return runner.ReplayResult{}, nil, err
	}
	module, err := retainedTargetModule(opened, manifest.Target)
	if err != nil {
		return runner.ReplayResult{}, nil, err
	}
	observeHostMemory(observer, "replay.read.before")
	data, err := opened.ReadPayload(ref.File, 512<<20)
	observeHostMemory(observer, "replay.read.after")
	if err != nil {
		return runner.ReplayResult{}, nil, err
	}
	payload := target.BackendPayload{Reference: ref, Data: data}
	observeHostMemory(observer, "replay.bound.before")
	observed, expected, completed, err := simulation.ReplayBoundRetained(ctx, payload, module, directory, installed)
	observeHostMemory(observer, "replay.bound.after")
	if err != nil {
		return runner.ReplayResult{}, completed, err
	}
	observeHostMemory(observer, "replay.encode.before")
	encoded, err := gomad3sim.EncodeClusterRecord(observed.Record)
	observeHostMemory(observer, "replay.encode.after")
	if err != nil {
		return runner.ReplayResult{}, completed, err
	}
	observeHostMemory(observer, "replay.compare.before")
	match := record.HashBytes(encoded) == expected
	observeHostMemory(observer, "replay.compare.after")
	// Cluster exactness covers the owner's sealed global lifecycle and each
	// incarnation's exact OR interrupted-prefix contract, not a global exact tape.
	result := runner.ReplayResult{Artifact: opened.Snapshot(), Verified: true, Match: match, ChoiceReplayStatus: runner.ChoiceReplayNone}
	if !match {
		result.Divergence = "retained WASM cluster execution changed"
	}
	return result, completed, nil
}

// retainedTargetModule supplies the module the retained evidence references by
// digest. The verified Artifact target supplies its bytes, copied without
// whole-file buffering once its listed size is within the module bound.
func retainedTargetModule(opened *artifact.Opened, target record.Target) (simulation.RetainedModule, error) {
	if err := simulation.CheckRetainedModuleSize(uint64(target.Size)); err != nil {
		return simulation.RetainedModule{}, err
	}
	return simulation.RetainedModule{SHA256: string(target.SHA256), Copy: func(destination string) error {
		return opened.CopyPayload(target.File, destination, 0o700)
	}}, nil
}

func observeHostMemory(observer func(string), stage string) {
	if observer != nil {
		observer(stage)
	}
}
