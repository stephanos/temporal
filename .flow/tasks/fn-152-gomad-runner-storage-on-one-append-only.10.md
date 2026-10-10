---
satisfies: [R2, R3, R7]
---
# fn-152-gomad-runner-storage-on-one-append-only.10 Persist bounded admission and terminal receipts with atomic policy stop/drain

## Description
Implement the bounded campaign admission, terminal-receipt and atomic stop/drain owner for R2/R3/R7. Runner caller integration belongs to the following task.

**Size:** M
**Files:** Core files (4-5): new campaign admission/receipt transition owner, typed record/replay support, pure policy-decision seam and focused state/fault tests. Wider touches: controller.go only for the existing pure stop/refill decisions; campaign live-reference projection consumed from the lifecycle owner. Do not include Runner scheduling/capture/resume wiring in this task.
**Touches:** [tools/gomad3/runner/internal/campaign/**]

### Approach

Use the campaign log as the sole authority for the frozen cursor, admitted outstanding ordinals and terminal-result receipts. Initial reservations commit before launch. Ordinary ordered outcome transactions include the exact permitted refill reservation, preserving current incremental completion/refill instead of fixed batches. Reserved, running and receipted entries all occupy slots until logically completed; their total cannot exceed configured Parallel. Validate ordinal/seed/shard identity and reject duplicate or conflicting receipts.

A terminal receipt binds bounded immutable actual completion observation/evidence and durably published owned payload references before ordered buffering. It does not apply retention, novelty, outcome counters or arbitrary error callbacks out of order. Runner supplies normalized observations under existing error precedence; do not define a generic Go-error serialization. Receipt payload references remain live through replay/recovery until an outcome transaction retires or adopts them. Metadata exists only in the log, never payload side-file state.

Derive stop intent from the next ordered durable receipt, committed policy and current admitted frontier; no separate stop-intent record is needed. A first-failure trigger requests cancellation, while a distinct-budget trigger drains normally. Freeze all new reservations. After every admitted group member has an actual terminal receipt, commit the trigger and all ordered drain outcomes, controller/counters/retention/novelty and receipt retirement in one typed transaction. Validate the whole group before any replay application. Never relabel an actual buffered success/failure or infer a cancellation merely from outstanding membership.

Replay restores the exact frontier and durable receipts before launch/refill. Pending stop selection precedes any normal reservation. Reuse receipts without rerun; retry only receipt-less admitted attempts under the existing first-policy cancellation or budget-policy execution action. A committed stop has no outstanding members and admits no new work. A torn group applies none and leaves prior reservations/receipts; a complete uncertain frame applies all once. Any uncertain append poisons the writer until reopen.

Checked frame and pending-payload bounds derive from existing per-attempt component envelopes and configured parallelism, with no smaller aggregate/parallel limit and no hidden charge against final retained-success capacity. Streaming replay retains only the bounded live frontier plus existing campaign state; historical disk growth follows the parent contract. All final and pending live payloads validate before owned-root cleanup.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/campaign/controller.go:88`
- `tools/gomad3/runner/internal/campaign/controller.go:112`
- `tools/gomad3/runner/internal/campaign/controller.go:152`
- `tools/gomad3/runner/internal/campaign/controller_test.go:55`
- `tools/gomad3/runner/campaign.go:5`
- `tools/gomad3/runner/runner_local.go:448`
- `tools/gomad3/runner/seed_completion_characterization_test.go:113`
- `tools/gomad3/runner/resume.go:212`
- `tools/gomad3/runner/runner.go:712`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/campaign. Retain exact selectors for new deterministic admission/receipt/stop-group state tests and fault controls. Runner integration owns the executable uninterrupted/resumed matrix.

Follow the parent Delivery and verification contract. Use deterministic transition/fault seams, not timing races. If the surviving owner exceeds this cohesive M scope, stop for conductor splitting before implementation.

## Acceptance
- [ ] R2/R3: all admission and receipt metadata lives in campaign log records; replay restores exact ordinal/seed/shard reservations before launch. The admitted set, counting pending receipts, stays within configured Parallel; ordinary outcome/refill is atomic and preserves incremental admission.
- [ ] R3: terminal receipts retain actual completion observations without out-of-order policy/retention/counter effects. Durable receipts never re-execute; conflicting receipts are corruption. Receipt-less unacknowledged physical attempts may repeat without fabricated logical outcomes.
- [ ] R3: first-policy cancel and budget-policy natural drain retain their distinction. Stop intent derives before refill, and trigger plus every actual admitted drain outcome commit and replay as one validated group or none; committed stopped replay has no outstanding member or new admission.
- [ ] R3: deterministic tests interrupt before/after reservation, launch, receipt, ordinary outcome/refill and stop-group append/sync, including complete uncertain frames, buffered success/distinct or duplicate failure, cancellation and repeated replay. Failed append cannot advance acknowledged policy or allow another append.
- [ ] R7: durable pending evidence is live until atomic retirement/adoption; missing/corrupt evidence prevents cleanup and is never silently retried. Checked component/parallel bounds preserve existing capacities and final-retention accounting; no second state authority or new history cap exists.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
