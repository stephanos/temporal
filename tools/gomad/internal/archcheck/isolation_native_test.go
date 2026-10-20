//go:build gomad_native

package archcheck_test

import (
	"bytes"
	"os"
	"testing"

	"golang.org/x/mod/modfile"
)

func TestNativeModulePinsMatchParent(t *testing.T) {
	parentBytes, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	childBytes, err := os.ReadFile("internal/backends/native/go.mod")
	if err != nil {
		t.Fatal(err)
	}
	parent, err := modfile.Parse("go.mod", parentBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	child, err := modfile.Parse("native/go.mod", childBytes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if parent.Go.Version != child.Go.Version {
		t.Fatalf("native Go pin %s differs from parent %s", child.Go.Version, parent.Go.Version)
	}
	pins := map[string]string{}
	for _, required := range parent.Require {
		if required.Mod.Path != modulePath+"/internal/backends/native" {
			pins[required.Mod.Path] = required.Mod.Version
		}
	}
	for _, required := range child.Require {
		if required.Mod.Path == modulePath {
			continue
		}
		if pins[required.Mod.Path] != required.Mod.Version {
			t.Errorf("native dependency %s@%s differs from parent pin %s", required.Mod.Path, required.Mod.Version, pins[required.Mod.Path])
		}
		delete(pins, required.Mod.Path)
	}
	if len(pins) != 0 {
		t.Errorf("native module omitted parent pins: %v", pins)
	}
	parentSum, err := os.ReadFile("go.sum")
	if err != nil {
		t.Fatal(err)
	}
	childSum, err := os.ReadFile("internal/backends/native/go.sum")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(parentSum, childSum) {
		t.Fatal("native and parent dependency checksums differ")
	}
}
