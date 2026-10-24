---
satisfies: [R2, R3]
---
# fn-237-activity-timeouts-heartbeat-and.3 Activity deadlines, cooperative cancellation and final lifecycle refinement

## Description
Moved from fn-227.4 on 2026-10-21.

This task is the activity-policy consumer of completed F (fn-237.1), O (fn-237.2) and fn-227.3. Remove framework fact/Option implementation from this task. Apply the approved R2/R3 contracts (fn-227's R4/R5 amendments at `bbf0835fbf`), retaining all laws and independently lifted source semantics. The final compatibility World and separate no-timeout World answer distinct questions; no all-timeouts-absent expiry witness is claimed.

**Size:** M after fn-227.3/F/O. One existing activity lifecycle extension using already-proved typed parameters, facts, source admission and replay. Native/TLC policy matrices, fairness and the existing refinement harness are substantial controls of that one extension. A newly required unsupported source construct returns to explicit support planning; it cannot be hidden by removing a law or using native-graph emission.
**Dependencies:** Consume fn-237-activity-timeouts-heartbeat-and.1 (F) and fn-237-activity-timeouts-heartbeat-and.2 (O). Cross-spec prerequisite fn-227.3 (retries and the retained-lifecycle refinement harness), satisfied through this spec's dependency on fn-227. No dependency on fn-227.12 (T); live timeout and cancellation admission belongs to fn-237.4, which consumes this task; fn-237.5 follows it.
**Files:** `model/temporal/feature/activity/{model.rs,tests.rs,temporal.rs,phase.rs}`; activity `expected/**`; the fn-227.3-owned `compatibility/**` projection/harness, without changing immutable old reference bytes; `model/temporal/realization/tests/realization.rs` only for compatibility with the new state/actions before fn-237.4's evidence implementation; `model/temporal/hosts/go/umpire_temporal_guest.wasm`; `model/README.md`.
**Touches:** [model/temporal/feature/activity/**, model/temporal/realization/tests/realization.rs, model/temporal/hosts/go/umpire_temporal_guest.wasm, model/README.md]

### Approach

1. Consume fn-227.2's checked scalar constructor and O's typed Ticks/Remaining representation for the four optional configured deadlines and dynamic countdowns. Use one explicit test-local checked World factory for offline fixtures and fn-237.4's extension of fn-227.5's native option admission for live configuration. Keep one canonical range declaration. Some(Ticks(0)) fails those explicit configuration boundaries and source binding; raw World::new remains infallible. Author and source-lift bounds invariants for countdown starts/updates, so a forged or broken native Model value refutes a claim rather than relying on nonexistent generic admission. None is absent. A one-tick armed budget expires on its first decrement without underflow. State explicitly whether terminal exhausted Remaining zero is retained or cleared. No armed nonterminal zero may deadlock instead of expiring.
2. Replace direct deadline actions with the one Tick. Decrement every live budget as authored, expire the first exhausted deadline in an explicit Model priority, and emit exactly its F enum fact. Ordinary/nonexpiring edges emit no fact. Schedule-to-close spans all attempts; start-to-close/heartbeat reset on Poll. State and test schedule-to-start arming/reset across retries and heartbeat reset while Started rather than leaving those implicit. Tick without a live budget is disabled.
3. Preserve atomic cancellation for offline compatibility. Cooperative Cancel during Started sets pending; Heartbeat reports pending without closing, completion/failure remain permitted races, and only explicit worker cancellation acknowledgement closes as Canceled. Use a named acknowledgement action. Keep pending state/facts distinct from terminal cancellation; retain ordinary terminal closure/rejection laws. Scheduled cancellation and rejected stale acknowledgements/attempts receive explicit matrix controls.
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

Committed `bbf0835fbf` native `checker/src/progress.rs` selects exact fair action values and the TLC emitter constructs WF for each admitted action operator. Checkpoint I (fn-227.8) owns exact concrete input identity/fairness; P (fn-227.11) consumes I. Reanchor I's receipt before this task and run its stale-Poll fairness control through the actual activity. If it emits only an existential input-family obligation, return the finite per-value repair to I's owner rather than adding an unbounded framework or weakening the Model/assumption.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`, refreshed `make umpire-guest`, `make umpire-check-model-slice`, `make umpire-check-model-gaps`. Required TLC must execute the activity and replay exact expiry facts and false claims. Retain native/default and larger-world counts, Cases(n), actual fairness descriptors and timing including soft-budget misses. Update only approved activity output; old independent evidence and unrelated artifacts/metadata remain exact.

### Worker actions under programmed workers (2026-10-24)

Heartbeat and the cancellation acknowledgement are Model actions the worker performs and Umpire observes in server history (fn-243); nothing in the Model changes for that. `temporal.rs` gains their program rows in fn-237.4, not here; this task only keeps the realization compiling against the new state and actions.
## Acceptance
- [ ] Independently specify repeated cancellation while a cancellation request is pending from the approved domain/API contract before comparing implementation behavior. The retained server code 9 observation does not itself authorize Model semantics or a NotFound remapping; preserve a genuine disagreement as a finding.
- [ ] All four deadline arming/reset/spanning and authored simultaneous-priority controls pass. Exact expiry facts, zero handling and disabled no-budget Tick survive native/TLC agreement and replay.
- [ ] Cooperative pending-heartbeat/acknowledgement and completion/failure races follow the approved policy; atomic compatibility and all terminal/stale-attempt laws remain. A heartbeat-only closure mutation fails.
- [ ] Exact concrete Poll and closing-action weak-fairness obligations verify progress natively and on required TLC Worlds, including no-timeout/retry Worlds. Union-fairness, missing-close and stale-Poll starvation controls remain discriminating; applicable tick-bound/reset/expiry defects refute without counter clamping or cuts.
- [ ] The final compatibility World passes exact retained-lifecycle projection/comparison; changed old edges fail. All old reference bytes and every old law remain independently checked.
- [ ] Activity goldens/coverage state Cases(n), source/TLC counts and timings. Existing live controls pass after the normal guest refresh; unrelated golden/expansion bytes remain exact. Live timeout/protocol admission is handed to fn-237.4, not inferred from a checker World.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
