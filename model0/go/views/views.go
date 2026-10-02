// Package views renders read-only views of Go Models for people who read them rather than write
// them: a transition table grouped by phase, a state diagram, a behavior summary, and a behavior
// diff between two revisions. Every view is computed from model data, never from Go source, so any
// front end that produces the same tables produces the same views. Views are deterministic and are
// checked in as goldens; a view is never edited by hand.
package views

import (
	"fmt"
	"slices"
	"strings"

	"go.temporal.io/server/model/go/umpire"
)

// phaseOf is a state key's phase: its first field. A composed state keeps its members apart.
func phaseOf(state string) string {
	parts := strings.Split(state, "_")
	for i, p := range parts {
		parts[i], _, _ = strings.Cut(p, "-")
	}
	return strings.Join(parts, " / ")
}

// restOf is a state key without its phase: the fields a folded row may leave unchanged.
func restOf(state string) string {
	_, rest, _ := strings.Cut(state, "-")
	return rest
}

// actionOf is an action key's action: the part before its first class value.
func actionOf(key string) string {
	name, _, _ := strings.Cut(key, "-")
	return name
}

type foldKey struct {
	fromPhase, action, outcome, toPhase, facts string
	keepsFields                                bool
	because                                    string
}

type fold struct {
	key     foldKey
	sources []string
}

// Table renders the machine's transitions as Markdown, one section per phase. Rows that differ only
// in fields the step carries over unchanged fold into one line, with the number of states it stands
// for, so a table of hundreds of rows reads in dozens.
func Table(t *umpire.Table) string {
	reachable := map[string]bool{}
	for _, s := range t.Reachable {
		reachable[s] = true
	}
	var folds []fold
	index := map[foldKey]int{}
	for _, r := range t.Rows {
		if !reachable[r.Source] {
			continue
		}
		for _, res := range r.Results {
			k := foldKey{phaseOf(r.Source), r.Action, res.Outcome, phaseOf(res.State),
				strings.Join(res.Facts, ", "), restOf(r.Source) == restOf(res.State), res.Because}
			if i, ok := index[k]; ok {
				folds[i].sources = append(folds[i].sources, r.Source)
				continue
			}
			index[k] = len(folds)
			folds = append(folds, fold{k, []string{r.Source}})
		}
	}
	var phases []string
	for _, f := range folds {
		if !slices.Contains(phases, f.key.fromPhase) {
			phases = append(phases, f.key.fromPhase)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s: transitions\n\n", t.Machine)
	fmt.Fprintf(&b, "%d reachable states, %d action classes, %d rows. Generated; do not edit.\n\n",
		len(t.Reachable), len(t.Actions), len(t.Rows))
	b.WriteString("\"Fields\" says whether the step keeps every field but the phase; \"States\" is how many\n")
	b.WriteString("reachable states the line stands for.\n")
	for _, phase := range phases {
		fmt.Fprintf(&b, "\n## From %s\n\n", phase)
		b.WriteString("| Action | Outcome | To | Facts | Fields | States | Because |\n")
		b.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
		for _, f := range folds {
			if f.key.fromPhase != phase {
				continue
			}
			fields := "change"
			if f.key.keepsFields {
				fields = "kept"
			}
			facts := f.key.facts
			if facts == "" {
				facts = "none"
			}
			fmt.Fprintf(&b, "| `%s` | %s | %s | %s | %s | %d | %s |\n", f.key.action, f.key.outcome, f.key.toPhase,
				facts, fields, len(f.sources), f.key.because)
		}
	}
	return b.String()
}

// Diagram renders the machine's phases and the actions between them as a Mermaid state diagram,
// which GitHub renders inline.
func Diagram(t *umpire.Table) string {
	reachable := map[string]bool{}
	for _, s := range t.Reachable {
		reachable[s] = true
	}
	id := func(phase string) string { return strings.NewReplacer(" / ", "__", "-", "_").Replace(phase) }
	edges := map[[2]string][]string{}
	var order [][2]string
	for _, r := range t.Rows {
		if !reachable[r.Source] {
			continue
		}
		for _, res := range r.Results {
			from, to := phaseOf(r.Source), phaseOf(res.State)
			if from == to {
				continue
			}
			e := [2]string{from, to}
			if _, ok := edges[e]; !ok {
				order = append(order, e)
			}
			if a := actionOf(r.Action); !slices.Contains(edges[e], a) {
				edges[e] = append(edges[e], a)
			}
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s: phases\n\nGenerated; do not edit. Self-loops and the fields besides the phase are left out.\n\n", t.Machine)
	b.WriteString("```mermaid\nstateDiagram-v2\n")
	var starts []string
	for _, s := range t.Starts {
		if p := phaseOf(s); !slices.Contains(starts, p) {
			starts = append(starts, p)
		}
	}
	for _, s := range starts {
		fmt.Fprintf(&b, "    [*] --> %s\n", id(s))
	}
	for _, e := range order {
		fmt.Fprintf(&b, "    %s --> %s: %s\n", id(e[0]), id(e[1]), strings.Join(edges[e], ", "))
	}
	var ends []string
	for _, s := range t.Ends {
		if p := phaseOf(s); reachable[s] && !slices.Contains(ends, p) {
			ends = append(ends, p)
		}
	}
	for _, e := range ends {
		fmt.Fprintf(&b, "    %s --> [*]\n", id(e))
	}
	b.WriteString("```\n")
	return b.String()
}

// Declarations is what a summary describes: machines, Queries and sets, in the order given.
type Declarations struct {
	Title    string
	Machines []umpire.Model
	Queries  []*umpire.Query
	Sets     []*umpire.Set
}

// Summary renders a compact, declarative description of a Model: what each machine starts and ends
// in and which actions it takes, and what each Query asks and answers.
func Summary(d Declarations) (string, error) {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nGenerated; do not edit.\n", d.Title)
	for _, m := range d.Machines {
		t, err := m.Table()
		if err != nil {
			return "", err
		}
		machineSummary(&b, t)
	}
	if err := querySummary(&b, d.Queries); err != nil {
		return "", err
	}
	for _, s := range d.Sets {
		if err := setSummary(&b, s); err != nil {
			return "", err
		}
	}
	return b.String(), nil
}

func phasesOf(states []string) []string {
	var phases []string
	for _, s := range states {
		if p := phaseOf(s); !slices.Contains(phases, p) {
			phases = append(phases, p)
		}
	}
	return phases
}

func machineSummary(b *strings.Builder, t *umpire.Table) {
	actions := map[string]int{}
	var names []string
	for _, a := range t.Actions {
		n := actionOf(a)
		if actions[n] == 0 {
			names = append(names, n)
		}
		actions[n]++
	}
	fmt.Fprintf(b, "\n## machine %s\n\n", t.Machine)
	if t.Entity != "" {
		fmt.Fprintf(b, "- for: %s\n", t.Entity)
	}
	fmt.Fprintf(b, "- starts: %s\n", strings.Join(t.Starts, ", "))
	fmt.Fprintf(b, "- ends in: %s\n", strings.Join(phasesOf(t.Ends), ", "))
	fmt.Fprintf(b, "- reaches: %s (%d of %d states)\n", strings.Join(phasesOf(t.Reachable), ", "), len(t.Reachable), len(t.States))
	var listed []string
	for _, n := range names {
		if actions[n] == 1 {
			listed = append(listed, n)
		} else {
			listed = append(listed, fmt.Sprintf("%s (%d classes)", n, actions[n]))
		}
	}
	fmt.Fprintf(b, "- actions: %s\n", strings.Join(listed, ", "))
	if len(t.Evidence) > 0 {
		var evidence []string
		for _, e := range t.Evidence {
			if e[0] == e[1] {
				evidence = append(evidence, e[0])
			} else {
				evidence = append(evidence, e[0]+" by "+e[1])
			}
		}
		fmt.Fprintf(b, "- evidence: %s\n", strings.Join(evidence, ", "))
	}
}

func querySummary(b *strings.Builder, queries []*umpire.Query) error {
	if len(queries) == 0 {
		return nil
	}
	b.WriteString("\n## Queries\n\n| Query | Asks | Property | On path | Limits | Answer |\n| --- | --- | --- | --- | --- | --- |\n")
	for _, q := range queries {
		a, err := q.Answer()
		if err != nil {
			return err
		}
		form := "find"
		if q.Form == umpire.VerifyForm {
			form = "verify"
		}
		path := strings.Join(q.Scenario.Actions, " → ")
		if path == "" {
			path = "any"
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s |\n", q.Name, form, q.Property.Name, path, q.Limits.Name, a.Outcome)
	}
	return nil
}

func setSummary(b *strings.Builder, s *umpire.Set) error {
	fmt.Fprintf(b, "\n## set %s (%s)\n\n", s.Name, s.Purpose)
	var parties []string
	for party, binding := range s.Bindings {
		parties = append(parties, fmt.Sprintf("%s %s", party, binding))
	}
	slices.Sort(parties)
	fmt.Fprintf(b, "- binds: %s\n", strings.Join(parties, ", "))
	if s.Purpose == umpire.Exploratory {
		targets, err := s.Targets()
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "- covers %s with %d targets under %s\n", s.Machine.Name(), len(targets), s.Budget.Name)
		return nil
	}
	var names []string
	for _, q := range s.Queries {
		names = append(names, q.Name)
	}
	fmt.Fprintf(b, "- queries: %s\n", strings.Join(names, ", "))
	return nil
}

// Diff renders what changed between two revisions of a machine: action classes and rows added or
// removed, and rows whose results changed. A reviewer reads this instead of the Go diff.
func Diff(title string, before, after *umpire.Table) string {
	rows := func(t *umpire.Table) map[string]string {
		out := map[string]string{}
		for _, r := range t.Rows {
			var results []string
			for _, res := range r.Results {
				results = append(results, fmt.Sprintf("%s → %s [%s]", res.Outcome, res.State, strings.Join(res.Facts, ", ")))
			}
			out[r.Key] = strings.Join(results, "; ")
		}
		return out
	}
	b4, af := rows(before), rows(after)
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\nGenerated; do not edit.\n\n", title)
	var addedClasses, removedClasses []string
	for _, a := range after.Actions {
		if !slices.Contains(before.Actions, a) {
			addedClasses = append(addedClasses, "`"+a+"`")
		}
	}
	for _, a := range before.Actions {
		if !slices.Contains(after.Actions, a) {
			removedClasses = append(removedClasses, "`"+a+"`")
		}
	}
	fmt.Fprintf(&b, "- action classes added: %s\n", orNone(addedClasses))
	fmt.Fprintf(&b, "- action classes removed: %s\n", orNone(removedClasses))
	fmt.Fprintf(&b, "- reachable states: %d → %d\n", len(before.Reachable), len(after.Reachable))
	var added, removed, changed []string
	for _, r := range after.Rows {
		if old, ok := b4[r.Key]; !ok {
			added = append(added, fmt.Sprintf("| `%s` | %s |", r.Key, af[r.Key]))
		} else if old != af[r.Key] {
			changed = append(changed, fmt.Sprintf("| `%s` | %s | %s |", r.Key, old, af[r.Key]))
		}
	}
	for _, r := range before.Rows {
		if _, ok := af[r.Key]; !ok {
			removed = append(removed, fmt.Sprintf("| `%s` | %s |", r.Key, b4[r.Key]))
		}
	}
	section := func(name, header string, lines []string) {
		fmt.Fprintf(&b, "\n## %s (%d)\n\n", name, len(lines))
		if len(lines) == 0 {
			b.WriteString("None.\n")
			return
		}
		b.WriteString(header)
		for _, l := range lines {
			b.WriteString(l + "\n")
		}
	}
	section("Rows added", "| Row | Results |\n| --- | --- |\n", added)
	section("Rows removed", "| Row | Results |\n| --- | --- |\n", removed)
	section("Rows changed", "| Row | Before | After |\n| --- | --- | --- |\n", changed)
	return b.String()
}

func orNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
