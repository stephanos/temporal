package lint

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"
	"text/tabwriter"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/model"
)

// The law view reads the law sidecar beside an IR file onto the per-operation table: each law a
// machine is held to, with what it promises and does not, rendered as the modality it pins on the
// cells of its capabilities' actions, and the cells of those actions no law pins. A law pins cells
// the step function wrote; it never adds, removes or rewrites a row (.plans/MODALITIES.md). Like the
// law kinds, it names no capability and no law: both are the sidecar's data.

// Must is the modality a same-step claim pins on the results of the rows it names: a postcondition.
const Must Modality = "MUST"

// EveryClass stands for the classes of a law whose capabilities name no action: it pins its cells
// on every class alike.
const EveryClass = "every class"

// LawTable is the laws an owner is held to as the table view prints them: each generated claim, the
// cells of its capabilities' actions no law pins, and the laws it waives with `except`, which
// generate no claim. A composition's has no cells: it has no per-operation table of its own.
type LawTable struct {
	Owner    string
	Laws     []LawView
	Unpinned []LawCell
	Excepted []model.LawWaiver
}

// LawView is one generated claim of a law on the table: the claim and the catalog's entry for its
// law, the modality it pins (MUST NOT for a transition claim, MUST for a same-step one), the waiver
// whose def it states where it is overridden, and the cells it pins, rule by rule. Witnessed is a
// claim no verify Query asks, a functional law's find: a MUST on its path, not on every cell it
// names. Inherited is a claim of the product the machine refines, read through the refinement;
// Unchecked, one no Query over the machine's own Scenarios asks, which the refinement does not by
// itself preserve.
type LawView struct {
	Claim      model.LawClaim
	Entry      *model.LawEntry
	Modality   Modality
	Overridden *model.LawWaiver
	Witnessed  bool
	Inherited  bool
	Unchecked  bool
	Pins       []LawCell
}

// LawCell is one rule of the table as the law view reads it: its class, the capability fields that
// name that class, the rule's label, and its own modality and text.
type LawCell struct {
	Class    string
	Fields   []string
	Label    string
	Modality Modality
	Text     string
}

// cellOf is a rule as the law view reads it, with the capability fields that name its class.
func cellOf(r Rule, fields []string) LawCell {
	return LawCell{Class: r.Class, Fields: fields, Label: r.Label, Modality: r.Modality, Text: r.Text}
}

// compositionLaws is the law tables of the owners with no per-operation table, the compositions.
// A file with no sidecar has none.
func (m *Model) compositionLaws() []*LawTable {
	if m.Laws == nil {
		return nil
	}
	var out []*LawTable
	seen := map[string]bool{}
	for _, c := range m.Laws.Claims {
		if m.Machines[c.Machine] != nil || seen[c.Machine] {
			continue
		}
		seen[c.Machine] = true
		lt := &LawTable{Owner: c.Machine, Excepted: m.excepted(c.Machine)}
		for _, own := range m.Laws.Claims {
			if own.Machine == c.Machine {
				lt.Laws = append(lt.Laws, m.lawView(own))
			}
		}
		out = append(out, lt)
	}
	return out
}

// excepted is the waivers of an owner made with `except`.
func (m *Model) excepted(owner string) []model.LawWaiver {
	var out []model.LawWaiver
	for _, w := range m.Laws.Waivers {
		if w.Machine == owner && w.Waiver == model.WaiverExcept {
			out = append(out, w)
		}
	}
	return out
}

// lawView is a claim with its law's entry, the modality it pins and the waiver overriding it.
func (m *Model) lawView(c model.LawClaim) LawView {
	lv := LawView{Claim: c, Entry: m.Laws.Entry(c.Law), Modality: Must}
	for _, p := range m.IR.GetProperties() {
		if p.GetMachine() == c.Machine && p.GetName() == c.Name && p.GetTransition() {
			lv.Modality = MustNot
		}
	}
	lv.Witnessed = !slices.ContainsFunc(m.IR.GetQueries(), func(q *umpirespb.Query) bool {
		return q.GetForm() == umpirespb.Query_FORM_VERIFY && q.GetProperty().GetMachine() == c.Machine && q.GetProperty().GetName() == c.Name
	})
	for i, w := range m.Laws.Waivers {
		if w.Machine == c.Machine && w.Law == c.Law && w.Waiver == model.WaiverOverriding {
			lv.Overridden = &m.Laws.Waivers[i]
		}
	}
	return lv
}

// lawTable is the laws a machine is held to on its table: its own claims, then those of the product
// it refines, each with the rules it pins of its capabilities' actions; and the rules of those
// actions that no law pins.
func (v *view) lawTable(t *Table) *LawTable {
	if v.m.Laws == nil {
		return nil
	}
	var claims []model.LawClaim
	for _, c := range v.m.Laws.Claims {
		if c.Machine == v.name || v.product != nil && c.Machine == v.product.name {
			claims = append(claims, c)
		}
	}
	excepted := v.m.excepted(v.name)
	if len(claims) == 0 && len(excepted) == 0 {
		return nil
	}
	lt := &LawTable{Owner: v.name, Excepted: excepted}
	laws := map[string]bool{}
	for _, c := range claims {
		laws[c.Name] = true
	}
	// acting is the capability fields that name each class of the machine.
	acting := map[string][]string{}
	for _, c := range claims {
		lv := v.m.lawView(c)
		if c.Machine != v.name {
			lv.Inherited, lv.Unchecked = true, !v.asks(c.Machine, c.Name)
		}
		lv.Pins = v.pins(t, c, acting)
		lt.Laws = append(lt.Laws, lv)
	}
	for _, r := range t.Rules {
		fields, ok := acting[r.Class]
		if !ok || slices.ContainsFunc(r.Pinned, func(p string) bool { return laws[p] }) {
			continue
		}
		lt.Unpinned = append(lt.Unpinned, cellOf(r, fields))
	}
	return lt
}

// asks reports whether a Query over a Scenario of the machine asks the claim.
func (v *view) asks(machine, name string) bool {
	return slices.ContainsFunc(v.m.IR.GetQueries(), func(q *umpirespb.Query) bool {
		return q.GetScenario().GetMachine() == v.name && q.GetProperty().GetMachine() == machine && q.GetProperty().GetName() == name
	})
}

// pins is the rules a claim pins of the classes its capabilities' actions name, noting each class's
// fields in acting. A law whose capabilities name no action pins its cells on every class, read by
// label (everyClass).
func (v *view) pins(t *Table, c model.LawClaim, acting map[string][]string) []LawCell {
	classes := v.actionClasses(c, acting)
	var out []LawCell
	var labels []string
	byLabel := map[string][]string{}
	for _, r := range t.Rules {
		if !slices.Contains(r.Pinned, c.Name) {
			continue
		}
		if fields, ok := classes[r.Class]; ok {
			out = append(out, cellOf(r, fields))
		}
		if _, ok := byLabel[r.Label]; !ok {
			labels = append(labels, r.Label)
		}
		if !slices.Contains(byLabel[r.Label], r.Class) {
			byLabel[r.Label] = append(byLabel[r.Label], r.Class)
		}
	}
	if len(c.Actions) > 0 {
		return out
	}
	return v.everyClass(labels, byLabel)
}

// actionClasses is the capability fields naming each class of the machine a claim's actions name, a
// class by its key or an action with inputs by its name, each also noted in acting.
func (v *view) actionClasses(c model.LawClaim, acting map[string][]string) map[string][]string {
	classes := map[string][]string{}
	for _, field := range slices.Sorted(maps.Keys(c.Actions)) {
		for _, class := range v.mm.Classes {
			if class.Key != c.Actions[field] && class.Action.GetName() != c.Actions[field] {
				continue
			}
			classes[class.Key] = append(classes[class.Key], field)
			if !slices.Contains(acting[class.Key], field) {
				acting[class.Key] = append(acting[class.Key], field)
			}
		}
	}
	return classes
}

// everyClass is one line per label of the classes a law pins there: every class, every class but the
// few it does not pin, or the classes it does.
func (v *view) everyClass(labels []string, byLabel map[string][]string) []LawCell {
	out := make([]LawCell, 0, len(labels))
	for _, label := range labels {
		var missing []string
		for _, class := range v.mm.Classes {
			if !slices.Contains(byLabel[label], class.Key) {
				missing = append(missing, class.Key)
			}
		}
		named := EveryClass
		switch {
		case len(missing) == 0:
		case len(missing) < len(byLabel[label]):
			named += " but " + strings.Join(missing, ", ")
		default:
			named = strings.Join(byLabel[label], ", ")
		}
		out = append(out, LawCell{Class: named, Label: label})
	}
	return out
}

// heading names a law: its claim, its law, the capabilities that bring it, the modality it pins and
// whether the machine reads it through its refinement.
func (lv LawView) heading() string {
	s := fmt.Sprintf("%s  %s of %s, %s", lv.Claim.Name, lv.Claim.Law, strings.Join(lv.Claim.Capabilities, " and "), lv.Modality)
	if lv.Witnessed {
		s += " on its find's path only"
	}
	if lv.Inherited {
		s += ", inherited from " + lv.Claim.Machine
		if lv.Unchecked {
			s += ", unchecked"
		}
	}
	return s
}

// class names a cell's class with the capability fields that name it.
func (c LawCell) class() string {
	if len(c.Fields) == 0 {
		return c.Class
	}
	return c.Class + " (" + strings.Join(c.Fields, ", ") + ")"
}

// writeLawTable writes an owner's laws: each law (writeLaw), then the cells of the capabilities'
// actions no law pins, and the laws waived with `except`.
func writeLawTable(w io.Writer, file string, lt *LawTable, table bool) error {
	suffix := ""
	if !table {
		suffix = " (a composition: no per-operation table of its own)"
	}
	if err := writeLines(w, fmt.Sprintf("laws %s %s%s", file, lt.Owner, suffix)); err != nil {
		return err
	}
	for _, lv := range lt.Laws {
		if err := writeLaw(w, lv, table); err != nil {
			return err
		}
	}
	if len(lt.Unpinned) > 0 {
		if err := writeLines(w, "  no law pins"); err != nil {
			return err
		}
		if err := writeCells(w, lt.Unpinned, func(c LawCell) (string, string) { return string(c.Modality), c.Text }); err != nil {
			return err
		}
	}
	var excepted []string
	for _, x := range lt.Excepted {
		excepted = append(excepted, fmt.Sprintf("  %s.%s  excepted: %s  %s", x.Machine, x.Law, x.Because, x.Position))
	}
	return writeLines(w, excepted...)
}

// writeLaw writes one law: its claim, what it promises and does not, how it is overridden, and the
// cells it pins with the modality it pins there beside the cell's own.
func writeLaw(w io.Writer, lv LawView, table bool) error {
	lines := []string{fmt.Sprintf("  %s  %s", lv.heading(), lv.Claim.Position)}
	if lv.Entry != nil {
		lines = append(lines, "    promises: "+lv.Entry.Promises, "    does not promise: "+lv.Entry.DoesNotPromise)
	}
	if lv.Overridden != nil {
		lines = append(lines, fmt.Sprintf("    overridden by %s: %s", short(lv.Overridden.By), lv.Overridden.Because))
	}
	if table && len(lv.Pins) == 0 {
		lines = append(lines, "    pins no cell of its capabilities' actions")
	}
	if err := writeLines(w, lines...); err != nil {
		return err
	}
	return writeCells(w, lv.Pins, func(c LawCell) (string, string) {
		if c.Modality == "" {
			return string(lv.Modality), ""
		}
		// On a cell the step function permits, a transition law forbids a shape of its results, not
		// the class: unpausing a paused activity is a MAY whose results may not land in running.
		modality := string(lv.Modality)
		if lv.Modality == MustNot && c.Modality == May {
			modality += " of its results"
		}
		return modality, "cell: " + strings.TrimSpace(string(c.Modality)+" "+c.Text)
	})
}

func writeLines(w io.Writer, lines ...string) error {
	for _, l := range lines {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return err
		}
	}
	return nil
}

// writeCells writes one aligned line per cell: its class, its label, then the modality and text
// columns, the text left out where it is empty.
func writeCells(w io.Writer, cells []LawCell, columns func(LawCell) (modality, text string)) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, c := range cells {
		modality, text := columns(c)
		row := "    " + c.class() + "\t" + c.Label + "\t" + modality
		if text != "" {
			row += "\t" + text
		}
		if _, err := fmt.Fprintln(tw, row); err != nil {
			return err
		}
	}
	return tw.Flush()
}
