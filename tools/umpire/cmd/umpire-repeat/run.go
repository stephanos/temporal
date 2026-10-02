package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.temporal.io/server/common/testing/testpilot/publish"
	"go.temporal.io/server/tools/umpire/internal/cli"
)

// Exit codes, as the other Umpire commands rank them: a failed iteration outranks a stop, because
// the failure is what the loop ran for; a tooling failure outranks everything, because nothing the
// loop reports can be trusted.
const (
	exitPassed       = 0
	exitFailed       = 1
	exitStopped      = 2
	exitToolingError = 3
)

const (
	modeProcess   = "process"
	modeInProcess = "in-process"

	defaultTimeout = 30 * time.Minute

	// runDirVariable names the directory the affected live tests write every closed Run to.
	runDirVariable = "UMPIRE_REPEAT_RUN_DIR"
)

// fingerprintRoots are the directories whose tracked and untracked files the live tests read at run
// time, relative to the repository root; fingerprintBinaries are the IR bridge binaries they execute.
var (
	fingerprintRoots    = []string{"tests", "common/testing/testpilot", "tools/umpire", "model/ir", "model/cases"}
	fingerprintBinaries = []string{".build/umpire-ir-bridge"}
)

type config struct {
	Selection  string
	Count      int
	Mode       string
	RecordPath string
	Timeout    time.Duration
}

// invocation is one process of the test binary.
type invocation struct {
	Binary    string
	Selection string
	Count     int
	Timeout   time.Duration
	RunDir    string
}

// fingerprint is the hash of every run-time input, with each path's own hash so a change can be
// named.
type fingerprint struct {
	Digest string
	Paths  map[string]string
}

func newFingerprint(paths map[string]string) fingerprint {
	names := make([]string, 0, len(paths))
	for name := range paths {
		names = append(names, name)
	}
	slices.Sort(names)
	digest := sha256.New()
	for _, name := range names {
		_, _ = fmt.Fprintf(digest, "%s\x00%s\n", name, paths[name])
	}
	return fingerprint{Digest: hex.EncodeToString(digest.Sum(nil)), Paths: paths}
}

// changed names the first path, in order, whose hash differs, appeared or went away.
func (f fingerprint) changed(other fingerprint) string {
	var names []string
	for name, hash := range f.Paths {
		if other.Paths[name] != hash {
			names = append(names, name)
		}
	}
	for name := range other.Paths {
		if _, ok := f.Paths[name]; !ok {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

type treeState struct {
	Commit string
	Dirty  bool
}

// environment is everything the loop does outside itself. The real one runs Go and git; a test
// supplies canned streams.
type environment struct {
	// root is the repository root the fingerprint roots are relative to.
	root string
	// build compiles the live test binary to the path it is given.
	build func(ctx context.Context, binary string) error
	// execute runs one process and writes its test2json stream to events. It reports whether the
	// process exited non-zero; an error is a process that could not be run at all.
	execute     func(ctx context.Context, run invocation, events io.Writer) (bool, error)
	fingerprint func() (fingerprint, error)
	tree        func() (treeState, error)
	host        func() hostLoad
	now         func() time.Time
}

// Run is the whole command: `run` loops the selection, `summarize` prints the summary of record
// files.
func Run(ctx context.Context, arguments []string, stdout, stderr io.Writer, env environment) int {
	if len(arguments) == 0 {
		cli.WriteLine(stderr, "usage: umpire-repeat run --select <regex> --count <n> --mode process|in-process --record <file> | umpire-repeat summarize <file>...")
		return exitToolingError
	}
	switch arguments[0] {
	case "run":
		return runLoop(ctx, arguments[1:], stdout, stderr, env)
	case "summarize":
		return summarize(arguments[1:], stdout, stderr)
	default:
		cli.WriteLine(stderr, "unknown subcommand %q: want run or summarize", arguments[0])
		return exitToolingError
	}
}

func summarize(paths []string, stdout, stderr io.Writer) int {
	if len(paths) == 0 {
		cli.WriteLine(stderr, "summarize needs at least one record file")
		return exitToolingError
	}
	var all []sourcedRecord
	for _, path := range paths {
		records, err := readRecords(path)
		if err != nil {
			cli.WriteLine(stderr, "%s", err)
			return exitToolingError
		}
		all = append(all, records...)
	}
	if err := sameFingerprint(all); err != nil {
		cli.WriteLine(stderr, "%s", err)
		return exitToolingError
	}
	plain := make([]record, 0, len(all))
	for _, sourced := range all {
		plain = append(plain, sourced.record)
	}
	printSummary(stdout, plain)
	return exitPassed
}

func parseConfig(arguments []string, stderr io.Writer) (config, error) {
	var configuration config
	flags := flag.NewFlagSet("umpire-repeat run", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&configuration.Selection, "select", "", "the -test.run selection, e.g. '^TestTestpilotNexusPairCase$'")
	flags.IntVar(&configuration.Count, "count", 0, "iterations to run")
	flags.StringVar(&configuration.Mode, "mode", "", "process (one process per iteration) or in-process (one process with -test.count)")
	flags.StringVar(&configuration.RecordPath, "record", "", "record file each finished iteration is appended to")
	flags.DurationVar(&configuration.Timeout, "timeout", defaultTimeout, "-test.timeout of each process")
	if err := flags.Parse(arguments); err != nil {
		return config{}, err
	}
	fail := func(format string, arguments ...any) (config, error) {
		cli.WriteLine(stderr, format, arguments...)
		return config{}, fmt.Errorf(format, arguments...)
	}
	switch {
	case flags.NArg() != 0:
		return fail("umpire-repeat run accepts no positional arguments")
	case configuration.Selection == "":
		return fail("--select is required")
	case configuration.Count <= 0:
		return fail("--count must be positive")
	case configuration.Mode != modeProcess && configuration.Mode != modeInProcess:
		return fail("--mode must be %s or %s", modeProcess, modeInProcess)
	case configuration.RecordPath == "":
		return fail("--record is required")
	case configuration.Timeout <= 0:
		return fail("--timeout must be positive")
	}
	return configuration, nil
}

// loop is one run of the command: its configuration, the state it measures, and what it finished.
type loop struct {
	configuration config
	env           environment
	stdout        io.Writer
	stderr        io.Writer
	baseline      fingerprint
	tree          treeState
	host          hostLoad
	started       string
	runRoot       string
	recordFile    *os.File
	finished      []record
	failed        bool
}

func runLoop(ctx context.Context, arguments []string, stdout, stderr io.Writer, env environment) int {
	configuration, err := parseConfig(arguments, stderr)
	if err != nil {
		return exitToolingError
	}
	recordPath, err := filepath.Abs(configuration.RecordPath)
	if err != nil {
		cli.WriteLine(stderr, "--record: %s", err)
		return exitToolingError
	}
	configuration.RecordPath = recordPath
	if root := fingerprintRootHolding(env.root, recordPath); root != "" {
		cli.WriteLine(stderr, "--record %s is under %s, which the loop fingerprints; its Run captures would stop the loop", recordPath, root)
		return exitToolingError
	}

	baseline, err := env.fingerprint()
	if err != nil {
		cli.WriteLine(stderr, "fingerprint: %s", err)
		return exitToolingError
	}
	if _, err := os.Stat(recordPath); err == nil {
		// A resumed loop appends to its record; one measured on other inputs could never be summed
		// with it, so the loop refuses before spending hours on iterations it would have to drop.
		existing, err := readRecords(recordPath)
		if err != nil {
			cli.WriteLine(stderr, "%s", err)
			return exitToolingError
		}
		if len(existing) > 0 && existing[0].Fingerprint != baseline.Digest {
			cli.WriteLine(stderr, "%s:%d has fingerprint %s but the tree has %s; use a new record file",
				existing[0].path, existing[0].line, existing[0].Fingerprint, baseline.Digest)
			return exitToolingError
		}
	}
	tree, err := env.tree()
	if err != nil {
		cli.WriteLine(stderr, "tree: %s", err)
		return exitToolingError
	}

	scratch, err := os.MkdirTemp("", "umpire-repeat-")
	if err != nil {
		cli.WriteLine(stderr, "%s", err)
		return exitToolingError
	}
	defer func() { _ = os.RemoveAll(scratch) }()
	binary := filepath.Join(scratch, "tests.test")
	if err := env.build(ctx, binary); err != nil {
		cli.WriteLine(stderr, "build the live test binary: %s", err)
		return exitToolingError
	}

	started := env.now()
	current := &loop{
		configuration: configuration,
		env:           env,
		stdout:        stdout,
		stderr:        stderr,
		baseline:      baseline,
		tree:          tree,
		host:          env.host(),
		started:       started.UTC().Format(time.RFC3339),
		runRoot:       strings.TrimSuffix(recordPath, filepath.Ext(recordPath)) + ".runs",
	}
	cli.WriteLine(stdout, "%s", loopKey{loop: current.started, commit: tree.Commit, dirty: tree.Dirty, host: current.host}.describe())
	code := current.iterate(ctx, binary, started.UTC().Format("20060102T150405"))
	if current.recordFile != nil {
		if err := current.recordFile.Close(); err != nil {
			cli.WriteLine(stderr, "close %s: %s", recordPath, err)
			code = exitToolingError
		}
	}
	printSummary(stdout, current.finished)
	return code
}

// fingerprintRootHolding names the fingerprinted directory path is under, if any.
func fingerprintRootHolding(root, path string) string {
	resolvedRoot, err := filepath.Abs(root)
	if err != nil {
		return ""
	}
	for _, candidate := range fingerprintRoots {
		directory := filepath.Join(resolvedRoot, filepath.FromSlash(candidate))
		if publish.Within(directory, path) {
			return directory
		}
	}
	return ""
}

func (l *loop) iterate(ctx context.Context, binary, stamp string) int {
	processes, perProcess := l.configuration.Count, 1
	if l.configuration.Mode == modeInProcess {
		processes, perProcess = 1, l.configuration.Count
	}
	for process := 1; process <= processes; process++ {
		if ctx.Err() != nil {
			cli.WriteLine(l.stdout, "interrupted after %d iterations", len(l.finished))
			return l.exitCode(true)
		}
		runDir := filepath.Join(l.runRoot, fmt.Sprintf("%s-%d", stamp, process))
		if code, stop := l.process(ctx, binary, runDir, perProcess); stop {
			return code
		}
	}
	return l.exitCode(false)
}

// process runs one process and records the iterations it finished; stop ends the loop with code.
func (l *loop) process(ctx context.Context, binary, runDir string, perProcess int) (code int, stop bool) {
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		cli.WriteLine(l.stderr, "run capture directory: %s", err)
		return exitToolingError, true
	}
	collected, processFailed, err := l.runProcess(ctx, invocation{
		Binary: binary, Selection: l.configuration.Selection, Count: perProcess,
		Timeout: l.configuration.Timeout, RunDir: runDir,
	})
	interrupted := ctx.Err() != nil
	switch {
	case err != nil && !interrupted:
		cli.WriteLine(l.stderr, "run the live test binary: %s", err)
		return exitToolingError, true
	case collected.failedBuild != "":
		cli.WriteLine(l.stderr, "build failed: %s", collected.failedBuild)
		return exitToolingError, true
	case !interrupted && len(collected.iterations) == 0 && !processFailed:
		// Go reports a selection that matches nothing as a passing package.
		cli.WriteLine(l.stderr, "selection %q matched no test", l.configuration.Selection)
		return exitToolingError, true
	default:
	}
	iterations, signatures := finishedIterations(collected, processFailed, interrupted, l.configuration.Mode)
	now, err := l.env.fingerprint()
	if err != nil {
		cli.WriteLine(l.stderr, "fingerprint: %s", err)
		return exitToolingError, true
	}
	if now.Digest != l.baseline.Digest {
		// The process may have read either version, so none of its iterations measures the
		// commit the loop started on.
		cli.WriteLine(l.stdout, "input changed: %s; stopping after %d iterations", l.baseline.changed(now), len(l.finished))
		return l.exitCode(true), true
	}
	// Runs are assigned over every iteration the process started, so a Run the interrupted
	// iteration wrote stays with it and is dropped with it.
	runs, err := capturedRuns(runDir, collected.iterations)
	if err != nil {
		cli.WriteLine(l.stderr, "run capture: %s", err)
		return exitToolingError, true
	}
	base := len(l.finished)
	for index, it := range iterations {
		var captured []string
		if index < len(runs) {
			captured = runs[index]
		}
		if err := l.finish(base+index+1, it, signatures[index], captured); err != nil {
			cli.WriteLine(l.stderr, "%s", err)
			return exitToolingError, true
		}
	}
	if interrupted {
		cli.WriteLine(l.stdout, "interrupted after %d iterations", len(l.finished))
		return l.exitCode(true), true
	}
	if l.configuration.Mode == modeInProcess && len(iterations) < perProcess {
		cli.WriteLine(l.stdout, "the process ended after %d of %d iterations", len(iterations), perProcess)
		return l.exitCode(true), true
	}
	return exitPassed, false
}

// finishedIterations are the iterations a process finished, with each one's signatures. The
// iteration an interrupt cut short is neither counted nor recorded; in one process holding many,
// the ones before it did finish. A process that failed with no test to blame is still one failed
// iteration.
func finishedIterations(collected *collector, processFailed, interrupted bool, mode string) ([]*iteration, [][]Signature) {
	iterations := collected.iterations
	if interrupted {
		if mode == modeProcess || len(iterations) == 0 {
			return nil, nil
		}
		iterations = iterations[:len(iterations)-1]
	}
	if len(iterations) == 0 && processFailed && !interrupted {
		iterations = []*iteration{{number: 1, tests: map[string]*testState{}, packageOutput: collected.preamble}}
	}
	signatures := make([][]Signature, len(iterations))
	anyFailure := false
	for index, it := range iterations {
		signatures[index] = it.failures()
		anyFailure = anyFailure || len(signatures[index]) > 0
	}
	if processFailed && !anyFailure && !interrupted {
		last := len(iterations) - 1
		signatures[last] = []Signature{iterations[last].processFailure()}
	}
	return iterations, signatures
}

func (l *loop) exitCode(stopped bool) int {
	switch {
	case l.failed:
		return exitFailed
	case stopped:
		return exitStopped
	default:
		return exitPassed
	}
}

// runProcess runs one process and collects its events as they stream.
func (l *loop) runProcess(ctx context.Context, run invocation) (*collector, bool, error) {
	collected := newCollector()
	reader, writer := io.Pipe()
	type result struct {
		failed bool
		err    error
	}
	done := make(chan result, 1)
	go func() {
		failed, err := l.env.execute(ctx, run, writer)
		_ = writer.CloseWithError(err)
		done <- result{failed: failed, err: err}
	}()
	readErr := readEvents(reader, collected.add)
	// Drain so the process never blocks on a reader that stopped.
	_, _ = io.Copy(io.Discard, reader)
	finished := <-done
	if finished.err != nil {
		return collected, finished.failed, finished.err
	}
	return collected, finished.failed || collected.packageFailed, readErr
}

// capturedRuns lists the Run files written under runDir, each assigned to the iteration that was
// running when it was written: the last one that started at or before its modification time.
func capturedRuns(runDir string, iterations []*iteration) ([][]string, error) {
	runs := make([][]string, len(iterations))
	if len(iterations) == 0 {
		return runs, nil
	}
	err := filepath.WalkDir(runDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		index := 0
		for candidate, it := range iterations {
			if !it.started.IsZero() && !info.ModTime().Before(it.started) {
				index = candidate
			}
		}
		runs[index] = append(runs[index], path)
		return nil
	})
	return runs, err
}

func (l *loop) finish(number int, it *iteration, signatures []Signature, runs []string) error {
	rec := record{
		Iteration:   number,
		Mode:        l.configuration.Mode,
		Selection:   l.configuration.Selection,
		Commit:      l.tree.Commit,
		Dirty:       l.tree.Dirty,
		Fingerprint: l.baseline.Digest,
		Outcome:     outcomePass,
		Failures:    []failure{},
		Runs:        runs,
		Tests:       it.topLevelTests(),
		Loop:        l.started,
		Host:        l.host,
	}
	if rec.Runs == nil {
		rec.Runs = []string{}
	}
	if rec.Tests == nil {
		rec.Tests = []string{}
	}
	hashes := make([]string, 0, len(signatures))
	for _, signature := range signatures {
		hash, err := signature.Hash()
		if err != nil {
			return err
		}
		rec.Failures = append(rec.Failures, failure{Test: signature.Test, SignatureHash: hash, Signature: signature})
		hashes = append(hashes, hash)
	}
	if len(signatures) > 0 {
		rec.Outcome = outcomeFail
		l.failed = true
	}
	if err := l.append(rec); err != nil {
		return err
	}
	l.finished = append(l.finished, rec)
	cli.WriteLine(l.stdout, "%s", strings.TrimSpace(strings.Join(append([]string{fmt.Sprint(number), rec.Outcome}, hashes...), " ")))
	return nil
}

// append writes one record line. The file is opened on the first finished iteration, so a loop that
// finishes none leaves no record behind.
func (l *loop) append(rec record) error {
	if l.recordFile == nil {
		file, err := os.OpenFile(l.configuration.RecordPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("open record: %w", err)
		}
		l.recordFile = file
	}
	encoded, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	if _, err := l.recordFile.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("write record: %w", err)
	}
	return nil
}
