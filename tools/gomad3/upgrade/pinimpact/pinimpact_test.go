package pinimpact_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/deterministicio"
	compatibility "go.temporal.io/server/tools/gomad3/internal/compatibilitypack"
	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
	"go.temporal.io/server/tools/gomad3/upgrade/pinimpact"
	"golang.org/x/mod/modfile"
)

const (
	sentryModule   = "github.com/getsentry/sentry-go"
	xxhashModule   = "github.com/cespare/xxhash/v2"
	compressModule = "github.com/klauspost/compress"
	xxhashPack     = "temporal-leaf-xxhash-darwin-arm64"
	compressRule   = xxhashPack + " github.com/klauspost/compress/zstd/internal/xxhash"
	xxhashRule     = xxhashPack + " github.com/cespare/xxhash/v2"
)

// requirement is one go.mod requirement with the zip sum go.sum records.
type requirement struct {
	path, version, sum string
	indirect           bool
}

// goModResolver selects exactly what go.mod requires, as the go command does
// for a tidy module, except for the overrides it applies to the module in
// overridden.
type goModResolver struct {
	overridden string
	overrides  map[string]string
	err        error
}

func (resolver goModResolver) Resolve(_ context.Context, files pinimpact.ModuleFiles) (map[string]string, error) {
	if resolver.err != nil {
		return nil, resolver.err
	}
	parsed, err := modfile.Parse("go.mod", files.GoMod, nil)
	if err != nil {
		return nil, err
	}
	selected := map[string]string{}
	for _, required := range parsed.Require {
		selected[required.Mod.Path] = required.Mod.Version
	}
	if files.Directory == resolver.overridden {
		for path, version := range resolver.overrides {
			selected[path] = version
		}
	}
	return selected, nil
}

func fakeSum(name string) string {
	digest := sha256.Sum256([]byte(name))
	return "h1:" + base64.StdEncoding.EncodeToString(digest[:])
}

func gomadRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func adapterIdentity(t *testing.T, path string) gomadversion.AdapterIdentity {
	t.Helper()
	for _, identity := range deterministicio.Default().Adapters() {
		if identity.Module == path {
			return identity
		}
	}
	t.Fatalf("adapter registry has no %s", path)
	return gomadversion.AdapterIdentity{}
}

func loadPack(t *testing.T, id string) compatibility.Pack {
	t.Helper()
	packs, err := compatibility.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	for _, validated := range packs {
		if validated.Pack().ID == id {
			return validated.Pack()
		}
	}
	t.Fatalf("compatibility pack %s is not loaded", id)
	return compatibility.Pack{}
}

func packModule(t *testing.T, pack compatibility.Pack, path string) requirement {
	t.Helper()
	for _, activation := range pack.Activation {
		if activation.Path == path {
			return requirement{path: activation.Path, version: activation.Version, sum: activation.Sum, indirect: true}
		}
	}
	t.Fatalf("pack %s does not activate on %s", pack.ID, path)
	return requirement{}
}

// baselineRequirements selects the sentry adapter and exactly one pack: the
// xxhash pack activates on its two modules, and every other pack needs a
// module this baseline lacks.
func baselineRequirements(t *testing.T) []requirement {
	t.Helper()
	sentry := adapterIdentity(t, sentryModule)
	pack := loadPack(t, xxhashPack)
	return []requirement{
		{path: sentry.Module, version: sentry.Version, sum: sentry.Sum},
		packModule(t, pack, xxhashModule),
		packModule(t, pack, compressModule),
	}
}

func moduleFiles(t *testing.T, requirements []requirement, extra string) pinimpact.ModuleFiles {
	t.Helper()
	var goMod, goSum strings.Builder
	goMod.WriteString("module example.test/pinimpact\n\ngo 1.27.0\n\nrequire (\n")
	for _, required := range requirements {
		comment := ""
		if required.indirect {
			comment = " // indirect"
		}
		fmt.Fprintf(&goMod, "\t%s %s%s\n", required.path, required.version, comment)
		fmt.Fprintf(&goSum, "%s %s %s\n", required.path, required.version, required.sum)
		fmt.Fprintf(&goSum, "%s %s/go.mod %s\n", required.path, required.version, fakeSum(required.path+"/go.mod"))
	}
	goMod.WriteString(")\n" + extra)
	directory := t.TempDir()
	files := pinimpact.ModuleFiles{GoMod: []byte(goMod.String()), GoSum: []byte(goSum.String()), Directory: directory}
	for name, contents := range map[string][]byte{"go.mod": files.GoMod, "go.sum": files.GoSum} {
		if err := os.WriteFile(filepath.Join(directory, name), contents, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return files
}

func replaceRequirement(requirements []requirement, path string, change func(*requirement)) []requirement {
	result := slices.Clone(requirements)
	for index := range result {
		if result[index].path == path {
			change(&result[index])
		}
	}
	return result
}

func removeRequirement(requirements []requirement, path string) []requirement {
	return slices.DeleteFunc(slices.Clone(requirements), func(required requirement) bool { return required.path == path })
}

func evaluate(t *testing.T, baseline, candidate pinimpact.ModuleFiles, resolver pinimpact.Resolver) pinimpact.Report {
	t.Helper()
	report, err := pinimpact.Evaluate(context.Background(), pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: candidate, Resolver: resolver})
	if err != nil {
		t.Fatal(err)
	}
	return report
}

// pinStatuses maps each reported pin to its status.
func pinStatuses(report pinimpact.Report) map[string]pinimpact.Status {
	statuses := map[string]pinimpact.Status{}
	for _, pin := range report.Pins {
		statuses[string(pin.Class)+" "+pin.ID] = pin.Status
	}
	return statuses
}

func requirePins(t *testing.T, report pinimpact.Report, want map[string]pinimpact.Status) {
	t.Helper()
	got := pinStatuses(report)
	if len(got) != len(want) {
		t.Fatalf("reported pins = %v, want %v", got, want)
	}
	for id, status := range want {
		if got[id] != status {
			t.Fatalf("reported pins = %v, want %v", got, want)
		}
	}
}

func pinReason(t *testing.T, report pinimpact.Report, class pinimpact.Class, id string) string {
	t.Helper()
	for _, pin := range report.Pins {
		if pin.Class == class && pin.ID == id {
			return pin.Reason
		}
	}
	t.Fatalf("report has no %s %s", class, id)
	return ""
}

func adapterPinID(t *testing.T) string {
	identity := adapterIdentity(t, sentryModule)
	return identity.Module + "@" + identity.Version
}

func TestUnchangedModuleInvalidatesNothing(t *testing.T) {
	baseline := moduleFiles(t, baselineRequirements(t), "")
	report := evaluate(t, baseline, baseline, goModResolver{})
	if report.Invalidated || len(report.Pins) != 0 {
		t.Fatalf("report = %+v, want no invalidated pin", report)
	}
	for _, summary := range report.Classes {
		if summary.Total == 0 || summary.Invalidated+summary.Unknown+summary.Stale != 0 {
			t.Fatalf("class summary = %+v", summary)
		}
	}
	if report.Classes[0].Unaffected != 1 || report.Classes[1].Unaffected != len(loadPack(t, xxhashPack).Rules) {
		t.Fatalf("selected pins = %+v, want the sentry adapter and the xxhash pack", report.Classes[:2])
	}
}

// TestFixtureBumpMatchesBuildRejections bumps one adapted and one packed
// module and checks that the report names exactly the pins the adapter
// registry and pack selection then reject.
func TestFixtureBumpMatchesBuildRejections(t *testing.T) {
	requirements := baselineRequirements(t)
	baseline := moduleFiles(t, requirements, "")
	bumped := replaceRequirement(requirements, sentryModule, func(required *requirement) {
		required.version, required.sum = "v0.47.0", fakeSum("sentry v0.47.0")
	})
	bumped = replaceRequirement(bumped, compressModule, func(required *requirement) {
		required.version, required.sum = "v1.18.6", fakeSum("compress v1.18.6")
	})
	candidate := moduleFiles(t, bumped, "")
	report := evaluate(t, baseline, candidate, goModResolver{})
	requirePins(t, report, map[string]pinimpact.Status{
		"adapter " + adapterPinID(t): pinimpact.StatusInvalidated,
		"pack-rule " + compressRule:  pinimpact.StatusInvalidated,
		"pack-rule " + xxhashRule:    pinimpact.StatusInvalidated,
	})
	if !report.Invalidated {
		t.Fatal("report does not mark the candidate invalidated")
	}
	if reason := pinReason(t, report, pinimpact.ClassPackRule, xxhashRule); !strings.Contains(reason, "pack activation "+compressModule) {
		t.Fatalf("unchanged xxhash rule reason = %q, want the pack's activation change", reason)
	}

	requireAdapterRejection(t, candidate.Directory, "unsupported "+sentryModule+" version")
	requireAdapterAcceptance(t, baseline.Directory)
	pack := loadPack(t, xxhashPack)
	requirePackDecisions(t, pack, requirements, true)
	requirePackDecisions(t, pack, bumped, false)
}

// TestCommittedBumpKeepsTheStrandedPackInvalidated evaluates the bump once it
// is the baseline too. The pack pinned to the old version is then selected by
// neither side, but the module still requires its activation module, so the
// pack is invalidated rather than not selected.
func TestCommittedBumpKeepsTheStrandedPackInvalidated(t *testing.T) {
	bumped := replaceRequirement(baselineRequirements(t), compressModule, func(required *requirement) {
		required.version, required.sum = "v1.18.6", fakeSum("compress v1.18.6")
	})
	committed := moduleFiles(t, bumped, "")
	report := evaluate(t, committed, committed, goModResolver{})
	requirePins(t, report, map[string]pinimpact.Status{
		"pack-rule " + compressRule: pinimpact.StatusInvalidated,
		"pack-rule " + xxhashRule:   pinimpact.StatusInvalidated,
	})
	if reason := pinReason(t, report, pinimpact.ClassPackRule, xxhashRule); !strings.Contains(reason, "pack activation "+compressModule) {
		t.Fatalf("stranded xxhash rule reason = %q, want the pack's activation change", reason)
	}
	// Without the activation module the pack is simply not selected.
	removed := moduleFiles(t, removeRequirement(bumped, compressModule), "")
	if report := evaluate(t, removed, removed, goModResolver{}); report.Invalidated || len(report.Pins) != 0 {
		t.Fatalf("report without the activation module = %+v", report.Pins)
	}
}

func TestSameVersionWithChangedSum(t *testing.T) {
	requirements := baselineRequirements(t)
	changed := replaceRequirement(requirements, sentryModule, func(required *requirement) { required.sum = fakeSum("sentry modified") })
	changed = replaceRequirement(changed, compressModule, func(required *requirement) { required.sum = fakeSum("compress modified") })
	candidate := moduleFiles(t, changed, "")
	report := evaluate(t, moduleFiles(t, requirements, ""), candidate, goModResolver{})
	requirePins(t, report, map[string]pinimpact.Status{
		"adapter " + adapterPinID(t): pinimpact.StatusInvalidated,
		"pack-rule " + compressRule:  pinimpact.StatusInvalidated,
		"pack-rule " + xxhashRule:    pinimpact.StatusInvalidated,
	})
	if reason := pinReason(t, report, pinimpact.ClassAdapter, adapterPinID(t)); !strings.Contains(reason, "go.sum records") {
		t.Fatalf("adapter reason = %q, want the changed sum", reason)
	}
	requireAdapterRejection(t, candidate.Directory, "module sum")
	requirePackDecisions(t, loadPack(t, xxhashPack), changed, false)
}

func TestRemovedModules(t *testing.T) {
	requirements := baselineRequirements(t)
	baseline := moduleFiles(t, requirements, "")

	withoutAdapter := evaluate(t, baseline, moduleFiles(t, removeRequirement(requirements, sentryModule), ""), goModResolver{})
	requirePins(t, withoutAdapter, map[string]pinimpact.Status{"adapter " + adapterPinID(t): pinimpact.StatusStale})
	if withoutAdapter.Invalidated {
		t.Fatal("a removed adapted module invalidated the candidate although the build stops using its adapter")
	}

	withoutPackModule := evaluate(t, baseline, moduleFiles(t, removeRequirement(requirements, compressModule), ""), goModResolver{})
	requirePins(t, withoutPackModule, map[string]pinimpact.Status{
		"pack-rule " + compressRule: pinimpact.StatusStale,
		"pack-rule " + xxhashRule:   pinimpact.StatusInvalidated,
	})
}

func TestReplacedModules(t *testing.T) {
	requirements := baselineRequirements(t)
	compress := packModule(t, loadPack(t, xxhashPack), compressModule)
	candidate := moduleFiles(t, requirements, fmt.Sprintf("\nreplace %s => ./sentry\n\nreplace %s %s => %s v1.18.6\n", sentryModule, compressModule, compress.version, compressModule))
	report := evaluate(t, moduleFiles(t, requirements, ""), candidate, goModResolver{})
	requirePins(t, report, map[string]pinimpact.Status{
		"adapter " + adapterPinID(t): pinimpact.StatusInvalidated,
		"pack-rule " + compressRule:  pinimpact.StatusInvalidated,
		"pack-rule " + xxhashRule:    pinimpact.StatusInvalidated,
	})
	if reason := pinReason(t, report, pinimpact.ClassAdapter, adapterPinID(t)); !strings.Contains(reason, "replaces") {
		t.Fatalf("adapter reason = %q, want the replacement", reason)
	}
	requireAdapterRejection(t, candidate.Directory, "already replaces "+sentryModule)
}

func TestIndirectOnlyBump(t *testing.T) {
	requirements := baselineRequirements(t)
	baseline := moduleFiles(t, requirements, "")
	want := map[string]pinimpact.Status{
		"pack-rule " + compressRule: pinimpact.StatusInvalidated,
		"pack-rule " + xxhashRule:   pinimpact.StatusInvalidated,
	}

	// A tidy bump rewrites the indirect requirement.
	bumped := replaceRequirement(requirements, compressModule, func(required *requirement) {
		required.version, required.sum = "v1.18.6", fakeSum("compress v1.18.6")
	})
	requirePins(t, evaluate(t, baseline, moduleFiles(t, bumped, ""), goModResolver{}), want)

	// An untidy bump leaves go.mod alone while a dependency's requirement
	// raises the version the module graph selects.
	unchanged := moduleFiles(t, requirements, "")
	untidy := evaluate(t, baseline, unchanged, goModResolver{overridden: unchanged.Directory, overrides: map[string]string{compressModule: "v1.18.6"}})
	requirePins(t, untidy, want)
	if reason := pinReason(t, untidy, pinimpact.ClassPackRule, compressRule); !strings.Contains(reason, "module graph selects "+compressModule+"@v1.18.6") {
		t.Fatalf("untidy reason = %q", reason)
	}
}

func TestUnknownPinsCountAsInvalidated(t *testing.T) {
	requirements := baselineRequirements(t)
	baseline := moduleFiles(t, requirements, "")

	newerGo := moduleFiles(t, requirements, "")
	newerGo.GoMod = []byte(strings.Replace(string(newerGo.GoMod), "go 1.27.0", "go 1.99.0", 1))
	report := evaluate(t, baseline, newerGo, goModResolver{})
	for _, summary := range report.Classes {
		if summary.Class == pinimpact.ClassInterception || summary.Class == pinimpact.ClassClockReference {
			if summary.Total == 0 || summary.Unknown != summary.Total || summary.Unaffected != 0 {
				t.Fatalf("%s summary = %+v, want every pin unknown", summary.Class, summary)
			}
		}
	}
	if !report.Invalidated {
		t.Fatal("unknown toolchain pins did not invalidate the candidate")
	}

	emptyRoot := t.TempDir()
	missingRoot, err := pinimpact.Evaluate(context.Background(), pinimpact.Spec{Root: emptyRoot, Baseline: baseline, Candidate: baseline, Resolver: goModResolver{}})
	if err != nil {
		t.Fatal(err)
	}
	if encoded, err := pinimpact.Encode(missingRoot); err != nil || strings.Contains(string(encoded), emptyRoot) {
		t.Fatalf("encoded report = %s, %v; want no host path", encoded, err)
	}
	requirePins(t, missingRoot, map[string]pinimpact.Status{
		"interception-fingerprint deterministicio/boundary/manifest.json": pinimpact.StatusUnknown,
		"clock-reference toolchain/clock_inventory_test.go":               pinimpact.StatusUnknown,
	})
	if !missingRoot.Invalidated {
		t.Fatal("unevaluated pins did not invalidate the candidate")
	}

	missingPacks := filepath.Join(t.TempDir(), "missing")
	t.Setenv(compatibility.ExternalPacksEnvironment, missingPacks)
	unloadable := evaluate(t, baseline, baseline, goModResolver{})
	requirePins(t, unloadable, map[string]pinimpact.Status{"pack-rule compatibility packs": pinimpact.StatusUnknown})
	if encoded, err := pinimpact.Encode(unloadable); err != nil || strings.Contains(string(encoded), missingPacks) {
		t.Fatalf("encoded report = %s, %v; want no host path", encoded, err)
	}
}

func TestInvalidInputAndResolutionFailures(t *testing.T) {
	baseline := moduleFiles(t, baselineRequirements(t), "")
	for name, candidate := range map[string]pinimpact.ModuleFiles{
		"malformed go.mod": {GoMod: []byte("module\n")},
		"no module":        {GoMod: []byte("go 1.27.0\n")},
		"malformed go.sum": {GoMod: baseline.GoMod, GoSum: []byte("github.com/a v1.0.0\n")},
	} {
		_, err := pinimpact.Evaluate(context.Background(), pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: candidate, Resolver: goModResolver{}})
		if !pinimpact.IsInputError(err) {
			t.Fatalf("%s: Evaluate() error = %v, want invalid input", name, err)
		}
	}
	unavailable := errors.New("proxy unavailable")
	_, err := pinimpact.Evaluate(context.Background(), pinimpact.Spec{Root: gomadRoot(t), Baseline: baseline, Candidate: baseline, Resolver: goModResolver{err: unavailable}})
	if !errors.Is(err, unavailable) || pinimpact.IsInputError(err) {
		t.Fatalf("Evaluate() error = %v, want an infrastructure failure", err)
	}
}

// requireAdapterRejection runs the adapter registry's build check on the
// module in directory and requires it to reject the configuration.
func requireAdapterRejection(t *testing.T, directory, want string) {
	t.Helper()
	_, _, err := deterministicio.Default().PrepareTargetBuildAdapters(context.Background(), target.Spec{
		PreparationRoot: t.TempDir(), WorkingDir: directory, ToolchainRoot: t.TempDir(),
	})
	if !deterministicio.IsInvalidBuildAdapterConfiguration(err) || !strings.Contains(err.Error(), want) {
		t.Fatalf("adapter build check error = %v, want rejection containing %q", err, want)
	}
}

// requireAdapterAcceptance requires the adapter registry's identity checks to
// pass; preparation then fails only because the toolchain root has no go
// command to download the pinned module with.
func requireAdapterAcceptance(t *testing.T, directory string) {
	t.Helper()
	_, _, err := deterministicio.Default().PrepareTargetBuildAdapters(context.Background(), target.Spec{
		PreparationRoot: t.TempDir(), WorkingDir: directory, ToolchainRoot: t.TempDir(),
	})
	if err == nil || deterministicio.IsInvalidBuildAdapterConfiguration(err) {
		t.Fatalf("adapter build check error = %v, want acceptance followed by a download failure", err)
	}
}

// requirePackDecisions builds each pack rule's package with the identities in
// requirements and checks pack selection's capability decision.
func requirePackDecisions(t *testing.T, pack compatibility.Pack, requirements []requirement, allowed bool) {
	t.Helper()
	packs, err := compatibility.LoadPacks()
	if err != nil {
		t.Fatal(err)
	}
	packages := make([]compatibility.Package, 0, len(pack.Rules))
	for _, rule := range pack.Rules {
		index := slices.IndexFunc(requirements, func(required requirement) bool { return required.path == rule.Module.Path })
		if index < 0 {
			t.Fatalf("requirements lack %s", rule.Module.Path)
		}
		pkg := compatibility.Package{
			ImportPath: rule.ImportPath, SourceSetSHA256: rule.SourceSetSHA256,
			Module: compatibility.Module{Path: rule.Module.Path, Version: requirements[index].version, Sum: requirements[index].sum},
		}
		for _, source := range rule.GoSources {
			pkg.GoSources = append(pkg.GoSources, compatibility.Source{Name: source.Name, SHA256: source.SHA256})
		}
		for _, source := range rule.ForeignSources {
			pkg.ForeignSources = append(pkg.ForeignSources, compatibility.ForeignSource{Kind: source.Kind, Name: source.Name, SHA256: source.SHA256})
		}
		packages = append(packages, pkg)
	}
	for _, platform := range pack.Governance.Platforms {
		selection, err := compatibility.SelectPacksForPlatform(packs, packages, platform)
		if err != nil {
			t.Fatal(err)
		}
		for index, rule := range pack.Rules {
			if got := selection.AllowsCapability(packages[index], rule.Capabilities[0]); got != allowed {
				t.Fatalf("%s %s on %s allowed = %t, want %t", pack.ID, rule.ImportPath, platform, got, allowed)
			}
		}
	}
}
