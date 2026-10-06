package model

import (
	"strings"

	umpirespb "go.temporal.io/server/api/umpire/v1"
)

// causeKinds is what a diagnostic calls each kind of asynchronous cause.
var causeKinds = map[umpirespb.CauseKind]string{
	umpirespb.CAUSE_KIND_ACTIVITY_ANSWER: "activity answer", umpirespb.CAUSE_KIND_WORKFLOW_TASK: "workflow task",
	umpirespb.CAUSE_KIND_HANDLER_REPLY: "handler reply", umpirespb.CAUSE_KIND_DELIVERY: "delivery", umpirespb.CAUSE_KIND_TIMER: "timer",
}

// ACause is a kind of cause with its article, as a diagnostic names it: "a delivery", "an activity answer".
func ACause(k umpirespb.CauseKind) string {
	if strings.HasPrefix(causeKinds[k], "a") {
		return "an " + causeKinds[k]
	}
	return "a " + causeKinds[k]
}

// activationOf begins what performs a class that a script's activation performs: an activity's
// delivery, which no command performs, so a server step may declare it.
const activationOf = "the activation of "

// behavior checks the hints a realization declares of how the APIs it calls behave, and its server
// steps (.plans/API_BEHAVIOR_HINTS.md). Each hint is named by an id no other hint takes; a visibility
// names a write and a read, and no pair twice; a cause bound is of one kind, which no other bounds;
// and every bound is positive, with an interval no greater than it. Attempts are numbered from a
// positive first number, and the limits of an instruction that writes none are positive. A server
// step is a class of the machine that no command performs, declared once, of a kind the realization
// bounds, and a timer's step names the deadline the realization set. It runs once the scripts are
// read, so that what the commands perform is known. Whether the API binds a write to POST and a read
// to GET is the lowering's to check: this package holds no descriptors.
func (a *realizing) behavior(mm *umpirespb.Machine) {
	a.visibilities()
	if n := a.r.GetBehavior().GetAttemptNumbering(); n != nil && n.GetFirst() < 1 {
		a.report(n.GetPosition(), "attempts are numbered from %d; the first attempt's number is positive", n.GetFirst())
	}
	if d := a.r.GetBehavior().GetInstructionDefaults(); d != nil && (d.GetTimeoutMs() <= 0 || d.GetAttempts() <= 0) {
		a.report(d.GetPosition(), "an instruction that writes no limits takes %d ms and %d attempts; both are positive", d.GetTimeoutMs(), d.GetAttempts())
	}
	bounded := a.causeBounds()
	declared := map[string]bool{}
	for _, s := range a.r.GetServerSteps() {
		a.serverStepKind(mm, s, bounded, declared)
	}
}

// visibilities checks each visibility: named, with a write and a read, a pair no other names, and a
// bound, where it is eventual, that holds.
func (a *realizing) visibilities() {
	pairs := map[string]string{}
	for _, h := range a.r.GetBehavior().GetVisibility() {
		at, id := h.GetPosition(), h.GetId()
		a.declared(at, "a hint", "hints", id)
		write := visibilityWrite(h)
		switch {
		case write == "":
			a.report(at, "visibility %s names no write", id)
		case h.GetRead() == "":
		default:
			pair := write + " " + h.GetRead()
			// Two hints of one derived id are one pair, which the id's refusal already names.
			if other, ok := pairs[pair]; ok && other != id {
				a.report(at, "visibility %s and %s both declare when %s is visible to %s; one hint declares a pair", other, id, write, h.GetRead())
			} else {
				pairs[pair] = id
			}
		}
		if h.GetRead() == "" {
			a.report(at, "visibility %s names no read", id)
		}
		if h.GetEventuallyWithin() != nil {
			a.waitBound(h.GetEventuallyWithin(), at, "visibility "+id)
		}
	}
}

// visibilityWrite is what a diagnostic calls a visibility's write: its method, or its kind of cause,
// or nothing where it names neither.
func visibilityWrite(h *umpirespb.Visibility) string {
	switch w := h.GetWrite().(type) {
	case *umpirespb.Visibility_Method:
		return w.Method
	case *umpirespb.Visibility_Cause:
		if Known(umpirespb.CauseKind_name, int32(w.Cause)) {
			return ACause(w.Cause)
		}
	default:
	}
	return ""
}

// causeBounds checks each cause bound: named, of a known kind no other bounds, with a bound that
// holds. It is the bound of each kind, by its hint's id.
func (a *realizing) causeBounds() map[umpirespb.CauseKind]string {
	bounded := map[umpirespb.CauseKind]string{}
	for _, c := range a.r.GetBehavior().GetCauses() {
		at, id := c.GetPosition(), c.GetId()
		a.declared(at, "a hint", "hints", id)
		if !Known(umpirespb.CauseKind_name, int32(c.GetKind())) {
			a.report(at, "cause bound %s is of no known kind", id)
		} else if other, ok := bounded[c.GetKind()]; ok {
			a.report(at, "cause bounds %s and %s both bound %s; one hint bounds a kind", other, id, ACause(c.GetKind()))
		} else {
			bounded[c.GetKind()] = id
		}
		if c.GetBound() == nil {
			a.report(at, "cause bound %s declares no bound", id)
		} else {
			a.waitBound(c.GetBound(), at, "cause bound "+id)
		}
	}
	return bounded
}

// serverStepKind checks one server step: its class, declared once, a kind the realization bounds, and
// a deadline where, and only where, it is a timer.
func (a *realizing) serverStepKind(mm *umpirespb.Machine, s *umpirespb.ServerStep, bounded map[umpirespb.CauseKind]string, declared map[string]bool) {
	at := s.GetPosition()
	step, ok := a.serverStep(mm, s)
	if ok {
		if declared[step] {
			a.report(at, "%s is declared twice", step)
		}
		declared[step] = true
	}
	kind := s.GetKind()
	switch _, bounds := bounded[kind]; {
	case !Known(umpirespb.CauseKind_name, int32(kind)):
		a.report(at, "%s is of no known kind", step)
	case !bounds:
		a.report(at, "%s is %s, and the realization bounds no %s", step, ACause(kind), causeKinds[kind])
	default:
	}
	switch {
	case kind == umpirespb.CAUSE_KIND_TIMER && s.GetDeadlineMs() <= 0:
		a.report(at, "%s is a timer and names no positive deadline", step)
	case kind != umpirespb.CAUSE_KIND_TIMER && s.GetDeadlineMs() != 0:
		a.report(at, "%s names a deadline of %d milliseconds, and only a timer's step has one", step, s.GetDeadlineMs())
	default:
	}
}

// serverStep checks the class a server step declares: one the machine binds, which no command
// performs. It is what a diagnostic calls the step, and whether the class is one of the machine's, so
// that the step can be told from the others.
func (a *realizing) serverStep(mm *umpirespb.Machine, s *umpirespb.ServerStep) (string, bool) {
	at, class := s.GetPosition(), s.GetStep()
	if class == nil {
		a.report(at, "a server step is of no class")
		return "a server step", false
	}
	if mm == nil {
		return "server step " + class.GetAction(), false
	}
	if at.GetFile() == "" {
		at = a.r.GetPosition()
	}
	before := a.d.Errors()
	a.d.ActionClass(a.owner+": a server step", mm, class, at)
	if a.d.Errors() != before {
		return "server step " + class.GetAction(), false
	}
	key := a.d.ClassKey(class)
	if by, ok := a.performed[key]; ok && !strings.HasPrefix(by, activationOf) {
		a.report(at, "server step %s is performed by %s; a server step is one no command performs", key, by)
	}
	return "server step " + key, true
}

// waitBound checks that a bound and its interval are positive, and that the interval is no greater
// than the bound. A bound written nowhere of its own is reported where its hint is.
func (a *realizing) waitBound(b *umpirespb.WaitBound, hint *umpirespb.Position, of string) {
	at := b.GetPosition()
	if at.GetFile() == "" {
		at = hint
	}
	if b.GetIntervalMs() <= 0 {
		a.report(at, "%s looks every %d milliseconds; an interval is positive", of, b.GetIntervalMs())
	}
	if b.GetAtMostMs() <= 0 {
		a.report(at, "%s waits at most %d milliseconds; a bound is positive", of, b.GetAtMostMs())
	}
	if b.GetIntervalMs() > 0 && b.GetAtMostMs() > 0 && b.GetIntervalMs() > b.GetAtMostMs() {
		a.report(at, "%s looks every %d milliseconds and waits at most %d; an interval is no greater than its bound", of, b.GetIntervalMs(), b.GetAtMostMs())
	}
}
