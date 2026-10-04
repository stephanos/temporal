# R18 preservation disclosure, 2026-10-04

This supplement gives task 21 three independently owned differences against
fn-109's first-task baseline. The [original preservation audit](preservation-audit/report.md)
and [baseline reconstruction](baseline-reconstruction/reconstruction.md) retain
their historical evidence and bounds. The [baseline source manifest](baseline-reconstruction/source.sha256)
still has SHA-256 `d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845`.
The owner records explain the migrations; R18 availability remains unproved.

| Baseline difference | Independent owner and immutable evidence | Remaining preservation obligation |
| --- | --- | --- |
| Choice Trace wire/profile v2 to v3. Readers refuse v2 traces by profile and v2 headers, tapes and terminal frames by version. | [fn-114.11](../../../tasks/fn-114-gomad-correct-search-path-defects-and.11.md) requires the version bump and previous-version refusal. Retained commit [00633b2b558beb1c762decffe9d55b14631200c1](preservation-audit/commit-00633b2b558beb1c762decffe9d55b14631200c1.log) records it. [Task 11 gates](../../fn-114-gomad-correct-search-path-defects-and/task-11/gates.md#toolchain-identity) retain the changed runtime build key and refusals; its [golden refresh](../../fn-114-gomad-correct-search-path-defects-and/task-11/gates.md#golden-refresh) repins choice-dependent v3 fields while the plain golden stays unchanged. | Refreshed v3 goldens do not establish v2 recorded-format availability or first-baseline fixed-identity byte equivalence. |
| Choice Exploration controller v2 to v3. Resume refuses controller-v2 journals under the changed expansion rule. This identity is separate from the Choice Trace wire profile. | [fn-114.12](../../../tasks/fn-114-gomad-correct-search-path-defects-and.12.md) requires retaining the recorded rule or refusing resume. Retained commit [b666b41c09ed152417e44b6c9738e16a58b8b43a](preservation-audit/commit-b666b41c09ed152417e44b6c9738e16a58b8b43a.log) records the refusal and journal repinning. The [select-reduction record](../../fn-114-gomad-correct-search-path-defects-and/select-reduction/README.md) retains task 11's unchanged build key and equal trace bytes, with changed expansion counts. | Behavioral soundness and unchanged trace bytes in that comparison do not establish controller-v2 journal availability or journal identity equivalence. |
| The previously selected v041 fixture and its pack were retired after tests moved to v047 coverage. | [fn-113.3](../../../tasks/fn-113-gomad-reduce-version-pin-maintenance.3.md) owns retirement. Retained commit [7fd67d5aaeca5d658ca45d571b7671a67ddbfc31](preservation-audit/commit-7fd67d5aaeca5d658ca45d571b7671a67ddbfc31.log) says the fixture selected the variant. The [final source-bound summary](../../fn-113-gomad-reduce-version-pin-maintenance/task-3/final-completion-summary.md) records `testdata/v041/go.mod` selecting `golang.org/x/sys@v0.41.0`, its Make qualification entry and tests loading the pack. [Source binding](../../fn-113-gomad-reduce-version-pin-maintenance/task-3/source-binding.json) retains the removed pack/request/report/fixture hashes. The [selector audit](../../fn-113-gomad-reduce-version-pin-maintenance/task-3/selector-audit.json) describes absence after retirement. | Post-retirement zero selectors and active v047 coverage do not prove historical non-selection or preservation of exact v041 fixture, pack and workload availability. |

The original audit calls the v041 variant "unselected". Fn-113.3's retained
summary establishes that the pre-retirement fixture actually selected it; the
zero-selector audit describes the tree after retirement. This dated supplement
corrects that characterization while preserving the original report and commit
captures.

The [original fn-109 API contract](../../../specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md#api-contracts)
requires compatible recorded formats and equal canonical projections and payload
bytes for fixed supplied identities. [R18/R19](../../../specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md#acceptance-criteria)
retain capability, format, ordinary CLI/default and qualified-workload
availability, independent replay identities, error precedence and transaction
guarantees. The [boundaries](../../../specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md#boundaries)
exclude schema migration from this refactor. Fn-113.3 and fn-114.11/.12 keep
their own acceptance criteria. Their migration evidence supplies no waiver,
compatibility decision or restored format/workload availability.

R18 preservation and R19 qualification remain incomplete. Matched first-baseline
fixed-identity evidence and required native darwin/arm64 and linux/amd64 gates,
including full/runtime/process, integration, functional smoke and affected
qualification suites, remain task 21 obligations. Historical platform receipts
and current-only projection checks keep their source bounds. Disclosure alone
closes none of the native/full/formal gates.
