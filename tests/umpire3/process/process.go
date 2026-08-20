package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var (
	ErrDeadline    = errors.New("worker execution deadline exceeded")
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
	commandArguments, err := commandWithLimits(request.Command, request.Limits)
	if err != nil {
		return Result{}, err
	}
	workerCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	command := exec.Command(commandArguments[0], commandArguments[1:]...)
	command.Env = append(os.Environ(), request.Environment...)
	command.Stdin = bytes.NewReader(request.Input)
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	output := &limitedBuffer{remaining: request.MaxOutputBytes}
	command.Stdout = output
	command.Stderr = output
	if err := command.Start(); err != nil {
		return Result{}, fmt.Errorf("start worker: %w", err)
	}

	waited := make(chan error, 1)
	go func() { waited <- command.Wait() }()
	select {
	case err := <-waited:
		result := Result{Output: output.Bytes(), ExitCode: command.ProcessState.ExitCode()}
		if output.exceeded {
			return result, ErrOutputLimit
		}
		if err != nil {
			return result, fmt.Errorf("worker exited with status %d: %w", result.ExitCode, err)
		}
		return result, nil
	case <-workerCtx.Done():
		_ = syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
		<-waited
		result := Result{Output: output.Bytes(), ExitCode: command.ProcessState.ExitCode(), TimedOut: true}
		if errors.Is(workerCtx.Err(), context.DeadlineExceeded) {
			return result, ErrDeadline
		}
		return result, workerCtx.Err()
	}
}

func commandWithLimits(command []string, limits Limits) ([]string, error) {
	if limits == (Limits{}) {
		return append([]string(nil), command...), nil
	}
	if limits.CPUSeconds <= 0 || limits.MemoryBytes <= 0 {
		return nil, errors.New("worker CPU and memory limits must both be positive")
	}
	memoryKiB := (limits.MemoryBytes + 1023) / 1024
	arguments := []string{
		"/bin/sh", "-c",
		"cpu_seconds=$1; memory_kib=$2; shift 2; " +
			"ulimit -t \"$cpu_seconds\"; ulimit -d \"$memory_kib\"; exec \"$@\"",
		"umpire3-worker", strconv.Itoa(limits.CPUSeconds), strconv.FormatInt(memoryKiB, 10),
	}
	return append(arguments, command...), nil
}

type limitedBuffer struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	remaining int64
	exceeded  bool
}

func (b *limitedBuffer) Write(value []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	original := len(value)
	if int64(len(value)) > b.remaining {
		value = value[:max(b.remaining, 0)]
		b.exceeded = true
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
