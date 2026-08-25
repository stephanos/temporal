package main

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCommandReportsUsageErrors(t *testing.T) {
	binaryName := "genleanmodeldescriptors"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(t.TempDir(), binaryName)
	build := exec.Command("go", "build", "-tags", "test_dep", "-o", binaryPath, ".")
	output, err := build.CombinedOutput()
	require.NoError(t, err, string(output))

	tests := map[string]struct {
		arguments []string
		contains  string
	}{
		"missing flags": {
			contains: "at least one --package-pattern is required",
		},
		"invalid flag": {
			arguments: []string{"--definitely-invalid"},
			contains:  "flag provided but not defined",
		},
		"positional argument": {
			arguments: []string{"extra"},
			contains:  "unexpected positional arguments",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			command := exec.Command(binaryPath, test.arguments...)
			var stderr bytes.Buffer
			command.Stderr = &stderr
			err := command.Run()
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			require.NotZero(t, exitErr.ExitCode())
			require.True(t, strings.HasPrefix(stderr.String(), "genleanmodeldescriptors: "), stderr.String())
			require.Contains(t, stderr.String(), test.contains)
		})
	}
}
