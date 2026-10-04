package lint

// The API values a realization tests: the enum values and oneof members of the fields its polls wait
// on and its Run Event guards select by. A field is in scope only where a condition tests it, so a
// request field a command writes, a message an observation keeps whole and a field no condition reads
// are not counted, and neither is a field of a Run Event's payload, the run's own record of a
// command. A history read's attributes oneof is never counted: the read sees every event of
// the workflow, and the IR does not say which of them a machine's operations produce.

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// apiTest is one test of one value: an `equal` of an enum field with one of its values, or a
// `present` of one member of a oneof. A test maps its value when its evidence kind records a fact.
type apiTest struct {
	field apiField
	value string
	maps  bool
	at    string
}

// apiField is a tested enum field or oneof: its full name, the name a message gives it, and every
// value it counts.
type apiField struct {
	name   string
	label  string
	values []string
	oneof  bool
}

func unmodeledAPIValues(m *Model) ([]Tally, error) {
	tests := map[string][]apiTest{}
	for _, r := range m.IR.GetRealizations() {
		owner := r.GetMachine()
		found, err := m.apiTests(r)
		if err != nil {
			return nil, fmt.Errorf("realization %s: %w", r.GetId(), err)
		}
		tests[owner] = append(tests[owner], found...)
	}
	history, err := m.historyOneofs(tests)
	if err != nil {
		return nil, err
	}
	var out []Tally
	for _, owner := range slices.Sorted(maps.Keys(tests)) {
		kept := slices.DeleteFunc(tests[owner], func(t apiTest) bool { return history[t.field.name] })
		out = append(out, apiTally(owner, kept))
	}
	return out, nil
}

// historyOneofs is the oneof of every member a history kind of evidence lifts, which is the history
// attributes oneof whatever its message is called. A history kind is never scanned, but a read of
// the history's events is an ordinary read whose poll waits for one member of that oneof, and the
// oneof counts no more there. It is looked up only where some test names a oneof.
func (m *Model) historyOneofs(tests map[string][]apiTest) (map[string]bool, error) {
	out := map[string]bool{}
	oneofs := false
	for _, owned := range tests {
		oneofs = oneofs || slices.ContainsFunc(owned, func(t apiTest) bool { return t.field.oneof })
	}
	if !oneofs {
		return out, nil
	}
	for _, r := range m.IR.GetRealizations() {
		for _, e := range r.GetEvidence() {
			if e.GetHistory() == "" {
				continue
			}
			element, err := m.lowering.Element(e)
			if err != nil {
				return nil, fmt.Errorf("evidence %s: %w", e.GetId(), err)
			}
			if member := element.Fields().ByName(protoreflect.Name(e.GetHistory())); member != nil && member.ContainingOneof() != nil {
				out[string(member.ContainingOneof().FullName())] = true
			}
		}
	}
	return out, nil
}

// apiTally is one owner's reading: every value of every field its tests name, and a finding for each
// value no test maps. A finding sits at the first test of its field.
func apiTally(owner string, tests []apiTest) Tally {
	t := Tally{Kind: UnmodeledAPIValue, Owner: owner}
	fields := map[string]apiField{}
	first := map[string]string{}
	mapped := map[[2]string]bool{}
	for _, test := range tests {
		if _, seen := fields[test.field.name]; !seen {
			fields[test.field.name], first[test.field.name] = test.field, test.at
		}
		if test.maps {
			mapped[[2]string{test.field.name, test.value}] = true
		}
	}
	for _, name := range slices.Sorted(maps.Keys(fields)) {
		f := fields[name]
		t.Population += len(f.values)
		for _, v := range f.values {
			if mapped[[2]string{name, v}] {
				continue
			}
			t.Findings = append(t.Findings, Finding{Kind: UnmodeledAPIValue, Owner: owner, Subject: name + " " + v,
				Message: fmt.Sprintf("%s %s is mapped to a fact by no poll or guard", f.label, v), Position: first[name]})
		}
	}
	return t
}

// apiTests is every test a realization's polls and Run Event guards make, in script order and then
// in evidence order.
func (m *Model) apiTests(r *umpirespb.Realization) ([]apiTest, error) {
	facts := map[string]bool{}
	if machine := m.Machines[r.GetMachine()]; machine != nil && machine.Table != nil {
		for _, line := range machine.Table.Evidence {
			facts[line[1]] = true
		}
	}
	evidence := map[string]*umpirespb.Evidence{}
	for _, e := range r.GetEvidence() {
		evidence[e.GetId()] = e
	}
	var out []apiTest
	for _, c := range commands(r) {
		poll := c.GetPoll()
		if poll == nil {
			continue
		}
		e := evidence[poll.GetEvidence()]
		if e == nil {
			return nil, fmt.Errorf("command %s polls evidence %s, which the realization does not declare", c.GetId(), poll.GetEvidence())
		}
		tests, err := m.apiScanOf(e, c.GetPosition(), poll.GetUntil(), facts[e.GetRecords()])
		if err != nil {
			return nil, err
		}
		out = append(out, tests...)
	}
	for _, e := range r.GetEvidence() {
		tests, err := m.apiScanOf(e, e.GetPosition(), e.GetRunEvent().GetGuard(), facts[e.GetRecords()])
		if err != nil {
			return nil, err
		}
		out = append(out, tests...)
	}
	return out, nil
}

// apiScanOf is the tests one condition over a kind of evidence makes; records is whether the kind
// records a fact.
func (m *Model) apiScanOf(e *umpirespb.Evidence, at *umpirespb.Position, condition *umpirespb.Operand, records bool) ([]apiTest, error) {
	// A history kind lifts one member of the attributes oneof, and no condition of it is counted.
	if condition == nil || e.GetHistory() != "" {
		return nil, nil
	}
	if m.lowering.Element == nil || m.lowering.Field == nil {
		return nil, fmt.Errorf("evidence %s: lint reads its fields through lowering's descriptors, and was given none", e.GetId())
	}
	element, err := m.lowering.Element(e)
	if err != nil {
		return nil, fmt.Errorf("evidence %s: %w", e.GetId(), err)
	}
	s := &apiScan{field: m.lowering.Field, element: element, maps: records}
	// A Run Event is the run's own record of a command, whose payload's vocabulary is the driver's:
	// a guard counts a value of the realized system's messages it carries, never of the record's own.
	if e.GetRunEvent() != nil {
		s.own = element.ParentFile().Package()
	}
	if err := s.operand(at, condition); err != nil {
		return nil, fmt.Errorf("evidence %s: %w", e.GetId(), err)
	}
	return s.tests, nil
}

// apiScan reads the tests of one condition over one evidence element.
type apiScan struct {
	field   func(at *umpirespb.Position, md protoreflect.MessageDescriptor, path string) (protoreflect.FieldDescriptor, error)
	element protoreflect.MessageDescriptor
	// own is the package whose fields are no test of the system: the Run Event payload's, or none.
	own   protoreflect.FullName
	maps  bool
	tests []apiTest
}

// operand collects the tests of a condition and of every condition inside it. Every other comparison
// is no test of a value: a text, a number, or a value the run computes says nothing of the field's
// domain.
func (s *apiScan) operand(at *umpirespb.Position, o *umpirespb.Operand) error {
	if o.GetPosition().GetFile() != "" {
		at = o.GetPosition()
	}
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Equal:
		if err := s.equal(at, k.Equal.GetLeft(), k.Equal.GetRight()); err != nil {
			return err
		}
		if err := s.equal(at, k.Equal.GetRight(), k.Equal.GetLeft()); err != nil {
			return err
		}
		return s.each(at, k.Equal.GetLeft(), k.Equal.GetRight())
	case *umpirespb.Operand_Present:
		if err := s.present(at, k.Present.GetOf()); err != nil {
			return err
		}
		return s.operand(at, k.Present.GetOf())
	case *umpirespb.Operand_Path:
		return s.operand(at, k.Path.GetOf())
	case *umpirespb.Operand_All:
		return s.each(at, k.All.GetOperands()...)
	case *umpirespb.Operand_Greater:
		return s.each(at, k.Greater.GetLeft(), k.Greater.GetRight())
	case *umpirespb.Operand_Not:
		return s.operand(at, k.Not.GetOf())
	default:
		return nil
	}
}

func (s *apiScan) each(at *umpirespb.Position, operands ...*umpirespb.Operand) error {
	for _, o := range operands {
		if err := s.operand(at, o); err != nil {
			return err
		}
	}
	return nil
}

// equal is the test `path == value` makes, where path reads the element and value is an enum value
// of the field it reaches.
func (s *apiScan) equal(at *umpirespb.Position, path, value *umpirespb.Operand) error {
	reads, ok := apiProjectedPath(path)
	name := value.GetLiteral().GetEnumName()
	if !ok || name == "" {
		return nil
	}
	fd, err := s.field(at, s.element, reads)
	if err != nil {
		return err
	}
	if fd.Kind() != protoreflect.EnumKind || fd.Enum().Values().ByName(protoreflect.Name(name)) == nil || s.own != "" && fd.ParentFile().Package() == s.own {
		return nil
	}
	var values []string
	for i := range fd.Enum().Values().Len() {
		// The zero value is the field unset, which no Model fact is.
		if v := fd.Enum().Values().Get(i); v.Number() != 0 {
			values = append(values, string(v.Name()))
		}
	}
	s.tests = append(s.tests, apiTest{field: apiField{name: string(fd.FullName()), label: apiLabel(fd.ContainingMessage(), fd.Name()), values: values},
		value: name, maps: s.maps, at: where(at)})
	return nil
}

// present is the test `present(path)` makes, where path ends at one member of a oneof. A synthetic
// oneof, which only gives a proto3 optional field presence, has one member and so no domain.
func (s *apiScan) present(at *umpirespb.Position, path *umpirespb.Operand) error {
	reads, ok := apiProjectedPath(path)
	if !ok {
		return nil
	}
	last := reads[strings.LastIndex(reads, ".")+1:]
	if !strings.HasSuffix(last, ">") || !strings.Contains(last, "<") {
		return nil
	}
	fd, err := s.field(at, s.element, reads)
	if err != nil {
		return err
	}
	oneof := fd.ContainingOneof()
	if oneof == nil || oneof.IsSynthetic() || s.own != "" && oneof.ParentFile().Package() == s.own {
		return nil
	}
	var members []string
	for i := range oneof.Fields().Len() {
		members = append(members, string(oneof.Fields().Get(i).Name()))
	}
	s.tests = append(s.tests, apiTest{field: apiField{name: string(oneof.FullName()), label: apiLabel(oneof.Parent(), oneof.Name()), values: members, oneof: true},
		value: string(fd.Name()), maps: s.maps, at: where(at)})
	return nil
}

// apiProjectedPath is the one dotted path a chain of paths reads out of the element, or false when the
// operand reads something else.
func apiProjectedPath(o *umpirespb.Operand) (string, bool) {
	p := o.GetPath()
	if p == nil {
		return "", false
	}
	if p.GetOf().GetProjected() != nil {
		return p.GetPath(), true
	}
	of, ok := apiProjectedPath(p.GetOf())
	if !ok {
		return "", false
	}
	return of + "." + p.GetPath(), true
}

// apiLabel names a field or oneof by its message's own name, as an author reads it.
func apiLabel(parent protoreflect.Descriptor, name protoreflect.Name) string {
	return fmt.Sprintf("%s.%s", parent.Name(), name)
}
