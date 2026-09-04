---
satisfies: [R5, R7]
---

# fn-26-local-qualification-receipts-and-staged.3 Add canonical Evaluation Receipts and publication closure
## Description
Define exact bounded Lean/Go Evaluation Receipt codecs binding Profile, Case, Program/Contract, preparation Profile/catalog, live Host, Run, Verdict/supporting events, cleanup, decision/reasons, evidence, Limits, and Known Gaps. Add immutable publication without modifying source Case Runtime values.

**Size:** L
**Touches:** `model/Umpire/Evaluation/Receipt.lean`, `api/umpire/**`, `tools/umpire/evaluation/receipt.go`, `tools/umpire/artifact/**`

## Acceptance
- [ ] Cross-language canonical bytes and identities agree.
- [ ] Unknown, duplicate, oversized, crossed, secret-bearing, incompatible, or invalid closure rejects.
- [ ] Same subject/Profile content publishes idempotently; different Profiles yield distinct receipts.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
