// Package worker reads the feature-specific table followed by reference executors.
package worker

import (
	"errors"
	"fmt"

	commonpb "go.temporal.io/api/common/v1"
	failurepb "go.temporal.io/api/failure/v1"
	"go.temporal.io/api/workflowservice/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	temporalv1 "umpire.local/temporal/umpire/temporal/v1"
)

const ScenarioHeader = "umpire-scenario"
const Unprogrammed = "umpire.unprogrammed"

type row struct {
	key      *temporalv1.ActivityTaskKey
	response *anypb.Any
}

// Program is an immutable, validated activity table.
type Program struct {
	Namespace string
	TaskQueue string
	rows      []row
}

// Decode refuses unknown keys and unsupported responses before any task is polled.
func Decode(data []byte) (*Program, error) {
	envelope := new(anypb.Any)
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, fmt.Errorf("decode program envelope: %w", err)
	}
	p := new(temporalv1.WorkerProgram)
	if err := envelope.UnmarshalTo(p); err != nil {
		return nil, fmt.Errorf("decode worker program: %w", err)
	}
	if p.GetNamespace() == "" || p.GetTaskQueue() == "" {
		return nil, errors.New("worker program requires namespace and task queue")
	}
	table := &Program{Namespace: p.GetNamespace(), TaskQueue: p.GetTaskQueue()}
	for i, r := range p.GetRules() {
		key := new(temporalv1.ActivityTaskKey)
		if err := r.GetKey().UnmarshalTo(key); err != nil {
			return nil, fmt.Errorf("rule %d key: %w", i, err)
		}
		if len(r.GetRespond()) != 1 {
			return nil, fmt.Errorf("rule %d: activity requires one response", i)
		}
		response := r.GetRespond()[0]
		var message proto.Message
		switch {
		case response.MessageIs(&workflowservice.RespondActivityTaskCompletedRequest{}):
			message = new(workflowservice.RespondActivityTaskCompletedRequest)
		case response.MessageIs(&workflowservice.RespondActivityTaskFailedRequest{}):
			message = new(workflowservice.RespondActivityTaskFailedRequest)
		default:
			return nil, fmt.Errorf("rule %d: unsupported activity response %q", i, response.GetTypeUrl())
		}
		if err := response.UnmarshalTo(message); err != nil {
			return nil, fmt.Errorf("rule %d response: %w", i, err)
		}
		table.rows = append(table.rows, row{key: key, response: response})
	}
	return table, nil
}

// Match returns a fresh response for the first exact task key, or nil.
func (p *Program) Match(scenario string, attempt int32, activityType string) *anypb.Any {
	for _, r := range p.rows {
		if r.key.GetScenario() == scenario && r.key.GetAttempt() == attempt && r.key.GetActivityType() == activityType {
			return proto.Clone(r.response).(*anypb.Any)
		}
	}
	return nil
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

// Responses returns independent copies for an executor's startup capability check.
func (p *Program) Responses() []*anypb.Any {
	responses := make([]*anypb.Any, 0, len(p.rows))
	for _, r := range p.rows {
		responses = append(responses, proto.Clone(r.response).(*anypb.Any))
	}
	return responses
}
