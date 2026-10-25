---
satisfies: [R2]
---
# fn-254-fast-validation-gates-and-a-real-inner.6 make model-quick: affected crates, native always, cached TLC for unchanged modules

Touches: [Makefile, model/scripts/quick.sh, model/umpire/dev/testing/src/**, model/umpire/backends/tlc/src/**]

## Description
The author's inner loop (spec R2): edit a Model, get native, interpreter and (where needed) TLC answers in under a minute.

**Size:** M
**Files:** `Makefile` (`model-quick`), `model/scripts/quick.sh` (new) or an xtask subcommand if fn-247's xtask is the convention, the TLC result cache in `umpire/dev/testing` or the runner.
**Touches:** [Makefile, model/scripts/quick.sh, model/umpire/dev/testing/src/**, model/umpire/backends/tlc/src/**]

### Approach
- Change set: `git diff --name-only` against `BASE` (default `HEAD`) plus untracked files. Map files to crates and add reverse dependencies from `cargo metadata`. Run `cargo fmt --check`, clippy on those crates, and their tests.
- TLC cache: key = SHA-256 over emitted module text, cfg text, TLC jar SHA, `java -version` output and the runner's version constant. Value = the exact `TlcResult`. Store under the target directory. Enabled only by an env var that `model-quick` sets; `model-validate` never sets it. Report hit or miss per fixture.
- Prefer xtask over a new shell script if fn-247 established xtask as the home for gate tooling; follow whichever is current.
- Update `time-gate.sh` with the realistic loop: touch `temporal/feature/activity/model.rs`, run `model-quick`, record the time.

### Investigation targets
**Required:**
- `model/scripts/xtask.sh` and the xtask crate — current tooling home after fn-247
- `model/umpire/dev/testing/src/backend.rs` — where fixtures call TLC
- `model/umpire/backends/tlc/src/runner.rs` — result types to cache

### Acceptance
- [ ] An activity `model.rs` edit re-runs the activity natively and through the interpreter; TLC runs only if its emitted module changed.
- [ ] Cache controls: changed module, cfg, jar, Java or runner version each miss; a planted entry with a wrong key is ignored; `model-validate` never reads the cache.
- [ ] Warm `model-quick` after an activity edit is under 60 s, or the miss is reported with its cause.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
