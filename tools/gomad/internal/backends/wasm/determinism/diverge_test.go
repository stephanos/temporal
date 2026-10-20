package determinism

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"go.temporal.io/server/tools/gomad/internal/model/choice"
)

var updateDivergeFixtures = flag.Bool("update-diverge-fixtures", false, "rewrite testdata/diverge golden reports")

// perturbation is a test-only hook into the fixture execution. Nothing like it
// exists in the release helper or runtime.
type perturbation struct {
	name string
	// drawAt adds one draw from the run-queue stream before this choice point
	// of instance drawInstance.
	drawInstance string
	drawAt       int
	// clockAt shifts the answer of this clock read of instance clockInstance
	// by one nanosecond.
	clockInstance string
	clockAt       int
	// swapStep exchanges the cluster handoffs at this World step and the next.
	swapStep int
}

var unperturbed = perturbation{name: "reference", drawAt: -1, clockAt: -1, swapStep: -1}

// simulate is a seeded stand-in for a WASM execution: instances take turns
// holding the baton in a seeded World order; at every choice point an
// instance reads the virtual clock on even points, draws from seeded streams
// and records a choice and a diagnostic record. It produces the same evidence
// shapes Compare consumes.
func simulate(t *testing.T, workload string, seed uint64, instances []string, hook perturbation) Run {
	t.Helper()
	order := rand.New(rand.NewPCG(seed, 1))
	steps := 3
	if len(instances) > 1 {
		steps = 6
	}
	var world []WorldStep
	for step := range steps {
		instance := instances[order.IntN(len(instances))]
		if len(instances) > 1 && step < len(instances) {
			instance = instances[step]
		}
		world = append(world, WorldStep{Order: uint64(step), Instance: instance, Incarnation: 1})
	}
	if hook.swapStep >= 0 {
		world[hook.swapStep].Instance, world[hook.swapStep+1].Instance = world[hook.swapStep+1].Instance, world[hook.swapStep].Instance
	}
	type state struct {
		evidence Incarnation
		random   *rand.Rand
		now      int64
		clocks   int
		records  []choice.DiagnosticRecord
		counters choice.DiagnosticRecord
	}
	states := map[string]*state{}
	for index, instance := range instances {
		states[instance] = &state{evidence: Incarnation{Instance: instance, Incarnation: 1}, random: rand.New(rand.NewPCG(seed, uint64(index)+2)), now: 946684800000000000}
	}
	for step := range world {
		s := states[world[step].Instance]
		// A single guest has no shared World order.
		shared := uint64(step)
		if len(instances) == 1 {
			shared = 0
		}
		ordinal := uint64(len(s.evidence.Choices))
		world[step].ChoiceOrdinal, world[step].VirtualTime = ordinal, s.now
		for range 3 {
			ordinal := uint64(len(s.evidence.Choices))
			at := Coordinate{Shared: shared, ChoiceOrdinal: ordinal, VirtualTime: s.now}
			if ordinal%2 == 0 {
				answer := s.now + 1000
				if s.evidence.Instance == hook.clockInstance && s.clocks == hook.clockAt {
					answer++
				}
				s.clocks++
				s.evidence.Operations = append(s.evidence.Operations, Operation{At: at, Op: "clock_time_get", Input: "clock_id=1", Output: strconv.FormatInt(answer, 10), Repeat: 1})
				s.now = answer
			}
			s.counters.RunqDraws += 1 + uint64(s.random.IntN(2))
			if s.evidence.Instance == hook.drawInstance && int(ordinal) == hook.drawAt {
				s.counters.RunqDraws++
			}
			s.counters.SchedulerDraws++
			record := s.counters
			record.Ordinal, record.VirtualTime = ordinal, s.now
			s.records = append(s.records, record)
			selected := uint32(s.random.IntN(3))
			s.evidence.Choices = append(s.evidence.Choices, ChoiceEvent{At: Coordinate{shared, ordinal, s.now}, Record: choice.Record{Ordinal: ordinal, Kind: 1, Alternatives: 3, Selected: selected}})
			s.evidence.Transitions = append(s.evidence.Transitions, Transition{At: Coordinate{shared, ordinal + 1, s.now}, Model: "world", Digest: fmt.Sprintf("%s-%d-%d", s.evidence.Instance, ordinal, selected)})
		}
		s.evidence.Stdout = append(s.evidence.Stdout, fmt.Sprintf("step %d at %d\n", step, s.now)...)
	}
	run := Run{Workload: workload, Seed: seed, Perturbation: hook.name, Identities: map[string]string{"source": "s", "helper": "h", "module": "m", "engine": "e", "model": "d"},
		Command: []string{"gomad", "explore", "--backend", "wasm", "--seed", strconv.FormatUint(seed, 10), DiagnosticsFlag, "go-run", workload}, ArtifactDir: "/artifacts/" + hook.name}
	if len(instances) > 1 {
		run.WorldOrder = world
	}
	for _, instance := range instances {
		s := states[instance]
		trace, err := choice.BuildDiagnosticTrace(s.records, 1<<16)
		if err != nil {
			t.Fatal(err)
		}
		s.evidence.Diagnostic = trace.Bytes
		run.Incarnations = append(run.Incarnations, s.evidence)
	}
	data, err := json.Marshal(run)
	if err != nil {
		t.Fatal(err)
	}
	run.EvidenceDigest = fmt.Sprintf("sha256:%x", sha256.Sum256(data))
	return run
}

func TestCompareFirstDivergenceFixtures(t *testing.T) {
	cluster := []string{"a", "b"}
	for _, test := range []struct {
		name      string
		instances []string
		hook      perturbation
		want      Divergence
	}{
		{"single-perturbed-draw", []string{"guest"}, perturbation{name: "draw", drawInstance: "guest", drawAt: 4, clockAt: -1, swapStep: -1},
			Divergence{Kind: KindDrawCounter, Instance: "guest", Incarnation: 1, Ordinal: 4, Field: "runq_draws"}},
		{"single-shifted-clock", []string{"guest"}, perturbation{name: "clock", drawAt: -1, clockInstance: "guest", clockAt: 1, swapStep: -1},
			Divergence{Kind: KindClock, Instance: "guest", Incarnation: 1, Ordinal: 1, Field: "operation"}},
		{"cluster-perturbed-draw", cluster, perturbation{name: "draw", drawInstance: "b", drawAt: 2, clockAt: -1, swapStep: -1},
			Divergence{Kind: KindDrawCounter, Instance: "b", Incarnation: 1, Ordinal: 2, Field: "runq_draws"}},
		{"cluster-shifted-clock", cluster, perturbation{name: "clock", drawAt: -1, clockInstance: "a", clockAt: 2, swapStep: -1},
			Divergence{Kind: KindClock, Instance: "a", Incarnation: 1, Ordinal: 2, Field: "operation"}},
		{"cluster-reordered-handoff", cluster, perturbation{name: "handoff", drawAt: -1, clockAt: -1, swapStep: 2},
			Divergence{Kind: KindWorldOrder, Ordinal: 2, Field: "step"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			reference := simulate(t, "fixture/"+test.name, 7, test.instances, unperturbed)
			again := simulate(t, "fixture/"+test.name, 7, test.instances, unperturbed)
			if report, err := Compare(reference, again); err != nil || report.Status != "identical" {
				t.Fatalf("fresh same-seed fixture runs differ: %+v %v", report, err)
			}
			perturbed := simulate(t, "fixture/"+test.name, 7, test.instances, test.hook)
			report, err := Compare(reference, perturbed)
			if err != nil {
				t.Fatal(err)
			}
			if report.Status != "diverged" || report.First == nil {
				t.Fatalf("perturbation not detected: %+v", report)
			}
			got := *report.First
			if got.Kind != test.want.Kind || got.Ordinal != test.want.Ordinal || got.Field != test.want.Field || test.want.Instance != "" && (got.Instance != test.want.Instance || got.Incarnation != test.want.Incarnation) {
				t.Fatalf("first divergence = %+v, want %+v", got, test.want)
			}
			if got.Expected == got.Actual || got.Expected == "" {
				t.Fatalf("divergence lacks both values: %+v", got)
			}
			requireReproducer(t, reference, perturbed, report)
			golden(t, test.name, report)
		})
	}
}

func requireReproducer(t *testing.T, expected, actual Run, report Report) {
	t.Helper()
	r := report.Reproducer
	if r == nil || r.Workload != expected.Workload || r.Seed != expected.Seed || r.Perturbation != actual.Perturbation || r.ExpectedDigest != expected.EvidenceDigest || r.ActualDigest != actual.EvidenceDigest ||
		!slices.Equal(r.ExpectedCommand, expected.Command) || !slices.Equal(r.ActualCommand, actual.Command) || len(r.Identities) != len(expected.Identities) {
		t.Fatalf("reproducer incomplete: %+v", r)
	}
	// The prefix cut is a gomad replay stop point: a valid inspection
	// request and the matching --stop flags.
	if err := r.Stop.Validate(); err != nil {
		t.Fatalf("prefix cut is not a valid replay stop: %v", err)
	}
	if r.Stop.Stop.Kind != "choice" || r.Stop.Stop.Value != report.First.At.ChoiceOrdinal {
		t.Fatalf("stop %+v does not cut at the divergence %+v", r.Stop.Stop, report.First.At)
	}
	want := []string{"gomad", "replay", "--stop", "choice=" + strconv.FormatUint(report.First.At.ChoiceOrdinal, 10)}
	if len(expected.WorldOrder) != 0 {
		if r.Stop.Instance != report.First.Instance || r.Stop.Incarnation != report.First.Incarnation {
			t.Fatalf("cluster stop does not name the divergent incarnation: %+v", r.Stop)
		}
		want = append(want, "--instance", report.First.Instance, "--incarnation", strconv.FormatUint(report.First.Incarnation, 10))
	}
	if !slices.Equal(r.ReplayCommand, append(want, expected.ArtifactDir)) {
		t.Fatalf("replay command = %q", r.ReplayCommand)
	}
}

func golden(t *testing.T, name string, report Report) {
	t.Helper()
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	text := []byte(report.Format())
	for path, contents := range map[string][]byte{filepath.Join("testdata", "diverge", name+".json"): data, filepath.Join("testdata", "diverge", name+".txt"): text} {
		if *updateDivergeFixtures {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, contents, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (regenerate with -update-diverge-fixtures)", err)
		}
		if !bytes.Equal(want, contents) {
			t.Fatalf("%s differs from the report:\n%s", path, contents)
		}
	}
}

func TestCompareRefusesIncomparableRunsAndReportsUndiagnosable(t *testing.T) {
	reference := simulate(t, "fixture/refusal", 7, []string{"guest"}, unperturbed)
	other := simulate(t, "fixture/refusal", 8, []string{"guest"}, unperturbed)
	if _, err := Compare(reference, other); !errors.Is(err, ErrIncomparable) {
		t.Fatalf("different seeds compared: %v", err)
	}
	changed := simulate(t, "fixture/refusal", 7, []string{"guest"}, unperturbed)
	changed.Identities["helper"] = "other"
	if _, err := Compare(reference, changed); !errors.Is(err, ErrIncomparable) {
		t.Fatalf("different identities compared: %v", err)
	}
	renamed := simulate(t, "fixture/other", 7, []string{"guest"}, unperturbed)
	if _, err := Compare(reference, renamed); !errors.Is(err, ErrIncomparable) {
		t.Fatalf("different workloads compared: %v", err)
	}
	untraced := simulate(t, "fixture/refusal", 7, []string{"guest"}, unperturbed)
	untraced.Incarnations[0].Diagnostic = nil
	report, err := Compare(reference, untraced)
	if err != nil || report.Status != "undiagnosable" || !strings.Contains(report.Reason, DiagnosticsFlag) || report.First != nil {
		t.Fatalf("missing trace not undiagnosable: %+v %v", report, err)
	}
}

func TestCompareOrdersCausesBeforeEffects(t *testing.T) {
	// A shifted clock read also moves the virtual time recorded at the next
	// choice point; the read itself is reported, not its effect.
	reference := simulate(t, "fixture/order", 3, []string{"guest"}, unperturbed)
	shifted := simulate(t, "fixture/order", 3, []string{"guest"}, perturbation{name: "clock", drawAt: -1, clockInstance: "guest", clockAt: 0, swapStep: -1})
	report, err := Compare(reference, shifted)
	if err != nil {
		t.Fatal(err)
	}
	if report.First.Kind != KindClock || report.First.Field != "operation" || report.First.At.ChoiceOrdinal != 0 {
		t.Fatalf("first divergence = %+v", report.First)
	}
	// A diagnostic-only virtual-time difference is a clock divergence.
	diagnosticOnly := simulate(t, "fixture/order", 3, []string{"guest"}, unperturbed)
	records := mustDecode(t, diagnosticOnly.Incarnations[0].Diagnostic)
	records[3].VirtualTime++
	for index := 4; index < len(records); index++ {
		records[index].VirtualTime = max(records[index].VirtualTime, records[3].VirtualTime)
	}
	trace, err := choice.BuildDiagnosticTrace(records, 1<<16)
	if err != nil {
		t.Fatal(err)
	}
	diagnosticOnly.Incarnations[0].Diagnostic = trace.Bytes
	report, err = Compare(reference, diagnosticOnly)
	if err != nil {
		t.Fatal(err)
	}
	if report.First.Kind != KindClock || report.First.Field != "virtual_time" || report.First.Ordinal != 3 {
		t.Fatalf("diagnostic clock divergence = %+v", report.First)
	}
}

func mustDecode(t *testing.T, data []byte) []choice.DiagnosticRecord {
	t.Helper()
	trace, err := choice.DecodeDiagnosticTrace(data)
	if err != nil {
		t.Fatal(err)
	}
	return trace.Records
}

func TestCompareTrailingDiagnosticRecordKeepsItsClusterCoordinate(t *testing.T) {
	cluster := []string{"a", "b"}
	reference := simulate(t, "fixture/trailing", 7, cluster, unperturbed)
	extended := simulate(t, "fixture/trailing", 7, cluster, unperturbed)
	records := mustDecode(t, extended.Incarnations[0].Diagnostic)
	extra := records[len(records)-1]
	extra.Ordinal++
	trace, err := choice.BuildDiagnosticTrace(append(records, extra), 1<<16)
	if err != nil {
		t.Fatal(err)
	}
	extended.Incarnations[0].Diagnostic = trace.Bytes
	report, err := Compare(reference, extended)
	if err != nil {
		t.Fatal(err)
	}
	instance := extended.Incarnations[0]
	last := instance.Choices[len(instance.Choices)-1].At.Shared
	if report.First == nil || report.First.Field != "record_presence" || report.First.At.Shared != last {
		t.Fatalf("trailing record placed at %+v, want shared %d", report.First, last)
	}
}
