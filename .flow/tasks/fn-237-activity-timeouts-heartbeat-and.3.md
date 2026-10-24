---
satisfies: [R2, R3]
---
# fn-237-activity-timeouts-heartbeat-and.3 Activity deadlines, cooperative cancellation and final lifecycle refinement

## Description
Moved from fn-227.4 on 2026-10-21.

This task is the activity-policy consumer of completed F (fn-237.1), O (fn-237.2), World-selected exact assumptions (fn-237.6) and fn-227.3. Remove framework fact/Option/conditional-assumption implementation from this task. Apply the approved R2/R3 contracts (fn-227's R4/R5 amendments at `bbf0835fbf`), retaining all laws and independently lifted source semantics. The final compatibility World and separate no-timeout World answer distinct questions; no all-timeouts-absent expiry witness is claimed.

**Size:** M after fn-227.3/F/O/fn-237.6. One existing activity lifecycle extension using already-proved typed parameters, facts, selected-World exact assumption source admission and replay. Native/TLC policy matrices, fairness and the existing refinement harness are substantial controls of that one extension. A newly required unsupported source construct returns to explicit support planning; it cannot be hidden by removing a law or using native-graph emission.
**Dependencies:** Consume fn-237-activity-timeouts-heartbeat-and.1 (F), fn-237-activity-timeouts-heartbeat-and.2 (O) and the committed implementation/Flow closure of fn-237-activity-timeouts-heartbeat-and.6. Cross-spec prerequisite fn-227.3 (retries and the retained-lifecycle refinement harness), satisfied through this spec's dependency on fn-227. No dependency on fn-227.12 (T); live timeout and cancellation admission belongs to fn-237.4, which consumes this task; fn-237.5 follows it.
**Files:** `model/temporal/feature/activity/{model.rs,tests.rs,temporal.rs,phase.rs}`; activity `expected/**`; the fn-227.3-owned `compatibility/**` projection/harness, without changing immutable old reference bytes; `model/temporal/realization/tests/realization.rs` only for compatibility with the new state/actions before fn-237.4's evidence implementation; `model/temporal/hosts/go/umpire_temporal_guest.wasm`; `model/README.md`.
**Touches:** [model/temporal/feature/activity/**, model/temporal/realization/src/live.rs, model/temporal/realization/tests/**, model/temporal/testing/**, model/temporal/hosts/go/umpire_temporal_guest.wasm, model/README.md, model/temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization-authored.snap, model/Cargo.lock, model/temporal/hosts/go/allocation_test.go, model/temporal/hosts/go/parity_test.go, model/temporal/hosts/go/annotations_test.go]

### Mechanical expansion mirror (root authorization, 2026-10-24)

The exact `model/temporal/realization/macros/src/snapshots/umpire_temporal_realization_macros__tests__activity-realization-authored.snap` mirror follows the authorized constructor and Model state-literal changes in `temporal.rs`. Regenerate only through its owning snapshot test. This adds no realization capability or policy and changes no acceptance criterion; all other macro snapshots remain protected.

The exact `model/Cargo.lock` package dependency-list mirror follows test-only edges from `model/temporal/testing/Cargo.toml` to existing workspace `umpire-front` and `syn`. No third-party package or version is added. Preserve every unrelated lock entry. Root authorized this mechanical mirror correction on 2026-10-24.

The exact Go test consumers `model/temporal/hosts/go/{allocation_test.go,parity_test.go,annotations_test.go}` follow the authorized timeoutless live constructors and explicit action availability. The refreshed guest reproduced 15/5 states against their obsolete 18/6 assertions. Root authorized this test-only correction on 2026-10-24. Preserve exact action/row sets, outcomes, allocation/publication rollback and response equality. This adds no production Go behavior or live admission.

### Approach

1. Consume fn-227.2's checked scalar constructor and O's typed Ticks/Remaining representation for the four optional configured deadlines and dynamic countdowns. Use one explicit test-local checked World factory for offline fixtures and fn-237.4's extension of fn-227.5's native option admission for live configuration. Keep one canonical range declaration. Some(Ticks(0)) fails those explicit configuration boundaries and source binding; raw World::new remains infallible. Author and source-lift bounds invariants for countdown starts/updates, so a forged or broken native Model value refutes a claim rather than relying on nonexistent generic admission. None is absent. A one-tick armed budget expires on its first decrement without underflow. State explicitly whether terminal exhausted Remaining zero is retained or cleared. No armed nonterminal zero may deadlock instead of expiring.
2. Replace direct deadline actions with the one Tick. Decrement every live budget as authored, expire the first exhausted deadline in an explicit Model priority, and emit exactly its F enum fact. Ordinary/nonexpiring edges emit no fact. Schedule-to-close spans all attempts; start-to-close/heartbeat reset on Poll. State and test schedule-to-start arming/reset across retries and heartbeat reset while Started rather than leaving those implicit. Tick without a live budget is disabled.
3. Preserve atomic cancellation for offline compatibility. Cooperative Cancel during Started sets pending; Heartbeat reports pending without closing, completion/failure remain permitted races, and current-attempt worker acknowledgement closes as Canceled. Pending retryable failure (including at the cap) and selected StartToClose/Heartbeat expiry also close Canceled; selected ScheduleToClose expiry is TimedOut. A fresh distinct pending Cancel rejects FailedPrecondition without changing state. These are explicitly chosen boundary policies; preserve original atomic/nonpending retry laws and add separate cooperative exceptions. Terminal closure clears budgets and pending state. Use a named acknowledgement action. Keep pending state/facts distinct from terminal cancellation; retain ordinary terminal closure/rejection laws. Scheduled cancellation and rejected stale acknowledgements/attempts receive explicit matrix controls.
4. Use existing exact-action weak fairness, separately for each admitted concrete Poll{expected_attempt:k} and RespondCompleted, while retaining heartbeat, failure, cancellation and Tick choices. Poll fairness is necessary after a retry returns to Scheduled; closing-action fairness prevents infinite heartbeat from satisfying the whole worker obligation. RespondCompleted remains enabled in Started even with cancellation pending under the approved race policy. Record the actual finite fair-action list and emit a separate WF obligation for every selected action value. Fairness over one existential Worker/Poll family is insufficient, because stale rejected Poll values could satisfy it forever. Verify eventual closure in a no-timeout World without inventing deadlines, excluding heartbeat or forcing completion behavior in the Model. Verify the schedule-to-close total-tick bound only over its declared applicable Worlds/starts. Its negative controls reset a spanning budget or break an expiry and must refute; no clamped counter, disabled invalid update or exploration cut may fabricate a proof.
5. Run fn-227.3's retained-graph comparison with one attempt, atomic cancellation, one-tick schedule-to-start/start-to-close and the other timeouts absent. State the full state/action/fact projection, including Tick-to-deadline mapping using expiry fact, input assignment deduplication and any explicitly erased new internal stutters. Show those rules preserve every old visible edge and terminal rejection behavior. The immutable old reference and changed-old-edge failure control remain active. Re-run after every timeout/cancellation policy change.

### Investigation targets

Required:

- `model/temporal/feature/activity/model.rs` and `tests.rs` for the post-fn-227.3 lifecycle, laws and retry matrix.
- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/{deadline-fact-contract.md,optional-ticks-contract.md}` for delivered support interfaces.
- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/cancellation-evidence-research.md` for pinned public protocol evidence and its approved correction.
- `model/temporal/feature/activity/compatibility/` for the fn-227.3 retained reference and comparison harness.
- `model/temporal/feature/activity/temporal.rs` and `model/temporal/realization/tests/realization.rs` for necessary interim binding compatibility.

### Verification

Quick from model/: `mise exec -- cargo test -p umpire-temporal-tests`; `mise exec -- cargo test --manifest-path temporal/realization/Cargo.toml`; repeat activity backend tests under `UMPIRE_BACKENDS=require`. Exercise each deadline alone, simultaneous expiry, reset/spanning across retries, zero/one/max, no-live-budget and pending cancellation races. On timeoutless Worlds, removing closing-action fairness must expose the infinite heartbeat counterexample, and omitting current-attempt Poll fairness must expose retry starvation despite other stale rejecting Polls. Run changed-edge, broken-budget, broken-expiry and heartbeat-closes mutations through actual tests. Scoped fmt and strict clippy.

Closed I (fn-227.8; `3fd1d2f551`, `0469334f91`) already delivers exact concrete-value WF. Consume fn-237.6's bounded exhaustive World-only helper selection to author Poll(1)..Poll(maximum_attempts) plus RespondCompleted independently in source and compiled native. Reanchor both receipts, preserve I's action_inputs controls and its explicitly nondiscriminating bounded-fairness behavioral limitation, and exercise stale-Poll starvation on the actual activity. A missing exact-WF regression goes through its owning support boundary; do not reopen closed I, infer policy by catalog intersection, or implement generic framework support inside this consumer.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, refreshed `make umpire-guest`, `make umpire-check-model-slice`, `make umpire-check-model-gaps`. Required TLC must execute the activity and replay exact expiry facts and false claims. Retain native/default and larger-world counts, Cases(n), actual fairness descriptors and timing including soft-budget misses. Update only approved activity output; old independent evidence and unrelated artifacts/metadata remain exact.

### Worker actions under programmed workers (2026-10-24)

Heartbeat and the cancellation acknowledgement are Model actions the worker performs and Umpire observes in server history (fn-243); nothing in the Model changes for that. `temporal.rs` gains their program rows in fn-237.4, not here; this task only keeps the realization compiling against the new state and actions.

### Boundary-policy proposal for fresh review

Consume the revised R2 matrix and retained independent public research. Tick selection is ScheduleToClose > StartToClose > Heartbeat > ScheduleToStart; one exact expiry fact records the selected trigger even when terminal phase is Canceled. Preserve ordinary/no-expiry empty facts. The older activity-specific fact=>TimedOut/Cancel=>empty statements are superseded, not the generic F contract. Keep frozen atomic compatibility artifacts and exact projection unchanged. Explicitly scope old retry-at-cap laws to atomic/nonpending states and independently check the new cooperative branch; do not remove laws or conceal an exception. Same-request-ID replay has no Model outcome promise. Public evidence versus design inference must be labelled before comparing the deployment. Add pending retryable-failure below/at-cap, pending attempt-expiry, global-priority and fact-only mutants to native/TLC controls.


Selected policy and provenance: `.flow/artifacts/fn-237-activity-timeouts-heartbeat-and/cancellation-boundary-contract.md`; exact live evidence limits are in `cancellation-boundary-evidence-shape.md`. Fresh plan review must pass before implementation.

### Constructor compatibility owner (plan-review P2)

This task owns minimal production constructor compatibility in `model/temporal/realization/src/live.rs` and the feature default-runtime constructors when the generated World gains fields, so its gates compile before .4 adds live admission. Preserve the exact pre-.4 admitted semantics: actual maximum_attempts plus atomic cancellation and the previous timeout-absent live defaults; do not enable cooperative mode, fabricate Start fields or admit nonzero timeout values here. The declared final offline compatibility World separately retains its one-tick deadlines and exact old projection. .4 later replaces the live defaults with validated request-derived fields and a pinned binding. Keep the existing unheld atomic scope refusal until .5 consumes .4's committed proof.
## Acceptance
- [ ] Independently specify repeated cancellation while a cancellation request is pending from the approved domain/API contract before comparing implementation behavior. The retained server code 9 observation does not itself authorize Model semantics or a NotFound remapping; preserve a genuine disagreement as a finding.
- [ ] All four deadline arming/reset/spanning and authored simultaneous-priority controls pass. Exact expiry facts, zero handling and disabled no-budget Tick survive native/TLC agreement and replay.
- [ ] Cooperative pending-heartbeat/acknowledgement and completion/failure races follow the approved policy; atomic compatibility and all terminal/stale-attempt laws remain. A heartbeat-only closure mutation fails.
- [ ] Exact concrete Poll and closing-action weak-fairness obligations verify progress natively and on required TLC Worlds, including no-timeout/retry Worlds. Union-fairness, missing-close and stale-Poll starvation controls remain discriminating; applicable tick-bound/reset/expiry defects refute without counter clamping or cuts.
- [ ] The final compatibility World passes exact retained-lifecycle projection/comparison; changed old edges fail. All old reference bytes and every old law remain independently checked.
- [ ] Activity goldens/coverage state Cases(n), source/TLC counts and timings. Existing live controls pass after the normal guest refresh; unrelated golden/expansion bytes remain exact. Live timeout/protocol admission is handed to fn-237.4, not inferred from a checker World.


## Done summary
Implemented the approved activity timeout, heartbeat, retry and cooperative cancellation policy, with exact selected fairness, native and TLC replay controls, and a byte-exact retained lifecycle projection. Live constructors retain atomic cancellation and absent timeouts; live cooperative admission and actual SUT ACK evidence remain assigned to later tasks.

Tier: session (jev-unavailable(no_key)); AGENTS implementer gpt-6.1-sol medium / reviewer gpt-6.1-sol high.

Baseline was green before edits after the handoff failed closed on an external MILESTONES commit. The fixed pre-edit base is 88f3f2098e434cdcd3078f8afe792ae1e64b0fe7. Source implementation is e04f0dc1a3a28d33c18ebd94b6e45ac039b01b98. External fn247/fn246 planning and root MILESTONES metadata in the range retain their original authorship and receive no implementation credit here.

Actual final exits were zero for make model-check (stable-head rerun 441.325s), UMPIRE_BACKENDS=require make model-check (960.602s), make model-time (373.928s), make umpire-check-model-slice (98.727s), and make umpire-check-model-gaps (362.102s). Both full model-check variants measured 722 Rust tests passed, zero failed, four inherited ignored, plus host checks. model-time inner work was 6.566s; its 367.163s gate exceeded the soft 60s target. An earlier default pass was invalidated by external MILESTONES changes and was not reused. Its 455.374s wrapper included deferred receipt coordination; exact Make-only wall was unobserved.

Focused policy and proof checks cover every budget, priority, pending failure boolean, cancellation race, current/stale action and selected fairness premise. Actual ACK witness replay retained four traces with 3, 7, 6 and 8 edges, against the committed Model and checked_step APIs. This is source/TLC proof, not actual SUT ACK proof. Compatibility remains exactly one start, six states and eleven old edges, with the changed-edge negative control. All 863 unrelated protected files remain byte-exact, including the task6 world_fairness artifacts and immutable originals. Guest SHA256 is eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0.

Six host subagents supplied four isolated implementation lanes and fresh correctness/standards audits. The one standards finding was fixed; both final audits have no remaining findings. All four owned worktrees and branches were removed normally after recovery verification. Shared caches were preserved. No attributable command remains running at handover.

Detailed evidence, exact commands, measured counts, initial failures, traces, scope exception hashes and recovery material are under .flow/artifacts/fn-237-activity-timeouts-heartbeat-and/activity-policy-verification.md and activity-policy-implementation-evidence.json. The execution archive and durable activity-policy-ack-replay.txt retain the raw proofs. The broader archived Scala/IR Go umbrella remains inherited red; this is not an all-repository green claim.

stage: impl-review - ran [2026-10-24T20:00:08.892841Z..2026-10-24T20:04:55.079940Z] (SHIP; three actual codex:gpt-6.1-sol:high draws, same-family disclosed; dispatcher/finalizer exits zero)
stage: plan-sync - skipped(config: planSync.enabled != true)
Tracker sync: n/a (bridge inactive)

GATE_SKIPPED:model-check:green-receipt 1ef0402c
GATE_SKIPPED:model-check-required:green-receipt 1ef0402c
GATE_SKIPPED:model-time:green-receipt 1ef0402c
## Evidence
- Commits: 40b9cb9ff15c1c24d69de9171dea0d2b340053e1, e04f0dc1a3a28d33c18ebd94b6e45ac039b01b98, 67a7c6af38c12cba9842e89932fb36d3c87a715a, 53a9c9ccbdbbc9ab4b006acd05deefdf711cdc4c, 505bc5013ab2e95ac12b3b6e84f9b76935ff7e43, 45c5a5a04c61994f1e41e3a15c1d9a77d5a4b4d0, 1ef0402c102a79ef1926c24ecd025372ad1bf123, 9d69325e2c366bc39c3782cef92cba507f81260f, 49ef7069fc4ef5cab58c87504fd20b523462ed68
- Tests: make model-check, UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps, GATE_SKIPPED:model-check:green-receipt 1ef0402c, GATE_SKIPPED:model-check-required:green-receipt 1ef0402c, GATE_SKIPPED:model-time:green-receipt 1ef0402c
- PRs: