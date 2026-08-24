package execution

import (
	protocolexecution "go.temporal.io/server/tools/umpire3/protocol/execution"
)

// ClassifyOutcome keeps lifecycle disposition separate from the qualified property claim.
func ClassifyOutcome(claim protocolexecution.ClaimKind, terminal *protocolexecution.TerminalEvidence) (protocolexecution.Outcome, error) {
	if terminal != nil {
		if err := terminal.Validate(); err != nil {
			return protocolexecution.Outcome{}, err
		}
	}
	if claim == protocolexecution.ClaimViolating {
		return outcomeFromTerminal(protocolexecution.OutcomeFlagged, terminal, "qualified property evidence violates the model"), nil
	}
	if terminal == nil {
		return protocolexecution.Outcome{Kind: protocolexecution.OutcomeUnreached, Reason: "no qualified lifecycle terminal was observed"}, nil
	}
	if terminal.Disposition == protocolexecution.TerminalDispositionFailure {
		return outcomeFromTerminal(protocolexecution.OutcomeDegraded, terminal, "a modeled failure terminal was observed"), nil
	}
	return outcomeFromTerminal(protocolexecution.OutcomeRecovered, terminal, "a modeled success or untagged terminal was observed"), nil
}

func outcomeFromTerminal(
	kind protocolexecution.OutcomeKind,
	terminal *protocolexecution.TerminalEvidence,
	reason string,
) protocolexecution.Outcome {
	outcome := protocolexecution.Outcome{Kind: kind, Reason: reason}
	if terminal != nil {
		outcome.Terminal = terminal.State
		outcome.Disposition = terminal.Disposition
		outcome.EvidenceReference = terminal.Reference
		outcome.EntityIdentity = terminal.EntityIdentity
	}
	return outcome
}
