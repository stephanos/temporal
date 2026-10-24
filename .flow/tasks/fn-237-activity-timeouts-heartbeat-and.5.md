---
satisfies: [R5]
---
# fn-237-activity-timeouts-heartbeat-and.5 After .4: DESIGN.md timeout and cancellation write-up

## Description
Split from fn-227.6 on 2026-10-21: fn-227.6 keeps the parameterized-worlds, inputs and retries write-up; this task records the timeout and cancellation additions.

Record R5's delivered timeout and cancellation surface and measured evidence after fn-237.4 and its transitive support tasks, extending the "Parameterized worlds and inputs" section fn-227.6 writes. No new behavior. MILESTONES remains root-owned.

**Size:** S. Extend existing DESIGN.md and README text to the completed facts, optional-value and timeout fixtures and the actual live cancellation proof, then run the existing final gates.
**Dependencies:** Consume fn-237-activity-timeouts-heartbeat-and.4. Cross-spec prerequisite fn-227.6 (the parameterized-worlds section this task extends), satisfied through this spec's dependency on fn-227.
**Files:** `model/DESIGN.md`, `model/README.md`; task-owned Flow verification artifacts through the normal lifecycle.
**Touches:** [model/DESIGN.md, model/README.md]

### Approach

1. Reanchor fn-237.1 to fn-237.4 receipts and actual implementation spellings. Keep DESIGN within its 1000-line limit, preserving unrelated architecture and the separate fn-232 failed certification result.
2. Describe the opt-in `Option<F>` fact result and bounded nominal optional values, including the checked scalar construction boundary, without claiming generic World/state validation.
3. Extend the activity World/state table with the timeouts, countdowns and `cancel_requested`. Record deadline priority and zero/reset laws, cooperative cancellation and its races, the exact-action fair-worker assumption and the schedule-to-close total-tick bound. Describe the final compatibility World's projection, including the Tick-to-deadline mapping, rather than claiming old activity byte equality. Record the timeout-to-tick table and the pinned cancellation binding.
4. Obtain native/TLC counts per actually selected fixture and activity World and the actual live cancellation evidence from retained outputs. Execute model-time on the final tree and retain its full output. State soft-target misses honestly; a zero command exit does not mean a performance target was met.

### Investigation targets

Required:

- `model/DESIGN.md` "Parameterized worlds and inputs" section as left by fn-227.6.
- `optional_facts` and `optional_ticks` fixtures and their completed task receipts for exact counts and public APIs.
- `model/temporal/feature/activity/{model.rs,tests.rs,temporal.rs}` and the compatibility harness for the final state/action/fact projection and live binding.
- The fn-237.4 live slice/gap/privacy evidence and its exact embedded guest hash.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-dsl --doc`. Run formatting, documentation link checks and DESIGN line-count verification using existing commands.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. No global model-update or stale counts. Preserve final timing output and prior evidence, including any unmet soft targets.

### Addendum (2026-10-24)

Also record the heartbeat and cancellation-acknowledgement program rows and the hold fault's role in the held attempt and schedule-to-start expiry, pointing at fn-243's DESIGN.md text rather than repeating it.
## Acceptance
- [ ] Consume committed fn-243.5 unheld cancellation admission/live/replay evidence and final R40 status so this final timeout/cancellation write-up reflects the actual supported scope.
- [ ] DESIGN documents facts, bounded optional values, the extended activity World/state table, deadline and cancellation laws, the fair-worker assumption and the final compatibility projection, each backed by completed fn-237 tasks.
- [ ] Actual native/TLC counts per selected World, the live cancellation proof and final model-time output are retained with all misses and refusal boundaries explicit.
- [ ] DESIGN remains within 1000 lines, existing documentation/formatting checks and full gates pass, and unrelated evidence/goldens and fn-232 status are preserved.


## Done summary
Blocked:
Waiting for committed fn-243-programmed-workers-the-model-issues.5.

The conductor may reset this block only after fn-243.5 is done and its implementation and Flow closure commits are ancestors of `umpire`. Final timeout/cancellation documentation consumes its unheld cancellation proof and final R40 status. Native fn-237.4 remains a separate prerequisite. The CLI cannot encode cross-spec task dependencies. This block must not block fn-237.1 through fn-237.4, which produce the independent semantics and live binding.
## Evidence
- Commits:
- Tests:
- PRs:
