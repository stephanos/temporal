---
satisfies: [R5]
---
# fn-237-activity-timeouts-heartbeat-and.5 After .4: DESIGN.md timeout and cancellation write-up

## Description
Split from fn-227.6 on 2026-10-21: fn-227.6 keeps the parameterized-worlds, inputs and retries write-up; this task records the timeout and cancellation additions.

Record R5's delivered timeout and cancellation surface and measured evidence after fn-237.4 and its transitive support tasks, extending the "Parameterized worlds and inputs" section fn-227.6 writes. No new behavior. MILESTONES remains root-owned.

**Size:** S. Extend existing DESIGN.md and README text to the completed facts, optional-value and timeout fixtures and the actual live cancellation proof, then run the existing final gates.
**Dependencies:** Consume fn-237-activity-timeouts-heartbeat-and.4 and committed fn-243.5, retaining the existing block until its implementation and Flow closure are ancestors of umpire. Task .4 already consumes committed whole fn-246, including fn-246.15’s retry correction, fn-246.13’s approved cancellation correction and renewed proofs, fn-246.14 and whole fn-247. Cross-spec prerequisite fn-227.6 (the parameterized-worlds section this task extends) is satisfied through this spec's dependency on fn-227. No new task/spec edge or ordering change is introduced.
**Files:** `model/DESIGN.md`, `model/README.md`; task-owned Flow verification artifacts through the normal lifecycle.
**Touches:** [model/DESIGN.md, model/README.md]

### Corrected policy and evidence provenance

Consume fn-246.15’s committed nonpending attempt-timeout retry correction and fn-246.13’s committed API-aligned cancellation correction, with their renewed independent native/TLC evidence. Approval for the latter is retained at `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task13/planning/approved-api-correction.json`. Pending worker failure closes Failed and pending StartToClose/Heartbeat expiry closes TimedOut, without retry; scheduled cancellation and explicit current-attempt worker acknowledgement remain Canceled. No further Model-policy change is authorized. Preserve fn-246.15’s retry behavior, the original atomic lifecycle checkpoint and unrelated independent policies.

Raw Start timeout fields and effective modeled budgets are separate contracts. Consume fn-246.13’s pinned per-field raw-request defaults and validation at an explicitly named admission boundary, then apply exact timeout-to-tick quantization and checked scalar construction. Effective parameters preserve None versus present-zero without hidden fallback, clamping or approximation. All-absent and other checker Worlds that cannot represent accepted raw requests remain explicitly labelled abstract scopes; preserve their exploration and the feature Model’s absence policy.

The earlier `cancellation-boundary-contract.md`, `cancellation-contract-research.md` and `cancellation-boundary-evidence-shape.md` remain historical provenance and evidence inventories. Their superseded pending-failure/expiry Canceled cells do not govern remaining work. Recheck their field inventory and cause-evidence limits against the pinned API and corrected contract; preserve the original artifacts and completed task records unchanged.

### Approach

1. Reanchor fn-237.1 to fn-237.4, fn-237.6 World-selected fairness, and committed fn-243.5 receipts and actual implementation spellings. Keep DESIGN within its 1000-line limit, preserving unrelated architecture and the separate fn-232 failed certification result.
2. Describe the opt-in `Option<F>` fact result and bounded nominal optional values, including the checked scalar construction boundary, without claiming generic World/state validation.
3. Extend the activity World/state table with the timeouts, countdowns and `cancel_requested`. Record corrected attempt-timeout retry, deadline priority and zero/reset laws, API-aligned cooperative cancellation and its races, the exact-action fair-worker assumption and the schedule-to-close total-tick bound. Describe the final compatibility World's projection, including the Tick-to-deadline mapping, rather than claiming old activity byte equality. Record the raw-request per-field defaults/validation, separate effective parameters, exact timeout-to-tick table, labelled abstract checker Worlds and pinned cancellation binding. Preserve the feature Model’s absence policy and None/present-zero distinction.
4. Obtain native/TLC counts per actually selected fixture and corrected activity World and the actual live cancellation evidence from retained outputs. Separate original fn-237.3 results from renewed fn-246.15/.13 proofs; the original evidence does not prove the corrected tree. Execute make model-validate on the final tree and retain its full required-mode timing report, including its one inner-loop benchmark. State soft-target misses honestly; a zero command exit does not mean a performance target was met.

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

Document the corrected independent R2 boundary matrix consumed from fn-246.15/.13 and its pinned API/approval provenance. Label the earlier Canceled pending-failure/expiry matrix as historical and preserve its artifacts unchanged. No further Model-policy change belongs to this write-up. Separate complete offline native/TLC coverage from actually admitted live evidence, naming any predeclared exact-expiry-cause gap. Terminal phase alone cannot identify a deadline; require typed current-run/attempt correlation and distinguish stale last_failure history. Retain every original mandatory live proof and fn-243.5 final R40 obligation; no coverage-gap note or refusal substitutes for them.


Current policy authority is fn-246.13’s explicit approval at `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/task13/planning/approved-api-correction.json` and its committed corrected contract/proofs, preserving fn-246.15. The earlier cancellation-boundary contract and evidence-shape inventory remain historical; revalidate their surviving cause-evidence limits against the corrected contract. Fresh plan review must pass before implementation. The compatibility-control removal proposal remains unapproved; preserve the current control and its historical evidence.
### Verification adoption

Use `make model-validate` for remaining final validation. It runs one complete gate with TLC required and one inner-loop benchmark. Apalache remains independently opt-in; none of these prior plans required it. Keep all standalone focused controls, actual raw/SDK live slice/gaps/replay and SUT proofs, and preserved original semantic counterexample evidence. Label new gate timing as required-mode and historical default-mode timing as historical; retain actual command provenance. Current focused commands remain available. No TLC batching, caching or reduced case coverage is introduced.

## Acceptance
- [ ] Consume committed fn-243.5 unheld cancellation admission/live/replay evidence and final R40 status so this final timeout/cancellation write-up reflects the actual supported scope.
- [ ] DESIGN documents facts, bounded optional values, the extended activity World/state table, deadline and cancellation laws, the fair-worker assumption and the final compatibility projection, backed by completed fn-237 tasks and consumed committed fn-246.15/.13 correction proofs. Historical and renewed results are labelled separately; raw defaults, effective budgets and abstract checker scopes are explicit.
- [ ] Actual native/TLC counts per selected corrected World, the live cancellation proof and final model-validate gate/benchmark output are retained with all misses and refusal boundaries explicit.
- [ ] DESIGN remains within 1000 lines, existing documentation/formatting checks and full gates pass, and unrelated semantic evidence and fn-232 status are preserved; affected goldens may regenerate.


## Done summary
Blocked:
Waiting for committed fn-243-programmed-workers-the-model-issues.5.

The conductor may reset this block only after fn-243.5 is done and its implementation and Flow closure commits are ancestors of `umpire`. Final timeout/cancellation documentation consumes its unheld cancellation proof and final R40 status. Native fn-237.4 remains a separate prerequisite. The CLI cannot encode cross-spec task dependencies. This block must not block fn-237.1 through fn-237.4, which produce the independent semantics and live binding.
## Evidence
- Commits:
- Tests:
- PRs:
