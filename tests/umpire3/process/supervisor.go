package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Termination string

const (
	TerminationRunning  Termination = "running"
	TerminationExit     Termination = "exit"
	TerminationCrash    Termination = "crash"
	TerminationStop     Termination = "stop"
	TerminationDeadline Termination = "deadline"
)

type Attempt struct {
	Generation        int         `json:"generation"`
	PID               int         `json:"pid"`
	StartedAtUnixNano int64       `json:"startedAtUnixNano"`
	StoppedAtUnixNano int64       `json:"stoppedAtUnixNano,omitempty"`
	ExitCode          int         `json:"exitCode"`
	Termination       Termination `json:"termination"`
	Output            []byte      `json:"output,omitempty"`
}

type supervisedAttempt struct {
	mu sync.Mutex

	command     *exec.Cmd
	output      *limitedBuffer
	done        chan struct{}
	waitErr     error
	generation  int
	startedAt   int64
	stoppedAt   int64
	termination Termination
}

type Supervisor struct {
	mu sync.Mutex

	request    Request
	generation int
	current    *supervisedAttempt
}

func NewSupervisor(request Request) (*Supervisor, error) {
	if len(request.Command) == 0 || request.Timeout <= 0 || request.MaxOutputBytes <= 0 {
		return nil, errors.New("supervised worker command, timeout, and output budget are required")
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
	command := exec.Command(s.request.Command[0], s.request.Command[1:]...)
	command.Env = append(os.Environ(), s.request.Environment...)
	command.Stdin = bytes.NewReader(s.request.Input)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	output := &limitedBuffer{remaining: s.request.MaxOutputBytes}
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		return Attempt{}, fmt.Errorf("start supervised worker: %w", err)
	}
	s.generation++
	attempt := &supervisedAttempt{
		command: command, output: output, done: make(chan struct{}), generation: s.generation,
		startedAt: time.Now().UnixNano(), termination: TerminationRunning,
	}
	s.current = attempt
	go attempt.wait()
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
	attempt.setTermination(TerminationCrash)
	if err := syscall.Kill(-attempt.command.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, os.ErrProcessDone) {
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
	attempt.setTermination(TerminationStop)
	if err := syscall.Kill(-attempt.command.Process.Pid, syscall.SIGTERM); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return attempt.snapshot(), fmt.Errorf("stop supervised worker: %w", err)
	}
	result, err := s.await(ctx, attempt)
	if !errors.Is(err, ErrDeadline) {
		return result, err
	}
	attempt.setTermination(TerminationDeadline)
	_ = syscall.Kill(-attempt.command.Process.Pid, syscall.SIGKILL)
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

func (s *Supervisor) runningAttempt() (*supervisedAttempt, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current == nil || !s.current.running() {
		return nil, errors.New("supervised worker is not running")
	}
	return s.current, nil
}

func (s *Supervisor) await(parent context.Context, attempt *supervisedAttempt) (Attempt, error) {
	ctx, cancel := context.WithTimeout(parent, s.request.Timeout)
	defer cancel()
	select {
	case <-attempt.done:
		return completedAttempt(attempt)
	case <-ctx.Done():
		return attempt.snapshot(), ErrDeadline
	}
}

func completedAttempt(attempt *supervisedAttempt) (Attempt, error) {
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

func (a *supervisedAttempt) wait() {
	err := a.command.Wait()
	a.mu.Lock()
	a.waitErr = err
	a.stoppedAt = time.Now().UnixNano()
	if a.termination == TerminationRunning {
		a.termination = TerminationExit
	}
	a.mu.Unlock()
	close(a.done)
}

func (a *supervisedAttempt) running() bool {
	select {
	case <-a.done:
		return false
	default:
		return true
	}
}

func (a *supervisedAttempt) setTermination(termination Termination) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.termination = termination
}

func (a *supervisedAttempt) snapshot() Attempt {
	a.mu.Lock()
	defer a.mu.Unlock()
	exitCode := -1
	if a.command.ProcessState != nil {
		exitCode = a.command.ProcessState.ExitCode()
	}
	return Attempt{
		Generation: a.generation, PID: a.command.Process.Pid,
		StartedAtUnixNano: a.startedAt, StoppedAtUnixNano: a.stoppedAt,
		ExitCode: exitCode, Termination: a.termination, Output: a.output.Bytes(),
	}
}
