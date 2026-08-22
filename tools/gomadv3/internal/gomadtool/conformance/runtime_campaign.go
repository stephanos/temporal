package conformance

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
)

var randomLinePattern = regexp.MustCompile(`^[0-9a-f]{16} [0-9a-f]{8}$`)
var goTestDurationPattern = regexp.MustCompile(`(?m)^(ok[[:space:]]+[^[:space:]]+)[[:space:]]+[0-9.]+s$`)

type runtimeCase struct {
	name        string
	request     hostexec.Request
	wantExit    int
	wantTimeout bool
	oracle      func(hostexec.Result) error
}

type runtimeCampaign struct {
	ctx         context.Context
	config      Config
	goRoot      string
	testdata    string
	execWrapper string
	workspace   string
	run         func(context.Context, hostexec.Request) (hostexec.Result, error)
	report      *Report
}

func runRuntimeCampaign(
	ctx context.Context,
	config Config,
	goRoot string,
	run func(context.Context, hostexec.Request) (hostexec.Result, error),
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
		ctx: ctx, config: config, goRoot: goRoot, testdata: filepath.Join(config.Root, "internal", "gomadtool", "conformance", "testdata"),
		execWrapper: filepath.Join(config.Root, "internal", "gomadtool", "conformance", "scripts", "exec.sh"), workspace: workspace, run: run, report: report,
	}
	if err := campaign.validateInstallation(); err != nil {
		return err
	}
	return campaign.execute()
}

func (campaign *runtimeCampaign) runCase(planned runtimeCase) (hostexec.Result, error) {
	result, runErr := campaign.run(campaign.ctx, planned.request)
	return campaign.finishCase(planned, result, runErr)
}

func (campaign *runtimeCampaign) finishCase(planned runtimeCase, result hostexec.Result, runErr error) (hostexec.Result, error) {
	caseResult := CaseResult{
		Tier: "test-runtime", Name: planned.name, ExitCode: result.ExitCode,
		TimedOut: result.WatchdogTimeout, Signaled: result.Termination == hostexec.TerminationSignal,
		Stdout: append([]byte(nil), result.Stdout.Bytes...), Stderr: append([]byte(nil), result.Stderr.Bytes...),
		Truncated: result.Stdout.Truncated || result.Stderr.Truncated,
	}
	if planned.wantTimeout {
		caseResult.Passed = runErr == nil && result.WatchdogTimeout && !caseResult.Truncated
	} else {
		caseResult.Passed = runErr == nil && !result.WatchdogTimeout && !caseResult.Signaled &&
			result.Termination == hostexec.TerminationExit && result.ExitCode == planned.wantExit && !caseResult.Truncated
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

func (campaign *runtimeCampaign) request(command []string, dir string, timeout time.Duration, unset []string, values ...string) hostexec.Request {
	return hostexec.Request{
		Command: command, Dir: dir, Env: append(filterEnvironment(os.Environ(), unset...), values...), Timeout: timeout,
		TerminateGrace: fixtureTerminationGrace, OutputLimit: fixtureOutputLimit,
	}
}

func (campaign *runtimeCampaign) command(name string, command []string, dir string, timeout time.Duration, unset []string, values ...string) (hostexec.Result, error) {
	return campaign.runCase(runtimeCase{name: name, request: campaign.request(command, dir, timeout, unset, values...)})
}

func (campaign *runtimeCampaign) expectedExit(name string, command []string, dir string, timeout time.Duration, wantExit int, oracle func(hostexec.Result) error, unset []string, values ...string) error {
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
		overlayRoot = filepath.Join(campaign.config.Root, "toolchain", "runtime", "overlay")
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

func commandOutput(result hostexec.Result) string {
	return strings.TrimRight(string(result.Stdout.RawBytes), "\n")
}

func commandErrorOutput(result hostexec.Result) string {
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
		{name: "choice-replay", packageName: "./choice_replay"},
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

func requireOutput(result hostexec.Result, want, label string) error {
	if actual := commandOutput(result); actual != want {
		return fmt.Errorf("%s output = %q, want %q", label, actual, want)
	}
	return nil
}
