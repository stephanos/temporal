package lower

// How a Case's reads wait, derived from the API behavior its realization declares
// (.plans/API_BEHAVIOR_HINTS.md, "Writes, reads and the order between them"). The declarations are
// the system's kit's; this file only reads them against the path. It holds no bound of its own and
// knows no API: what a method does is read from its HTTP binding, what a worker's command is from
// the activation of its script, and how long anything takes from the hints.
//
//   - A read is a poll, or a call the API binds to HTTP GET that reads its response; a write is a
//     call bound to POST, or a command of a worker's script, which is the kind of cause its
//     activation is. Driver controls and waits are neither.
//   - A script synchronizes at each read: what was written before the step the read waited for is
//     visible to its later reads. Every write between a read's last synchronization and the step it
//     waits for, in path order across scripts, needs a declared visibility to the read's method.
//   - A poll that writes no interval waits as derived. If its script's own write performs the step
//     it waits for, that write's visibility decides: at once it reads once, eventually it polls
//     within the visibility's bound. Otherwise it polls within the sum of the bounds of every
//     asynchronous cause in its window, plus the visibility's bound where the write that performs
//     the step is visible only eventually.
//   - A closing read checks nothing: its realization declares it is made after its sources report
//     nothing more. A poll that writes its own interval keeps it and checks nothing (fn-118.5 then
//     requires the reason it is explicit). A call that reads, in a realization that declares no
//     behavior at all, is taken as written.

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// httpRule is the google.api.http method option the API declares each method's binding with, as
// linked into this binary.
var httpRule = sync.OnceValue(func() protoreflect.ExtensionType {
	xt, err := protoregistry.GlobalTypes.FindExtensionByName("google.api.http")
	if err != nil {
		return nil
	}
	return xt
})

// httpVerb is the HTTP verb the API binds a method to, "get" or "post", or empty where it binds it
// to none.
func httpVerb(md protoreflect.MethodDescriptor) string {
	opts, xt := md.Options(), httpRule()
	if xt == nil || opts == nil || !proto.HasExtension(opts, xt) {
		return ""
	}
	rule, ok := proto.GetExtension(opts, xt).(proto.Message)
	if !ok {
		return ""
	}
	r := rule.ProtoReflect()
	pattern := r.Descriptor().Oneofs().ByName("pattern")
	if pattern == nil {
		return ""
	}
	if f := r.WhichOneof(pattern); f != nil {
		return string(f.Name())
	}
	return ""
}

// visibilityBindings checks what only descriptors can tell of a realization's visibilities: each
// write method is one the API binds to POST, and each read one it binds to GET.
func (a *adapter) visibilityBindings() []error {
	var problems []error
	bound := func(at *umpirespb.Position, id, role, method, verb string) {
		md, err := methodNamed(at, method)
		switch {
		case err != nil:
			problems = append(problems, err)
		case httpVerb(md) != verb:
			problems = append(problems, errorAt(at, "visibility %s names %s as its %s, which the API binds to no HTTP %s", id, method, role,
				strings.ToUpper(verb)))
		default:
		}
	}
	for _, v := range a.r.GetBehavior().GetVisibility() {
		if write := v.GetMethod(); write != "" {
			bound(v.GetPosition(), v.GetId(), "write", write, "post")
		}
		if v.GetRead() != "" {
			bound(v.GetPosition(), v.GetId(), "read", v.GetRead(), "get")
		}
	}
	return problems
}

// derivedWait is how one read of a Case waits: once, or every interval within the hints' bounds.
type derivedWait struct {
	once     bool
	interval int64
	hints    []*testpilotspb.WaitHint
}

// derive applies the wait derived for a node of a script, if one is.
func (a *adapter) derive(script string, node *testpilotspb.InstructionNode) *testpilotspb.InstructionNode {
	w, ok := a.waits[script+"/"+node.GetInstructionId()]
	switch {
	case !ok:
	case w.once:
		cp.ReadOnce()(node)
	default:
		cp.WaitWithin(w.interval, w.hints...)(node)
	}
	return node
}

// carried is one command a Case of the path carries in a script: its instruction id, and the step of
// the path it performs, or -1 for a command the script places.
type carried struct {
	c    *umpirespb.Command
	id   string
	step int
}

// carriedBy is the commands a Case of the path carries in a script, in the order the script runs
// them: a placed command where the path takes it, and a performance once per step of its class, in
// path order, a class taken again under its ordinal, as the producer emits them.
func (l *lowering) carriedBy(s *umpirespb.Script) []carried {
	var out []carried
	for _, item := range s.GetItems() {
		if c := item.GetCommand(); c != nil {
			if l.takes(item, nil) {
				out = append(out, carried{c: c, id: c.GetId(), step: -1})
			}
			continue
		}
		seen := map[string]int{}
		for i, key := range l.keys {
			ordinal := seen[key]
			seen[key]++
			for _, p := range item.GetPerforms() {
				if l.adapter.classKey(p.GetStep()) != key {
					continue
				}
				id := p.GetCommand().GetId()
				if ordinal > 0 {
					id += fmt.Sprintf("-%d", ordinal+1)
				}
				out = append(out, carried{c: p.GetCommand(), id: id, step: i})
			}
		}
	}
	return out
}

// access is what a command is to a wait: a read, a write, or neither.
type access int

const (
	neither access = iota
	reads
	writes
)

// call is one command as a wait reads it: what it is, the method it reads or writes, and the kind of
// cause a write that is no call is. A write with neither is one no hint can name.
type call struct {
	access access
	method string
	cause  umpirespb.CauseKind
	what   string
}

// workerCause is the kind of cause a worker's command is, by the activation of its script.
func workerCause(s *umpirespb.Script) umpirespb.CauseKind {
	switch s.GetActivation().(type) {
	case *umpirespb.Script_Activity:
		return umpirespb.CAUSE_KIND_ACTIVITY_ANSWER
	case *umpirespb.Script_Workflow:
		return umpirespb.CAUSE_KIND_WORKFLOW_TASK
	case *umpirespb.Script_NexusHandler:
		return umpirespb.CAUSE_KIND_HANDLER_REPLY
	default:
		return umpirespb.CAUSE_KIND_UNSPECIFIED
	}
}

// callOf classifies one command of a script. A call the API binds to neither GET nor POST was refused
// where the realization was read.
func (l *lowering) callOf(s *umpirespb.Script, c *umpirespb.Command) call {
	switch in := c.GetInstruction().(type) {
	case *umpirespb.Command_Rpc:
		md, err := methodNamed(c.GetPosition(), in.Rpc.GetMethod())
		switch {
		case err != nil:
			return call{}
		case httpVerb(md) == "post":
			return call{access: writes, method: in.Rpc.GetMethod(), what: "calls " + in.Rpc.GetMethod()}
		case len(in.Rpc.GetReads()) > 0:
			return call{access: reads, method: in.Rpc.GetMethod()}
		default:
			return call{}
		}
	case *umpirespb.Command_Poll:
		e := l.adapter.evidence[in.Poll.GetEvidence()]
		read := e.GetRead()
		if read == nil {
			read = e.GetSingle()
		}
		return call{access: reads, method: read.GetMethod()}
	case *umpirespb.Command_Finish, *umpirespb.Command_AttemptFailure, *umpirespb.Command_AttemptCanceled,
		*umpirespb.Command_WorkflowCommand, *umpirespb.Command_NexusReply:
		kind := workerCause(s)
		if kind == umpirespb.CAUSE_KIND_UNSPECIFIED {
			return call{access: writes, what: "is the answer of no worker"}
		}
		return call{access: writes, cause: kind, what: "is " + umpiremodel.ACause(kind)}
	case *umpirespb.Command_NexusCompletion:
		return call{access: writes, what: "completes a Nexus operation through its callback"}
	default:
		return call{}
	}
}

// writeKey is how a visibility names a write: its method, or its kind of cause.
func writeKey(method string, cause umpirespb.CauseKind) string {
	if method != "" {
		return method
	}
	return "cause " + cause.String()
}

// event is one thing in a read's window, in path order: a write, an asynchronous cause, or both.
type event struct {
	order int
	step  int
	label string
	call  call
	// async is a step or command no command of the read's own script runs, which the read waits for;
	// kind is the cause it is, unspecified where nothing names one.
	async bool
	kind  umpirespb.CauseKind
	// server is the declaration of a step no command performs.
	server *umpirespb.ServerStep
}

// behaviorOf is a realization's API behavior, looked up.
type behaviorOf struct {
	visible map[string]*umpirespb.Visibility
	bounds  map[umpirespb.CauseKind]*umpirespb.CauseBound
	steps   map[string]*umpirespb.ServerStep
}

func (l *lowering) behaviorOf() behaviorOf {
	b := behaviorOf{visible: map[string]*umpirespb.Visibility{}, bounds: map[umpirespb.CauseKind]*umpirespb.CauseBound{},
		steps: map[string]*umpirespb.ServerStep{}}
	for _, v := range l.a.r.GetBehavior().GetVisibility() {
		b.visible[writeKey(v.GetMethod(), v.GetCause())+" "+v.GetRead()] = v
	}
	for _, c := range l.a.r.GetBehavior().GetCauses() {
		b.bounds[c.GetKind()] = c
	}
	for _, s := range l.a.r.GetServerSteps() {
		b.steps[l.adapter.classKey(s.GetStep())] = s
	}
	return b
}

// waiting is the waits of one Case as they are derived.
type waiting struct {
	l        *lowering
	b        behaviorOf
	scripts  map[string][]carried
	performs []performer
	waits    map[string]derivedWait
	// uses is, for each hint and server step a wait reads, the instructions that read it.
	uses     map[string][]string
	problems []error
}

// performer is the command that performs one step of the path, and the script that carries it. A
// step no command performs, such as an activity's delivery, which its script's activation is, has
// none: it is a server step.
type performer struct {
	script  *umpirespb.Script
	command *carried
}

// waits derives how every read of the Case waits, and refuses a read a declaration is missing for.
// It is read once the path's confirmations are known.
func (l *lowering) waits() (map[string]derivedWait, map[string][]string, []error) {
	w := &waiting{l: l, b: l.behaviorOf(), scripts: map[string][]carried{}, performs: make([]performer, len(l.keys)),
		waits: map[string]derivedWait{}, uses: map[string][]string{}}
	for _, s := range l.a.r.GetScripts() {
		seq := l.carriedBy(s)
		w.scripts[s.GetId()] = seq
		for i := range seq {
			if seq[i].step >= 0 {
				w.performs[seq[i].step] = performer{script: s, command: &seq[i]}
			}
		}
	}
	for _, s := range l.a.r.GetScripts() {
		w.script(s)
	}
	return w.waits, w.uses, w.problems
}

// confirmed is the last step of the path a kind of evidence confirms, or -1.
func (l *lowering) confirmed(kind string) int {
	last := -1
	for _, c := range l.confirmations {
		if c.Source.KindID == kind && len(c.Steps) > 0 {
			last = max(last, c.Steps[len(c.Steps)-1])
		}
	}
	return last
}

// script walks one script's commands, synchronizing at each read.
func (w *waiting) script(s *umpirespb.Script) {
	seq := w.scripts[s.GetId()]
	synced, syncedAt := -1, -1
	for i, r := range seq {
		c := w.l.callOf(s, r.c)
		if c.access != reads {
			continue
		}
		poll := r.c.GetPoll()
		target := -1
		if poll != nil {
			target = w.l.confirmed(poll.GetEvidence())
		}
		end := target
		for _, before := range seq[:i] {
			end = max(end, before.step)
		}
		end = max(end, r.step)
		// A call that reads is checked only against a declared behavior: a realization no kit declares
		// one for reads as it is written, as before hints existed.
		if len(r.c.GetCloses()) == 0 && poll.GetIntervalMs() == 0 && (poll != nil || w.l.a.r.GetBehavior() != nil) {
			w.read(s, r, c.method, target, w.window(s, synced, i, syncedAt, end), poll != nil)
		}
		synced, syncedAt = i, end
	}
}

// window is what lies between a read's last synchronization and the step it waits for: the steps of
// the path after the synchronization up to end, but no step its own script performs after it; its
// script's own commands since the synchronization; and each other script's command that runs before
// a step of the window.
func (w *waiting) window(s *umpirespb.Script, synced, at, syncedAt, end int) []event {
	out := w.steps(s, synced, at, syncedAt, end)
	out = append(out, w.placedOwn(s, synced, at)...)
	for _, x := range w.l.a.r.GetScripts() {
		if x.GetId() != s.GetId() {
			out = append(out, w.placedOther(x, syncedAt, end)...)
		}
	}
	slices.SortStableFunc(out, func(a, b event) int { return a.order - b.order })
	return out
}

// steps is the steps of the path after syncedAt up to end, less the ones the read's own script
// performs outside its commands between synced and at.
func (w *waiting) steps(s *umpirespb.Script, synced, at, syncedAt, end int) []event {
	seq := w.scripts[s.GetId()]
	var out []event
	for i := syncedAt + 1; i <= end; i++ {
		p := w.performs[i]
		switch {
		case p.command != nil && p.script.GetId() == s.GetId():
			if j := slices.IndexFunc(seq, func(c carried) bool { return c.step == i }); j > synced && j < at {
				out = append(out, event{order: 2*i + 1, step: i, label: w.commandLabel(s, *p.command), call: w.l.callOf(s, p.command.c)})
			}
		case p.command != nil:
			out = append(out, w.other(p.script, *p.command, 2*i+1))
		default:
			e := event{order: 2*i + 1, step: i, label: "step " + w.l.keys[i], async: true, server: w.b.steps[w.l.keys[i]]}
			e.kind = e.server.GetKind()
			out = append(out, e)
		}
	}
	return out
}

// placedOwn is the commands the read's script places between synced and at, each after the steps its
// script performed before it.
func (w *waiting) placedOwn(s *umpirespb.Script, synced, at int) []event {
	var out []event
	performed := -1
	for j, c := range w.scripts[s.GetId()][:at] {
		performed = max(performed, c.step)
		if j > synced && c.step < 0 {
			out = append(out, event{order: 2*performed + 2, step: -1, label: w.commandLabel(s, c), call: w.l.callOf(s, c.c)})
		}
	}
	return out
}

// placedOther is the commands another script places that run before a step of the window: before the
// next step their script performs.
func (w *waiting) placedOther(x *umpirespb.Script, syncedAt, end int) []event {
	var out []event
	xs := w.scripts[x.GetId()]
	for j, c := range xs {
		if c.step >= 0 {
			continue
		}
		next := slices.IndexFunc(xs[j:], func(c carried) bool { return c.step >= 0 })
		if next < 0 {
			continue
		}
		if hi := xs[j+next].step; hi > syncedAt && hi <= end {
			out = append(out, w.other(x, c, 2*hi))
		}
	}
	return out
}

// other is a command of another script than the read's, which the read waits for where it does
// anything the server acts on.
func (w *waiting) other(x *umpirespb.Script, c carried, order int) event {
	e := event{order: order, step: c.step, label: w.commandLabel(x, c), call: w.l.callOf(x, c.c)}
	if e.call.access == writes {
		e.async, e.kind = true, e.call.cause
	}
	return e
}

func (w *waiting) commandLabel(s *umpirespb.Script, c carried) string {
	label := "command " + s.GetId() + "/" + c.id
	if c.step >= 0 {
		label += " (step " + w.l.keys[c.step] + ")"
	}
	return label
}

// reading is one read whose wait is being derived: where it is declared, its name and its part of the
// Case, the method it reads, and what it has read of the behavior so far.
type reading struct {
	at     *umpirespb.Position
	name   string
	method string
	used   []string
	// visibility is the declared visibility of each write of the window, by its place there.
	visibility map[int]*umpirespb.Visibility
	hints      []*testpilotspb.WaitHint
	interval   int64
}

// bound adds a hint's bound to the wait, which then looks at the smallest interval of its hints.
func (r *reading) bound(id string, at *umpirespb.Position, b *umpirespb.WaitBound) {
	r.hints = append(r.hints, waitHint(id, at, b.GetAtMostMs()))
	if r.interval == 0 || b.GetIntervalMs() < r.interval {
		r.interval = b.GetIntervalMs()
	}
}

// read derives one read's wait from its window, or refuses it.
func (w *waiting) read(s *umpirespb.Script, c carried, method string, target int, window []event, poll bool) {
	r := &reading{at: c.c.GetPosition(), name: s.GetId() + "/" + c.id, method: method, visibility: map[int]*umpirespb.Visibility{}}
	part := "program.entrypoints[" + s.GetId() + "].instructions[" + c.id + "]"
	before := len(w.problems)
	w.visibilities(r, window)
	if !poll {
		for i := range window {
			if v := r.visibility[i]; v.GetEventuallyWithin() != nil {
				w.problems = append(w.problems, errorAt(r.at, "command %s reads %s once, after %s, which is visible to it only eventually (%s): a read that waits is a poll",
					r.name, method, window[i].label, v.GetId()))
			}
		}
		w.use(part, r.used)
		return
	}
	decided := slices.IndexFunc(window, func(e event) bool { return e.step == target && target >= 0 })
	if decided >= 0 && !window[decided].async && window[decided].call.access == writes {
		// The read's own script performed the step it waits for: that write's visibility decides.
		if v := r.visibility[decided]; v.GetEventuallyWithin() != nil {
			r.bound(v.GetId(), v.GetPosition(), v.GetEventuallyWithin())
		}
	} else {
		w.causes(r, window)
		// The write that performs the step waited for adds its own bound where it is eventual; with
		// no such step, every eventual write of the window does.
		for i := range window {
			if v := r.visibility[i]; v.GetEventuallyWithin() != nil && (decided < 0 || i == decided) {
				r.bound(v.GetId(), v.GetPosition(), v.GetEventuallyWithin())
			}
		}
	}
	if len(w.problems) > before {
		return
	}
	w.use(part, r.used)
	if len(r.hints) == 0 {
		w.waits[r.name] = derivedWait{once: true}
		return
	}
	w.waits[r.name] = derivedWait{interval: r.interval, hints: r.hints}
}

// visibilities finds the declared visibility of each write of a read's window to the read's method,
// and refuses each write, once, that has none.
func (w *waiting) visibilities(r *reading, window []event) {
	refused := map[string]bool{}
	for i, e := range window {
		if e.call.access != writes {
			continue
		}
		key := writeKey(e.call.method, e.call.cause)
		v := w.b.visible[key+" "+r.method]
		switch {
		case v != nil:
			r.visibility[i] = v
			r.used = append(r.used, "behavior:"+v.GetId())
		case refused[key]:
		case e.call.method == "" && e.call.cause == umpirespb.CAUSE_KIND_UNSPECIFIED:
			refused[key] = true
			w.problems = append(w.problems, errorAt(r.at, "command %s reads %s after %s, which %s, and no hint can declare when that is visible to a read",
				r.name, r.method, e.label, e.call.what))
		default:
			refused[key] = true
			w.problems = append(w.problems, errorAt(r.at, "command %s reads %s after %s, which %s, and the realization declares no visibility of %s to %s",
				r.name, r.method, e.label, e.call.what, w.written(e.call), r.method))
		}
	}
}

// causes adds the bound of each asynchronous cause of a read's window, a timer's deadline before its
// slack, and refuses a cause no kind names or whose kind no bound bounds.
func (w *waiting) causes(r *reading, window []event) {
	for _, e := range window {
		if !e.async {
			continue
		}
		switch {
		case e.kind != umpirespb.CAUSE_KIND_UNSPECIFIED:
		case e.call.access == neither:
			w.problems = append(w.problems, errorAt(r.at, "command %s waits for %s, which no command performs and no server step declares the kind of cause of",
				r.name, e.label))
			continue
		default:
			w.problems = append(w.problems, errorAt(r.at, "command %s waits for %s, which another script runs and no kind of cause names", r.name, e.label))
			continue
		}
		cause := w.b.bounds[e.kind]
		if cause == nil {
			w.problems = append(w.problems, errorAt(r.at, "command %s waits for %s, which is %s, and the realization declares no bound of %s",
				r.name, e.label, umpiremodel.ACause(e.kind), umpiremodel.ACause(e.kind)))
			continue
		}
		if e.server != nil {
			r.used = append(r.used, "server_steps:"+w.l.keys[e.step])
			if e.kind == umpirespb.CAUSE_KIND_TIMER {
				r.hints = append(r.hints, waitHint("deadline."+hintName(w.l.keys[e.step]), e.server.GetPosition(), e.server.GetDeadlineMs()))
			}
		}
		r.used = append(r.used, "behavior:"+cause.GetId())
		r.bound(cause.GetId(), cause.GetPosition(), cause.GetBound())
	}
}

// use records that an instruction reads hints and server steps.
func (w *waiting) use(part string, used []string) {
	for _, u := range used {
		if !slices.Contains(w.uses[u], part) {
			w.uses[u] = append(w.uses[u], part)
		}
	}
}

// written is how a diagnostic names a write a visibility would name.
func (w *waiting) written(c call) string {
	if c.method != "" {
		return c.method
	}
	return umpiremodel.ACause(c.cause)
}

// waitHint is one hint a wait's bound is the sum of, where it is declared.
func waitHint(id string, at *umpirespb.Position, ms int64) *testpilotspb.WaitHint {
	return &testpilotspb.WaitHint{HintId: id, AtMostMilliseconds: ms,
		Source: &testpilotspb.SourceLocation{Path: at.GetFile(), Line: at.GetLine(), Provenance: "scala-model"}}
}

// hintName spells a class key in the characters a hint id takes.
func hintName(key string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-', r == '.':
			return r
		default:
			return '_'
		}
	}, key)
}
