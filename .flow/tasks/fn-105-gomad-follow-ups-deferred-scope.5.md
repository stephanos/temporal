---
satisfies: [R5]
---
# fn-105-gomad-follow-ups-deferred-scope.5 D5: reconcile architecture, platform, and determinism documentation

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: fn-109-owned implementation, static both-source-set checks, preservation, review and Darwin gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Origin: fn-102.6 (F8 R6). Originally deferred 2026-09-29; revived 2026-09-30 by the architecture request under fn-109 R9. Reuse or transfer this obligation exactly once during task breakdown. Brief: .flow/tasks/fn-102-gomad-architecture-consolidate.6.md. Fn-109 supplies current documentation and qualification criteria.

## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

Revival is recorded. Satisfy original fn-102 R6 together with fn-109 R9. Reconcile both qualified platforms, implemented replay/exploration, both backends, residual findings, and intentional public migrations against current evidence. Cross-referencing or transferring the task does not claim completion.

## Done summary
Blocked:
# D5 acceptance remains open under its current owner

Fn-109.20 is the sole documentation owner of D5 (fn-105.5), as its current
task specification states. Reuse fn-111's valid vocabulary evidence, reconcile
the final delivered owners/interfaces, preserve separate capability/repeatability/
exact-replay/expectation claims, and retain residual findings and both-platform
qualification limits.

Do not close fn-105.5 until fn-109.20 supplies its required current guidance and
evidence; then close by reference exactly once. The prior fn-109.15 adopting-task
reference is obsolete: task 15 now owns progress characterization/design.

Task 20 current guidance now has formal three-draw SHIP. D5 remains blocked on its inherited original fn-102 R6 native and bounded-control qualification. [Current review and acceptance evidence](../artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-20/acceptance-open.md) retains the source checkpoint and actual verdict; it grants no qualification waiver or duplicate owner.

## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for fn-109-owned implementation, static both-source-set checks, preservation, review and Darwin gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
