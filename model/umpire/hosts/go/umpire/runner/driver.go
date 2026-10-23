package runner

import (
	"context"

	"google.golang.org/protobuf/types/known/anypb"
	modelv1 "umpire.local/model/umpire/model/v1"
)

// Driver acts on one system. It sends what the Model proposed and reports what happened as
// correlated events; it never decides what an event means.
type Driver interface {
	// Start sends one proposed request. The request number correlates every event the
	// exchange reports. An error is a Driver fault.
	Start(ctx context.Context, request uint64, payload *anypb.Any) (Exchange, error)
}

// NewDriver makes a fresh Driver for one Run.
type NewDriver func(ctx context.Context) (Driver, error)

// Exchange is one request in flight. Its events arrive in order while the request is active;
// the Driver waits for an event's decision before its next dependent read or worker response.
type Exchange interface {
	// Events reports the exchange's events, ending with Done or Fault. A stream that closes
	// without either is a Driver fault.
	Events() <-chan Event
	// Respond delivers the decision for an Observation, Callback or Annotate event.
	Respond(ctx context.Context, event uint64, decision Decision) error
	// Close cancels and joins the exchange's work.
	Close() error
}

// EventKind says what a Driver event carries.
type EventKind int

const (
	EventUnspecified EventKind = iota
	// EventObservation carries the actual request and response evidence of one read.
	EventObservation
	// EventCallback carries a worker's call for the Model to answer.
	EventCallback
	// EventAnnotate carries a message the Driver asks the Model to annotate.
	EventAnnotate
	// EventDone says the exchange completed; nothing follows it.
	EventDone
	// EventFault says the transport failed; nothing follows it.
	EventFault
)

// Event is one Driver event. ID increases strictly within an exchange; Request repeats the
// number the exchange was started with.
type Event struct {
	ID      uint64
	Request uint64
	Kind    EventKind
	// Payload is the Observation, Callback or Annotate input.
	Payload *anypb.Any
	// Call correlates a Callback with the worker call it came from.
	Call uint64
	// Fault describes a transport failure.
	Fault string
}

// Decision is the Runner's answer to one event. Exactly one field is set.
type Decision struct {
	// Judgment answers an Observation; ContinuePolling asks for another read.
	Judgment *modelv1.JudgeResponse
	// Reply answers a Callback.
	Reply *anypb.Any
	// Annotation answers an Annotate event.
	Annotation *anypb.Any
	// Refusal says the Model refused the event; the exchange ends.
	Refusal *modelv1.Refusal
}
