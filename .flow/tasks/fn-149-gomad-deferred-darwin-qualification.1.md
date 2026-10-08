---
satisfies: [R1]
---
# fn-149-gomad-deferred-darwin-qualification.1 Retain native Darwin baseline, runtime and clock evidence

## Description
Implements R1. Consume fn-105.31/.32, fn-109.13 and inherited native controls, fn-110.2-.5, fn-112.5 and fn-114.13.

**Size:** M
**Files:** .flow/artifacts/fn-149-gomad-deferred-darwin-qualification/runtime/**
**Touches:** [.flow/artifacts/fn-149-gomad-deferred-darwin-qualification/runtime/**]

### Approach
Freeze the integrated reviewed source. Build/launch the patched runtime and run inherited process/runtime/upstream/live-capability/overlay, time-wire and clock/draw controls, strict preservation and forward traced workloads, Darwin sandbox and DTrace controls. Preserve native baseline/candidate U3/U1 checks. Record failures without labeling qualification passed.

### Investigation targets
**Required:**
- tools/gomad3/Makefile:122-155
- .flow/tasks/fn-105-gomad-follow-ups-deferred-scope.31.md
- .flow/tasks/fn-110-gomad-minimize-the-runtime-patch.2.md
- .flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md

### Quick commands
Use the pinned donors' exact runtime/clock/control ledger; make -C tools/gomad3 test-runtime and make -C tools/gomad3 overlay-test retain their existing prerequisites.

### Revival and evidence
Keep deferred until the owner requests Darwin qualification and supplies native darwin/arm64 execution, a supported pinned toolchain/profile and a frozen source candidate. No PR, push or CI action is authorized by this handoff or its revival. Retain the original command/evidence ledgers at d28d67c40ce74dd8886cf11b36fe7d2ddaf23675 and the native transfer manifest. All tests retain -tags test_dep; use integration only for integration tests. Bind source closure, prepared targets, packs and shards to the same identity. Repairs return to their original source owner; refresh affected evidence after changes.

## Acceptance
- [ ] Native darwin/arm64 runtime builds and launches with source/toolchain/profile identities and all R1 inherited native controls pass with retained exact commands, outcomes and baseline/candidate identities. Missing or failing gates remain open.
- [ ] Independent review accepts the evidence and any separately admitted source changes; no static, developmental or historical result is substituted for native execution.

## Done summary
Blocked:
Deferred by the owner on 2026-10-07. Revival requires an explicit Darwin qualification request, native darwin/arm64 execution, the supported pinned toolchain/profile and a frozen reviewed candidate. Native evidence is unverified. No PR, push or CI action is authorized; scheduled/dispatched soak execution remains deferred pending separate authority. Downstream qualification also needs the reviewed shared D8-D10 candidate and actual consumer checkout. Portable/source requirements stay with the donor tasks; this handoff completes none of them.
## Evidence
- Commits:
- Tests:
- PRs:
