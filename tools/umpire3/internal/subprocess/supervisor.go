package subprocess

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"syscall"
)

type Termination string

const (
	TerminationRunning  Termination = "running"
	TerminationExit     Termination = "exit"
	TerminationCrash    Termination = "crash"
	TerminationStop     Termination = "stop"
	TerminationDeadline Termination = "deadline"
	TerminationLimit    Termination = "limit"
)

type Attempt struct {
	Generation        int         `json:"generation"`
	PID               int         `json:"pid"`
	StartedAtUnixNano int64       `json:"startedAtUnixNano"`
	StoppedAtUnixNano int64       `json:"stoppedAtUnixNano,omitempty"`
	DurationNanos     int64       `json:"durationNanos,omitempty"`
	PeakMemoryBytes   int64       `json:"peakMemoryBytes,omitempty"`
	ExitCode          int         `json:"exitCode"`
	Termination       Termination `json:"termination"`
	Output            []byte      `json:"output,omitempty"`
}

type Supervisor struct {
	mu sync.Mutex

	request    Request
	generation int
	current    *processAttempt
}

func NewSupervisor(request Request) (*Supervisor, error) {
	if len(request.Command) == 0 || request.Timeout <= 0 || request.MaxOutputBytes <= 0 {
		return nil, errors.New("supervised worker command, timeout, and output budget are required")
	}
	if _, err := commandWithLimits(request.Command, request.Limits); err != nil {
		return nil, err
	}
	return &Supervisor{request: request}, nil
}

func (s *Supervisor) Start(ctx context.Context) (Attempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return Attempt{}, err
	}
	if s.current != nil && s.current.running() {
		return Attempt{}, errors.New("supervised worker is already running")
	}
	attempt, err := startProcessAttempt(s.request, s.generation+1)
	if err != nil {
		return Attempt{}, fmt.Errorf("start supervised worker: %w", err)
	}
	s.generation++
	s.current = attempt
	return attempt.snapshot(), nil
}

func (s *Supervisor) Restart(ctx context.Context) (Attempt, error) {
	s.mu.Lock()
	running := s.current != nil && s.current.running()
	s.mu.Unlock()
	if running {
		return Attempt{}, errors.New("cannot restart a supervised worker while it is still running")
	}
	return s.Start(ctx)
}

func (s *Supervisor) Crash(ctx context.Context) (Attempt, error) {
	attempt, err := s.runningAttempt()
	if err != nil {
		return Attempt{}, err
	}
	if err := attempt.kill(syscall.SIGKILL, TerminationCrash); err != nil {
		return attempt.snapshot(), fmt.Errorf("crash supervised worker: %w", err)
	}
	return s.await(ctx, attempt)
}

func (s *Supervisor) Stop(ctx context.Context) (Attempt, error) {
	s.mu.Lock()
	attempt := s.current
	s.mu.Unlock()
	if attempt == nil {
		return Attempt{}, nil
	}
	if !attempt.running() {
		return completedAttempt(attempt)
	}
	if err := attempt.kill(syscall.SIGTERM, TerminationStop); err != nil {
		return attempt.snapshot(), fmt.Errorf("stop supervised worker: %w", err)
	}
	result, err := s.await(ctx, attempt)
	if !errors.Is(err, ErrDeadline) {
		return result, err
	}
	_ = attempt.kill(syscall.SIGKILL, TerminationDeadline)
	return s.await(context.Background(), attempt)
}

func (s *Supervisor) Wait(ctx context.Context) (Attempt, error) {
	s.mu.Lock()
	attempt := s.current
	s.mu.Unlock()
	if attempt == nil {
		return Attempt{}, errors.New("supervised worker has not started")
	}
	return s.await(ctx, attempt)
}

func (s *Supervisor) Snapshot() Attempt {
	s.mu.Lock()
	attempt := s.current
	s.mu.Unlock()
	if attempt == nil {
		return Attempt{}
	}
	return attempt.snapshot()
}

func (s *Supervisor) runningAttempt() (*processAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || !s.current.running() {
		return nil, errors.New("supervised worker is not running")
	}
	return s.current, nil
}

func (s *Supervisor) await(parent context.Context, attempt *processAttempt) (Attempt, error) {
	ctx, cancel := context.WithTimeout(parent, s.request.Timeout)
	defer cancel()
	select {
	case <-attempt.done:
		return completedAttempt(attempt)
	case <-attempt.memoryExceeded:
		_ = attempt.kill(syscall.SIGKILL, TerminationLimit)
		<-attempt.done
		return attempt.snapshot(), ErrMemoryLimit
	case <-attempt.output.exceededSignal:
		_ = attempt.kill(syscall.SIGKILL, TerminationLimit)
		<-attempt.done
		return attempt.snapshot(), ErrOutputLimit
	case <-ctx.Done():
		return attempt.snapshot(), ErrDeadline
	}
}

func completedAttempt(attempt *processAttempt) (Attempt, error) {
	result := attempt.snapshot()
	if attempt.output.Exceeded() {
		return result, ErrOutputLimit
	}
	attempt.mu.Lock()
	waitErr := attempt.waitErr
	attempt.mu.Unlock()
	if waitErr != nil && result.Termination == TerminationExit {
		return result, fmt.Errorf("supervised worker exited with status %d: %w", result.ExitCode, waitErr)
	}
	return result, nil
}
