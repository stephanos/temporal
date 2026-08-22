package execution

import (
	"strings"

	"go.temporal.io/server/tools/gomadv3/record"
	"go.temporal.io/server/tools/gomadv3/world"
)

type Classification struct {
	Domain       string
	Reason       string
	Termination  string
	ExitCode     *record.Uint64String
	Signal       *string
	Deadline     *string
	ArtifactKind string
	ReplayMode   string
}

func Classify(result Result, cancelled bool, terminal record.WorldTerminal) Classification {
	if cancelled || result.Cancelled {
		return Classification{Domain: "runner", Reason: "runner_cancelled", Termination: "none", ArtifactKind: record.ArtifactRunnerFailure, ReplayMode: record.ReplayNone}
	}
	if result.WatchdogTimeout {
		deadline := "execution_timeout"
		return Classification{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout", Deadline: &deadline, ArtifactKind: record.ArtifactWatchdogTimeout, ReplayMode: record.ReplayDiagnostic}
	}
	if reason := worldFailureReason(terminal.Kind); reason != "" {
		classified := Classification{Domain: "target", Reason: reason, ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}
		setTargetTermination(&classified, result)
		return classified
	}
	if result.Termination == TerminationExit && result.ExitCode == 0 {
		reason := "success"
		if terminal.Kind == string(world.TerminalIdle) {
			reason = "world_idle"
		}
		exitCode := record.Uint64String(0)
		return Classification{Domain: "success", Reason: reason, Termination: "exit", ExitCode: &exitCode, ArtifactKind: record.ArtifactSuccess, ReplayMode: record.ReplayExact}
	}
	classified := Classification{Domain: "target", Reason: diagnosticReason(result.Stderr.Bytes), ArtifactKind: record.ArtifactTargetFailure, ReplayMode: record.ReplayExact}
	setTargetTermination(&classified, result)
	if result.Termination == TerminationSignal {
		classified.Reason = "external_signal"
	}
	return classified
}

func setTargetTermination(classified *Classification, result Result) {
	if result.Termination == TerminationSignal {
		classified.Termination = "signal"
		classified.Signal = &result.Signal
		return
	}
	classified.Termination = "exit"
	exitCode := record.Uint64String(result.ExitCode)
	classified.ExitCode = &exitCode
}

func diagnosticReason(stderr []byte) string {
	diagnostic := string(stderr)
	switch {
	case strings.HasPrefix(diagnostic, "runtime: GOMADSEED does not support cgo or external linking"):
		return "unsupported_deterministic_mode"
	case strings.HasPrefix(diagnostic, "fatal error: all goroutines are asleep - deadlock!"):
		return "deterministic_deadlock"
	case strings.HasPrefix(diagnostic, "panic: test timed out after"):
		return "logical_test_timeout"
	case strings.HasPrefix(diagnostic, "fatal error: GOMAD_CAPABILITY_DENIED"):
		return "denied_capability"
	case strings.HasPrefix(diagnostic, "panic:") || strings.HasPrefix(diagnostic, "fatal error:"):
		return "panic_or_runtime_fatal"
	default:
		return "nonzero_exit"
	}
}

func worldFailureReason(kind string) string {
	switch world.TerminalKind(kind) {
	case world.TerminalDeadlock:
		return "world_deadlock"
	case world.TerminalCapacity:
		return "world_capacity"
	case world.TerminalReplayDivergence:
		return "world_replay_divergence"
	case world.TerminalInvalidInput:
		return "world_invalid_input"
	default:
		return ""
	}
}
