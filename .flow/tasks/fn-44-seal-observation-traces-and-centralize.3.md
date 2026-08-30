---
satisfies: [R3, R4, R6]
---
# fn-44-seal-observation-traces-and-centralize.3 Seal accepted Observation traces

## Description
Establish the hard accepted-trace admission boundary required by R3 and R4. Keep unchecked mutation data at the low-level test seam while making the ordinary semantic `EvidenceBackedTrace` opaque.

**Size:** M
**Files:** `model/Umpire/Observation/Evaluation.lean`, `model/Umpire/Observation/Tests/Fixtures.lean`, `model/Umpire/Observation/Tests/EvidenceLink.lean`, `model/Umpire/Observation/Tests/Mutations.lean`, `model/Umpire/Observation/Tests/Disposition.lean`, `model/Umpire/Observation/Tests/Verdict.lean`, `model/Umpire/Observation/ImportTests.lean`
**Touches:** [model/Umpire/Observation/Evaluation.lean, model/Umpire/Observation/Tests/Fixtures.lean, model/Umpire/Observation/Tests/EvidenceLink.lean, model/Umpire/Observation/Tests/Mutations.lean, model/Umpire/Observation/Tests/Disposition.lean, model/Umpire/Observation/Tests/Verdict.lean, model/Umpire/Observation/ImportTests.lean]

### Approach
- Separate the wide unchecked carrier used during evaluation/tests from the opaque accepted semantic type.
- Make the existing complete validator return the accepted value and have `evaluateEvidence` admit immediately before constructing `.accepted`.
- Provide only the read-only projections and compatibility instances demonstrated by current consumers; do not expose construction, record update, proof-only admission, or a raw ordinary overload.
- Rewrite every EvidenceLink, mutation, disposition, and verdict fixture that forges accepted records to mutate unchecked data and assert the same admission diagnostics and precedence.

### Investigation targets
**Required** (read before coding):
- `model/Umpire/Observation/Evaluation.lean:224-250,1611-1670` — forgeable record, accepted result, and complete admission boundary.
- `model/Umpire/Observation/Tests/Fixtures.lean:285-319` — accepted fixture construction.
- `model/Umpire/Observation/Tests/EvidenceLink.lean:82-147` — exact diagnostic matrix.
- `model/Umpire/Observation/Tests/Mutations.lean:200-270` — raw wrapper mutation patterns.
- `model/Umpire/Observation/Tests/Disposition.lean:9-38` — disposition fixtures that update accepted records.
- `model/Umpire/Observation/Tests/Verdict.lean:38-48,160-172` — verdict fixtures that forge accepted values.
- `model/Umpire/Observation/ImportTests.lean:5-19` — public facade contract.

### Key context
Preserve live `BEq`, `DecidableEq`, and `Repr` behavior where tests or consumers demonstrate it, but do not let those instances expose a constructor. Raw Evidence must remain absent from the accepted value.

### Quick commands
```bash
cd model && mise exec -- lake build Umpire.Observation.Tests
```
## Acceptance
- [ ] The ordinary `EvidenceBackedTrace` is opaque and only successful runtime admission can produce it.
- [ ] `ObservationResult.accepted` carries the opaque accepted type; every invalid evaluation still returns the exact existing non-success status and diagnostic.
- [ ] EvidenceLink, mutation, disposition, and verdict fixtures mutate unchecked carriers and retain complete negative coverage without exporting a production constructor or record-update path.
- [ ] Missing, duplicate, extra, shifted, zero, inconsistent, ordering, closure, disposition, bound, identity, and fingerprint failures preserve exact precedence and related-ID order.
- [ ] Valid accepted projections, equality/debug behavior required by current consumers, trace fingerprints, and Evidence Links remain identical.
- [ ] Public import checks expose the semantic accepted type but not an unchecked ordinary construction path.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
