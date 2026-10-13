package native

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strconv"
	"time"

	"go.temporal.io/server/tools/gomad3/backends"
	nativebuild "go.temporal.io/server/tools/gomad3/backends/native/build"
	"go.temporal.io/server/tools/gomad3/backends/native/execution"
	"go.temporal.io/server/tools/gomad3/backends/native/preparation"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/captured"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/target"
	"go.temporal.io/server/tools/gomad3/world"
)

type Config struct {
	ToolchainRoot      string
	SupervisorCommand  []string
	CoordinatorCommand []string
	BootstrapCommand   []string
	RunnerBuild        string
}

type Provider struct {
	config    Config
	bootstrap func(transcript.Spec, target.Prepared, string, uint64) ([]byte, error)
}

func (provider *Provider) bootstrapFrame(profile transcript.Spec, prepared target.Prepared, runner string, seed uint64) ([]byte, error) {
	if provider.bootstrap != nil {
		return provider.bootstrap(profile, prepared, runner, seed)
	}
	return profile.BootstrapFrame(prepared, runner, seed)
}

func New(config Config) *Provider {
	config.SupervisorCommand = append([]string(nil), config.SupervisorCommand...)
	config.CoordinatorCommand = append([]string(nil), config.CoordinatorCommand...)
	config.BootstrapCommand = append([]string(nil), config.BootstrapCommand...)
	if len(config.BootstrapCommand) == 0 && len(config.SupervisorCommand) != 0 {
		config.BootstrapCommand = []string{config.SupervisorCommand[0], "__target_bootstrap"}
	}
	return &Provider{config: config}
}

func (*Provider) Capabilities() backends.Capabilities {
	return backends.Capabilities{
		Name: "native", ReplayModes: []string{record.ReplayExact, record.ReplayDiagnostic}, ChoiceProfiles: []string{choice.Profile},
		Simulation: true, ReadOnlyMounts: true, IOTranscript: true,
		Commands: []string{"explore", "plan", "execute-shard", "merge", "recover", "resume", "inspect", "replay", "minimize", "qualify", "analyze", "qualify-set", "merge-set", "compare-support", "doctor"},
	}
}

func (provider *Provider) Describe(ctx context.Context, config backends.Configuration) (backends.Description, error) {
	if err := ctx.Err(); err != nil {
		return backends.Description{}, err
	}
	if config.Target.SoftAssertionReportOnly {
		return backends.Description{}, errors.New("native provider does not support soft assertion report-only mode")
	}
	if config.ForExecution {
		if len(provider.config.SupervisorCommand) == 0 || provider.config.SupervisorCommand[0] == "" {
			return backends.Description{}, errors.New("supervisor command is required")
		}
		if len(provider.config.BootstrapCommand) == 0 || provider.config.BootstrapCommand[0] == "" {
			return backends.Description{}, errors.New("target bootstrap command is required")
		}
	}
	profile := transcript.Default()
	if config.Target.Backend != "" && config.Target.Backend != "native" {
		return backends.Description{}, fmt.Errorf("native provider does not support backend %q", config.Target.Backend)
	}
	if config.IOProfile != "" && config.IOProfile != profile.Name() {
		return backends.Description{}, fmt.Errorf("unknown native I/O profile %q", config.IOProfile)
	}
	switch config.ClockTick {
	case "", record.ClockTickStrict, record.ClockTickForward:
	default:
		return backends.Description{}, fmt.Errorf("clock tick policy %q must be strict or forward", config.ClockTick)
	}
	if _, err := transcript.MissingRequiredSemanticProbes(record.SemanticCoverage{}, config.RequiredSemanticProbes); err != nil {
		return backends.Description{}, err
	}
	version, boundary := transcript.BoundaryManifestIdentity()
	return backends.Description{
		Isolated: len(provider.config.CoordinatorCommand) != 0, RunnerBuild: provider.config.RunnerBuild, IOProfile: recordedProfile(profile), BoundaryVersion: version, BoundarySHA256: record.SHA256(boundary),
		SemanticInstrumentationSHA256: record.SHA256(transcript.SemanticInstrumentationIdentity()), SemanticProbes: transcript.RequiredSemanticProbes(),
	}, nil
}

func recordedProfile(profile transcript.Spec) record.IOProfile {
	return record.IOProfile{Name: profile.Name(), ImplementationSHA256: record.SHA256(profile.ImplementationSHA256()), Inventory: string(profile.Inventory()), InventorySHA256: record.SHA256(profile.InventorySHA256())}
}

func (provider *Provider) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	if err := ctx.Err(); err != nil {
		return target.Prepared{}, err
	}
	if spec.SoftAssertionReportOnly {
		return target.Prepared{}, errors.New("native provider does not support soft assertion report-only mode")
	}
	if spec.Backend != "" && spec.Backend != "native" {
		return target.Prepared{}, fmt.Errorf("native provider does not support backend %q", spec.Backend)
	}
	spec = spec.Clone()
	spec.Backend = ""
	if spec.ToolchainRoot == "" {
		spec.ToolchainRoot = provider.config.ToolchainRoot
	}
	prepared, err := preparation.Prepare(ctx, preparation.Request{Target: spec})
	if err != nil {
		return target.Prepared{}, &backends.PreparationError{Stage: string(preparation.StageOf(err)), Err: err}
	}
	return prepared, nil
}

func (provider *Provider) ValidatePrepared(spec target.Spec, prepared target.Prepared, environment []string) error {
	if err := nativebuild.VerifyPrepared(prepared); err != nil {
		return err
	}
	root := spec.ToolchainRoot
	if root == "" {
		root = provider.config.ToolchainRoot
	}
	if root != "" {
		identity, err := nativebuild.ReadToolchainIdentity(root)
		if err != nil {
			return err
		}
		if identity.GoVersion != prepared.GoVersion || identity.BuildKey != prepared.BuildKey || identity.TargetGOOS != prepared.TargetGOOS || identity.TargetGOARCH != prepared.TargetGOARCH {
			return errors.New("prepared target toolchain identity does not match the pinned toolchain")
		}
	}
	if err := transcript.Default().ValidatePreparedTarget(spec, prepared, environment); err != nil {
		return err
	}
	if prepared.Backend != nil || len(prepared.BackendPayloads) != 0 {
		return errors.New("native preparation returned external backend metadata")
	}
	return nil
}

func (provider *Provider) Run(ctx context.Context, request backends.Request) (result backends.Result, retErr error) {
	if len(request.Target.Argv) == 0 {
		return backends.Result{}, errors.New("target arguments are required")
	}
	workDirectory, err := os.MkdirTemp("", "gomad3-native-")
	if err != nil {
		return backends.Result{}, err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(workDirectory); cleanupErr != nil {
			retErr = errors.Join(retErr, fmt.Errorf("remove native working directory: %w", cleanupErr))
		}
	}()
	environment := append([]string(nil), request.Environment...)
	environment = append(environment, "GOMADSEED="+strconv.FormatUint(request.Seed, 10), "TZ=UTC", "GOMAD3_IO_PROFILE="+transcript.Deterministic)
	execute := backends.ExecutionRequest{
		Target: request.Target, Seed: request.Seed, RunnerBuild: provider.config.RunnerBuild,
		Command: request.Target.Path, Args: append([]string(nil), request.Target.Argv[1:]...), Argv0: request.Target.Argv[0], Dir: workDirectory,
		Env: environment, ExecutionTimeout: request.Timeout, TerminateGrace: time.Second, OutputLimit: request.OutputBytes, Diagnostics: request.Diagnostics,
		World:      backends.WorldCapability{RecordLimit: world.MaximumRecordingBytes, TransitionLimit: world.MaximumRecordingBytes, Seed: request.Seed},
		IO:         &backends.IOCapability{Profile: recordedProfile(transcript.Default()), Transcript: &backends.IOTranscriptCapability{Limit: request.TranscriptBytes}, ReadOnlyMount: &backends.ReadOnlyMountCapability{Limits: captured.DefaultLimits()}},
		Simulation: &backends.SimulationCapability{Role: backends.SimulationRoleCoordinator},
	}
	if request.Choice != nil {
		execute.Choice = &backends.ChoiceCapability{Mode: request.Choice.Mode, Profile: choice.Profile, ImplementationSHA256: request.Choice.ExecutionIdentity.ImplementationSHA256, ExecutionIdentity: request.Choice.ExecutionIdentity, Limit: request.Choice.Limit, ReplayPlan: request.Choice.Tape}
	}
	observed, err := provider.Execute(ctx, execute)
	if err != nil {
		return backends.Result{}, err
	}
	return backends.Result{
		VirtualTimeElapsedNanos: observed.VirtualTimeElapsedNanos, Termination: backends.Termination(observed.Termination), ExitCode: observed.ExitCode, Message: observed.Signal,
		Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes, Evidence: observed.IOTranscript.Bytes,
		Reaped: observed.GroupGone, Cancelled: observed.Cancelled, WatchdogTimeout: observed.WatchdogTimeout,
		ChoiceTrace: observed.ChoiceTrace.Trace, DiagnosticTrace: observed.DiagnosticTrace, ImplementationSHA256: observed.ChoiceTrace.ImplementationSHA256,
	}, nil
}

func (provider *Provider) Execute(ctx context.Context, request backends.ExecutionRequest) (backends.ExecutionResult, error) {
	if request.Coordinator != nil {
		targetRequest := request
		targetRequest.Coordinator = nil
		if !reflect.DeepEqual(targetRequest, backends.ExecutionRequest{}) {
			err := errors.New("coordinator request cannot include target execution")
			if request.Coordinator.Stdout != nil {
				err = errors.Join(err, request.Coordinator.Stdout.Close())
			}
			return backends.ExecutionResult{CoordinatorErrorReason: "coordinator_encode"}, err
		}
		return provider.executeCoordinator(ctx, *request.Coordinator)
	}
	if err := ctx.Err(); err != nil {
		return backends.ExecutionResult{}, err
	}
	if len(provider.config.SupervisorCommand) == 0 || provider.config.SupervisorCommand[0] == "" {
		return backends.ExecutionResult{}, errors.New("supervisor command is required")
	}
	if len(provider.config.BootstrapCommand) == 0 || provider.config.BootstrapCommand[0] == "" {
		return backends.ExecutionResult{}, errors.New("target bootstrap command is required")
	}
	if request.IO != nil {
		identity := request.IO.Profile
		if !transcript.Default().MatchesRecorded(identity.Name, string(identity.ImplementationSHA256), string(identity.InventorySHA256), identity.Inventory) {
			return backends.ExecutionResult{}, errors.New("execution I/O profile identity does not match the native backend")
		}
	}
	if len(request.Target.Argv) == 0 {
		return backends.ExecutionResult{}, errors.New("prepared target arguments are required")
	}
	if request.Command != "" && request.Command != request.Target.Path {
		return backends.ExecutionResult{}, errors.New("execution command does not match the prepared target")
	}
	if request.Args != nil && !slices.Equal(request.Args, request.Target.Argv[1:]) {
		return backends.ExecutionResult{}, errors.New("execution arguments do not match the prepared target")
	}
	if request.Argv0 != "" && request.Argv0 != request.Target.Argv[0] {
		return backends.ExecutionResult{}, errors.New("execution argv0 does not match the prepared target")
	}
	if request.Target.Backend != nil || len(request.Target.BackendPayloads) != 0 {
		return backends.ExecutionResult{}, errors.New("native execution cannot accept external backend metadata")
	}
	if err := nativebuild.VerifyPrepared(request.Target); err != nil {
		return backends.ExecutionResult{}, err
	}
	native := execution.Spec{
		SupervisorCommand: append([]string(nil), provider.config.SupervisorCommand...), BootstrapCommand: append([]string(nil), provider.config.BootstrapCommand...),
		Command: request.Target.Path, Args: append([]string(nil), request.Target.Argv[1:]...), Argv0: request.Target.Argv[0], Dir: request.Dir, Env: append([]string(nil), request.Env...),
		ExecutionTimeout: request.ExecutionTimeout, TerminateGrace: request.TerminateGrace, OutputLimit: request.OutputLimit, Diagnostics: request.Diagnostics,
		World: execution.WorldCapability{RecordLimit: request.World.RecordLimit, TransitionLimit: request.World.TransitionLimit, Seed: request.World.Seed, ExpectedInitial: request.World.ExpectedInitial, ReplayPlan: request.World.ReplayPlan},
	}
	if request.StdoutHead != nil {
		native.StdoutHead = request.StdoutHead
	}
	if request.StderrHead != nil {
		native.StderrHead = request.StderrHead
	}
	if request.IO != nil {
		profile := transcript.Default()
		runnerBuild := request.RunnerBuild
		if runnerBuild == "" {
			runnerBuild = provider.config.RunnerBuild
		}
		frame, err := provider.bootstrapFrame(profile, request.Target, runnerBuild, request.Seed)
		if err != nil {
			return backends.ExecutionResult{}, err
		}
		native.IO = &execution.IOCapability{Config: frame}
		if capability := request.IO.Transcript; capability != nil {
			native.IO.Transcript = &execution.IOTranscriptCapability{Limit: capability.Limit, Replay: capability.Replay, Expected: capability.Expected}
		}
		if capability := request.IO.ReadOnlyMount; capability != nil {
			native.IO.ReadOnlyMount = &execution.ReadOnlyMountCapability{Mappings: capability.Mappings, Limits: capability.Limits, Replay: capability.Replay}
		}
	}
	if capability := request.Choice; capability != nil {
		native.Choice = &execution.ChoiceCapability{Mode: capability.Mode, Profile: capability.Profile, ImplementationSHA256: capability.ImplementationSHA256, ExecutionIdentity: capability.ExecutionIdentity, Limit: capability.Limit, ReplayPlan: capability.ReplayPlan}
	}
	if capability := request.Simulation; capability != nil {
		if capability.Role != backends.SimulationRoleCoordinator {
			return backends.ExecutionResult{}, errors.New("native target requires the coordinator simulation role")
		}
		native.Simulation = &execution.SimulationCapability{Role: execution.SimulationRoleCoordinator, ExplorationPlan: capability.ExplorationPlan, ExplorationRecordLimit: capability.ExplorationRecordLimit, ExplorationRecordCount: capability.ExplorationRecordCount}
	}
	observed, err := execution.Run(ctx, native)
	return executionResult(observed), err
}

func executionResult(observed execution.Result) backends.ExecutionResult {
	return backends.ExecutionResult{
		BackendEvidence: observed.BackendEvidence, Captured: observed.Captured, Termination: backends.ExecutionTermination(observed.Termination), ExitCode: observed.ExitCode, Signal: observed.Signal,
		WatchdogTimeout: observed.WatchdogTimeout, Cancelled: observed.Cancelled, Stdout: observed.Stdout, Stderr: observed.Stderr, PID: observed.PID, PGID: observed.PGID, GroupGone: observed.GroupGone,
		WorldRecord: observed.WorldRecord, IOTranscript: backends.Transcript{Bytes: observed.IOTranscript.Bytes, SHA256: observed.IOTranscript.SHA256, Records: observed.IOTranscript.Records, Complete: observed.IOTranscript.Complete, ReplayDivergence: observed.IOTranscript.ReplayDivergence},
		IOROMounts: observed.IOROMounts, DiagnosticTrace: observed.DiagnosticTrace,
		ChoiceTrace:       backends.ChoiceTrace{Profile: observed.ChoiceTrace.Profile, ImplementationSHA256: observed.ChoiceTrace.ImplementationSHA256, Limit: observed.ChoiceTrace.Limit, Trace: observed.ChoiceTrace.Trace, TapeSHA256: observed.ChoiceTrace.TapeSHA256, Decisions: observed.ChoiceTrace.Decisions},
		SimulationRecords: observed.SimulationRecords, VirtualTimeElapsedNanos: observed.VirtualTimeElapsedNanos,
	}
}

func (*Provider) Evidence(ctx context.Context, request backends.EvidenceRequest) (backends.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return backends.Evidence{}, err
	}
	if request.Transcript != nil && request.Probes != nil {
		return backends.Evidence{}, errors.New("semantic evidence must supply transcript or probes, not both")
	}
	var evidence backends.Evidence
	var err error
	if request.Transcript != nil {
		evidence.Coverage, err = transcript.DecodeSemanticCoverage(request.Transcript)
		if err != nil {
			return backends.Evidence{}, err
		}
		operations, err := transcript.DecodeTranscript(request.Transcript)
		if err != nil {
			return backends.Evidence{}, err
		}
		evidence.Operations = make([]backends.OperationOutcome, 0, len(operations))
		for _, operation := range operations {
			evidence.Operations = append(evidence.Operations, backends.OperationOutcome{Name: operation.Name, Result: operation.Result})
		}
	} else {
		evidence.Coverage, err = transcript.SummarizeSemanticProbes(request.Probes)
	}
	if err != nil {
		return backends.Evidence{}, err
	}
	evidence.MissingRequiredProbes, err = transcript.MissingRequiredSemanticProbes(evidence.Coverage, request.RequiredSemanticProbes)
	return evidence, err
}
