---
satisfies: [R12]
---
# fn-105-gomad-follow-ups-deferred-scope.12 D12: record transfer of Linux replay correction

## Description

Administrative handoff to [fn-128.2](../tasks/fn-128-gomad-deferred-linux-qualification-and.2.md) authorized by the owner on 2026-10-04. Preserve the original Linux obligation at commit 10d884c6f9d97681d08aaf2636f5850407f1586a and in the receiving task. This record completes the transfer only; Linux execution stays deferred and no audit, fix or qualification is claimed.

## Acceptance

- [x] The receiving task and spec exist and retain all original Linux acceptance, restrictions, exact command/evidence requirements and revival prerequisites.
- [x] The transfer manifest maps this original requirement to its new owner; no source task/spec depends on the deferred Linux execution.
- [x] The source task's milestone and completion record explicitly identify this as an administrative transfer.

## Done summary
Recorded the owner-authorized administrative transfer of D12 to fn-128.2. The receiving task retains native first-divergence diagnosis, an actual causal fix and regression, traced F5/F6 seeds 11/17 repeated under host load with exact replay, and restoration of qualified expectations and strict dispatch/smoke CI only after verification passes.

The receiving ownership, source scope amendments and milestone/agent policy are committed in 76c0bdd667a2896b219756c01ced3356e49aa0e6. Linux execution remains deferred under fn-128.2. No divergence fix, successful Linux replay, CI restoration or Linux qualification pass is claimed. Missing transferred Linux evidence no longer holds the source specs open; their independent acceptance remains intact.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 76c0bdd667a2896b219756c01ced3356e49aa0e6
- Tests: /home/agent/.codex/scripts/flowctl validate --all --json, /home/agent/.codex/scripts/flowctl validate --spec fn-128 --coverage --json, node .flow/tmp/linux-scope-transfer/verify.mjs, git diff --check
- PRs: