package agentworkflow

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrCapacity    = errors.New("agentworkflow capacity exhausted")
	ErrAgent       = errors.New("agentworkflow agent failed")
	ErrCorrupt     = errors.New("agentworkflow state is corrupt")
	ErrLocked      = errors.New("agentworkflow run is active")
	ErrSourceDrift = errors.New("agentworkflow source changed")
	ErrUnsupported = errors.New("agentworkflow operation is unsupported")
)

type Backend interface {
	Describe(context.Context) (BackendInfo, error)
	Execute(context.Context, Invocation, EventSink) (InvocationResult, error)
}

type BackendInfo struct {
	Name                string       `json:"name"`
	Version             string       `json:"version"`
	ConfigurationDigest string       `json:"configuration_digest"`
	Capabilities        []Capability `json:"capabilities"`
}

type Capability string

const (
	CapabilityReadOnly         Capability = "read-only"
	CapabilityWorkspaceWrite   Capability = "workspace-write"
	CapabilityStructuredOutput Capability = "structured-output"
	CapabilityMachineEvents    Capability = "machine-events"
	CapabilityResume           Capability = "resume"
	CapabilityIsolatedConfig   Capability = "isolated-config"
	CapabilityCancellation     Capability = "cancellation"
)

type Permission string

const (
	PermissionReadOnly       Permission = "read-only"
	PermissionWorkspaceWrite Permission = "workspace-write"
)

type Invocation struct {
	ID             string
	Phase          string
	Workspace      string
	Prompt         string
	OutputSchema   json.RawMessage
	Permission     Permission
	Session        string
	RetainSession  bool
	Timeout        time.Duration
	MaxOutputBytes int64
	MaxEvents      int
}

type InvocationResult struct {
	Session string
	Output  json.RawMessage
	Usage   Usage
}

type Usage struct {
	InputTokens  int64 `json:"input_tokens,omitempty"`
	OutputTokens int64 `json:"output_tokens,omitempty"`
}

type EventSink interface {
	Emit(Event) error
}

type Event struct {
	Kind    EventKind       `json:"kind"`
	Session string          `json:"session,omitempty"`
	Message string          `json:"message,omitempty"`
	Usage   Usage           `json:"usage,omitempty"`
	Raw     json.RawMessage `json:"raw,omitempty"`
}

type EventKind string

const (
	EventInvocationStarted   EventKind = "invocation-started"
	EventSessionIdentified   EventKind = "session-identified"
	EventProgress            EventKind = "progress"
	EventCommand             EventKind = "command"
	EventFileChange          EventKind = "file-change"
	EventTool                EventKind = "tool"
	EventAgentMessage        EventKind = "agent-message"
	EventUsage               EventKind = "usage"
	EventInvocationCompleted EventKind = "invocation-completed"
	EventInvocationFailed    EventKind = "invocation-failed"
	EventDiagnostic          EventKind = "diagnostic"
)
