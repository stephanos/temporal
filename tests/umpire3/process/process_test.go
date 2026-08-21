package process

import (
	"context"
	"io"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.temporal.io/server/common/testing/await"
)

func TestRunReturnsBoundedWorkerOutput(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=echo"},
		Input:       []byte("request"), Timeout: time.Second, MaxOutputBytes: 64,
	})
	require.NoError(t, err)
	require.Equal(t, []byte("request"), result.Output)
	require.Equal(t, 0, result.ExitCode)
	require.False(t, result.TimedOut)
	require.Positive(t, result.DurationNanos)
	require.Positive(t, result.PeakMemoryBytes)
}

func TestRunPassesOnlyExplicitEnvironment(t *testing.T) {
	t.Setenv("UMPIRE3_PROCESS_PARENT_SECRET", "must-not-cross-boundary")

	result, err := Run(context.Background(), Request{
		Command: []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{
			"UMPIRE3_PROCESS_MODE=environment",
			"UMPIRE3_PROCESS_EXPLICIT=allowed",
		},
		Timeout: time.Second, MaxOutputBytes: 64,
	})
	require.NoError(t, err)
	require.Equal(t, "|allowed", string(result.Output))
}

func TestRunTerminatesNonCooperativeWorker(t *testing.T) {
	t.Parallel()

	started := time.Now()
	result, err := Run(context.Background(), Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=block"},
		Timeout:     100 * time.Millisecond, MaxOutputBytes: 64,
	})
	require.ErrorIs(t, err, ErrDeadline)
	require.True(t, result.TimedOut)
	require.Less(t, time.Since(started), 2*time.Second)
}

func TestRunRejectsOutputBeyondBudget(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=large"},
		Timeout:     time.Second, MaxOutputBytes: 8,
	})
	require.ErrorIs(t, err, ErrOutputLimit)
}

func TestRunKillsNonCooperativeWorkerBeyondOutputBudget(t *testing.T) {
	t.Parallel()

	started := time.Now()
	_, err := Run(context.Background(), Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=flood"},
		Timeout:     time.Second, MaxOutputBytes: 8,
	})
	require.ErrorIs(t, err, ErrOutputLimit)
	require.Less(t, time.Since(started), time.Second)
}

func TestRunAppliesWorkerCPULimit(t *testing.T) {
	t.Parallel()

	result, err := Run(context.Background(), Request{
		Command: []string{"/bin/sh", "-c", "ulimit -t"},
		Timeout: time.Second, MaxOutputBytes: 1024,
		Limits: Limits{CPUSeconds: 2, MemoryBytes: 64 << 20},
	})
	require.NoError(t, err)
	require.Equal(t, "2\n", string(result.Output))
}

func TestRunKillsWorkerBeyondMemoryLimit(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=block"},
		Timeout:     time.Second, MaxOutputBytes: 64,
		Limits: Limits{CPUSeconds: 10, MemoryBytes: 1},
	})
	require.ErrorIs(t, err, ErrMemoryLimit)
}

func TestRunRejectsPartialWorkerResourceLimits(t *testing.T) {
	t.Parallel()

	_, err := Run(context.Background(), Request{
		Command: []string{"true"}, Timeout: time.Second, MaxOutputBytes: 64,
		Limits: Limits{CPUSeconds: 1},
	})
	require.ErrorContains(t, err, "CPU and memory limits")
}

func TestSupervisorCrashesRestartsAndCleansUpIsolatedProcess(t *testing.T) {
	t.Parallel()

	supervisor, err := NewSupervisor(Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=supervised"}, Timeout: time.Second, MaxOutputBytes: 64,
	})
	require.NoError(t, err)
	first, err := supervisor.Start(context.Background())
	require.NoError(t, err)
	require.Equal(t, 1, first.Generation)
	require.Positive(t, first.PID)
	await.RequireTrue(t, func() bool {
		return string(supervisor.Snapshot().Output) == "ready"
	}, time.Second, 10*time.Millisecond)

	crashed, err := supervisor.Crash(context.Background())
	require.NoError(t, err)
	require.Equal(t, TerminationCrash, crashed.Termination)
	require.NotZero(t, crashed.StoppedAtUnixNano)

	second, err := supervisor.Restart(context.Background())
	require.NoError(t, err)
	require.Equal(t, 2, second.Generation)
	require.NotEqual(t, first.PID, second.PID)
	await.RequireTrue(t, func() bool {
		return string(supervisor.Snapshot().Output) == "ready"
	}, time.Second, 10*time.Millisecond)

	stopped, err := supervisor.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, TerminationStop, stopped.Termination)
	again, err := supervisor.Stop(context.Background())
	require.NoError(t, err)
	require.Equal(t, stopped, again)
}

func TestSupervisorRejectsRestartWhileProcessIsRunning(t *testing.T) {
	t.Parallel()

	supervisor, err := NewSupervisor(Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=supervised"}, Timeout: time.Second, MaxOutputBytes: 64,
	})
	require.NoError(t, err)
	_, err = supervisor.Start(context.Background())
	require.NoError(t, err)
	t.Cleanup(func() {
		_, cleanupErr := supervisor.Stop(context.Background())
		require.NoError(t, cleanupErr)
	})

	_, err = supervisor.Restart(context.Background())
	require.ErrorContains(t, err, "still running")
}

func TestSupervisorKillsWorkerBeyondOutputBudget(t *testing.T) {
	t.Parallel()

	supervisor, err := NewSupervisor(Request{
		Command:     []string{os.Args[0], "-test.run=TestProcessWorker"},
		Environment: []string{"UMPIRE3_PROCESS_MODE=flood"}, Timeout: time.Second, MaxOutputBytes: 8,
	})
	require.NoError(t, err)
	_, err = supervisor.Start(context.Background())
	require.NoError(t, err)
	attempt, err := supervisor.Wait(context.Background())
	require.ErrorIs(t, err, ErrOutputLimit)
	require.Equal(t, TerminationLimit, attempt.Termination)
	require.NotZero(t, attempt.StoppedAtUnixNano)
}

func TestSupervisorRejectsPartialWorkerResourceLimits(t *testing.T) {
	t.Parallel()

	_, err := NewSupervisor(Request{
		Command: []string{"true"}, Timeout: time.Second, MaxOutputBytes: 64,
		Limits: Limits{CPUSeconds: 1},
	})
	require.ErrorContains(t, err, "CPU and memory limits")
}

func TestProcessWorker(t *testing.T) {
	switch os.Getenv("UMPIRE3_PROCESS_MODE") {
	case "echo":
		input, err := io.ReadAll(os.Stdin)
		require.NoError(t, err)
		_, err = os.Stdout.Write(input)
		require.NoError(t, err)
		//nolint:revive // The helper process must not append the Go test runner's PASS output to stdout.
		os.Exit(0)
	case "environment":
		_, err := os.Stdout.WriteString(os.Getenv("UMPIRE3_PROCESS_PARENT_SECRET") + "|" +
			os.Getenv("UMPIRE3_PROCESS_EXPLICIT"))
		require.NoError(t, err)
		//nolint:revive // The helper process must terminate before the Go test runner writes to stdout.
		os.Exit(0)
	case "block":
		for {
			runtime.Gosched()
		}
	case "large":
		_, err := os.Stdout.Write([]byte("0123456789abcdef"))
		require.NoError(t, err)
		//nolint:revive // The helper process must terminate before the Go test runner writes to stdout.
		os.Exit(0)
	case "flood":
		for {
			_, err := os.Stdout.Write([]byte("0123456789abcdef"))
			if err != nil {
				//nolint:revive // The subprocess helper reports a closed output stream through its exit status.
				os.Exit(4)
			}
		}
	case "supervised":
		_, err := os.Stdout.Write([]byte("ready"))
		require.NoError(t, err)
		for {
			runtime.Gosched()
		}
	default:
		return
	}
}
