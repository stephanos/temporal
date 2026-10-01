// Package testpilot lowers a find Query of an admitted IR Model into a Testpilot Case, through the
// realization the Model declares for the Query's machine.
//
// A Case is built here and nowhere else for a Model the IR carries: the front end declares the
// realization, and this package places a Query's witness in it. It decides nothing about the feature.
// Which commands a Case carries follows from the classes the Scenario pins; what its Contract requires
// follows from the Property; what a declaration needs that Testpilot cannot run yet is reported, with
// the task that owns it, instead of a Case.
package testpilot

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/go/umpire"
	"go.temporal.io/server/model/scalav2/goir"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Standing is what lowering a Query came to.
type Standing string

const (
	// Lowered is a find Query whose witness became a Case.
	Lowered Standing = "lowered"
	// NothingToRealize is a verify Query: it is searched and verified, and realizes nothing.
	NothingToRealize Standing = "nothing-to-realize"
	// NoRealization is a find Query whose machine declares no realization.
	NoRealization Standing = "no-realization"
	// NotSupported is a find Query whose realization declares what Testpilot cannot run yet.
	NotSupported Standing = "unsupported"
)

// Unsupported is one declaration of a realization that Testpilot has no primitive for: what it is,
// where it was written, and the task that owns the primitive.
type Unsupported struct {
	Construct string
	ID        string
	Position  string
	Owner     string
	Why       string
}

// The tasks of fn-107 that own a Testpilot primitive this package has nothing to lower into.
const (
	ownerControls   = "fn-107.10"
	ownerAssessment = "fn-107.12"
	ownerActivities = "fn-107.13"
)

// Disposition is what became of one declaration of a realization in one Case.
type Disposition string

const (
	// InCase is a declaration the Case carries, in the parts As names.
	InCase Disposition = "in-case"
	// OffPath is a command the Query's path does not perform, or a kind of evidence no step of it
	// records.
	OffPath Disposition = "off-path"
	// Names is what identifies the realization in the IR, which no part of a Case repeats.
	Names Disposition = "names"
)

// Entry is one declaration of a realization and what became of it. Kind is the field of the
// realization that declares it, `realization` for a field of the realization itself, or `command`;
// ID is the declaration's id, or the field's name. As names the parts of the Case that carry it.
type Entry struct {
	Kind        string
	ID          string
	Position    string
	Disposition Disposition
	As          []string
}

// Lowering is what lowering one Query came to: its standing, the Case of a lowered one with every
// declaration accounted for, and the gaps of an unsupported one.
type Lowering struct {
	Standing    Standing
	Case        *testpilotspb.Case
	Unsupported []Unsupported
	Inventory   []Entry
}

// Producer lowers the Queries of one admitted, checked Model.
type Producer struct {
	realizer *goir.Realizer
	// found is each Query's receipt by its name, which a Model gives one Query.
	found map[string]goir.Receipt
	// realizations is the realizations the Model declares.
	realizations []*modelirspb.Realization
}

// NewProducer admits a Model, binds it and answers its Queries once, for every Case lowered from it.
func NewProducer(m *modelirspb.Model) (*Producer, error) {
	realizer, err := goir.NewRealizer(m, goir.DefaultScope)
	if err != nil {
		return nil, err
	}
	p := &Producer{realizer: realizer, found: map[string]goir.Receipt{}, realizations: realizer.Realizations()}
	for _, r := range goir.Check(m, goir.DefaultScope).Receipts {
		if r.Subject == goir.QuerySubject {
			p.found[r.Key.Name] = r
		}
	}
	return p, nil
}

// asked is one Query with the declarations it names.
type asked struct {
	key      goir.ClaimKey
	q        *modelirspb.Query
	property *modelirspb.Property
	scenario *modelirspb.Scenario
	r        *modelirspb.Realization
}

// ask finds a Query and what it is before anything of it is checked: a verify Query, a find Query
// with no realization, or a find Query with the one realization of its machine. The Query is the one
// Check gave a receipt, resolved by that receipt's key.
func (p *Producer) ask(query string) (*asked, Standing, error) {
	receipt, ok := p.found[query]
	if !ok {
		// The Realizer's refusal names the Model the Query is missing from.
		if _, err := p.realizer.Declared(goir.ClaimKey{Name: query}); err != nil {
			return nil, "", err
		}
		return nil, "", &goir.Error{Message: "no Query " + query}
	}
	declared, err := p.realizer.Declared(receipt.Key)
	if err != nil {
		return nil, "", err
	}
	a := &asked{key: receipt.Key, q: declared.Query, property: declared.Property, scenario: declared.Scenario}
	if a.q.GetForm() != modelirspb.Query_FORM_FIND {
		return a, NothingToRealize, nil
	}
	var realizations []*modelirspb.Realization
	for _, r := range p.realizations {
		if r.GetMachine() == a.scenario.GetMachine() {
			realizations = append(realizations, r)
		}
	}
	switch len(realizations) {
	case 0:
		return a, NoRealization, nil
	case 1:
		a.r = realizations[0]
		return a, Lowered, nil
	default:
		return nil, "", errorAt(a.q.GetPosition(), "query %s runs on %s, which %d realizations run; a Query is lowered through one",
			query, a.scenario.GetMachine(), len(realizations))
	}
}

// standingOf is the one place the standing of a Query is decided. An error of the realization or of
// the Query's path comes first, every one of them, whatever else is true of the Query. Then what
// Testpilot cannot run: a gap is reported only of a realization that is otherwise sound. Then what
// the Query is: one that realizes nothing or has no realization has that standing, and any other is
// lowered.
func standingOf(problems []error, gaps []Unsupported, realizable Standing) (Standing, error) {
	switch {
	case len(problems) > 0:
		return "", errors.Join(problems...)
	case len(gaps) > 0:
		return NotSupported, nil
	default:
		return realizable, nil
	}
}

// Lower lowers one Query under a Case identity. A Query that realizes nothing and one whose machine
// declares no realization have that standing and no Case. One whose realization needs what Testpilot
// cannot run lists every such declaration and has no Case: nothing is lowered around a gap. The
// realization and the Query's path are checked whole before either is said.
func (p *Producer) Lower(query string, identity cp.Identity) (*Lowering, error) {
	a, realizable, err := p.ask(query)
	if err != nil {
		return nil, err
	}
	var l *lowering
	var problems []error
	var gaps []Unsupported
	if realizable == Lowered {
		l, problems = p.check(a, identity)
		gaps = p.gaps(a.r)
	}
	standing, err := standingOf(problems, gaps, realizable)
	if err != nil {
		return nil, err
	}
	if standing != Lowered {
		return &Lowering{Standing: standing, Unsupported: gaps}, nil
	}
	produced, err := cp.Produce(l.query, identity, l.realization, p.source(a.q))
	if err != nil {
		return nil, errorAt(a.q.GetPosition(), "query %s: %v", query, err)
	}
	inventory, err := l.inventory(produced)
	if err != nil {
		return nil, err
	}
	return &Lowering{Standing: Lowered, Case: produced, Inventory: inventory}, nil
}

// gaps lists every declaration of a realization, and of the machine it runs, that Testpilot has no
// primitive for, in the order the IR lists them.
func (p *Producer) gaps(r *modelirspb.Realization) []Unsupported {
	var out []Unsupported
	var watching []*modelirspb.Monitor
	if mm := p.realizer.Machine(r.GetMachine()); mm != nil {
		watching = mm.Monitors
	}
	for _, mo := range watching {
		out = append(out, Unsupported{Construct: "authored monitor", ID: mo.GetName(), Position: locate(mo.GetPosition()), Owner: ownerAssessment,
			Why: "a Case's Contract carries no authored monitor: it is assessed beside the Contract through the prepared assessment seam"})
	}
	for _, e := range r.GetEvidence() {
		if e.GetCommitment() == modelirspb.Evidence_COMMITMENT_DURABLE {
			out = append(out, Unsupported{Construct: "durable-commit observation", ID: e.GetId(), Position: locate(e.GetPosition()), Owner: ownerControls,
				Why: "a Run records what an RPC returned and what history holds, and no durable commit of the receiver"})
		}
	}
	for _, c := range r.GetControls() {
		out = append(out, Unsupported{Construct: "hold-delivery control", ID: c.GetId(), Position: locate(c.GetPosition()), Owner: ownerControls,
			Why: "a Driver's faults are worker lifecycle transitions; none holds a delivery inside the server"})
	}
	for _, s := range r.GetScripts() {
		if s.GetActivity() != nil {
			out = append(out, Unsupported{Construct: "activity activation", ID: s.GetId(), Position: locate(s.GetPosition()), Owner: ownerActivities,
				Why: "no instruction is admitted in an activity entrypoint, and the worker Driver registers no activity"})
		}
		for _, c := range commandsOf(s) {
			switch c.GetInstruction().(type) {
			case *modelirspb.Command_Hold, *modelirspb.Command_Release:
				out = append(out, Unsupported{Construct: "hold-delivery command", ID: s.GetId() + "/" + c.GetId(), Position: locate(c.GetPosition()),
					Owner: ownerControls, Why: "no instruction holds or releases a delivery"})
			default:
			}
		}
	}
	return out
}

// commandsOf is every command a script declares, in declaration order.
func commandsOf(s *modelirspb.Script) []*modelirspb.Command {
	var out []*modelirspb.Command
	for _, item := range s.GetItems() {
		if item.GetCommand() != nil {
			out = append(out, item.GetCommand())
		}
		for _, p := range item.GetPerforms() {
			out = append(out, p.GetCommand())
		}
	}
	return out
}

// lowering is one Query checked and ready to be produced: the realization translated for the Case's
// identity, and the Query as the checker answers it.
type lowering struct {
	a           *asked
	mm          *goir.Machine
	adapter     *adapter
	realization *cp.Realization
	query       *umpire.Query
	keys        []string
}

func (p *Producer) source(q *modelirspb.Query) cp.Source {
	return cp.Source{Path: q.GetPosition().GetFile(), Provenance: "scala-model"}
}

// check reads a realization and a Query's path whole and emits nothing: what the realization writes
// against its descriptors, that a command performs every step a party takes, that the search found a
// witness, that the Property lowers to clauses, and, once the realization itself is sound, everything
// the producer decides before it writes a Case. It reports every problem it finds, and the lowering
// is ready to be produced when it finds none.
func (p *Producer) check(a *asked, identity cp.Identity) (*lowering, []error) {
	at, name := a.q.GetPosition(), a.q.GetName()
	mm := p.realizer.Machine(a.scenario.GetMachine())
	if a.scenario.GetFree() || len(a.scenario.GetKeys()) > 0 || mm == nil {
		return nil, []error{errorAt(at, "query %s runs %s, which pins no schedule of a machine's classes; only a pinned schedule is realized",
			name, a.scenario.GetName())}
	}
	if a.property.GetMachine() != a.scenario.GetMachine() || a.q.GetThrough() {
		return nil, []error{errorAt(at, "query %s reads a Property of %s on %s; a Case is lowered from a Property of the machine it runs", name,
			a.property.GetMachine(), a.scenario.GetMachine())}
	}
	query, err := p.realizer.Find(a.key)
	if err != nil {
		return nil, []error{fmt.Errorf("%s: query %s: %w", locate(at), name, err)}
	}
	table, err := query.Scenario.Machine.Table()
	if err != nil {
		return nil, []error{fmt.Errorf("%s: query %s: %w", locate(at), name, err)}
	}
	l := &lowering{a: a, mm: mm, query: query, keys: query.Scenario.Actions, adapter: newAdapter(a.r, table, p.realizer.ClassKey, identity.Fixture)}
	var problems []error
	l.realization, problems = l.adapter.realization()
	sound := len(problems) == 0
	if err := l.performed(); err != nil {
		problems = append(problems, err)
	}
	receipt, ok := p.found[name]
	witnessed := ok && receipt.Kind == goir.Found && receipt.Witness != nil
	if !witnessed {
		problems = append(problems, errorAt(at, "query %s has no witness to realize: its check is %s: %s", name, receipt.Kind, receipt.Explanation))
	}
	if _, err := query.Property.Lower(); err != nil {
		// The error keeps its cause: a predicate that reached a hole is told apart from one that is malformed.
		problems = append(problems, fmt.Errorf("%s: %w", locate(a.property.GetPosition()), err))
	} else if sound && witnessed {
		// A realization that crosses a descriptor is not all there for the producer to read, and a Query
		// with no witness or no clauses has nothing for it to decide; either is already reported.
		if err := cp.Preflight(query, identity, l.realization); err != nil {
			problems = append(problems, fmt.Errorf("%s: query %s: %w", locate(at), name, err))
		}
	}
	return l, problems
}

// performed rejects a path with a step a party takes that no command performs: a Case that does not
// drive it would wait for something nothing does. A step of the system needs no command.
func (l *lowering) performed() error {
	bound := map[string]bool{}
	for _, s := range l.a.r.GetScripts() {
		for _, item := range s.GetItems() {
			for _, performance := range item.GetPerforms() {
				bound[l.adapter.classKey(performance.GetStep())] = true
			}
		}
	}
	party := map[string]string{}
	for _, class := range l.mm.Classes {
		party[class.Key] = class.Action.GetParty()
	}
	var unperformed []error
	for _, key := range l.keys {
		if !bound[key] && party[key] != "system" {
			unperformed = append(unperformed, errorAt(l.a.scenario.GetPosition(), "scenario %s takes %s, a step of %s, and no script of realization %s performs it",
				l.a.scenario.GetName(), key, party[key], l.a.r.GetName()))
		}
	}
	return errors.Join(unperformed...)
}

// accounting is the inventory of one Case as it is taken: the entries so far, and the parts of the
// Case a declaration accounts for.
type accounting struct {
	l       *lowering
	c       *testpilotspb.Case
	entries []Entry
	owned   map[string]bool
	local   map[string]string
}

// realizationFields says how each field of a Realization is accounted for in a Case, and
// correlationFields each field of its Correlation. The inventory walks the two messages' descriptors,
// so a field the IR gains is an error of every lowering until it is accounted for here.
var realizationFields = map[protoreflect.Name]func(*accounting) error{
	"id":       func(a *accounting) error { return a.names("id") },
	"name":     func(a *accounting) error { return a.names("name") },
	"position": func(*accounting) error { return nil },
	"machine":  (*accounting).machine,
	"producer": func(a *accounting) error {
		return a.field("realization", "producer", a.l.a.r.GetProducer(), a.c.GetProvenance().GetProducerId(), "provenance.producer_id")
	},
	"producer_version": func(a *accounting) error {
		return a.field("realization", "producer_version", a.l.a.r.GetProducerVersion(), a.c.GetProvenance().GetProducerVersion(), "provenance.producer_version")
	},
	"roles":        (*accounting).roles,
	"learned":      (*accounting).learned,
	"observations": (*accounting).observations,
	"evidence":     (*accounting).evidence,
	"correlation":  (*accounting).correlation,
	"controls":     (*accounting).controls,
	"scripts":      (*accounting).scripts,
	"cleanup": func(a *accounting) error {
		return a.field("realization", "cleanup", a.l.a.r.GetCleanup(), a.c.GetProgram().GetCleanup().GetEntrypointId(), "program.cleanup")
	},
}

var correlationFields = map[protoreflect.Name]func(a *accounting, c *modelirspb.Correlation, contract *testpilotspb.CorrelatedContract) error{
	"position": func(*accounting, *modelirspb.Correlation, *testpilotspb.CorrelatedContract) error { return nil },
	"projection": func(a *accounting, c *modelirspb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		for _, rule := range a.c.GetProvenance().GetCorrelatedRules() {
			if rule.GetProjectionId() != c.GetProjection() {
				return a.differs("correlation", "projection", c.GetProjection(), rule.GetProjectionId(), "provenance.correlated_rules")
			}
		}
		return a.field("correlation", "projection", a.named(c.GetProjection()), contract.GetProjectionId(), "contract.correlated.projection_id")
	},
	"run": func(a *accounting, c *modelirspb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		for _, e := range a.c.GetProgram().GetEvidence() {
			if len(e.GetScope()) != 1 || e.GetScope()[0].GetFieldId() != a.named(c.GetRun()) {
				return a.differs("correlation", "run", a.named(c.GetRun()), fmt.Sprint(e.GetScope()), "program.evidence["+e.GetEvidenceId()+"]")
			}
		}
		return a.field("correlation", "run", a.named(c.GetRun()), strings.Join(contract.GetScopeFields(), ","), "contract.correlated.scope_fields")
	},
	"operation": func(a *accounting, c *modelirspb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		return a.field("correlation", "operation", a.named(c.GetOperation()), contract.GetOperationField(), "contract.correlated.operation_field")
	},
	"observation": func(a *accounting, c *modelirspb.Correlation, contract *testpilotspb.CorrelatedContract) error {
		return a.field("correlation", "observation", c.GetObservation(), contract.GetEvidenceObservationId(), "contract.correlated.evidence_observation_id")
	},
	"events":     (*accounting).window,
	"buffered":   (*accounting).window,
	"keys":       (*accounting).window,
	"support":    (*accounting).window,
	"work":       (*accounting).window,
	"event_size": (*accounting).window,
}

// derived is the parts of a Case no declaration of the realization accounts for, and what they are
// derived from instead.
var derived = map[string]string{
	"case_id":                                  "the Case's identity",
	"version":                                  "the Case format",
	"program.program_id":                       "the Case's identity",
	"contract.contract_id":                     "the Case's identity",
	"provenance.sources":                       "where the Query was written",
	"provenance.known_gaps":                    "the steps of the witness that record nothing",
	"provenance.correlated_rules":              "the Property's clauses",
	"provenance.local_names":                   "the Case's own names",
	"provenance.abstraction_claims":            "the examples of the machine's actions",
	"contract.correlated.initial_state":        "the Scenario's start",
	"contract.correlated.initial_state_fields": "the Scenario's start",
	"contract.correlated.transitions":          "the machine's rows",
	"contract.correlated.projection_rules":     "the witness and the machine's evidence",
	"contract.correlated.rules":                "the Property's clauses",
}

// inventory accounts for every declaration of the realization against the Case, and for every part of
// the Case against the declarations: a declaration the Case does not carry as declared, and a part of
// the Case that neither a declaration nor the Query accounts for, are errors of this package.
func (l *lowering) inventory(c *testpilotspb.Case) ([]Entry, error) {
	a := &accounting{l: l, c: c, owned: map[string]bool{}, local: map[string]string{}}
	for _, n := range c.GetProvenance().GetLocalNames() {
		a.local[n.GetDefinitionId()] = n.GetLocalName()
	}
	r := l.a.r.ProtoReflect()
	fields := r.Descriptor().Fields()
	for i := range fields.Len() {
		f := fields.Get(i)
		account, ok := realizationFields[f.Name()]
		if !ok {
			return nil, errorAt(l.a.r.GetPosition(), "a realization's %s has no place in the inventory of a Case", f.Name())
		}
		if !r.Has(f) && f.Name() != "cleanup" {
			continue
		}
		if err := account(a); err != nil {
			return nil, err
		}
	}
	if err := a.unaccounted(); err != nil {
		return nil, err
	}
	return a.entries, nil
}

// named is the Case-local name of a Definition ID.
func (a *accounting) named(id string) string {
	if name, ok := a.local[id]; ok {
		return name
	}
	return id
}

func (a *accounting) differs(kind, id, declared, carried, part string) error {
	return errorAt(a.l.a.r.GetPosition(), "%s %s of realization %s is %q, and the Case's %s carries %q", kind, id,
		a.l.a.r.GetName(), declared, part, carried)
}

// field records a declaration one part of the Case carries as the value it declares. A declaration
// left empty has no entry, and the part it would fill is then empty too.
func (a *accounting) field(kind, id, declared, carried, part string) error {
	if declared != carried {
		return a.differs(kind, id, declared, carried, part)
	}
	if declared == "" {
		return nil
	}
	a.own(kind, id, a.l.a.r.GetPosition(), part)
	return nil
}

func (a *accounting) own(kind, id string, at *modelirspb.Position, parts ...string) {
	for _, part := range parts {
		a.owned[part] = true
	}
	a.entries = append(a.entries, Entry{Kind: kind, ID: id, Position: locate(at), Disposition: InCase, As: parts})
}

// names records what identifies the realization in the IR, which no part of a Case repeats.
func (a *accounting) names(field string) error {
	a.entries = append(a.entries, Entry{Kind: "realization", ID: field, Position: locate(a.l.a.r.GetPosition()), Disposition: Names})
	return nil
}

// machine records the machine a realization runs as the target the Case's provenance binds.
func (a *accounting) machine() error {
	target := a.l.query.Scenario.Machine
	table, err := target.Table()
	if err != nil {
		return err
	}
	for _, d := range a.c.GetProvenance().GetDefinitions() {
		if d.GetKind() == testpilotspb.DEFINITION_KIND_TARGET {
			return a.field("realization", "machine", table.IDs().Target, d.GetDefinitionId(), "provenance.definitions")
		}
	}
	return a.differs("realization", "machine", table.IDs().Target, "", "provenance.definitions")
}

// carried records a declaration every Case carries under its own id.
func (a *accounting) carried(kind, id string, at *modelirspb.Position, part string, has bool) error {
	if !has {
		return errorAt(at, "%s %s of realization %s is in no part of the Case", kind, id, a.l.a.r.GetName())
	}
	a.own(kind, id, at, part)
	return nil
}

func (a *accounting) roles() error {
	for _, role := range a.l.a.r.GetRoles() {
		has := slices.ContainsFunc(a.c.GetProgram().GetRoles(), func(x *testpilotspb.Role) bool { return x.GetRoleId() == role.GetId() })
		if err := a.carried("roles", role.GetId(), role.GetPosition(), "program.roles["+role.GetId()+"]", has); err != nil {
			return err
		}
	}
	return nil
}

func (a *accounting) learned() error {
	for _, learned := range a.l.a.r.GetLearned() {
		has := slices.ContainsFunc(a.c.GetProgram().GetSlots(), func(x *testpilotspb.Slot) bool { return x.GetSlotId() == learned.GetId() })
		if err := a.carried("learned", learned.GetId(), learned.GetPosition(), "program.slots["+learned.GetId()+"]", has); err != nil {
			return err
		}
	}
	return nil
}

func (a *accounting) observations() error {
	for _, o := range a.l.a.r.GetObservations() {
		has := slices.ContainsFunc(a.c.GetProgram().GetObservations(), func(x *testpilotspb.Observation) bool { return x.GetObservationId() == o.GetId() })
		if err := a.carried("observations", o.GetId(), o.GetPosition(), "program.observations["+o.GetId()+"]", has); err != nil {
			return err
		}
	}
	return nil
}

// evidence records each kind of evidence under the Case-local name the Case declares it by, with the
// source the Contract counts it in, or as off the path where no step of the path records it.
func (a *accounting) evidence() error {
	sources := a.c.GetContract().GetCorrelated().GetSources()
	for _, e := range a.l.a.r.GetEvidence() {
		name := a.named(e.GetId())
		if !slices.ContainsFunc(a.c.GetProgram().GetEvidence(), func(x *testpilotspb.EvidenceDeclaration) bool { return x.GetEvidenceId() == name }) {
			a.entries = append(a.entries, Entry{Kind: "evidence", ID: e.GetId(), Position: locate(e.GetPosition()), Disposition: OffPath})
			continue
		}
		if !slices.Contains(sources, a.named(e.GetSource())) {
			return a.differs("evidence", e.GetId(), a.named(e.GetSource()), strings.Join(sources, ","), "contract.correlated.sources")
		}
		a.own("evidence", e.GetId(), e.GetPosition(), "program.evidence["+name+"]", "contract.correlated.sources")
	}
	return nil
}

// correlation records each field of the correlation, walking its descriptor as inventory walks the
// realization's.
func (a *accounting) correlation() error {
	c, contract := a.l.a.r.GetCorrelation(), a.c.GetContract().GetCorrelated()
	message := c.ProtoReflect()
	fields := message.Descriptor().Fields()
	for i := range fields.Len() {
		f := fields.Get(i)
		account, ok := correlationFields[f.Name()]
		if !ok {
			return errorAt(c.GetPosition(), "a correlation's %s has no place in the inventory of a Case", f.Name())
		}
		if !message.Has(f) {
			continue
		}
		before := len(a.entries)
		if err := account(a, c, contract); err != nil {
			return err
		}
		for i := before; i < len(a.entries); i++ {
			a.entries[i].ID, a.entries[i].Position = string(f.Name()), locate(c.GetPosition())
		}
	}
	return nil
}

// window records one bound of the window a check keeps. The Case carries the window in the
// fingerprint of its projection alone.
func (a *accounting) window(*modelirspb.Correlation, *testpilotspb.CorrelatedContract) error {
	if a.c.GetContract().GetCorrelated().GetProjectionFingerprint() == "" {
		return a.differs("correlation", "window", "a window", "", "contract.correlated.projection_fingerprint")
	}
	a.own("correlation", "", a.l.a.r.GetCorrelation().GetPosition(), "contract.correlated.projection_fingerprint")
	return nil
}

// controls refuses a control in a Case: Testpilot has no part that carries one, and a realization
// that declares one is not lowered.
func (a *accounting) controls() error {
	for _, c := range a.l.a.r.GetControls() {
		return errorAt(c.GetPosition(), "controls %s of realization %s is in no part of the Case", c.GetId(), a.l.a.r.GetName())
	}
	return nil
}

func (a *accounting) scripts() error {
	for _, s := range a.l.a.r.GetScripts() {
		if err := a.script(s); err != nil {
			return err
		}
	}
	return nil
}

// script records a script and each of its commands: a plain command under its id where the Case
// carries it, and a performance once per step of the path that takes its class.
func (a *accounting) script(s *modelirspb.Script) error {
	var entrypoint *testpilotspb.Entrypoint
	for _, e := range a.c.GetProgram().GetEntrypoints() {
		if e.GetEntrypointId() == s.GetId() {
			entrypoint = e
		}
	}
	if err := a.carried("scripts", s.GetId(), s.GetPosition(), "program.entrypoints["+s.GetId()+"]", entrypoint != nil); err != nil {
		return err
	}
	carries := func(id string) bool {
		return slices.ContainsFunc(entrypoint.GetInstructions(), func(n *testpilotspb.InstructionNode) bool { return n.GetInstructionId() == id })
	}
	part := func(id string) string { return "program.entrypoints[" + s.GetId() + "].instructions[" + id + "]" }
	for _, item := range s.GetItems() {
		if cmd := item.GetCommand(); cmd != nil {
			entry := Entry{Kind: "command", ID: s.GetId() + "/" + cmd.GetId(), Position: locate(cmd.GetPosition()), Disposition: OffPath}
			if carries(cmd.GetId()) {
				entry.Disposition, entry.As = InCase, []string{part(cmd.GetId())}
				a.owned[part(cmd.GetId())] = true
			}
			a.entries = append(a.entries, entry)
		}
		for _, performance := range item.GetPerforms() {
			if err := a.performance(s, performance, carries, part); err != nil {
				return err
			}
		}
	}
	return nil
}

// performance records the command of one class: once per step of the path that takes the class, a
// class taken again under its ordinal after the command's id.
func (a *accounting) performance(s *modelirspb.Script, performance *modelirspb.Performance, carries func(id string) bool,
	part func(id string) string) error {
	cmd := performance.GetCommand()
	key := a.l.adapter.classKey(performance.GetStep())
	entry := Entry{Kind: "command", ID: fmt.Sprintf("%s/%s [%s]", s.GetId(), cmd.GetId(), key), Position: locate(cmd.GetPosition()),
		Disposition: OffPath}
	for range slices.DeleteFunc(slices.Clone(a.l.keys), func(taken string) bool { return taken != key }) {
		id := cmd.GetId()
		if len(entry.As) > 0 {
			id = fmt.Sprintf("%s-%d", id, len(entry.As)+1)
		}
		if !carries(id) {
			return errorAt(cmd.GetPosition(), "the path performs %s, and the Case carries no %s/%s for it", key, s.GetId(), id)
		}
		entry.Disposition, entry.As = InCase, append(entry.As, part(id))
		a.owned[part(id)] = true
	}
	a.entries = append(a.entries, entry)
	return nil
}

// parts lists the parts of a Case: every populated field of the Case, its provenance, its Program and
// its Contract, a Program's repeated declarations each by its id, and an entrypoint's instructions
// each by its id. It walks the messages' descriptors, so a part the protocol gains is listed.
func parts(c *testpilotspb.Case) []string {
	var out []string
	var walk func(prefix string, m protoreflect.Message, depth int)
	walk = func(prefix string, m protoreflect.Message, depth int) {
		fields := m.Descriptor().Fields()
		for i := range fields.Len() {
			f := fields.Get(i)
			if !m.Has(f) {
				continue
			}
			path := prefix + string(f.Name())
			switch {
			case f.IsList() && f.Kind() == protoreflect.MessageKind && depth > 0 && strings.HasPrefix(path, "program."):
				list := m.Get(f).List()
				for j := range list.Len() {
					element := list.Get(j).Message()
					each := path + "[" + element.Get(element.Descriptor().Fields().ByNumber(1)).String() + "]"
					out = append(out, each)
					if instructions := element.Descriptor().Fields().ByName("instructions"); instructions != nil {
						nodes := element.Get(instructions).List()
						for k := range nodes.Len() {
							node := nodes.Get(k).Message()
							out = append(out, each+".instructions["+node.Get(node.Descriptor().Fields().ByNumber(1)).String()+"]")
						}
					}
				}
			case !f.IsList() && f.Kind() == protoreflect.MessageKind && depth < 2 && path != "version" && path != "program.cleanup":
				walk(path+".", m.Get(f).Message(), depth+1)
			default:
				out = append(out, path)
			}
		}
	}
	walk("", c.ProtoReflect(), 0)
	return out
}

// unaccounted is the first part of the Case that neither a declaration of the realization nor the
// Query accounts for.
func (a *accounting) unaccounted() error {
	for _, part := range parts(a.c) {
		if _, ok := derived[part]; !ok && !a.owned[part] {
			return errorAt(a.l.a.r.GetPosition(), "the Case of query %s carries %s, which no declaration of realization %s accounts for",
				a.l.a.q.GetName(), part, a.l.a.r.GetName())
		}
	}
	return nil
}
