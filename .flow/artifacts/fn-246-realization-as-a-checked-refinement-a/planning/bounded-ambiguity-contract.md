> Preserved planning evidence from the earlier draft-only pass. Statements that no Flow calls or canonical writes ran describe that pass. The separate canonical authoring pass now records fourteen tasks. No implementation/build/probe or primary review has established feasibility. All source coordinates require final post-fn-237.3/fn-247 reanchor.

# fn-246 bounded ambiguity and explanation publication contract

Scratch design amendment only. No implementation, Flow mutation, build, probe, dependency or schema change ran. This supersedes the hidden-state audit's proposal to leave non-equivalent live alternatives to fn-237.4. The conductor selected direction A on 2026-10-24: explicit semantic protocol2 whole-observation-prefix witnesses, discovered through opt-in immutable Wasm metadata and recorded in existing Run.protocol_version. Legacy1/Exact/Kettle keeps delta behavior. This is a deliberate semantic amendment; layouts/exports stay fixed. `prefix-version-carrier-audit.md` records source verification and the mandatory production-retention/type-coupling proof gate.

## Counterexample

Use the independently authored pending-cancellation policy with Started, ScheduleToClose=3 and Heartbeat=2. Both histories below fit the same actual heartbeat response/public snapshot.

| History before/at the heartbeat | State after heartbeat | Two later Ticks |
| --- | --- | --- |
| Tick, Heartbeat | global=2, heartbeat=2 | Global expiry wins priority and yields TimedOut |
| Heartbeat | global=3, heartbeat=2 | Heartbeat expires first and yields Canceled while pending |

Greedily committing the shorter history after the first heartbeat loses the legitimate later global-expiry continuation. Actual typed/correlated evidence can establish that later cause; this is not the narrow missing-expiry-cause case. No single raw-state representative has been proved sufficient for every promised admitted continuation. Do not erase budgets, infer a clock, weaken priority/closure laws, invent a heartbeat or waive the case through future admission.

## Existing source boundary

At stable `d4eff7eedb`, `runtime.rs:379-400` picks the first replayable candidate and publishes its exact steps/outcomes; `explained` prepares one state; `Change::Run` and `commit:689` install that state. `model.proto::Explanation` has only repeated steps and outcomes. The existing response has no alternative paths, ambiguous-state identity, proof-mode or revision field. Publication before commit is necessary and remains authoritative.

Keeping other states privately while treating the first published delta path as the committed concrete history is insufficient. Later selecting the Tick-before-Heartbeat alternative would contradict that published prefix. Keeping the old path and replacing its hidden budget is equally invalid. Replaying deterministic bytes alone cannot repair an incoherent explanation history.

## Minimum internal correctness contract

Reach needs a bounded evidence-compatible frontier for partial-observation realizations unless a genuine continuation-preserving representative theorem is supplied for every promised case.

- A branch contains full Model state, typed checked-transition ancestry including emitted Outcome and Facts, the actual observation-prefix/evidence-consumption position and data required for pending correlation/read policy. Starts are singleton. Exact/Kettle remains the compatible singleton case. Facts such as ExpiredDeadline are checked alongside source/action/choice/target, never inferred from terminal phase or Outcome alone; preserve typed owned/shared ancestry without new arbitrary Clone bounds.
- Each actual observation refines all prior compatible branches through the existing catalog, checked_step, replay, typed target and per-edge evidence owners. Keep all compatible successors required by the declared scope, not merely the first witness. A branch matches the complete actual observation prefix, not just the newest snapshot.
- The environmental-action evidence contract must explicitly state which unobserved Tick interleavings are compatible with actual observations. Such a Tick is an existential independently declared internal progression, not a fabricated observed Tick/count/cause. Preserve every admitted interleaving, including the counterexample. Target compatibility alone cannot authorize a Tick, and visible equality cannot turn its full-state budget update into a stutter. An absent predicate remains insufficient. Do not ban the required counterexample by silently tightening that declaration or infer an exact Tick count from elapsed time/desired expiry.
- Canonical order chooses a report witness, never silently prunes a future-relevant branch. Deduplication needs full state, evidence cursor/consumption state, position and witness/publication compatibility; state-only or public-view-only merging is unsound. Distinct hidden global budgets in the counterexample remain distinct.
- To preserve successful first-valid behavior without pretending every longer cyclic path was exhausted, retain any unexamined evidence-compatible search work explicitly as unresolved frontier work. It is not a successful complete branch and not a discarded alternative. Future refinement must account for it before rejecting an observation. An unexplored node retains its typed prefix, target/evidence cursor and remaining declared scope. Exact finite depth/work/retention bounds remain explicit.
- If no witness exists and compatible frontier work remains at a bound, return named Refused. Unexplained requires every viable branch/work item exhausted. A successful witness may prepare a normal judgment while alternatives/work remain, subject to the explicit publication contract below; it makes no uniqueness/completeness claim. Resource refusal cannot silently truncate retained branches or turn missing proof into acceptance.
- Prepare computes a prospective frontier, evidence-consumption cut, pending/finality decision and explanation together. Only successful publication commits them. Failed publication/allocation preserves the entire old frontier, ancestry, evidence and pending request. Reset starts fresh. Freeze and retry use the same transaction owner.
- Proposal/callback/read decisions cannot consult one arbitrary hidden branch. Existing capability decisions must agree on the actual request/action/finality across retained possibilities or obtain an explicitly supported disambiguating read. Required behavior cannot be discharged by ambiguity refusal; a disagreement remains a concrete blocking contract finding. This is not a general planning engine or a Model callback from Go.
- A retained history or observation budget exhaustion has a named refusal and preserves prior state/evidence. No guessed Remaining value, wall-clock inference, new source interpreter or changed Model semantics is authorized.

These are generic support requirements for R1, not optional fn-237.4 limitations. Use a separate cohesive runtime prerequisite after typed target/capability composition and before Temporal lens/activity/strategy integration. It must share the existing checked-step/replay and transaction paths and keep Exact/singleton behavior pinned.

## Original publication conflict and selected semantic amendment

The following three claims cannot all hold for the counterexample.

1. Every Explained response publishes a complete concrete delta path from the previously committed single state, and concatenating published deltas is the one committed Model history.
2. A partial heartbeat snapshot publishes success immediately despite unresolved non-equivalent hidden histories, with no proved universally sufficient representative.
3. Every later fully evidenced promised continuation remains explainable without revising prior published facts or fabricating hidden state.

The original single-path semantics supplies no explicit alternative-proof carrier. The conductor resolves the conflict by selecting explicit version2 whole-prefix witness semantics, with opt-in metadata tied to actual proof mode, admitted/discovered Host version and matching Run.version/replay handling. Version1 keeps the original contract. Primary review must challenge this deliberate amendment and its production/version controls. No invented action-name DAG encoding, overloaded continue_polling, silent empty-Current reinterpretation or rewritten old Run is permitted.

### Selected A and rejected larger alternatives

**A, selected by conductor.** Prefix2 steps/outcomes is a canonical witness for the complete actual observation prefix from Reset, not a committed delta history. Preserve all alternatives internally. A later witness satisfies every prior actual observation with original causal/evidence order; prior reports remain immutable existential prefix claims. Replay rebuilds the frontier from the exact recorded calls and deterministically reproduces every report. Earlier witnesses need not be prefixes of later witnesses, so consumers never concatenate them as deltas. Select semantics through an opt-in immutable closed guest metadata section tied to actual runtime proof-mode type, read through pinned wazero and recorded in existing Run.protocol_version. Unmarked legacy guests are1, explicit prefix guests2, unknown/malformed/duplicate advertisements refuse, and loaded guest/Run versions must match. Preserve old exact/v1 guests and recordings. No version function export, probe RPC or protobuf field/layout addition. Source verifies the reader API; production Rust/xtask LTO/strip retention and type coupling remain mandatory earliest controls, not completed evidence. See the carrier audit for exact sources and limitations.

**B. Explicit alternative-proof representation.** Publish the bounded compatible proof frontier, or a sufficient committed-prefix/deferred-suffix proof with explicit identity/status, so every later refinement links coherently without retracting prior published claims. This is the direct representation, but the current Explanation schema lacks it. It requires a narrowly approved core proof-contract/schema/version amendment; no such expansion is authorized by the current scratch task. Wire compatibility, guest/Go/replay consumers and old exact fixtures would need explicit gates.

**C. Preserve strict delta commitment by proving representative sufficiency for every promised admitted case.** The theorem must quantify over all evidence-compatible hidden states and all promised future observations/commands and preserve outcome/fact/closure behavior. The concrete global/heartbeat example refutes the existing greedy heuristic. Restricting cases, choosing a preferred cause, deferring a refusal to fn-237.4 or erasing budgets does not prove this theorem or satisfy the original request. No such theorem is currently available.

Option A is the explicit approved direction; B's core alternative-proof schema expansion is not approved, and C's greedy representative is refuted. If actual custom-metadata production fails, report the exact blocker and return for the minimal carrier decision rather than silently adding B, an export or a probe RPC. Do not claim every old semantic contract remains unchanged.

## Required proof controls

- Native generic fixture reproduces the exact global=3/heartbeat=2, hidden Tick-before-Heartbeat example; later fully evidenced global expiry remains explained and the greedy-single-state mutant fails. Also retain the alternative heartbeat-expiry continuation with its actual cause.
- Every chosen later prefix witness satisfies all previous actual observation payloads/order and edge evidence. Changing only an earlier hidden Tick, heartbeat reset, cause/fact or evidence cursor invalidates the wrong branch.
- Failed publication/allocation and repeated same observation preserve/reproduce the full frontier and report bytes; Reset, freeze, pending proposals, read finality and guest/native/Go replay remain deterministic.
- Same snapshot with distinct Remaining budgets stays distinct; omitted/clamped/default budgets, public-view-only merge, arbitrary first-state proposal and unsupported evidence-free Tick fail.
- Finite work/depth/history exhaustion retains unresolved branches/work and produces explicit refusal, never false Unexplained, false uniqueness or silent coverage loss. A found canonical witness still succeeds without claiming cyclic alternatives exhausted.
- Published explanations follow exactly the chosen explicit contract. Old concrete deltas cannot be retroactively reinterpreted or concatenated with a revised past. Existing Exact/Kettle singleton bytes remain pinned.
- Independent full protocol/feature refinement still accounts every timer field, reset/spanning/priority/fact and universal closure law. This runtime issue cannot be fixed by weakening either Model or invoking the narrow missing-cause exception.
