package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const (
	fixturePackage       = "go.temporal.io/server/tools/umpire/cmd/umpire-gen-model-descriptors/testdata/godescriptors"
	brokenFixturePackage = "go.temporal.io/server/tools/umpire/cmd/umpire-gen-model-descriptors/testdata/godescriptorsbroken"
)

func TestCommandReportsUsageErrors(t *testing.T) {
	binaryName := "umpire-gen-model-descriptors"
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
			require.True(t, strings.HasPrefix(stderr.String(), "umpire-gen-model-descriptors: "), stderr.String())
			require.Contains(t, stderr.String(), test.contains)
		})
	}
}

func TestRunExportsMatchingDescriptorsAndTransitiveImportsDeterministically(t *testing.T) {
	firstPath := filepath.Join(t.TempDir(), "first.pb")
	arguments := []string{
		"--package-pattern", fixturePackage,
		"--package-pattern", fixturePackage,
		"--file-prefix", "fixture/public/",
		"--output", firstPath,
	}
	require.NoError(t, Run(context.Background(), arguments))
	first := readDescriptorSet(t, firstPath)
	require.Equal(t, []string{"fixture/dependency.proto", "fixture/public/model.proto"}, descriptorPaths(first))

	secondPath := filepath.Join(t.TempDir(), "second.pb")
	arguments[len(arguments)-1] = secondPath
	require.NoError(t, Run(context.Background(), arguments))
	firstBytes, err := os.ReadFile(firstPath)
	require.NoError(t, err)
	secondBytes, err := os.ReadFile(secondPath)
	require.NoError(t, err)
	require.Equal(t, firstBytes, secondBytes)
}

func TestRunValidatesFlagsAndReportsEmptySelections(t *testing.T) {
	valid := []string{
		"--package-pattern", fixturePackage,
		"--file-prefix", "fixture/public/",
		"--output", filepath.Join(t.TempDir(), "output.pb"),
	}
	tests := map[string]struct {
		arguments []string
		contains  string
	}{
		"packages":   {removePair(valid, "--package-pattern"), "at least one --package-pattern"},
		"prefixes":   {removePair(valid, "--file-prefix"), "at least one --file-prefix"},
		"output":     {removePair(valid, "--output"), "--output is required"},
		"positional": {append(valid, "extra"), "unexpected positional"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, Run(context.Background(), test.arguments), test.contains)
		})
	}
}

func TestRunFailuresDoNotReplaceExistingOutput(t *testing.T) {
	tests := map[string]struct {
		arguments func(string) []string
		context   func() context.Context
		contains  string
	}{
		"package list": {
			arguments: func(output string) []string {
				return validArguments("./definitely-missing-package", "fixture/public/", output)
			},
			contains: "go list package pattern",
		},
		"empty selection": {
			arguments: func(output string) []string {
				return validArguments(fixturePackage, "missing/", output)
			},
			contains: "no registered protobuf descriptors matched",
		},
		"cancellation": {
			arguments: func(output string) []string {
				return validArguments(fixturePackage, "fixture/public/", output)
			},
			context:  canceledContext,
			contains: "go list package pattern",
		},
		"descriptor helper": {
			arguments: func(output string) []string {
				return validArguments(brokenFixturePackage, "fixture/broken/", output)
			},
			contains: "run descriptor helper",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "output.pb")
			existing := []byte("existing descriptor set")
			require.NoError(t, os.WriteFile(output, existing, 0o600))
			ctx := context.Background()
			if test.context != nil {
				ctx = test.context()
			}
			err := Run(ctx, test.arguments(output))
			require.ErrorContains(t, err, test.contains)
			actual, readErr := os.ReadFile(output)
			require.NoError(t, readErr)
			require.Equal(t, existing, actual)
		})
	}
}

func TestExportDescriptorsJoinsCleanupError(t *testing.T) {
	cleanupErr := errors.New("controlled cleanup failure")
	original := removeDescriptorHelper
	removeDescriptorHelper = func(path string) error {
		require.NoError(t, os.RemoveAll(path))
		return cleanupErr
	}
	t.Cleanup(func() {
		removeDescriptorHelper = original
	})

	_, err := exportDescriptors(
		context.Background(),
		[]string{brokenFixturePackage},
		[]string{"fixture/broken/"},
	)
	require.ErrorContains(t, err, "run descriptor helper")
	require.ErrorIs(t, err, cleanupErr)
}

func TestListDescriptorPackagesFiltersCompatibilityCopiesByProtobufPrefix(t *testing.T) {
	packages, err := listDescriptorPackages(
		context.Background(),
		[]string{
			fixturePackage,
			"go.temporal.io/server/tools/umpire/cmd/umpire-gen-model-descriptors/testdata/godescriptorscompat",
		},
		[]string{"fixture/public/"},
	)
	require.NoError(t, err)
	require.Equal(t, []string{fixturePackage}, packages)
}

func readDescriptorSet(t *testing.T, path string) *descriptorpb.FileDescriptorSet {
	t.Helper()
	encoded, err := os.ReadFile(path)
	require.NoError(t, err)
	set := &descriptorpb.FileDescriptorSet{}
	require.NoError(t, proto.Unmarshal(encoded, set))
	return set
}

func descriptorPaths(set *descriptorpb.FileDescriptorSet) []string {
	paths := make([]string, len(set.File))
	for index, file := range set.File {
		paths[index] = file.GetName()
	}
	return paths
}

func removePair(arguments []string, name string) []string {
	result := make([]string, 0, len(arguments)-2)
	for index := 0; index < len(arguments); index++ {
		if arguments[index] == name {
			index++
			continue
		}
		result = append(result, arguments[index])
	}
	return result
}

func validArguments(packagePattern, filePrefix, output string) []string {
	return []string{
		"--package-pattern", packagePattern,
		"--file-prefix", filePrefix,
		"--output", output,
	}
}

func canceledContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}
