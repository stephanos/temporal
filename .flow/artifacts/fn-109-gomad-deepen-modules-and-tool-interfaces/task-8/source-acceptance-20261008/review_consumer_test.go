package analysis

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
)

func TestPreparedReviewSourceActualOwnerFailedClose(t *testing.T) {
	working := t.TempDir()
	for file, body := range map[string]string{"go.mod": "module example.com/sourceconsumer\n\ngo 1.27.1\n", "main.go": "package main\nfunc main() {}\n"} {
		if err := os.WriteFile(filepath.Join(working, file), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".toolchain"))
	if err != nil {
		t.Fatal(err)
	}
	primary := errors.New("SOURCE-remove-failed")
	calls := 0
	prepared, err := PrepareCapabilityReviewSourceControl(context.Background(), target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: working, ToolchainRoot: root}, func(path string) error {
		calls++
		if path == "" {
			t.Fatal("empty owned root")
		}
		return primary
	})
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Review.Schema != target.CapabilityReviewSchema || prepared.Spec.PreparationRoot == "" || prepared.Adapters == nil || prepared.BuildAdapters == nil {
		t.Fatalf("incomplete wrapper: %#v", prepared)
	}
	before := prepared.Review
	info, err := os.Stat(prepared.Spec.PreparationRoot)
	if err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("private kept root: %v %v", info, err)
	}
	if err := prepared.Close(); !errors.Is(err, primary) {
		t.Fatalf("failed Close identity: %v", err)
	}
	if err := prepared.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if calls != 1 || !reflect.DeepEqual(before, prepared.Review) {
		t.Fatalf("Close changed review/calls: %d", calls)
	}
	if _, err := os.Stat(prepared.Spec.PreparationRoot); err != nil {
		t.Fatalf("failed removal falsely deleted root: %v", err)
	}
	if err := os.RemoveAll(prepared.Spec.PreparationRoot); err != nil {
		t.Fatal(err)
	}
}
