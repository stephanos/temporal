package pinimpact_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
)

func TestPortableFixturePinDecisions(t *testing.T) {
	requirements := baselineRequirements(t)
	pack := loadPack(t, xxhashPack)
	for _, test := range []struct {
		name   string
		change func(*requirement)
		extra  string
		reason string
	}{
		{
			name: "version bump",
			change: func(required *requirement) {
				if required.path == sentryModule {
					required.version, required.sum = "v0.47.0", fakeSum("sentry v0.47.0")
				} else {
					required.version, required.sum = "v1.18.6", fakeSum("compress v1.18.6")
				}
			},
			reason: "candidate requires " + sentryModule + "@v0.47.0",
		},
		{
			name: "changed sum",
			change: func(required *requirement) {
				required.sum = fakeSum(required.path + " modified")
			},
			reason: "go.sum records",
		},
		{
			name:   "replacement",
			extra:  fmt.Sprintf("\nreplace %s => ./sentry\n\nreplace %s %s => %s v1.18.6\n", sentryModule, compressModule, packModule(t, pack, compressModule).version, compressModule),
			reason: "replaces",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			baseline := moduleFiles(t, requirements, "")
			changed := requirements
			if test.change != nil {
				changed = replaceRequirement(changed, sentryModule, test.change)
				changed = replaceRequirement(changed, compressModule, test.change)
			}
			candidate := moduleFiles(t, changed, test.extra)
			report := evaluate(t, baseline, candidate, goModResolver{})
			requirePins(t, report, map[string]pinimpact.Status{
				"adapter " + adapterPinID(t): pinimpact.StatusInvalidated,
				"pack-rule " + compressRule:  pinimpact.StatusInvalidated,
				"pack-rule " + xxhashRule:    pinimpact.StatusInvalidated,
			})
			if !report.Invalidated || !strings.Contains(pinReason(t, report, pinimpact.ClassAdapter, adapterPinID(t)), test.reason) {
				t.Fatalf("report = %+v, want actionable adapter %s", report, test.reason)
			}
			if reason := pinReason(t, report, pinimpact.ClassPackRule, xxhashRule); !strings.Contains(reason, "pack activation "+compressModule) {
				t.Fatalf("xxhash reason = %q, want changed activation", reason)
			}
			requirePackDecisions(t, pack, requirements, true)
			if test.change != nil {
				requirePackDecisions(t, pack, changed, false)
			} else {
				requireReplacedPackRejection(t, pack)
			}
			for _, files := range []pinimpact.ModuleFiles{baseline, candidate} {
				for name, want := range map[string][]byte{"go.mod": files.GoMod, "go.sum": files.GoSum} {
					got, err := os.ReadFile(filepath.Join(files.Directory, name))
					if err != nil || string(got) != string(want) {
						t.Fatalf("%s changed: %q, %v", name, got, err)
					}
				}
			}
		})
	}
}

func requireReplacedPackRejection(t *testing.T, pack compatibility.Pack) {
	t.Helper()
	packs, err := compatibility.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	var packages []compatibility.Package
	for _, rule := range pack.Rules {
		pkg := compatibility.Package{
			ImportPath: rule.ImportPath, SourceSetSHA256: rule.SourceSetSHA256,
			Module: compatibility.Module{Path: rule.Module.Path, Version: rule.Module.Version, Sum: rule.Module.Sum, Replaced: rule.Module.Path == compressModule},
		}
		for _, source := range rule.GoSources {
			pkg.GoSources = append(pkg.GoSources, compatibility.Source(source))
		}
		for _, source := range rule.ForeignSources {
			pkg.ForeignSources = append(pkg.ForeignSources, compatibility.ForeignSource(source))
		}
		packages = append(packages, pkg)
	}
	for _, platform := range pack.Governance.Platforms {
		selection, err := compatibility.SelectPacksForPlatform(packs, packages, platform)
		if err != nil {
			t.Fatal(err)
		}
		for index, rule := range pack.Rules {
			if selection.AllowsCapability(packages[index], rule.Capabilities[0]) {
				t.Fatalf("replaced activation admitted %s on %s", rule.ImportPath, platform)
			}
		}
	}
}
