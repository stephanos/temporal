---
satisfies: [R10]
---
# fn-105-gomad-follow-ups-deferred-scope.10 D10: downstream-seam guide

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.6](../tasks/fn-128-gomad-deferred-linux-qualification-and.6.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Shared consumer/source reconciliation, reviews, adapters and Darwin analyses/packs/exact replay/guidance; checkout prerequisite. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.


Origin: fn-104 R4. Deferred 2026-09-29 pending another downstream adoption. Revived 2026-09-30 by fn-107 R12, which requires a reusable guide for the completed integration. Follow fn-105.9. Write the generic Gomad-side seam guide without downstream repository or component names; cover paired tags, native defaults, injected resources, closure/linked analysis, exact dependency policy, GOWORK=off replacements, CLI driver, qualification/replay and external-service limits. Add concrete downstream instructions alongside its profile, manifest and packs. Files/Touches: tools/gomad3 documentation and downstream localcell/gomad README. Quick: check commands against retained successful Darwin execution/pack evidence, documentation link checks.

Deferral (2026-10-04): fn-107 now closes its implementation scope only.
Its original qualification requirements remain the acceptance reference, not
passing evidence. This task stays open under its Darwin-owned acceptance; references
to fn-107.5 consuming/reconciling evidence describe the pre-closure workflow.
D8 retains adapters and supported analyses; D9 retains final consumer/source/native
reconciliation and reviews, packs/driver and Darwin workflow/exact replay;
D10 retains qualification-bound guidance and documentation. Resume when the
downstream checkout and qualified Darwin host are available. Linux workflow and
exact replay remain deferred under fn-128.6/.7. No qualified downstream
support claim or D12 waiver follows from the fn-107 closure.
## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.
Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.6](../tasks/fn-128-gomad-deferred-linux-qualification-and.6.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.


Revival is recorded. Generic guide satisfies original fn-104 R4 without naming downstream repositories/components. Consumer guide gives concrete repeatable Darwin commands and bounded support claims matching retained measured darwin/arm64 success. Shared contracts and static coverage of both supported source sets remain required. Linux qualification commands and support-claim evidence belong to [fn-128.6](../tasks/fn-128-gomad-deferred-linux-qualification-and.6.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md); missing transferred Linux evidence does not block this task or justify a Linux support claim. No commands rely on undocumented local edits or treat classified failures as passing. fn-107.5 reconciles the roadmap after consuming this guide.
## Done summary

TBD

## Evidence

- Commits:
- Tests:
- PRs:
