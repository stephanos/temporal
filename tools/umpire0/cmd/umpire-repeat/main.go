// Command umpire-repeat runs a live Testpilot selection many times and counts its failures by
// signature.
//
// `umpire-repeat run` builds the live test binary once, runs the selection N times, one process per
// iteration or one process with `-test.count=N`, reads each process's test2json stream, and appends
// one record line per finished iteration. A failing leaf test's signature is the
// `TESTPILOT-SIGNATURE` line it printed, or else one built from its first failing assertion. Before
// and after every process it fingerprints what the tests read from the working tree, and stops when
// that changed, so a loop measures one tree or says it could not. `umpire-repeat summarize` prints
// the summary of one or more record files. It only observes: it never changes the suite.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	testPackageDirectory = "tests"
	testPackagePath      = "go.temporal.io/server/tests"
	testBuildTags        = "test_dep integration"
	// interruptGrace is how long an interrupted test binary gets to exit before it is killed.
	interruptGrace = 30 * time.Second
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root, err := gitOutput(context.Background(), ".", "rev-parse", "--show-toplevel")
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "find the repository root: %s\n", err)
		os.Exit(exitToolingError)
	}
	code := Run(ctx, os.Args[1:], os.Stdout, os.Stderr, liveEnvironment(root))
	stop()
	os.Exit(code)
}

func liveEnvironment(root string) environment {
	return environment{
		root:  root,
		build: func(ctx context.Context, binary string) error { return buildTests(ctx, root, binary) },
		execute: func(ctx context.Context, run invocation, events io.Writer) (bool, error) {
			return executeTests(ctx, root, run, events)
		},
		fingerprint: func() (fingerprint, error) { return fingerprintTree(root) },
		tree:        func() (treeState, error) { return readTree(root) },
		host:        readHostLoad,
		now:         time.Now,
	}
}

func buildTests(ctx context.Context, root, binary string) error {
	command := exec.CommandContext(ctx, "go", "test", "-c", "-tags", testBuildTags, "-o", binary, "./"+testPackageDirectory)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// executeTests runs the binary with its output piped through test2json rather than under it, so an
// interrupt reaches the binary itself and test2json still converts everything up to that point.
func executeTests(ctx context.Context, root string, run invocation, events io.Writer) (bool, error) {
	temporary, err := filepath.EvalSymlinks(os.TempDir())
	if err != nil {
		return false, err
	}
	test := exec.CommandContext(ctx, run.Binary,
		"-test.v=test2json",
		"-test.run", run.Selection,
		"-test.count", strconv.Itoa(run.Count),
		"-test.timeout", run.Timeout.String())
	// Test fixtures are read relative to the package directory, as `go test` runs them.
	test.Dir = filepath.Join(root, testPackageDirectory)
	test.Env = append(os.Environ(), "TMPDIR="+temporary, runDirVariable+"="+run.RunDir)
	test.Cancel = func() error { return test.Process.Signal(os.Interrupt) }
	test.WaitDelay = interruptGrace

	convert := exec.Command("go", "tool", "test2json", "-t", "-p", testPackagePath)
	convert.Dir = root
	convert.Stdout = events
	convert.Stderr = os.Stderr
	pipeReader, pipeWriter, err := os.Pipe()
	if err != nil {
		return false, err
	}
	convert.Stdin = pipeReader
	test.Stdout = pipeWriter
	test.Stderr = pipeWriter
	if err := convert.Start(); err != nil {
		_ = pipeReader.Close()
		_ = pipeWriter.Close()
		return false, fmt.Errorf("start test2json: %w", err)
	}
	startErr := test.Start()
	// Both ends now belong to the children; the converter sees end of input when the binary exits.
	_ = pipeReader.Close()
	_ = pipeWriter.Close()
	if startErr != nil {
		_ = convert.Wait()
		return false, fmt.Errorf("start the test binary: %w", startErr)
	}
	testErr := test.Wait()
	if err := convert.Wait(); err != nil {
		return false, fmt.Errorf("test2json: %w", err)
	}
	var exit *exec.ExitError
	if errors.As(testErr, &exit) {
		return true, nil
	}
	return false, testErr
}

// fingerprintTree hashes the tracked and untracked (not ignored) files under the fingerprint roots
// and the Lean helper binaries.
func fingerprintTree(root string) (fingerprint, error) {
	listing, err := gitOutput(context.Background(), root,
		append([]string{"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--"}, fingerprintRoots...)...)
	if err != nil {
		return fingerprint{}, err
	}
	paths := map[string]string{}
	for name := range strings.SplitSeq(listing, "\x00") {
		if name == "" {
			continue
		}
		hash, err := hashPath(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return fingerprint{}, err
		}
		paths[name] = hash
	}
	for _, name := range fingerprintBinaries {
		hash, err := hashPath(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			return fingerprint{}, err
		}
		paths[name] = hash
	}
	return newFingerprint(paths), nil
}

func hashPath(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "missing", nil
	}
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		return "symlink:" + target, nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:]), nil
}

func readTree(root string) (treeState, error) {
	commit, err := gitOutput(context.Background(), root, "rev-parse", "HEAD")
	if err != nil {
		return treeState{}, err
	}
	status, err := gitOutput(context.Background(), root,
		append([]string{"status", "--porcelain", "--"}, fingerprintRoots...)...)
	if err != nil {
		return treeState{}, err
	}
	return treeState{Commit: commit, Dirty: status != ""}, nil
}

func gitOutput(ctx context.Context, directory string, arguments ...string) (string, error) {
	command := exec.CommandContext(ctx, "git", arguments...)
	command.Dir = directory
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(arguments, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(output)), nil
}

var goTestProcess = regexp.MustCompile(`(^|/)go test\b|\.test\s.*-test\.`)

// readHostLoad reads what the loop starts under. It is best effort: a host that cannot say records
// "unknown" rather than stopping the loop.
func readHostLoad() hostLoad {
	load := "unknown"
	if content, err := os.ReadFile("/proc/loadavg"); err == nil && len(strings.Fields(string(content))) >= 3 {
		load = strings.Join(strings.Fields(string(content))[:3], " ")
	} else if output, err := exec.Command("sysctl", "-n", "vm.loadavg").Output(); err == nil {
		load = strings.Trim(strings.TrimSpace(string(output)), "{} ")
	}
	processes := 0
	if output, err := exec.Command("ps", "-Ao", "command=").Output(); err == nil {
		for line := range strings.SplitSeq(string(output), "\n") {
			if goTestProcess.MatchString(line) {
				processes++
			}
		}
	}
	return hostLoad{LoadAverage: load, GoTestProcesses: processes}
}
