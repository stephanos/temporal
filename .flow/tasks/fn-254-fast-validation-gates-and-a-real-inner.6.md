---
satisfies: [R2]
---
# fn-254-fast-validation-gates-and-a-real-inner.6 make model-quick: affected crates, native always, cached TLC for unchanged modules


## Description
The author's inner loop (spec R2): edit a Model, get native, interpreter and (where needed) TLC answers in under a minute.

**Size:** M
**Files:** `Makefile` (`model-quick`), `model/scripts/quick.sh` (new) or an xtask subcommand if fn-247's xtask is the convention, the TLC result cache in `umpire/dev/testing` or the runner.
**Touches:** [Makefile, model/scripts/quick.sh, model/scripts/xtask.sh, model/dev/xtask/**, model/umpire/dev/xtask/**, model/umpire/dev/testing/src/**, model/umpire/backends/tlc/src/**]

### Approach
- Change set: `git diff --name-only` against `BASE` (default `HEAD`) plus untracked files. Map files to crates and add reverse dependencies from `cargo metadata`. Run `cargo fmt --check`, clippy on those crates, and their tests.
- Handle renames/deletions and cross-workspace reverse dependencies. Invalid BASE fails explicitly. Manifest/lock/tool/build-input changes or unmappable paths conservatively select the affected whole workspace(s). State the unchanged-invocation selection so native/interpreter execution remains explicit.
- TLC cache: content key covers emitted module, all effective cfg/progress/negative inputs, decoding schema, TLC jar SHA, Java version and runner/serialization version. Cache only complete successful results; refutations and execution/decoding failures run fresh and retain failing verdicts. Publish atomically under the target and validate the embedded key/schema/completeness on read. Corrupt/truncated/wrong-key/interrupted/concurrent writers yield a complete validated hit or a fresh run, never partial success. Report hit/miss per configuration.
- Full gates explicitly disable cache use, overriding inherited quick-cache environment. Prove fresh execution with a populated cache and inherited enablement, not merely by omitting a flag.
- Prefer xtask over a new shell script if fn-247 established xtask as the home for gate tooling; follow whichever is current.
- Benchmark a reversible content edit on an isolated exact candidate snapshot, assert activity selection and native/interpreter execution, and restore it without overwriting user work. Measure both an emission-preserving cache hit and changed-emission fresh TLC; an mtime touch does not test Git selection. Place framework logic under .10's entry point and keep Temporal selection in its extension. Register both paths for .12.

### Investigation targets
**Required:**
- `model/scripts/xtask.sh` and the xtask crate — current tooling home after fn-247
- `model/umpire/dev/testing/src/backend.rs` — where fixtures call TLC
- `model/umpire/backends/tlc/src/runner.rs` — result types to cache

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
