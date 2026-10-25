---
satisfies: [R5]
---
# fn-246-realization-as-a-checked-refinement-a.15 Correct nonpending attempt-timeout retry and renew native/TLC proofs

## Description
Implement the owner-approved public-contract correction before independent protocol authoring. The approval is retained in `.flow/artifacts/fn-246-realization-as-a-checked-refinement-a/planning/timeout-retry-correction/owner-approval.json`; the original disagreement analysis and closed timeout-policy evidence remain immutable historical records.

**Size:** M (one transition owner, its existing law/control owners and generated semantic fixtures).
**Files:** activity Model, native/source controls, current semantic fixtures and README. The conductor owns joined guest promotion and full validation.
**Touches:** [model/temporal/feature/activity/model.rs, model/temporal/feature/activity/tests.rs, model/temporal/feature/activity/policy_checks.rs, model/temporal/feature/activity/proof_checks.rs, model/temporal/feature/activity/README.md, model/temporal/feature/activity/expected/Activity.cfg.snap, model/temporal/feature/activity/expected/Activity.tla.snap, model/temporal/feature/activity/expected/rows.txt.snap, model/temporal/feature/activity/expected/lift.txt.snap, model/temporal/feature/activity/expected/report.txt.snap, model/temporal/feature/activity/expected/tlc.txt.snap, model/temporal/feature/activity/expected/coverage.txt.snap, model/temporal/feature/activity/expected/expansion/closings.txt.snap, model/temporal/feature/activity/expected/states.txt.snap]

### Contract and authority

StartToClose and Heartbeat expiry end a selected attempt. In the existing admitted retry domain, a nonpending attempt below the maximum retries while the spanning ScheduleToClose budget permits. Approval changes only these two previously terminal cells. ScheduleToStart expiry, ScheduleToClose expiry, exhausted attempts, pending cancellation, worker-failure behavior and all other selected lifecycle semantics retain their meaning. The absent ScheduleToStart/effective-default question is independent task .13 work, not authority to substitute defaults or exclude Worlds here.

Primary authority is the pinned API module `go.temporal.io/api@v1.63.6-0.20260909222256-20151aa90480`: workflowservice/v1/request_response.pb.go:16585-16611 and service_grpc.pb.go:266-271, plus https://docs.temporal.io/encyclopedia/detecting-activity-failures#start-to-close-timeout . Server code corroborates behavior but supplies no Model policy. Retain the selected abstract retry scheduling already used for worker failure; do not invent clock/backoff inference or a new World domain.

### Approach

1. Retain an exact pre-correction counterexample in the already selected maximum_attempts=3, waiting=1, running=1, global=None, heartbeat=None World. Commit a meaningful red control for Tick retaining StartToClose fact and reaching Scheduled(attempt2), rather than terminal attempt1 or a relabeled Worker failure.
2. Reuse one admitted retry transition with retryable worker failure. An eligible local expiry increments once, schedules the next attempt, rearms waiting from the selected World, clears running/heartbeat budgets and preserves the already-decremented global countdown. Next accepted Poll arms its own running/heartbeat timers. Preserve selected fact and outcome identity.
3. Reanchor activity/model.rs:208,244,481,530,545. Update countdown/tick-priority laws with the exact approved retry branch; independently check its reset/clearing, global descent, attempt bounds and finality. Keep every existing obligation rather than dropping assertions that reject new behavior.
4. Reuse tests.rs:82 and policy_checks.rs:229,292,319,774 for native/source/TLC World controls, with proof_checks.rs:457 forged armed-zero controls. Cover maxima1/2/3, both local causes, below/at cap, pending/nonpending, absent/positive global budgets, local ties with StartToClose precedence and global ties with ScheduleToClose dominance. Include repeated timeout retries, retry then Complete, waiting rearm, current/stale Poll and terminal rejection. Add exact-edge/fact mutants for old terminalization, cap/pending retry, global reset, wrong rearm/attempt/priority/fact and timeout-as-worker-failure.
5. Preserve compatibility/checkpoint.rs's independent maximum_attempts=1 lifecycle (six states/eleven edges), its original artifacts and changed-old-edge negative control. Regenerate only affected current semantic fixtures. Counts may stay18/63 while destinations change, so aggregate/state-set equality alone is insufficient; exact transitions/facts, source-lifted agreement and discriminating invariant/mutant results must prove the correction.
6. Update the current activity README and retain a correction contract with parameter/World/type provenance, original-versus-corrected scope, actual tool versions/counts/bounds and complete command exits. Scope labels must distinguish original evidence from renewed results. Protocol Model authoring stays .13, refinement installation .4 and actual live proof fn-237.4.

### Verification

From model/: `mise exec -- cargo test -p umpire-temporal-tests --test fixtures`; repeat with `UMPIRE_BACKENDS=require` so real TLC executes comparable selected Worlds through the existing harness. Run `mise exec -- cargo test -p umpire-temporal --test realization named_guest_adapter_preserves_the_independent_report_and_ordered_rows`, scoped fmt/strict clippy and the retained global-bound/fairness/mutant controls. Use existing traversal/fixture/emitter owners, without another BFS or backend extension. The conductor verifies the joined source, refreshes/promotes its Temporal guest, renews freshness/current host checks and runs fresh `make model-validate` with TLC required before closure. Report any opt-in Apalache result separately and honestly; no old receipt proves new semantics.

### Admission and ownership

Task .13 starts only after this implementation, proof evidence and Flow closure are committed ancestors of umpire. No artificial dependency on active .11/.14 is added: source owners are disjoint, while the conductor serializes shared build resources/guest publication and rechecks their integration against the corrected Model. Do not edit .14's realization/gap snapshots, API/proto/program/history/catalog/manifest owners, .11's front/macros/registry owners, closed fn-237 records or the original compatibility artifacts.

## Acceptance
- [ ] Approved delta/public provenance is recorded before code; the former disagreement and closed original evidence stay intact. No additional Model-policy/default/World change is introduced.
- [ ] Eligible nonpending StartToClose/Heartbeat expiry emits its exact fact, schedules exactly the next attempt, rearms/clears the right timers and preserves decremented global budget; next Poll rearms attempt timers. At-cap/pending/global/ScheduleToStart behavior and priority/fairness remain selected as before.
- [ ] Native, interpreted/source-lifted and required real-TLC controls cover the stated World/cause/attempt/budget/priority matrix, repeated retries and successful continuation. Exact-edge/fact/reset/cap/pending/global mutants fail; counts or state-set equality alone earn no transition proof.
- [ ] Original max-one independent checkpoint and changed-edge control pass without rewriting originals. Current generated semantic outputs/README and renewed law obligations are reviewed, with no realization/gaps owner collision.
- [ ] Actual current World/bound/count/tool/replay/command evidence, scoped fmt/strict clippy, joined guest freshness/host checks and fresh TLC-required model validation pass before committed closure unblocks .13. Live/journal/gate and exact R5 full-state obligations retain their original owners.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
