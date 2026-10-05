---
satisfies: [R6]
---
# fn-128-gomad-deferred-linux-qualification-and.6 Qualify downstream Linux analyses, packs and exact replay

## Description
Implements R6. Transfer source: Linux portions of fn-105.8/.9/.10 / R8/R9/R10; deferred fn-107 consuming qualifications.

**Size:** M
**Files:** downstream localcell/gomad adapters, pack inputs, qualification manifest/driver and documentation
**Touches:** [downstream localcell/gomad adapters, pack inputs, qualification manifest/driver and documentation]

### Approach
Require the real downstream checkout and shared D8 implementation. Consume the final target source from the fn-107 implementation checkpoint and reconcile current consumer/source reviews. Exercise closure and linked support/refusal/drift tests with exact reachable adapters. Discover/review/generate consumer-owned Linux packs on the actual host, retain negative identity/platform tests and execute seeds 11/17 twice plus exact replay of every retained success. Write Linux commands/support guidance only from qualified evidence.

### Investigation targets
**Required:**
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.8.md`
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.9.md`
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.10.md`

### Revival prerequisites
Owner requests qualification and supplies a native linux/amd64 host or CI and a pinned supported toolchain/source candidate. This task stays deferred until those prerequisites exist. Retain original policy and exact command/evidence requirements from the transfer manifest and source clauses at commit 10d884c6f9d97681d08aaf2636f5850407f1586a. Commit verified task progress separately; do not convert an unavailable gate into a pass.

## Acceptance
- [ ] All original Linux D8/D9/D10 obligations pass with actual source reviews, analyses, exact pack/source/tool identities, repeated execution and retained exact replay. Absent checkout, missing shared implementation, classified failures or mismatched bindings keep this task open. No qualified downstream claim follows from the earlier fn-107 implementation-only closure.
- [ ] Retain exact commands, exit codes, source/build identities and an independent review for accepted changes; commit task progress before completing it.

## Done summary
Blocked:
Deferred by owner-authorized Linux scope transfer on 2026-10-04. No native linux/amd64 host or CI is available. Revival requires an owner request, native execution, pinned supported toolchain/profile and a frozen source candidate. This task owns R6; Linux evidence is incomplete and no pass or waiver is claimed. Additional prerequisite: actual downstream checkout and shared fn-105.8 candidate; the source Darwin D8-D9-D10 chain remains intact.
## Evidence
- Commits:
- Tests:
- PRs:
