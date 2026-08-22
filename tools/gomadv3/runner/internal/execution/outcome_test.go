package execution

import (
	"crypto/sha256"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
	"go.temporal.io/server/tools/gomadv3/record"
)

func TestClassifyOwnsRecordingAndReplaySemantics(t *testing.T) {
	exitTwo := record.Uint64String(2)
	exitZero := record.Uint64String(0)
	signal := "SIGTERM"
	deadline := "execution_timeout"
	tests := map[string]struct {
		result    Result
		cancelled bool
		terminal  record.WorldTerminal
		want      Classification
	}{
		"cancelled": {
			cancelled: true,
			want:      Classification{Domain: "runner", Reason: "runner_cancelled", Termination: "none", ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone},
		},
		"watchdog": {
			result: Result{WatchdogTimeout: true},
			want:   Classification{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline, ArtifactKind: record.ArtifactWatchdogTimeout, ReplayMode: record.ReplayDiagnostic},
		},
		"World deadlock": {
			result: Result{Termination: TerminationExit}, terminal: record.WorldTerminal{Kind: "deadlock"},
			want: Classification{Domain: "target", Reason: "world_deadlock", Termination: "exit", ExitCode: new(record.Uint64String), ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact},
		},
		"success": {
			result: Result{Termination: TerminationExit}, terminal: record.WorldTerminal{Kind: "none"},
			want: Classification{Domain: "success", Reason: "success", Termination: "exit", ExitCode: &exitZero, ArtifactKind: record.ArtifactSuccess, ReplayMode: record.ReplayExact},
		},
		"signal": {
			result: Result{Termination: TerminationSignal, Signal: signal},
			want:   Classification{Domain: "target", Reason: "external_signal", Termination: "signal", Signal: &signal, ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact},
		},
		"panic": {
			result: failedResult("panic: broken\n"),
			want:   Classification{Domain: "target", Reason: "panic_or_runtime_fatal", Termination: "exit", ExitCode: &exitTwo, ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact},
		},
		"guarded capability": {
			result: failedResult("fatal error: GOMAD_CAPABILITY_DENIED\n\nstack\n"),
			want:   Classification{Domain: "target", Reason: "denied_capability", Termination: "exit", ExitCode: &exitTwo, ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact},
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
		Stderr:      hostexec.Output{Bytes: data, FullSHA256: digest, RetainedSHA256: digest},
	}
}
