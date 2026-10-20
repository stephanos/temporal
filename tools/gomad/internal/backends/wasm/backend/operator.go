package backend

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends"
	simulation "go.temporal.io/server/tools/gomad/internal/backends/wasm/cluster"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	model "go.temporal.io/server/tools/gomad/internal/model/cluster"
)

func (p *Provider) normalizeScenario(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}
	if p.options.Config.Profile != wasi.CooperativeProfile {
		return nil, errors.New("typed WASM scenarios require the cooperative profile")
	}
	if p.options.Config.Clock.ReadStepNanos != 0 {
		return nil, errors.New("typed WASM scenarios require a frozen clock")
	}
	s, canonical, err := simulation.NormalizeOperator(data)
	if err != nil {
		return nil, err
	}
	config := p.options.Config
	config.Args = []string{"gomad-target"}
	_, _, err = s.Bind(wasi.Request{Config: config, Runtime: &wasi.RuntimeControl{Choice: &backends.ChoiceRequest{}}})
	if err != nil {
		return nil, err
	}
	return canonical, nil
}

func (p *Provider) executeScenario(ctx context.Context, request backends.ExecutionRequest, pr provenance) (backends.ExecutionResult, error) {
	converted, err := observedRequest(request)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	if converted.Choice != nil && converted.Choice.Mode != choice.ModeRecord {
		return backends.ExecutionResult{}, errors.New("typed WASM scenarios support seeded recording only")
	}
	if err := request.Target.ValidateBackendPayloads(); err != nil {
		return backends.ExecutionResult{}, err
	}
	if err := p.validateProvenance(pr, request.Target); err != nil {
		return backends.ExecutionResult{}, err
	}
	if err := request.Target.Verify(); err != nil {
		return backends.ExecutionResult{}, err
	}
	s, _, err := simulation.NormalizeOperator(pr.Scenario)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	implementation, err := choice.ImplementationIdentity(pr.BuildKey)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	identity := choice.ExecutionIdentity{ToolchainBuildKey: pr.BuildKey, GOOS: "wasip1", GOARCH: "wasm", ImplementationSHA256: implementation}
	module, err := hex.DecodeString(strings.TrimPrefix(pr.ModuleSHA256, "sha256:"))
	if err != nil || len(module) != len(identity.TargetSHA256) {
		return backends.ExecutionResult{}, errors.New("invalid WASM scenario module identity")
	}
	copy(identity.TargetSHA256[:], module)
	limit := uint64(choice.MaximumTraceBytes)
	if converted.Choice != nil {
		limit = converted.Choice.Limit
	}
	config := pr.Config
	config.Args = append([]string(nil), request.Target.Argv...)
	config.Environment = scenarioEnvironment(config.Environment, request.Env, config.WorkingDirectory)
	if converted.OutputBytes > 0 {
		config.Limits.OutputBytes = converted.OutputBytes
	}
	if converted.TranscriptBytes > 0 {
		config.Limits.TranscriptBytes = converted.TranscriptBytes
	}
	base := wasi.Request{WorkloadKind: request.Target.Kind, HelperPath: p.options.HelperPath, HelperSHA256: strings.TrimPrefix(pr.HelperSHA256, "sha256:"), ModulePath: request.Target.Path, ModuleSHA256: strings.TrimPrefix(pr.ModuleSHA256, "sha256:"), Engine: pr.Engine, Imports: pr.Imports, InitialMemoryPages: pr.InitialMemoryPages, MemoryBytes: pr.MemoryBytes, Fuel: pr.Fuel, Config: config, Timeout: converted.Timeout, Runtime: &wasi.RuntimeControl{Seed: request.Seed, Choice: &backends.ChoiceRequest{Mode: choice.ModeRecord, ExecutionIdentity: identity, Limit: limit}, Diagnostics: request.Diagnostics}, CompileCache: p.options.CompileCacheRoot}
	if stress := p.options.StressExecution; stress != nil && stress.Diagnostics {
		base.Runtime.Diagnostics = true
	}
	if err := p.stressRequest(&base); err != nil {
		return backends.ExecutionResult{}, err
	}
	spec, provider, err := s.Bind(base)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	if stress := p.options.StressExecution; stress != nil && stress.BindScenario != nil {
		spec, err = p.bindStressScenario(base, spec, provider)
		if err != nil {
			return backends.ExecutionResult{}, err
		}
	}
	provider.Oracles = scenarioSoftAssertions(s, request.Target.SoftAssertionReportOnly)
	var scenarioErr error
	observed, err := model.RunWASM(ctx, spec, func(ctx context.Context, c model.Cluster) error { scenarioErr = s.Run(ctx, c); return scenarioErr }, provider)
	if p.options.ObserveResults != nil {
		p.options.ObserveResults(ExecutionObservation{Target: request.Target, Spec: &spec, Incarnations: provider.CompletedResults(), Cluster: provider.CompletedClusterResult()})
	}
	interrupted := func(cancelled, watchdog, reaped bool) (backends.ExecutionResult, error) {
		return backends.ProjectObservedResult(converted, backends.Result{Cancelled: cancelled, WatchdogTimeout: watchdog, Reaped: reaped}, request.StdoutHead, request.StderrHead)
	}
	var hostError *wasi.ClusterHostError
	if (errors.As(scenarioErr, &hostError) || scenarioErr == nil && errors.As(err, &hostError)) && (hostError.Cancelled || hostError.WatchdogTimeout) {
		return interrupted(hostError.Cancelled, hostError.WatchdogTimeout, hostError.GroupGone)
	}
	if host := provider.CompletedClusterResult(); host != nil {
		if host.Cancelled || host.WatchdogTimeout {
			return interrupted(host.Cancelled, host.WatchdogTimeout, host.GroupGone)
		}
		if !host.GroupGone {
			return backends.ExecutionResult{}, &backends.FailureError{Termination: backends.Infrastructure, Message: "WASM cluster helper was not reaped"}
		}
	}
	if err != nil {
		if ctx.Err() != nil {
			results := provider.CompletedSummaries()
			reaped := len(results) > 0
			for _, r := range results {
				reaped = reaped && r.Reaped
			}
			return interrupted(true, false, reaped)
		}
		var divergence *model.ReplayDivergenceError
		if scenarioErr == nil && hostError == nil && !errors.As(err, &divergence) && errors.Is(err, choice.ErrInvalidReplayPlan) {
			for _, incarnation := range provider.CompletedResults() {
				if r := incarnation.Result; r.Termination == string(backends.Capacity) {
					return backends.ExecutionResult{}, &backends.FailureError{Termination: backends.Capacity, Message: r.Message}
				}
			}
		}
		return backends.ExecutionResult{}, scenarioFailure(err)
	}
	if failure := scenarioFailure(scenarioErr); failure != nil {
		return backends.ExecutionResult{}, failure
	}
	if observed.Outcome == model.OutcomeReplayDiverged {
		return backends.ExecutionResult{}, &backends.FailureError{Termination: backends.Divergence, Message: observed.Reason}
	}
	for _, incarnation := range provider.CompletedResults() {
		r := incarnation.Result
		if r.Cancelled {
			return interrupted(true, false, r.Reaped)
		}
		if r.WatchdogTimeout {
			return interrupted(false, true, r.Reaped)
		}
		switch backends.Termination(r.Termination) {
		case "invalid":
			return backends.ExecutionResult{}, &backends.FailureError{Termination: backends.Infrastructure, Message: r.Message}
		case backends.Capacity, backends.Unsupported, backends.Infrastructure, backends.Divergence:
			return backends.ExecutionResult{}, &backends.FailureError{Termination: backends.Termination(r.Termination), Message: r.Message}
		case backends.Exit, backends.Trap:
		}
	}
	payload, err := provider.RetentionPayload(spec, observed.Record)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	result := backends.ExecutionResult{Captured: true, GroupGone: true, Termination: backends.TerminationExit, BackendEvidence: payload.Data, BackendEvidenceSchema: payload.Reference.Schema}
	if observed.WASM != nil {
		result.VirtualTimeElapsedNanos = observed.WASM.VirtualElapsedNanos
	}
	if observed.Outcome != model.OutcomeCompleted {
		result.ExitCode = 1
	}
	for _, node := range observed.Record.Nodes {
		if node.State == model.NodeStateFailed {
			result.ExitCode = 1
		}
	}
	var stdout, stderr []byte
	for _, output := range observed.Outputs {
		if output.Stream == model.OutputStdout {
			stdout = append(stdout, output.Bytes...)
		} else {
			stderr = append(stderr, output.Bytes...)
		}
	}
	projected, err := backends.ProjectObservedResult(backends.Request{OutputBytes: config.Limits.OutputBytes}, backends.Result{Termination: backends.Exit, ExitCode: result.ExitCode, Stdout: stdout, Stderr: stderr, Evidence: payload.Data, Reaped: true}, request.StdoutHead, request.StderrHead)
	if err != nil {
		return backends.ExecutionResult{}, err
	}
	result.Stdout, result.Stderr = projected.Stdout, projected.Stderr
	result.SoftAssertions = observeSoftAssertions(result, request.Target.SoftAssertionReportOnly)
	return result, nil
}

func scenarioFailure(err error) error {
	if err == nil {
		return nil
	}
	termination := backends.Infrastructure
	var boundary *wasi.BoundaryError
	var divergence *model.ReplayDivergenceError
	var runtimeDivergence *wasi.RuntimeReplayError
	var failure *backends.FailureError
	if errors.As(err, &failure) {
		return err
	}
	if errors.Is(err, model.ErrCapacity) {
		termination = backends.Capacity
	}
	if errors.As(err, &boundary) {
		switch boundary.Kind {
		case "capacity":
			termination = backends.Capacity
		case "unsupported":
			termination = backends.Unsupported
		case "divergence":
			termination = backends.Divergence
		}
	}
	if errors.As(err, &divergence) || errors.As(err, &runtimeDivergence) {
		termination = backends.Divergence
	}
	return &backends.FailureError{Termination: termination, Message: err.Error()}
}

func scenarioSoftAssertions(s simulation.OperatorScenario, reportOnly bool) *simulation.OraclePlan {
	if !reportOnly {
		return nil
	}
	plan := &simulation.OraclePlan{Schema: simulation.SoftAssertionObservationSchema, MaximumBytes: s.Spec.Limits.ScenarioEvidenceBytes}
	for _, action := range s.Actions {
		if action.Action == "start" || action.Action == "restart" {
			plan.SoftAssertions = append(plan.SoftAssertions, simulation.SoftAssertionPolicy{Handle: action.Handle, ReportOnly: true})
		}
	}
	return plan
}

func scenarioEnvironment(configured, requested []string, directory string) []string {
	var result []string
	indices := map[string]int{}
	for _, values := range [][]string{configured, requested, {"PWD=" + directory}} {
		for _, entry := range values {
			name, _, _ := strings.Cut(entry, "=")
			if i, ok := indices[name]; ok {
				result[i] = entry
			} else {
				indices[name] = len(result)
				result = append(result, entry)
			}
		}
	}
	return result
}
