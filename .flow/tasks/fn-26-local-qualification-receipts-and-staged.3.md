---
satisfies: [R5]
---
# fn-26-local-qualification-receipts-and-staged.3 Add QualificationReceipt v1 and ArtifactSet v2

## Description
**Size:** L
**Files:** model/Umpire/Artifact/**, tools/umpire/artifact/**
**Touches:** receipt family, set v2, strict publication

Extend the Lean canonical artifact vocabulary and tools/umpire/artifact with the exact 64-MiB/1,048,576-token bounded umpire-qualification-receipt/v1 schema, pathless ArtifactReference, nested limits, identities, strict reader/encoder, and a non-breaking umpire-artifact-set/v2 containing the six byte-identical v1 members plus one receipt and qualification-result relation. Keep fn-18 v1 decoders at 131,072 tokens. Bind only the source set identity reconstructible from those members, never the absent original-manifest byte digest. Reuse fn-18 path, recovery, and atomic publication logic; add no migration route.

## Acceptance
Canonical receipt and v2 fixtures round-trip byte-for-byte through every byte/token/cardinality equality/N+1 limit including the maximum omission union, every identity-bearing mutation fails or changes identity, ArtifactReference matches the path-bearing Result member exactly while excluding path from claim identity, v1 fixtures/readers and token ceilings remain byte-identical, v1 rejects v2, v2 enforces the exact seven-member closure/source-set/Result relation, and publication is atomic/idempotent with no rewritten input member.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
