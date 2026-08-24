package api

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGenerationConfig(t *testing.T) {
	configuration, err := parseGenerationConfig([]string{
		"--descriptor", "fixtures/second.pb",
		"--lean-root", "Acme.Model",
		"--descriptor", "fixtures/first.pb",
		"--output-root", "generated",
	})
	require.NoError(t, err)
	require.Equal(t, []descriptorSpec{
		{Locator: "fixtures/first.pb", Path: filepath.FromSlash("fixtures/first.pb")},
		{Locator: "fixtures/second.pb", Path: filepath.FromSlash("fixtures/second.pb")},
	}, configuration.Descriptors)
	require.Equal(t, "generated", configuration.OutputRoot)
	require.Equal(t, "Acme.Model", configuration.Layout.RootModule)
	require.Equal(t, "Acme/Model/API.lean", configuration.Layout.APIPath)
	require.Equal(t, "Acme/Model/API", configuration.Layout.APIDirectory)
	require.Equal(t, "Acme/Model/API/Proto.lean", configuration.Layout.ProtoPath)
	require.Equal(t, "Acme/Model/API/Types.lean", configuration.Layout.TypesPath)
}

func TestParseGenerationConfigValidation(t *testing.T) {
	valid := []string{
		"--descriptor", "fixture.pb",
		"--lean-root", "Fixture",
		"--output-root", "generated",
	}
	tests := map[string]struct {
		arguments []string
		contains  string
	}{
		"empty descriptor path":  {replaceFlagValue(valid, "--descriptor", ""), "descriptor path"},
		"duplicate descriptor":   {append(valid, "--descriptor", "./fixture.pb"), "duplicate descriptor locator"},
		"missing descriptor":     {removeFlag(valid, "--descriptor"), "at least one --descriptor"},
		"missing root":           {removeFlag(valid, "--lean-root"), "--lean-root is required"},
		"invalid root":           {replaceFlagValue(valid, "--lean-root", "Acme.bad-segment"), "Lean root"},
		"anonymous root":         {replaceFlagValue(valid, "--lean-root", "_"), "Lean root"},
		"type root":              {replaceFlagValue(valid, "--lean-root", "Type"), "Lean root"},
		"prop root":              {replaceFlagValue(valid, "--lean-root", "Prop"), "Lean root"},
		"sort root":              {replaceFlagValue(valid, "--lean-root", "Sort"), "Lean root"},
		"missing output":         {removeFlag(valid, "--output-root"), "--output-root is required"},
		"operation word":         {append([]string{"generate"}, valid...), "unexpected positional"},
		"empty dotted root part": {replaceFlagValue(valid, "--lean-root", "Acme..Model"), "Lean root"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseGenerationConfig(test.arguments)
			require.ErrorContains(t, err, test.contains)
		})
	}
}

func replaceFlagValue(arguments []string, name, value string) []string {
	result := append([]string(nil), arguments...)
	for index := range result {
		if result[index] == name {
			result[index+1] = value
			return result
		}
	}
	return result
}

func removeFlag(arguments []string, name string) []string {
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
