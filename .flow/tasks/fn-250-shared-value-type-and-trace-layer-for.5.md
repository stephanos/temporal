---
satisfies: [R5, R6]
---
# fn-250-shared-value-type-and-trace-layer-for.5 PinnedJar admission with in-process sha2 and one identifier() for TLC and Apalache

## Description
Implement **Pinned jar admission (A9)**: `PinnedJar::admit(root, cache, jar, sha)` in `umpire-backend-process` (cache containment, in-process `sha2` digest, pinned Java, version probe) replaces `tlc/src/runner.rs:104-142` and `apalache/src/lib.rs:124-160`, including their `shasum -a 256` subprocesses (moved here from fn-248.6 so the digest change lands once); one `identifier()` replaces the tlc and apalache copies. Remove `shasum` from documented prerequisites; record `sha2` under fn-210 R18 if fn-247 has not already.
## Acceptance
- [ ] Both runners admit jars through `PinnedJar::admit`; no `shasum` invocation remains.
- [ ] Wrong digest, wrong Java or cache escape refuses before use.
- [ ] Re-accept goldens after reviewing their diffs; `make model-check` passes.
## Done summary
`PinnedJar::admit` (process crate, `jar.rs`) admits the TLC and Apalache jars: no symlink or `..` from root to jar, regular file, in-process `sha2` digest, pinned Java, then the tool's own version probe, which returns the tool. Every refusal happens before the tool runs; the scratch directory is cleaned up on every path. One `umpire_tla::identifier()` replaces both copies. No `shasum` call remains in either runner (`model/scripts/fetch-tlc.sh` keeps its fetch-time fallback, left for fn-252.7). TLC containment is now as strict as Apalache's (a symlinked `model/build` is refused). Fresh review SHIP; cleanups bb246d544d. Focused default and require-mode tests passed with the real tools; covered by the joined fn-248.7 `UMPIRE_APALACHE=require make model-validate` at 9b3e26654a (exit 0).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 1e304716dc, bb246d544d
- Tests: UMPIRE_BACKENDS=require UMPIRE_APALACHE=require cargo test -p umpire-backend-process -p umpire-tlc -p umpire-apalache -p umpire-tla, UMPIRE_APALACHE=require make model-validate (9b3e26654a, exit 0)
- PRs: