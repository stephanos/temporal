package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/backends/target"
)

func TestSourceReviewProfileDoesNotAdmitExecution(t *testing.T) {
	t.Parallel()
	for _, platform := range []TargetContract{{GOOS: "darwin", GOARCH: "arm64"}, {GOOS: "linux", GOARCH: "amd64"}} {
		identity, err := SourceReviewProfile(platform.GOOS, platform.GOARCH)
		if err != nil || identity.Name != Deterministic || identity.ImplementationSHA256 == "" {
			t.Fatalf("source identity for %#v = %#v, %v", platform, identity, err)
		}
		golden := profileGoldens[platform.GOOS+"/"+platform.GOARCH]
		if string(identity.ImplementationSHA256) != golden.implementationSHA256 || string(identity.InventorySHA256) != golden.inventorySHA256 {
			t.Fatalf("source identity changed the approved platform profile: %#v", identity)
		}
		definition := *deterministicProfile.definition
		definition.target.GOOS, definition.target.GOARCH = platform.GOOS, platform.GOARCH
		source := mustSpec(definition)
		if identity != source.Identity() {
			t.Fatal("source profile identity does not bind declared platform")
		}
		if _, err := source.validated(); err == nil {
			t.Fatal("source profile admitted execution")
		}
	}
	if _, err := SourceReviewProfile("linux", "arm64"); err == nil {
		t.Fatal("unsupported native platform admitted")
	}
}

func TestSourceReviewAdaptersUseDeclaredProfileAndSourcePin(t *testing.T) {
	t.Parallel()
	command := os.Getenv("GOMAD_STOCK_GO")
	if command == "" {
		t.Skip("set GOMAD_STOCK_GO to pinned stock Go")
	}
	alias := filepath.Join(t.TempDir(), "pinned-go")
	if err := os.Symlink(command, alias); err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	files := map[string]string{
		"go.mod": fmt.Sprintf("module example.test/source\n\ngo 1.27.1\n\nrequire %s %s\n", fxAdapter.module, fxAdapter.version),
		"go.sum": fmt.Sprintf("%s %s %s\n", fxAdapter.module, fxAdapter.version, fxAdapter.sum),
	}
	for name, contents := range files {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, platform := range []struct{ name, goos, goarch, command string }{
		{"darwin direct", "darwin", "arm64", command},
		{"darwin alias", "darwin", "arm64", alias},
		{"linux direct", "linux", "amd64", command},
		{"linux alias", "linux", "amd64", alias},
	} {
		t.Run(platform.name, func(t *testing.T) {
			t.Parallel()
			prepared, adapters, err := PrepareSourceReviewAdapters(t.Context(), target.Spec{Kind: target.KindGoRun, Source: ".", WorkingDir: directory, PreparationRoot: t.TempDir()}, platform.command, platform.goos, platform.goarch)
			if err != nil {
				t.Fatal(err)
			}
			profile, err := SourceReviewProfile(platform.goos, platform.goarch)
			if err != nil {
				t.Fatal(err)
			}
			if len(adapters) != 1 || len(prepared.AdapterReplacements) != 1 || adapters[0].PreparedSourceSetSHA256 != fxPreparedSourceSetSHA256ByHost[platform.goos+"/"+platform.goarch] || prepared.AdapterReplacements[0].ProfileImplementationSHA256 != string(profile.ImplementationSHA256) {
				t.Fatalf("declared platform adapter = %#v, %#v", adapters, prepared.AdapterReplacements)
			}
		})
	}
}
