package adapterregen

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"go.temporal.io/server/tools/gomad3/deterministicio"
)

// Render writes the human review of a dry run or the summary of an apply.
func Render(w io.Writer, result Result) error {
	regeneration := result.Regeneration
	previous, proposed := regeneration.Previous, regeneration.Proposed
	var out strings.Builder
	fmt.Fprintf(&out, "gomad3 adapter regeneration: %s %s -> %s\n", regeneration.Module, previous.Version, proposed.Version)
	changed := 0
	for _, diff := range result.Diffs {
		if diff.Changed {
			changed++
		}
	}
	fmt.Fprintf(&out, "\nchanged upstream source: %d of %d rewritten files\n", changed, len(result.Diffs))
	for _, diff := range result.Diffs {
		if diff.Changed {
			out.WriteString("\n" + diff.Diff)
		} else {
			fmt.Fprintf(&out, "  unchanged %s\n", diff.Path)
		}
	}
	out.WriteString("\nproposed anchors:\n")
	anchor := func(name, before, after string) {
		marker := "changed"
		if before == after {
			marker = "same"
		}
		fmt.Fprintf(&out, "  %-7s %s\n          %s\n       -> %s\n", marker, name, before, after)
	}
	anchor("version", previous.Version, proposed.Version)
	anchor("sum", previous.Sum, proposed.Sum)
	anchor("original source inventory", previous.OriginalSourceInventorySHA256, proposed.OriginalSourceInventorySHA256)
	anchor("replacement source inventory", previous.ReplacementSourceInventorySHA256, proposed.ReplacementSourceInventorySHA256)
	for index, rewrite := range proposed.Rewrites {
		before := previous.Rewrites[index]
		anchor(rewrite.Path+" source", before.SourceSHA256, rewrite.SourceSHA256)
		if rewrite.Base != "" {
			anchor(rewrite.Path+" base "+rewrite.Base, before.BaseSHA256, rewrite.BaseSHA256)
		}
		anchor(rewrite.Path+" replacement", before.ReplacementSHA256, rewrite.ReplacementSHA256)
	}
	for _, platform := range sortedPlatforms(proposed) {
		anchor(proposed.PreparedPackage+" prepared source set "+platform, previous.PreparedSourceSetSHA256[platform], proposed.PreparedSourceSetSHA256[platform])
	}
	if len(result.StalePacks) == 0 {
		out.WriteString("\nstale compatibility packs: none\n")
	} else {
		fmt.Fprintf(&out, "\nstale compatibility packs: %d bindings still name %s@%s; refresh the packs after applying\n", len(result.StalePacks), regeneration.Module, previous.Version)
		for _, stale := range result.StalePacks {
			fmt.Fprintf(&out, "  %s (%s, %s): %s\n", stale.File, stale.ID, strings.Join(stale.Platforms, ","), stale.Location)
		}
	}
	if len(result.Staged) != 0 {
		verb := "publishes"
		if !result.Applied {
			verb = "would publish"
		}
		fmt.Fprintf(&out, "\nstaged and verified; the apply %s %d files:\n", verb, len(result.Staged))
		for _, file := range result.Staged {
			fmt.Fprintf(&out, "  %-7s %s\n", file.Change, file.Path)
		}
		for _, file := range result.Staged {
			if file.Diff != "" {
				out.WriteString("\n" + file.Diff)
			}
		}
	}
	switch {
	case !result.Applied && len(result.Staged) != 0:
		fmt.Fprintf(&out, "\nstaged only; nothing was published. Apply with:\n  gomadtool adapter-regenerate --module=%s --version=%s --approve-review=%s\n", regeneration.Module, proposed.Version, regeneration.ApprovalSHA256)
	case !result.Applied:
		fmt.Fprintf(&out, "\napproval: %s\nreview the changed source above, then see every file the apply publishes with:\n  gomadtool adapter-regenerate --module=%s --version=%s --approve-review=%s --stage-only\nand apply with:\n  gomadtool adapter-regenerate --module=%s --version=%s --approve-review=%s\n", regeneration.ApprovalSHA256, regeneration.Module, proposed.Version, regeneration.ApprovalSHA256, regeneration.Module, proposed.Version, regeneration.ApprovalSHA256)
	default:
		fmt.Fprintf(&out, "\napplied %s; published %d files:\n", regeneration.ApprovalSHA256, len(result.Published))
		for _, path := range result.Published {
			fmt.Fprintf(&out, "  %s\n", path)
		}
		if len(result.Residual) != 0 {
			fmt.Fprintf(&out, "references to %s@%s left for review:\n", regeneration.Module, previous.Version)
			for _, location := range result.Residual {
				fmt.Fprintf(&out, "  %s\n", location)
			}
		}
		for _, warning := range result.Warnings {
			fmt.Fprintf(&out, "warning: %s\n", warning)
		}
		out.WriteString("rebuild .bin/gomad and requalify the adapter's workloads; the adapter changes target identity\n")
	}
	_, err := io.WriteString(w, out.String())
	return err
}

func sortedPlatforms(anchors deterministicio.AdapterAnchors) []string {
	return slices.Sorted(maps.Keys(anchors.PreparedSourceSetSHA256))
}
