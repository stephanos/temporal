package runner

import (
	"bytes"
	"reflect"
	"testing"

	backend "go.temporal.io/server/tools/gomad3/backends"
	"go.temporal.io/server/tools/gomad3/backends/native/transcript"
	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/world"
)

func TestAssessWorldValidatesTheRecordAgainstItsSeed(t *testing.T) {
	recording := completionWorldRecord(t, 7)
	decoded, err := world.DecodeRecording(recording)
	if err != nil {
		t.Fatal(err)
	}
	composed, err := backend.ComposeRecording(decoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	malformed := bytes.Clone(recording)
	malformed[len(malformed)-1] ^= 1
	for _, test := range []struct {
		name   string
		record []byte
		seed   uint64
		limit  uint64
		want   backend.Bundle
		cause  string
	}{
		{name: "no record", seed: 7, limit: 1 << 20, want: noneWorldBundle()},
		{name: "valid record", record: recording, seed: 7, limit: 1 << 20, want: composed},
		// TestCompletionFaultsKeepReasonPrecedenceAndEvidence pins the malformed
		// record and seed mismatch errors through every strategy.
		{name: "transition limit", record: recording, seed: 7, cause: "World transition limit must be positive"},
		{name: "malformed record before seed mismatch", record: malformed, seed: 8, limit: 1 << 20, cause: "decode World terminal: invalid character '|' after object key:value pair"},
		{name: "seed mismatch", record: recording, seed: 8, limit: 1 << 20, cause: "World record seed or schema does not match seed 8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			captured := bytes.Clone(test.record)
			bundle, err := assessWorld(backend.ExecutionResult{WorldRecord: captured}, test.seed, test.limit)
			if test.cause != "" {
				if err == nil || err.Error() != test.cause {
					t.Fatalf("assessWorld() error = %v, want %q", err, test.cause)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			clear(captured)
			if !reflect.DeepEqual(bundle, test.want) {
				t.Fatalf("assessWorld() = %#v, want %#v", bundle, test.want)
			}
		})
	}
}

func TestAssessCompletionProjectsCoverageInOrderAndClassifies(t *testing.T) {
	prepared := newFakePreparer(t).prepared
	limit := choiceTraceLimit(t, 1)
	trace := completeChoiceTrace(t, prepared.BuildKey, limit, []choice.Record{{
		Ordinal: 0, Kind: choice.KindRunnable, Flags: choice.FlagDecision, Alternatives: 2,
	}})
	projection, features, err := projectChoiceFeatures(trace, prepared)
	if err != nil {
		t.Fatal(err)
	}
	noCoverage, err := transcript.SummarizeSemanticProbes(nil)
	if err != nil {
		t.Fatal(err)
	}
	probeCoverage, err := transcript.SummarizeSemanticProbes([]string{completionProbe})
	if err != nil {
		t.Fatal(err)
	}
	wellFormed := processResult(0, "", "")
	wellFormed.IOTranscript = semanticTranscript(t, completionProbe)
	wellFormed.ChoiceTrace = trace
	malformed := wellFormed
	malformed.IOTranscript = backend.Transcript{Bytes: []byte("not an I/O transcript")}
	malformed.ChoiceTrace.Trace.Bytes = []byte("not a choice trace")
	malformedChoices := malformed
	malformedChoices.IOTranscript = wellFormed.IOTranscript
	watchdog, cancelled := malformedChoices, malformedChoices
	watchdog.WatchdogTimeout = true
	cancelled.Cancelled = true
	exitCode := record.Uint64String(0)
	deadline := "execution_timeout"
	success := backend.Classification{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode, ArtifactKind: record.ArtifactSuccess, ReplayMode: record.ReplayExact}
	deadlock := backend.Classification{Domain: "target", Reason: "world_deadlock", Termination: "exit", ExitCode: &exitCode, ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}
	for _, test := range []struct {
		name     string
		result   backend.ExecutionResult
		terminal record.WorldTerminal
		mode     CoverageMode
		want     completedExecution
		// reason is the host failure reason; the error text of each reason is
		// pinned by TestCompletionFaultsKeepReasonPrecedenceAndEvidence.
		reason string
		cause  string
	}{
		{name: "no coverage ignores malformed evidence", result: malformed, mode: CoverageNone, want: completedExecution{coverage: noCoverage, choiceFeatures: []string{}, outcome: success}},
		{name: "unset coverage", result: malformed, want: completedExecution{coverage: noCoverage, choiceFeatures: []string{}, outcome: success}},
		{name: "semantic coverage", result: malformedChoices, mode: CoverageSemantic, want: completedExecution{coverage: probeCoverage, choiceFeatures: []string{}, outcome: success}},
		{name: "choice coverage", result: wellFormed, mode: CoverageChoice, want: completedExecution{coverage: noCoverage, choiceFeatures: features, choiceProjection: &projection, outcome: success}},
		{
			name: "semantic and choice coverage with a World terminal", result: wellFormed, terminal: record.WorldTerminal{Kind: string(world.TerminalDeadlock)}, mode: CoverageSemanticChoice,
			want: completedExecution{coverage: probeCoverage, choiceFeatures: features, choiceProjection: &projection, outcome: deadlock},
		},
		{name: "malformed semantic coverage", result: malformed, mode: CoverageSemantic, reason: "semantic_coverage", cause: "I/O transcript has invalid length 21"},
		{name: "malformed semantic coverage before malformed choices", result: malformed, mode: CoverageSemanticChoice, reason: "semantic_coverage", cause: "I/O transcript has invalid length 21"},
		{name: "malformed choices", result: malformedChoices, mode: CoverageSemanticChoice, reason: "choice_coverage", cause: "project choice coverage: malformed choice trace\ninvalid choice terminal values"},
		{name: "malformed semantic coverage of a watchdog kill", result: func() backend.ExecutionResult { killed := malformed; killed.WatchdogTimeout = true; return killed }(), mode: CoverageSemanticChoice, reason: "semantic_coverage", cause: "I/O transcript has invalid length 21"},
		{
			name: "watchdog kill projects no choices", result: watchdog, mode: CoverageSemanticChoice,
			want: completedExecution{coverage: probeCoverage, choiceFeatures: []string{}, outcome: backend.Classification{
				Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline, ArtifactKind: record.ArtifactWatchdogTimeout, ReplayMode: record.ReplayDiagnostic,
			}},
		},
		{
			name: "cancellation projects no choices", result: cancelled, mode: CoverageChoice,
			want: completedExecution{coverage: noCoverage, choiceFeatures: []string{}, outcome: backend.Classification{
				Domain: "runner", Reason: "runner_cancelled", Termination: "none", ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone,
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assessed, hostError := assessCompletion(t.Context(), &boundaryProvider{}, test.result, test.terminal, test.mode, prepared)
			if test.reason != "" {
				if hostError == nil || hostError.Reason != test.reason || hostError.Err == nil || hostError.Err.Error() != test.cause {
					t.Fatalf("assessCompletion() error = %v, want %s: %s", hostError, test.reason, test.cause)
				}
				return
			}
			if hostError != nil {
				t.Fatal(hostError)
			}
			if !reflect.DeepEqual(assessed, test.want) {
				t.Fatalf("assessCompletion() = %#v, want %#v", assessed, test.want)
			}
		})
	}
}

func TestClassifyStableTargetDiagnostics(t *testing.T) {
	for name, stderr := range map[string]string{
		"panic_or_runtime_fatal":         "panic: broken\n",
		"deterministic_deadlock":         "fatal error: all goroutines are asleep - deadlock!\n",
		"logical_test_timeout":           "panic: test timed out after 1m0s\n",
		"unsupported_deterministic_mode": "runtime: GOMADSEED does not support cgo or external linking\n",
	} {
		t.Run(name, func(t *testing.T) {
			outcome := backend.Classify(processResult(2, "", stderr), false, record.WorldTerminal{Kind: "none"})
			if outcome.Domain != "target" || outcome.Reason != name {
				t.Fatalf("outcome = %#v", outcome)
			}
		})
	}
}

func TestClassifyStructuredWorldFailures(t *testing.T) {
	for kind, reason := range map[string]string{
		"deadlock": "world_deadlock", "capacity": "world_capacity", "replay-divergence": "world_replay_divergence", "invalid-input": "world_invalid_input",
	} {
		t.Run(kind, func(t *testing.T) {
			outcome := backend.Classify(processResult(0, "", ""), false, record.WorldTerminal{Kind: kind, Detail: "detail"})
			if outcome.Domain != "target" || outcome.Reason != reason || outcome.Termination != "exit" || outcome.ExitCode == nil || *outcome.ExitCode != 0 {
				t.Fatalf("classify() = %#v", outcome)
			}
		})
	}
}
