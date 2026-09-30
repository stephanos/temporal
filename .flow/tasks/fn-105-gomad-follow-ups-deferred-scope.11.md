---
satisfies: [R1, R2]
---
# fn-105-gomad-follow-ups-deferred-scope.11 D11: dynamic linux/amd64 host-clock audit

## Description
Origin: fn-101.3 (F7 R5, before its 2026-09-29 amendment). Deferred: the static inventory pins every host-clock reference on both platforms and darwin DTrace exercises the platform-neutral interception. Revive if a linux-only clock escape is ever observed. Design: the fixture zeroes the runtime vDSO clock symbols, a seccomp filter kills the process on clock_gettime/gettimeofday/time after the activation marker; unseeded positive control must die, seeded run must pass.

## Acceptance
Revival trigger recorded; the audit runs in core-linux with a positive control.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
