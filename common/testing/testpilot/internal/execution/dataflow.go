package execution

import (
	"context"
	"fmt"
	"maps"
	"math"
	"slices"

	celpb "cel.dev/expr"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	cel "go.temporal.io/server/common/testing/testpilot/cel"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/emptypb"
)

// opcodeRow is everything the runtime knows about one Opcode: the Instruction oneof arm that
// declares it, the entrypoint kind it runs in, whether an activity entrypoint runs it as well,
// whether its outcome carries a protocol code, how admission binds the instruction and its
// dataflow, and how the scheduler accepts its effect. A nil dataflow binds no Program expression; a
// nil accept is an instruction no controller runs.
type opcodeRow struct {
	arm          protoreflect.Name
	context      contract.EntrypointKind
	activity     bool
	protocolCode bool
	bind         func(a *admission, g *graph, i int, n *node) error
	dataflow     func(a *admission, g *graph, n *node, scope map[ir.Reference]ir.Binding, bind scopedBind) error
	accept       func(s *scheduler, ctx context.Context, task scheduledNode, c contract.Coordinate, n *node, request proto.Message) (contract.EffectHandle, contract.HandleBridge, error)
}

// scopedBind binds one Program expression of a node under its guard, in the given scope.
type scopedBind func(scope map[ir.Reference]ir.Binding, value *testpilotspb.Expression, field string, expected *ir.Type) (*ir.Expression, error)

// opcodes is indexed by Opcode; the zero Opcode's row is empty.
var opcodes [contract.MaxOpcode + 1]opcodeRow

// The table is assigned here rather than in its declaration because its rows reach
// InstructionOpcode, which reads it, and Go rejects that as an initialization cycle.
func init() {
	opcodes = [...]opcodeRow{
		contract.InvokeRPC: {
			arm: "invoke_rpc", context: contract.ControllerEntrypoint, protocolCode: true,
			bind: (*admission).bindRPC,
			dataflow: func(a *admission, g *graph, n *node, scope map[ir.Reference]ir.Binding, bind scopedBind) error {
				return a.bindRequestDataflow(g, n, n.source.Instruction.GetInvokeRpc().GetRequestAssignments(), scope, bind)
			},
			accept: (*scheduler).acceptRPC,
		},
		contract.AwaitSlot: {
			arm: "await_slot", context: contract.ControllerEntrypoint,
			bind: func(a *admission, g *graph, _ int, n *node) error { return a.bindAwaitSlot(g, n) },
			dataflow: func(a *admission, g *graph, n *node, _ map[ir.Reference]ir.Binding, _ scopedBind) error {
				return a.bindAwaitSlotDataflow(g, n)
			},
			accept: (*scheduler).acceptAwaitSlot,
		},
		contract.Await: {
			arm: "await_instruction", context: contract.WorkflowEntrypoint,
			bind: func(a *admission, g *graph, _ int, n *node) error { return a.bindAwait(g, n) },
		},
		contract.Finish: {
			arm: "finish", context: contract.WorkflowEntrypoint, activity: true,
			bind: func(a *admission, g *graph, _ int, n *node) error { return a.bindFinish(g, n) },
			dataflow: func(_ *admission, _ *graph, n *node, scope map[ir.Reference]ir.Binding, bind scopedBind) error {
				var err error
				n.input, err = bind(scope, n.source.Instruction.GetFinish().Result, "instruction.finish.result", nil)
				return err
			},
		},
		contract.InjectFault: {
			arm: "inject_fault", context: contract.ControllerEntrypoint,
			bind:   func(a *admission, g *graph, _ int, n *node) error { return a.bindFault(g, n) },
			accept: (*scheduler).acceptFault,
		},
		contract.WorkflowCommand: {
			arm: "workflow_command", context: contract.WorkflowEntrypoint,
			bind: func(a *admission, g *graph, _ int, n *node) error { return a.bindWorkflowCommand(g, n) },
		},
		contract.NexusHandlerReply: {
			arm: "nexus_handler_reply", context: contract.NexusHandlerEntrypoint,
			bind: (*admission).bindNexusHandlerReply,
		},
		contract.NexusOperationCompletion: {
			arm: "nexus_operation_completion", context: contract.ControllerEntrypoint, protocolCode: true,
			bind: func(a *admission, g *graph, _ int, n *node) error { return a.bindNexusOperationCompletion(g, n) },
			dataflow: func(_ *admission, g *graph, n *node, scope map[ir.Reference]ir.Binding, _ scopedBind) error {
				if !scope[ir.Reference{Kind: ir.SlotReference, ID: n.source.Instruction.GetNexusOperationCompletion().GetHandleSlotId()}].Available {
					return ir.Invalid(ir.Unavailable, nodePath(g, n), "completion requires successful AwaitSlot dependency")
				}
				return nil
			},
			accept: (*scheduler).acceptOperationCompletion,
		},
		contract.ReadEvidence: {
			arm: "read_evidence", context: contract.ControllerEntrypoint, protocolCode: true,
			bind: func(a *admission, g *graph, _ int, n *node) error { return a.bindReadEvidence(g, n) },
			dataflow: func(a *admission, g *graph, n *node, scope map[ir.Reference]ir.Binding, bind scopedBind) error {
				return a.bindRequestDataflow(g, n, n.source.Instruction.GetReadEvidence().GetRequestAssignments(), scope, bind)
			},
			accept: (*scheduler).acceptReadEvidence,
		},
		contract.ActivityAttemptFailure: {
			arm: "activity_attempt_failure", context: contract.ActivityEntrypoint,
			bind: func(_ *admission, g *graph, _ int, n *node) error { return bindAttemptFailure(g, n) },
		},
		contract.ActivityAttemptCancellation: {
			arm: "activity_attempt_cancellation", context: contract.ActivityEntrypoint,
			// The canceled answer carries nothing to bind.
			bind: func(*admission, *graph, int, *node) error { return nil },
		},
		contract.ActivityAttemptWithholding: {
			arm: "activity_attempt_withholding", context: contract.ActivityEntrypoint,
			bind: func(_ *admission, g *graph, _ int, n *node) error {
				switch n.source.Instruction.GetActivityAttemptWithholding().GetMode() {
				case testpilotspb.ACTIVITY_WITHHOLDING_MODE_CONTEXT, testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING:
					return nil
				default:
					return ir.Invalid(ir.Unsupported, nodePath(g, n), "unknown activity withholding mode")
				}
			},
		},
		contract.ActivityHeartbeat: {
			arm: "activity_heartbeat", context: contract.ActivityEntrypoint,
			bind: func(a *admission, g *graph, _ int, n *node) error {
				details := n.source.Instruction.GetActivityHeartbeat().GetDetails()
				if int64(proto.Size(details)) > a.prepared.limits.GetMaxRequestBytes() {
					return ir.Invalid(ir.Unsupported, nodePath(g, n), "activity heartbeat details exceed the request byte ceiling")
				}
				for _, payload := range details.GetPayloads() {
					if payload == nil {
						return ir.Invalid(ir.Malformed, nodePath(g, n), "activity heartbeat details contain a missing payload")
					}
				}
				return nil
			},
		},
	}
}

// runsIn reports whether an entrypoint of the kind runs the Opcode: an activity's script ends the
// way a workflow's does, so the one instruction that ends a workflow also completes an activity attempt.
func (r opcodeRow) runsIn(kind contract.EntrypointKind) bool {
	return r.context == kind || r.activity && kind == contract.ActivityEntrypoint
}

var instructionArms = (&testpilotspb.Instruction{}).ProtoReflect().Descriptor().Oneofs().ByName("instruction")

// InstructionOpcode is the single mapping from a declared instruction to the Opcode a Profile must
// authorize. Profile derivation reads it through the facade so a Case's instructions and the
// Opcodes that authorize them cannot drift apart.
func InstructionOpcode(instruction *testpilotspb.Instruction) contract.Opcode {
	if instruction == nil {
		return 0
	}
	arm := instruction.ProtoReflect().WhichOneof(instructionArms)
	if arm == nil {
		return 0
	}
	for opcode, row := range opcodes {
		if opcode != 0 && row.arm == arm.Name() {
			return contract.Opcode(opcode)
		}
	}
	return 0
}
func scalarSchema(kind testpilotspb.ScalarKind) *testpilotspb.ValueType {
	return &testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Scalar{Scalar: &testpilotspb.ScalarType{Kind: kind}}}}}
}
func (a *admission) bindInstructions() error {
	var err error
	if a.outcomeTypes.status, err = a.prepared.catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Enumeration{Enumeration: &testpilotspb.NamedType{ProtobufType: string(testpilotspb.InstructionOutcomeStatus(0).Descriptor().FullName())}}}}}); err != nil {
		return err
	}
	if a.outcomeTypes.text, err = a.prepared.catalog.BindType(scalarSchema(testpilotspb.SCALAR_KIND_TEXT)); err != nil {
		return err
	}
	if a.outcomeTypes.any, err = a.prepared.catalog.BindType(&testpilotspb.ValueType{Shape: &testpilotspb.ValueType_Singular{Singular: &testpilotspb.SingularType{Type: &testpilotspb.SingularType_Any{Any: &emptypb.Empty{}}}}}); err != nil {
		return err
	}
	for _, g := range a.prepared.graphs {
		for i, n := range g.nodes {
			if err := a.bindInstruction(g, i, n); err != nil {
				return err
			}
		}
	}
	return nil
}
func (a *admission) bindInstruction(g *graph, i int, n *node) error {
	n.opcode = InstructionOpcode(n.source.Instruction)
	row := opcodes[n.opcode]
	if n.opcode == 0 || !row.runsIn(g.context) {
		return ir.Invalid(ir.Unsupported, nodePath(g, n), "unsupported instruction context or Driver capability")
	}
	if !a.opcodes[n.opcode] {
		return ir.Invalid(ir.Unsupported, nodePath(g, n), "instruction "+string(row.arm)+" the Profile does not authorize")
	}
	if err := a.bindNodeBounds(g, n); err != nil {
		return err
	}
	a.bindOutcomes(g, n)
	return row.bind(a, g, i, n)
}

func (a *admission) bindAwaitSlot(g *graph, n *node) error {
	if _, exists := a.prepared.slots[n.source.Instruction.GetAwaitSlot().GetSlotId()]; !exists {
		return ir.Invalid(ir.Unknown, nodePath(g, n), "AwaitSlot requires a declared Slot")
	}
	return nil
}

func (a *admission) bindAwaitSlotDataflow(g *graph, n *node) error {
	if _, exists := a.writers[n.source.Instruction.GetAwaitSlot().SlotId]; !exists {
		return ir.Invalid(ir.Unavailable, nodePath(g, n), "awaited Slot has no writer")
	}
	return nil
}

func (a *admission) bindFinish(g *graph, n *node) error {
	finish := n.source.Instruction.GetFinish()
	if finish == nil {
		return ir.Invalid(ir.Malformed, nodePath(g, n), "nil Finish")
	}
	return nil
}

// bindAttemptFailure admits the failure an activity attempt fails with. The worker answers through
// the SDK, which reports an activity's own failure as an application failure, retried unless that
// failure says otherwise, so a failure of any other kind, or one carrying a field the Driver
// cannot set, rejects naming the field.
func bindAttemptFailure(g *graph, n *node) error {
	path := expressionPath(g, n, "instruction.activity_attempt_failure.failure")
	failure := n.source.Instruction.GetActivityAttemptFailure().GetFailure()
	if failure == nil {
		return ir.Invalid(ir.Malformed, path, "an attempt failure carries the failure")
	}
	if failure.GetFailureInfo() != nil && failure.GetApplicationFailureInfo() == nil {
		reflection := failure.ProtoReflect()
		info := reflection.WhichOneof(reflection.Descriptor().Oneofs().ByName("failure_info"))
		return ir.Invalid(ir.Unsupported, path+"."+string(info.Name()), "an activity attempt fails with an application failure")
	}
	return checkReach(failure, path)
}

// bindAwait admits an Await of an earlier Nexus or activity schedule of the same entrypoint. A
// scheduled command's result is whatever payload the handler or the activity answered, so its VALUE
// is that payload, whole.
func (a *admission) bindAwait(g *graph, n *node) error {
	reference := n.source.Instruction.GetAwaitInstruction().GetInstruction()
	dependency, exists := g.index[reference.GetInstructionId()]
	if !exists || !n.ancestors[dependency] {
		return ir.Invalid(ir.Unavailable, nodePath(g, n), "Await requires an earlier local instruction")
	}
	started := g.nodes[dependency].source.Instruction
	if !startsAwaitable(started) {
		return ir.Invalid(ir.TypeMismatch, nodePath(g, n), "Await requires a Nexus or activity schedule command")
	}
	n.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE] = a.outcomeTypes.any
	return nil
}

// bindNodeBounds resolves a node's limits: each one the Case writes, or the Program's declared default
// or else the Profile's where it writes none, within the Profile's ceilings. A hinted wait's timeout is the sum of its hints,
// scaled by the Profile's bound scale.
func (a *admission) bindNodeBounds(g *graph, n *node) error {
	bounds := n.source.GetLimits()
	var err error
	n.timeoutMilliseconds, n.maxAttempts, err = a.prepared.instructionDefaults.Resolve(bounds)
	if err != nil {
		return ir.Invalid(ir.Malformed, expressionPath(g, n, "limits.timeout"), err.Error())
	}
	if len(n.source.GetWaitHints()) > 0 {
		if err := a.bindWaitHints(g, n); err != nil {
			return err
		}
	}
	if (bounds.GetTimeout() == nil && n.timeoutMilliseconds == 0) || (bounds.GetMaxAttempts() == 0 && n.maxAttempts == 0) {
		return ir.Invalid(ir.Malformed, nodePath(g, n), "instruction writes no limit that neither the Program nor the Profile has a default for")
	}
	// A hinted timeout is scaled, so it is checked against the scaled ceiling; every other one is
	// unscaled and checked against the declared ceiling, so a scale admits no Case it would refuse.
	ceiling := a.durationCeiling(g, a.declaredLimits)
	if len(n.source.GetWaitHints()) > 0 {
		ceiling = a.durationCeiling(g, a.prepared.limits)
	}
	if n.timeoutMilliseconds <= 0 || n.timeoutMilliseconds > ceiling || n.maxAttempts <= 0 || n.maxAttempts > a.prepared.limits.MaxAttempts {
		return ir.Invalid(ir.LimitExceeded, nodePath(g, n), "instruction bounds exceed Profile ceilings")
	}
	return nil
}

// durationCeiling is the duration ceiling of limits for an instruction of g.
func (a *admission) durationCeiling(g *graph, limits *testpilotspb.ProgramLimits) int64 {
	if g.cleanup {
		return limits.CleanupDuration.AsDuration().Milliseconds()
	}
	return limits.MaxDuration.AsDuration().Milliseconds()
}

// bindWaitHints admits the hints a polling ReadEvidence waits within. The node writes its own
// timeout, the sum of their bounds, so a reported bound is the declared one and no Profile default
// stands in for it; the timeout it runs under is that sum scaled, within the scaled ceiling.
func (a *admission) bindWaitHints(g *graph, n *node) error {
	hints := n.source.GetWaitHints()
	if n.opcode != contract.ReadEvidence {
		return ir.Invalid(ir.Unsupported, expressionPath(g, n, "wait_hints"), "only a polling ReadEvidence waits within wait hints")
	}
	if n.source.Instruction.GetReadEvidence().GetInterval() == nil {
		return ir.Invalid(ir.Malformed, expressionPath(g, n, "wait_hints"), "a read once does not wait, so no wait hint bounds it")
	}
	timeoutPath := expressionPath(g, n, "limits.timeout")
	if n.source.GetLimits().GetTimeout() == nil {
		return ir.Invalid(ir.Malformed, timeoutPath, "a hinted wait writes its own timeout; no Profile default applies to it")
	}
	if err := a.charge(int64(len(hints))); err != nil {
		return err
	}
	var sum int64
	for i, hint := range hints {
		path := fmt.Sprintf("%s[%d]", expressionPath(g, n, "wait_hints"), i)
		if !ir.ValidID(hint.GetHintId()) {
			return ir.Invalid(ir.Malformed, path+".hint_id", "wait hint requires a valid identity")
		}
		if hint.GetSource().GetPath() == "" || hint.GetSource().GetLine() < 1 {
			return ir.Invalid(ir.Malformed, path+".source", "wait hint requires the source path and line it is declared at")
		}
		bound, err := ir.DurationMilliseconds(path+".at_most", hint.GetAtMost())
		if err != nil {
			return err
		}
		if bound <= 0 {
			return ir.Invalid(ir.Malformed, path+".at_most", "wait hint requires a positive bound")
		}
		if bound > math.MaxInt64-sum {
			return ir.Invalid(ir.LimitExceeded, path+".at_most", "wait hint bounds overflow their sum")
		}
		sum += bound
	}
	declared, err := ir.DurationMilliseconds(timeoutPath, n.source.GetLimits().GetTimeout())
	if err != nil {
		return err
	}
	if declared != sum {
		return ir.Invalid(ir.Malformed, timeoutPath, fmt.Sprintf("hinted wait timeout %d ms is not %d ms, the sum of its wait hints' bounds", declared, sum))
	}
	scale := a.prepared.boundScale
	n.timeoutMilliseconds = scale.Apply(declared)
	ceilingKind := "total"
	if g.cleanup {
		ceilingKind = "cleanup"
	}
	if ceiling := a.durationCeiling(g, a.prepared.limits); n.timeoutMilliseconds > ceiling {
		if scale.Scaled() {
			return ir.Invalid(ir.LimitExceeded, timeoutPath, fmt.Sprintf("scaled wait bound %d ms (%d ms declared, scaled by %d%%) exceeds the scaled Profile %s duration ceiling %d ms", n.timeoutMilliseconds, declared, scale.Percent(), ceilingKind, ceiling))
		}
		return ir.Invalid(ir.LimitExceeded, timeoutPath, fmt.Sprintf("wait bound %d ms exceeds the Profile %s duration ceiling %d ms", declared, ceilingKind, ceiling))
	}
	return nil
}
func (a *admission) bindRPC(g *graph, i int, n *node) error {
	rpc := n.source.Instruction.GetInvokeRpc()
	if rpc == nil {
		return ir.Invalid(ir.Malformed, nodePath(g, n), "nil RPC")
	}
	if err := a.role(rpc.EndpointRoleId, testpilotspb.ROLE_KIND_ENDPOINT); err != nil {
		return err
	}
	if !a.methods[rpc.EndpointRoleId][rpc.Method] {
		return ir.Invalid(ir.Unsupported, nodePath(g, n), "unauthorized RPC method")
	}
	method, err := a.prepared.catalog.Method(rpc.Method)
	if err != nil {
		return err
	}
	n.method = method
	return a.bindResponseReads(g, i, n)
}

// A fault names the task-queue role whose worker the Driver stops or resumes, or whose delivery it
// holds or releases; the role's own resource binding identifies the queue, so the instruction
// carries no queue of its own.
func (a *admission) bindFault(g *graph, n *node) error {
	fault := n.source.Instruction.GetInjectFault()
	if fault == nil || fault.Kind < testpilotspb.FAULT_KIND_WORKER_STOP || fault.Kind > testpilotspb.FAULT_KIND_ADMISSION_RESPONSE_LOSS {
		return ir.Invalid(ir.Malformed, nodePath(g, n), "fault injection requires a known fault kind")
	}
	// The role check does not go through a.role: a fault aimed at the wrong role kind is a
	// malformed instruction, not an unknown role reference.
	if a.roles[fault.RoleId] != testpilotspb.ROLE_KIND_TASK_QUEUE {
		return ir.Invalid(ir.Malformed, nodePath(g, n), "fault injection requires a declared task-queue role")
	}
	// A delivery control reaches inside the server, which only some environments can: one that
	// cannot refuses the actuator here, naming the instruction, before any Driver is asked.
	if fault.Kind >= testpilotspb.FAULT_KIND_DELIVERY_HOLD && !a.deliveryControl {
		return ir.Invalid(ir.Unsupported, nodePath(g, n), "the Profile's environment supplies no delivery control")
	}
	return nil
}

// bindOutcomes gives a node the outcome fields its instruction produces: every instruction a status and
// a detail; a controller protocol effect its protocol code; a workflow or Nexus-handler instruction its
// SDK failure code; and an awaited Nexus operation its result as the value. A Case declares none of
// them. An RPC response is read only through response reads, and a Finish result or a
// NexusHandlerReply ends its activation, so neither is an outcome value.
func (a *admission) bindOutcomes(g *graph, n *node) {
	n.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS] = a.outcomeTypes.status
	if opcodes[n.opcode].protocolCode {
		n.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_PROTOCOL_CODE] = a.outcomeTypes.text
	} else if g.context != contract.ControllerEntrypoint {
		n.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_SDK_FAILURE_CODE] = a.outcomeTypes.text
	}
	// Other controller instructions (AwaitSlot, InjectFault) have neither code.
	n.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_DETAIL] = a.outcomeTypes.text
	// An Await's VALUE is typed by the start it awaits, in bindAwait.
}
func (a *admission) addWriter(id string, writer slotWriter) error {
	if _, exists := a.writers[id]; exists {
		return ir.Invalid(ir.Malformed, "slots", "Slot has multiple writers")
	}
	a.writers[id] = writer
	return nil
}
func (a *admission) bindResponseReads(g *graph, index int, n *node) error {
	output, err := messageType(a.prepared.catalog, n.method.Output())
	if err != nil {
		return err
	}
	seen := map[string]bool{}
	var events int64
	for read, source := range n.source.Instruction.GetInvokeRpc().ResponseReads {
		if source == nil || len(source.Targets) == 0 {
			return ir.Invalid(ir.Malformed, nodePath(g, n), "response read requires a path and targets")
		}
		path, err := a.prepared.catalog.BindPath(output, expressionPath(g, n, fmt.Sprintf("instruction.invoke_rpc.response_reads[%d].path", read)), source.Path, a.expressionLimits())
		if err != nil {
			return err
		}
		typ := path.Type()
		count := int64(1)
		switch path.Fanout() {
		case false:
		case true:
			if typ.Cardinality() != ir.Repeated {
				return ir.Invalid(ir.TypeMismatch, nodePath(g, n), "EmitEach requires repeated values")
			}
			typ = typ.Element()
			count = a.prepared.limits.MaxPathFanout
		default:
			return ir.Invalid(ir.Unknown, nodePath(g, n), "unknown response read cardinality")
		}
		lifts, emits, err := a.bindReadTargets(g, index, n, read, source, path, typ, seen)
		if err != nil {
			return err
		}
		if emits {
			if count > a.prepared.limits.MaxInstructionEmittedEvents-events {
				return ir.Invalid(ir.LimitExceeded, nodePath(g, n), "response read emission exceeds instruction bound")
			}
			events += count
		}
		n.responseReads = append(n.responseReads, responseRead{path: path, emitEach: path.Fanout(), targets: source.Targets, lifts: lifts})
	}
	return nil
}
func (a *admission) bindReadTargets(g *graph, index int, n *node, read int, source *testpilotspb.ResponseRead, path *ir.Path, typ ir.Type, seen map[string]bool) ([]*evidenceLift, bool, error) {
	emits := false
	lifts := make([]*evidenceLift, len(source.Targets))
	for i, readTarget := range source.Targets {
		if readTarget == nil || ir.IsNil(readTarget.Target) {
			return nil, false, ir.Invalid(ir.Malformed, nodePath(g, n), "missing response read target")
		}
		var target ir.Type
		var exists bool
		var key string
		switch destination := readTarget.Target.(type) {
		case *testpilotspb.ReadTarget_SlotId:
			key = "slot:" + destination.SlotId
			target, exists = a.prepared.slots[destination.SlotId]
			if path.Fanout() {
				return nil, false, ir.Invalid(ir.Unsupported, nodePath(g, n), "EmitEach cannot repeatedly assign an immutable Slot")
			}
			if err := a.addWriter(destination.SlotId, slotWriter{graph: g, node: index, optional: path.MayBeAbsent()}); err != nil {
				return nil, false, err
			}
		case *testpilotspb.ReadTarget_ObservationId:
			key = "observation:" + destination.ObservationId
			target, exists = a.observations[destination.ObservationId]
			emits = true
		case *testpilotspb.ReadTarget_CorrelatedEvidence:
			location := expressionPath(g, n, fmt.Sprintf("instruction.invoke_rpc.response_reads[%d].targets[%d].correlated_evidence", read, i))
			lift, err := a.bindEvidenceLift(g, n, location, destination.CorrelatedEvidence, typ)
			if err != nil {
				return nil, false, err
			}
			lifts[i], emits = lift, true
			key = "observation:" + lift.observationID
			if seen[key] {
				return nil, false, ir.Invalid(ir.Malformed, nodePath(g, n), "conflicting response read targets")
			}
			seen[key] = true
			continue
		default:
			return nil, false, ir.Invalid(ir.Unsupported, nodePath(g, n), "unknown response read target")
		}
		if !exists || target.Opaque() || !typ.Equal(target) {
			return nil, false, ir.Invalid(ir.TypeMismatch, nodePath(g, n), "response read type differs from declared target")
		}
		if seen[key] {
			return nil, false, ir.Invalid(ir.Malformed, nodePath(g, n), "conflicting response read targets")
		}
		seen[key] = true
	}

	return lifts, emits, nil
}

// bindEvidenceLift type-checks one declared CorrelatedEvidence lift against the value being projected.
// The target Observation must be the exact CorrelatedEvidence message the correlated capability decodes, and
// every bound path must read a scalar the portable evidence domain admits, so a lift that cannot
// produce decodable evidence rejects at Prepare rather than at the first recorded event.
func (a *admission) bindEvidenceLift(g *graph, n *node, location string, source *testpilotspb.CorrelatedEvidenceProjection, typ ir.Type) (*evidenceLift, error) {
	target, exists := a.observations[source.GetObservationId()]
	if !exists || target.Cardinality() != ir.Singular || !ir.SameMessage(target.Message(), (&testpilotspb.CorrelatedEvidence{}).ProtoReflect().Descriptor()) {
		return nil, ir.Invalid(ir.TypeMismatch, nodePath(g, n), "evidence lift requires an exact declared CorrelatedEvidence Observation")
	}
	if typ.Cardinality() != ir.Singular || typ.Message() == nil || typ.Opaque() || typ.Any() {
		return nil, ir.Invalid(ir.TypeMismatch, nodePath(g, n), "evidence lift requires a singular message read")
	}
	if len(source.GetEvidenceIds()) == 0 {
		return nil, ir.Invalid(ir.Malformed, nodePath(g, n), "evidence lift requires at least one rule")
	}
	// A source ordinal is dense per Run and only the emitting instruction counts it, so one source
	// belongs to one instruction on an entrypoint that activates exactly once. A worker entrypoint
	// activates per task and a second instruction would restart the count, and either would be
	// rejected by the verifier's ordering rather than here.
	if g.context != contract.ControllerEntrypoint {
		return nil, ir.Invalid(ir.Unsupported, nodePath(g, n), "evidence lift requires a controller entrypoint")
	}
	lift := &evidenceLift{observationID: source.GetObservationId(), element: typ}
	owner := contract.Coordinate{EntrypointID: g.id, InstructionID: n.source.InstructionId}
	for index, id := range source.GetEvidenceIds() {
		bound, err := a.bindDeclaredRule(g, n, fmt.Sprintf("%s.evidence_ids[%d]", location, index), id, typ)
		if err != nil {
			return nil, err
		}
		if claimed, exists := a.evidenceSources[bound.source]; exists && claimed != owner {
			return nil, ir.Invalid(ir.Malformed, nodePath(g, n), "evidence source is already lifted by another instruction")
		}
		a.evidenceSources[bound.source] = owner
		lift.rules = append(lift.rules, *bound)
	}
	return lift, nil
}

// evidenceKeyKinds are the scalars an operation key may read: text, or an integer in its canonical
// decimal spelling.
var evidenceKeyKinds = append([]testpilotspb.ScalarKind{testpilotspb.SCALAR_KIND_TEXT}, evidenceIntegerKinds...)

// evidenceFieldKinds are the scalars a lifted evidence field may read; the portable evidence domain
// admits text, unsigned integer and boolean, and every integer kind narrows into an unsigned integer.
var evidenceFieldKinds = append([]testpilotspb.ScalarKind{
	testpilotspb.SCALAR_KIND_TEXT, testpilotspb.SCALAR_KIND_BOOLEAN,
}, evidenceIntegerKinds...)

// evidenceIntegerKinds narrow into the portable evidence domain's unsigned integer.
var evidenceIntegerKinds = []testpilotspb.ScalarKind{
	testpilotspb.SCALAR_KIND_INT32, testpilotspb.SCALAR_KIND_INT64, testpilotspb.SCALAR_KIND_UINT32,
	testpilotspb.SCALAR_KIND_UINT64, testpilotspb.SCALAR_KIND_SINT32, testpilotspb.SCALAR_KIND_SINT64,
	testpilotspb.SCALAR_KIND_FIXED32, testpilotspb.SCALAR_KIND_FIXED64, testpilotspb.SCALAR_KIND_SFIXED32,
	testpilotspb.SCALAR_KIND_SFIXED64,
}

// bindEvidenceBindings binds the named expressions at location, rejecting at errorPath. Each is a
// text literal or a path read directly from the projected value; the lift reads its paths itself,
// so no other expression is admitted.

func (a *admission) bindEvidenceExpression(errorPath string, typ ir.Type, guard *testpilotspb.Expression, location string, source *testpilotspb.Expression, kinds ...testpilotspb.ScalarKind) (*ir.Expression, error) {
	projected := map[ir.Reference]ir.Binding{{Kind: ir.ProjectedValueReference}: {Type: typ, Available: true}}
	bound, err := a.prepared.catalog.BindConditionedExpression([]ir.Condition{{Expression: guard, Path: errorPath + ".guard", Matches: true}}, ir.Site{Context: ir.EvidenceLiftContext, Path: location}, source, nil, projected, a.expressionLimits())
	if err != nil {
		return nil, err
	}
	read := bound.Type()
	if read.Cardinality() != ir.Singular || read.Message() != nil || read.Enum() != nil || !slices.Contains(kinds, read.Scalar()) {
		return nil, ir.Invalid(ir.TypeMismatch, errorPath, "evidence binding reads an unsupported scalar")
	}
	if literal := bound.Literal(); literal != nil {
		if text, ok := literal.Kind.(*celpb.Value_StringValue); ok && text.StringValue == "" {
			return nil, ir.Invalid(ir.Malformed, errorPath, "evidence literal binding requires a value")
		}
	}
	return bound, nil
}

func (a *admission) bindEvidenceBindings(errorPath, location string, typ ir.Type, guard *testpilotspb.Expression, sources []*testpilotspb.NamedExpression, kinds ...testpilotspb.ScalarKind) ([]evidenceBinding, error) {
	bound := make([]evidenceBinding, 0, len(sources))
	seen := map[string]bool{}
	for index, source := range sources {
		if source == nil || !ir.ValidID(source.GetFieldId()) || seen[source.GetFieldId()] {
			return nil, ir.Invalid(ir.Malformed, errorPath, "evidence binding requires one unique declared field")
		}
		seen[source.FieldId] = true
		value, err := a.bindEvidenceExpression(errorPath, typ, guard, fmt.Sprintf("%s[%d].value", location, index), source.Value, kinds...)
		if err != nil {
			return nil, err
		}
		bound = append(bound, evidenceBinding{fieldID: source.FieldId, value: value})
	}
	return bound, nil
}

func (a *admission) scope(g *graph, n *node) map[ir.Reference]ir.Binding {
	scope := map[ir.Reference]ir.Binding{}
	for id, typ := range a.prepared.slots {
		if writer, exists := a.writers[id]; exists && !typ.Opaque() && writer.graph != g &&
			(writer.graph.context != contract.ControllerEntrypoint || g.context != contract.ControllerEntrypoint) {
			continue
		}
		scope[ir.Reference{Kind: ir.SlotReference, ID: id}] = ir.Binding{Type: typ}
	}
	for index := range n.ancestors {
		previous := g.nodes[index]
		for field, typ := range previous.outcomes {
			scope[ir.Reference{Kind: ir.OutcomeReference, Entrypoint: g.id, ID: previous.source.InstructionId, Field: int32(field)}] = ir.Binding{Type: typ, Available: previous.guardSource == nil && field != testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE}
		}
	}
	if g.cleanup {
		for _, b := range a.prepared.external {
			if b.cleanup != n {
				continue
			}
			for _, previous := range []*node{b.carrier, b.settlement} {
				scope[ir.Reference{Kind: ir.OutcomeReference, Entrypoint: b.controller.id, ID: previous.source.InstructionId, Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS)}] = ir.Binding{Type: previous.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS], Available: true}
			}
			binding := scope[ir.Reference{Kind: ir.SlotReference, ID: b.runSlot}]
			binding.Available = true
			scope[ir.Reference{Kind: ir.SlotReference, ID: b.runSlot}] = binding
		}
		for _, b := range a.prepared.resets {
			if b.cleanup != n {
				continue
			}
			for _, previous := range []*node{b.carrier, b.settlement} {
				scope[ir.Reference{Kind: ir.OutcomeReference, Entrypoint: b.controller.id, ID: previous.source.InstructionId, Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS)}] = ir.Binding{Type: previous.outcomes[testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS], Available: true}
			}
			binding := scope[ir.Reference{Kind: ir.SlotReference, ID: b.runSlot}]
			binding.Available = true
			scope[ir.Reference{Kind: ir.SlotReference, ID: b.runSlot}] = binding
		}
	}
	return scope
}

// Successful dependencies establish nonoptional response reads and successful AwaitSlot readiness.
func (a *admission) successScope(g *graph, n *node, guard *ir.Expression, scope map[ir.Reference]ir.Binding) {
	for id := range successFacts(guard) {
		index, exists := g.index[id]
		if !exists || !n.ancestors[index] {
			continue
		}
		for slotID, writer := range a.writers {
			if writer.graph == g && writer.node == index && !writer.optional && !writer.asynchronous {
				reference := ir.Reference{Kind: ir.SlotReference, ID: slotID}
				binding := scope[reference]
				binding.Available = true
				scope[reference] = binding
			}
		}
		previous := g.nodes[index]
		if previous.opcode == contract.AwaitSlot {
			reference := ir.Reference{Kind: ir.SlotReference, ID: previous.source.Instruction.GetAwaitSlot().SlotId}
			binding := scope[reference]
			binding.Available = true
			scope[reference] = binding
		}
		reference := ir.Reference{Kind: ir.OutcomeReference, Entrypoint: g.id, ID: id, Field: int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_VALUE)}
		if binding, exists := scope[reference]; exists {
			binding.Available = true
			scope[reference] = binding
		}
	}
}
func successFacts(expression *ir.Expression) map[string]bool {
	result := map[string]bool{}
	if expression == nil {
		return result
	}
	children := expression.Children()
	switch expression.Operator() {
	case ir.Compare:
		if expression.Comparison() != "_==_" {
			break
		}
		for i := range 2 {
			reference := children[i].Reference()
			literal := children[1-i].Literal()
			if reference.Kind == ir.OutcomeReference && reference.Field == int32(testpilotspb.INSTRUCTION_OUTCOME_FIELD_STATUS) && proto.Equal(literal, cel.Enum(testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED)) {
				result[reference.ID] = true
			}
		}
	case ir.All, ir.Any:
		for i, child := range children {
			facts := successFacts(child)
			if expression.Operator() == ir.All || i == 0 {
				maps.Copy(result, facts)
			} else {
				for id := range result {
					if !facts[id] {
						delete(result, id)
					}
				}
			}
		}
	default:
	}
	return result
}
func (a *admission) bindDataflow() error {
	boolean, err := a.prepared.catalog.BindType(scalarSchema(testpilotspb.SCALAR_KIND_BOOLEAN))
	if err != nil {
		return err
	}
	a.runID, err = a.prepared.catalog.BindType(scalarSchema(testpilotspb.SCALAR_KIND_TEXT))
	if err != nil {
		return err
	}
	for _, g := range a.prepared.graphs {
		for _, index := range g.order {
			if err := a.bindNodeDataflow(g, g.nodes[index], boolean); err != nil {
				return err
			}
		}
		g.runtimeWork = runtimeWorkLimit(g, a.prepared.limits)
	}
	return nil
}
func (a *admission) bindNodeDataflow(g *graph, n *node, boolean ir.Type) error {
	var err error
	if err := a.charge(int64(len(a.prepared.slots)) + int64(len(n.ancestors))*6); err != nil {
		return err
	}
	scope := a.scope(g, n)
	if err := a.charge(int64(proto.Size(n.guardSource)) + 1); err != nil {
		return err
	}
	guard := ir.Condition{Expression: n.guardSource, Path: expressionPath(g, n, "guard")}
	if n.guardSource != nil {
		n.guard, err = a.prepared.catalog.BindExpression(ir.Site{Context: ir.ProgramContext, Path: guard.Path}, n.guardSource, &boolean, scope, a.expressionLimits())
		if err != nil {
			return err
		}
	}
	if err := a.charge(int64(len(successFacts(n.guard))) * (int64(len(a.writers)) + 1)); err != nil {
		return err
	}
	a.successScope(g, n, n.guard, scope)
	row := opcodes[n.opcode]
	if row.dataflow == nil {
		// A fault names its target role statically and a typed instruction carries its message
		// whole; neither binds a Program expression.
		return nil
	}
	return row.dataflow(a, g, n, scope, func(expressionScope map[ir.Reference]ir.Binding, value *testpilotspb.Expression, field string, expected *ir.Type) (*ir.Expression, error) {
		if err := a.charge(int64(proto.Size(n.guardSource)) + int64(proto.Size(value)) + 1); err != nil {
			return nil, err
		}
		site := ir.Site{Context: ir.ProgramContext, Path: expressionPath(g, n, field)}
		_, expression, err := a.prepared.catalog.BindGuardedExpression(guard, site, value, expected, expressionScope, a.expressionLimits())
		return expression, err
	})
}

// bindRequestDataflow binds the request assignments of an instruction that builds a request, in the
// node's scope plus the Run ID intrinsic that only a Program input may read.
func (a *admission) bindRequestDataflow(g *graph, n *node, sources []*testpilotspb.RequestAssignment, scope map[ir.Reference]ir.Binding, bind scopedBind) error {
	inputScope := maps.Clone(scope)
	inputScope[ir.Reference{Kind: ir.EventReference, Field: int32(testpilotspb.RUN_EVENT_FIELD_RUN_ID)}] = ir.Binding{Type: a.runID, Available: true}
	return a.bindAssignments(g, n, sources, "instruction."+string(opcodes[n.opcode].arm), func(value *testpilotspb.Expression, field string, expected *ir.Type) (*ir.Expression, error) {
		return bind(inputScope, value, field, expected)
	})
}

// expressionPath locates one expression field of an instruction node. Entrypoints and instructions are
// named by identity rather than by index, so a path stays stable when declarations are reordered.
func expressionPath(g *graph, n *node, field string) string {
	if g.cleanup {
		return fmt.Sprintf("program.cleanup.instructions[%s].%s", n.source.InstructionId, field)
	}
	return fmt.Sprintf("program.entrypoints[%s].instructions[%s].%s", g.id, n.source.InstructionId, field)
}

// bindAssignments binds the request assignments of an instruction that builds a request for
// n.method; field locates the instruction arm the assignments sit under.
func (a *admission) bindAssignments(g *graph, n *node, sources []*testpilotspb.RequestAssignment, field string, bind func(*testpilotspb.Expression, string, *ir.Type) (*ir.Expression, error)) error {
	input, err := messageType(a.prepared.catalog, n.method.Input())
	if err != nil {
		return err
	}
	for index, source := range sources {
		if source == nil {
			return ir.Invalid(ir.Malformed, nodePath(g, n), "nil request assignment")
		}
		target, err := a.prepared.catalog.BindPath(input, expressionPath(g, n, fmt.Sprintf("%s.request_assignments[%d].target", field, index)), source.Target, a.expressionLimits())
		if err != nil {
			return err
		}
		if target.Fanout() {
			return ir.Invalid(ir.Unsupported, nodePath(g, n), "assignment cannot fan out across destination elements")
		}
		for _, step := range target.Steps() {
			if step.Selector == ir.Presence {
				return ir.Invalid(ir.Unsupported, nodePath(g, n), "presence is not an assignment destination")
			}
		}
		for _, previous := range n.assignments {
			if err := a.charge(int64(len(previous.target.Steps())+len(target.Steps())) + 1); err != nil {
				return err
			}
			if previous.target.Conflicts(target) {
				return ir.Invalid(ir.Malformed, nodePath(g, n), "request assignments overlap")
			}
		}
		typ := target.Type()
		valueSource := source.Value
		if reference, ok := expressionReference(source.Value).GetReference().(*testpilotspb.Reference_EnvironmentBindingId); ok {
			if reference == nil || typ.Cardinality() != ir.Singular || typ.Scalar() != testpilotspb.SCALAR_KIND_TEXT {
				return ir.Invalid(ir.TypeMismatch, nodePath(g, n), "environment reference requires a singular text destination")
			}
			resolved, err := a.resolveEnvironment(reference.EnvironmentBindingId)
			if err != nil {
				return err
			}
			valueSource = cel.Literal(&celpb.Value{Kind: &celpb.Value_StringValue{StringValue: resolved}})
		}
		value, err := bind(valueSource, fmt.Sprintf("%s.request_assignments[%d].value", field, index), &typ)
		if err != nil {
			return err
		}
		n.assignments = append(n.assignments, assignment{target: target, value: value})
	}

	return nil
}
