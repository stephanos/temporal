---
satisfies: [R11]
---
# fn-105-gomad-follow-ups-deferred-scope.11 D11: dynamic linux/amd64 host-clock audit

## Description
Origin: fn-101.3 (F7 R5, before its 2026-09-29 amendment). Decision on 2026-09-30: depend on D21's host-clock investigation and keep implementation deferred until its findings establish audit need and feasible scope. Static inventories already cover both platforms, and Darwin DTrace exercises interception. The proposed fixture disables its runtime vDSO clock symbols and uses seccomp to detect clock_gettime/gettimeofday/time after activation. D21 must establish how this bounded check accounts for intentionally retained host-clock paths and the current patch restrictions before implementation.

## Acceptance
- D21 completes first and records the recommendation, exposure evidence, feasible audit scope, and next action in fn-105. Retain the formal dependency on fn-105 D21.
- If the findings require the audit, retain a bounded fixture, activation boundary, an unseeded positive control that fails for the expected forbidden clock read, and a seeded run that passes in core-linux. Retain commands, platform/toolchain identity, and CI evidence.
- Account for known host-by-design paths without widening generic syscall access or overriding collector/assembly patch policy. A policy change needs its own recorded decision.
- D21 completion does not claim an implemented audit. Keep a required audit open until verified; if it remains deferred, record why and what evidence would revive it.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
