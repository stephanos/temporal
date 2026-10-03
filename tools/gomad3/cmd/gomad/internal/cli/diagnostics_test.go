package cli

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/qualification"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

func TestQualifyDiagnosticsRetainsFreshPairAndReplaysSuccesses(t *testing.T) {
	for _, replay := range []bool{false, true} {
		t.Run(fmt.Sprint(replay), func(t *testing.T) {
			directory := t.TempDir()
			dependencies := qualificationDependencies(t)
			calls, replayCalls := 0, 0
			dependencies.run = func(_ context.Context, config runner.CampaignSpec) (runner.CampaignResult, error) {
				if !config.Diagnostics || config.ChoiceTraceLimit != 8<<20 || config.Coverage != runner.CoverageSemanticChoice {
					t.Fatalf("diagnostic config = %+v", config)
				}
				data := make([]byte, 160)
				copy(data, []byte{'G', 'O', 'M', 'A', 'D', 'D', 'G', 1})
				binary.BigEndian.PutUint32(data[8:12], 1)
				data[12] = 1
				binary.BigEndian.PutUint64(data[16:24], 160)
				binary.BigEndian.PutUint64(data[24:32], 160)
				binary.BigEndian.PutUint64(data[32:40], 1)
				data[143] = byte(calls)
				trace, err := choice.DecodeDiagnosticTrace(data)
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(directory, fmt.Sprintf("trace-%d.bin", calls))
				if err := os.WriteFile(path, data, 0o600); err != nil {
					t.Fatal(err)
				}
				evidence := qualificationEvidence(7)
				evidence.Diagnostics = &runner.DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.SHA256FromSum(trace.SHA256), Records: 1}
				summary := runner.CampaignResult{CampaignPath: fmt.Sprintf("/campaign-%d", calls), ExecutionEvidence: &evidence, Diagnostics: &runner.DiagnosticTraceReference{Path: path, DiagnosticEvidence: *evidence.Diagnostics}}
				if replay {
					summary.RetainedSuccesses = 1
					summary.SuccessArtifacts = []string{fmt.Sprintf("/success-%d", calls)}
				}
				calls++
				return summary, nil
			}
			dependencies.replay = func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
				replayCalls++
				return runner.ReplayResult{Match: true, ChoiceReplayStatus: runner.ChoiceReplayExact}, nil
			}
			dependencies.write = qualification.WriteQualificationReport
			arguments := []string{"--diagnostics", "--seed=7", "--artifacts=" + directory, "go-run", "./fixture"}
			if replay {
				arguments = append([]string{"--replay-successes", "--success-limit=1", "--success-bytes=1MiB"}, arguments...)
			}
			var stdout, stderr bytes.Buffer
			status := runQualifyWith(arguments, &stdout, &stderr, dependencies)
			if status != 1 || !strings.Contains(stdout.String(), "diagnostic-first-ordinal=0 fields=runtime_cheap_rand_draws") {
				t.Fatalf("status %d stdout %s stderr %s", status, stdout.String(), stderr.String())
			}
			if calls != 2 || replay && replayCalls != 2 || !replay && replayCalls != 0 {
				t.Fatalf("calls %d replay calls %d", calls, replayCalls)
			}
			paths, err := filepath.Glob(filepath.Join(directory, "qualifications", "v1", "*.json"))
			if err != nil || len(paths) != 1 {
				t.Fatalf("reports %v: %v", paths, err)
			}
			report, err := qualification.OpenQualificationReport(paths[0])
			if err != nil {
				t.Fatal(err)
			}
			if report.Deterministic || report.FirstDivergence != "diagnostics" || report.DiagnosticDivergence == nil || len(report.Executions) != 2 {
				t.Fatalf("report %+v", report)
			}
			for _, run := range report.Executions {
				if run.Diagnostics == nil {
					t.Fatal("trace reference missing")
				}
				if _, err := choice.ReadDiagnosticTrace(run.Diagnostics.Path); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestExploreDiagnosticsRejectsForcedPrefixesBeforePreparation(t *testing.T) {
	var stdout, stderr bytes.Buffer
	status := hostApplication().runExplore([]string{"--diagnostics", "--strategy=choice-exploration", "--max-executions=2", "--max-choice-depth=1", "--max-exploration-bytes=1MiB", "go-run", "./fixture"}, &stdout, &stderr)
	if status != 2 || !strings.Contains(stderr.String(), "forced-prefix exploration is unsupported") {
		t.Fatalf("status %d: %s", status, stderr.String())
	}
}

func TestQualifyDiagnosticsRetainsInterruptedReports(t *testing.T) {
	for _, cancellation := range []bool{false, true} {
		for _, interruptedFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("cancel=%t/first=%t", cancellation, interruptedFirst), func(t *testing.T) {
				directory := t.TempDir()
				data := make([]byte, 160)
				copy(data, []byte{'G', 'O', 'M', 'A', 'D', 'D', 'G', 1})
				binary.BigEndian.PutUint32(data[8:12], 1)
				data[12] = 1
				binary.BigEndian.PutUint64(data[16:24], 160)
				binary.BigEndian.PutUint64(data[24:32], 160)
				binary.BigEndian.PutUint64(data[32:40], 1)
				trace, err := choice.DecodeDiagnosticTrace(data)
				if err != nil {
					t.Fatal(err)
				}
				tracePath := filepath.Join(directory, "complete.bin")
				if err := os.WriteFile(tracePath, data, 0o600); err != nil {
					t.Fatal(err)
				}
				dependencies := qualificationDependencies(t)
				calls := 0
				dependencies.run = func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
					evidence := qualificationEvidence(7)
					evidence.Environment = append(evidence.Environment, record.Environment{Name: choice.DiagnosticProfileEnvironment, Value: choice.DiagnosticProfile})
					interrupted := calls == 0 && interruptedFirst || calls == 1 && !interruptedFirst
					summary := runner.CampaignResult{CampaignPath: fmt.Sprintf("/campaign-%d", calls), ExecutionEvidence: &evidence}
					calls++
					if interrupted {
						summary.Artifacts = []string{"/failure"}
						if cancellation {
							evidence.Outcome = runner.OutcomeEvidence{Domain: "runner", Reason: "runner_cancelled", Termination: "none"}
							return summary, &runner.HostError{Reason: "cancelled", Err: context.Canceled}
						}
						evidence.Outcome = runner.OutcomeEvidence{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout"}
					} else {
						evidence.Diagnostics = &runner.DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.SHA256FromSum(trace.SHA256), Records: 1}
						summary.Diagnostics = &runner.DiagnosticTraceReference{Path: tracePath, DiagnosticEvidence: *evidence.Diagnostics}
					}
					return summary, nil
				}
				dependencies.replay = func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
					return runner.ReplayResult{Match: true, Diagnostic: true}, nil
				}
				dependencies.write = qualification.WriteQualificationReport
				var stdout, stderr bytes.Buffer
				status := runQualifyWith([]string{"--diagnostics", "--seed=7", "--artifacts=" + directory, "go-run", "./fixture"}, &stdout, &stderr, dependencies)
				wantStatus := 1
				if cancellation {
					wantStatus = 3
				}
				if status != wantStatus || !strings.Contains(stdout.String(), "qualification-") {
					t.Fatalf("status %d stdout %s stderr %s", status, stdout.String(), stderr.String())
				}
				paths, err := filepath.Glob(filepath.Join(directory, "qualifications", "v1", "*.json"))
				if err != nil || len(paths) != 1 {
					t.Fatalf("reports %v: %v", paths, err)
				}
				report, err := qualification.OpenQualificationReport(paths[0])
				if err != nil {
					t.Fatal(err)
				}
				if report.Qualified || report.DiagnosticDivergence != nil || cancellation && (report.Failure == nil || report.Failure.Classification != "cancelled") {
					t.Fatalf("report %+v", report)
				}
				available, unavailable := 0, 0
				for _, run := range report.Executions {
					switch run.DiagnosticStatus {
					case qualification.DiagnosticComplete:
						available++
						if run.Diagnostics == nil || run.Diagnostics.Path != tracePath {
							t.Fatalf("trace reference %+v", run.Diagnostics)
						}
					case qualification.DiagnosticUnavailable:
						unavailable++
						if run.Diagnostics != nil || run.ArtifactPath != "/failure" {
							t.Fatalf("interrupted run %+v", run)
						}
					default:
						t.Fatalf("status %q", run.DiagnosticStatus)
					}
				}
				if unavailable != 1 || (!cancellation || !interruptedFirst) && available != 1 {
					t.Fatalf("available %d unavailable %d", available, unavailable)
				}
			})
		}
	}
}
