# Original-base lint inventory

The architecture digest repair removes two of the 215 findings reported against the original source baseline. The earlier 24-finding inventory used a later Git revision filter and understated the remaining original-base acceptance work.

The actual Make commands use `--new-from-rev`. [Original before receipt](digest-integrated-before-receipt.json) selects `951c5516e9e7b3066e7e069adda9565cfd68844c`; [original after receipt](digest-integrated-after-receipt.json) retains that selection. Both exit2 across 55 host packages before integrated errortype. Findings fall from215 to213, with errcheck160 to158 and exhaustive2, forbidigo9 and staticcheck44 unchanged. Complete raw reports remain in [before stdout](digest-integrated-before.stdout) and [after stdout](digest-integrated-after.stdout).

The [later-base before receipt](digest-integrated-later-before-receipt.json) and [after receipt](digest-integrated-later-after-receipt.json) select `d635e23f00d926a43b942f25a9d05bd0ccb72025`. They report24 to22, with errcheck20 to18, forbidigo1 and staticcheck3 unchanged. Those results describe only that filtered comparison. Historical task9/task40 packets retain their raw bytes and this narrower meaning. Their scoped target-context ST1005 findings remain open and are included in the original inventory.

## Ownership accounting

Root's research scout classified every original-base diagnostic from the retained215-finding report. These counts describe observed sites, without granting implementation authority.

| Group | Before | After digest repair | Scope limit |
| --- | ---: | ---: | --- |
| Gomad CLI unchecked output | 50 | 50 | Existing operation owners require exact output admission |
| Gomadtool unchecked diagnostics | 68 | 68 | Diagnostic, authoring, generator and build owners remain separate |
| Architecture source-digest writes | 2 | 0 | The current task19 two-statement admission |
| Test cleanup | 16 | 16 | Preserve lifetime/order and admit each fixture correction |
| Test child output | 7 | 7 | Preserve bytes, exit, release and EOF behavior |
| Production cleanup | 17 | 17 | Build15, inspection1 and conformance1; retain genuine fault obligations |
| Missing exhaustive test cases | 2 | 2 | Divergence/completion owners retain their compatibility requirements |
| Capitalized compatibility error messages | 42 | 42 | No casing migration or analyzer evasion admitted |
| Forbidden production panics | 8 | 8 | Existing two exact campaign-controller exceptions do not cover these sites |
| Intentional spin controls | 2 | 2 | No mechanism change admitted |
| Competing-build test sleep | 1 | 1 | Any correction requires observed synchronization proof |
| Total | 215 | 213 | Original-base integrated lint remains red |

The earlier24-filtered inventory omitted191 original-base findings, comprising errcheck140, exhaustive2, forbidigo8 and staticcheck41. The original265 to215 historical delta accounts for38 fn113 diagnostics, two fn113 lock-release checks, six task8 diagnostics, two task5 report checks and two exact campaign-controller panic exceptions. The original report confirms those fixes remain effective.

The265 endpoint remains locally retained at `.flow/tmp/next-gate-8486dcb98d/root-full-lint.log`. The scout compared complete diagnostic-message and source-line bytes, ignoring only location numbers, and found exactly50 removals with zero additions. That observed lint multiset supplies no broader source-correctness verdict. Supporting tracked attribution is retained in the following artifacts.

- fn113.1 has18 diagnostics in [final provenance](../../../fn-113-gomad-reduce-version-pin-maintenance/task-1/source-acceptance-20261008/final/provenance.json), under `lint.removed_task_owned_findings`.
- fn113.2 has12 diagnostics and two lock releases in [lint attribution](../../../fn-113-gomad-reduce-version-pin-maintenance/task-2/source-acceptance-20261008/lint-attribution.json), under `owned_resolved`.
- fn113.3 has eight diagnostics in [lint attribution](../../../fn-113-gomad-reduce-version-pin-maintenance/task-3/source-acceptance-20261008/lint-attribution.json).
- fn109.8 has six diagnostics in [exact-site attribution](../../task-8/source-acceptance-20261008/final-lint-attribution-exact-sites.json).
- fn109.5 has two restored reporting checks in [source proof](../../task-5/source-acceptance-20261008/source-proof-final.json) and its sibling `restored-unfiltered-lint.stdout`.
- fn109.28 has two exact campaign-controller exceptions in [handover](../../task-28/exact-invariant-exception-20261007/handover.md), with actual before/after log containers and receipts beside it.

## Next bounded candidate

The scout recommends the five reachable terminal generator diagnostics in `qualification_manifest.go` at22/26, `protocol.go` at20, `version.go` at20 and `boundary.go` at55. Root must admit the exact statements and additive tests under an appropriate R18/R19 owner before another worker edits them. Existing task46 stdout authority does not cover these stderr writes. Preserve the selected status, literal bytes, one write attempt, empty stdout and absence of publication mutations, using healthy and actual EBADF writer controls.

The older batch's `diagnostic.go` DiffDiagnostics error site is unreachable through the public inputs because both preceding trace reads already decode the same copied bytes. That site remains open. No invented fault seam, blanket suppression, native revival, PR, push or CI authority follows from this inventory.
