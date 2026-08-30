//go:build linux

package runevaluation

import (
	"os"
	"os/exec"
)

func protectCheckerSnapshot(string) error {
	return nil
}

func unprotectCheckerSnapshot(string) error {
	return nil
}

func verifyCheckerSnapshotPath(string, *os.File) error {
	return nil
}

func bindCheckerSnapshot(command *exec.Cmd, file *os.File) {
	command.Path = "/proc/self/fd/3"
	command.ExtraFiles = []*os.File{file}
}
