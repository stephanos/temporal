package packdev

import (
	"strings"
	"testing"
)

func TestRenderReviewIncludesGovernanceEvidenceAndSecurityLabels(t *testing.T) {
	request := validRequest()
	report, digest, err := RenderReview(request)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"# Compatibility Pack Review: example-pack",
		"Owner: `runtime-team`",
		"Platform: `darwin/arm64`",
		"Workload: `core-fixture`",
		"Target module: `example.com/main`",
		"Test arguments: `-test.run ^TestFixture$`",
		"Module: `example.com/dependency@v1.2.3`",
		"`runtime.go`",
		"`import:syscall`",
		"security-sensitive",
		digest,
	} {
		if !strings.Contains(string(report), expected) {
			t.Fatalf("review report omitted %q:\n%s", expected, report)
		}
	}
	if strings.Contains(string(report), "/Users/") {
		t.Fatalf("review report contains an operational path:\n%s", report)
	}
}
