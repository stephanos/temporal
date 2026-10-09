package target

import (
	"context"
	"debug/buildinfo"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	"go.temporal.io/server/tools/gomad3/record"
	targetbuild "go.temporal.io/server/tools/gomad3/target/internal/build"
	"go.temporal.io/server/tools/gomad3/target/internal/gocommand"
)

func TestGoCommandSourcePreparationRestoresWholeIdentity(t *testing.T) {
	fixture, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	info, err := buildinfo.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	module := writeModule(t, map[string]string{
		"go.mod":  "module example.com/task9\n\ngo 1.27.1\n",
		"main.go": "package main\nfunc main() {}\n",
	})
	root := t.TempDir()
	key := strings.Repeat("9", 64)
	writeToolchainInstallation(t, root, key)
	isolatePreparedTargetCache(t)
	cache := filepath.Join(root, "builds", key, "target-cache")
	builds := 0
	commands := []string{}
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	runner := gocommand.New(func(actual context.Context, request hostexec.Request) (hostexec.Result, error) {
		wantContext := ctx
		if request.Command[1] == "env" {
			wantContext = context.Background()
		}
		if actual != wantContext || request.Timeout != 15*time.Minute || request.TerminateGrace != 100*time.Millisecond || !request.PreserveCommandError || request.CombinedOutput != (request.Command[1] == "build") {
			t.Fatalf("command policy = %#v", request)
		}
		commands = append(commands, request.Command[1])
		result := hostexec.Result{Termination: hostexec.TerminationExit}
		wantEnv := targetbuild.Environment()
		switch request.Command[1] {
		case "env":
			if !reflect.DeepEqual(request.Command, []string{filepath.Join(root, "bin", "go"), "env", "GOVERSION", "GOOS", "GOARCH", "CGO_ENABLED"}) || request.Dir != root || request.OutputLimit != maximumGoEnvironmentBytes {
				t.Fatalf("identity request = %#v", request)
			}
			result.Stdout.RawBytes = []byte("go1.27.1\n" + runtime.GOOS + "\n" + runtime.GOARCH + "\n0\n")
		case "list":
			if !reflect.DeepEqual(request.Command, []string{filepath.Join(root, "bin", "go"), "list", "-deps", "-json", "-mod=readonly", "-tags", "test_dep", "."}) || request.Dir != module || request.OutputLimit != maximumCapabilityReviewOutputBytes {
				t.Fatalf("listing request = %#v", request)
			}
			listing, err := json.Marshal(listedPackage{ImportPath: "example.com/task9", Name: "main", Dir: module, GoFiles: []string{"main.go"}, Module: &listedModule{Path: "example.com/task9", Main: true, GoVersion: "1.27.1", GoMod: filepath.Join(module, "go.mod")}})
			if err != nil {
				return hostexec.Result{}, err
			}
			result.Stdout.RawBytes = listing
		case "build":
			builds++
			output := request.Command[slices.Index(request.Command, "-o")+1]
			want := []string{filepath.Join(root, "bin", "go"), "build", "-trimpath", "-buildvcs=false", "-o", output, "-tags", "test_dep", "."}
			if !reflect.DeepEqual(request.Command, want) || request.Dir != module || request.OutputLimit != maximumGoBuildDiagnosticBytes {
				t.Fatalf("build request = %#v", request)
			}
			wantEnv = append(wantEnv, "GOCACHE="+cache)
			if lock, err := hostfs.Try(filepath.Join(cache, "gomad-cache.lock")); !errors.Is(err, hostfs.ErrContended) {
				if lock != nil {
					if err := lock.Release(); err != nil {
						t.Fatal(err)
					}
				}
				t.Fatalf("build does not hold shared cache lock: %v", err)
			}
			if err := os.WriteFile(output, data, 0o700); err != nil {
				return hostexec.Result{}, err
			}
		default:
			t.Fatalf("unexpected command = %#v", request.Command)
		}
		if !reflect.DeepEqual(request.Env, wantEnv) {
			t.Fatalf("environment = %#v, want %#v", request.Env, wantEnv)
		}
		return result, nil
	})
	spec := Spec{Kind: KindGoRun, Source: ".", WorkingDir: module, PreparationRoot: t.TempDir(), ToolchainRoot: root, BuildTags: []string{"test_dep"}, Args: []string{"one", "two words"}}
	want := Prepared{
		Kind: KindGoRun, Source: ".", SHA256: string(record.HashBytes(data)), Size: uint64(len(data)),
		Argv: []string{"gomad3-target", "one", "two words"}, BuildTags: []string{"test_dep"},
		Adapters: []record.TargetAdapter{}, Compatibility: []record.CompatibilityPack{}, BuildInfo: ProjectBuildInfo(info),
		GoVersion: "go1.27.1", BuildKey: key, TargetGOOS: runtime.GOOS, TargetGOARCH: runtime.GOARCH, CapabilityMode: CapabilityModeClosure,
	}
	for _, name := range []string{"fresh", "restored", "changed source"} {
		t.Run(name, func(t *testing.T) {
			if name == "changed source" {
				if err := os.WriteFile(filepath.Join(module, "main.go"), []byte("package main\nfunc main() { println(1) }\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			prepared, err := prepareWith(ctx, spec, runner)
			if err != nil {
				t.Fatal(err)
			}
			if err := prepared.Verify(); err != nil {
				t.Fatal(err)
			}
			if file, err := os.Stat(prepared.Path); err != nil || file.Mode().Perm() != 0o500 {
				t.Fatalf("prepared binary = %v, %v", file, err)
			}
			prepared.Path = ""
			if !reflect.DeepEqual(prepared, want) || !reflect.DeepEqual(prepared.RecordTarget(), want.RecordTarget()) || prepared.RecordToolchain() != want.RecordToolchain() {
				t.Fatalf("whole prepared/provenance differs: got %#v, want %#v", prepared, want)
			}
			requireSourceCacheReleased(t, cache)
		})
		wantBuilds := 1
		if name == "changed source" {
			wantBuilds = 2
		}
		if builds != wantBuilds {
			t.Fatalf("builds after %s = %d, want %d", name, builds, wantBuilds)
		}
	}
	if !reflect.DeepEqual(commands, []string{"env", "list", "build", "env", "list", "env", "list", "build"}) {
		t.Fatalf("commands = %v", commands)
	}
}

func TestGoCommandSourceBuildReleasesCacheForEveryOutcome(t *testing.T) {
	failure := errors.New("source command infrastructure failed")
	for _, test := range []struct {
		name   string
		result hostexec.Result
		err    error
		want   error
		text   string
	}{
		{name: "success", result: hostexec.Result{Termination: hostexec.TerminationExit}},
		{name: "compiler", result: hostexec.Result{Termination: hostexec.TerminationExit, ExitCode: 7, Stderr: hostexec.Output{Bytes: []byte("compiler diagnostic")}}, text: "prepare go-run target: exit status 7: compiler diagnostic"},
		{name: "cancel", result: hostexec.Result{Cancelled: true}, want: context.Canceled, text: "prepare go-run target: context canceled: "},
		{name: "timeout", result: hostexec.Result{WatchdogTimeout: true}, want: context.DeadlineExceeded, text: "prepare go-run target: command watchdog exceeded 15m0s: context deadline exceeded: "},
		{name: "infrastructure", err: failure, want: failure, text: "prepare go-run target: source command infrastructure failed: "},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			key := strings.Repeat("8", 64)
			writeToolchainInstallation(t, root, key)
			identity, err := readPinnedToolchainWith(t.Context(), root, identityRunner("go1.27.1\n"+runtime.GOOS+"\n"+runtime.GOARCH+"\n0\n", nil))
			if err != nil {
				t.Fatal(err)
			}
			cache := identity.installation.PinnedBuild().TargetCache()
			calls := 0
			runner := gocommand.New(func(context.Context, hostexec.Request) (hostexec.Result, error) {
				calls++
				return test.result, test.err
			})
			got, err := buildGoTargetWith(t.Context(), Spec{Kind: KindGoRun, CapabilityMode: CapabilityModeClosure}, nil, identity, filepath.Join(t.TempDir(), "target"), identity.installation.GoCommand(), t.TempDir(), ".", CapabilityReview{}, rejectUnsupported, nil, runner)
			if test.text == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || err.Error() != test.text || test.want != nil && !errors.Is(err, test.want) || !reflect.DeepEqual(got, preparation{}) {
				t.Fatalf("build outcome = %#v, %T %v", got, err, err)
			}
			if test.name == "compiler" {
				var exit *gocommand.ExitError
				if !errors.As(err, &exit) || exit.Code != 7 || exit.Signal != "" {
					t.Fatalf("compiler cause = %T %v", err, err)
				}
			}
			if calls != 1 {
				t.Fatalf("command calls = %d", calls)
			}
			requireSourceCacheReleased(t, cache)
		})
	}
}

func requireSourceCacheReleased(t *testing.T, cache string) {
	t.Helper()
	lock, err := hostfs.Try(filepath.Join(cache, "gomad-cache.lock"))
	if err != nil {
		t.Fatalf("exclusive cache reacquisition failed: %v", err)
	}
	if err := lock.Release(); err != nil {
		t.Fatal(err)
	}
}

func TestGoCommandSourcePublicQueries(t *testing.T) {
	root := t.TempDir()
	command := filepath.Join(root, "bin", "go")
	if err := os.MkdirAll(filepath.Dir(command), 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(t.TempDir(), "command")
	t.Setenv("TASK9_COMMAND_MARKER", marker)
	writeCommand := func(body string) {
		t.Helper()
		script := "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" > \"$TASK9_COMMAND_MARKER\"\n" + body + "\n"
		if err := os.WriteFile(command, []byte(script), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	cache := t.TempDir()
	for _, test := range []struct {
		name, output, want string
	}{
		{"cache", cache + "\n", ""},
		{"relative cache", "relative\n", "invalid module cache"},
		{"multiple cache lines", cache + "\nextra\n", "invalid module cache"},
		{"missing cache", filepath.Join(cache, "absent") + "\n", "resolve pinned module cache"},
	} {
		t.Run(test.name, func(t *testing.T) {
			writeCommand("printf '%s' '" + test.output + "'")
			got, err := ReadModuleCache(t.Context(), root)
			if test.want == "" {
				if err != nil || got != cache {
					t.Fatalf("cache = %q, %v", got, err)
				}
			} else if got != "" || err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("cache rejection = %q, %v", got, err)
			}
			request, err := os.ReadFile(marker)
			if err != nil || string(request) != root+"\nenv\nGOMODCACHE\n" {
				t.Fatalf("cache request = %q, %v", request, err)
			}
		})
	}
	for _, test := range []struct {
		name, body, want string
		exit, syntax     bool
	}{
		{name: "download", body: `printf '{"Sum":"h1:abc"}'`},
		{name: "checksum", body: `printf '{"Sum":"h1:other"}'`, want: `pinned module example.com/m@v1.0.0 checksum mismatch: got "h1:other", want "h1:abc"`},
		{name: "reported download error", body: `printf '{"Error":"module refused"}'; exit 7`, want: "download pinned module example.com/m@v1.0.0: module refused"},
		{name: "command error", body: `printf '{"Sum":"h1:abc"}'; printf 'diagnostic' >&2; exit 7`, want: "download pinned module example.com/m@v1.0.0: exit status 7: diagnostic", exit: true},
		{name: "malformed and command error", body: `printf '{bad}'; printf 'diagnostic' >&2; exit 7`, want: "download pinned module example.com/m@v1.0.0: exit status 7", exit: true, syntax: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			writeCommand(test.body)
			err := DownloadModule(t.Context(), root, ModuleIdentity{Path: "example.com/m", Version: "v1.0.0", Sum: "h1:abc"})
			if test.want == "" {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("download = %T %v, want %s", err, err, test.want)
			}
			var exit *exec.ExitError
			var syntax *json.SyntaxError
			if errors.As(err, &exit) != test.exit || errors.As(err, &syntax) != test.syntax || test.exit && (exit.ExitCode() != 7 || exit.Stderr != nil) {
				t.Fatalf("download causes = %T %v", err, err)
			}
			request, err := os.ReadFile(marker)
			if err != nil {
				t.Fatal(err)
			}
			directory, args, ok := strings.Cut(string(request), "\n")
			if !ok || directory == root || args != "mod\ndownload\n-json\nexample.com/m@v1.0.0\n" {
				t.Fatalf("download request = %q", request)
			}
			if _, err := os.Stat(directory); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("download directory remains: %v", err)
			}
		})
	}
	t.Chdir(root)
	writeCommand("printf 'fmt\\nos\\n'")
	closure := CapabilityClosure{Packages: []CapabilityPackage{{ImportPath: "fmt", Standard: true}, {ImportPath: "example.com/target"}}}
	if err := validateExecStandardPackages(t.Context(), command, closure); err != nil {
		t.Fatal(err)
	}
	closure.Packages[1].Standard = true
	if err := validateExecStandardPackages(t.Context(), command, closure); err == nil || err.Error() != "exec provenance standard package classification is invalid for example.com/target" {
		t.Fatalf("standard membership = %v", err)
	}
	request, err := os.ReadFile(marker)
	if err != nil || string(request) != fmt.Sprintf("%s\nlist\nstd\n", root) {
		t.Fatalf("standard request = %q, %v", request, err)
	}
}

func TestGoCommandSourceDownloadCancellationPreservesDecoderPrecedence(t *testing.T) {
	for _, name := range []string{"before cancellation", "acknowledged cancellation", "acknowledged deadline"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			command := filepath.Join(root, "bin", "go")
			if err := os.Mkdir(filepath.Dir(command), 0o700); err != nil {
				t.Fatal(err)
			}
			marker := filepath.Join(root, "started")
			body := "#!/bin/sh\nprintf '{\"Error\":\"module refused\"}'\nprintf diagnostic >&2\n" + "touch '" + marker + "'\nexec /bin/sleep 30\n"
			if err := os.WriteFile(command, []byte(body), 0o700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			switch name {
			case "before cancellation":
				cancel()
			case "acknowledged deadline":
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), time.Second)
				defer cancel()
			}
			done := make(chan error, 1)
			go func() {
				done <- DownloadModule(ctx, root, ModuleIdentity{Path: "example.com/m", Version: "v1.0.0", Sum: "h1:abc"})
			}()
			if name != "before cancellation" {
				ticker := time.NewTicker(time.Millisecond)
				defer ticker.Stop()
				started := false
				for !started {
					select {
					case <-ticker.C:
						_, err := os.Stat(marker)
						started = err == nil
						if err != nil && !errors.Is(err, os.ErrNotExist) {
							cancel()
							<-done
							t.Fatal(err)
						}
					case <-ctx.Done():
						<-done
						t.Fatal("download child did not acknowledge startup")
					}
				}
				if name == "acknowledged cancellation" {
					cancel()
				}
			}
			err := <-done
			var syntax *json.SyntaxError
			if name == "before cancellation" {
				want := "download pinned module example.com/m@v1.0.0: context canceled\nunexpected end of JSON input: "
				if err == nil || err.Error() != want || !errors.Is(err, context.Canceled) || !errors.As(err, &syntax) {
					t.Fatalf("before cancellation = %T %v", err, err)
				}
			} else if err == nil || err.Error() != "download pinned module example.com/m@v1.0.0: module refused" || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &syntax) {
				t.Fatalf("reported module error lost precedence = %T %v", err, err)
			}
		})
	}
}

func TestGoCommandSourceQueryOutputPreservesRawExit(t *testing.T) {
	for _, kind := range []string{"identity", "cache"} {
		for _, capacity := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s capacity=%t", kind, capacity), func(t *testing.T) {
				root := t.TempDir()
				key := strings.Repeat("9", 64)
				writeToolchainInstallation(t, root, key)
				stderr := "query diagnostic"
				body := "#!/bin/sh\nprintf 'go1.27.1\\n" + runtime.GOOS + "\\n" + runtime.GOARCH + "\\n0\\n'\n"
				if kind == "cache" {
					body = "#!/bin/sh\nprintf '%s\\n' '" + t.TempDir() + "'\n"
				}
				if capacity {
					stderr = strings.Repeat("0", 64<<10)
					body += "printf '%065536d' 0 >&2\n"
				} else {
					body += "printf 'query diagnostic' >&2\n"
				}
				body += "exit 7\n"
				if err := os.WriteFile(filepath.Join(root, "bin/go"), []byte(body), 0o700); err != nil {
					t.Fatal(err)
				}
				var err error
				if kind == "identity" {
					_, err = ReadToolchainIdentity(root)
				} else {
					_, err = ReadModuleCache(t.Context(), root)
				}
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ProcessState == nil || exit.ExitCode() != 7 || string(exit.Stderr) != stderr {
					t.Fatalf("query lost actual exit/stderr = %T %v", err, err)
				}
			})
		}
	}
}

func TestGoCommandSourceBuildPreservesRawOrderedDiagnostics(t *testing.T) {
	root := t.TempDir()
	key := strings.Repeat("9", 64)
	writeToolchainInstallation(t, root, key)
	identity, err := readPinnedToolchainWith(t.Context(), root, identityRunner("go1.27.1\n"+runtime.GOOS+"\n"+runtime.GOARCH+"\n0\n", nil))
	if err != nil {
		t.Fatal(err)
	}
	command := filepath.Join(root, "bin/go")
	if err := os.WriteFile(command, []byte("#!/bin/sh\nprintf 'stderr first\\n' >&2\nprintf 'stdout second\\n'\nexit 7\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := buildGoTargetWith(t.Context(), Spec{Kind: KindGoRun}, nil, identity, filepath.Join(t.TempDir(), "target"), command, t.TempDir(), ".", CapabilityReview{}, rejectUnsupported, nil, gocommand.Default())
	var exit *exec.ExitError
	want := "prepare go-run target: exit status 7: stderr first\nstdout second\n"
	if !errors.As(err, &exit) || exit.ProcessState == nil || exit.ExitCode() != 7 || exit.Stderr != nil || err.Error() != want || !reflect.DeepEqual(got, preparation{}) {
		t.Fatalf("build lost raw ordered diagnostics = %#v, %T %v", got, err, err)
	}
	requireSourceCacheReleased(t, filepath.Join(root, "builds", key, "target-cache"))
}

func TestGoCommandSourcePreparationIdentityKeepsBackgroundTiming(t *testing.T) {
	root := t.TempDir()
	writeToolchainInstallation(t, root, strings.Repeat("9", 64))
	module := writeModule(t, map[string]string{"go.mod": "module example.com/task9\n\ngo 1.27.1\n", "main.go": "package main\nfunc main() {}\n"})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	backgroundIdentity := false
	listingCaller := false
	runner := gocommand.New(func(actual context.Context, request hostexec.Request) (hostexec.Result, error) {
		if request.Command[1] == "env" {
			backgroundIdentity = actual == context.Background()
			return hostexec.Result{Termination: hostexec.TerminationExit, Stdout: hostexec.Output{RawBytes: []byte("go1.27.1\n" + runtime.GOOS + "\n" + runtime.GOARCH + "\n0\n")}}, nil
		}
		if request.Command[1] == "list" {
			listingCaller = actual == ctx
			return hostexec.Result{Cancelled: true, CommandError: context.Canceled}, nil
		}
		t.Fatalf("unexpected command = %v", request.Command)
		return hostexec.Result{}, nil
	})
	prepared, err := prepareWith(ctx, Spec{Kind: KindGoRun, Source: ".", WorkingDir: module, ToolchainRoot: root, PreparationRoot: t.TempDir()}, runner)
	if !backgroundIdentity || !listingCaller || !reflect.DeepEqual(prepared, Prepared{}) || !errors.Is(err, context.Canceled) || err.Error() != "inspect target capability closure: context canceled" {
		t.Fatalf("identity/list timing = %t/%t, %#v, %v", backgroundIdentity, listingCaller, prepared, err)
	}
}

func TestGoCommandSourceStandardInventoryPreservesInheritedDirectoryAndRawError(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.WriteFile(filepath.Join(root, "go"), []byte("#!/bin/sh\nprintf '%s' \"$PWD\" >&2; exit 7\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	err := validateExecStandardPackages(t.Context(), "./go", CapabilityClosure{})
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ProcessState == nil || exit.ExitCode() != 7 || string(exit.Stderr) != root || err.Error() != "inspect pinned standard packages: exit status 7: "+root {
		t.Fatalf("inherited-directory outcome = %T %v", err, err)
	}
}
