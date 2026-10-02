//go:build unix

package main

import (
	"bytes"
	"testing"
)

func TestCLIWatchdogDiagnosticReplay(t *testing.T) {
	root := t.TempDir()
	runCLI(t, 1, "explore", "--json", "--toolchain-root", cliToolchainRoot, "--artifacts", root,
		"--seeds", "1", "--execution-timeout", "2s", "--overall-timeout", "1m", "--terminate-grace", "100ms",
		"go-run", "./cmd/gomad/testdata/watchdog", "--", "watchdog")
	campaign := inspectCLI(t, campaignPath(t, root)).Campaign
	if campaign == nil || campaign.Watchdogs != 1 || len(campaign.FailureArtifacts) != 1 {
		t.Fatalf("watchdog campaign = %#v", campaign)
	}
	path := campaign.FailureArtifacts[0].Path
	retained := inspectCLI(t, path).Artifact
	if retained == nil || retained.Outcome.Domain != "watchdog" || retained.Transcript != nil {
		t.Fatalf("watchdog artifact = %#v", retained)
	}
	runCLI(t, 0, "replay", "--toolchain-root", cliToolchainRoot, "--verify-only", path)
	output := runCLI(t, 1, "replay", "--toolchain-root", cliToolchainRoot, path)
	if !bytes.Contains(output, []byte("reproduced=true diagnostic=true result=watchdog_observation choice-replay=none")) {
		t.Fatalf("watchdog replay output = %s", output)
	}
}
