package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/qualification/soak"
)

func TestSoakRejectsInvalidInvocationsAsInvalidInput(t *testing.T) {
	directory := t.TempDir()
	for name, arguments := range map[string][]string{
		"missing directories": {"--manifest=" + filepath.Join(directory, "soak.json")},
		"missing manifest": {
			"--manifest=" + filepath.Join(directory, "absent.json"), "--gomad=gomad",
			"--work=" + filepath.Join(directory, "work"), "--ledger=" + filepath.Join(directory, "ledger"), "--output=" + filepath.Join(directory, "output"),
		},
		"positional argument": {"extra"},
		"invalid seed":        {"--seed=eleven"},
	} {
		t.Run(name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if status := run(append([]string{"soak"}, arguments...), &stdout, &stderr); status != 2 || stdout.Len() != 0 || stderr.Len() == 0 {
				t.Fatalf("status %d, stdout %q, stderr %q", status, stdout.String(), stderr.String())
			}
		})
	}
}

func TestSoakDiagnosticWritesPreserveStatus(t *testing.T) {
	for _, test := range []struct {
		name, diagnostic string
		arguments        []string
		status           int
		publish          bool
		publicationError bool
		stdoutError      bool
	}{
		{
			name: "positional argument", arguments: []string{"extra"}, status: 2,
			diagnostic: "usage: gomadtool soak --manifest=FILE --gomad=FILE --work=DIR --ledger=DIR --output=DIR [flags]\n",
		},
		{name: "invalid seed", arguments: []string{"--seed=eleven"}, status: 2, diagnostic: "invalid --seed \"eleven\"\n"},
		{
			name: "invalid input", status: 2,
			diagnostic: "soak needs a manifest, a gomad executable, and work, ledger, and output directories\n",
		},
		{name: "report publication", publish: true, publicationError: true, status: 3},
		{name: "summary output", publish: true, stdoutError: true, status: 3},
		{name: "healthy summary", publish: true, status: 3},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				name := "healthy stderr"
				if failed {
					name = "EBADF stderr"
				}
				t.Run(name, func(t *testing.T) {
					root := t.TempDir()
					arguments, diagnostic := append([]string{"soak"}, test.arguments...), test.diagnostic
					var stdout, stderr bytes.Buffer
					output := &generatorDiagnosticOutput{writer: &stdout}
					diagnostics := &generatorDiagnosticOutput{writer: &stderr}
					var failedOutput, failedDiagnostics *refreshOutput
					if failed {
						failedDiagnostics = newRefreshOutput(t, 0)
						diagnostics.writer = failedDiagnostics
					}
					if test.publish {
						arguments = soakDiagnosticArguments(t, root)
					}
					if test.publicationError {
						path := filepath.Join(root, "output", "soak-report.json")
						if err := os.MkdirAll(path, 0o700); err != nil {
							t.Fatal(err)
						}
						diagnostic = fmt.Sprintf("rename %s %s: %s\n", path+".tmp", path, syscall.EEXIST)
					}
					if test.stdoutError {
						failedOutput = newRefreshOutput(t, 0)
						output.writer = failedOutput
						diagnostic = fmt.Sprintf("write %s: %s\n", failedOutput.file.Name(), syscall.EBADF)
					}
					status := run(arguments, output, diagnostics)
					wantAttempts := 1
					if diagnostic == "" {
						wantAttempts = 0
					}
					if status != test.status || diagnostics.attempts != wantAttempts || string(diagnostics.contents) != diagnostic {
						t.Fatalf("status=%d attempts=%d diagnostic=%q, want %d, %d, %q", status, diagnostics.attempts, diagnostics.contents, test.status, wantAttempts, diagnostic)
					}
					if failed && wantAttempts != 0 {
						if !errors.Is(failedDiagnostics.err, syscall.EBADF) || failedDiagnostics.Len() != 0 || stderr.Len() != 0 {
							t.Fatalf("stderr error=%v written=%q healthy=%q", failedDiagnostics.err, failedDiagnostics.String(), stderr.String())
						}
						t.Logf("stderr Write returned actual EBADF; attempted=%q; status=%d", diagnostics.contents, status)
					} else if stderr.String() != diagnostic && !failed {
						t.Fatalf("stderr=%q, want %q", stderr.String(), diagnostic)
					}
					if test.publish {
						path := filepath.Join(root, "output", "soak-report.json")
						if test.publicationError {
							info, err := os.Stat(path)
							if err != nil || !info.IsDir() {
								t.Fatalf("publication obstruction changed: %v", err)
							}
							if _, err := os.Stat(filepath.Join(root, "output", "soak-summary.md")); !errors.Is(err, os.ErrNotExist) {
								t.Fatalf("summary unexpectedly published: %v", err)
							}
							path += ".tmp"
						}
						contents, err := os.ReadFile(path)
						if err != nil {
							t.Fatal(err)
						}
						var report soak.Report
						if err := json.Unmarshal(contents, &report); err != nil {
							t.Fatal(err)
						}
						if report.Schema != soak.ReportSchema || report.Passed || report.Verdict != "fail" || len(report.Toolchains) != 0 || len(report.Cohorts) != 0 || report.Totals != (soak.Counts{Batches: 1, InfrastructureFailures: 1}) || len(report.BatchReports) != 1 || report.BatchReports[0].Message != "soak budget exhausted before the batch could start" {
							t.Fatalf("unexpected budget-exhausted report: %+v", report)
						}
						ledger, err := soak.LoadLedger(filepath.Join(root, "ledger"))
						if err != nil {
							t.Fatal(err)
						}
						if ledger.Schema != soak.LedgerSchema || len(ledger.Cohorts) != 0 || len(ledger.Runs) != 1 || ledger.Runs[0].ID != "diagnostic-control" || ledger.Runs[0].ManifestSHA256 != report.ManifestSHA256 || ledger.Runs[0].Platform != report.Platform || ledger.Runs[0].Started != report.Started {
							t.Fatalf("published ledger differs from initialized report: %+v", ledger)
						}
					}
					if !test.publish || test.publicationError {
						if output.attempts != 0 || len(output.contents) != 0 || stdout.Len() != 0 {
							t.Fatalf("stdout attempts=%d attempted=%q written=%q", output.attempts, output.contents, stdout.String())
						}
						return
					}
					wantSummary := fmt.Sprintf("## Determinism soak diagnostic-control on %s/%s: fail\n\n"+
						"- N = 2 fresh repetitions per workload and seed (1 batches of 2; minimum 1, maximum 1, stopped at maximum_batches), seeds 11, choice tracing and diagnostics on\n"+
						"- Load: 0 busy host threads on %d CPUs\n"+
						"- This run: 1 batches, 0 repetitions, 0 divergences, 0 overflows, 0 target failures, 1 infrastructure failures\n"+
						"- Ledger: 1 retained runs\n\n"+
						"| Cohort | Workload | Seed | Run repetitions | Cumulative repetitions | Cumulative divergences | Runs | Bound |\n"+
						"| --- | --- | --- | --- | --- | --- | --- | --- |\n"+
						"\n- control seed 11 batch 1: infrastructure. soak budget exhausted before the batch could start.\n", runtime.GOOS, runtime.GOARCH, runtime.NumCPU())
					if output.attempts != 1 || string(output.contents) != wantSummary {
						t.Fatalf("stdout attempts=%d attempted=%q, want one write of %q", output.attempts, output.contents, wantSummary)
					}
					if test.stdoutError {
						if !errors.Is(failedOutput.err, syscall.EBADF) || failedOutput.Len() != 0 || stdout.Len() != 0 {
							t.Fatalf("stdout error=%v written=%q healthy=%q", failedOutput.err, failedOutput.String(), stdout.String())
						}
					} else if stdout.String() != wantSummary {
						t.Fatalf("stdout=%q, want %q", stdout.String(), wantSummary)
					}
					published, err := os.ReadFile(filepath.Join(root, "output", "soak-summary.md"))
					if err != nil || string(published) != wantSummary {
						t.Fatalf("published summary=%q error=%v", published, err)
					}
				})
			}
		})
	}
}

func soakDiagnosticArguments(t *testing.T, root string) []string {
	t.Helper()
	maintainerWrite(t, root, "set.json", `{"schema":"gomad3.qualification-set/v3","name":"diagnostic-control","description":"adapter control","module":"example.test","seeds":[11],"repeat":2,"run_timeout":"1m","overall_timeout":"5m","terminate_grace":"2s","output_bytes":1048576,"world_transition_bytes":1048576,"suites":[{"id":"control","name":"Control","tier":1,"capability_mode":"closure","invariant":"adapter control","package":"./pkg","test":"TestControl","choice_bytes":8388608,"replay_successes":true,"success_artifact_limit":1,"success_bytes_limit":134217728,"expectation":{"classification":"qualified"}}]}`)
	maintainerWrite(t, root, "soak.json", `{"schema":"gomad3.determinism-soak/v1","name":"diagnostic-control","description":"adapter control","seeds":[11],"repeat":2,"minimum_batches":1,"batches":1,"qualify_timeout":"1m","budget":"2m","load_workers":0,"sizing":"adapter control","selections":[{"manifest":"set.json","working_dir":".","suites":["control"]}]}`)
	return []string{"soak", "--manifest=" + filepath.Join(root, "soak.json"), "--gomad=" + filepath.Join(root, "absent-gomad"), "--work=" + filepath.Join(root, "work"), "--ledger=" + filepath.Join(root, "ledger"), "--output=" + filepath.Join(root, "output"), "--run-id=diagnostic-control", "--budget=1ns", "--batches=1"}
}
