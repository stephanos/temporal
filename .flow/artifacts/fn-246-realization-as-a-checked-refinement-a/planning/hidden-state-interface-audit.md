> Preserved planning evidence from the earlier draft-only pass. Statements that no Flow calls or canonical writes ran describe that pass. The separate canonical authoring pass now records fourteen tasks. No implementation/build/probe or primary review has established feasibility. All source coordinates require final post-fn-237.3/fn-247 reanchor.

# fn-246 snapshot/hidden-state interface audit

Scratch contract refinement only. No Flow, source change, library, build or probe ran. The final fn-237.3 constructors/nominal accessors and emitted graph still require reanchor after its committed closure and fn-247.

Correction after the conductor's concrete global/heartbeat counterexample: typed targets remain necessary but greedy commitment is unsound; the proposal to leave non-equivalent alternatives to fn-237.4 is withdrawn. The conductor selected direction A, explicit semantic2 whole-prefix witnesses with opt-in immutable guest metadata and existing Run.version admission/replay. `bounded-ambiguity-contract.md` records the corrected bounded frontier; `prefix-version-carrier-audit.md` records verified reader source and the mandatory production-retention/type-coupling test. Legacy1/Exact/Kettle remains delta. This is an explicit semantic amendment preserving layouts/exports, not an unresolved representative workaround or blanket unchanged-semantics claim.

## Information boundary

fn-237.3's Model state carries four dynamic Remaining budgets and pending cancellation, with independent arming/reset/spanning/decrement/priority/terminal-clear laws. Its task explicitly keeps pre-.4 live admission at actual maximum_attempts plus atomic cancellation and timeout-absent defaults; .4 later admits request-derived nonzero timeouts and a pinned cooperative binding. Public Describe classes generally cannot report the exact remaining abstract ticks. The draft's snapshot-to-full-State table and literal Reach(M::State) target therefore overstate available information.

The existing runtime already separates validated observation decoding from checked_step/replay and the chosen prepared state. Extend that seam with one typed compatibility target. Do not construct a full State from snapshot defaults, configured timeout values, desired next action, terminal Canceled or elapsed wall time. The complete candidate State comes only from the current Model state and its actual checked transitions.

## Minimal proposed typed contract

- Generic runtime owns `ReachTarget<S>` with one pure `compatible(&self, candidate: &S) -> bool` contract and `Exact<S>` for complete observations. The predicate filters candidates; the separately required bounded-frontier prerequisite preserves compatible histories through existing checked-step/replay, without another interpreter.
- Reach carries the realization's associated target type rather than requiring a manufactured M::State. Existing Current/Candidates remain. Wire envelopes do not change. Direct and derived adapters explicitly supply Exact where appropriate; do not rely on unstable Rust associated-type defaults to keep legacy implementations compiling.
- The activity feature owns `ActivityTarget`, created only from validated actual observation fields, identity and declared read semantics. Each field is known from actual evidence or explicitly unobserved under its contract. Unknown is never represented by a guessed default value and cannot erase contradictory observed data. Exact remaining budgets are unobserved unless an actual admitted evidence kind establishes them.
- The feature-local `ReachTarget<activity::Model>` implementation compares the observation's known constraints with the candidate's visible semantic view. Result/failure/worker-ACK/cause requirements that depend on transition/history remain in typed per-edge evidence predicates; state matching cannot replace them. Predicate decisions are pure over admitted inputs; diagnostics retain field identity without private values.
- Shared protocol owns the neutral observable-view types and protocol-state-to-view conversion. Feature realization owns Model-state-to-view compatibility. Offline checks validate their commutation alongside the total full-state map. Thus online and offline share one observable contract while preserving their different source types.

The exact Rust generic spelling follows the reanchored Realize/Projection owner. The binding contract above is fixed before source changes; it needs only the target payload, pure compatibility hook and Exact adapter already implied by Reach, not arbitrary generic runtime expansion.

## Full offline protocol state

The protocol Model is not merely a set of public Describe classes. Give it independently authored latent timer/budget state plus its public observable view. Its local World, starts, timer arming/reset/spanning/decrement/priority, expiry causes/facts and terminal clearing remain complete and independently checked natively/TLC. Include explicit protocol timer actions wherever its independent semantics require them; public visibility does not delete those transitions.

The native refinement declaration remains a total map from full protocol state to full feature state, including each active/absent Remaining budget, attempt, phase and pending state, with exact World/nominal/domain provenance. The shared parameter-contract and existing typed replay validate it. Maintain an explicit every-field state-accounting inventory. An omitted/erased budget, wrong reset/spanning behavior, clamping, default substitution or wrong expiry fact is a violated/incomplete relation, never a convenient observational quotient or gap that earns preservation credit.

Check that taking the public view before/after this total map agrees with the declared observable compatibility. Snapshot decoding produces an ActivityTarget, not a complete protocol state. No exact full-state conversion is inferred from that partial target.

## Evidence and latent ambiguity limits

Reach target compatibility only filters candidate states. Every traversed edge still needs the existing typed evidence contract or a lawful declared stutter. Unobserved budget changes do not authorize arbitrary Tick edges; a Tick that changes full feature state is not a full-state stutter merely because Describe looks the same. Any complete path's hidden values come from checked Model transitions, and the judgment is a replayed witness consistent with observed evidence, not a measurement of actual remaining ticks or proof of a unique hidden state.

The earlier representative-at-later-admission proposal is refuted. R1 retains bounded compatible histories/work; canonical order selects reports, not pruning. The global=3/heartbeat=2 example has a legitimate fully evidenced global expiry lost by greedy commitment; it is not missing cause and cannot be waived by fn-237.4. Selected semantic2 whole-prefix witnesses resolve publication coherently: every latest witness satisfies all observations from Reset and old reports stay immutable existential claims. Legacy1 keeps deltas. Use actual mode-coupled metadata/discovered version and the earliest production carrier controls from the new audit, with no inferred clock, fabricated budget, rewritten old Run or weakened law.

Exact pending-expiry cause remains fn-237's named narrow scope. Canceled alone does not distinguish current-attempt ACK, retryable failure, StartToClose or Heartbeat expiry. Configured durations, transition count, stale last_failure, elapsed time and the requested action cannot supply the missing cause. Only actual typed/correlated evidence supports the corresponding expiry fact. Preserve original mandatory held-started heartbeat/ACK, completion-after-cancel, held schedule-to-start and inherited retry/control rows, plus fn-243.5's separate unheld proof.

## Required discriminating controls

- One public Started snapshot is compatible with distinct reachable full states whose Remaining budgets differ; target construction invents neither value. A real observed contradiction still fails.
- Unknown budget differs from explicit absent budget and present zero. Wrong-kind/bound, omitted field, clamping/default and None/present-zero conversion mutants fail.
- Kettle Exact matching retains its complete-state behavior and current/zero-step positive-evidence controls.
- Removing/changing protocol countdown, reset, spanning, expiry priority/fact or terminal-clear fields breaks full-state refinement even when public snapshot classes match.
- Snapshot-target compatibility commutes with the total protocol-to-feature map while every full-state field remains accounted.
- Evidence-free hidden Tick and visible-equal/full-state-changing purported stutter fail. Same-state/different-evidence-cursor paths remain distinct under existing Reach dedup rules.
- Same terminal Canceled snapshot with missing/changed actual cause/ACK/failure evidence cannot earn a specific expiry fact; exact missing cause retains the predeclared narrow field-named scope and original rows remain mandatory.
- Multiple compatible hidden histories retain all required continuations through the bounded frontier, without uniqueness/measurement claims. The concrete global/heartbeat counterexample and immutable explanation/publication contract are R1 correctness gates, not a passable future admission limitation.
