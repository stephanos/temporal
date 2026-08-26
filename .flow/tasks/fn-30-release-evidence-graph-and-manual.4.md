---
satisfies: [R3, R4, R5, R7]
---
# fn-30-release-evidence-graph-and-manual.4 Qualify and publish the immutable release evidence graph

## Description
### Umpire4 reconciliation (normative)

Release evidence policy, retention/signing, human roles, revocation, workflows, and authorization are owned by a named downstream release-policy component under the standalone canary/release boundary or an existing external release platform—not by `tools/umpire`. Umpire receipts are immutable generic inputs only. The release owner consumes retained standalone-canary evidence plus external build/deployment attestations, preserves each trust class, and acquires no semantic reinterpretation or deployment authority. Replace legacy `tools/umpire/release` paths and reusable Umpire release-policy types accordingly.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Implement fixed-slot graph construction, bounded policy-process evaluation, qualification receipt creation, and ReleaseEvidenceSet publication for R3-R5 and the qualify portion of R7. This is the single deep boundary from checked inputs to an inspectable immutable decision.

**Size:** M
**Files:** graph, policy-process bridge, qualifier, evidence-set, publisher, and focused tests under `tools/umpire/release/`
**Touches:** [tools/umpire/release/graph*.go, tools/umpire/release/policy_process*.go, tools/umpire/release/qualif*.go, tools/umpire/release/evidence_set*.go, tools/umpire/release/publish*.go]

### Approach
- Build the closed DAG only from one checked signed evidence index and its checked present/gap/stale/revoked slot values; bind the index node/identity, recompute every member/edge, enforce canonical topology/cardinality/depth/byte caps, and reject malformed closure before policy evaluation.
- Invoke the fixed sibling `umpire-release-policy` with one canonical stdin request and stdout response. Verify the manifest/executable/policy/input digests; enforce 2 MiB input, 128 KiB stdout, 64 KiB stderr, and five seconds; map any process/protocol failure to status 1 with no set.
- Preserve complete sorted reasons and rejected-over-held precedence without reinterpreting semantic Result data in Go.
- Compute graph/decision identity and earliest expiry from candidate, all seven slots, trust, policy, evaluation context, keys, attestations, and accepted omissions.
- Publish one ReleaseEvidenceSet transaction containing the signed index, present artifacts, and canonical gaps through the existing validated atomic-set seam; reference retained source members without copying or rewriting them.

### Investigation targets
**Required** (read before coding):
- `tools/common/artifactio/set.go:16-103` — lock, recovery, revalidation, and atomic install seam
- `tools/common/artifactio/set.go:106-166` — staged validation before publication
- `tools/common/artifactio/set_test.go` — transaction failure matrix
- `model/Umpire/ARCHITECTURE.md:210-235` — portable artifact/result boundary

**Optional** (reference as needed):
- `tools/umpire/regression/projection.go` — generated Lean-to-Go projection convention

### Acceptance
- [ ] Canonical graph construction admits only signed-index complete or explicit-gap closures and rejects cycles, dangling/crossed edges, duplicate slots, mixed versions, noncanonical order, and every limit+1 case.
- [ ] Missing/authentic-stale slots publish held status-2 sets, valid contradictions/revocations publish rejected status-2 sets, and invalid trust/bytes/process responses publish no set with status 1.
- [ ] Policy-process fixtures cover missing/digest-mismatched executable, timeout, nonzero exit, extra/oversize output, malformed response, and policy/input mismatch.
- [ ] The release receipt/set preserves all present source claims and explicit gaps, including non-eligibility, and never authorizes or deploys.
- [ ] Publication is immutable, path/symlink/concurrency safe, and leaves prior artifact readers/bytes unchanged.
## Acceptance
- [ ] Checked present/gap slots produce a canonical bounded graph and deterministic pure decision.
- [ ] The fixed Lean bridge has one bounded protocol and fail-closed status mapping.
- [ ] ReleaseEvidenceSet publication is closed, immutable, and inspectable for held/rejected/qualified outcomes.
- [ ] Graph, bridge, compatibility, and atomic-publication tests pass.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
