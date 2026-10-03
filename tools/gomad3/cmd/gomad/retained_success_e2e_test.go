//go:build unix

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"go.temporal.io/server/tools/gomad3/record"
)

func TestCLIReplaysDistinctSeedsWithOneSuccessSignature(t *testing.T) {
	module := t.TempDir()
	for name, contents := range map[string]string{
		"go.mod":  "module example.com/retained-success\n\ngo 1.27.1\n",
		"main.go": "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"same output\") }\n",
	} {
		if err := os.WriteFile(filepath.Join(module, name), []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	root := t.TempDir()
	runCLI(t, 0, "explore", "--json", "--toolchain-root", cliToolchainRoot, "--working-dir", module, "--artifacts", root,
		"--seeds", "1-2", "--parallel", "1", "--keep-successes", "all", "--success-limit", "2", "--success-bytes", "64MiB", "go-run", ".")
	path := campaignPath(t, root)
	campaign := inspectCLI(t, path).Campaign
	if campaign == nil || campaign.Attempted != 2 || campaign.Succeeded != 2 || campaign.RetainedSuccesses != 2 || len(campaign.SuccessArtifacts) != 2 {
		t.Fatalf("campaign = %#v", campaign)
	}
	entries, err := os.ReadDir(filepath.Join(path, "successes"))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || campaign.SuccessArtifacts[0].Path == campaign.SuccessArtifacts[1].Path {
		t.Fatalf("success artifacts = %v, disk entries = %v", campaign.SuccessArtifacts, entries)
	}
	var signature record.SHA256
	for index, retained := range campaign.SuccessArtifacts {
		observed := inspectCLI(t, retained.Path).Artifact
		if observed == nil || observed.Seed != uint64(index+1) {
			t.Fatalf("retained success %d = %#v", index, observed)
		}
		if index == 0 {
			signature = observed.Outcome.FailureSignature
		} else if observed.Outcome.FailureSignature != signature {
			t.Fatalf("outcome signatures differ: %s, %s", signature, observed.Outcome.FailureSignature)
		}
		output := runCLI(t, 0, "replay", "--toolchain-root", cliToolchainRoot, retained.Path)
		if !bytes.Contains(output, []byte("reproduced=true")) {
			t.Fatalf("retained success did not replay: %s", output)
		}
	}
}
