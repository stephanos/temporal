package compatibility

import (
	"bytes"
	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/record"
	"strings"
	"testing"
)

func TestGovernanceReviewTimePreservesGrammarAndPrecedence(t *testing.T) {
	base := PackGovernance{Owner: "owner", ReviewedAt: "2026-01-02T03:04:05Z", Justification: "reviewed", Workloads: []string{"unit"}, Platforms: []string{"linux/amd64"}, ApprovalSHA256: "sha256:" + strings.Repeat("a", 64)}
	for _, test := range []struct {
		text  string
		valid bool
	}{{"2026-01-02T03:04:05Z", true}, {"2026-01-02T03:04:05.123Z", true}, {"2026-01-02T03:04:05,123Z", true}, {"2026-01-02T3:04:05Z", true}, {"2026-01-02T03:04:05+00:00", false}, {"2026-01-02T03:04:05-08:00", false}, {"2026-01-02T03:04:05+24:00", false}, {"2026-01-02T03:04:05+00:60", false}, {"2026-02-30T03:04:05Z", false}, {"2026-01-02T03:04:05Z trailing", false}} {
		g := base
		g.ReviewedAt = test.text
		err := validatePackGovernance(g)
		if test.valid && err != nil {
			t.Fatalf("%q: %v", test.text, err)
		}
		if !test.valid && (err == nil || err.Error() != "compatibility pack governance review time is invalid") {
			t.Fatalf("%q: %v", test.text, err)
		}
	}
	base.Owner = ""
	base.ReviewedAt = "bad"
	if err := validatePackGovernance(base); err == nil || !strings.Contains(err.Error(), "owner") {
		t.Fatalf("owner precedence: %v", err)
	}
}

func TestGovernanceFullPackPreservesCanonicalBytesAndTimestampSpelling(t *testing.T) {
	input := strings.Replace(validPackV2, "2026-08-15T00:00:00Z", "2026-08-15T0:00:00,125Z", 1)
	pack, err := DecodePack([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"activation":[{"path":"example.com/dependency","replacement":{"kind":"none"},"sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","version":"v1.2.3"}],"governance":{"approval_sha256":"sha256:2222222222222222222222222222222222222222222222222222222222222222","justification":"Allows one exact reviewed import for the synthetic workload.","owner":"runtime-team","platforms":["darwin/arm64"],"reviewed_at":"2026-08-15T0:00:00,125Z","workloads":["core-fixture"]},"id":"example-pack","request_sha256":"sha256:1111111111111111111111111111111111111111111111111111111111111111","rules":[{"capabilities":["import:syscall"],"foreign_sources":[],"go_sources":[{"name":"runtime.go","sha256":"sha256:4444444444444444444444444444444444444444444444444444444444444444"}],"import_path":"example.com/dependency/internal/runtime","linknames":[],"module":{"path":"example.com/dependency","replacement":{"kind":"none"},"sum":"h1:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","version":"v1.2.3"},"source_set_sha256":"sha256:8ae49dab0499a1c49b23aac2cde0cd0c4edeb8e291faf0e53c0461ebd8416859"}],"schema":"gomad3.compatibility-pack/v2"}`
	actual, err := canonicaljson.CanonicalJSON(pack)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(actual, []byte(want)) {
		t.Fatalf("full pack bytes changed: %s", actual)
	}
	decoded, err := DecodePack(actual)
	if err != nil {
		t.Fatal(err)
	}
	again, err := canonicaljson.CanonicalJSON(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Governance.ReviewedAt != "2026-08-15T0:00:00,125Z" || record.HashBytes(again) != record.HashBytes([]byte(want)) {
		t.Fatal("pack spelling or canonical identity changed")
	}
}

func TestGovernanceValidationRetainsCompleteErrorPrecedence(t *testing.T) {
	pack, err := DecodePack([]byte(validPackV2))
	if err != nil {
		t.Fatal(err)
	}
	good := pack.Governance
	for _, test := range []struct {
		name, want string
		mutate     func(*PackGovernance)
	}{
		{"owner", "compatibility pack governance: owner is invalid", func(g *PackGovernance) { *g = PackGovernance{} }},
		{"justification", "compatibility pack governance: justification is invalid", func(g *PackGovernance) {
			g.Justification = ""
			g.ReviewedAt = "bad"
			g.Workloads = nil
			g.Platforms = nil
			g.ApprovalSHA256 = "bad"
		}},
		{"time", "compatibility pack governance review time is invalid", func(g *PackGovernance) {
			g.ReviewedAt = "bad"
			g.Workloads = nil
			g.Platforms = nil
			g.ApprovalSHA256 = "bad"
		}},
		{"workloads", "compatibility pack governance workloads are not canonical", func(g *PackGovernance) { g.Workloads = nil; g.Platforms = nil; g.ApprovalSHA256 = "bad" }},
		{"platforms", "compatibility pack governance platforms are not canonical", func(g *PackGovernance) { g.Platforms = nil; g.ApprovalSHA256 = "bad" }},
		{"approval", `compatibility pack governance approval is invalid: invalid SHA-256 "bad"`, func(g *PackGovernance) { g.ApprovalSHA256 = "bad" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			g := good
			test.mutate(&g)
			if err := validatePackGovernance(g); err == nil || err.Error() != test.want {
				t.Fatalf("precedence = %v, want %q", err, test.want)
			}
		})
	}
}
