package determinism

import (
	"bytes"
	"cmp"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/backends"
	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

// ReportSchema identifies a first-divergence report.
const ReportSchema = "gomad.wasm-first-divergence/v1"

// EventKind names the evidence stream in which two executions first differ.
type EventKind string

const (
	KindChoice          EventKind = "choice"
	KindDrawCounter     EventKind = "draw-counter"
	KindClock           EventKind = "clock"
	KindEntropy         EventKind = "entropy"
	KindWASIOp          EventKind = "wasi-op"
	KindModelTransition EventKind = "model-transition"
	KindOutput          EventKind = "output"
	KindWorldOrder      EventKind = "world-order"
)

// Run is the decoded evidence of one fresh execution. The caller decodes the
// retained evidence (choice tape, diagnostic trace, WASI transcript with
// folded clock runs, model transitions, outputs and the shared World order)
// into these records; the formats themselves are unchanged.
type Run struct {
	Workload     string
	Seed         uint64
	Perturbation string
	// Identities binds the execution: source, helper, module, engine, model.
	Identities     map[string]string
	EvidenceDigest string
	// Command reruns this execution exactly; ArtifactDir is its retained
	// artifact, the input of gomad replay.
	Command     []string
	ArtifactDir string
	// Incarnations lists every executed incarnation; never-started guests
	// contribute nothing.
	Incarnations []Incarnation
	// WorldOrder is the shared execution order of cluster steps. A single
	// guest has none.
	WorldOrder []WorldStep
}

// Coordinate places an event in helper execution order. Shared is the fn-180
// shared execution-order coordinate (the World step the event ran in; zero
// for a single guest), ChoiceOrdinal is the number of choice records the
// incarnation committed before the event (the tape ordinal), and VirtualTime
// is the incarnation's virtual clock at the event.
type Coordinate struct {
	Shared        uint64
	ChoiceOrdinal uint64
	VirtualTime   int64
}

type Incarnation struct {
	Instance    string
	Incarnation uint64
	// Choices are the decoded choice tape records, each with its coordinate.
	Choices []ChoiceEvent
	// Diagnostic is the gomad-diagnostic-trace/v1 bytes, nil when the run
	// did not record diagnostics.
	Diagnostic  []byte
	Operations  []Operation
	Transitions []Transition
	Stdout      []byte
	Stderr      []byte
}

type ChoiceEvent struct {
	At     Coordinate
	Record choice.Record
}

// Operation is one WASI transcript record. Repeat counts a folded clock run.
type Operation struct {
	At     Coordinate
	Op     string
	Input  string
	Output string
	Repeat uint64
}

// Transition is one model transition digest (world, network, volume or
// namespace).
type Transition struct {
	At     Coordinate
	Model  string
	Digest string
}

// WorldStep is one step of the shared World order: the instance that held
// the baton and its tape ordinal when it received it.
type WorldStep struct {
	Instance      string
	Incarnation   uint64
	ChoiceOrdinal uint64
	VirtualTime   int64
	Digest        string
}

// Divergence is the earliest differing event.
type Divergence struct {
	Kind        EventKind  `json:"kind"`
	Instance    string     `json:"instance,omitempty"`
	Incarnation uint64     `json:"incarnation,omitempty"`
	Ordinal     uint64     `json:"ordinal"`
	At          Coordinate `json:"coordinate"`
	Field       string     `json:"field"`
	Expected    string     `json:"expected"`
	Actual      string     `json:"actual"`
	// source orders events that share a World step and tape ordinal.
	source source
}

// source is the evidence stream a divergence was found in, in the order the
// helper produces them around one choice point: the handoff itself, then the
// operations between choice points, then the diagnostic record and the
// decision at the choice point, then output.
type source int

const (
	sourceWorld source = iota
	sourceClock
	sourceEntropy
	sourceOperation
	sourceTransition
	sourceDiagnostic
	sourceChoice
	sourceOutput
)

// Reproducer carries what reruns both executions and stops a replay at the
// last common prefix.
type Reproducer struct {
	Workload        string                     `json:"workload"`
	Seed            uint64                     `json:"seed"`
	Perturbation    string                     `json:"perturbation"`
	ExpectedDigest  string                     `json:"expected_digest"`
	ActualDigest    string                     `json:"actual_digest"`
	Identities      map[string]string          `json:"identities"`
	ExpectedCommand []string                   `json:"expected_command"`
	ActualCommand   []string                   `json:"actual_command"`
	Stop            backends.InspectionRequest `json:"stop"`
	ReplayCommand   []string                   `json:"replay_command"`
}

type Report struct {
	Schema string `json:"schema"`
	// Status is identical, diverged or undiagnosable.
	Status     string      `json:"status"`
	First      *Divergence `json:"first,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	Reproducer *Reproducer `json:"reproducer,omitempty"`
}

// DiagnosticsFlag enables the diagnostic trace on explore and qualify runs.
const DiagnosticsFlag = "--diagnostics"

// ErrIncomparable refuses a comparison of runs that are not the same
// workload, seed and identities.
var ErrIncomparable = errors.New("runs are not comparable")

// Compare reports the earliest event in helper execution order at which
// actual differs from expected. Runs of different workloads, seeds or
// identities are refused. Without a diagnostic trace on every executed
// incarnation the report is undiagnosable and names the flag that records one.
func Compare(expected, actual Run) (Report, error) {
	if expected.Workload != actual.Workload || expected.Seed != actual.Seed {
		return Report{}, fmt.Errorf("%w: workload/seed %s/%d against %s/%d", ErrIncomparable, expected.Workload, expected.Seed, actual.Workload, actual.Seed)
	}
	if len(expected.Identities) == 0 || !maps.Equal(expected.Identities, actual.Identities) {
		return Report{}, fmt.Errorf("%w: identities differ", ErrIncomparable)
	}
	for _, run := range []Run{expected, actual} {
		if len(run.Command) == 0 {
			return Report{}, errors.New("run has no rerun command")
		}
		for _, incarnation := range run.Incarnations {
			if incarnation.Diagnostic == nil {
				return Report{Schema: ReportSchema, Status: "undiagnosable", Reason: fmt.Sprintf("incarnation %s/%d has no diagnostic trace; rerun with %s", incarnation.Instance, incarnation.Incarnation, DiagnosticsFlag)}, nil
			}
		}
	}
	candidates, err := divergences(expected, actual)
	if err != nil {
		return Report{}, err
	}
	if len(candidates) == 0 {
		return Report{Schema: ReportSchema, Status: "identical"}, nil
	}
	first := slices.MinFunc(candidates, compareDivergence)
	return Report{Schema: ReportSchema, Status: "diverged", First: &first, Reproducer: reproducer(expected, actual, first)}, nil
}

func compareDivergence(a, b Divergence) int {
	return cmp.Or(cmp.Compare(a.At.Shared, b.At.Shared), cmp.Compare(a.At.ChoiceOrdinal, b.At.ChoiceOrdinal), cmp.Compare(a.source, b.source),
		cmp.Compare(a.Instance, b.Instance), cmp.Compare(a.Incarnation, b.Incarnation), cmp.Compare(a.Ordinal, b.Ordinal))
}

func divergences(expected, actual Run) ([]Divergence, error) {
	var found []Divergence
	if step, ok := firstDifference(expected.WorldOrder, actual.WorldOrder, func(a, b WorldStep) bool { return a == b }); ok {
		e, a := at(expected.WorldOrder, step), at(actual.WorldOrder, step)
		ref := cmp.Or(e, a)
		found = append(found, Divergence{Kind: KindWorldOrder, Instance: ref.Instance, Incarnation: ref.Incarnation, Ordinal: uint64(step),
			At: Coordinate{Shared: uint64(step), ChoiceOrdinal: ref.ChoiceOrdinal, VirtualTime: ref.VirtualTime}, Field: "step", Expected: describeStep(e), Actual: describeStep(a)})
	}
	type key struct {
		instance    string
		incarnation uint64
	}
	index := func(run Run) (map[key]Incarnation, error) {
		incarnations := map[key]Incarnation{}
		for _, incarnation := range run.Incarnations {
			k := key{incarnation.Instance, incarnation.Incarnation}
			if _, duplicate := incarnations[k]; duplicate {
				return nil, fmt.Errorf("incarnation %s/%d listed twice", k.instance, k.incarnation)
			}
			incarnations[k] = incarnation
		}
		return incarnations, nil
	}
	left, err := index(expected)
	if err != nil {
		return nil, err
	}
	right, err := index(actual)
	if err != nil {
		return nil, err
	}
	keys := slices.Collect(maps.Keys(left))
	for k := range right {
		if _, ok := left[k]; !ok {
			keys = append(keys, k)
		}
	}
	for _, k := range keys {
		e, eok := left[k]
		a, aok := right[k]
		if !eok || !aok {
			present := e
			if !eok {
				present = a
			}
			first := Coordinate{}
			if len(present.Choices) != 0 {
				first = present.Choices[0].At
			}
			found = append(found, Divergence{Kind: KindWorldOrder, Instance: k.instance, Incarnation: k.incarnation, At: first, Field: "incarnation_presence", Expected: strconv.FormatBool(eok), Actual: strconv.FormatBool(aok)})
			continue
		}
		more, err := incarnationDivergences(e, a)
		if err != nil {
			return nil, fmt.Errorf("incarnation %s/%d: %w", k.instance, k.incarnation, err)
		}
		found = append(found, more...)
	}
	return found, nil
}

func incarnationDivergences(expected, actual Incarnation) ([]Divergence, error) {
	base := Divergence{Instance: expected.Instance, Incarnation: expected.Incarnation}
	var found []Divergence
	add := func(source source, kind EventKind, ordinal uint64, at Coordinate, field, e, a string) {
		divergence := base
		divergence.source, divergence.Kind, divergence.Ordinal, divergence.At, divergence.Field, divergence.Expected, divergence.Actual = source, kind, ordinal, at, field, e, a
		found = append(found, divergence)
	}
	// Choice tape.
	if i, ok := firstDifference(expected.Choices, actual.Choices, func(a, b ChoiceEvent) bool { return a.Record == b.Record }); ok {
		e, a := at(expected.Choices, i), at(actual.Choices, i)
		ref := e
		if i >= len(expected.Choices) {
			ref = a
		}
		add(sourceChoice, KindChoice, uint64(i), Coordinate{ref.At.Shared, uint64(i), ref.At.VirtualTime}, "record", describeChoice(e, i < len(expected.Choices)), describeChoice(a, i < len(actual.Choices)))
	}
	// Diagnostic counters at every choice point, through DiffDiagnostics.
	difference, err := choice.DiffDiagnostics(expected.Diagnostic, actual.Diagnostic)
	if err != nil {
		return nil, err
	}
	if difference != nil {
		kind := KindDrawCounter
		if len(difference.Fields) == 1 && difference.Fields[0] == "virtual_time" {
			kind = KindClock
		}
		ref := difference.Expected
		if ref == nil {
			ref = difference.Actual
		}
		shared := uint64(0)
		if ordinal := int(difference.Ordinal); ordinal < len(expected.Choices) {
			shared = expected.Choices[ordinal].At.Shared
		}
		e, a := describeDiagnostic(difference.Expected), describeDiagnostic(difference.Actual)
		add(sourceDiagnostic, kind, difference.Ordinal, Coordinate{shared, difference.Ordinal, ref.VirtualTime}, strings.Join(difference.Fields, ","), e, a)
	}
	// WASI transcript, with clock reads and entropy separated.
	if i, ok := firstDifference(expected.Operations, actual.Operations, func(a, b Operation) bool { return a == b }); ok {
		e, a := at(expected.Operations, i), at(actual.Operations, i)
		ref := e
		if i >= len(expected.Operations) {
			ref = a
		}
		kind, from := KindWASIOp, sourceOperation
		switch op := cmp.Or(e.Op, a.Op); op {
		case "clock_time_get", "poll_oneoff":
			kind, from = KindClock, sourceClock
		case "random_get":
			kind, from = KindEntropy, sourceEntropy
		}
		add(from, kind, uint64(i), ref.At, "operation", describeOperation(e), describeOperation(a))
	}
	// Model transition digests.
	if i, ok := firstDifference(expected.Transitions, actual.Transitions, func(a, b Transition) bool { return a == b }); ok {
		e, a := at(expected.Transitions, i), at(actual.Transitions, i)
		ref := e
		if i >= len(expected.Transitions) {
			ref = a
		}
		add(sourceTransition, KindModelTransition, uint64(i), ref.At, cmp.Or(e.Model, a.Model), e.Model+":"+e.Digest, a.Model+":"+a.Digest)
	}
	// Outputs: an output difference without an earlier operation difference
	// sorts after every recorded event of the incarnation.
	end := Coordinate{}
	if count := len(expected.Choices); count != 0 {
		last := expected.Choices[count-1].At
		end = Coordinate{last.Shared, uint64(count), last.VirtualTime}
	}
	for _, stream := range []struct {
		name string
		e, a []byte
	}{{"stdout", expected.Stdout, actual.Stdout}, {"stderr", expected.Stderr, actual.Stderr}} {
		if offset, ok := firstDifference(stream.e, stream.a, func(a, b byte) bool { return a == b }); ok {
			add(sourceOutput, KindOutput, uint64(offset), end, stream.name, window(stream.e, offset), window(stream.a, offset))
		}
	}
	return found, nil
}

// firstDifference returns the first index at which the sequences differ,
// including a length difference.
func firstDifference[T any](expected, actual []T, equal func(T, T) bool) (int, bool) {
	for index := range min(len(expected), len(actual)) {
		if !equal(expected[index], actual[index]) {
			return index, true
		}
	}
	if len(expected) != len(actual) {
		return min(len(expected), len(actual)), true
	}
	return 0, false
}

func at[T any](values []T, index int) T {
	var zero T
	if index < len(values) {
		return values[index]
	}
	return zero
}

func describeStep(step WorldStep) string {
	if step.Instance == "" {
		return "absent"
	}
	return fmt.Sprintf("%s/%d@%d", step.Instance, step.Incarnation, step.ChoiceOrdinal)
}

func describeChoice(event ChoiceEvent, present bool) string {
	if !present {
		return "absent"
	}
	r := event.Record
	return fmt.Sprintf("kind=%d selected=%d/%d data=%d identity=%s", r.Kind, r.Selected, r.Alternatives, r.Data, hex.EncodeToString(r.SelectedIdentity[:8]))
}

func describeDiagnostic(record *choice.DiagnosticRecord) string {
	if record == nil {
		return "absent"
	}
	return fmt.Sprintf("time=%d runq=%d scheduler=%d select=%d rand=%d cheaprand=%d timer=%d clock_tick=%d", record.VirtualTime, record.RunqDraws, record.SchedulerDraws, record.SelectDraws, record.RuntimeRandDraws, record.RuntimeCheapRandDraws, record.TimerDraws, record.ClockTickDraws)
}

func describeOperation(op Operation) string {
	if op.Op == "" {
		return "absent"
	}
	text := fmt.Sprintf("%s(%s)=%s", op.Op, op.Input, op.Output)
	if op.Repeat > 1 {
		text += fmt.Sprintf(" x%d", op.Repeat)
	}
	return text
}

func window(data []byte, offset int) string {
	if offset >= len(data) {
		return "end"
	}
	return strconv.Quote(string(data[offset:min(len(data), offset+32)]))
}

func reproducer(expected, actual Run, first Divergence) *Reproducer {
	stop := backends.InspectionRequest{
		Stop:         backends.InspectionStop{Kind: "choice", Value: first.At.ChoiceOrdinal},
		Dumps:        []string{"stacks", "heap"},
		MaximumBytes: backends.MaximumInspectionBytes, PerDumpBytes: backends.MaximumInspectionBytes,
	}
	command := []string{"gomad", "replay", "--stop", "choice=" + strconv.FormatUint(first.At.ChoiceOrdinal, 10)}
	if len(expected.WorldOrder) != 0 || len(expected.Incarnations) > 1 {
		stop.Instance, stop.Incarnation = first.Instance, first.Incarnation
		command = append(command, "--instance", first.Instance, "--incarnation", strconv.FormatUint(first.Incarnation, 10))
	}
	command = append(command, cmp.Or(expected.ArtifactDir, "ARTIFACT_DIR"))
	return &Reproducer{
		Workload: expected.Workload, Seed: expected.Seed, Perturbation: actual.Perturbation,
		ExpectedDigest: expected.EvidenceDigest, ActualDigest: actual.EvidenceDigest,
		Identities:      maps.Clone(expected.Identities),
		ExpectedCommand: slices.Clone(expected.Command), ActualCommand: slices.Clone(actual.Command),
		Stop: stop, ReplayCommand: command,
	}
}

// Format renders a report for people.
func (report Report) Format() string {
	var out bytes.Buffer
	fmt.Fprintf(&out, "status: %s\n", report.Status)
	if report.Reason != "" {
		fmt.Fprintf(&out, "reason: %s\n", report.Reason)
	}
	if first := report.First; first != nil {
		fmt.Fprintf(&out, "first: %s", first.Kind)
		if first.Instance != "" {
			fmt.Fprintf(&out, " %s/%d", first.Instance, first.Incarnation)
		}
		fmt.Fprintf(&out, " ordinal=%d shared=%d tape=%d time=%d field=%s\n  expected: %s\n  actual:   %s\n", first.Ordinal, first.At.Shared, first.At.ChoiceOrdinal, first.At.VirtualTime, first.Field, first.Expected, first.Actual)
	}
	if r := report.Reproducer; r != nil {
		fmt.Fprintf(&out, "reproduce: workload=%s seed=%d cell=%s\n  expected: %s (%s)\n  actual:   %s (%s)\n  replay:   %s\n", r.Workload, r.Seed, r.Perturbation,
			strings.Join(r.ExpectedCommand, " "), r.ExpectedDigest, strings.Join(r.ActualCommand, " "), r.ActualDigest, strings.Join(r.ReplayCommand, " "))
	}
	return out.String()
}
