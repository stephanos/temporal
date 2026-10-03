package runner

import (
	"bytes"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/deterministicio"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
	"go.temporal.io/server/tools/gomad3/world"
)

func TestAssessWorldValidatesTheRecordAgainstItsSeed(t *testing.T) {
	recording := completionWorldRecord(t, 7)
	decoded, err := world.DecodeRecording(recording)
	if err != nil {
		t.Fatal(err)
	}
	composed, err := execution.ComposeRecording(decoded, 1<<20)
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
		want   execution.Bundle
		cause  string
	}{
		{name: "no record", seed: 7, limit: 1 << 20, want: noneWorldBundle()},
		{name: "valid record", record: recording, seed: 7, limit: 1 << 20, want: composed},
		{name: "malformed record", record: malformed, seed: 7, limit: 1 << 20, cause: "decode World terminal: invalid character '|' after object key:value pair"},
		{name: "seed mismatch", record: recording, seed: 8, limit: 1 << 20, cause: "World record seed or schema does not match seed 8"},
		{name: "transition limit", record: recording, seed: 7, cause: "World transition limit must be positive"},
		{name: "malformed record before seed mismatch", record: malformed, seed: 8, limit: 1 << 20, cause: "decode World terminal: invalid character '|' after object key:value pair"},
	} {
		t.Run(test.name, func(t *testing.T) {
			captured := bytes.Clone(test.record)
			bundle, err := assessWorld(execution.Result{WorldRecord: captured}, test.seed, test.limit)
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
	noCoverage, err := deterministicio.SummarizeSemanticProbes(nil)
	if err != nil {
		t.Fatal(err)
	}
	probeCoverage, err := deterministicio.SummarizeSemanticProbes([]string{completionProbe})
	if err != nil {
		t.Fatal(err)
	}
	wellFormed := processResult(0, "", "")
	wellFormed.IOTranscript = semanticTranscript(t, completionProbe)
	wellFormed.ChoiceTrace = trace
	malformed := wellFormed
	malformed.IOTranscript = deterministicio.Transcript{Bytes: []byte("not an I/O transcript")}
	malformed.ChoiceTrace.Trace.Bytes = []byte("not a choice trace")
	malformedChoices := malformed
	malformedChoices.IOTranscript = wellFormed.IOTranscript
	watchdog, cancelled := malformedChoices, malformedChoices
	watchdog.WatchdogTimeout = true
	cancelled.Cancelled = true
	exitCode := record.Uint64String(0)
	deadline := "execution_timeout"
	success := execution.Classification{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitCode, ArtifactKind: record.ArtifactSuccess, ReplayMode: record.ReplayExact}
	deadlock := execution.Classification{Domain: "target", Reason: "world_deadlock", Termination: "exit", ExitCode: &exitCode, ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}
	for _, test := range []struct {
		name     string
		result   execution.Result
		terminal record.WorldTerminal
		mode     CoverageMode
		want     completedExecution
		reason   string
		cause    string
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
		{name: "malformed semantic coverage of a watchdog kill", result: func() execution.Result { killed := malformed; killed.WatchdogTimeout = true; return killed }(), mode: CoverageSemanticChoice, reason: "semantic_coverage", cause: "I/O transcript has invalid length 21"},
		{
			name: "watchdog kill projects no choices", result: watchdog, mode: CoverageSemanticChoice,
			want: completedExecution{coverage: probeCoverage, choiceFeatures: []string{}, outcome: execution.Classification{
				Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline, ArtifactKind: record.ArtifactWatchdogTimeout, ReplayMode: record.ReplayDiagnostic,
			}},
		},
		{
			name: "cancellation projects no choices", result: cancelled, mode: CoverageChoice,
			want: completedExecution{coverage: noCoverage, choiceFeatures: []string{}, outcome: execution.Classification{
				Domain: "runner", Reason: "runner_cancelled", Termination: "none", ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone,
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assessed, hostError := assessCompletion(test.result, test.terminal, test.mode, prepared)
			if test.reason != "" {
				if hostError == nil || hostError.Reason != test.reason || hostError.Err.Error() != test.cause {
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
