---
satisfies: [R2]
---
# fn-254-fast-validation-gates-and-a-real-inner.6 make model-quick: affected crates, native always, cached TLC for unchanged modules


## Description
The author's inner loop (spec R2): edit a Model, get native, interpreter and (where needed) TLC answers in under a minute.

**Size:** M
**Files:** native framework quick command and Temporal selection extension; their actual CLI/Make registration; TLC cache and backend call sites; focused unit/cache controls. Use the established xtask home, without introducing a quick shell wrapper.
**Touches:** [Makefile, model/dev/xtask/src/main.rs, model/dev/xtask/src/quick/**, model/umpire/dev/xtask/src/main.rs, model/umpire/dev/xtask/src/lib.rs, model/umpire/dev/xtask/src/quick/**, model/umpire/dev/testing/src/**, model/umpire/backends/tlc/src/**, model/umpire/backends/tlc/tests/**, model/umpire/backends/process/src/fingerprint.rs, model/umpire/backends/process/src/lib.rs]

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

## Throughput handoff

The correct five-consumer content-edit workload passed in 444.027 s, with 388.875 s in realization tests and 14 TLC cache hits/zero executions. Preserve this complete coverage and report the 60 s miss. .5 owns compiler-case batching/reuse and fixture granularity; .17 owns nextest reuse and stage scheduling. Earlier approximately 20 s runs omitted consumers and remain historical incomplete coverage. Neither lower test selection nor a cache-only argument satisfies the inner-loop contract.

## Done summary
Implemented affected-package quick validation and bounded raw TLC artifact caching with current decoder replay, complete configuration keys, atomic publication, corruption/concurrency controls and authoritative full-gate bypass. Verified sibling configurations cache after complete module assembly; refuted/error configurations remain fresh. Identity work is shared per invocation and refreshed between calls; strict cache-root refusals retain their cause.

All real activity source consumers and shared Kettle consumers are selected. Test-only Kettle readers retain full package tests without unrelated reverse expansion. Known metadata/prose is reported ignored; workspace README/docs, package/mapping ownership, explicit globals and unknown external build inputs remain conservative, including checkouts beneath .flow.

Current controls: 16 quick tests, 12 cache tests, strict focused clippy/format, red/green selection and publication controls. Real Kettle module cold/warm: 4.593/1.591 s; warm verified siblings hit and negative claim executes fresh. Complete six-consumer shipping Kettle edit: 254.858/129.783 s cold/warm, 60 fixture tests passed and one ignored, native/interpreter agreement retained; final cdfd3720c8 select-only confirms the exact same six consumers with dirty Flow evidence ignored. Locks unchanged and isolated source restored clean. Joined/source/artifact hashes verified in task6/resumed.

The 60 s target is missed honestly: warm fixture stage 99.527 s and TLA tests 20.229 s dominate. The retained complete five-consumer activity workload remains 444.027 s (388.875 s realization tests); the Kettle measurement does not replace it. .5/.17 own subsequent throughput improvements. Full model-check/model-validate and joined final measurements remain .8 obligations under owner cadence; they were not reported green here. Retained source-projected guest/cache/input proof remains applicable; current cache changes have renewed focused and real-TLC evidence.

stage: impl-review - ran (three-lens NEEDS_WORK, fixed, re-review SHIP; fresh README follow-up SHIP, model: claude-fable-5-1 high). The optional P3 additional nested-workspace assertion had no failing scenario and was omitted at the evidence gate.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Source preparation checkpoint (2026-10-26)

The retained review identifies an additional shared-source consumer outside the Temporal
feature mapping: `sample_kettle/model.rs` is read directly by `umpire-reference-kettle`,
`umpire-suite`, `umpire-tla` and `umpire-apalache`. The reference's reverse dependent is
`umpire-kettle-guest`. Bring the source-selection seed prepared in .17 (`836242a09a`)
into this task without the nextest/compiler-only changes, expanding it to the other
three direct consumers found in fresh source review. The regression graph requires
all six affected packages, excludes an unrelated package, and selects only the fixture
package for its tests-only edit. Deleted Model paths use the same selection.

This is source preparation under the current no-new-validation instruction. The new
control and the real Kettle content-edit path still require execution when validation
resumes; retained Temporal and cache evidence does not prove this correction. Keep the
task open and reconcile this overlapping hunk when integrating .17. The three retained
SHIP draws were not finalized; their abandoned reservation is not a completed review.

## Evidence
- Commits: 80870b1695, 413bd2c351, d0469357c1, cb9bedfce8, c88fdc5506, 0c83ab3d1e, cdfd3720c8
- Tests: mise exec -- cargo test --locked -p xtask --lib quick:: (16 passed), mise exec -- cargo test --locked -p umpire-tlc --lib cache -- --nocapture (12 passed), strict focused cargo clippy and cargo fmt for xtask and umpire-tlc, task6/resumed/cache-review-kettle.py (real TLC cold/warm module proof), task6/resumed/joined-validate.py (six-consumer shipping quick cold/warm), task6/resumed/joined-final-selection.py (final source and dirty Flow selection), retained consumer-fix/proof.json and quick-proof.json source-projected complete activity/cache/bypass/guest proof
- PRs: