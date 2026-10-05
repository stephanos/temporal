---
satisfies: [R4]
---
# fn-128-gomad-deferred-linux-qualification-and.4 Qualify Linux models, architecture, packs and affected workloads

## Description
Implements R4. Transfer source: fn-105.3/.4/.5; fn-109 open tasks under R7/R19; residual Linux fn-112.7 / R8; fn-113.3/.4 / R6; fn-114.14 / R12; Linux core/smoke/representative gates inherited by fn-110 and fn-112.

**Size:** M
**Files:** tools/gomad3/**, tools/gomad3sim/**, tools/gomad3integration/**, Linux pack requests and qualification manifests
**Touches:** [tools/gomad3/**, tools/gomad3sim/**, tools/gomad3integration/**, Linux pack requests and qualification manifests]

### Approach
Use the obligation manifest and original command ledgers to execute actual Linux model-OS comparisons, lifecycle/fault/network/filesystem/process/race suites, root gomad3sim, full Gomad tiers, default integration and affected consumers. Discover/review/generate --approve-review=<digest>/qualify invalidated Linux packs on the actual host and rebuild embedded packs. Run core/smoke/representative workloads and their required exact replays. Keep both-source-set static checks in their original specs. Record D12 replay status separately.

### Investigation targets
**Required:**
- `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md`
- `.flow/tasks/fn-112-gomad-determinism-assurance-and-test.7.md`
- `.flow/tasks/fn-113-gomad-reduce-version-pin-maintenance.4.md`

### Revival prerequisites
Owner requests qualification and supplies a native linux/amd64 host or CI and a pinned supported toolchain/source candidate. This task stays deferred until those prerequisites exist. Retain original policy and exact command/evidence requirements from the transfer manifest and source clauses at commit 10d884c6f9d97681d08aaf2636f5850407f1586a. Commit verified task progress separately; do not convert an unavailable gate into a pass.

## Acceptance
- [ ] Every mapped Linux model/architecture/maintenance/affected-suite obligation has source-bound native evidence, including residual fn-112.7. Packs match module/go.sum/files/profile and approved generation inputs. Required exact replay is recorded; D12 failures remain explicitly unresolved and informational until task 2, and task 7 refreshes final-candidate evidence.
- [ ] Retain exact commands, exit codes, source/build identities and an independent review for accepted changes; commit task progress before completing it.

## Done summary
Blocked:
Deferred by owner-authorized Linux scope transfer on 2026-10-04. No native linux/amd64 host or CI is available. Revival requires an owner request, native execution, pinned supported toolchain/profile and a frozen source candidate. This task owns R4; Linux evidence is incomplete and no pass or waiver is claimed.
## Evidence
- Commits:
- Tests:
- PRs:
