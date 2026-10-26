---
satisfies: [R6]
---
# fn-253-reusable-deadline-countdowns-for-models.10 Activity: always cooperative cancellation, remove asynchronous_cancel

Touches: [model/temporal/feature/activity/**, model/temporal/realization/src/live.rs, model/temporal/realization/tests/**, model/temporal/realization/macros/tests/renamed/src/main.rs, .flow/specs/fn-237-activity-timeouts-heartbeat-and.md]

## Description
Remove `World::asynchronous_cancel`, so the Model always does what the pinned request-cancel contract describes: immediate Canceled with no running attempt, pending while an attempt runs (spec R6). The atomic setting matched no live behaviour. It existed for fn-227's retained lifecycle comparison (fn-237 R3, `compatibility/checkpoint.rs`) and the atomic live binding.

**Owner decision first (blocking):** prepare a concrete compatibility-control proposal before asking the owner to choose projection or retirement. A pending Cancel plus acknowledgement resembles one old atomic cancel edge, but that sketch alone does not account for every pending edge, including the corrected Failed/TimedOut outcomes. Record the chosen proposal and answer in fn-237's Decision Context (`flowctl spec set-plan fn-237`) before implementation.
- *Projection proposal:* define a total state map and explicit action/edge or path accounting for the selected old and cooperative Worlds, including pending Complete, Fail, local/global expiry, heartbeat, rejection and acknowledgement behavior. State any excluded scope and its authority; do not claim exact graph equivalence from the two-edge cancel sketch. Specify and demonstrate a changed-old-lifecycle-edge control that the proposed comparison rejects. If no such defensible projection is established, report that limitation for the owner's decision.
- *Retire:* delete the checkpoint and its control tests, and amend fn-237 R3 to state why (the extension it guarded is complete).

**Cross-spec gates:** after fn-246.15 and fn-246.13 implementations and Flow closures are committed in `umpire`, and after .9 resolves and is integrated. Preserve their corrected timeout-retry and pending-cancellation failure/expiry outcomes; removing the flag does not restore cancel-wins policy. Update the exact World parameter contract and affected feature/protocol adapters. Coordinate with fn-237.4, which owns the live cancellation binding. Today `realization/src/live.rs:136,193,401` binds `false`. With the flag gone, live admission is cooperative. A cancel in `Started` that the binding cannot evidence (pending heartbeat and acknowledgement) must stay a field-named Unsupported refusal, never a silent pass. If fn-237.4 is in flight, land together with it, or after it. .8 validates the final combined source after this task is integrated.

**Size:** M
**Files:** activity `model.rs`, `tests.rs`, `policy_checks.rs`, `proof_checks.rs`, `temporal.rs`, `compatibility/checkpoint.rs`, `README.md`, `expected/`; `realization/src/live.rs`; World constructor call sites in `realization/tests/**` and `realization/macros/tests/renamed/src/main.rs` (29 positional `World::new` calls).
**Touches:** [model/temporal/feature/activity/**, model/temporal/realization/src/live.rs, model/temporal/realization/tests/**, model/temporal/realization/macros/tests/renamed/src/main.rs, .flow/specs/fn-237-activity-timeouts-heartbeat-and.md]

### Approach
- Before deleting the flag, capture native/interpreted/TLC state/edge/fact/outcome baselines for the cooperative twin of every selected final World at this task's admitted starting tip. Existing default snapshots may select atomic Worlds and cannot serve as cooperative evidence. Retain exact source revision, selected World values and normalization rules with the task evidence.
- `client`: the `Started` arm records pending unconditionally; `Scheduled` and every other open phase stay immediate. Drop `world.asynchronous_cancel` from `pending_is_cooperative` (it becomes `!cancel_requested || phase == Started`) and from the progress preconditions (or the helpers .1 introduced).
- `World::new` loses its last argument. Update every call site; `policy_checks.rs:86` and `:776` stop iterating `[false, true]`, and the `cooperative=` labels in its reports go.
- Keep every cooperative law and mutant from the corrected fn-246.13 baseline, including pending failure → Failed and pending local expiry → TimedOut. Add a mutant that reintroduces the atomic `Started` cancel; native and TLC must refute it.
- Apply the owner's checkpoint decision (projection or retirement) and update the activity README's World table.
- Record actual new World and state counts per selected World; do not infer a state-count ratio from removing one Boolean field.
- **Equivalence pin:** compare with the cooperative twin baselines captured before flag deletion at the integrated .9 outcome over the corrected fn-246.15/fn-246.13 baseline. Remove only the obsolete World field from comparison; the paired state/edge/fact/outcome sets must match. Regenerate default `expected/states.txt.snap`, `rows.txt.snap` and associated fixture goldens honestly when the prior defaults were atomic; their behavior may change to the captured cooperative twin, beyond spelling/header differences. Account for every selected World and apply only the owner's separately recorded checkpoint decision.

### Investigation targets
**Required:**
- `model/temporal/feature/activity/model.rs:166-179` — `client`
- `model/temporal/feature/activity/compatibility/checkpoint.rs:82-93` — the compatibility control
- `model/temporal/realization/src/live.rs:130-240,395-410` — live World binding and pending-cancel evidence
- `.flow/specs/fn-237-activity-timeouts-heartbeat-and.md` (R3, R4) — the control and binding contracts being amended
- `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/cancellation-contract-research.md` — authority for the single live behaviour

## Acceptance
- [ ] Owner decision on the compatibility control recorded in fn-237's Decision Context after a concrete total projection/edge-accounting and changed-edge-control proposal, or a documented projection limitation and retirement proposal, is available for review.
- [ ] `asynchronous_cancel` gone from the World, constructors, claims, tests, live binding and README.
- [ ] Atomic-`Started`-cancel mutant refuted by native and TLC; all corrected-baseline cooperative laws and mutants still pass, including Failed/TimedOut pending outcomes.
- [ ] Pre-deletion cooperative twin baselines and normalized comparisons cover every selected final World; regenerated default goldens account for prior atomic defaults, and actual new state counts are recorded.
- [ ] Live admission refuses unevidenced `Started` cancellation as field-named Unsupported; existing live gates pass.
- [ ] Fixtures and realization tests pass with `UMPIRE_BACKENDS=require`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
