package execution

import (
	"fmt"

	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const resetMethod = "temporal.api.workflowservice.v1.WorkflowService.ResetActivityExecution"

// activityResetSettlement is one admitted reset declaration: the controller's carrier, held read,
// reset request, terminal read and cleanup, and the Slot the carrier's run is learned into.
type activityResetSettlement struct {
	source                                    *testpilotspb.ActivityResetSettlement
	controller                                *graph
	carrier, held, reset, settlement, cleanup *node
	runSlot                                   string
}

func (a *admission) bindActivityResetSettlements() error {
	claimed := map[*node]bool{}
	for _, b := range a.prepared.external {
		for _, n := range []*node{b.carrier, b.held, b.answer, b.cancel, b.settlement, b.cleanup} {
			if n != nil && n != b.carrier {
				claimed[n] = true
			}
		}
	}
	for i, declaration := range a.prepared.source.GetActivityResetSettlements() {
		if err := a.charge(9); err != nil {
			return err
		}
		path := fmt.Sprintf("program.activity_reset_settlements[%d]", i)
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, path, detail) }
		resolve := func(ref *testpilotspb.InstructionReference) (*graph, *node) {
			if ref == nil {
				return nil, nil
			}
			g := a.graphIndex[ref.GetEntrypointId()]
			if g == nil {
				return nil, nil
			}
			index, ok := g.index[ref.GetInstructionId()]
			if !ok {
				return nil, nil
			}
			return g, g.nodes[index]
		}
		g, carrier := resolve(declaration.GetCarrier())
		if carrier == nil || g.cleanup || g.context != contract.ControllerEntrypoint || carrier.opcode != contract.InvokeRPC || externalMethod(carrier) != "temporal.api.workflowservice.v1.WorkflowService.StartActivityExecution" {
			return invalid("reset settlement requires an exact standalone activity carrier")
		}
		basis := &activityResetSettlement{source: declaration, controller: g, carrier: carrier}
		for _, read := range carrier.source.Instruction.GetInvokeRpc().GetResponseReads() {
			if read.GetPath() == "run_id" && read.GetCardinality() == testpilotspb.READ_CARDINALITY_ONE && len(read.GetTargets()) == 1 {
				basis.runSlot = read.GetTargets()[0].GetSlotId()
			}
		}
		if basis.runSlot == "" {
			return invalid("carrier must learn the actual activity run_id into one Slot")
		}
		for _, binding := range []struct {
			ref     *testpilotspb.InstructionReference
			target  **node
			cleanup bool
		}{
			{declaration.GetHeld(), &basis.held, false}, {declaration.GetResetRequest(), &basis.reset, false},
			{declaration.GetSettlement(), &basis.settlement, false}, {declaration.GetCleanup(), &basis.cleanup, true},
		} {
			owner, n := resolve(binding.ref)
			if n == nil || binding.cleanup != owner.cleanup || !binding.cleanup && owner != g || n.timeoutMilliseconds <= 0 || n.maxAttempts <= 0 || claimed[n] {
				return invalid("missing, duplicate or unbounded held/reset/settlement/cleanup reference")
			}
			*binding.target = n
			claimed[n] = true
		}
		if basis.reset.opcode != contract.InvokeRPC || externalMethod(basis.reset) != resetMethod {
			return invalid("exact typed ResetActivityExecution required")
		}
		if basis.cleanup.opcode != contract.InvokeRPC || externalMethod(basis.cleanup) != "temporal.api.workflowservice.v1.WorkflowService.TerminateActivityExecution" {
			return invalid("bounded Cleanup must terminate this standalone activity")
		}
		for _, read := range []*node{basis.held, basis.settlement} {
			if read.opcode != contract.ReadEvidence || externalMethod(read) != "temporal.api.workflowservice.v1.WorkflowService.DescribeActivityExecution" || read.source.Instruction.GetReadEvidence().GetUntil() == nil {
				return invalid("bounded held and terminal Describe reads required")
			}
		}
		activity := a.graphIndex[declaration.GetActivityEntrypointId()]
		ordinal, fresh := declaration.GetReservationOrdinal(), declaration.GetFreshReservationOrdinal()
		if activity == nil || activity.context != contract.ActivityEntrypoint || ordinal < 0 || fresh != ordinal+1 || fresh >= int64(len(activity.activityAttempts)) {
			return invalid("reset needs an actual held reservation and the fresh reservation after it")
		}
		for _, other := range a.prepared.resets {
			if other.source.GetActivityEntrypointId() == declaration.GetActivityEntrypointId() {
				return invalid("an activity entrypoint declares one reset numbering")
			}
		}
		group := activity.activityAttempts[ordinal]
		terminal := activity.nodes[group[len(group)-1]].source.Instruction.GetActivityAttemptWithholding()
		if terminal == nil || terminal.GetMode() != testpilotspb.ACTIVITY_WITHHOLDING_MODE_SDK_PENDING || terminal.GetExternalSettlement() != nil {
			return invalid("the held reservation must end in an SDK_PENDING disposition on its timer basis")
		}
		typ, exists := a.prepared.slots[declaration.GetPendingSlotId()]
		if !exists || typ.Cardinality() != ir.Singular || !ir.SameMessage(typ.Message(), (&testpilotspb.ActivityAttempt{}).ProtoReflect().Descriptor()) {
			return invalid("publication Slot must contain the exact ActivityAttempt message")
		}
		if err := a.addWriter(declaration.GetPendingSlotId(), slotWriter{graph: g, node: g.index[carrier.source.InstructionId], asynchronous: true}); err != nil {
			return err
		}
		for _, n := range []*node{basis.held, basis.reset, basis.settlement, basis.cleanup} {
			for _, field := range []string{"namespace", "activity_id"} {
				if externalAssignment(carrier, field) == nil || !proto.Equal(externalAssignment(carrier, field), externalAssignment(n, field)) {
					return invalid("namespace name and activity ID must be those of the carrier")
				}
			}
			if externalAssignment(n, "run_id").GetReference().GetSlotId() != basis.runSlot || externalAssignment(n, "workflow_id") != nil {
				return invalid("standalone request needs learned actual run_id and no workflow_id")
			}
		}
		for _, n := range []*node{basis.carrier, basis.held, basis.reset, basis.settlement, basis.cleanup} {
			if !externalExactMethod(n) {
				return invalid("reset settlement requires exact WorkflowService method and request/response descriptors")
			}
		}
		a.prepared.resets = append(a.prepared.resets, basis)
	}
	for _, graph := range a.prepared.graphs {
		for _, n := range graph.nodes {
			if externalMethod(n) != resetMethod || externalAssignment(n, "run_id") == nil {
				continue
			}
			if !claimed[n] {
				return ir.Invalid(ir.Malformed, nodePath(graph, n), "a reset of the learned execution requires an explicit reset settlement")
			}
		}
	}
	return nil
}

// restartOf is the reservation ordinal at which the server numbers an activity entrypoint's
// attempts from its first number again, after its declared reset; zero declares none.
func (a *admission) restartOf(entrypointID string) int64 {
	for _, b := range a.prepared.resets {
		if b.source.GetActivityEntrypointId() == entrypointID {
			return b.source.GetFreshReservationOrdinal()
		}
	}
	return 0
}

func (a *admission) checkActivityResetSettlements() error {
	for _, b := range a.prepared.resets {
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, "activity_reset_settlement", detail) }
		guard := &testpilotspb.Expression{Expression: &testpilotspb.Expression_All{All: &testpilotspb.AllExpression{Operands: []*testpilotspb.Expression{externalSuccessExpression(b.source.Carrier), {Expression: &testpilotspb.Expression_Not{Not: &testpilotspb.NotExpression{Operand: externalSuccessExpression(b.source.Settlement)}}}}}}}
		if !proto.Equal(b.cleanup.source.Guard, guard) {
			return invalid("Cleanup must follow successful carrier unless terminal Describe already succeeded")
		}
		carried := false
		for _, reservation := range b.carrier.reservations {
			if reservation.EntrypointID == b.source.ActivityEntrypointId && b.source.FreshReservationOrdinal < reservation.Count && reservation.Restart == b.source.FreshReservationOrdinal {
				carried = true
			}
		}
		if !carried {
			return invalid("reset basis does not match the carrier reservation and its declared restart")
		}
		var awaited *node
		for _, n := range b.controller.nodes {
			if n.opcode == contract.AwaitSlot && n.source.Instruction.GetAwaitSlot().GetSlotId() == b.source.PendingSlotId {
				if awaited != nil {
					return invalid("publication is awaited twice")
				}
				awaited = n
			}
		}
		if !externalRequires(b.controller, b.held, awaited, map[*node]bool{}) || !externalRequires(b.controller, b.reset, b.held, map[*node]bool{}) || !externalRequires(b.controller, b.settlement, b.reset, map[*node]bool{}) {
			return invalid("reset requires successful publication and held-state reads in order, and settlement follows it")
		}
	}
	return nil
}

func (s *valueStore) resetPublication(id contract.ReservationIdentity, attempt *testpilotspb.ActivityAttempt) (string, func() error, error) {
	for _, b := range s.program.resets {
		if id.Origin.EntrypointID != b.controller.id || id.Origin.InstructionID != b.carrier.source.InstructionId || id.EntrypointID != b.source.ActivityEntrypointId || id.Ordinal != b.source.ReservationOrdinal {
			continue
		}
		if id.Origin.RunID != s.runID || attempt.GetResponse() != testpilotspb.ACTIVITY_ATTEMPT_RESPONSE_PENDING || attempt.GetActivityRunId() == "" || attempt.GetDeliveryId() == "" || attempt.GetNamespaceName() == "" || attempt.GetActivityId() == "" {
			return "", nil, ir.Invalid(ir.Malformed, "activity_publication", "actual pending delivery identity required")
		}
		encoded, err := anypb.New(attempt)
		if err != nil {
			return "", nil, err
		}
		value := &testpilotspb.Value{Value: &testpilotspb.Value_MessageValue{MessageValue: encoded}}
		slot := b.source.PendingSlotId
		return slot, func() error {
			s.mu.Lock()
			defer s.mu.Unlock()
			if s.sealed || s.slots[slot] != nil || s.pendingWrites[slot] != nil {
				return ir.Invalid(ir.Malformed, "activity_publication", "closed or duplicate pending publication")
			}
			if learned := s.slots[b.runSlot]; learned != nil && learned.GetTextValue() != attempt.ActivityRunId {
				return ir.Invalid(ir.Malformed, "activity_publication", "pending activity run differs from learned carrier run")
			}
			carrier := s.externalRequests[b.carrier]
			if externalText(carrier, "namespace") != attempt.NamespaceName || externalText(carrier, "activity_id") != attempt.ActivityId {
				return ir.Invalid(ir.Malformed, "activity_publication", "pending namespace name or activity ID differs from carrier")
			}
			if s.pendingWrites == nil {
				s.pendingWrites = map[string]*testpilotspb.Value{}
			}
			s.pendingWrites[slot] = value
			return nil
		}, nil
	}
	return "", nil, nil
}

// admitResetRequest holds a reset settlement's requests to the actual learned execution and to the
// declared order: no reset reaches the server before the held attempt's pending record was
// published and a held Describe of the same run succeeded.
func (s *valueStore) admitResetRequest(n *node, request proto.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.program.resets {
		if n == b.carrier {
			if s.externalRequests == nil {
				s.externalRequests = map[*node]proto.Message{}
			}
			s.externalRequests[n] = proto.Clone(request)
			continue
		}
		if n != b.held && n != b.reset && n != b.settlement && n != b.cleanup {
			continue
		}
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, "activity_reset_request", detail) }
		if ir.IsNil(request) || !ir.SameMessage(request.ProtoReflect().Descriptor(), n.method.Input()) {
			return invalid("exact typed reset settlement request required")
		}
		carrier := s.externalRequests[b.carrier]
		run := s.slots[b.runSlot].GetTextValue()
		if run == "" || externalText(request, "namespace") != externalText(carrier, "namespace") || externalText(request, "activity_id") != externalText(carrier, "activity_id") || externalText(request, "run_id") != run || externalText(request, "workflow_id") != "" {
			return invalid("reset settlement request differs from actual standalone activity identity")
		}
		if n == b.held || n == b.reset {
			pending := &testpilotspb.ActivityAttempt{}
			value := s.slots[b.source.PendingSlotId]
			if value == nil || anypb.UnmarshalTo(value.GetMessageValue(), pending, proto.UnmarshalOptions{}) != nil || pending.ActivityRunId != run || pending.NamespaceName != externalText(request, "namespace") || pending.ActivityId != externalText(request, "activity_id") {
				return invalid("reset requires a successfully published actual pending identity")
			}
		}
		if n == b.reset && !s.externalSucceeded[b.held] {
			return invalid("reset requires a successful held-state read")
		}
		if n == b.settlement && !s.externalSucceeded[b.reset] {
			return invalid("settlement requires a successful reset")
		}
		if s.externalRequests == nil {
			s.externalRequests = map[*node]proto.Message{}
		}
		s.externalRequests[n] = proto.Clone(request)
	}
	return nil
}

func (s *valueStore) checkResetReceipt(n *node, result contract.EffectResult) error {
	if result.Outcome.GetStatus() != testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.program.resets {
		if n != b.held && n != b.settlement {
			continue
		}
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, "activity_reset_receipt", detail) }
		response := &workflowservice.DescribeActivityExecutionResponse{}
		if ir.IsNil(result.Response) || !ir.SameMessage(result.Response.ProtoReflect().Descriptor(), response.ProtoReflect().Descriptor()) {
			return invalid("exact Describe response required")
		}
		bytes, err := proto.Marshal(result.Response)
		if err != nil {
			return err
		}
		if err = proto.Unmarshal(bytes, response); err != nil {
			return err
		}
		info := response.GetInfo()
		request := s.externalRequests[n]
		if info == nil || info.GetRunId() != externalText(request, "run_id") || info.GetActivityId() != externalText(request, "activity_id") || response.GetRunId() != info.GetRunId() {
			return invalid("Describe response must identify the actual activity run")
		}
		if n == b.held {
			if info.GetStatus() != enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING || info.GetRunState() != enumspb.PENDING_ACTIVITY_STATE_STARTED || info.GetCloseTime() != nil {
				return invalid("Describe did not positively prove the held attempt")
			}
			continue
		}
		switch info.GetStatus() {
		case enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING, enumspb.ACTIVITY_EXECUTION_STATUS_PAUSED, enumspb.ACTIVITY_EXECUTION_STATUS_UNSPECIFIED:
			return invalid("Describe did not positively prove a terminal settlement")
		default:
		}
		if info.GetRunState() != enumspb.PENDING_ACTIVITY_STATE_UNSPECIFIED || info.GetCloseTime() == nil || info.GetCloseTime().CheckValid() != nil || info.GetCloseTime().AsTime().UnixNano() <= 0 {
			return invalid("Describe did not positively prove a terminal settlement")
		}
	}
	return nil
}

func (s *valueStore) markResetSuccess(n *node, outcome *testpilotspb.InstructionOutcome) {
	if outcome.GetStatus() != testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.program.resets {
		if n == b.held || n == b.reset {
			if s.externalSucceeded == nil {
				s.externalSucceeded = map[*node]bool{}
			}
			s.externalSucceeded[n] = true
		}
	}
}

func (s *valueStore) checkResetCarrierBatch(n *node, batch *valueBatch) error {
	if batch.outcome.GetStatus() != testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.program.resets {
		if n != b.carrier {
			continue
		}
		run := batch.writes[b.runSlot].GetTextValue()
		if run == "" {
			return ir.Invalid(ir.Malformed, "activity_reset_carrier", "carrier must return its actual nonempty activity run")
		}
		if value := s.slots[b.source.PendingSlotId]; value != nil {
			pending := &testpilotspb.ActivityAttempt{}
			if anypb.UnmarshalTo(value.GetMessageValue(), pending, proto.UnmarshalOptions{}) != nil || pending.ActivityRunId != run {
				return ir.Invalid(ir.Malformed, "activity_reset_carrier", "learned carrier run differs from published pending delivery")
			}
		}
	}
	return nil
}
