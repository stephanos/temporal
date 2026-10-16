//go:build gomad_native

package cli

import (
	nativebuild "go.temporal.io/server/tools/gomad/internal/backends/native/build"
	gomadversion "go.temporal.io/server/tools/gomad/internal/backends/native/toolchain/version"
	"go.temporal.io/server/tools/gomad/internal/backends/native/transcript"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

func Check(config Config) Report {
	profile := transcript.Default()
	profileIdentity := profile.Identity()
	report := Report{
		Schema: reportSchema, Host: config.HostOS + "/" + config.HostArch,
		SupportedPlatforms:      append([]string(nil), gomadversion.SupportedPlatforms[:]...),
		BoundaryManifestVersion: gomadversion.BoundaryManifestVersion,
		IOInventorySHA256:       record.SHA256(profileIdentity.InventorySHA256), IOImplementationSHA256: record.SHA256(profileIdentity.ImplementationSHA256),
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
	identity, err := nativebuild.ReadToolchainIdentity(config.ToolchainRoot)
	if err != nil {
		report.Checks = append(report.Checks, failedCheck("toolchain", err.Error()+"; "+report.RepairInstruction))
	} else {
		report.GoVersion = identity.GoVersion
		report.ToolchainBuild = identity.BuildKey
		report.Checks = append(report.Checks, passedCheck("toolchain", identity.GoVersion+" build="+identity.BuildKey))
	}
	runnerDigest, err := digestRunner(config.RunnerPath)
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
