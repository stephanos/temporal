package protocol

import (
	"errors"
	"fmt"
)

type OutcomeKind string

const (
	OutcomeRecovered OutcomeKind = "recovered"
	OutcomeDegraded  OutcomeKind = "degraded"
	OutcomeFlagged   OutcomeKind = "flagged"
	OutcomeUnreached OutcomeKind = "unreached"
)

type TerminalDisposition string

const (
	TerminalDispositionSuccess  TerminalDisposition = "success"
	TerminalDispositionFailure  TerminalDisposition = "failure"
	TerminalDispositionUntagged TerminalDisposition = "untagged"
)

type TerminalEvidence struct {
	State          string              `json:"state"`
	Disposition    TerminalDisposition `json:"disposition"`
	Reference      string              `json:"reference"`
	EntityIdentity string              `json:"entityIdentity"`
}

type Outcome struct {
	Kind              OutcomeKind         `json:"kind"`
	Terminal          string              `json:"terminal,omitempty"`
	Disposition       TerminalDisposition `json:"disposition,omitempty"`
	EvidenceReference string              `json:"evidenceReference,omitempty"`
	EntityIdentity    string              `json:"entityIdentity,omitempty"`
	Reason            string              `json:"reason"`
}

func (e TerminalEvidence) Validate() error {
	if e.State == "" {
		return errors.New("terminal state is required")
	}
	switch e.Disposition {
	case TerminalDispositionSuccess, TerminalDispositionFailure, TerminalDispositionUntagged:
	default:
		return fmt.Errorf("terminal disposition %q is invalid", e.Disposition)
	}
	if e.Reference == "" || e.EntityIdentity == "" {
		return errors.New("terminal reference and entity identity are required")
	}
	return nil
}

func ClassifyOutcome(claim ClaimKind, terminal *TerminalEvidence) (Outcome, error) {
	if terminal != nil {
		if err := terminal.Validate(); err != nil {
			return Outcome{}, err
		}
	}
	if claim == ClaimViolating {
		return outcomeFromTerminal(OutcomeFlagged, terminal, "qualified property evidence violates the model"), nil
	}
	if terminal == nil {
		return Outcome{Kind: OutcomeUnreached, Reason: "no qualified lifecycle terminal was observed"}, nil
	}
	if terminal.Disposition == TerminalDispositionFailure {
		return outcomeFromTerminal(OutcomeDegraded, terminal, "a modeled failure terminal was observed"), nil
	}
	return outcomeFromTerminal(OutcomeRecovered, terminal, "a modeled success or untagged terminal was observed"), nil
}

func outcomeFromTerminal(kind OutcomeKind, terminal *TerminalEvidence, reason string) Outcome {
	outcome := Outcome{Kind: kind, Reason: reason}
	if terminal != nil {
		outcome.Terminal = terminal.State
		outcome.Disposition = terminal.Disposition
		outcome.EvidenceReference = terminal.Reference
		outcome.EntityIdentity = terminal.EntityIdentity
	}
	return outcome
}
