---
satisfies: [R1]
---
# fn-212-tla-lifting-and-the-tlc-backend-for.1 Decide: is the second opinion worth the lifting and TLC machinery?

## Description
Write `model/design/second-opinion.md`: what TLC adds over the native checker on the Models then available (independent reading, exhaustive agreement, liveness), what it costs (`umpire-lift`, the subset's constraint on authors, `umpire-tlc`, a JVM in the gate, upkeep), and the choice of lifting source (`#[machine]` token recording or a build-time `syn` pass). On "no": close this spec and fn-211 citing the note.

**Files:** model/design/second-opinion.md.
**Touches:** [model/design/second-opinion.md]

**Quick commands:** none

Parallel execution authorized by the owner on 2026-10-19. Implement the decision note in an isolated worktree; the conductor owns review, task completion, and any consequent spec closures after local integration.
## Acceptance
- Note with cost, benefit, lifting-source choice and a yes or no.
- On no: fn-212 and fn-211 closed with a reference.

## Done summary
The owner requires TLC for scaling larger Models. The integrated decision note selects build-time syn lifting, describes current native evidence and source/subset costs, and states the maximal-path/stuttering encoding as a candidate requiring real TLC controls. It does not change Model transitions or silently add fairness. Fn-212.2 implements the backend; fn-212 and fn-211 remain open.

Worker commits 567f0b38b8 and 3423f87d5c, plus root planning commit 654d639244, were integrated on umpire in 4f93a3be4d. The clean decision worktree and its integrated branch were removed after handoff; all commits remain in umpire history.

Verification: git diff --check passed; local Markdown references and source/fixture claims were checked by the worker, conductor and independent reviewers. Quick commands are none. No compilation or backend execution is claimed for this decision-only task. The classifier's unmatched-path FULL fallback was recorded in the worker handover; it does not supply a test requirement for this documentation task.

Formal implementation review used three fresh read-only Codex contexts on gpt-6.1-sol at high, all SHIP with zero findings; writer and reviewers share the Codex family. Receipt: /tmp/impl-review-receipt-8f37faba39e2-fn-212-tla-lifting-and-the-tlc-backend-for.1.json. Fan-out: .flow/review-fanout/1b3ac0c0604c441e9fc72eaea5562aa1. Root reviewed the integrated diff from e721b61f6f.

stage: impl-review - ran (model: gpt-6.1-sol at high)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tier: session (jev-unavailable(no_key))
Tracker sync: n/a (bridge inactive)

No publication or external messages were sent. The settled yes decision supersedes the original conditional option to close these specs without a backend.
## Evidence
- Commits: 567f0b38b89b16f16fa3ba5678d2c3fbbd7af7b6, 3423f87d5c222c9c09ae1c7579f4d8b7df6b5f36, 4f93a3be4d
- Tests: git diff --check e721b61f6f..HEAD, Local Markdown references and source/fixture claims verified; task Quick commands none, Three-draw Codex implementation review: SHIP, zero findings; gpt-6.1-sol at high
- PRs: