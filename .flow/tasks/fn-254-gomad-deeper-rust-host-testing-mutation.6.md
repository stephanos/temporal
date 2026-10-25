---
satisfies: [R6]
---
# fn-254-gomad-deeper-rust-host-testing-mutation.6 Miri for unsafe and occasional ThreadSanitizer

## Description
R6: Miri and ThreadSanitizer.

- **Miri** (`deep:miri`) runs the tests of the non-JIT modules containing `unsafe`, as identified by the survey. Modules Miri can't execute (anything that runs wasmtime-compiled code) are listed with what covers them instead.
- **ThreadSanitizer** (`deep:tsan`, nightly, `-Zsanitizer=thread`, separate target directory) runs the threading tests occasionally. Suppressions for wasmtime/JIT false positives live in a committed suppressions file, each with a reason.

Neither enters the fast gates or the cycle.

## Acceptance
- `deep:miri` passes over the surveyed modules; the uncovered `unsafe` sites are listed with their alternative coverage.
- `deep:tsan` runs with a reasoned suppressions file; any real race found is fixed and gets a shuttle regression test.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
