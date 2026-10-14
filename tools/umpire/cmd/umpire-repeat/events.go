package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"
)

// event is one test2json event (`go doc cmd/test2json`).
type event struct {
	Time        time.Time
	Action      string
	Package     string
	Test        string
	Elapsed     float64
	Output      string
	OutputType  string
	FailedBuild string
}

// readEvents decodes one event per line until the stream ends. A line that is not an event, such
// as text from an early fatal error, becomes package output: it is never dropped.
func readEvents(stream io.Reader, handle func(event)) error {
	reader := bufio.NewReader(stream)
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			var decoded event
			if decodeErr := json.Unmarshal([]byte(line), &decoded); decodeErr != nil || decoded.Action == "" {
				decoded = event{Action: "output", Output: line}
			}
			handle(decoded)
		}
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// testState is what one test did within one iteration.
type testState struct {
	name     string
	started  bool
	finished bool
	failed   bool
	output   []outputLine
}

// iteration is one repeat of the selection. In process mode a process holds one; in in-process mode
// a process holds one per repeat, told apart by the order of each top-level test's run events,
// because repeats share a test name.
type iteration struct {
	number  int
	started time.Time
	order   []string
	tests   map[string]*testState
	// packageOutput is what the process printed outside any test while this iteration was current.
	packageOutput []outputLine
}

// collector splits one process's event stream into iterations.
type collector struct {
	iterations []*iteration
	runs       map[string]int
	// preamble is package output before any test started.
	preamble      []outputLine
	packageFailed bool
	failedBuild   string
}

func newCollector() *collector {
	return &collector{runs: map[string]int{}}
}

func topLevel(test string) string {
	name, _, _ := strings.Cut(test, "/")
	return name
}

func (c *collector) add(e event) {
	if e.Test == "" {
		switch e.Action {
		case "output":
			lines := splitOutput(e)
			if current := c.current(); current != nil {
				current.packageOutput = append(current.packageOutput, lines...)
			} else {
				c.preamble = append(c.preamble, lines...)
			}
		case "fail":
			c.packageFailed = true
			if e.FailedBuild != "" {
				c.failedBuild = e.FailedBuild
			}
		default:
			// start, pass and the rest say nothing a package-level failure needs.
		}
		return
	}
	top := topLevel(e.Test)
	if e.Action == "run" && e.Test == top {
		c.runs[top]++
	}
	number := max(c.runs[top], 1)
	for len(c.iterations) < number {
		c.iterations = append(c.iterations, &iteration{number: len(c.iterations) + 1, tests: map[string]*testState{}})
	}
	current := c.iterations[number-1]
	if e.Action == "run" && e.Test == top && current.started.IsZero() {
		current.started = e.Time
	}
	state, ok := current.tests[e.Test]
	if !ok {
		state = &testState{name: e.Test}
		current.tests[e.Test] = state
		current.order = append(current.order, e.Test)
	}
	switch e.Action {
	case "run":
		state.started = true
	case "pass", "skip":
		state.finished = true
	case "fail":
		state.finished = true
		state.failed = true
	case "output":
		state.output = append(state.output, splitOutput(e)...)
	default:
		// pause, cont and bench change nothing the loop counts.
	}
}

func (c *collector) current() *iteration {
	if len(c.iterations) == 0 {
		return nil
	}
	return c.iterations[len(c.iterations)-1]
}

func splitOutput(e event) []outputLine {
	var lines []outputLine
	for text := range strings.SplitSeq(strings.TrimSuffix(e.Output, "\n"), "\n") {
		lines = append(lines, outputLine{text: text, kind: e.OutputType})
	}
	return lines
}

// topLevelTests are the selected tests that started in this iteration, in start order.
func (it *iteration) topLevelTests() []string {
	var tests []string
	for _, name := range it.order {
		if state := it.tests[name]; name == topLevel(name) && state.started {
			tests = append(tests, name)
		}
	}
	return tests
}

// failures are the iteration's signatures: one per failing leaf test, and one per started leaf
// test that never finished.
func (it *iteration) failures() []Signature {
	var signatures []Signature
	for _, name := range it.order {
		state := it.tests[name]
		switch {
		case state.failed && !it.hasDescendant(name, func(s *testState) bool { return s.failed }):
			signatures = append(signatures, signatureOf(name, state.output))
		case state.started && !state.finished && !it.hasDescendant(name, func(s *testState) bool { return s.started && !s.finished }):
			signatures = append(signatures, processSignature(name, state.output, it.packageOutput))
		default:
		}
	}
	return signatures
}

// processFailure is the signature of a process that failed with no failing or unfinished test to
// blame: a timeout panic if it printed one, else unparsed with what it printed last.
func (it *iteration) processFailure() Signature {
	if timeout, ok := timeoutLine(it.packageOutput); ok {
		return Signature{Reserved: reservedProcess, Assertion: timeout}
	}
	return Signature{Reserved: reservedUnparsed, Detail: tail(it.packageOutput, 5)}
}

func (it *iteration) hasDescendant(name string, matches func(*testState) bool) bool {
	prefix := name + "/"
	for other, state := range it.tests {
		if strings.HasPrefix(other, prefix) && matches(state) {
			return true
		}
	}
	return false
}
