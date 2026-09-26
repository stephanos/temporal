package replay

import (
	"go.temporal.io/server/tools/umpire/internal/cli"
)

// What became of a reduction's proposal.
const (
	// ProposalNone: the result was incomplete, not reproduced or not attempted; nothing is proposed.
	ProposalNone = "none"
	// ProposalNotCompiled: the retained candidate's proposal did not compile; Error says why.
	ProposalNotCompiled = "not-compiled"
	// ProposalCompiled: the proposal compiled and no promotion root was named, so nothing was written.
	ProposalCompiled = "compiled"
	// ProposalWritten: the proposal was written under the promotion root.
	ProposalWritten = "written"
	// ProposalWriteFailed: the write failed or the destination already existed; nothing is
	// replaced and nothing reruns.
	ProposalWriteFailed = "write-failed"
)

// ProposalReport is what became of the proposal: the digest it is keyed on, the source's SHA-256
// and the path the bridge named, where it was written, its status and the error behind a failure.
type ProposalReport struct {
	Digest  string `json:"digest"`
	SHA256  string `json:"sha256,omitempty"`
	Path    string `json:"path,omitempty"`
	Written string `json:"written,omitempty"`
	Status  string `json:"status"`
	Error   string `json:"error,omitempty"`
}

// WriteProposal writes the bridge's proposal under root through the writer every umpire command
// shares: the path must stay under the root, the file is created exclusively, and a failure is
// the proposal's status, never a rerun. No root writes nothing.
func WriteProposal(root string, proposal *BridgeProposal) ProposalReport {
	if proposal == nil {
		return ProposalReport{Status: ProposalNone}
	}
	report := ProposalReport{Digest: proposal.Digest, Path: proposal.Path}
	if proposal.SHA256 == nil || proposal.Path == "" {
		report.Status, report.Error = ProposalNotCompiled, proposal.Error
		return report
	}
	report.SHA256 = *proposal.SHA256
	if root == "" {
		report.Status = ProposalCompiled
		return report
	}
	written, err := cli.WriteProposals(root, []cli.Proposal{{Candidate: proposal.Digest, Path: proposal.Path, Source: proposal.Source}})
	if err != nil {
		report.Status, report.Error = ProposalWriteFailed, err.Error()
		return report
	}
	report.Status, report.Written = ProposalWritten, written[proposal.Digest]
	return report
}
