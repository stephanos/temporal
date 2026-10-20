package sweep

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
)

// Class is the single outcome class of one test function.
type Class string

const (
	ClassPass                Class = "pass"
	ClassGuestFailure        Class = "guest-failure"
	ClassRefusedImport       Class = "refused-import"
	ClassCapacityExceeded    Class = "capacity-exceeded"
	ClassTimeout             Class = "timeout"
	ClassBuildFailure        Class = "build-failure"
	ClassInfrastructure      Class = "infrastructure"
	ClassReplayDivergence    Class = "replay-divergence"
	ClassNotRun              Class = "not-run"
	maximumReasonRunes             = 240
	exploreEventSchemaPrefix       = "gomad.explore-event/"
)

// Classes lists every valid class in report order.
var Classes = []Class{ClassPass, ClassGuestFailure, ClassRefusedImport, ClassCapacityExceeded, ClassTimeout, ClassBuildFailure, ClassInfrastructure, ClassReplayDivergence, ClassNotRun}

// Event is the subset of gomad.explore-event/v3 the classifier reads.
type Event struct {
	Schema            string `json:"schema"`
	Type              string `json:"type"`
	Classification    string `json:"classification"`
	Message           string `json:"message"`
	CampaignPath      string `json:"campaign_path"`
	Path              string `json:"path"`
	Attempted         uint64 `json:"attempted"`
	Succeeded         uint64 `json:"succeeded"`
	Failures          uint64 `json:"failures"`
	Watchdogs         uint64 `json:"watchdogs"`
	ReplayDivergences uint64 `json:"replay_divergences"`
}

// ArtifactFacts is what the classifier needs from one retained artifact.
type ArtifactFacts struct {
	Kind            string // record artifact kind (record.Artifact*)
	ModuleSHA256    string // prepared guest module identity
	Domain, Reason  string // record outcome domain and reason
	Termination     string
	ExitCode        *uint64
	Stdout, Stderr  []byte
	StdoutTruncated bool
	StoredBytes     uint64
	ReadError       string
}

// Observation is one explore subprocess and the artifacts it reported.
type Observation struct {
	Test      string
	Exit      int // gomad exit status; -1 when it did not exit on its own
	ExecError string
	Stdout    []byte // JSON event stream
	Stderr    []byte
	Artifacts map[string]ArtifactFacts
}

// Subtests summarizes `-test.v` verdict lines below the top-level function.
type Subtests struct {
	Pass   int      `json:"pass"`
	Fail   int      `json:"fail"`
	Skip   int      `json:"skip"`
	Failed []string `json:"failed,omitempty"` // first 20 failing names
}

// Outcome is the classification of one observation.
type Outcome struct {
	Class        Class     `json:"class"`
	Reason       string    `json:"reason,omitempty"`
	Skipped      bool      `json:"skipped,omitempty"`
	ArtifactPath string    `json:"-"`
	ModuleSHA256 string    `json:"-"`
	Subtests     *Subtests `json:"subtests,omitempty"`
}

// ParseEvents decodes the explore JSON event stream. Lines that are not
// explore events are returned as an error: --json stdout carries only events.
func ParseEvents(stdout []byte) ([]Event, error) {
	var events []Event
	scanner := bufio.NewScanner(bytes.NewReader(stdout))
	scanner.Buffer(make([]byte, 0, 1<<20), 16<<20)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var event Event
		if err := json.Unmarshal(line, &event); err != nil || !strings.HasPrefix(event.Schema, exploreEventSchemaPrefix) {
			return events, fmt.Errorf("explore stdout line is not an explore event: %.200s", line)
		}
		events = append(events, event)
	}
	return events, scanner.Err()
}

// Classify maps one explore observation to exactly one class and a
// normalized reason. Every non-pass outcome has a nonempty reason.
func Classify(observation Observation) Outcome {
	outcome := classify(observation)
	if outcome.Class != ClassPass && outcome.Reason == "" {
		outcome.Reason = "unclassified: no diagnostic"
	}
	return outcome
}

func classify(observation Observation) Outcome {
	if observation.ExecError != "" {
		return Outcome{Class: ClassInfrastructure, Reason: "gomad subprocess: " + Normalize(observation.ExecError)}
	}
	events, err := ParseEvents(observation.Stdout)
	if err != nil {
		return Outcome{Class: ClassInfrastructure, Reason: "explore output: " + Normalize(err.Error())}
	}
	var result, failure *Event
	var artifacts []Event
	for index := range events {
		switch event := &events[index]; event.Type {
		case "result":
			result = event
		case "error":
			if failure == nil {
				failure = event
			}
		case "artifact":
			artifacts = append(artifacts, *event)
		}
	}
	if failure != nil {
		return classifyError(*failure)
	}
	if result == nil {
		return Outcome{Class: ClassInfrastructure, Reason: fmt.Sprintf("gomad exit %d without a result event: %s", observation.Exit, firstLine(observation.Stderr))}
	}
	var facts ArtifactFacts
	var artifactPath string
	for _, event := range artifacts {
		if candidate, ok := observation.Artifacts[event.Path]; ok {
			facts, artifactPath = candidate, event.Path
			if event.Classification != "success" {
				break
			}
		}
	}
	outcome := classifyResult(observation.Test, *result, facts, artifactPath != "")
	outcome.ArtifactPath, outcome.ModuleSHA256 = artifactPath, facts.ModuleSHA256
	if artifactPath != "" && facts.ReadError == "" {
		outcome.Subtests = parseSubtests(observation.Test, facts.Stdout)
	}
	return outcome
}

var backendFailure = regexp.MustCompile(`backend (unsupported|capacity|trap|divergence|infrastructure|invalid): (.*)`)

func classifyError(event Event) Outcome {
	message := event.Message
	if match := backendFailure.FindStringSubmatch(message); match != nil {
		detail := Normalize(match[2])
		switch match[1] {
		case "unsupported":
			return Outcome{Class: ClassRefusedImport, Reason: "execute: " + detail}
		case "capacity":
			return Outcome{Class: ClassCapacityExceeded, Reason: detail}
		case "trap":
			return Outcome{Class: ClassGuestFailure, Reason: "trap: " + detail}
		case "divergence":
			return Outcome{Class: ClassReplayDivergence, Reason: detail}
		default:
			return Outcome{Class: ClassInfrastructure, Reason: "backend " + match[1] + ": " + detail}
		}
	}
	switch event.Classification {
	case "overall_timeout":
		return Outcome{Class: ClassTimeout, Reason: "overall-timeout: preparation plus execution exceeded explore --overall-timeout"}
	case "capacity":
		return Outcome{Class: ClassCapacityExceeded, Reason: "artifact or journal capacity: " + Normalize(message)}
	case "unsupported_target":
		return Outcome{Class: ClassRefusedImport, Reason: "prepare: " + Normalize(message)}
	}
	if strings.Contains(message, "choice_trace_overflow") {
		return Outcome{Class: ClassCapacityExceeded, Reason: "choice trace overflow"}
	}
	if index := strings.Index(message, "WASM compiler command failed:"); index >= 0 {
		if reason := compileReason(message[index+len("WASM compiler command failed:"):]); reason != "" {
			return Outcome{Class: ClassBuildFailure, Reason: reason}
		}
		// The provider also reports its own 10-minute compiler kill this way,
		// usually without a diagnostic; that is not a compile error.
		return Outcome{Class: ClassInfrastructure, Reason: "compiler command failed without a diagnostic (provider compiler limit or host failure)"}
	}
	return Outcome{Class: ClassInfrastructure, Reason: event.Classification + ": " + Normalize(firstLine([]byte(message)))}
}

var compileLocation = regexp.MustCompile(`^(?:\S*/)?([^/\s]+\.(?:go|s)):\d+(?::\d+)?: (.*)$`)

// compileReason picks the first compiler diagnostic, keeping only the file's
// base name so the same error in every function of a package groups.
func compileReason(output string) string {
	for line := range strings.SplitSeq(output, "\n") {
		line = strings.TrimSpace(line)
		if match := compileLocation.FindStringSubmatch(line); match != nil {
			return Normalize(match[1] + ": " + match[2])
		}
	}
	for line := range strings.SplitSeq(output, "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			return Normalize(line)
		}
	}
	return ""
}

func classifyResult(test string, result Event, facts ArtifactFacts, haveArtifact bool) Outcome {
	switch result.Classification {
	case "success":
		if !haveArtifact {
			return Outcome{Class: ClassInfrastructure, Reason: "success without a retained success artifact; verdict unverifiable"}
		}
		if facts.ReadError != "" {
			return Outcome{Class: ClassInfrastructure, Reason: "open success artifact: " + Normalize(facts.ReadError)}
		}
		switch {
		case verdictLine(facts.Stdout, "PASS", test):
			return Outcome{Class: ClassPass}
		case verdictLine(facts.Stdout, "SKIP", test):
			return Outcome{Class: ClassPass, Skipped: true}
		case bytes.Contains(facts.Stdout, []byte("testing: warning: no tests to run")):
			return Outcome{Class: ClassBuildFailure, Reason: "guest-file-set: test not linked in the wasip1 guest (no tests to run)"}
		case facts.StdoutTruncated:
			return Outcome{Class: ClassInfrastructure, Reason: "verdict-unverifiable: stdout truncated before the test verdict"}
		default:
			return Outcome{Class: ClassInfrastructure, Reason: "verdict-unverifiable: exit 0 without a PASS or SKIP line for the test"}
		}
	case "watchdog_observation":
		return Outcome{Class: ClassTimeout, Reason: "execution-timeout: watchdog stopped the guest"}
	case "replay_divergence":
		return Outcome{Class: ClassReplayDivergence, Reason: "replay of the observed execution diverged" + artifactSuffix(facts)}
	case "target_failure", "mixed_failure":
		if !haveArtifact || facts.ReadError != "" {
			return Outcome{Class: ClassGuestFailure, Reason: "target failure; artifact unreadable: " + Normalize(facts.ReadError)}
		}
		switch facts.Reason {
		case "world_capacity":
			return Outcome{Class: ClassCapacityExceeded, Reason: "world capacity"}
		case "world_replay_divergence":
			return Outcome{Class: ClassReplayDivergence, Reason: "world replay divergence"}
		}
		if facts.Kind == record.ArtifactWatchdogTimeout || facts.Termination == "timeout" {
			return Outcome{Class: ClassTimeout, Reason: "execution-timeout: watchdog stopped the guest"}
		}
		return Outcome{Class: ClassGuestFailure, Reason: guestFailureReason(test, facts)}
	default:
		return Outcome{Class: ClassInfrastructure, Reason: "unknown explore classification " + Normalize(result.Classification)}
	}
}

func artifactSuffix(facts ArtifactFacts) string {
	if facts.Reason == "" {
		return ""
	}
	return ": " + facts.Reason
}

func verdictLine(stdout []byte, verdict, test string) bool {
	prefix := []byte("--- " + verdict + ": " + test + " (")
	for line := range bytes.SplitSeq(stdout, []byte("\n")) {
		if bytes.HasPrefix(line, prefix) {
			return true
		}
	}
	return false
}

var testifyError = regexp.MustCompile(`^\s*Error:\s+(.*)$`)

// guestFailureReason combines the recorded outcome reason with the first
// decisive guest diagnostic: a panic or fatal error, the first testify error,
// or the first failing test line.
func guestFailureReason(test string, facts ArtifactFacts) string {
	reason := facts.Reason
	if reason == "" {
		reason = "nonzero_exit"
	}
	if facts.ExitCode != nil && reason == "nonzero_exit" {
		reason = fmt.Sprintf("exit %d", *facts.ExitCode)
	}
	for _, stream := range [][]byte{facts.Stderr, facts.Stdout} {
		for line := range strings.SplitSeq(string(stream), "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "panic: ") || strings.HasPrefix(trimmed, "fatal error: ") {
				return Normalize(reason + ": " + trimmed)
			}
		}
	}
	for line := range strings.SplitSeq(string(facts.Stdout), "\n") {
		if match := testifyError.FindStringSubmatch(line); match != nil && strings.TrimSpace(match[1]) != "" {
			return Normalize(reason + ": " + strings.TrimSpace(match[1]))
		}
	}
	for line := range strings.SplitSeq(string(facts.Stdout), "\n") {
		if trimmed := strings.TrimSpace(line); strings.HasPrefix(trimmed, "--- FAIL: ") && !strings.HasPrefix(trimmed, "--- FAIL: "+test+" (") {
			return Normalize(reason + ": " + trimmed)
		}
	}
	if line := firstLine(facts.Stderr); line != "" {
		return Normalize(reason + ": " + line)
	}
	return Normalize(reason)
}

var subtestVerdict = regexp.MustCompile(`^\s+--- (PASS|FAIL|SKIP): (\S+)`)

func parseSubtests(test string, stdout []byte) *Subtests {
	summary := &Subtests{}
	for line := range strings.SplitSeq(string(stdout), "\n") {
		match := subtestVerdict.FindStringSubmatch(line)
		if match == nil || !strings.HasPrefix(match[2], test+"/") {
			continue
		}
		switch match[1] {
		case "PASS":
			summary.Pass++
		case "SKIP":
			summary.Skip++
		case "FAIL":
			summary.Fail++
			if len(summary.Failed) < 20 {
				summary.Failed = append(summary.Failed, match[2])
			}
		}
	}
	if summary.Pass+summary.Fail+summary.Skip == 0 {
		return nil
	}
	return summary
}

func firstLine(data []byte) string {
	for line := range strings.SplitSeq(string(data), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

var (
	normalizeHexAddress = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	normalizeDigest     = regexp.MustCompile(`(?:sha256:)?\b[0-9a-f]{16,}\b`)
	normalizeUUID       = regexp.MustCompile(`\b[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}\b`)
	normalizePath       = regexp.MustCompile(`(?:^|[\s"'=:(\[])(?:/[A-Za-z0-9._@+~-]+)+/?`)
	normalizeNumber     = regexp.MustCompile(`\b\d+(?:\.\d+)?(?:ns|µs|us|ms|s|m|h|B|KiB|MiB|GiB)?\b`)
	normalizeSpace      = regexp.MustCompile(`\s+`)
)

// Normalize makes a diagnostic comparable across tests and runs: addresses,
// digests, UUIDs and numbers are replaced, absolute paths are cut to their
// base name, whitespace is collapsed and the result is bounded.
func Normalize(message string) string {
	message = normalizeUUID.ReplaceAllString(message, "<uuid>")
	message = normalizeHexAddress.ReplaceAllString(message, "0x?")
	message = normalizeDigest.ReplaceAllStringFunc(message, func(value string) string {
		if strings.IndexAny(strings.TrimPrefix(value, "sha256:"), "abcdef") < 0 && !strings.HasPrefix(value, "sha256:") {
			return value // a long decimal number; normalized below
		}
		return "<hex>"
	})
	message = normalizePath.ReplaceAllStringFunc(message, func(value string) string {
		lead := ""
		if !strings.HasPrefix(value, "/") {
			lead, value = value[:1], value[1:]
		}
		return lead + "<path>/" + path.Base(strings.TrimSuffix(value, "/"))
	})
	message = normalizeNumber.ReplaceAllString(message, "N")
	message = strings.TrimSpace(normalizeSpace.ReplaceAllString(message, " "))
	if utf8.RuneCountInString(message) > maximumReasonRunes {
		runes := []rune(message)
		message = string(runes[:maximumReasonRunes]) + "…"
	}
	return message
}
