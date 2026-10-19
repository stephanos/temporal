package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/measurement"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../../../../../../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

type failingOutput struct {
	writes int
}

func (output *failingOutput) Write(data []byte) (int, error) {
	if output.writes == 0 {
		return 0, fmt.Errorf("output unavailable")
	}
	output.writes--
	return len(data), nil
}

func TestOutputFailureIsInfrastructureFailure(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		args        []string
		stdoutFails bool
		writes      int
	}{
		{name: "help-header", args: []string{"--help"}, stdoutFails: true},
		{name: "help-list", args: []string{"--help"}, stdoutFails: true, writes: 1},
		{name: "unknown-measurement", args: []string{"unknown"}},
		{name: "flag-parser", args: []string{"runtime-cost-profile", "--unknown"}},
		{name: "invalid-repository", args: []string{"runtime-cost-profile", "--repo", "relative"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout, stderr io.Writer = io.Discard, &failingOutput{writes: test.writes}
			if test.stdoutFails {
				stdout, stderr = stderr, stdout
			}
			start := func(context.Context, invocation, io.Writer, io.Writer) int {
				t.Fatal("output failure must precede producer execution")
				return 0
			}
			if code := run(t.Context(), test.args, nil, stdout, stderr, start); code != measurement.ExitFailed {
				t.Fatalf("exit %d, want infrastructure failure", code)
			}
		})
	}
}

func TestExecutorRefusesAnotherCheckout(t *testing.T) {
	t.Parallel()
	var stderr bytes.Buffer
	if code := execute(t.Context(), invocation{name: "runtime-cost-profile", repo: t.TempDir()}, io.Discard, &stderr); code != measurement.ExitRefused || !strings.Contains(stderr.String(), "executor was not built from --repo") {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
}

func TestKeepFlagReachesProducer(t *testing.T) {
	t.Parallel()
	called := false
	start := func(_ context.Context, call invocation, _, _ io.Writer) int {
		called = true
		if !call.keep {
			t.Error("explicit keep was lost")
		}
		return 0
	}
	code := run(t.Context(), []string{"runtime-cost-profile", "--repo", repoRoot(t), "--stock-go", "/go", "--profile-root", "/evidence", "--test-helper", "/helper", "--keep"}, nil, io.Discard, io.Discard, start)
	if code != 0 || !called {
		t.Fatalf("producer called=%t exit=%d", called, code)
	}
}

func TestProducerBuildsCommandFromSelectedCheckout(t *testing.T) {
	t.Parallel()
	for _, code := range []int{0, measurement.ExitFailed, measurement.ExitRefused, 7} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			goCommand, child := filepath.Join(root, "go"), filepath.Join(root, "child")
			buildLog := filepath.Join(root, "build.log")
			environmentLog := filepath.Join(root, "environment.log")
			for path, body := range map[string]string{
				goCommand: "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > \"$BUILD_LOG\"\nprintf '%s\\n' \"$GOFLAGS\" \"$GOROOT\" \"$GOWORK\" \"$GOENV\" \"$GOTOOLCHAIN\" > \"$ENVIRONMENT_LOG\"\n/bin/cp \"$CHILD\" \"$4\"\n",
				child:     "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\"\nexit " + fmt.Sprint(code) + "\n",
			} {
				if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			call := invocation{name: "runtime-cost-profile", repo: repoRoot(t), goCommand: goCommand,
				inputs:  map[string]string{"GOMAD_WASM_PROFILE_ROOT": "/a b", "GOMAD_STOCK_GO": "/go"},
				environ: []string{"BUILD_LOG=" + buildLog, "ENVIRONMENT_LOG=" + environmentLog, "CHILD=" + child, "GOFLAGS=-overlay=/foreign/overlay.json", "GOROOT=/foreign/go", "GOWORK=/foreign/go.work", "GOENV=/foreign/env", "GOTOOLCHAIN=auto"}}
			call.keep = code == measurement.ExitFailed
			var stdout, stderr bytes.Buffer
			wantCode := code
			if wantCode == 7 {
				wantCode = measurement.ExitFailed
			}
			if got := produce(t.Context(), call, &stdout, &stderr); got != wantCode {
				t.Fatalf("exit %d, want %d: %s", got, wantCode, stderr.String())
			}
			build, err := os.ReadFile(buildLog)
			if err != nil {
				t.Fatal(err)
			}
			module := filepath.Join(call.repo, "tools", "gomad")
			if !strings.HasPrefix(string(build), module+"\nbuild\n-trimpath=false\n-o\n") || !strings.HasSuffix(string(build), "\n./internal/backends/wasm/cmd/wasmmeasure\n") {
				t.Fatalf("build: %s", build)
			}
			environment, err := os.ReadFile(environmentLog)
			if err != nil {
				t.Fatal(err)
			}
			if string(environment) != "\n\noff\noff\nlocal\n" {
				t.Fatalf("ambient build inputs redirected executor: %q", environment)
			}
			wantArgs := []string{module, "--internal-run", call.name, "--repo", call.repo, "--go", goCommand}
			if code == measurement.ExitFailed {
				wantArgs = append(wantArgs, "--keep")
			}
			want := strings.Join(append(wantArgs, "--stock-go", "/go", "--profile-root", "/a b", ""), "\n")
			if stdout.String() != want {
				t.Fatalf("child arguments: %q, want %q", stdout.String(), want)
			}
		})
	}
}

// Every subcommand refuses before building or producing anything without its
// required inputs, with an inherited GOMAD_* input, or without an explicit
// checkout that carries the producers.
func TestEverySubcommandRefusesWithoutExplicitInputs(t *testing.T) {
	repo := repoRoot(t)
	names := measurement.Names()
	if len(names) < 22 {
		t.Fatalf("measurements: %v", names)
	}
	never := func(context.Context, invocation, io.Writer, io.Writer) int {
		t.Fatal("producer started")
		return 0
	}
	for _, name := range names {
		var required int
		for _, input := range measurement.Inputs(name) {
			if input.Required {
				required++
			}
		}
		if required == 0 {
			t.Fatalf("%s has no required input", name)
		}
		for label, args := range map[string][]string{
			"no inputs":    {name, "--repo", repo},
			"no checkout":  {name},
			"relative":     {name, "--repo", "relative"},
			"not a gomad":  {name, "--repo", t.TempDir()},
			"extra":        {name, "--repo", repo, "positional"},
			"unknown flag": {name, "--repo", repo, "--not-an-input", "x"},
		} {
			var stderr bytes.Buffer
			if code := run(t.Context(), args, nil, io.Discard, &stderr, never); code != measurement.ExitRefused {
				t.Fatalf("%s %s: exit %d, %s", name, label, code, stderr.String())
			}
		}
		var stderr bytes.Buffer
		if code := run(t.Context(), []string{name}, []string{"GOMAD_WASM_PROFILE_ROOT=/tmp/x"}, io.Discard, &stderr, never); code != measurement.ExitRefused || !strings.Contains(stderr.String(), "unset GOMAD_WASM_PROFILE_ROOT") {
			t.Fatalf("%s inherited input: exit %d, %s", name, code, stderr.String())
		}
	}
	var stderr bytes.Buffer
	if code := run(t.Context(), []string{"no-such-measurement"}, nil, io.Discard, &stderr, never); code != measurement.ExitRefused {
		t.Fatalf("unknown measurement: exit %d", code)
	}
}

// Flags become the producer's inputs for the named checkout, and the
// producer's exit status is the command's.
func TestFlagsBecomeProducerInputs(t *testing.T) {
	repo := repoRoot(t)
	for _, code := range []int{0, measurement.ExitFailed, measurement.ExitRefused} {
		var seen invocation
		start := func(_ context.Context, call invocation, _, _ io.Writer) int {
			seen = call
			return code
		}
		args := []string{"runtime-cost-profile", "--repo", repo, "--profile-root", "/evidence", "--test-helper", "/helper", "--stock-go", "/go", "--go", "/build/go"}
		if got := run(t.Context(), args, []string{"PATH=/bin"}, io.Discard, io.Discard, start); got != code {
			t.Fatalf("exit %d, want %d", got, code)
		}
		want := map[string]string{"GOMAD_WASM_PROFILE_ROOT": "/evidence", "GOMAD_WASM_TEST_HELPER": "/helper", "GOMAD_STOCK_GO": "/go"}
		if seen.name != "runtime-cost-profile" || seen.repo != repo || seen.goCommand != "/build/go" || len(seen.inputs) != len(want) || len(seen.environ) != 1 {
			t.Fatalf("invocation %+v", seen)
		}
		for variable, value := range want {
			if seen.inputs[variable] != value {
				t.Fatalf("input %s = %q", variable, seen.inputs[variable])
			}
		}
	}
	var seen invocation
	start := func(_ context.Context, call invocation, _, _ io.Writer) int { seen = call; return 0 }
	if code := run(t.Context(), []string{"diagnostic-helper-build", "--repo", repo, "--diagnostic-helper-build-root", "/root", "--features", "diagnostic-profile,guest-profile"}, nil, io.Discard, io.Discard, start); code != 0 || seen.inputs["GOMAD_WASM_DIAGNOSTIC_HELPER_FEATURES"] != "diagnostic-profile,guest-profile" {
		t.Fatalf("diagnostic helper build: exit %d, %+v", code, seen)
	}
}
