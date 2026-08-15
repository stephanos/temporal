package packdev

import (
	"errors"
	"fmt"
	"strings"
)

const MaximumReportBytes = 16 << 20

func RenderReview(request Request) ([]byte, string, error) {
	if err := ValidateRequest(request); err != nil {
		return nil, "", err
	}
	digest, err := ApprovalSHA256(request)
	if err != nil {
		return nil, "", err
	}
	var report strings.Builder
	fmt.Fprintf(&report, "# Compatibility Pack Review: %s\n\n", request.ID)
	fmt.Fprintf(&report, "Review SHA-256: `%s`\n\n", digest)
	fmt.Fprintf(&report, "Owner: `%s`\n\nReviewed at: `%s`\n\n", request.Owner, request.ReviewedAt)
	fmt.Fprintf(&report, "Justification: %s\n\n", request.Justification)
	fmt.Fprintf(&report, "Target: `%s %s`\n\n", request.Target.Kind, request.Target.Package)
	fmt.Fprintf(&report, "Target module: `%s`\n\n", request.Target.ExpectedModule)
	fmt.Fprintf(&report, "Test arguments: `%s`\n\n", strings.Join(request.Target.TestArguments, " "))
	fmt.Fprintf(&report, "Build tags: `%s`\n\n", strings.Join(request.Target.BuildTags, ","))
	for _, platform := range request.Platforms {
		fmt.Fprintf(&report, "Platform: `%s`\n\n", platform)
	}
	for _, workload := range request.Workloads {
		fmt.Fprintf(&report, "Workload: `%s`\n\n", workload)
	}
	report.WriteString("## Activation\n\n")
	for _, activation := range request.Activation {
		fmt.Fprintf(&report, "- `%s@%s` (`%s`), replacement `%s`\n", activation.Evidence.Path, activation.Evidence.Version, activation.Evidence.Sum, activation.Evidence.Replacement.Kind)
		if adapter := activation.Evidence.Replacement.Adapter; adapter != nil {
			fmt.Fprintf(&report, "  - profile `%s` / `%s`\n", adapter.ProfileName, adapter.ProfileImplementationSHA256)
			fmt.Fprintf(&report, "  - adapter `%s@%s` / `%s`\n", adapter.Module, adapter.Version, adapter.Sum)
			fmt.Fprintf(&report, "  - source inventories `%s` → `%s`\n", adapter.OriginalSourceInventorySHA256, adapter.ReplacementSourceInventorySHA256)
			fmt.Fprintf(&report, "  - prepared source set `%s`\n", adapter.PreparedSourceSetSHA256)
		}
	}
	report.WriteString("\n## Reviewed packages\n")
	for _, pkg := range request.Packages {
		fmt.Fprintf(&report, "\n### `%s`\n\n", pkg.ImportPath)
		fmt.Fprintf(&report, "Module: `%s@%s` (`%s`), replacement `%s`\n\n", pkg.Evidence.Module.Path, pkg.Evidence.Module.Version, pkg.Evidence.Module.Sum, pkg.Evidence.Module.Replacement.Kind)
		fmt.Fprintf(&report, "Source set: `%s`\n\n", pkg.Evidence.SourceSetSHA256)
		report.WriteString("Go sources:\n\n")
		for _, source := range pkg.Evidence.GoSources {
			fmt.Fprintf(&report, "- `%s`: `%s`\n", source.Name, source.SHA256)
		}
		if len(pkg.Evidence.ForeignSources) != 0 {
			report.WriteString("\nForeign sources:\n\n")
			for _, source := range pkg.Evidence.ForeignSources {
				fmt.Fprintf(&report, "- `%s:%s`: `%s`\n", source.Kind, source.Name, source.SHA256)
			}
		}
		report.WriteString("\nRequested facts:\n\n")
		for _, fact := range pkg.Facts {
			identity := fact.Capability
			if fact.Kind == FactLinkname {
				identity = "linkname:" + fact.Source
			}
			label := ""
			if securitySensitive(identity) {
				label = " — **security-sensitive**"
			}
			fmt.Fprintf(&report, "- `%s`: **%s**%s\n", identity, fact.Disposition, label)
			if fact.Kind == FactLinkname {
				fmt.Fprintf(&report, "  - source `%s`\n", fact.SHA256)
				for _, directive := range fact.Directives {
					fmt.Fprintf(&report, "  - directive `%s`\n", directive)
				}
			}
		}
	}
	report.WriteByte('\n')
	if report.Len() > MaximumReportBytes {
		return nil, "", errors.New("compatibility-pack review exceeds its size bound")
	}
	return []byte(report.String()), digest, nil
}

func securitySensitive(capability string) bool {
	return strings.HasPrefix(capability, "import:syscall") ||
		strings.HasPrefix(capability, "import:os/exec") ||
		strings.HasPrefix(capability, "import:golang.org/x/sys/") ||
		strings.HasPrefix(capability, "foreign:") ||
		strings.HasPrefix(capability, "linkname:")
}
