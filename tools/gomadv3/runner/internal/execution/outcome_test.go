package execution

import (
	"crypto/sha256"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomadv3/evidence"
)

func TestClassifyOwnsRecordingAndReplaySemantics(t *testing.T) {
	exitTwo := evidence.Uint64String(2)
	exitZero := evidence.Uint64String(0)
	signal := "SIGTERM"
	deadline := "run_timeout"
	tests := map[string]struct {
		result    Result
		cancelled bool
		terminal  evidence.WorldTerminal
		want      Classification
	}{
		"cancelled": {
			cancelled: true,
			want:      Classification{Domain: "runner", Reason: "runner_cancelled", Termination: "none", ArtifactKind: evidence.ArtifactRunnerFailure, ReplayMode: evidence.ReplayNone},
		},
		"watchdog": {
			result: Result{WatchdogTimeout: true},
			want:   Classification{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline, ArtifactKind: evidence.ArtifactWatchdogTimeout, ReplayMode: evidence.ReplayDiagnostic},
		},
		"World deadlock": {
			result: Result{Termination: TerminationExit}, terminal: evidence.WorldTerminal{Kind: "deadlock"},
			want: Classification{Domain: "target", Reason: "world_deadlock", Termination: "exit", ExitCode: new(evidence.Uint64String), ArtifactKind: evidence.ArtifactTargetFailure, ReplayMode: evidence.ReplayExact},
		},
		"success": {
			result: Result{Termination: TerminationExit}, terminal: evidence.WorldTerminal{Kind: "none"},
			want: Classification{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitZero, ArtifactKind: evidence.ArtifactSuccess, ReplayMode: evidence.ReplayExact},
		},
		"signal": {
			result: Result{Termination: TerminationSignal, Signal: signal},
			want:   Classification{Domain: "target", Reason: "external_signal", Termination: "signal", Signal: &signal, ArtifactKind: evidence.ArtifactTargetFailure, ReplayMode: evidence.ReplayExact},
		},
		"panic": {
			result: failedResult("panic: broken\n"),
			want:   Classification{Domain: "target", Reason: "panic_or_runtime_fatal", Termination: "exit", ExitCode: &exitTwo, ArtifactKind: evidence.ArtifactTargetFailure, ReplayMode: evidence.ReplayExact},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := Classify(test.result, test.cancelled, test.terminal); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("Classify() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func failedResult(stderr string) Result {
	data := []byte(stderr)
	digest := sha256.Sum256(data)
	return Result{
		Termination: TerminationExit,
		ExitCode:    2,
		Stderr:      Output{Bytes: data, FullSHA256: digest, RetainedSHA256: digest},
	}
}
