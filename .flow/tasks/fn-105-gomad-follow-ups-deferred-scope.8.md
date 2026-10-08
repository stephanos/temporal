---
satisfies: [R8]
---
# fn-105-gomad-follow-ups-deferred-scope.8 D8: closure-mode support for downstream targets

## Description


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.6](../tasks/fn-128-gomad-deferred-linux-qualification-and.6.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Shared consumer/source reconciliation, reviews, adapters and Darwin analyses/packs/exact replay/guidance; checkout prerequisite. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.


Origin: fn-104 C3/R2. Deferred 2026-09-29 because F9 qualifies in linked mode. Revived 2026-09-30 by fn-107 R7/R9: the downstream workflow manifest requires closure-mode preparation. Consume the final fn-107.4 target after its native workflow smoke passes; do not duplicate this implementation in fn-107. Add exact signal-handling metrics adapters only for libraries still reachable in the final closure, with per-file digests, inventories, per-platform pins and upstream-edit/version negative tests. Elimination is acceptable with fresh closure evidence. Actual final-target analysis additionally found Sentry optional Git release discovery through Pebble/Cockroach errors: add an exact pinned Sentry adapter that skips subprocess discovery with explicit diagnostics and unknown release, preserving explicit/environment/build-info releases and native upstream behavior. Never admit os/exec or os/signal in packs. Adapter/source freeze permits D9 pack preparation before D8 closes; D9 execution qualification follows D8 supported analyses and source review. Files/Touches: tools/gomad3 adapter registry, exact adapter overlays and focused tests. Quick: focused adapter/prepare/doctor tests, generator checks and final-target closure analysis.

The actual downstream closure is supported on both platforms after v13, but strict linked analysis retains denied native fallback callbacks. Extend exact pinned adapters for Sprig DNS, Validator address resolution, Cactus StatsD UDP, memberlist native UDP, Pebble native VFS, and existing gRPC DNS/Sockaddr interface discovery. Unsupported alternatives must visibly refuse, while ordinary validation/templates/metrics, injected TCP membership and MemFS remain functional. Keep native upstream unchanged, preserve comments, bind exact module/file/inventory/platform identities, and verify real refusal/working-path/drift behavior. Record paired application seams for configured IP/injected WAL and Storage literal TCP in fn-107.5; no boundary disposition weakening or guarded substitute. Refresh the profile and Darwin pack only after coordinated source freeze. Linux packs and native qualification belong to fn-128.6 and fn-128.7.

Deferral (2026-10-04): fn-107 now closes its implementation scope only.
Its original qualification requirements remain the acceptance reference, not
passing evidence. This task stays open under its unchanged acceptance; references
to fn-107.5 consuming/reconciling evidence describe the pre-closure workflow.
D8 retains adapters and supported analyses; D9 retains final consumer/source/native
reconciliation and reviews, packs/driver and both-platform workflow/exact replay;
D10 retains qualification-bound guidance and documentation. Resume when the
downstream checkout and qualified hosts are available. No qualified downstream
support claim or D12 waiver follows from the fn-107 closure.

## Acceptance


Owner amendment (2026-10-07). This task's remaining native darwin/arm64 execution, native reports/packs/replay, native qualification measurements, soak and platform-specific qualification guidance transfer to [fn-149-gomad-deferred-darwin-qualification](../specs/fn-149-gomad-deferred-darwin-qualification.md), with exact owners in the [native transfer manifest](../artifacts/native-scope-transfer-2026-10-07.md). Native linux/amd64 qualification and Linux CI work remain deferred under fn-128. Missing transferred native evidence cannot block this task or its source admission. This supersedes older native-first, missing-Darwin and no-renewed-deferral clauses only for transferred obligations. Implementation, ordinary host-source coverage, lint, both-source-set static checks, generated-output validation, byte equivalence, fixed-identity/matched-first-baseline preservation, non-native measurements, source review, docs consistency, actual checkout prerequisites and predecessor source integration/review/retained acceptance remain required. Full native test-host execution belongs to the native owner; partial portable runs cannot stand in for it or excuse portable failures. All other criteria and historical evidence below retain their original meaning. No task completion, native pass, PR, push or CI action follows from this transfer.

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.6](../tasks/fn-128-gomad-deferred-linux-qualification-and.6.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.


Revival is recorded. Closure-mode analysis of the final downstream Storage-backed workflow target reports supported on qualified Darwin here; Linux supported-analysis reports and native qualification belong to fn-128.6 and fn-128.7. Every reachable metrics adapter is exact, excludes signal registration, has identity-drift negative evidence and passes its Darwin native focused tests; otherwise prove the library was eliminated. Linux native focused tests belong to fn-128.6 and fn-128.7. fn-107.5 consumes the reports.

## Done summary

TBD

## Evidence

- Commits:
- Tests:
- PRs:
