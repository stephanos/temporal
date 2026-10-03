package target

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

func TestPrepareWithGoCommandAdapterPreservesFreshAndCachedTarget(t *testing.T) {
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/task9\n\ngo 1.26.4\n",
		"main.go": "package main\nfunc main() {}\n",
	})
	spec := Spec{Kind: KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: toolchainRoot(t)}
	actual, err := Prepare(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	isolatePreparedTargetCache(t)
	builds := 0
	runner := gocommand.New(func(ctx context.Context, request hostexec.Request) (hostexec.Result, error) {
		if len(request.Command) > 2 && request.Command[1] == "build" {
			builds++
			for i, arg := range request.Command {
				if arg == "-o" && i+1 < len(request.Command) {
					data, err := os.ReadFile(actual.Path)
					if err != nil {
						return hostexec.Result{}, err
					}
					if err := os.WriteFile(request.Command[i+1], data, 0o700); err != nil {
						return hostexec.Result{}, err
					}
					return hostexec.Result{Termination: hostexec.TerminationExit}, nil
				}
			}
			t.Fatal("build command has no output path")
		}
		return hostexec.Run(ctx, request)
	})
	fresh, err := prepareWith(context.Background(), spec, runner)
	if err != nil {
		t.Fatal(err)
	}
	cached, err := prepareWith(context.Background(), spec, runner)
	if err != nil {
		t.Fatal(err)
	}
	if builds != 1 {
		t.Fatalf("builds = %d, want one fresh build and one cache restore", builds)
	}
	actual.Path, fresh.Path, cached.Path = "", "", ""
	if !reflect.DeepEqual(fresh, actual) || !reflect.DeepEqual(cached, actual) {
		t.Fatalf("prepared targets differ: actual=%#v fresh=%#v cached=%#v", actual, fresh, cached)
	}
	if !reflect.DeepEqual(fresh.RecordTarget(), actual.RecordTarget()) || !reflect.DeepEqual(cached.RecordTarget(), actual.RecordTarget()) {
		t.Fatal("recorded target provenance differs")
	}
}

func TestBuildGoTargetReleasesCacheAfterCommandFailure(t *testing.T) {
	identity, err := readPinnedToolchainWith(context.Background(), toolchainRoot(t), gocommand.Default())
	if err != nil {
		t.Fatal(err)
	}
	cache, err := targetbuild.PrepareCache(identity.installation.PinnedBuild().TargetCache())
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		result hostexec.Result
		want   error
	}{
		{name: "compiler exit", result: hostexec.Result{Termination: hostexec.TerminationExit, ExitCode: 1, Stderr: hostexec.Output{Bytes: []byte("compiler failed")}}},
		{name: "cancellation", result: hostexec.Result{Cancelled: true}, want: context.Canceled},
		{name: "watchdog", result: hostexec.Result{WatchdogTimeout: true}, want: context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := gocommand.New(func(context.Context, hostexec.Request) (hostexec.Result, error) { return test.result, nil })
			_, err := buildGoTargetWith(context.Background(), Spec{Kind: KindGoRun, ToolchainRoot: toolchainRoot(t)}, nil, identity,
				filepath.Join(t.TempDir(), "target"), filepath.Join(toolchainRoot(t), "bin", "go"), t.TempDir(), ".",
				CapabilityReview{}, rejectUnsupported, nil, runner)
			if err == nil || !strings.Contains(err.Error(), "prepare go-run target") || (test.want != nil && !errors.Is(err, test.want)) {
				t.Fatalf("buildGoTargetWith() error = %v, want wrapped %v", err, test.want)
			}
			lock, err := hostfs.Try(filepath.Join(cache, "gomad-cache.lock"))
			if err != nil {
				t.Fatalf("build cache lock remained held: %v", err)
			}
			if err := lock.Release(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestReadToolchainIdentityRejectsMalformedAndOverflowedEnvironment(t *testing.T) {
	root := t.TempDir()
	key := strings.Repeat("0", 64)
	for _, path := range []string{filepath.Join(root, "bin", "go"), filepath.Join(root, "builds", key, "bin", "go")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "build-key"), []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name         string
		output       hostexec.Output
		wantOverflow bool
	}{
		{name: "malformed", output: hostexec.Output{RawBytes: []byte("invalid\n")}},
		{name: "overflow", output: hostexec.Output{RawBytes: []byte("go1.27.1\n" + runtime.GOOS + "\n" + runtime.GOARCH + "\n0\n"), Truncated: true}, wantOverflow: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := gocommand.New(func(context.Context, hostexec.Request) (hostexec.Result, error) {
				return hostexec.Result{Termination: hostexec.TerminationExit, Stdout: test.output}, nil
			})
			_, err := readPinnedToolchainWith(context.Background(), root, runner)
			var overflow *gocommand.OverflowError
			if err == nil || !strings.Contains(err.Error(), map[bool]string{true: "query pinned Go command", false: "invalid identity"}[test.wantOverflow]) || errors.As(err, &overflow) != test.wantOverflow {
				t.Fatalf("readPinnedToolchainWith() error = %v", err)
			}
		})
	}
}

func TestGoCommandQueriesRejectMalformedData(t *testing.T) {
	root := t.TempDir()
	command := filepath.Join(root, "bin", "go")
	if err := os.MkdirAll(filepath.Dir(command), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf 'relative\\n'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadModuleCache(context.Background(), root); err == nil || !strings.Contains(err.Error(), "invalid module cache") {
		t.Fatalf("ReadModuleCache() error = %v", err)
	}
	if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf '{bad json}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := DownloadModule(context.Background(), root, ModuleIdentity{Path: "example.com/m", Version: "v1.0.0", Sum: "h1:abc"}); err == nil || !strings.Contains(err.Error(), "download pinned module example.com/m@v1.0.0") {
		t.Fatalf("DownloadModule() error = %v", err)
	}
}
