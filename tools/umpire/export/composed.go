package export

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/check"
)

// composedExport is one exported composition as its part of a dump is read back: its members in
// member order, each with the index of its machine among the export's, and its syncs.
type composedExport struct {
	decl    *umpirespb.Composition
	members []composedMember
	syncs   []composedSync
}

type composedMember struct {
	field   string
	machine int
}

// composedSync is one sync: its name, and each side's member and action name.
type composedSync struct {
	name          string
	first, second int
	firstAction   string
	secondAction  string
}

// compositions writes every composition the reader builds, and lists the ones it does not build with
// the reader's reason: a composition past a ceiling of the scope, and one whose member does not refine what
// it replaces. A replacement that holds is the reader's verdict: the module holds the composed table and
// no refinement.
func (q *quint) compositions() (fields, views []string, err error) {
	for _, decl := range q.s.Model.GetCompositions() {
		name := decl.GetName()
		left := Receipt{Backend: quintBackend, Claim: TransitionAgreement, Subject: name, Kind: Unsupported}
		c, err := q.s.bound.Composition(name)
		var limit *check.ComposeLimitError
		var rejected *check.RefinementError
		switch {
		case errors.As(err, &limit):
			left.Kind = ResourceLimit
			left.Explanation = fmt.Sprintf("the composition %s is not exported: goir builds no table of it within the scope (%v), and a part of one would read as the whole", name, err)
		case errors.As(err, &rejected):
			left.Explanation = fmt.Sprintf("the composition %s is not exported: goir rejects the replacement in it and builds no composed table (%v); the rejection is goir's own receipt", name, err)
		case err != nil:
			return nil, nil, err
		case len(c.Table.Unknown) > 0:
			return nil, nil, q.unsupported(decl.GetPosition(), "the composition %s, whose pair '%s' a member's hole leaves unknown: it is neither a result nor a disabled pair, and the module has no value for it",
				name, c.Table.Unknown[0].Row)
		default:
		}
		if err != nil {
			q.x.Unsupported = append(q.x.Unsupported, left)
			continue
		}
		j := len(q.x.Compositions)
		typ, err := q.composition(j, c)
		if err != nil {
			return nil, nil, err
		}
		q.x.Compositions = append(q.x.Compositions, name)
		fields = append(fields, fmt.Sprintf("c%d: %s", j, typ))
		views = append(views, fmt.Sprintf("c%d: c%d_view", j, j))
		for _, mb := range decl.GetMembers() {
			if mb.GetReplaces() == "" {
				continue
			}
			q.x.Unsupported = append(q.x.Unsupported, Receipt{Backend: quintBackend, Claim: ModuleRefinement, Subject: name, Kind: Unsupported,
				Explanation: fmt.Sprintf("that %s stands in for %s in %s is goir's verdict, which holds: Quint is given the composed table, and no refinement is exported or claimed of it",
					mb.GetMachine(), mb.GetReplaces(), name)})
		}
	}
	return fields, views, nil
}

// isBinding writes whether the class `k` of machine i is of its step binding b.
func (q *quint) isBinding(i, b int) string {
	tag := fmt.Sprintf("K%d_%d", i, b)
	if len(q.x.bindings[i][b].GetInputs()) == 0 {
		return "(k == " + tag + ")"
	}
	return fmt.Sprintf("match k { | %s(_) => true | _ => false }", tag)
}

// bindingsOf is the step bindings of machine i that are of the action of one name.
func (q *quint) bindingsOf(i int, action string) []int {
	var out []int
	for b, a := range q.x.bindings[i] {
		if a.GetName() == action {
			out = append(out, b)
		}
	}
	return out
}

// spelled writes the function that spells a member's outcome or fact as the composed key a claim
// reads: `<field>_<key>`. Quint builds no string, so every member of the type is spelled out.
func (q *quint) spelled(name, field, typ string) error {
	members, err := q.s.in.Members(named(typ))
	if err != nil {
		return err
	}
	chain := "List().head()"
	for i := len(members) - 1; i >= 0; i-- {
		written, err := q.literal(literal(members[i]), nil)
		if err != nil {
			return err
		}
		chain = fmt.Sprintf("if (v == %s) %q\n    else %s", written, field+"_"+members[i].Key(), chain)
	}
	fmt.Fprintf(&q.out, "  pure def %s(v) = %s\n", name, chain)
	return nil
}

// composition writes one composition over the machines the module already holds: a class is a
// member's own class, of an action no sync takes, or the pair of two members' classes a sync takes
// together; a step of a pair is the product of the members' results, the first member's first, with
// the first member's outcome and both members' facts; a claim reads a step's state as the
// composition's state record and its outcome and facts as strings. It gives the type of the
// composition's part of the dump.
func (q *quint) composition(j int, c *check.Composed) (string, error) {
	decl := c.Decl
	state, err := q.typeRef(named(decl.GetStateType()))
	if err != nil {
		return "", err
	}
	x := &composedExport{decl: decl}
	index := func(field string) int {
		return slices.IndexFunc(x.members, func(m composedMember) bool { return m.field == field })
	}
	for _, mb := range decl.GetMembers() {
		x.members = append(x.members, composedMember{field: mb.GetField(), machine: slices.Index(q.x.Machines, mb.GetMachine())})
	}
	for _, sync := range decl.GetSyncs() {
		first, second := sync.GetFirst(), sync.GetSecond()
		x.syncs = append(x.syncs, composedSync{name: sync.GetName(), first: index(first.GetMember()), second: index(second.GetMember()),
			firstAction: first.GetAction(), secondAction: second.GetAction()})
	}
	// The composition's definitions are prefixed c<j> and its class variants C<j>: the two share one
	// namespace.
	p := fmt.Sprintf("c%d", j)
	w := &q.out
	fmt.Fprintf(w, "\n  // The composition %s.\n", decl.GetName())
	step := func(m int, result string) (string, error) {
		mm := q.s.machines[q.x.Machines[x.members[m].machine]]
		if err := q.spelled(fmt.Sprintf("%s_o%d", p, m), x.members[m].field, mm.Decl.GetOutcomeType()); err != nil {
			return "", err
		}
		if mm.Decl.GetFactType() == "" {
			return "[]", nil
		}
		if err := q.spelled(fmt.Sprintf("%s_x%d", p, m), x.members[m].field, mm.Decl.GetFactType()); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.f_facts.foldl([], (fs, f) => fs.append(%s_x%d(f)))", result, p, m), nil
	}
	variants, classes, arms, err := q.composedClasses(j, x, step)
	if err != nil {
		return "", err
	}
	starts, err := q.composedStarts(x)
	if err != nil {
		return "", err
	}
	ends, err := q.ends(decl.GetEnds(), decl.GetName())
	if err != nil {
		return "", err
	}
	fmt.Fprintf(w, "  type C%d =\n    | %s\n", j, strings.Join(variants, "\n    | "))
	fmt.Fprintf(w, "  pure val %s_classes: Set[C%d] = %s\n", p, j, strings.Join(classes, ".union(")+strings.Repeat(")", len(classes)-1))
	fmt.Fprintf(w, "  pure def %s_step(s, c) = match c {\n    %s\n  }\n", p, strings.Join(arms, "\n    "))
	fmt.Fprintf(w, "  pure val %s_starts = [%s]\n", p, strings.Join(starts, ", "))
	fmt.Fprintf(w, "  pure def %s_ends(s) = %s\n", p, ends)
	fmt.Fprintf(w, "  pure def %s_succ(s) = %s_classes.fold(Set(), (acc, c) => %s_step(s, c).foldl(acc, (a, r) => a.union(Set(r.f_state))))\n", p, p, p)
	q.reach(p+"_bfs", "Set("+strings.Join(starts, ", ")+")", p+"_succ", depth(c.Table.Starts, func(s string) []string {
		var out []string
		for _, row := range c.Table.RowsFrom(s) {
			for _, r := range row.Results {
				out = append(out, r.State)
			}
		}
		return out
	})+1)
	typ := fmt.Sprintf("starts: List[%s], reach: Set[%s], closed: bool, ends: Set[%s], classes: Set[C%d], rows: Set[{src: %s, by: Set[{cls: C%d, steps: List[{f_outcome: str, f_state: %s, f_facts: List[str], f_because: str, f_choice: str}]}]}]",
		state, state, state, j, state, j, state)
	view := fmt.Sprintf("starts: %s_starts, reach: %s_bfs.seen, closed: %s_bfs.frontier == Set(), ends: %s_bfs.seen.filter(s => %s_ends(s)), classes: %s_classes,\n"+
		"    rows: %s_bfs.seen.map(s => {src: s, by: %s_classes.map(c => {cls: c, steps: %s_step(s, c)})})", p, p, p, p, p, p, p, p, p)
	if properties := q.s.properties(decl.GetName()); len(properties) > 0 {
		claimsType, err := q.claims(p, fmt.Sprintf("C%d", j), properties, state, func(pr *umpirespb.Property) (string, error) { return q.composedAbout(j, x, pr) })
		if err != nil {
			return "", err
		}
		typ += ", claims: " + claimsType
		view += fmt.Sprintf(",\n    claims: %s_bfs.seen.map(s => {src: s, by: %s_classes.map(c => {cls: c, steps: %s_step(s, c).foldl([], (l, r) => l.append(%s_claims(s, c, r)))})})", p, p, p, p)
	}
	fmt.Fprintf(w, "  pure val %s_view = {%s}\n", p, view)
	q.x.composed = append(q.x.composed, x)
	return "{" + typ + "}", nil
}

// composedClasses writes a composition's classes: for each member the variant, the set and the step
// of its own classes, the ones of no action a sync takes, and for each sync the same of the pairs it
// takes. facts writes a member's spelling functions and gives the expression of its result's facts.
func (q *quint) composedClasses(j int, x *composedExport, facts func(m int, result string) (string, error)) (variants, classes, arms []string, err error) {
	p, tag := fmt.Sprintf("c%d", j), fmt.Sprintf("C%d", j)
	synced := map[[2]string]bool{}
	for _, sync := range x.syncs {
		synced[[2]string{x.members[sync.first].field, sync.firstAction}], synced[[2]string{x.members[sync.second].field, sync.secondAction}] = true, true
	}
	written := make([]string, len(x.members))
	for m, mb := range x.members {
		if written[m], err = facts(m, fmt.Sprintf("r%d", m)); err != nil {
			return nil, nil, nil, err
		}
		// A member's own classes are the ones of no action a sync takes.
		var taken []string
		for b, a := range q.x.bindings[mb.machine] {
			if synced[[2]string{mb.field, a.GetName()}] {
				taken = append(taken, q.isBinding(mb.machine, b))
			}
		}
		own := fmt.Sprintf("m%d_classes", mb.machine)
		if len(taken) == len(q.x.bindings[mb.machine]) {
			continue
		}
		if len(taken) > 0 {
			own += fmt.Sprintf(".filter(k => not(%s))", strings.Join(taken, " or "))
		}
		variants = append(variants, fmt.Sprintf("%s_o%d(K%d)", tag, m, mb.machine))
		classes = append(classes, fmt.Sprintf("%s.map(k => %s_o%d(k))", own, tag, m))
		arms = append(arms, fmt.Sprintf("| %s_o%d(k) => m%d_step(s.f_%s, k).foldl([], (l, r%d) => l.append({f_outcome: %s_o%d(r%d.f_outcome), f_state: {...s, f_%s: r%d.f_state}, f_facts: %s, f_because: \"\", f_choice: \"\"}))",
			tag, m, mb.machine, plain(mb.field), m, p, m, m, plain(mb.field), m, written[m]))
	}
	for n, sync := range x.syncs {
		a, b := x.members[sync.first], x.members[sync.second]
		variants = append(variants, fmt.Sprintf("%s_s%d({a: K%d, b: K%d})", tag, n, a.machine, b.machine))
		classes = append(classes, fmt.Sprintf("tuples(%s, %s).map(t => %s_s%d({a: t._1, b: t._2}))", q.classesOf(a, sync.firstAction), q.classesOf(b, sync.secondAction), tag, n))
		arms = append(arms, fmt.Sprintf("| %s_s%d(k) => m%d_step(s.f_%s, k.a).foldl([], (l, r%d) => m%d_step(s.f_%s, k.b).foldl(l, (l2, r%d) =>\n"+
			"      l2.append({f_outcome: %s_o%d(r%d.f_outcome), f_state: {...s, f_%s: r%d.f_state, f_%s: r%d.f_state}, f_facts: %s.concat(%s), f_because: \"\", f_choice: \"\"})))",
			tag, n, a.machine, plain(a.field), sync.first, b.machine, plain(b.field), sync.second,
			p, sync.first, sync.first, plain(a.field), sync.first, plain(b.field), sync.second, written[sync.first], written[sync.second]))
	}
	return variants, classes, arms, nil
}

// classesOf writes the set of a member's classes of the action of one name.
func (q *quint) classesOf(mb composedMember, action string) string {
	var is []string
	for _, bound := range q.bindingsOf(mb.machine, action) {
		is = append(is, q.isBinding(mb.machine, bound))
	}
	if len(is) == 0 {
		return "Set()"
	}
	return fmt.Sprintf("m%d_classes.filter(k => %s)", mb.machine, strings.Join(is, " or "))
}

// composedStarts is every start of a composition: the product of its members' starts in member
// order, the last member varying fastest.
func (q *quint) composedStarts(x *composedExport) ([]string, error) {
	out := []string{""}
	for _, mb := range x.members {
		starts, err := q.exprs(q.s.machines[q.x.Machines[mb.machine]].Decl.GetStarts(), scope{})
		if err != nil {
			return nil, err
		}
		var next []string
		for _, prefix := range out {
			for _, start := range starts {
				next = append(next, fmt.Sprintf("%sf_%s: %s, ", prefix, plain(mb.field), start))
			}
		}
		out = next
	}
	for i, fields := range out {
		out[i] = "{" + strings.TrimSuffix(fields, ", ") + "}"
	}
	return out, nil
}

// composedAbout writes whether a Property of a composition is about the step of the class `c`:
// every step, or the steps of every class whose key begins with one name, a sync's or
// `<field>_<action>` for a member's own action.
func (q *quint) composedAbout(j int, x *composedExport, p *umpirespb.Property) (string, error) {
	switch w := p.GetWhen().(type) {
	case nil:
		return "true", nil
	case *umpirespb.Property_WhenAction:
		for n, sync := range x.syncs {
			if sync.name == w.WhenAction {
				return fmt.Sprintf("match c { | C%d_s%d(_) => true | _ => false }", j, n), nil
			}
		}
		for m, mb := range x.members {
			action, own := strings.CutPrefix(w.WhenAction, mb.field+"_")
			if bound := q.bindingsOf(mb.machine, action); own && len(bound) > 0 {
				is := make([]string, len(bound))
				for i, b := range bound {
					is[i] = q.isBinding(mb.machine, b)
				}
				return fmt.Sprintf("match c { | C%d_o%d(k) => %s | _ => false }", j, m, strings.Join(is, " or ")), nil
			}
		}
		return "", q.unsupported(p.GetPosition(), "the Property %s, about an action %s has no class of", p.GetName(), x.decl.GetName())
	default:
		return "", q.unsupported(p.GetPosition(), "the Property %s of the composition %s, about one composed class: only a composition's Properties about every step or about an action are exported",
			p.GetName(), x.decl.GetName())
	}
}

// composed reads composition j's part of a dump back as keys.
func (x *QuintExport) composedReader(j int) reader {
	return reader{x: x, index: j, composed: x.composed[j]}
}

// composedState is the key of a composed state: its members' state keys in member order, joined by
// "_".
func (r reader) composedState(raw any) (string, error) {
	c := r.composed
	v, err := r.x.value(raw, named(c.decl.GetStateType()))
	if err != nil {
		return "", err
	}
	fields := r.x.from.types[c.decl.GetStateType()].GetRecord().GetFields()
	parts := make([]string, len(c.members))
	for m, mb := range c.members {
		k := slices.IndexFunc(fields, func(f *umpirespb.Field) bool { return f.GetName() == mb.field })
		if k < 0 {
			return "", fmt.Errorf("no member fills the field %s", mb.field)
		}
		parts[m] = v.Fields[k].Key()
	}
	return strings.Join(parts, "_"), nil
}

// composedClass is the key of a composed class: `<field>_<class>` for a member's own class, and for a
// sync its name followed by each class's inputs.
func (r reader) composedClass(raw any) (string, error) {
	c := r.composed
	tagged, err := record(raw, "a composed class")
	if err != nil {
		return "", err
	}
	tag := text(tagged["tag"])
	var kind byte
	var n int
	if _, err := fmt.Sscanf(strings.TrimPrefix(tag, fmt.Sprintf("C%d_", r.index)), "%c%d", &kind, &n); err != nil {
		return "", fmt.Errorf("%q is no class of the composition", tag)
	}
	switch {
	case kind == 'o' && n >= 0 && n < len(c.members):
		key, err := r.x.reader(c.members[n].machine).class(tagged["value"])
		return c.members[n].field + "_" + key, err
	case kind == 's' && n >= 0 && n < len(c.syncs):
		sync := c.syncs[n]
		pair, err := record(tagged["value"], "a sync's classes")
		if err != nil {
			return "", err
		}
		first, err := r.x.reader(c.members[sync.first].machine).class(pair["a"])
		if err != nil {
			return "", err
		}
		second, err := r.x.reader(c.members[sync.second].machine).class(pair["b"])
		return sync.name + strings.TrimPrefix(first, sync.firstAction) + strings.TrimPrefix(second, sync.secondAction), err
	default:
		return "", fmt.Errorf("%q is no class of the composition", tag)
	}
}

// composedResult reads one composed step record: its state, and its outcome and facts as the strings
// a claim reads.
func (r reader) composedResult(raw any) (result, error) {
	fields, err := record(raw, "a composed step")
	if err != nil {
		return result{}, err
	}
	target, err := r.composedState(fields["f_state"])
	if err != nil {
		return result{}, err
	}
	facts, err := each(fields["f_facts"], func(raw any) (string, error) {
		f, ok := raw.(string)
		if !ok {
			return "", fmt.Errorf("the composed fact %v is no string", raw)
		}
		return f, nil
	})
	return result{Outcome: text(fields["f_outcome"]), State: target, Facts: facts, Because: text(fields["f_because"]), Choice: text(fields["f_choice"])}, err
}
