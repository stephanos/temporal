package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/backend"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

type prefixFailureProvider struct {
	boundaryProvider
	controls       []int
	replayMismatch bool
	environment    []string
}

type cancellationMinimizationProvider struct {
	*prefixFailureProvider
	cancel      context.CancelFunc
	cancelAt    int
	validations int
	observed    error
}

func (p *cancellationMinimizationProvider) ValidateReplay(ctx context.Context, _ *artifact.Opened) (target.Prepared, error) {
	p.validations++
	if p.validations == p.cancelAt {
		p.cancel()
	}
	p.observed = ctx.Err()
	return p.prepared, p.observed
}

func TestBackendMinimizePreservesCancellationThroughRetainedPreflightAndPreparation(t *testing.T) {
	for _, cancelAt := range []int{1, 2} {
		t.Run([]string{"preflight", "preparation"}[cancelAt-1], func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			observed := &cancellationMinimizationProvider{prefixFailureProvider: provider, cancel: cancel, cancelAt: cancelAt}
			_, err := Minimize(ctx, MinimizeSpec{Backend: observed, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if !errors.Is(err, context.Canceled) || !errors.Is(observed.observed, context.Canceled) || observed.validations != cancelAt || len(provider.controls) != 0 {
				t.Fatalf("retained validation lost caller cancellation: err=%v observed=%v validations=%d controls=%v", err, observed.observed, observed.validations, provider.controls)
			}
		})
	}
}

func (p *prefixFailureProvider) Run(ctx context.Context, request backend.Request) (backend.Result, error) {
	if err := ctx.Err(); err != nil {
		return backend.Result{}, err
	}
	if p.err != nil {
		return backend.Result{}, p.err
	}
	if request.Choice == nil {
		return backend.Result{}, errors.New("missing choice controls")
	}
	length := 0
	if request.Choice.Tape != nil {
		length = len(request.Choice.Tape.Decisions)
	}
	p.controls = append(p.controls, length)
	if request.Target.Backend == nil || !reflect.DeepEqual(request.Target.BackendPayloads, p.prepared.BackendPayloads) || !reflect.DeepEqual(request.Target.Argv, p.prepared.Argv) || !reflect.DeepEqual(request.Environment, p.environment) || request.Seed != 7 || request.Timeout != time.Second || request.OutputBytes != 4096 || request.TranscriptBytes != 64<<20 {
		return backend.Result{}, errors.New("minimization changed retained backend inputs or limits")
	}
	result := p.result
	if p.replayMismatch && request.Choice.Mode == choice.ModeReplay {
		result.Evidence = []byte("different replay observations")
	}
	if length == 0 {
		result.ExitCode = 0
		result.Stdout = []byte("success")
	}
	return result, nil
}

func backendMinimizationParent(t *testing.T) (artifact.Artifact, *prefixFailureProvider) {
	t.Helper()
	prepared := newFakePreparer(t).prepared
	prepared.Argv = []string{"gomad3-target", "fixed", "arguments"}
	provenance := []byte("immutable model and faults")
	prepared.Backend = &record.BackendMetadata{Name: "fixture-cooperative", ReplayMode: record.ReplayExact, Provenance: record.BackendPayload{
		Schema: "fixture/v1", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance)),
	}}
	prepared.BackendPayloads = []target.BackendPayload{{Reference: prepared.Backend.Provenance, Data: provenance}}
	prepared.BuildInfo.Settings = []record.BuildSetting{}
	prepared.CapabilityMode = target.CapabilityModeClosure
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.ExecutionTimeout, config.OutputLimit, config.ChoiceTraceLimit = time.Second, 4096, 4096
	config.Target.Backend, config.Backend = "fixture-cooperative", &boundaryProvider{}
	_, environment, err := validateConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	trace := completeChoiceTrace(t, prepared.BuildKey, 4096, []choice.Record{
		{Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0},
		{Ordinal: 1, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0},
		{Ordinal: 2, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0},
	})
	completion := runCompletion{job: runJob{seed: 7}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(17, "failure", "")}
	completion.result.BackendEvidence, completion.result.ChoiceTrace = []byte("model observations"), trace
	world := noneWorldBundle()
	manifest, err := manifestForRun(campaignRequestFromSpec(config), prepared, environment, completion, execution.Classify(completion.result, false, world.Manifest.Terminal), "fixture", world.Manifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, executionArtifactInput(manifest, prepared, completion.result, nil, world))
	if err != nil {
		t.Fatal(err)
	}
	provider := &prefixFailureProvider{environment: replayEnvironment(manifest.Environment), boundaryProvider: boundaryProvider{prepared: prepared, result: backend.Result{
		Termination: backend.Exit, Reaped: true, ExitCode: 17, Stdout: []byte("failure"), Evidence: []byte("model observations"), ChoiceTrace: trace.Trace, ImplementationSHA256: trace.ImplementationSHA256,
	}}}
	return published, provider
}

func TestBackendMinimizePreservesEssentialPrefixAndDetachedEvidence(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Changed || result.Attempts != 2 || len(result.Accepted) != 1 || !reflect.DeepEqual(provider.controls, []int{0, 1, 3, 3}) {
		t.Fatalf("effective prefix order/reductions: result=%+v controls=%v", result, provider.controls)
	}
	evidence := result.Artifact.Manifest.Minimization
	if evidence == nil || evidence.Schema != "gomad3.minimization/choice-prefix/v1" || evidence.OriginalForcedDecisions != 3 || evidence.FinalForcedDecisions != 1 || evidence.Predicate.ChoiceReplay != "exact" || evidence.Predicate.SimulationReplay != "not_present" || result.Artifact.Manifest.SimulationProfile != nil {
		t.Fatalf("choice-only evidence: %+v", evidence)
	}
	replay, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: result.Artifact.Path})
	if err != nil || !replay.Match || replay.ChoiceReplayStatus != ChoiceReplayExact || result.Artifact.Manifest.Outcome.FailureSignature != parent.Manifest.Outcome.FailureSignature {
		t.Fatalf("fresh retained replay: %+v err=%v", replay, err)
	}
	opened, err := artifact.OpenArtifact(parent.Path)
	if err != nil {
		t.Fatal(err)
	}
	if opened.Manifest().RecordHash != parent.Manifest.RecordHash {
		t.Fatal("parent changed")
	}
	if err := opened.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestBackendMinimizeResumeUsesCommittedPrefixAndRetainedPayloads(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	baseline, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	provider.controls = nil
	root := filepath.Join(t.TempDir(), "minimized")
	spec := MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: root, AttemptBudget: 8}
	session, err := openMinimizationSession(t.Context(), spec, executionDependencies{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := session.close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := session.reduce(t.Context()); err != nil {
		t.Fatal(err)
	}
	if session.workspace.Checkpoint().Accepted == nil {
		t.Fatal("missing committed reduction")
	}
	if err := session.close(); err != nil {
		t.Fatal(err)
	}
	provider.controls, spec.Resume = nil, true
	resumed, err := Minimize(t.Context(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(minimizationOutcomeOf(baseline), minimizationOutcomeOf(resumed)) || !reflect.DeepEqual(provider.controls, []int{3}) {
		t.Fatalf("resume repeated committed work or lost evidence: %+v controls=%v", resumed, provider.controls)
	}
}

func TestBackendMinimizeBindsEffectiveControlsToRetainedTape(t *testing.T) {
	for _, name := range []string{"prefix digest", "removed identity"} {
		t.Run(name, func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if err != nil {
				t.Fatal(err)
			}
			manifest := result.Artifact.Manifest
			if name == "prefix digest" {
				manifest.Minimization.ChoicePrefix.Final.SHA256 = record.HashBytes([]byte("forged effective tape"))
				identity, err := manifest.Minimization.ChoicePrefix.Final.CandidateSHA256()
				if err != nil {
					t.Fatal(err)
				}
				manifest.Minimization.FinalCandidateSHA256 = identity
				manifest.Minimization.Accepted[0].AfterSHA256 = identity
			} else {
				manifest.Minimization.Accepted[0].Removed[0].Identity = record.HashBytes([]byte("not the removed choice"))
			}
			manifest, encoded, err := record.FinalizeExecutionRecord(manifest)
			if err != nil {
				t.Fatalf("self-consistent forged record rejected without tape: %v", err)
			}
			world := noneWorldBundle()
			payloads := append([]target.BackendPayload(nil), provider.prepared.BackendPayloads...)
			payloads = append(payloads, target.BackendPayload{Reference: *manifest.Target.Backend.Evidence, Data: provider.result.Evidence})
			_, err = artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{Manifest: manifest, TargetPath: provider.prepared.Path, Stdout: provider.result.Stdout, ChoiceTrace: provider.result.ChoiceTrace.Bytes, BackendPayloads: payloads, World: world.Payloads})
			if err == nil || !strings.Contains(err.Error(), "retained tape") {
				t.Fatalf("forged controls published: %v", err)
			}
			if err := os.WriteFile(filepath.Join(result.Artifact.Path, "manifest.json"), encoded, 0600); err != nil {
				t.Fatal(err)
			}
			opened, err := artifact.OpenArtifact(result.Artifact.Path)
			if opened != nil {
				_ = opened.Close()
			}
			if err == nil || !strings.Contains(err.Error(), "retained tape") {
				t.Fatalf("forged controls opened: %v", err)
			}
		})
	}
}

func TestBackendMinimizeRejectsInvalidEvidenceShape(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*record.Minimization)
	}{
		{"count", func(m *record.Minimization) { m.ChoicePrefix.Final.Length++ }},
		{"non-suffix", func(m *record.Minimization) { m.Accepted[0].Removed[0].Ordinal = 0 }},
		{"native schema", func(m *record.Minimization) { m.Schema = "gomad3.minimization/v1" }},
		{"fake simulation replay", func(m *record.Minimization) { m.Predicate.SimulationReplay = "exact" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opened, err := artifact.OpenArtifact(result.Artifact.Path)
			if err != nil {
				t.Fatal(err)
			}
			manifest := opened.Manifest()
			if err := opened.Close(); err != nil {
				t.Fatal(err)
			}
			tc.change(manifest.Minimization)
			if _, _, err := record.FinalizeExecutionRecord(manifest); err == nil {
				t.Fatal("invalid minimization evidence admitted")
			}
		})
	}
}

func TestBackendMinimizeInfrastructureFailureIsNotCandidateRejection(t *testing.T) {
	parent, provider := backendMinimizationParent(t)
	provider.err = errors.New("helper infrastructure unavailable")
	_, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if !errors.Is(err, provider.err) || len(provider.controls) != 0 {
		t.Fatalf("infrastructure error consumed attempt: %v controls=%v", err, provider.controls)
	}
}

func TestBackendMinimizeRejectsIncompatibleArtifactsBeforeExecution(t *testing.T) {
	for _, name := range []string{"observed", "success", "watchdog", "missing choices"} {
		t.Run(name, func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			world := noneWorldBundle()
			payloads := append([]target.BackendPayload(nil), provider.prepared.BackendPayloads...)
			payloads = append(payloads, target.BackendPayload{Reference: *parent.Manifest.Target.Backend.Evidence, Data: provider.result.Evidence})
			input := artifact.ArtifactInput{Manifest: parent.Manifest, TargetPath: provider.prepared.Path, Stdout: provider.result.Stdout, ChoiceTrace: provider.result.ChoiceTrace.Bytes, BackendPayloads: payloads, World: world.Payloads}
			switch name {
			case "observed":
				input.Manifest.Target.Backend.ReplayMode, input.Manifest.ReplayMode = record.ReplayObserved, record.ReplayObserved
				input.Manifest.ChoiceProfile, input.ChoiceTrace = nil, nil
			case "success":
				code := record.Uint64String(0)
				input.Manifest.ArtifactKind = record.ArtifactSuccess
				input.Manifest.Outcome.Domain, input.Manifest.Outcome.Reason, input.Manifest.Outcome.ExitCode = "success", "zero_exit", &code
			case "watchdog":
				deadline := "execution"
				input.Manifest.ArtifactKind, input.Manifest.ReplayMode = record.ArtifactWatchdogTimeout, record.ReplayDiagnostic
				input.Manifest.Outcome = record.Outcome{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline}
				input.Manifest.ChoiceProfile, input.ChoiceTrace = nil, nil
			case "missing choices":
				input.Manifest.ChoiceProfile, input.ChoiceTrace = nil, nil
			}
			if input.Manifest.ChoiceProfile == nil {
				input.Manifest.Limits.ChoiceTraceBytes = 0
				environment := make([]record.Environment, 0, len(input.Manifest.Environment))
				for _, entry := range input.Manifest.Environment {
					if entry.Name != "GOMAD3_CHOICE_PROFILE" {
						environment = append(environment, entry)
					}
				}
				input.Manifest.Environment = environment
			}
			incompatible, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, input)
			if err != nil {
				t.Fatal(err)
			}
			_, err = Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: incompatible.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if err == nil || len(provider.controls) != 0 {
				t.Fatalf("incompatible artifact executed: %v controls=%v", err, provider.controls)
			}
		})
	}
}

func TestBackendMinimizeRejectsChangedFailureOrReplayEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*prefixFailureProvider)
	}{
		{"outcome", func(p *prefixFailureProvider) { p.result.ExitCode = 18 }},
		{"output", func(p *prefixFailureProvider) { p.result.Stdout = []byte("other failure") }},
		{"backend observations", func(p *prefixFailureProvider) { p.result.Evidence = []byte("changed fixed model effects") }},
		{"fresh replay", func(p *prefixFailureProvider) { p.replayMismatch = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			parent, provider := backendMinimizationParent(t)
			tc.change(provider)
			result, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
			if err != nil || result.Changed || result.Artifact.Manifest.RecordHash != parent.Manifest.RecordHash || len(result.Accepted) != 0 {
				t.Fatalf("changed failure accepted: result=%+v err=%v", result, err)
			}
		})
	}
}
