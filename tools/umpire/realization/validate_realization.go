// Package realization admits a Model's realizations, through the validator's Admitter, and types the
// guards and payloads a realization reads: what a producer of Cases needs of one beyond the Model.
package realization

import (
	"fmt"
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
	"go.temporal.io/server/tools/umpire/interp"
)

// Admitter is what admitting a realization asks of the validator of its Model: the realization's
// declarations are keyed, its problems reported and its classes checked as the rest of the Model's
// are, against the Model's machines and channels.
type Admitter interface {
	// Once reports a declaration whose key an earlier one of its kind took, and is whether it did.
	Once(at *umpirespb.Position, words ...string) bool
	// Report records a problem of the Model.
	Report(at *umpirespb.Position, format string, args ...any)
	// Errors is how many problems are recorded so far.
	Errors() int
	// ActionClass checks a class of an action against a machine that must bind it.
	ActionClass(owner string, mm *umpirespb.Machine, c *umpirespb.ActionClass, at *umpirespb.Position)
	// ClassKey keys a class as the machine's table keys it.
	ClassKey(c *umpirespb.ActionClass) string
	// Machine is the Model's machine of a name, if it declares one.
	Machine(name string) (*umpirespb.Machine, bool)
	// Channel is whether the Model declares a channel of an id.
	Channel(id string) bool
}

// realizing is one realization being admitted: what it declares, by id, and what its commands bind,
// read and perform.
type realizing struct {
	d Admitter
	r *umpirespb.Realization
	// label is what scopes the realization's declarations, and owner what a diagnostic calls it: its
	// name, its id where it has no name, or where it was written where it has neither.
	label        string
	owner        string
	roles        map[string]*umpirespb.Role
	learned      map[string]*umpirespb.Learned
	observations map[string]bool
	evidence     map[string]*umpirespb.Evidence
	controls     map[string]bool
	// bound and performed name the command that binds a learned value and that performs a class;
	// read holds the learned values some command reads.
	bound     map[string]string
	performed map[string]string
	read      map[string]bool
	// closed names the command whose read closes an exhaustive kind of evidence.
	closed map[string]string
}

var (
	roleKinds    = map[umpirespb.Role_Kind]string{umpirespb.Role_KIND_ENDPOINT: "an endpoint", umpirespb.Role_KIND_WORKER: "a worker", umpirespb.Role_KIND_TASK_QUEUE: "a task queue", umpirespb.Role_KIND_PARTICIPANT: "a participant"}
	learnedKinds = map[umpirespb.Learned_Kind]string{umpirespb.Learned_KIND_TEXT: "a text", umpirespb.Learned_KIND_HANDLE: "a handle"}
	fieldRoles   = map[umpirespb.EvidenceField_Role]string{umpirespb.EvidenceField_ROLE_OPERATION: "the operation", umpirespb.EvidenceField_ROLE_ATTEMPT: "the attempt", umpirespb.EvidenceField_ROLE_DELIVERY: "the delivery"}
)

// Admit checks that a realization names what it declares, declares each thing once, binds each
// learned value and performs each class once, crosses no kind and no correlation, and orders its
// commands without a cycle. It is named by its id and by its name, and one that lacks either, or
// names no machine of the Model, is still read whole: only its classes, which are read against the
// machine, are left unchecked.
func Admit(d Admitter, r *umpirespb.Realization) {
	at := r.GetPosition()
	a := &realizing{d: d, r: r, label: r.GetName(), owner: "realization " + r.GetName(), roles: map[string]*umpirespb.Role{},
		learned: map[string]*umpirespb.Learned{}, observations: map[string]bool{}, evidence: map[string]*umpirespb.Evidence{},
		controls: map[string]bool{}, bound: map[string]string{}, performed: map[string]string{}, read: map[string]bool{},
		closed: map[string]string{}}
	switch {
	case r.GetName() != "":
		d.Once(at, "realizations named", r.GetName())
	case r.GetId() != "":
		d.Report(at, "a realization has no name")
		a.label, a.owner = r.GetId(), "realization "+r.GetId()
	default:
		d.Report(at, "a realization has no name")
		a.label, a.owner = fmt.Sprintf("at %s", interp.Where(at)), "a realization"
	}
	if r.GetId() == "" {
		d.Report(at, "%s has no id", a.owner)
	} else {
		d.Once(at, "realizations with id", r.GetId())
	}
	mm, ok := d.Machine(r.GetMachine())
	if !ok {
		a.report(at, "no machine %s", r.GetMachine())
	}
	if r.GetProducer() == "" {
		a.report(at, "it names no producer")
	}
	a.rejectionCodes()
	a.declarations()
	a.requiredSettings()
	a.correlation()
	for _, s := range r.GetScripts() {
		a.script(mm, s)
	}
	for _, l := range r.GetLearned() {
		switch _, bound := a.bound[l.GetId()]; {
		case l.GetId() == "" || bound:
		case a.read[l.GetId()]:
			a.report(l.GetPosition(), "learned value %s is read and no command binds it", l.GetId())
		default:
			a.report(l.GetPosition(), "learned value %s is neither bound nor read", l.GetId())
		}
	}
	for _, e := range r.GetEvidence() {
		if _, closed := a.closed[e.GetId()]; e.GetExhaustive() && !closed {
			a.report(e.GetPosition(), "evidence %s is exhaustive and no command closes it", e.GetId())
		}
		a.recordedBy(e)
		a.attemptOf(e)
	}
	a.confirms(mm)
	a.held(mm)
	a.behavior(mm)
}

// attemptOf checks evidence that is the Run's record of an attempt: it names an attempt, counted from
// one, of a script an activity activates, whose activation is a delivery a path can take, and it is
// what a worker reports of an activation, which is how a Run records an attempt. What a worker reports
// of an activation is such a record whatever the realization says of it, so it names its attempt: a
// reader places it by that attempt, and has nothing else to place it by.
func (a *realizing) attemptOf(e *umpirespb.Evidence) {
	of := e.GetRunEvent().GetAttempt()
	if of == nil {
		if e.GetRunEvent().GetKind() == umpirespb.RunEventSource_KIND_DIAGNOSTIC {
			a.report(e.GetPosition(), "evidence %s is what a worker reports of an activation and is declared the record of no attempt: "+
				"a Run records it once the attempt is answered, and the realization says which attempt that is", e.GetId())
		}
		return
	}
	at := of.GetPosition()
	if at.GetFile() == "" {
		at = e.GetPosition()
	}
	if e.GetRunEvent().GetKind() != umpirespb.RunEventSource_KIND_DIAGNOSTIC {
		a.report(at, "evidence %s is the record of an attempt and of no diagnostic: a Run records an attempt as what a worker reports of an activation", e.GetId())
	}
	if of.GetScript() == "" {
		a.report(at, "evidence %s is the record of an attempt of no script", e.GetId())
		return
	}
	for _, s := range a.r.GetScripts() {
		if s.GetId() != of.GetScript() {
			continue
		}
		switch {
		case s.GetActivity() == nil:
			a.report(at, "evidence %s is the record of an attempt of script %s, which no activity activates", e.GetId(), s.GetId())
		case len(s.GetActivity().GetStarts()) == 0:
			a.report(at, "evidence %s is the record of an attempt of script %s, which starts with no delivery: no step of a path is an attempt of it", e.GetId(), s.GetId())
		case of.GetNumber() < 1:
			a.report(at, "evidence %s is the record of attempt %d of script %s; the attempts of an activity are counted from one", e.GetId(), of.GetNumber(), s.GetId())
		default:
		}
		return
	}
	a.report(at, "evidence %s is the record of an attempt of script %s, which the realization does not declare", e.GetId(), of.GetScript())
}

// confirms checks the steps of a path that kinds of evidence name as the ones they confirm: each is a
// step of a class the machine binds, counted from one, which its kind names once and no other kind
// names; and no kind that names steps is exhaustive, or records a fact an exhaustive kind records.
func (a *realizing) confirms(mm *umpirespb.Machine) {
	// An exhaustive kind reports every occurrence of the fact it records, so it is the one kind of
	// that fact: read with a second kind beside it, its silence would say nothing of the steps the
	// second confirms.
	exhaustive := map[string]string{}
	for _, e := range a.r.GetEvidence() {
		if e.GetExhaustive() && e.GetRecords() != "" {
			exhaustive[e.GetRecords()] = e.GetId()
		}
	}
	named := map[string]string{}
	for _, e := range a.r.GetEvidence() {
		switch reporting, reported := exhaustive[e.GetRecords()]; {
		case len(e.GetConfirms()) == 0:
		case e.GetExhaustive():
			a.report(e.GetPosition(), "evidence %s is exhaustive and names the steps it confirms; an exhaustive kind reports every step that records its fact", e.GetId())
		case reported:
			a.report(e.GetPosition(), "evidence %s records %s, every occurrence of which the exhaustive %s reports", e.GetId(), e.GetRecords(), reporting)
		default:
		}
		own := map[string]bool{}
		for _, taking := range e.GetConfirms() {
			at := taking.GetPosition()
			if at.GetFile() == "" {
				at = e.GetPosition()
			}
			if taking.GetStep() == nil {
				a.report(at, "evidence %s confirms a step of no class", e.GetId())
				continue
			}
			if mm == nil {
				continue
			}
			before := a.d.Errors()
			a.d.ActionClass(a.owner+": evidence "+e.GetId(), mm, taking.GetStep(), at)
			if a.d.Errors() != before {
				continue
			}
			step := fmt.Sprintf("step %d of class %s", taking.GetOccurrence(), a.d.ClassKey(taking.GetStep()))
			switch other, taken := named[step]; {
			case taking.GetOccurrence() < 1:
				a.report(at, "evidence %s confirms %s; the steps of a class on a path are counted from one", e.GetId(), step)
			case own[step]:
				a.report(at, "evidence %s confirms %s twice", e.GetId(), step)
			case taken:
				a.report(at, "evidence %s and %s both confirm %s; one kind confirms a step", other, e.GetId(), step)
			default:
				named[step], own[step] = e.GetId(), true
			}
		}
	}
}

// recordedBy checks that a Run Event's evidence names a command of a script the realization declares.
func (a *realizing) recordedBy(e *umpirespb.Evidence) {
	source := e.GetRunEvent()
	if source == nil {
		return
	}
	for _, s := range a.r.GetScripts() {
		if s.GetId() != source.GetScript() || s.GetId() == "" {
			continue
		}
		for _, item := range s.GetItems() {
			if item.GetCommand() != nil && item.GetCommand().GetId() == source.GetCommand() {
				return
			}
			for _, p := range item.GetPerforms() {
				if p.GetCommand() != nil && p.GetCommand().GetId() == source.GetCommand() {
					return
				}
			}
		}
		a.report(e.GetPosition(), "evidence %s: no command %s of script %s", e.GetId(), source.GetCommand(), s.GetId())
		return
	}
	a.report(e.GetPosition(), "evidence %s: no script %s", e.GetId(), source.GetScript())
}

func (a *realizing) report(at *umpirespb.Position, format string, args ...any) {
	if at.GetFile() == "" {
		at = a.r.GetPosition()
	}
	a.d.Report(at, a.owner+": "+format, args...)
}

// declared reports a declaration with no id, and one whose id an earlier one of its kind took. It is
// whether the declaration can be named.
func (a *realizing) declared(at *umpirespb.Position, kind, plural, id string) bool {
	if id == "" {
		a.report(at, "%s has no id", kind)
		return false
	}
	if at.GetFile() == "" {
		at = a.r.GetPosition()
	}
	return !a.d.Once(at, plural+" with id", id, "of realization", a.label)
}

func (a *realizing) declarations() {
	for _, role := range a.r.GetRoles() {
		if a.declared(role.GetPosition(), "a role", "roles", role.GetId()) {
			a.roles[role.GetId()] = role
		}
		if !interp.Known(umpirespb.Role_Kind_name, int32(role.GetKind())) {
			a.report(role.GetPosition(), "role %s is of no known kind", role.GetId())
		}
	}
	for _, l := range a.r.GetLearned() {
		if a.declared(l.GetPosition(), "a learned value", "learned values", l.GetId()) {
			a.learned[l.GetId()] = l
		}
		if !interp.Known(umpirespb.Learned_Kind_name, int32(l.GetKind())) {
			a.report(l.GetPosition(), "learned value %s is of no known kind", l.GetId())
		}
	}
	for _, o := range a.r.GetObservations() {
		if a.declared(o.GetPosition(), "an observation", "observations", o.GetId()) {
			a.observations[o.GetId()] = true
		}
		if o.GetMessage() == "" {
			a.report(o.GetPosition(), "observation %s names no message", o.GetId())
		}
	}
	for _, e := range a.r.GetEvidence() {
		a.evidenceKind(e)
	}
	a.controlDeclarations()
}

// requiredSettings checks that each required setting names a key and a value, and that no key is
// required twice: a dynamic-configuration key is read case-insensitively, so neither is a respelling.
func (a *realizing) requiredSettings() {
	required := map[string]bool{}
	for _, s := range a.r.GetRequiredSettings() {
		key := strings.ToLower(s.GetKey())
		if s.GetKey() == "" || s.GetValue() == "" {
			a.report(a.r.GetPosition(), "a required setting names no key or no value")
		} else if required[key] {
			a.report(a.r.GetPosition(), "it requires setting %s twice", s.GetKey())
		}
		required[key] = true
	}
}

func (a *realizing) controlDeclarations() {
	for _, c := range a.r.GetControls() {
		if a.declared(c.GetPosition(), "a control", "controls", c.GetId()) {
			a.controls[c.GetId()] = true
		}
		switch k := c.GetKind().(type) {
		case *umpirespb.Control_HoldDelivery:
			if !a.d.Channel(k.HoldDelivery) {
				a.report(c.GetPosition(), "control %s: no channel %s", c.GetId(), k.HoldDelivery)
			}
		case *umpirespb.Control_HoldDispatched:
			// A run holds a dispatch through the deliveries of a queue, so the control names one.
			switch {
			case k.HoldDispatched.GetStep() == nil:
				a.report(c.GetPosition(), "control %s holds what a step of no class dispatches", c.GetId())
			case c.GetRole() == "":
				a.report(c.GetPosition(), "control %s holds deliveries and names no task-queue role", c.GetId())
			default:
				a.role(c.GetPosition(), "control "+c.GetId(), c.GetRole(), umpirespb.Role_KIND_TASK_QUEUE)
			}
		default:
			a.report(c.GetPosition(), "control %s is of no known kind", c.GetId())
		}
	}
}

// held checks that a control that holds what a step dispatches names a class the machine binds.
func (a *realizing) held(mm *umpirespb.Machine) {
	if mm == nil {
		return
	}
	for _, c := range a.r.GetControls() {
		if step := c.GetHoldDispatched().GetStep(); step != nil {
			at := c.GetPosition()
			if at.GetFile() == "" {
				at = a.r.GetPosition()
			}
			a.d.ActionClass(a.owner+": control "+c.GetId(), mm, step, at)
		}
	}
}

func (a *realizing) evidenceKind(e *umpirespb.Evidence) {
	at, id := e.GetPosition(), e.GetId()
	if a.declared(at, "a kind of evidence", "kinds of evidence", id) {
		a.evidence[id] = e
	}
	switch {
	case e.GetRecords() == "":
		a.report(at, "evidence %s names no recorded kind", id)
	case len(e.GetConfirms()) == 0:
		// One kind confirms the step of a path that records a fact. Kinds that name the steps they
		// confirm are told apart by those steps, and may record one fact beside it.
		a.d.Once(at, "kinds of evidence recording", e.GetRecords(), "of realization", a.label)
	default:
	}
	if e.GetSource() == "" {
		a.report(at, "evidence %s names no source", id)
	}
	switch {
	case e.GetRunEvent() != nil && e.GetOperation() != "":
		a.report(at, "evidence %s names a field that keys its operation, and a Run Event's key is its source's", id)
	case e.GetRunEvent() == nil && e.GetOperation() == "":
		a.report(at, "evidence %s names no field that keys its operation", id)
	default:
	}
	switch from := e.GetFrom().(type) {
	case *umpirespb.Evidence_History:
		if from.History == "" {
			a.report(at, "evidence %s names no history event", id)
		}
	case *umpirespb.Evidence_Read:
		if from.Read.GetMethod() == "" || from.Read.GetPath() == "" {
			a.report(at, "evidence %s reads no method or no path", id)
		}
	case *umpirespb.Evidence_Single:
		if from.Single.GetMethod() == "" || from.Single.GetPath() == "" {
			a.report(at, "evidence %s reads no method or no path", id)
		}
	case *umpirespb.Evidence_RunEvent:
		a.runEvent(e, from.RunEvent)
	default:
		a.report(at, "evidence %s is recorded nowhere", id)
	}
	if !interp.Known(umpirespb.Evidence_Commitment_name, int32(e.GetCommitment())) {
		a.report(at, "evidence %s is of no known commitment", id)
	}
	a.evidenceFields(e)
}

// runEvent checks the Run's own record as a source of evidence: it is of a known kind, its key is the
// run's id or a path of the event's payload, and its guard reads the payload alone. The command it
// names is checked once the scripts are read.
func (a *realizing) runEvent(e *umpirespb.Evidence, source *umpirespb.RunEventSource) {
	at, id := e.GetPosition(), e.GetId()
	if !interp.Known(umpirespb.RunEventSource_Kind_name, int32(source.GetKind())) {
		a.report(at, "evidence %s is a Run Event of no known kind", id)
	}
	key := source.GetKey()
	if key.GetRun() == nil && (key.GetPath() == nil || key.GetPath().GetOf().GetProjected() == nil || key.GetPath().GetPath() == "") {
		a.report(at, "evidence %s: a Run Event's key is the run's id or a path of its payload", id)
	}
	if source.GetGuard() != nil {
		if err := GuardProblem(source.GetGuard(), nil); err != nil {
			a.report(at, "evidence %s: its guard %s", id, err)
		}
	}
}

// evidenceFields checks the fields a kind of evidence carries: each is named once and read from a
// path, and an identity is named by one field of the kind, which the evidence retains.
func (a *realizing) evidenceFields(e *umpirespb.Evidence) {
	declared, named := map[string]bool{}, map[umpirespb.EvidenceField_Role]string{}
	for _, f := range e.GetFields() {
		at := f.GetPosition()
		if at.GetFile() == "" {
			at = e.GetPosition()
		}
		switch {
		case f.GetId() == "":
			a.report(at, "evidence %s has a field with no id", e.GetId())
			continue
		case declared[f.GetId()]:
			a.report(at, "evidence %s declares field %s twice", e.GetId(), f.GetId())
			continue
		default:
			declared[f.GetId()] = true
		}
		if f.GetPath() == "" {
			a.report(at, "evidence %s: field %s names no path", e.GetId(), f.GetId())
		}
		role := f.GetRole()
		if role == umpirespb.EvidenceField_ROLE_UNSPECIFIED {
			continue
		}
		spelled, isKnown := fieldRoles[role]
		switch other, taken := named[role]; {
		case !isKnown:
			a.report(at, "evidence %s: field %s names an identity of no known role", e.GetId(), f.GetId())
		case f.GetRedacted():
			a.report(at, "evidence %s: field %s names %s and is redacted; an identity is read from a field the evidence retains", e.GetId(), f.GetId(), spelled)
		case taken:
			a.report(at, "evidence %s: fields %s and %s both name %s; one field of a kind names an identity", e.GetId(), other, f.GetId(), spelled)
		default:
			named[role] = f.GetId()
		}
	}
}

// correlation checks that evidence is keyed by two different fields and carried by one declared
// observation, within a window that keeps something.
func (a *realizing) correlation() {
	c := a.r.GetCorrelation()
	if c == nil {
		a.d.Report(a.r.GetPosition(), "%s declares no correlation", a.owner)
		return
	}
	at := c.GetPosition()
	if c.GetProjection() == "" || c.GetRun() == "" || c.GetOperation() == "" {
		a.report(at, "the correlation names no projection, no run field or no operation field")
	} else if c.GetRun() == c.GetOperation() {
		a.report(at, "the correlation keys its runs and its operations by one field, %s", c.GetRun())
	}
	if !a.observations[c.GetObservation()] {
		a.report(at, "the correlation: no observation %s", c.GetObservation())
	}
	for _, bound := range []struct {
		name string
		n    int64
	}{{"events", c.GetEvents()}, {"buffered", c.GetBuffered()}, {"keys", c.GetKeys()}, {"support", c.GetSupport()},
		{"work", c.GetWork()}, {"event size", c.GetEventSize()}} {
		if bound.n < 1 {
			a.report(at, "the correlation keeps %d %s; a window keeps at least one", bound.n, bound.name)
		}
	}
}

// commandOf is one command of a script with what a diagnostic calls it.
type commandOf struct {
	c    *umpirespb.Command
	name string
	at   *umpirespb.Position
	// always says every Case carries the command: it is no performance, and is under no condition.
	always bool
	// script is the id of the script the command is of.
	script string
}

func (a *realizing) script(mm *umpirespb.Machine, s *umpirespb.Script) {
	at := s.GetPosition()
	if !a.declared(at, "a script", "scripts", s.GetId()) && s.GetId() == "" {
		return
	}
	a.activation(mm, s)
	a.items(mm, s)
	fixed, all := map[string]bool{}, map[string]*umpirespb.Command{}
	var commands []commandOf
	note := func(c *umpirespb.Command, performs, always bool) {
		pos := c.GetPosition()
		if pos.GetFile() == "" {
			pos = at
		}
		if c.GetId() == "" {
			a.report(pos, "a command of script %s has no id", s.GetId())
			return
		}
		switch {
		case !performs:
			a.d.Once(pos, "commands with id", c.GetId(), "of script", s.GetId(), "of realization", a.label)
			fixed[c.GetId()] = true
		case fixed[c.GetId()]:
			a.report(pos, "two commands with id %s of script %s", c.GetId(), s.GetId())
		default:
		}
		all[c.GetId()] = c
		commands = append(commands, commandOf{c, fmt.Sprintf("%s of script %s", c.GetId(), s.GetId()), pos, always, s.GetId()})
	}
	for _, item := range s.GetItems() {
		if item.GetCommand() != nil && len(item.GetPerforms()) == 0 {
			note(item.GetCommand(), false, len(item.GetWhen()) == 0)
		}
	}
	for _, item := range s.GetItems() {
		for _, p := range item.GetPerforms() {
			if p.GetCommand() == nil {
				a.report(p.GetPosition(), "script %s performs a class with no command", s.GetId())
				continue
			}
			note(p.GetCommand(), true, false)
			a.performs(mm, s, p)
		}
	}
	for _, c := range commands {
		a.command(s, c, all)
	}
	a.cycles(s, all)
}

// items checks that each item of a script is a command, with the classes it is carried for, or the
// place steps are performed, and not both.
func (a *realizing) items(mm *umpirespb.Machine, s *umpirespb.Script) {
	for _, item := range s.GetItems() {
		pos := item.GetPosition()
		if pos.GetFile() == "" {
			pos = s.GetPosition()
		}
		switch {
		case item.GetCommand() == nil && len(item.GetPerforms()) == 0:
			a.report(pos, "script %s has an item that is neither a command nor the place steps are performed", s.GetId())
		case item.GetCommand() != nil && len(item.GetPerforms()) > 0:
			a.report(pos, "script %s has an item that is both a command and the place steps are performed", s.GetId())
		case item.GetCommand() != nil && mm != nil:
			if item.GetCommand().GetAttemptWithheld() != nil {
				a.withholding(s, item)
			}
			for _, c := range item.GetWhen() {
				a.d.ActionClass(a.owner+": script "+s.GetId(), mm, c, pos)
			}
		case item.GetCommand() != nil:
		case len(item.GetWhen()) > 0:
			a.report(pos, "script %s performs steps under a condition; the path decides which are performed", s.GetId())
		default:
		}
	}
}

// performs checks the class a performance binds, and that no other performance of the realization
// binds it.
func (a *realizing) performs(mm *umpirespb.Machine, s *umpirespb.Script, p *umpirespb.Performance) {
	if p.GetCommand().GetAttemptWithheld() != nil {
		a.report(p.GetPosition(), "command %s withholds an attempt under a performance: withholding is onPath of its server timer, not an action it performs", p.GetCommand().GetId())
	}
	a.performing(mm, "script "+s.GetId(), p.GetStep(), p.GetPosition(), fmt.Sprintf("%s of script %s", p.GetCommand().GetId(), s.GetId()))
}

// Withholding runs only for a path that takes its one armed, bounded server timer. It does not
// perform the timeout: the server does. The activity's context supplies the execution deadline.
func (a *realizing) withholding(s *umpirespb.Script, item *umpirespb.Item) {
	at := item.GetPosition()
	if len(item.GetWhen()) != 1 {
		a.report(at, "command %s withholds an attempt without exactly one armed bounded server timer", item.GetCommand().GetId())
		return
	}
	key := a.d.ClassKey(item.GetWhen()[0])
	for _, step := range a.r.GetServerSteps() {
		if a.d.ClassKey(step.GetStep()) == key && step.GetKind() == umpirespb.CAUSE_KIND_TIMER && step.GetDeadlineMs() > 0 {
			return
		}
	}
	a.report(at, "command %s withholds an attempt without an armed bounded server timer in script %s", item.GetCommand().GetId(), s.GetId())
}

// performing records what performs one class of the machine, a command or the activation of a script,
// and reports a class the machine does not bind and one something else performs already.
func (a *realizing) performing(mm *umpirespb.Machine, where string, class *umpirespb.ActionClass, at *umpirespb.Position, by string) {
	if mm == nil {
		return
	}
	before := a.d.Errors()
	a.d.ActionClass(a.owner+": "+where, mm, class, at)
	if a.d.Errors() != before {
		return
	}
	key := a.d.ClassKey(class)
	if other, ok := a.performed[key]; ok {
		a.report(at, "class %s is performed by %s and by %s; a class is performed once", key, other, by)
		return
	}
	a.performed[key] = by
}

// role checks that a role is declared and of the kind its use needs.
func (a *realizing) role(at *umpirespb.Position, where, id string, kind umpirespb.Role_Kind) {
	switch role, ok := a.roles[id]; {
	case !ok:
		a.report(at, "%s: no role %s", where, id)
	case role.GetKind() != kind && interp.Known(umpirespb.Role_Kind_name, int32(role.GetKind())):
		a.report(at, "%s: role %s is %s, not %s", where, id, roleKinds[role.GetKind()], roleKinds[kind])
	default:
	}
}

func (a *realizing) activation(mm *umpirespb.Machine, s *umpirespb.Script) {
	at, where := s.GetPosition(), "script "+s.GetId()
	worker := func(name *umpirespb.Name, workerRole, queue string) {
		if name != nil && name.GetPrefix() == "" && name.GetSuffix() == "" && !name.GetFixture() {
			a.report(at, "%s is activated by a type with no name", where)
		}
		a.role(at, where, workerRole, umpirespb.Role_KIND_WORKER)
		a.role(at, where, queue, umpirespb.Role_KIND_TASK_QUEUE)
	}
	switch act := s.GetActivation().(type) {
	case *umpirespb.Script_Controller:
	case *umpirespb.Script_Workflow:
		worker(act.Workflow.GetWorkflowType(), act.Workflow.GetWorker(), act.Workflow.GetTaskQueue())
	case *umpirespb.Script_Activity:
		worker(act.Activity.GetActivityType(), act.Activity.GetWorker(), act.Activity.GetTaskQueue())
		for _, class := range act.Activity.GetStarts() {
			a.performing(mm, where, class, at, activationOf+where)
		}
	case *umpirespb.Script_NexusHandler:
		if act.NexusHandler.GetService() == "" || act.NexusHandler.GetOperation() == "" {
			a.report(at, "%s answers no service or no operation", where)
		}
		worker(nil, act.NexusHandler.GetWorker(), act.NexusHandler.GetTaskQueue())
	default:
		a.report(at, "%s names no activation", where)
	}
}

// binds records the one command that binds a learned value of the kind the binding gives it.
func (a *realizing) binds(c commandOf, id string, kind umpirespb.Learned_Kind) {
	if !a.reads(c, id, kind) {
		return
	}
	if other, ok := a.bound[id]; ok {
		a.report(c.at, "learned value %s is bound by %s and by %s; a learned value is bound once", id, other, c.name)
		return
	}
	a.bound[id] = c.name
}

// reads checks that a learned value is declared and of the kind its reader needs, any kind when kind
// is unspecified, and is whether it is declared.
func (a *realizing) reads(c commandOf, id string, kind umpirespb.Learned_Kind) bool {
	l, ok := a.learned[id]
	switch {
	case !ok:
		a.report(c.at, "command %s: no learned value %s", c.name, id)
		return false
	case kind != umpirespb.Learned_KIND_UNSPECIFIED && l.GetKind() != kind && interp.Known(umpirespb.Learned_Kind_name, int32(l.GetKind())):
		a.report(c.at, "command %s: learned value %s is %s, not %s", c.name, id, learnedKinds[l.GetKind()], learnedKinds[kind])
	default:
	}
	return true
}

func (a *realizing) command(s *umpirespb.Script, c commandOf, all map[string]*umpirespb.Command) {
	named := func(id string) {
		if _, ok := all[id]; !ok {
			a.report(c.at, "command %s: no command %s of script %s", c.name, id, s.GetId())
		}
	}
	for _, id := range c.c.GetAfter().GetCommands() {
		named(id)
	}
	if c.c.GetTimeoutMs() < 0 {
		a.report(c.at, "command %s has a deadline of %d milliseconds", c.name, c.c.GetTimeoutMs())
	}
	for _, id := range c.c.GetCloses() {
		a.closes(c, id)
	}
	switch in := c.c.GetInstruction().(type) {
	case *umpirespb.Command_Rpc:
		a.rpc(c, in.Rpc)
	case *umpirespb.Command_Poll:
		a.poll(c, in.Poll)
	case *umpirespb.Command_AwaitLearned:
		if a.reads(c, in.AwaitLearned, umpirespb.Learned_KIND_UNSPECIFIED) {
			a.read[in.AwaitLearned] = true
		}
	case *umpirespb.Command_AwaitCommand:
		named(in.AwaitCommand)
	case *umpirespb.Command_Finish:
		a.typed(c, in.Finish.GetResult(), false)
	case *umpirespb.Command_Fault:
		a.role(c.at, "command "+c.name, in.Fault.GetRole(), umpirespb.Role_KIND_TASK_QUEUE)
		if !interp.Known(umpirespb.Fault_Kind_name, int32(in.Fault.GetKind())) {
			a.report(c.at, "command %s is a fault of no known kind", c.name)
		}
	case *umpirespb.Command_WorkflowCommand:
		a.message(c, in.WorkflowCommand.GetCommand())
	case *umpirespb.Command_NexusReply:
		a.message(c, in.NexusReply.GetReply())
		if in.NexusReply.GetBinds() != "" {
			a.binds(c, in.NexusReply.GetBinds(), umpirespb.Learned_KIND_HANDLE)
		}
	case *umpirespb.Command_NexusCompletion:
		if a.reads(c, in.NexusCompletion.GetHandle(), umpirespb.Learned_KIND_HANDLE) {
			a.read[in.NexusCompletion.GetHandle()] = true
		}
		a.message(c, in.NexusCompletion.GetResult())
	case *umpirespb.Command_Hold:
		a.control(c, in.Hold)
	case *umpirespb.Command_Release:
		a.control(c, in.Release)
	case *umpirespb.Command_AttemptFailure:
		if s.GetActivity() == nil {
			a.report(c.at, "command %s fails an attempt, and script %s is no activity's", c.name, s.GetId())
		}
		a.message(c, in.AttemptFailure.GetFailure())
	case *umpirespb.Command_AttemptCanceled:
		if s.GetActivity() == nil {
			a.report(c.at, "command %s cancels an attempt, and script %s is no activity's", c.name, s.GetId())
		}
	case *umpirespb.Command_AttemptWithheld:
		if s.GetActivity() == nil {
			a.report(c.at, "command %s withholds an attempt, and script %s is no activity's", c.name, s.GetId())
		}
	default:
		a.report(c.at, "command %s names no instruction", c.name)
	}
}

// closes checks that a command's read is the one closing read of an exhaustive kind of evidence: it
// reads the kind, and every Case carries it.
func (a *realizing) closes(c commandOf, id string) {
	e, declared := a.evidence[id]
	switch other, closed := a.closed[id]; {
	case !declared:
		a.report(c.at, "command %s closes evidence %s, which is not declared", c.name, id)
		return
	case !e.GetExhaustive():
		a.report(c.at, "command %s closes evidence %s, which is not exhaustive", c.name, id)
		return
	case closed:
		a.report(c.at, "evidence %s is closed by %s and by %s; an exhaustive kind has one closing read", id, other, c.name)
		return
	default:
		a.closed[id] = c.name
	}
	reads := c.c.GetPoll().GetEvidence() == id
	if e.GetHistory() != "" {
		reads = false
		for _, read := range c.c.GetRpc().GetReads() {
			for _, target := range read.GetTargets() {
				reads = reads || target.GetLift() != ""
			}
		}
	}
	// The Run's own record of a command is read by that command: what the command records is all
	// there is of the kind. A Case that does not carry the command has no record of it, and nothing
	// is inferred from a source that was never read, so such a command need not be in every Case.
	if record := e.GetRunEvent(); record != nil && record.GetScript() == c.script && record.GetCommand() == c.c.GetId() {
		return
	}
	if !reads {
		a.report(c.at, "command %s closes evidence %s and does not read it: a history kind is closed by the read that lifts it, the Run's own record of a command by that command, and any other by a poll of it", c.name, id)
	}
	if !c.always {
		a.report(c.at, "command %s closes evidence %s and is not a command every Case carries", c.name, id)
	}
}

func (a *realizing) control(c commandOf, id string) {
	if !a.controls[id] {
		a.report(c.at, "command %s: no control %s", c.name, id)
	}
}

func (a *realizing) assignments(c commandOf, assign []*umpirespb.Assignment, polls bool) {
	targets := map[string]bool{}
	for _, as := range assign {
		if as.GetTarget() == "" {
			a.report(c.at, "command %s assigns a value to no field", c.name)
		} else if targets[as.GetTarget()] {
			a.report(c.at, "command %s assigns %s twice", c.name, as.GetTarget())
		}
		targets[as.GetTarget()] = true
		a.typed(c, as.GetValue(), polls)
	}
}

// typed checks a value a command computes, and the types of what it computes it from, and is its
// shape.
func (a *realizing) typed(c commandOf, o *umpirespb.Operand, polls bool) Shape {
	a.operand(c, o, polls)
	computes, err := TypeOf(o, nil, nil)
	if err != nil {
		a.report(c.at, "command %s: %s", c.name, err)
	}
	return computes.Shape
}

func (a *realizing) rpc(c commandOf, rpc *umpirespb.Rpc) {
	a.role(c.at, "command "+c.name, rpc.GetRole(), umpirespb.Role_KIND_ENDPOINT)
	if rpc.GetMethod() == "" {
		a.report(c.at, "command %s calls no method", c.name)
	}
	a.assignments(c, rpc.GetAssign(), false)
	held := a.r.GetCorrelation().GetObservation()
	for _, read := range rpc.GetReads() {
		if !interp.Known(umpirespb.ResponseRead_Cardinality_name, int32(read.GetCardinality())) {
			a.report(c.at, "command %s reads %s at no known cardinality", c.name, read.GetPath())
		}
		for _, target := range read.GetTargets() {
			switch tg := target.GetTarget().(type) {
			case *umpirespb.Target_Observe:
				switch {
				case !a.observations[tg.Observe]:
					a.report(c.at, "command %s: no observation %s", c.name, tg.Observe)
				case tg.Observe == held:
					a.report(c.at, "command %s observes a value into %s, which holds the correlation's evidence", c.name, held)
				default:
				}
			case *umpirespb.Target_Bind:
				if read.GetCardinality() == umpirespb.ResponseRead_CARDINALITY_EACH {
					a.report(c.at, "command %s binds %s from each element of %s; a learned value is one value", c.name, tg.Bind, read.GetPath())
				}
				a.binds(c, tg.Bind, umpirespb.Learned_KIND_TEXT)
			case *umpirespb.Target_Lift:
				if tg.Lift != held {
					a.report(c.at, "command %s lifts evidence into %s, and the correlation reads %s", c.name, tg.Lift, held)
				}
			default:
				a.report(c.at, "command %s reads %s into nothing", c.name, read.GetPath())
			}
		}
	}
}

func (a *realizing) poll(c commandOf, poll *umpirespb.Poll) {
	a.role(c.at, "command "+c.name, poll.GetRole(), umpirespb.Role_KIND_ENDPOINT)
	switch e, ok := a.evidence[poll.GetEvidence()]; {
	case !ok:
		a.report(c.at, "command %s: no evidence %s", c.name, poll.GetEvidence())
	case e.GetRunEvent() != nil:
		a.report(c.at, "command %s: evidence %s is a Run Event, which a poll does not read", c.name, poll.GetEvidence())
	case e.GetRead() == nil && e.GetSingle() == nil && e.GetFrom() != nil:
		a.report(c.at, "command %s: evidence %s is a history event, which a poll does not read", c.name, poll.GetEvidence())
	default:
	}
	a.assignments(c, poll.GetAssign(), false)
	if computes := a.typed(c, poll.GetUntil(), true); computes != AnyShape && computes != ConditionShape {
		a.report(c.at, "command %s polls until %s, and a poll's condition is a condition", c.name, computes)
	}
	// A poll that writes no interval waits as the lowering derives from the API behavior; it then
	// writes no deadline either, since its bound is the hints' (.plans/API_BEHAVIOR_HINTS.md).
	switch {
	case poll.GetIntervalMs() < 0:
		a.report(c.at, "command %s polls every %d milliseconds", c.name, poll.GetIntervalMs())
	case poll.GetIntervalMs() == 0 && c.c.GetTimeoutMs() > 0:
		a.report(c.at, "command %s waits within the bound the API behavior derives, and writes a deadline of %d milliseconds besides",
			c.name, c.c.GetTimeoutMs())
	default:
	}
}

// operand checks a value a command computes. Only a poll's condition reads the value the poll is
// looking at.
func (a *realizing) operand(c commandOf, o *umpirespb.Operand, polls bool) {
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Literal:
		switch k.Literal.GetKind().(type) {
		case *umpirespb.ProtoValue_Text, *umpirespb.ProtoValue_Flag, *umpirespb.ProtoValue_Number,
			*umpirespb.ProtoValue_EnumName, *umpirespb.ProtoValue_Named:
		default:
			a.report(c.at, "command %s: a literal operand is a text, a flag, a number, an enum value or a name", c.name)
		}
	case *umpirespb.Operand_Environment:
		if k.Environment == "" {
			a.report(c.at, "command %s reads an environment binding with no id", c.name)
		}
	case *umpirespb.Operand_Run:
	case *umpirespb.Operand_LearnedValue:
		if a.reads(c, k.LearnedValue, umpirespb.Learned_KIND_TEXT) {
			a.read[k.LearnedValue] = true
		}
		if c.c.GetRegardless() {
			a.report(c.at, "command %s runs whatever became of the commands before it, and reads learned value %s, which is read only once it is bound",
				c.name, k.LearnedValue)
		}
	case *umpirespb.Operand_Projected:
		if !polls {
			a.report(c.at, "command %s reads the value a poll is looking at, and is no poll's condition", c.name)
		}
	case *umpirespb.Operand_Path:
		a.operand(c, k.Path.GetOf(), polls)
	case *umpirespb.Operand_Present:
		a.operand(c, k.Present.GetOf(), polls)
	case *umpirespb.Operand_Equal:
		a.operand(c, k.Equal.GetLeft(), polls)
		a.operand(c, k.Equal.GetRight(), polls)
	case *umpirespb.Operand_All:
		if len(k.All.GetOperands()) == 0 {
			a.report(c.at, "command %s: a conjunction of no operand", c.name)
		}
		for _, operand := range k.All.GetOperands() {
			a.operand(c, operand, polls)
		}
	case *umpirespb.Operand_Greater:
		a.operand(c, k.Greater.GetLeft(), polls)
		a.operand(c, k.Greater.GetRight(), polls)
	case *umpirespb.Operand_Not:
		a.operand(c, k.Not.GetOf(), polls)
	default:
		a.report(c.at, "command %s: an operand of no known kind", c.name)
	}
}

// message checks a protobuf message written out: it names its type, sets no field twice, and every
// value it sets is of a known kind.
func (a *realizing) message(c commandOf, m *umpirespb.Proto) {
	if m.GetMessage() == "" {
		a.report(c.at, "command %s: a message with no name", c.name)
		return
	}
	set := map[string]bool{}
	for _, f := range m.GetFields() {
		if set[f.GetName()] {
			a.report(c.at, "command %s: %s sets %s twice", c.name, m.GetMessage(), f.GetName())
		}
		set[f.GetName()] = true
		a.protoValue(c, m.GetMessage()+"."+f.GetName(), f.GetValue())
	}
}

func (a *realizing) protoValue(c commandOf, field string, value *umpirespb.ProtoValue) {
	switch k := value.GetKind().(type) {
	case *umpirespb.ProtoValue_Text, *umpirespb.ProtoValue_Flag, *umpirespb.ProtoValue_Number,
		*umpirespb.ProtoValue_EnumName, *umpirespb.ProtoValue_Utf8, *umpirespb.ProtoValue_Named:
	case *umpirespb.ProtoValue_Message:
		a.message(c, k.Message)
	case *umpirespb.ProtoValue_Mapping:
		keys := map[string]bool{}
		for _, e := range k.Mapping.GetEntries() {
			if keys[e.GetKey()] {
				a.report(c.at, "command %s: %s has the key %s twice", c.name, field, e.GetKey())
			}
			keys[e.GetKey()] = true
			a.protoValue(c, field, e.GetValue())
		}
	case *umpirespb.ProtoValue_RoleId:
		if _, ok := a.roles[k.RoleId]; !ok {
			a.report(c.at, "command %s: no role %s", c.name, k.RoleId)
		}
	default:
		a.report(c.at, "command %s: %s is set to a value of no known kind", c.name, field)
	}
}

// cycles reports each command of a script that runs after itself. A command with no `after` runs
// after the item before it, which is earlier in the script and so closes no cycle of its own.
func (a *realizing) cycles(s *umpirespb.Script, all map[string]*umpirespb.Command) {
	reported := map[string]bool{}
	for _, item := range s.GetItems() {
		ids := []string{item.GetCommand().GetId()}
		for _, p := range item.GetPerforms() {
			ids = append(ids, p.GetCommand().GetId())
		}
		for _, start := range ids {
			through, cyclic := runsAfterItself(all, start)
			if all[start] == nil || reported[start] || !cyclic {
				continue
			}
			reported[start] = true
			for _, id := range through {
				reported[id] = true
			}
			at := all[start].GetPosition()
			if at.GetFile() == "" {
				at = s.GetPosition()
			}
			if len(through) == 0 {
				a.report(at, "script %s: command %s runs after itself", s.GetId(), start)
			} else {
				a.report(at, "script %s: command %s runs after itself, through %s", s.GetId(), start, strings.Join(through, ", "))
			}
		}
	}
}

// runsAfterItself is whether following the commands a command runs after leads back to it, and the
// commands the way back goes through.
func runsAfterItself(all map[string]*umpirespb.Command, start string) ([]string, bool) {
	var path []string
	seen := map[string]bool{}
	var visit func(id string) bool
	visit = func(id string) bool {
		for _, next := range all[id].GetAfter().GetCommands() {
			if next == start {
				return true
			}
			if all[next] == nil || seen[next] {
				continue
			}
			seen[next] = true
			path = append(path, next)
			if visit(next) {
				return true
			}
			path = path[:len(path)-1]
		}
		return false
	}
	return path, visit(start)
}
