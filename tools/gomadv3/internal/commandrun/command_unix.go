//go:build unix

package commandrun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/outputcapture"
)

const cleanupLimit = 2 * time.Second

type captureResult struct {
	name string
	err  error
}

func Run(ctx context.Context, request Request) (Result, error) {
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}
	timeout, err := effectiveTimeout(ctx, request.Timeout)
	if err != nil {
		return Result{}, err
	}
	stdout, err := outputcapture.New(request.OutputLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create stdout capture: %w", err)
	}
	stderr, err := outputcapture.New(request.OutputLimit)
	if err != nil {
		return Result{}, fmt.Errorf("create stderr capture: %w", err)
	}
	stdoutRead, stdoutWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create stdout pipe: %w", err)
	}
	defer stdoutRead.Close()
	defer stdoutWrite.Close()
	stderrRead, stderrWrite, err := os.Pipe()
	if err != nil {
		return Result{}, fmt.Errorf("create stderr pipe: %w", err)
	}
	defer stderrRead.Close()
	defer stderrWrite.Close()

	command := exec.Command(request.Command[0], request.Command[1:]...)
	command.Dir = request.Dir
	command.Env = append([]string(nil), request.Env...)
	command.Stdin = request.Stdin
	command.Stdout = stdoutWrite
	command.Stderr = stderrWrite
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
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
	result := Result{PID: pid, PGID: pgid}
	select {
	case waitErr = <-waits:
		leaderReaped = true
	case <-timer.C:
		result.WatchdogTimeout = true
	case <-ctx.Done():
		result.Cancelled = true
	}

	groupPresent, probeErr := groupExists(pgid)
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
	result.GroupGone, probeErr = groupGone(pgid)
	if probeErr != nil || !result.GroupGone {
		return Result{}, errors.Join(probeErr, fmt.Errorf("command process group %d remains after cleanup", pgid))
	}

	if err := collectCaptures(captures); err != nil {
		return Result{}, err
	}
	result.Stdout = stdout.Result()
	result.Stderr = stderr.Result()
	if err := classifyWait(command, waitErr, &result); err != nil {
		return Result{}, err
	}
	return result, nil
}

func copyOutput(name string, capture *outputcapture.Capture, reader io.Reader, results chan<- captureResult) {
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
	if err := signalGroup(pgid, syscall.SIGTERM); err != nil {
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
		gone, err := groupGone(pgid)
		if err != nil {
			return errors.Join(fmt.Errorf("probe command group during termination: %w", err), killAndReapIfNeeded(command, waits, *leaderReaped, pgid, cleanupLimit))
		}
		if gone {
			break
		}
		time.Sleep(min(5*time.Millisecond, time.Until(graceDeadline)))
	}
	groupPresent, err := groupExists(pgid)
	if err != nil {
		return errors.Join(fmt.Errorf("probe command group before kill: %w", err), killAndReapIfNeeded(command, waits, *leaderReaped, pgid, cleanupLimit))
	}
	if groupPresent {
		if err := signalGroup(pgid, syscall.SIGKILL); err != nil {
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
		groupPresent, err = groupExists(pgid)
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
		return killGroupBounded(pgid, limit)
	}
	return killAndReap(command, pgid, limit)
}

func killAndReap(command *exec.Cmd, pgid int, limit time.Duration) error {
	signalErr := signalGroup(pgid, syscall.SIGKILL)
	killErr := command.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	waitErr := command.Wait()
	var exitError *exec.ExitError
	if errors.As(waitErr, &exitError) {
		waitErr = nil
	}
	return errors.Join(signalErr, killErr, waitErr, killGroupBounded(pgid, limit))
}

func killGroupBounded(pgid int, limit time.Duration) error {
	if err := signalGroup(pgid, syscall.SIGKILL); err != nil {
		return err
	}
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		gone, err := groupGone(pgid)
		if err != nil || gone {
			return err
		}
		time.Sleep(5 * time.Millisecond)
	}
	return fmt.Errorf("command process group %d remains after cleanup", pgid)
}

func signalGroup(pgid int, signal syscall.Signal) error {
	if pgid <= 0 {
		return fmt.Errorf("invalid process group %d", pgid)
	}
	if err := syscall.Kill(-pgid, signal); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

func groupExists(pgid int) (bool, error) {
	if pgid <= 0 {
		return false, fmt.Errorf("invalid process group %d", pgid)
	}
	err := syscall.Kill(-pgid, 0)
	switch {
	case err == nil, errors.Is(err, syscall.EPERM):
		return true, nil
	case errors.Is(err, syscall.ESRCH):
		return false, nil
	default:
		return false, err
	}
}

func groupGone(pgid int) (bool, error) {
	exists, err := groupExists(pgid)
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
