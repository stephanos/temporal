package pinimpact_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/canonicaljson"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
)

func withoutZipSum(t *testing.T, files pinimpact.ModuleFiles, required requirement) pinimpact.ModuleFiles {
	t.Helper()
	line := fmt.Sprintf("%s %s %s\n", required.path, required.version, required.sum)
	contents := strings.Replace(string(files.GoSum), line, "", 1)
	if contents == string(files.GoSum) || !strings.Contains(contents, required.path+" "+required.version+"/go.mod ") {
		t.Fatal("fixture must omit only the zip sum and retain the go.mod sum")
	}
	files.GoSum = []byte(contents)
	if err := os.WriteFile(filepath.Join(files.Directory, "go.sum"), files.GoSum, 0o600); err != nil {
		t.Fatal(err)
	}
	return files
}

func TestMissingPackActivationSumIsUnknown(t *testing.T) {
	requirements := baselineRequirements(t)
	compress := packModule(t, loadPack(t, xxhashPack), compressModule)
	for _, name := range []string{"both missing", "candidate missing", "baseline bumped"} {
		t.Run(name, func(t *testing.T) {
			baselineRequirements := requirements
			if name == "baseline bumped" {
				baselineRequirements = replaceRequirement(requirements, compressModule, func(required *requirement) { required.version = "v1.18.6" })
			}
			baseline := moduleFiles(t, baselineRequirements, "")
			if name == "both missing" {
				baseline = withoutZipSum(t, baseline, compress)
			}
			candidate := withoutZipSum(t, moduleFiles(t, requirements, ""), compress)
			before := []pinimpact.ModuleFiles{baseline, candidate}
			report := evaluate(t, baseline, candidate, goModResolver{})
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
			want := map[string]pinimpact.Status{"pack-rule " + compressRule: pinimpact.StatusUnknown, "pack-rule " + xxhashRule: pinimpact.StatusUnknown}
			requirePins(t, decoded, want)
			if !decoded.Invalidated {
				t.Fatal("missing activation identity must invalidate the report")
			}
			for _, pin := range decoded.Pins {
				if !strings.Contains(pin.Reason, "module_sum_missing") || !strings.Contains(pin.Reason, compressModule+"@"+compress.version) {
					t.Errorf("rule %s has no missing activation sum diagnostic: %q", pin.ID, pin.Reason)
				}
				if !strings.Contains(human.String(), "unknown pack-rule "+pin.ID) || !strings.Contains(human.String(), pin.Reason) {
					t.Errorf("human report omits unknown rule diagnostic: %s", human.String())
				}
			}
			for _, files := range before {
				for name, want := range map[string][]byte{"go.mod": files.GoMod, "go.sum": files.GoSum} {
					got, err := os.ReadFile(filepath.Join(files.Directory, name))
					if err != nil || !slices.Equal(got, want) {
						t.Fatalf("report changed %s: %v", name, err)
					}
				}
			}
		})
	}
}

func TestPackActivationSumControls(t *testing.T) {
	requirements := baselineRequirements(t)
	compress := packModule(t, loadPack(t, xxhashPack), compressModule)
	for _, test := range []struct {
		name            string
		missingBaseline bool
		absent          bool
		want            pinimpact.Status
	}{
		{name: "exact", want: pinimpact.StatusUnaffected},
		{name: "repaired candidate", missingBaseline: true, want: pinimpact.StatusNotSelected},
		{name: "absent", absent: true, want: pinimpact.StatusNotSelected},
	} {
		t.Run(test.name, func(t *testing.T) {
			required := requirements
			if test.absent {
				required = removeRequirement(required, compressModule)
			}
			baseline, candidate := moduleFiles(t, required, ""), moduleFiles(t, required, "")
			if test.missingBaseline {
				baseline = withoutZipSum(t, baseline, compress)
			}
			report, err := pinimpact.Evaluate(t.Context(), pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: candidate, Resolver: goModResolver{}, IncludeAll: true})
			if err != nil {
				t.Fatal(err)
			}
			count := 0
			for _, pin := range report.Pins {
				if pin.Pack == xxhashPack {
					count++
					if pin.Status != test.want || pin.Reason != "" {
						t.Errorf("rule %s = %s %q, want %s without a reason", pin.ID, pin.Status, pin.Reason, test.want)
					}
				}
			}
			if count != 2 || report.Invalidated {
				t.Fatalf("fully known candidate pack count=%d, invalidated=%t", count, report.Invalidated)
			}
			logPackReportDigests(t, report)
		})
	}
}

func TestKnownPackActivationExclusionWinsOverMissingSum(t *testing.T) {
	requirements := baselineRequirements(t)
	pack := loadPack(t, xxhashPack)
	for _, missing := range []string{xxhashModule, compressModule} {
		other := compressModule
		if missing == compressModule {
			other = xxhashModule
		}
		for _, kind := range []string{"absent", "version", "sum", "replacement"} {
			for _, baselineMissing := range []bool{false, true} {
				t.Run(fmt.Sprintf("missing=%s/exclusion=%s/baselineMissing=%t", missing, kind, baselineMissing), func(t *testing.T) {
					baseline := moduleFiles(t, requirements, "")
					if baselineMissing {
						baseline = withoutZipSum(t, baseline, packModule(t, pack, missing))
					}
					required, extra := requirements, ""
					switch kind {
					case "absent":
						required = removeRequirement(required, other)
					case "version":
						required = replaceRequirement(required, other, func(required *requirement) {
							required.version = "v1.99.0"
							if other == xxhashModule {
								required.version = "v2.99.0"
							}
						})
					case "sum":
						required = replaceRequirement(required, other, func(required *requirement) { required.sum = fakeSum("changed") })
					case "replacement":
						extra = "\nreplace " + other + " => ./replacement\n"
					}
					resolved := moduleFiles(t, required, extra)
					candidate := withoutZipSum(t, moduleFiles(t, required, extra), packModule(t, pack, missing))
					spec := pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: resolved, Resolver: goModResolver{}, IncludeAll: true}
					control, err := pinimpact.Evaluate(t.Context(), spec)
					if err != nil {
						t.Fatal(err)
					}
					for _, pin := range control.Pins {
						if pin.Pack != xxhashPack {
							continue
						}
						want := pinimpact.StatusInvalidated
						if kind == "absent" {
							if baselineMissing {
								want = pinimpact.StatusNotSelected
							} else if pin.Module == other {
								want = pinimpact.StatusStale
							}
						}
						if pin.Status != want || (pin.Reason == "") != (want == pinimpact.StatusNotSelected) {
							t.Fatalf("resolved exclusion rule = %+v, want %s", pin, want)
						}
					}
					logPackReportDigests(t, control)
					spec.Candidate = candidate
					actual, err := pinimpact.Evaluate(t.Context(), spec)
					if err != nil {
						t.Fatal(err)
					}
					for index, pin := range actual.Pins {
						if pin.Class == pinimpact.ClassPackRule && (pin.Status == pinimpact.StatusUnknown || pin.Status != control.Pins[index].Status || pin.Reason != control.Pins[index].Reason) {
							t.Errorf("rule %s = %s %q, want resolved exclusion %s %q", pin.ID, pin.Status, pin.Reason, control.Pins[index].Status, control.Pins[index].Reason)
						}
					}
				})
			}
		}
	}
}

func logPackReportDigests(t *testing.T, report pinimpact.Report) {
	t.Helper()
	encoded, err := pinimpact.Encode(report)
	if err != nil {
		t.Fatal(err)
	}
	var rendered strings.Builder
	if err := pinimpact.Render(&rendered, report); err != nil {
		t.Fatal(err)
	}
	t.Logf("resolved canonical=%x human=%x", sha256.Sum256(encoded), sha256.Sum256([]byte(rendered.String())))
}

func TestPackRuleExclusionWinsOverUnknownActivation(t *testing.T) {
	pack := loadPack(t, xxhashPack)
	pack.Activation = pack.Activation[1:]
	pack.Rules = pack.Rules[:1]
	encoded, err := canonicaljson.CanonicalJSON(pack)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err := os.WriteFile(filepath.Join(directory, pack.ID+".json"), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	requirements := baselineRequirements(t)
	compress := packModule(t, pack, compressModule)
	for _, baselineMissing := range []bool{false, true} {
		for _, kind := range []string{"exact", "missing", "absent", "version", "sum", "replacement"} {
			t.Run(fmt.Sprintf("baselineMissing=%t/rule=%s", baselineMissing, kind), func(t *testing.T) {
				baseline := moduleFiles(t, requirements, "")
				if baselineMissing {
					baseline = withoutZipSum(t, baseline, compress)
				}
				required, extra := requirements, ""
				switch kind {
				case "absent":
					required = removeRequirement(required, xxhashModule)
				case "version":
					required = replaceRequirement(required, xxhashModule, func(required *requirement) { required.version = "v2.99.0" })
				case "sum":
					required = replaceRequirement(required, xxhashModule, func(required *requirement) { required.sum = fakeSum("changed rule") })
				case "replacement":
					extra = "\nreplace " + xxhashModule + " => ./replacement\n"
				}
				candidate := withoutZipSum(t, moduleFiles(t, required, extra), compress)
				if kind == "missing" {
					candidate = withoutZipSum(t, candidate, requirements[1])
				}
				report, err := pinimpact.Evaluate(t.Context(), pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: candidate, Resolver: goModResolver{}, PacksDirectory: directory, IncludeAll: true})
				if err != nil {
					t.Fatal(err)
				}
				want := pinimpact.StatusUnknown
				if kind != "exact" && kind != "missing" {
					want = pinimpact.StatusInvalidated
					if baselineMissing {
						want = pinimpact.StatusNotSelected
					} else if kind == "absent" {
						want = pinimpact.StatusStale
					}
				}
				index := slices.IndexFunc(report.Pins, func(pin pinimpact.Pin) bool { return pin.Pack == xxhashPack })
				if index < 0 {
					t.Fatal("separate rule report has no pack rule")
				}
				t.Logf("separate rule disposition=%s reason=%q", report.Pins[index].Status, report.Pins[index].Reason)
				if report.Pins[index].Status != want {
					t.Fatalf("separate rule = %+v, want %s", report.Pins[index], want)
				}
			})
		}
	}
}

type candidateGraphErrorResolver struct {
	goModResolver
	directory string
}

func (resolver candidateGraphErrorResolver) Resolve(ctx context.Context, files pinimpact.ModuleFiles) (map[string]string, error) {
	if files.Directory == resolver.directory {
		return nil, &pinimpact.NewerGoError{Version: "1.99.0", Detail: "fixture dependency requires newer Go"}
	}
	return resolver.goModResolver.Resolve(ctx, files)
}

func TestPackActivationGraphUnknownRetainsReason(t *testing.T) {
	requirements := baselineRequirements(t)
	baseline, candidate := moduleFiles(t, requirements, ""), moduleFiles(t, requirements, "")
	report := evaluate(t, baseline, candidate, candidateGraphErrorResolver{directory: candidate.Directory})
	count := 0
	for _, pin := range report.Pins {
		if pin.Pack == xxhashPack {
			count++
			if pin.Status != pinimpact.StatusUnknown || pin.Reason != "candidate module graph requires go 1.99.0, newer than the pinned go1.27.1, so it cannot be resolved; a Go upgrade re-derives this pin through upgrade-dossier" {
				t.Errorf("graph-unknown rule = %+v", pin)
			}
		}
	}
	if count != 2 || !report.Invalidated {
		t.Fatalf("graph-unknown report = %+v", report)
	}
}
