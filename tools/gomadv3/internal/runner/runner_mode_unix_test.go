//go:build unix

package runner

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestRunEnforcesBatchModesIndependentOfUmask(t *testing.T) {
	config := testConfig(t, newFakePreparer(t), &fakeExecutor{}, "1", PolicyAll, 1)
	oldUmask := syscall.Umask(0o777)
	defer syscall.Umask(oldUmask)

	summary, err := Run(context.Background(), config)
	if err != nil {
		t.Fatal(err)
	}
	for path, mode := range map[string]os.FileMode{
		".":          0o700,
		"failures":   0o700,
		".partial":   0o700,
		"batch.json": 0o600,
		"runs.jsonl": 0o600,
	} {
		info, statErr := os.Stat(filepath.Join(summary.BatchPath, path))
		if statErr != nil {
			t.Fatal(statErr)
		}
		if info.Mode().Perm() != mode {
			t.Fatalf("%s mode = %#o, want %#o", path, info.Mode().Perm(), mode)
		}
	}
}
