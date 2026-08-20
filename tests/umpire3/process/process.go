package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrDeadline    = errors.New("worker execution deadline exceeded")
	ErrMemoryLimit = errors.New("worker memory limit exceeded")
	ErrOutputLimit = errors.New("worker output limit exceeded")
)

type Request struct {
	Command        []string
	Environment    []string
	Input          []byte
	Timeout        time.Duration
	MaxOutputBytes int64
	Limits         Limits
}

type Limits struct {
	CPUSeconds  int
	MemoryBytes int64
}

type Result struct {
	Output   []byte
	ExitCode int
	TimedOut bool
}

func Run(ctx context.Context, request Request) (Result, error) {
	if len(request.Command) == 0 || request.Timeout <= 0 || request.MaxOutputBytes <= 0 {
		return Result{}, errors.New("worker command, timeout, and output budget are required")
	}
	attempt, err := startProcessAttempt(request, 1)
	if err != nil {
		return Result{}, err
	}
	workerCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()
	select {
	case <-attempt.done:
		return completedRun(attempt)
	case <-workerCtx.Done():
		_ = attempt.kill(syscall.SIGKILL, TerminationDeadline)
		<-attempt.done
		result, _ := completedRun(attempt)
		result.TimedOut = true
		if errors.Is(workerCtx.Err(), context.DeadlineExceeded) {
			return result, ErrDeadline
		}
		return result, workerCtx.Err()
	case <-attempt.memoryExceeded:
		_ = attempt.kill(syscall.SIGKILL, TerminationLimit)
		<-attempt.done
		result, _ := completedRun(attempt)
		return result, ErrMemoryLimit
	case <-attempt.output.exceededSignal:
		_ = attempt.kill(syscall.SIGKILL, TerminationLimit)
		<-attempt.done
		result, _ := completedRun(attempt)
		return result, ErrOutputLimit
	}
}

func completedRun(attempt *processAttempt) (Result, error) {
	snapshot := attempt.snapshot()
	result := Result{Output: snapshot.Output, ExitCode: snapshot.ExitCode}
	if attempt.output.Exceeded() {
		return result, ErrOutputLimit
	}
	attempt.mu.Lock()
	waitErr := attempt.waitErr
	attempt.mu.Unlock()
	if waitErr != nil && snapshot.Termination == TerminationExit {
		return result, fmt.Errorf("worker exited with status %d: %w", result.ExitCode, waitErr)
	}
	return result, nil
}

func commandWithLimits(command []string, limits Limits) ([]string, error) {
	if limits == (Limits{}) {
		return append([]string(nil), command...), nil
	}
	if limits.CPUSeconds <= 0 || limits.MemoryBytes <= 0 {
		return nil, errors.New("worker CPU and memory limits must both be positive")
	}
	arguments := []string{
		"/bin/sh", "-c",
		"cpu_seconds=$1; shift; ulimit -t \"$cpu_seconds\"; exec \"$@\"",
		"umpire3-worker", strconv.Itoa(limits.CPUSeconds),
	}
	return append(arguments, command...), nil
}

func watchProcessGroupMemory(pid int, limit int64, exceeded chan<- struct{}, stop <-chan struct{}) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			usage, err := processGroupMemoryBytes(pid)
			if err == nil && usage > limit {
				select {
				case exceeded <- struct{}{}:
				default:
				}
				return
			}
		case <-stop:
			return
		}
	}
}

func processGroupMemoryBytes(groupID int) (int64, error) {
	output, err := exec.Command("/bin/ps", "-axo", "pgid=,rss=").Output()
	if err != nil {
		return 0, err
	}
	var totalKiB int64
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[0] != strconv.Itoa(groupID) {
			continue
		}
		value, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		totalKiB += value
	}
	return totalKiB * 1024, nil
}

type limitedBuffer struct {
	mu             sync.Mutex
	buffer         bytes.Buffer
	remaining      int64
	exceeded       bool
	exceededSignal chan struct{}
}

func newLimitedBuffer(limit int64) *limitedBuffer {
	return &limitedBuffer{remaining: limit, exceededSignal: make(chan struct{}, 1)}
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	original := len(value)
	if int64(len(value)) > b.remaining {
		value = value[:max(b.remaining, 0)]
		b.exceeded = true
		select {
		case b.exceededSignal <- struct{}{}:
		default:
		}
	}
	if len(value) != 0 {
		_, _ = b.buffer.Write(value)
		b.remaining -= int64(len(value))
	}
	return original, nil
}

func (b *limitedBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]byte(nil), b.buffer.Bytes()...)
}

func (b *limitedBuffer) Exceeded() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.exceeded
}
