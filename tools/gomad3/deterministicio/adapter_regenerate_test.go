package deterministicio

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad3/target"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
)

// pinnedReleaseGo returns a go command of the pinned release, whose release
// tags select prepared package files exactly as the target build does.
func pinnedReleaseGo(t *testing.T) string {
	t.Helper()
	goCommand, err := filepath.Abs(filepath.Join("..", ".toolchain", "bin", "go"))
	if err != nil {
		t.Fatal(err)
	}
	output, err := exec.CommandContext(t.Context(), goCommand, "env", "GOVERSION").Output()
	if err != nil {
		t.Skipf("pinned toolchain is unavailable: %v", err)
	}
	if version, _, _ := strings.Cut(strings.TrimSpace(string(output)), " "); version != gomadversion.GoVersion {
		t.Fatalf("pinned toolchain reports %s, want %s", version, gomadversion.GoVersion)
	}
	return goCommand
}

// TestAdapterPreparedSourceSetPinsReproduceOnEveryHost recomputes every
// adapter's prepared source-set pin for every platform it is pinned for from
// the prepared replacement's sources, so a host of any platform derives the
// pins the qualified platforms' capability reviews check.
func TestAdapterPreparedSourceSetPinsReproduceOnEveryHost(t *testing.T) {
	goCommand := pinnedReleaseGo(t)
	moduleCache := pinnedModuleCache(t)
	pins := map[string]map[string]string{libcModulePath: libcPreparedSourceSetSHA256ByHost}
	for _, definition := range deterministicAdapters.definitions {
		if definition.implementation.rewritten != nil {
			pins[definition.identity.Module] = definition.implementation.rewritten.preparedSourceSetSHA256ByHost
		}
	}
	if len(pins) != len(deterministicAdapters.definitions) {
		t.Fatalf("pins cover %d of %d adapters", len(pins), len(deterministicAdapters.definitions))
	}
	checked := 0
	for _, definition := range deterministicAdapters.definitions {
		module := definition.identity.Module
		downloadPinnedModule(t, module, definition.identity.Version)
		prepared, err := definition.implementation.prepare(moduleCache, t.TempDir(), definition.identity)
		if err != nil {
			t.Fatalf("%s: %v", module, err)
		}
		directory := prepared.replacement
		if prepared.evidence.PreparedPackage != module {
			directory = filepath.Join(directory, filepath.FromSlash(strings.TrimPrefix(prepared.evidence.PreparedPackage, module+"/")))
		}
		if len(pins[module]) != 2 {
			t.Fatalf("%s pins %d platforms: %v", module, len(pins[module]), pins[module])
		}
		for _, platform := range sortedKeys(pins[module]) {
			goos, goarch, _ := strings.Cut(platform, "/")
			got, err := target.AdapterPreparedSourceSetSHA256(t.Context(), goCommand, directory, prepared.evidence.PreparedPackage, goos, goarch)
			if err != nil {
				t.Fatalf("%s %s: %v", module, platform, err)
			}
			if got != pins[module][platform] {
				t.Fatalf("%s %s prepared source set = %s, want %s", module, platform, got, pins[module][platform])
			}
			checked++
		}
	}
	t.Logf("reproduced %d prepared source-set pins of %d adapters", checked, len(deterministicAdapters.definitions))
}

type regenerationFixture struct {
	spec                rewrittenModule
	previous, candidate string
	goCommand           string
}

// newRegenerationFixture copies the pinned module of adapter as both the
// previous and the candidate version; change edits the candidate.
func newRegenerationFixture(t *testing.T, spec rewrittenModule, change func(candidate string)) regenerationFixture {
	t.Helper()
	downloadPinnedModule(t, spec.module, spec.version)
	pinned := filepath.Join(append([]string{pinnedModuleCache(t)}, spec.cacheElements...)...)
	fixture := regenerationFixture{spec: spec, previous: filepath.Join(t.TempDir(), "previous"), candidate: filepath.Join(t.TempDir(), "candidate"), goCommand: pinnedReleaseGo(t)}
	for _, destination := range []string{fixture.previous, fixture.candidate} {
		copyFixtureTree(t, pinned, destination)
	}
	if change != nil {
		change(fixture.candidate)
	}
	return fixture
}

func copyFixtureTree(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, contents, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func editFixtureFile(t *testing.T, path string, edit func([]byte) []byte) {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, edit(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (fixture regenerationFixture) regenerate(t *testing.T) (AdapterRegeneration, error) {
	t.Helper()
	return regenerateRewrittenModule(context.Background(), fixture.spec, AdapterRegenerationRequest{
		Module: fixture.spec.module, Version: "v99.0.0-fixture", Sum: "h1:fixturefixturefixturefixturefixturefixture0=",
		PreviousModule: fixture.previous, CandidateModule: fixture.candidate, GoCommand: fixture.goCommand, Scratch: t.TempDir(),
	})
}

func TestRegenerateAdapterDerivesAnchorsForChangedUpstreamSource(t *testing.T) {
	fixture := newRegenerationFixture(t, sentryAdapter, func(candidate string) {
		editFixtureFile(t, filepath.Join(candidate, sentryUtilPath), func(contents []byte) []byte {
			return append(contents, []byte("\n// An upstream change outside every anchor.\n")...)
		})
	})
	regeneration, err := fixture.regenerate(t)
	if err != nil {
		t.Fatal(err)
	}
	if regeneration.Previous.Version != sentryVersion || regeneration.Proposed.Version != "v99.0.0-fixture" || !strings.HasPrefix(regeneration.ApprovalSHA256, "sha256:") {
		t.Fatalf("regeneration = %+v", regeneration)
	}
	previous, proposed := regeneration.Previous.Rewrites[0], regeneration.Proposed.Rewrites[0]
	if previous.SourceSHA256 != sentryUtilSourceSHA256 || proposed.SourceSHA256 == previous.SourceSHA256 || proposed.ReplacementSHA256 == previous.ReplacementSHA256 {
		t.Fatalf("util.go anchors = %+v -> %+v", previous, proposed)
	}
	if regeneration.Proposed.OriginalSourceInventorySHA256 == sentryOriginalSourceInventorySHA256 || regeneration.Proposed.ReplacementSourceInventorySHA256 == sentryReplacementSourceInventorySHA256 {
		t.Fatalf("inventories did not change: %+v", regeneration.Proposed)
	}
	// Appending a comment changes the file but not which files each platform
	// compiles, so the source set changes on both platforms the same way.
	for platform, pin := range regeneration.Proposed.PreparedSourceSetSHA256 {
		if pin == sentryPreparedSourceSetSHA256ByHost[platform] {
			t.Fatalf("%s prepared source set did not change", platform)
		}
	}
	if len(regeneration.Sources) != 1 || regeneration.Sources[0].PreviousSHA256 == regeneration.Sources[0].CandidateSHA256 || !bytes.Contains(regeneration.Sources[0].Candidate, []byte("An upstream change")) {
		t.Fatalf("changed sources = %+v", regeneration.Sources)
	}
	again, err := fixture.regenerate(t)
	if err != nil || again.ApprovalSHA256 != regeneration.ApprovalSHA256 {
		t.Fatalf("second regeneration approval = %s, %v; want %s", again.ApprovalSHA256, err, regeneration.ApprovalSHA256)
	}
}

func TestRegenerateAdapterRederivesBaseRewrites(t *testing.T) {
	rewrite := grpcLinuxRewrites[1]
	fixture := newRegenerationFixture(t, grpcAdapter, func(candidate string) {
		editFixtureFile(t, filepath.Join(candidate, filepath.FromSlash(rewrite.base)), func(contents []byte) []byte {
			return append(contents, []byte("\n// An upstream change to the non-Linux implementation.\n")...)
		})
	})
	regeneration, err := fixture.regenerate(t)
	if err != nil {
		t.Fatal(err)
	}
	for index, proposed := range regeneration.Proposed.Rewrites {
		previous := regeneration.Previous.Rewrites[index]
		changed := proposed.ReplacementSHA256 != previous.ReplacementSHA256 || proposed.BaseSHA256 != previous.BaseSHA256 || proposed.SourceSHA256 != previous.SourceSHA256
		if changed != (proposed.Path == rewrite.path) {
			t.Fatalf("rewrite %s changed = %v: %+v -> %+v", proposed.Path, changed, previous, proposed)
		}
	}
}

func TestRegenerateAdapterFailsWithoutWritingWhenAnchorsDoNotMatchOnce(t *testing.T) {
	anchor := sentryRewrites[0].rewrites[1].anchor
	for _, test := range []struct {
		name   string
		change func([]byte) []byte
		count  int
	}{
		{name: "moved", count: 0, change: func(contents []byte) []byte {
			return bytes.Replace(contents, anchor, bytes.Replace(anchor, []byte("exec.LookPath"), []byte("exec.LookPath "), 1), 1)
		}},
		{name: "twice", count: 2, change: func(contents []byte) []byte {
			return append(contents, append([]byte("\nfunc duplicated() {\n"), append(anchor, []byte("}\n")...)...)...)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newRegenerationFixture(t, sentryAdapter, func(candidate string) {
				editFixtureFile(t, filepath.Join(candidate, sentryUtilPath), test.change)
			})
			scratch := t.TempDir()
			_, err := regenerateRewrittenModule(context.Background(), fixture.spec, AdapterRegenerationRequest{
				Module: sentryModulePath, Version: "v99.0.0-fixture", Sum: "h1:fixture", PreviousModule: fixture.previous, CandidateModule: fixture.candidate,
				GoCommand: fixture.goCommand, Scratch: scratch,
			})
			var mismatch *AnchorMismatchError
			if !errors.As(err, &mismatch) || mismatch.Count != test.count || mismatch.Path != sentryUtilPath || !IsAdapterRegenerationBlocked(err) {
				t.Fatalf("regeneration error = %v, want anchor matching %d times", err, test.count)
			}
			if entries, err := os.ReadDir(scratch); err != nil || len(entries) != 0 {
				t.Fatalf("regeneration wrote %v, %v", entries, err)
			}
		})
	}
}

func TestRegenerateAdapterFailsWhenARewrittenFileIsDeletedUpstream(t *testing.T) {
	fixture := newRegenerationFixture(t, sentryAdapter, func(candidate string) {
		if err := os.Remove(filepath.Join(candidate, sentryUtilPath)); err != nil {
			t.Fatal(err)
		}
	})
	_, err := fixture.regenerate(t)
	var missing *AdapterSourceMissingError
	if !errors.As(err, &missing) || missing.Path != sentryUtilPath || !IsAdapterRegenerationBlocked(err) {
		t.Fatalf("regeneration error = %v, want a missing rewritten file", err)
	}
}

func TestRegenerateAdapterRejectsAPreviousModuleThatIsNotThePin(t *testing.T) {
	fixture := newRegenerationFixture(t, sentryAdapter, nil)
	editFixtureFile(t, filepath.Join(fixture.previous, sentryUtilPath), func(contents []byte) []byte { return append(contents, '\n') })
	if _, err := fixture.regenerate(t); err == nil || !strings.Contains(err.Error(), "source inventory identity mismatch") {
		t.Fatalf("regeneration error = %v", err)
	}
}

func TestRegenerateAdapterRefusesCustomPreparation(t *testing.T) {
	_, err := RegenerateAdapter(context.Background(), AdapterRegenerationRequest{Module: libcModulePath})
	var custom *AdapterNotRegenerableError
	if !errors.As(err, &custom) || !IsAdapterRegenerationBlocked(err) {
		t.Fatalf("libc regeneration error = %v", err)
	}
	if modules := RegenerableAdapters(); len(modules) != len(deterministicAdapters.definitions)-1 {
		t.Fatalf("regenerable adapters = %v", modules)
	}
}

// TestAdapterSourceEditsReplaceEveryAnchor regenerates memberlist, whose
// version v0.5.4 is also go-metrics' version, and checks that only the
// memberlist literals change.
func TestAdapterSourceEditsReplaceEveryAnchor(t *testing.T) {
	fixture := newRegenerationFixture(t, memberlistAdapter, func(candidate string) {
		for _, rewrite := range memberlistRewrites {
			editFixtureFile(t, filepath.Join(candidate, filepath.FromSlash(rewrite.path)), func(contents []byte) []byte {
				return append(contents, []byte("\n// upstream change\n")...)
			})
		}
	})
	regeneration, err := fixture.regenerate(t)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	copyFixtureTree(t, ".", filepath.Join(root, "deterministicio"))
	edits, err := regeneration.SourceEdits(root)
	if err != nil {
		t.Fatal(err)
	}
	adapter, found := edits["deterministicio/memberlist_adapter.go"]
	if !found {
		t.Fatalf("edited files = %v", keysOf(edits))
	}
	for _, removed := range []string{memberlistSum, memberlistOriginalSourceInventorySHA256, memberlistReplacementSourceInventorySHA256, `"` + memberlistVersion + `"`} {
		if bytes.Contains(adapter, []byte(removed)) {
			t.Fatalf("regenerated adapter retains %s", removed)
		}
	}
	for _, added := range []string{regeneration.Proposed.Sum, regeneration.Proposed.OriginalSourceInventorySHA256, regeneration.Proposed.ReplacementSourceInventorySHA256, `"v99.0.0-fixture"`} {
		if !bytes.Contains(adapter, []byte(added)) {
			t.Fatalf("regenerated adapter lacks %s", added)
		}
	}
	for _, pin := range regeneration.Proposed.PreparedSourceSetSHA256 {
		if !bytes.Contains(adapter, []byte(pin)) {
			t.Fatalf("regenerated adapter lacks prepared source set %s", pin)
		}
	}
	for _, rewrite := range regeneration.Proposed.Rewrites {
		if !bytes.Contains(adapter, []byte(rewrite.SourceSHA256)) || !bytes.Contains(adapter, []byte(rewrite.ReplacementSHA256)) {
			t.Fatalf("regenerated adapter lacks %s anchors", rewrite.Path)
		}
	}
	for path := range edits {
		if path == "deterministicio/hashicorpmetrics_adapter.go" {
			t.Fatal("regenerating memberlist edited the go-metrics adapter, which shares its version")
		}
	}
}

func TestAdapterSourceEditsRejectAnAnchorThatIsNotALiteral(t *testing.T) {
	fixture := newRegenerationFixture(t, sentryAdapter, func(candidate string) {
		editFixtureFile(t, filepath.Join(candidate, sentryUtilPath), func(contents []byte) []byte { return append(contents, '\n') })
	})
	regeneration, err := fixture.regenerate(t)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	copyFixtureTree(t, ".", filepath.Join(root, "deterministicio"))
	editFixtureFile(t, filepath.Join(root, "deterministicio", "sentry_adapter.go"), func(contents []byte) []byte {
		return bytes.Replace(contents, []byte(`"`+sentryUtilSourceSHA256+`"`), []byte(`"sha256:" + "`+strings.TrimPrefix(sentryUtilSourceSHA256, "sha256:")+`"`), 1)
	})
	if _, err := regeneration.SourceEdits(root); err == nil || !strings.Contains(err.Error(), "is not declared as a literal") {
		t.Fatalf("SourceEdits() error = %v", err)
	}
}

func keysOf[V any](values map[string]V) []string {
	return sortedKeys(values)
}

func TestVerifyRegisteredAdapterChecksEveryPin(t *testing.T) {
	fixture := newRegenerationFixture(t, sentryAdapter, nil)
	if err := VerifyRegisteredAdapter(t.Context(), sentryModulePath, fixture.candidate, fixture.goCommand); err != nil {
		t.Fatal(err)
	}
	editFixtureFile(t, filepath.Join(fixture.candidate, sentryUtilPath), func(contents []byte) []byte { return append(contents, '\n') })
	if err := VerifyRegisteredAdapter(t.Context(), sentryModulePath, fixture.candidate, fixture.goCommand); err == nil {
		t.Fatal("VerifyRegisteredAdapter() accepted a changed module")
	}
}
