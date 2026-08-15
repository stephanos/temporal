package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestChoiceImplementationIdentityBindsGeneratedAndInstrumentedInputs(t *testing.T) {
	inputs := choiceImplementationInputs{
		Schema:          []byte("schema"),
		CodecTemplate:   []byte("codec"),
		RuntimeTemplate: []byte("runtime-codec"),
		RuntimeOverlay:  []byte("runtime-instrumentation"),
		ToolchainPatch:  []byte("toolchain-patch"),
		HostTrace:       []byte("host-trace"),
		HostTape:        []byte("host-tape"),
	}
	want := choiceImplementationIdentity(inputs)
	if want == ([32]byte{}) {
		t.Fatal("choice implementation identity is empty")
	}
	mutations := [][]byte{inputs.Schema, inputs.CodecTemplate, inputs.RuntimeTemplate, inputs.RuntimeOverlay, inputs.ToolchainPatch, inputs.HostTrace, inputs.HostTape}
	for index, original := range mutations {
		changed := inputs
		value := append(bytes.Clone(original), '!')
		switch index {
		case 0:
			changed.Schema = value
		case 1:
			changed.CodecTemplate = value
		case 2:
			changed.RuntimeTemplate = value
		case 3:
			changed.RuntimeOverlay = value
		case 4:
			changed.ToolchainPatch = value
		case 5:
			changed.HostTrace = value
		case 6:
			changed.HostTape = value
		}
		if got := choiceImplementationIdentity(changed); got == want {
			t.Fatalf("identity did not bind input %d", index)
		}
	}
}

func TestRunGeneratesAndChecksEveryEndpoint(t *testing.T) {
	root := t.TempDir()
	for _, relative := range []string{
		"go1.26.4.patch",
		"internal/choicewire/tape.go",
		"internal/choicewire/trace.go",
		"overlay/src/runtime/gomad.go",
		"protocol/choicewire.go.tmpl",
		"protocol/choicewire.json",
		"protocol/choicewire_runtime.go.tmpl",
		"protocol/choicewire_test.go.tmpl",
		"protocol/iowire.go.tmpl",
		"protocol/iowire.json",
		"protocol/iowire_test.go.tmpl",
	} {
		contents, err := os.ReadFile(filepath.Join("..", "..", relative))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(root, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := run(root, false); err != nil {
		t.Fatal(err)
	}
	if err := run(root, true); err != nil {
		t.Fatal(err)
	}
	for _, relative := range []string{
		"internal/choicewire/wire_generated.go",
		"internal/choicewire/wire_generated_test.go",
		"internal/iowire/wire_generated.go",
		"internal/iowire/wire_generated_test.go",
		"overlay/src/internal/gomadchoicewire/wire_generated.go",
		"overlay/src/internal/gomadchoicewire/wire_generated_test.go",
		"overlay/src/runtime/gomad_choicewire_generated.go",
		"overlay/src/internal/gomadwire/wire_generated.go",
		"overlay/src/internal/gomadwire/wire_generated_test.go",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(relative))); err != nil {
			t.Fatalf("generated endpoint %q: %v", relative, err)
		}
	}
	for _, relative := range []string{"internal/choicewire/wire_generated.go", "internal/iowire/wire_generated.go"} {
		stale := filepath.Join(root, filepath.FromSlash(relative))
		current, err := os.ReadFile(stale)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(stale, append(current, []byte("stale")...), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := run(root, true); err == nil || !strings.Contains(err.Error(), "stale") {
			t.Fatalf("run(check) after changing %q error = %v", relative, err)
		}
		if err := run(root, false); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReadSchemaRejectsTrailingData(t *testing.T) {
	contents, err := os.ReadFile(filepath.Join("..", "..", "protocol", "iowire.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "iowire.json")
	contents = append(contents, []byte("\n{}\n")...)
	if err := os.WriteFile(path, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSchema(path); err == nil || !strings.Contains(err.Error(), "trailing data") {
		t.Fatalf("readSchema() error = %v", err)
	}
}
