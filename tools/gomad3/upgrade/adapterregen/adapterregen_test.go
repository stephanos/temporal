package adapterregen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"go.temporal.io/server/tools/gomad3/internal/hostfs"
	gomadversion "go.temporal.io/server/tools/gomad3/toolchain/version"
	"golang.org/x/mod/module"
	modzip "golang.org/x/mod/zip"
)

const (
	sentryModule  = "github.com/getsentry/sentry-go"
	sentryUtil    = "util.go"
	goodVersion   = "v0.46.1"
	movedVersion  = "v0.46.2"
	twiceVersion  = "v0.46.3"
	gonePrefix    = "v0.46.4"
	sentryAnchor  = "\texec \"golang.org/x/sys/execabs\"\n"
	generatedFile = "toolchain/version/generated.txt"
)

// The fixture generator derives a file from the version descriptor, as
// version-generate does, so a test can tell a published descriptor from its
// published generated output.
var fixtureGenerator = []string{"sh", "-c", "sha256sum toolchain/version/version.json > " + generatedFile}

type fixture struct {
	t         *testing.T
	root      string
	goCommand string
	env       []string
	pinned    string
}

// newFixture builds a checkout holding the parts adapter regeneration reads
// and serves the pinned sentry module and four candidates from a file proxy.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	goCommand, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go command is unavailable")
	}
	if output, err := exec.Command(goCommand, "env", "GOVERSION").Output(); err != nil || strings.TrimSpace(string(output)) != gomadversion.GoVersion {
		t.Skipf("go command is not the pinned %s: %s", gomadversion.GoVersion, output)
	}
	var pinned gomadversion.AdapterIdentity
	for _, identity := range gomadversion.Adapters {
		if identity.Module == sentryModule {
			pinned = identity
		}
	}
	source, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	for _, relative := range []string{"deterministicio", "toolchain/version/version.json", "internal/compatibilitypack/packs"} {
		copyTree(t, filepath.Join(source, relative), filepath.Join(root, relative))
	}
	if err := os.WriteFile(filepath.Join(root, generatedFile), []byte("initial\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	command := exec.Command(goCommand, "mod", "download", "-json", sentryModule+"@"+pinned.Version)
	command.Dir = t.TempDir()
	command.Env = append(os.Environ(), "GOWORK=off", "GOFLAGS=")
	output, err := command.Output()
	if err != nil {
		t.Fatalf("download pinned sentry: %v", err)
	}
	var downloaded struct{ Dir string }
	if err := json.Unmarshal(output, &downloaded); err != nil {
		t.Fatal(err)
	}
	proxy := t.TempDir()
	serve := func(version string, change func(directory string)) {
		directory := filepath.Join(t.TempDir(), "module")
		copyTree(t, downloaded.Dir, directory)
		if change != nil {
			change(directory)
		}
		escaped, err := module.EscapePath(sentryModule)
		if err != nil {
			t.Fatal(err)
		}
		versions := filepath.Join(proxy, filepath.FromSlash(escaped), "@v")
		if err := os.MkdirAll(versions, 0o700); err != nil {
			t.Fatal(err)
		}
		var archive bytes.Buffer
		if err := modzip.CreateFromDir(&archive, module.Version{Path: sentryModule, Version: version}, directory); err != nil {
			t.Fatal(err)
		}
		goMod, err := os.ReadFile(filepath.Join(directory, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		for name, contents := range map[string][]byte{
			version + ".zip":  archive.Bytes(),
			version + ".mod":  goMod,
			version + ".info": fmt.Appendf(nil, `{"Version":%q,"Time":"2026-01-01T00:00:00Z"}`, version),
		} {
			if err := os.WriteFile(filepath.Join(versions, name), contents, 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	editUtil := func(edit func([]byte) []byte) func(string) {
		return func(directory string) {
			path := filepath.Join(directory, sentryUtil)
			contents, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, edit(contents), 0o600); err != nil {
				t.Fatal(err)
			}
		}
	}
	serve(pinned.Version, nil)
	serve(goodVersion, editUtil(func(contents []byte) []byte {
		return append(contents, []byte("\n// An upstream change outside every anchor.\n")...)
	}))
	serve(movedVersion, editUtil(func(contents []byte) []byte {
		return bytes.Replace(contents, []byte(sentryAnchor), []byte("\texec \"golang.org/x/sys/execabs\" // moved\n"), 1)
	}))
	serve(twiceVersion, editUtil(func(contents []byte) []byte {
		return append(contents, []byte("\n/*\n"+sentryAnchor+"*/\n")...)
	}))
	serve(gonePrefix, func(directory string) {
		if err := os.Remove(filepath.Join(directory, sentryUtil)); err != nil {
			t.Fatal(err)
		}
	})
	return &fixture{
		t: t, root: root, goCommand: goCommand, pinned: pinned.Version,
		env: append(os.Environ(), "GOPROXY=file://"+filepath.ToSlash(proxy), "GOSUMDB=off", "GONOSUMDB=", "GOPRIVATE="),
	}
}

func copyTree(t *testing.T, source, destination string) {
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
			return os.MkdirAll(target, 0o755)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, contents, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func (fixture *fixture) spec(version, approval string) Spec {
	return Spec{
		Root: fixture.root, Module: sentryModule, Version: version, GoCommand: fixture.goCommand, Environment: fixture.env, Approval: approval,
		Generators: [][]string{fixtureGenerator}, Verifiers: [][]string{}, Tidy: []string{"true"},
	}
}

// snapshot digests every checkout file outside the transaction's state.
func (fixture *fixture) snapshot() map[string]string {
	fixture.t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(fixture.root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, _ := filepath.Rel(fixture.root, path)
		if entry.IsDir() {
			if relative == ".toolchain" {
				return filepath.SkipDir
			}
			return nil
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = digest(contents)
		return nil
	})
	if err != nil {
		fixture.t.Fatal(err)
	}
	return files
}

func (fixture *fixture) read(relative string) string {
	fixture.t.Helper()
	contents, err := os.ReadFile(filepath.Join(fixture.root, filepath.FromSlash(relative)))
	if err != nil {
		fixture.t.Fatal(err)
	}
	return string(contents)
}

func (fixture *fixture) dryRun(version string) Result {
	fixture.t.Helper()
	result, err := Run(context.Background(), fixture.spec(version, ""))
	if err != nil {
		fixture.t.Fatal(err)
	}
	return result
}

func requireUnchanged(t *testing.T, before, after map[string]string) {
	t.Helper()
	if !maps.Equal(before, after) {
		for path, sum := range after {
			if before[path] != sum {
				t.Errorf("%s changed", path)
			}
		}
		for path := range before {
			if _, found := after[path]; !found {
				t.Errorf("%s was removed", path)
			}
		}
		t.FailNow()
	}
}

func TestDryRunPrintsChangedSourceAndAnchorsAndWritesNothing(t *testing.T) {
	fixture := newFixture(t)
	before := fixture.snapshot()
	result := fixture.dryRun(goodVersion)
	requireUnchanged(t, before, fixture.snapshot())
	if result.Applied || !strings.HasPrefix(result.Regeneration.ApprovalSHA256, "sha256:") || len(result.Diffs) != 1 || !result.Diffs[0].Changed ||
		!strings.Contains(result.Diffs[0].Diff, "+// An upstream change outside every anchor.") || !strings.Contains(result.Diffs[0].Diff, "--- previous/util.go") {
		t.Fatalf("dry run = %+v", result)
	}
	var rendered bytes.Buffer
	if err := Render(&rendered, result); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"changed upstream source: 1 of 1", "+// An upstream change", "changed util.go source", "prepared source set darwin/arm64", "prepared source set linux/amd64", "--approve-review=" + result.Regeneration.ApprovalSHA256} {
		if !strings.Contains(rendered.String(), want) {
			t.Fatalf("rendered review lacks %q:\n%s", want, rendered.String())
		}
	}
	if again := fixture.dryRun(goodVersion); again.Regeneration.ApprovalSHA256 != result.Regeneration.ApprovalSHA256 {
		t.Fatalf("approval is not reproducible: %s, %s", again.Regeneration.ApprovalSHA256, result.Regeneration.ApprovalSHA256)
	}
}

func TestApplyPublishesConstantsDescriptorTestDataAndGeneratedOutputTogether(t *testing.T) {
	fixture := newFixture(t)
	review := fixture.dryRun(goodVersion)
	result, err := Run(context.Background(), fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256))
	if err != nil {
		t.Fatal(err)
	}
	proposed := review.Regeneration.Proposed
	adapter := fixture.read("deterministicio/sentry_adapter.go")
	for _, want := range []string{`"` + goodVersion + `"`, proposed.Sum, proposed.OriginalSourceInventorySHA256, proposed.ReplacementSourceInventorySHA256, proposed.Rewrites[0].SourceSHA256, proposed.Rewrites[0].ReplacementSHA256} {
		if !strings.Contains(adapter, want) {
			t.Fatalf("published adapter lacks %s", want)
		}
	}
	descriptor := fixture.read("toolchain/version/version.json")
	if !strings.Contains(descriptor, fmt.Sprintf("%q,\n      \"sum\": %q", goodVersion, proposed.Sum)) {
		t.Fatalf("published descriptor lacks the candidate entry:\n%s", descriptor)
	}
	if generated := fixture.read(generatedFile); generated == "initial\n" || !strings.HasPrefix(generated, strings.TrimPrefix(digest([]byte(descriptor)), "sha256:")) {
		t.Fatalf("generated output %q does not derive from the published descriptor", generated)
	}
	if fixtureMod, fixtureSum := fixture.read("deterministicio/testdata/sentry/go.mod"), fixture.read("deterministicio/testdata/sentry/go.sum"); !strings.Contains(fixtureMod, sentryModule+" "+goodVersion) ||
		!strings.Contains(fixtureSum, sentryModule+" "+goodVersion+" "+proposed.Sum) || strings.Contains(fixtureSum, sentryModule+" "+fixture.pinned+" ") {
		t.Fatalf("pinned test data was not moved:\n%s\n%s", fixtureMod, fixtureSum)
	}
	for _, want := range []string{"deterministicio/sentry_adapter.go", "deterministicio/testdata/sentry/go.mod", "deterministicio/testdata/sentry/go.sum", "toolchain/version/version.json", generatedFile} {
		if !strings.Contains(strings.Join(result.Published, "\n"), want) {
			t.Fatalf("published %v, want %s", result.Published, want)
		}
	}
	if pending, err := publicationPending(fixture.root); err != nil || pending {
		t.Fatalf("journal remains after a complete publication: %v %v", pending, err)
	}
}

func TestApplyWithAWrongDigestWritesNothing(t *testing.T) {
	fixture := newFixture(t)
	before := fixture.snapshot()
	_, err := Run(context.Background(), fixture.spec(goodVersion, "sha256:"+strings.Repeat("0", 64)))
	var input *InputError
	if !errors.As(err, &input) || !strings.Contains(err.Error(), "does not match the review digest") {
		t.Fatalf("wrong digest error = %v", err)
	}
	requireUnchanged(t, before, fixture.snapshot())
}

func TestRegenerationThatNeedsAPersonWritesNothing(t *testing.T) {
	fixture := newFixture(t)
	before := fixture.snapshot()
	for _, test := range []struct{ version, want string }{
		{movedVersion, "occurs 0 times"},
		{twiceVersion, "occurs 2 times"},
		{gonePrefix, "no longer provides rewritten file util.go"},
	} {
		for _, approval := range []string{"", "sha256:" + strings.Repeat("1", 64)} {
			_, err := Run(context.Background(), fixture.spec(test.version, approval))
			var blocked *BlockedError
			if !errors.As(err, &blocked) || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("%s regeneration error = %v, want %q", test.version, err, test.want)
			}
		}
	}
	requireUnchanged(t, before, fixture.snapshot())
}

func TestGenerationFailureInStagingPublishesNothing(t *testing.T) {
	fixture := newFixture(t)
	review := fixture.dryRun(goodVersion)
	before := fixture.snapshot()
	spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
	spec.Generators = [][]string{fixtureGenerator, {"sh", "-c", "echo generator broke >&2; exit 7"}}
	_, err := Run(context.Background(), spec)
	var blocked *BlockedError
	if !errors.As(err, &blocked) || !strings.Contains(err.Error(), "generator broke") || !strings.Contains(err.Error(), "nothing was published") {
		t.Fatalf("generation failure error = %v", err)
	}
	requireUnchanged(t, before, fixture.snapshot())
	spec.Generators, spec.Verifiers = [][]string{fixtureGenerator}, [][]string{{"false"}}
	if _, err := Run(context.Background(), spec); !errors.As(err, &blocked) || !strings.Contains(err.Error(), "verify") {
		t.Fatalf("verification failure error = %v", err)
	}
	requireUnchanged(t, before, fixture.snapshot())
}

func TestCheckoutChangedBetweenStagingAndPublicationPublishesNothing(t *testing.T) {
	fixture := newFixture(t)
	review := fixture.dryRun(goodVersion)
	spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
	edited := filepath.Join(fixture.root, "deterministicio", "profile.go")
	spec.afterStage = func() error {
		contents, err := os.ReadFile(edited)
		if err != nil {
			return err
		}
		return os.WriteFile(edited, append(contents, []byte("\n// edited during the transaction\n")...), 0o644)
	}
	before := fixture.snapshot()
	_, err := Run(context.Background(), spec)
	var blocked *BlockedError
	if !errors.As(err, &blocked) || !strings.Contains(err.Error(), "deterministicio/profile.go changed") {
		t.Fatalf("changed checkout error = %v", err)
	}
	after := fixture.snapshot()
	delete(before, "deterministicio/profile.go")
	delete(after, "deterministicio/profile.go")
	requireUnchanged(t, before, after)
}

func TestCompetingApplyOperationsPublishOnce(t *testing.T) {
	fixture := newFixture(t)
	review := fixture.dryRun(goodVersion)
	before := fixture.snapshot()

	state := filepath.Join(fixture.root, filepath.FromSlash(stateDirectory))
	if err := os.MkdirAll(state, 0o700); err != nil {
		t.Fatal(err)
	}
	held, err := hostfs.Try(filepath.Join(state, "lock"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Run(context.Background(), fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)); !errors.Is(err, hostfs.ErrContended) {
		t.Fatalf("apply under a held lock = %v, want contention", err)
	}
	requireUnchanged(t, before, fixture.snapshot())
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}

	// A second apply that starts while the first is staging is refused, and
	// the first publishes the complete set.
	first := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
	var competing error
	var once sync.Once
	first.afterStage = func() error {
		once.Do(func() {
			_, competing = Run(context.Background(), fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256))
		})
		return nil
	}
	if _, err := Run(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(competing, hostfs.ErrContended) {
		t.Fatalf("competing apply = %v, want contention", competing)
	}
	if !strings.Contains(fixture.read("deterministicio/sentry_adapter.go"), `"`+goodVersion+`"`) {
		t.Fatal("first apply did not publish")
	}
}

func TestInterruptedPublicationCompletesOnTheNextRun(t *testing.T) {
	complete := newFixture(t)
	review := complete.dryRun(goodVersion)
	if _, err := Run(context.Background(), complete.spec(goodVersion, review.Regeneration.ApprovalSHA256)); err != nil {
		t.Fatal(err)
	}
	want := complete.snapshot()

	for _, resume := range []string{"recover", "apply"} {
		t.Run(resume, func(t *testing.T) {
			fixture := newFixture(t)
			before := fixture.snapshot()
			spec := fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256)
			interrupted := errors.New("interrupted")
			spec.beforeApplyFile = func(index int) error {
				if index == 2 {
					return interrupted
				}
				return nil
			}
			if _, err := Run(context.Background(), spec); !errors.Is(err, interrupted) {
				t.Fatalf("interrupted apply = %v", err)
			}
			if mixed := fixture.snapshot(); maps.Equal(mixed, before) || maps.Equal(mixed, want) {
				t.Fatal("the interruption did not leave a partial publication to recover")
			}
			if _, err := Run(context.Background(), fixture.spec(goodVersion, "")); err == nil || !strings.Contains(err.Error(), "--recover") {
				t.Fatalf("dry run over a pending publication = %v", err)
			}
			switch resume {
			case "recover":
				if err := Recover(fixture.root); err != nil {
					t.Fatal(err)
				}
			case "apply":
				// The next apply completes the interrupted publication and
				// stops, so its review is made against the completed set.
				_, err := Run(context.Background(), fixture.spec(goodVersion, review.Regeneration.ApprovalSHA256))
				var blocked *BlockedError
				if !errors.As(err, &blocked) || !strings.Contains(err.Error(), "completed an interrupted adapter publication") {
					t.Fatalf("resuming apply = %v", err)
				}
			}
			requireUnchanged(t, want, fixture.snapshot())
		})
	}
}

func TestUncommittedJournalRollsBack(t *testing.T) {
	fixture := newFixture(t)
	before := fixture.snapshot()
	state := filepath.Join(fixture.root, filepath.FromSlash(stateDirectory))
	pending := filepath.Join(state, "pending-1")
	if err := os.MkdirAll(pending, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pending, journalManifest), []byte(`{"schema":"`+journalSchema+`","entries":[{"path":"deterministicio/profile.go","previous":"","next":"sha256:x","blob":"blob-0"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Recover(fixture.root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(pending); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("uncommitted journal remains: %v", err)
	}
	requireUnchanged(t, before, fixture.snapshot())
}

func TestCommittedJournalOverLaterEditsIsLeftForAPerson(t *testing.T) {
	fixture := newFixture(t)
	path := "deterministicio/profile.go"
	state := filepath.Join(fixture.root, filepath.FromSlash(stateDirectory))
	journalPath := filepath.Join(state, journalDirectory)
	if err := os.MkdirAll(journalPath, 0o700); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(journal{Schema: journalSchema, Entries: []journalEntry{{Path: path, Previous: "sha256:old", Next: "sha256:new", Blob: "blob-0", Mode: 0o644}}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(journalPath, journalManifest), encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	before := fixture.snapshot()
	var blocked *BlockedError
	if err := Recover(fixture.root); !errors.As(err, &blocked) || !strings.Contains(err.Error(), path+" changed after the publication started") {
		t.Fatalf("recover over a later edit = %v", err)
	}
	requireUnchanged(t, before, fixture.snapshot())
}

func TestStalePacksNameBindingsOfThePreviousIdentity(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	var memory gomadversion.AdapterIdentity
	for _, identity := range gomadversion.Adapters {
		if identity.Module == "modernc.org/memory" {
			memory = identity
		}
	}
	stale, err := stalePacks(root, memory.Module, "v1.99.0")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]bool{}
	for _, binding := range stale {
		files[binding.File] = true
	}
	if !files["modernc-libc-xsys-v047.json"] {
		t.Fatalf("stale bindings = %+v, want the libc-bound packs", stale)
	}
	if current, err := stalePacks(root, memory.Module, memory.Version); err != nil || len(current) != 0 {
		t.Fatalf("bindings of the pinned version = %+v, %v", current, err)
	}
}

func TestDefaultGeneratorsAreTheModuleLocalMakeGenerateSteps(t *testing.T) {
	makefile, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	recipe := string(makefile)
	start := strings.Index(recipe, "\ngenerate:")
	end := strings.Index(recipe[start+1:], "\n\n")
	recipe = recipe[start : start+1+end]
	var steps []string
	for _, line := range strings.Split(recipe, "\n")[2:] {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "tests-qualification-generate") {
			continue
		}
		steps = append(steps, strings.Replace(strings.TrimPrefix(line, "GOCACHE=$(CURDIR)/.toolchain/generator-cache go "), "$(CURDIR)", "{root}", 1))
	}
	var defaults []string
	for _, command := range DefaultGenerators {
		defaults = append(defaults, strings.Join(command[1:], " "))
	}
	if strings.Join(steps, "\n") != strings.Join(defaults, "\n") {
		t.Fatalf("make generate steps:\n%s\ndefault generators:\n%s", strings.Join(steps, "\n"), strings.Join(defaults, "\n"))
	}
}

func TestUnifiedDiff(t *testing.T) {
	previous := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	candidate := "a\nB\nc\nd\ne\nf\ng\nh\ni\nj\nk\n"
	want := "--- previous/x\n+++ candidate/x\n@@ -1,5 +1,5 @@\n a\n-b\n+B\n c\n d\n e\n@@ -8,3 +8,4 @@\n h\n i\n j\n+k\n"
	if got := unifiedDiff("x", []byte(previous), []byte(candidate)); got != want {
		t.Fatalf("diff =\n%s\nwant\n%s", got, want)
	}
	if got := unifiedDiff("x", []byte(previous), []byte(previous)); got != "" {
		t.Fatalf("equal diff = %q", got)
	}
}
