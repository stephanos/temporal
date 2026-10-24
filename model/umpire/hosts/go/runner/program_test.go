package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProgramCommandRequiresProgram(t *testing.T) {
	directory := t.TempDir()
	model := filepath.Join(directory, "guest.wasm")
	world := filepath.Join(directory, "world.binpb")
	output := filepath.Join(directory, "program.binpb")
	if err := os.WriteFile(model, statefulGuest(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(world, mustMarshal(t, control(0)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := command(t, nil, "program", "--model", model, "--world", world, "--out", output)
	if result.code != ExitError || !strings.Contains(result.stderr, "Reset returned no program") {
		t.Fatalf("missing program: %+v", result)
	}
	contents, err := os.ReadFile(output)
	if err != nil || string(contents) != "existing" {
		t.Fatalf("missing program overwrote output: %q, %v", contents, err)
	}
}

func TestProgramCommandWritesOpaqueProgram(t *testing.T) {
	directory := t.TempDir()
	model := filepath.Join(directory, "guest.wasm")
	world := filepath.Join(directory, "world.binpb")
	output := filepath.Join(directory, "program.binpb")
	if err := os.WriteFile(model, statefulGuest(t), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(world, mustMarshal(t, control(42)), 0o644); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		result := command(t, nil, "program", "--model", model, "--world", world, "--out", output)
		if result.code != ExitExplained || result.stdout != "" || result.stderr != "" {
			t.Fatalf("program: %+v", result)
		}
		contents, err := os.ReadFile(output)
		if err != nil || string(contents) != string(mustMarshal(t, control(42))) {
			t.Fatalf("opaque program: %x, %v", contents, err)
		}
	}
}

func TestProgramCommandErrors(t *testing.T) {
	directory := t.TempDir()
	model := filepath.Join(directory, "guest.wasm")
	world := filepath.Join(directory, "world.binpb")
	badWorld := filepath.Join(directory, "malformed.binpb")
	output := filepath.Join(directory, "program.binpb")
	for path, contents := range map[string][]byte{
		model: statefulGuest(t), world: mustMarshal(t, control(42)), badWorld: {0xff},
	} {
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for name, tc := range map[string]struct {
		args    []string
		message string
	}{
		"required flags":    {nil, "--model, --world and --out are required"},
		"positional":        {[]string{"extra"}, "no positional arguments"},
		"unknown flag":      {[]string{"--driver", "fake"}, "flag provided but not defined"},
		"missing model":     {[]string{"--model", filepath.Join(directory, "missing"), "--world", world, "--out", output}, "read the model"},
		"bad world":         {[]string{"--model", model, "--world", badWorld, "--out", output}, "read the world"},
		"bad model":         {[]string{"--model", world, "--world", world, "--out", output}, "compile model"},
		"unwritable output": {[]string{"--model", model, "--world", world, "--out", directory}, "write the program"},
	} {
		t.Run(name, func(t *testing.T) {
			result := command(t, nil, append([]string{"program"}, tc.args...)...)
			if result.code != ExitError || !strings.Contains(result.stderr, tc.message) {
				t.Fatalf("program error: %+v", result)
			}
		})
	}
}
