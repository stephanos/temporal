//go:build !unix

package process

import (
	"os"
	"os/exec"
)

func configureProcessTree(_ *exec.Cmd) {}

func killProcessTree(command *exec.Cmd) error {
	if command.Process == nil {
		return os.ErrProcessDone
	}
	return command.Process.Kill()
}
