package set

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/artifact"
	"go.temporal.io/server/tools/gomad3/qualification"
)

// pruneQualifiedCampaigns removes the Campaigns a qualified seed retained once
// the set report has projected their evidence. The seed's qualification report
// stays under the artifact root, and removal goes through an os.Root so a
// recorded path can never delete anything outside that root. The root's shared
// targets go with them once no retained artifact shares them.
func pruneQualifiedCampaigns(artifactRoot string, report qualification.QualificationReport) error {
	if !report.Qualified || len(report.Executions) == 0 {
		return errors.New("only a qualified seed's retained Campaigns can be pruned")
	}
	campaigns := make([]string, 0, len(report.Executions))
	for index, run := range report.Executions {
		// Replay is the last use of a retained success artifact; pruning an
		// execution that was not replayed exactly would discard evidence nobody
		// has verified.
		if run.ArtifactPath == "" || run.Replay == nil || !run.Replay.Attempted || !run.Replay.Match {
			return fmt.Errorf("qualification execution %d has no exact successful replay", index)
		}
		campaign, err := artifactRootRelative(artifactRoot, run.CampaignPath)
		if err != nil {
			return fmt.Errorf("qualification execution %d campaign: %w", index, err)
		}
		if parts := strings.Split(campaign, string(filepath.Separator)); len(parts) != 2 || parts[0] != "v1" || !strings.HasPrefix(parts[1], "campaign-") {
			return fmt.Errorf("qualification execution %d campaign %q is not a retained Campaign", index, campaign)
		}
		artifactPath, err := artifactRootRelative(artifactRoot, run.ArtifactPath)
		if err != nil || !strings.HasPrefix(artifactPath, campaign+string(filepath.Separator)) {
			return errors.Join(fmt.Errorf("qualification execution %d artifact is outside its campaign", index), err)
		}
		campaigns = append(campaigns, campaign)
	}
	slices.Sort(campaigns)
	campaigns = slices.Compact(campaigns)
	root, err := os.OpenRoot(artifactRoot)
	if err != nil {
		return err
	}
	for _, campaign := range campaigns {
		info, err := root.Lstat(campaign)
		if err != nil {
			return errors.Join(err, root.Close())
		}
		if !info.IsDir() {
			return errors.Join(fmt.Errorf("retained Campaign %q is not a directory", campaign), root.Close())
		}
		if err := root.RemoveAll(campaign); err != nil {
			return errors.Join(err, root.Close())
		}
	}
	if err := root.Close(); err != nil {
		return err
	}
	// Another Campaign of this root may be publishing, so its staging stays.
	return artifact.PruneTargetPool(artifact.TargetPool(artifactRoot), false)
}

func artifactRootRelative(artifactRoot, path string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", fmt.Errorf("path %q is not absolute", path)
	}
	relative, err := filepath.Rel(artifactRoot, path)
	if err != nil {
		return "", err
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside the artifact root", path)
	}
	return relative, nil
}
