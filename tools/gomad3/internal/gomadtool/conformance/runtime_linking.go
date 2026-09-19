package conformance

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad3/internal/hostexec"
)

func (campaign *runtimeCampaign) requireLinkModes() error {
	ccResult, err := campaign.command(
		"go-env-cc", []string{campaign.config.Go, "env", "CC"}, campaign.testdata, 10*time.Second, []string{"GOMADSEED"},
	)
	if err != nil {
		return err
	}
	ccFields := strings.Fields(commandOutput(ccResult))
	if len(ccFields) == 0 || !executableAvailable(ccFields[0]) {
		return nil
	}
	clockCGO, err := campaign.build("clock-cgo", "./clock_cgo", true)
	if err != nil {
		return err
	}
	disabled, err := campaign.command("clock-cgo-disabled", []string{clockCGO}, campaign.testdata, 5*time.Second, []string{"GOMADSEED"})
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "42", "disabled cgo clock"); err != nil {
		return err
	}
	if err := campaign.requireUnsupportedLink("clock-cgo-enabled", clockCGO, nil); err != nil {
		return err
	}
	clockExternal, err := campaign.build("clock-external", "./clock", true, "-ldflags=-linkmode=external")
	if err != nil {
		return err
	}
	disabled, err = campaign.command(
		"clock-external-disabled", []string{clockExternal, "disabled"}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "TZ"}, "TZ=UTC",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabled, "clock disabled ok", "disabled externally linked clock"); err != nil {
		return err
	}
	if err := campaign.requireUnsupportedLink("clock-external-enabled", clockExternal, []string{"initial"}); err != nil {
		return err
	}
	goos, err := campaign.command("go-env-goos", []string{campaign.config.Go, "env", "GOOS"}, campaign.testdata, 10*time.Second, []string{"GOMADSEED"})
	if err != nil {
		return err
	}
	if commandOutput(goos) != "darwin" {
		return nil
	}
	clockAMD64 := filepath.Join(campaign.workspace, "clock-external-amd64")
	_, err = campaign.command(
		"clock-external-amd64-build",
		[]string{campaign.config.Go, "build", "-ldflags=-linkmode=external", "-o", clockAMD64, "./clock"},
		campaign.testdata, time.Minute, []string{"GOMADSEED", "GOARCH", "CGO_ENABLED"}, "GOARCH=amd64", "CGO_ENABLED=1",
	)
	if err != nil {
		return err
	}
	disabledAMD64, err := campaign.command(
		"clock-external-amd64-disabled", []string{clockAMD64, "disabled"}, campaign.testdata, 5*time.Second,
		[]string{"GOMADSEED", "TZ"}, "TZ=UTC",
	)
	if err != nil {
		return err
	}
	if err := requireOutput(disabledAMD64, "clock disabled ok", "disabled Darwin/amd64 externally linked clock"); err != nil {
		return err
	}
	return campaign.requireUnsupportedLink("clock-external-amd64-enabled", clockAMD64, []string{"initial"})
}

func (campaign *runtimeCampaign) requireUnsupportedLink(name, binary string, arguments []string) error {
	return campaign.expectedExit(
		name, append([]string{binary}, arguments...), campaign.testdata, 5*time.Second, 2,
		func(result hostexec.Result) error {
			if commandErrorOutput(result) != "runtime: GOMADSEED does not support cgo or external linking" {
				return fmt.Errorf("%s emitted an unexpected diagnostic: %q", name, string(result.Stderr.RawBytes))
			}
			return nil
		},
		[]string{"GOMADSEED", "TZ"}, "GOMADSEED=1", "TZ=UTC",
	)
}
