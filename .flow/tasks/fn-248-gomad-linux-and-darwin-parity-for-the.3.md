---
satisfies: [R3, R4]
---
# fn-248-gomad-linux-and-darwin-parity-for-the.3 Replace hand-written C bindings with libc and signal-hook

## Description
Replace the hand-written `extern "C"` blocks in `qualify/usage.rs` (`getrusage`, `malloc_trim`), `wasmhost/src/metrics.rs` (`getrusage`) and `sweep/process.rs` (`signal`) with `libc` (pinned `=`), one portable `getrusage` wrapper normalizing `ru_maxrss` units, and `signal-hook` (pinned `=`) for sweep's SIGINT/SIGTERM flag. `malloc_trim` stays glibc-only behind one function.

## Acceptance
- [ ] No hand-written `extern "C"` or `Rusage` struct remains outside the supervisor (task 4).
- [ ] Sweep interruption tests pass on both platforms.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
