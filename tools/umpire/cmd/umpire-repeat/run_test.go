package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeProcess is one canned process: the test2json stream it prints, whether it exits non-zero,
// and what it does to the world while it runs.
type fakeProcess struct {
	stream string
	failed bool
	during func(run invocation) error
}

type fakeWorld struct {
	env          environment
	invocations  []invocation
	fingerprints []fingerprint
}

var baselineFingerprint = newFingerprint(map[string]string{"tests/live_test.go": "one", "tools/umpire/cmd/umpire-run/run.go": "two"})

func newFakeWorld(t *testing.T, processes ...fakeProcess) *fakeWorld {
	t.Helper()
	world := &fakeWorld{}
	world.env = environment{
		root:  t.TempDir(),
		build: func(context.Context, string) error { return nil },
		execute: func(_ context.Context, run invocation, events io.Writer) (bool, error) {
			// This runs on the loop's goroutine, not the test's, so it reports through its error: a
			// require here would stop the goroutine without closing the stream.
			index := len(world.invocations)
			world.invocations = append(world.invocations, run)
			if index >= len(processes) {
				return false, fmt.Errorf("process %d was not canned", index+1)
			}
			process := processes[index]
			if process.during != nil {
				if err := process.during(run); err != nil {
					return false, err
				}
			}
			stream, err := os.ReadFile(filepath.Join("testdata", process.stream+".jsonl"))
			if err != nil {
				return false, err
			}
			_, err = events.Write(stream)
			return process.failed, err
		},
		fingerprint: func() (fingerprint, error) {
			if len(world.fingerprints) == 0 {
				return baselineFingerprint, nil
			}
			next := world.fingerprints[0]
			world.fingerprints = world.fingerprints[1:]
			return next, nil
		},
		tree: func() (treeState, error) { return treeState{Commit: "0c39927b13", Dirty: false}, nil },
		host: func() hostLoad { return hostLoad{LoadAverage: "1.00 2.00 3.00", GoTestProcesses: 2} },
		now:  func() time.Time { return time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC) },
	}
	return world
}

func runArguments(record string, count int, mode string) []string {
	return []string{"run", "--select", "^TestTestpilotSample$", "--count", fmt.Sprint(count), "--mode", mode, "--record", record}
}

func readRecordFile(t *testing.T, path string) []record {
	t.Helper()
	sourced, err := readRecords(path)
	require.NoError(t, err)
	records := make([]record, 0, len(sourced))
	for _, entry := range sourced {
		records = append(records, entry.record)
	}
	return records
}

func iterationLines(stdout string) []string {
	var lines []string
	for line := range strings.SplitSeq(stdout, "\n") {
		if strings.HasSuffix(line, " PASS") || strings.Contains(line, " PASS ") || strings.Contains(line, " FAIL") {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestRunRecordsOneSignaturePerFailingLeafTest(t *testing.T) {
	for _, tc := range []struct {
		name     string
		process  fakeProcess
		code     int
		expected []Signature
	}{
		{name: "all pass", process: fakeProcess{stream: "pass"}, code: exitPassed},
		{
			name: "the first signature line", process: fakeProcess{stream: "signature", failed: true}, code: exitFailed,
			expected: []Signature{{
				Test: "TestTestpilotSample/hsm", Assertion: "live_test.go:88: run disposition Incomplete, want Completed",
				RunDisposition: "Incomplete", VerdictStatus: "Inconclusive",
				UnresolvedRules: []UnresolvedRule{{RuleID: "clause-two", Status: "Pending"}, {RuleID: "clause-one", Status: "Pending"}},
				Diagnostics:     []Diagnostic{{Kind: "instruction-timeout", Code: "finish-workflow"}},
				Location:        "live_test.go:88",
			}},
		},
		{
			name: "a testify failure without a signature line", process: fakeProcess{stream: "fallback", failed: true}, code: exitFailed,
			expected: []Signature{{Test: "TestTestpilotSample", Assertion: "Not equal", Location: "/a/x_test.go:6"}},
		},
		{
			name: "a failure with nothing readable", process: fakeProcess{stream: "unparsed", failed: true}, code: exitFailed,
			expected: []Signature{{Test: "TestTestpilotSample/hsm", Reserved: reservedUnparsed}},
		},
		{
			name: "a crash before the test finished", process: fakeProcess{stream: "crash", failed: true}, code: exitFailed,
			expected: []Signature{{Test: "TestTestpilotSample", Reserved: reservedProcess, Assertion: "about to exit"}},
		},
		{
			name: "a test binary timeout", process: fakeProcess{stream: "timeout", failed: true}, code: exitFailed,
			expected: []Signature{{Test: "TestTestpilotSample", Reserved: reservedProcess, Assertion: "panic: test timed out after 2s"}},
		},
		{
			name: "text from an early fatal error", process: fakeProcess{stream: "early-fatal", failed: true}, code: exitFailed,
			expected: []Signature{{Reserved: reservedUnparsed, Detail: "fatal error: runtime: cannot allocate memory"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := newFakeWorld(t, tc.process)
			path := filepath.Join(t.TempDir(), "record.jsonl")
			var stdout, stderr bytes.Buffer

			code := Run(context.Background(), runArguments(path, 1, modeProcess), &stdout, &stderr, world.env)

			require.Equal(t, tc.code, code, stderr.String())
			records := readRecordFile(t, path)
			require.Len(t, records, 1)
			var signatures []Signature
			var hashes []string
			for _, f := range records[0].Failures {
				require.Equal(t, f.Signature.Test, f.Test)
				require.Equal(t, mustHash(t, f.Signature), f.SignatureHash)
				signatures = append(signatures, f.Signature)
				hashes = append(hashes, f.SignatureHash)
			}
			require.Equal(t, tc.expected, signatures)
			require.Equal(t, []invocation{{
				Binary: world.invocations[0].Binary, Selection: "^TestTestpilotSample$", Count: 1,
				Timeout: defaultTimeout, RunDir: world.invocations[0].RunDir,
			}}, world.invocations)
			line := "1 PASS"
			if len(hashes) > 0 {
				line = "1 FAIL " + strings.Join(hashes, " ")
			}
			require.Equal(t, []string{line}, iterationLines(stdout.String()))
		})
	}
}

func TestRunRecordsTheIterationContext(t *testing.T) {
	world := newFakeWorld(t, fakeProcess{stream: "pass"})
	path := filepath.Join(t.TempDir(), "record.jsonl")
	var stdout, stderr bytes.Buffer

	require.Equal(t, exitPassed, Run(context.Background(), runArguments(path, 1, modeProcess), &stdout, &stderr, world.env), stderr.String())

	require.Equal(t, []record{{
		Iteration: 1, Mode: modeProcess, Selection: "^TestTestpilotSample$", Commit: "0c39927b13",
		Fingerprint: baselineFingerprint.Digest, Outcome: outcomePass, Failures: []failure{}, Runs: []string{},
		Tests: []string{"TestTestpilotSample"}, Loop: "2026-09-26T12:00:00Z",
		Host: hostLoad{LoadAverage: "1.00 2.00 3.00", GoTestProcesses: 2},
	}}, readRecordFile(t, path))
	require.Contains(t, stdout.String(), "TestTestpilotSample 0/1 rate 0.00% (95% CI 0.00%-97.50%, rule of three <= 300.00%)")
}

// Neither a build failure nor a selection that matches nothing ever counts as an iteration.
func TestRunRecordsNothingWhenNoIterationCouldRun(t *testing.T) {
	for _, tc := range []struct {
		name    string
		build   error
		process fakeProcess
		stderr  string
	}{
		{name: "build failure", build: errors.New("exit status 1\ntests/live_test.go:1: syntax error"), stderr: "build the live test binary: exit status 1"},
		{name: "no test matched", process: fakeProcess{stream: "no-match"}, stderr: `selection "^TestTestpilotSample$" matched no test`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := newFakeWorld(t, tc.process)
			if tc.build != nil {
				world.env.build = func(context.Context, string) error { return tc.build }
			}
			path := filepath.Join(t.TempDir(), "record.jsonl")
			var stdout, stderr bytes.Buffer

			code := Run(context.Background(), runArguments(path, 3, modeProcess), &stdout, &stderr, world.env)

			require.Equal(t, exitToolingError, code)
			require.Contains(t, stderr.String(), tc.stderr)
			require.NoFileExists(t, path)
			require.Empty(t, iterationLines(stdout.String()))
		})
	}
}

func TestRunSplitsAnInProcessCountIntoIterations(t *testing.T) {
	world := newFakeWorld(t, fakeProcess{stream: "in-process-3", failed: true})
	path := filepath.Join(t.TempDir(), "record.jsonl")
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), runArguments(path, 3, modeInProcess), &stdout, &stderr, world.env)

	require.Equal(t, exitFailed, code, stderr.String())
	require.Len(t, world.invocations, 1)
	require.Equal(t, 3, world.invocations[0].Count)
	records := readRecordFile(t, path)
	require.Len(t, records, 3)
	hash := mustHash(t, Signature{Test: "TestTestpilotSample/hsm", Assertion: "verdict Inconclusive, want Satisfied"})
	require.Equal(t, []string{"1 PASS", "2 FAIL " + hash, "3 PASS"}, iterationLines(stdout.String()))
	require.Equal(t, []Signature{{
		Test: "TestTestpilotSample/hsm", Assertion: "verdict Inconclusive, want Satisfied", Location: "live_test.go:42",
	}}, []Signature{records[1].Failures[0].Signature})
	require.Contains(t, stdout.String(), "TestTestpilotSample 1/3 ")
	require.Contains(t, stdout.String(), hash+" 1/3 ")
}

func TestRunStopsWhenAnInputChangesAndNamesIt(t *testing.T) {
	changed := newFingerprint(map[string]string{"tests/live_test.go": "edited", "tools/umpire/cmd/umpire-run/run.go": "two"})
	world := newFakeWorld(t, fakeProcess{stream: "pass"}, fakeProcess{stream: "pass"}, fakeProcess{stream: "pass"})
	// The baseline, unchanged after the first process, changed after the second.
	world.fingerprints = []fingerprint{baselineFingerprint, baselineFingerprint, changed}
	path := filepath.Join(t.TempDir(), "record.jsonl")
	var stdout, stderr bytes.Buffer

	code := Run(context.Background(), runArguments(path, 3, modeProcess), &stdout, &stderr, world.env)

	require.Equal(t, exitStopped, code, stderr.String())
	require.Len(t, world.invocations, 2)
	require.Len(t, readRecordFile(t, path), 1, "the process that may have read the edit is not recorded")
	require.Contains(t, stdout.String(), "input changed: tests/live_test.go; stopping after 1 iterations")
	require.Contains(t, stdout.String(), "summary 1 iterations")
}

func TestRunRecordsOnlyCompletedIterationsWhenInterrupted(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	world := newFakeWorld(t,
		fakeProcess{stream: "pass"},
		fakeProcess{stream: "crash", failed: true, during: func(invocation) error { cancel(); return nil }})
	path := filepath.Join(t.TempDir(), "record.jsonl")
	var stdout, stderr bytes.Buffer

	code := Run(ctx, runArguments(path, 3, modeProcess), &stdout, &stderr, world.env)

	require.Equal(t, exitStopped, code, stderr.String())
	require.Len(t, world.invocations, 2)
	records := readRecordFile(t, path)
	require.Len(t, records, 1)
	require.Equal(t, outcomePass, records[0].Outcome)
	require.Equal(t, []string{"1 PASS"}, iterationLines(stdout.String()))
	require.Contains(t, stdout.String(), "interrupted after 1 iterations")
	require.Contains(t, stdout.String(), "TestTestpilotSample 0/1 ")
}

func TestRunListsTheRunsEachIterationCaptured(t *testing.T) {
	capture := func(name string) func(invocation) error {
		return func(run invocation) error {
			return os.WriteFile(filepath.Join(run.RunDir, name), []byte("{}"), 0o644)
		}
	}
	world := newFakeWorld(t,
		fakeProcess{stream: "pass", during: capture("hsm.json")},
		fakeProcess{stream: "pass", during: capture("chasm.json")})
	path := filepath.Join(t.TempDir(), "record.jsonl")
	var stdout, stderr bytes.Buffer

	require.Equal(t, exitPassed, Run(context.Background(), runArguments(path, 2, modeProcess), &stdout, &stderr, world.env), stderr.String())

	records := readRecordFile(t, path)
	require.Len(t, records, 2)
	require.Equal(t, []string{filepath.Join(world.invocations[0].RunDir, "hsm.json")}, records[0].Runs)
	require.Equal(t, []string{filepath.Join(world.invocations[1].RunDir, "chasm.json")}, records[1].Runs)
	require.NotEqual(t, world.invocations[0].RunDir, world.invocations[1].RunDir)
	require.Equal(t, filepath.Dir(path), filepath.Dir(filepath.Dir(world.invocations[0].RunDir)), "captures live beside the record")
}

func writeRecords(t *testing.T, path string, records ...record) {
	t.Helper()
	var content []byte
	for _, rec := range records {
		encoded, err := json.Marshal(rec)
		require.NoError(t, err)
		content = append(append(content, encoded...), '\n')
	}
	require.NoError(t, os.WriteFile(path, content, 0o644))
}

func sampleRecord(iteration int, fingerprint string, failures ...failure) record {
	outcome := outcomePass
	if len(failures) > 0 {
		outcome = outcomeFail
	}
	return record{
		Iteration: iteration, Mode: modeProcess, Selection: "^TestTestpilotSample$", Commit: "c",
		Fingerprint: fingerprint, Outcome: outcome, Failures: append([]failure{}, failures...), Runs: []string{},
		Tests: []string{"TestTestpilotSample"}, Loop: "l", Host: hostLoad{LoadAverage: "1"},
	}
}

func TestSummarizeAddsRecordFiles(t *testing.T) {
	signature := Signature{Test: "TestTestpilotSample/hsm", Assertion: "verdict Inconclusive"}
	failed := failure{Test: signature.Test, SignatureHash: mustHash(t, signature), Signature: signature}
	directory := t.TempDir()
	first, second := filepath.Join(directory, "first.jsonl"), filepath.Join(directory, "second.jsonl")
	writeRecords(t, first, sampleRecord(1, "f", failed), sampleRecord(2, "f"))
	writeRecords(t, second, sampleRecord(1, "f", failed), sampleRecord(2, "f"))
	var stdout, stderr bytes.Buffer

	require.Equal(t, exitPassed, Run(context.Background(), []string{"summarize", first, second}, &stdout, &stderr, environment{}), stderr.String())

	require.Contains(t, stdout.String(), "summary 4 iterations")
	require.Contains(t, stdout.String(), "TestTestpilotSample 2/4 rate 50.00% (95% CI 6.76%-93.24%)")
	require.Contains(t, stdout.String(), mustHash(t, signature)+" 2/4 rate 50.00% (95% CI 6.76%-93.24%) TestTestpilotSample/hsm: verdict Inconclusive")
}

func TestSummarizeRefusesWhatCannotBeSummed(t *testing.T) {
	directory := t.TempDir()
	good := filepath.Join(directory, "good.jsonl")
	writeRecords(t, good, sampleRecord(1, "f"))
	other := filepath.Join(directory, "other.jsonl")
	writeRecords(t, other, sampleRecord(1, "f"), sampleRecord(2, "g"))
	malformed := filepath.Join(directory, "malformed.jsonl")
	encoded, err := json.Marshal(sampleRecord(1, "f"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(malformed, append(append(encoded, '\n'), []byte("{\"iteration\":\n")...), 0o644))

	for _, tc := range []struct {
		name   string
		files  []string
		stderr string
	}{
		{name: "mixed fingerprints", files: []string{good, other}, stderr: other + ":2 has fingerprint g but " + good + ":1 has f"},
		{name: "malformed line", files: []string{good, malformed}, stderr: malformed + ":2: malformed record"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(context.Background(), append([]string{"summarize"}, tc.files...), &stdout, &stderr, environment{})
			require.Equal(t, exitToolingError, code)
			require.Contains(t, stderr.String(), tc.stderr)
			require.Empty(t, stdout.String())
		})
	}
}

// In one process, a Run belongs to the iteration running when it was written, and one written by
// the iteration an interrupt cut short is dropped with it.
func TestRunAssignsInProcessRunsByWhenTheyWereWritten(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// The start times of the second and third repeats in testdata/in-process-3.jsonl.
	second := time.Date(2026, 9, 26, 18, 37, 55, 431026000, time.FixedZone("", -7*3600))
	third := time.Date(2026, 9, 26, 18, 37, 55, 431053000, time.FixedZone("", -7*3600))
	written := func(run invocation, name string, at time.Time) error {
		path := filepath.Join(run.RunDir, name)
		if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
			return err
		}
		return os.Chtimes(path, at, at)
	}
	world := newFakeWorld(t, fakeProcess{stream: "in-process-3", failed: true, during: func(run invocation) error {
		// The loop is interrupted while the third repeat runs.
		cancel()
		return errors.Join(
			written(run, "first.json", second.Add(-time.Microsecond)),
			written(run, "second.json", second),
			written(run, "third.json", third))
	}})
	path := filepath.Join(t.TempDir(), "record.jsonl")
	var stdout, stderr bytes.Buffer

	code := Run(ctx, runArguments(path, 3, modeInProcess), &stdout, &stderr, world.env)

	require.Equal(t, exitFailed, code, stderr.String())
	records := readRecordFile(t, path)
	require.Len(t, records, 2)
	require.Equal(t, []string{filepath.Join(world.invocations[0].RunDir, "first.json")}, records[0].Runs)
	require.Equal(t, []string{filepath.Join(world.invocations[0].RunDir, "second.json")}, records[1].Runs)
	require.Contains(t, stdout.String(), "interrupted after 2 iterations")
}
