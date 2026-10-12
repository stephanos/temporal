package qualification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"go.temporal.io/server/tools/gomad3/volume"
	"go.temporal.io/server/tools/gomad3/hostexec"
	"go.temporal.io/server/tools/gomad3/hostfs"
	gomadqualification "go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/qualification/workload"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
	runnerbackend "go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad_wasm/backend"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

type Spec struct {
	// OperatorStart is captured before decoding/config/input capture in a fresh
	// operator process. It is host observation, never retained input configuration.
	OperatorStart                                       *hostexec.ResourceUsage `json:"-"`
	observeHostMemoryStage                              func(string)
	Manifest                                            Manifest
	Options                                             backend.Options
	Campaign                                            runner.CampaignSpec
	Command                                             []string
	ArtifactRoot                                        string
	initialCompilerCommands                             []backend.CompilerCommandMetrics
	coldPreparationWallNanos                            uint64
	integratedSourceManifest                            []byte
	preparedTarget                                      target.Prepared
	preparationReused                                   bool
	operatorPreparationBefore, operatorPreparationAfter *hostexec.ResourceUsage
}

// Result keeps the existing retained support report distinct from measured
// costs. CostFailure prevents missing telemetry from becoming a cost pass.
type Result struct {
	NativeCrashCatalogues                               []NativeCrashCatalogue `json:",omitempty"`
	NativeCrashSelection                                *volume.CrashState     `json:",omitempty"`
	NativeCrashSourceResultSHA256                       record.SHA256          `json:",omitempty"`
	PreparationProvenanceSHA256                         record.SHA256
	PreparationReusedWithinProducer                     bool
	OperatorProcessID                                   uint64
	OperatorPreparationBefore, OperatorPreparationAfter *hostexec.ResourceUsage
	OperatorExecutionBefore, OperatorExecutionAfter     *hostexec.ResourceUsage
	RunOrder                                            []string
	ResidentGuests                                      uint64
	Budget                                              *BudgetEvaluation
	BudgetFailure                                       string
	Manifest                                            Manifest
	Qualification                                       gomadqualification.QualificationReport
	QualificationPath                                   string
	QualificationBytes                                  []byte
	Samples                                             []Sample
	Cost                                                *Cost
	CostFailure                                         string
	CompilerCommands                                    []backend.CompilerCommandMetrics
	InitialPreparationWallNanos                         uint64
	InitialPreparationCompiled                          bool
	InitialCompilerCommandCount                         uint64
	IntegratedSourceManifest                            []byte
}

type measuredProvider struct {
	runnerbackend.Provider
	identityProvider *backend.Provider
	manifest         Manifest
	preparationWall  uint64
}

func (p *measuredProvider) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	started := time.Now()
	prepared, err := p.Provider.Prepare(ctx, spec)
	p.preparationWall = uint64(time.Since(started))
	if err == nil {
		identity, identityErr := p.identityProvider.QualificationIdentity(prepared)
		if identityErr != nil {
			return target.Prepared{}, identityErr
		}
		if identity.WorkloadSourceSHA256 != p.manifest.WorkloadSourceSHA256 || identity.RuntimeSHA256 != p.manifest.RuntimeSHA256 {
			return target.Prepared{}, errors.New("prepared source/runtime differs from frozen WASM manifest")
		}
	}
	return prepared, err
}

func Run(ctx context.Context, spec Spec) (Result, error) {
	result := Result{Manifest: spec.Manifest, Samples: []Sample{}, CompilerCommands: append([]backend.CompilerCommandMetrics{}, spec.initialCompilerCommands...), InitialPreparationWallNanos: spec.coldPreparationWallNanos, InitialPreparationCompiled: preparationCompiled(spec.initialCompilerCommands), InitialCompilerCommandCount: uint64(len(spec.initialCompilerCommands)), PreparationReusedWithinProducer: spec.preparationReused, PreparationProvenanceSHA256: preparationProvenance(spec), IntegratedSourceManifest: append([]byte{}, spec.integratedSourceManifest...), OperatorPreparationBefore: spec.operatorPreparationBefore, OperatorPreparationAfter: spec.operatorPreparationAfter, OperatorExecutionBefore: hostexec.CurrentProcessResourceUsage()}
	if err := spec.Manifest.Validate(); err != nil {
		return result, err
	}
	if len(spec.integratedSourceManifest) != 0 {
		if err := validateSourceManifest(spec.integratedSourceManifest, true); err != nil {
			return result, err
		}
	}
	engine, err := json.Marshal(spec.Options.Engine)
	if err != nil {
		return result, err
	}
	config, err := json.Marshal(spec.Options.Config)
	if err != nil {
		return result, err
	}
	if record.HashBytes(config) != spec.Manifest.HostConfigSHA256 {
		return result, errors.New("host config differs from frozen WASM manifest")
	}
	profile := "controlled"
	if spec.Options.Config.Profile == wasi.CooperativeProfile {
		profile = "cooperative"
	}
	if profile != spec.Manifest.Profile || spec.Options.Fuel != spec.Manifest.Fuel || spec.Options.MemoryBytes != spec.Manifest.MemoryBytes || record.SHA256(spec.Options.CompilerSHA256) != spec.Manifest.CompilerSHA256 || record.SHA256(spec.Options.HelperSHA256) != spec.Manifest.HelperSHA256 || record.SHA256(spec.Options.ModelSHA256) != spec.Manifest.ModelSHA256 || record.HashBytes(engine) != spec.Manifest.EngineSHA256 || spec.Options.Engine.HostOS != spec.Manifest.HostOS || spec.Options.Engine.HostArch != spec.Manifest.HostArch {
		return result, errors.New("backend options differ from frozen WASM manifest")
	}
	var mutex sync.Mutex
	phase := "execute"
	iteration := uint64(0)
	previous := spec.Options.ObserveExecution
	previousCompiler := spec.Options.ObserveCompilerCommand
	spec.Options.ObserveCompilerCommand = func(metrics backend.CompilerCommandMetrics) {
		mutex.Lock()
		result.CompilerCommands = append(result.CompilerCommands, metrics)
		mutex.Unlock()
		if previousCompiler != nil {
			previousCompiler(metrics)
		}
	}
	spec.Options.ObserveExecution = func(metrics backend.ExecutionMetrics) {
		mutex.Lock()
		sample := Sample{Phase: phase, Iteration: iteration, HelperWallNanos: metrics.WallElapsedNanos, GuestPeakMemoryBytes: metrics.GuestPeakMemoryBytes, FuelUsed: metrics.FuelUsed, TraceBytes: metrics.WASIBytes + metrics.ChoiceBytes + metrics.DiagnosticBytes, VirtualTimeNanos: metrics.VirtualTimeElapsedNanos}
		sample.AuthorizationToTerminalNanos = metrics.AuthorizationToTerminalNanos
		sample.WASITraceBytes, sample.ChoiceTraceBytes, sample.DiagnosticTraceBytes, sample.StorageTraceBytes, sample.NetworkTraceBytes = metrics.WASIBytes, metrics.ChoiceBytes, metrics.DiagnosticBytes, metrics.StorageBytes, metrics.NetworkBytes
		sample.TraceBytes += metrics.StorageBytes + metrics.NetworkBytes
		if telemetry := metrics.Telemetry; telemetry != nil {
			sample.EnginePrepareNanos = telemetry.PrepareNanos
			sample.EngineCompileNanos = telemetry.CompileNanos
			sample.CacheLoadNanos = telemetry.CacheLoadNanos
			sample.CacheWriteNanos = telemetry.CacheWriteNanos
			sample.CompileCacheHit = telemetry.CacheHit
			sample.InstantiationNanos = telemetry.InstantiateNanos
			sample.ExecutionWallNanos = telemetry.ExecuteNanos
			if telemetry.ResourceUsageAvailable {
				sample.CPUNanos = telemetry.ProcessUserNanos + telemetry.ProcessSystemNanos
				sample.HostPeakRSSBytes = telemetry.PeakRSSBytes
			}
		}
		if metrics.HelperResourceUsage != nil {
			sample.CPUNanos = metrics.HelperResourceUsage.UserNanos + metrics.HelperResourceUsage.SystemNanos
			sample.HostPeakRSSBytes = metrics.HelperResourceUsage.PeakRSSBytes
		}
		result.Samples = append(result.Samples, sample)
		mutex.Unlock()
		if previous != nil {
			previous(metrics)
		}
	}
	provider, err := backend.New(spec.Options)
	if err != nil {
		return result, err
	}
	measured := &measuredProvider{Provider: provider, identityProvider: provider, manifest: spec.Manifest}
	campaign := spec.Campaign
	campaign.Backend = measured
	campaign.Target.Backend = backend.Name
	campaign.Seeds = fmt.Sprint(spec.Manifest.Seed)
	campaign.Parallel = 1
	campaign.Strategy = runner.StrategySeed
	campaign.Coverage = runner.CoverageNone
	if profile == "cooperative" {
		campaign.ChoiceTraceLimit = max(campaign.ChoiceTraceLimit, 1<<20)
		campaign.Coverage = runner.CoverageChoice
	}
	campaign.CollectExecutionEvidence = true
	campaign.KeepSuccesses = runner.KeepSuccessesAll
	campaign.SuccessArtifactLimit = 1
	campaign.SuccessBytesLimit = max(campaign.SuccessBytesLimit, 256<<20)
	qualified, err := workload.Run(ctx, workload.Spec{Command: spec.Command, Seed: spec.Manifest.Seed, Repeat: spec.Manifest.Repeat, ArtifactRoot: spec.ArtifactRoot, Campaign: campaign, Replay: runner.ReplaySpec{Backend: measured}, ReplaySuccesses: true,
		Explore: func(ctx context.Context, campaign runner.CampaignSpec) (runner.CampaignResult, error) {
			mutex.Lock()
			phase = "execute"
			iteration++
			first := len(result.Samples)
			mutex.Unlock()
			summary, err := runner.Explore(ctx, campaign)
			mutex.Lock()
			defer mutex.Unlock()
			if len(result.Samples) == first+1 && summary.ExecutionEvidence != nil {
				encoded, encodeErr := json.Marshal(struct {
					Outcome               runner.OutcomeEvidence
					Stdout, Stderr        record.Stream
					BackendEvidenceSHA256 record.SHA256
				}{summary.ExecutionEvidence.Outcome, summary.ExecutionEvidence.Stdout, summary.ExecutionEvidence.Stderr, summary.ExecutionEvidence.BackendEvidenceSHA256})
				if encodeErr != nil {
					return summary, errors.Join(err, encodeErr)
				}
				result.Samples[first].OutcomeSHA256 = record.HashBytes(encoded)
				result.Samples[first].PreparationWallNanos = measured.preparationWall
			}
			return summary, err
		},
		ReplayArtifact: func(ctx context.Context, replay runner.ReplaySpec) (runner.ReplayResult, error) {
			mutex.Lock()
			if phase != "replay" {
				iteration = 0
			}
			phase = "replay"
			iteration++
			mutex.Unlock()
			return runner.Replay(ctx, replay)
		},
	})
	result.Qualification, result.QualificationPath = qualified.Report, qualified.ReportPath
	if err != nil {
		return result, err
	}
	result.QualificationBytes, err = hostfs.ReadBounded(qualified.ReportPath, 16<<20)
	if err != nil {
		return result, err
	}
	for i, run := range qualified.Report.Executions {
		if i >= len(result.Samples) || run.Replay == nil || !run.Replay.Match || qualified.Report.Evidence == nil || qualified.Report.Evidence.Outcome.Domain == "success" {
			continue
		}
		// The retained failure's semantic outcome is the bug identity; repeat
		// and replay count do not inflate distinct bug yield.
		result.Samples[i].ReplayableBugSHA256 = result.Samples[i].OutcomeSHA256
	}
	finishMeasurements(&result, 1)
	return result, nil
}

func finishMeasurements(result *Result, residentGuests uint64) {
	result.OperatorProcessID = uint64(os.Getpid())
	result.OperatorExecutionAfter = hostexec.CurrentProcessResourceUsage()
	result.ResidentGuests = residentGuests
	result.RunOrder = sampleOrder(result.Samples)
	cost, err := measuredCost(*result)
	if err != nil {
		result.CostFailure = err.Error()
	} else {
		result.Cost = &cost
	}
	budget, err := EvaluateBudget(result.Samples, result.CompilerCommands, result.InitialPreparationWallNanos, residentGuests, operatorPeak(*result))
	if err != nil {
		result.BudgetFailure = err.Error()
	} else {
		result.Budget = &budget
	}
}

func preparationProvenance(spec Spec) record.SHA256 {
	if spec.preparedTarget.Backend == nil {
		return ""
	}
	return spec.preparedTarget.Backend.Provenance.SHA256
}

func preparationCompiled(commands []backend.CompilerCommandMetrics) bool {
	for _, command := range commands {
		if len(command.Arguments) != 0 && (command.Arguments[0] == "build" || len(command.Arguments) > 1 && command.Arguments[0] == "test" && command.Arguments[1] == "-c") {
			return true
		}
	}
	return false
}

func operatorPeak(result Result) uint64 {
	var peak uint64
	for _, usage := range []*hostexec.ResourceUsage{result.OperatorPreparationAfter, result.OperatorExecutionAfter} {
		if usage != nil {
			peak = max(peak, usage.PeakRSSBytes)
		}
	}
	return peak
}

func sampleOrder(samples []Sample) []string {
	order := make([]string, 0, len(samples))
	for _, sample := range samples {
		order = append(order, fmt.Sprintf("%s:%d:%s", sample.Phase, sample.Iteration, sample.Incarnation))
	}
	return order
}
