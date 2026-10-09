package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/runner/internal/execution"
)

func TestCleanupMatchingReplayerPreservesDetachedResultAndCalls(t *testing.T) {
	published := publishInspectArtifact(t)
	replayer := &matchingReplayer{}
	result, err := replayer.Replay(t.Context(), ReplaySpec{ArtifactPath: published.Path})
	if err != nil {
		t.Fatal(err)
	}
	want := ReplayResult{Artifact: published, Verified: true, Match: true}
	want.Artifact.TargetSharing = ""
	if !reflect.DeepEqual(result, want) || replayer.calls != 1 {
		t.Fatalf("first replay = %#v, calls = %d, want %#v", result, replayer.calls, want)
	}
	result.Artifact.Manifest.Target.Argv[0] = "changed snapshot"
	result, err = replayer.Replay(t.Context(), ReplaySpec{ArtifactPath: published.Path})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, want) || replayer.calls != 2 {
		t.Fatalf("second replay = %#v, calls = %d, want %#v", result, replayer.calls, want)
	}
	result, err = replayer.Replay(t.Context(), ReplaySpec{ArtifactPath: filepath.Join(t.TempDir(), "missing")})
	if !errors.Is(err, os.ErrNotExist) || !reflect.DeepEqual(result, ReplayResult{}) || replayer.calls != 3 {
		t.Fatalf("missing replay = %#v, %v, calls = %d", result, err, replayer.calls)
	}
}

func TestCleanupMutatingExecutorPreservesPreparedTargetAndErrors(t *testing.T) {
	for _, name := range []string{"file", "missing", "directory"} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "target")
			data := []byte("prepared target")
			switch name {
			case "file":
				if err := os.WriteFile(path, data, 0o500); err != nil {
					t.Fatal(err)
				}
			case "directory":
				if err := os.Mkdir(path, 0o500); err != nil {
					t.Fatal(err)
				}
			case "missing":
			}
			result, err := (mutatingExecutor{}).Run(context.Background(), execution.Spec{Command: path})
			if name != "file" {
				var pathErr *os.PathError
				if !errors.As(err, &pathErr) || pathErr.Path != path || !reflect.DeepEqual(result, execution.Result{}) {
					t.Fatalf("mutation error = %#v, %v", result, err)
				}
				wantOp := "open"
				if name == "missing" {
					wantOp = "chmod"
					if !errors.Is(err, os.ErrNotExist) {
						t.Fatalf("missing target error = %v", err)
					}
				}
				if pathErr.Op != wantOp {
					t.Fatalf("mutation operation = %q, want %q", pathErr.Op, wantOp)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(result, processResult(1, "failure", "")) {
				t.Fatalf("mutation result = %#v, %v", result, err)
			}
			actual, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != "prepared targetmutation" {
				t.Fatalf("mutated bytes = %q", actual)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0o700 {
				t.Fatalf("mutated mode = %v", info.Mode())
			}
		})
	}
}
