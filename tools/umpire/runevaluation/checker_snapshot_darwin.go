//go:build darwin

package runevaluation

import (
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

func protectCheckerSnapshot(_ string, file *os.File) error {
	// Darwin has no descriptor-based exec, so pin the exact open vnode immutable across launch.
	return unix.Fchflags(int(file.Fd()), unix.UF_IMMUTABLE)
}

func unprotectCheckerSnapshot(_ string, file *os.File) error {
	return unix.Fchflags(int(file.Fd()), 0)
}

func verifyCheckerSnapshotPath(path string, file *os.File) error {
	pathInfo, err := os.Lstat(path)
	if err != nil || !pathInfo.Mode().IsRegular() {
		return &checkerFailure{code: checkerFailureUnsafe}
	}
	openedInfo, err := file.Stat()
	if err != nil || !openedInfo.Mode().IsRegular() || !os.SameFile(pathInfo, openedInfo) {
		return &checkerFailure{code: checkerFailureUnsafe}
	}
	return nil
}

func bindCheckerSnapshot(_ *exec.Cmd, _ *os.File) {
}
