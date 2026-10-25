---
satisfies: [R5]
---
# fn-237-activity-timeouts-heartbeat-and.5 After .4: DESIGN.md timeout and cancellation write-up

## Description
Split from fn-227.6 on 2026-10-21: fn-227.6 keeps the parameterized-worlds, inputs and retries write-up; this task records the timeout and cancellation additions.

Record R5's delivered timeout and cancellation surface and measured evidence after fn-237.4 and its transitive support tasks, extending the "Parameterized worlds and inputs" section fn-227.6 writes. No new behavior. MILESTONES remains root-owned.

**Size:** S. Extend existing DESIGN.md and README text to the completed facts, optional-value and timeout fixtures and the actual live cancellation proof, then run the existing final gates.
**Dependencies:** Consume fn-237-activity-timeouts-heartbeat-and.4 and committed fn-243.5, retaining the existing block until its implementation and Flow closure are ancestors of umpire. Task .4 already consumes committed whole fn-246, including fn-246.14 and whole fn-247. Cross-spec prerequisite fn-227.6 (the parameterized-worlds section this task extends) is satisfied through this spec's dependency on fn-227. No new task/spec edge or ordering change is introduced.
**Files:** `model/DESIGN.md`, `model/README.md`; task-owned Flow verification artifacts through the normal lifecycle.
**Touches:** [model/DESIGN.md, model/README.md]

### Approach

1. Reanchor fn-237.1 to fn-237.4, fn-237.6 World-selected fairness, and committed fn-243.5 receipts and actual implementation spellings. Keep DESIGN within its 1000-line limit, preserving unrelated architecture and the separate fn-232 failed certification result.
2. Describe the opt-in `Option<F>` fact result and bounded nominal optional values, including the checked scalar construction boundary, without claiming generic World/state validation.
3. Extend the activity World/state table with the timeouts, countdowns and `cancel_requested`. Record deadline priority and zero/reset laws, cooperative cancellation and its races, the exact-action fair-worker assumption and the schedule-to-close total-tick bound. Describe the final compatibility World's projection, including the Tick-to-deadline mapping, rather than claiming old activity byte equality. Record the timeout-to-tick table and the pinned cancellation binding.
4. Obtain native/TLC counts per actually selected fixture and activity World and the actual live cancellation evidence from retained outputs. Execute make model-validate on the final tree and retain its full required-mode timing report, including its one inner-loop benchmark. State soft-target misses honestly; a zero command exit does not mean a performance target was met.

### Investigation targets

Required:

- `model/DESIGN.md` "Parameterized worlds and inputs" section as left by fn-227.6.
- `optional_facts` and `optional_ticks` fixtures and their completed task receipts for exact counts and public APIs.
- `world_fairness` fixture and fn-237.6 receipt for the bounded World-only source form, selected exact fair lists and actual native/TLC controls.
- Reanchor `model/temporal/feature/activity/{model.rs,tests.rs,realization.rs}`, fn-246's shared registry/lens/projection and fn-246.14's worker key/codec/raw-and-SDK owners after their committed closures; `temporal.rs` is an inherited pre-fn-246 anchor. Retain the compatibility harness for the final independent state/action/fact projection.
- The fn-237.4 live slice/gap/privacy evidence and its exact embedded guest hash.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-dsl --doc`. Run formatting, documentation link checks and DESIGN line-count verification using existing commands.

Full from root: `make model-validate`. No global model-update or stale counts. Preserve final timing output and prior evidence, including any unmet soft targets.

### Addendum (2026-10-24)

Also record the heartbeat and cancellation-acknowledgement program rows, the original max-one-heartbeat unknown/false/true regression subset and its provenance within fn-246's broader finite-history capability. Distinguish the prepoll hold for schedule-to-start from the response gate for the held-started pending/ACK and completion/failure-race proofs, pointing at the delivered fn-243 DESIGN text and fn-237.4 retained evidence rather than repeating their mechanics. Document actual raw/native-SDK RPC journal evidence, cancellation delivery and server-free replay from retained Runs; mocks or raw surrogates never earn SDK/live credit. Describe fn-246's consumed current whole-prefix witness contract and immutable earlier reports without adding legacy delta or old-format readers.

### Boundary-policy documentation proposal for fresh review

Document the revised independent R2 boundary matrix and its public-evidence/design-inference provenance. Separate complete offline native/TLC coverage from actually admitted live evidence, naming any predeclared exact-expiry-cause gap. Retain every original mandatory live proof and fn-243.5 final R40 obligation; no coverage-gap note or refusal substitutes for them.


Selected policy and provenance: `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/cancellation-boundary-contract.md`; exact live evidence limits are in `cancellation-boundary-evidence-shape.md`. Fresh plan review must pass before implementation.
### Verification adoption

Use `make model-validate` for remaining final validation. It runs one complete gate with TLC required and one inner-loop benchmark. Apalache remains independently opt-in; none of these prior plans required it. Keep all standalone focused controls, actual raw/SDK live slice/gaps/replay and SUT proofs, and preserved original semantic counterexample evidence. Label new gate timing as required-mode and historical default-mode timing as historical; retain actual command provenance. Current focused commands remain available. No TLC batching, caching or reduced case coverage is introduced.

## Acceptance
- [ ] Consume committed fn-243.5 unheld cancellation admission/live/replay evidence and final R40 status so this final timeout/cancellation write-up reflects the actual supported scope.
- [ ] DESIGN documents facts, bounded optional values, the extended activity World/state table, deadline and cancellation laws, the fair-worker assumption and the final compatibility projection, each backed by completed fn-237 tasks.
- [ ] Actual native/TLC counts per selected World, the live cancellation proof and final model-time output are retained with all misses and refusal boundaries explicit.
- [ ] DESIGN remains within 1000 lines, existing documentation/formatting checks and full gates pass, and unrelated semantic evidence and fn-232 status are preserved; affected goldens may regenerate.


## Done summary
Blocked:
Waiting for committed fn-243-programmed-workers-the-model-issues.5.

The conductor may reset this block only after fn-243.5 is done and its implementation and Flow closure commits are ancestors of `umpire`. Final timeout/cancellation documentation consumes its unheld cancellation proof and final R40 status. Native fn-237.4 remains a separate prerequisite. The CLI cannot encode cross-spec task dependencies. This block must not block fn-237.1 through fn-237.4, which produce the independent semantics and live binding.
## Evidence
- Commits:
- Tests:
- PRs:
