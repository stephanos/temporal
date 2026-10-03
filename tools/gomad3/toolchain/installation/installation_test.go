package installation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Locations recorded in prepared target identities and published by older
// builders must not move, so each is pinned as a literal.
func TestLayoutPinsEveryLocation(t *testing.T) {
	const root = "/opt/gomad/toolchain"
	const key = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	layout, err := At(root)
	if err != nil {
		t.Fatal(err)
	}
	build := layout.Build(key)
	got := map[string]string{
		"root": layout.Root(), "go": layout.GoCommand(), "bin": layout.Bin(), "build-key": layout.BuildKeyFile(),
		"builds": layout.Builds(), "locks": layout.Locks(), "lock": layout.Lock(key), "downloads": layout.Downloads(),
		"adapters": layout.Adapters(), "build": build.Directory(), "build-go": build.GoCommand(),
		"target-cache": build.TargetCache(), "prepared-targets": build.PreparedTargets(),
	}
	want := map[string]string{
		"root":             "/opt/gomad/toolchain",
		"go":               "/opt/gomad/toolchain/bin/go",
		"bin":              "/opt/gomad/toolchain/bin",
		"build-key":        "/opt/gomad/toolchain/build-key",
		"builds":           "/opt/gomad/toolchain/builds",
		"locks":            "/opt/gomad/toolchain/locks",
		"lock":             "/opt/gomad/toolchain/locks/" + key + ".lock",
		"downloads":        "/opt/gomad/toolchain/downloads",
		"adapters":         "/opt/gomad/toolchain/adapters",
		"build":            "/opt/gomad/toolchain/builds/" + key,
		"build-go":         "/opt/gomad/toolchain/builds/" + key + "/bin/go",
		"target-cache":     "/opt/gomad/toolchain/builds/" + key + "/target-cache",
		"prepared-targets": "/opt/gomad/toolchain/builds/" + key + "/prepared-targets",
	}
	if len(got) != len(want) {
		t.Fatalf("locations = %#v, want %#v", got, want)
	}
	for name, location := range want {
		if got[name] != location {
			t.Errorf("%s = %q, want %q", name, got[name], location)
		}
	}
}

func TestAtResolvesRelativeRootsAgainstTheWorkingDirectory(t *testing.T) {
	parent := t.TempDir()
	t.Chdir(parent)
	layout, err := At("toolchain/../toolchain")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(parent, "toolchain"); layout.Root() != want || layout.Adapters() != filepath.Join(want, "adapters") {
		t.Fatalf("layout = %#v, want root %s", layout, want)
	}
}

func TestDescribeSuppliesThePinnedBuild(t *testing.T) {
	root := filepath.Join(t.TempDir(), "toolchain")
	key := strings.Repeat("e", 64)
	for _, path := range []string{filepath.Join(root, "bin", "go"), filepath.Join(root, "builds", key, "bin", "go")} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "build-key"), []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	description, err := Describe(root)
	if err != nil {
		t.Fatal(err)
	}
	if description.Root() != root || description.BuildKey() != key || description.PinnedBuild() != description.Build(key) {
		t.Fatalf("description = %#v", description)
	}
	if err := os.Remove(filepath.Join(root, "builds", key, "bin", "go")); err != nil {
		t.Fatal(err)
	}
	want := "toolchain build " + key + " is missing or stale in " + root + "; set --toolchain-root or GOMAD3_TOOLCHAIN_DIR to a complete Gomad installation"
	if _, err := Describe(root); err == nil || err.Error() != want {
		t.Fatalf("Describe() error = %v, want %q", err, want)
	}
}
