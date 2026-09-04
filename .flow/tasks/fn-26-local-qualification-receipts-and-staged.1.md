---
satisfies: [R1, R3, R7]
---

# fn-26-local-qualification-receipts-and-staged.1 Define reusable Evaluation Profiles and local policy
## Description
Define the Temporal-free Evaluation Profile contract and one Temporal-owned `local-ephemeral` instance over Case Runtime outcomes. Freeze claim, Case/Run/Verdict requirements, cleanup, evidence, trust, Limits, Known Gaps, stable identity, and deterministic accumulating reason precedence.

**Size:** M
**Touches:** `model/Umpire/Evaluation.lean`, `model/Umpire/EvaluationTests.lean`, `model/Temporal/System/Evaluation/Local.lean`

## Acceptance
- [ ] Empty, duplicate, contradictory, unknown, stale, and N+1 inputs fail deterministically.
- [ ] No endpoint, credential, path, Host, execution authority, or Temporal value enters reusable Umpire.
- [ ] Same Profile bytes yield the same identity; a different Profile remains an independent assessment.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
