package api

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGenerationConfig(t *testing.T) {
	configuration, err := parseGenerationConfig([]string{
		"generate",
		"--source", "Internal=internal/",
		"--descriptor", "second=fixtures/second.pb",
		"--lean-root", "Acme.Model",
		"--default-source", "External",
		"--descriptor", "first=fixtures/first.pb",
		"--source", "Public=public/",
		"--source", "Internal=internal/private/",
		"--output-root", "generated",
	})
	require.NoError(t, err)
	require.Equal(t, "generate", configuration.Operation)
	require.Equal(t, []descriptorSpec{
		{Name: "first", Locator: "fixtures/first.pb", Path: filepath.FromSlash("fixtures/first.pb")},
		{Name: "second", Locator: "fixtures/second.pb", Path: filepath.FromSlash("fixtures/second.pb")},
	}, configuration.Descriptors)
	require.Equal(t, []sourceGroup{"External", "Internal", "Public"}, configuration.Groups)
	require.Equal(t, []sourceRule{
		{Group: "Internal", Prefix: "internal/private/"},
		{Group: "Internal", Prefix: "internal/"},
		{Group: "Public", Prefix: "public/"},
	}, configuration.Sources)
	require.Equal(t, sourceGroup("External"), configuration.DefaultSource)
	require.Equal(t, sourceGroup("Internal"), configuration.Classify("internal/private/request.proto"))
	require.Equal(t, sourceGroup("External"), configuration.Classify("shared/model.proto"))
	require.Equal(t, "Acme.Model", configuration.Layout.RootModule)
	require.Equal(t, "Acme/Model/Proto/Core.lean", configuration.Layout.CorePath)
	require.Equal(t, "Acme/Model/Generated.lean", configuration.Layout.UmbrellaPath)
	require.Equal(t, "Acme/Model/Generated/manifest.json", configuration.Layout.ManifestPath)
}

func TestParseGenerationConfigValidation(t *testing.T) {
	valid := []string{
		"generate",
		"--descriptor", "input=fixture.pb",
		"--source", "Public=public/",
		"--default-source", "External",
		"--lean-root", "Fixture",
		"--output-root", "generated",
	}
	tests := map[string]struct {
		arguments []string
		contains  string
	}{
		"operation":              {valid[1:], "operation is required"},
		"unknown operation":      {append([]string{"render"}, valid[1:]...), "unknown operation"},
		"descriptor value":       {replaceFlagValue(valid, "--descriptor", "missing-separator"), "NAME=PATH"},
		"empty descriptor name":  {replaceFlagValue(valid, "--descriptor", "=fixture.pb"), "descriptor name"},
		"empty descriptor path":  {replaceFlagValue(valid, "--descriptor", "input="), "descriptor path"},
		"duplicate descriptor":   {append(valid, "--descriptor", "input=other.pb"), "duplicate descriptor name"},
		"missing descriptor":     {removeFlag(valid, "--descriptor"), "at least one --descriptor"},
		"source value":           {replaceFlagValue(valid, "--source", "Public"), "GROUP=PREFIX"},
		"empty source prefix":    {replaceFlagValue(valid, "--source", "Public="), "source prefix"},
		"invalid group":          {replaceFlagValue(valid, "--source", "bad-group=public/"), "source group"},
		"conflicting prefix":     {append(valid, "--source", "Internal=public/"), "assigned to both"},
		"missing default":        {removeFlag(valid, "--default-source"), "--default-source is required"},
		"invalid default":        {replaceFlagValue(valid, "--default-source", "bad-group"), "default source"},
		"missing root":           {removeFlag(valid, "--lean-root"), "--lean-root is required"},
		"invalid root":           {replaceFlagValue(valid, "--lean-root", "Acme.bad-segment"), "Lean root"},
		"anonymous root":         {replaceFlagValue(valid, "--lean-root", "_"), "Lean root"},
		"type root":              {replaceFlagValue(valid, "--lean-root", "Type"), "Lean root"},
		"prop root":              {replaceFlagValue(valid, "--lean-root", "Prop"), "Lean root"},
		"sort root":              {replaceFlagValue(valid, "--lean-root", "Sort"), "Lean root"},
		"missing output":         {removeFlag(valid, "--output-root"), "--output-root is required"},
		"positional":             {append(valid, "extra"), "unexpected positional"},
		"duplicate same rule":    {append(valid, "--source", "Public=public/"), "duplicate source rule"},
		"empty dotted root part": {replaceFlagValue(valid, "--lean-root", "Acme..Model"), "Lean root"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := parseGenerationConfig(test.arguments)
			require.ErrorContains(t, err, test.contains)
		})
	}
}

func TestInspectDoesNotRequireOutputRoot(t *testing.T) {
	configuration, err := parseGenerationConfig([]string{
		"inspect",
		"--descriptor", "input=fixture.pb",
		"--default-source", "External",
		"--lean-root", "Fixture",
	})
	require.NoError(t, err)
	require.Empty(t, configuration.OutputRoot)
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
