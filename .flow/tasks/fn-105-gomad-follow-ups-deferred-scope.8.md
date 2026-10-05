---
satisfies: [R8]
---
# fn-105-gomad-follow-ups-deferred-scope.8 D8: closure-mode support for downstream targets

## Description

Origin: fn-104 C3/R2. Deferred 2026-09-29 because F9 qualifies in linked mode. Revived 2026-09-30 by fn-107 R7/R9: the downstream workflow manifest requires closure-mode preparation. Consume the final fn-107.4 target after its native workflow smoke passes; do not duplicate this implementation in fn-107. Add exact signal-handling metrics adapters only for libraries still reachable in the final closure, with per-file digests, inventories, per-platform pins and upstream-edit/version negative tests. Elimination is acceptable with fresh closure evidence. Actual final-target analysis additionally found Sentry optional Git release discovery through Pebble/Cockroach errors: add an exact pinned Sentry adapter that skips subprocess discovery with explicit diagnostics and unknown release, preserving explicit/environment/build-info releases and native upstream behavior. Never admit os/exec or os/signal in packs. Adapter/source freeze permits D9 pack preparation before D8 closes; D9 execution qualification follows D8 supported analyses and source review. Files/Touches: tools/gomad3 adapter registry, exact adapter overlays and focused tests. Quick: focused adapter/prepare/doctor tests, generator checks and final-target closure analysis.

The actual downstream closure is supported on both platforms after v13, but strict linked analysis retains denied native fallback callbacks. Extend exact pinned adapters for Sprig DNS, Validator address resolution, Cactus StatsD UDP, memberlist native UDP, Pebble native VFS, and existing gRPC DNS/Sockaddr interface discovery. Unsupported alternatives must visibly refuse, while ordinary validation/templates/metrics, injected TCP membership and MemFS remain functional. Keep native upstream unchanged, preserve comments, bind exact module/file/inventory/platform identities, and verify real refusal/working-path/drift behavior. Record paired application seams for configured IP/injected WAL and Storage literal TCP in fn-107.5; no boundary disposition weakening or guarded substitute. Refresh the profile and both platform packs only after coordinated source freeze.

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

Revival is recorded. Closure-mode analysis of the final downstream Storage-backed workflow target reports supported on both qualified platforms. Every reachable metrics adapter is exact, excludes signal registration, has identity-drift negative evidence and passes its native focused tests; otherwise prove the library was eliminated. fn-107.5 consumes the reports.

## Done summary

TBD

## Evidence

- Commits:
- Tests:
- PRs:
