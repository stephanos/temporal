// Package runner is the one host loop: it resets one Model instance, asks it to Propose, has a
// Driver act on each proposal, hands every observation to Judge and every callback to Answer,
// and records a Run. Modes are arguments to that loop.
package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	runv1 "umpire.local/model/gen/umpire/run/v1"
	model "umpire.local/model/host"
)

// ProtocolVersion is the umpire.model.v1 protocol version a Run records.
const ProtocolVersion = 1

// The per-Run budgets used when Options leaves them zero.
const (
	DefaultBudget       = 32
	DefaultEventBudget  = 1024
	DefaultEventTimeout = 30 * time.Second
	DefaultRunTimeout   = 5 * time.Minute
)

// Run is the recorded Run, the loop's only artifact.
type Run = runv1.Run

// Mode selects what the loop does.
type Mode int

const (
	ModeUnspecified Mode = iota
	// ModeQuestion replays a recorded prefix, then asks one Propose or Judge; no Driver.
	ModeQuestion
	// ModeExplore runs seeds 1..Seed.
	ModeExplore
	// ModeReduce finds the shortest proposal prefix of a recorded Unexplained Run that still
	// reproduces it.
	ModeReduce
	// ModeReplay replays a recorded Run's protocol calls against a fresh instance; no Driver.
	ModeReplay
	// ModeRepeat runs seeds 1..Seed, Count times each.
	ModeRepeat
)

var modeNames = map[Mode]string{
	ModeQuestion: "question", ModeExplore: "explore", ModeReduce: "reduce", ModeReplay: "replay", ModeRepeat: "repeat",
}

func (m Mode) String() string {
	if name, ok := modeNames[m]; ok {
		return name
	}
	return fmt.Sprintf("mode(%d)", int(m))
}

// ParseMode reads a mode by its name.
func ParseMode(name string) (Mode, error) {
	for mode, known := range modeNames {
		if known == name {
			return mode, nil
		}
	}
	return ModeUnspecified, fmt.Errorf("unknown mode %q", name)
}

// Options are the loop's arguments. Zero budgets and timeouts take the defaults above.
type Options struct {
	// World is the Reset configuration of explore and repeat, and of a question without a
	// recorded prefix.
	World *anypb.Any
	// Seed is the last seed of explore and repeat, or the seed of a question without a
	// recorded prefix.
	Seed uint64
	// Count is how many times repeat runs each seed.
	Count int
	// Budget caps proposals per Run; EventBudget caps recorded events per Run.
	Budget      int
	EventBudget int
	// EventTimeout bounds every wait on a Driver; RunTimeout bounds one Run.
	EventTimeout time.Duration
	RunTimeout   time.Duration
	// Recorded is the Run that question, reduce and replay start from.
	Recorded *Run
	// Prefix is how many recorded events a question replays first; zero replays them all.
	Prefix int
	// Observation makes a question Judge it instead of asking Propose.
	Observation *anypb.Any
	// Redact may mark an event's evidence as removed once the Run is judged; a redacted Run
	// cannot be replayed.
	Redact func(*runv1.Event) bool
	// Record receives every Run in execution order, except that reduce records only a reproduction.
	Record func(context.Context, *Run) error
	// Progress receives one redacted line per proposal and per finished Run.
	Progress io.Writer
}

// Runner runs one Model's guest against Drivers it makes, one fresh Host per Run.
type Runner struct {
	wasm      []byte
	hash      string
	newDriver NewDriver
	newHost   func(context.Context, []byte) (*model.Host, error)
	deadline  deadline
}

// deadlineKind says which budget bounds a wait: the read timeout or the run timeout.
type deadlineKind int

const (
	readDeadline deadlineKind = iota
	runDeadline
)

// deadline derives a context that ends after d. Every wait the Runner bounds goes through it, so
// tests can end a wait exactly when they choose instead of racing a clock.
type deadline func(ctx context.Context, kind deadlineKind, d time.Duration) (context.Context, context.CancelFunc)

func wallClock(ctx context.Context, _ deadlineKind, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, d)
}

// New returns a Runner for the guest bytes. The Driver factory may be nil for question and
// replay, which make no Driver.
func New(wasm []byte, driver NewDriver) *Runner {
	wasm = slices.Clone(wasm)
	sum := sha256.Sum256(wasm)
	return &Runner{wasm: wasm, hash: hex.EncodeToString(sum[:]), newDriver: driver, newHost: model.New, deadline: wallClock}
}

// ModelHash is the lower-case hex SHA-256 of the guest bytes.
func (r *Runner) ModelHash() string { return r.hash }

// Run runs the loop in one mode and returns the Run deciding its outcome: the reduced Run for
// reduce, the first Unexplained or else the first Refused or else the last Run for explore and
// repeat. An error means the loop could not run, not a verdict.
func (r *Runner) Run(ctx context.Context, mode Mode, opts Options) (*Run, error) {
	opts = opts.withDefaults()
	switch mode {
	case ModeQuestion:
		return r.question(ctx, opts)
	case ModeReplay:
		if opts.Recorded == nil {
			return nil, errors.New("replay needs a recorded Run")
		}
		return r.replayRun(ctx, opts)
	case ModeExplore:
		return r.batch(ctx, mode, opts, 1)
	case ModeRepeat:
		if opts.Count < 1 {
			return nil, errors.New("repeat needs a count of at least 1")
		}
		return r.batch(ctx, mode, opts, opts.Count)
	case ModeReduce:
		return r.reduce(ctx, opts)
	default:
		return nil, fmt.Errorf("unknown mode %v", mode)
	}
}

func (o Options) withDefaults() Options {
	if o.Budget <= 0 {
		o.Budget = DefaultBudget
	}
	if o.EventBudget <= 0 {
		o.EventBudget = DefaultEventBudget
	}
	if o.EventTimeout <= 0 {
		o.EventTimeout = DefaultEventTimeout
	}
	if o.RunTimeout <= 0 {
		o.RunTimeout = DefaultRunTimeout
	}
	return o
}

func (r *Runner) batch(ctx context.Context, mode Mode, opts Options, count int) (*Run, error) {
	if opts.Seed < 1 {
		return nil, fmt.Errorf("%s needs a seed of at least 1", mode)
	}
	if r.newDriver == nil {
		return nil, fmt.Errorf("%s needs a Driver", mode)
	}
	var decisive *Run
	for iteration := 1; iteration <= count; iteration++ {
		for seed := uint64(1); seed <= opts.Seed; seed++ {
			run, err := r.execute(ctx, mode, iteration, opts, opts.World, seed, opts.Budget)
			if err != nil {
				return nil, err
			}
			if err := record(ctx, opts, run); err != nil {
				return nil, err
			}
			decisive = decide(decisive, run)
		}
	}
	return decisive, nil
}

// decide keeps the first Unexplained Run, else the first Refused, else the latest.
func decide(current, next *Run) *Run {
	rank := func(run *Run) int {
		switch {
		case run == nil:
			return -1
		case run.GetVerdict().GetUnexplained() != nil:
			return 2
		case run.GetVerdict().GetRefused() != nil:
			return 1
		default:
			return 0
		}
	}
	if rank(current) > 0 && rank(current) >= rank(next) {
		return current
	}
	return next
}

func (r *Runner) reduce(ctx context.Context, opts Options) (*Run, error) {
	recorded := opts.Recorded
	if recorded == nil || recorded.GetVerdict().GetUnexplained() == nil {
		return nil, errors.New("reduce needs a recorded Unexplained Run")
	}
	// The recorded Run must replay here, which checks its model, protocol, world and seeds.
	check, err := r.replayRun(ctx, Options{Recorded: recorded}.withDefaults())
	if err != nil {
		return nil, err
	}
	if check.GetVerdict().GetUnexplained() == nil {
		return nil, fmt.Errorf("reduce: the recorded Run does not replay: %s", check.GetVerdict().GetRefused().GetReason())
	}
	proposals := int(proposals(recorded.GetEvents()))
	var last *Run
	for budget := 1; budget <= proposals; budget++ {
		run, err := r.execute(ctx, ModeReduce, budget, opts, recorded.GetWorld(), recorded.GetSeed(), budget)
		if err != nil {
			return nil, err
		}
		if run.GetVerdict().GetUnexplained() != nil {
			return run, record(ctx, opts, run)
		}
		last = run
	}
	if last == nil {
		return nil, errors.New("reduce: the recorded Run has no proposal")
	}
	// Nothing reproduced: report the last candidate Refused and publish nothing.
	report := proto.CloneOf(last)
	report.Verdict = refused(uint64(len(report.GetEvents())), fmt.Sprintf("reduce: not reproduced within %d candidates", proposals))
	return report, nil
}

func record(ctx context.Context, opts Options, run *Run) error {
	if opts.Record == nil {
		return nil
	}
	return opts.Record(ctx, run)
}

// ProposalSeed is the seed of a Run's Propose for one request number (splitmix64).
func ProposalSeed(seed, request uint64) uint64 {
	z := seed + request*0x9e3779b97f4a7c15
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}
