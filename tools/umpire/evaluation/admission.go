// Package evaluation is offline Claim Assessment: it admits one canonical Case and one recorded Run
// of it strictly, assesses the recorded Verdict against a Lean-declared Evaluation Profile, and
// renders the decision as a canonical receipt. It never prepares, runs or replays a Case, and it
// never reads an event's payload: verification is the recorded Verdict's status, and evidence is
// each rule's supporting sequences.
package evaluation

import (
	"bytes"
	"errors"
	"fmt"
	"slices"

	testpilotspb "go.temporal.io/server/api/testpilot/v1"
	"go.temporal.io/server/common/testing/testpilot"
	"go.temporal.io/server/tools/umpire/internal/casefile"
	"go.temporal.io/server/tools/umpire/recordedrun"
)

// Caps are the admission caps a subject is admitted under, recorded in its receipt. They are the
// only enforcement: nothing larger is ever held.
type Caps struct {
	CaseBytes    int `json:"caseBytes"`
	RunBytes     int `json:"runBytes"`
	RunEvents    int `json:"runEvents"`
	ReceiptBytes int `json:"receiptBytes"`
}

// The fixed caps: a 4 MiB Case, a recorded Run as large as the bridge frame cap with at most
// 65,536 events, and a 1 MiB receipt.
const (
	MaxCaseBytes    = 4 << 20
	MaxRunBytes     = 16 << 20
	MaxRunEvents    = 65536
	MaxReceiptBytes = 1 << 20
)

// AdmissionCaps are the fixed caps as a receipt records them.
func AdmissionCaps() Caps {
	return Caps{CaseBytes: MaxCaseBytes, RunBytes: MaxRunBytes, RunEvents: MaxRunEvents, ReceiptBytes: MaxReceiptBytes}
}

// The rejection reasons, each its own class.
const (
	ReasonOversized    = "oversized"
	ReasonNoncanonical = "noncanonical"
	ReasonMalformed    = "malformed"
	ReasonIncompatible = "incompatible"
	ReasonOpen         = "open"
	ReasonCrossed      = "crossed"
	ReasonInconsistent = "inconsistent"
	ReasonStale        = "stale"
)

// Rejection is why a subject was not admitted: the reason names the class, the detail what was
// found.
type Rejection struct {
	Reason string
	Detail string
}

func (r *Rejection) Error() string { return r.Reason + ": " + r.Detail }

func reject(reason, format string, arguments ...any) error {
	return &Rejection{Reason: reason, Detail: fmt.Sprintf(format, arguments...)}
}

// IsRejection reports the rejection an error carries, when it is one.
func IsRejection(err error) (*Rejection, bool) {
	var rejection *Rejection
	if errors.As(err, &rejection) {
		return rejection, true
	}
	return nil, false
}

// Subject is one admitted closed Run of one Case: what an assessment reads and a receipt binds.
type Subject struct {
	// CaseIdentity is the hex SHA-256 of the canonical Case bytes.
	CaseIdentity string
	// RunIdentity is the hex SHA-256 of the canonical recorded-Run bytes.
	RunIdentity string
	CaseID      string
	ProgramID   string
	ContractID  string
	Driver      testpilot.DriverIdentity
	RunID       string
	Disposition testpilotspb.RunDisposition
	Cleanup     testpilotspb.CleanupStatus
	Verdict     *testpilotspb.Verdict
	// KnownGaps are the Case's, as its provenance declares them.
	KnownGaps []*testpilotspb.KnownGap
	// Caps are the caps the subject was admitted under.
	Caps Caps
}

// Admit reads one Case and one recorded Run and admits them as a subject, or rejects them with a
// reason, without preparing, running or replaying anything. catalog is the catalog fingerprint
// the recorded identity must carry: the command passes the tree's static catalog's. The checks
// run in a fixed order, the Case's first, then the recorded Run's, then the pair's.
func Admit(caseInput, recordedInput []byte, catalog string) (*Subject, error) {
	caps := AdmissionCaps()
	if len(caseInput) > caps.CaseBytes {
		return nil, reject(ReasonOversized, "the Case is %d bytes, over the %d-byte cap", len(caseInput), caps.CaseBytes)
	}
	canonical, err := casefile.Canonical(caseInput)
	if err != nil {
		return nil, reject(ReasonNoncanonical, "%s", err)
	}
	source, err := testpilot.DecodeCaseProtoJSON(canonical)
	if err != nil {
		return nil, reject(ReasonMalformed, "the Case does not decode: %s", err)
	}
	if version := source.GetVersion(); version.GetMajor() != 1 || version.GetMinor() != 0 {
		return nil, reject(ReasonIncompatible, "the Case is format version %d.%d, not 1.0", version.GetMajor(), version.GetMinor())
	}

	if len(recordedInput) > caps.RunBytes {
		return nil, reject(ReasonOversized, "the recorded Run is %d bytes, over the %d-byte cap", len(recordedInput), caps.RunBytes)
	}
	decoded, err := recordedrun.Decode(recordedInput)
	if err != nil && !errors.Is(err, recordedrun.ErrNoCase) {
		return nil, reject(ReasonMalformed, "%s", err)
	}
	run := decoded.Run
	if events := len(run.GetEvents()); events > caps.RunEvents {
		return nil, reject(ReasonOversized, "the Run has %d events, over the %d-event cap", events, caps.RunEvents)
	}
	if err != nil {
		return nil, reject(ReasonIncompatible, "%s", err)
	}
	// protojson reads an enum's number as readily as its name, and a number the proto does not
	// declare re-encodes to itself; such a value is no status at all.
	if detail := undeclaredStatus(run); detail != "" {
		return nil, reject(ReasonMalformed, "%s", detail)
	}
	reencoded, err := recordedrun.Encode(decoded.Case, decoded.Driver, run)
	if err != nil {
		return nil, reject(ReasonMalformed, "%s", err)
	}
	if !bytes.Equal(reencoded, recordedInput) {
		return nil, reject(ReasonNoncanonical, "the recorded Run is not in the form its writer produces")
	}

	verdict := run.GetVerdict()
	if detail := openness(run); detail != "" {
		return nil, reject(ReasonOpen, "%s", detail)
	}
	caseIdentity := recordedrun.Digest(canonical)
	if decoded.Case != caseIdentity {
		return nil, reject(ReasonCrossed, "the Run was recorded from Case %s, the Case is %s", decoded.Case, caseIdentity)
	}
	if crossed := recordedrun.Crossed(source, run); crossed != "" {
		return nil, reject(ReasonCrossed, "%s", crossed)
	}
	if detail := ruleSetCrossed(source.GetContract(), verdict); detail != "" {
		return nil, reject(ReasonCrossed, "%s", detail)
	}
	if problem := recordedrun.CheckSupport(run, verdict); problem != nil {
		return nil, reject(ReasonInconsistent, "%s", problem)
	}
	if agrees, detail := recordedrun.Agreement(run, verdict); !agrees {
		return nil, reject(ReasonInconsistent, "%s", detail)
	}
	for _, gap := range source.GetProvenance().GetKnownGaps() {
		if _, declared := testpilotspb.KnownGapKind_name[int32(gap.GetKind())]; !declared || gap.GetKind() == testpilotspb.KNOWN_GAP_KIND_UNSPECIFIED {
			return nil, reject(ReasonMalformed, "Known Gap %q has kind %d, which is not a declared kind", gap.GetCode(), gap.GetKind())
		}
	}
	if decoded.Driver.Catalog != catalog {
		return nil, reject(ReasonStale, "the Run was recorded under catalog %s, the tree's is %s", decoded.Driver.Catalog, catalog)
	}
	return &Subject{
		CaseIdentity: caseIdentity,
		RunIdentity:  recordedrun.Digest(recordedInput),
		CaseID:       source.GetCaseId(),
		ProgramID:    source.GetProgram().GetProgramId(),
		ContractID:   source.GetContract().GetContractId(),
		Driver:       decoded.Driver,
		RunID:        run.GetRunId(),
		Disposition:  run.GetDisposition(),
		Cleanup:      run.GetCleanup().GetStatus(),
		Verdict:      verdict,
		KnownGaps:    source.GetProvenance().GetKnownGaps(),
		Caps:         caps,
	}, nil
}

// openness names what a closed Run lacks, or returns "": a Run ID, an event, a terminal
// disposition, a cleanup outcome and a Verdict.
func openness(run *testpilotspb.Run) string {
	switch {
	case run.GetRunId() == "":
		return "the Run has no Run ID"
	case len(run.GetEvents()) == 0:
		return "the Run has no events"
	case run.GetDisposition() == testpilotspb.RUN_DISPOSITION_UNSPECIFIED:
		return "the Run has no terminal disposition"
	case run.GetCleanup().GetStatus() == testpilotspb.CLEANUP_STATUS_UNSPECIFIED:
		return "the Run has no cleanup outcome"
	case run.GetVerdict() == nil:
		return "the Run has no Verdict"
	default:
		return ""
	}
}

// ruleSetCrossed names why the Verdict's rules are not exactly the Contract's rules, its plain rules
// and its correlated rules together, each once, or returns "". A Run carries no Contract ID, so
// this is how a Run of another Contract shows.
func ruleSetCrossed(contract *testpilotspb.Contract, verdict *testpilotspb.Verdict) string {
	var expected []string
	for _, rule := range contract.GetRules() {
		expected = append(expected, rule.GetRuleId())
	}
	for _, rule := range contract.GetCorrelated().GetRules() {
		expected = append(expected, rule.GetRuleId())
	}
	var named []string
	for _, rule := range verdict.GetRules() {
		named = append(named, rule.GetRuleId())
	}
	if duplicate := firstRepeated(named); duplicate != "" {
		return fmt.Sprintf("the Verdict names rule %q twice", duplicate)
	}
	slices.Sort(expected)
	slices.Sort(named)
	if !slices.Equal(expected, named) {
		return fmt.Sprintf("the Verdict names rules %v, the Contract's are %v", named, expected)
	}
	return ""
}

// undeclaredStatus names a disposition, cleanup, Verdict or rule status the proto does not declare,
// or returns "".
func undeclaredStatus(run *testpilotspb.Run) string {
	if _, ok := testpilotspb.RunDisposition_name[int32(run.GetDisposition())]; !ok {
		return fmt.Sprintf("the Run's disposition %d is not a declared disposition", run.GetDisposition())
	}
	if _, ok := testpilotspb.CleanupStatus_name[int32(run.GetCleanup().GetStatus())]; !ok {
		return fmt.Sprintf("the Run's cleanup status %d is not a declared status", run.GetCleanup().GetStatus())
	}
	if _, ok := testpilotspb.VerdictStatus_name[int32(run.GetVerdict().GetStatus())]; !ok {
		return fmt.Sprintf("the Verdict's status %d is not a declared status", run.GetVerdict().GetStatus())
	}
	for _, rule := range run.GetVerdict().GetRules() {
		if _, ok := testpilotspb.RuleVerdictStatus_name[int32(rule.GetStatus())]; !ok {
			return fmt.Sprintf("rule %s has status %d, which is not a declared status", rule.GetRuleId(), rule.GetStatus())
		}
	}
	return ""
}
