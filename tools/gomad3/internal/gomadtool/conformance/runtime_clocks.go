package conformance

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

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
		func(result hostexec.Result) error {
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
		campaign.testdata, time.Minute, []string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMAD3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMAD3_CHILD_SEED=1",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(clockRun, "clock initial ok", "clock go run"); err != nil {
		return err
	}
	clockGoTest, err := campaign.command(
		"clock-go-test", []string{campaign.config.Go, "test", "-v", "-exec", campaign.execWrapper, "-count=1", "-tags=test_dep", "-timeout=48h", "./clock_gotest"},
		campaign.testdata, time.Minute, []string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMAD3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMAD3_CHILD_SEED=1",
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
		campaign.testdata, time.Minute, []string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMAD3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMAD3_CHILD_SEED=1",
	); err != nil {
		return err
	}
	return campaign.expectedExit(
		"clock-logical-timeout",
		[]string{campaign.config.Go, "test", "-exec", campaign.execWrapper, "-count=1", "-tags=test_dep", "-timeout=1h", "-run", "^TestLogicalTimeout$", "./clock_gotest", "-args", "-gomad-logical-timeout"},
		campaign.testdata, 5*time.Second, 1,
		func(result hostexec.Result) error {
			if result.WatchdogTimeout || !strings.Contains(string(result.Stdout.RawBytes)+string(result.Stderr.RawBytes), "panic: test timed out after 1h0m0s") {
				return errors.New("logical go test timeout was not distinct from the wall watchdog")
			}
			return nil
		},
		[]string{"GOMADSEED", "CGO_ENABLED", "TZ", "GOMAD3_CHILD_SEED"},
		"CGO_ENABLED=0", "TZ=UTC", "GOMAD3_CHILD_SEED=1",
	)
}
