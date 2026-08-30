//go:build !darwin && !linux

package runevaluation

import (
	"errors"
	"os"
	"os/exec"
)

func protectCheckerSnapshot(string) error {
	return errors.New("verified checker execution is unsupported on this platform")
}

func unprotectCheckerSnapshot(string) error {
	return nil
}

func verifyCheckerSnapshotPath(string, *os.File) error {
	return &checkerFailure{code: checkerFailureUnsafe}
}

func bindCheckerSnapshot(_ *exec.Cmd, _ *os.File) {
}
