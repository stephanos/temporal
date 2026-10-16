package cli

import (
	"fmt"
	"os"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

const reportSchema = "gomad.doctor/v3"

type Config struct {
	ToolchainRoot      string
	InstallationSource string
	RepairInstruction  string
	RunnerPath         string
	ArtifactRoot       string
	HostOS             string
	HostArch           string
}

type Report struct {
	Schema                  string        `json:"schema"`
	Available               bool          `json:"available"`
	Host                    string        `json:"host"`
	SupportedPlatforms      []string      `json:"supported_platforms"`
	GoVersion               string        `json:"go_version,omitempty"`
	ToolchainBuild          string        `json:"toolchain_build,omitempty"`
	RunnerBuild             record.SHA256 `json:"runner_build,omitempty"`
	BoundaryManifestVersion string        `json:"boundary_manifest_version"`
	IOInventorySHA256       record.SHA256 `json:"io_inventory_sha256"`
	IOImplementationSHA256  record.SHA256 `json:"io_implementation_sha256"`
	Adapters                []Adapter     `json:"adapters"`
	InstallationSource      string        `json:"installation_source"`
	ToolchainRoot           string        `json:"toolchain_root"`
	ArtifactDirectory       string        `json:"artifact_directory"`
	RepairInstruction       string        `json:"repair_instruction"`
	Checks                  []CheckResult `json:"checks"`
}

type Adapter struct {
	Module  string `json:"module"`
	Version string `json:"version"`
	Sum     string `json:"sum"`
	Status  string `json:"status"`
}

type CheckResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func hostCheck(host string, supported []string) CheckResult {
	for _, platform := range supported {
		if host == platform {
			return passedCheck("host", host)
		}
	}
	return failedCheck("host", host+" is unsupported; supported="+strings.Join(supported, ","))
}

func checkArtifactDirectory(path string) error {
	if path == "" {
		return fmt.Errorf("artifact directory is required")
	}
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create artifact directory %s: %w", path, err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat artifact directory %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("artifact path is not a directory: %s", path)
	}
	file, err := os.CreateTemp(path, ".gomad-doctor-")
	if err != nil {
		return fmt.Errorf("write artifact directory %s: %w", path, err)
	}
	name := file.Name()
	if closeErr := file.Close(); closeErr != nil {
		return fmt.Errorf("close artifact probe %s: %w", name, closeErr)
	}
	if removeErr := os.Remove(name); removeErr != nil {
		return fmt.Errorf("remove artifact probe %s: %w", name, removeErr)
	}
	return nil
}

func passedCheck(name, detail string) CheckResult {
	return CheckResult{Name: name, Status: "ok", Detail: detail}
}

func failedCheck(name, detail string) CheckResult {
	return CheckResult{Name: name, Status: "error", Detail: detail}
}
