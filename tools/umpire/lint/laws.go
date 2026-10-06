package lint

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"

	"go.temporal.io/server/tools/umpire/check"
)

// The law kinds read the law sidecar beside an IR file (model.LawSidecar), never the IR or Scala
// text: what a declaration waives and why, which parameters a law has each instance cite and what
// each binding cites, and the catalog's instantiating machines, each at the Scala position the
// sidecar records. They name no capability and no law: both are the sidecar's data.

// CatalogOwner is the owner of a finding about the catalog rather than one machine.
const CatalogOwner = "catalog"

// Instances is each law's instantiating entities across the law sidecars a run reads: the state
// types of its instantiating machines, each with the least machine name a sidecar gives it, since a
// machine derived from another shares its state type and counts once.
type Instances map[string]map[string]string

// CountInstances counts the instantiating entities of every law the sidecars list.
func CountInstances(sidecars ...*check.LawSidecar) Instances {
	out := Instances{}
	for _, s := range sidecars {
		if s == nil {
			continue
		}
		for _, e := range s.Catalog {
			states := out[e.Law]
			if states == nil {
				states = map[string]string{}
				out[e.Law] = states
			}
			for _, x := range e.Instantiating {
				if was, ok := states[x.State]; !ok || x.Machine < was {
					states[x.State] = x.Machine
				}
			}
		}
	}
	return out
}

// machines is a law's instantiating machines, one per state type, sorted.
func (in Instances) machines(law string) []string {
	return slices.Sorted(maps.Values(in[law]))
}

// waiverSubject is what a finding about a waiver is about, and what its acceptance names:
// `<machine>.<law>`, the name the waived law's claim has or would have.
func waiverSubject(w check.LawWaiver) string { return w.Machine + "." + w.Law }

// forwarded reports whether a waiver's reason is forwarded into the accepted findings: it gives one,
// and names a law the catalog brings. Another is a finding of its own kind, and its reason is none.
func forwarded(s *check.LawSidecar, w check.LawWaiver) bool {
	return strings.TrimSpace(w.Because) != "" && s.Entry(w.Law) != nil
}

// waivedLaws (WaivedLaw) is each law a declaration waives with a reason, which the accepted findings
// carry forwarded from the sidecar (Forward), so the reason is read in one place with every other
// accepted finding, and a waiver the sidecar no longer records leaves a stale acceptance.
func waivedLaws(m *Model) ([]Tally, error) {
	return waiverTallies(m, WaivedLaw, func(s *check.LawSidecar, w check.LawWaiver) (bool, string) {
		if w.Waiver == check.WaiverOverriding {
			return !forwarded(s, w), fmt.Sprintf("%s is overridden by %s", w.Law, w.By)
		}
		return !forwarded(s, w), fmt.Sprintf("%s is waived with except", w.Law)
	})
}

// lawsWaivedWithoutReason (LawWaivedWithoutReason) is a waiver whose reason is empty, which the
// lifter refuses, so a sidecar holding one was not written by it.
func lawsWaivedWithoutReason(m *Model) ([]Tally, error) {
	return waiverTallies(m, LawWaivedWithoutReason, func(_ *check.LawSidecar, w check.LawWaiver) (bool, string) {
		return strings.TrimSpace(w.Because) != "", fmt.Sprintf(
			"%s of %s gives no reason: say why %s differs from the law, citing the server code", w.Waiver, w.Law, w.Machine)
	})
}

// reasonsNamingNoLaw (ReasonNamesNoLaw) is a waiver, with its reason, of a law the sidecar's catalog
// does not list, so no capability the file declares brings it: the reason excuses nothing.
func reasonsNamingNoLaw(m *Model) ([]Tally, error) {
	return waiverTallies(m, ReasonNamesNoLaw, func(s *check.LawSidecar, w check.LawWaiver) (bool, string) {
		return strings.TrimSpace(w.Because) == "" || s.Entry(w.Law) != nil,
			fmt.Sprintf("because %q names %s, a law the catalog does not bring", w.Because, w.Law)
	})
}

// waiverTallies counts each waiver of the sidecar for its machine, satisfied or not as judge says,
// each finding keyed `<machine>.<law>` at the waiver's position.
func waiverTallies(m *Model, k Kind, judge func(*check.LawSidecar, check.LawWaiver) (bool, string)) ([]Tally, error) {
	if m.Laws == nil {
		return nil, nil
	}
	t := tally(k)
	for _, w := range m.Laws.Waivers {
		satisfied, message := judge(m.Laws, w)
		t.addAt(w.Machine, satisfied, waiverSubject(w), w.Position, "%s", message)
	}
	return t.list(), nil
}

// parametersWithoutCitation (ParameterWithoutCitation) is a binding of a parameter its law has each
// instance back with server code, which the machine binds without citing any: such a parameter is
// where entities differ on purpose, so the value one takes rests on its server's answer. One per
// machine and parameter, however many of its claims take it, at the first claim's position.
func parametersWithoutCitation(m *Model) ([]Tally, error) {
	if m.Laws == nil {
		return nil, nil
	}
	t := tally(ParameterWithoutCitation)
	seen := map[[2]string]bool{}
	for _, c := range m.Laws.Claims {
		for _, p := range m.Laws.Entry(c.Law).Parameters {
			if key := [2]string{c.Machine, p}; !seen[key] {
				seen[key] = true
				t.addAt(c.Machine, len(c.Cites[p]) > 0, p, c.Position,
					"%s = %s cites no server code, and %s has each instance cite it: bind cited(value, \"<server file>\")",
					p, cmp.Or(c.Bindings[p], "nothing"), c.Law)
			}
		}
	}
	return t.list(), nil
}

// lawsWithOneInstance (LawWithOneInstance) is a law of the file's catalog with fewer than two
// instantiating entities across every sidecar the run reads (Options.Instances, or the file's own
// sidecar alone): a law joins the catalog only once two machines with their own state types adopt
// it, the definition the catalog test and the lifter count by.
func lawsWithOneInstance(m *Model) ([]Tally, error) {
	if m.Laws == nil {
		return nil, nil
	}
	instances := m.options.Instances
	if instances == nil {
		instances = CountInstances(m.Laws)
	}
	t := tally(LawWithOneInstance)
	for _, e := range m.Laws.Catalog {
		machines := instances.machines(e.Law)
		by := "no machine"
		if len(machines) > 0 {
			by = strings.Join(machines, ", ")
		}
		t.addAt(CatalogOwner, len(machines) >= 2, e.Law, e.Position,
			"%s is instantiated by %s with its own state type: a law joins the catalog with two", e.Law, by)
	}
	return t.list(), nil
}

// Forward is the accepted findings with the law waivers of the sidecar forwarded into them: every
// acceptance of WaivedLaw replaced by one per waiver the sidecar records with a reason, of a law its
// catalog brings, keyed by `<machine>.<law>` and accepted for that reason, in the sidecar's order and
// after the acceptances an author wrote. The sidecar is the one source of these reasons; the model
// gate's update writes the result (umpire-lint --update), and a check fails while the file differs.
func Forward(a *Accepted, s *check.LawSidecar) *Accepted {
	out := &Accepted{Accepted: slices.DeleteFunc(slices.Clone(a.Accepted), func(x Acceptance) bool { return x.Kind == WaivedLaw })}
	if s == nil {
		return out
	}
	for _, w := range s.Waivers {
		if forwarded(s, w) {
			out.Accepted = append(out.Accepted, Acceptance{Kind: WaivedLaw, Owner: w.Machine, Subjects: []string{waiverSubject(w)}, Because: w.Because})
		}
	}
	return out
}
