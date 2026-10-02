---
satisfies: [R27]
---
# fn-105-gomad-follow-ups-deferred-scope.32 D27: state, pin, and remedy host-clock reporting escapes

## Description
Origin: D21 investigation (`docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md`). Its proposals were left for a subsequent decision and had no owning task.

State the host-time escapes in the README contract. Pin the darwin `gettimeofday` path and `cputicks` in `toolchain/clock_inventory_test.go`, each with a fixture. Obtain and record the patch-policy owner's decision on overwriting the `LastGC` and `PauseEnd` stamps with stored virtual time from `runtime/proc.go`; implement the overwrite only with that approval.

## Acceptance
- The README contract names each host-time escape and what a target that reads it can observe.
- The clock inventory classifies the darwin `gettimeofday` path and `cputicks`; a fixture covers each.
- The stamp overwrite has a recorded patch-policy decision. If approved, it is implemented and the `LastGC` fixture qualifies; if declined, the limitation is stated in the contract.
- No prohibited collector or assembly file is edited without that approval.
- `make -C tools/gomad3 test-toolchain` passes on darwin/arm64; linux/amd64 status is recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
