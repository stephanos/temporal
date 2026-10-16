//go:build !gomad_native

package cli

import (
	"fmt"
	"io"

	"go.temporal.io/server/tools/gomad/internal/backends"
)

const nativeBuiltIn = false
const nativeInstallationManifest = "gomad-install.json"

func configureNative(*application)     {}
func registerNative(backends.Registry) {}
func validateRequiredSemanticProbes(required []string) error {
	if len(required) != 0 {
		return fmt.Errorf("%s", nativeNotBuilt)
	}
	return nil
}
func (application) resolveInstallation(_, _ string) (nativeInstallation, error) {
	return nativeInstallation{Source: "not built in", RepairInstruction: nativeNotBuilt}, nil
}
func dispatchPrivateMode(string, io.Reader, io.Writer) error { return fmt.Errorf("%s", nativeNotBuilt) }
func Check(config Config) Report {
	return Report{Schema: reportSchema, Host: config.HostOS + "/" + config.HostArch, SupportedPlatforms: []string{}, Adapters: []Adapter{}, InstallationSource: "not built in", ArtifactDirectory: config.ArtifactRoot, RepairInstruction: nativeNotBuilt, Checks: []CheckResult{{Name: "native", Status: "not built in", Detail: nativeNotBuilt}}}
}
func (application) runAnalyze(_ []string, _, stderr io.Writer) int {
	return writeCommandError(stderr, 2, "%s\n", nativeNotBuilt)
}
func (application) runQualifySet(_ []string, _, stderr io.Writer) int {
	return writeCommandError(stderr, 2, "%s\n", nativeNotBuilt)
}
func runCompareSupport(_ []string, _, stderr io.Writer) int {
	return writeCommandError(stderr, 2, "%s\n", nativeNotBuilt)
}
func runMergeSet(_ []string, _, stderr io.Writer) int {
	return writeCommandError(stderr, 2, "%s\n", nativeNotBuilt)
}
