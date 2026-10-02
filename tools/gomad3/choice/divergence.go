package choice

import (
	"crypto/sha256"
	"errors"
	"fmt"
)

type DivergenceEvidence struct {
	Ordinal     uint64            `json:"ordinal"`
	Reason      DivergenceReason  `json:"reason"`
	Expected    *DecisionEvidence `json:"expected,omitempty"`
	Observed    *DecisionEvidence `json:"observed,omitempty"`
	TapeRecords uint64            `json:"tape_records"`
}

type DecisionEvidence struct {
	Ordinal              uint64            `json:"ordinal"`
	Kind                 Kind              `json:"kind"`
	SiteOffset           uint64            `json:"site_offset"`
	SiteMissing          bool              `json:"site_missing"`
	RankOverride         bool              `json:"rank_override"`
	Alternatives         uint32            `json:"alternatives"`
	Selected             uint32            `json:"selected"`
	Data                 uint32            `json:"data"`
	SelectedIdentity     [sha256.Size]byte `json:"selected_identity"`
	AlternativeSetDigest [sha256.Size]byte `json:"alternative_set_digest"`
}

func ProjectDivergenceEvidence(divergence Divergence) DivergenceEvidence {
	evidence := DivergenceEvidence{Ordinal: divergence.Ordinal, Reason: divergence.Reason, TapeRecords: divergence.TapeRecords}
	if divergence.Expected != nil {
		expected := DecisionEvidence(*divergence.Expected)
		evidence.Expected = &expected
	}
	if divergence.Observed != nil {
		observed := DecisionEvidence(*divergence.Observed)
		evidence.Observed = &observed
	}
	return evidence
}

func (evidence DivergenceEvidence) ReasonName() string {
	return DivergenceReasonName(evidence.Reason)
}

func (evidence DivergenceEvidence) Divergence() Divergence {
	divergence := Divergence{Ordinal: evidence.Ordinal, Reason: evidence.Reason, TapeRecords: evidence.TapeRecords}
	if evidence.Expected != nil {
		expected := Decision(*evidence.Expected)
		divergence.Expected = &expected
	}
	if evidence.Observed != nil {
		observed := Decision(*evidence.Observed)
		divergence.Observed = &observed
	}
	return divergence
}

func ValidatePrefixReplayDivergence(tape ReplayPlan, divergence Divergence) error {
	terminal := terminal{State: TerminalDiverged, DivergentOrdinal: divergence.Ordinal, DivergenceReason: divergence.Reason, TapeRecords: divergence.TapeRecords}
	if divergence.Expected != nil {
		if _, err := encodeRecord(divergence.Expected.Record()); err != nil {
			return fmt.Errorf("choice divergence expected decision: %w", err)
		}
		terminal.ExpectedPresent, terminal.Expected = true, divergence.Expected.Record()
	}
	if divergence.Observed != nil {
		if divergence.Observed.Ordinal != divergence.Ordinal || divergence.Observed.RankOverride {
			return errors.New("choice divergence observed decision is invalid")
		}
		if _, err := encodeRecord(divergence.Observed.Record()); err != nil {
			return fmt.Errorf("choice divergence observed decision: %w", err)
		}
		terminal.ObservedPresent, terminal.Observed = true, divergence.Observed.Record()
	}
	_, err := validateDivergenceTerminal(tape, ModePrefix, terminal)
	return err
}
