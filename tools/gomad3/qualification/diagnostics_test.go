package qualification

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"

	"go.temporal.io/server/tools/gomad3/choice"
	"go.temporal.io/server/tools/gomad3/record"
	"go.temporal.io/server/tools/gomad3/runner"
)

func qualificationDiagnostic(t *testing.T, draw byte) *runner.DiagnosticTraceReference {
	t.Helper()
	data := make([]byte, 160)
	copy(data, []byte{'G', 'O', 'M', 'A', 'D', 'D', 'G', 1})
	binary.BigEndian.PutUint32(data[8:12], 1)
	data[12] = 1
	binary.BigEndian.PutUint64(data[16:24], 160)
	binary.BigEndian.PutUint64(data[24:32], 160)
	binary.BigEndian.PutUint64(data[32:40], 1)
	data[143] = draw
	trace, err := choice.DecodeDiagnosticTrace(data)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "diagnostic.bin")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return &runner.DiagnosticTraceReference{Path: path, DiagnosticEvidence: runner.DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.SHA256FromSum(trace.SHA256), Records: 1}}
}

func TestDiagnosticsIdentifyFreshPairAfterEarlierOutputDivergence(t *testing.T) {
	plain, changed := qualificationDiagnostic(t, 0), qualificationDiagnostic(t, 1)
	baseline := successfulEvidence()
	baseline.Diagnostics = &plain.DiagnosticEvidence
	outputDifference := baseline
	outputDifference.Stdout.FullSHA256 = record.HashBytes([]byte("different output"))
	diagnosticDifference := baseline
	diagnosticDifference.Diagnostics = &changed.DiagnosticEvidence
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{
		{CampaignPath: "/first", Evidence: baseline, Diagnostics: plain},
		{CampaignPath: "/second", Evidence: outputDifference, Diagnostics: plain},
		{CampaignPath: "/third", Evidence: diagnosticDifference, Diagnostics: changed},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if report.FirstDivergence != "stdout.full_sha256" || report.DiagnosticDivergence == nil || report.DiagnosticDivergence.BaselineIteration != 1 || report.DiagnosticDivergence.Iteration != 3 || report.DiagnosticDivergence.Ordinal != 0 {
		t.Fatalf("report %+v", report)
	}
	path, err := WriteQualificationReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenQualificationReport(path); err != nil {
		t.Fatal(err)
	}
}

func TestQualificationRejectsChangedDiagnosticTraceAndEvidence(t *testing.T) {
	for _, corruption := range []string{"file", "evidence", "second evidence", "missing"} {
		t.Run(corruption, func(t *testing.T) {
			reference := qualificationDiagnostic(t, 0)
			evidence := successfulEvidence()
			evidence.Diagnostics = &reference.DiagnosticEvidence
			second := reference
			secondEvidence := evidence
			switch corruption {
			case "file":
				if err := os.WriteFile(reference.Path, []byte("truncated"), 0o600); err != nil {
					t.Fatal(err)
				}
			case "evidence":
				evidence.Diagnostics = &runner.DiagnosticEvidence{Profile: choice.DiagnosticProfile, SHA256: record.HashBytes([]byte("wrong")), Records: 1}
			case "second evidence":
				changed := qualificationDiagnostic(t, 1)
				secondEvidence.Diagnostics = &changed.DiagnosticEvidence
			case "missing":
				second = nil
			}
			if _, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{{CampaignPath: "/first", Evidence: evidence, Diagnostics: reference}, {CampaignPath: "/second", Evidence: secondEvidence, Diagnostics: second}}}); err == nil {
				t.Fatal("invalid diagnostic trace accepted")
			}
		})
	}
}

func TestDiagnosticsPreserveInterruptedQualificationEvidence(t *testing.T) {
	for _, outcome := range []runner.OutcomeEvidence{
		{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout"},
		{Domain: "runner", Reason: "runner_cancelled", Termination: "none"},
	} {
		for _, interruptedFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/first=%t", outcome.Reason, interruptedFirst), func(t *testing.T) {
				reference := qualificationDiagnostic(t, 0)
				complete := successfulEvidence()
				complete.Environment = append(complete.Environment, record.Environment{Name: choice.DiagnosticProfileEnvironment, Value: choice.DiagnosticProfile})
				complete.Diagnostics = &reference.DiagnosticEvidence
				interrupted := complete
				interrupted.Diagnostics = nil
				interrupted.Outcome = outcome
				runs := []QualificationExecution{{CampaignPath: "/complete", Evidence: complete, Diagnostics: reference}, {CampaignPath: "/interrupted", ArtifactPath: "/failure", Evidence: interrupted}}
				if interruptedFirst {
					runs[0], runs[1] = runs[1], runs[0]
				}
				report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: runs})
				if err != nil {
					t.Fatal(err)
				}
				if report.Qualified || report.Deterministic || report.TargetSuccess || report.DiagnosticDivergence != nil {
					t.Fatalf("report %+v", report)
				}
				data, err := canonicaljson.CanonicalJSON(report)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Contains(data, []byte(`"diagnostic_status":"unavailable"`)) {
					t.Fatalf("unavailable diagnostics omitted: %s", data)
				}
				if _, err := DecodeQualificationReport(data); err != nil {
					t.Fatal(err)
				}
				path, err := WriteQualificationReport(t.TempDir(), report)
				if err != nil {
					t.Fatal(err)
				}
				saved, err := OpenQualificationReport(path)
				if err != nil {
					t.Fatal(err)
				}
				if len(saved.Executions) != 2 || saved.Executions[boolIndex(interruptedFirst)].Diagnostics == nil {
					t.Fatalf("retained traces missing: %+v", saved)
				}
				failure, err := BuildQualificationFailure([]string{"gomad", "qualify"}, 7, 2, runs, QualificationFailure{Classification: "cancelled", Message: "interrupted", Iteration: 2})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := WriteQualificationReport(t.TempDir(), failure); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func boolIndex(value bool) int {
	if value {
		return 1
	}
	return 0
}

func TestDiagnosticsRejectCorruptedSavedReportBindings(t *testing.T) {
	for _, different := range []bool{false, true} {
		for _, corruption := range []string{"missing", "contradictory", "evidence missing", "evidence altered", "status missing", "status contradictory", "baseline digest"} {
			t.Run(fmt.Sprintf("different=%t/%s", different, corruption), func(t *testing.T) {
				reference := qualificationDiagnostic(t, 0)
				first := successfulEvidence()
				first.Diagnostics = &reference.DiagnosticEvidence
				second := first
				secondReference := reference
				if different {
					secondReference = qualificationDiagnostic(t, 1)
					second.Diagnostics = &secondReference.DiagnosticEvidence
				}
				report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{{CampaignPath: "/first", Evidence: first, Diagnostics: reference}, {CampaignPath: "/second", Evidence: second, Diagnostics: secondReference}}})
				if err != nil {
					t.Fatal(err)
				}
				switch corruption {
				case "missing":
					report.Executions[1].Diagnostics = nil
				case "contradictory":
					report.Executions[1].Diagnostics.SHA256 = record.HashBytes([]byte("contradiction"))
				case "evidence missing":
					report.Executions[1].Evidence = nil
				case "evidence altered":
					report.Executions[1].Evidence.Stdout.FullSHA256 = record.HashBytes([]byte("contradiction"))
				case "status missing":
					report.Executions[1].DiagnosticStatus = ""
				case "status contradictory":
					report.Executions[1].DiagnosticStatus = DiagnosticUnavailable
				case "baseline digest":
					changed, err := cloneEvidence(*report.Executions[0].Evidence)
					if err != nil {
						t.Fatal(err)
					}
					changed.Stdout.FullSHA256 = record.HashBytes([]byte("contradiction"))
					report.Executions[0].Evidence = &changed
					report.Executions[0].EvidenceDigest, err = evidenceDigest(changed)
					if err != nil {
						t.Fatal(err)
					}
				}
				data, err := canonicaljson.CanonicalJSON(report)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := DecodeQualificationReport(data); err == nil {
					t.Fatal("corrupt saved diagnostic binding accepted")
				}
			})
		}
	}
}

func TestDiagnosticsCompareAvailablePairsAfterInterruptedBaseline(t *testing.T) {
	reference, different := qualificationDiagnostic(t, 0), qualificationDiagnostic(t, 1)
	complete := successfulEvidence()
	complete.Environment = append(complete.Environment, record.Environment{Name: choice.DiagnosticProfileEnvironment, Value: choice.DiagnosticProfile})
	complete.Diagnostics = &reference.DiagnosticEvidence
	interrupted := complete
	interrupted.Diagnostics = nil
	interrupted.Outcome = runner.OutcomeEvidence{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout"}
	changed := complete
	changed.Diagnostics = &different.DiagnosticEvidence
	runs := []QualificationExecution{
		{CampaignPath: "/interrupted", ArtifactPath: "/failure", Evidence: interrupted},
		{CampaignPath: "/complete", Evidence: complete, Diagnostics: reference},
		{CampaignPath: "/changed", Evidence: changed, Diagnostics: different},
	}
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: runs})
	if err != nil {
		t.Fatal(err)
	}
	if report.DiagnosticDivergence == nil || report.DiagnosticDivergence.BaselineIteration != 2 || report.DiagnosticDivergence.Iteration != 3 {
		t.Fatalf("pair %+v", report.DiagnosticDivergence)
	}
	if _, err := WriteQualificationReport(t.TempDir(), report); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(different.Path, []byte("truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: runs}); err == nil {
		t.Fatal("corrupt trace after unavailable baseline accepted")
	}
}

func TestDiagnosticsUnavailableForEveryInterruptedRun(t *testing.T) {
	interrupted := successfulEvidence()
	interrupted.Environment = append(interrupted.Environment, record.Environment{Name: choice.DiagnosticProfileEnvironment, Value: choice.DiagnosticProfile})
	interrupted.Outcome = runner.OutcomeEvidence{Domain: "watchdog", Reason: "watchdog_timeout", Termination: "timeout"}
	runs := []QualificationExecution{{CampaignPath: "/first", ArtifactPath: "/first-failure", Evidence: interrupted}, {CampaignPath: "/second", ArtifactPath: "/second-failure", Evidence: interrupted}}
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: runs})
	if err != nil {
		t.Fatal(err)
	}
	if report.Qualified || !report.Deterministic || report.TargetSuccess || report.DiagnosticDivergence != nil {
		t.Fatalf("report %+v", report)
	}
	for _, run := range report.Executions {
		if run.DiagnosticStatus != DiagnosticUnavailable || run.Evidence == nil || run.Diagnostics != nil {
			t.Fatalf("execution %+v", run)
		}
	}
	path, err := WriteQualificationReport(t.TempDir(), report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := OpenQualificationReport(path); err != nil {
		t.Fatal(err)
	}
	interrupted.Outcome = successfulEvidence().Outcome
	runs[1].Evidence = interrupted
	if _, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: runs}); err == nil {
		t.Fatal("uninterrupted execution without diagnostics accepted")
	}
}

func TestDiagnosticsOffReportFieldsRemainAbsent(t *testing.T) {
	report, err := BuildQualificationReport(QualificationInput{Command: []string{"gomad", "qualify"}, Executions: []QualificationExecution{{CampaignPath: "/first", Evidence: successfulEvidence()}, {CampaignPath: "/second", Evidence: successfulEvidence()}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, run := range report.Executions {
		if run.Evidence != nil || run.DiagnosticStatus != "" || run.Diagnostics != nil {
			t.Fatalf("off execution %+v", run)
		}
	}
	data, err := canonicaljson.CanonicalJSON(report)
	if err != nil {
		t.Fatal(err)
	}
	if got := record.HashBytes(data); got != "sha256:6f8f10d12c1e605c1b0e5c66bda697ace3dccce020f0f37f4f751309cca9978e" {
		t.Fatalf("off report baseline changed: %s", got)
	}
	failure, err := BuildQualificationFailure([]string{"gomad", "qualify"}, 7, 2, []QualificationExecution{{CampaignPath: "/first", Evidence: successfulEvidence()}, {CampaignPath: "/second", Evidence: successfulEvidence()}}, QualificationFailure{Classification: "cancelled", Message: "interrupted", Iteration: 2})
	if err != nil {
		t.Fatal(err)
	}
	failureData, err := canonicaljson.CanonicalJSON(failure)
	if err != nil {
		t.Fatal(err)
	}
	if got := record.HashBytes(failureData); got != "sha256:3e8f1fda4a5755b9b0cf74e60e97e5f4ea953a712f20c6b89b1df8f362b12096" {
		t.Fatalf("off failure report baseline changed: %s", got)
	}
	if bytes.Contains(data, []byte(`"diagnostic_status"`)) || bytes.Count(data, []byte(`"evidence":`)) != 1 {
		t.Fatalf("off report fields changed: %s", data)
	}
}
