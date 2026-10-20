package sweep

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"
)

type deadlineExecutor func(context.Context, Command) Completed

func (execute deadlineExecutor) Exec(ctx context.Context, command Command) Completed {
	return execute(ctx, command)
}

func TestSweepBoundsEveryOperationByOverallDeadline(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	sweeper := f.sweeper()
	latest := time.Now().Add(f.options.OverallBudget)
	var invocationDeadline time.Time
	operations := map[string]bool{}
	sweeper.Executor = deadlineExecutor(func(ctx context.Context, command Command) Completed {
		deadline, ok := ctx.Deadline()
		if !ok || deadline.After(latest) {
			t.Errorf("operation %s %v has deadline %v, want at most %v", command.Path, command.Args, deadline, latest)
		}
		if invocationDeadline.IsZero() {
			invocationDeadline = deadline
		} else if !deadline.Equal(invocationDeadline) {
			t.Errorf("operation %s %v changed the invocation deadline from %v to %v", command.Path, command.Args, invocationDeadline, deadline)
		}
		operations[command.Args[0]] = true
		return f.Exec(ctx, command)
	})
	// Allow Run to set its deadline after the test's reference time.
	latest = latest.Add(time.Second)
	if _, err := sweeper.Run(context.WithoutCancel(t.Context())); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []string{"list", "test", "env", "-C", "install", "--installation"} {
		if !operations[operation] {
			t.Fatalf("operation %s was not exercised", operation)
		}
	}
}

func TestSweepWarmupReservesItsPreparationBoundAndOneTest(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name    string
		budget  time.Duration
		warmups int
	}{{"does-not-fit", 9 * time.Minute, 0}, {"fits", 10 * time.Minute, 2}} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.options.PerTestTimeout, f.options.PrepareTimeout, f.options.OverallBudget = time.Minute, 3*time.Minute, test.budget
			sweeper := f.sweeper()
			now := time.Now()
			sweeper.Now = func() time.Time { return now }
			if _, err := sweeper.Run(t.Context()); err != nil {
				t.Fatal(err)
			}
			if len(f.warmups) != test.warmups || len(f.explores) != 3 {
				t.Fatalf("warm-ups=%d tests=%d, want warm-ups=%d tests=3", len(f.warmups), len(f.explores), test.warmups)
			}
		})
	}
}

func TestSweepRechecksAdmissionAfterWarmup(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	sweeper := f.sweeper()
	now := time.Now()
	sweeper.Now = func() time.Time { return now }
	sweeper.Executor = deadlineExecutor(func(ctx context.Context, command Command) Completed {
		if flagValue(command.Args, "-test.run") == "^$" {
			now = now.Add(f.options.OverallBudget)
		}
		return f.Exec(ctx, command)
	})
	if _, err := sweeper.Run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(f.warmups) != 1 || len(f.explores) != 0 {
		t.Fatalf("warm-ups=%d tests=%d, want one warm-up and no test after it consumes the budget", len(f.warmups), len(f.explores))
	}
	if sidecar := readSidecar(t, f.options.Out); sidecar.ByClass[ClassNotRun] != 3 {
		t.Fatalf("post-warm-up cutoff classes: %v", sidecar.ByClass)
	}
}

func TestSweepOverallDeadlineCutoff(t *testing.T) {
	t.Parallel()
	for _, phase := range []string{"enumeration", "installation", "warm-up", "test"} {
		t.Run(phase, func(t *testing.T) {
			t.Parallel()
			f := newFixture(t)
			f.options.PerTestTimeout, f.options.PrepareTimeout = time.Nanosecond, time.Nanosecond
			f.options.OverallBudget = 500 * time.Millisecond
			sweeper := f.sweeper()
			// Admission time stays fixed so only the real context deadline can stop work.
			now := time.Now()
			sweeper.Now = func() time.Time { return now }
			cutoff := false
			sweeper.Executor = deadlineExecutor(func(ctx context.Context, command Command) Completed {
				if cutoff {
					t.Fatal("subprocess started after the overall deadline")
				}
				run := flagValue(command.Args, "-test.run")
				stop := phase == "enumeration" && command.Args[0] == "list" ||
					phase == "installation" && command.Args[0] == "install" ||
					phase == "warm-up" && run == "^$" ||
					phase == "test" && slices.Contains(command.Args, "explore") && run != "^$"
				if !stop {
					return f.Exec(ctx, command)
				}
				if _, ok := ctx.Deadline(); !ok {
					t.Fatal("subprocess has no overall deadline")
				}
				<-ctx.Done()
				cutoff = true
				return Completed{Exit: -1, Err: ctx.Err(), Elapsed: f.options.OverallBudget}
			})
			findings, err := sweeper.Run(context.WithoutCancel(t.Context()))
			if !cutoff {
				t.Fatal("deadline cutoff was not exercised")
			}
			if phase == "enumeration" || phase == "installation" {
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("pre-header cutoff returned %v, want deadline error", err)
				}
				if _, err := os.Stat(SidecarPath(f.options.Out)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("pre-header cutoff published a sidecar: %v", err)
				}
				return
			}
			if err != nil || len(findings) != 0 {
				t.Fatalf("cutoff failed to finalize: %v %v", err, findings)
			}
			sidecar := readSidecar(t, f.options.Out)
			if sidecar.ByClass[ClassNotRun] != 3 || sidecar.ByClass[ClassBuildFailure] != 1 || len(sidecar.Rows) != 4 {
				t.Fatalf("cutoff fabricated test results: %+v", sidecar.Rows)
			}
			for _, row := range sidecar.Rows {
				if row.Class == ClassNotRun && !strings.Contains(row.Reason, "overall budget") {
					t.Fatalf("cutoff lost its reason: %+v", row)
				}
			}
		})
	}
}
