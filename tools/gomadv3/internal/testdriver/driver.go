package testdriver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/boundary"
	"go.temporal.io/server/tools/gomadv3/internal/commandrun"
	"go.temporal.io/server/tools/gomadv3/internal/testtier"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

const fixtureOutputLimit = 1 << 20
const fixtureTerminationGrace = 2 * time.Second

type Config struct {
	Root     string
	Mode     string
	Go       string
	Compiler string
}

type Report struct {
	Mode   string
	Passed bool
	Cases  []CaseResult
}

type CaseResult struct {
	Tier      string
	Name      string
	Passed    bool
	ExitCode  int
	TimedOut  bool
	Signaled  bool
	Stdout    []byte
	Stderr    []byte
	Truncated bool
}

type fixture struct {
	tier     string
	name     string
	command  []string
	dir      string
	env      []string
	timeout  time.Duration
	wantExit int
	oracle   func(commandrun.Result) error
}

func Run(ctx context.Context, config Config) (Report, error) {
	return runWith(ctx, config, commandrun.Run)
}

func runWith(ctx context.Context, config Config, run func(context.Context, commandrun.Request) (commandrun.Result, error)) (Report, error) {
	mode, err := testtier.Resolve(config.Mode)
	if err != nil {
		return Report{}, err
	}
	root, err := filepath.Abs(config.Root)
	if err != nil || root == string(filepath.Separator) {
		return Report{}, errors.Join(errors.New("gomadv3 test root must be an absolute non-root directory"), err)
	}
	goInfo, err := os.Stat(config.Go)
	if err != nil || !goInfo.Mode().IsRegular() || goInfo.Mode()&0o111 == 0 {
		return Report{}, errors.Join(errors.New("gomadv3 toolchain Go is unavailable"), err)
	}
	if slices.Contains(mode.Tiers, "test-interception") {
		compilerInfo, compilerErr := os.Stat(config.Compiler)
		if compilerErr != nil || !compilerInfo.Mode().IsRegular() || compilerInfo.Mode()&0o111 == 0 {
			return Report{}, errors.Join(errors.New("gomadv3 test compiler is unavailable"), compilerErr)
		}
	}
	report := Report{Mode: config.Mode, Cases: []CaseResult{}}
	goRoot, err := resolveGoRoot(ctx, root, config.Go, run)
	if err != nil {
		return report, err
	}
	for _, tier := range mode.Tiers {
		if tier == "test-runtime" {
			if err := runRuntimeCampaign(ctx, config, goRoot, run, &report); err != nil {
				return report, err
			}
			continue
		}
		var fixtures []fixture
		cleanup := func() error { return nil }
		switch tier {
		case "test-builder":
			fixtures = builderFixtures(config)
		case "test-upstream":
			fixtures, cleanup, err = upstreamFixtures(goRoot)
			if err != nil {
				return report, err
			}
		case "test-interception":
			fixtures, cleanup, err = interceptionFixtures(config, goRoot)
			if err != nil {
				return report, err
			}
		default:
			return report, fmt.Errorf("gomadv3 typed test tier is not implemented: %s", tier)
		}
		for index, planned := range fixtures {
			result, runErr := run(ctx, commandrun.Request{
				Command: planned.command, Dir: planned.dir, Env: planned.env, Timeout: planned.timeout,
				TerminateGrace: fixtureTerminationGrace, OutputLimit: fixtureOutputLimit,
			})
			caseResult := CaseResult{
				Tier: planned.tier, Name: planned.name, ExitCode: result.ExitCode,
				TimedOut: result.WatchdogTimeout, Signaled: result.Termination == commandrun.TerminationSignal,
				Stdout: append([]byte(nil), result.Stdout.Bytes...), Stderr: append([]byte(nil), result.Stderr.Bytes...),
				Truncated: result.Stdout.Truncated || result.Stderr.Truncated,
			}
			caseResult.Passed = runErr == nil && !caseResult.TimedOut && !caseResult.Signaled && result.Termination == commandrun.TerminationExit && result.ExitCode == planned.wantExit && !caseResult.Truncated
			if caseResult.Passed && planned.oracle != nil {
				if oracleErr := planned.oracle(result); oracleErr != nil {
					caseResult.Passed = false
					runErr = oracleErr
				}
			}
			report.Cases = append(report.Cases, caseResult)
			if runErr != nil {
				return report, errors.Join(fmt.Errorf("run gomadv3 fixture %s: %w", planned.name, runErr), cleanup())
			}
			if !caseResult.Passed {
				return report, errors.Join(fmt.Errorf("gomadv3 fixture %s failed with status %d: %s%s", planned.name, result.ExitCode, result.Stdout.Bytes, result.Stderr.Bytes), cleanup())
			}
			if index == len(fixtures)-1 {
				if err := cleanup(); err != nil {
					return report, err
				}
			}
		}
	}
	report.Passed = true
	return report, nil
}

func builderFixtures(config Config) []fixture {
	return []fixture{{
		tier: "test-builder", name: "toolchain-builder-unit",
		command: []string{config.Go, "test", "-count=1", "-tags=test_dep", "./internal/toolchainbuild"},
		dir:     config.Root, env: append(filterEnvironment(os.Environ(), "GOMADSEED", "GOMADV3_CHILD_SEED", "GOWORK"), "GOWORK=off"),
		timeout: 10 * time.Minute,
	}}
}

func interceptionFixtures(config Config, goRoot string) ([]fixture, func() error, error) {
	descriptor, err := gomadversion.Load(config.Root)
	if err != nil {
		return nil, nil, err
	}
	compilerCases, err := boundary.CompilerTestCases(config.Root)
	if err != nil {
		return nil, nil, err
	}
	expected, err := os.ReadFile(filepath.Join(config.Root, "expected-intercepts-"+descriptor.GoVersion+".txt"))
	if err != nil {
		return nil, nil, fmt.Errorf("read expected interception report: %w", err)
	}
	workspace, err := os.MkdirTemp(filepath.Join(config.Root, ".toolchain"), "interception-test-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create interception test workspace: %w", err)
	}
	cleanup := func() error { return os.RemoveAll(workspace) }
	baseEnvironment := filterEnvironment(os.Environ(), "GOMADSEED", "GOMADV3_CHILD_SEED", "GOWORK", "GOCACHE", "GOOS", "GOARCH", "CGO_ENABLED", "GOMADV3_TEST_COMPILE")
	var actualReport []string
	collectReport := func(result commandrun.Result) error {
		for _, line := range strings.Split(string(result.Stderr.RawBytes), "\n") {
			if applied, found := strings.CutPrefix(line, "gomad intercept applied: "); found {
				actualReport = append(actualReport, applied)
			}
		}
		return nil
	}
	var fixtures []fixture
	for _, packageName := range []string{"os", "net"} {
		fixtures = append(fixtures, fixture{
			tier: "test-interception", name: "interception-report-" + packageName,
			command: []string{config.Go, "install", "-a", "-gcflags=" + packageName + "=-m", packageName}, dir: config.Root,
			env:     append(append([]string(nil), baseEnvironment...), "GOCACHE="+filepath.Join(workspace, "cache-"+packageName), "GOWORK=off"),
			timeout: 10 * time.Minute, oracle: collectReport,
		})
	}
	fixtures = append(fixtures, fixture{
		tier: "test-interception", name: "interception-report-unsupported-platform",
		command: []string{config.Go, "install", "-a", "-gcflags=os=-m", "os"}, dir: config.Root,
		env: append(append([]string(nil), baseEnvironment...),
			"CGO_ENABLED=0", "GOOS=linux", "GOARCH=arm64", "GOCACHE="+filepath.Join(workspace, "cache-unsupported"), "GOWORK=off"),
		timeout: 10 * time.Minute,
		oracle: func(result commandrun.Result) error {
			if strings.Contains(string(result.Stderr.RawBytes), "gomad intercept applied: ") {
				return errors.New("compiler applied interceptions to unsupported linux/arm64 target")
			}
			slices.Sort(actualReport)
			actual := strings.Join(actualReport, "\n") + "\n"
			if actual != string(expected) {
				return fmt.Errorf("applied interceptions differ from the %s manifest: got %q, want %q", descriptor.GoVersion, actual, expected)
			}
			return nil
		},
	})
	testdata := filepath.Join(config.Root, "testdata")
	toolexec := filepath.Join(config.Root, "compiler_test_exec.sh")
	fixtures = append(fixtures, fixture{
		tier: "test-interception", name: "interception-baseline-missing-hook",
		command: []string{config.Go, "test", "./interceptfail/missing_hook"}, dir: testdata,
		env: append(append([]string(nil), baseEnvironment...), "GOWORK=off"), timeout: time.Minute,
	})
	seenNegative := make(map[string]struct{})
	for _, compilerCase := range compilerCases {
		if compilerCase.Diagnostic == "" {
			continue
		}
		name := strings.TrimPrefix(compilerCase.Package, "gomadv3.test/interceptfail/")
		if name == compilerCase.Package {
			cleanup()
			return nil, nil, fmt.Errorf("negative compiler fixture package is invalid: %s", compilerCase.Package)
		}
		if _, duplicate := seenNegative[name]; duplicate {
			continue
		}
		seenNegative[name] = struct{}{}
		diagnostic := compilerCase.Diagnostic
		fixtures = append(fixtures, fixture{
			tier: "test-interception", name: "interception-negative-" + name,
			command: []string{config.Go, "test", "-toolexec=" + toolexec, "./interceptfail/" + name}, dir: testdata,
			env: append(append([]string(nil), baseEnvironment...),
				"GOMADV3_TEST_COMPILE="+config.Compiler, "GOCACHE="+filepath.Join(workspace, "cache-negative-"+name), "GOWORK=off"),
			timeout: time.Minute, wantExit: 1,
			oracle: func(result commandrun.Result) error {
				if !strings.Contains(string(result.Stderr.RawBytes), diagnostic) {
					return fmt.Errorf("compiler omitted interception diagnostic %q", diagnostic)
				}
				return nil
			},
		})
	}
	for _, disabledInlining := range []bool{false, true} {
		command := []string{config.Go, "test", "-toolexec=" + toolexec}
		name := "interception-invocation"
		if disabledInlining {
			command = append(command, "-gcflags=all=-l")
			name += "-no-inline"
		}
		command = append(command, "./intercept", "-count=1")
		fixtures = append(fixtures, fixture{
			tier: "test-interception", name: name, command: command, dir: testdata,
			env:     append(append([]string(nil), baseEnvironment...), "GOMADV3_TEST_COMPILE="+config.Compiler, "GOWORK=off"),
			timeout: time.Minute,
		})
	}
	_ = goRoot
	return fixtures, cleanup, nil
}

func resolveGoRoot(ctx context.Context, root, goCommand string, run func(context.Context, commandrun.Request) (commandrun.Result, error)) (string, error) {
	result, err := executeCommand(ctx, run, commandrun.Request{
		Command: []string{goCommand, "env", "GOROOT"}, Dir: root,
		Env:     filterEnvironment(os.Environ(), "GOMADSEED", "GOMADV3_CHILD_SEED", "GOROOT"),
		Timeout: 30 * time.Second, TerminateGrace: fixtureTerminationGrace, OutputLimit: 64 << 10,
	})
	if err != nil {
		return "", fmt.Errorf("resolve gomadv3 toolchain root: %w", err)
	}
	goRoot := strings.TrimSpace(string(result.Stdout.RawBytes))
	info, err := os.Stat(filepath.Join(goRoot, "src"))
	if err != nil || !info.IsDir() {
		return "", errors.Join(errors.New("gomadv3 toolchain root has no source directory"), err)
	}
	return goRoot, nil
}

func upstreamFixtures(goRoot string) ([]fixture, func() error, error) {
	workspace, err := os.MkdirTemp("", "gomadv3-upstream-*")
	if err != nil {
		return nil, nil, fmt.Errorf("create upstream test workspace: %w", err)
	}
	linkedRoot := filepath.Join(workspace, "go")
	if err := os.Symlink(goRoot, linkedRoot); err != nil {
		return nil, nil, errors.Join(fmt.Errorf("link upstream Go root: %w", err), os.RemoveAll(workspace))
	}
	goCommand := filepath.Join(linkedRoot, "bin", "go")
	workingDirectory := filepath.Join(linkedRoot, "src")
	baseEnvironment := filterEnvironment(os.Environ(), "GOMADSEED", "GOMADV3_CHILD_SEED", "GO111MODULE", "GODEBUG", "GOWORK", "GOMADV3_IO_PROFILE")
	return []fixture{
		{
			tier: "test-upstream", name: "upstream-clock", command: []string{
				goCommand, "test", "-tags=test_dep", "runtime", "time", "testing/synctest",
			}, dir: workingDirectory, env: baseEnvironment, timeout: 10 * time.Minute,
		},
		{
			tier: "test-upstream", name: "upstream-dist", command: []string{
				goCommand, "tool", "dist", "test", "-no-rebuild",
				"-run=^(bytes|context|encoding/json|io|cmd/compile/internal/gc|cmd/compile/internal/ssa|cmd/link/internal/ld|cmd/link/internal/loader|cmd/go/internal/load|cmd/go/internal/modload|cmd/go/internal/work)$",
			}, dir: workingDirectory, env: append(baseEnvironment, "GOTOOLCHAIN=local", "GOWORK=off"), timeout: 10 * time.Minute,
		},
	}, func() error { return os.RemoveAll(workspace) }, nil
}

func executeCommand(ctx context.Context, run func(context.Context, commandrun.Request) (commandrun.Result, error), request commandrun.Request) (commandrun.Result, error) {
	result, err := run(ctx, request)
	if err != nil {
		return result, err
	}
	if result.WatchdogTimeout {
		return result, context.DeadlineExceeded
	}
	if result.Cancelled {
		return result, context.Canceled
	}
	if result.Termination != commandrun.TerminationExit || result.ExitCode != 0 {
		return result, fmt.Errorf("command failed with status %d: %s%s", result.ExitCode, result.Stdout.Bytes, result.Stderr.Bytes)
	}
	return result, nil
}

func filterEnvironment(environment []string, names ...string) []string {
	filtered := make([]string, 0, len(environment))
	for _, value := range environment {
		keep := true
		for _, name := range names {
			if strings.HasPrefix(value, name+"=") {
				keep = false
				break
			}
		}
		if keep {
			filtered = append(filtered, value)
		}
	}
	return filtered
}
