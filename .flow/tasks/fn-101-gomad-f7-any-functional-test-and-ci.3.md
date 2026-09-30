---
satisfies: [R5]
---
# fn-101-gomad-f7-any-functional-test-and-ci.3 Pin host-clock references statically on both platforms

## Description
Amended 2026-09-29 (user decision): no dynamic Linux audit. Add a toolchain-tier test that inventories every
standard-library reference to the host clock in the patched GOROOT for each qualified platform against a
reviewed, classified allowlist, plus an AST check that the clock entry points test `gomadEnabled` first.
Escapes it finds are recorded with findings, not fixed here.

## Acceptance
- `make test-toolchain` runs the inventory on darwin/arm64 and linux/amd64; a mutated count fails it
- the host-tools job (stock Go, no toolchain) skips it

## Done summary
Replaced the planned dynamic linux audit with a static host-clock inventory (R5 amended 2026-09-29, user decision). On linux/amd64 the runtime reads the clock through the vDSO, which seccomp and ptrace cannot observe, and the interception is platform-neutral Go that the darwin DTrace audit already exercises; what differs per platform is who reaches the host clock without passing through it.

`tools/gomad3/toolchain/clock_inventory_test.go` runs in `make test-toolchain`. It counts every standard-library reference to `nanotime1`, `walltime`, `time_now`, and the vDSO clock symbols in the patched GOROOT per qualified platform (go/build MatchFile per platform, comments stripped, `//go:` directives kept) against a reviewed allowlist classified as implementation, guarded, host-by-design, escape (with a finding), or unrelated, and an AST check requires `nanotime` and `time_runtimeNow` to return on `gomadEnabled` before their host call. A new, removed, or recounted reference fails the tier on either host; a mutated count fails it; stock Go without a toolchain skips it.

Escapes recorded, not fixed: `gcMarkTermination` stamps `MemStats.LastGC` with host wall time; the FIPS entropy source's `monoTime`; the execution tracer's clock snapshot; on linux, `syscall.Gettimeofday` behind the syscall pack gate. The dynamic seccomp audit is fn-105 D11.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 49d7cde43
- Tests: make test-toolchain (darwin/arm64), mutation: changed count fails the inventory, fork run 36668156879 core-linux step 5 (conformance tiers incl. test-toolchain): success
- PRs: