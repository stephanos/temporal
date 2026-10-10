---
satisfies: [R2, R3, R7]
---
# fn-152-gomad-runner-storage-on-one-append-only.4 Commit seed outcomes and controller progress atomically during run and resume

## Description
Integrate R3's durable seed scheduler and resume behavior using the campaign lifecycle and admission/receipt owners.

**Size:** M
**Files:** Core files (4-5): runner/runner_local.go, campaign.go, resume.go, internal/campaign/controller.go and a focused scripted resume matrix. Wider touches: recovery.go caller and existing completion/retention tests only where the new transaction seam requires them.
**Touches:** [tools/gomad3/runner/runner_local.go, tools/gomad3/runner/campaign.go, tools/gomad3/runner/resume.go, tools/gomad3/runner/recovery.go, tools/gomad3/runner/internal/campaign/controller*.go, tools/gomad3/runner/*completion*_test.go, tools/gomad3/runner/*retention*_test.go, tools/gomad3/runner/*resume*_test.go]

### Approach

Consume the admission/receipt owner's checked transitions from Runner's ordered completion flow rather than duplicate their state machine. Launch only durable reservations. Publish bounded immutable actual completion evidence before acknowledging its terminal receipt into the ordering buffer; preserve existing result normalization, error callbacks and error-precedence rules without serializing arbitrary Go errors. A receipt alone cannot increment outcome counters or release an active slot. Ordinary outcome commits atomically bind ordinal, classification, retention/novelty, controller progress and the normal permitted refill reservation. Publish user-visible committed progress only after success.

When the next ordered receipt stops policy, freeze admissions, cancel active attempts only for first-failure, and consume all actual admitted completions into one atomic ordered stop group. Budget exhaustion lets active attempts finish. Reuse buffered real success/failure evidence; never synthesize cancellation for an outstanding ordinal. Resume restores the exact durable frontier and receipts before selection/launch, processes a pending stopping receipt before refill, retries only receipt-less admitted attempts, and restores committed runner_cancelled ordinals and Cancelled counts. A committed stop launches nothing. Uncertain append stops scheduling and poisons the writer; reopen decides complete-frame survival. Host cancellation, aggregate deadline, watchdog and unclassified-attempt precedence remain unchanged.

The scripted matrix compares uninterrupted and interrupted ordinal/classification/counter/novelty/evidence sets. Enumerate admission sync and pre-launch failures, out-of-order terminal receipts, ordinary outcome/refill commits, first cancel request, each drain receipt, stop-group append/sync and publication. Include buffered real success/failure before a lower stopping failure and incremental replacement admissions before a later stop. No separate per-member logical drain commit exists; the complete group replays atomically.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/campaign.go:5`
- `tools/gomad3/runner/resume.go:158`
- `tools/gomad3/runner/internal/campaign/controller.go:88`
- `tools/gomad3/runner/runner_local.go:318`
- `tools/gomad3/runner/seed_completion_characterization_test.go:74`
- `tools/gomad3/runner/retention_characterization_test.go:612`
- `tools/gomad3/runner/resume.go:212`
- `tools/gomad3/runner/runner_local.go:448`

### Verification

Focused controls: go -C tools/gomad3 test -tags test_dep -count=1 -run 'TestSeedCompletion|TestRetentionFailureLeavesNoveltyAndCountersAtTheCommittedState|TestCancellationIsAHostFailure' ./runner plus the new resume matrix's retained exact selector.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R3: at every admission/receipt/outcome-refill/stop-group append/sync failure point, run plus resume has the uninterrupted logical ordinal/outcome set and policy/classification/novelty state. Acknowledged outcomes and durable terminal receipts never re-execute; only receipt-less uncommitted physical attempts may repeat.
- [ ] R3: first-failure, distinct-failure budget, all-failures, parallel active attempts, cancellation, aggregate deadlines and watchdog classifications are explicitly covered; uncommitted physical attempts may repeat without becoming duplicate committed outcomes.
- [ ] R3: failed/uncertain append cannot advance persisted or user-visible committed policy progress or allow subsequent appends; reopen decides whether a complete uncertain frame survived.
- [ ] R2/R3: frozen guided selection, prepared identity, retained byte/count limits and repeated resume behavior survive, including an already-satisfied stopping policy.
- [ ] R3: first-failure parallel3 retains one real failure plus two real cancellations; budget stop never policy-cancels active work. Buffered real successes/failures retain their classifications through pre-group crashes; normal completion preserves incremental refill, not fixed batches.
- [ ] R3/R7: complete stop-group replay restores every cancellation ordinal and Cancelled counter once, launches no new work, and retains durable pending payloads until adopted or retired. Missing/corrupt receipt payloads fail before cleanup, not by silently retrying a durably observed completion.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
