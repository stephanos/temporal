package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"go.temporal.io/server/tools/gomad3/qualification"
	qualificationworkload "go.temporal.io/server/tools/gomad3/qualification/workload"
	"go.temporal.io/server/tools/gomad3/runner"
)

type terminalDiagnosticWriter struct {
	output   bytes.Buffer
	file     *os.File
	failures map[int]bool
	attempts []string
	errors   []error
}

func (writer *terminalDiagnosticWriter) Write(contents []byte) (int, error) {
	writer.attempts = append(writer.attempts, string(contents))
	if writer.failures[len(writer.attempts)] {
		written, err := writer.file.Write(contents)
		writer.errors = append(writer.errors, err)
		return written, err
	}
	writer.errors = append(writer.errors, nil)
	return writer.output.Write(contents)
}

func terminalDiagnostics(t *testing.T, failures map[int]bool) *terminalDiagnosticWriter {
	t.Helper()
	path := filepath.Join(t.TempDir(), "read-only")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := file.Close(); err != nil {
			t.Error(err)
		}
	})
	return &terminalDiagnosticWriter{file: file, failures: failures}
}

func checkTerminalDiagnostics(t *testing.T, writer *terminalDiagnosticWriter, want []string) {
	t.Helper()
	if !reflect.DeepEqual(writer.attempts, want) {
		t.Fatalf("attempts = %q, want %q", writer.attempts, want)
	}
	var delivered strings.Builder
	for index, contents := range want {
		if writer.failures[index+1] {
			if !errors.Is(writer.errors[index], syscall.EBADF) {
				t.Fatalf("attempt %d error = %v, want actual EBADF", index+1, writer.errors[index])
			}
		} else {
			if writer.errors[index] != nil {
				t.Fatal(writer.errors[index])
			}
			delivered.WriteString(contents)
		}
	}
	if writer.output.String() != delivered.String() {
		t.Fatalf("delivered = %q, want %q", writer.output.String(), delivered.String())
	}
	contents, err := os.ReadFile(writer.file.Name())
	if err != nil {
		t.Fatal(err)
	}
	if len(contents) != 0 {
		t.Fatalf("read-only file received %q", contents)
	}
}

func TestTerminalDiagnosticsPublicFailures(t *testing.T) {
	t.Setenv("GOMAD3_TOOLCHAIN_DIR", "relative")
	root := t.TempDir()
	plan := filepath.Join(root, "plan.json")
	if err := os.WriteFile(plan, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name       string
		arguments  []string
		status     int
		diagnostic string
		reporter   bool
	}{
		{"usage", nil, 2, usage, false},
		{"unknown", []string{"unknown"}, 2, "unknown gomad command \"unknown\"\n" + usage, false},
		{"doctor usage", []string{"doctor", "extra"}, 2, usage, false},
		{"replay usage", []string{"replay"}, 2, usage, false},
		{"shard usage", []string{"execute-shard"}, 2, usage, false},
		{"shard syntax", []string{"execute-shard", "--shard=1", plan}, 2, "invalid shard \"1\": want zero-based INDEX/COUNT\n", false},
		{"merge usage", []string{"merge"}, 2, usage, false},
		{"merge invalid plan", []string{"merge", "--output=" + filepath.Join(root, "merged"), plan, filepath.Join(root, "shard")}, 2, "gomad: invalid_input: merge campaign shards: decode campaign plan: JSON is not canonical\n", false},
		{"explore output", []string{"explore", "--output=plan", "go-run", "./pkg"}, 2, "gomad: invalid_input: --output is only valid with gomad plan\n", true},
		{"explore capability", []string{"explore", "--capability-mode=open"}, 2, "gomad: invalid_input: unknown capability mode \"open\"\n", true},
		{"explore strategy", []string{"explore", "--strategy=random"}, 2, "gomad: invalid_input: unknown exploration strategy \"random\"\n", true},
		{"explore seeds", []string{"explore", "--count=0"}, 2, "gomad: invalid_input: --count must be greater than zero\n", true},
		{"explore regression", []string{"explore", "--guide-regression"}, 2, "gomad: invalid_input: --guide-regression requires --guide\n", true},
		{"explore guidance", []string{"explore", "--guide"}, 2, "gomad: invalid_input: --guide requires --corpus DIR\n", true},
		{"explore coverage", []string{"explore", "--require-probe=stdlib.os.openfile"}, 2, "gomad: invalid_input: --require-probe requires --coverage=semantic\n", true},
		{"explore choices", []string{"explore", "--choice-bytes=1MiB"}, 2, "gomad: invalid_input: --choice-bytes requires --choices\n", true},
		{"explore target", []string{"explore"}, 2, "gomad: invalid_input: target kind is required\n", true},
		{"explore working directory", []string{"explore", "--working-dir=module", "go-run", "./pkg"}, 2, "gomad: invalid_input: working directory \"module\" must be an absolute, clean path\n", true},
		{"qualify input", []string{"qualify", "--repeat=0"}, 2, "gomad: invalid_input: --repeat must be between 2 and 32\n", true},
		{"resume input", []string{"resume"}, 2, "gomad: invalid_input: resume requires one interrupted campaign path\n", true},
		{"replay installation", []string{"replay", plan}, 3, "resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", false},
		{"shard installation", []string{"execute-shard", "--shard=0/1", plan}, 3, "resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", false},
		{"explore installation", []string{"explore", "go-run", "./pkg"}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
		{"qualify installation", []string{"qualify", "go-run", "./pkg"}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
		{"resume installation", []string{"resume", plan}, 3, "gomad: runner_failure: resolve Gomad installation: GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: \"relative\"\n", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failures := range []map[int]bool{nil, {1: true}, {1: true, 2: true}} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, failures)
				status := Run(test.arguments, stdout, stderr)
				want, wantStatus := []string{test.diagnostic}, test.status
				if failures[1] && test.reporter {
					want = append(want, "write "+stderr.file.Name()+": bad file descriptor\n")
					wantStatus = 3
				}
				if status != wantStatus || len(stdout.attempts) != 0 {
					t.Fatalf("status = %d, want %d; stdout attempts = %q", status, wantStatus, stdout.attempts)
				}
				checkTerminalDiagnostics(t, stderr, want)
				contents, err := os.ReadFile(plan)
				if err != nil || string(contents) != "{}\n" {
					t.Fatalf("plan = %q, %v", contents, err)
				}
				entries, err := os.ReadDir(root)
				if err != nil || len(entries) != 1 || entries[0].Name() != "plan.json" {
					t.Fatalf("publication entries = %v, %v", entries, err)
				}
			}
		})
	}
}

func TestTerminalDiagnosticsPrivateDispatch(t *testing.T) {
	for _, mode := range []string{coordinatorMode, supervisorMode, targetBootstrapMode} {
		t.Run(mode, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				input, output := strings.NewReader("request"), new(bytes.Buffer)
				calls := 0
				app := application{privateInput: input, privateOutput: output, dispatch: func(got string, in io.Reader, out io.Writer) error {
					if got != mode || in != input || out != output || len(stderr.attempts) != 0 {
						t.Fatal("private dispatch streams or ordering changed")
					}
					calls++
					return errors.New("decode private request: EOF")
				}}
				if status := app.run([]string{mode, "ignored"}, stdout, stderr); status != 3 || calls != 1 || len(stdout.attempts) != 0 || output.Len() != 0 {
					t.Fatalf("status = %d, calls = %d, stdout = %q, private output = %q", status, calls, stdout.attempts, output.String())
				}
				checkTerminalDiagnostics(t, stderr, []string{"decode private request: EOF\n"})
			}
		})
	}
}

func TestTerminalDiagnosticsReplayClassifications(t *testing.T) {
	for _, test := range []struct {
		name   string
		err    error
		status int
		text   string
	}{
		{"preflight", &runner.ReplayPreflightError{Err: errors.New("invalid retained artifact")}, 2, "incompatible replay artifact: invalid retained artifact\n"},
		{"host", &runner.HostError{Reason: "replay", Err: errors.New("child failed")}, 3, "gomad3 Runner/host failure: replay: child failed\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				stdout := terminalDiagnostics(t, nil)
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				calls := 0
				dependencies := newFakeInstallation().replayDependencies(func(_ context.Context, spec runner.ReplaySpec) (runner.ReplayResult, error) {
					if spec.ArtifactPath != "/artifact" || len(stderr.attempts) != 0 {
						t.Fatal("replay operation or ordering changed")
					}
					calls++
					return runner.ReplayResult{}, test.err
				})
				if status := runReplayWith([]string{"/artifact"}, stdout, stderr, dependencies); status != test.status || calls != 1 || len(stdout.attempts) != 0 {
					t.Fatalf("status = %d, calls = %d, stdout = %q", status, calls, stdout.attempts)
				}
				checkTerminalDiagnostics(t, stderr, []string{test.text})
			}
		})
	}
}

func TestTerminalDiagnosticsCompletedOperations(t *testing.T) {
	for _, command := range []string{"explore", "execute-shard", "resume", "replay", "qualify"} {
		t.Run(command, func(t *testing.T) {
			for _, failed := range []bool{false, true} {
				root := t.TempDir()
				published := filepath.Join(root, "completed")
				stdout := terminalDiagnostics(t, map[int]bool{1: true})
				stderr := terminalDiagnostics(t, map[int]bool{1: failed})
				completed := 0
				finish := func() {
					if len(stdout.attempts) != 0 || len(stderr.attempts) != 0 {
						t.Fatal("reporting preceded the operation")
					}
					if err := os.WriteFile(published, []byte("committed\n"), 0o600); err != nil {
						t.Fatal(err)
					}
					completed++
				}
				result := runner.CampaignResult{CampaignPath: published, Attempted: 1, Succeeded: 1}
				fake := newFakeInstallation()
				var status int
				var wantOutput string
				switch command {
				case "explore":
					dependencies := fake.exploreDependencies(func(context.Context, runner.CampaignSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					}, nil)
					status = runExploreWith([]string{"go-run", "./pkg"}, stdout, stderr, dependencies)
				case "execute-shard":
					dependencies := fake.campaignShardDependencies(func(context.Context, runner.CampaignShardSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					})
					status = runCampaignShardWith([]string{"--shard=0/1", "/plan"}, stdout, stderr, dependencies)
				case "resume":
					dependencies := fake.resumeDependencies(func(context.Context, runner.ResumeSpec) (runner.CampaignResult, error) {
						finish()
						return result, nil
					})
					status = runResumeWith([]string{"/campaign"}, stdout, stderr, dependencies)
				case "replay":
					dependencies := fake.replayDependencies(func(context.Context, runner.ReplaySpec) (runner.ReplayResult, error) {
						finish()
						return runner.ReplayResult{Match: true}, nil
					})
					status = runReplayWith([]string{"/artifact"}, stdout, stderr, dependencies)
					wantOutput = "gomad: reproduced=true diagnostic=false result=target_failure choice-replay=\n"
				case "qualify":
					dependencies := qualifyDependencies{
						install: fake.install, workingDirectory: func() (string, error) { return "/workspace", nil },
						workload: func(context.Context, qualificationworkload.Spec) (qualificationworkload.Result, error) {
							finish()
							return qualificationworkload.Result{ReportPath: published, Report: qualification.QualificationReport{Qualified: true, Deterministic: true, TargetSuccess: true}}, nil
						},
					}
					status = runQualifyWith([]string{"go-run", "./pkg"}, stdout, stderr, dependencies)
					wantOutput = "gomad: qualification qualified=true deterministic=true target-success=true seed=0 repeat=0 report=" + published + "\n"
				}
				if wantOutput == "" {
					wantOutput = "gomad: classification=success attempted=1 succeeded=1 failures=0 watchdogs=0 replay-divergences=0 distinct=0 retained-successes=0 retained-success-bytes=0 stop= artifact=" + published + "\n"
				}
				if status != 3 || completed != 1 {
					t.Fatalf("status = %d, completed = %d", status, completed)
				}
				checkTerminalDiagnostics(t, stdout, []string{wantOutput})
				checkTerminalDiagnostics(t, stderr, []string{"write " + stdout.file.Name() + ": bad file descriptor\n"})
				contents, err := os.ReadFile(published)
				if err != nil || string(contents) != "committed\n" {
					t.Fatalf("completed operation file = %q, %v", contents, err)
				}
			}
		})
	}
}
