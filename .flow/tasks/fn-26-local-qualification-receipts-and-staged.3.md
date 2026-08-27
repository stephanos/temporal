---
satisfies: [R5, R7]
---
# fn-26-local-qualification-receipts-and-staged.3 Add EvaluationReceipt v2 and the post-v2 ArtifactSet successor

## Description
Add the exact bounded canonical `umpire-evaluation-receipt/v2` schema and strict Lean/Go codecs. Bind the named Evaluation Profile Behavior Fingerprint, source Result and Artifact Checksums, model Behavior Fingerprints, independent statuses, decision, claim strength, Limits, Known Gaps, Evidence Links, and cleanup. Define a post-v2 ArtifactSet successor containing the byte-identical v2 source members plus exactly one receipt and one typed receipt-to-Result relation; add no pre-v2 reader or migration route.

## Acceptance
- [ ] Cross-language v2 receipt goldens and Artifact Checksums agree byte-for-byte.
- [ ] The successor set preserves source bytes and validates exact receipt/Result closure.
- [ ] Unknown, duplicate, oversized, noncanonical, crossed, pre-v2, or checksum-invalid input rejects.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
