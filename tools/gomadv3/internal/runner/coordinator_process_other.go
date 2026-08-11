//go:build !unix

package runner

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

func configureCoordinatorCommand(_ *exec.Cmd) {}

func terminateCoordinator(command *exec.Cmd, waited <-chan error, deadline time.Time) error {
	killErr := command.Process.Kill()
	if errors.Is(killErr, os.ErrProcessDone) {
		killErr = nil
	}
	timer := time.NewTimer(max(time.Until(deadline), 0))
	defer timer.Stop()
	select {
	case waitErr := <-waited:
		var exitError *exec.ExitError
		if errors.As(waitErr, &exitError) {
			waitErr = nil
		}
		return errors.Join(killErr, waitErr)
	case <-timer.C:
		return errors.Join(killErr, fmt.Errorf("coordinator could not be reaped before the overall deadline"))
	}
}
