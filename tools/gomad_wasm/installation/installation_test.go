package installation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad_wasm/backend"
	"go.temporal.io/server/tools/gomad_wasm/wasi"
)

func installedManifest(t *testing.T) (Manifest, string) {
	t.Helper()
	directory := t.TempDir()
	compiler, helper := filepath.Join(directory, "go"), filepath.Join(directory, "helper")
	for _, path := range []string{compiler, helper} {
		if err := os.WriteFile(path, []byte("pinned executable"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	manifest, err := Build(backend.Options{CompilerPath: compiler, HelperPath: helper, CacheRoot: filepath.Join(directory, "cache"), GoVersion: "go1.27.1", MemoryBytes: 256 << 20, Fuel: 1000000000, Config: DefaultConfig(wasi.StockProfile)})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, Sibling)
	if err := Write(path, manifest); err != nil {
		t.Fatal(err)
	}
	return manifest, path
}

func TestResolveInstallationPrecedenceAndSourceLessReplayInputs(t *testing.T) {
	manifest, path := installedManifest(t)
	if err := os.Remove(manifest.Options.CompilerPath); err != nil {
		t.Fatal(err)
	}
	for _, spec := range []Spec{
		{ExplicitManifest: path, EnvironmentManifest: "/missing/environment", Executable: "/missing/bin/gomad-wasm"},
		{EnvironmentManifest: path, Executable: "/missing/bin/gomad-wasm"},
		{Executable: filepath.Join(filepath.Dir(path), "gomad-wasm")},
	} {
		description, err := Resolve(spec)
		if err != nil {
			t.Fatal(err)
		}
		if description.Path != path || description.SHA256 == "" {
			t.Fatalf("description=%#v", description)
		}
		if _, err := description.Provider(); err != nil {
			t.Fatalf("source-less replay provider: %v", err)
		}
	}
	if _, err := Resolve(Spec{ExplicitManifest: "/missing/explicit", EnvironmentManifest: path}); err == nil {
		t.Fatal("invalid explicit manifest fell back to environment")
	}
}

func TestResolveRejectsChangedIdentityAndUnsafeManifest(t *testing.T) {
	manifest, path := installedManifest(t)
	for _, mutate := range []func(*Manifest){
		func(m *Manifest) { m.Schema = "unknown" },
		func(m *Manifest) { m.Options.ModelSHA256 = "sha256:" + strings.Repeat("0", 64) },
		func(m *Manifest) { m.Options.Engine.Version = "unknown" },
		func(m *Manifest) { m.Options.Config.Profile = "unknown" },
		func(m *Manifest) { m.Options.Config.Clock.ReadStepNanos = 0 },
		func(m *Manifest) { m.Options.Config.Network = &wasi.NetworkModel{} },
		func(m *Manifest) { m.Options.HelperSHA256 = "sha256:" + strings.Repeat("0", 64) },
		func(m *Manifest) { m.Options.CacheRoot = "relative" },
	} {
		changed := manifest
		mutate(&changed)
		data, err := json.Marshal(changed)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Resolve(Spec{ExplicitManifest: path}); err == nil {
			t.Fatalf("accepted altered manifest: %s", data)
		}
	}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalid := range [][]byte{append(append([]byte(nil), data...), []byte(" {}")...), []byte(`{"schema":"gomad-wasm.installation/v1","unknown":true}`), []byte(`{"schema":"unknown","Schema":"gomad-wasm.installation/v1"}`), []byte(`{"options":{"HelperSHA256":"first","HelperSHA256":"second"}}`)} {
		if err := os.WriteFile(path, invalid, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Resolve(Spec{ExplicitManifest: path}); err == nil {
			t.Fatal("accepted non-strict installation JSON")
		}
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "manifest-link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(Spec{ExplicitManifest: link}); err == nil {
		t.Fatal("accepted symbolic-link manifest")
	}
	if err := os.WriteFile(manifest.Options.HelperPath, []byte("changed executable"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve(Spec{ExplicitManifest: path}); err == nil {
		t.Fatal("accepted changed helper")
	}
}

func TestWritePreservesExistingInstallation(t *testing.T) {
	manifest, path := installedManifest(t)
	if err := Write(path, manifest); err == nil {
		t.Fatal("overwrote existing manifest")
	}
	if _, err := Resolve(Spec{ExplicitManifest: path}); err != nil {
		t.Fatal(err)
	}
}
