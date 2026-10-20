package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends"
)

type replayInspectionFlags struct {
	stop, instance, output, check, choiceRange, mode string
	incarnation, attempts                            uint64
	wallTime                                         time.Duration
	maximumBytes, perDumpBytes                       byteSize
	dumps, probes                                    stringList
}

type stressReplayProvider interface {
	ForStressReplay() (backends.Provider, error)
}

func newReplayInspectionFlags(flags *flag.FlagSet) *replayInspectionFlags {
	options := &replayInspectionFlags{maximumBytes: backends.DefaultInspectionBytes, perDumpBytes: backends.DefaultInspectionDumpBytes}
	flags.StringVar(&options.stop, "stop", "", "inspection boundary: choice=N, time=DURATION (or nanoseconds), or failure")
	flags.StringVar(&options.instance, "instance", "", "recorded instance ID for inspection")
	flags.Uint64Var(&options.incarnation, "incarnation", 0, "positive recorded incarnation for inspection")
	flags.Var(&options.dumps, "dump", "target dump name; repeat to select multiple dumps")
	flags.Var(&options.probes, "probe", "target probe glob; repeat to select multiple patterns")
	flags.StringVar(&options.output, "debug-output", "", "inspection output directory (default adjacent to the artifact)")
	flags.Var(&options.maximumBytes, "debug-bytes", "maximum total diagnostic bytes")
	flags.Var(&options.perDumpBytes, "dump-bytes", "maximum bytes per dump")
	flags.StringVar(&options.check, "check", "", "registered invariant to search in fresh exact replays")
	flags.StringVar(&options.choiceRange, "choice-range", "", "inclusive local invariant-search range FIRST:LAST")
	flags.StringVar(&options.mode, "search-mode", "scan", "scan or declared-monotonic bisect")
	flags.Uint64Var(&options.attempts, "search-attempts", 128, "positive maximum number of fresh search replays")
	flags.DurationVar(&options.wallTime, "search-timeout", time.Minute, "positive whole-search wall bound, including startup and cleanup")
	return options
}

func (options *replayInspectionFlags) request(flags *flag.FlagSet, artifactPath string) (*backends.InspectionRequest, error) {
	enabled, searchOptions := false, false
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "stop", "instance", "incarnation", "dump", "probe", "debug-output", "debug-bytes", "dump-bytes":
			enabled = true
		case "check", "choice-range", "search-mode", "search-attempts", "search-timeout":
			enabled = true
			searchOptions = true
		}
	})
	if !enabled {
		return nil, nil
	}
	output := options.output
	if output == "" {
		output = filepath.Clean(artifactPath) + ".inspection"
	}
	output, err := filepath.Abs(output)
	if err != nil {
		return nil, err
	}
	request := &backends.InspectionRequest{Instance: options.instance, Incarnation: options.incarnation, Dumps: append([]string(nil), options.dumps...), Probes: append([]string(nil), options.probes...), MaximumBytes: uint64(options.maximumBytes), PerDumpBytes: uint64(options.perDumpBytes), OutputDirectory: output}
	if searchOptions {
		if options.check == "" || options.choiceRange == "" || options.stop != "" {
			return nil, fmt.Errorf("search requires --check and --choice-range, without --stop")
		}
		first, last, ok := strings.Cut(options.choiceRange, ":")
		if !ok {
			return nil, fmt.Errorf("choice range must be FIRST:LAST")
		}
		start, err := strconv.ParseUint(first, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid first choice: %w", err)
		}
		end, err := strconv.ParseUint(last, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid last choice: %w", err)
		}
		request.Search = &backends.InspectionSearchRequest{Check: options.check, First: start, Last: end, Mode: options.mode, AttemptBudget: options.attempts, WallTime: options.wallTime}
	} else {
		stop := options.stop
		if stop == "" {
			stop = "failure"
		}
		request.Stop, err = parseInspectionStop(stop)
		if err != nil {
			return nil, err
		}
		if len(request.Dumps) == 0 {
			request.Dumps = []string{"stacks", "heap"}
		}
	}
	if err := request.Validate(); err != nil {
		return nil, err
	}
	return request, nil
}

func parseInspectionStop(value string) (backends.InspectionStop, error) {
	if value == "failure" {
		return backends.InspectionStop{Kind: "failure"}, nil
	}
	kind, coordinate, ok := strings.Cut(value, "=")
	if !ok || kind != "choice" && kind != "time" {
		return backends.InspectionStop{}, fmt.Errorf("stop must be choice=N, time=T or failure")
	}
	ordinal, err := strconv.ParseUint(coordinate, 10, 64)
	if err == nil {
		return backends.InspectionStop{Kind: kind, Value: ordinal}, nil
	}
	if kind == "time" {
		duration, err := time.ParseDuration(coordinate)
		if err == nil && duration >= 0 {
			return backends.InspectionStop{Kind: kind, Value: uint64(duration)}, nil
		}
	}
	return backends.InspectionStop{}, fmt.Errorf("invalid nonnegative %s stop %q", kind, coordinate)
}

func reportReplayInspection(output io.Writer, result backends.InspectionResult) (int, error) {
	status := 1
	if result.Status == backends.InspectionStatusInspectedPrefix && result.PrefixVerified && result.WitnessMatch {
		status = 0
	}
	if result.Search != nil && result.Search.Status == "no_failure" {
		status = 0
	}
	return status, json.NewEncoder(output).Encode(result)
}
