package evaluation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

// ReceiptFormatVersion is the receipt format this package writes and reads.
const ReceiptFormatVersion = 1

// Receipt is one Evaluation Receipt: one Profile's decision on one closed Run of one Case, with
// everything the decision was read from, in a fixed key order. It carries identities, IDs, statuses
// and sequence numbers, never a payload, event body, credential, path or endpoint. It is not
// self-authenticating: its identity says which bytes it is, not who wrote them.
type Receipt struct {
	Version int            `json:"version"`
	Profile ReceiptProfile `json:"profile"`
	Case    ReceiptCase    `json:"case"`
	Run     ReceiptRun     `json:"run"`
	Verdict ReceiptVerdict `json:"verdict"`
	// Decision is accepted, rejected or incomplete.
	Decision string `json:"decision"`
	// Reasons are the Profile's rows that hold, in the table's order.
	Reasons []Reason `json:"reasons"`
	// UnsupportedRules are the rules the Verdict names at a terminal state with no supporting event.
	UnsupportedRules []string          `json:"unsupportedRules"`
	KnownGaps        []ReceiptKnownGap `json:"knownGaps"`
	// Caps are the admission caps the subject was admitted under.
	Caps Caps `json:"caps"`
}

// ReceiptProfile names the Profile the decision was made under, its claim and its asserted trust.
type ReceiptProfile struct {
	Name     string `json:"name"`
	Identity string `json:"identity"`
	Claim    string `json:"claim"`
	Trust    string `json:"trust"`
}

// ReceiptCase is the Case the Run was of.
type ReceiptCase struct {
	Identity   string `json:"identity"`
	CaseID     string `json:"caseId"`
	ProgramID  string `json:"programId"`
	ContractID string `json:"contractId"`
}

// ReceiptRun is the closed Run as recorded.
type ReceiptRun struct {
	Identity    string               `json:"identity"`
	RunID       string               `json:"runId"`
	Driver      recordedrun.Identity `json:"driver"`
	Disposition string               `json:"disposition"`
	Cleanup     string               `json:"cleanup"`
}

// ReceiptVerdict is the recorded Verdict: its status and each rule's conclusion with the supporting
// sequences that are its evidence links.
type ReceiptVerdict struct {
	Status string        `json:"status"`
	Rules  []ReceiptRule `json:"rules"`
}

// ReceiptRule is one rule's recorded conclusion.
type ReceiptRule struct {
	RuleID                   string  `json:"ruleId"`
	Status                   string  `json:"status"`
	TerminalState            string  `json:"terminalState"`
	SupportingEventSequences []int64 `json:"supportingEventSequences"`
}

// ReceiptKnownGap is one of the Case's Known Gaps, by kind and code.
type ReceiptKnownGap struct {
	Kind string `json:"kind"`
	Code string `json:"code"`
}

// ReceiptOversizedError says a rendered receipt is over its cap. It is a tooling failure, never a
// decision: the receipt is not published.
type ReceiptOversizedError struct {
	Size int
	Cap  int
}

func (e *ReceiptOversizedError) Error() string {
	return fmt.Sprintf("receipt-oversized: the receipt is %d bytes, over the %d-byte cap", e.Size, e.Cap)
}

func checkReceiptSize(size int) error {
	if size > MaxReceiptBytes {
		return &ReceiptOversizedError{Size: size, Cap: MaxReceiptBytes}
	}
	return nil
}

// Render is the canonical receipt of a Decision on a subject under the Profile it was made under.
func Render(subject *Subject, profile Profile, decision Decision) ([]byte, error) {
	if decision.ProfileIdentity != profile.Identity || decision.ProfileName != profile.Name {
		return nil, errors.New("the Decision was not made under this Profile")
	}
	if decision.Verdict != subject.Verdict.GetStatus() || decision.Disposition != subject.Disposition || decision.Cleanup != subject.Cleanup {
		return nil, errors.New("the Decision was not made on this subject")
	}
	receipt := Receipt{
		Version: ReceiptFormatVersion,
		Profile: ReceiptProfile{Name: profile.Name, Identity: profile.Identity, Claim: profile.Claim, Trust: profile.Trust},
		Case: ReceiptCase{
			Identity: subject.CaseIdentity, CaseID: subject.CaseID, ProgramID: subject.ProgramID, ContractID: subject.ContractID,
		},
		Run: ReceiptRun{
			Identity: subject.RunIdentity,
			RunID:    subject.RunID,
			Driver: recordedrun.Identity{
				Profile: subject.Driver.Profile, Catalog: subject.Driver.Catalog, Bindings: subject.Driver.Bindings,
			},
			Disposition: testpilotspb.RunDisposition_name[int32(subject.Disposition)],
			Cleanup:     testpilotspb.CleanupStatus_name[int32(subject.Cleanup)],
		},
		Verdict: ReceiptVerdict{
			Status: testpilotspb.VerdictStatus_name[int32(subject.Verdict.GetStatus())],
			Rules:  []ReceiptRule{},
		},
		Decision:         decision.Outcome,
		Reasons:          append([]Reason{}, decision.Reasons...),
		UnsupportedRules: append([]string{}, decision.UnsupportedRules...),
		KnownGaps:        []ReceiptKnownGap{},
		Caps:             subject.Caps,
	}
	for _, rule := range subject.Verdict.GetRules() {
		receipt.Verdict.Rules = append(receipt.Verdict.Rules, ReceiptRule{
			RuleID:                   rule.GetRuleId(),
			Status:                   testpilotspb.RuleVerdictStatus_name[int32(rule.GetStatus())],
			TerminalState:            rule.GetTerminalStateId(),
			SupportingEventSequences: append([]int64{}, rule.GetSupportingEventSequences()...),
		})
	}
	for _, gap := range decision.KnownGaps {
		receipt.KnownGaps = append(receipt.KnownGaps, ReceiptKnownGap(gap))
	}
	rendered, err := renderReceipt(&receipt)
	if err != nil {
		return nil, err
	}
	if err := checkReceiptSize(len(rendered)); err != nil {
		return nil, err
	}
	return rendered, nil
}

func renderReceipt(receipt *Receipt) ([]byte, error) {
	var rendered bytes.Buffer
	encoder := json.NewEncoder(&rendered)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(receipt); err != nil {
		return nil, err
	}
	return rendered.Bytes(), nil
}

// ReceiptIdentity is the hex SHA-256 of a receipt's bytes, the name it is published under.
func ReceiptIdentity(rendered []byte) string {
	return recordedrun.Digest(rendered)
}

// DecodeReceipt reads a receipt back strictly: at most the cap, this format version, and exactly
// the canonical rendering of what it decodes to, so an unknown, repeated or case-folded key, a
// trailing document or other spacing is refused.
func DecodeReceipt(encoded []byte) (*Receipt, error) {
	if err := checkReceiptSize(len(encoded)); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var receipt Receipt
	if err := decoder.Decode(&receipt); err != nil {
		return nil, fmt.Errorf("decode receipt: %w", err)
	}
	if receipt.Version != ReceiptFormatVersion {
		return nil, fmt.Errorf("receipt format version %d, not %d", receipt.Version, ReceiptFormatVersion)
	}
	canonical, err := renderReceipt(&receipt)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(canonical, encoded) || hasNullList(&receipt) {
		return nil, errors.New("the receipt is not in its canonical form")
	}
	return &receipt, nil
}

// hasNullList reports a list Render always writes as an array but the bytes gave as null.
func hasNullList(receipt *Receipt) bool {
	if receipt.Reasons == nil || receipt.UnsupportedRules == nil || receipt.KnownGaps == nil || receipt.Verdict.Rules == nil {
		return true
	}
	for _, rule := range receipt.Verdict.Rules {
		if rule.SupportingEventSequences == nil {
			return true
		}
	}
	return false
}
