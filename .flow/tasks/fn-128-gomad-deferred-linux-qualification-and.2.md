---
satisfies: [R2]
---
# fn-128-gomad-deferred-linux-qualification-and.2 Fix Linux replay divergence and restore strict CI

## Description
Implements R2. Transfer source: whole fn-105.12 / D12 / R12.

**Size:** M
**Files:** tools/gomad3/toolchain/**, tools/gomad3/runner/**, .github/workflows/**, Linux qualification manifests
**Touches:** [tools/gomad3/toolchain/**, tools/gomad3/runner/**, .github/workflows/**, Linux qualification manifests]

### Approach
Use the original D12 investigation and fn-112 diagnostic differ to localize the first divergent event under native host load. Test the one-shot suspendG syscall wait and uncontrolled ASLR/NumCPU candidates against evidence before choosing a fix. Retain a causal regression. Re-run affected traced F5/F6 cohorts with seeds 11/17, repetitions and host-load conditions before restoring strict Linux expectations and CI allowances.

### Investigation targets
**Required:**
- `.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.12.md`
- `tools/gomad3/toolchain/runtime/go1.27.1.patch`
- `.github/workflows/gomad3.yml`

### Revival prerequisites
Owner requests qualification and supplies a native linux/amd64 host or CI and a pinned supported toolchain/source candidate. This task stays deferred until those prerequisites exist. Retain original policy and exact command/evidence requirements from the transfer manifest and source clauses at commit 10d884c6f9d97681d08aaf2636f5850407f1586a. Commit verified task progress separately; do not convert an unavailable gate into a pass.

## Acceptance
- [ ] Every original D12 acceptance clause passes, including a reproduced cause, actual fix, regression and repeated native traced exact replay. Restore qualified expectations and remove both nondeterministic and replay_divergence allowances from dispatch-only and required smoke CI without relaxing other failures. Diagnosis or host absence alone cannot finish this task.
- [ ] Retain exact commands, exit codes, source/build identities and an independent review for accepted changes; commit task progress before completing it.

## Done summary
Blocked:
Deferred by owner-authorized Linux scope transfer on 2026-10-04. No native linux/amd64 host or CI is available. Revival requires an owner request, native execution, pinned supported toolchain/profile and a frozen source candidate. This task owns R2; Linux evidence is incomplete and no pass or waiver is claimed.
## Evidence
- Commits:
- Tests:
- PRs:
