package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/internal/sweep"
)

func validArgs(root string) []string {
	return []string{"--checkout", filepath.Join(root, "temporal"), "--gomad", "/bin/gomad", "--compiler", "/go/bin/go", "--helper", "/bin/helper",
		"--runtime-root", "/runtimesource", "--config-source", "throwaway", "--packages", "./tests/gomadfunctional,./tests", "--max-tests", "10",
		"--per-test-timeout", "10m", "--prepare-timeout", "30m", "--overall-budget", "24h", "--retain-bytes", "5GiB",
		"--out", filepath.Join(root, "out"), "--work", filepath.Join(root, "work"), "--artifacts", filepath.Join(root, "artifacts")}
}

func without(args []string, name string) []string {
	var kept []string
	for index := 0; index < len(args); index++ {
		if args[index] == name {
			index++
			continue
		}
		kept = append(kept, args[index])
	}
	return kept
}

func replace(args []string, name, value string) []string {
	out := append([]string(nil), args...)
	for index := range out {
		if out[index] == name {
			out[index+1] = value
		}
	}
	return out
}

func TestRunPassesValidatedOptions(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	var got sweep.Options
	var harness string
	code := run(t.Context(), validArgs(root), []string{"HOME=" + root}, &bytes.Buffer{}, &bytes.Buffer{}, func(_ context.Context, sweeper *sweep.Sweeper) ([]string, error) {
		got, harness = sweeper.Options, sweeper.HarnessSHA256
		return nil, nil
	})
	if !strings.HasPrefix(harness, "sha256:") {
		t.Fatalf("harness identity %q", harness)
	}
	if code != 0 || got.RetainBytes != 5<<30 || got.MaxTests != 10 || len(got.Packages) != 2 || got.Packages[1] != "./tests" || got.ConfigSource != sweep.ConfigThrowaway {
		t.Fatalf("exit %d options %+v", code, got)
	}
}

func TestRunRefusals(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	never := func(context.Context, *sweep.Sweeper) ([]string, error) {
		t.Error("sweep started after a refusal")
		return nil, nil
	}
	for _, refusal := range []struct {
		name    string
		args    []string
		environ []string
		want    string
	}{
		{"gomad-environment", validArgs(root), []string{"GOMAD_WASM_INSTALLATION=/x"}, "unset GOMAD_WASM_INSTALLATION"},
		{"missing-retain-bytes", without(validArgs(root), "--retain-bytes"), nil, "--retain-bytes is required"},
		{"missing-per-test-timeout", without(validArgs(root), "--per-test-timeout"), nil, "--per-test-timeout must be a positive bound"},
		{"missing-overall-budget", without(validArgs(root), "--overall-budget"), nil, "--overall-budget must be a positive bound"},
		{"zero-overall-budget", replace(validArgs(root), "--overall-budget", "0s"), nil, "--overall-budget must be a positive bound"},
		{"bad-retain-bytes", replace(validArgs(root), "--retain-bytes", "5GB"), nil, "whole byte count"},
		{"bad-config-source", replace(validArgs(root), "--config-source", "fn220"), nil, "--config-source"},
		{"extra-argument", append(validArgs(root), "extra"), nil, "unexpected arguments"},
		{"verify-with-sweep-flags", append(validArgs(root), "--verify", "/x.json"), nil, "--verify takes no other flags"},
	} {
		t.Run(refusal.name, func(t *testing.T) {
			t.Parallel()
			var stderr bytes.Buffer
			if code := run(t.Context(), refusal.args, refusal.environ, &bytes.Buffer{}, &stderr, never); code != exitRefused || !strings.Contains(stderr.String(), refusal.want) {
				t.Fatalf("exit %d stderr %q, want refusal %q", code, stderr.String(), refusal.want)
			}
		})
	}
}

func TestRunMapsSweepErrors(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, outcome := range []struct {
		err      error
		findings []string
		want     int
	}{{nil, nil, 0}, {nil, []string{"TestA: 0 rows"}, exitFailed}, {sweep.ErrEnumerationMismatch, nil, exitFailed}, {sweep.ErrRefused, nil, exitRefused}, {errors.New("install failed"), nil, exitFailed}} {
		code := run(t.Context(), validArgs(root), nil, &bytes.Buffer{}, &bytes.Buffer{}, func(context.Context, *sweep.Sweeper) ([]string, error) {
			return outcome.findings, outcome.err
		})
		if code != outcome.want {
			t.Fatalf("%v %v: exit %d, want %d", outcome.err, outcome.findings, code, outcome.want)
		}
	}
}

func TestVerifyMode(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sweep.json")
	encoded, _ := json.Marshal(sweep.Sidecar{Schema: sweep.SidecarSchema, Statement: sweep.ObservationStatement})
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	if code := run(t.Context(), []string{"--verify", path}, nil, &stdout, &bytes.Buffer{}, nil); code != exitFailed || !strings.Contains(stdout.String(), "candidate identity missing") {
		t.Fatalf("exit %d: %s", code, stdout.String())
	}
	if code := run(t.Context(), []string{"--verify", filepath.Join(t.TempDir(), "absent.json")}, nil, &bytes.Buffer{}, &bytes.Buffer{}, nil); code != exitFailed {
		t.Fatalf("absent sidecar exit %d", code)
	}
}

func TestParseBytes(t *testing.T) {
	t.Parallel()
	for input, want := range map[string]uint64{"0": 0, "1024": 1024, "5GiB": 5 << 30, "16MiB": 16 << 20, "3KiB": 3 << 10} {
		if got, err := parseBytes(input); err != nil || got != want {
			t.Fatalf("%s: %d %v", input, got, err)
		}
	}
	for _, input := range []string{"", "-1", "1.5GiB", "5GB", "18446744073709551615GiB"} {
		if _, err := parseBytes(input); err == nil {
			t.Fatalf("%q accepted", input)
		}
	}
}
