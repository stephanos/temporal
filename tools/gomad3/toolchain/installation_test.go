package toolchain

import (
	"os"
	"path/filepath"
	"testing"
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
