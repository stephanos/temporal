package testpilot

// A realization of the IR as the caseproducer.Realization a producer consumes. The translation is
// one declaration to one part, in the order the IR lists them: nothing here chooses what a Case
// does. What a Query's path selects among these parts is the producer's rule, the same for every
// realization.

import (
	"slices"

	commandpb "go.temporal.io/api/command/v1"
	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	nexuspb "go.temporal.io/api/nexus/v1"
	modelirspb "go.temporal.io/server/api/modelir/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cp "go.temporal.io/server/model/go/caseproducer"
	"go.temporal.io/server/model/go/umpire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// adapter translates one realization for one Case identity, checking what it writes and reads
// against the protobuf descriptors it names. It reads every declaration and reports every problem:
// one declaration that crosses a descriptor hides none after it.
type adapter struct {
	r *modelirspb.Realization
	t *umpire.Table
	// classKey is the key of one class of an action, as the Model's tables key it.
	classKey func(*modelirspb.ActionClass) string
	w        *writer
	// evidence and element are each kind of evidence and the message its recorded data is; observed
	// is each observation's message.
	evidence map[string]*modelirspb.Evidence
	element  map[string]protoreflect.MessageDescriptor
	observed map[string]protoreflect.MessageDescriptor
}

func newAdapter(r *modelirspb.Realization, t *umpire.Table, classKey func(*modelirspb.ActionClass) string, fixture string) *adapter {
	a := &adapter{r: r, t: t, classKey: classKey, w: &writer{fixture: fixture},
		evidence: map[string]*modelirspb.Evidence{}, element: map[string]protoreflect.MessageDescriptor{},
		observed: map[string]protoreflect.MessageDescriptor{}}
	for _, e := range r.GetEvidence() {
		a.evidence[e.GetId()] = e
	}
	return a
}

var roleKinds = map[modelirspb.Role_Kind]testpilotspb.RoleKind{
	modelirspb.Role_KIND_ENDPOINT:    testpilotspb.ROLE_KIND_ENDPOINT,
	modelirspb.Role_KIND_WORKER:      testpilotspb.ROLE_KIND_WORKER,
	modelirspb.Role_KIND_TASK_QUEUE:  testpilotspb.ROLE_KIND_TASK_QUEUE,
	modelirspb.Role_KIND_PARTICIPANT: testpilotspb.ROLE_KIND_PARTICIPANT,
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
		if l.GetKind() == modelirspb.Learned_KIND_HANDLE {
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

// source is one kind of evidence as a producer reads it, once its recorded message and the field
// that keys its operation are found in the descriptors it names.
func (a *adapter) source(e *modelirspb.Evidence) (*cp.EvidenceSource, error) {
	at := e.GetPosition()
	out := &cp.EvidenceSource{EventKind: e.GetRecords(), OperationKeyPath: e.GetOperation(), KindID: e.GetId(), SourceID: e.GetSource()}
	var element protoreflect.MessageDescriptor
	switch from := e.GetFrom().(type) {
	case *modelirspb.Evidence_History:
		event, err := messageNamed(at, historyEventMessage)
		if err != nil {
			return nil, err
		}
		if event.Oneofs().ByName("attributes").Fields().ByName(protoreflect.Name(from.History)) == nil {
			return nil, errorAt(at, "evidence %s: a history event has no attributes %s", e.GetId(), from.History)
		}
		element, out.Recorded = event, cp.Recorded{HistoryAttributes: from.History}
	case *modelirspb.Evidence_Read:
		method, err := methodNamed(at, from.Read.GetMethod())
		if err != nil {
			return nil, err
		}
		end, err := walk(at, method.Output(), from.Read.GetPath())
		if err != nil {
			return nil, err
		}
		if !end.field.IsList() || end.message() == nil {
			return nil, errorAt(at, "evidence %s is read from %s, which is no repeated message of %s", e.GetId(), from.Read.GetPath(), method.Output().FullName())
		}
		element, out.Recorded = end.message(), cp.Recorded{Method: from.Read.GetMethod(), Path: from.Read.GetPath()}
	default:
		return nil, errorAt(at, "evidence %s is recorded nowhere", e.GetId())
	}
	key, err := walk(at, element, e.GetOperation())
	if err != nil {
		return nil, err
	}
	if key.fanned || key.field.IsList() || key.message() != nil {
		return nil, errorAt(at, "evidence %s keys its operation by %s, which is no single scalar of %s", e.GetId(), e.GetOperation(), element.FullName())
	}
	a.element[e.GetId()] = element
	return out, nil
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

func (a *adapter) script(s *modelirspb.Script) (cp.EntrypointPlan, []cp.ActionBinding, []error) {
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
func (a *adapter) placed(s *modelirspb.Script, item *modelirspb.Item) (cp.Item, error) {
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
func (a *adapter) activation(s *modelirspb.Script) (func() *testpilotspb.Entrypoint, error) {
	switch act := s.GetActivation().(type) {
	case *modelirspb.Script_Controller:
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Controller{Controller: &testpilotspb.ControllerActivation{}}}
		}, nil
	case *modelirspb.Script_Workflow:
		workflowType := a.w.name(act.Workflow.GetWorkflowType())
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Workflow{Workflow: &testpilotspb.WorkflowActivation{
				WorkflowType: workflowType, WorkerRoleId: act.Workflow.GetWorker(), TaskQueueRoleId: act.Workflow.GetTaskQueue()}}}
		}, nil
	case *modelirspb.Script_NexusHandler:
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_NexusHandler{NexusHandler: &testpilotspb.NexusHandlerActivation{
				Service: act.NexusHandler.GetService(), Operation: act.NexusHandler.GetOperation(),
				WorkerRoleId: act.NexusHandler.GetWorker(), TaskQueueRoleId: act.NexusHandler.GetTaskQueue()}}}
		}, nil
	case *modelirspb.Script_Activity:
		activityType := a.w.name(act.Activity.GetActivityType())
		return func() *testpilotspb.Entrypoint {
			return &testpilotspb.Entrypoint{Activation: &testpilotspb.Entrypoint_Activity{Activity: &testpilotspb.ActivityActivation{
				ActivityType: activityType, WorkerRoleId: act.Activity.GetWorker(), TaskQueueRoleId: act.Activity.GetTaskQueue()}}}
		}, nil
	default:
		return nil, errorAt(s.GetPosition(), "script %s is activated by nothing this reader lowers", s.GetId())
	}
}

func (a *adapter) command(s *modelirspb.Script, c *modelirspb.Command) (built, error) {
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
func learnedBy(c *modelirspb.Command) []string {
	var out []string
	var read func(o *modelirspb.Operand)
	read = func(o *modelirspb.Operand) {
		switch k := o.GetKind().(type) {
		case *modelirspb.Operand_LearnedValue:
			if !slices.Contains(out, k.LearnedValue) {
				out = append(out, k.LearnedValue)
			}
		case *modelirspb.Operand_Path:
			read(k.Path.GetOf())
		case *modelirspb.Operand_Present:
			read(k.Present.GetOf())
		case *modelirspb.Operand_Equal:
			read(k.Equal.GetLeft())
			read(k.Equal.GetRight())
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

func (a *adapter) instruction(s *modelirspb.Script, c *modelirspb.Command) (*testpilotspb.Instruction, error) {
	at := c.GetPosition()
	switch in := c.GetInstruction().(type) {
	case *modelirspb.Command_Rpc:
		return a.rpc(c, in.Rpc)
	case *modelirspb.Command_Poll:
		return a.poll(c, in.Poll)
	case *modelirspb.Command_AwaitLearned:
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitSlot{AwaitSlot: &testpilotspb.AwaitSlot{SlotId: in.AwaitLearned}}}, nil
	case *modelirspb.Command_AwaitCommand:
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_AwaitInstruction{AwaitInstruction: &testpilotspb.AwaitInstruction{
			Instruction: &testpilotspb.InstructionReference{EntrypointId: s.GetId(), InstructionId: in.AwaitCommand}}}}, nil
	case *modelirspb.Command_Finish:
		result, err := a.operand(at, in.Finish.GetResult(), nil)
		if err != nil {
			return nil, err
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_Finish{Finish: &testpilotspb.Finish{Result: result}}}, nil
	case *modelirspb.Command_Fault:
		kind := testpilotspb.FAULT_KIND_WORKER_STOP
		if in.Fault.GetKind() == modelirspb.Fault_KIND_WORKER_RESUME {
			kind = testpilotspb.FAULT_KIND_WORKER_RESUME
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_InjectFault{InjectFault: &testpilotspb.InjectFault{
			RoleId: in.Fault.GetRole(), Kind: kind}}}, nil
	case *modelirspb.Command_WorkflowCommand:
		command := &commandpb.Command{}
		if err := a.w.into(in.WorkflowCommand.GetCommand(), command); err != nil {
			return nil, err
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_WorkflowCommand{WorkflowCommand: &testpilotspb.WorkflowCommand{Command: command}}}, nil
	case *modelirspb.Command_NexusReply:
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
	case *modelirspb.Command_NexusCompletion:
		completion := &testpilotspb.NexusOperationCompletion{HandleSlotId: in.NexusCompletion.GetHandle()}
		payload, failure := &commonpb.Payload{}, &failurepb.Failure{}
		switch written := in.NexusCompletion.GetResult(); protoreflect.FullName(written.GetMessage()) {
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
			return nil, errorAt(at, "command %s completes with %s; a Nexus operation completes with a payload or a failure", c.GetId(), written.GetMessage())
		}
		return &testpilotspb.Instruction{Instruction: &testpilotspb.Instruction_NexusOperationCompletion{NexusOperationCompletion: completion}}, nil
	case *modelirspb.Command_Hold, *modelirspb.Command_Release:
		// No instruction holds or releases a delivery, and a realization that declares one is reported
		// as a gap and never produced; the command writes nothing to check.
		return &testpilotspb.Instruction{}, nil
	default:
		return nil, errorAt(at, "command %s is no instruction this reader lowers", c.GetId())
	}
}

// assignments builds a request's assignments, each a field of the request the value fits.
func (a *adapter) assignments(c *modelirspb.Command, request protoreflect.MessageDescriptor, assign []*modelirspb.Assignment) ([]*testpilotspb.RequestAssignment, error) {
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
func fits(at *modelirspb.Position, command string, fd protoreflect.FieldDescriptor, o *modelirspb.Operand) error {
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
	case *modelirspb.Operand_LearnedValue:
		return text("the learned text " + k.LearnedValue)
	case *modelirspb.Operand_Environment:
		return text("the environment binding " + k.Environment)
	case *modelirspb.Operand_Run:
		return text("the run id")
	case *modelirspb.Operand_Literal:
		switch v := k.Literal.GetKind().(type) {
		case *modelirspb.ProtoValue_Text, *modelirspb.ProtoValue_Named:
			return text("a text")
		case *modelirspb.ProtoValue_Flag:
			if fd.Kind() != protoreflect.BoolKind {
				return crossed("a flag")
			}
		case *modelirspb.ProtoValue_Number:
			if _, err := number(at, fd, v.Number); err != nil {
				return crossed("a number")
			}
		case *modelirspb.ProtoValue_EnumName:
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

func (a *adapter) rpc(c *modelirspb.Command, rpc *modelirspb.Rpc) (*testpilotspb.Instruction, error) {
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
		if read.GetCardinality() == modelirspb.ResponseRead_CARDINALITY_EACH {
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
func (a *adapter) target(c *modelirspb.Command, path string, end reached, target *modelirspb.Target) (*testpilotspb.ReadTarget, error) {
	at := c.GetPosition()
	switch tg := target.GetTarget().(type) {
	case *modelirspb.Target_Observe:
		// An observation whose own message the descriptors do not have was reported where it is declared.
		if want := a.observed[tg.Observe]; want != nil && (end.message() == nil || end.message().FullName() != want.FullName()) {
			return nil, errorAt(at, "command %s observes %s into %s, which is a %s, and the path reads %s", c.GetId(), path,
				tg.Observe, want.FullName(), end.field.FullName())
		}
		return cp.ObservationTarget(tg.Observe), nil
	case *modelirspb.Target_Bind:
		if end.fanned || end.field.IsList() || end.field.Kind() != protoreflect.StringKind {
			return nil, errorAt(at, "command %s binds the learned text %s from %s, which is no single text", c.GetId(), tg.Bind, path)
		}
		return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_SlotId{SlotId: tg.Bind}}, nil
	case *modelirspb.Target_Lift:
		if end.message() == nil || end.message().FullName() != historyEventMessage {
			return nil, errorAt(at, "command %s lifts evidence from %s, which reads no history event", c.GetId(), path)
		}
		return &testpilotspb.ReadTarget{Target: &testpilotspb.ReadTarget_CorrelatedEvidence{
			CorrelatedEvidence: &testpilotspb.CorrelatedEvidenceProjection{ObservationId: tg.Lift}}}, nil
	default:
		return nil, errorAt(at, "command %s reads %s into nothing", c.GetId(), path)
	}
}

func (a *adapter) poll(c *modelirspb.Command, poll *modelirspb.Poll) (*testpilotspb.Instruction, error) {
	at := c.GetPosition()
	e := a.evidence[poll.GetEvidence()]
	method, err := methodNamed(at, e.GetRead().GetMethod())
	if err != nil {
		return nil, err
	}
	assignments, err := a.assignments(c, method.Input(), poll.GetAssign())
	if err != nil {
		return nil, err
	}
	until, err := a.operand(at, poll.GetUntil(), a.element[poll.GetEvidence()])
	if err != nil {
		return nil, err
	}
	return cp.ReadEvidence(poll.GetEvidence(), poll.GetRole(), assignments, until, poll.GetIntervalMs()), nil
}

// operand is a value a command computes, as the expression that computes it. A path read out of the
// value a poll is looking at is checked against that value's message, projected.
func (a *adapter) operand(at *modelirspb.Position, o *modelirspb.Operand, projected protoreflect.MessageDescriptor) (*testpilotspb.Expression, error) {
	if o.GetPosition().GetFile() != "" {
		at = o.GetPosition()
	}
	switch k := o.GetKind().(type) {
	case *modelirspb.Operand_Literal:
		switch v := k.Literal.GetKind().(type) {
		case *modelirspb.ProtoValue_Text:
			return cp.Literal(cp.Text(v.Text)), nil
		case *modelirspb.ProtoValue_Named:
			return cp.Literal(cp.Text(a.w.name(v.Named))), nil
		case *modelirspb.ProtoValue_Flag:
			return cp.Literal(cp.Bool(v.Flag)), nil
		case *modelirspb.ProtoValue_Number:
			return cp.Literal(cp.SignedInteger(v.Number)), nil
		case *modelirspb.ProtoValue_EnumName:
			return cp.Literal(cp.Enum(v.EnumName)), nil
		default:
			return nil, errorAt(at, "a literal operand is a text, a flag, a number, an enum value or a name")
		}
	case *modelirspb.Operand_Environment:
		return cp.Environment(k.Environment), nil
	case *modelirspb.Operand_Run:
		return cp.Run(), nil
	case *modelirspb.Operand_LearnedValue:
		return slot(k.LearnedValue), nil
	case *modelirspb.Operand_Projected:
		return cp.ProjectedValue(), nil
	case *modelirspb.Operand_Path:
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
	case *modelirspb.Operand_Present:
		of, err := a.operand(at, k.Present.GetOf(), projected)
		if err != nil {
			return nil, err
		}
		return cp.Present(of), nil
	case *modelirspb.Operand_Equal:
		left, err := a.operand(at, k.Equal.GetLeft(), projected)
		if err != nil {
			return nil, err
		}
		right, err := a.operand(at, k.Equal.GetRight(), projected)
		if err != nil {
			return nil, err
		}
		return cp.Equal(left, right), nil
	default:
		return nil, errorAt(at, "an operand of no known kind")
	}
}
