---
satisfies: [R4]
---
# fn-241-gomad-a-race-profile-for-wasip1-deferred.11 gomad.wasi-race/v1 profile identity, per-run selection and refusal on mismatch

## Description
Touches: rust/gomad/src/install/{config,manifest,derive_cache}.rs, shim.rs profile checks, evidence profile field, replay admission. Define gomad.wasi-race/v1 as a third profile composing the cooperative overlay with the race overlay; its identity binds the rebuilt tool digests (.1), the race overlay identity (.6), the shadow policy (.7) and its limits (placeholders until .4 measures them). Add the per-run selector; default stays cooperative. The race profile is refused, not downgraded, when its toolchain, overlay or shadow-policy identity does not match the installation. Artifacts record their profile; replay of a race artifact on the cooperative profile is refused. Prove the cooperative profile's identities, overlay sources and qualified expectations are byte-unchanged. Depends on .6 and .7.

## Acceptance
Race profile has a recorded identity; mismatched installation is refused with a named reason; a race artifact is refused on the cooperative profile; cooperative identities, rewrite table and qualified expectations equal their pre-task bytes (golden diff); the cooperative CLI, manifest, status, environment and wire schema are unchanged.

## Done summary
Blocked:
Deferred by owner decision on 2026-10-23 (GOMAD_RACES.md §9: for memory races Gomad is the fourth-best detector; the fault matrix and scheduling bugs rank higher as proof of value). Revival trigger: the persistence fault matrix has saturated, the known-bug benchmark table exists, and it shows a data-race or lock-discipline bug that a workload under Gomad reaches but Gomad does not report. No implementation, PR, push or CI run until the owner revives this spec.

## Evidence
- Commits:
- Tests:
- PRs:
