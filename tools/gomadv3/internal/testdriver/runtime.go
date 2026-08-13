package testdriver

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/commandrun"
	gomadversion "go.temporal.io/server/tools/gomadv3/internal/version"
)

var randomLinePattern = regexp.MustCompile(`^[0-9a-f]{16} [0-9a-f]{8}$`)
var goTestDurationPattern = regexp.MustCompile(`(?m)^(ok[[:space:]]+[^[:space:]]+)[[:space:]]+[0-9.]+s$`)

type runtimeCase struct {
	name        string
	request     commandrun.Request
	wantExit    int
	wantTimeout bool
	oracle      func(commandrun.Result) error
}

type runtimeCampaign struct {
	ctx         context.Context
	config      Config
	goRoot      string
	testdata    string
	execWrapper string
	workspace   string
	run         func(context.Context, commandrun.Request) (commandrun.Result, error)
	report      *Report
}

func runRuntimeCampaign(
	ctx context.Context,
	config Config,
	goRoot string,
	run func(context.Context, commandrun.Request) (commandrun.Result, error),
	report *Report,
) (returnedErr error) {
	workspace, err := os.MkdirTemp(filepath.Join(config.Root, ".toolchain"), "runtime-test-*")
	if err != nil {
		return fmt.Errorf("create runtime test workspace: %w", err)
	}
	defer func() {
		returnedErr = errors.Join(returnedErr, os.RemoveAll(workspace))
	}()
	campaign := runtimeCampaign{
		ctx: ctx, config: config, goRoot: goRoot, testdata: filepath.Join(config.Root, "testdata"),
		execWrapper: filepath.Join(config.Root, "exec.sh"), workspace: workspace, run: run, report: report,
	}
	if err := campaign.validateInstallation(); err != nil {
		return err
	}
	return campaign.execute()
}

func (campaign *runtimeCampaign) runCase(planned runtimeCase) (commandrun.Result, error) {
	result, runErr := campaign.run(campaign.ctx, planned.request)
	return campaign.finishCase(planned, result, runErr)
}

func (campaign *runtimeCampaign) finishCase(planned runtimeCase, result commandrun.Result, runErr error) (commandrun.Result, error) {
	caseResult := CaseResult{
		Tier: "test-runtime", Name: planned.name, ExitCode: result.ExitCode,
		TimedOut: result.WatchdogTimeout, Signaled: result.Termination == commandrun.TerminationSignal,
		Stdout: append([]byte(nil), result.Stdout.Bytes...), Stderr: append([]byte(nil), result.Stderr.Bytes...),
		Truncated: result.Stdout.Truncated || result.Stderr.Truncated,
	}
	if planned.wantTimeout {
		caseResult.Passed = runErr == nil && result.WatchdogTimeout && !caseResult.Truncated
	} else {
		caseResult.Passed = runErr == nil && !result.WatchdogTimeout && !caseResult.Signaled &&
			result.Termination == commandrun.TerminationExit && result.ExitCode == planned.wantExit && !caseResult.Truncated
	}
	if caseResult.Passed && planned.oracle != nil {
		if oracleErr := planned.oracle(result); oracleErr != nil {
			caseResult.Passed = false
			runErr = oracleErr
		}
	}
	campaign.report.Cases = append(campaign.report.Cases, caseResult)
	if runErr != nil {
		return result, fmt.Errorf("run gomadv3 fixture %s: %w", planned.name, runErr)
	}
	if !caseResult.Passed {
		return result, fmt.Errorf("gomadv3 fixture %s failed with status %d: %s%s", planned.name, result.ExitCode, result.Stdout.Bytes, result.Stderr.Bytes)
	}
	return result, nil
}

func (campaign *runtimeCampaign) request(command []string, dir string, timeout time.Duration, unset []string, values ...string) commandrun.Request {
	return commandrun.Request{
		Command: command, Dir: dir, Env: append(filterEnvironment(os.Environ(), unset...), values...), Timeout: timeout,
		TerminateGrace: fixtureTerminationGrace, OutputLimit: fixtureOutputLimit,
	}
}

func (campaign *runtimeCampaign) command(name string, command []string, dir string, timeout time.Duration, unset []string, values ...string) (commandrun.Result, error) {
	return campaign.runCase(runtimeCase{name: name, request: campaign.request(command, dir, timeout, unset, values...)})
}

func (campaign *runtimeCampaign) expectedExit(name string, command []string, dir string, timeout time.Duration, wantExit int, oracle func(commandrun.Result) error, unset []string, values ...string) error {
	_, err := campaign.runCase(runtimeCase{
		name: name, request: campaign.request(command, dir, timeout, unset, values...), wantExit: wantExit, oracle: oracle,
	})
	return err
}

func (campaign *runtimeCampaign) expectedTimeout(name string, command []string, dir string, timeout time.Duration, unset []string, values ...string) error {
	_, err := campaign.runCase(runtimeCase{
		name: name, request: campaign.request(command, dir, timeout, unset, values...), wantTimeout: true,
	})
	return err
}

func (campaign *runtimeCampaign) validateInstallation() error {
	key, err := os.ReadFile(filepath.Join(campaign.config.Root, ".toolchain", "build-key"))
	if err != nil {
		return fmt.Errorf("read gomadv3 build key: %w", err)
	}
	expectedRoot := filepath.Join(campaign.config.Root, ".toolchain", "builds", strings.TrimSpace(string(key)))
	if campaign.goRoot != expectedRoot {
		return fmt.Errorf("gomadv3 stable path is stale: expected %s, got %s", expectedRoot, campaign.goRoot)
	}
	overlayRoot := os.Getenv("GOMADV3_OVERLAY_DIR")
	if overlayRoot == "" {
		overlayRoot = filepath.Join(campaign.config.Root, "overlay")
	}
	return filepath.WalkDir(overlayRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(overlayRoot, path)
		if err != nil {
			return err
		}
		want, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		actual, err := os.ReadFile(filepath.Join(campaign.goRoot, relative))
		if err != nil {
			return fmt.Errorf("read installed overlay %s: %w", relative, err)
		}
		if !slices.Equal(actual, want) {
			return fmt.Errorf("gomadv3 toolchain does not contain overlay source: %s", relative)
		}
		return nil
	})
}

func (campaign *runtimeCampaign) build(name, packageName string, cgo bool, extra ...string) (string, error) {
	output := filepath.Join(campaign.workspace, name)
	command := []string{campaign.config.Go, "build"}
	command = append(command, extra...)
	command = append(command, "-o", output, packageName)
	values := []string{}
	if cgo {
		values = append(values, "CGO_ENABLED=1")
	} else if strings.HasPrefix(name, "clock") {
		values = append(values, "CGO_ENABLED=0")
	}
	_, err := campaign.command(name+"-build", command, campaign.testdata, time.Minute, []string{"GOMADSEED", "CGO_ENABLED"}, values...)
	return output, err
}

func (campaign *runtimeCampaign) runEnabled(seed, packageName, mode string, iteration int) (string, error) {
	result, err := campaign.command(
		fmt.Sprintf("%s-seed-%s-%s-%d", strings.TrimPrefix(packageName, "./"), seed, mode, iteration),
		[]string{campaign.config.Go, "run", "-exec", campaign.execWrapper, packageName}, campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMADV3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMADV3_CHILD_SEED="+seed,
	)
	return commandOutput(result), err
}

func commandOutput(result commandrun.Result) string {
	return strings.TrimRight(string(result.Stdout.RawBytes), "\n")
}

func commandErrorOutput(result commandrun.Result) string {
	return strings.TrimRight(string(result.Stderr.RawBytes), "\n")
}

func validateRandomContract(seed, output string) error {
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != 8 {
		return fmt.Errorf("seed %s runtime random output has %d lines, expected 8", seed, len(lines))
	}
	for _, line := range lines {
		if !randomLinePattern.MatchString(line) {
			return fmt.Errorf("seed %s runtime random output is malformed: %q", seed, line)
		}
	}
	return nil
}

func validateClockRaceOutput(output string, count int) error {
	fields := strings.Fields(strings.Trim(strings.TrimSpace(output), "[]"))
	if len(fields) != count {
		return fmt.Errorf("simultaneous clock timers completed %d times, want %d: %s", len(fields), count, output)
	}
	identifiers := make([]int, 0, len(fields))
	for _, field := range fields {
		identifier, err := strconv.Atoi(field)
		if err != nil {
			return fmt.Errorf("simultaneous clock timer identifier %q is invalid", field)
		}
		identifiers = append(identifiers, identifier)
	}
	sort.Ints(identifiers)
	for index, identifier := range identifiers {
		if identifier != index {
			return fmt.Errorf("simultaneous clock timers did not each complete exactly once: %s", output)
		}
	}
	return nil
}

func benchmarkMedianNS(output string) (float64, error) {
	var samples []float64
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "BenchmarkDisabledClockNow") {
			continue
		}
		for index := 0; index+1 < len(fields); index++ {
			if fields[index+1] != "ns/op" {
				continue
			}
			value, err := strconv.ParseFloat(fields[index], 64)
			if err != nil {
				return 0, fmt.Errorf("parse clock benchmark sample: %w", err)
			}
			samples = append(samples, value)
		}
	}
	if len(samples) != 14 {
		return 0, fmt.Errorf("clock benchmark produced %d samples, want 14", len(samples))
	}
	slices.Sort(samples)
	return (samples[6] + samples[7]) / 2, nil
}

func outputDigest(output string) [sha256.Size]byte {
	return sha256.Sum256([]byte(output))
}

func (campaign *runtimeCampaign) execute() error {
	ambientRoot, err := campaign.command(
		"go-env-ambient-goroot", []string{campaign.config.Go, "env", "GOROOT"}, campaign.config.Root, 10*time.Second,
		[]string{"GOMADSEED", "GOMADV3_CHILD_SEED", "GOROOT"}, "GOROOT="+filepath.Join(campaign.workspace, "foreign-goroot"),
	)
	if err != nil {
		return err
	}
	if actual := commandOutput(ambientRoot); actual != campaign.goRoot {
		return fmt.Errorf("gomadv3 stable path inherited foreign GOROOT: expected %s, got %s", campaign.goRoot, actual)
	}
	disabled, err := campaign.command(
		"activation-disabled", []string{campaign.config.Go, "run", "./activation"}, campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "GOMAXPROCS"}, "GOMAXPROCS=2",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "init GOMAXPROCS=2\nmain GOMAXPROCS=2", "disabled activation"); err != nil {
		return err
	}
	if err := campaign.requireStockCompatibility(); err != nil {
		return err
	}

	binaries := make(map[string]string)
	for _, fixture := range []struct {
		name        string
		packageName string
		cgo         bool
	}{
		{name: "maps", packageName: "./maps"},
		{name: "scheduler", packageName: "./scheduler"},
		{name: "scheduler-min", packageName: "./scheduler_min"},
		{name: "select", packageName: "./select"},
		{name: "channels", packageName: "./channels"},
		{name: "sync", packageName: "./sync"},
		{name: "runqueue", packageName: "./runqueue"},
		{name: "automatic-gc", packageName: "./automatic_gc"},
		{name: "activation", packageName: "./activation"},
		{name: "activation-io", packageName: "./activation_io"},
		{name: "clock", packageName: "./clock"},
		{name: "clock-race", packageName: "./clock_race"},
		{name: "clock-spin", packageName: "./clock_spin"},
		{name: "clock-deadlock", packageName: "./clock_deadlock"},
		{name: "clock-io", packageName: "./clock_io"},
	} {
		binary, err := campaign.build(fixture.name, fixture.packageName, fixture.cgo)
		if err != nil {
			return err
		}
		binaries[fixture.name] = binary
	}
	if err := campaign.requireClockBehavior(binaries); err != nil {
		return err
	}
	if err := campaign.requireLinkModes(); err != nil {
		return err
	}
	if err := campaign.requireSchedulingBehavior(binaries); err != nil {
		return err
	}
	return campaign.requireRepeatability(binaries)
}

func requireOutput(result commandrun.Result, want, label string) error {
	if actual := commandOutput(result); actual != want {
		return fmt.Errorf("%s output = %q, want %q", label, actual, want)
	}
	return nil
}

func (campaign *runtimeCampaign) requireStockCompatibility() error {
	wantVersion := gomadversion.GoVersion
	stockGo := os.Getenv("GOMADV3_STOCK_GO")
	if stockGo == "" {
		launcher, err := exec.LookPath("go")
		if err == nil {
			root, commandErr := campaign.command(
				"stock-go-resolve", []string{launcher, "env", "GOROOT"}, campaign.testdata, 10*time.Second,
				[]string{"GOMADSEED", "GONOPROXY", "GONOSUMDB", "GOPRIVATE", "GOPROXY", "GOSUMDB", "GOTOOLCHAIN"}, "GOTOOLCHAIN="+wantVersion,
			)
			if commandErr == nil {
				stockGo = filepath.Join(commandOutput(root), "bin", "go")
			}
		}
	}
	info, err := os.Stat(stockGo)
	if err != nil || !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return errors.Join(fmt.Errorf("stock Go is missing; set GOMADV3_STOCK_GO to a %s executable", wantVersion), err)
	}
	versionResult, err := campaign.command(
		"stock-go-version", []string{stockGo, "version"}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOTOOLCHAIN"}, "GOTOOLCHAIN=local",
	)
	if err != nil {
		return err
	}
	if actual := commandOutput(versionResult); !strings.HasPrefix(actual, "go version "+wantVersion+" ") {
		return fmt.Errorf("stock Go must report %s; %s reported: %s", wantVersion, stockGo, actual)
	}
	rootResult, err := campaign.command(
		"stock-go-goroot", []string{stockGo, "env", "GOROOT"}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOTOOLCHAIN"}, "GOTOOLCHAIN=local",
	)
	if err != nil {
		return err
	}
	stockRoot := commandOutput(rootResult)
	if info, err := os.Stat(stockRoot); err != nil || !info.IsDir() {
		return errors.Join(fmt.Errorf("stock Go reported an invalid GOROOT: %s", stockRoot), err)
	}
	canonicalStock, err := filepath.EvalSymlinks(stockRoot)
	if err != nil {
		return err
	}
	canonicalCustom, err := filepath.EvalSymlinks(campaign.goRoot)
	if err != nil {
		return err
	}
	if canonicalStock == canonicalCustom {
		return fmt.Errorf("stock Go resolves to the gomadv3 custom GOROOT: %s", canonicalStock)
	}
	customRun, err := campaign.command(
		"activation-custom-compatibility", []string{campaign.config.Go, "run", "./activation"}, campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
		"GODEBUG=", "GOMAXPROCS=2", "GOTOOLCHAIN=local", "GOWORK=off",
	)
	if err != nil {
		return err
	}
	stockRun, err := campaign.command(
		"activation-stock-compatibility", []string{stockGo, "run", "./activation"}, campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
		"GODEBUG=", "GOMAXPROCS=2", "GOTOOLCHAIN=local", "GOWORK=off",
	)
	if err != nil {
		return err
	}
	if commandOutput(customRun) != commandOutput(stockRun) {
		return fmt.Errorf("disabled go run output differs from stock %s", wantVersion)
	}
	customTest, err := campaign.compatibilityGoTest(campaign.config.Go, "custom")
	if err != nil {
		return err
	}
	stockTest, err := campaign.compatibilityGoTest(stockGo, "stock")
	if err != nil {
		return err
	}
	customLine, customCount := prefixedLine(customTest, "GOMADV3_COMPAT ")
	stockLine, stockCount := prefixedLine(stockTest, "GOMADV3_COMPAT ")
	if customCount != 1 || stockCount != 1 {
		return fmt.Errorf("disabled go test compatibility output must appear exactly once: custom=%q stock=%q", customLine, stockLine)
	}
	if customLine != stockLine {
		return fmt.Errorf("disabled go test output differs from stock %s", wantVersion)
	}
	customBenchmark, err := campaign.disabledClockBenchmark(campaign.config.Go, "custom")
	if err != nil {
		return err
	}
	stockBenchmark, err := campaign.disabledClockBenchmark(stockGo, "stock")
	if err != nil {
		return err
	}
	customNS, err := benchmarkMedianNS(customBenchmark)
	if err != nil {
		return err
	}
	stockNS, err := benchmarkMedianNS(stockBenchmark)
	if err != nil {
		return err
	}
	limit := max(stockNS*2, stockNS+10)
	if customNS > limit {
		return fmt.Errorf("disabled clock read regression: custom median %v ns/op, stock median %v ns/op", customNS, stockNS)
	}
	return nil
}

func (campaign *runtimeCampaign) compatibilityGoTest(goCommand, implementation string) (string, error) {
	result, err := campaign.command(
		"gotest-"+implementation+"-compatibility",
		[]string{goCommand, "test", "-count=1", "-tags=test_dep", "-run", "^TestDisabledCompatibility$", "-v", "./gotest"},
		campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
		"GODEBUG=", "GOMAXPROCS=2", "GOTOOLCHAIN=local", "GOWORK=off",
	)
	return commandOutput(result), err
}

func prefixedLine(output, prefix string) (string, int) {
	var found string
	count := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			found = line
			count++
		}
	}
	return found, count
}

func (campaign *runtimeCampaign) disabledClockBenchmark(goCommand, implementation string) (string, error) {
	var combined strings.Builder
	for iteration := range 2 {
		result, err := campaign.command(
			fmt.Sprintf("clock-bench-%s-%d", implementation, iteration),
			[]string{goCommand, "test", "-run", "^$", "-bench", "^BenchmarkDisabledClockNow$", "-benchtime=250ms", "-count=7", "-cpu=1", "./clock_bench"},
			campaign.testdata, time.Minute,
			[]string{"GOMADSEED", "GODEBUG", "GOMAXPROCS", "GOTOOLCHAIN", "GOWORK"},
			"GODEBUG=", "GOMAXPROCS=1", "GOTOOLCHAIN=local", "GOWORK=off",
		)
		if err != nil {
			return "", err
		}
		combined.Write(result.Stdout.RawBytes)
	}
	return combined.String(), nil
}

func (campaign *runtimeCampaign) runClockDirect(seed, binary, mode string, iteration int) (string, error) {
	result, err := campaign.command(
		fmt.Sprintf("clock-%s-seed-%s-%d", mode, seed, iteration), []string{binary, mode}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "TZ"}, "GOMADSEED="+seed, "TZ=UTC",
	)
	return commandOutput(result), err
}

func (campaign *runtimeCampaign) requireClockBehavior(binaries map[string]string) error {
	disabled, err := campaign.command(
		"clock-disabled", []string{binaries["clock"], "disabled"}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "TZ"}, "TZ=UTC",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "clock disabled ok", "disabled clock"); err != nil {
		return err
	}
	for _, mode := range []string{"initial", "sleep", "runnable", "timers", "contexts", "edges"} {
		output, err := campaign.runClockDirect("1", binaries["clock"], mode, 0)
		if err != nil {
			return err
		}
		if output != "clock "+mode+" ok" {
			return fmt.Errorf("clock mode %s emitted unexpected output: %q", mode, output)
		}
	}
	for _, mode := range []string{"new", "active-reset", "stopped-reset", "contexts", "tickers"} {
		count := 24
		if mode == "tickers" {
			count = 48
		}
		expected, err := campaign.runClockDirect("1", binaries["clock-race"], mode, 0)
		if err != nil {
			return err
		}
		if err := validateClockRaceOutput(expected, count); err != nil {
			return err
		}
		for iteration := 1; iteration < 20; iteration++ {
			actual, err := campaign.runClockDirect("1", binaries["clock-race"], mode, iteration)
			if err != nil {
				return err
			}
			if actual != expected {
				return fmt.Errorf("same-seed %s timer order diverged on run %d", mode, iteration)
			}
		}
		diversity := make(map[[sha256.Size]byte]struct{})
		for seed := range 16 {
			output, err := campaign.runClockDirect(strconv.Itoa(seed), binaries["clock-race"], mode, seed)
			if err != nil {
				return err
			}
			if err := validateClockRaceOutput(output, count); err != nil {
				return err
			}
			diversity[outputDigest(output)] = struct{}{}
		}
		if len(diversity) <= 1 {
			return fmt.Errorf("different seeds produced no %s timer-order diversity", mode)
		}
	}
	for _, mode := range []string{"loop", "select"} {
		if err := campaign.expectedTimeout(
			"clock-spin-"+mode, []string{binaries["clock-spin"], mode}, campaign.testdata, 2*time.Second,
			[]string{"GOMADSEED", "TZ"}, "GOMADSEED=1", "TZ=UTC",
		); err != nil {
			return err
		}
	}
	if err := campaign.expectedTimeout(
		"clock-blocking-io", []string{binaries["clock-io"]}, campaign.testdata, 2*time.Second,
		[]string{"GOMADSEED", "TZ"}, "GOMADSEED=1", "TZ=UTC",
	); err != nil {
		return err
	}
	if err := campaign.expectedExit(
		"clock-deadlock", []string{binaries["clock-deadlock"]}, campaign.testdata, 2*time.Second, 2,
		func(result commandrun.Result) error {
			if !strings.Contains(string(result.Stderr.RawBytes), "fatal error: all goroutines are asleep - deadlock!") {
				return errors.New("clock deadlock emitted an unexpected diagnostic")
			}
			return nil
		},
		[]string{"GOMADSEED", "TZ"}, "GOMADSEED=1", "TZ=UTC",
	); err != nil {
		return err
	}
	for _, seed := range []string{"0", "18446744073709551615"} {
		output, err := campaign.runClockDirect(seed, binaries["clock"], "initial", 0)
		if err != nil {
			return err
		}
		if output != "clock initial ok" {
			return fmt.Errorf("boundary clock seed %s output = %q", seed, output)
		}
	}
	clockRun, err := campaign.command(
		"clock-go-run", []string{campaign.config.Go, "run", "-exec", campaign.execWrapper, "./clock", "initial"},
		campaign.testdata, time.Minute, []string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMADV3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMADV3_CHILD_SEED=1",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(clockRun, "clock initial ok", "clock go run"); err != nil {
		return err
	}
	clockGoTest, err := campaign.command(
		"clock-go-test", []string{campaign.config.Go, "test", "-v", "-exec", campaign.execWrapper, "-count=1", "-tags=test_dep", "-timeout=48h", "./clock_gotest"},
		campaign.testdata, time.Minute, []string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMADV3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMADV3_CHILD_SEED=1",
	)
	if err != nil {
		return err
	}
	goTestOutput := commandOutput(clockGoTest)
	if !strings.Contains(goTestOutput, "--- PASS: TestVirtualClockAndDeadline (86400.00s)") ||
		!strings.Contains(goTestOutput, "--- PASS: TestSecondVirtualClockTest (21600.00s)") {
		return errors.New("go test did not report repeatable logical durations for both tests")
	}
	if _, err := campaign.command(
		"clock-synctest", []string{campaign.config.Go, "test", "-exec", campaign.execWrapper, "-count=1", "-tags=test_dep", "-timeout=48h", "./clock_synctest"},
		campaign.testdata, time.Minute, []string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMADV3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMADV3_CHILD_SEED=1",
	); err != nil {
		return err
	}
	return campaign.expectedExit(
		"clock-logical-timeout",
		[]string{campaign.config.Go, "test", "-exec", campaign.execWrapper, "-count=1", "-tags=test_dep", "-timeout=1h", "-run", "^TestLogicalTimeout$", "./clock_gotest"},
		campaign.testdata, 5*time.Second, 1,
		func(result commandrun.Result) error {
			if result.WatchdogTimeout || !strings.Contains(string(result.Stdout.RawBytes)+string(result.Stderr.RawBytes), "panic: test timed out after 1h0m0s") {
				return errors.New("logical go test timeout was not distinct from the wall watchdog")
			}
			return nil
		},
		[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMADV3_CHILD_SEED", "GOMADV3_TEST_LOGICAL_TIMEOUT"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMADV3_CHILD_SEED=1", "GOMADV3_TEST_LOGICAL_TIMEOUT=1",
	)
}

func (campaign *runtimeCampaign) requireLinkModes() error {
	ccResult, err := campaign.command(
		"go-env-cc", []string{campaign.config.Go, "env", "CC"}, campaign.testdata, 10*time.Second, []string{"GOMADSEED"},
	)
	if err != nil {
		return err
	}
	ccFields := strings.Fields(commandOutput(ccResult))
	if len(ccFields) == 0 || !executableAvailable(ccFields[0]) {
		return nil
	}
	clockCGO, err := campaign.build("clock-cgo", "./clock_cgo", true)
	if err != nil {
		return err
	}
	disabled, err := campaign.command("clock-cgo-disabled", []string{clockCGO}, campaign.testdata, 5*time.Second, []string{"GOMADSEED"})
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "42", "disabled cgo clock"); err != nil {
		return err
	}
	if err := campaign.requireUnsupportedLink("clock-cgo-enabled", clockCGO, nil); err != nil {
		return err
	}
	clockExternal, err := campaign.build("clock-external", "./clock", true, "-ldflags=-linkmode=external")
	if err != nil {
		return err
	}
	disabled, err = campaign.command(
		"clock-external-disabled", []string{clockExternal, "disabled"}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "TZ"}, "TZ=UTC",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "clock disabled ok", "disabled externally linked clock"); err != nil {
		return err
	}
	if err := campaign.requireUnsupportedLink("clock-external-enabled", clockExternal, []string{"initial"}); err != nil {
		return err
	}
	goos, err := campaign.command("go-env-goos", []string{campaign.config.Go, "env", "GOOS"}, campaign.testdata, 10*time.Second, []string{"GOMADSEED"})
	if err != nil {
		return err
	}
	if commandOutput(goos) != "darwin" {
		return nil
	}
	clockAMD64 := filepath.Join(campaign.workspace, "clock-external-amd64")
	_, err = campaign.command(
		"clock-external-amd64-build",
		[]string{campaign.config.Go, "build", "-ldflags=-linkmode=external", "-o", clockAMD64, "./clock"},
		campaign.testdata, time.Minute, []string{"GOMADSEED", "GOARCH", "CGO_ENABLED"}, "GOARCH=amd64", "CGO_ENABLED=1",
	)
	if err != nil {
		return err
	}
	disabledAMD64, err := campaign.command(
		"clock-external-amd64-disabled", []string{clockAMD64, "disabled"}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "TZ"}, "TZ=UTC",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabledAMD64, "clock disabled ok", "disabled Darwin/amd64 externally linked clock"); err != nil {
		return err
	}
	return campaign.requireUnsupportedLink("clock-external-amd64-enabled", clockAMD64, []string{"initial"})
}

func (campaign *runtimeCampaign) requireUnsupportedLink(name, binary string, arguments []string) error {
	return campaign.expectedExit(
		name, append([]string{binary}, arguments...), campaign.testdata, 5*time.Second, 2,
		func(result commandrun.Result) error {
			if commandErrorOutput(result) != "runtime: GOMADSEED does not support cgo or external linking" {
				return fmt.Errorf("%s emitted an unexpected diagnostic: %q", name, string(result.Stderr.RawBytes))
			}
			return nil
		},
		[]string{"GOMADSEED", "TZ"}, "GOMADSEED=1", "TZ=UTC",
	)
}

func (campaign *runtimeCampaign) requireSchedulingBehavior(binaries map[string]string) error {
	for _, padding := range []string{"", "invalid", "4194305"} {
		if err := campaign.expectedExit(
			"layout-invalid-padding-"+padding, []string{binaries["scheduler"]}, campaign.testdata, 10*time.Second, 2,
			func(result commandrun.Result) error {
				if len(result.Stdout.RawBytes) != 0 || !strings.Contains(string(result.Stderr.RawBytes), "GOMADV3_ADDRESS_PADDING must be a decimal byte count up to 4194304") {
					return fmt.Errorf("invalid address padding %q emitted an unexpected diagnostic", padding)
				}
				return nil
			},
			[]string{"GODEBUG", "GOGC", "GOMADSEED", "GOMADV3_ADDRESS_PADDING", "GOMAXPROCS"},
			"GODEBUG=asyncpreemptoff=1", "GOGC=off", "GOMADSEED=1", "GOMADV3_ADDRESS_PADDING="+padding, "GOMAXPROCS=1",
		); err != nil {
			return err
		}
	}
	for _, packageName := range []string{"./scheduler", "./select", "./maps", "./channels", "./sync", "./runqueue"} {
		if err := campaign.requireAddressPerturbation(packageName); err != nil {
			return err
		}
	}
	if err := campaign.requireHostLoad(binaries); err != nil {
		return err
	}
	if err := campaign.requireMapFamilies(); err != nil {
		return err
	}
	for packageName, marker := range map[string]string{
		"./select": "select-oracle:ok", "./channels": "channels-oracle:ok", "./maps": "maps-oracle:ok", "./sync": "sync-oracle:ok",
	} {
		output, err := campaign.runEnabled("1", packageName, "semantic-oracle", 0)
		if err != nil {
			return err
		}
		if strings.Count(output, marker) != 1 {
			return fmt.Errorf("%s did not report its semantic oracle", packageName)
		}
	}
	for _, seed := range []string{"0", "1", "18446744073709551615"} {
		output, err := campaign.runEnabled(seed, "./activation", "enabled", 0)
		if err != nil {
			return err
		}
		if output != "init GOMAXPROCS=1\nmain GOMAXPROCS=1" {
			return fmt.Errorf("enabled activation seed %s output = %q", seed, output)
		}
	}
	disabled, err := campaign.command(
		"activation-explicit-disabled", []string{binaries["activation"]}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOMADV3_IO_PROFILE", "GODEBUG", "GOMAXPROCS"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "init GOMAXPROCS=1\nmain GOMAXPROCS=1", "explicitly disabled activation"); err != nil {
		return errors.New("supporting runtime settings activated Gomad without GOMADSEED")
	}
	var disabledScheduler string
	for iteration := 1; iteration <= 32; iteration++ {
		result, err := campaign.command(
			fmt.Sprintf("scheduler-min-explicit-disabled-%d", iteration), []string{binaries["scheduler-min"]}, campaign.testdata, 10*time.Second,
			[]string{"GOMADSEED", "GOMADV3_IO_PROFILE", "GODEBUG", "GOMAXPROCS"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1",
		)
		if err != nil {
			return err
		}
		output := commandOutput(result)
		if iteration == 1 {
			disabledScheduler = output
		} else if output != disabledScheduler {
			return errors.New("supporting runtime settings activated scheduler randomization without GOMADSEED")
		}
	}
	disabledIO, err := campaign.command(
		"activation-io-disabled", []string{binaries["activation-io"]}, campaign.testdata, 10*time.Second,
		[]string{"GOMADSEED", "GOMADV3_IO_PROFILE"},
	)
	if err != nil {
		return err
	}
	seededIO, err := campaign.command(
		"activation-io-direct", []string{binaries["activation-io"]}, campaign.testdata, 10*time.Second,
		[]string{"GOMADV3_IO_PROFILE", "GOMADSEED"}, "GOMADSEED=1",
	)
	if err != nil {
		return err
	}
	if commandOutput(seededIO) != "gomad-host" || commandOutput(seededIO) == commandOutput(disabledIO) {
		return errors.New("direct GOMADSEED activation did not select the deterministic boundary explicitly")
	}
	if err := campaign.expectedExit(
		"activation-missing-profile-configuration", []string{binaries["activation"]}, campaign.testdata, 10*time.Second, 2,
		func(result commandrun.Result) error {
			if len(result.Stdout.RawBytes) != 0 || commandErrorOutput(result) != "runtime: missing Gomad bootstrap configuration" {
				return errors.New("profile activation without Runner configuration emitted an unexpected diagnostic")
			}
			return nil
		},
		[]string{"GOMADSEED", "GOMADV3_IO_PROFILE"}, "GOMADV3_IO_PROFILE=deterministic",
	); err != nil {
		return err
	}
	for _, invalid := range []struct{ seed, name string }{
		{seed: "", name: "empty"}, {seed: "+1", name: "signed-plus"}, {seed: "-1", name: "signed-minus"},
		{seed: " 1", name: "leading-whitespace"}, {seed: "1 ", name: "trailing-whitespace"},
		{seed: "invalid", name: "nondecimal"}, {seed: "0x1", name: "hexadecimal"},
		{seed: "18446744073709551616", name: "overflow"},
	} {
		seed := invalid.seed
		if err := campaign.expectedExit(
			"activation-invalid-seed-"+invalid.name, []string{binaries["activation"]}, campaign.testdata, 10*time.Second, 2,
			func(result commandrun.Result) error {
				if len(result.Stdout.RawBytes) != 0 || commandErrorOutput(result) != "runtime: invalid GOMADSEED" {
					return fmt.Errorf("invalid GOMADSEED=%q reached user initialization or emitted an unexpected diagnostic", seed)
				}
				return nil
			},
			[]string{"GOMADSEED"}, "GOMADSEED="+seed,
		); err != nil {
			return err
		}
	}
	return nil
}

func (campaign *runtimeCampaign) requireAddressPerturbation(packageName string) error {
	var expectedPayload string
	markers := make(map[string]struct{})
	for _, padding := range []string{"0", "1048576", "4194304"} {
		result, err := campaign.command(
			fmt.Sprintf("%s-address-padding-%s", strings.TrimPrefix(packageName, "./"), padding),
			[]string{campaign.config.Go, "run", "-exec", campaign.execWrapper, packageName}, campaign.testdata, time.Minute,
			[]string{"GOMADSEED", "CGO_ENABLED", "GOGC", "TZ", "GOMADV3_ADDRESS_PADDING", "GOMADV3_CHILD_SEED"},
			"CGO_ENABLED=0", "GOGC=off", "TZ=UTC", "GOMADV3_ADDRESS_PADDING="+padding, "GOMADV3_CHILD_SEED=1",
		)
		if err != nil {
			return err
		}
		lines := strings.Split(commandOutput(result), "\n")
		var marker string
		var payload []string
		for _, line := range lines {
			if strings.HasPrefix(line, "GOMADV3_ADDRESS ") {
				if marker != "" {
					return fmt.Errorf("%s address perturbation emitted no unique marker for padding %s", packageName, padding)
				}
				marker = strings.TrimPrefix(line, "GOMADV3_ADDRESS ")
				continue
			}
			payload = append(payload, line)
		}
		if marker == "" || lines[len(lines)-1] != "GOMADV3_ADDRESS "+marker {
			return fmt.Errorf("%s address perturbation emitted its marker before completing output for padding %s", packageName, padding)
		}
		if _, err := strconv.ParseUint(strings.TrimPrefix(marker, "0x"), 16, 64); err != nil || !strings.HasPrefix(marker, "0x") {
			return fmt.Errorf("%s address perturbation emitted invalid marker for padding %s: %s", packageName, padding, marker)
		}
		markers[marker] = struct{}{}
		joined := strings.Join(payload, "\n")
		if expectedPayload == "" {
			expectedPayload = joined
		} else if joined != expectedPayload {
			return fmt.Errorf("%s output changed under address perturbation for padding %s", packageName, padding)
		}
	}
	if len(markers) <= 1 {
		return fmt.Errorf("%s address perturbation did not produce distinct layouts", packageName)
	}
	return nil
}

func (campaign *runtimeCampaign) requireHostLoad(binaries map[string]string) (returnedErr error) {
	fixtures := []struct{ label, binary string }{
		{label: "scheduler", binary: binaries["scheduler"]}, {label: "maps", binary: binaries["maps"]},
		{label: "select", binary: binaries["select"]}, {label: "channels", binary: binaries["channels"]},
		{label: "sync", binary: binaries["sync"]}, {label: "runqueue", binary: binaries["runqueue"]},
		{label: "automatic-gc", binary: binaries["automatic-gc"]},
	}
	expected := make([]string, len(fixtures))
	for index, fixture := range fixtures {
		result, err := campaign.hostFixture(fixture.label+"-baseline", fixture.binary, 0)
		if err != nil {
			return err
		}
		expected[index] = commandOutput(result)
	}
	stop, err := startCPULoadWorkers(2)
	if err != nil {
		return err
	}
	defer func() { returnedErr = errors.Join(returnedErr, stop()) }()
	for iteration := 1; iteration <= 8; iteration++ {
		for index, fixture := range fixtures {
			result, err := campaign.hostFixture(fixture.label+"-host-load", fixture.binary, iteration)
			if err != nil {
				return err
			}
			if commandOutput(result) != expected[index] {
				return fmt.Errorf("%s output changed under unrelated host CPU load on run %d", fixture.label, iteration)
			}
		}
	}
	return stop()
}

func (campaign *runtimeCampaign) hostFixture(label, binary string, iteration int) (commandrun.Result, error) {
	return campaign.command(
		fmt.Sprintf("%s-%d", label, iteration), []string{binary}, campaign.testdata, 10*time.Second,
		[]string{"GODEBUG", "GOMAXPROCS", "GOMADSEED"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1", "GOMADSEED=1",
	)
}

var mapFamilyLabels = []string{
	"create", "string", "clone", "delete-reinsert", "clear", "uint8", "uint16", "uint32", "uint64",
	"float32", "float64", "complex64", "complex128", "empty-interface", "non-empty-interface", "array", "struct",
	"growth", "small", "nan",
}

func (campaign *runtimeCampaign) requireMapFamilies() error {
	diversity := make(map[string]map[string]struct{}, len(mapFamilyLabels))
	for _, family := range mapFamilyLabels {
		diversity[family] = make(map[string]struct{})
	}
	for seed := range 32 {
		output, err := campaign.runEnabled(strconv.Itoa(seed), "./maps", "map-families", seed)
		if err != nil {
			return err
		}
		for _, family := range mapFamilyLabels {
			prefix := family + ":"
			line, count := prefixedLine(output, prefix)
			if count == 0 {
				return fmt.Errorf("map hashing audit is missing family %s", family)
			}
			if count != 1 {
				return fmt.Errorf("map hashing audit emitted family %s more than once", family)
			}
			diversity[family][line] = struct{}{}
		}
	}
	for _, family := range mapFamilyLabels {
		if len(diversity[family]) <= 1 {
			return fmt.Errorf("different seeds produced no map diversity for family %s", family)
		}
	}
	return nil
}

func (campaign *runtimeCampaign) requireRepeatability(binaries map[string]string) error {
	seeds := []string{"0", "1", "18446744073709551615"}
	// Runtime randomness also drives scheduler and map choices, so package
	// initialization can consume a per-M stream before main runs.
	randomOutputs := make(map[string]string, len(seeds))
	for _, seed := range seeds {
		output, err := campaign.runEnabled(seed, "./random", "golden-random", 0)
		if err != nil {
			return err
		}
		if err := validateRandomContract(seed, output); err != nil {
			return err
		}
		randomOutputs[seed] = output
	}
	if randomOutputs[seeds[0]] == randomOutputs[seeds[1]] || randomOutputs[seeds[0]] == randomOutputs[seeds[2]] || randomOutputs[seeds[1]] == randomOutputs[seeds[2]] {
		return errors.New("boundary seeds alias another pinned runtime random sequence")
	}
	for _, seed := range seeds {
		for _, packageName := range []string{"./random", "./scheduler_min", "./scheduler", "./select", "./maps", "./channels", "./sync", "./runqueue", "./automatic_gc"} {
			if err := campaign.requireRepeatable(packageName, seed, 100); err != nil {
				return err
			}
		}
	}
	for _, packageName := range []string{"./scheduler", "./select", "./channels", "./sync", "./runqueue", "./automatic_gc"} {
		if err := campaign.requireDiverse(packageName); err != nil {
			return err
		}
	}
	for _, seed := range seeds {
		result, err := campaign.command(
			"scheduler-direct-seed-"+seed+"-0", []string{binaries["scheduler"]}, campaign.testdata, 10*time.Second,
			[]string{"GODEBUG", "GOMAXPROCS", "GOMADSEED"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1", "GOMADSEED="+seed,
		)
		if err != nil {
			return err
		}
		expected := commandOutput(result)
		for iteration := 1; iteration < 100; iteration++ {
			result, err := campaign.command(
				fmt.Sprintf("scheduler-direct-seed-%s-%d", seed, iteration), []string{binaries["scheduler"]}, campaign.testdata, 10*time.Second,
				[]string{"GODEBUG", "GOMAXPROCS", "GOMADSEED"}, "GODEBUG=asyncpreemptoff=1", "GOMAXPROCS=1", "GOMADSEED="+seed,
			)
			if err != nil {
				return err
			}
			if commandOutput(result) != expected {
				return fmt.Errorf("same-seed direct scheduler output diverged for seed %s on run %d", seed, iteration)
			}
		}
	}
	if err := campaign.requireParallelRepeatability(); err != nil {
		return err
	}
	cache := filepath.Join(campaign.workspace, "go-cache")
	var cached string
	for _, mode := range []string{"cold-cache", "warm-cache"} {
		result, err := campaign.command(
			"scheduler-"+mode, []string{campaign.config.Go, "run", "-exec", campaign.execWrapper, "./scheduler"}, campaign.testdata, time.Minute,
			[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOCACHE", "GOMADV3_CHILD_SEED"},
			"CGO_ENABLED=0", "TZ=UTC", "GOCACHE="+cache, "GOMADV3_CHILD_SEED=1",
		)
		if err != nil {
			return err
		}
		if cached == "" {
			cached = commandOutput(result)
		} else if commandOutput(result) != cached {
			return errors.New("cold and warm cache scheduler output differs")
		}
	}
	disabledMaps := make(map[[sha256.Size]byte]struct{})
	for iteration := 1; iteration <= 16; iteration++ {
		result, err := campaign.command(
			fmt.Sprintf("maps-disabled-%d", iteration), []string{campaign.config.Go, "run", "./maps"}, campaign.testdata, time.Minute,
			[]string{"GOMADSEED"},
		)
		if err != nil {
			return err
		}
		disabledMaps[sha256.Sum256(result.Stdout.RawBytes)] = struct{}{}
	}
	if len(disabledMaps) <= 1 {
		return errors.New("disabled gomadv3 unexpectedly fixed map iteration order")
	}
	preemption, err := campaign.build("preemption", "./preemption", false)
	if err != nil {
		return err
	}
	disabledPreemption, err := campaign.command(
		"preemption-disabled", []string{preemption}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "GOMAXPROCS"}, "GOMAXPROCS=1",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabledPreemption, "async preemption enabled", "disabled preemption"); err != nil {
		return err
	}
	if err := campaign.expectedTimeout(
		"preemption-enabled", []string{preemption}, campaign.testdata, 2*time.Second,
		[]string{"GODEBUG", "GOMAXPROCS", "GOMADSEED"}, "GODEBUG=", "GOMAXPROCS=1", "GOMADSEED=1",
	); err != nil {
		return err
	}
	for _, seed := range seeds {
		expected, err := campaign.runSeededGoTest(seed, 0)
		if err != nil {
			return err
		}
		for iteration := 1; iteration < 100; iteration++ {
			actual, err := campaign.runSeededGoTest(seed, iteration)
			if err != nil {
				return err
			}
			if actual != expected {
				return fmt.Errorf("same-seed go test output diverged for seed %s on run %d", seed, iteration)
			}
		}
	}
	return nil
}

func (campaign *runtimeCampaign) requireRepeatable(packageName, seed string, runs int) error {
	expected, err := campaign.runEnabled(seed, packageName, "repeatable", 0)
	if err != nil {
		return err
	}
	for iteration := 1; iteration < runs; iteration++ {
		actual, err := campaign.runEnabled(seed, packageName, "repeatable", iteration)
		if err != nil {
			return err
		}
		if actual != expected {
			return campaign.repeatabilityMismatch(
				fmt.Sprintf("same-seed output diverged for %s seed %s on run %d", packageName, seed, iteration),
				expected,
				actual,
			)
		}
	}
	return nil
}

func (campaign *runtimeCampaign) repeatabilityMismatch(label, expected, actual string) error {
	if campaign.report != nil && len(campaign.report.Cases) != 0 {
		campaign.report.Cases[len(campaign.report.Cases)-1].Passed = false
	}
	return fmt.Errorf(
		"%s: expected sha256:%x actual sha256:%x expected-output=%q actual-output=%q",
		label,
		outputDigest(expected),
		outputDigest(actual),
		repeatabilityOutputExcerpt(expected),
		repeatabilityOutputExcerpt(actual),
	)
}

func repeatabilityOutputExcerpt(output string) string {
	const maximumBytes = 2048
	if len(output) <= maximumBytes {
		return output
	}
	return output[:maximumBytes] + "..."
}

func (campaign *runtimeCampaign) requireDiverse(packageName string) error {
	outputs := make(map[[sha256.Size]byte]struct{})
	for seed := range 32 {
		output, err := campaign.runEnabled(strconv.Itoa(seed), packageName, "diverse", seed)
		if err != nil {
			return err
		}
		outputs[outputDigest(output)] = struct{}{}
	}
	if len(outputs) <= 1 {
		return fmt.Errorf("different seeds produced no diversity for %s", packageName)
	}
	return nil
}

func (campaign *runtimeCampaign) requireParallelRepeatability() error {
	type parallelRun struct {
		planned runtimeCase
		seed    string
		label   string
		result  commandrun.Result
		err     error
	}
	var runs []*parallelRun
	for _, seed := range []string{"3", "7", "11", "19"} {
		for _, label := range []string{"first", "second"} {
			runs = append(runs, &parallelRun{
				seed: seed, label: label,
				planned: runtimeCase{
					name: "scheduler-parallel-seed-" + seed + "-" + label,
					request: campaign.request(
						[]string{campaign.config.Go, "run", "-exec", campaign.execWrapper, "./scheduler"}, campaign.testdata, time.Minute,
						[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMADV3_CHILD_SEED"},
						"CGO_ENABLED=0", "TZ=UTC", "GOMADV3_CHILD_SEED="+seed,
					),
				},
			})
		}
	}
	var group sync.WaitGroup
	group.Add(len(runs))
	for _, planned := range runs {
		go func() {
			defer group.Done()
			planned.result, planned.err = campaign.run(campaign.ctx, planned.planned.request)
		}()
	}
	group.Wait()
	outputs := make(map[string]map[string]string)
	for _, planned := range runs {
		result, err := campaign.finishCase(planned.planned, planned.result, planned.err)
		if err != nil {
			return err
		}
		if outputs[planned.seed] == nil {
			outputs[planned.seed] = make(map[string]string)
		}
		outputs[planned.seed][planned.label] = commandOutput(result)
	}
	for seed, output := range outputs {
		if output["first"] != output["second"] {
			return fmt.Errorf("parallel same-seed scheduler output diverged for seed %s", seed)
		}
	}
	return nil
}

func (campaign *runtimeCampaign) runSeededGoTest(seed string, iteration int) (string, error) {
	// The go driver reports host wall time after the deterministic test binary exits.
	result, err := campaign.command(
		fmt.Sprintf("gotest-seed-%s-repeatable-%d", seed, iteration),
		[]string{campaign.config.Go, "test", "-exec", campaign.execWrapper, "-count=1", "-tags=test_dep", "-v", "./gotest"},
		campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMADV3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMADV3_CHILD_SEED="+seed,
	)
	if err != nil {
		return "", err
	}
	return goTestDurationPattern.ReplaceAllString(commandOutput(result), "$1"), nil
}

func startCPULoadWorkers(count int) (func() error, error) {
	stop := make(chan struct{})
	started := make(chan struct{}, count)
	var workers sync.WaitGroup
	workers.Add(count)
	for range count {
		go func() {
			runtime.LockOSThread()
			defer runtime.UnlockOSThread()
			defer workers.Done()
			started <- struct{}{}
			for {
				select {
				case <-stop:
					return
				default:
				}
			}
		}()
	}
	for range count {
		select {
		case <-started:
		case <-time.After(time.Second):
			close(stop)
			workers.Wait()
			return nil, errors.New("gomadv3 host-load worker failed to start")
		}
	}
	var once sync.Once
	return func() error {
		once.Do(func() {
			close(stop)
			workers.Wait()
		})
		return nil
	}, nil
}

func executableAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
