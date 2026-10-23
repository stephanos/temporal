package runner

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// The command's exit codes. ExitError means the loop could not run at all.
const (
	ExitExplained   = 0
	ExitUnexplained = 1
	ExitRefused     = 2
	ExitError       = 3
)

const usage = "usage: umpire <question|explore|reduce|replay|repeat> --model <guest.wasm> [--seed n] [--budget n] [--run <path>] " +
	"[--world <any.binpb>] [--observation <any.binpb>] [--events n] [--count n] [--store <dir>] [--driver <name>] " +
	"[--event-budget n] [--event-timeout d] [--run-timeout d]"

// Main is the umpire command over the named Drivers. Explore, reduce and repeat need a Driver and
// a store; question and replay use neither Driver nor network. Progress goes to stderr, each
// published key and the deciding verdict to stdout.
func Main(ctx context.Context, args []string, stdout, stderr io.Writer, drivers map[string]NewDriver) int {
	fail := func(format string, values ...any) int {
		_, _ = fmt.Fprintf(stderr, "umpire: "+format+"\n", values...)
		return ExitError
	}
	if len(args) == 0 {
		return fail("%s", usage)
	}
	mode, err := ParseMode(args[0])
	if err != nil {
		return fail("%v\n%s", err, usage)
	}
	parsed, err := parse(mode, args[1:], stderr)
	if err != nil {
		return fail("%v", err)
	}
	wasm, opts, err := parsed.options(stderr)
	if err != nil {
		return fail("%v", err)
	}
	var driver NewDriver
	if mode == ModeExplore || mode == ModeReduce || mode == ModeRepeat {
		if driver, err = selectDriver(drivers, parsed.driver); err != nil {
			return fail("%s: %v", mode, err)
		}
		if parsed.store == "" {
			return fail("%s needs --store", mode)
		}
	}
	if parsed.store != "" {
		opts.Record = publisher(parsed.store, stdout)
	}
	run, err := New(wasm, driver).Run(ctx, mode, opts)
	if err == nil && (mode == ModeQuestion || mode == ModeReplay) {
		err = record(ctx, opts, run)
	}
	if err != nil {
		return fail("%s: %v", mode, err)
	}
	return report(run, stdout)
}

type commandFlags struct {
	model, run, world, observation, store, driver string
	seed                                          uint64
	budget, eventBudget, events, count            int
	eventTimeout, runTimeout                      time.Duration
}

func parse(mode Mode, args []string, stderr io.Writer) (*commandFlags, error) {
	parsed := new(commandFlags)
	flags := flag.NewFlagSet("umpire "+mode.String(), flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&parsed.model, "model", "", "the guest wasm")
	flags.Uint64Var(&parsed.seed, "seed", 0, "explore and repeat: run seeds 1..n; question without --run: the Run's seed")
	flags.IntVar(&parsed.budget, "budget", DefaultBudget, "proposals per Run")
	flags.IntVar(&parsed.eventBudget, "event-budget", DefaultEventBudget, "recorded events per Run")
	flags.StringVar(&parsed.run, "run", "", "the recorded Run question, reduce and replay start from")
	flags.StringVar(&parsed.world, "world", "", "a binary google.protobuf.Any: the Reset configuration")
	flags.StringVar(&parsed.observation, "observation", "", "question: a binary google.protobuf.Any to Judge instead of asking Propose")
	flags.IntVar(&parsed.events, "events", 0, "question: how many recorded events to replay first; 0 replays all")
	flags.IntVar(&parsed.count, "count", 1, "repeat: executions of each seed")
	flags.StringVar(&parsed.store, "store", "", "the directory Runs are published under")
	flags.StringVar(&parsed.driver, "driver", "", "the Driver, when more than one is available")
	flags.DurationVar(&parsed.eventTimeout, "event-timeout", DefaultEventTimeout, "the longest wait on the Driver")
	flags.DurationVar(&parsed.runTimeout, "run-timeout", DefaultRunTimeout, "the longest Run")
	if err := flags.Parse(args); err != nil {
		return nil, err
	}
	if flags.NArg() != 0 {
		return nil, fmt.Errorf("%s accepts no positional arguments", mode)
	}
	if parsed.model == "" {
		return nil, fmt.Errorf("--model is required\n%s", usage)
	}
	return parsed, nil
}

// options reads the files the flags name.
func (c *commandFlags) options(progress io.Writer) ([]byte, Options, error) {
	opts := Options{
		Seed: c.seed, Count: c.count, Budget: c.budget, EventBudget: c.eventBudget, Prefix: c.events,
		EventTimeout: c.eventTimeout, RunTimeout: c.runTimeout, Progress: progress,
	}
	wasm, err := os.ReadFile(c.model)
	if err != nil {
		return nil, opts, fmt.Errorf("read the model: %w", err)
	}
	if c.world != "" {
		if opts.World, err = readAny(c.world); err != nil {
			return nil, opts, fmt.Errorf("read the world: %w", err)
		}
	}
	if c.observation != "" {
		if opts.Observation, err = readAny(c.observation); err != nil {
			return nil, opts, fmt.Errorf("read the observation: %w", err)
		}
	}
	if c.run != "" {
		if opts.Recorded, err = Load(c.run); err != nil {
			return nil, opts, fmt.Errorf("read the recorded Run: %w", err)
		}
	}
	return wasm, opts, nil
}

func publisher(store string, stdout io.Writer) func(context.Context, *Run) error {
	return func(ctx context.Context, run *Run) error {
		publication, err := Publish(ctx, store, run)
		if err != nil {
			return err
		}
		status := "published"
		if publication.Existing {
			status = "already-published"
		}
		_, err = fmt.Fprintf(stdout, "%s %s\n", status, publication.Key)
		return err
	}
}

func report(run *Run, stdout io.Writer) int {
	verdict := run.GetVerdict()
	switch {
	case verdict.GetUnexplained() != nil:
		_, _ = fmt.Fprintf(stdout, "unexplained at event %d\n", verdict.GetUnexplained().GetEvent())
		return ExitUnexplained
	case verdict.GetRefused() != nil:
		_, _ = fmt.Fprintf(stdout, "refused at event %d: %s\n", verdict.GetRefused().GetEvent(), verdict.GetRefused().GetReason())
		return ExitRefused
	default:
		termination := strings.TrimPrefix(verdict.GetExplained().GetTermination().String(), "TERMINATION_")
		_, _ = fmt.Fprintf(stdout, "explained: %s\n", strings.ToLower(termination))
		return ExitExplained
	}
}

func selectDriver(drivers map[string]NewDriver, name string) (NewDriver, error) {
	if name != "" {
		if driver := drivers[name]; driver != nil {
			return driver, nil
		}
		return nil, fmt.Errorf("unknown driver %q", name)
	}
	if len(drivers) == 0 {
		return nil, errors.New("this build has no Driver")
	}
	names := make([]string, 0, len(drivers))
	for name := range drivers {
		names = append(names, name)
	}
	slices.Sort(names)
	if len(names) == 1 {
		return drivers[names[0]], nil
	}
	return nil, fmt.Errorf("choose a --driver: %s", strings.Join(names, ", "))
}

func readAny(path string) (*anypb.Any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	value := new(anypb.Any)
	if err := proto.Unmarshal(data, value); err != nil {
		return nil, err
	}
	if value.GetTypeUrl() == "" {
		return nil, errors.New("the Any has no type URL")
	}
	return value, nil
}
