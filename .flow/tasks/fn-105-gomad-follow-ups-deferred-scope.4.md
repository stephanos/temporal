---
satisfies: [R4]
---
# fn-105-gomad-follow-ups-deferred-scope.4 D4: architecture fitness checks for package coverage, purity, and signature visibility

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: fn-109-owned implementation, static both-source-set checks, preservation, review and Darwin gates. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Origin: fn-102.5 (F8 R5). Originally deferred 2026-09-29; revived 2026-09-30 by the architecture request under fn-109 R8. Reuse or transfer this obligation exactly once during task breakdown. Brief: .flow/tasks/fn-102-gomad-architecture-consolidate.5.md. Fn-109 supplies current discovery, negative-fixture, and qualification criteria.

## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

Revival is recorded. Satisfy original fn-102 R5 together with fn-109 R8. Discover both qualified platform source sets and retain rejecting fixtures for ownership, host effects, and inaccessible public signatures. Cross-referencing or transferring the task does not claim completion.

## Done summary
Blocked:
# D4 acceptance remains open under its current owner

Fn-109.19 is the sole implementation owner of D4 (fn-105.4), as its current
task specification states. It runs after the changed progress and backend owners,
with complete package discovery, targeted host-effect/public-signature rules and
executable negative fixtures for both qualified platform source sets.

Current architecture checks and task references alone do not satisfy that scope.
Do not close fn-105.4 until fn-109.19 supplies its required source-bound evidence;
then close by reference exactly once. The prior fn-109.15 adopting-task reference
is obsolete: task 15 now owns progress characterization/design, not architecture.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for fn-109-owned implementation, static both-source-set checks, preservation, review and Darwin gates. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
