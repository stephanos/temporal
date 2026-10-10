package runner

import (
	"bytes"
	"context"
	"encoding/json"
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
	"go.temporal.io/server/tools/gomad3/runner/internal/campaign"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/target"
)

type boundaryProvider struct {
	result   backend.Result
	calls    int
	prepared target.Prepared
	request  backend.Request
	err      error
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
	manifest, err := manifestForRun(campaignRequestFromSpec(config), prepared, environment, completion, execution.Classify(completion.result, false, world.Manifest.Terminal), "fixture", world.Manifest, nil)
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

func TestExternalBackendTransportPreservesPortableNativeCanonicalBytes(t *testing.T) {
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
			if manifest.Target.Backend != nil || finalized.RecordHash != manifest.RecordHash || finalized.Outcome.FailureSignature != manifest.Outcome.FailureSignature || !bytes.Equal(encoded, retained.Artifact) {
				t.Fatal("external choice transport changed native canonical artifact bytes or identities")
			}
		})
	}
}

func TestExternalExecutionDoesNotClassifyNonExitAsApplicationFailure(t *testing.T) {
	for _, termination := range []backend.Termination{backend.Capacity, backend.Unsupported, backend.Divergence, backend.Trap, backend.Infrastructure} {
		t.Run(string(termination), func(t *testing.T) {
			p := &boundaryProvider{result: backend.Result{Termination: termination, ExitCode: 1, Reaped: true, Evidence: []byte("fixture")}}
			result, err := runBackend(t.Context(), p, backend.Request{OutputBytes: 1024}, nil, nil)
			var failure *backend.FailureError
			if !errors.As(err, &failure) || failure.Termination != termination || result.Termination != "" {
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
	manifest, err := manifestForRun(campaignRequestFromSpec(config), prepared, nil, completion, execution.Classification{Domain: "target", Reason: "nonzero_exit", Termination: "exit", ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayObserved}, "run", record.World{}, nil)
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
	_, _, _, _, _, err := resumeConfiguration(campaignRequest{RunnerBuild: "fixture"}, plan)
	if err == nil || !strings.Contains(err.Error(), "recorded I/O profile identity") {
		t.Fatalf("changed external profile did not fail first: %v", err)
	}
}

func TestIsolatedCampaignRejectsBackendBeforeLaunch(t *testing.T) {
	provider := &boundaryProvider{}
	_, err := Explore(t.Context(), CampaignSpec{Backend: provider, CoordinatorCommand: []string{"must-not-launch"}})
	if err == nil || !strings.Contains(err.Error(), "isolated Runner does not accept injected") || provider.calls != 0 {
		t.Fatalf("isolated backend was admitted: calls=%d err=%v", provider.calls, err)
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
