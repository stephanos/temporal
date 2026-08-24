package conformance

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

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
			[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOCACHE", "GOMAD3_CHILD_SEED"},
			"CGO_ENABLED=0", "TZ=UTC", "GOCACHE="+cache, "GOMAD3_CHILD_SEED=1",
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
		return errors.New("disabled gomad3 unexpectedly fixed map iteration order")
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
		result  hostexec.Result
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
						[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMAD3_CHILD_SEED"},
						"CGO_ENABLED=0", "TZ=UTC", "GOMAD3_CHILD_SEED="+seed,
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
		[]string{campaign.config.Go, "test", "-exec", campaign.execWrapper, "-count=1", "-tags=test_dep", "-run", "^TestSeedReachesTestBinary$", "-v", "./gotest"},
		campaign.testdata, time.Minute,
		[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMAD3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMAD3_CHILD_SEED="+seed,
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
			return nil, errors.New("gomad3 host-load worker failed to start")
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
