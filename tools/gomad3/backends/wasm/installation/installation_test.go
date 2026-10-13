package installation

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad3/backends/wasm/toolchain"
	"go.temporal.io/server/tools/gomad3/backends/wasm/wasi"
	netmodel "go.temporal.io/server/tools/gomad3/simulation/network"
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

func TestReadConfigPreservesOpenMapKeysAndRejectsInvalidText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	data := []byte(`{"CapturedInputs":{"Payloads":{"Name":"","name":""}}}`)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	config, err := ReadConfig(path)
	if err != nil {
		t.Fatalf("open payload map keys: %v", err)
	}
	if len(config.CapturedInputs.Payloads) != 2 {
		t.Fatalf("payload keys = %#v", config.CapturedInputs.Payloads)
	}
	invalidUTF8 := append([]byte(`{"Profile":"`), 0xff)
	invalidUTF8 = append(invalidUTF8, []byte(`"}`)...)
	if err := os.WriteFile(path, invalidUTF8, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(path); err == nil {
		t.Fatal("invalid UTF-8 config was accepted")
	}
}

func TestReadConfigPreservesFirstValueErrorsBeforeTrailingData(t *testing.T) {
	for _, test := range []struct {
		name, input string
	}{
		{name: "type", input: `{"Profile":1} {}`},
		{name: "unknown", input: `{"UnknownOwned":true} {}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			var legacy wasi.Config
			decoder := json.NewDecoder(strings.NewReader(test.input))
			decoder.DisallowUnknownFields()
			want := decoder.Decode(&legacy)
			if want == nil {
				t.Fatal("legacy first-value decoder accepted invalid input")
			}
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(test.input), 0600); err != nil {
				t.Fatal(err)
			}
			if _, got := ReadConfig(path); got == nil || got.Error() != want.Error() {
				t.Fatalf("ReadConfig() error = %v, want legacy first-value error %v", got, want)
			}
		})
	}
}

func TestInstallationRejectsUnqualifiedModelsBeforePreparation(t *testing.T) {
	stock, _ := installedManifest(t)
	options := stock.Options
	options.Config = DefaultConfig(wasi.CooperativeProfile)
	options.RuntimeRoot = t.TempDir()
	cooperative, err := Build(options)
	if err != nil {
		t.Fatal(err)
	}
	if cooperative.RuntimeImplementationSHA256 != toolchain.ImplementationSHA256() {
		t.Fatal("cooperative installation lost its runtime identity")
	}
	if _, err := (Description{Manifest: cooperative}).Provider(); err != nil {
		t.Fatalf("valid cooperative installation: %v", err)
	}
	for _, test := range []struct {
		name   string
		mutate func(*wasi.Config)
	}{
		{"network v2", func(c *wasi.Config) {
			c.NetworkV2 = &wasi.NetworkModelV2{Profile: wasi.NetworkStreamProfile, Limits: netmodel.Limits{Listeners: 1, Connections: 1, Bytes: 1024, ConnectionBytes: 4 << 20, StallNanos: 10_000_000_000, Transitions: 16}}
		}},
		{"storage", func(c *wasi.Config) { c.Storage = &wasi.StorageModel{} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			manifest := cooperative
			test.mutate(&manifest.Options.Config)
			const rejected = "operator network and storage profiles await qualification"
			check := func(operation string, err error) {
				t.Helper()
				if err == nil || err.Error() != rejected {
					t.Errorf("%s: expected qualification rejection, got %v", operation, err)
				}
			}
			_, err := Build(manifest.Options)
			check("build", err)
			path := filepath.Join(t.TempDir(), Sibling)
			check("write", Write(path, manifest))
			_, err = (Description{Manifest: manifest}).Provider()
			check("provider", err)
			manifest.Options.HelperPath = filepath.Join(t.TempDir(), "missing-helper")
			data, err := json.Marshal(manifest)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			_, err = Resolve(Spec{ExplicitManifest: path})
			check("resolve before helper access", err)
			manifest.Options.CacheRoot = "relative"
			if err := validate(manifest); err == nil || err.Error() != "WASM installation schema and absolute tool/cache paths are required" {
				t.Fatalf("qualification rejection changed path error precedence: %v", err)
			}
			if err := os.WriteFile(path, append(data, []byte(" {}")...), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := Resolve(Spec{ExplicitManifest: path}); err == nil || !strings.Contains(err.Error(), "trailing JSON") {
				t.Fatalf("qualification rejection changed decode error precedence: %v", err)
			}
		})
	}
}
