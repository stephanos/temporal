package process

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type processAttempt struct {
	mu sync.Mutex

	command        *exec.Cmd
	output         *limitedBuffer
	done           chan struct{}
	memoryExceeded chan struct{}
	stopMemory     chan struct{}
	stopMemoryOnce sync.Once
	waitErr        error
	generation     int
	startedAt      int64
	stoppedAt      int64
	termination    Termination
}

func startProcessAttempt(request Request, generation int) (*processAttempt, error) {
	commandArguments, err := commandWithLimits(request.Command, request.Limits)
	if err != nil {
		return nil, err
	}
	command := exec.Command(commandArguments[0], commandArguments[1:]...)
	command.Env = append(os.Environ(), request.Environment...)
	command.Stdin = bytes.NewReader(request.Input)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	output := newLimitedBuffer(request.MaxOutputBytes)
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start worker: %w", err)
	}
	attempt := &processAttempt{
		command: command, output: output, done: make(chan struct{}), generation: generation,
		startedAt: time.Now().UnixNano(), termination: TerminationRunning,
	}
	if request.Limits.MemoryBytes > 0 {
		attempt.memoryExceeded = make(chan struct{}, 1)
		attempt.stopMemory = make(chan struct{})
		go watchProcessGroupMemory(command.Process.Pid, request.Limits.MemoryBytes,
			attempt.memoryExceeded, attempt.stopMemory)
	}
	go attempt.wait()
	return attempt, nil
}

func (a *processAttempt) wait() {
	err := a.command.Wait()
	a.stopMemoryWatch()
	a.mu.Lock()
	a.waitErr = err
	a.stoppedAt = time.Now().UnixNano()
	if a.termination == TerminationRunning {
		a.termination = TerminationExit
	}
	a.mu.Unlock()
	close(a.done)
}

func (a *processAttempt) stopMemoryWatch() {
	if a.stopMemory != nil {
		a.stopMemoryOnce.Do(func() { close(a.stopMemory) })
	}
}

func (a *processAttempt) kill(signal syscall.Signal, termination Termination) error {
	a.setTermination(termination)
	if err := syscall.Kill(-a.command.Process.Pid, signal); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func (a *processAttempt) running() bool {
	select {
	case <-a.done:
		return false
	default:
		return true
	}
}

func (a *processAttempt) setTermination(termination Termination) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.termination = termination
}

func (a *processAttempt) snapshot() Attempt {
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
