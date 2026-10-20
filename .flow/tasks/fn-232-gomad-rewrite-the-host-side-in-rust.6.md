---
satisfies: [R5, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.6 replay in Rust (exact and observed) on the helper library tape path

Touches: tools/gomad/rust/gomad/src/cli/replay.rs, gomad/src/replay/{preflight.rs,divergence.rs}

## Description
Implement preflight (helper, module, runtime and implementation identity checks without reopening build inputs), exact replay through the helper library's fn-228 tape path, observed replay for the stock profile, divergence projection with ordinal, reason, expected and observed decisions, and the storage-crash interrupted path. Preflight and execution use task .3's authenticated pinned-root handles throughout, without reopening the artifact pathname or build inputs; malformed/boundedness/digest failures start no guest. Add replay to the harness using task .4's declared golden view and independent raw-integrity verification.

## Acceptance
The owned behavior has reviewed golden fixtures, repeated-execution agreement and byte-identical second regeneration (R14); public verbs also run their fresh Go/Rust semantic parity lane (R13).
Harness replay lane shows the same match/first-divergence ordinal and reason as Go on its own artifacts across the corpus; identity mismatch, missing payloads and divergence produce today's statuses and messages. CLI fault fixtures refuse substituted/symlink roots, traversal, wrong private modes/links, oversized or changed payloads and digest corruption before guest start, and prove post-open pathname replacement cannot redirect the pinned reads; intentional target-pool module hard links still replay.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
