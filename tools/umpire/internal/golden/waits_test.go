package golden

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/proto"
)

// waitJob is a baseline whose realization spells out three waits, of which the delta lists two as
// derived, and its current spelling, which leaves exactly those two to the API behavior.
func waitJob(t *testing.T) (key string, delta Delta, baseline, current *umpirespb.Model) {
	t.Helper()
	key, baseline, _ = "ir/job.json", JobModel("once", "submit", "take", "finish"), Delta{}
	poll := func(id string, interval int64) *umpirespb.Command {
		return &umpirespb.Command{Id: id, Instruction: &umpirespb.Command_Poll{Poll: &umpirespb.Poll{Evidence: "e", Role: "r", IntervalMs: interval}}}
	}
	baseline.Realizations = []*umpirespb.Realization{{Id: "job.realization", Machine: "job", Scripts: []*umpirespb.Script{
		{Id: "controller", Items: []*umpirespb.Item{
			{Command: poll("await", 250)},
			{Command: poll("check", 250)},
			{Command: &umpirespb.Command{Id: "close", Instruction: &umpirespb.Command_Rpc{Rpc: &umpirespb.Rpc{Role: "r", Method: "/a.S/Close"}}}},
		}},
		{Id: "handler", Items: []*umpirespb.Item{{Performs: []*umpirespb.Performance{{Command: &umpirespb.Command{Id: "respond", TimeoutMs: 5000,
			Instruction: &umpirespb.Command_NexusReply{NexusReply: &umpirespb.NexusReply{}}}}}}}},
	}}}
	delta = Delta{DerivedWaits: []DerivedWait{{Model: key, Script: "controller", Command: "await"}, {Model: key, Script: "handler", Command: "respond"}}}
	require.NoError(t, delta.check())
	current = proto.CloneOf(baseline)
	current.Source = "model: moved roots"
	current.Realizations[0].Scripts[0].Items[0].Command.GetPoll().IntervalMs = 0
	current.Realizations[0].Scripts[1].Items[0].Performs[0].Command.TimeoutMs = 0
	return key, delta, baseline, current
}

// TestOriginalDerivedWaitsAreExact admits the listed waits a realization leaves to the API behavior,
// and nothing beside them: an unlisted wait that changes, a listed one still written or written
// otherwise, and an entry whose command the baseline lacks or writes no wait of, each fail.
func TestOriginalDerivedWaitsAreExact(t *testing.T) {
	key, delta, baseline, current := waitJob(t)
	compare := func(d Delta, baseline, current *umpirespb.Model) error {
		applied := Applied{}
		expected, err := d.Expected(key, baseline, applied)
		if err != nil {
			return err
		}
		return errors.Join(d.Unapplied(applied), d.MatchOriginalAt(key, expected, current))
	}
	require.NoError(t, compare(delta, baseline, current))
	expected, err := delta.Expected(key, baseline, Applied{})
	require.NoError(t, err)
	require.EqualValues(t, 250, expected.Realizations[0].Scripts[0].Items[0].Command.GetPoll().GetIntervalMs(), "the expected Model keeps the wait")
	require.EqualValues(t, 250, baseline.Realizations[0].Scripts[0].Items[0].Command.GetPoll().GetIntervalMs(), "the baseline is not changed")
	require.Zero(t, current.Realizations[0].Scripts[1].Items[0].Performs[0].Command.GetTimeoutMs(), "the current Model is not changed")
	require.Error(t, delta.MatchOriginal(expected, current), "without the key no derived wait applies")
	require.Error(t, Delta{}.MatchOriginalAt(key, expected, current), "a wait the delta does not list")

	controller := func(m *umpirespb.Model, i int) *umpirespb.Command {
		return m.Realizations[0].Scripts[0].Items[i].Command
	}
	rejected := map[string]struct {
		delta   func(*Delta)
		current func(*umpirespb.Model)
	}{
		"unlisted interval changed":     {current: func(m *umpirespb.Model) { controller(m, 1).GetPoll().IntervalMs = 500 }},
		"unlisted interval left":        {current: func(m *umpirespb.Model) { controller(m, 1).GetPoll().IntervalMs = 0 }},
		"unlisted timeout added":        {current: func(m *umpirespb.Model) { controller(m, 2).TimeoutMs = 5000 }},
		"listed wait still written":     {current: func(m *umpirespb.Model) { controller(m, 0).GetPoll().IntervalMs = 250 }},
		"listed wait written otherwise": {current: func(m *umpirespb.Model) { controller(m, 0).TimeoutMs = 5000 }},
		"listed command changed beside its wait": {current: func(m *umpirespb.Model) {
			controller(m, 0).GetPoll().Evidence = "other"
		}},
		"listed command dropped": {current: func(m *umpirespb.Model) { m.Realizations[0].Scripts[0].Items = m.Realizations[0].Scripts[0].Items[1:] }},
		"entry of no command": {delta: func(d *Delta) {
			d.DerivedWaits = append(d.DerivedWaits, DerivedWait{Model: key, Script: "controller", Command: "nothing"})
		}},
		"entry of no written wait": {delta: func(d *Delta) {
			d.DerivedWaits = append(d.DerivedWaits, DerivedWait{Model: key, Script: "controller", Command: "close"})
		}},
		"entry of another script": {delta: func(d *Delta) { d.DerivedWaits[0].Script = "handler" }},
		"entry of no baseline Model": {delta: func(d *Delta) {
			d.DerivedWaits = append(d.DerivedWaits, DerivedWait{Model: "ir/other.json", Script: "controller", Command: "await"})
		}},
		"entry left out": {delta: func(d *Delta) { d.DerivedWaits = d.DerivedWaits[1:] }},
	}
	for name, c := range rejected {
		t.Run(name, func(t *testing.T) {
			d := Delta{DerivedWaits: slices.Clone(delta.DerivedWaits)}
			if c.delta != nil {
				c.delta(&d)
			}
			changed := proto.CloneOf(current)
			if c.current != nil {
				c.current(changed)
			}
			require.Error(t, compare(d, baseline, changed))
		})
	}
	_, err = delta.ProjectBaseline(key, current)
	require.ErrorContains(t, err, "writes no wait", "a baseline must write each listed wait")
	_, err = delta.ProjectCurrent(key, baseline)
	require.ErrorContains(t, err, "still writes its wait", "a current Model may write none")
}

func TestOriginalDerivedWaitsAreClosed(t *testing.T) {
	valid := DerivedWait{Model: "ir/a.json", Script: "controller", Command: "await"}
	require.NoError(t, Delta{DerivedWaits: []DerivedWait{valid, {Model: "ir/a.json", Script: "handler", Command: "await"}}}.check())
	with := func(change func(*DerivedWait)) []DerivedWait {
		w := valid
		change(&w)
		return []DerivedWait{w}
	}
	for name, invalid := range map[string]Delta{
		"not an IR file":   {DerivedWaits: with(func(w *DerivedWait) { w.Model = "lifts/a.json" })},
		"a sidecar":        {DerivedWaits: with(func(w *DerivedWait) { w.Model = "ir/a.laws.json" })},
		"no file":          {DerivedWaits: with(func(w *DerivedWait) { w.Model = "" })},
		"no script":        {DerivedWaits: with(func(w *DerivedWait) { w.Script = "" })},
		"no command":       {DerivedWaits: with(func(w *DerivedWait) { w.Command = "" })},
		"listed twice":     {DerivedWaits: []DerivedWait{valid, valid}},
		"of a new IR file": {DerivedWaits: []DerivedWait{valid}, NewIRFiles: []string{"ir/a.json"}},
	} {
		require.Error(t, invalid.check(), name)
	}
}

// TestOriginalDerivedWaitsAreWhatTheArchiveDiffers holds the recorded list to the archived IR files:
// each entry leaves one difference, so the list without it fails its file's comparison, and a command
// the list leaves out cannot be added, its baseline writing no wait or its current one still writing it.
func TestOriginalDerivedWaitsAreWhatTheArchiveDiffers(t *testing.T) {
	root, err := Root()
	require.NoError(t, err)
	delta, err := OriginalDelta()
	require.NoError(t, err)
	require.Len(t, delta.DerivedWaits, 23)
	archived, err := OriginalArchive(root)
	require.NoError(t, err)
	current, err := OriginalCurrent(root)
	require.NoError(t, err)
	baselines, err := OriginalModels(delta.Compared(archived))
	require.NoError(t, err)
	models, err := OriginalModels(current)
	require.NoError(t, err)
	match := func(d Delta, key string) error {
		expected, err := d.Expected(key, baselines[key], Applied{})
		if err != nil {
			return err
		}
		ungenerated, err := d.Ungenerated(key, models[key])
		if err != nil {
			return err
		}
		return d.MatchOriginalAt(key, expected, ungenerated)
	}
	for i, w := range delta.DerivedWaits {
		require.NoError(t, match(delta, w.Model), w.Model)
		d := delta
		d.DerivedWaits = slices.Delete(slices.Clone(delta.DerivedWaits), i, i+1)
		require.Error(t, match(d, w.Model), "%+v left out", w)
	}
	// Every other command of a script of an archived IR file writes no wait or keeps it.
	keys := slices.Sorted(maps.Keys(baselines))
	added := 0
	for _, key := range keys {
		for _, r := range baselines[key].GetRealizations() {
			for _, s := range r.GetScripts() {
				for _, item := range s.GetItems() {
					commands := []*umpirespb.Command{item.GetCommand()}
					for _, p := range item.GetPerforms() {
						commands = append(commands, p.GetCommand())
					}
					for _, c := range commands {
						w := DerivedWait{Model: key, Script: s.GetId(), Command: c.GetId()}
						if c == nil || slices.Contains(delta.DerivedWaits, w) {
							continue
						}
						d := delta
						d.DerivedWaits = append(slices.Clone(delta.DerivedWaits), w)
						require.Error(t, match(d, key), "%+v added", w)
						added++
					}
				}
			}
		}
	}
	require.Positive(t, added)
}

// nodeWaits are the waits of one instruction node: the members its read has after `until`, and the
// members the node has after its instruction.
type nodeWaits struct{ read, node string }

// waitCase is a compact Case, as the lowering writes it, whose controller carries a listed command
// twice, the second time under its ordinal, an unlisted command, and one whose id only begins with
// the listed one's, and whose handler carries a command of the listed command's id. waitsOf gives
// each instruction's waits, by entrypoint and instruction id.
func waitCase(waitsOf func(entrypoint, id string) nodeWaits, contract string) string {
	node := func(entrypoint, id string) string {
		w := waitsOf(entrypoint, id)
		return `{"instructionId":"` + id + `","instruction":{"readEvidence":{"evidenceId":"e.` + entrypoint + `.` + id +
			`","until":{"present":{}}` + w.read + `}}` + w.node + `}`
	}
	return `{"caseId":"c","version":{"major":1},"provenance":{"producerId":"p"},"program":{"programId":"p","entrypoints":[` +
		`{"entrypointId":"controller","instructions":[` + node("controller", "await") + `,` + node("controller", "await-2") + `,` +
		node("controller", "check") + `,` + node("controller", "await-x") + `]},` +
		`{"entrypointId":"handler","instructions":[` + node("handler", "await") + `]},` +
		`{"entrypointId":"cleanup"}]},"contract":{"contractId":"k","state":"` + contract + `"}}` + "\n"
}

// TestDerivedWaitsProjectOnlyTheDerivedMembers drops the waits the lowering derives from the
// instructions a listed command is carried as, and keeps every other byte: the Contract's, another
// instruction's and the rest of a listed instruction's.
func TestDerivedWaitsProjectOnlyTheDerivedMembers(t *testing.T) {
	waits := Waits{{Model: "ir/a.json", Script: "controller", Command: "await"}}
	explicit := nodeWaits{read: `,"pollIntervalMilliseconds":"250"`, node: `,"limits":{"timeoutMilliseconds":"5000"}`}
	derived := nodeWaits{read: `,"pollIntervalMilliseconds":"250","once":true`,
		node: `,"limits":{"timeoutMilliseconds":"7000"},"waitHints":[{"hintId":"cause","source":{"path":"Behavior.scala"},"atMostMilliseconds":"7000"}]`}
	listed := func(entrypoint, id string) bool {
		return entrypoint == "controller" && (id == "await" || id == "await-2")
	}
	// spelled gives the listed instructions w and every other one the explicit waits, with change
	// applied to the waits of one other instruction.
	spelled := func(w nodeWaits, change func(entrypoint, id string, w *nodeWaits)) func(string, string) nodeWaits {
		return func(entrypoint, id string) nodeWaits {
			out := explicit
			if listed(entrypoint, id) {
				out = w
			}
			if change != nil {
				change(entrypoint, id, &out)
			}
			return out
		}
	}
	base := waitCase(spelled(explicit, nil), "s")
	projected, err := waits.Case([]byte(base))
	require.NoError(t, err)
	require.Equal(t, waitCase(spelled(nodeWaits{}, nil), "s"), string(projected), "exactly the listed instructions' waits are dropped")

	compare := func(want, got string) error {
		w, err := waits.Case([]byte(want))
		if err != nil {
			return err
		}
		g, err := waits.Case([]byte(got))
		if err != nil {
			return err
		}
		return Compare(map[string][]byte{"case": w}, map[string][]byte{"case": g})
	}
	require.NoError(t, compare(base, waitCase(spelled(derived, nil), "s")), "a listed instruction's derived waits")
	require.NoError(t, compare(base, waitCase(spelled(nodeWaits{}, nil), "s")), "a listed instruction without waits")

	limits := func(at string) func(entrypoint, id string, w *nodeWaits) {
		return func(entrypoint, id string, w *nodeWaits) {
			if entrypoint+"/"+id == at {
				w.node = `,"limits":{"timeoutMilliseconds":"6000"}`
			}
		}
	}
	for name, changed := range map[string]string{
		"Contract byte":                 waitCase(spelled(derived, nil), "t"),
		"provenance":                    strings.Replace(base, `"producerId":"p"`, `"producerId":"q"`, 1),
		"listed instruction's until":    strings.Replace(base, `"until":{"present":{}}`, `"until":{"absent":{}}`, 1),
		"listed instruction's read":     strings.Replace(base, `"evidenceId":"e.controller.await"`, `"evidenceId":"e.other"`, 1),
		"listed instruction's member":   strings.Replace(base, `{"instructionId":"await",`, `{"instructionId":"await","guard":{},`, 1),
		"unlisted instruction's limits": waitCase(spelled(explicit, limits("controller/check")), "s"),
		"unlisted ordinal's limits":     waitCase(spelled(explicit, limits("controller/await-x")), "s"),
		"listed id in another script":   waitCase(spelled(explicit, limits("handler/await")), "s"),
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEqual(t, base, changed)
			require.Error(t, compare(base, changed))
		})
	}

	const indented = "{\n  \"indented\": true\n}"
	kept, err := Waits(nil).Case([]byte(indented))
	require.NoError(t, err, "with no waits listed a Case is kept as it is")
	require.Equal(t, indented, string(kept))
	_, err = waits.Case([]byte(indented))
	require.ErrorContains(t, err, "not compact JSON", "a Case that cannot be rewritten member by member")
	_, err = waits.Case([]byte(`{"program":{"entrypoints":[{"instructions":[]}]}}`))
	require.ErrorContains(t, err, "no entrypointId")
	program := base[strings.Index(base, `"program":`)+len(`"program":`) : strings.Index(base, `,"contract":`)]
	got, err := waits.Program([]byte(program))
	require.NoError(t, err)
	require.Contains(t, string(projected), `"program":`+string(got)+`,"contract":`, "a Program alone is read as within its Case")
}
