package cli

import (
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

const reportSchema = "gomad.doctor/v3"

type Config struct {
	ArtifactRoot string
	HostOS       string
	HostArch     string
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

func Check(config Config) Report {
	return Report{
		Schema: reportSchema, Host: config.HostOS + "/" + config.HostArch,
		SupportedPlatforms: []string{}, Adapters: []Adapter{}, Checks: []CheckResult{},
		ArtifactDirectory: config.ArtifactRoot,
		RepairInstruction: "use gomad discover to inspect a WASM installation",
	}
}
