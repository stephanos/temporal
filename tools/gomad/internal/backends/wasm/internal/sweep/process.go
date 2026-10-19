package sweep

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"syscall"
	"time"
)

// Command is one subprocess invocation.
type Command struct {
	Path string
	Args []string
	Dir  string
	Env  []string
	// Timeout bounds the whole process; zero is refused by Exec.
	Timeout time.Duration
}

// Completed is a finished subprocess. Exit is -1 when the process did not
// exit on its own (start failure, timeout or signal).
type Completed struct {
	Stdout, Stderr []byte
	Exit           int
	Err            error
	Elapsed        time.Duration
}

// Executor runs subprocesses; tests substitute a fake.
type Executor interface {
	Exec(ctx context.Context, command Command) Completed
}

// HostExecutor runs real subprocesses with bounded captured output.
type HostExecutor struct{}

const maximumCapturedOutput = 64 << 20

type boundedBuffer struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	room := buffer.limit - buffer.Len()
	if room <= 0 {
		buffer.truncated = true
		return len(data), nil
	}
	if len(data) > room {
		buffer.truncated = true
		buffer.Buffer.Write(data[:room])
		return len(data), nil
	}
	return buffer.Buffer.Write(data)
}

func (HostExecutor) Exec(ctx context.Context, command Command) Completed {
	if command.Timeout <= 0 {
		return Completed{Exit: -1, Err: errors.New("subprocess timeout is required")}
	}
	ctx, cancel := context.WithTimeout(ctx, command.Timeout)
	defer cancel()
	process := exec.CommandContext(ctx, command.Path, command.Args...)
	process.Dir, process.Env = command.Dir, command.Env
	// gomad starts the Go compiler and the WASM helper; a timeout or an
	// interrupted sweep must stop the whole group, not only gomad.
	process.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	process.Cancel = func() error { return syscall.Kill(-process.Process.Pid, syscall.SIGKILL) }
	process.WaitDelay = 30 * time.Second
	stdout := &boundedBuffer{limit: maximumCapturedOutput}
	stderr := &boundedBuffer{limit: maximumCapturedOutput}
	process.Stdout, process.Stderr = stdout, stderr
	started := time.Now()
	err := process.Run()
	completed := Completed{Stdout: stdout.Bytes(), Stderr: stderr.Bytes(), Exit: -1, Err: err, Elapsed: time.Since(started)}
	if exit, ok := errors.AsType[*exec.ExitError](err); ok && exit.Exited() {
		completed.Exit, completed.Err = exit.ExitCode(), nil
	} else if err == nil {
		completed.Exit = 0
	}
	if ctx.Err() != nil && completed.Err == nil && completed.Exit != 0 {
		completed.Err = ctx.Err()
	}
	if stdout.truncated || stderr.truncated {
		completed.Err = errors.Join(completed.Err, errors.New("subprocess output exceeded the capture bound"))
	}
	return completed
}
