package sweep

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/tools/gomad/internal/backends/wasm/backend"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/installation"
	"go.temporal.io/server/tools/gomad/internal/backends/wasm/wasi"
	"go.temporal.io/server/tools/gomad/internal/evidence/artifact"
	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

// Explore inputs that are fixed for every row. Choice recording is required:
// cooperative WASM execution refuses to run without it. For WASM the output
// limit is the guest's hard combined-output capacity (wasi.Limits.OutputBytes),
// so it matches the measure fixture's 256 MiB. The WASI transcript keeps
// explore's 64 MiB default (--io-transcript-bytes is native-only); transcript
// capacity rows are therefore bounded by this harness, not the guest.
const (
	exploreChoiceBytes  = "64MiB"
	exploreOutputLimit  = "256MiB"
	exploreSuccessBytes = "4GiB"
	exploreGrace        = "2s"
	subprocessSlack     = 5 * time.Minute
)

// Sweeper runs one sweep invocation.
type Sweeper struct {
	Options      Options
	Executor     Executor
	ReadArtifact func(path string) ArtifactFacts
	Now          func() time.Time
	Log          io.Writer
	Command      []string // recorded in the header
	// HarnessSHA256 identifies the wasmsweep build; it is part of the
	// candidate so a changed classifier or limit never resumes old rows.
	HarnessSHA256 string
}

// JournalPath and SidecarPath are the sweep's outputs under --out.
func JournalPath(out string) string { return filepath.Join(out, "sweep.jsonl") }
func SidecarPath(out string) string { return filepath.Join(out, "sweep.json") }

// GuestTags are the slim Temporal guest build tags every row is built with.
func GuestTags() []string { return backend.TemporalGuestBuildTags() }

// Run enumerates, installs, runs every selected test once and writes the
// journal and sidecar. It returns verification findings of the sidecar.
func (sweeper *Sweeper) Run(parent context.Context) (findings []string, runErr error) {
	options := sweeper.Options
	if err := options.Validate(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(parent, options.OverallBudget)
	defer cancel()
	if err := requireRegularFiles(options); err != nil {
		return nil, err
	}
	if sweeper.Executor == nil {
		sweeper.Executor = HostExecutor{}
	}
	if sweeper.ReadArtifact == nil {
		sweeper.ReadArtifact = ReadArtifact
	}
	if sweeper.Now == nil {
		sweeper.Now = time.Now
	}
	if sweeper.Log == nil {
		sweeper.Log = io.Discard
	}
	started := sweeper.Now()
	for _, dir := range []string{options.Out, options.Work, options.Artifacts} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return nil, err
		}
	}
	tags := GuestTags()
	packages, err := Enumerate(ctx, sweeper.Executor, options, tags, sweeper.Log)
	if err != nil || ctx.Err() != nil {
		return nil, fmt.Errorf("enumerate sweep: %w", errors.Join(err, ctx.Err()))
	}
	candidate, err := sweeper.baseCandidate(ctx, tags)
	candidate.HarnessSHA256 = sweeper.HarnessSHA256
	candidate.ExploreFlags = fixedExploreFlags()
	if err != nil || ctx.Err() != nil {
		return nil, fmt.Errorf("identify sweep candidate: %w", errors.Join(err, ctx.Err()))
	}
	manifests := map[string]string{}
	for index, pkg := range packages {
		dirs := dirsFor(options.Work, pkg.Pattern)
		manifest, err := installPackage(ctx, sweeper.Executor, options, pkg.Pattern, dirs)
		if err != nil || ctx.Err() != nil {
			return nil, fmt.Errorf("install sweep package %s: %w", pkg.Pattern, errors.Join(err, ctx.Err()))
		}
		installed := candidate
		fillFromManifest(&installed, manifest)
		if index == 0 {
			candidate = installed
		} else if installed.Key() != candidate.Key() {
			return nil, fmt.Errorf("installation for %s has a different candidate identity", pkg.Pattern)
		}
		manifests[pkg.Pattern] = dirs.manifest
	}
	key := candidate.Key()
	previous, err := ReadJournal(JournalPath(options.Out))
	if err != nil {
		return nil, err
	}
	journal, err := OpenJournal(JournalPath(options.Out))
	if err != nil {
		return nil, err
	}
	defer func() { runErr = errors.Join(runErr, journal.Close()) }()
	if err := journal.Append(Row{Kind: "header", Candidate: key, Time: sweeper.Now().UTC().Format(time.RFC3339), Header: &Header{
		Statement: ObservationStatement, Candidate: candidate, Host: currentHost(), Packages: packages,
		OverallBudget: options.OverallBudget.String(), MaxTests: options.MaxTests, RetainBytes: options.RetainBytes, Command: sweeper.Command,
	}}); err != nil {
		return nil, err
	}
	latest := Latest(previous, key)
	retention := newRetention(options.RetainBytes, latest)
	for _, pkg := range packages {
		dirs := dirsFor(options.Work, pkg.Pattern)
		warmed := false
		for index, name := range pkg.Tests {
			if err := parent.Err(); err != nil {
				return nil, err
			}
			if options.MaxTests != 0 && index >= options.MaxTests {
				break
			}
			if entry, ok := latest[pkg.Pattern+"\x00"+name]; ok && entry.Class != ClassNotRun {
				if _, err := fmt.Fprintf(sweeper.Log, "wasmsweep: resume: %s %s already %s\n", pkg.Pattern, name, entry.Class); err != nil {
					return nil, err
				}
				continue
			}
			var entry TestEntry
			switch {
			case pkg.GuestExcluded[name] != "":
				entry = TestEntry{Package: pkg.Pattern, Name: name, Class: ClassBuildFailure, Reason: "guest-file-set: " + pkg.GuestExcluded[name] + " is excluded from the wasip1 guest build under the slim tags"}
			case ctx.Err() != nil || !sweeper.fits(started, 1):
				entry = TestEntry{Package: pkg.Pattern, Name: name, Class: ClassNotRun, Reason: "overall budget exhausted before start"}
			default:
				if !warmed && sweeper.Now().Sub(started)+options.PerTestTimeout+3*options.PrepareTimeout <= options.OverallBudget {
					// The helper compiles the module (Cranelift) inside the
					// execution timeout. One unrecorded run selecting no test
					// fills this package's engine cache, so the first row's
					// budget is not spent compiling.
					warmup, err := sweeper.warm(ctx, pkg, manifests[pkg.Pattern], dirs)
					if err != nil {
						if ctx.Err() != nil && parent.Err() == nil && errors.Is(err, ctx.Err()) {
							entry = TestEntry{Package: pkg.Pattern, Name: name, Class: ClassNotRun, Reason: "overall budget exhausted during warm-up"}
							break
						}
						return nil, err
					}
					if err := journal.Append(Row{Kind: "warmup", Candidate: key, Time: sweeper.Now().UTC().Format(time.RFC3339), Test: &warmup}); err != nil {
						return nil, err
					}
					if _, err := fmt.Fprintf(sweeper.Log, "wasmsweep: %s warm-up exit=%d wall=%.1fs engine-entries-added=%d\n", pkg.Pattern, warmup.GomadExit, warmup.WallSeconds, warmup.EngineEntriesAdded); err != nil {
						return nil, err
					}
				}
				warmed = true
				if ctx.Err() != nil || !sweeper.fits(started, 1) {
					entry = TestEntry{Package: pkg.Pattern, Name: name, Class: ClassNotRun, Reason: "overall budget exhausted after warm-up"}
					break
				}
				entry, err = sweeper.runTest(ctx, pkg, name, manifests[pkg.Pattern], dirs, retention, started)
				if err != nil {
					if ctx.Err() != nil && parent.Err() == nil && errors.Is(err, ctx.Err()) {
						entry.Class, entry.Reason = ClassNotRun, "overall budget exhausted during test"
						break
					}
					return nil, err
				}
			}
			entry.ConfigSource = options.ConfigSource
			if err := journal.Append(Row{Kind: "test", Candidate: key, Time: sweeper.Now().UTC().Format(time.RFC3339), Test: &entry}); err != nil {
				return nil, err
			}
			if _, err := fmt.Fprintf(sweeper.Log, "wasmsweep: %s %s class=%s wall=%.1fs reason=%q\n", pkg.Pattern, name, entry.Class, entry.WallSeconds, entry.Reason); err != nil {
				return nil, err
			}
			if parent.Err() != nil {
				return nil, parent.Err()
			}
		}
		// The engine cache serves only this package's modules.
		if err := os.RemoveAll(dirs.root); err != nil {
			return nil, err
		}
	}
	rows, err := ReadJournal(JournalPath(options.Out))
	if err != nil {
		return nil, err
	}
	sidecar, err := BuildSidecar(rows)
	if err != nil {
		return nil, err
	}
	if err := WriteSidecar(SidecarPath(options.Out), sidecar); err != nil {
		return nil, err
	}
	return Verify(sidecar), nil
}

// fixedExploreFlags are the explore flags every row shares; they are part of
// the candidate identity.
func fixedExploreFlags() []string {
	flags := []string{"--json", "--parallel=1", "--terminate-grace=" + exploreGrace,
		"--choices", "--choice-bytes=" + exploreChoiceBytes, "--output-limit=" + exploreOutputLimit,
		"--keep-successes=all", "--success-limit=1", "--success-bytes=" + exploreSuccessBytes}
	for _, tag := range GuestTags() {
		flags = append(flags, "--build-tag="+tag)
	}
	for _, env := range []string{"TEMPORAL_ROOT=/workspace", "TMPDIR=/tmp", "TEMPORAL_TEST_LOG_LEVEL=warn"} {
		flags = append(flags, "--env="+env)
	}
	return flags
}

// ExploreArgs is the gomad command line for one test function.
func ExploreArgs(manifest, artifacts string, options Options, pattern, test string) []string {
	return exploreArgs(manifest, artifacts, options, pattern, testRunPattern(test), options.PerTestTimeout)
}

func exploreArgs(manifest, artifacts string, options Options, pattern, run string, executionTimeout time.Duration) []string {
	args := []string{"--installation", manifest, "explore", fmt.Sprintf("--seeds=%d", options.Seed),
		"--execution-timeout=" + executionTimeout.String(),
		"--overall-timeout=" + (executionTimeout + options.PrepareTimeout).String(),
		"--artifacts", artifacts, "--working-dir", options.Checkout}
	args = append(args, fixedExploreFlags()...)
	return append(args, "go-test", pattern, "--", "-test.run="+run, "-test.v", "-test.count=1")
}

// fits reports whether runs more per-test executions (each with its
// preparation allowance) still fit the overall budget.
func (sweeper *Sweeper) fits(started time.Time, runs int) bool {
	options := sweeper.Options
	return sweeper.Now().Sub(started)+time.Duration(runs)*(options.PerTestTimeout+options.PrepareTimeout) <= options.OverallBudget
}

// warm runs the package's guest once with a run expression that selects no
// test, under the preparation allowance, and records only its costs.
func packageExecution(options Options, pkg Package) (Options, string) {
	if pkg.ModuleDirectory != "" {
		options.Checkout = pkg.ModuleDirectory
	}
	if pkg.Target != "" {
		return options, pkg.Target
	}
	return options, pkg.Pattern
}

func (sweeper *Sweeper) warm(ctx context.Context, pkg Package, manifest string, dirs packageDirs) (TestEntry, error) {
	options, target := packageExecution(sweeper.Options, pkg)
	pattern := pkg.Pattern
	entry := TestEntry{Package: pattern, Name: "(warm-up)", Attempts: 1, ConfigSource: options.ConfigSource}
	runRoot := filepath.Join(dirs.runs, "warm-up")
	if err := resetDirectory(runRoot); err != nil {
		return entry, err
	}
	_, before := treeSize(dirs.engine)
	completed := sweeper.Executor.Exec(ctx, Command{Path: options.Gomad, Args: exploreArgs(manifest, filepath.Join(runRoot, "artifacts"), options, target, "^$", options.PrepareTimeout), Dir: runRoot, Env: sanitizedEnviron(options.Environ), Timeout: 2*options.PrepareTimeout + subprocessSlack})
	if ctx.Err() != nil {
		return entry, errors.Join(ctx.Err(), os.RemoveAll(runRoot))
	}
	entry.GomadExit, entry.WallSeconds = completed.Exit, completed.Elapsed.Seconds()
	entry.PreparationBytes, _ = treeSize(dirs.preparation)
	entry.ArtifactBytes, _ = treeSize(runRoot)
	var after int
	entry.EngineCacheBytes, after = treeSize(dirs.engine)
	entry.EngineEntriesAdded = after - before
	entry.Class, entry.Reason = ClassNotRun, "warm-up only; not a test row"
	if completed.Err != nil {
		entry.Reason += ": " + Normalize(completed.Err.Error())
	}
	return entry, os.RemoveAll(runRoot)
}

func (sweeper *Sweeper) runTest(ctx context.Context, pkg Package, name, manifest string, dirs packageDirs, retention *retention, started time.Time) (TestEntry, error) {
	options, target := packageExecution(sweeper.Options, pkg)
	pattern := pkg.Pattern
	entry := TestEntry{Package: pattern, Name: name}
	var runRoot string
	var outcome Outcome
	for attempt := 1; attempt <= 2; attempt++ {
		runRoot = filepath.Join(dirs.runs, fmt.Sprintf("%s-%d", name, attempt))
		if err := resetDirectory(runRoot); err != nil {
			return entry, err
		}
		_, entriesBefore := treeSize(dirs.engine)
		completed := sweeper.Executor.Exec(ctx, Command{Path: options.Gomad, Args: ExploreArgs(manifest, filepath.Join(runRoot, "artifacts"), options, target, name), Dir: runRoot, Env: sanitizedEnviron(options.Environ), Timeout: options.PerTestTimeout + options.PrepareTimeout + subprocessSlack})
		entry.Attempts, entry.GomadExit, entry.WallSeconds = attempt, completed.Exit, completed.Elapsed.Seconds()
		if ctx.Err() != nil {
			// An interrupted sweep records nothing for this function; a
			// resumed sweep runs it again.
			return entry, errors.Join(ctx.Err(), os.RemoveAll(runRoot))
		}
		observation := Observation{Test: name, Exit: completed.Exit, Stdout: completed.Stdout, Stderr: completed.Stderr, Artifacts: map[string]ArtifactFacts{}}
		if completed.Err != nil {
			observation.ExecError = completed.Err.Error()
		}
		if events, err := ParseEvents(completed.Stdout); err == nil {
			for _, event := range events {
				if event.Type == "artifact" && event.Path != "" {
					observation.Artifacts[event.Path] = sweeper.ReadArtifact(event.Path)
				}
			}
		}
		outcome = Classify(observation)
		entry.PreparationBytes, _ = treeSize(dirs.preparation)
		entry.ArtifactBytes, _ = treeSize(runRoot)
		var entriesAfter int
		entry.EngineCacheBytes, entriesAfter = treeSize(dirs.engine)
		entry.EngineEntriesAdded = entriesAfter - entriesBefore
		entry.EngineCacheHit = entriesBefore > 0 && entriesAfter == entriesBefore
		if outcome.Class != ClassInfrastructure || attempt == 2 || !sweeper.fits(started, 1) {
			break
		}
		entry.FirstAttemptReason = outcome.Reason
		if err := os.RemoveAll(runRoot); err != nil {
			return entry, err
		}
	}
	entry.Class, entry.Reason, entry.Skipped, entry.Subtests, entry.ModuleSHA256 = outcome.Class, outcome.Reason, outcome.Skipped, outcome.Subtests, outcome.ModuleSHA256
	retained, note := retention.keep(entry, runRoot, filepath.Join(options.Artifacts, filepath.Base(dirs.root), name))
	entry.ArtifactRetained, entry.RetentionNote = retained != "", note
	if retained != "" && outcome.ArtifactPath != "" {
		entry.ArtifactPath = filepath.Join(retained, strings.TrimPrefix(outcome.ArtifactPath, runRoot))
	}
	if retained == "" {
		if err := os.RemoveAll(runRoot); err != nil {
			return entry, err
		}
	}
	return entry, nil
}

// retention keeps one exemplar artifact root per distinct non-pass reason
// within the byte budget.
type retention struct {
	budget, used uint64
	reasons      map[string]bool
}

func newRetention(budget uint64, latest map[string]TestEntry) *retention {
	kept := &retention{budget: budget, reasons: map[string]bool{}}
	for _, entry := range latest {
		if entry.ArtifactRetained {
			kept.used += entry.ArtifactBytes
			kept.reasons[string(entry.Class)+"\x00"+entry.Reason] = true
		}
	}
	return kept
}

func (kept *retention) keep(entry TestEntry, source, destination string) (string, string) {
	if entry.Class == ClassPass || entry.ArtifactBytes == 0 {
		return "", ""
	}
	key := string(entry.Class) + "\x00" + entry.Reason
	switch {
	case kept.reasons[key]:
		return "", "exemplar for this reason already retained"
	case kept.used+entry.ArtifactBytes > kept.budget:
		return "", "retention budget exhausted"
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return "", "retain failed: " + err.Error()
	}
	if err := os.RemoveAll(destination); err != nil {
		return "", "retain failed: " + err.Error()
	}
	if err := os.Rename(source, destination); err != nil {
		return "", "retain failed: " + err.Error()
	}
	kept.used += entry.ArtifactBytes
	kept.reasons[key] = true
	return destination, ""
}

func resetDirectory(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return os.MkdirAll(dir, 0o700)
}

// treeSize returns the apparent bytes and regular-file count under root.
func treeSize(root string) (uint64, int) {
	var total uint64
	var files int
	_ = filepath.WalkDir(root, func(_ string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.Type().IsRegular() {
			if info, err := entry.Info(); err == nil {
				total += uint64(info.Size())
				files++
			}
		}
		return nil
	})
	return total, files
}

func (sweeper *Sweeper) baseCandidate(ctx context.Context, tags []string) (Candidate, error) {
	options := sweeper.Options
	candidate := Candidate{ConfigSource: options.ConfigSource, BuildTags: tags, Seed: options.Seed, PerTestTimeout: options.PerTestTimeout.String(), PrepareTimeout: options.PrepareTimeout.String()}
	var err error
	for _, digest := range []struct {
		path   string
		target *string
	}{{options.Gomad, &candidate.GomadSHA256}, {options.Helper, &candidate.HelperSHA256}, {options.Compiler, &candidate.CompilerSHA256}} {
		if *digest.target, err = fileSHA256(digest.path); err != nil {
			return candidate, err
		}
	}
	version := sweeper.Executor.Exec(ctx, Command{Path: options.Compiler, Args: []string{"env", "GOVERSION"}, Dir: options.Work, Env: goEnvironment(options.Environ, "", ""), Timeout: time.Minute})
	if version.Err != nil || version.Exit != 0 {
		return candidate, fmt.Errorf("go env GOVERSION: exit %d %v", version.Exit, version.Err)
	}
	candidate.GoVersion = strings.TrimSpace(string(version.Stdout))
	gitEnv := append(sanitizedEnviron(options.Environ), "GIT_DISCOVERY_ACROSS_FILESYSTEM=1")
	head := sweeper.Executor.Exec(ctx, Command{Path: "git", Args: []string{"-C", options.Checkout, "rev-parse", "HEAD"}, Dir: options.Work, Env: gitEnv, Timeout: time.Minute})
	if head.Err != nil || head.Exit != 0 {
		return candidate, fmt.Errorf("git rev-parse HEAD in %s: exit %d %v %s", options.Checkout, head.Exit, head.Err, head.Stderr)
	}
	candidate.GitSHA = strings.TrimSpace(string(head.Stdout))
	status := sweeper.Executor.Exec(ctx, Command{Path: "git", Args: []string{"-C", options.Checkout, "status", "--porcelain", "--untracked-files=no"}, Dir: options.Work, Env: gitEnv, Timeout: time.Minute})
	if status.Err != nil || status.Exit != 0 {
		return candidate, fmt.Errorf("git status in %s: exit %d %v", options.Checkout, status.Exit, status.Err)
	}
	candidate.GitDirty = strings.TrimSpace(string(status.Stdout)) != ""
	return candidate, nil
}

func fillFromManifest(candidate *Candidate, manifest installation.Manifest) {
	candidate.ModelSHA256 = manifest.Options.ModelSHA256
	candidate.RuntimeImplementationSHA256 = manifest.RuntimeImplementationSHA256
	engine := manifest.Options.Engine
	candidate.Engine = strings.Join([]string{engine.Name, engine.Version, engine.Configuration, engine.HostOS, engine.HostArch}, "/")
	candidate.Profile = manifest.Options.Config.Profile
	candidate.MemoryBytes, candidate.Fuel = manifest.Options.MemoryBytes, manifest.Options.Fuel
	candidate.GuestConfigSHA256 = packageIndependentConfigDigest(manifest.Options.Config)
	if manifest.Options.HelperSHA256 != "" && strings.TrimPrefix(manifest.Options.HelperSHA256, "sha256:") != strings.TrimPrefix(candidate.HelperSHA256, "sha256:") {
		candidate.HelperSHA256 = "mismatch:" + manifest.Options.HelperSHA256
	}
}

// packageIndependentConfigDigest hashes the installed guest configuration
// without the per-package working directory, so every package of one sweep
// shares it.
func packageIndependentConfigDigest(config wasi.Config) string {
	config.WorkingDirectory = ""
	config.Environment = slices.DeleteFunc(slices.Clone(config.Environment), func(entry string) bool { return strings.HasPrefix(entry, "PWD=") })
	config.WritableDirectories = slices.DeleteFunc(slices.Clone(config.WritableDirectories), func(dir string) bool { return strings.HasPrefix(dir, "/workspace/") })
	encoded, err := json.Marshal(config)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// FileSHA256 is the "sha256:<hex>" digest of a file.
func FileSHA256(path string) (string, error) { return fileSHA256(path) }

func fileSHA256(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	if err := errors.Join(copyErr, file.Close()); err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil)), nil
}

// ReadArtifact opens a retained artifact through the evidence package and
// returns the outcome and bounded streams the classifier reads.
func ReadArtifact(path string) ArtifactFacts {
	opened, err := artifact.OpenArtifact(path)
	if err != nil {
		return ArtifactFacts{ReadError: err.Error()}
	}
	manifest := opened.Manifest()
	facts := ArtifactFacts{Kind: manifest.ArtifactKind, ModuleSHA256: string(manifest.Target.SHA256), Domain: manifest.Outcome.Domain, Reason: manifest.Outcome.Reason, Termination: manifest.Outcome.Termination, StoredBytes: opened.StoredBytes()}
	if manifest.Outcome.ExitCode != nil {
		code := uint64(*manifest.Outcome.ExitCode)
		facts.ExitCode = &code
	}
	var readErrors []error
	for _, stream := range []struct {
		record record.Stream
		target *[]byte
	}{{manifest.Streams.Stdout, &facts.Stdout}, {manifest.Streams.Stderr, &facts.Stderr}} {
		if stream.record.File == "" {
			continue
		}
		data, err := opened.ReadPayload(stream.record.File, uint64(stream.record.RetainedBytes))
		readErrors = append(readErrors, err)
		*stream.target = data
	}
	facts.StdoutTruncated = manifest.Streams.Stdout.Truncated
	readErrors = append(readErrors, opened.Close())
	if err := errors.Join(readErrors...); err != nil {
		facts.ReadError = err.Error()
	}
	return facts
}
