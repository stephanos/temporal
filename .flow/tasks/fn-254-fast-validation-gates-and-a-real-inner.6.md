---
satisfies: [R2]
---
# fn-254-fast-validation-gates-and-a-real-inner.6 make model-quick: affected crates, native always, cached TLC for unchanged modules


## Description
The author's inner loop (spec R2): edit a Model, get native, interpreter and (where needed) TLC answers in under a minute.

**Size:** M
**Files:** native framework quick command and Temporal selection extension; their actual CLI/Make registration; TLC cache and backend call sites; focused unit/cache controls. Use the established xtask home, without introducing a quick shell wrapper.
**Touches:** [Makefile, model/dev/xtask/src/main.rs, model/dev/xtask/src/quick/**, model/umpire/dev/xtask/src/main.rs, model/umpire/dev/xtask/src/lib.rs, model/umpire/dev/xtask/src/quick/**, model/umpire/dev/testing/src/**, model/umpire/backends/tlc/src/**, model/umpire/backends/tlc/tests/**, model/umpire/backends/process/src/**]

### Parallel ownership
- Own the listed quick command, CLI/Make wiring, cache/backend source and narrow backend-process hashing/owned artifact-I/O helpers. Reuse existing dependencies and the established external framework executable bridge; no Cargo manifest or lock edit is needed for this design. Keep quick command controls as unit tests within its owned quick source; TLC cache controls use the owned TLC test tree. Framework xtask integration-test aggregation belongs to .11. Task .8 owns consolidated README/tier/cache documentation; .12 owns budgets.
- .10 already forces the authoritative full-gate cache-disable policy before invoking stages. Consume that exact policy in the cache and prove populated-cache bypass here; do not edit the full-gate implementation or .3's working `test_run.rs` leaves. If actual implementation requires an additional owner's file, amend Touches and serialize instead of inventing narrower ownership.
- Preserve coarse full-lock/driver/compiler guest identity. Required changed-input guest reproduction is not waived by parallelism. If actual generated committed bytes change, declare those exact paths and recheck admission before publication; unchanged-byte reproduction remains evidence, not a fabricated source touch.

### Existing-dependency cache design
- Cache the existing bounded TLC output/dump execution artifacts, not serialized typed results. Re-enter the same current completion/property/error/limit and schema/dump validators on a hit. Publish only a complete all-Verified configuration bundle; withhold publication for failed/refuted module configurations. Key the entire effective configuration inventory and current schema, with per-configuration hit/miss reporting.
- Reuse Type/Bounded's existing structural Hash implementations for schema identity through a narrow SHA-256 helper in backend-process, which already owns sha2/tempfile/filesystem support. Version the key/runner/decoder/manifest encoding and keep a strict bounded raw-artifact manifest, complete byte/hash/count checks and atomic private staging/publication. Do not add a hand-written typed-state serializer or dependency on native tooling in TLC.
- Keep both existing lockfiles unchanged and verify their hashes plus locked metadata/build admission on the implemented candidate. If a genuine implementation need requires another dependency/lock/shared path, amend Touches before editing and restore serial admission; parallelism never overrides correct dependency declarations or the full acceptance criteria.

### Approach
- Change set: `git diff --name-only` against `BASE` (default `HEAD`) plus untracked files. Map files to crates and add reverse dependencies from `cargo metadata`. Run `cargo fmt --check`, clippy on those crates, and their tests.
- Handle renames/deletions and cross-workspace reverse dependencies. Invalid BASE fails explicitly. Manifest/lock/tool/build-input changes or unmappable paths conservatively select the affected whole workspace(s). State the unchanged-invocation selection so native/interpreter execution remains explicit.
- TLC cache: content key covers emitted module, all effective cfg/progress/negative inputs, decoding schema, TLC jar SHA, Java version and runner/serialization version. Cache only complete successful results; refutations and execution/decoding failures run fresh and retain failing verdicts. Publish atomically under the target and validate the embedded key/schema/completeness on read. Corrupt/truncated/wrong-key/interrupted/concurrent writers yield a complete validated hit or a fresh run, never partial success. Report hit/miss per configuration.
- Full gates explicitly disable cache use, overriding inherited quick-cache environment. Prove fresh execution with a populated cache and inherited enablement, not merely by omitting a flag.
- Use the established xtask command home; put quick source in the listed `src/quick/**` trees and register the real command in the owned dispatcher/Make files.
- Benchmark a reversible content edit on an isolated exact candidate snapshot, assert activity selection and native/interpreter execution, and restore it without overwriting user work. Measure both an emission-preserving cache hit and changed-emission fresh TLC; an mtime touch does not test Git selection. Place framework logic under .10's entry point and keep Temporal selection in its extension. Register both paths for .12.

### Investigation targets
**Required:**
- `model/scripts/xtask.sh` and the xtask crate — current tooling home after fn-247
- `model/umpire/dev/testing/src/backend.rs` — where fixtures call TLC
- `model/umpire/backends/tlc/src/runner.rs` — result types to cache

## Owner validation cadence (2026-10-25)

Keep actual activity quick checks, native/interpreter selection, hit/miss/invalidation/corruption/concurrency controls and authoritative full-gate cache bypass now. Unrelated full model gates wait for .8. Deferred checks remain final .8 requirements and are never reported green. This owner instruction supersedes earlier broad per-task verification commands.

## Acceptance
- [ ] An activity `model.rs` edit re-runs the activity natively and through the interpreter; TLC runs only if its emitted module changed.
- [ ] Cache controls exercise every effective input, corruption/interruption/concurrent publication and failure exclusion; inherited enablement cannot make `model-validate` read a cache. Selection controls cover rename/delete/global inputs and cross-workspace reverse dependencies.
- [ ] Warm `model-quick` after an activity edit is under 60 s, or the miss is reported with its cause.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
