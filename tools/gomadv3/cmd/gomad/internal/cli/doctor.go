package cli

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"strings"

	"go.temporal.io/server/tools/gomadv3/deterministicio"
	"go.temporal.io/server/tools/gomadv3/evidence"
	"go.temporal.io/server/tools/gomadv3/target"
	gomadversion "go.temporal.io/server/tools/gomadv3/toolchain/version"
)

const reportSchema = "gomadv3.doctor/v3"

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
	Schema                  string          `json:"schema"`
	Available               bool            `json:"available"`
	Host                    string          `json:"host"`
	SupportedPlatforms      []string        `json:"supported_platforms"`
	GoVersion               string          `json:"go_version,omitempty"`
	ToolchainBuild          string          `json:"toolchain_build,omitempty"`
	RunnerBuild             evidence.SHA256 `json:"runner_build,omitempty"`
	BoundaryManifestVersion string          `json:"boundary_manifest_version"`
	IOInventorySHA256       evidence.SHA256 `json:"io_inventory_sha256"`
	IOImplementationSHA256  evidence.SHA256 `json:"io_implementation_sha256"`
	Adapters                []Adapter       `json:"adapters"`
	InstallationSource      string          `json:"installation_source"`
	ToolchainRoot           string          `json:"toolchain_root"`
	ArtifactDirectory       string          `json:"artifact_directory"`
	RepairInstruction       string          `json:"repair_instruction"`
	Checks                  []CheckResult   `json:"checks"`
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
	profile := deterministicio.Default()
	profileIdentity := profile.Identity()
	report := Report{
		Schema: reportSchema, Host: config.HostOS + "/" + config.HostArch,
		SupportedPlatforms:      append([]string(nil), gomadversion.SupportedPlatforms[:]...),
		BoundaryManifestVersion: gomadversion.BoundaryManifestVersion,
		IOInventorySHA256:       evidence.SHA256(profileIdentity.InventorySHA256), IOImplementationSHA256: evidence.SHA256(profileIdentity.ImplementationSHA256),
		Adapters:           []Adapter{},
		InstallationSource: config.InstallationSource,
		ToolchainRoot:      config.ToolchainRoot,
		ArtifactDirectory:  config.ArtifactRoot,
		RepairInstruction:  config.RepairInstruction,
		Checks:             make([]CheckResult, 0, 4+len(profile.Adapters())),
	}
	for _, identity := range profile.Adapters() {
		report.Adapters = append(report.Adapters, Adapter{
			Module: identity.Module, Version: identity.Version, Sum: identity.Sum, Status: "available",
		})
	}
	report.Checks = append(report.Checks, hostCheck(report.Host, report.SupportedPlatforms))
	identity, err := target.ReadToolchainIdentity(config.ToolchainRoot)
	if err != nil {
		report.Checks = append(report.Checks, failedCheck("toolchain", err.Error()+"; "+report.RepairInstruction))
	} else {
		report.GoVersion = identity.GoVersion
		report.ToolchainBuild = identity.BuildKey
		report.Checks = append(report.Checks, passedCheck("toolchain", identity.GoVersion+" build="+identity.BuildKey))
	}
	runnerDigest, err := hashExecutable(config.RunnerPath)
	if err != nil {
		report.Checks = append(report.Checks, failedCheck("runner", err.Error()+"; reinstall the Gomad executable at "+config.RunnerPath))
	} else {
		report.RunnerBuild = runnerDigest
		report.Checks = append(report.Checks, passedCheck("runner", string(runnerDigest)))
	}
	for _, adapter := range report.Adapters {
		report.Checks = append(report.Checks, passedCheck("adapter:"+adapter.Module, adapter.Module+"@"+adapter.Version+" "+adapter.Sum))
	}
	if err := checkArtifactDirectory(config.ArtifactRoot); err != nil {
		report.Checks = append(report.Checks, failedCheck("artifacts", err.Error()))
	} else {
		report.Checks = append(report.Checks, passedCheck("artifacts", config.ArtifactRoot))
	}
	report.Available = true
	for _, check := range report.Checks {
		if check.Status != "ok" {
			report.Available = false
			break
		}
	}
	return report
}

func hostCheck(host string, supported []string) CheckResult {
	for _, platform := range supported {
		if host == platform {
			return passedCheck("host", host)
		}
	}
	return failedCheck("host", host+" is unsupported; supported="+strings.Join(supported, ","))
}

func hashExecutable(path string) (evidence.SHA256, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open Runner: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat Runner: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&0o111 == 0 {
		return "", fmt.Errorf("Runner is not a regular executable: %s", path)
	}
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash Runner: %w", err)
	}
	return evidence.SHA256(fmt.Sprintf("sha256:%x", hasher.Sum(nil))), nil
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
	file, err := os.CreateTemp(path, ".gomadv3-doctor-")
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
