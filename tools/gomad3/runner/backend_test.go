package runner

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/artifact"
	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/inspect"
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/exploration"
	"go.temporal.io/server/tools/gomad3/target"
)

type boundaryProvider struct {
	result   backend.Result
	calls    int
	prepared target.Prepared
	request  backend.Request
	err      error
	isolated bool
}

func TestExternalVirtualTimeIsPreserved(t *testing.T) {
	provider := &boundaryProvider{result: backend.Result{Termination: backend.Exit, Reaped: true, Evidence: []byte("observations"), VirtualTimeElapsedNanos: 1234}}
	result, err := runBackend(t.Context(), provider, backend.Request{OutputBytes: 1024}, nil, nil)
	if err != nil || result.VirtualTimeElapsedNanos != 1234 {
		t.Fatalf("external virtual time lost: %+v %v", result, err)
	}
}

func (p *boundaryProvider) Prepare(context.Context, target.Spec) (target.Prepared, error) {
	return target.Prepared{}, nil
}
func (p *boundaryProvider) ValidatePrepared(target.Spec, target.Prepared, []string) error { return nil }
func (p *boundaryProvider) ValidateReplay(context.Context, *artifact.Opened) (target.Prepared, error) {
	return p.prepared, nil
}
func (p *boundaryProvider) Run(_ context.Context, request backend.Request) (backend.Result, error) {
	p.calls++
	p.request = request
	return p.result, p.err
}

func (p *boundaryProvider) Capabilities() backend.Capabilities {
	name := "fixture-cooperative"
	if p.prepared.Backend != nil {
		name = p.prepared.Backend.Name
	}
	return backend.Capabilities{Name: name, ReplayModes: []string{record.ReplayObserved, record.ReplayExact}, ChoiceProfiles: []string{choice.Profile}}
}

func (p *boundaryProvider) Describe(_ context.Context, config backend.Configuration) (backend.Description, error) {
	profile := record.IOProfile{Name: config.Target.Backend}
	if p.prepared.Backend != nil {
		profile = record.BackendIOProfile(*p.prepared.Backend)
	}
	return backend.Description{Isolated: p.isolated, IOProfile: profile}, nil
}

func (p *boundaryProvider) Execute(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	return executeBoundaryFixture(ctx, p, request)
}

func executeBoundaryFixture(ctx context.Context, provider backend.Provider, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	projected := backend.Request{Target: request.Target, Seed: request.Seed, Environment: request.Env, Timeout: request.ExecutionTimeout, OutputBytes: request.OutputLimit, Choice: backendChoiceRequest(request.Choice)}
	if request.IO != nil && request.IO.Transcript != nil {
		projected.TranscriptBytes = request.IO.Transcript.Limit
	}
	observed, err := provider.Run(ctx, projected)
	if err != nil {
		return backend.ExecutionResult{}, err
	}
	return backend.ProjectObservedResult(projected, observed, request.StdoutHead, request.StderrHead)
}

func (*boundaryProvider) Evidence(ctx context.Context, request backend.EvidenceRequest) (backend.Evidence, error) {
	return fixtureSemanticEvidence(ctx, request)
}

func fixtureSemanticEvidence(ctx context.Context, request backend.EvidenceRequest) (backend.Evidence, error) {
	if err := ctx.Err(); err != nil {
		return backend.Evidence{}, err
	}
	if request.Transcript != nil && request.Probes != nil {
		return backend.Evidence{}, errors.New("semantic evidence must supply transcript or probes, not both")
	}
	var evidence backend.Evidence
	var err error
	if request.Transcript != nil {
		evidence.Coverage, err = transcript.DecodeSemanticCoverage(request.Transcript)
		if err != nil {
			return backend.Evidence{}, err
		}
		operations, err := transcript.DecodeTranscript(request.Transcript)
		if err != nil {
			return backend.Evidence{}, err
		}
		evidence.Operations = make([]backend.OperationOutcome, 0, len(operations))
		for _, operation := range operations {
			evidence.Operations = append(evidence.Operations, backend.OperationOutcome{Name: operation.Name, Result: operation.Result})
		}
	} else {
		evidence.Coverage, err = transcript.SummarizeSemanticProbes(request.Probes)
	}
	if err != nil {
		return backend.Evidence{}, err
	}
	evidence.MissingRequiredProbes, err = transcript.MissingRequiredSemanticProbes(evidence.Coverage, request.RequiredSemanticProbes)
	return evidence, err
}

func TestExternalReplayForcesRecordedTapeAndComparesFinalObservations(t *testing.T) {
	prepared := newFakePreparer(t).prepared
	provenance := []byte("fixture")
	prepared.Backend = &record.BackendMetadata{Name: "fixture-cooperative", ReplayMode: record.ReplayExact, Provenance: record.BackendPayload{
		Schema: "fixture/v1", File: "backend/provenance.json", SHA256: record.HashBytes(provenance), Bytes: record.Uint64String(len(provenance)),
	}}
	prepared.BuildInfo.Settings = []record.BuildSetting{}
	prepared.CapabilityMode = target.CapabilityModeClosure
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.Target.Backend = "fixture-cooperative"
	config.Backend = &boundaryProvider{}
	config.ChoiceTraceLimit = 4096
	_, environment, err := validateConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	trace := completeChoiceTrace(t, prepared.BuildKey, 4096, []choice.Record{{Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0}})
	completion := runCompletion{job: runJob{seed: 7}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(1, "done", "")}
	completion.result.BackendEvidence = []byte("run")
	completion.result.ChoiceTrace = trace
	world := noneWorldBundle()
	manifest, err := manifestForRun(campaignRequestFromSpec(config), prepared, environment, completion, backend.Classify(completion.result, false, world.Manifest.Terminal), "fixture", world.Manifest, nil)
	if err != nil {
		t.Fatal(err)
	}
	metadata := manifest.Target.Backend
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: prepared.Path, Stdout: []byte("done"), ChoiceTrace: trace.Trace.Bytes, World: world.Payloads,
		BackendPayloads: []target.BackendPayload{{Reference: metadata.Provenance, Data: provenance}, {Reference: *metadata.Evidence, Data: []byte("run")}},
	})
	if err != nil {
		t.Fatal(err)
	}
	base := backend.Result{Termination: backend.Exit, Reaped: true, ExitCode: 1, Stdout: []byte("done"), Evidence: []byte("run"), ChoiceTrace: trace.Trace, ImplementationSHA256: trace.ImplementationSHA256}
	for _, test := range []struct {
		name           string
		change         func(*backend.Result)
		verifyOnly     bool
		wantMatch      bool
		wantStatus     string
		wantDivergence string
	}{
		{name: "exact", wantMatch: true, wantStatus: ChoiceReplayExact},
		{name: "verify", verifyOnly: true, wantStatus: ChoiceReplayAvailable},
		{name: "stdout", change: func(r *backend.Result) { r.Stdout = []byte("changed") }, wantStatus: ChoiceReplayExact, wantDivergence: "stdout.full_sha256"},
		{name: "evidence", change: func(r *backend.Result) { r.Evidence = []byte("changed") }, wantStatus: ChoiceReplayExact, wantDivergence: "backend.evidence.sha256"},
		{name: "choice", change: func(r *backend.Result) {
			changed := forceTestChoiceRank(r.ChoiceTrace.Records[0], 1)
			r.ChoiceTrace, err = choice.BuildTrace([]choice.Record{changed}, choice.TerminalComplete)
			if err != nil {
				t.Fatal(err)
			}
		}, wantStatus: ChoiceReplayDiverged, wantDivergence: "choice_profile.trace.sha256"},
		{name: "forced divergence", change: func(r *backend.Result) { r.Termination = backend.Divergence }, wantStatus: ChoiceReplayDiverged, wantDivergence: "choice_profile.divergence"},
		{name: "precise forced divergence", change: func(r *backend.Result) {
			r.Termination = backend.Divergence
			r.ChoiceDivergence = &choice.Divergence{Ordinal: 3, Reason: choice.DivergenceSite}
		}, wantStatus: ChoiceReplayDiverged, wantDivergence: "choice_profile.divergence.ordinal[3].site"},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed := base
			if test.change != nil {
				test.change(&observed)
			}
			provider := &boundaryProvider{prepared: prepared, result: observed}
			result, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: published.Path, VerifyOnly: test.verifyOnly})
			if err != nil {
				t.Fatal(err)
			}
			if result.Match != test.wantMatch || result.ChoiceReplayStatus != test.wantStatus || result.Divergence != test.wantDivergence || result.ObservedRepetition {
				t.Fatalf("external exact replay: %+v", result)
			}
			if test.verifyOnly {
				if provider.calls != 0 {
					t.Fatal("verification launched backend")
				}
				return
			}
			request := provider.request.Choice
			if provider.calls != 1 || request == nil || request.Mode != choice.ModeReplay || request.Tape == nil || len(request.Tape.Decisions) != 1 || provider.request.Diagnostics {
				t.Fatalf("exact tape was not supplied: %+v", provider.request)
			}
			if _, err := choice.ValidateReplayPlan(*request.Tape, request.ExecutionIdentity); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestExternalChoiceDivergencePreservesJoinedInfrastructureError(t *testing.T) {
	infrastructure := errors.New("helper cleanup failed")
	err := errors.Join(&backend.FailureError{Termination: backend.Divergence, ChoiceDivergence: &choice.Divergence{Ordinal: 3, Reason: choice.DivergenceSite}}, infrastructure)
	_, observed := runBackend(t.Context(), &boundaryProvider{err: err}, backend.Request{OutputBytes: 1024}, nil, nil)
	if !errors.Is(observed, infrastructure) || backendChoiceReplayDivergence(observed) {
		t.Fatalf("choice divergence hid infrastructure failure: %v", observed)
	}
}

func TestExternalBackendTransportPreservesPortableNativeRecordSemantics(t *testing.T) {
	for _, profile := range []string{"plain", "choices"} {
		t.Run(profile, func(t *testing.T) {
			fixture, err := os.ReadFile(filepath.Join("testdata", "diagnostic-identity-"+profile+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var retained struct {
				Artifact json.RawMessage `json:"artifact"`
			}
			if err := json.Unmarshal(fixture, &retained); err != nil {
				t.Fatal(err)
			}
			manifest, err := record.DecodeExecutionRecord(retained.Artifact)
			if err != nil {
				t.Fatal(err)
			}
			finalized, encoded, err := record.FinalizeExecutionRecord(manifest)
			if err != nil {
				t.Fatal(err)
			}
			roundTripped, err := record.DecodeExecutionRecord(encoded)
			if err != nil {
				t.Fatal(err)
			}
			if manifest.Target.Backend != nil || !reflect.DeepEqual(finalized, manifest) || !reflect.DeepEqual(roundTripped, manifest) {
				t.Fatal("external choice transport changed native artifact semantics or identities")
			}
		})
	}
}

func TestExternalExecutionDoesNotClassifyNonExitAsApplicationFailure(t *testing.T) {
	for _, termination := range []backend.Termination{backend.Capacity, backend.Unsupported, backend.Divergence, backend.Trap, backend.Infrastructure} {
		t.Run(string(termination), func(t *testing.T) {
			p := &boundaryProvider{result: backend.Result{Termination: termination, Message: "explicit bounded stop", ExitCode: 1, Reaped: true, Evidence: []byte("fixture")}}
			result, err := runBackend(t.Context(), p, backend.Request{OutputBytes: 1024}, nil, nil)
			var failure *backend.FailureError
			if !errors.As(err, &failure) || failure.Termination != termination || failure.Message != p.result.Message || result.Termination != "" {
				t.Fatalf("nonexit boundary: result=%#v err=%v", result, err)
			}
		})
	}
}

func TestExternalManifestUsesCompleteRecordedTargetSnapshot(t *testing.T) {
	preparer := newFakePreparer(t)
	prepared := preparer.prepared
	prepared.Backend = &record.BackendMetadata{Name: "fixture", ReplayMode: record.ReplayObserved, Provenance: record.BackendPayload{Schema: "fixture/v1", File: "backend/provenance.json", SHA256: record.HashBytes([]byte("source")), Bytes: 6}}
	prepared.BuildInfo.Settings = []record.BuildSetting{}
	prepared.CapabilityMode = target.CapabilityModeClosure
	config, _ := testConfig(t, preparer, &fakeExecutor{}, "1", PolicyFirst, 1)
	completion := runCompletion{job: runJob{seed: 1}, startedAt: time.Unix(1, 0), finishedAt: time.Unix(2, 0), result: processResult(1, "", "")}
	completion.result.BackendEvidence = []byte("run")
	manifest, err := manifestForRun(campaignRequestFromSpec(config), prepared, nil, completion, backend.Classification{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayObserved}, "run", record.World{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	expected := prepared.RecordTarget()
	expected.Backend = recordedBackend(prepared, completion.result)
	if !reflect.DeepEqual(manifest.Target, expected) {
		t.Fatalf("external recorded target differs: got %#v want %#v", manifest.Target, expected)
	}
}

func TestResumeRejectsChangedExternalProfileBeforePreparedTarget(t *testing.T) {
	metadata := &record.BackendMetadata{Name: "fixture", ReplayMode: record.ReplayObserved, Provenance: record.BackendPayload{Schema: "fixture/v1", File: "backend/provenance.json", SHA256: record.HashBytes([]byte("source")), Bytes: 6}}
	plan := campaign.CampaignPlan{RunnerBuild: "fixture", Prepared: campaign.PreparedTargetPlan{Target: record.Target{Backend: metadata}}}
	_, _, _, _, _, err := resumeConfiguration(campaignRequest{Backend: &boundaryProvider{}, RunnerBuild: "fixture"}, plan)
	if err == nil || !strings.Contains(err.Error(), "recorded I/O profile identity") {
		t.Fatalf("changed external profile did not fail first: %v", err)
	}
}

func TestIsolatedCampaignRejectsInvalidRequestBeforeProviderLaunch(t *testing.T) {
	provider := &boundaryProvider{isolated: true}
	_, err := Explore(t.Context(), CampaignSpec{Backend: provider})
	if err == nil || err.Error() != "invalid seed selection: empty input" || provider.calls != 0 {
		t.Fatalf("invalid isolated request was admitted: calls=%d err=%v", provider.calls, err)
	}
}

func TestExternalProviderRequiresBackendSelector(t *testing.T) {
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.Backend = &boundaryProvider{}
	if _, environment, err := validateConfig(config); err == nil || err.Error() != "backend provider requires an explicit backend selector" {
		t.Fatalf("external provider without selector: environment=%v err=%v", environment, err)
	}
	// Without a selector the external profile checks do not apply.
	config.ClockTick = record.ClockTickForward
	if _, _, err := validateConfig(config); err == nil || err.Error() != "backend provider requires an explicit backend selector" {
		t.Fatalf("external provider without selector: err=%v", err)
	}
}

func TestExternalCooperativeSeedAdmitsChoiceDiagnostics(t *testing.T) {
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.Target.Backend = "fixture-cooperative"
	config.Backend = &boundaryProvider{}
	config.ChoiceTraceLimit = 4096
	config.Diagnostics = true
	_, environment, err := validateConfig(config)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"GOMAD3_CHOICE_PROFILE": choice.Profile, choice.DiagnosticProfileEnvironment: choice.DiagnosticProfile}
	for _, entry := range environment {
		if value, ok := want[entry.Name]; ok && value == entry.Value {
			delete(want, entry.Name)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing recorded profiles: %v", want)
	}
}

func TestExternalChoiceExplorationAdmitsOnlySupportedCoverage(t *testing.T) {
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.Target.Backend = "fixture-cooperative"
	config.Backend = &boundaryProvider{}
	config.Strategy = StrategyChoiceExploration
	config.ChoiceTraceLimit = 4096
	config.MaxExecutions, config.MaxChoiceDepth, config.MaxExplorationBytes = 8, 4, 1<<20
	for _, coverage := range []CoverageMode{CoverageNone, CoverageChoice, CoverageSemantic, CoverageSemanticChoice} {
		t.Run(string(coverage), func(t *testing.T) {
			config.Coverage = coverage
			_, _, err := validateConfig(config)
			wantSupported := coverage == CoverageNone || coverage == CoverageChoice
			if (err == nil) != wantSupported {
				t.Fatalf("external coverage=%s supported=%v err=%v", coverage, wantSupported, err)
			}
		})
	}
	config.Coverage = CoverageNone
	config.Diagnostics = true
	if _, _, err := validateConfig(config); err == nil {
		t.Fatal("choice exploration silently admitted unsupported diagnostic retention")
	}
}

func TestExternalChoiceOutcomeBindsBackendEvidence(t *testing.T) {
	outcome := backend.Classification{Domain: "success", Termination: "exit"}
	result := processResult(0, "same output", "")
	native, err := explorationOutcomeSHA256(outcome, result, record.World{})
	if err != nil {
		t.Fatal(err)
	}
	result.BackendEvidence = []byte("first modeled outcome")
	first, err := explorationOutcomeSHA256(outcome, result, record.World{})
	if err != nil {
		t.Fatal(err)
	}
	result.BackendEvidence = []byte("second modeled outcome")
	second, err := explorationOutcomeSHA256(outcome, result, record.World{})
	if err != nil || first == second || first == native || second == native {
		t.Fatalf("backend effects disappeared from the outcome: native=%s first=%s second=%s err=%v", native, first, second, err)
	}
}

func TestChoiceOutcomeIdentityUsesACompleteStdlibProjection(t *testing.T) {
	exitCode := record.Uint64String(0)
	baseOutcome := backend.Classification{Domain: "success", Reason: "world_idle", Termination: "exit", ExitCode: &exitCode}
	baseResult := processResult(0, "stdout", "stderr")
	baseResult.IOTranscript.SHA256[0] = 1
	baseResult.BackendEvidence = []byte("backend evidence")
	baseWorld := record.World{
		Terminal: record.WorldTerminal{Kind: "idle", Detail: "settled"},
		Final:    record.WorldPayload{SemanticDigest: record.HashBytes([]byte("world final"))},
	}
	base, err := explorationOutcomeSHA256(baseOutcome, baseResult, baseWorld)
	if err != nil {
		t.Fatal(err)
	}
	projection := choiceOutcomeIdentityProjection{
		Domain: baseOutcome.Domain, Reason: baseOutcome.Reason, Termination: baseOutcome.Termination,
		ExitCode: baseOutcome.ExitCode, Signal: baseOutcome.Signal,
		StdoutSHA256: record.SHA256FromSum(baseResult.Stdout.FullSHA256), StderrSHA256: record.SHA256FromSum(baseResult.Stderr.FullSHA256),
		IOTranscriptSHA256: record.SHA256FromSum(baseResult.IOTranscript.SHA256), BackendSHA256: record.HashBytes(baseResult.BackendEvidence),
		WorldTerminal: baseWorld.Terminal, WorldFinal: baseWorld.Final.SemanticDigest,
	}
	encoded, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	if want := record.DomainHash("gomad3-choice-exploration-outcome/v1", encoded); base != want {
		t.Fatalf("outcome identity = %s, want stdlib projection %s", base, want)
	}

	for _, test := range []struct {
		name   string
		change func(*backend.Classification, *backend.ExecutionResult, *record.World)
	}{
		{name: "domain", change: func(value *backend.Classification, _ *backend.ExecutionResult, _ *record.World) {
			value.Domain = "target"
		}},
		{name: "reason", change: func(value *backend.Classification, _ *backend.ExecutionResult, _ *record.World) {
			value.Reason = "nonzero_exit"
		}},
		{name: "termination", change: func(value *backend.Classification, _ *backend.ExecutionResult, _ *record.World) {
			value.Termination = "signal"
		}},
		{name: "exit code", change: func(value *backend.Classification, _ *backend.ExecutionResult, _ *record.World) {
			changed := record.Uint64String(1)
			value.ExitCode = &changed
		}},
		{name: "signal", change: func(value *backend.Classification, _ *backend.ExecutionResult, _ *record.World) {
			changed := "SIGTERM"
			value.Signal = &changed
		}},
		{name: "stdout", change: func(_ *backend.Classification, value *backend.ExecutionResult, _ *record.World) {
			value.Stdout.FullSHA256[0]++
		}},
		{name: "stderr", change: func(_ *backend.Classification, value *backend.ExecutionResult, _ *record.World) {
			value.Stderr.FullSHA256[0]++
		}},
		{name: "I/O transcript", change: func(_ *backend.Classification, value *backend.ExecutionResult, _ *record.World) {
			value.IOTranscript.SHA256[0]++
		}},
		{name: "backend evidence", change: func(_ *backend.Classification, value *backend.ExecutionResult, _ *record.World) {
			value.BackendEvidence = []byte("changed backend evidence")
		}},
		{name: "World terminal kind", change: func(_ *backend.Classification, _ *backend.ExecutionResult, value *record.World) {
			value.Terminal.Kind = "deadlock"
		}},
		{name: "World terminal detail", change: func(_ *backend.Classification, _ *backend.ExecutionResult, value *record.World) {
			value.Terminal.Detail = "changed detail"
		}},
		{name: "World final", change: func(_ *backend.Classification, _ *backend.ExecutionResult, value *record.World) {
			value.Final.SemanticDigest = record.HashBytes([]byte("changed world final"))
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			outcome, result, worldRecord := baseOutcome, baseResult, baseWorld
			test.change(&outcome, &result, &worldRecord)
			changed, err := explorationOutcomeSHA256(outcome, result, worldRecord)
			if err != nil {
				t.Fatal(err)
			}
			if changed == base {
				t.Fatalf("outcome identity did not bind %s", test.name)
			}
		})
	}

	invalid := baseOutcome
	invalid.Reason = string([]byte{0xff})
	if _, err := explorationOutcomeSHA256(invalid, baseResult, baseWorld); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("invalid outcome string error = %v", err)
	}
}

func TestExternalExecutionValidatesChoiceAndDiagnosticTransport(t *testing.T) {
	prepared := newFakePreparer(t).prepared
	capability, err := choiceCapabilityForJob(campaignRequest{ChoiceTraceLimit: 4096}, prepared, runJob{seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	trace := completeChoiceTrace(t, prepared.BuildKey, 4096, []choice.Record{{Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2, Selected: 0}})
	trace.Trace.Summary.PeakGoroutines = 2
	diagnosticLimit, err := choice.DiagnosticLimit(4096)
	if err != nil {
		t.Fatal(err)
	}
	diagnostics, err := choice.BuildDiagnosticTrace([]choice.DiagnosticRecord{{Ordinal: 0, RunqDraws: 1}}, diagnosticLimit)
	if err != nil {
		t.Fatal(err)
	}
	base := backend.Result{Termination: backend.Exit, Reaped: true, Evidence: []byte("fixture"), ChoiceTrace: trace.Trace, DiagnosticTrace: diagnostics, ImplementationSHA256: capability.ImplementationSHA256}
	request := backend.Request{Target: prepared, OutputBytes: 1024, Choice: backendChoiceRequest(capability), Diagnostics: true}
	result, err := runBackend(t.Context(), &boundaryProvider{result: base}, request, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result.ChoiceTrace.Profile != choice.Profile || result.ChoiceTrace.Trace.SHA256 != trace.Trace.SHA256 || result.DiagnosticTrace.SHA256 != diagnostics.SHA256 || result.ChoiceTrace.Trace.Summary.PeakGoroutines != 2 {
		t.Fatalf("backend omitted validated choice or diagnostic transport: %+v", result)
	}
	for _, test := range []struct {
		name   string
		change func(*backend.Result)
	}{
		{"implementation", func(result *backend.Result) { result.ImplementationSHA256 = [32]byte{} }},
		{"missing trace", func(result *backend.Result) { result.ChoiceTrace = choice.Trace{} }},
		{"corrupt trace", func(result *backend.Result) { result.ChoiceTrace.Bytes = []byte("invalid") }},
		{"unterminated trace", func(result *backend.Result) { result.ChoiceTrace.Summary.Terminal = 0 }},
		{"missing diagnostics", func(result *backend.Result) { result.DiagnosticTrace = choice.DiagnosticTrace{} }},
		{"corrupt diagnostics", func(result *backend.Result) { result.DiagnosticTrace.Bytes = []byte("invalid") }},
		{"diagnostic count", func(result *backend.Result) {
			result.DiagnosticTrace, err = choice.BuildDiagnosticTrace(nil, diagnosticLimit)
			if err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed := base
			test.change(&observed)
			if _, err := runBackend(t.Context(), &boundaryProvider{result: observed}, request, nil, nil); err == nil {
				t.Fatal("invalid backend choice transport accepted")
			}
		})
	}
}

func TestRunnerHasNoConcreteBackendDependency(t *testing.T) {
	command := exec.CommandContext(t.Context(), "go", "list", "-deps", "-tags", "test_dep", ".")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("list Runner dependencies: %v: %s", err, output)
	}
	for _, path := range strings.Fields(string(output)) {
		if strings.HasPrefix(path, "go.temporal.io/server/tools/gomad3/backends/") {
			t.Errorf("Runner depends on concrete backend package %s", path)
		}
	}
}

type staticDescriptionMinimizationProvider struct{ *prefixFailureProvider }

func (provider *staticDescriptionMinimizationProvider) Describe(ctx context.Context, config backend.Configuration) (backend.Description, error) {
	description, err := provider.prefixFailureProvider.Describe(ctx, config)
	description.IOProfile.ImplementationSHA256 = record.HashBytes([]byte("static backend implementation"))
	return description, err
}

func TestExternalMinimizeUsesProvenanceProfileInsteadOfStaticDescription(t *testing.T) {
	parent, execution := backendMinimizationParent(t)
	provider := &staticDescriptionMinimizationProvider{prefixFailureProvider: execution}
	minimized, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent.Path, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !minimized.Changed || minimized.Attempts != 2 || minimized.Artifact.Manifest.IOProfile.ImplementationSHA256 != parent.Manifest.IOProfile.ImplementationSHA256 || minimized.Artifact.Manifest.Minimization.FinalForcedDecisions != 1 {
		t.Fatalf("provenance-bound minimization = %+v", minimized)
	}
	replayed, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: minimized.Artifact.Path})
	if err != nil || !replayed.Match || replayed.ChoiceReplayStatus != ChoiceReplayExact {
		t.Fatalf("external reduced exact replay = %+v, %v", replayed, err)
	}
}

type strategyRegressionProvider struct {
	backend.Provider
	preparer   *fakePreparer
	executor   executionRunner
	transcript backend.Transcript
}

func (provider *strategyRegressionProvider) Describe(ctx context.Context, config backend.Configuration) (backend.Description, error) {
	config.ForExecution = false
	return provider.Provider.Describe(ctx, config)
}

func (provider *strategyRegressionProvider) Prepare(ctx context.Context, spec target.Spec) (target.Prepared, error) {
	return provider.preparer.Prepare(ctx, spec)
}

func (provider *strategyRegressionProvider) ValidateReplay(ctx context.Context, _ *artifact.Opened) (target.Prepared, error) {
	if err := ctx.Err(); err != nil {
		return target.Prepared{}, err
	}
	return provider.preparer.prepared.CloneBackend(), nil
}

func (provider *strategyRegressionProvider) Execute(ctx context.Context, request backend.ExecutionRequest) (backend.ExecutionResult, error) {
	result, err := provider.executor.Run(ctx, request)
	if err == nil {
		result.IOTranscript = provider.transcript
	}
	return result, err
}

func TestExplorationStrategiesKeepExplicitProvider(t *testing.T) {
	for _, strategy := range []Strategy{StrategyChoiceExploration, StrategySimulationExploration} {
		t.Run(string(strategy), func(t *testing.T) {
			preparer := newFakePreparer(t)
			limit := choiceTraceLimit(t, 1)
			provider := &strategyRegressionProvider{Provider: scriptProvider(), preparer: preparer, transcript: semanticTranscript(t, "stdlib.os.openfile")}
			config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
			config.Backend, config.Strategy, config.ChoiceTraceLimit = provider, strategy, limit
			config.MaxExecutions, config.MaxExplorationBytes = 4, 1<<20
			if strategy == StrategyChoiceExploration {
				config.MaxChoiceDepth = 2
				provider.executor = &explorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
			} else {
				config.MaxForcedDecisions, config.MaxExplorationResultBytes = 2, 1<<20
				config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}
				provider.executor = &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit}
			}
			summary, err := Explore(t.Context(), config)
			if err != nil {
				t.Fatal(err)
			}
			if summary.Attempted != 2 || summary.Succeeded != 2 || summary.StopReason != StopExplorationExhausted {
				t.Fatalf("provider exploration = %+v", summary)
			}
			inspected, err := inspect.Inspect(summary.CampaignPath, inspect.InspectOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if len(inspected.Campaign.Executions) != 2 || inspected.Campaign.Executions[1].ParentCandidateSHA256 == "" {
				t.Fatalf("forced provider candidate was not committed: %+v", inspected.Campaign.Executions)
			}
		})
	}
}

func TestSimulationMinimizeKeepsProviderProfileAndExactReplay(t *testing.T) {
	preparer := newFakePreparer(t)
	limit := choiceTraceLimit(t, 1)
	provider := &strategyRegressionProvider{Provider: scriptProvider(), preparer: preparer,
		transcript: semanticTranscript(t, "stdlib.os.openfile"),
		executor:   &simulationExplorationExecutor{t: t, buildKey: preparer.prepared.BuildKey, limit: limit, fail: true}}
	config, _ := testConfig(t, nil, nil, "7", PolicyAll, 1)
	config.Backend, config.Strategy, config.ChoiceTraceLimit = provider, StrategySimulationExploration, limit
	config.MaxExecutions, config.MaxForcedDecisions = 4, 2
	config.MaxExplorationBytes, config.MaxExplorationResultBytes = 1<<20, 1<<20
	config.SimulationDimensionLimits = SimulationDimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}
	summary, err := Explore(t.Context(), config)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Attempted != 2 || summary.Failures != 2 || len(summary.Artifacts) != 1 {
		t.Fatalf("provider failure discovery = %+v", summary)
	}
	parent := providerForcedSimulationArtifact(t, provider, summary.Artifacts[0])
	preflight, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: parent, VerifyOnly: true})
	if err != nil || !preflight.Verified || preflight.ChoiceReplayStatus != ChoiceReplayAvailable {
		t.Fatalf("provider minimization preflight = %+v, %v", preflight, err)
	}
	minimized, err := Minimize(t.Context(), MinimizeSpec{Backend: provider, ArtifactPath: parent, OutputRoot: t.TempDir(), AttemptBudget: 8})
	if err != nil {
		t.Fatal(err)
	}
	if !minimized.Changed || minimized.Attempts == 0 || minimized.Artifact.Manifest.IOProfile.Name != "gomad3-deterministic/v1" || minimized.Artifact.Manifest.SimulationProfile == nil {
		t.Fatalf("provider minimization = %+v", minimized)
	}
	replayed, err := Replay(t.Context(), ReplaySpec{Backend: provider, ArtifactPath: minimized.Artifact.Path})
	if err != nil || !replayed.Match || replayed.ChoiceReplayStatus != ChoiceReplayExact {
		t.Fatalf("minimized provider exact replay = %+v, %v", replayed, err)
	}
}

func providerForcedSimulationArtifact(t *testing.T, provider *strategyRegressionProvider, root string) string {
	t.Helper()
	opened, err := artifact.OpenArtifact(root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := opened.Close(); err != nil {
			t.Error(err)
		}
	}()
	manifest := opened.Manifest()
	profile := manifest.SimulationProfile
	observedRecord, err := opened.ReadPayload(profile.Record.File, uint64(profile.Record.Bytes))
	if err != nil {
		t.Fatal(err)
	}
	capability, _, err := choiceCapabilityForArtifact(opened)
	if err != nil {
		t.Fatal(err)
	}
	config := exploration.SimulationConfig{ExecutionSHA256: profile.ExecutionSHA256, ControllerSHA256: profile.ControllerSHA256,
		BaseSeed: uint64(manifest.Seed), Parallel: 1, MaxExecutions: 4, MaxForcedDecisions: 2,
		MaxExplorationBytes: 1 << 20, MaxResultBytes: 1 << 20, FailureBudget: 4,
		Limits: exploration.DimensionLimits{Runtime: 2, Scenario: 2, Network: 2, Storage: 2, Fault: 2, Crash: 2}}
	rootCandidate, err := exploration.CanonicalCandidate(config, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if rootCandidate.SHA256 != profile.CandidateSHA256 {
		t.Fatal("fixture root candidate identity changed")
	}
	rootResult, err := exploration.ResultForRecord(config, rootCandidate, observedRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	forced, err := exploration.ForceDecision(rootResult.Decisions[0], 1)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := exploration.CanonicalCandidate(config, []exploration.ForcedDecision{forced}, rootCandidate.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	execution, err := exploration.ExecutionForCandidate(config, candidate, capability.ExecutionIdentity)
	if err != nil {
		t.Fatal(err)
	}
	request := backend.ExecutionRequest{Target: provider.preparer.prepared, Seed: uint64(manifest.Seed),
		Choice: capability, Simulation: &backend.SimulationCapability{Role: backend.SimulationRoleCoordinator,
			ExplorationPlan: execution.SimulationPlan, ExplorationRecordLimit: uint64(profile.Record.Limit), ExplorationRecordCount: 1}}
	observed, err := provider.Execute(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	retained, err := exploration.ProjectArtifact(config, candidate, execution.SimulationPlan, observed.SimulationRecords[0], nil, uint64(profile.Record.Limit))
	if err != nil {
		t.Fatal(err)
	}
	manifest.SimulationProfile = &retained
	world, err := readWorldPayloads(opened)
	if err != nil {
		t.Fatal(err)
	}
	published, err := artifact.PublishArtifact(artifact.Store{Root: t.TempDir()}, artifact.ArtifactInput{
		Manifest: manifest, TargetPath: provider.preparer.prepared.Path, World: world,
		Stdout: observed.Stdout.Bytes, Stderr: observed.Stderr.Bytes, IOTranscript: observed.IOTranscript.Bytes, ChoiceTrace: observed.ChoiceTrace.Trace.Bytes,
		Simulation: &artifact.SimulationPayloads{Plan: execution.SimulationPlan, Record: observed.SimulationRecords[0]},
	})
	if err != nil {
		t.Fatal(err)
	}
	return published.Path
}
