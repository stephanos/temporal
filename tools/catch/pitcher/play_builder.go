// The MIT License
//
// Copyright (c) 2025 Temporal Technologies Inc.  All rights reserved.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package pitcher

import (
	omeskitchensink "github.com/temporalio/omes/loadgen/kitchensink"
	sdkclient "go.temporal.io/sdk/client"
)

// Pitch represents a single action in a test scenario.
// A Pitch can be a fault injection, a server action, a workflow start, or a client action.
type Pitch interface {
	Type() PitchType
}

// PitchType identifies the type of pitch
type PitchType string

const (
	// PitchTypeFault injects a fault (delay, error, etc.)
	PitchTypeFault PitchType = "fault"

	// PitchTypeServerAction manually unblocks/manipulates a server response
	PitchTypeServerAction PitchType = "server_action"

	// PitchTypeStartWorkflow starts a workflow execution
	PitchTypeStartWorkflow PitchType = "start_workflow"

	// PitchTypeClientAction performs a client action (signal, update, etc.)
	PitchTypeClientAction PitchType = "client_action"
)

// FaultPitch injects a fault with specific matching criteria
type FaultPitch struct {
	Target string         // The target type (e.g., "*matchingservice.AddWorkflowTaskRequest")
	Fault  Play           // The fault to inject (delay, fail, etc.)
	Match  *MatchCriteria // When to inject this fault
}

func (p *FaultPitch) Type() PitchType { return PitchTypeFault }

// ServerActionPitch manually unblocks a gRPC call from worker to server
type ServerActionPitch struct {
	// Method is the gRPC method name (e.g., "RespondWorkflowTaskCompleted")
	Method string

	// ErrorOverride optionally forces the server to return this error
	ErrorOverride error

	// ResponseManipulator optionally modifies the server's response
	ResponseManipulator func(response any) any
}

func (p *ServerActionPitch) Type() PitchType { return PitchTypeServerAction }

// StartWorkflowPitch starts a workflow execution
type StartWorkflowPitch struct {
	Client        sdkclient.Client
	Options       sdkclient.StartWorkflowOptions
	Workflow      any
	WorkflowInput *omeskitchensink.WorkflowInput
}

func (p *StartWorkflowPitch) Type() PitchType { return PitchTypeStartWorkflow }

// ClientActionPitch performs a client action during workflow execution
type ClientActionPitch struct {
	// Action is the client action to perform (signal, update, cancel, etc.)
	Action any
}

func (p *ClientActionPitch) Type() PitchType { return PitchTypeClientAction }

// ScenarioPlay represents a complete test scenario as a sequence of pitches
type ScenarioPlay struct {
	Pitches []Pitch
}

// PlayBuilder helps construct test scenarios
type PlayBuilder struct {
	pitches []Pitch
}

// NewPlayBuilder creates a new play builder
func NewPlayBuilder() *PlayBuilder {
	return &PlayBuilder{
		pitches: []Pitch{},
	}
}

// WithFault adds a fault pitch to the play
func (b *PlayBuilder) WithFault(target string, fault Play, match *MatchCriteria) *PlayBuilder {
	b.pitches = append(b.pitches, &FaultPitch{
		Target: target,
		Fault:  fault,
		Match:  match,
	})
	return b
}

// WithServerAction adds a server action pitch to manually unblock a gRPC call
func (b *PlayBuilder) WithServerAction(method string) *PlayBuilder {
	b.pitches = append(b.pitches, &ServerActionPitch{
		Method: method,
	})
	return b
}

// WithServerActionError adds a server action that returns an error
func (b *PlayBuilder) WithServerActionError(method string, err error) *PlayBuilder {
	b.pitches = append(b.pitches, &ServerActionPitch{
		Method:        method,
		ErrorOverride: err,
	})
	return b
}

// WithServerActionManipulator adds a server action with response manipulation
func (b *PlayBuilder) WithServerActionManipulator(method string, manipulator func(any) any) *PlayBuilder {
	b.pitches = append(b.pitches, &ServerActionPitch{
		Method:              method,
		ResponseManipulator: manipulator,
	})
	return b
}

// WithStartWorkflow adds a workflow start pitch
func (b *PlayBuilder) WithStartWorkflow(client sdkclient.Client, options sdkclient.StartWorkflowOptions, workflow any, input *omeskitchensink.WorkflowInput) *PlayBuilder {
	b.pitches = append(b.pitches, &StartWorkflowPitch{
		Client:        client,
		Options:       options,
		Workflow:      workflow,
		WorkflowInput: input,
	})
	return b
}

// WithClientAction adds a client action pitch
func (b *PlayBuilder) WithClientAction(action any) *PlayBuilder {
	b.pitches = append(b.pitches, &ClientActionPitch{
		Action: action,
	})
	return b
}

// Build constructs the final play
func (b *PlayBuilder) Build() *ScenarioPlay {
	return &ScenarioPlay{
		Pitches: b.pitches,
	}
}
