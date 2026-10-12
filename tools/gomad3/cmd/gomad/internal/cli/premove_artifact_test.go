package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The goldens are the output of `gomad inspect` built at e43c7a59c7, the
// parent of the backend move, with the artifact path replaced by ARTIFACT.
var preMoveArtifacts = []struct{ name, source string }{
	{"native", "../../../../../../.flow/artifacts/fn-113-gomad-reduce-version-pin-maintenance/task-4/core-qualification-artifacts/v1/campaign-20261003T095108.150283000Z-f19cf22b25bc9df20ac82dfdf302641b/successes/sha256-57dc45d96a58b85a4480d2b2c067b0ac"},
	{"wasm", "../../../../backends/wasm/backend/testdata/premove-stock-artifact"},
}

// stageRetainedArtifact copies a checked-in artifact with the private modes
// that OpenArtifact requires and version control does not keep.
func stageRetainedArtifact(t *testing.T, source string) string {
	t.Helper()
	destination := filepath.Join(t.TempDir(), "artifact")
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.Mkdir(target, 0o700)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := os.FileMode(0o600)
		if info.Mode()&0o100 != 0 {
			mode = 0o700
		}
		return os.WriteFile(target, data, mode)
	})
	if err != nil {
		t.Fatal(err)
	}
	return destination
}

func TestInspectPreMoveArtifactsMatchesPreMoveBinary(t *testing.T) {
	for _, fixture := range preMoveArtifacts {
		path := stageRetainedArtifact(t, fixture.source)
		for _, format := range []struct{ golden, flag string }{{".txt", ""}, {".json", "--json"}} {
			t.Run(fixture.name+format.golden, func(t *testing.T) {
				want, err := os.ReadFile(filepath.Join("testdata", "premove-inspect", fixture.name+format.golden))
				if err != nil {
					t.Fatal(err)
				}
				arguments := []string{"inspect", path}
				if format.flag != "" {
					arguments = []string{"inspect", format.flag, path}
				}
				var stdout, stderr bytes.Buffer
				status := Run(arguments, &stdout, &stderr)
				if got := strings.ReplaceAll(stdout.String(), path, "ARTIFACT"); status != 0 || stderr.Len() != 0 || got != string(want) {
					t.Fatalf("status=%d stderr=%q\ngot:\n%s\nwant:\n%s", status, stderr.String(), got, want)
				}
			})
		}
	}
}

func TestReplayPreMoveNativeArtifactKeepsPreMovePreflight(t *testing.T) {
	want, err := os.ReadFile(filepath.Join("testdata", "premove-inspect", "native-replay.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := Run([]string{"replay", "--toolchain-root=/must-not-resolve-historical-toolchain", stageRetainedArtifact(t, preMoveArtifacts[0].source)}, &stdout, &stderr)
	if status != 2 || stdout.Len() != 0 || stderr.String() != string(want) {
		t.Fatalf("status=%d stdout=%q stderr=%q; want status 2 and %q", status, stdout.String(), stderr.String(), want)
	}
}
