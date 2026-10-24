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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
