package pinimpact_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
)

const (
	xsysPack    = "golang-x-sys-v047-darwin-arm64"
	xsysModule  = "golang.org/x/sys"
	xtermModule = "golang.org/x/term"
)

func ruleSumRequirements(t *testing.T) []requirement {
	t.Helper()
	pack := loadPack(t, xsysPack)
	activation := packModule(t, pack, xsysModule)
	for _, rule := range pack.Rules {
		if rule.Module.Path == xtermModule {
			return []requirement{activation, {path: rule.Module.Path, version: rule.Module.Version, sum: rule.Module.Sum}}
		}
	}
	t.Fatal("checked x/sys pack has no x/term rule")
	return nil
}

func TestMissingPackRuleSumIsUnknown(t *testing.T) {
	requirements := ruleSumRequirements(t)
	for _, baselineKind := range []string{"exact", "missing", "absent", "bumped"} {
		for _, includeAll := range []bool{false, true} {
			t.Run(fmt.Sprintf("baseline=%s/all=%t", baselineKind, includeAll), func(t *testing.T) {
				required := requirements
				switch baselineKind {
				case "absent":
					required = removeRequirement(required, xtermModule)
				case "bumped":
					required = replaceRequirement(required, xtermModule, func(required *requirement) { required.version = "v0.99.0" })
				}
				baseline := moduleFiles(t, required, "")
				if baselineKind == "missing" {
					baseline = withoutZipSum(t, baseline, requirements[1])
				}
				candidate := withoutZipSum(t, moduleFiles(t, requirements, ""), requirements[1])
				report, err := pinimpact.Evaluate(t.Context(), pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: candidate, Resolver: goModResolver{}, IncludeAll: includeAll})
				if err != nil {
					t.Fatal(err)
				}
				encoded, err := pinimpact.Encode(report)
				if err != nil {
					t.Fatal(err)
				}
				var decoded pinimpact.Report
				if err := json.Unmarshal(encoded, &decoded); err != nil {
					t.Fatal(err)
				}
				var human strings.Builder
				if err := pinimpact.Render(&human, report); err != nil {
					t.Fatal(err)
				}
				for _, files := range []pinimpact.ModuleFiles{baseline, candidate} {
					for name, want := range map[string][]byte{"go.mod": files.GoMod, "go.sum": files.GoSum} {
						got, err := os.ReadFile(filepath.Join(files.Directory, name))
						if err != nil || !slices.Equal(got, want) {
							t.Fatalf("report changed %s: %v", name, err)
						}
					}
				}
				if !includeAll {
					requirePins(t, decoded, map[string]pinimpact.Status{"pack-rule " + xsysPack + " " + xtermModule: pinimpact.StatusUnknown})
				}
				index := slices.IndexFunc(decoded.Pins, func(pin pinimpact.Pin) bool { return pin.Pack == xsysPack && pin.Module == xtermModule })
				if index < 0 {
					t.Fatal("report omits rule-only missing checksum")
				}
				pin := decoded.Pins[index]
				wantReason := "module_sum_missing: candidate go.sum has no sum for golang.org/x/term@v0.45.0"
				if pin.Status != pinimpact.StatusUnknown || pin.Reason != wantReason || !decoded.Invalidated {
					t.Errorf("rule = %+v, invalidated=%t; want unknown with %q", pin, decoded.Invalidated, wantReason)
				}
				if !strings.Contains(human.String(), "unknown pack-rule "+xsysPack+" "+xtermModule) || !strings.Contains(human.String(), wantReason) {
					t.Errorf("human report omits missing rule checksum diagnostic:\n%s", human.String())
				}
				if includeAll {
					count := 0
					for _, pin := range decoded.Pins {
						if pin.Pack != xsysPack || pin.Module == xtermModule {
							continue
						}
						count++
						want := pinimpact.StatusUnaffected
						if pin.Module == "golang.org/x/crypto" {
							want = pinimpact.StatusNotSelected
						}
						if pin.Status != want || pin.Reason != "" {
							t.Errorf("other checked pack rule = %+v, want %s without reason", pin, want)
						}
					}
					if count != 3 {
						t.Fatalf("other checked pack rule count=%d, want 3", count)
					}
				}
			})
		}
	}
}

func TestPackRuleSumControls(t *testing.T) {
	requirements := ruleSumRequirements(t)
	for _, test := range []struct {
		name, baseline, module, change string
		missingRule                    bool
		want                           pinimpact.Status
		wantReason                     string
	}{
		{name: "exact", want: pinimpact.StatusUnaffected},
		{name: "repaired candidate", baseline: "missing", want: pinimpact.StatusNotSelected},
		{name: "baseline rule absent", baseline: "absent", want: pinimpact.StatusNotSelected},
		{name: "baseline rule bumped", baseline: "bumped", want: pinimpact.StatusNotSelected},
		{name: "rule absent", module: xtermModule, change: "absent", want: pinimpact.StatusStale, wantReason: "candidate no longer requires golang.org/x/term"},
		{name: "rule version", module: xtermModule, change: "version", want: pinimpact.StatusInvalidated, wantReason: "candidate requires golang.org/x/term@v0.99.0; pinned v0.45.0"},
		{name: "rule sum", module: xtermModule, change: "sum", want: pinimpact.StatusInvalidated},
		{name: "rule replacement", module: xtermModule, change: "replacement", want: pinimpact.StatusInvalidated, wantReason: "candidate replaces golang.org/x/term"},
		{name: "activation absent", module: xsysModule, change: "absent", missingRule: true, want: pinimpact.StatusInvalidated, wantReason: "pack activation golang.org/x/sys: candidate does not require golang.org/x/sys"},
		{name: "activation version", module: xsysModule, change: "version", missingRule: true, want: pinimpact.StatusInvalidated, wantReason: "pack activation golang.org/x/sys: candidate requires golang.org/x/sys@v0.99.0; pinned v0.47.0"},
		{name: "activation sum", module: xsysModule, change: "sum", missingRule: true, want: pinimpact.StatusInvalidated},
		{name: "activation replacement", module: xsysModule, change: "replacement", missingRule: true, want: pinimpact.StatusInvalidated, wantReason: "pack activation golang.org/x/sys: candidate replaces golang.org/x/sys"},
	} {
		t.Run(test.name, func(t *testing.T) {
			baselineRequired := requirements
			switch test.baseline {
			case "absent":
				baselineRequired = removeRequirement(baselineRequired, xtermModule)
			case "bumped":
				baselineRequired = replaceRequirement(baselineRequired, xtermModule, func(required *requirement) { required.version = "v0.99.0" })
			}
			baseline := moduleFiles(t, baselineRequired, "")
			if test.baseline == "missing" {
				baseline = withoutZipSum(t, baseline, requirements[1])
			}
			required, extra := requirements, ""
			switch test.change {
			case "absent":
				required = removeRequirement(required, test.module)
			case "version":
				required = replaceRequirement(required, test.module, func(required *requirement) { required.version = "v0.99.0" })
			case "sum":
				required = replaceRequirement(required, test.module, func(required *requirement) { required.sum = fakeSum("changed rule control") })
				pinned := requirements[1]
				if test.module == xsysModule {
					pinned = requirements[0]
				}
				test.wantReason = fmt.Sprintf("candidate go.sum records %s for %s@%s; pinned %s", fakeSum("changed rule control"), pinned.path, pinned.version, pinned.sum)
				if test.module == xsysModule {
					test.wantReason = "pack activation golang.org/x/sys: " + test.wantReason
				}
			case "replacement":
				extra = "\nreplace " + test.module + " => ./replacement\n"
			}
			candidate := moduleFiles(t, required, extra)
			if test.missingRule {
				candidate = withoutZipSum(t, candidate, requirements[1])
			}
			for _, includeAll := range []bool{false, true} {
				t.Run(fmt.Sprintf("all=%t", includeAll), func(t *testing.T) {
					report, err := pinimpact.Evaluate(t.Context(), pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: candidate, Resolver: goModResolver{}, IncludeAll: includeAll})
					if err != nil {
						t.Fatal(err)
					}
					index := slices.IndexFunc(report.Pins, func(pin pinimpact.Pin) bool { return pin.Pack == xsysPack && pin.Module == xtermModule })
					if !includeAll && (test.want == pinimpact.StatusUnaffected || test.want == pinimpact.StatusNotSelected) {
						if index >= 0 || report.Invalidated {
							t.Fatalf("resolved non-actionable rule = %+v", report)
						}
					} else if index < 0 || report.Pins[index].Status != test.want || report.Pins[index].Reason != test.wantReason || report.Invalidated != (test.want == pinimpact.StatusInvalidated) {
						t.Fatalf("resolved rule report = %+v, want %s %q", report, test.want, test.wantReason)
					}
					logPackReportDigests(t, report)
				})
			}
		})
	}
}
