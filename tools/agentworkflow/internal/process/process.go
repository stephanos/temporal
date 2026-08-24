package process

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

var (
	ErrOutputLimit = errors.New("process output limit exceeded")
	ErrTimeout     = errors.New("process timed out")
	ErrCancelled   = errors.New("process cancelled")
)

type Request struct {
	Command        []string
	Directory      string
	Environment    []string
	Stdin          string
	Timeout        time.Duration
	MaxOutputBytes int64
}

type Result struct {
	Stdout    []byte
	Stderr    []byte
	ExitCode  int
	Duration  time.Duration
	Truncated bool
}

type Runner struct{}

func (Runner) Run(ctx context.Context, request Request) (Result, error) {
	if len(request.Command) == 0 || strings.TrimSpace(request.Command[0]) == "" {
		return Result{}, errors.New("process command is required")
	}
	if request.Directory == "" {
		return Result{}, errors.New("process directory is required")
	}
	if request.Timeout <= 0 {
		return Result{}, errors.New("process timeout must be positive")
	}
	if request.MaxOutputBytes <= 0 {
		return Result{}, errors.New("process output limit must be positive")
	}

	timeoutContext, timeoutCancel := context.WithTimeoutCause(ctx, request.Timeout, ErrTimeout)
	defer timeoutCancel()
	runContext, cancel := context.WithCancelCause(timeoutContext)
	defer cancel(nil)

	stdout := newLimitBuffer(request.MaxOutputBytes, func() { cancel(ErrOutputLimit) })
	stderr := newLimitBuffer(request.MaxOutputBytes, func() { cancel(ErrOutputLimit) })
	command := exec.CommandContext(runContext, request.Command[0], request.Command[1:]...)
	command.Dir = request.Directory
	command.Stdin = strings.NewReader(request.Stdin)
	command.Stdout = stdout
	command.Stderr = stderr
	command.Env = append([]string{}, request.Environment...)
	configureProcessTree(command)
	command.Cancel = func() error { return killProcessTree(command) }
	command.WaitDelay = 5 * time.Second

	startedAt := time.Now()
	err := command.Run()
	result := Result{
		Stdout:    stdout.Bytes(),
		Stderr:    stderr.Bytes(),
		ExitCode:  exitCode(err),
		Duration:  time.Since(startedAt),
		Truncated: stdout.Exceeded() || stderr.Exceeded(),
	}
	if result.Truncated || errors.Is(context.Cause(runContext), ErrOutputLimit) {
		return result, ErrOutputLimit
	}
	if errors.Is(context.Cause(timeoutContext), ErrTimeout) {
		return result, ErrTimeout
	}
	if ctx.Err() != nil {
		return result, ErrCancelled
	}
	if err == nil {
		return result, nil
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return result, nil
	}
	return result, fmt.Errorf("start or supervise process: %w", err)
}

func SelectEnvironment(names ...string) []string {
	result := make([]string, 0, len(names))
	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if name == "" || strings.ContainsRune(name, '=') {
			continue
		}
		if _, found := seen[name]; found {
			continue
		}
		seen[name] = struct{}{}
		if value, found := os.LookupEnv(name); found {
			result = append(result, name+"="+value)
		}
	}
	return result
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		return exitError.ExitCode()
	}
	return -1
}

type limitBuffer struct {
	mu       sync.Mutex
	buffer   bytes.Buffer
	limit    int64
	exceeded bool
	onLimit  func()
}

func newLimitBuffer(limit int64, onLimit func()) *limitBuffer {
	return &limitBuffer{limit: limit, onLimit: onLimit}
}

func (buffer *limitBuffer) Write(data []byte) (int, error) {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	written := len(data)
	remaining := buffer.limit - int64(buffer.buffer.Len())
	if remaining > 0 {
		keep := int64(len(data))
		if keep > remaining {
			keep = remaining
		}
		_, _ = buffer.buffer.Write(data[:keep])
	}
	if int64(len(data)) > remaining && !buffer.exceeded {
		buffer.exceeded = true
		buffer.onLimit()
	}
	return written, nil
}

func (buffer *limitBuffer) Bytes() []byte {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return append([]byte(nil), buffer.buffer.Bytes()...)
}

func (buffer *limitBuffer) Exceeded() bool {
	buffer.mu.Lock()
	defer buffer.mu.Unlock()
	return buffer.exceeded
}
