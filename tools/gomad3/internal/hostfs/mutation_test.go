package hostfs

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestReadBoundedOpenedFileMutationPreservesBoundsAndErrors(t *testing.T) {
	for _, test := range []struct {
		name, contents, want string
		maximum              uint64
		closed, wantError    bool
	}{
		{"unchanged", "four", "four", 4, false, false},
		{"shrink", "fo", "fo", 4, false, false},
		{"grow-at-bound", "fourxx", "fourxx", 6, false, false},
		{"grow-over-bound", "fourxxx", "", 6, false, true},
		{"read-error", "four", "", 4, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "file")
			if err := os.WriteFile(path, []byte("four"), 0o600); err != nil {
				t.Fatal(err)
			}
			file, info, err := OpenPath(path)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := file.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
					t.Error(err)
				}
			})
			if err := os.WriteFile(path, []byte(test.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			if test.closed {
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
			}
			data, err := readBounded(file, path, info.Size(), test.maximum)
			if (err != nil) != test.wantError || string(data) != test.want {
				t.Fatalf("opened file read = %q, %v", data, err)
			}
			if test.closed && !errors.Is(err, os.ErrClosed) {
				t.Fatalf("read error lost cause: %v", err)
			}
		})
	}
	if _, err := readBounded(nil, "file", 0, math.MaxUint64); err == nil {
		t.Fatal("overflowing sentinel bound accepted")
	}
}
