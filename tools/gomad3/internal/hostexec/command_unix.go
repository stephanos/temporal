//go:build unix

package hostexec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const cleanupLimit = 2 * time.Second

type captureResult struct {
	name string
	err  error
}

func Run(ctx context.Context, request Request) (result Result, retErr error) {
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}
	timeout := request.Timeout
	if !request.PreserveCommandError {
		var err error
		timeout, err = effectiveTimeout(ctx, request.Timeout)
		if err != nil {
			return Result{}, err
		}
	}
	stdout, err := New(request.OutputLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create stdout capture: %w", err)
	}
	stderr, err := New(request.OutputLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create stderr capture: %w", err)
	}
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create stdout pipe: %w", err)
	}
	defer func() {
		if closeErr := stdoutRead.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
	defer func() {
		if closeErr := stdoutWrite.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
	stderrRead, stderrWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create stderr pipe: %w", err)
	}
	defer func() {
		if closeErr := stderrRead.Close(); closeErr != nil {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()
	defer func() {
		if closeErr := stderrWrite.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			if retErr == nil {
				retErr = closeErr
			} else {
				retErr = errors.Join(retErr, closeErr)
			}
		}
	}()

	name := ""
	var arguments []string
	if len(request.Command) > 0 {
		name, arguments = request.Command[0], request.Command[1:]
	}
	var command *exec.Cmd
	callerCancellationDelivered := false
	if request.PreserveCommandError {
		command = exec.CommandContext(ctx, name, arguments...)
		cancel := command.Cancel
		command.Cancel = func() error {
			err := cancel()
			callerCancellationDelivered = err == nil
			return err
		}
	} else {
		command = exec.Command(name, arguments...)
	}
	command.Dir = request.Dir
	command.Env = append([]string(nil), request.Env...)
	command.Stdin = request.Stdin
	command.Stdout = stdoutWrite
	command.Stderr = stderrWrite
	if request.CombinedOutput {
		command.Stderr = stdoutWrite
	}
	if request.PreserveCommandError && command.Err == nil && name != "" && ctx.Err() == nil && request.Dir != "" {
		// Setpgid bypasses os.StartProcess's upstream directory preflight.
		if _, err := os.Stat(request.Dir); err != nil {
			var pathErr *os.PathError
			if errors.As(err, &pathErr) {
				pathErr.Op = "chdir"
			}
			return Result{CommandError: err}, nil
		}
	}
	ConfigureProcessGroup(command)
	if err := command.Start(); err != nil {
		if request.PreserveCommandError {
			return Result{CommandError: err}, nil
		}
		return Result{}, fmt.Errorf("start command %q: %w", request.Command[0], err)
	}
	pid := command.Process.Pid
	pgid := pid
	if err := errors.Join(stdoutWrite.Close(), stderrWrite.Close()); err != nil {
		return Result{}, errors.Join(fmt.Errorf("close inherited output pipe ends: %w", err), killAndReap(command, pgid, cleanupLimit))
	}

	captures := make(chan captureResult, 2)
	go copyOutput("stdout", stdout, stdoutRead, captures)
	go copyOutput("stderr", stderr, stderrRead, captures)
	waits := make(chan error, 1)
	go func() { waits <- command.Wait() }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	leaderReaped := false
	var waitErr error
	result = Result{PID: pid, PGID: pgid}
	select {
	case waitErr = <-waits:
		leaderReaped = true
	case <-timer.C:
		result.WatchdogTimeout = true
	case <-ctx.Done():
		result.Cancelled = true
	}
	if request.PreserveCommandError && !leaderReaped {
		killErr := command.Process.Kill()
		if killErr != nil && !errors.Is(killErr, os.ErrProcessDone) {
			return Result{}, errors.Join(killErr, killAndReapIfNeeded(command, waits, leaderReaped, pgid, cleanupLimit))
		}
		select {
		case waitErr = <-waits:
			leaderReaped = true
		case <-time.After(cleanupLimit):
			return Result{}, errors.Join(fmt.Errorf("command could not be reaped after leader kill"), KillGroupBounded(pgid, cleanupLimit))
		}
		if waitErr == nil || errors.Is(killErr, os.ErrProcessDone) {
			result.WatchdogTimeout = false
			result.Cancelled = false
		}
	}

	groupPresent, probeErr := GroupExists(pgid)
	if probeErr != nil {
		return Result{}, errors.Join(fmt.Errorf("probe command process group: %w", probeErr), killAndReapIfNeeded(command, waits, leaderReaped, pgid, cleanupLimit))
	}
	if groupPresent {
		terminationErr := terminateGroup(command, waits, &waitErr, &leaderReaped, pgid, request.TerminateGrace)
		if terminationErr != nil {
			return Result{}, terminationErr
		}
	} else if !leaderReaped {
		select {
		case waitErr = <-waits:
			leaderReaped = true
		case <-time.After(cleanupLimit):
			return Result{}, errors.Join(fmt.Errorf("command could not be reaped"), killAndReapIfNeeded(command, waits, leaderReaped, pgid, cleanupLimit))
		}
	}
	if !leaderReaped {
		return Result{}, fmt.Errorf("command was not reaped")
	}
	result.GroupGone, probeErr = GroupGone(pgid)
	if probeErr != nil || !result.GroupGone {
		return Result{}, errors.Join(probeErr, fmt.Errorf("command process group %d remains after cleanup", pgid))
	}

	if err := collectCaptures(captures); err != nil {
		return Result{}, err
	}
	result.Stdout = stdout.Result()
	result.Stderr = stderr.Result()
	classificationErr := waitErr
	if request.PreserveCommandError {
		var exit *exec.ExitError
		// Wait receives watchCtx's result before sending waits, synchronizing Cancel.
		callerOutcome := callerCancellationDelivered && waitErr == ctx.Err()
		if waitErr != nil && !errors.As(waitErr, &exit) && !callerOutcome {
			return result, waitErr
		}
		result.CommandError = waitErr
		classificationErr = nil
	}
	if err := classifyWait(command, classificationErr, &result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func copyOutput(name string, capture *Capture, reader io.Reader, results chan<- captureResult) {
	_, err := io.Copy(capture, reader)
	results <- captureResult{name: name, err: err}
}

func collectCaptures(captures <-chan captureResult) error {
	deadline := time.NewTimer(cleanupLimit)
	defer deadline.Stop()
	var result error
	for range 2 {
		select {
		case captured := <-captures:
			if captured.err != nil {
				result = errors.Join(result, fmt.Errorf("capture %s: %w", captured.name, captured.err))
			}
		case <-deadline.C:
			return errors.Join(result, fmt.Errorf("command output pipes remained open after cleanup"))
		}
	}
	return result
}

func terminateGroup(command *exec.Cmd, waits <-chan error, waitErr *error, leaderReaped *bool, pgid int, grace time.Duration) error {
	if err := SignalGroup(pgid, syscall.SIGTERM); err != nil {
		return errors.Join(fmt.Errorf("terminate command process group: %w", err), killAndReapIfNeeded(command, waits, *leaderReaped, pgid, cleanupLimit))
	}
	graceDeadline := time.Now().Add(grace)
	for time.Now().Before(graceDeadline) {
		if !*leaderReaped {
			select {
			case *waitErr = <-waits:
				*leaderReaped = true
			default:
			}
		}
		gone, err := GroupGone(pgid)
		if err != nil {
			return errors.Join(fmt.Errorf("probe command group during termination: %w", err), killAndReapIfNeeded(command, waits, *leaderReaped, pgid, cleanupLimit))
		}
		if gone {
			break
		}
		time.Sleep(min(5*time.Millisecond, time.Until(graceDeadline)))
	}
	groupPresent, err := GroupExists(pgid)
	if err != nil {
		return errors.Join(fmt.Errorf("probe command group before kill: %w", err), killAndReapIfNeeded(command, waits, *leaderReaped, pgid, cleanupLimit))
	}
	if groupPresent {
		if err := SignalGroup(pgid, syscall.SIGKILL); err != nil {
			return errors.Join(fmt.Errorf("kill command process group: %w", err), killAndReapIfNeeded(command, waits, *leaderReaped, pgid, cleanupLimit))
		}
	}
	deadline := time.Now().Add(cleanupLimit)
	for !*leaderReaped || groupPresent {
		if !*leaderReaped {
			select {
			case *waitErr = <-waits:
				*leaderReaped = true
			default:
			}
		}
		groupPresent, err = GroupExists(pgid)
		if err != nil {
			return fmt.Errorf("probe command group after kill: %w", err)
		}
		if *leaderReaped && !groupPresent {
			return nil
		}
		if !time.Now().Before(deadline) {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	var reapErr, groupErr error
	if !*leaderReaped {
		reapErr = fmt.Errorf("command could not be reaped after termination")
	}
	if groupPresent {
		groupErr = fmt.Errorf("command process group %d remains after termination", pgid)
	}
	return errors.Join(reapErr, groupErr)
}

func killAndReapIfNeeded(command *exec.Cmd, waits <-chan error, leaderReaped bool, pgid int, limit time.Duration) error {
	if leaderReaped {
		return KillGroupBounded(pgid, limit)
	}
	return killAndReap(command, pgid, limit)
}

func killAndReap(command *exec.Cmd, pgid int, limit time.Duration) error {
	signalErr := SignalGroup(pgid, syscall.SIGKILL)
	killErr := command.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	waitErr := command.Wait()
	var exitError *exec.ExitError
	if errors.As(waitErr, &exitError) {
		waitErr = nil
	}
	return errors.Join(signalErr, killErr, waitErr, KillGroupBounded(pgid, limit))
}

func KillGroupBounded(pgid int, limit time.Duration) error {
	if err := SignalGroup(pgid, syscall.SIGKILL); err != nil {
		return err
	}
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		gone, err := GroupGone(pgid)
		if err != nil || gone {
			return err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("command process group %d remains after cleanup", pgid)
}

func KillGroupBefore(pgid int, deadline time.Time) error {
	return KillGroupBounded(pgid, max(time.Until(deadline), 0))
}

func ConfigureProcessGroup(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func SignalGroup(pgid int, signal syscall.Signal) error {
	if pgid <= 0 {
		return fmt.Errorf("invalid process group %d", pgid)
	}
	return classifyGroupSignal(syscall.Kill(-pgid, signal))
}

// classifyGroupSignal accepts EPERM because darwin reports it for a group whose
// remaining members are all zombies awaiting their reaper. Every caller confirms
// the group is gone by probing afterwards, so a group that truly stays
// unsignalable still fails there.
func classifyGroupSignal(err error) error {
	if err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EPERM) {
		return err
	}
	return nil
}

func GroupExists(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, fmt.Errorf("invalid process group %d", pgid)
	}
	err := syscall.Kill(-pgid, 0)
	return ClassifyGroupProbe(err)
}

func ClassifyGroupProbe(err error) (bool, error) {
	switch {
	case err == nil, errors.Is(err, syscall.EPERM):
		return true, nil
	case errors.Is(err, syscall.ESRCH):
		return false, nil
	default:
		return false, err
	}
}

func GroupGone(pgid int) (bool, error) {
	exists, err := GroupExists(pgid)
	return !exists, err
}

func classifyWait(command *exec.Cmd, waitErr error, result *Result) error {
	if waitErr != nil {
		var exitError *exec.ExitError
		if !errors.As(waitErr, &exitError) {
			return fmt.Errorf("wait for command: %w", waitErr)
		}
	}
	if command.ProcessState == nil {
		return fmt.Errorf("command wait completed without process state")
	}
	status, ok := command.ProcessState.Sys().(syscall.WaitStatus)
	if !ok {
		return fmt.Errorf("command wait status has type %T", command.ProcessState.Sys())
	}
	if status.Signaled() {
		result.Termination = TerminationSignal
		result.Signal = status.Signal().String()
		result.SignalNumber = int(status.Signal())
		return nil
	}
	result.Termination = TerminationExit
	result.ExitCode = status.ExitStatus()
	return nil
}
