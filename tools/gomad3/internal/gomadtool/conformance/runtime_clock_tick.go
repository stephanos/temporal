package conformance

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

// requireClockTickBehavior checks the virtual-clock tick policies on the actual
// runtime: strict keeps every read of one busy stretch at the same instant,
// forward advances each read by a seeded 1 to 1024 nanoseconds that repeats for
// one seed, and an unknown policy stops before user initialization.
func (campaign *runtimeCampaign) requireClockTickBehavior(binary string) error {
	run := func(name, seed string, values ...string) ([]int64, error) {
		environment := append([]string{"GOMADSEED=" + seed, "TZ=UTC"}, values...)
		result, err := campaign.command(name, []string{binary}, campaign.testdata, 5*time.Second, []string{"GOMADSEED", "TZ", "GOMAD3_CLOCK_TICK"}, environment...)
		if err != nil {
			return nil, err
		}
		fields := strings.Fields(commandOutput(result))
		if len(fields) != 8 {
			return nil, fmt.Errorf("%s printed %d readings, want 8", name, len(fields))
		}
		readings := make([]int64, len(fields))
		for index, field := range fields {
			readings[index], err = strconv.ParseInt(field, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("%s reading %d: %w", name, index, err)
			}
		}
		return readings, nil
	}

	strict, err := run("clock-tick-strict", "1")
	if err != nil {
		return err
	}
	for _, reading := range strict[1:] {
		if reading != strict[0] {
			return fmt.Errorf("strict clock moved within one busy stretch: %v", strict)
		}
	}

	forward := map[string][]int64{}
	for _, seed := range []string{"1", "1", "2"} {
		readings, err := run("clock-tick-forward-seed-"+seed+"-"+strconv.Itoa(len(forward)), seed, "GOMAD3_CLOCK_TICK=forward")
		if err != nil {
			return err
		}
		for index := 1; index < len(readings); index++ {
			if delta := readings[index] - readings[index-1]; delta < 1 || delta > 1024 {
				return fmt.Errorf("forward clock seed %s advanced %d ns between reads, want 1 to 1024: %v", seed, delta, readings)
			}
		}
		if previous, found := forward[seed]; found && fmt.Sprint(previous) != fmt.Sprint(readings) {
			return fmt.Errorf("forward clock seed %s did not repeat: %v then %v", seed, previous, readings)
		}
		forward[seed] = readings
	}
	if fmt.Sprint(forward["1"]) == fmt.Sprint(forward["2"]) {
		return errors.New("forward clock draws did not depend on the seed")
	}

	return campaign.expectedExit(
		"clock-tick-invalid", []string{binary}, campaign.testdata, 5*time.Second, 2,
		func(result hostexec.Result) error {
			if len(result.Stdout.RawBytes) != 0 || commandErrorOutput(result) != "runtime: invalid GOMAD3_CLOCK_TICK" {
				return errors.New("an unknown clock tick policy reached user initialization or emitted an unexpected diagnostic")
			}
			return nil
		},
		[]string{"GOMADSEED", "TZ", "GOMAD3_CLOCK_TICK"}, "GOMADSEED=1", "TZ=UTC", "GOMAD3_CLOCK_TICK=strict",
	)
}
