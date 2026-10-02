//go:build unix

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestCLIWatchdogRetainsIncompleteEvidence(t *testing.T) {
	for _, coverage := range []string{"none", "semantic", "choice", "semantic+choice"} {
		t.Run(coverage, func(t *testing.T) {
			root := t.TempDir()
			arguments := []string{"explore", "--json", "--toolchain-root", cliToolchainRoot, "--artifacts", root,
				"--seeds", "1", "--parallel", "1", "--on-failure", "all", "--execution-timeout", "2s", "--overall-timeout", "1m", "--terminate-grace", "100ms", "--coverage", coverage}
			if coverage == "choice" || coverage == "semantic+choice" {
				arguments = append(arguments, "--choices", "--diagnostics")
			}
			arguments = append(arguments, "go-run", "./cmd/gomad/testdata/watchdog", "--", "watchdog")
			runCLI(t, 1, arguments...)
			campaign := inspectCLI(t, campaignPath(t, root)).Campaign
			if campaign == nil || campaign.Attempted != 1 || campaign.Watchdogs != 1 || campaign.Failures != 1 || campaign.Succeeded != 0 || len(campaign.FailureArtifacts) != 1 {
				t.Fatalf("watchdog campaign = %#v", campaign)
			}
			path := campaign.FailureArtifacts[0].Path
			observed := inspectCLI(t, path).Artifact
			if observed == nil || observed.Outcome.Domain != "watchdog" || observed.Outcome.Reason != "watchdog_timeout" || observed.Transcript != nil {
				t.Fatalf("watchdog artifact = %#v", observed)
			}
			data, err := os.ReadFile(filepath.Join(path, "manifest.json"))
			if err != nil {
				t.Fatal(err)
			}
			manifest, err := record.DecodeExecutionRecord(data)
			if err != nil {
				t.Fatal(err)
			}
			if manifest.ArtifactKind != record.ArtifactWatchdogTimeout || manifest.ReplayMode != record.ReplayDiagnostic || manifest.IOProfile.Transcript != nil || manifest.ChoiceProfile != nil || !bytes.Contains(runCLI(t, 0, "replay", "--toolchain-root", cliToolchainRoot, "--verify-only", path), []byte("verified")) {
				t.Fatalf("watchdog manifest = %#v", manifest)
			}
			if _, err := os.Stat(filepath.Join(path, "io", "transcript.bin")); !os.IsNotExist(err) {
				t.Fatalf("incomplete transcript published: %v", err)
			}
		})
	}
}
