package cli

import (
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestApplicationRetainsInstallationIdentityAndPrivateCommands(t *testing.T) {
	directory := t.TempDir()
	executable := filepath.Join(directory, "gomad")
	if err := os.WriteFile(executable, []byte("runner binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(directory, "toolchain")
	app := applicationWithExecutable(executable)
	resolvedRoot, resolvedExecutable, build, err := app.identity(root)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedRoot != root || resolvedExecutable != executable || build != string(record.HashBytes([]byte("runner binary"))) {
		t.Fatalf("identity = %q, %q, %q", resolvedRoot, resolvedExecutable, build)
	}
	commands := app.commands(executable)
	if len(commands.supervisor) != 2 || commands.supervisor[0] != executable || commands.supervisor[1] != "__supervisor" || len(commands.coordinator) != 2 || commands.coordinator[0] != executable || commands.coordinator[1] != "__coordinator" {
		t.Fatalf("private commands = %#v", commands)
	}
	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	secondRoot, secondExecutable, secondBuild, err := app.identity(root)
	if err != nil || secondRoot != resolvedRoot || secondExecutable != resolvedExecutable || secondBuild != build {
		t.Fatalf("cached identity = %q, %q, %q, %v", secondRoot, secondExecutable, secondBuild, err)
	}
}
