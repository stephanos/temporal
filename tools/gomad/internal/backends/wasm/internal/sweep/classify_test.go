package sweep

import (
	"encoding/json"
	"strings"
	"testing"
)

func events(t *testing.T, values ...Event) []byte {
	t.Helper()
	var lines []string
	for _, value := range values {
		value.Schema = "gomad.explore-event/v3"
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(encoded))
	}
	return []byte(strings.Join(lines, "\n") + "\n")
}

func exitCode(code uint64) *uint64 { return &code }

func TestClassifyFixtureObservations(t *testing.T) {
	t.Parallel()
	const test = "TestFrontendSystemInfo"
	const artifactPath = "/work/runs/TestFrontendSystemInfo-1/artifacts/v1/run/0"
	result := func(classification string) Event {
		return Event{Type: "result", Classification: classification, Attempted: 1}
	}
	artifactEvent := func(classification string) Event {
		return Event{Type: "artifact", Classification: classification, Path: artifactPath}
	}
	errorEvent := func(classification, message string) Event {
		return Event{Type: "error", Classification: classification, Message: message}
	}
	for _, fixture := range []struct {
		name        string
		exit        int
		execError   string
		stdout      []byte
		stderr      string
		facts       *ArtifactFacts
		class       Class
		reason      string // required substring; empty requires an empty reason
		skipped     bool
		subtestFail int
	}{
		{name: "pass", stdout: events(t, result("success"), artifactEvent("success")),
			facts: &ArtifactFacts{Stdout: []byte("=== RUN   TestFrontendSystemInfo\n--- PASS: TestFrontendSystemInfo (0.01s)\nPASS\n")}, class: ClassPass},
		{name: "pass-suite-with-subtests", stdout: events(t, result("success"), artifactEvent("success")),
			facts: &ArtifactFacts{Stdout: []byte("--- PASS: TestFrontendSystemInfo (9.00s)\n    --- PASS: TestFrontendSystemInfo/A (1.00s)\n    --- SKIP: TestFrontendSystemInfo/B (0.00s)\nPASS\n")}, class: ClassPass},
		{name: "skip", stdout: events(t, result("success"), artifactEvent("success")),
			facts: &ArtifactFacts{Stdout: []byte("--- SKIP: TestFrontendSystemInfo (0.00s)\n    frontend_test.go:12: needs cassandra\nPASS\n")}, class: ClassPass, skipped: true},
		{name: "zero-match-is-not-a-pass", exit: 0, stdout: events(t, result("success"), artifactEvent("success")),
			facts: &ArtifactFacts{Stdout: []byte("testing: warning: no tests to run\nPASS\n")}, class: ClassBuildFailure, reason: "no tests to run"},
		{name: "other-test-passed", stdout: events(t, result("success"), artifactEvent("success")),
			facts: &ArtifactFacts{Stdout: []byte("--- PASS: TestFrontendSystemInfoOther (0.00s)\nPASS\n")}, class: ClassInfrastructure, reason: "verdict-unverifiable"},
		{name: "truncated-success", stdout: events(t, result("success"), artifactEvent("success")),
			facts: &ArtifactFacts{Stdout: []byte("=== RUN"), StdoutTruncated: true}, class: ClassInfrastructure, reason: "stdout truncated"},
		{name: "success-without-artifact", stdout: events(t, result("success")), class: ClassInfrastructure, reason: "unverifiable"},
		{name: "guest-panic", exit: 1, stdout: events(t, result("target_failure"), artifactEvent("target_failure")),
			facts: &ArtifactFacts{Kind: "target_failure", Reason: "panic_or_runtime_fatal", Termination: "exit", ExitCode: exitCode(2), Stderr: []byte("panic: listen tcp 127.0.0.1:43123: bind: address in use\n\ngoroutine 7 [running]:\n")},
			class: ClassGuestFailure, reason: "panic_or_runtime_fatal: panic: listen tcp N.N:N: bind: address in use"},
		{name: "guest-testify", exit: 1, stdout: events(t, result("target_failure"), artifactEvent("target_failure")),
			facts: &ArtifactFacts{Kind: "target_failure", Reason: "nonzero_exit", ExitCode: exitCode(1), Stdout: []byte("    --- FAIL: TestFrontendSystemInfo/TestX (2.00s)\n        Error Trace:\t/src/tests/x_test.go:42\n        Error:      \tReceived unexpected error:\n--- FAIL: TestFrontendSystemInfo (3.00s)\nFAIL\n")},
			class: ClassGuestFailure, reason: "exit N: Received unexpected error:", subtestFail: 1},
		{name: "guest-fail-line", exit: 1, stdout: events(t, result("target_failure"), artifactEvent("target_failure")),
			facts: &ArtifactFacts{Kind: "target_failure", Reason: "nonzero_exit", ExitCode: exitCode(1), Stdout: []byte("    --- FAIL: TestFrontendSystemInfo/TestY (0.10s)\n--- FAIL: TestFrontendSystemInfo (0.20s)\n")},
			class: ClassGuestFailure, reason: "--- FAIL: TestFrontendSystemInfo/TestY", subtestFail: 1},
		{name: "soft-assertion", exit: 1, stdout: events(t, result("target_failure"), artifactEvent("target_failure")),
			facts: &ArtifactFacts{Kind: "target_failure", Reason: "soft_assertion", ExitCode: exitCode(0)}, class: ClassGuestFailure, reason: "soft_assertion"},
		{name: "world-capacity", exit: 1, stdout: events(t, result("target_failure"), artifactEvent("target_failure")),
			facts: &ArtifactFacts{Kind: "target_failure", Reason: "world_capacity"}, class: ClassCapacityExceeded, reason: "world capacity"},
		{name: "world-replay-divergence", exit: 1, stdout: events(t, result("target_failure"), artifactEvent("target_failure")),
			facts: &ArtifactFacts{Kind: "target_failure", Reason: "world_replay_divergence"}, class: ClassReplayDivergence, reason: "world replay divergence"},
		{name: "watchdog", exit: 1, stdout: events(t, Event{Type: "result", Classification: "watchdog_observation", Failures: 1, Watchdogs: 1}, artifactEvent("watchdog_observation")),
			facts: &ArtifactFacts{Kind: "watchdog_timeout", Termination: "timeout"}, class: ClassTimeout, reason: "execution-timeout"},
		{name: "replay-divergence-result", exit: 1, stdout: events(t, Event{Type: "result", Classification: "replay_divergence", ReplayDivergences: 1}, artifactEvent("replay_divergence")),
			facts: &ArtifactFacts{Reason: "replay_mismatch"}, class: ClassReplayDivergence, reason: "diverged"},
		{name: "backend-divergence", exit: 3, stdout: events(t, errorEvent("runner_failure", "gomad Runner/host failure: target_supervision: backend divergence: choice tape exhausted at ordinal 1234")),
			class: ClassReplayDivergence, reason: "choice tape exhausted at ordinal N"},
		{name: "refused-import-execute", exit: 3, stdout: events(t, errorEvent("runner_failure", "gomad Runner/host failure: target_supervision: backend unsupported: sock_accept: socket accept is not modeled")),
			class: ClassRefusedImport, reason: "execute: sock_accept: socket accept is not modeled"},
		{name: "refused-import-prepare", exit: 2, stdout: events(t, errorEvent("unsupported_target", "target requires os/exec capability")),
			class: ClassRefusedImport, reason: "prepare: target requires os/exec capability"},
		{name: "refused-import-prepare-invalid-input", exit: 2, stdout: events(t, errorEvent("invalid_input", "WASM module has unsupported import env.foo")),
			class: ClassRefusedImport, reason: "prepare: WASM module has unsupported import env.foo"},
		{name: "capacity-backend", exit: 3, stdout: events(t, errorEvent("runner_failure", "gomad Runner/host failure: target_supervision: backend capacity: fuel exhausted after 1125899906842624 units")),
			class: ClassCapacityExceeded, reason: "fuel exhausted after N units"},
		{name: "capacity-choice-trace", exit: 3, stdout: events(t, errorEvent("runner_failure", "gomad Runner/host failure: choice_trace_overflow: trace exceeds 67108864 bytes")),
			class: ClassCapacityExceeded, reason: "choice trace overflow"},
		{name: "capacity-artifact", exit: 3, stdout: events(t, errorEvent("capacity", "success artifact bytes exceed 4294967296")),
			class: ClassCapacityExceeded, reason: "artifact or journal capacity"},
		{name: "trap", exit: 3, stdout: events(t, errorEvent("runner_failure", "gomad Runner/host failure: target_supervision: backend trap: wasm trap: out of bounds memory access")),
			class: ClassGuestFailure, reason: "trap: wasm trap: out of bounds memory access"},
		{name: "overall-timeout", exit: 3, stdout: events(t, errorEvent("overall_timeout", "gomad Runner/host failure: overall_timeout: context deadline exceeded")),
			class: ClassTimeout, reason: "overall-timeout"},
		{name: "build-failure", exit: 2, stdout: events(t, errorEvent("invalid_input", "WASM compiler command failed: # go.temporal.io/server/tests/mixedbrain\n/abs/checkout/tests/mixedbrain/build_util.go:17:2: undefined: exec.LookPath\n")),
			class: ClassBuildFailure, reason: "build_util.go: undefined: exec.LookPath"},
		{name: "backend-infrastructure", exit: 3, stdout: events(t, errorEvent("runner_failure", "gomad Runner/host failure: target_supervision: backend infrastructure: helper exited")),
			class: ClassInfrastructure, reason: "backend infrastructure: helper exited"},
		{name: "runner-failure-other", exit: 3, stdout: events(t, errorEvent("runner_failure", "gomad Runner/host failure: runs_append: disk full")),
			class: ClassInfrastructure, reason: "runner_failure: gomad Runner/host failure: runs_append: disk full"},
		{name: "no-result", exit: 3, stderr: "gomad: helper handshake failed\n", class: ClassInfrastructure, reason: "without a result event: gomad: helper handshake failed"},
		{name: "not-json", exit: 0, stdout: []byte("gomad: classification=success\n"), class: ClassInfrastructure, reason: "explore output"},
		{name: "exec-error", exit: -1, execError: "signal: killed", class: ClassInfrastructure, reason: "gomad subprocess: signal: killed"},
		{name: "unreadable-artifact", exit: 1, stdout: events(t, result("target_failure"), artifactEvent("target_failure")),
			facts: &ArtifactFacts{ReadError: "artifact directory mode is 0755, want 0700"}, class: ClassGuestFailure, reason: "artifact unreadable"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			observation := Observation{Test: test, Exit: fixture.exit, ExecError: fixture.execError, Stdout: fixture.stdout, Stderr: []byte(fixture.stderr), Artifacts: map[string]ArtifactFacts{}}
			if fixture.facts != nil {
				observation.Artifacts[artifactPath] = *fixture.facts
			}
			outcome := Classify(observation)
			if outcome.Class != fixture.class {
				t.Fatalf("class %s (%q), want %s", outcome.Class, outcome.Reason, fixture.class)
			}
			if fixture.reason == "" && outcome.Reason != "" || !strings.Contains(outcome.Reason, fixture.reason) {
				t.Fatalf("reason %q, want it to contain %q", outcome.Reason, fixture.reason)
			}
			if outcome.Class != ClassPass && outcome.Reason == "" {
				t.Fatal("non-pass outcome without a reason")
			}
			if outcome.Skipped != fixture.skipped {
				t.Fatalf("skipped %t, want %t", outcome.Skipped, fixture.skipped)
			}
			if fixture.subtestFail != 0 && (outcome.Subtests == nil || outcome.Subtests.Fail != fixture.subtestFail) {
				t.Fatalf("subtests %+v, want %d failures", outcome.Subtests, fixture.subtestFail)
			}
		})
	}
}

func TestEveryClassIsReachableAndListed(t *testing.T) {
	t.Parallel()
	if len(Classes) != 9 {
		t.Fatalf("classes %v: want the seven spec classes, replay divergence and not-run", Classes)
	}
	seen := map[Class]bool{}
	for _, class := range Classes {
		if seen[class] {
			t.Fatalf("duplicate class %s", class)
		}
		seen[class] = true
	}
}

func TestNormalizeGroupsEquivalentDiagnostics(t *testing.T) {
	t.Parallel()
	for _, pair := range [][2]string{
		{"panic: runtime error at 0xc000123456 goroutine 17", "panic: runtime error at 0xc000999999 goroutine 3"},
		{"open /tmp/a/b/sqlite.db: no such file", "open /var/x/sqlite.db: no such file"},
		{"workflow 4f1c2d3e-0000-4000-8000-1234567890ab timed out after 30s", "workflow 00000000-1111-4222-8333-444444444444 timed out after 5m"},
		{"module sha256:" + strings.Repeat("ab", 32) + " mismatch", "module sha256:" + strings.Repeat("cd", 32) + " mismatch"},
	} {
		if a, b := Normalize(pair[0]), Normalize(pair[1]); a != b {
			t.Fatalf("Normalize(%q)=%q differs from Normalize(%q)=%q", pair[0], a, pair[1], b)
		}
	}
	if got := Normalize("--- FAIL: TestSuite/TestMethod (1.00s)"); got != "--- FAIL: TestSuite/TestMethod (N)" {
		t.Fatalf("test names must survive normalization: %q", got)
	}
	if got := Normalize(strings.Repeat("x ", 400)); len([]rune(got)) > maximumReasonRunes+1 {
		t.Fatalf("reason not bounded: %d runes", len([]rune(got)))
	}
}
