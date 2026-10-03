package toolchain

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/toolchain/installation"
)

func TestResolvePrefersExplicitThenEnvironment(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "bin", "gomad")
	explicit := filepath.Join(root, "explicit")
	environment := filepath.Join(root, "environment")
	resolution, err := ResolveInstallation(InstallationSpec{Executable: executable, ExplicitToolchainRoot: explicit, EnvironmentToolchainRoot: environment})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ToolchainRoot != explicit || resolution.Source != "cli" {
		t.Fatalf("resolution = %#v", resolution)
	}
	resolution, err = ResolveInstallation(InstallationSpec{Executable: executable, EnvironmentToolchainRoot: environment})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ToolchainRoot != environment || resolution.Source != "environment" {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestResolveReadsAdjacentBundleManifest(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	manifest := filepath.Join(root, InstallationManifestName)
	if err := os.WriteFile(manifest, []byte(`{"schema":"gomad3.installation/v1","toolchain_root":"lib/gomad3/toolchain"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	resolution, err := ResolveInstallation(InstallationSpec{Executable: filepath.Join(bin, "gomad")})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ToolchainRoot != filepath.Join(root, "lib", "gomad3", "toolchain") || resolution.Source != "manifest" || resolution.ManifestPath != manifest {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestResolveUsesExistingAdjacentCheckoutFallback(t *testing.T) {
	root := t.TempDir()
	bin := filepath.Join(root, ".bin")
	toolchain := filepath.Join(root, ".toolchain")
	for _, directory := range []string{bin, toolchain} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	resolution, err := ResolveInstallation(InstallationSpec{Executable: filepath.Join(bin, "gomad")})
	if err != nil {
		t.Fatal(err)
	}
	if resolution.ToolchainRoot != toolchain || resolution.Source != "adjacent" {
		t.Fatalf("resolution = %#v", resolution)
	}
}

func TestResolveRejectsRelativeOverridesAndMalformedManifest(t *testing.T) {
	if _, err := ResolveInstallation(InstallationSpec{Executable: "/bundle/bin/gomad", ExplicitToolchainRoot: "relative"}); err == nil {
		t.Fatal("ResolveInstallation() accepted a relative CLI root")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.MkdirAll(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, InstallationManifestName), []byte(`{"schema":"unknown","toolchain_root":"toolchain"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveInstallation(InstallationSpec{Executable: filepath.Join(bin, "gomad")}); err == nil {
		t.Fatal("ResolveInstallation() accepted a malformed manifest")
	}
}

func writeCompleteInstallation(t *testing.T, root, key string) {
	t.Helper()
	layout, err := installation.At(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{layout.GoCommand(), layout.Build(key).GoCommand()} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(layout.BuildKeyFile(), []byte(key+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Every resolution source yields a root that the validated installation
// description accepts and maps to the same build, cache and adapter locations.
func TestEveryResolutionSourceYieldsAValidatedDescription(t *testing.T) {
	key := strings.Repeat("f", 64)
	for _, test := range []struct {
		name   string
		source string
		setup  func(t *testing.T, base string) (InstallationSpec, string)
	}{
		{name: "explicit", source: "cli", setup: func(t *testing.T, base string) (InstallationSpec, string) {
			root := filepath.Join(base, "explicit")
			return InstallationSpec{Executable: filepath.Join(base, "bin", "gomad"), ExplicitToolchainRoot: root, EnvironmentToolchainRoot: filepath.Join(base, "environment")}, root
		}},
		{name: "environment", source: "environment", setup: func(t *testing.T, base string) (InstallationSpec, string) {
			root := filepath.Join(base, "environment")
			return InstallationSpec{Executable: filepath.Join(base, "bin", "gomad"), EnvironmentToolchainRoot: root}, root
		}},
		{name: "manifest", source: "manifest", setup: func(t *testing.T, base string) (InstallationSpec, string) {
			if err := os.MkdirAll(filepath.Join(base, "bin"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(base, InstallationManifestName), []byte(`{"schema":"gomad3.installation/v1","toolchain_root":"lib/gomad3/toolchain"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			return InstallationSpec{Executable: filepath.Join(base, "bin", "gomad")}, filepath.Join(base, "lib", "gomad3", "toolchain")
		}},
		{name: "executable-relative", source: "adjacent", setup: func(t *testing.T, base string) (InstallationSpec, string) {
			if err := os.MkdirAll(filepath.Join(base, ".bin"), 0o700); err != nil {
				t.Fatal(err)
			}
			return InstallationSpec{Executable: filepath.Join(base, ".bin", "gomad")}, filepath.Join(base, installation.CheckoutDirectory)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			base := t.TempDir()
			spec, root := test.setup(t, base)
			writeCompleteInstallation(t, root, key)
			resolution, err := ResolveInstallation(spec)
			if err != nil {
				t.Fatal(err)
			}
			if resolution.ToolchainRoot != root || resolution.Source != test.source {
				t.Fatalf("resolution = %#v, want root %s from %s", resolution, root, test.source)
			}
			description, err := installation.Describe(resolution.ToolchainRoot)
			if err != nil {
				t.Fatal(err)
			}
			build := description.PinnedBuild()
			if description.BuildKey() != key || description.GoCommand() != filepath.Join(root, "bin", "go") ||
				build.GoCommand() != filepath.Join(root, "builds", key, "bin", "go") ||
				build.TargetCache() != filepath.Join(root, "builds", key, "target-cache") ||
				build.PreparedTargets() != filepath.Join(root, "builds", key, "prepared-targets") ||
				description.Adapters() != filepath.Join(root, "adapters") {
				t.Fatalf("description = %#v", description)
			}
		})
	}
}

func TestResolveRejectsMalformedManifestsAndInvalidRoots(t *testing.T) {
	for _, test := range []struct {
		name     string
		manifest string
		want     string
	}{
		{name: "unknown schema", manifest: `{"schema":"unknown","toolchain_root":"toolchain"}`, want: "installation manifest identity is invalid"},
		{name: "empty root", manifest: `{"schema":"gomad3.installation/v1","toolchain_root":""}`, want: "installation manifest identity is invalid"},
		{name: "backslash root", manifest: `{"schema":"gomad3.installation/v1","toolchain_root":"lib\\toolchain"}`, want: "installation manifest identity is invalid"},
		{name: "unknown field", manifest: `{"schema":"gomad3.installation/v1","toolchain_root":"toolchain","extra":true}`, want: `decode installation manifest: json: unknown field "extra"`},
		{name: "trailing data", manifest: `{"schema":"gomad3.installation/v1","toolchain_root":"toolchain"} {}`, want: "installation manifest has trailing data"},
		{name: "filesystem root", manifest: `{"schema":"gomad3.installation/v1","toolchain_root":"/"}`, want: `installation manifest toolchain root must be an absolute non-root clean path: "/"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "bin"), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, InstallationManifestName), []byte(test.manifest), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := ResolveInstallation(InstallationSpec{Executable: filepath.Join(root, "bin", "gomad")}); err == nil || err.Error() != test.want {
				t.Fatalf("ResolveInstallation() error = %v, want %q", err, test.want)
			}
		})
	}
	for _, root := range []string{"relative", "/", "/opt/gomad/../toolchain"} {
		want := "GOMAD3_TOOLCHAIN_DIR toolchain root must be an absolute non-root clean path: " + strconv.Quote(root)
		if _, err := ResolveInstallation(InstallationSpec{Executable: "/bundle/bin/gomad", EnvironmentToolchainRoot: root}); err == nil || err.Error() != want {
			t.Fatalf("ResolveInstallation(%q) error = %v, want %q", root, err, want)
		}
	}
}
