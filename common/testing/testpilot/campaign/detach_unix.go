//go:build unix

package campaign

import (
	"os/exec"
	"syscall"
)

// detach puts the bridge in a process group of its own, so a terminal's SIGINT reaches the
// coordinator and not the bridge: the bridge's summary is still the bridge's to give after a stop.
func detach(command *exec.Cmd) {
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
