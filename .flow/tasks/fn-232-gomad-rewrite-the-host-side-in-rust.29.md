---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.29 Diagnose and repair primary inspection transition capacity refusal

## Description
Touches to be determined from actual diagnosis; start with the primary Temporal inspection workload, Rust world/broker accounting and diagnostic evidence seams.

Investigate actual C8 temporal-inspection/primary-final failure: onebox explore exits3 instead of expected1, runner_failure transition-bytes limit1048576 used1048560 delta188. Both World owners can return this string and retained evidence has no failed execution prefix, so no source repair is justified yet. The full TestReplayInspectionWorkflowFailure/TestPrototypeWorkflow canary, its two one-minute timers/retries and required results, original1MiB bound and assertions remain unchanged. Never raise the bound or relabel capacity as expected.

Read /home/agent/wt/evidence/fn232-split-closure/evidence-preparation/cycle8-primary-inspection-capacity-diagnosis.{json,md}. First run a separately labelled bounded diagnostic reproduction with fresh outputs and explicit owner/op/byte/virtual-time/pending-request evidence, retaining failed World/network prefix and guest stdout where feasible. Diagnostics are not qualification or a new baseline. Then isolate root cause and implement only an evidence-backed minimal repair, or report the precise remaining blocker without inventing a fix. Parent .13 cannot pass until the original inspection behavior actually satisfies its unchanged requirements.

## Acceptance
- [ ] Reproduce the exact required failure in a fresh bounded diagnostic context with original1MiB/scenario/assertions and actual source/compiler/runtime/model/host bindings. Distinguish selected World owner, exact transition accounting/op counts, virtual time and guest progress; preserve original C8 raw attempt unchanged.
- [ ] Identify cause with concrete evidence distinguishing legitimate workload demand from duplicate/redundant transitions, model loop or accounting error. Do not invent a fix from the capacity string alone.
- [ ] If a repair is justified, retain RED/GREEN behavioral regression under the same bound and full canary semantics, complete required merge gates and independent green-tree SHIP review before integration. Any instrumentation is separately identified and qualification implications recorded.
- [ ] Parent full96/final145 and all existing runtime/replay/capacity/receiving obligations remain intact and pending until fresh actual proof. No budget increase, scenario shortcut, identity normalization or new unsupported disposition discharges this task.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
