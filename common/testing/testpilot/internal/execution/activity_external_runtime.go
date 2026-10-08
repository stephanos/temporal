package execution

import (
	enumspb "go.temporal.io/api/enums/v1"
	"go.temporal.io/api/workflowservice/v1"
	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot/contract"
	"go.temporal.io/server/common/testing/testpilot/internal/ir"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
)

func externalText(message proto.Message, field string) string {
	if ir.IsNil(message) {
		return ""
	}
	m := message.ProtoReflect()
	f := m.Descriptor().Fields().ByName(protoreflect.Name(field))
	if f == nil || f.Kind() != protoreflect.StringKind {
		return ""
	}
	return m.Get(f).String()
}

func (s *valueStore) externalPublication(id contract.ReservationIdentity, attempt *testpilotspb.ActivityAttempt) (string, func() error, error) {
	for _, b := range s.program.external {
		if b.held == nil || id.Origin.EntrypointID != b.controller.id || id.Origin.InstructionID != b.carrier.source.InstructionId || id.EntrypointID != b.source.ActivityEntrypointId || id.Ordinal != b.source.ReservationOrdinal {
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

func (s *valueStore) checkExternalCarrierBatch(n *node, batch *valueBatch) error {
	if batch.outcome.GetStatus() != testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.program.external {
		if n != b.carrier {
			continue
		}
		run := batch.writes[b.runSlot].GetTextValue()
		if run == "" {
			return ir.Invalid(ir.Malformed, "activity_external_carrier", "carrier must return its actual nonempty activity run")
		}
		if value := s.slots[b.source.PendingSlotId]; value != nil {
			pending := &testpilotspb.ActivityAttempt{}
			if anypb.UnmarshalTo(value.GetMessageValue(), pending, proto.UnmarshalOptions{}) != nil || pending.ActivityRunId != run {
				return ir.Invalid(ir.Malformed, "activity_external_carrier", "learned carrier run differs from published pending delivery")
			}
		}
	}
	return nil
}

func (s *valueStore) finishExternalPublication(slot string, successful bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	value := s.pendingWrites[slot]
	delete(s.pendingWrites, slot)
	if !successful {
		return nil
	}
	if s.sealed || value == nil || s.slots[slot] != nil {
		return ir.Invalid(ir.Malformed, "activity_publication", "publication was not committed exactly once")
	}
	s.slots[slot] = value
	close(s.changed)
	s.changed = make(chan struct{})
	return nil
}

func (s *valueStore) admitExternalRequest(n *node, request proto.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if externalByID(n) && externalText(request, "workflow_id") == "" {
		found := false
		for _, b := range s.program.external {
			if n == b.answer {
				found = true
			}
		}
		if !found {
			return ir.Invalid(ir.Malformed, "activity_external_request", "standalone ByID answer has no admitted settlement basis")
		}
	}
	for _, b := range s.program.external {
		if n == b.carrier {
			if s.externalRequests == nil {
				s.externalRequests = map[*node]proto.Message{}
			}
			s.externalRequests[n] = proto.Clone(request)
			continue
		}
		if n != b.held && n != b.cancel && n != b.answer && n != b.settlement && n != b.cleanup {
			continue
		}
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, "activity_external_request", detail) }
		if ir.IsNil(request) || !ir.SameMessage(request.ProtoReflect().Descriptor(), n.method.Input()) {
			return invalid("exact typed external request required")
		}
		carrier := s.externalRequests[b.carrier]
		run := s.slots[b.runSlot].GetTextValue()
		if run == "" || externalText(request, "namespace") != externalText(carrier, "namespace") || externalText(request, "activity_id") != externalText(carrier, "activity_id") || externalText(request, "run_id") != run || externalText(request, "workflow_id") != "" {
			return invalid("external request differs from actual standalone activity identity")
		}
		if b.held != nil && n != b.cleanup {
			pending := &testpilotspb.ActivityAttempt{}
			value := s.slots[b.source.PendingSlotId]
			if value == nil || anypb.UnmarshalTo(value.GetMessageValue(), pending, proto.UnmarshalOptions{}) != nil || pending.ActivityRunId != run || pending.NamespaceName != externalText(request, "namespace") || pending.ActivityId != externalText(request, "activity_id") {
				return invalid("external effect requires a successfully published actual pending identity")
			}
			if n == b.held && b.cancel != nil && !s.externalSucceeded[b.cancel] {
				return invalid("held cancellation read requires successful RequestCancel")
			}
			if n == b.answer && !s.externalSucceeded[b.held] {
				return invalid("ByID answer requires successful held-state read")
			}
		}
		if n == b.settlement && !s.externalSucceeded[b.answer] {
			return invalid("settlement requires successful ByID answer")
		}
		if s.externalRequests == nil {
			s.externalRequests = map[*node]proto.Message{}
		}
		s.externalRequests[n] = proto.Clone(request)
	}
	return nil
}

func (s *valueStore) checkExternalReceipt(n *node, result contract.EffectResult) error {
	if result.Outcome.GetStatus() != testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.program.external {
		if n != b.held && n != b.settlement {
			continue
		}
		invalid := func(detail string) error { return ir.Invalid(ir.Malformed, "activity_external_receipt", detail) }
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
			state := enumspb.PENDING_ACTIVITY_STATE_STARTED
			if b.cancel != nil {
				state = enumspb.PENDING_ACTIVITY_STATE_CANCEL_REQUESTED
			}
			if info.GetStatus() != enumspb.ACTIVITY_EXECUTION_STATUS_RUNNING || info.GetRunState() != state || info.GetCloseTime() != nil {
				return invalid("Describe did not positively prove the required held state")
			}
			continue
		}
		status := enumspb.ACTIVITY_EXECUTION_STATUS_FAILED
		switch externalMethod(b.answer) {
		case "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCanceledById":
			status = enumspb.ACTIVITY_EXECUTION_STATUS_CANCELED
		case "temporal.api.workflowservice.v1.WorkflowService.RespondActivityTaskCompletedById":
			status = enumspb.ACTIVITY_EXECUTION_STATUS_COMPLETED
		}
		if info.GetStatus() != status || info.GetRunState() != enumspb.PENDING_ACTIVITY_STATE_UNSPECIFIED || info.GetCloseTime() == nil || info.GetCloseTime().CheckValid() != nil || info.GetCloseTime().AsTime().UnixNano() <= 0 {
			return invalid("Describe did not positively prove the requested terminal settlement")
		}
		if b.cancel != nil {
			cancel := s.externalRequests[b.cancel]
			answer := &workflowservice.RespondActivityTaskCanceledByIdRequest{}
			bytes, err := proto.Marshal(s.externalRequests[b.answer])
			if err != nil {
				return err
			}
			if err = proto.Unmarshal(bytes, answer); err != nil {
				return err
			}
			canceled := response.GetOutcome().GetFailure().GetCanceledFailureInfo()
			if canceled == nil || externalText(cancel, "identity") == "" || canceled.GetIdentity() != externalText(cancel, "identity") || !proto.Equal(canceled.GetDetails(), answer.GetDetails()) {
				return invalid("canceled outcome must retain RequestCancel identity and ByID requested details")
			}
		}
	}
	return nil
}

func (s *valueStore) markExternalSuccess(n *node, outcome *testpilotspb.InstructionOutcome) {
	if outcome.GetStatus() != testpilotspb.INSTRUCTION_OUTCOME_STATUS_SUCCEEDED {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.program.external {
		if n == b.held || n == b.cancel || n == b.answer {
			if s.externalSucceeded == nil {
				s.externalSucceeded = map[*node]bool{}
			}
			s.externalSucceeded[n] = true
		}
	}
}
