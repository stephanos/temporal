package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/internal/cli"
)

const (
	outcomePass = "PASS"
	outcomeFail = "FAIL"
)

// record is one finished iteration, one JSON line in the record file.
type record struct {
	Iteration int    `json:"iteration"`
	Mode      string `json:"mode"`
	Selection string `json:"selection"`
	Commit    string `json:"commit"`
	// Dirty says the tree the live tests read had uncommitted edits when the loop started.
	Dirty       bool      `json:"dirty"`
	Fingerprint string    `json:"fingerprint"`
	Outcome     string    `json:"outcome"`
	Failures    []failure `json:"failures"`
	Runs        []string  `json:"runs"`
	// Tests are the selected top-level tests that started in this iteration.
	Tests []string `json:"tests"`
	// Loop names the loop the iteration belongs to by its start time; Host is the load it started
	// under.
	Loop string   `json:"loop"`
	Host hostLoad `json:"host"`
}

type failure struct {
	Test          string    `json:"test"`
	SignatureHash string    `json:"signature_hash"`
	Signature     Signature `json:"signature"`
}

type hostLoad struct {
	LoadAverage     string `json:"load_average"`
	GoTestProcesses int    `json:"go_test_processes"`
}

// sourcedRecord is a record with the file and line it was read from, so a refusal can name them.
type sourcedRecord struct {
	record
	path string
	line int
}

// readRecords reads a record file. A line that does not decode, or lacks what every record has,
// fails naming the file and line.
func readRecords(path string) ([]sourcedRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()
	var records []sourcedRecord
	reader := bufio.NewReader(file)
	for number := 1; ; number++ {
		line, err := reader.ReadBytes('\n')
		if len(bytes.TrimSpace(line)) > 0 {
			decoded, decodeErr := decodeRecord(line)
			if decodeErr != nil {
				return nil, fmt.Errorf("%s:%d: malformed record: %w", path, number, decodeErr)
			}
			records = append(records, sourcedRecord{record: decoded, path: path, line: number})
		}
		if errors.Is(err, io.EOF) {
			return records, nil
		}
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, number, err)
		}
	}
}

func decodeRecord(line []byte) (record, error) {
	decoder := json.NewDecoder(bytes.NewReader(line))
	decoder.DisallowUnknownFields()
	var decoded record
	if err := decoder.Decode(&decoded); err != nil {
		return record{}, err
	}
	switch {
	case decoded.Iteration < 1:
		return record{}, errors.New("iteration must be positive")
	case decoded.Fingerprint == "":
		return record{}, errors.New("fingerprint is missing")
	case decoded.Outcome != outcomePass && decoded.Outcome != outcomeFail:
		return record{}, fmt.Errorf("outcome %q is neither %s nor %s", decoded.Outcome, outcomePass, outcomeFail)
	case decoded.Outcome == outcomeFail && len(decoded.Failures) == 0:
		return record{}, errors.New("a FAIL record names no failure")
	}
	return decoded, nil
}

// sameFingerprint refuses records measured on different inputs: their counts never add up to one.
func sameFingerprint(records []sourcedRecord) error {
	for _, candidate := range records[min(1, len(records)):] {
		if first := records[0]; candidate.Fingerprint != first.Fingerprint {
			return fmt.Errorf("%s:%d has fingerprint %s but %s:%d has %s; records of different inputs are never summed",
				candidate.path, candidate.line, candidate.Fingerprint, first.path, first.line, first.Fingerprint)
		}
	}
	return nil
}

type loopKey struct {
	loop, commit string
	dirty        bool
	host         hostLoad
}

// describe is the line that says what a loop measured and under what load.
func (k loopKey) describe() string {
	dirty := ""
	if k.dirty {
		dirty = " dirty"
	}
	return fmt.Sprintf("loop %s commit %s%s load %s other-go-test-processes %d",
		k.loop, k.commit, dirty, k.host.LoadAverage, k.host.GoTestProcesses)
}

type signatureCount struct {
	count     int
	signature Signature
}

// tally is what a summary counts: iterations per test identity, failed iterations per identity,
// and iterations per signature, each in first-seen order.
type tally struct {
	loops      []loopKey
	tests      []string
	ran        map[string]int
	failed     map[string]int
	hashes     []string
	signatures map[string]*signatureCount
}

func count(records []record) tally {
	counted := tally{ran: map[string]int{}, failed: map[string]int{}, signatures: map[string]*signatureCount{}}
	for _, rec := range records {
		key := loopKey{loop: rec.Loop, commit: rec.Commit, dirty: rec.Dirty, host: rec.Host}
		if !slices.Contains(counted.loops, key) {
			counted.loops = append(counted.loops, key)
		}
		for _, test := range rec.Tests {
			if counted.ran[test] == 0 {
				counted.tests = append(counted.tests, test)
			}
			counted.ran[test]++
		}
		failedTests := map[string]bool{}
		for _, f := range rec.Failures {
			if f.Test != "" {
				failedTests[topLevel(f.Test)] = true
			}
		}
		for test := range failedTests {
			counted.failed[test]++
		}
		counted.addSignatures(rec.Failures)
	}
	return counted
}

// addSignatures counts each signature once per iteration.
func (t *tally) addSignatures(failures []failure) {
	seen := map[string]bool{}
	for _, f := range failures {
		if seen[f.SignatureHash] {
			continue
		}
		seen[f.SignatureHash] = true
		if t.signatures[f.SignatureHash] == nil {
			t.signatures[f.SignatureHash] = &signatureCount{signature: f.Signature}
			t.hashes = append(t.hashes, f.SignatureHash)
		}
		t.signatures[f.SignatureHash].count++
	}
}

// printSummary prints one line per test identity and one per distinct signature, each rate with
// its 95% Clopper-Pearson interval.
func printSummary(stdout io.Writer, records []record) {
	cli.WriteLine(stdout, "summary %d iterations", len(records))
	counted := count(records)
	for _, loop := range counted.loops {
		cli.WriteLine(stdout, "%s", loop.describe())
	}
	for _, test := range counted.tests {
		cli.WriteLine(stdout, "%s %d/%d %s", test, counted.failed[test], counted.ran[test], describeRate(counted.failed[test], counted.ran[test]))
	}
	for _, hash := range counted.hashes {
		entry := counted.signatures[hash]
		// A signature is counted against the iterations its test ran in; one with no test, the
		// process's own, against every iteration.
		denominator := len(records)
		if test := topLevel(entry.signature.Test); counted.ran[test] > 0 {
			denominator = counted.ran[test]
		}
		assertion := entry.signature.Assertion
		if entry.signature.Reserved != "" {
			assertion = entry.signature.Reserved + ": " + assertion
		}
		first, _, _ := strings.Cut(assertion, "\n")
		cli.WriteLine(stdout, "%s %d/%d %s %s: %s", hash, entry.count, denominator,
			describeRate(entry.count, denominator), entry.signature.Test, first)
	}
}

func describeRate(failures, trials int) string {
	if trials == 0 {
		return "rate n/a"
	}
	lower, upper := clopperPearson(failures, trials)
	text := fmt.Sprintf("rate %s (95%% CI %s-%s", percent(float64(failures)/float64(trials)), percent(lower), percent(upper))
	if failures == 0 {
		text += fmt.Sprintf(", rule of three <= %s", percent(3/float64(trials)))
	}
	return text + ")"
}

func percent(value float64) string {
	return fmt.Sprintf("%.2f%%", 100*value)
}

// clopperPearson is the exact 95% interval for failures out of trials, found by bisection on the
// binomial distribution function.
func clopperPearson(failures, trials int) (lower, upper float64) {
	const alpha = 0.05
	lower, upper = 0.0, 1.0
	if failures > 0 {
		// The lower bound is the rate at which seeing this many failures or more has probability alpha/2.
		lower = bisect(func(p float64) bool { return 1-binomialCDF(failures-1, trials, p) < alpha/2 })
	}
	if failures < trials {
		// The upper bound is the rate at which seeing this many failures or fewer has probability alpha/2.
		upper = bisect(func(p float64) bool { return binomialCDF(failures, trials, p) > alpha/2 })
	}
	return lower, upper
}

// bisect finds the boundary in [0,1] below which below holds.
func bisect(below func(float64) bool) float64 {
	low, high := 0.0, 1.0
	for range 100 {
		middle := (low + high) / 2
		if below(middle) {
			low = middle
		} else {
			high = middle
		}
	}
	return (low + high) / 2
}

func binomialCDF(k, n int, p float64) float64 {
	if k < 0 {
		return 0
	}
	if k >= n {
		return 1
	}
	if p <= 0 {
		return 1
	}
	if p >= 1 {
		return 0
	}
	logN, _ := math.Lgamma(float64(n + 1))
	sum := 0.0
	for i := 0; i <= k; i++ {
		logI, _ := math.Lgamma(float64(i + 1))
		logRest, _ := math.Lgamma(float64(n - i + 1))
		sum += math.Exp(logN - logI - logRest + float64(i)*math.Log(p) + float64(n-i)*math.Log1p(-p))
	}
	return math.Min(sum, 1)
}
