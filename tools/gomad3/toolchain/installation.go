package toolchain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const InstallationManifestName = "gomad3-install.json"

const manifestSchema = "gomad3.installation/v1"

type InstallationSpec struct {
	Executable               string
	ExplicitToolchainRoot    string
	EnvironmentToolchainRoot string
}

type Installation struct {
	ToolchainRoot     string
	Source            string
	ManifestPath      string
	RepairInstruction string
}

type manifest struct {
	Schema        string `json:"schema"`
	ToolchainRoot string `json:"toolchain_root"`
}

func ResolveInstallation(config InstallationSpec) (Installation, error) {
	executable, err := filepath.Abs(config.Executable)
	if err != nil {
		return Installation{}, fmt.Errorf("resolve gomad executable path: %w", err)
	}
	if resolved, resolveErr := filepath.EvalSymlinks(executable); resolveErr == nil {
		executable = resolved
	}
	if config.ExplicitToolchainRoot != "" {
		return overrideResolution(config.ExplicitToolchainRoot, "CLI --toolchain-root", "cli")
	}
	if config.EnvironmentToolchainRoot != "" {
		return overrideResolution(config.EnvironmentToolchainRoot, "GOMAD3_TOOLCHAIN_DIR", "environment")
	}
	executableDirectory := filepath.Dir(executable)
	for _, path := range []string{
		filepath.Join(executableDirectory, InstallationManifestName),
		filepath.Join(filepath.Dir(executableDirectory), InstallationManifestName),
	} {
		decoded, found, manifestErr := readManifest(path)
		if manifestErr != nil {
			return Installation{}, manifestErr
		}
		if !found {
			continue
		}
		root := decoded.ToolchainRoot
		if !filepath.IsAbs(root) {
			root = filepath.Join(filepath.Dir(path), filepath.FromSlash(root))
		}
		root, err = normalizeRoot(root, "installation manifest")
		if err != nil {
			return Installation{}, err
		}
		return Installation{
			ToolchainRoot: root, Source: "manifest", ManifestPath: path,
			RepairInstruction: "reinstall the Gomad bundle described by " + path,
		}, nil
	}
	fallbacks := []string{
		filepath.Join(executableDirectory, ".toolchain"),
		filepath.Join(filepath.Dir(executableDirectory), ".toolchain"),
	}
	root := fallbacks[0]
	for _, candidate := range fallbacks {
		if info, statErr := os.Stat(candidate); statErr == nil && info.IsDir() {
			root = candidate
			break
		}
	}
	root, err = normalizeRoot(root, "adjacent installation")
	if err != nil {
		return Installation{}, err
	}
	return Installation{
		ToolchainRoot: root, Source: "adjacent",
		RepairInstruction: "install the Gomad toolchain at " + root + " or set GOMAD3_TOOLCHAIN_DIR",
	}, nil
}

func overrideResolution(root, name, source string) (Installation, error) {
	root, err := normalizeRoot(root, name)
	if err != nil {
		return Installation{}, err
	}
	return Installation{
		ToolchainRoot: root, Source: source,
		RepairInstruction: "install the Gomad toolchain at " + root,
	}, nil
}

func normalizeRoot(root, name string) (string, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || root == string(filepath.Separator) || strings.IndexByte(root, 0) >= 0 {
		return "", fmt.Errorf("%s toolchain root must be an absolute non-root clean path: %q", name, root)
	}
	return root, nil
}

func readManifest(path string) (manifest, bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return manifest{}, false, nil
	}
	if err != nil {
		return manifest{}, false, fmt.Errorf("inspect installation manifest: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() > 64<<10 {
		return manifest{}, false, fmt.Errorf("installation manifest is not a bounded regular file: %s", path)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return manifest{}, false, fmt.Errorf("read installation manifest: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	var decoded manifest
	if err := decoder.Decode(&decoded); err != nil {
		return manifest{}, false, fmt.Errorf("decode installation manifest: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return manifest{}, false, errors.New("installation manifest has trailing data")
	}
	if decoded.Schema != manifestSchema || decoded.ToolchainRoot == "" || strings.Contains(decoded.ToolchainRoot, "\\") || strings.IndexByte(decoded.ToolchainRoot, 0) >= 0 {
		return manifest{}, false, errors.New("installation manifest identity is invalid")
	}
	return decoded, true, nil
}
