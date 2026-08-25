package main

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOutputRoot(t *testing.T) {
	t.Parallel()
	moduleRoot := t.TempDir()
	absoluteRoot := filepath.Join(t.TempDir(), "generated-model")

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "relative to module",
			args: []string{"--output-root", "model"},
			want: filepath.Join(moduleRoot, "model"),
		},
		{
			name: "absolute",
			args: []string{"--output-root", absoluteRoot},
			want: absoluteRoot,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseOutputRoot(test.args, moduleRoot)
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestParseOutputRootRejectsMissingOrUnexpectedArguments(t *testing.T) {
	t.Parallel()
	moduleRoot := t.TempDir()

	_, err := parseOutputRoot(nil, moduleRoot)
	require.EqualError(t, err, "--output-root is required")

	_, err = parseOutputRoot([]string{"--output-root", "model", "extra"}, moduleRoot)
	require.EqualError(t, err, "unexpected arguments")

	_, err = parseOutputRoot([]string{"--unknown"}, moduleRoot)
	require.ErrorContains(t, err, "parse arguments")
}
