package lower

// A realization of the IR as the producer.Realization a producer consumes. The translation is
// one declaration to one part, in the order the IR lists them: nothing here chooses what a Case
// does. What a Query's path selects among these parts is the producer's rule, the same for every
// realization.

import (
	"errors"
	"math"
	"slices"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	umpirespb "go.temporal.io/server/api/umpire/v1"
	cp "go.temporal.io/server/tools/umpire/lower/internal/producer"
	umpiremodel "go.temporal.io/server/tools/umpire/model"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// adapter translates one realization for one Case identity, checking what it writes and reads
// against the protobuf descriptors it names. It reads every declaration and reports every problem:
// one declaration that crosses a descriptor hides none after it.
type adapter struct {
	r *umpirespb.Realization
	t *umpiremodel.Table
	// classKey is the key of one class of an action, as the Model's tables key it.
	classKey func(*umpirespb.ActionClass) string
	w        *writer
	// evidence and element are each kind of evidence and the message its recorded data is; observed
	// is each observation's message.
	evidence map[string]*umpirespb.Evidence
	element  map[string]protoreflect.MessageDescriptor
	observed map[string]protoreflect.MessageDescriptor
}

func newAdapter(r *umpirespb.Realization, t *umpiremodel.Table, classKey func(*umpirespb.ActionClass) string, fixture string) *adapter {
	a := &adapter{r: r, t: t, classKey: classKey, w: &writer{fixture: fixture},
		evidence: map[string]*umpirespb.Evidence{}, element: map[string]protoreflect.MessageDescriptor{},
		observed: map[string]protoreflect.MessageDescriptor{}}
	for _, e := range r.GetEvidence() {
		a.evidence[e.GetId()] = e
	}
	return a
}

var faultKinds = map[umpirespb.Fault_Kind]testpilotspb.FaultKind{
	umpirespb.Fault_KIND_WORKER_STOP:             testpilotspb.FAULT_KIND_WORKER_STOP,
	umpirespb.Fault_KIND_WORKER_RESUME:           testpilotspb.FAULT_KIND_WORKER_RESUME,
	umpirespb.Fault_KIND_ADMISSION_RESPONSE_LOSS: testpilotspb.FAULT_KIND_ADMISSION_RESPONSE_LOSS,
}

var roleKinds = map[umpirespb.Role_Kind]testpilotspb.RoleKind{
	umpirespb.Role_KIND_ENDPOINT:    testpilotspb.ROLE_KIND_ENDPOINT,
	umpirespb.Role_KIND_WORKER:      testpilotspb.ROLE_KIND_WORKER,
	umpirespb.Role_KIND_TASK_QUEUE:  testpilotspb.ROLE_KIND_TASK_QUEUE,
	umpirespb.Role_KIND_PARTICIPANT: testpilotspb.ROLE_KIND_PARTICIPANT,
}

func (a *adapter) realization() (*cp.Realization, []error) {
	var problems []error
	c := a.r.GetCorrelation()
	out := &cp.Realization{ProducerID: a.r.GetProducer(), ProducerVersion: a.r.GetProducerVersion(),
		ProjectionID: c.GetProjection(), ScopeField: c.GetRun(), OperationKey: c.GetOperation(),
		CorrelatedObservation: c.GetObservation(),
		ProjectionLimits: cp.ProjectionLimits{Events: c.GetEvents(), Buffered: c.GetBuffered(), Keys: c.GetKeys(),
			Support: c.GetSupport(), Work: c.GetWork(), EventSize: c.GetEventSize()}}
	for _, role := range a.r.GetRoles() {
		out.Plan.Roles = append(out.Plan.Roles, cp.Role(role.GetId(), roleKinds[role.GetKind()], role.GetNamespace(), role.GetResource()))
	}
	for _, l := range a.r.GetLearned() {
		if l.GetKind() == umpirespb.Learned_KIND_HANDLE {
			out.Plan.Slots = append(out.Plan.Slots, cp.HandleSlot(l.GetId()))
			continue
		}
		out.Plan.Slots = append(out.Plan.Slots, &testpilotspb.Slot{SlotId: l.GetId(), Content: &testpilotspb.Slot_Value{
			Value: &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{
				Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: testpilotspb.SCALAR_KIND_TEXT}}}}}}})
	}
	for _, o := range a.r.GetObservations() {
		md, err := messageNamed(o.GetPosition(), o.GetMessage())
		if err != nil {
			problems = append(problems, err)
			continue
		}
		a.observed[o.GetId()] = md
		out.Plan.Observations = append(out.Plan.Observations, cp.MessageObservation(o.GetId(), o.GetMessage()))
	}
	for _, e := range a.r.GetEvidence() {
		source, err := a.source(e)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		out.Sources = append(out.Sources, source)
	}
	for _, s := range a.r.GetScripts() {
		plan, bindings, crossed := a.script(s)
		problems = append(problems, crossed...)
		out.Plan.Entrypoints = append(out.Plan.Entrypoints, plan)
		out.Actions = append(out.Actions, bindings...)
	}
	if a.r.GetCleanup() != "" {
		out.Plan.Cleanup = &testpilotspb.Cleanup{EntrypointId: a.r.GetCleanup()}
	}
	return out, problems
}

var runEventKinds = map[umpirespb.RunEventSource_Kind]testpilotspb.RunEventKind{
	umpirespb.RunEventSource_KIND_INSTRUCTION_COMPLETED: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_COMPLETED,
	umpirespb.RunEventSource_KIND_INSTRUCTION_TIMED_OUT: testpilotspb.RUN_EVENT_KIND_INSTRUCTION_TIMED_OUT,
	umpirespb.RunEventSource_KIND_DIAGNOSTIC:            testpilotspb.RUN_EVENT_KIND_DIAGNOSTIC,
}

var fieldRoles = map[umpirespb.EvidenceField_Role]string{
	umpirespb.EvidenceField_ROLE_OPERATION: "operation",
	umpirespb.EvidenceField_ROLE_ATTEMPT:   "attempt",
	umpirespb.EvidenceField_ROLE_DELIVERY:  "delivery",
}

// source is one kind of evidence as a producer reads it, once its recorded message and the field
// that keys its operation are found in the descriptors it names.
func (a *adapter) source(e *umpirespb.Evidence) (*cp.EvidenceSource, error) {
	at := e.GetPosition()
	out := &cp.EvidenceSource{EventKind: e.GetRecords(), OperationKeyPath: e.GetOperation(), KindID: e.GetId(), SourceID: e.GetSource(),
		Exhaustive: e.GetExhaustive()}
	var element protoreflect.MessageDescriptor
	switch from := e.GetFrom().(type) {
	case *umpirespb.Evidence_History:
		event, err := messageNamed(at, historyEventMessage)
		if err != nil {
			return nil, err
		}
		if event.Oneofs().ByName("attributes").Fields().ByName(protoreflect.Name(from.History)) == nil {
			return nil, errorAt(at, "evidence %s: a history event has no attributes %s", e.GetId(), from.History)
		}
		element, out.Recorded = event, cp.Recorded{HistoryAttributes: from.History}
	case *umpirespb.Evidence_Read:
		read, err := readFrom(e, from.Read, true)
		if err != nil {
			return nil, err
		}
		element, out.Recorded = read, cp.Recorded{Method: from.Read.GetMethod(), Path: from.Read.GetPath()}
	case *umpirespb.Evidence_Single:
		read, err := readFrom(e, from.Single, false)
		if err != nil {
			return nil, err
		}
		element, out.Recorded = read, cp.Recorded{Method: from.Single.GetMethod(), Path: from.Single.GetPath(), Single: true}
	case *umpirespb.Evidence_RunEvent:
		payload, err := a.payloadOf(e, from.RunEvent)
		if err != nil {
			return nil, err
		}
		recorded, err := a.runEvent(e, from.RunEvent, payload)
		if err != nil {
			return nil, err
		}
		element, out.Recorded = payload, cp.Recorded{RunEvent: recorded}
		// A key that is a path of the payload is the kind's operation key path; the run's id is none.
		out.OperationKeyPath = from.RunEvent.GetKey().GetPath().GetPath()
	default:
		return nil, errorAt(at, "evidence %s is recorded nowhere", e.GetId())
	}
	if e.GetRunEvent() == nil {
		key, err := walk(at, element, e.GetOperation())
		if err != nil {
			return nil, err
		}
		if key.fanned || key.field.IsList() || key.message() != nil {
			return nil, errorAt(at, "evidence %s keys its operation by %s, which is no single scalar of %s", e.GetId(), e.GetOperation(), element.FullName())
		}
	}
	for _, f := range e.GetFields() {
		kind, err := carriedField(e, f, element)
		if err != nil {
			return nil, err
		}
		// A field is carried with its value. One the realization redacts is a gap before any Case is
		// produced, and has no place in a Case's inventory.
		out.Fields = append(out.Fields, cp.EvidenceField{ID: f.GetId(), Path: f.GetPath(), Type: kind, Role: fieldRoles[f.GetRole()]})
	}
	for _, taking := range e.GetConfirms() {
		out.Confirms = append(out.Confirms, cp.Taking{Key: a.classKey(taking.GetStep()), Occurrence: int(min(taking.GetOccurrence(), math.MaxInt32))})
	}
	a.element[e.GetId()] = element
	return out, nil
}

// runEvent is the Run's own record as a producer declares it: the events of one kind that one
// instruction records, under the source's guard, keyed by the Run or by a path of the payload. A Run
// records the events of a controller's instructions under their own coordinates and no other
// script's, so the command is a controller's.
func (a *adapter) runEvent(e *umpirespb.Evidence, source *umpirespb.RunEventSource, payload protoreflect.MessageDescriptor) (*cp.RunEvent, error) {
	at := e.GetPosition()
	kind, known := runEventKinds[source.GetKind()]
	if !known {
		return nil, errorAt(at, "evidence %s is a Run Event of no kind this reader lowers", e.GetId())
	}
	for _, s := range a.r.GetScripts() {
		if s.GetId() == source.GetScript() && s.GetController() == nil {
			return nil, errorAt(at, "evidence %s is the Run's record of a command of script %s, which no controller runs: a Run records the events of a controller's instructions",
				e.GetId(), s.GetId())
		}
	}
	out := &cp.RunEvent{Kind: kind, EntrypointID: source.GetScript(), InstructionID: source.GetCommand(), RunKeyed: source.GetKey().GetRun() != nil}
	var err error
	if out.Guard, err = a.guardOf(e, source, payload); err != nil {
		return nil, err
	}
	return out, nil
}

// attemptNumber is where the Run's record of an attempt holds the number the server counts it by.
const attemptNumber = "activity_attempt.sdk_attempt"

// guardOf is the guard a Case's declaration of the Run's own record states: the source's own, and,
// for the record of an attempt, that the record is of the attempt the source names. The declaration of
// which attempt a record is of is then what the runtime selects the record by, and no guard a
// realization writes can take another attempt's record for it. The source's own guard is read first: it
// says the record is of an attempt at all.
func (a *adapter) guardOf(e *umpirespb.Evidence, source *umpirespb.RunEventSource, payload protoreflect.MessageDescriptor) (*testpilotspb.Expression, error) {
	at := e.GetPosition()
	var guard *testpilotspb.Expression
	if source.GetGuard() != nil {
		var err error
		if guard, err = a.operand(at, source.GetGuard(), payload); err != nil {
			return nil, err
		}
	}
	of := source.GetAttempt()
	if of == nil {
		return guard, nil
	}
	if _, err := walk(at, payload, attemptNumber); err != nil {
		return nil, err
	}
	numbered := cp.Equal(cp.Path(cp.ProjectedValue(), attemptNumber), cp.Literal(cp.SignedInteger(of.GetNumber())))
	if guard == nil {
		return numbered, nil
	}
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{
		Operands: []*testpilotspb.Expression{guard, numbered}}}}, nil
}

// readFrom is the message a kind of evidence is read from at a path of a method's response: the
// element of a repeated field, or the one message a single read names.
func readFrom(e *umpirespb.Evidence, read *umpirespb.ReadSource, repeated bool) (protoreflect.MessageDescriptor, error) {
	at := e.GetPosition()
	method, err := methodNamed(at, read.GetMethod())
	if err != nil {
		return nil, err
	}
	end, err := walk(at, method.Output(), read.GetPath())
	if err != nil {
		return nil, err
	}
	switch {
	case repeated && (!end.field.IsList() || end.message() == nil):
		return nil, errorAt(at, "evidence %s is read from %s, which is no repeated message of %s", e.GetId(), read.GetPath(), method.Output().FullName())
	case !repeated && (end.fanned || end.field.IsList() || end.message() == nil):
		return nil, errorAt(at, "evidence %s is read from %s, which is no single message of %s", e.GetId(), read.GetPath(), method.Output().FullName())
	default:
		return end.message(), nil
	}
}

// payloadOf is the payload of the Run Events a kind of evidence is: the instruction outcome, which
// the source's guard, and its key where that is a path, are read against.
func (a *adapter) payloadOf(e *umpirespb.Evidence, source *umpirespb.RunEventSource) (protoreflect.MessageDescriptor, error) {
	at := e.GetPosition()
	payload, err := messageNamed(at, instructionOutcomeMessage)
	if err != nil {
		return nil, err
	}
	if guard := source.GetGuard(); guard != nil {
		if err := umpiremodel.GuardProblem(guard, payload); err != nil {
			return nil, errorAt(at, "evidence %s: its guard %s", e.GetId(), err)
		}
	}
	// The key names one operation: the run's own id, or one text or integer of the payload.
	if key := source.GetKey(); key.GetPath() != nil {
		computes, err := umpiremodel.TypeOf(key, payload, lifted(at))
		if err != nil {
			return nil, mistyped(at, err, "evidence %s: its key", e.GetId())
		}
		if computes.Shape != umpiremodel.TextShape && computes.Shape != umpiremodel.NumberShape {
			return nil, errorAt(at, "evidence %s: its key reads %s, which is no single text or integer of %s", e.GetId(), key.GetPath().GetPath(), payload.FullName())
		}
	}
	return payload, nil
}

// mistyped locates a type problem under what it is a problem of, and leaves any other error as it is.
func mistyped(at *umpirespb.Position, err error, format string, args ...any) error {
	var problem *umpiremodel.Mistype
	if errors.As(err, &problem) {
		return errorAt(at, format+" %s", append(args, problem.Says)...)
	}
	return err
}

// lifted types a path as Testpilot reads one where it lifts evidence: a field, each element of a
// repeated field, or a member of a oneof, of the message the path is read from.
func lifted(at *umpirespb.Position) umpiremodel.Paths {
	return func(of umpiremodel.Typed, path string) (umpiremodel.Typed, error) {
		if of.Message == nil {
			return umpiremodel.Typed{}, nil
		}
		end, err := walk(at, of.Message, path)
		if err != nil {
			return umpiremodel.Typed{}, err
		}
		return fieldValue(end), nil
	}
}

func fieldValue(end reached) umpiremodel.Typed {
	if end.fanned || end.field.IsList() {
		return umpiremodel.Typed{Shape: umpiremodel.SeveralShape}
	}
	switch end.field.Kind() {
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return umpiremodel.Typed{Shape: umpiremodel.MessageShape, Message: end.message()}
	case protoreflect.BoolKind:
		return umpiremodel.Typed{Shape: umpiremodel.ConditionShape}
	case protoreflect.StringKind:
		return umpiremodel.Typed{Shape: umpiremodel.TextShape}
	case protoreflect.EnumKind:
		return umpiremodel.Typed{Shape: umpiremodel.EnumShape, Enum: end.field.Enum()}
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return umpiremodel.Typed{Shape: umpiremodel.NumberShape}
	default:
		return umpiremodel.Typed{Shape: umpiremodel.OtherShape}
	}
}

// carriedField checks that a field of evidence reads what evidence can carry, one text, flag or
// integer of the recorded message, and is the scalar evidence carries it as.
func carriedField(e *umpirespb.Evidence, f *umpirespb.EvidenceField, element protoreflect.MessageDescriptor) (testpilotspb.ScalarKind, error) {
	at := f.GetPosition()
	if at.GetFile() == "" {
		at = e.GetPosition()
	}
	end, err := walk(at, element, f.GetPath())
	if err != nil {
		return 0, err
	}
	carried := testpilotspb.SCALAR_KIND_UNSPECIFIED
	switch end.field.Kind() {
	case protoreflect.StringKind:
		carried = testpilotspb.SCALAR_KIND_TEXT
	case protoreflect.BoolKind:
		carried = testpilotspb.SCALAR_KIND_BOOLEAN
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind, protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind, protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		carried = testpilotspb.SCALAR_KIND_UINT64
	default:
	}
	if carried == testpilotspb.SCALAR_KIND_UNSPECIFIED || end.fanned || end.field.IsList() {
		return 0, errorAt(at, "evidence %s: field %s reads %s, which is no single text, flag or integer of %s", e.GetId(), f.GetId(), f.GetPath(), element.FullName())
	}
	return carried, nil
}

// built is one command as an instruction node, with the lifts a producer fills from the rules a
// path resolves.
type built struct {
	node *testpilotspb.InstructionNode
}

// with is the node under an id, carrying the history kinds among the rules where the command lifts
// evidence. A lift with no rule is left out: a path that records no history kind lifts nothing.
func (b built) with(id string, rules []cp.EvidenceRule) *testpilotspb.InstructionNode {
	node := proto.CloneOf(b.node)
	node.InstructionId = id
	lifts := false
	for _, r := range rules {
		lifts = lifts || r.ReadsHistory()
	}
	for _, read := range node.GetInstruction().GetInvokeRpc().GetResponseReads() {
		var targets []*testpilotspb.ReadTarget
		for _, target := range read.GetTargets() {
			lift := target.GetCorrelatedEvidence()
			switch {
			case lift == nil:
				targets = append(targets, target)
			case lifts:
				targets = append(targets, cp.EvidenceTarget(lift.GetObservationId(), rules))
			default:
			}
		}
		read.Targets = targets
	}
	return node
}

func (a *adapter) script(s *umpirespb.Script) (cp.EntrypointPlan, []cp.ActionBinding, []error) {
	var plan cp.EntrypointPlan
	var bindings []cp.ActionBinding
	var problems []error
	for _, item := range s.GetItems() {
		if item.GetCommand() != nil {
			placed, err := a.placed(s, item)
			if err != nil {
				problems = append(problems, err)
				continue
			}
			plan.Items = append(plan.Items, placed)
			continue
		}
		var classes []string
		for _, p := range item.GetPerforms() {
			b, err := a.command(s, p.GetCommand())
			if err != nil {
				problems = append(problems, err)
				continue
			}
			key := a.classKey(p.GetStep())
			action := a.t.ActionAtom(key).ID
			classes = append(classes, action)
			bindings = append(bindings, cp.ActionBinding{Action: action, Key: key, InstructionID: p.GetCommand().GetId(),
				Node: func(_ cp.Placement, id string) *testpilotspb.InstructionNode { return b.with(id, nil) }})
		}
		plan.Items = append(plan.Items, cp.Actions{Classes: classes})
	}
	activate, err := a.activation(s)
	if err != nil {
		return plan, nil, append(problems, err)
	}
	plan.Activate = func(_ cp.Placement, nodes []*testpilotspb.InstructionNode) *testpilotspb.Entrypoint {
		e := activate()
		e.EntrypointId, e.Instructions = s.GetId(), nodes
		return e
	}
	return plan, bindings, problems
}

// placed is a command every Case carries, or one the Cases whose path takes one of its classes carry.
func (a *adapter) placed(s *umpirespb.Script, item *umpirespb.Item) (cp.Item, error) {
	c := item.GetCommand()
	b, err := a.command(s, c)
	if err != nil {
		return nil, err
	}
	node := func(_ cp.Placement, rules []cp.EvidenceRule) *testpilotspb.InstructionNode {
		return b.with(c.GetId(), rules)
	}
	if len(item.GetWhen()) == 0 {
		return cp.Fixed{Node: node}, nil
	}
	var keys []string
	for _, class := range item.GetWhen() {
		keys = append(keys, a.classKey(class))
	}
	return cp.WhenOnPath{Keys: keys, Node: node}, nil
}

// activation is who runs a script, as a fresh entrypoint for each Case that carries it.
func (a *adapter) activation(s *umpirespb.Script) (func() *testpilotspb.Entrypoint, error) {
	switch act := s.GetActivation().(type) {
	case *umpirespb.Script_Controller:
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}
		}, nil
	case *umpirespb.Script_Workflow:
		workflowType := a.w.name(act.Workflow.GetWorkflowType())
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{
				WorkflowType: workflowType, WorkerRoleId: act.Workflow.GetWorker(), TaskQueueRoleId: act.Workflow.GetTaskQueue()}}}
		}, nil
	case *umpirespb.Script_NexusHandler:
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{
				Service: act.NexusHandler.GetService(), Operation: act.NexusHandler.GetOperation(),
				WorkerRoleId: act.NexusHandler.GetWorker(), TaskQueueRoleId: act.NexusHandler.GetTaskQueue()}}}
		}, nil
	case *umpirespb.Script_Activity:
		activityType := a.w.name(act.Activity.GetActivityType())
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{
				ActivityType: activityType, WorkerRoleId: act.Activity.GetWorker(), TaskQueueRoleId: act.Activity.GetTaskQueue()}}}
		}, nil
	default:
		return nil, errorAt(s.GetPosition(), "script %s is activated by nothing this reader lowers", s.GetId())
	}
}

func (a *adapter) command(s *umpirespb.Script, c *umpirespb.Command) (built, error) {
	// The commands of an activity's script are the answers to its attempts, and Testpilot admits no
	// other instruction there.
	if s.GetActivity() != nil && c.GetFinish() == nil && c.GetAttemptFailure() == nil && c.GetAttemptCanceled() == nil {
		return built{}, errorAt(c.GetPosition(), "command %s of activity script %s is no answer to an attempt: an attempt ends with a result or a failure",
			c.GetId(), s.GetId())
	}
	instruction, err := a.instruction(s, c)
	if err != nil {
		return built{}, err
	}
	var opts []cp.NodeOption
	if c.GetTimeoutMs() > 0 {
		opts = append(opts, cp.TimeoutMilliseconds(c.GetTimeoutMs()))
	}
	if c.GetRegardless() {
		opts = append(opts, cp.Guard(cp.Literal(cp.Bool(true))))
	}
	// A command reads a learned text only once it is bound: it runs where every one it reads is present.
	if reads := learnedBy(c); len(reads) > 0 {
		present := make([]*testpilotspb.Expression, len(reads))
		for i, id := range reads {
			present[i] = cp.Present(slot(id))
		}
		guard := present[0]
		if len(present) > 1 {
			guard = &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: present}}}
		}
		opts = append(opts, cp.Guard(guard))
	}
	node := cp.Node(c.GetId(), instruction, opts...)
	if c.GetAfter() != nil {
		node.After = &testpilotspb.After{}
		for _, id := range c.GetAfter().GetCommands() {
			node.After.Instructions = append(node.After.Instructions, &testpilotspb.InstructionReference{EntrypointId: s.GetId(), InstructionId: id})
		}
	}
	return built{node}, nil
}

func slot(id string) *testpilotspb.Expression {
	return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Reference{Reference: &testpilotspb.Reference{
		Reference: &testpilotspb.Reference_SlotId{SlotId: id}}}}
}

// learnedBy is the learned texts a command's operands read, each once, in the order it reads them.
func learnedBy(c *umpirespb.Command) []string {
	var out []string
	var read func(o *umpirespb.Operand)
	read = func(o *umpirespb.Operand) {
		switch k := o.GetKind().(type) {
		case *umpirespb.Operand_LearnedValue:
			if !slices.Contains(out, k.LearnedValue) {
				out = append(out, k.LearnedValue)
			}
		case *umpirespb.Operand_Path:
			read(k.Path.GetOf())
		case *umpirespb.Operand_Present:
			read(k.Present.GetOf())
		case *umpirespb.Operand_Equal:
			read(k.Equal.GetLeft())
			read(k.Equal.GetRight())
		case *umpirespb.Operand_All:
			for _, operand := range k.All.GetOperands() {
				read(operand)
			}
		case *umpirespb.Operand_Greater:
			read(k.Greater.GetLeft())
			read(k.Greater.GetRight())
		case *umpirespb.Operand_Not:
			read(k.Not.GetOf())
		default:
		}
	}
	for _, as := range c.GetRpc().GetAssign() {
		read(as.GetValue())
	}
	for _, as := range c.GetPoll().GetAssign() {
		read(as.GetValue())
	}
	read(c.GetPoll().GetUntil())
	read(c.GetFinish().GetResult())
	return out
}

func (a *adapter) instruction(s *umpirespb.Script, c *umpirespb.Command) (*testpilotspb.Instruction, error) {
	at := c.GetPosition()
	switch in := c.GetInstruction().(type) {
	case *umpirespb.Command_Rpc:
		return a.rpc(c, in.Rpc)
	case *umpirespb.Command_Poll:
		return a.poll(c, in.Poll)
	case *umpirespb.Command_AwaitLearned:
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: in.AwaitLearned}}}, nil
	case *umpirespb.Command_AwaitCommand:
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{
			Instruction: &testpilotspb.InstructionReference{EntrypointId: s.GetId(), InstructionId: in.AwaitCommand}}}}, nil
	case *umpirespb.Command_Finish:
		result, err := a.operand(at, in.Finish.GetResult(), nil)
		if err != nil {
			return nil, err
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: result}}}, nil
	case *umpirespb.Command_Fault:

		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{
			RoleId: in.Fault.GetRole(), Kind: faultKinds[in.Fault.GetKind()]}}}, nil
	case *umpirespb.Command_WorkflowCommand:
		command := &commandpb.Command{}
		if err := a.w.into(in.WorkflowCommand.GetCommand(), command); err != nil {
			return nil, err
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: command}}}, nil
	case *umpirespb.Command_NexusReply:
		reply := &testpilotspb.NexusHandlerReply{HandleSlotId: in.NexusReply.GetBinds()}
		response, failed := &nexuspb.StartOperationResponse{}, &nexuspb.HandlerError{}
		switch written := in.NexusReply.GetReply(); protoreflect.FullName(written.GetMessage()) {
		case response.ProtoReflect().Descriptor().FullName():
			if err := a.w.into(written, response); err != nil {
				return nil, err
			}
			reply.Reply = &testpilotspb.NexusHandlerReply_Response{Response: response}
		case failed.ProtoReflect().Descriptor().FullName():
			if err := a.w.into(written, failed); err != nil {
				return nil, err
			}
			reply.Reply = &testpilotspb.NexusHandlerReply_Error{Error: failed}
		default:
			return nil, errorAt(at, "command %s answers with %s; a Nexus handler answers with a start response or a handler error", c.GetId(), written.GetMessage())
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusHandlerReply{NexusHandlerReply: reply}}, nil
	case *umpirespb.Command_NexusCompletion:
		return a.nexusCompletion(c, in.NexusCompletion)
	case *umpirespb.Command_Hold, *umpirespb.Command_Release:
		control := controlOf(a.r, c)
		if !heldByDriver(control) {
			// No instruction holds or releases a delivery, and a realization that declares one is reported
			// as a gap and never produced; the command writes nothing to check.
			return &testpilotspb.Instruction{}, nil
		}
		// A hold waits until the Driver holds what the control's step dispatched to its queue, and a
		// release delivers it: both are the Driver's delivery control of the control's task-queue role.
		kind := testpilotspb.FAULT_KIND_DELIVERY_HOLD
		if c.GetRelease() != "" {
			kind = testpilotspb.FAULT_KIND_DELIVERY_RELEASE
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{
			RoleId: control.GetRole(), Kind: kind}}}, nil
	case *umpirespb.Command_AttemptCanceled:
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptCancellation{
			ActivityAttemptCancellation: &testpilotspb.ActivityAttemptCancellation{}}}, nil
	case *umpirespb.Command_AttemptFailure:
		return a.attemptFailure(c, in.AttemptFailure)
	default:
		return nil, errorAt(at, "command %s is no instruction this reader lowers", c.GetId())
	}
}

// nexusCompletion completes the Nexus operation a handle names, with the payload or the failure a
// command writes out.
func (a *adapter) nexusCompletion(c *umpirespb.Command, in *umpirespb.NexusCompletion) (*testpilotspb.Instruction, error) {
	completion := &testpilotspb.NexusOperationCompletion{HandleSlotId: in.GetHandle()}
	payload, failure := &commonpb.Payload{}, &failurepb.Failure{}
	switch written := in.GetResult(); protoreflect.FullName(written.GetMessage()) {
	case payload.ProtoReflect().Descriptor().FullName():
		if err := a.w.into(written, payload); err != nil {
			return nil, err
		}
		completion.Result = &testpilotspb.NexusOperationCompletion_Payload{Payload: payload}
	case failure.ProtoReflect().Descriptor().FullName():
		if err := a.w.into(written, failure); err != nil {
			return nil, err
		}
		completion.Result = &testpilotspb.NexusOperationCompletion_Failure{Failure: failure}
	default:
		return nil, errorAt(c.GetPosition(), "command %s completes with %s; a Nexus operation completes with a payload or a failure", c.GetId(), written.GetMessage())
	}
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusOperationCompletion{NexusOperationCompletion: completion}}, nil
}

// attemptFailure fails an activity's attempt with the failure a command writes out: what an activity
// itself can return, which is what Testpilot offers the server and all it admits.
func (a *adapter) attemptFailure(c *umpirespb.Command, in *umpirespb.AttemptFailure) (*testpilotspb.Instruction, error) {
	failure := &failurepb.Failure{}
	if err := a.w.into(in.GetFailure(), failure); err != nil {
		return nil, err
	}
	message := failure.ProtoReflect()
	if info := message.WhichOneof(message.Descriptor().Oneofs().ByName("failure_info")); info != nil && failure.GetApplicationFailureInfo() == nil {
		return nil, errorAt(c.GetPosition(), "command %s fails its attempt with a %s; an attempt fails with an application failure or one that names no kind",
			c.GetId(), info.Name())
	}
	return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_ActivityAttemptFailure{
		ActivityAttemptFailure: &testpilotspb.ActivityAttemptFailure{Failure: failure}}}, nil
}

// assignments builds a request's assignments, each a field of the request the value fits.
func (a *adapter) assignments(c *umpirespb.Command, request protoreflect.MessageDescriptor, assign []*umpirespb.Assignment) ([]*testpilotspb.RequestAssignment, error) {
	at := c.GetPosition()
	var out []*testpilotspb.RequestAssignment
	for _, as := range assign {
		target, err := walk(at, request, as.GetTarget())
		if err != nil {
			return nil, err
		}
		if target.fanned || target.field.IsList() {
			return nil, errorAt(at, "command %s assigns %s, which is no single field of %s", c.GetId(), as.GetTarget(), request.FullName())
		}
		if err := fits(at, c.GetId(), target.field, as.GetValue()); err != nil {
			return nil, err
		}
		value, err := a.operand(at, as.GetValue(), nil)
		if err != nil {
			return nil, err
		}
		out = append(out, cp.Assign(as.GetTarget(), value))
	}
	return out, nil
}

// fits rejects a value written into a request field of another kind: a learned value, an environment
// binding and the run id are texts, and a literal is of the kind it is written as.
func fits(at *umpirespb.Position, command string, fd protoreflect.FieldDescriptor, o *umpirespb.Operand) error {
	crossed := func(written string) error {
		return errorAt(at, "command %s assigns %s to %s, which is of kind %s", command, written, fd.FullName(), fd.Kind())
	}
	text := func(written string) error {
		if fd.Kind() != protoreflect.StringKind {
			return crossed(written)
		}
		return nil
	}
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_LearnedValue:
		return text("the learned text " + k.LearnedValue)
	case *umpirespb.Operand_Environment:
		return text("the environment binding " + k.Environment)
	case *umpirespb.Operand_Run:
		return text("the run id")
	case *umpirespb.Operand_Literal:
		switch v := k.Literal.GetKind().(type) {
		case *umpirespb.ProtoValue_Text, *umpirespb.ProtoValue_Named:
			return text("a text")
		case *umpirespb.ProtoValue_Flag:
			if fd.Kind() != protoreflect.BoolKind {
				return crossed("a flag")
			}
		case *umpirespb.ProtoValue_Number:
			if _, err := number(at, fd, v.Number); err != nil {
				return crossed("a number")
			}
		case *umpirespb.ProtoValue_EnumName:
			if fd.Kind() != protoreflect.EnumKind {
				return crossed("an enum value")
			}
			if fd.Enum().Values().ByName(protoreflect.Name(v.EnumName)) == nil {
				return errorAt(at, "command %s: %s has no value %s", command, fd.Enum().FullName(), v.EnumName)
			}
		default:
		}
	default:
	}
	return nil
}

func (a *adapter) rpc(c *umpirespb.Command, rpc *umpirespb.Rpc) (*testpilotspb.Instruction, error) {
	at := c.GetPosition()
	method, err := methodNamed(at, rpc.GetMethod())
	if err != nil {
		return nil, err
	}
	assignments, err := a.assignments(c, method.Input(), rpc.GetAssign())
	if err != nil {
		return nil, err
	}
	var reads []*testpilotspb.ResponseRead
	for _, read := range rpc.GetReads() {
		end, err := walk(at, method.Output(), read.GetPath())
		if err != nil {
			return nil, err
		}
		cardinality := testpilotspb.READ_CARDINALITY_ONE
		if read.GetCardinality() == umpirespb.ResponseRead_CARDINALITY_EACH {
			cardinality = testpilotspb.READ_CARDINALITY_EMIT_EACH
			if !end.fanned && !end.field.IsList() {
				return nil, errorAt(at, "command %s reads each element of %s, which is one value of %s", c.GetId(), read.GetPath(), method.Output().FullName())
			}
		}
		var targets []*testpilotspb.ReadTarget
		for _, target := range read.GetTargets() {
			lowered, err := a.target(c, read.GetPath(), end, target)
			if err != nil {
				return nil, err
			}
			targets = append(targets, lowered)
		}
		reads = append(reads, cp.ResponseRead(read.GetPath(), cardinality, targets...))
	}
	return cp.InvokeRPC(rpc.GetRole(), rpc.GetMethod(), assignments, reads), nil
}

// target is where a read value goes, which the value at the end of the read's path must fit: an
// observation of its message, a learned text, or a lift of history events.
func (a *adapter) target(c *umpirespb.Command, path string, end reached, target *umpirespb.Target) (*testpilotspb.ReadTarget, error) {
	at := c.GetPosition()
	switch tg := target.GetTarget().(type) {
	case *umpirespb.Target_Observe:
		// An observation whose own message the descriptors do not have was reported where it is declared.
		if want := a.observed[tg.Observe]; want != nil && (end.message() == nil || end.message().FullName() != want.FullName()) {
			return nil, errorAt(at, "command %s observes %s into %s, which is a %s, and the path reads %s", c.GetId(), path,
				tg.Observe, want.FullName(), end.field.FullName())
		}
		return cp.ObservationTarget(tg.Observe), nil
	case *umpirespb.Target_Bind:
		if end.fanned || end.field.IsList() || end.field.Kind() != protoreflect.StringKind {
			return nil, errorAt(at, "command %s binds the learned text %s from %s, which is no single text", c.GetId(), tg.Bind, path)
		}
		return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_SlotId{SlotId: tg.Bind}}, nil
	case *umpirespb.Target_Lift:
		if end.message() == nil || end.message().FullName() != historyEventMessage {
			return nil, errorAt(at, "command %s lifts evidence from %s, which reads no history event", c.GetId(), path)
		}
		return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{
			CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{ObservationId: tg.Lift}}}, nil
	default:
		return nil, errorAt(at, "command %s reads %s into nothing", c.GetId(), path)
	}
}

func (a *adapter) poll(c *umpirespb.Command, poll *umpirespb.Poll) (*testpilotspb.Instruction, error) {
	at := c.GetPosition()
	e := a.evidence[poll.GetEvidence()]
	read := e.GetRead()
	if read == nil {
		read = e.GetSingle()
	}
	method, err := methodNamed(at, read.GetMethod())
	if err != nil {
		return nil, err
	}
	assignments, err := a.assignments(c, method.Input(), poll.GetAssign())
	if err != nil {
		return nil, err
	}
	if outside := outsideTheElement(poll.GetUntil()); outside != "" {
		return nil, errorAt(at, "command %s polls until a condition that reads %s; a poll's condition reads only the value the poll is looking at",
			c.GetId(), outside)
	}
	switch computes, err := umpiremodel.TypeOf(poll.GetUntil(), a.element[poll.GetEvidence()], lifted(at)); {
	case err != nil:
		return nil, mistyped(at, err, "command %s polls until a condition that", c.GetId())
	case computes.Shape != umpiremodel.AnyShape && computes.Shape != umpiremodel.ConditionShape:
		return nil, errorAt(at, "command %s polls until %s, and a poll's condition is a condition", c.GetId(), computes.Shape)
	default:
	}
	until, err := a.operand(at, poll.GetUntil(), a.element[poll.GetEvidence()])
	if err != nil {
		return nil, err
	}
	return cp.ReadEvidence(poll.GetEvidence(), poll.GetRole(), assignments, until, poll.GetIntervalMs()), nil
}

// literal is a value an operand writes out, as the expression of it.
func (a *adapter) literal(at *umpirespb.Position, written *umpirespb.ProtoValue) (*testpilotspb.Expression, error) {
	switch v := written.GetKind().(type) {
	case *umpirespb.ProtoValue_Text:
		return cp.Literal(cp.Text(v.Text)), nil
	case *umpirespb.ProtoValue_Named:
		return cp.Literal(cp.Text(a.w.name(v.Named))), nil
	case *umpirespb.ProtoValue_Flag:
		return cp.Literal(cp.Bool(v.Flag)), nil
	case *umpirespb.ProtoValue_Number:
		return cp.Literal(cp.SignedInteger(v.Number)), nil
	case *umpirespb.ProtoValue_EnumName:
		return cp.Literal(cp.Enum(v.EnumName)), nil
	default:
		return nil, errorAt(at, "a literal operand is a text, a flag, a number, an enum value or a name")
	}
}

// outsideTheElement is what an operand reads beside the value a poll is looking at and what is
// written out, or empty: Testpilot evaluates a poll's condition over one element of the response, where
// neither the run, nor its environment, nor what it has learned is in reach.
func outsideTheElement(o *umpirespb.Operand) string {
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Run:
		return "the run's id"
	case *umpirespb.Operand_Environment:
		return "the environment binding " + k.Environment
	case *umpirespb.Operand_LearnedValue:
		return "the learned value " + k.LearnedValue
	case *umpirespb.Operand_Path:
		return outsideTheElement(k.Path.GetOf())
	case *umpirespb.Operand_Present:
		return outsideTheElement(k.Present.GetOf())
	case *umpirespb.Operand_Equal:
		if left := outsideTheElement(k.Equal.GetLeft()); left != "" {
			return left
		}
		return outsideTheElement(k.Equal.GetRight())
	case *umpirespb.Operand_All:
		for _, operand := range k.All.GetOperands() {
			if outside := outsideTheElement(operand); outside != "" {
				return outside
			}
		}
		return ""
	case *umpirespb.Operand_Greater:
		if left := outsideTheElement(k.Greater.GetLeft()); left != "" {
			return left
		}
		return outsideTheElement(k.Greater.GetRight())
	case *umpirespb.Operand_Not:
		return outsideTheElement(k.Not.GetOf())
	default:
		return ""
	}
}

// operand is a value a command computes, as the expression that computes it. A path read out of the
// value a poll is looking at is checked against that value's message, projected.
func (a *adapter) operand(at *umpirespb.Position, o *umpirespb.Operand, projected protoreflect.MessageDescriptor) (*testpilotspb.Expression, error) {
	if o.GetPosition().GetFile() != "" {
		at = o.GetPosition()
	}
	switch k := o.GetKind().(type) {
	case *umpirespb.Operand_Literal:
		return a.literal(at, k.Literal)
	case *umpirespb.Operand_Environment:
		return cp.Environment(k.Environment), nil
	case *umpirespb.Operand_Run:
		return cp.Run(), nil
	case *umpirespb.Operand_LearnedValue:
		return slot(k.LearnedValue), nil
	case *umpirespb.Operand_Projected:
		return cp.ProjectedValue(), nil
	case *umpirespb.Operand_Path:
		if k.Path.GetOf().GetProjected() != nil && projected != nil {
			if _, err := walk(at, projected, k.Path.GetPath()); err != nil {
				return nil, err
			}
		}
		of, err := a.operand(at, k.Path.GetOf(), projected)
		if err != nil {
			return nil, err
		}
		return cp.Path(of, k.Path.GetPath()), nil
	case *umpirespb.Operand_Present:
		of, err := a.operand(at, k.Present.GetOf(), projected)
		if err != nil {
			return nil, err
		}
		return cp.Present(of), nil
	case *umpirespb.Operand_Equal:
		left, right, err := a.sides(at, k.Equal.GetLeft(), k.Equal.GetRight(), projected)
		if err != nil {
			return nil, err
		}
		return cp.Equal(left, right), nil
	case *umpirespb.Operand_All:
		all := &testpilotspb.AllExpression{}
		for _, operand := range k.All.GetOperands() {
			lowered, err := a.operand(at, operand, projected)
			if err != nil {
				return nil, err
			}
			all.Operands = append(all.Operands, lowered)
		}
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: all}}, nil
	case *umpirespb.Operand_Greater:
		left, right, err := a.sides(at, k.Greater.GetLeft(), k.Greater.GetRight(), projected)
		if err != nil {
			return nil, err
		}
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Compare{Compare: &testpilotspb.CompareExpression{
			Operator: testpilotspb.COMPARISON_OPERATOR_GREATER_THAN, Left: left, Right: right}}}, nil
	case *umpirespb.Operand_Not:
		of, err := a.operand(at, k.Not.GetOf(), projected)
		if err != nil {
			return nil, err
		}
		return &testpilotspb.Expression{Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: of}}}, nil
	default:
		return nil, errorAt(at, "an operand of no known kind")
	}
}

// sides is the two operands of a comparison, as the expressions that compute them.
func (a *adapter) sides(at *umpirespb.Position, left, right *umpirespb.Operand, projected protoreflect.MessageDescriptor) (l, r *testpilotspb.Expression, err error) {
	if l, err = a.operand(at, left, projected); err != nil {
		return nil, nil, err
	}
	if r, err = a.operand(at, right, projected); err != nil {
		return nil, nil, err
	}
	return l, r, nil
}
