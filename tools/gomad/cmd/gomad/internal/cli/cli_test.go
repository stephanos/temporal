package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
	qualificationworkload "go.temporal.io/server/tools/gomad/internal/backends/wasm/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/qualification"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
	"go.temporal.io/server/tools/gomad/internal/runner"
	"go.temporal.io/server/tools/gomad/internal/runner/inspect"
)

func TestByteSizeFlagParsesBinaryUnitsCanonically(t *testing.T) {
	for input, want := range map[string]uint64{"1": 1, "8KiB": 8 << 10, "8MiB": 8 << 20, "2GiB": 2 << 30} {
		var value byteSize
		if err := value.Set(input); err != nil {
			t.Fatalf("Set(%q): %v", input, err)
		}
		if uint64(value) != want {
			t.Fatalf("Set(%q) = %d, want %d", input, value, want)
		}
	}
	for _, input := range []string{"", "0", "1MB", "-1", "01", "18446744073709551615GiB"} {
		var value byteSize
		if err := value.Set(input); err == nil {
			t.Fatalf("Set(%q) succeeded", input)
		}
	}
}

func TestResolveExploreSeedsSupportsCountWithoutAmbiguity(t *testing.T) {
	for _, test := range []struct {
		name               string
		seeds              string
		count              uint64
		seedsSet, countSet bool
		want               string
		wantError          bool
	}{
		{name: "default", seeds: "1", want: "1"},
		{name: "explicit seeds", seeds: "7,9", seedsSet: true, want: "7,9"},
		{name: "one", seeds: "1", count: 1, countSet: true, want: "0"},
		{name: "three", seeds: "1", count: 3, countSet: true, want: "0-2"},
		{name: "zero", seeds: "1", countSet: true, wantError: true},
		{name: "conflict", seeds: "7", count: 3, seedsSet: true, countSet: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := resolveExploreSeeds(test.seeds, test.count, test.seedsSet, test.countSet)
			if (err != nil) != test.wantError || got != test.want {
				t.Fatalf("resolveExploreSeeds() = %q, %v, want %q, error=%t", got, err, test.want, test.wantError)
			}
		})
	}
}

func TestResolveExploreStrategyRequiresExplicitBoundedSingleSeedExploration(t *testing.T) {
	valid := exploreStrategyOptions{
		Value: "choice-exploration", Seeds: "7", MaxExecutions: 8, MaxChoiceDepth: 4, MaxExplorationBytes: 1 << 20,
		MaxExecutionsSet: true, MaxChoiceDepthSet: true, MaxExplorationBytesSet: true,
	}
	strategy, choices, err := resolveExploreStrategy(valid)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != runner.StrategyChoiceExploration || !choices {
		t.Fatalf("resolveExploreStrategy() = %q, %t", strategy, choices)
	}
	withStart := valid
	withStart.ChoiceStartOrdinalSet = true
	if _, _, err := resolveExploreStrategy(withStart); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		configure func(*exploreStrategyOptions)
		want      string
	}{
		{name: "count", configure: func(options *exploreStrategyOptions) { options.CountSet = true }, want: "does not accept --count"},
		{name: "multiple seeds", configure: func(options *exploreStrategyOptions) { options.Seeds = "7-8" }, want: "exactly one base seed"},
		{name: "guidance", configure: func(options *exploreStrategyOptions) { options.Guide = true }, want: "does not support --guide"},
		{name: "missing max executions", configure: func(options *exploreStrategyOptions) { options.MaxExecutionsSet = false }, want: "--max-executions"},
		{name: "zero max executions", configure: func(options *exploreStrategyOptions) { options.MaxExecutions = 0 }, want: "--max-executions"},
		{name: "missing max depth", configure: func(options *exploreStrategyOptions) { options.MaxChoiceDepthSet = false }, want: "--max-choice-depth"},
		{name: "missing exploration bytes", configure: func(options *exploreStrategyOptions) { options.MaxExplorationBytesSet = false }, want: "--max-exploration-bytes"},
		{name: "unknown", configure: func(options *exploreStrategyOptions) { options.Value = "random" }, want: "unknown exploration strategy"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := valid
			test.configure(&options)
			if _, _, err := resolveExploreStrategy(options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("resolveExploreStrategy() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestResolveExploreStrategyRequiresExplicitBoundedSimulationExploration(t *testing.T) {
	valid := exploreStrategyOptions{
		Value: "simulation-exploration", Seeds: "7", MaxExecutions: 8, MaxForcedDecisions: 4,
		MaxExplorationBytes: 1 << 20, MaxExplorationResultBytes: 1 << 20,
		SimulationDimensionLimits: runner.SimulationDimensionLimits{Runtime: 4, Scenario: 4, Network: 4, Storage: 4, Fault: 4, Crash: 4},
		MaxExecutionsSet:          true, MaxForcedDecisionsSet: true, MaxExplorationBytesSet: true, MaxExplorationResultBytesSet: true,
		RuntimeLimitSet: true, ScenarioLimitSet: true, NetworkLimitSet: true, StorageLimitSet: true, FaultLimitSet: true, CrashLimitSet: true,
	}
	strategy, choices, err := resolveExploreStrategy(valid)
	if err != nil {
		t.Fatal(err)
	}
	if strategy != runner.StrategySimulationExploration || !choices {
		t.Fatalf("resolveExploreStrategy() = %q, %t", strategy, choices)
	}

	for _, test := range []struct {
		name      string
		configure func(*exploreStrategyOptions)
		want      string
	}{
		{name: "count", configure: func(options *exploreStrategyOptions) { options.CountSet = true }, want: "does not accept --count"},
		{name: "multiple seeds", configure: func(options *exploreStrategyOptions) { options.Seeds = "7-8" }, want: "exactly one base seed"},
		{name: "guidance", configure: func(options *exploreStrategyOptions) { options.Guide = true }, want: "does not support --guide"},
		{name: "choice depth", configure: func(options *exploreStrategyOptions) { options.MaxChoiceDepthSet = true; options.MaxChoiceDepth = 1 }, want: "does not accept --max-choice-depth"},
		{name: "choice start ordinal", configure: func(options *exploreStrategyOptions) { options.ChoiceStartOrdinalSet = true }, want: "does not accept --choice-start-ordinal"},
		{name: "missing max executions", configure: func(options *exploreStrategyOptions) { options.MaxExecutionsSet = false }, want: "--max-executions"},
		{name: "missing forced decisions", configure: func(options *exploreStrategyOptions) { options.MaxForcedDecisionsSet = false }, want: "--max-forced-decisions"},
		{name: "missing exploration bytes", configure: func(options *exploreStrategyOptions) { options.MaxExplorationBytesSet = false }, want: "--max-exploration-bytes"},
		{name: "missing result bytes", configure: func(options *exploreStrategyOptions) { options.MaxExplorationResultBytesSet = false }, want: "--max-exploration-result-bytes"},
		{name: "missing runtime bound", configure: func(options *exploreStrategyOptions) { options.RuntimeLimitSet = false }, want: "--max-runtime-decisions"},
		{name: "zero crash bound", configure: func(options *exploreStrategyOptions) { options.SimulationDimensionLimits.Crash = 0 }, want: "--max-crash-decisions"},
	} {
		t.Run(test.name, func(t *testing.T) {
			options := valid
			test.configure(&options)
			if _, _, err := resolveExploreStrategy(options); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("resolveExploreStrategy() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestResolveExploreStrategyRejectsExplorationBoundsForSeeds(t *testing.T) {
	for _, options := range []exploreStrategyOptions{
		{Value: "seed", Seeds: "7", MaxExecutions: 1, MaxExecutionsSet: true},
		{Value: "seed", Seeds: "7", ChoiceStartOrdinalSet: true},
	} {
		if _, _, err := resolveExploreStrategy(options); err == nil || !strings.Contains(err.Error(), "require") || !strings.Contains(err.Error(), "--strategy=choice-exploration") {
			t.Fatalf("resolveExploreStrategy(%#v) error = %v", options, err)
		}
	}
}

func TestResolveExploreTypedErrorPresentation(t *testing.T) {
	for _, strategy := range []string{string(runner.StrategyChoiceExploration), string(runner.StrategySimulationExploration)} {
		_, _, err := resolveExploreStrategy(exploreStrategyOptions{Value: strategy, Seeds: "7,8"})
		want := "--strategy=" + strategy + " requires exactly one base seed"
		if err == nil || err.Error() != want {
			t.Fatalf("strategy %q error = %v, want %q", strategy, err, want)
		}
	}
	_, err := resolveExploreCoverage("none", []string{"stdlib.os.openfile"})
	if err == nil || err.Error() != "--require-probe requires --coverage=semantic" {
		t.Fatalf("coverage error = %v", err)
	}
}

func TestResolveExploreGuidanceEnablesSemanticCoverageAndRequiresCorpus(t *testing.T) {
	for _, test := range []struct {
		guide, coverageSet bool
		corpus, coverage   string
		want               string
		wantError          bool
	}{
		{guide: true, corpus: "/corpus", coverage: "none", want: "semantic"},
		{guide: true, corpus: "/corpus", coverage: "semantic", coverageSet: true, want: "semantic"},
		{guide: true, corpus: "/corpus", coverage: "choice", coverageSet: true, want: "choice"},
		{guide: true, corpus: "/corpus", coverage: "semantic+choice", coverageSet: true, want: "semantic+choice"},
		{guide: true, coverage: "none", wantError: true},
		{corpus: "/corpus", coverage: "none", wantError: true},
		{guide: true, corpus: "/corpus", coverage: "none", coverageSet: true, wantError: true},
	} {
		got, err := resolveExploreGuidance(test.guide, test.corpus, test.coverage, test.coverageSet)
		if (err != nil) != test.wantError || got != test.want {
			t.Fatalf("resolveExploreGuidance(%t, %q, %q, %t) = %q, %v", test.guide, test.corpus, test.coverage, test.coverageSet, got, err)
		}
	}
}

func TestResolveChoiceTraceRequiresEnablementAndBoundedCapacity(t *testing.T) {
	for _, test := range []struct {
		name      string
		enabled   bool
		limit     byteSize
		limitSet  bool
		want      uint64
		wantError bool
	}{
		{name: "disabled", limit: 8 << 20},
		{name: "enabled default", enabled: true, limit: 8 << 20, want: 8 << 20},
		{name: "enabled explicit", enabled: true, limit: 1 << 20, limitSet: true, want: 1 << 20},
		{name: "bytes without choices", limit: 1 << 20, limitSet: true, wantError: true},
		{name: "enabled zero", enabled: true, limitSet: true, wantError: true},
		{name: "too small", enabled: true, limit: 1, limitSet: true, wantError: true},
		{name: "too large", enabled: true, limit: 65 << 20, limitSet: true, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			observed, err := resolveChoiceTrace(test.enabled, test.limit, test.limitSet)
			if (err != nil) != test.wantError || observed != test.want {
				t.Fatalf("resolveChoiceTrace() = %d, %v, want %d error=%t", observed, err, test.want, test.wantError)
			}
		})
	}
}

func TestRunRejectsUnknownCommandWithUsageStatus(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := Run([]string{"unknown"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, stderr = %q", status, stderr.String())
	}
}

func TestParseTargetPreservesArgumentVector(t *testing.T) {
	spec, err := parseTarget([]string{"go-test", "./pkg", "--", "-test.run=Test Name", "literal;$value"})
	if err != nil {
		t.Fatal(err)
	}
	if spec.source != "./pkg" || len(spec.arguments) != 2 || spec.arguments[0] != "-test.run=Test Name" || spec.arguments[1] != "literal;$value" {
		t.Fatalf("target = %#v", spec)
	}
}

func TestParseCapabilityModeUsesClosedVocabulary(t *testing.T) {
	for _, value := range []string{"closure", "linked", "guarded"} {
		mode, err := parseCapabilityMode(value)
		if err != nil || string(mode) != value {
			t.Fatalf("parseCapabilityMode(%q) = %q, %v", value, mode, err)
		}
	}
	if _, err := parseCapabilityMode("auto"); err == nil {
		t.Fatal("parseCapabilityMode() accepted an unknown mode")
	}
}

// TestRunAnalyzeForwardsTargetAndClassifiesReport pins the target an analyze
// request reviews and the status each review, report, cleanup and output
// result maps to. Each row starts from dependencies that review the target
// and report it unsupported.

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

func TestRunInspectReportsBatchAsTextAndJSON(t *testing.T) {
	path := writeInspectBatchFixture(t)
	for _, test := range []struct {
		name      string
		arguments []string
		want      []string
	}{
		{name: "text", arguments: []string{path}, want: []string{"gomad inspect: kind=campaign", "run-inspect-command", "attempted=1", "seed=7 domain=success", "log: format=gomad.campaign-log/v1 frame-version=1 state=valid records=5 ", "tail=none tail-bytes=0 repaired-bytes=0", "committed: outcomes=1 "}},
		{name: "json", arguments: []string{"--json", path}, want: []string{`"schema":"gomad.inspect/v8"`, `"kind":"campaign"`, `"campaign_id":"run-inspect-command"`, `"log":{"format":"gomad.campaign-log/v1","frame_version":1,"state":"valid"`, `"committed":{"outcomes":1,`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := runInspect(test.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
				t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
			}
			for _, want := range test.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("inspect output = %q, missing %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestRunInspectRejectsChoicesForBatch(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := runInspect([]string{"--choices", writeInspectBatchFixture(t)}, &stdout, &stderr); status != 2 || !strings.Contains(stderr.String(), "traced artifact") {
		t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

func TestPrintInspectionReportsInterruptedSimulationExplorationWork(t *testing.T) {
	report := inspect.Inspection{
		Kind: "batch", Path: "/batch",
		Lifecycle: &inspect.CampaignLifecycleInspection{State: "running", Resumable: true},
		SimulationExploration: &inspect.SimulationExplorationInspection{
			Schema: "gomad.simulation-exploration-inspection/v1",
			Summary: inspect.SimulationExplorationSummary{
				MaxExecutions: 8, MaxForcedDecisions: 2, MaxExplorationBytes: 4096, MaxResultBytes: 2048,
				Pending: 1, PendingBytes: 512,
			},
			Pending: []inspect.SimulationCandidateInspection{{
				SHA256: "sha256:candidate", Overrides: []inspect.SimulationOverrideInspection{{
					Dimension: "fault", Ordinal: 0, Selected: 1, Alternatives: 2, Identity: "sha256:override", ControlBytes: 64, ControlSHA256: "sha256:control",
				}},
			}},
			StagedRound: &inspect.SimulationStagedRoundInspection{Index: 3, Candidates: 2, Attempted: 1},
		},
	}
	limits := &report.SimulationExploration.Summary.Limits
	limits.Runtime, limits.Scenario, limits.Network, limits.Storage, limits.Fault, limits.Crash = 1, 2, 3, 4, 5, 6
	var output bytes.Buffer
	if err := printInspection(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"simulation-exploration:", "pending=1", "runtime=1", "scenario=2", "network=3", "storage=4", "fault=5", "crash=6", "pending-candidate:", "forced-decision:", "staged-round: index=3 candidates=2 attempted=1"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("inspection output = %q, missing %q", output.String(), want)
		}
	}
}

func TestPrintInspectionReportsSimulationExplorationEvidence(t *testing.T) {
	report := inspect.Inspection{
		Kind: "artifact",
		Artifact: &inspect.ArtifactInspection{
			Simulation: &inspect.SimulationInspection{
				Profile: "gomad-simulation-exploration/v1", ControllerSHA256: "sha256:controller", ExecutionSHA256: "sha256:execution",
				CandidateSHA256: "sha256:candidate", OutcomeSHA256: "sha256:outcome", FailureSHA256: "sha256:failure",
				Plan:   inspect.SimulationPayloadInspection{Schema: "gomad.simulation-exploration-plan/v1", SHA256: "sha256:plan", Bytes: 123},
				Record: inspect.SimulationRecordInspection{Schema: "gomad.cluster-record/v8", SHA256: "sha256:record", Bytes: 456, Limit: 789},
			},
		},
	}
	var output bytes.Buffer
	if err := printInspection(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"simulation:", "controller=sha256:controller", "execution=sha256:execution", "candidate=sha256:candidate", "outcome=sha256:outcome", "failure=sha256:failure", "plan-bytes=123", "record-bytes=456", "record-limit=789"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("inspection output = %q, missing %q", output.String(), want)
		}
	}
}

func TestRunInspectReportsOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	if status := runInspect([]string{writeInspectBatchFixture(t)}, failingWriter{}, &stderr); status != 3 || !strings.Contains(stderr.String(), "write inspection report") {
		t.Fatalf("status=%d stderr=%q", status, stderr.String())
	}
}

func TestRunRecoverReportsStableTextAndJSON(t *testing.T) {
	for _, test := range []struct {
		name      string
		arguments []string
		want      []string
	}{
		{name: "text", arguments: []string{"/artifacts/v1/run-interrupted"}, want: []string{"gomad recover:", "action=restore-running", "changed=true", "state=recoverable-failure", "resumable=true"}},
		{name: "json", arguments: []string{"--json", "/artifacts/v1/run-interrupted"}, want: []string{`"schema":"gomad.recovery/v1"`, `"action":"restore-running"`, `"changed":true`, `"state":"recoverable-failure"`, `"resumable":true`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runRecoverWith(test.arguments, &stdout, &stderr, recoverDependencies{
				recover: func(context.Context, string) (runner.Recovery, error) {
					result := runner.Recovery{
						Schema: "gomad.recovery/v1", Path: "/artifacts/v1/run-interrupted", Action: "restore-running", Changed: true,
					}
					result.Before.State, result.Before.Repairable, result.Before.Action = "committing", true, "restore-running"
					result.After.State, result.After.LastStableState, result.After.Resumable = "recoverable-failure", "running", true
					return result, nil
				},
			})
			if status != 0 || stderr.Len() != 0 {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
			for _, want := range test.want {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("recover output = %q, missing %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestRunRecoverDistinguishesInvalidInputFromInfrastructureFailure(t *testing.T) {
	_, invalidErr := runner.Recover(context.Background(), t.TempDir())
	if invalidErr == nil || !runner.IsInvalidRecoveryError(invalidErr) {
		t.Fatalf("runner.Recover() error = %T %v, want invalid recovery error", invalidErr, invalidErr)
	}
	for _, test := range []struct {
		name       string
		recoverErr error
		wantStatus int
	}{
		{name: "invalid", recoverErr: invalidErr, wantStatus: 2},
		{name: "infrastructure", recoverErr: errors.New("disk unavailable"), wantStatus: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runRecoverWith([]string{"/artifacts/v1/run-interrupted"}, &stdout, &stderr, recoverDependencies{
				recover: func(context.Context, string) (runner.Recovery, error) {
					return runner.Recovery{}, test.recoverErr
				},
			})
			if status != test.wantStatus || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.recoverErr.Error()) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestRunRecoverRepairsPublishedBatchPrivateState(t *testing.T) {
	path := writeInspectBatchFixture(t)
	stale := filepath.Join(path, ".prepared")
	if err := os.MkdirAll(stale, 0o700); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if status := runRecover([]string{path}, &stdout, &stderr); status != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "action=finalize-publication") || !strings.Contains(stdout.String(), "changed=true") {
		t.Fatalf("status output stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	if _, err := os.Lstat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale private state remains: %v", err)
	}
}

// inspectFrame encodes one record as the Runner's log store frames it.
func inspectFrame(kind uint16, payload []byte) []byte {
	table := crc32.MakeTable(crc32.Castagnoli)
	frame := make([]byte, 16+len(payload)+4)
	copy(frame, "GMLG")
	frame[4] = 1
	binary.LittleEndian.PutUint16(frame[6:], kind)
	binary.LittleEndian.PutUint32(frame[8:], uint32(len(payload)))
	binary.LittleEndian.PutUint32(frame[12:], crc32.Checksum(frame[:12], table))
	copy(frame[16:], payload)
	binary.LittleEndian.PutUint32(frame[16+len(payload):], crc32.Update(crc32.Checksum(frame[:16], table), table, payload))
	return frame
}

// writeInspectLog writes log and an empty writer lock into the private store
// directory store.
func writeInspectLog(t *testing.T, store string, log []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(store, "log"), log, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(store, "lock"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
}

// writeInspectBatchFixture writes a published one-execution campaign as the
// campaign log the Runner writes: a private store holding a log of framed
// JSON records and its lock file.
func writeInspectBatchFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "run-inspect-command")
	store := filepath.Join(path, "store")
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	const execution = `{"selection_ordinal":"0","seed":"7","domain":"success","reason":"success","termination":"exit","failure_signature":null,"artifact":null,"elapsed_nanos":"5","io_transcript_sha256":null,"io_transcript_records":null}`
	records := []struct {
		kind    uint16
		payload string
	}{
		{1, `{"format":"gomad.campaign-log/v1","campaign_id":"run-inspect-command","strategy":"seed","selection":"7","selection_count":"1"}`},
		{2, `{"state":"prepared"}`},
		{2, `{"state":"running"}`},
		{4, `{"executions":[` + execution + `]}`},
		{5, fmt.Sprintf(`{"schema_version":%d,"schema":"gomad.campaign/v1","campaign_id":"run-inspect-command","strategy":"seed","selection":"7","selection_count":"1","attempted":"1","succeeded":"1","failures":"0","watchdogs":"0","cancelled":"0","distinct_failures":"0","stop_reason":"seeds_exhausted","journal":{"schema":"gomad.campaign-log/v1","records":"1","bytes":"%d"},"failure_signatures":[]}`, record.SchemaVersion, len(execution))},
	}
	var log []byte
	for _, entry := range records {
		log = append(log, inspectFrame(entry.kind, []byte(entry.payload))...)
	}
	writeInspectLog(t, store, log)
	return path
}

func TestExploreReporterEmitsStableJSONEventsAndEveryArtifact(t *testing.T) {
	var stdout, stderr bytes.Buffer
	reporter := newExploreReporter(true, &stdout, &stderr)
	if err := reporter.Progress(runner.CampaignEvent{
		Phase: runner.ProgressPreparing, CampaignPath: "/batch", Selected: 3,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Result(runner.CampaignResult{
		CampaignPath: "/batch", SelectionCount: 3, Attempted: 3, Succeeded: 1, Failures: 2, DistinctFailures: 2,
		StopReason: runner.StopSeedsExhausted, Artifacts: []string{"/batch/failures/one", "/batch/failures/two"},
		SemanticCoverage: &record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}); err != nil {
		t.Fatal(err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	for _, value := range []string{
		`"schema":"gomad.explore-event/v3"`, `"type":"progress"`, `"phase":"preparing"`,
		`"type":"result"`, `"classification":"target_failure"`, `"novelty":2`,
		`"semantic_coverage":{"schema":"gomad.semantic-coverage/v1","digest":"sha256:coverage","probes":["stdlib.os.openfile"]}`,
		`"path":"/batch/failures/one"`, `"path":"/batch/failures/two"`,
	} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("events = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestExploreReporterReportsRetainedSuccessfulRuns(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Result(runner.CampaignResult{
			CampaignPath: "/batch", SelectionCount: 1, Attempted: 1, Succeeded: 1, RetainedSuccesses: 1,
			RetainedSuccessBytes: 4096, SuccessArtifacts: []string{"/batch/successes/sha256-case"}, StopReason: runner.StopSeedsExhausted,
		}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"retained", "4096", "/batch/successes/sha256-case", "gomad replay"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("json=%t output = %q, missing %q", jsonOutput, stdout.String(), want)
			}
		}
		if stderr.Len() != 0 {
			t.Fatalf("json=%t stderr = %q", jsonOutput, stderr.String())
		}
	}
}

func TestExploreReporterReportsGuidedCorpusUpdates(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Result(runner.CampaignResult{
			CampaignPath: "/batch", SelectionCount: 4, Attempted: 4, Succeeded: 4, StopReason: runner.StopSeedsExhausted,
			CorpusPath: "/corpus", CorpusEntries: 12, CorpusAdded: 2,
		}); err != nil {
			t.Fatal(err)
		}
		for _, want := range []string{"/corpus", "12", "2"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("json=%t output = %q, missing %q", jsonOutput, stdout.String(), want)
			}
		}
	}
}

func TestExploreReporterReportsSimulationExplorationBoundsAndRemainingWork(t *testing.T) {
	exploration := &runner.SimulationExplorationSummary{
		Parallel: 2, MaxExecutions: 16, MaxForcedDecisions: 4, MaxExplorationBytes: 1 << 20, MaxResultBytes: 64 << 10,
		Limits:            runner.SimulationDimensionLimits{Runtime: 2, Scenario: 3, Network: 4, Storage: 5, Fault: 6, Crash: 7},
		LogicalExecutions: 5, CommittedRounds: 3, Pending: 4, PendingBytes: 2048, SeenCandidates: 9,
		DeduplicatedOutcomes: 3, DistinctFailures: 1, DeepestOverride: 2, OmittedByDimension: 8,
	}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		reporter := newExploreReporter(jsonOutput, &stdout, &stderr)
		if err := reporter.Result(runner.CampaignResult{
			CampaignPath: "/batch", SelectionCount: 1, Attempted: 5, Failures: 1,
			StopReason: runner.StopDimensionDepthComplete, SimulationExploration: exploration, RecoveryExecutions: 2,
		}); err != nil {
			t.Fatal(err)
		}
		explorationName := "simulation-exploration"
		if jsonOutput {
			explorationName = "simulation_exploration"
		}
		for _, want := range []string{explorationName, "pending", "2048", "runtime", "scenario", "network", "storage", "fault", "crash", "recovery"} {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("json=%t output = %q, missing %q", jsonOutput, stdout.String(), want)
			}
		}
		if stderr.Len() != 0 {
			t.Fatalf("json=%t stderr = %q", jsonOutput, stderr.String())
		}
	}
}

func TestRunExploreReportsFlagErrorsAsJSON(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := hostApplication().runExplore([]string{"--json", "--parallel", "invalid"}, &stdout, &stderr)
	if status != 2 || stderr.Len() != 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout.String(), stderr.String())
	}
	for _, value := range []string{`"schema":"gomad.explore-event/v3"`, `"type":"error"`, `"classification":"invalid_input"`} {
		if !strings.Contains(stdout.String(), value) {
			t.Fatalf("explore output = %q, missing %q", stdout.String(), value)
		}
	}
}

func TestExploreErrorReportsClassificationAfterChoiceDiagnosticWriterFailure(t *testing.T) {
	var stdout bytes.Buffer
	stderr := failingWriter{}
	reporter := newExploreReporter(true, &stdout, stderr)
	status := reportExploreFailure(runner.CampaignResult{ChoiceTrace: &record.ChoiceTraceSummary{}}, errors.New("bad request"), reporter, stderr)
	if status != 2 || !strings.Contains(stdout.String(), `"classification":"invalid_input"`) || !strings.Contains(stdout.String(), "bad request") {
		t.Fatalf("status=%d stdout=%q", status, stdout.String())
	}
}

// qualifyObservation is what one qualify request hands its dependencies: the
// toolchain roots it resolved, every Campaign and replay request, and the
// report it retained, if any. Every request that runs a Campaign retains a
// report.
type qualifyObservation struct {
	requested []string
	runs      []runner.CampaignSpec
	replays   []runner.ReplaySpec
	written   bool
	report    qualification.QualificationReport
}

// TestRunQualifyForwardsFlagsAndClassifiesOutcome pins, for each qualify
// request, the Campaign and replay requests it forwards, the report it
// retains, and the status and result event it reports.

// TestRunResumeForwardsCampaignAndClassifiesResult pins the resume request
// built from the stored Campaign and the installation, and how its result and
// errors are reported.
func TestRunResumeForwardsCampaignAndClassifiesResult(t *testing.T) {
	for _, test := range []struct {
		name          string
		arguments     []string
		result        runner.CampaignResult
		err           error
		want          runner.ResumeSpec
		wantRequested []string
		wantStatus    int
		wantOutput    []string
	}{
		{
			name: "stored campaign", arguments: []string{"--json", "--toolchain-root", "/bundle/toolchain", "/artifacts/v1/run-partial"},
			result:        runner.CampaignResult{CampaignPath: "/artifacts/v1/run-partial", SelectionCount: 3, Attempted: 3, Succeeded: 3, StopReason: runner.StopSeedsExhausted},
			wantRequested: []string{"/bundle/toolchain"},
			wantOutput:    []string{`"schema":"gomad.explore-event/v3"`, `"type":"result"`, `"classification":"success"`, `"campaign_path":"/artifacts/v1/run-partial"`},
		},
		{
			name: "invalid journal is an input error", arguments: []string{"--json", "/artifacts/v1/run-partial"},
			err:           &runner.HostError{Reason: "resume_setup", Err: errors.New("batch plan changed")},
			wantRequested: []string{""}, wantStatus: 2, wantOutput: []string{`"classification":"invalid_input"`},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			installation := newFakeInstallation()
			var observed []runner.ResumeSpec
			dependencies := installation.resumeDependencies(func(_ context.Context, config runner.ResumeSpec) (runner.CampaignResult, error) {
				config.Progress = nil
				observed = append(observed, config)
				return test.result, test.err
			})
			var stdout, stderr bytes.Buffer
			status := runResumeWith(test.arguments, &stdout, &stderr, dependencies)
			want := runner.ResumeSpec{
				CampaignPath: "/artifacts/v1/run-partial", RunnerBuild: "sha256:runner", ToolchainRoot: "/toolchain",
				ProgressInterval: 5 * time.Second,
			}
			if status != test.wantStatus || stderr.Len() != 0 || len(observed) != 1 || !reflect.DeepEqual(observed[0], want) || !reflect.DeepEqual(*installation.requested, test.wantRequested) {
				t.Fatalf("status=%d requests=%#v roots=%q stdout=%q stderr=%q", status, observed, *installation.requested, stdout.String(), stderr.String())
			}
			for _, want := range test.wantOutput {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output = %q, missing %q", stdout.String(), want)
				}
			}
		})
	}
}

func TestExploreReporterHumanOutputIncludesProgressAndReplayCommands(t *testing.T) {
	var stdout, stderr bytes.Buffer
	reporter := newExploreReporter(false, &stdout, &stderr)
	if err := reporter.Progress(runner.CampaignEvent{
		Phase: runner.ProgressRunning, CampaignPath: "/batch", Selected: 5, Attempted: 2, Running: 2, Succeeded: 1, Failures: 1, DistinctFailures: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := reporter.Result(runner.CampaignResult{
		CampaignPath: "/batch", SelectionCount: 5, Attempted: 5, Succeeded: 4, Failures: 1, DistinctFailures: 1,
		StopReason: runner.StopSeedsExhausted, Artifacts: []string{"/batch/failures/one"},
		ChoiceTrace: &record.ChoiceTraceSummary{Seed: 7, Profile: choice.Profile, Records: 3, BranchingRecords: 2, SHA256: record.HashBytes([]byte("choices")), TerminalState: "complete"},
	}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stderr.String(), "attempted=2 running=2") || !strings.Contains(stdout.String(), "retained failure: /batch/failures/one") || !strings.Contains(stdout.String(), "gomad replay /batch/failures/one") || !strings.Contains(stdout.String(), "choices-records=3 choices-decisions=0 choices-branching=2") {
		t.Fatalf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestClassifyExploreErrorDistinguishesInputTargetAndRunner(t *testing.T) {
	for _, test := range []struct {
		err  error
		want string
	}{
		{err: os.ErrInvalid, want: "invalid_input"},
		{err: &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}, want: "unsupported_target"},
		{err: &runner.HostError{Reason: "target_preparation", Err: &target.UnsupportedCapabilityError{ImportPath: "example.com/target", Capability: "imports os/exec"}}, want: "unsupported_target"},
		{err: &record.MissingSemanticProbesError{Probes: []string{"stdlib.os.openfile"}}, want: "semantic_coverage_failure"},
		{err: &runner.HostError{Reason: "cancelled", Err: context.Canceled}, want: "cancelled"},
		{err: &runner.HostError{Reason: "overall_timeout", Err: context.DeadlineExceeded}, want: "overall_timeout"},
		{err: &runner.HostError{Reason: "coordinator_exit", Err: os.ErrClosed}, want: "runner_failure"},
	} {
		if got := classifyExploreError(test.err); got != test.want {
			t.Fatalf("classifyExploreError(%T) = %q, want %q", test.err, got, test.want)
		}
	}
}

func TestExploreErrorStatusDistinguishesUserAndOperationalFailures(t *testing.T) {
	for classification, want := range map[string]int{
		"invalid_input": 2, "unsupported_target": 2, "semantic_coverage_failure": 1,
		"cancelled": 3, "overall_timeout": 3, "runner_failure": 3,
	} {
		if got := exploreErrorStatus(classification); got != want {
			t.Fatalf("exploreErrorStatus(%q) = %d, want %d", classification, got, want)
		}
	}
}

func TestClassifyExploreSummaryDistinguishesWatchdogAndReplayDivergence(t *testing.T) {
	for _, test := range []struct {
		summary runner.CampaignResult
		want    string
	}{
		{summary: runner.CampaignResult{Failures: 1, Watchdogs: 1}, want: "watchdog_observation"},
		{summary: runner.CampaignResult{Failures: 1, ReplayDivergences: 1}, want: "replay_divergence"},
		{summary: runner.CampaignResult{Failures: 2, Watchdogs: 1}, want: "mixed_failure"},
		{summary: runner.CampaignResult{Failures: 1}, want: "target_failure"},
	} {
		if got := classifyExploreSummary(test.summary); got != test.want {
			t.Fatalf("classifyExploreSummary(%#v) = %q, want %q", test.summary, got, test.want)
		}
	}
}

func TestReportReplayResultStatesWhetherFailureWasReproduced(t *testing.T) {
	for _, test := range []struct {
		name   string
		result runner.ReplayResult
		want   string
		status int
	}{
		{name: "success", result: runner.ReplayResult{Artifact: artifact.Artifact{Manifest: record.ExecutionRecord{Outcome: record.Outcome{Domain: "success"}}}, Match: true}, want: "reproduced=true diagnostic=false result=success", status: 0},
		{name: "target failure", result: runner.ReplayResult{Match: true}, want: "reproduced=true diagnostic=false result=target_failure", status: 1},
		{name: "watchdog observation", result: runner.ReplayResult{Match: true, Diagnostic: true}, want: "reproduced=true diagnostic=true result=watchdog_observation", status: 1},
		{name: "divergence", result: runner.ReplayResult{Divergence: "stdout.full_sha256"}, want: "reproduced=false divergence=stdout.full_sha256", status: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			status, err := reportReplayResult(&output, test.result)
			if err != nil || status != test.status || !strings.Contains(output.String(), test.want) {
				t.Fatalf("status = %d, error = %v, output = %q, want %q", status, err, output.String(), test.want)
			}
		})
	}
}

// TestRunMinimizeForwardsFlags pins the minimization request each flag set
// forwards with the resolved installation, and the summary it reports.
func TestRunMinimizeForwardsFlags(t *testing.T) {
	defaultRoot, err := filepath.Abs(".gomad/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		result     runner.MinimizeResult
		want       runner.MinimizeSpec
		wantOutput string
	}{
		{
			name: "bounded store and installation", arguments: []string{"--artifacts", "/artifacts", "--attempt-budget", "16", "--max-bytes", "8MiB", "/failure"},
			result: runner.MinimizeResult{
				Artifact: artifact.Artifact{Path: "/artifacts/minimized/sha256-result"}, Changed: true,
				Attempts: 7, AttemptBudget: 16, Accepted: []record.MinimizationReduction{{Kind: "fault_entries"}}, StopReason: "minimal",
			},
			want:       runner.MinimizeSpec{ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 16, MaximumBytes: 8 << 20, ToolchainRoot: "/toolchain"},
			wantOutput: "accepted=1",
		},
		// Minimization resumes only on request.
		{name: "initial run", arguments: []string{"/failure"}, want: runner.MinimizeSpec{ArtifactPath: "/failure", OutputRoot: filepath.Join(defaultRoot, "minimized"), AttemptBudget: 64, ToolchainRoot: "/toolchain"}},
		{name: "resume", arguments: []string{"--resume", "/failure"}, want: runner.MinimizeSpec{ArtifactPath: "/failure", OutputRoot: filepath.Join(defaultRoot, "minimized"), AttemptBudget: 64, ToolchainRoot: "/toolchain", Resume: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed []runner.MinimizeSpec
			dependencies := newFakeInstallation().minimizeDependencies(func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
				observed = append(observed, spec)
				return test.result, nil
			})
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith(test.arguments, &stdout, &stderr, dependencies)
			if status != 0 || stderr.Len() != 0 || len(observed) != 1 || !reflect.DeepEqual(observed[0], test.want) || !strings.Contains(stdout.String(), test.wantOutput) {
				t.Fatalf("status=%d requests=%#v want %#v stdout=%q stderr=%q", status, observed, test.want, stdout.String(), stderr.String())
			}
		})
	}
}

// TestRunQualifySetForwardsFlags pins the qualification set request each
// accepted flag set forwards with the current executable, and the shard
// checks that reject a request before the set runs.

func TestFullyAnsweredGuidanceReportsZeroExecutionsAndSuccessStatus(t *testing.T) {
	summary := runner.CampaignResult{CampaignPath: "/campaign", StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 4, Answered: 4}}
	for _, jsonOutput := range []bool{false, true} {
		var stdout, stderr bytes.Buffer
		if err := newExploreReporter(jsonOutput, &stdout, &stderr).Result(summary); err != nil {
			t.Fatal(err)
		}
		if exploreSummaryStatus(summary) != 0 || stderr.Len() != 0 {
			t.Fatalf("status=%d stderr=%s", exploreSummaryStatus(summary), &stderr)
		}
		if jsonOutput {
			var event exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &event); err != nil {
				t.Fatal(err)
			}
			if event.Classification != "success" || event.Guidance == nil || *event.Guidance != *summary.Guidance {
				t.Fatalf("event=%#v", event)
			}
			for _, want := range []string{`"requested":4`, `"answered":4`, `"guided":0`, `"new_executions":0`} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output=%s missing=%s", &stdout, want)
				}
			}
		} else {
			for _, want := range []string{"requested=4 answered=4 guided=0 new-executions=0", "all requested seeds are answered", "--guide-regression", "exit 0"} {
				if !strings.Contains(stdout.String(), want) {
					t.Fatalf("output=%s missing=%s", &stdout, want)
				}
			}
		}
	}
}

func TestGuidanceReportsNoCorpusSelectionAndNewExecutions(t *testing.T) {
	var stdout, stderr bytes.Buffer
	summary := runner.CampaignResult{SelectionCount: 3, Attempted: 3, Succeeded: 3, StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 4, Answered: 1, NewExecutions: 3}}
	if err := newExploreReporter(false, &stdout, &stderr).Result(summary); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "guidance selected no corpus seeds") || !strings.Contains(stdout.String(), "new-executions=3") {
		t.Fatalf("output=%s", &stdout)
	}
}

func TestResumeGuidanceModePreservesExplicitFalseAndTrue(t *testing.T) {
	for _, value := range []string{"false", "true"} {
		var got *bool
		dependencies := resumeDependencies{install: func(string) (installation, error) {
			return installation{toolchainRoot: "/toolchain", executable: "/gomad", runnerBuild: "runner"}, nil
		}, run: func(_ context.Context, spec runner.ResumeSpec) (runner.CampaignResult, error) {
			got = spec.GuideRegression
			return runner.CampaignResult{StopReason: runner.StopSeedsExhausted}, nil
		}}
		var stdout, stderr bytes.Buffer
		status := runResumeWith([]string{"--guide-regression=" + value, "/campaign"}, &stdout, &stderr, dependencies)
		if status != 0 || got == nil || *got != (value == "true") {
			t.Fatalf("status=%d mode=%v", status, got)
		}
	}
}

func TestGuideRegressionRequiresGuidance(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := hostApplication().runExplore([]string{"--json", "--guide-regression", "go-run", "./fixture"}, &stdout, &stderr)
	if status != 2 || !strings.Contains(stdout.String(), "--guide-regression requires --guide") {
		t.Fatalf("status=%d stdout=%s stderr=%s", status, &stdout, &stderr)
	}
}

func TestEmptyShardDoesNotReportPartiallyAnsweredPlanAsFullyAnswered(t *testing.T) {
	var stdout, stderr bytes.Buffer
	summary := runner.CampaignResult{SelectionCount: 0, StopReason: runner.StopSeedsExhausted, Guidance: &runner.GuidanceSummary{Requested: 2, Answered: 1}}
	if err := newExploreReporter(false, &stdout, &stderr).Result(summary); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stdout.String(), "all requested seeds are answered") || !strings.Contains(stdout.String(), "requested=2 answered=1") || !strings.Contains(stdout.String(), "guidance selected no corpus seeds") {
		t.Fatalf("empty shard output=%s", &stdout)
	}
}

func TestChoiceExplorationDivergenceClassificationAndStatus(t *testing.T) {
	for _, test := range []struct {
		name           string
		summary        runner.CampaignResult
		classification string
		status         int
	}{
		{"choice confidence failure", runner.CampaignResult{Failures: 1, ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "replay_divergence", 3},
		{"choice mixed failure", runner.CampaignResult{Failures: 2, ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "mixed_failure", 3},
		{"confidence counter without target failure", runner.CampaignResult{ReplayDivergences: 1, ChoiceExploration: &runner.ChoiceExplorationSummary{}}, "replay_divergence", 3},
		{"world replay divergence", runner.CampaignResult{Failures: 1, ReplayDivergences: 1}, "replay_divergence", 1},
		{"target failure", runner.CampaignResult{Failures: 1}, "target_failure", 1},
		{"success", runner.CampaignResult{}, "success", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := classifyExploreSummary(test.summary); got != test.classification {
				t.Fatalf("classification = %s", got)
			}
			if got := exploreSummaryStatus(test.summary); got != test.status {
				t.Fatalf("status = %d", got)
			}
			var stdout, stderr bytes.Buffer
			status := runResumeWith([]string{"--json", "campaign"}, &stdout, &stderr, resumeDependencies{
				install: func(string) (installation, error) {
					return installation{toolchainRoot: "toolchain", executable: "gomad", runnerBuild: "runner"}, nil
				},
				run: func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) { return test.summary, nil },
			})
			if status != test.status {
				t.Fatalf("resume status=%d output=%s stderr=%s", status, stdout.String(), stderr.String())
			}
			var result exploreEvent
			if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Classification != test.classification {
				t.Fatalf("reported classification=%s", result.Classification)
			}
		})
	}
}

func TestInspectReportsCandidateReplayDivergence(t *testing.T) {
	candidate := inspect.ExecutionInspection{Strategy: string(runner.StrategyChoiceExploration), Domain: "runner", Reason: "replay_divergence", Divergence: &choice.DivergenceEvidence{Ordinal: 7, Reason: choice.DivergenceAlternativeSet}}
	inspected := inspect.CampaignInspection{ReplayDivergences: 1, Executions: []inspect.ExecutionInspection{candidate}}
	var output bytes.Buffer
	printer := &inspectionPrinter{output: &output}
	printCampaignInspection(printer, &inspected)
	if printer.err != nil {
		t.Fatal(printer.err)
	}
	for _, expected := range []string{"reason=replay_divergence", "divergence-ordinal=7", "divergence-reason=alternative_set"} {
		if !strings.Contains(output.String(), expected) {
			t.Fatalf("inspect missing %s: %s", expected, output.String())
		}
	}
	encoded, err := json.Marshal(inspected)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"\"divergence\"", "\"ordinal\":7", "\"reason\":5", "\"replay_divergences\":1"} {
		if !bytes.Contains(encoded, []byte(expected)) {
			t.Fatalf("inspect JSON missing %s: %s", expected, encoded)
		}
	}
}

func TestExploreDiagnosticsRejectsForcedPrefixesBeforePreparation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := hostApplication().runExplore([]string{"--diagnostics", "--strategy=choice-exploration", "--max-executions=2", "--max-choice-depth=1", "--max-exploration-bytes=1MiB", "go-run", "./fixture"}, &stdout, &stderr)
	if status != 2 || !strings.Contains(stderr.String(), "forced-prefix exploration is unsupported") {
		t.Fatalf("status %d: %s", status, stderr.String())
	}
}

func TestReplayOutputVerifyOnly(t *testing.T) {
	for _, failed := range []bool{false, true} {
		t.Run(map[bool]string{false: "healthy", true: "EBADF"}[failed], func(t *testing.T) {
			stdout := terminalDiagnostics(t, map[int]bool{1: failed})
			stderr := terminalDiagnostics(t, nil)
			installation := newFakeInstallation()
			calls := 0
			var observed runner.ReplaySpec
			dependencies := installation.replayDependencies(func(ctx context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
				calls++
				observed = spec
				if ctx != context.Background() || len(stdout.attempts) != 0 || len(stderr.attempts) != 0 || !reflect.DeepEqual(*installation.requested, []string{"/bundle"}) {
					t.Fatal("replay callback did not follow installation and precede reporting")
				}
				return runner.ReplayResult{Artifact: artifact.Artifact{Path: "/verified artifact/%s"}, Verified: true}, nil
			})
			status := runReplayWith([]string{"--toolchain-root=/bundle", "--observed=/observed", "--verify-only", "/requested-artifact"}, stdout, stderr, dependencies)
			want := runner.ReplaySpec{ArtifactPath: "/requested-artifact", VerifyOnly: true, ToolchainRoot: "/toolchain", ObservedDir: "/observed"}
			if calls != 1 || !reflect.DeepEqual(observed, want) {
				t.Fatalf("replay calls = %d, request = %#v, want %#v", calls, observed, want)
			}
			checkTerminalDiagnostics(t, stdout, []string{"gomad: verified /verified artifact/%s\n"})
			checkTerminalDiagnostics(t, stderr, nil)
			wantStatus := 0
			if failed {
				wantStatus = 3
			}
			if status != wantStatus {
				t.Fatalf("status = %d, want %d after completed verification", status, wantStatus)
			}
		})
	}
}

func TestReplayOutputEarlierErrors(t *testing.T) {
	for _, test := range []struct {
		name       string
		installErr error
		replayErr  error
		status     int
		calls      int
		diagnostic string
	}{
		{"installation", errFakeInstallation, nil, 3, 0, "resolve Gomad installation: fake installation is unavailable\n"},
		{"preflight", nil, &runner.ReplayPreflightError{Err: errors.New("toolchain changed")}, 2, 1, "incompatible replay artifact: toolchain changed\n"},
		{"replay", nil, errors.New("supervisor failed"), 3, 1, "supervisor failed\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			stdout := terminalDiagnostics(t, map[int]bool{1: true})
			stderr := terminalDiagnostics(t, nil)
			installation := newFakeInstallation().failing(test.installErr)
			calls := 0
			dependencies := installation.replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
				calls++
				want := runner.ReplaySpec{ArtifactPath: "/artifact", VerifyOnly: true, ToolchainRoot: "/toolchain"}
				if !reflect.DeepEqual(spec, want) || len(stdout.attempts) != 0 || len(stderr.attempts) != 0 {
					t.Fatalf("replay request = %#v, want %#v before reporting", spec, want)
				}
				return runner.ReplayResult{}, test.replayErr
			})
			status := runReplayWith([]string{"--toolchain-root=/bundle", "--verify-only", "/artifact"}, stdout, stderr, dependencies)
			if status != test.status || calls != test.calls || !reflect.DeepEqual(*installation.requested, []string{"/bundle"}) {
				t.Fatalf("status = %d, calls = %d, installation requests = %q", status, calls, *installation.requested)
			}
			checkTerminalDiagnostics(t, stdout, nil)
			checkTerminalDiagnostics(t, stderr, []string{test.diagnostic})
		})
	}
}

func TestExternalMinimizeForwardsProviderBoundsResumeAndSharedOutput(t *testing.T) {
	defaultRoot, err := filepath.Abs(".gomad/artifacts")
	if err != nil {
		t.Fatal(err)
	}
	provider := commandBackend{}
	for _, test := range []struct {
		name      string
		arguments []string
		want      runner.MinimizeSpec
		output    string
	}{
		{
			name: "defaults", arguments: []string{"/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: filepath.Join(defaultRoot, "minimized"), AttemptBudget: 64},
			output: "gomad minimize: changed=true attempts=2/64 accepted=0 stop=minimal artifact=/minimized\n",
		},
		{
			name: "resume bounded root", arguments: []string{"--resume", "--artifacts=/artifacts", "--attempt-budget=4", "--max-bytes=2MiB", "/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 4, MaximumBytes: 2 << 20, Resume: true},
			output: "gomad minimize: changed=true attempts=2/4 accepted=0 stop=minimal artifact=/minimized\n",
		},
		{
			name: "shared JSON", arguments: []string{"--json", "--artifacts=/artifacts", "--attempt-budget=4", "/failure"},
			want:   runner.MinimizeSpec{Backend: provider, ArtifactPath: "/failure", OutputRoot: "/artifacts/minimized", AttemptBudget: 4},
			output: "{\"artifact_path\":\"/minimized\",\"record_hash\":\"\",\"changed\":true,\"attempts\":2,\"attempt_budget\":4,\"accepted\":null,\"stop_reason\":\"minimal\"}\n",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var observed []runner.MinimizeSpec
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith(test.arguments, &stdout, &stderr, minimizeDependencies{
				install: func(root string) (installation, error) {
					if root != "" {
						t.Fatalf("external minimization selected native toolchain: %s", root)
					}
					return installation{backend: provider, backendName: "wasm"}, nil
				},
				minimize: func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
					observed = append(observed, spec)
					return runner.MinimizeResult{Artifact: artifact.Artifact{Path: "/minimized"}, Changed: true, Attempts: 2, AttemptBudget: test.want.AttemptBudget, StopReason: "minimal"}, nil
				},
			})
			if status != 0 || stderr.Len() != 0 || len(observed) != 1 || !reflect.DeepEqual(observed[0], test.want) || stdout.String() != test.output {
				t.Fatalf("status=%d requests=%#v want=%#v stdout=%q stderr=%q", status, observed, test.want, stdout.String(), stderr.String())
			}
		})
	}
}

func TestExternalMinimizePreservesErrorClassification(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
	}{
		{"preflight", &runner.ReplayPreflightError{Err: errors.New("incompatible retained choice failure")}, 2},
		{"runner", errors.New("minimization infrastructure failure"), 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := runMinimizeWith([]string{"/failure"}, &stdout, &stderr, minimizeDependencies{
				install: func(string) (installation, error) {
					return installation{backend: commandBackend{}, backendName: "wasm"}, nil
				},
				minimize: func(_ context.Context, spec runner.MinimizeSpec) (runner.MinimizeResult, error) {
					if spec.Backend == nil || spec.ToolchainRoot != "" {
						t.Fatalf("external backend lost or native launch configured: %+v", spec)
					}
					return runner.MinimizeResult{}, test.err
				},
			})
			if status != test.status || stdout.Len() != 0 || !strings.Contains(stderr.String(), test.err.Error()) {
				t.Fatalf("status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
			}
		})
	}
}

type terminalDiagnosticWriter struct {
	output   bytes.Buffer
	file     *os.File
	failures map[int]bool
	attempts []string
	errors   []error
}

func (writer *terminalDiagnosticWriter) Write(contents []byte) (int, error) {
	writer.attempts = append(writer.attempts, string(contents))
	if writer.failures[len(writer.attempts)] {
		written, err := writer.file.Write(contents)
		writer.errors = append(writer.errors, err)
		return written, err
	}
	writer.errors = append(writer.errors, nil)
	return writer.output.Write(contents)
}

func terminalDiagnostics(t *testing.T, failures map[int]bool) *terminalDiagnosticWriter {
	t.Helper()
	path := filepath.Join(t.TempDir(), "read-only")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	return &terminalDiagnosticWriter{file: file, failures: failures}
}

func checkTerminalDiagnostics(t *testing.T, writer *terminalDiagnosticWriter, want []string) {
	t.Helper()
	if !reflect.DeepEqual(writer.attempts, want) {
		t.Fatalf("attempts = %q, want %q", writer.attempts, want)
	}
	var delivered strings.Builder
	for index, contents := range want {
		if writer.failures[index+1] {
			if !errors.Is(writer.errors[index], syscall.EBADF) {
				t.Fatalf("attempt %d error = %v, want actual EBADF", index+1, writer.errors[index])
			}
		} else {
			if writer.errors[index] != nil {
				t.Fatal(writer.errors[index])
			}
			delivered.WriteString(contents)
		}
	}
	if writer.output.String() != delivered.String() {
		t.Fatalf("delivered = %q, want %q", writer.output.String(), delivered.String())
	}
	contents, err := os.ReadFile(writer.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 0 {
		t.Fatalf("read-only file received %q", contents)
	}
}

func TestTerminalDiagnosticsPrivateDispatch(t *testing.T) {
	for _, mode := range []string{coordinatorMode, supervisorMode, targetBootstrapMode} {
		t.Run(mode, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				input, output := strings.NewReader("request"), new(bytes.Buffer)
				calls := 0
				app := application{privateInput: input, privateOutput: output, dispatch: func(got string, in io.Reader, out io.Writer) error {
					if got != mode || in != input || out != output || len(stderr.attempts) != 0 {
						t.Fatal("private dispatch streams or ordering changed")
					}
					calls++
					return errors.New("decode private request: EOF")
				}}
				if status := app.run([]string{mode, "ignored"}, stdout, stderr); status != 3 || calls != 1 || len(stdout.attempts) != 0 || output.Len() != 0 {
					t.Fatalf("status = %d, calls = %d, stdout = %q, private output = %q", status, calls, stdout.attempts, output.String())
				}
				checkTerminalDiagnostics(t, stderr, []string{"decode private request: EOF\n"})
			}
		})
	}
}

func TestTerminalDiagnosticsReplayClassifications(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		text   string
	}{
		{"preflight", &runner.ReplayPreflightError{Err: errors.New("invalid retained artifact")}, 2, "incompatible replay artifact: invalid retained artifact\n"},
		{"host", &runner.HostError{Reason: "replay", Err: errors.New("child failed")}, 3, "gomad Runner/host failure: replay: child failed\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				calls := 0
				dependencies := newFakeInstallation().replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
					if spec.ArtifactPath != "/artifact" || len(stderr.attempts) != 0 {
						t.Fatal("replay operation or ordering changed")
					}
					calls++
					return runner.ReplayResult{}, test.err
				})
				if status := runReplayWith([]string{"/artifact"}, stdout, stderr, dependencies); status != test.status || calls != 1 || len(stdout.attempts) != 0 {
					t.Fatalf("status = %d, calls = %d, stdout = %q", status, calls, stdout.attempts)
				}
				checkTerminalDiagnostics(t, stderr, []string{test.text})
			}
		})
	}
}

func TestTerminalDiagnosticsCompletedOperations(t *testing.T) {
	for _, command := range []string{"explore", "execute-shard", "resume", "replay", "qualify"} {
		t.Run(command, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				root := t.TempDir()
				published := filepath.Join(root, "completed")
				stdout := terminalDiagnostics(t, map[int]bool{1: true})
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				completed := 0
				finish := func() {
					if len(stdout.attempts) != 0 || len(stderr.attempts) != 0 {
						t.Fatal("reporting preceded the operation")
					}
					if err := os.WriteFile(published, []byte("committed\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					completed++
				}
				result := runner.CampaignResult{CampaignPath: published, Attempted: 1, Succeeded: 1}
				fake := newFakeInstallation()
				var status int
				var wantOutput string
				switch command {
				case "explore":
					dependencies := fake.exploreDependencies(func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					}, nil)
					status = runExploreWith([]string{"go-run", "./pkg"}, stdout, stderr, dependencies)
				case "execute-shard":
					dependencies := fake.campaignShardDependencies(func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					})
					status = runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, dependencies)
				case "resume":
					dependencies := fake.resumeDependencies(func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					})
					status = runResumeWith([]string{"/campaign"}, stdout, stderr, dependencies)
				case "replay":
					dependencies := fake.replayDependencies(func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
						finish()
						return runner.ReplayResult{Match: true}, nil
					})
					status = runReplayWith([]string{"/artifact"}, stdout, stderr, dependencies)
					wantOutput = "gomad: reproduced=true diagnostic=false result=target_failure choice-replay=\n"
				case "qualify":
					dependencies := qualifyDependencies{
						install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil },
						workload: func(context.Context, qualificationworkload.WorkloadSpec) (qualificationworkload.WorkloadResult, error) {
							finish()
							return qualificationworkload.WorkloadResult{ReportPath: published, Report: qualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true}}, nil
						},
					}
					status = runQualifyWith([]string{"go-run", "./pkg"}, stdout, stderr, dependencies)
					wantOutput = "gomad: qualification qualified=true deterministic=true target-success=true seed=0 repeat=0 report=" + published + "\n"
				}
				if wantOutput == "" {
					wantOutput = "gomad: classification=success attempted=1 succeeded=1 failures=0 watchdogs=0 replay-divergences=0 distinct=0 retained-successes=0 retained-success-bytes=0 stop= artifact=" + published + "\n"
				}
				if status != 3 || completed != 1 {
					t.Fatalf("status = %d, completed = %d", status, completed)
				}
				checkTerminalDiagnostics(t, stdout, []string{wantOutput})
				checkTerminalDiagnostics(t, stderr, []string{"write " + stdout.file.Name() + ": bad file descriptor\n"})
				contents, err := os.ReadFile(published)
				if err != nil || string(contents) != "committed\n" {
					t.Fatalf("completed operation file = %q, %v", contents, err)
				}
			}
		})
	}
}

func TestCampaignSemanticValidationBelongsToRunner(t *testing.T) {
	files := token.NewFileSet()
	file, err := parser.ParseFile(files, "cli.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, rule := range []struct {
		function string
		owner    string
		calls    map[string]int
		forbid   map[string]bool
	}{
		{"resolveExploreStrategy", "runner", map[string]int{"ParseSingleBaseSeed": 2}, map[string]bool{"Count": true, "ParseSeeds": true}},
		{"resolveExploreCoverage", "campaignoptions", map[string]int{"ValidateCoverage": 1}, map[string]bool{"MissingRequiredSemanticProbes": true, "CoverageNone": true, "CoverageChoice": true, "CoverageSemantic": true, "CoverageSemanticChoice": true}},
		{"resolveChoiceTrace", "campaignoptions", map[string]int{"ValidateChoiceTraceLimit": 1}, nil},
		{"parseCampaignRequest", "campaignoptions", map[string]int{"ValidateChoiceCoverage": 1}, map[string]bool{"CoverageChoice": true, "CoverageSemanticChoice": true}},
		{"resolveExploreGuidance", "campaignoptions", map[string]int{"NormalizeCoverage": 1}, nil},
	} {
		t.Run(rule.function, func(t *testing.T) {
			var body *ast.BlockStmt
			for _, declaration := range file.Decls {
				if function, ok := declaration.(*ast.FuncDecl); ok && function.Name.Name == rule.function {
					body = function.Body
				}
			}
			if body == nil {
				t.Fatal("campaign validation function is missing")
			}
			calls := make(map[string]int)
			ast.Inspect(body, func(node ast.Node) bool {
				if comparison, ok := node.(*ast.BinaryExpr); ok && rule.function == "resolveChoiceTrace" {
					ast.Inspect(comparison, func(node ast.Node) bool {
						if selector, ok := node.(*ast.SelectorExpr); ok && (selector.Sel.Name == "MinimumChoiceTraceBytes" || selector.Sel.Name == "MaximumChoiceTraceBytes") {
							t.Errorf("%s: CLI duplicates Runner's trace capacity rule", files.Position(selector.Pos()))
						}
						return true
					})
				}
				if selector, ok := node.(*ast.SelectorExpr); ok && rule.forbid[selector.Sel.Name] {
					t.Errorf("%s: CLI duplicates Runner's semantic responsibility through %s", files.Position(selector.Pos()), selector.Sel.Name)
				}
				var expressions []ast.Expr
				switch value := node.(type) {
				case *ast.IfStmt:
					condition, ok := value.Cond.(*ast.BinaryExpr)
					if !ok || condition.Op != token.NEQ {
						break
					}
					left, leftOK := condition.X.(*ast.Ident)
					right, rightOK := condition.Y.(*ast.Ident)
					if !leftOK || !rightOK || left.Name != "err" || right.Name != "nil" {
						break
					}
					if assignment, ok := value.Init.(*ast.AssignStmt); ok {
						expressions = assignment.Rhs
					}
				case *ast.ReturnStmt:
					expressions = value.Results
				}
				for _, expression := range expressions {
					ast.Inspect(expression, func(node ast.Node) bool {
						if call, ok := node.(*ast.CallExpr); ok {
							if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
								if owner, ok := selector.X.(*ast.Ident); ok && owner.Name == rule.owner {
									calls[selector.Sel.Name]++
								}
							}
						}
						return true
					})
				}
				return true
			})
			for name, want := range rule.calls {
				if got := calls[name]; got != want {
					t.Errorf("%s: %d checked %s.%s calls, want %d in actual validation", rule.function, got, rule.owner, name, want)
				}
			}
		})
	}
}

func readInspectFixtureLog(t *testing.T, path string) []byte {
	t.Helper()
	log, err := os.ReadFile(filepath.Join(path, "store", "log"))
	if err != nil {
		t.Fatal(err)
	}
	return log
}

// A recognized store whose replay fails is diagnosed on stderr alone, as text
// or as one JSON report, with status 2; a diagnosis that cannot be written
// returns status 3. Nothing reaches stdout and the log is left as it was.
func TestRunInspectDiagnosesAnInvalidStoreOnStderrOnly(t *testing.T) {
	path := writeInspectBatchFixture(t)
	log := readInspectFixtureLog(t, path)
	envelope := int64(16 + 4 + len(`{"format":"gomad.campaign-log/v1","campaign_id":"run-inspect-command","strategy":"seed","selection":"7","selection_count":"1"}`))
	log[envelope+20] ^= 0xff
	writeInspectLog(t, filepath.Join(path, "store"), log)

	var stdout, stderr bytes.Buffer
	if status := runInspect([]string{path}, &stdout, &stderr); status != 2 || stdout.Len() != 0 {
		t.Fatalf("text status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	text := stderr.String()
	for _, want := range []string{"gomad inspect: kind=campaign path=" + path + "\n", "log: format=gomad.campaign-log/v1 frame-version=1 state=invalid records=1 valid-bytes=" + itoa(envelope), "invalid-store: problem=corrupt offset=" + itoa(envelope) + " records=1 reason=payload checksum mismatch before the final frame\n"} {
		if !strings.Contains(text, want) {
			t.Fatalf("text diagnosis = %q, missing %q", text, want)
		}
	}
	if strings.Contains(text, "inspect "+path+":") || strings.Contains(text, "lifecycle:") || strings.Contains(text, "campaign:") {
		t.Fatalf("text diagnosis repeats the error or reports state: %q", text)
	}

	stdout.Reset()
	stderr.Reset()
	if status := runInspect([]string{"--json", path}, &stdout, &stderr); status != 2 || stdout.Len() != 0 {
		t.Fatalf("json status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
	if lines := strings.Count(stderr.String(), "\n"); lines != 1 {
		t.Fatalf("json diagnosis has %d lines: %q", lines, stderr.String())
	}
	var report inspect.Inspection
	decoder := json.NewDecoder(&stderr)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&report); err != nil {
		t.Fatal(err)
	}
	if report.Schema != "gomad.inspect/v8" || report.Kind != "campaign" || report.Log == nil || report.Log.State != "invalid" || report.Log.Invalid == nil ||
		report.Log.Invalid.Offset != uint64(envelope) || report.Log.Invalid.Records != 1 || report.Campaign != nil || report.Lifecycle != nil || report.Committed != nil {
		t.Fatalf("json diagnosis = %#v %#v", report, report.Log)
	}

	for _, arguments := range [][]string{{path}, {"--json", path}} {
		stdout.Reset()
		if status := runInspect(arguments, &stdout, failingWriter{}); status != 3 || stdout.Len() != 0 {
			t.Fatalf("%v undeliverable diagnosis status=%d stdout=%q", arguments, status, stdout.String())
		}
	}
	if after := readInspectFixtureLog(t, path); !bytes.Equal(after, log) {
		t.Fatal("inspect changed the damaged log")
	}
}

// An observed torn final record is a successful report on stdout that leaves
// the torn bytes for the next writer to repair.
func TestRunInspectReportsATornTailWithoutRepairingIt(t *testing.T) {
	path := writeInspectBatchFixture(t)
	clean := readInspectFixtureLog(t, path)
	torn := append(bytes.Clone(clean), inspectFrame(2, []byte(`{"state":"running"}`))[:11]...)
	writeInspectLog(t, filepath.Join(path, "store"), torn)
	for _, test := range []struct {
		arguments []string
		want      []string
	}{
		{arguments: []string{path}, want: []string{"state=torn-tail records=5 valid-bytes=" + itoa(int64(len(clean))) + " snapshot-bytes=" + itoa(int64(len(torn))) + " tail=short-frame tail-bytes=11 repaired-bytes=0\n", "lifecycle: state=published", "campaign: id=run-inspect-command"}},
		{arguments: []string{"--json", path}, want: []string{`"state":"torn-tail"`, `"tail":"short-frame","tail_bytes":11,"repaired_bytes":0`, `"campaign_id":"run-inspect-command"`}},
	} {
		var stdout, stderr bytes.Buffer
		if status := runInspect(test.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
			t.Fatalf("%v status=%d stdout=%q stderr=%q", test.arguments, status, stdout.String(), stderr.String())
		}
		for _, want := range test.want {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("%v output = %q, missing %q", test.arguments, stdout.String(), want)
			}
		}
	}
	if after := readInspectFixtureLog(t, path); !bytes.Equal(after, torn) {
		t.Fatal("inspect repaired the torn tail")
	}
}

// A guided corpus directory is inspected by the existing PATH dispatch: its
// log lineage and live state report in text and JSON, and --choices still
// requires a traced artifact.
func TestRunInspectReportsAGuidedCorpusDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corpus")
	store := filepath.Join(path, "store")
	if err := os.MkdirAll(store, 0o700); err != nil {
		t.Fatal(err)
	}
	header := `{"format":"gomad.guide-corpus-log/v1","identity":{"environment_sha256":"sha256:` + strings.Repeat("e", 64) + `","target_sha256":"sha256:` + strings.Repeat("a", 64) + `","toolchain":{"go_version":"go1.26.4","build_key":"key","target_goos":"linux","target_goarch":"arm64"},"boundary_version":"v1","boundary_sha256":"sha256:` + strings.Repeat("b", 64) + `","instrumentation_schema":"gomad.semantic-features/v1","instrumentation_sha256":"sha256:` + strings.Repeat("c", 64) + `","manifest_schema_version":1,"manifest_record_contract":"gomad.execution-record/v1"}}`
	writeInspectLog(t, store, inspectFrame(1, []byte(header)))
	for _, test := range []struct {
		arguments []string
		want      []string
	}{
		{arguments: []string{path}, want: []string{"gomad inspect: kind=corpus path=" + path + "\n", "log: format=gomad.guide-corpus-log/v1 frame-version=1 state=valid records=1 ", "corpus: schema=gomad.guide-corpus/v2 generation=0 entries=0 max-entries=1024 entry-stored-bytes=0 max-bytes=1073741824 target=sha256:" + strings.Repeat("a", 64) + " toolchain=key instrumentation=gomad.semantic-features/v1"}},
		{arguments: []string{"--json", path}, want: []string{`"kind":"corpus"`, `"log":{"format":"gomad.guide-corpus-log/v1"`, `"corpus":{"schema":"gomad.guide-corpus/v2","generation":0,`, `"maximum_entries":1024`, `"cases":[]`}},
	} {
		var stdout, stderr bytes.Buffer
		if status := runInspect(test.arguments, &stdout, &stderr); status != 0 || stderr.Len() != 0 {
			t.Fatalf("%v status=%d stdout=%q stderr=%q", test.arguments, status, stdout.String(), stderr.String())
		}
		for _, want := range test.want {
			if !strings.Contains(stdout.String(), want) {
				t.Fatalf("%v output = %q, missing %q", test.arguments, stdout.String(), want)
			}
		}
	}
	var stdout, stderr bytes.Buffer
	if status := runInspect([]string{"--choices", path}, &stdout, &stderr); status != 2 || stdout.Len() != 0 || !strings.Contains(stderr.String(), "traced artifact") {
		t.Fatalf("corpus --choices status=%d stdout=%q stderr=%q", status, stdout.String(), stderr.String())
	}
}

// Committed outcomes, the admission frontier, exploration rounds and merged
// source provenance print one line each.
func TestPrintInspectionReportsCommittedStateRoundsAndMergedSources(t *testing.T) {
	report := inspect.Inspection{
		Kind: "campaign", Path: "/batch",
		Log:       &inspect.LogInspection{Format: "gomad.campaign-log/v1", FrameVersion: 1, State: "valid", Records: 9, ValidBytes: 900, SnapshotBytes: 900, Tail: "none"},
		Lifecycle: &inspect.CampaignLifecycleInspection{State: "running", Resumable: true},
		Committed: &inspect.CommittedCampaignInspection{
			Outcomes: 2, OutcomeBytes: 400, Limits: inspect.ExecutionJournalLimitsInspection{MaximumExecutions: 4, MaximumBytes: 4096, MaximumPartialExecutions: 2},
			Admitted:  []inspect.AdmittedOrdinalInspection{{Ordinal: 2, Seed: 9, Receipt: &inspect.AdmissionReceiptInspection{Domain: "target", Reason: "exit", Termination: "exit"}}, {Ordinal: 3, Seed: 10}},
			Available: 0, PendingStop: "first_failure",
		},
		Exploration: &inspect.ExplorationInspection{
			Strategy: "choice-exploration", ChainSHA256: "sha256:chain", RecoveryExecutions: 1,
			Rounds:      []inspect.ExplorationRoundInspection{{Index: 0, RoundBytes: 77, FirstOrdinal: 0, Executions: 2, RoundSHA256: "sha256:round", SegmentSHA256: "sha256:segment"}},
			StagedRound: &inspect.ExplorationStagedRound{Index: 1, Candidates: 3},
		},
	}
	var output bytes.Buffer
	if err := printInspection(&output, report); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"committed: outcomes=2 bytes=400 max-executions=4 max-bytes=4096 max-partials=2 admitted=2 available=0 pending-stop=first_failure stopped=\n",
		"admitted: ordinal=2 seed=9 receipt=target/exit/exit\n", "admitted: ordinal=3 seed=10 receipt=none\n",
		"exploration-rounds: strategy=choice-exploration rounds=1 recovery-executions=1 chain=sha256:chain\n",
		"exploration-round: index=0 bytes=77 first-ordinal=0 executions=2 sha256=sha256:round segment-sha256=sha256:segment\n",
		"exploration-staged-round: index=1 candidates=3\n",
	} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("inspection output = %q, missing %q", output.String(), want)
		}
	}

	merged := inspect.Inspection{Kind: "merged-campaign", Path: "/merged", Merged: &inspect.MergedCampaignInspection{
		PlanSHA256: "sha256:plan", Shards: 1, SourceCampaignIDs: []string{"campaign-a"},
		Limits:           inspect.ExecutionJournalLimitsInspection{MaximumExecutions: 4, MaximumBytes: 4096, MaximumPartialExecutions: 2},
		ArtifactCapacity: inspect.ArtifactCapacityInspection{TotalBytes: 8192},
		Sources:          []inspect.MergedSourceInspection{{Index: 0, Count: 1, CampaignID: "campaign-a", Attempted: 4, JournalBytes: 400, Records: 9, LogSHA256: "sha256:log", Path: "/shards/a"}},
	}}
	output.Reset()
	if err := printInspection(&output, merged); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"merged-limits: max-executions=4 max-bytes=4096 max-partials=2 artifact-max-bytes=8192\n", "source-campaign: campaign-a\n", "source-log: index=0 count=1 campaign=campaign-a attempted=4 journal-bytes=400 records=9 log-sha256=sha256:log path=/shards/a\n"} {
		if !strings.Contains(output.String(), want) {
			t.Fatalf("merged output = %q, missing %q", output.String(), want)
		}
	}
}

func itoa(value int64) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func qualificationDependencies(t *testing.T) qualifyDependencies {
	t.Helper()
	return qualifyDependencies{
		install: func(string) (installation, error) {
			return installation{toolchainRoot: "/toolchain", executable: "/bin/gomad", runnerBuild: "sha256:runner"}, nil
		},
		workingDirectory: func() (string, error) { return "/workspace", nil },
		run: func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
			t.Fatal("qualification runner is not configured")
			return runner.CampaignResult{}, nil
		},
		replay: func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
			t.Fatal("unexpected replay")
			return runner.ReplayResult{}, nil
		},
		write: func(string, qualification.QualificationReport) (string, error) {
			t.Fatal("qualification writer is not configured")
			return "", nil
		},
	}
}

func qualificationEvidence(seed uint64) record.ExecutionEvidence {
	return record.ExecutionEvidence{
		Schema: record.ExecutionEvidenceSchema, Seed: record.Uint64String(seed), RunnerBuild: "sha256:runner",
		Toolchain:   record.Toolchain{GoVersion: "go1.26.4", BuildKey: "build", TargetGOOS: "darwin", TargetGOARCH: "arm64"},
		Target:      record.Target{Kind: "go-test", Source: "./pkg", SHA256: "sha256:target", Size: 12, Argv: []string{"gomad-target"}, BuildTags: []string{"gomad_fixture"}},
		IOProfile:   record.IOContract{Name: "deterministic", ImplementationSHA256: "sha256:io", InventorySHA256: "sha256:inventory"},
		Environment: []record.Environment{{Name: "GOMADSEED", Value: fmt.Sprintf("%d", seed)}, {Name: "TZ", Value: "UTC"}},
		Outcome:     record.OutcomeEvidence{Domain: "success", Reason: "success", Termination: "exit"}, GroupGone: true,
		Stdout: record.Stream{FullSHA256: "sha256:stdout"}, Stderr: record.Stream{FullSHA256: "sha256:stderr"},
		IOTranscriptSHA256: "sha256:transcript", IOTranscriptRecords: 1, IOTranscriptComplete: true,
		SemanticCoverage: record.SemanticCoverage{Schema: record.SemanticCoverageSchema, Digest: "sha256:coverage", Probes: []string{"stdlib.os.openfile"}},
	}
}
