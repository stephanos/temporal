package target

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

func TestAdapterSourceSetPublicStreamBounds(t *testing.T) {
	const capacity = 4 << 20
	for _, test := range []struct {
		name           string
		stdout, stderr int
		stream         string
	}{
		{name: "stdout exact", stdout: capacity},
		{name: "stderr exact", stderr: capacity},
		{name: "stdout overflow", stdout: capacity + 1, stream: "stdout"},
		{name: "stderr overflow", stderr: capacity + 1, stream: "stderr"},
		{name: "both overflow", stdout: capacity + 1, stderr: capacity + 1, stream: "stdout"},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			binary, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			command := filepath.Join(root, "go")
			wrapper := fmt.Sprintf("#!/bin/sh\nexec '%s' -test.run='^TestAdapterSourceSetChild$' -- \"$@\"\n", strings.ReplaceAll(binary, "'", "'\\''"))
			if err := os.WriteFile(command, []byte(wrapper), 0o700); err != nil {
				t.Fatal(err)
			}
			config, err := json.Marshal(struct {
				Root           string
				Stdout, Stderr int
			}{root, test.stdout, test.stderr})
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("ADAPTER_SOURCE_SET_CHILD", string(config))
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			digest, err := AdapterPreparedSourceSetSHA256(ctx, command, root, "example.test/adapter", "darwin", "arm64")
			marker, readErr := os.ReadFile(filepath.Join(root, "started.json"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			var started struct {
				PID    int
				GOPATH string
			}
			if err := json.Unmarshal(marker, &started); err != nil {
				t.Fatal(err)
			}
			if !errors.Is(syscall.Kill(started.PID, 0), syscall.ESRCH) {
				t.Fatal("child remains after helper returned")
			}
			if _, err := os.Stat(started.GOPATH); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("GOPATH cleanup = %v", err)
			}
			var overflow *gocommand.OverflowError
			if test.stream == "" {
				if err != nil || digest == "" {
					t.Fatalf("exact capacity = %q, %v", digest, err)
				}
			} else if !errors.As(err, &overflow) || overflow.Stream != test.stream || overflow.Limit != capacity || digest != "" {
				t.Fatalf("capacity + 1 = %q, %T %v; want %s overflow and empty digest", digest, err, err, test.stream)
			}
		})
	}
}

func TestAdapterSourceSetChild(t *testing.T) {
	config := os.Getenv("ADAPTER_SOURCE_SET_CHILD")
	if config == "" {
		return
	}
	var settings struct {
		Root           string
		Stdout, Stderr int
	}
	if err := json.Unmarshal([]byte(config), &settings); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(os.Args[len(os.Args)-5:], []string{"list", "-e", "-find", "-json", "."}) {
		t.Fatal("unexpected child argv")
	}
	marker, err := json.Marshal(struct {
		PID    int
		GOPATH string
	}{os.Getpid(), os.Getenv("GOPATH")})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(settings.Root, "started.json"), marker, 0o600); err != nil {
		t.Fatal(err)
	}
	listing := fmt.Sprintf(`{"Dir":%q,"Name":"adapter","ImportPath":"ignored"}`, settings.Root)
	if _, err := io.WriteString(os.Stdout, listing); err != nil {
		t.Fatal(err)
	}
	if settings.Stdout > len(listing) {
		if _, err := io.WriteString(os.Stdout, strings.Repeat(" ", settings.Stdout-len(listing))); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := io.WriteString(os.Stderr, strings.Repeat(" ", settings.Stderr)); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

func TestAdapterSourceSetRequestAndLiteralProjection(t *testing.T) {
	root := adapterSourceSetFixture(t)
	t.Setenv("GO111MODULE", "on")
	t.Setenv("GOPATH", "ambient-gopath")
	t.Setenv("GOOS", "windows")
	t.Setenv("GOARCH", "386")
	for _, test := range []struct{ goos, goarch, directory, want string }{
		{"darwin", "arm64", "", "sha256:2eeca986c81436b15fed9df5b22be9aa11d1b3a6349917817fe26497fe6527f0"},
		{"linux", "amd64", "relative/package", "sha256:f07021259f831522a922eb7f5e36a72d58a7d5efcd96c5197945d79484579191"},
	} {
		t.Run(test.goos, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			var gopath string
			runner := gocommand.New(func(actual context.Context, request hostexec.Request) (hostexec.Result, error) {
				if actual != ctx {
					t.Fatal("caller context was replaced")
				}
				deadline, ok := actual.Deadline()
				wantDeadline, _ := ctx.Deadline()
				if !ok || deadline != wantDeadline {
					t.Fatal("caller deadline was replaced")
				}
				if !reflect.DeepEqual(request.Command, []string{"./go", "list", "-e", "-find", "-json", "."}) || request.Dir != test.directory {
					t.Fatalf("command/directory = %v, %q", request.Command, request.Dir)
				}
				if !request.PreserveCommandError || request.Timeout != 15*time.Minute || request.TerminateGrace != 100*time.Millisecond || request.OutputLimit != 4<<20 {
					t.Fatalf("command policy = %#v", request)
				}
				gopath = strings.TrimPrefix(request.Env[len(request.Env)-3], "GOPATH=")
				if info, err := os.Stat(gopath); err != nil || !info.IsDir() {
					t.Fatalf("GOPATH during call = %v, %v", info, err)
				}
				wantEnv := append(targetbuild.Environment(), "GO111MODULE=off", "GOPATH="+gopath, "GOOS="+test.goos, "GOARCH="+test.goarch)
				if !reflect.DeepEqual(request.Env, wantEnv) {
					t.Fatalf("environment = %v, want %v", request.Env, wantEnv)
				}
				platform := test.goos + "_" + test.goarch
				listing := map[string]any{"Dir": root, "Name": "adapter", "ImportPath": "wrong/import", "GoFiles": []string{"selected_" + platform + ".go", "common.go", "common.go"}, "HFiles": []string{"common.h"}, "SFiles": []string{"selected_" + platform + ".s", "common.s"}, "IgnoredGoFiles": []string{"excluded_windows.go", "disabled_cgo.go"}, "TestGoFiles": []string{"excluded_test.go"}, "Error": map[string]string{"Err": "directory expects import \"example.test/adapter\""}}
				return adapterSourceSetListing(t, listing), nil
			})
			digest, err := adapterPreparedSourceSetSHA256With(ctx, "./go", test.directory, "example.test/adapter", test.goos, test.goarch, runner)
			if err != nil || digest != test.want {
				t.Fatalf("literal source projection = %q, %v; want %s", digest, err, test.want)
			}
			if _, err := os.Stat(gopath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("GOPATH after call = %v", err)
			}
		})
	}
}

func TestAdapterSourceSetDecodeAndSourceFailures(t *testing.T) {
	root := adapterSourceSetFixture(t)
	valid := fmt.Sprintf(`{"Dir":%q,"Name":"adapter","ImportPath":"ignored"}`, root)
	for _, test := range []struct {
		name, listing, want string
		sourceError         bool
	}{
		{"malformed", "{broken", "decode prepared package example.test/adapter listing", false},
		{"trailing", valid + " {}", "decode prepared package example.test/adapter listing", false},
		{"missing directory", `{"Name":"adapter"}`, "prepared package listing has no directory or name", false},
		{"missing name", fmt.Sprintf(`{"Dir":%q}`, root), "prepared package listing has no directory or name", false},
		{"wrong quotes", `{"Error":{"Err":"directory expects import 'example.test/adapter'"}}`, "directory expects import 'example.test/adapter'", false},
		{"trailing import comment", `{"Error":{"Err":"directory expects import \"example.test/adapter\" trailing"}}`, "trailing", false},
		{"other import comment", `{"Error":{"Err":"directory expects import \"example.test/other\""}}`, "example.test/other", false},
		{"listed error", `{"Error":{"Err":"no Go files"}}`, "no Go files", false},
		{"Go read failure", fmt.Sprintf(`{"Dir":%q,"Name":"adapter","GoFiles":["absent.go"]}`, root), "unreadable source absent.go", true},
		{"foreign read failure", fmt.Sprintf(`{"Dir":%q,"Name":"adapter","HFiles":["absent.h"]}`, root), "unreadable source absent.h", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var gopath string
			runner := gocommand.New(func(_ context.Context, request hostexec.Request) (hostexec.Result, error) {
				gopath = strings.TrimPrefix(request.Env[len(request.Env)-3], "GOPATH=")
				return hostexec.Result{Stdout: hostexec.Output{RawBytes: []byte(test.listing)}}, nil
			})
			digest, err := adapterPreparedSourceSetSHA256With(t.Context(), "go", root, "example.test/adapter", "darwin", "arm64", runner)
			if digest != "" || err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("failed projection = %q, %v; want %s", digest, err, test.want)
			}
			if test.sourceError && !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("source error lost original cause: %v", err)
			}
			if _, err := os.Stat(gopath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("GOPATH after failure = %v", err)
			}
		})
	}
}

func TestAdapterSourceSetExecutionPrecedesDecode(t *testing.T) {
	raw := &os.PathError{Op: "fork/exec", Path: "go", Err: syscall.ENOENT}
	nonzero := exec.CommandContext(t.Context(), "/bin/sh", "-c", "exit 7").Run()
	var exit *exec.ExitError
	if !errors.As(nonzero, &exit) || exit.ExitCode() != 7 {
		t.Fatalf("nonzero fixture = %T %v", nonzero, nonzero)
	}
	cleanup := errors.New("command cleanup failed")
	infrastructure := errors.Join(raw, cleanup)
	for _, test := range []struct {
		name                  string
		result                hostexec.Result
		infrastructure, cause error
		stream                string
		watchdog              bool
		stderr                string
	}{
		{name: "raw command", result: hostexec.Result{CommandError: raw}, cause: raw, stderr: "diagnostic"},
		{name: "raw nonzero", result: hostexec.Result{Termination: hostexec.TerminationExit, ExitCode: 7, CommandError: nonzero}, cause: nonzero, stderr: "diagnostic"},
		{name: "raw canceled", result: hostexec.Result{Cancelled: true, CommandError: context.Canceled}, cause: context.Canceled, stderr: "diagnostic"},
		{name: "raw deadline", result: hostexec.Result{Cancelled: true, CommandError: context.DeadlineExceeded}, cause: context.DeadlineExceeded, stderr: "diagnostic"},
		{name: "watchdog", result: hostexec.Result{WatchdogTimeout: true, CommandError: raw}, cause: raw, watchdog: true, stderr: "diagnostic"},
		{name: "stdout overflow plus raw", result: hostexec.Result{CommandError: raw, Stdout: hostexec.Output{Truncated: true}}, stream: "stdout", stderr: "diagnostic"},
		{name: "stdout overflow plus nonzero", result: hostexec.Result{Termination: hostexec.TerminationExit, ExitCode: 7, CommandError: nonzero, Stdout: hostexec.Output{Truncated: true}}, stream: "stdout", stderr: "diagnostic"},
		{name: "stderr overflow plus raw", result: hostexec.Result{CommandError: raw, Stderr: hostexec.Output{Truncated: true}}, stream: "stderr"},
		{name: "dual overflow plus cancellation", result: hostexec.Result{Cancelled: true, CommandError: context.Canceled, Stdout: hostexec.Output{Truncated: true}, Stderr: hostexec.Output{Truncated: true}}, stream: "stdout"},
		{name: "stdout overflow plus cancellation", result: hostexec.Result{Cancelled: true, CommandError: context.Canceled, Stdout: hostexec.Output{Truncated: true}}, stream: "stdout", stderr: "diagnostic"},
		{name: "overflow plus watchdog", result: hostexec.Result{WatchdogTimeout: true, CommandError: raw, Stdout: hostexec.Output{Truncated: true}}, stream: "stdout", stderr: "diagnostic"},
		{name: "cleanup plus dual overflow", result: hostexec.Result{CommandError: raw, WatchdogTimeout: true, Stdout: hostexec.Output{Truncated: true}, Stderr: hostexec.Output{Truncated: true}}, infrastructure: infrastructure, cause: infrastructure},
		{name: "cleanup plus canceled", result: hostexec.Result{Cancelled: true, CommandError: context.Canceled}, infrastructure: cleanup, cause: cleanup, stderr: "diagnostic"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			var gopath string
			runner := gocommand.New(func(actual context.Context, request hostexec.Request) (hostexec.Result, error) {
				if actual != ctx {
					t.Fatal("canceled caller context replaced")
				}
				gopath = strings.TrimPrefix(request.Env[len(request.Env)-3], "GOPATH=")
				result := test.result
				result.Stdout.RawBytes = []byte(`{"Dir":"/plausible","Name":"adapter"}`)
				result.Stderr.RawBytes = []byte(" \tdiagnostic\n\n ")
				return result, test.infrastructure
			})
			digest, err := adapterPreparedSourceSetSHA256With(ctx, "go", "", "example.test/adapter", "darwin", "arm64", runner)
			if digest != "" || err == nil {
				t.Fatalf("execution failure = %q, %v", digest, err)
			}
			var overflow *gocommand.OverflowError
			var watchdog *gocommand.WatchdogError
			if errors.As(err, &overflow) != (test.stream != "") || errors.As(err, &watchdog) != test.watchdog {
				t.Fatalf("error type = %T %v", err, err)
			}
			if test.stream != "" && (overflow.Stream != test.stream || overflow.Limit != 4<<20) {
				t.Fatalf("overflow = %#v", overflow)
			}
			if test.watchdog && watchdog.Timeout != 15*time.Minute {
				t.Fatalf("watchdog = %#v", watchdog)
			}
			if test.cause != nil && !errors.Is(err, test.cause) {
				t.Fatalf("original cause lost: %v", err)
			}
			if test.infrastructure == infrastructure && (!errors.Is(err, raw) || !errors.Is(err, cleanup) || errors.Unwrap(err) != infrastructure) {
				t.Fatalf("cleanup cause/order lost: %v", err)
			}
			if err.Error() != fmt.Sprintf("list prepared package example.test/adapter for darwin/arm64: %v: %s", errors.Unwrap(err), test.stderr) {
				t.Fatalf("outer error = %q", err.Error())
			}
			if _, err := os.Stat(gopath); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("GOPATH after execution failure = %v", err)
			}
		})
	}
}

func adapterSourceSetListing(t *testing.T, listing map[string]any) hostexec.Result {
	t.Helper()
	data, err := json.Marshal(listing)
	if err != nil {
		t.Fatal(err)
	}
	return hostexec.Result{Stdout: hostexec.Output{RawBytes: data}}
}

func adapterSourceSetFixture(t *testing.T) string {
	t.Helper()
	return writeModule(t, map[string]string{
		"common.go":                "package adapter\nconst Common = 1\n",
		"selected_darwin_arm64.go": "package adapter\nconst Platform = \"darwin/arm64\"\n",
		"selected_linux_amd64.go":  "package adapter\nconst Platform = \"linux/amd64\"\n",
		"excluded_windows.go":      "package adapter\nconst Excluded = true\n",
		"excluded_test.go":         "package adapter\n",
		"disabled_cgo.go":          "//go:build cgo\n\npackage adapter\nimport \"C\"\n",
		"common.s":                 "// Shared assembly fixture; listing only.\n",
		"selected_darwin_arm64.s":  "// Darwin assembly fixture; listing only.\n",
		"selected_linux_amd64.s":   "// Linux assembly fixture; listing only.\n",
		"common.h":                 "// Shared header fixture; listing only.\n",
	})
}
