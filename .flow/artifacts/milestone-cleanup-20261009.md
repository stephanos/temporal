# Milestone cleanup

The owner requested removing completed specs from MILESTONES.md and prioritizing
near-complete specs on 2026-10-09. The root closed fn-113 and fn-114 through
`flowctl spec close` after checking their current task and completion-review states.

| Spec | Done tasks | Retained completion review | Reviewed head |
| --- | --- | --- | --- |
| fn-113 | 4/4 | SHIP | `1aa6fdbd13f77dbe7bd64c56656e7b96af2dcfaf` |
| fn-114 | 16/16 | SHIP | `fd59a054172e7e63eb75af67d35bad05373f73fa` |

The root checked the fn-113 review's completion type and empty findings and
unaddressed lists in `fn-113-gomad-reduce-version-pin-maintenance/conductor-completion-review-20261008/review-receipt.json`.
The fn-114 receipt embedded in `fn-114-gomad-correct-search-path-defects-and/source-completion-review-20261008.json`
decoded to 16,571 bytes with SHA-256
`b73daa076cd7a65cb154e52f13702a6c242f632df3400d21530dcef7b6b6e971`.
It also records completion SHIP and empty findings and unaddressed lists.
Both reviewed heads are ancestors of the cleanup's source base,
`4422e2c6c22bf293475a362e79baa9bbf36c60ab`.

Closure records the retained source acceptance. It adds no current-candidate
qualification, test execution or aggregate-lint claim. Native fn-128 and fn-149
remain deferred and unverified. Their milestone rows and all incomplete source
task rows remain present. The removed sections remain recoverable from Git and
the retained Flow specs, tasks and evidence.

MILESTONES.md prioritizes fn-112 task 10 next, with 15 of 16 tasks already Done.
Task dependencies and independent acceptance remain unchanged. Historical anchors
for the removed sections now link to their retained spec records.

The root's fresh `flowctl validate --all --json` reports 22 specs, 200 tasks,
zero errors and two existing coverage warnings. A read-only comparison against
`flowctl list --json` confirms six open specs and exactly 115 milestone task rows,
with no missing, duplicate, closed-spec or mismatched-status rows.
`git diff --check` exits 0. This orchestration-only change runs document and
lifecycle checks under milestone verification instruction 3.
