package conformance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
)

func TestRunUpstreamExecutesTypedGatesInOrder(t *testing.T) {
	root := t.TempDir()
	goRoot := filepath.Join(root, "go")
	if err := os.MkdirAll(filepath.Join(goRoot, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(goRoot, "bin", "go")
	if err := os.MkdirAll(filepath.Dir(goCommand), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goCommand, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	var requests []hostexec.Request
	report, err := runWith(context.Background(), Config{Root: root, Mode: "test-upstream", Go: goCommand}, func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
		requests = append(requests, request)
		if len(request.Command) == 3 && request.Command[1] == "env" {
			result := successfulCommand()
			result.Stdout = hostexec.Output{Bytes: []byte(goRoot + "\n"), RawBytes: []byte(goRoot + "\n")}
			return result, nil
		}
		return successfulCommand(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || report.Mode != "test-upstream" || len(report.Cases) != 2 {
		t.Fatalf("Run() report = %+v", report)
	}
	if got := []string{requests[1].Command[1], requests[2].Command[1]}; !slices.Equal(got, []string{"test", "tool"}) {
		t.Fatalf("upstream commands = %v", got)
	}
	if filepath.Base(requests[1].Dir) != "src" || requests[1].Dir != requests[2].Dir || !strings.Contains(requests[1].Dir, "gomadv3-upstream-") {
		t.Fatalf("upstream working directories = %q, %q", requests[1].Dir, requests[2].Dir)
	}
	if slices.ContainsFunc(requests[1].Env, func(value string) bool { return strings.HasPrefix(value, "GOMADSEED=") }) {
		t.Fatal("upstream gate inherited GOMADSEED")
	}
}

func TestRunBuilderExecutesTypedPackageGate(t *testing.T) {
	root := writeGoFixture(t)
	goCommand := filepath.Join(root, "go", "bin", "go")
	var requests []hostexec.Request
	report, err := runWith(context.Background(), Config{Root: root, Mode: "test-builder", Go: goCommand}, func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
		requests = append(requests, request)
		if len(request.Command) == 3 && request.Command[1] == "env" {
			result := successfulCommand()
			goRoot := filepath.Join(root, "go")
			result.Stdout = hostexec.Output{Bytes: []byte(goRoot + "\n"), RawBytes: []byte(goRoot + "\n")}
			return result, nil
		}
		return successfulCommand(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || len(report.Cases) != 1 || len(requests) != 2 {
		t.Fatalf("Run() report = %+v, requests = %d", report, len(requests))
	}
	if got := requests[1].Command; !slices.Equal(got, []string{goCommand, "test", "-count=1", "-tags=test_dep", "./toolchain"}) {
		t.Fatalf("builder command = %v", got)
	}
}

func TestRunLiveCapabilityExecutesPinnedSemanticFixtures(t *testing.T) {
	root := writeGoFixture(t)
	goCommand := filepath.Join(root, "go", "bin", "go")
	var requests []hostexec.Request
	report, err := runWith(context.Background(), Config{Root: root, Mode: "test-live-capability", Go: goCommand}, func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
		requests = append(requests, request)
		if len(request.Command) == 3 && request.Command[1] == "env" {
			result := successfulCommand()
			result.Stdout = hostexec.Output{Bytes: []byte(filepath.Join(root, "go") + "\n"), RawBytes: []byte(filepath.Join(root, "go") + "\n")}
			return result, nil
		}
		return successfulCommand(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || len(report.Cases) != 1 || len(requests) != 2 {
		t.Fatalf("Run() report = %+v, requests = %d", report, len(requests))
	}
	if got := requests[1].Command; !slices.Equal(got, []string{goCommand, "test", "-count=1", "-tags=test_dep", "./target/internal/livecap"}) {
		t.Fatalf("live-capability command = %v", got)
	}
}

func TestRunAcceptsExecutableSymlink(t *testing.T) {
	root := writeGoFixture(t)
	target := filepath.Join(root, "go", "bin", "go")
	link := filepath.Join(root, "go-link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	_, err := runWith(context.Background(), Config{Root: root, Mode: "test-builder", Go: link}, func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
		result := successfulCommand()
		if len(request.Command) == 3 && request.Command[1] == "env" {
			goRoot := filepath.Join(root, "go")
			result.Stdout = hostexec.Output{Bytes: []byte(goRoot + "\n"), RawBytes: []byte(goRoot + "\n")}
		}
		return result, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunUpstreamStopsAtFailedGateWithBoundedEvidence(t *testing.T) {
	root := writeGoFixture(t)
	calls := 0
	report, err := runWith(context.Background(), Config{Root: root, Mode: "test-upstream", Go: filepath.Join(root, "go", "bin", "go")}, func(context.Context, hostexec.Request) (hostexec.Result, error) {
		calls++
		if calls == 1 {
			result := successfulCommand()
			goRoot := filepath.Join(root, "go")
			result.Stdout = hostexec.Output{Bytes: []byte(goRoot + "\n"), RawBytes: []byte(goRoot + "\n")}
			return result, nil
		}
		return hostexec.Result{
			Termination: hostexec.TerminationExit, ExitCode: 2,
			Stderr: hostexec.Output{Bytes: []byte("failed"), RawBytes: []byte("failed")},
		}, nil
	})
	if err == nil || !strings.Contains(err.Error(), "upstream-clock") {
		t.Fatalf("Run() error = %v", err)
	}
	if calls != 2 || report.Passed || len(report.Cases) != 1 || string(report.Cases[0].Stderr) != "failed" {
		t.Fatalf("calls = %d, report = %+v", calls, report)
	}
}

func TestRunRejectsUnknownModeAndMissingToolchain(t *testing.T) {
	for _, test := range []struct {
		name   string
		config Config
		want   string
	}{
		{name: "mode", config: Config{Root: t.TempDir(), Mode: "unknown", Go: "/go"}, want: "unknown gomadv3 test mode"},
		{name: "toolchain", config: Config{Root: t.TempDir(), Mode: "test-upstream", Go: "/missing/go"}, want: "toolchain Go is unavailable"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := Run(context.Background(), test.config)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Run() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestRunReportsInfrastructureError(t *testing.T) {
	root := writeGoFixture(t)
	want := errors.New("runner unavailable")
	_, err := runWith(context.Background(), Config{Root: root, Mode: "test-upstream", Go: filepath.Join(root, "go", "bin", "go")}, func(context.Context, hostexec.Request) (hostexec.Result, error) {
		return hostexec.Result{}, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestRunInterceptionExecutesManifestDrivenCompilerCases(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goRoot := t.TempDir()
	if err := os.MkdirAll(filepath.Join(goRoot, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	goCommand := writeExecutable(t, filepath.Join(t.TempDir(), "go"))
	compiler := writeExecutable(t, filepath.Join(t.TempDir(), "compile"))
	expected, err := os.ReadFile(filepath.Join(root, "expected-intercepts-go1.26.4.txt"))
	if err != nil {
		t.Fatal(err)
	}
	diagnostics := map[string]string{
		"missing_target":   "gomad interception target is missing: Target",
		"missing_hook":     "gomad interception hook is missing: Hook",
		"bad_parameter":    "gomad interception signature mismatch for Target: hook parameter 1",
		"bad_result":       "gomad interception signature mismatch for Target: hook result 1",
		"bad_handled":      "gomad interception signature mismatch for Target: hook final result must be bool",
		"duplicate_target": "gomad interception target is duplicated in manifest: Target and Target",
		"bodyless_target":  "missing function body",
		"variadic":         "gomad interception signature mismatch for Target: hook variadic form does not match target",
		"body_mismatch":    "gomad interception declaration fingerprint mismatch for Target",
	}
	var requests []hostexec.Request
	report, err := runWith(context.Background(), Config{Root: root, Mode: "test-interception", Go: goCommand, Compiler: compiler}, func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
		requests = append(requests, request)
		if len(request.Command) == 3 && request.Command[1] == "env" && request.Command[2] == "GOROOT" {
			result := successfulCommand()
			result.Stdout = hostexec.Output{Bytes: []byte(goRoot + "\n"), RawBytes: []byte(goRoot + "\n")}
			return result, nil
		}
		joined := strings.Join(request.Command, " ")
		if strings.Contains(joined, "install -a") && strings.HasSuffix(joined, " os") && !slices.Contains(request.Env, "GOOS=linux") {
			result := successfulCommand()
			var applied strings.Builder
			for _, line := range strings.Split(strings.TrimSpace(string(expected)), "\n") {
				applied.WriteString("gomad intercept applied: " + line + "\n")
			}
			result.Stderr = hostexec.Output{Bytes: []byte(applied.String()), RawBytes: []byte(applied.String())}
			return result, nil
		}
		for name, diagnostic := range diagnostics {
			if strings.Contains(joined, "./interceptfail/"+name) && strings.Contains(joined, "-toolexec=") {
				return hostexec.Result{
					Termination: hostexec.TerminationExit, ExitCode: 1,
					Stderr: hostexec.Output{Bytes: []byte(diagnostic), RawBytes: []byte(diagnostic)},
				}, nil
			}
		}
		return successfulCommand(), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Passed || len(report.Cases) != 15 {
		t.Fatalf("Run() report = %+v", report)
	}
	if len(requests) != 16 {
		t.Fatalf("compiler requests = %d, want 16", len(requests))
	}
	for _, request := range requests {
		if strings.Contains(strings.Join(request.Command, " "), "./interceptfail/") && strings.Contains(strings.Join(request.Command, " "), "-toolexec=") && !slices.Contains(request.Env, "GOMADV3_TEST_COMPILE="+compiler) {
			t.Fatalf("negative compiler fixture omitted compiler identity: %v", request.Env)
		}
	}
}

func TestRunInterceptionRejectsMissingCompiler(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goCommand := writeExecutable(t, filepath.Join(t.TempDir(), "go"))
	_, err = Run(context.Background(), Config{Root: root, Mode: "test-interception", Go: goCommand})
	if err == nil || !strings.Contains(err.Error(), "test compiler is unavailable") {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestExecWrapperIsStrictPOSIX(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(root, "exec.sh"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(contents)
	if !strings.HasPrefix(source, "#!/bin/sh\n") || strings.Contains(source, "BASH_") || strings.Contains(source, "[[") || strings.Contains(source, "((") {
		t.Fatalf("exec wrapper is not strict POSIX shell:\n%s", source)
	}
}

func writeGoFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	goCommand := filepath.Join(root, "go", "bin", "go")
	if err := os.MkdirAll(filepath.Join(root, "go", "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(goCommand), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goCommand, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeExecutable(t *testing.T, path string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte("fixture"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func successfulCommand() hostexec.Result {
	return hostexec.Result{Termination: hostexec.TerminationExit, ExitCode: 0, GroupGone: true}
}
