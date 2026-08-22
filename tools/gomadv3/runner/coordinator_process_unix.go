//go:build unix

package runner

import (
	"errors"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"go.temporal.io/server/tools/gomadv3/internal/hostexec"
)

func configureCoordinatorCommand(command *exec.Cmd) {
	hostexec.ConfigureProcessGroup(command)
}

func terminateCoordinator(command *exec.Cmd, waited <-chan error, deadline time.Time) error {
	pgid := command.Process.Pid
	signalErr := hostexec.SignalGroup(pgid, syscall.SIGTERM)
	reaped := false
	var waitErr error
	var probeErr error
	groupGone := false
	graceDeadline := time.Now().Add(max(time.Until(deadline)/2, 0))
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	for time.Now().Before(graceDeadline) {
		exists, err := hostexec.GroupExists(pgid)
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
	killErr := hostexec.SignalGroup(pgid, syscall.SIGKILL)
	for (!reaped || !groupGone) && time.Now().Before(deadline) {
		if !groupGone {
			exists, err := hostexec.GroupExists(pgid)
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
