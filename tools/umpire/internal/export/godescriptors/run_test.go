package godescriptors

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
)

const fixturePackage = "go.temporal.io/server/tools/umpire/testdata/godescriptors"

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
		"no match":   {replacePair(valid, "--file-prefix", "missing/"), "no registered protobuf descriptors matched"},
		"go list":    {replacePair(valid, "--package-pattern", "./definitely-missing-package"), "go list package pattern"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.ErrorContains(t, Run(context.Background(), test.arguments), test.contains)
		})
	}
}

func TestListDescriptorPackagesFiltersCompatibilityCopiesByProtobufPrefix(t *testing.T) {
	packages, err := listDescriptorPackages(
		context.Background(),
		[]string{
			fixturePackage,
			"go.temporal.io/server/tools/umpire/testdata/godescriptorscompat",
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

func replacePair(arguments []string, name, value string) []string {
	result := append([]string(nil), arguments...)
	for index := range result {
		if result[index] == name {
			result[index+1] = value
			return result
		}
	}
	return result
}
