package set

import (
	"path/filepath"
	"slices"
	"testing"
)

func TestQualificationCommandsPreserveSelectedInstallation(t *testing.T) {
	manifest := Manifest{RunTimeout: "1s", OverallTimeout: "2s", TerminateGrace: "1s"}
	workload := Workload{Package: "./fixture", Test: "TestFixture"}
	selected := filepath.Join(t.TempDir(), "installation.json")
	for _, path := range []string{"", selected} {
		config := Spec{InstallationManifest: path}
		for _, command := range []struct {
			name  string
			value Command
		}{
			{"qualify", workloadCommand(config, manifest, workload, 7)},
			{"analyze", analysisCommand(config, manifest, workload)},
		} {
			prefix := []string{command.name}
			if path != "" {
				prefix = []string{"--installation", path, command.name}
			}
			if len(command.value.Args) < len(prefix) || !slices.Equal(command.value.Args[:len(prefix)], prefix) {
				t.Fatalf("%s installation %q argv = %q, want prefix %q", command.name, path, command.value.Args, prefix)
			}
		}
	}
}
