---
satisfies: [R6]
---
# fn-253-reusable-deadline-countdowns-for-models.10 Activity: always cooperative cancellation, remove asynchronous_cancel

Touches: [model/temporal/feature/activity/**, model/temporal/realization/src/live.rs, model/temporal/realization/tests/**, model/temporal/realization/macros/tests/renamed/src/main.rs, .flow/specs/fn-237-activity-timeouts-heartbeat-and.md]

## Description
Remove `World::asynchronous_cancel`, so the Model always does what the pinned request-cancel contract describes: immediate Canceled with no running attempt, pending while an attempt runs (spec R6). The atomic setting matched no live behaviour. It existed for fn-227's retained lifecycle comparison (fn-237 R3, `compatibility/checkpoint.rs`) and the atomic live binding.

**Owner decision first (blocking):** re-express the compatibility control as a projection, or retire it. Ask before implementation and record the answer in fn-237's Decision Context (`flowctl spec set-plan fn-237`).
- *Projection* (recommended; keeps independent evidence): in the one-attempt World, `Client::Cancel` in `Started` followed by `Worker::RespondCanceled` maps to the old graph's single atomic cancel edge, and the pending intermediate state maps to its pre-cancel `Started` state. A changed old lifecycle edge must still fail the control.
- *Retire:* delete the checkpoint and its control tests, and amend fn-237 R3 to state why (the extension it guarded is complete).

**Cross-spec gates:** after fn-246.15's committed closure (same file; its plan preserves `checkpoint.rs`, which this task may change). Coordinate with fn-246.13 (it validates the World schema) and fn-237.4, which owns the live cancellation binding. Today `realization/src/live.rs:136,193,401` binds `false`. With the flag gone, live admission is cooperative. A cancel in `Started` that the binding cannot evidence (pending heartbeat and acknowledgement) must stay a field-named Unsupported refusal, never a silent pass. If fn-237.4 is in flight, land together with it, or after it.

**Size:** M
**Files:** activity `model.rs`, `tests.rs`, `policy_checks.rs`, `proof_checks.rs`, `temporal.rs`, `compatibility/checkpoint.rs`, `README.md`, `expected/`; `realization/src/live.rs`; World constructor call sites in `realization/tests/**` and `realization/macros/tests/renamed/src/main.rs` (29 positional `World::new` calls).
**Touches:** [model/temporal/feature/activity/**, model/temporal/realization/src/live.rs, model/temporal/realization/tests/**, model/temporal/realization/macros/tests/renamed/src/main.rs, .flow/specs/fn-237-activity-timeouts-heartbeat-and.md]

### Approach
- `client`: the `Started` arm records pending unconditionally; `Scheduled` and every other open phase stay immediate. Drop `world.asynchronous_cancel` from `pending_is_cooperative` (it becomes `!cancel_requested || phase == Started`) and from the progress preconditions (or the helpers .1 introduced).
- `World::new` loses its last argument. Update every call site; `policy_checks.rs:86` and `:776` stop iterating `[false, true]`, and the `cooperative=` labels in its reports go.
- Keep every cooperative law and mutant. Add a mutant that reintroduces the atomic `Started` cancel; native and TLC must refute it.
- Apply the owner's checkpoint decision (projection or retirement) and update the activity README's World table.
- Record the new World and state counts per selected World; the space should roughly halve.
- **Equivalence pin:** for each previously cooperative World (`asynchronous_cancel = true`), `expected/states.txt.snap` and `rows.txt.snap` match their old cooperative rows exactly. Only the atomic Worlds disappear.

### Investigation targets
**Required:**
- `model/temporal/feature/activity/model.rs:166-179` — `client`
- `model/temporal/feature/activity/compatibility/checkpoint.rs:82-93` — the compatibility control
- `model/temporal/realization/src/live.rs:130-240,395-410` — live World binding and pending-cancel evidence
- `.flow/specs/fn-237-activity-timeouts-heartbeat-and.md` (R3, R4) — the control and binding contracts being amended
- `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/cancellation-contract-research.md` — authority for the single live behaviour

### Acceptance
- [ ] Owner decision on the compatibility control recorded in fn-237's Decision Context before implementation.
- [ ] `asynchronous_cancel` gone from the World, constructors, claims, tests, live binding and README.
- [ ] Atomic-`Started`-cancel mutant refuted by native and TLC; all existing cooperative laws and mutants still pass.
- [ ] Equivalence pin holds for the former cooperative Worlds; new state counts recorded.
- [ ] Live admission refuses unevidenced `Started` cancellation as field-named Unsupported; existing live gates pass.
- [ ] Fixtures and realization tests pass with `UMPIRE_BACKENDS=require`.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
