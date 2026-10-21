---
satisfies: [R5, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.6 replay in Rust (exact and observed) on the helper library tape path

Touches: tools/gomad/rust/gomad/src/cli/replay.rs, gomad/src/replay/{preflight.rs,divergence.rs}

## Description
Implement preflight (helper, module, runtime and implementation identity checks without reopening build inputs), exact replay through the helper library's fn-228 tape path, observed replay for the stock profile, divergence projection with ordinal, reason, expected and observed decisions, and the storage-crash interrupted path. Preflight and execution use task .3's authenticated pinned-root handles throughout, without reopening the artifact pathname or build inputs; malformed/boundedness/digest failures start no guest. Add replay to the harness using task .4's declared golden view and independent raw-integrity verification.

This task owns the complete existing replay grammar and its current WASM availability/refusal behavior, not just the exact/observed happy path: preserve --verify-only, --observed, the native-only --toolchain-root refusal, and replay inspection stop/stack/heap/dump/probe/search/stress controls in their actually supported profiles. Port the supported inspection-controller path through a typed linked-library seam where needed; no unspecified later owner, alternate subprocess protocol or silent flag deletion. Preserve fresh per-run state, existing bounds, diagnostics-only behavior and exact refusal/status semantics. Verify-only must exercise helper-owned semantic tape/trace validation even for a resealed artifact with valid storage digests, without executing a guest; authentication alone is not semantic validation.

Existing diagnostic role selectors map to matching compiled Rust binaries: preserve positive --determinism-fixture and --stress-jitter replay/inspection paths with explicit opt-in and actual linked helper-role identities. Ordinary release/qualification cannot activate these roles or private test-only controls. Document the old launcher-plus-helper to single-binary mapping and its plain-release availability difference, rather than silently dropping every positive path or claiming blanket refusals are parity. Separate host fixture hooks from helper-role feature forwarding; preserve ordinary-host fixture behavior. Fresh typed per-execution conditions and bounded event-order sinks replace ambient environment/global recorder state even in diagnostic builds. Match actual Go wrong-role/profile/conflict refusals and preserve truthful compiled identities; no ordinary-release role exception or alternate helper is authorized.

Cluster replay and cluster stop-inspection are also existing owned paths. Port the retained scenario and multiple-incarnation orchestration through the existing helper cluster core using a typed linked seam, with authenticated module handles/bytes and retained configuration/namespace/storage inputs. Preserve incarnation ordering, fresh stores, broker/clock/storage semantics and existing bounds; no path-based reopening, transport reconstruction, or blanket unsupported replacement. Task .8 receives and reuses this execution seam rather than building a second cluster implementation. Actual same-target Go/Rust cluster replay/inspection fixtures are required before acceptance.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness replay lane shows the same match/first-divergence ordinal and reason as Go on its own artifacts across the corpus; identity mismatch, missing payloads and divergence produce today's statuses and messages. CLI fault fixtures refuse substituted/symlink roots, traversal, wrong private modes/links, oversized or changed payloads and digest corruption before guest start, and prove post-open pathname replacement cannot redirect the pinned reads; intentional target-pool module hard links still replay.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
