//go:build unix

package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func configureCoordinatorCommand(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateCoordinator(command *exec.Cmd, waited <-chan error, deadline time.Time) error {
	pgid := command.Process.Pid
	signalErr := syscall.Kill(-pgid, syscall.SIGTERM)
	if errors.Is(signalErr, os.ErrProcessDone) || errors.Is(signalErr, syscall.ESRCH) {
		signalErr = nil
	}
	reaped := false
	var waitErr error
	var probeErr error
	groupGone := false
	graceDeadline := time.Now().Add(max(time.Until(deadline)/2, 0))
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	for time.Now().Before(graceDeadline) {
		exists, err := coordinatorGroupExists(pgid)
		if err != nil {
			probeErr = errors.Join(probeErr, err)
			break
		}
		if !exists {
			groupGone = true
			break
		}
		select {
		case waitErr = <-waited:
			reaped = true
		case <-poll.C:
		}
	}
	killErr := syscall.Kill(-pgid, syscall.SIGKILL)
	if errors.Is(killErr, syscall.ESRCH) {
		killErr = nil
	}
	for (!reaped || !groupGone) && time.Now().Before(deadline) {
		if !groupGone {
			exists, err := coordinatorGroupExists(pgid)
			if err != nil {
				probeErr = errors.Join(probeErr, err)
			} else {
				groupGone = !exists
			}
		}
		select {
		case waitErr = <-waited:
			reaped = true
		case <-poll.C:
		}
	}
	var reapErr error
	if !reaped {
		reapErr = fmt.Errorf("coordinator could not be reaped before the overall deadline")
	}
	var groupErr error
	if !groupGone {
		groupErr = fmt.Errorf("coordinator process group %d remains after cleanup", pgid)
	}
	var exitError *exec.ExitError
	if errors.As(waitErr, &exitError) {
		waitErr = nil
	}
	return errors.Join(signalErr, killErr, waitErr, probeErr, reapErr, groupErr)
}

func coordinatorGroupExists(pgid int) (bool, error) {
	return classifyCoordinatorGroupProbe(syscall.Kill(-pgid, 0))
}

func classifyCoordinatorGroupProbe(err error) (bool, error) {
	switch {
	case err == nil, errors.Is(err, syscall.EPERM):
		return true, nil
	case errors.Is(err, syscall.ESRCH):
		return false, nil
	default:
		return false, err
	}
}
