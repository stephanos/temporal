package wasi

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHelperBuildPublishesConfiguredTarget(t *testing.T) {
	root := t.TempDir()
	makefile, err := os.ReadFile("../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), makefile, 0600); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(root, "bin")
	target := filepath.Join(root, "cargo-target")
	if err := os.MkdirAll(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "release"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "cargo"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	compiled := []byte("current configured helper")
	if err := os.WriteFile(filepath.Join(target, "release/gomad3-wasmhost"), compiled, 0700); err != nil {
		t.Fatal(err)
	}
	command := exec.Command("make", "-C", root, "helper")
	command.Env = append(os.Environ(), "CARGO_TARGET_DIR="+target, "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("helper build: %v\n%s", err, output)
	}
	installed, err := os.ReadFile(filepath.Join(root, "wasmhost/target/release/gomad3-wasmhost"))
	if err != nil || !bytes.Equal(installed, compiled) {
		t.Fatalf("configured Cargo target was not published to guest execution path: %q %v", installed, err)
	}
}
