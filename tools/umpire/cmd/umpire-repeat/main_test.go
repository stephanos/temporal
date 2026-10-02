package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelRuntimeInputsInvalidateRepeatRecords(t *testing.T) {
	for _, directory := range []string{"model/ir", "model/cases"} {
		t.Run(directory, func(t *testing.T) {
			root := t.TempDir()
			initialize := exec.CommandContext(t.Context(), "git", "init", "--quiet", root)
			output, err := initialize.CombinedOutput()
			require.NoError(t, err, "%s", output)
			relative := directory + "/input.json"
			input := filepath.Join(root, filepath.FromSlash(relative))
			require.NoError(t, os.MkdirAll(filepath.Dir(input), 0755))
			require.NoError(t, os.WriteFile(input, []byte(`{"version":1}`), 0644))
			before, err := fingerprintTree(root)
			require.NoError(t, err)
			require.Contains(t, before.Paths, relative)
			require.NoError(t, os.WriteFile(input, []byte(`{"version":2}`), 0644))
			after, err := fingerprintTree(root)
			require.NoError(t, err)
			require.NotEqual(t, before.Digest, after.Digest)
			require.Equal(t, relative, before.changed(after))

			records := t.TempDir()
			first, second := filepath.Join(records, "before.jsonl"), filepath.Join(records, "after.jsonl")
			writeRecords(t, first, sampleRecord(1, before.Digest))
			writeRecords(t, second, sampleRecord(1, after.Digest))
			world := newFakeWorld(t)
			world.env.fingerprint = func() (fingerprint, error) { return fingerprintTree(root) }
			var stdout, stderr bytes.Buffer
			require.Equal(t, exitToolingError, Run(context.Background(), runArguments(first, 1, modeProcess), &stdout, &stderr, world.env))
			require.Contains(t, stderr.String(), "use a new record file")
			require.Empty(t, world.invocations)
			require.Len(t, readRecordFile(t, first), 1)

			stdout.Reset()
			stderr.Reset()
			require.Equal(t, exitToolingError, Run(context.Background(), []string{"summarize", first, second}, &stdout, &stderr, environment{}))
			require.Contains(t, stderr.String(), "has fingerprint "+after.Digest)
			require.Empty(t, stdout.String())
		})
	}
}
