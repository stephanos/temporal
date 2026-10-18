//go:build unix

package wasm_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

func TestHelperBuildPublishesIndependentExecutable(t *testing.T) {
	makefile, err := os.ReadFile("Makefile")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"default-target", "custom-target", "custom-target-matching-output"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			fixture := filepath.Join(root, "fixture")
			want := []byte("#!/bin/sh\nexit 0\n")
			for filename, data := range map[string][]byte{
				"Makefile":  makefile,
				"fixture":   want,
				"bin/cargo": []byte("#!/bin/sh\nset -eu\noutput=${CARGO_TARGET_DIR:-wasmhost/target}/release/gomad-wasmhost\nmkdir -p \"$(dirname \"$output\")\"\nif ! test -e \"$output\"; then ln \"$FIXTURE\" \"$output\"; fi\n"),
			} {
				path := filepath.Join(root, filename)
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0700); err != nil {
					t.Fatal(err)
				}
			}
			output := filepath.Join(root, "wasmhost/target/release/gomad-wasmhost")
			if name == "custom-target-matching-output" {
				if err := os.MkdirAll(filepath.Dir(output), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.Link(fixture, output); err != nil {
					t.Fatal(err)
				}
			}
			command := exec.CommandContext(t.Context(), "make", "helper")
			command.Dir = root
			for _, variable := range os.Environ() {
				if !strings.HasPrefix(variable, "CARGO_TARGET_DIR=") && !strings.HasPrefix(variable, "PATH=") && !strings.HasPrefix(variable, "FIXTURE=") {
					command.Env = append(command.Env, variable)
				}
			}
			command.Env = append(command.Env, "PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"), "FIXTURE="+fixture)
			if name != "default-target" {
				command.Env = append(command.Env, "CARGO_TARGET_DIR="+filepath.Join(root, "custom"))
			}
			if data, err := command.CombinedOutput(); err != nil {
				t.Fatalf("make helper: %v\n%s", err, data)
			}
			actual, err := hostfs.ReadBounded(output, uint64(len(want)))
			if err != nil || string(actual) != string(want) {
				t.Fatalf("published helper: %q, %v", actual, err)
			}
			info, err := os.Stat(output)
			if err != nil {
				t.Fatal(err)
			}
			if info.Mode().Perm() != 0755 {
				t.Fatalf("published helper permissions: %v", info.Mode())
			}
		})
	}
}
