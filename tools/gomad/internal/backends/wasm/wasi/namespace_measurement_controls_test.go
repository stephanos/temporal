package wasi

import (
	"crypto/sha256"
	"encoding/json"
	"strings"
	"testing"
)

func TestNamespaceMeasurementPerSeedMedianControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func([]namespaceMeasurementSample, []namespaceMeasurementSample)
		want   bool
	}{
		{"at-quarter-reduction", func(_, _ []namespaceMeasurementSample) {}, true},
		{"one-seed-over-bound", func(_, after []namespaceMeasurementSample) {
			for i := 5; i < 10; i++ {
				after[i].AuthorizedNanos++
			}
		}, false},
		{"one-outlier-not-median", func(_, after []namespaceMeasurementSample) { after[0].AuthorizedNanos = 1000 }, true},
		{"different-seed", func(_, after []namespaceMeasurementSample) { after[0].Seed++ }, false},
		{"different-repetition", func(_, after []namespaceMeasurementSample) { after[0].Repetition++ }, false},
		{"zero-timing", func(_, after []namespaceMeasurementSample) { after[0].AuthorizedNanos = 0 }, false},
		{"nonempty-zero-count-map", func(_, after []namespaceMeasurementSample) {
			after[0].GoFilesystemCalls = map[string]uint64{"fd_write": 0}
		}, false},
		{"served-go-filesystem", func(_, after []namespaceMeasurementSample) {
			after[0].GoFilesystemCalls = map[string]uint64{"fd_pwrite": 1}
		}, false},
		{"missing-real-counter", func(_, after []namespaceMeasurementSample) { after[0].CounterPresent = false }, false},
		{"changed-wasi-choice-output", func(_, after []namespaceMeasurementSample) { after[0].EvidenceSHA256 = strings.Repeat("f", 64) }, false},
		{"changed-call-count", func(_, after []namespaceMeasurementSample) { after[0].Calls++ }, false},
		{"changed-time-advance", func(_, after []namespaceMeasurementSample) { after[0].TimeAdvances++ }, false},
		{"changed-environment", func(_, after []namespaceMeasurementSample) { after[0].EnvironmentIdentity = "other" }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var before, after []namespaceMeasurementSample
			for _, seed := range []uint64{0, 1000003, 2000006, 3000009} {
				for repetition := range 5 {
					sample := namespaceMeasurementSample{Role: "baseline", Seed: seed, Repetition: repetition, AuthorizedNanos: 100, EvidenceSHA256: strings.Repeat("e", 64), EnvironmentIdentity: "environment", Calls: 10, TimeAdvances: 3}
					before = append(before, sample)
					sample.Role, sample.AuthorizedNanos, sample.CounterPresent = "candidate", 75, true
					after = append(after, sample)
				}
			}
			test.mutate(before, after)
			if err := namespaceMeasurementCompare(before, after); (err == nil) != test.want {
				t.Fatalf("comparison=%v want accepted=%v", err, test.want)
			}
		})
	}
}

func TestNamespaceMeasurementActualCounterControls(t *testing.T) {
	for _, test := range []struct {
		name, data      string
		candidate, want bool
	}{
		{"historical-absence", `{}`, false, true},
		{"candidate-empty", `{"GoFilesystemCalls":{}}`, true, true},
		{"candidate-missing", `{}`, true, false},
		{"candidate-null", `{"GoFilesystemCalls":null}`, true, false},
		{"candidate-zero-entry", `{"GoFilesystemCalls":{"fd_write":0}}`, true, false},
		{"candidate-served", `{"GoFilesystemCalls":{"fd_write":1}}`, true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := namespaceMeasurementCounter([]byte(test.data), test.candidate)
			if (err == nil) != test.want {
				t.Fatalf("actual counter=%v want admitted=%v", err, test.want)
			}
		})
	}
}

func TestNamespaceMeasurementOriginalRunControls(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Result, *Request, *string)
		want   bool
	}{
		{"successful-actual-warm-run", func(*Result, *Request, *string) {}, true},
		{"original-run-error", func(_ *Result, _ *Request, e *string) { *e = "original transport error" }, false},
		{"failed-suite", func(r *Result, _ *Request, _ *string) {
			r.Stdout = []byte("=== RUN   TestActivityTestSuite\n--- FAIL: TestActivityTestSuite")
		}, false},
		{"missing-selection", func(r *Result, _ *Request, _ *string) { r.Stdout = []byte("PASS") }, false},
		{"unreaped", func(r *Result, _ *Request, _ *string) { r.Reaped = false }, false},
		{"cancelled", func(r *Result, _ *Request, _ *string) { r.Cancelled = true }, false},
		{"watchdog", func(r *Result, _ *Request, _ *string) { r.WatchdogTimeout = true }, false},
		{"wrong-module", func(r *Result, _ *Request, _ *string) { r.Started.ModuleSHA256 = strings.Repeat("f", 64) }, false},
		{"wrong-engine", func(r *Result, _ *Request, _ *string) { r.Started.Engine.Configuration += "-other" }, false},
		{"cold-cache", func(r *Result, _ *Request, _ *string) { r.Telemetry.CacheHit = false }, false},
		{"compiled-run", func(r *Result, _ *Request, _ *string) { r.Telemetry.CompileNanos = 1 }, false},
		{"missing-authorized-boundary", func(r *Result, _ *Request, _ *string) { r.AuthorizationToTerminalNanos = 0 }, false},
		{"raw-digest-changed", func(r *Result, _ *Request, _ *string) { r.TranscriptSHA256[0]++ }, false},
		{"invalid-raw-transcript-rehashed", func(r *Result, _ *Request, _ *string) {
			r.Transcript = []byte("invalid")
			r.TranscriptSHA256 = sha256.Sum256(r.Transcript)
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := testConfig()
			environment := newTestEnvironment(t, config)
			request := Request{ModuleSHA256: strings.Repeat("a", 64), Engine: PinnedEngine(), Config: config}
			code := uint32(0)
			result := Result{Termination: "exit", ExitCode: &code, Reaped: true, Started: &Started{Engine: request.Engine, ModuleSHA256: request.ModuleSHA256}, Telemetry: &Telemetry{CacheHit: true}, AuthorizationToTerminalNanos: 1, Stdout: []byte("=== RUN   TestActivityTestSuite\n--- PASS: TestActivityTestSuite"), Transcript: environment.Transcript(), TranscriptSHA256: environment.TranscriptSHA256()}
			runError := ""
			test.mutate(&result, &request, &runError)
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			if err := namespaceMeasurementResultCheck(result, data, request, false, runError); (err == nil) != test.want {
				t.Fatalf("original Run admission=%v want=%v", err, test.want)
			}
		})
	}
}

func TestNamespaceMeasurementRequestKeepsCommonImmutable(t *testing.T) {
	common := namespaceMeasurementCommon{Request: Request{Runtime: &RuntimeControl{Seed: 0}}}
	copy := namespaceMeasurementRequest(t, common)
	copy.Runtime.Seed = 1000003
	if common.Request.Runtime.Seed != 0 {
		t.Fatal("paired driver mutated immutable common seed")
	}
}
