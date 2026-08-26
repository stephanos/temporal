---
satisfies: [R1, R6]
---
# fn-24-lean-native-verification-receipts-and.1 Define canonical formal receipt and trust vocabulary

## Description
Create the domain-neutral `Umpire.Formal` vertical module and facade. Define the closed checker/outcome/claim/trust/reason/error, request, evidence, candidate, replay, diagnostics, and receipt data contracts from the spec with private matched/result/receipt constructors. Implement exact canonical serializers, canonical ordering and bounds, candidate/replay/receipt identity derivation, source normalization, and constructor validation without Temporal vocabulary or backend semantics. Reuse existing canonical target/Query/Property/bounds values rather than duplicating semantic fields and preserve existing comments.

**Size:** M
**Files:** `model/Umpire/Formal/Language.lean`, `model/Umpire/Formal/Canonical.lean`, `model/Umpire/Formal.lean`, `model/Umpire/Formal/Tests/Canonicalization.lean`, `model/Umpire/Formal/Tests/AntiForgery.lean`
**Touches:** [model/Umpire/Formal/Language.lean, model/Umpire/Formal/Canonical.lean, model/Umpire/Formal.lean, model/Umpire/Formal/Tests/Canonicalization.lean, model/Umpire/Formal/Tests/AntiForgery.lean]

## Acceptance
Canonical fixtures pin every field, enum spelling, order, null, source projection, typed bound, and identity formula. Tests reject caller-supplied or crossed outcome/trust/digest/identity, duplicate/unknown/oversized evidence/reasons/omissions, noncanonical setup/trace/evaluation ordering, malformed hashes/sources, and every identity-bearing single-field mutation; diagnostics-only mutations preserve receipt identity. Reusable sources contain no Temporal/Workflow/Nexus/Veil/runtime/promotion vocabulary or dependency.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
