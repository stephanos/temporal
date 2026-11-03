// Package worker reads the feature-specific table followed by reference executors.
package worker

import (
	"errors"
	"fmt"
	"slices"

	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

const ScenarioHeader = "umpire-scenario"
const Unprogrammed = "umpire.unprogrammed"

// Row is one exact observable key and its single response.
type Row struct {
	Key      *temporalv1.ActivityTaskKey
	Response *anypb.Any
}

type taskKey struct {
	scenario     string
	attempt      int32
	activityType string
	history      string
}

func exactKey(key *temporalv1.ActivityTaskKey) taskKey {
	return taskKey{scenario: key.GetScenario(), attempt: key.GetAttempt(), activityType: key.GetActivityType(), history: historyKey(key.GetHeartbeatHistory())}
}
func historyKey(history []bool) string {
	bits := make([]byte, len(history))
	for i, bit := range history {
		if bit {
			bits[i] = 1
		}
	}
	return string(bits)
}

// Program is an immutable, validated finite activity table.
type Program struct {
	Namespace     string
	TaskQueue     string
	rows          []Row
	index         map[taskKey]*anypb.Any
	maxHeartbeats int
}

// Decode refuses incomplete observable domains before any task is polled.
func Decode(data []byte) (*Program, error) {
	envelope := new(anypb.Any)
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, fmt.Errorf("decode program envelope: %w", err)
	}
	if err := noUnknownFields(envelope); err != nil {
		return nil, err
	}
	p := new(temporalv1.WorkerProgram)
	if err := envelope.UnmarshalTo(p); err != nil {
		return nil, fmt.Errorf("decode worker program: %w", err)
	}
	if err := noUnknownFields(p); err != nil {
		return nil, err
	}
	if p.GetNamespace() == "" || p.GetTaskQueue() == "" {
		return nil, errors.New("worker program requires namespace and task queue")
	}
	table, err := decodeRows(p)
	if err != nil {
		return nil, err
	}
	domains, err := table.decodeDomains(p.GetContinuationDomains())
	if err != nil {
		return nil, err
	}
	if err := table.validateClosure(domains); err != nil {
		return nil, err
	}
	return table, nil
}

func decodeRows(p *temporalv1.WorkerProgram) (*Program, error) {
	table := &Program{Namespace: p.GetNamespace(), TaskQueue: p.GetTaskQueue(), index: make(map[taskKey]*anypb.Any, len(p.GetRules()))}
	for i, r := range p.GetRules() {
		key := new(temporalv1.ActivityTaskKey)
		if err := r.GetKey().UnmarshalTo(key); err != nil {
			return nil, fmt.Errorf("rule %d key: %w", i, err)
		}
		if err := validKey(key); err != nil {
			return nil, fmt.Errorf("rule %d key: %w", i, err)
		}
		k := exactKey(key)
		if _, exists := table.index[k]; exists {
			return nil, fmt.Errorf("rule %d: duplicate exact activity key", i)
		}
		if len(r.GetRespond()) != 1 {
			return nil, fmt.Errorf("rule %d: activity requires one response", i)
		}
		response := r.GetRespond()[0]
		if _, err := ResponseMessage(response); err != nil {
			return nil, fmt.Errorf("rule %d response: %w", i, err)
		}
		response = proto.CloneOf(response)
		table.rows = append(table.rows, Row{Key: key, Response: response})
		table.index[k] = response
		table.maxHeartbeats = max(table.maxHeartbeats, len(key.GetHeartbeatHistory()))
	}
	return table, nil
}

func (table *Program) decodeDomains(packedDomains []*anypb.Any) (map[taskKey][]bool, error) {
	domains := make(map[taskKey][]bool, len(packedDomains))
	for i, packed := range packedDomains {
		domain := new(temporalv1.ActivityTaskSuccessors)
		if err := noUnknownFields(packed); err != nil {
			return nil, fmt.Errorf("successor %d envelope: %w", i, err)
		}
		if err := packed.UnmarshalTo(domain); err != nil {
			return nil, fmt.Errorf("successor %d domain: %w", i, err)
		}
		if err := noUnknownFields(domain); err != nil {
			return nil, fmt.Errorf("successor %d domain: %w", i, err)
		}
		if err := validKey(domain.GetKey()); err != nil {
			return nil, fmt.Errorf("successor %d key: %w", i, err)
		}
		k := exactKey(domain.GetKey())
		if _, exists := domains[k]; exists {
			return nil, fmt.Errorf("successor %d: duplicate declared domain", i)
		}
		if len(domain.GetCancelRequested()) == 0 || len(domain.GetCancelRequested()) > 2 {
			return nil, fmt.Errorf("successor %d: nonempty unique bool domain required", i)
		}
		if len(domain.GetCancelRequested()) == 2 && domain.GetCancelRequested()[0] == domain.GetCancelRequested()[1] {
			return nil, fmt.Errorf("successor %d: duplicate declared bit", i)
		}
		response := table.index[k]
		if response == nil || !response.MessageIs(&workflowservice.RecordActivityTaskHeartbeatRequest{}) {
			return nil, fmt.Errorf("successor %d: domain requires an exact heartbeat row", i)
		}
		domains[k] = slices.Clone(domain.GetCancelRequested())
	}
	return domains, nil
}

func (table *Program) validateClosure(domains map[taskKey][]bool) error {
	for _, row := range table.rows {
		k := exactKey(row.Key)
		if row.Response.MessageIs(&workflowservice.RecordActivityTaskHeartbeatRequest{}) {
			domain, exists := domains[k]
			if !exists {
				return errors.New("heartbeat row has no declared successor domain")
			}
			for _, bit := range domain {
				child := k
				child.history += historyKey([]bool{bit})
				if table.index[child] == nil {
					return errors.New("declared heartbeat successor has no response row")
				}
			}
		}
		history := row.Key.GetHeartbeatHistory()
		if len(history) == 0 {
			continue
		}
		parent := k
		parent.history = parent.history[:len(parent.history)-1]
		response := table.index[parent]
		if response == nil || !response.MessageIs(&workflowservice.RecordActivityTaskHeartbeatRequest{}) {
			return errors.New("activity history has no heartbeat parent")
		}
		if !slices.Contains(domains[parent], history[len(history)-1]) {
			return errors.New("activity history is outside its declared successor domain")
		}
	}
	return nil
}

func validKey(key *temporalv1.ActivityTaskKey) error {
	if key == nil || key.GetScenario() == "" || key.GetActivityType() == "" || key.GetAttempt() <= 0 {
		return errors.New("activity key requires scenario, positive attempt and activity type")
	}
	return noUnknownFields(key)
}

// Match returns a fresh response for the exact observed heartbeat history.
func (table *Program) Match(scenario string, attempt int32, activityType string, history ...bool) *anypb.Any {
	response := table.index[taskKey{scenario: scenario, attempt: attempt, activityType: activityType, history: historyKey(history)}]
	if response == nil {
		return nil
	}
	return proto.CloneOf(response)
}

// MaxHeartbeats is the finite depth proved by admission of all declared children.
func (table *Program) MaxHeartbeats() int { return table.maxHeartbeats }

// Rows returns independent copies of the admitted key/response pairs.
func (table *Program) Rows() []Row {
	rows := make([]Row, len(table.rows))
	for i, row := range table.rows {
		rows[i] = Row{Key: proto.CloneOf(row.Key), Response: proto.CloneOf(row.Response)}
	}
	return rows
}

// Responses returns independent copies for an executor's startup capability check.
func (table *Program) Responses() []*anypb.Any {
	responses := make([]*anypb.Any, 0, len(table.rows))
	for _, row := range table.rows {
		responses = append(responses, proto.CloneOf(row.Response))
	}
	return responses
}

// ResponseMessage decodes one admitted partial worker request.
func ResponseMessage(response *anypb.Any) (proto.Message, error) {
	if response == nil {
		return nil, errors.New("missing activity response")
	}
	if err := noUnknownFields(response); err != nil {
		return nil, err
	}
	var message proto.Message
	var body protoreflect.Name
	switch {
	case response.MessageIs(&workflowservice.RecordActivityTaskHeartbeatRequest{}):
		message = new(workflowservice.RecordActivityTaskHeartbeatRequest)
		body = "details"
	case response.MessageIs(&workflowservice.RespondActivityTaskCompletedRequest{}):
		message = new(workflowservice.RespondActivityTaskCompletedRequest)
		body = "result"
	case response.MessageIs(&workflowservice.RespondActivityTaskFailedRequest{}):
		message = new(workflowservice.RespondActivityTaskFailedRequest)
		body = "failure"
	case response.MessageIs(&workflowservice.RespondActivityTaskCanceledRequest{}):
		message = new(workflowservice.RespondActivityTaskCanceledRequest)
		body = "details"
	default:
		return nil, fmt.Errorf("unsupported activity response %q", response.GetTypeUrl())
	}
	if err := response.UnmarshalTo(message); err != nil {
		return nil, err
	}
	if err := noUnknownFields(message); err != nil {
		return nil, err
	}
	var unsupported error
	message.ProtoReflect().Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
		switch field.Name() {
		case body, "task_token", "namespace", "identity":
		default:
			unsupported = fmt.Errorf("unsupported activity response field %s", field.Name())
		}
		return unsupported == nil
	})
	return message, unsupported
}

func noUnknownFields(message proto.Message) error {
	if message == nil || !message.ProtoReflect().IsValid() {
		return errors.New("missing protobuf message")
	}
	return unknownFields(message.ProtoReflect())
}
func unknownFields(message protoreflect.Message) error {
	if len(message.GetUnknown()) != 0 {
		return fmt.Errorf("unknown fields in %s", message.Descriptor().FullName())
	}
	var found error
	message.Range(func(field protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case field.IsMap() && field.MapValue().Message() != nil:
			value.Map().Range(func(_ protoreflect.MapKey, item protoreflect.Value) bool {
				found = unknownFields(item.Message())
				return found == nil
			})
		case field.IsList() && field.Message() != nil:
			list := value.List()
			for i := 0; i < list.Len() && found == nil; i++ {
				found = unknownFields(list.Get(i).Message())
			}
		case !field.IsMap() && !field.IsList() && field.Message() != nil:
			found = unknownFields(value.Message())
		default:
		}
		return found == nil
	})
	return found
}

// Scenario accepts the Realization's binary/plain header encoding.
func Scenario(payload *commonpb.Payload) string {
	if payload == nil || len(payload.GetMetadata()) != 1 || string(payload.GetMetadata()["encoding"]) != "binary/plain" {
		return ""
	}
	return string(payload.GetData())
}

// UnprogrammedFailure cannot be overridden by a program's rows.
func UnprogrammedFailure() *failurepb.Failure {
	return &failurepb.Failure{Message: "no worker program rule matches the task", FailureInfo: &failurepb.Failure_ApplicationFailureInfo{ApplicationFailureInfo: &failurepb.ApplicationFailureInfo{Type: Unprogrammed, NonRetryable: true}}}
}

func successfulRPC(rpc RPC) error {
	if rpc.Stage != RPCCompleted || !rpc.Sent || !rpc.Answered {
		return errors.New("missing actual completed worker RPC")
	}
	if rpc.CallError != nil {
		return errors.New("worker RPC has an actual call error")
	}
	if rpc.TransportError != "" {
		return fmt.Errorf("worker transport failed: %s", rpc.TransportError)
	}
	if rpc.ServerError != nil {
		return fmt.Errorf("worker RPC failed with status %d", rpc.ServerError.GetCode())
	}
	if rpc.Request == nil || rpc.Response == nil {
		return errors.New("worker RPC lacks request or response")
	}
	if err := noUnknownFields(rpc.Request); err != nil {
		return err
	}
	return noUnknownFields(rpc.Response)
}

// HeartbeatResult reads a bit only from a successful actual heartbeat completion.
func HeartbeatResult(expected *anypb.Any, rpc RPC) (bool, error) {
	if err := successfulRPC(rpc); err != nil {
		return false, err
	}
	if rpc.Method != workflowservice.WorkflowService_RecordActivityTaskHeartbeat_FullMethodName {
		return false, errors.New("worker RPC is not a heartbeat completion")
	}
	request := new(workflowservice.RecordActivityTaskHeartbeatRequest)
	if err := rpc.Request.UnmarshalTo(request); err != nil {
		return false, err
	}
	if err := noUnknownFields(request); err != nil {
		return false, err
	}
	selected, err := ResponseMessage(expected)
	if err != nil {
		return false, err
	}
	chosen, ok := selected.(*workflowservice.RecordActivityTaskHeartbeatRequest)
	if !ok {
		return false, errors.New("expected worker response is not a heartbeat")
	}
	chosen.Identity = ""
	request.Identity = ""
	if !proto.Equal(chosen, request) {
		return false, errors.New("actual heartbeat request differs from chosen response")
	}
	response := new(workflowservice.RecordActivityTaskHeartbeatResponse)
	if err := rpc.Response.UnmarshalTo(response); err != nil {
		return false, err
	}
	if err := noUnknownFields(response); err != nil {
		return false, err
	}
	if response.GetActivityPaused() || response.GetActivityReset() {
		return false, errors.New("heartbeat response has unsupported pause or reset notification")
	}
	return response.GetCancelRequested(), nil
}

// TerminalResult validates the actual chosen terminal and its server acknowledgment.
func TerminalResult(expected *anypb.Any, rpc RPC) error {
	if err := successfulRPC(rpc); err != nil {
		return err
	}
	message, err := ResponseMessage(expected)
	if err != nil {
		return err
	}
	var method string
	var response proto.Message
	switch message.(type) {
	case *workflowservice.RespondActivityTaskCompletedRequest:
		method = workflowservice.WorkflowService_RespondActivityTaskCompleted_FullMethodName
		response = new(workflowservice.RespondActivityTaskCompletedResponse)
	case *workflowservice.RespondActivityTaskFailedRequest:
		method = workflowservice.WorkflowService_RespondActivityTaskFailed_FullMethodName
		response = new(workflowservice.RespondActivityTaskFailedResponse)
	case *workflowservice.RespondActivityTaskCanceledRequest:
		method = workflowservice.WorkflowService_RespondActivityTaskCanceled_FullMethodName
		response = new(workflowservice.RespondActivityTaskCanceledResponse)
	default:
		return errors.New("expected worker response is not terminal")
	}
	if rpc.Method != method || rpc.Request.GetTypeUrl() != expected.GetTypeUrl() {
		return errors.New("actual terminal kind differs from chosen response")
	}
	actual := message.ProtoReflect().Type().New().Interface()
	if err := rpc.Request.UnmarshalTo(actual); err != nil {
		return err
	}
	if err := noUnknownFields(actual); err != nil {
		return err
	}
	for _, request := range []proto.Message{message, actual} {
		reflected := request.ProtoReflect()
		var unsupported error
		reflected.Range(func(field protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			switch field.Name() {
			case "namespace", "task_token", "result", "failure", "details":
			case "identity", "worker_version", "deployment", "deployment_options":
				reflected.Clear(field)
			default:
				unsupported = fmt.Errorf("unsupported actual terminal field %s", field.Name())
			}
			return unsupported == nil
		})
		if unsupported != nil {
			return unsupported
		}
	}
	if !proto.Equal(message, actual) {
		return errors.New("actual terminal payload or assignment differs from chosen response")
	}
	if err := rpc.Response.UnmarshalTo(response); err != nil {
		return err
	}
	if err := noUnknownFields(response); err != nil {
		return err
	}
	if failed, ok := response.(*workflowservice.RespondActivityTaskFailedResponse); ok && len(failed.GetFailures()) != 0 {
		return errors.New("failed terminal acknowledgment contains validation failures")
	}
	return nil
}
