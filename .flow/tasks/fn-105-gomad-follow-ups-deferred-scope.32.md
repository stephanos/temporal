---
satisfies: [R27]
---
# fn-105-gomad-follow-ups-deferred-scope.32 D27: state, pin, and remedy host-clock reporting escapes

## Description

Owner amendment (2026-10-04): this task transfers every remaining native Linux execution, Linux pack/report/replay and Linux-specific qualification-documentation requirement to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). Native execution/full/affected gates still owned here apply to Darwin. Missing transferred Linux proof cannot block this task. Static coverage of both supported source sets, shared implementation, preservation, review and other non-Linux requirements remain unchanged. Retained scope: Contract/inventory/remedies, collector/assembly policy and native Darwin test-toolchain proof. See the [transfer manifest](../artifacts/linux-scope-transfer-2026-10-04.md). Historical progress below retains its original meaning and is not current-candidate proof.

Origin: D21 investigation (`docs/research/gomad/GOMAD_HOST_CLOCK_ESCAPES.md`). Its proposals were left for a subsequent decision and had no owning task.

State the host-time escapes in the README contract. Pin the darwin `gettimeofday` path and `cputicks` in `toolchain/clock_inventory_test.go`, each with a fixture. Obtain and record the patch-policy owner's decision on overwriting the `LastGC` and `PauseEnd` stamps with stored virtual time from `runtime/proc.go`; implement the overwrite only with that approval.

## Acceptance

Current native-execution acceptance is Darwin-only here. The corresponding Linux clauses and any older missing-Linux completion rule are transferred to [fn-128.1](../tasks/fn-128-gomad-deferred-linux-qualification-and.1.md), [fn-128.4](../tasks/fn-128-gomad-deferred-linux-qualification-and.4.md), [fn-128.7](../tasks/fn-128-gomad-deferred-linux-qualification-and.7.md). All other acceptance below remains in force.

- The README contract names each host-time escape and what a target that reads it can observe.
- The clock inventory classifies the darwin `gettimeofday` path and `cputicks`; a fixture covers each.
- The stamp overwrite has a recorded patch-policy decision. If approved, it is implemented and the `LastGC` fixture qualifies; if declined, the limitation is stated in the contract.
- No prohibited collector or assembly file is edited without that approval.
- `make -C tools/gomad3 test-toolchain` passes on darwin/arm64; linux/amd64 status is recorded.


## Done summary
Blocked:
D27 implementation is merged at `bfb2bdb8ef136d3eb38cbd539735661d5d7c9af5`, reachable from the current `gomad` HEAD. The README contract names host-clock reporting escapes and the declined collector-stamp overwrite; the static inventory and fixtures pin Darwin `gettimeofday` and `cputicks`. No prohibited collector or assembly source was changed. Independent correctness review returned SHIP for the implementation, not native qualification.

Fresh development verification: `GOTOOLCHAIN=go1.27.1 GOWORK=off go -C tools/gomad3 test -count=1 -tags test_dep ./toolchain -run TestHostClockInventoryPinsPlatformSpecificEscapes` exits 0. Earlier implementation commands, review, and source evidence remain in `/tmp/flow-next-fn105-32/summary.md` and `/tmp/flow-next-fn105-32/evidence.json`.

R27 remains incomplete: `make -C tools/gomad3 test-toolchain` must pass on native darwin/arm64 and linux/amd64. This host reports Linux aarch64 (linux/arm64), which the qualification contract does not support; the available local Docker builder is also arm64, and there are no usable GitHub Actions credentials. Cross-compiled or emulated checks cannot satisfy native qualification. Resume with a qualified host or CI; do not widen platform policy or mark this task done from synthetic inventory fixtures.
## Evidence
- Commits:
- Tests:
- PRs:

## Linux ownership blocker (2026-10-04)

Linux ownership amendment (2026-10-04): all native Linux execution obligations moved to fn-128. Missing transferred Linux evidence no longer blocks this task. Source-owned acceptance remains incomplete for Contract/inventory/remedies, collector/assembly policy and native Darwin test-toolchain proof. Keep the task blocked for those independent requirements, with current-source evidence required by its original acceptance. See the scoped Description/Acceptance and .flow/artifacts/linux-scope-transfer-2026-10-04.md.
