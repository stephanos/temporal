---
satisfies: [R1, R6]
---
# fn-24-lean-native-verification-receipts-and.1 Define Umpire Verify Native receipts and trust vocabulary

## Description
### Umpire4 reconciliation (normative)

Place the generic native verification API under `Umpire.Verify.Native`. Receipts bind a named model-defined check/profile, source and Behavior Fingerprints, assumptions, Limits, Known Gaps, checker version, replay status, provenance, and honest trust class.

The legacy implementation detail below is retained for context but is subordinate to this reconciliation.

Create the domain-neutral `Umpire.Formal` vertical module and facade. Define the closed checker/outcome/claim/trust/reason/error, request, evidence, candidate, replay, diagnostics, and receipt data contracts from the spec with private matched/result/receipt constructors. Implement exact canonical serializers, canonical ordering and Limits, candidate/replay/receipt identity derivation, source normalization, and constructor validation without Temporal vocabulary or backend semantics. Reuse existing canonical target/Query/Property/Limits values rather than duplicating semantic fields and preserve existing comments.

**Size:** M
**Files:** `model/Umpire/Formal/Language.lean`, `model/Umpire/Formal/Canonical.lean`, `model/Umpire/Formal.lean`, `model/Umpire/Formal/Tests/Canonicalization.lean`, `model/Umpire/Formal/Tests/AntiForgery.lean`
**Touches:** [model/Umpire/Formal/Language.lean, model/Umpire/Formal/Canonical.lean, model/Umpire/Formal.lean, model/Umpire/Formal/Tests/Canonicalization.lean, model/Umpire/Formal/Tests/AntiForgery.lean]
## Acceptance
Canonical fixtures pin every field, enum spelling, order, null, source Generated View, typed Limit, and identity formula. Tests reject caller-supplied or crossed outcome/trust/digest/identity, duplicate/unknown/oversized evidence/reasons/Known Gaps, noncanonical setup/trace/evaluation ordering, malformed hashes/sources, and every identity-bearing single-field mutation; diagnostics-only mutations preserve receipt identity. Reusable sources contain no Temporal/Workflow/Nexus/Veil/runtime/promotion vocabulary or dependency.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
