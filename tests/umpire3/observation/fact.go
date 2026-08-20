package observation

import (
	"errors"
	"fmt"
	"slices"
)

const (
	FactTypeHistoryEvent     = "history-event"
	FactTypeMechanismReceipt = "mechanism-receipt"
	FactTypeEvidenceWindow   = "evidence-window"

	NexusCancellationAccepted  = "nexus-cancellation-accepted"
	NexusCancellationCommitted = "nexus-cancellation-committed"
	NexusSuccessRecorded       = "nexus-success-recorded"
	NexusOwnershipAcquired     = "nexus-ownership-acquired"
	NexusCancellationWindow    = "nexus-cancellation"
)

type Source struct {
	Identity         string   `json:"identity"`
	ClockDomain      string   `json:"clockDomain"`
	Sequence         int64    `json:"sequence"`
	Reference        string   `json:"reference"`
	CausalReferences []string `json:"causalReferences,omitempty"`
	EntityIdentity   string   `json:"entityIdentity"`
	Lineage          []string `json:"lineage"`
	PayloadDigest    string   `json:"payloadDigest,omitempty"`
}

type HistoryEvent struct {
	EventType             string `json:"eventType"`
	EventID               int64  `json:"eventID"`
	WorkflowID            string `json:"workflowID,omitempty"`
	RunID                 string `json:"runID,omitempty"`
	OperationID           string `json:"operationID,omitempty"`
	OwnerEpoch            *int64 `json:"ownerEpoch,omitempty"`
	CurrentOwnerEpoch     *int64 `json:"currentOwnerEpoch,omitempty"`
	CancellationCommitted *bool  `json:"cancellationCommitted,omitempty"`
}

type MechanismReceipt struct {
	Action     string `json:"action"`
	Resource   string `json:"resource"`
	Attempt    int64  `json:"attempt"`
	OwnerEpoch int64  `json:"ownerEpoch"`
	Outcome    string `json:"outcome"`
}

type EvidenceWindow struct {
	Purpose         string `json:"purpose"`
	Closed          bool   `json:"closed"`
	ThroughSequence int64  `json:"throughSequence"`
}

type Fact struct {
	Identifier string            `json:"identifier"`
	Source     Source            `json:"source"`
	History    *HistoryEvent     `json:"history,omitempty"`
	Mechanism  *MechanismReceipt `json:"mechanism,omitempty"`
	Window     *EvidenceWindow   `json:"window,omitempty"`
}

func (f Fact) Validate() error {
	if f.Identifier == "" {
		return errors.New("fact identifier is required")
	}
	if err := f.Source.validate(); err != nil {
		return fmt.Errorf("fact %q: %w", f.Identifier, err)
	}
	variants := 0
	if f.History != nil {
		variants++
	}
	if f.Mechanism != nil {
		variants++
	}
	if f.Window != nil {
		variants++
	}
	if variants != 1 {
		return fmt.Errorf("fact %q must contain exactly one typed value", f.Identifier)
	}
	if f.History != nil {
		return f.History.validate(f.Identifier, f.Source)
	}
	if f.Mechanism != nil {
		return f.Mechanism.validate(f.Identifier)
	}
	return f.Window.validate(f.Identifier, f.Source.Sequence)
}

func (s Source) validate() error {
	if s.Identity == "" || s.ClockDomain == "" || s.Sequence <= 0 || s.Reference == "" ||
		s.EntityIdentity == "" || len(s.Lineage) == 0 {
		return errors.New("complete source, order, reference, identity, and lineage are required")
	}
	if slices.Contains(s.Lineage, "") {
		return errors.New("lineage contains an empty identity")
	}
	return nil
}

func (e HistoryEvent) validate(identifier string, source Source) error {
	if e.EventType == "" || e.EventID <= 0 {
		return fmt.Errorf("history fact %q requires an event type and positive event ID", identifier)
	}
	if e.EventID != source.Sequence {
		return fmt.Errorf("history fact %q event ID must match its source sequence", identifier)
	}
	switch e.EventType {
	case NexusCancellationAccepted, NexusCancellationCommitted:
		if e.OperationID == "" {
			return fmt.Errorf("history fact %q requires an operation ID", identifier)
		}
	case NexusSuccessRecorded:
		if e.OperationID == "" || e.OwnerEpoch == nil || e.CurrentOwnerEpoch == nil {
			return fmt.Errorf("history fact %q requires operation, owner epoch, and current owner epoch", identifier)
		}
		if e.CancellationCommitted == nil {
			return fmt.Errorf("history fact %q requires cancellation commitment", identifier)
		}
	default:
		return fmt.Errorf("history fact %q has unsupported event type %q", identifier, e.EventType)
	}
	return nil
}

func (r MechanismReceipt) validate(identifier string) error {
	if r.Action == "" || r.Resource == "" || r.Attempt <= 0 || r.OwnerEpoch < 0 || r.Outcome == "" {
		return fmt.Errorf("mechanism fact %q is incomplete", identifier)
	}
	return nil
}

func (w EvidenceWindow) validate(identifier string, sourceSequence int64) error {
	if w.Purpose == "" || w.ThroughSequence <= 0 || w.ThroughSequence > sourceSequence {
		return fmt.Errorf("evidence window fact %q has an invalid bounded interval", identifier)
	}
	return nil
}

func (f Fact) factTypeAndKind() (string, string) {
	switch {
	case f.History != nil:
		return FactTypeHistoryEvent, f.History.EventType
	case f.Mechanism != nil:
		return FactTypeMechanismReceipt, f.Mechanism.Action
	case f.Window != nil:
		return FactTypeEvidenceWindow, f.Window.Purpose
	default:
		return "", ""
	}
}
