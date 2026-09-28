# Veil concrete checker as the Umpire search backend

> HTML render lens (local): open `.flow/artifacts/fn-88-veil-concrete-checker-as-the-umpire/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Umpire4 architecture reconciliation

This spec makes Veil's concrete model-checker library a search backend behind `Umpire.Search`,
consuming the explicit `FiniteTable` the `machine` command already produces through the
proof-carrying `SearchView`. It adopts no Veil DSL, no SMT path, and no Veil-authored semantics.
The authoring commands, `CheckedModel`, the Property evaluator and its denotation proofs, Case
production, Testpilot, and the Plan artifact format are unchanged. `Umpire.Search` owns the backend
seam and the shared finalization; the Veil adapter imports `Umpire.Search` and is imported only by
the selection module, so `Umpire.Search` itself never imports Veil, and a direct-import
`ModelLint` rule pins that. It is separate from the optional symbolic checker slot
(`Umpire.Verify.Veil`, fn-23, fn-24, fn-25), which it neither starts nor closes; fn-24 and fn-25
now depend on it, and fn-23 needs re-scoping (Decision Context). The word "backend" is used
because fn-82 retired `Engine` under SEM-20; the existing `SearchStats.backendPulls` counter is
renamed `enumeratorPulls` so "backend" keeps one meaning (SEM-19). The direction note
`.plans/UMPIRE4_DIRECTION.md`, sections 1 and 6, records the reasoning this spec executes.

Rule text that conflicts with a Veil dependency in the primary Lake project exists in at least ten
places (Decision Context lists them). One task drafts every amendment under GOV-02, and the task
that adds the dependency depends on it.

## Overview

Replace the path-enumerating depth-first search inside `Umpire.Search` with a product-state
reachability check run by Veil's concrete breadth-first checker, for the Query forms and clause
kinds a monitor lowering supports, and keep today's traversal as the frozen `reference` backend and
differential oracle. The first task is a compatibility probe whose receipt selects `adopt`,
`defer-closure`, or `defer-incompatible`; only `adopt` continues.

## Goal & Context
<!-- scope: business -->

`Umpire.Search` is a 1.2k-line hand-written checker: iterative-deepening depth-first search over
traces, capped by a candidate count, with no visited-state set. It revisits every state once per
path, so the multi-instance models fn-85 introduced (two interleaved operations today, five planned)
reach `limit-reached` on questions a visited-set search answers in milliseconds. The roadmap's next
features, symmetry canonicalization, implicit fault placement, seeded walks, are a model checker's
feature list, and building them into `Search.lean` means owning a fourth explicit checker beside
TLC, Stateright, and Veil.

The team wants to own as little checker code as possible while keeping Lean as the frontend,
because the `machine` command's enumeration into `FiniteTable` and the `Machine` sound and complete
proofs are what make the compile step from author's code to checked rows trustworthy. Veil's
concrete checker is the only mature explicit-state engine written in Lean. Consuming it as a
library keeps counterexamples as Lean values, needs no serialization, and lets the kernel replay
gate stand as the trust boundary.

The 2026-09-06 probe under `experiments/umpire-dsl/veil` showed the data adapter works: it imported
Veil's `EnumerableTransitionSystem` and `RelationalTransitionSystem` unchanged, proved transition
and initial-state equivalence in both directions with only `propext` and `Quot.sound`, and replayed
335 paths. It never ran Veil's breadth-first checker: the full dependency graph failed on Batteries
under Lean 4.33.1 and the machine ran out of disk. This spec finishes that run inside the model's
own dependency graph and, if the checker builds, makes it the default backend.

Who benefits: a model author whose exhaustive `verify` on a two-instance model completes instead
of reporting `limit-reached`, and reviewers who diff one adapter instead of a growing checker.

## Architecture & Data Models
<!-- scope: technical -->

```text
machine command ──enumerates──▶ CheckedTable + Machine proofs ──▶ SearchView (ordered, proven)   unchanged
CheckedScenario ──lower──▶ progress automaton (R14)                                              new
CheckedProperty ──lower──▶ monitor automata, v1 clause kinds, three-valued answers (R5, R6)      new
                                        │
                                        ▼
        Product = model state × scenario progress × monitor states × fired-clause bitset
                                        │
                        ┌───────────────┴────────────────┐
                        ▼                                ▼
   Backend.reference (in Umpire.Search, frozen)    Backend.veil (Umpire.Search.Backend.Veil)
                        │                                │
                        └───────────────┬────────────────┘
                                        ▼
   Umpire.Search.finalizeBackendResult: kernel replay of any witness (R10) → the existing
   finalization → PlanResult
                                        │
                                        ▼
                    Producer, Cases, fixtures                                                     unchanged
```

Import direction: `Umpire.Search` (semantic root) ◀── `Umpire.Search.Product` ◀──
`Umpire.Search.Backend.Veil` (the only importer of `Veil.*`) ◀── `Umpire.Search.Selection`
◀── `Umpire.Search.Admission`. `Umpire.Search` imports none of them.

Components:

- **Backend seam inside `Umpire.Search`** (R4). `Search.lean` keeps its private traversal and
  gains three public declarations: the `BackendResult` type, `Backend.reference` (today's
  traversal returning a `BackendResult`), and `finalizeBackendResult : CheckedQuery → SearchView →
  BackendResult → Except KnownGapError PlanResult`, which owns kernel replay, the observation
  aggregate, `finalizePlanning`, and the `stillPending`/`neverTriggered` post-pass. The existing
  `search` becomes `finalizeBackendResult` applied to `Backend.reference`. A `Backend` is a
  function `CheckedQuery → SearchView → Limits → SearchStrategy → BackendResult`. Nothing else in
  `Search.lean` changes, and its private functions stay private.
- **`Umpire.Search.Product`** (new, R14). Builds the product from `SearchView`, the Scenario
  progress automaton, and the Property monitors. Product transitions are labeled with the model
  `Step`, and a product path decodes to a `Umpire.Scenario.Trace` with its setup and initial state.
  Product state is `BEq` and `Hashable` and fully determines the future verdict, which is what
  makes visited-set dedup sound. It carries a fired-clause bitset and the per-clause
  three-valued answers so `never-triggered` and `still-pending` become reachability questions.
- **Scenario progress automaton** (new, R14). Version one encodes what `CheckedScenario.admits`
  reads for: per-action occurrence counts saturating one above the declared maximum, subsequence
  (`sequences`) progress indices, the exact-trace index for `traceExactly` and `actionsExactly`,
  and the fixed setup. `ordering` constraints, whose slot assignment depends on the set of
  remaining occurrences, and `adjacencies`, whose substring containment needs active partial
  matches, are `Unsupported` over a free schedule in version one; when `traceExactly` or
  `actionsExactly` fixes the schedule, each is decided once when the automaton is built (amended
  2026-09-27 by fn-88.3: every `scenario`-command Scenario carries an `ordering` through
  `Scenario.exactly`, so rejecting them would keep every feature Model off the new search).
  `admits` and `admitsPrefix` stay as the oracle.
- **Property monitor lowering** (new, R5). A function from `CheckedPropertyClause` to a bounded
  monitor that answers `PropertyEndpointAnswer` (`satisfied`, `violated`, `unresolved`) for closed
  endings (`final` or `terminal`) and for `partial`, or a typed `Unsupported` naming the clause kind. The
  state and the closed and partial verdicts of each monitor are read off the evaluator, not the
  clause's name: `stateInvariant` keeps a seen bit and a failed bit and is `unresolved` while no
  matching state has been seen; `identityRelation` is existential over the trace and keeps a
  monotone seen bit, `unresolved` until it fires under `partial`; `transitionContract` and
  `inputOutput` are one-step implications with a seen bit for coverage; `ordered` keeps a
  seen-before bit per ordered pair and is `unresolved` under `partial` until decided;
  `eventuallyWithin` and `neverWithin` keep a countdown bounded by the clause Limit and are
  `unresolved` before the deadline. `LimitUnit.steps` and `LimitUnit.actions` both count product
  steps, as the evaluator treats them; `logicalTime` is `Unsupported`. `branches`, the guarded
  variants, and correlated clauses are `Unsupported` in version one. The Case Producer's lowering
  is not changed and not shared; it is trace-bound and targets runtime Observations.
- **`Umpire.Search.Backend.Veil`** (new). The only module that imports `Veil.*`. Wraps the
  product as a Veil `EnumerableTransitionSystem`, hands successors to the checker in table order,
  runs the concrete breadth-first checker with the depth and state bounds derived from `Limits`,
  and decodes the result into a `BackendResult`. Carries the probe's equivalence theorems restated
  over the product (R7). For `verify` it continues to the frontier's end after a violation so the
  witness is the shortest one; it does not attempt to reproduce the reference's path counters.
- **`Umpire.Search.Selection`** (new). One function that chooses `veil` or `reference` for an
  admitted Query and records the reason; `AdmittedQuery.search` calls it, and a test-visible
  `AdmittedQuery.searchWith (backend)` runs a named backend for the differential test (R8). This
  module and the adapter are the closed set the rollback drill deletes (R21).
- **Kernel replay** (new, R10), owned by `finalizeBackendResult`. A decoded `Scenario.Trace` is
  accepted only when every step is a `SearchView` member from a proven initial state and when the
  same decision `observeCandidate` makes today holds: `behavior.admits`, the `isTerminal`
  condition for `ending = terminal`, the endpoint answer with the partial flag, and the `find`
  and `pick` coverage condition. No standalone function does this today.
- **Veil dependency.** One `require` in `model/lakefile.lean` at one pinned commit. Lake builds
  only the imported `Veil.Core.Tools.ModelChecker` closure, though `lake update` fetches Veil's
  whole package graph. The probe (R1) records the closure, its Batteries pin against the model's
  `v4.33.0`, and whether Loom, lean-smt, mathlib, or widget modules are in it.
- **`ModelLint` rule `search-backend-isolation`** (R3). A direct-import rule in the shape of
  `authoring-path-isolation`: `Veil.*` may be imported only by `Umpire.Search.Backend.Veil`, and
  that module only by `Umpire.Search.Selection`. The complete-mode lint already loads external
  metadata (it reaches `Lean.Elab.Term` for the semantic-root rule), so it will also traverse
  Veil's; R1 measures that cost.
- **`Search.lean` frozen** (R12). Its private traversal gains no feature. `analyzeBranches` keeps
  consuming the path fold.

Data shapes:

- `BackendResult` (in `Umpire.Search`):
  - `violationFound (trace : Scenario.Trace) (observations : PlanningObservations)`
  - `complete (observations : PlanningObservations)`, meaning every product state reachable within
    `maximumDepth` (the minimum of `steps` and `actions`, as today) was visited and no violation
    was found; exhausting the depth bound is `complete`, exactly as `pullCandidate` returns
    `.complete` when iterative deepening reaches `maximumDepth`
  - `stateBound (visited : Nat) (observations : PlanningObservations)`
  - `PlanningObservations` is the aggregate the reference produces today, made public so an
    adapter can build it: whether any trace was admitted, whether any endpoint was `unresolved`,
    the realized and requested triggers, `nonempty`, the `ExploredCounts` that feed the Plan's
    `explored` and `boundWasHit`, and the `SearchStats` counters, so `finalizeBackendResult` has
    everything the Plan and receipt need.
  There is no `cancelled`: `search` is a pure total function.
- `SearchStats` gains `searchBackend ∈ {reference, veil}`, `backendReason ∈ {default,
  unsupported-clause:<kind>, unsupported-scenario:<construct>, unsupported-strategy:seeded,
  unsupported-form:<form>}`, `searchUnit ∈ {paths, states}`, and `veilCommit` (present only when
  `searchBackend = veil`); `backendPulls` is renamed `enumeratorPulls`.
- `canonicalPlanningReceiptJson` moves to `umpire-planning-receipt/v2` and carries those fields.
  The Plan artifact codec is unchanged; its `explored` counts keep their field names, and under
  `veil` `explored.traces` is the number of product states visited so `boundWasHit` keeps
  comparing it with `limits.search`. The Go artifact decoder is untouched.
- Outcome mapping through the existing `finalizePlanning`: `violationFound` → `found` for `find`
  and `pick`, or the violation for `findViolation` and `verify`; `complete` → `verified-within-
  limits` or `none-found` under the same completeness-evidence gate as today, or `unsatisfiable`
  when the observations show no admitted endpoint; `stateBound` → `limit-reached`;
  `never-triggered` and `still-pending` from the observations as today. The receipt's `triggers`
  evidence becomes one witness per clause.
- `QueryErrorKind.unreplayableWitness` (new constructor of the existing kind enumeration); the
  decoded trace and the replay diagnostic are rendered into `QueryError.offendingValue`.

## API Contracts
<!-- scope: technical -->

- `Search.admit`, `AdmittedQuery.search`, `search`, and `searchWithPlanRequest` keep their
  signatures and `Except` result types. `AdmittedQuery.searchWith` is added for tests. Selection is
  internal: `veil` when every clause lowers, the Scenario lowers, the strategy is not `seeded`, and
  the form is `verify`, `find`, `findViolation`, or `pick`; `reference` otherwise, with the reason
  recorded. No CLI flag, Query field, or environment variable selects the backend (CLI-02).
- `Limits`: `steps` and `actions` bound product depth as `maximumDepth` does today; `search`
  bounds candidate paths on `reference` and product states visited on `veil`, and the receipt
  names the unit (PLN-01, PLN-04).
- Witness order (R15): the reported witness is the lexicographically least shortest trace in the
  key order `pullCandidate` uses today, sorted setup, initial index, action index, outcome index.
  `veil` reproduces it with table-ordered successors, first-discovery parents, and a
  single-threaded frontier.
- Every `violationFound` trace passes kernel replay inside `finalizeBackendResult`; a trace that
  fails is `invalid` with `QueryErrorKind.unreplayableWitness` and is never `found` (VER-05).
- Differential comparison (R8) is defined as: equal `PlanningOutcome`; equal witness
  `Scenario.Trace`; equal Plan artifact bytes except `explored`; equal receipt JSON except the four
  backend fields, every `SearchStats` counter (`enumeratorPulls`, `generatedCandidates`,
  `peakActiveFrontierDepth`, the kernel-pull counts), the `triggers` evidence, and, for a `verify`
  that found a counterexample, `validity.searchComplete`, `searchTermination`, and `coverage`.
  Nothing else is exempt. (Amended 2026-09-27 by fn-88.9: the receipt's `explored` object is the
  same `ExploredCounts` as the Plan's exempt `explored`, and under `veil` its `traces` count product
  states by design (Data shapes), so it is exempt in the receipt too; each run's receipt `explored`
  must still equal its own Plan's.)
- Determinism: identical `CheckedQuery` and `Limits` produce identical Plan bytes, receipt JSON,
  and witness on one machine and across machines (PLN-02).
- The Veil dependency is a normal Lake requirement of `model/`. `make umpire-build-model`,
  `make lint-model`, and `make umpire-check-regression` build it. No opt-in target.

## Quick commands

```bash
cd model && lake build Umpire.Search Umpire.Search.Product Umpire.Search.Selection
cd model && lake build Umpire.Search.Tests Umpire.Search.VisibilityTests
make umpire-check-goldens
LEAN_NUM_THREADS=1 make lint-model
make umpire-check-regression   # final gate
```

## Edge Cases & Constraints
<!-- scope: technical -->

- **Three probe decisions.** `adopt` requires all of: the checker closure builds unchanged under
  `model/lean-toolchain` inside a copy of `model/lakefile.lean` plus the `require`; the closure
  contains no Loom, lean-smt, mathlib, or widget module; the checker entry is a pure or
  fuel-bounded function, not `IO`; the definitions the R7 theorems unfold are `@[expose]` under
  Veil's module system. A build that succeeds but fails any other condition is `defer-closure`. A
  compiler or Lake error is `defer-incompatible`. Either defer closes every later task as not
  applicable, each task closing itself with the R1 receipt identity as evidence (R13).
- **Amended adopt conditions (2026-09-27, R22).** R1's receipt was `defer-incompatible` (Veil
  declares Lean 4.32.0; its proofs fail under 4.33.1) and found the checker entry `findReachable`
  in `IO` with an unbounded loop. Because Veil removes a large amount of checker code Umpire would
  otherwise own, the decision is re-taken under two amendments: the model's toolchain may move to
  Veil's declared toolchain, and the checker entry may be `IO` when Umpire runs it during command
  elaboration, every witness it returns passes the R10 kernel replay gate, and an absence answer is
  recorded as trusted from the checker, with task .9's differential test as its oracle
  on the pinned models. Veil's 64-bit state-hash deduplication can merge distinct states; the
  receipt records it and `AUTHORING.md` states it as the trust assumption of a `veil` absence
  answer. The other `adopt` conditions stand. Tasks after .1 wait on R22 instead of closing under
  R13; a `defer-*` R22 decision closes them under R13 with the R22 receipt.
- **As built (fn-88.5, 2026-09-27).** The adapter does not call the `IO` entry `findReachable`: it
  drives Veil's pure one-step function `bfsStep` itself, at most `Limits.search` steps, so
  `AdmittedQuery.search` stays pure and nothing runs during elaboration. Visited states are
  compared as whole product states, not by Veil's 64-bit hash, so the hash-collision trust
  assumption above does not apply; a `veil` absence answer rests on the adapter's equivalence
  theorems and the differential test.
- **Selecting roots is not vendoring.** Lake building only the imported modules of an unmodified
  pinned checkout is allowed. Patching, copying, or forking any Veil source is not.
- **Nondeterministic frontier order.** If R1 shows the checker iterates a hash map or runs in
  parallel, the adapter still meets R15; the probe records the facts the adapter task needs.
- **Unsupported clause, Scenario construct, strategy, or form.** The Query runs on `reference`
  with the reason recorded; not an error.
- **Exploration and Replay digests.** Both are `artifactChecksum` of the Plan. `explored` differs
  between backends, so Plan bytes for Queries that move to `veil` change once at cutover (R18).
  R19 requires ledger credit and Replay keys to compare by outcome and witness, never by `explored`
  or the checksum alone.
- **Two backends disagree** where both terminate: the differential test fails the build. No vote.
- **Build cost.** R1 records `lake update` seconds and bytes, cold and warm build seconds, peak RSS
  of the closure, and complete-mode `lint-model` time. `.github/workflows/umpire.yml` builds Lean
  from a cold `.lake` inside 30 and 40 minute job timeouts; R20 requires the cold build to fit them
  or adds a `.lake` cache in the same task.
- **Rollback (R21).** The drill's permitted diff is closed: the `require` and its manifest entry;
  `Umpire.Search.Backend.Veil`; `Umpire.Search.Selection` reduced to always choosing `reference`;
  veil-only tests, pins, and the differential test's veil arm; the lint rule and its controlled
  violation; the R18 goldens flipping back. Anything else in the diff fails the drill.
- **Proof trust.** Adapter theorems print only `propext`, `Quot.sound`, `Classical.choice`;
  `native_decide` where the repository already allows it and nowhere new.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A compatibility probe builds `Veil.Core.Tools.ModelChecker.Concrete.Checker` and its
  transitive imports at one pinned commit, unchanged, under `model/lean-toolchain`, in a temporary
  copy of `model/` whose `lakefile.lean` has the one added `require`, and appends a receipt to
  `experiments/umpire-dsl/VEIL_RESULTS.md` naming: the commit, Veil's declared toolchain, the
  imported module closure and whether Loom, lean-smt, mathlib, or widget modules are in it, the
  Batteries revision Veil resolves against the model's `v4.33.0`, `lake update` seconds and bytes,
  cold and warm build seconds, peak RSS, complete-mode `lint-model` seconds in the copy, whether
  the checker entry is pure, `partial`, or `IO`, whether its frontier order is deterministic and
  single-threaded, whether the definitions R7 unfolds are `@[expose]`, and the decision `adopt`,
  `defer-closure`, or `defer-incompatible`. Errors: a compiler or Lake error is
  `defer-incompatible` with the first error message; a successful build failing any other `adopt`
  condition is `defer-closure`; a network or disk failure is retried, never recorded as a decision.
- **R2:** In `adopt` mode `model/lakefile.lean` requires Veil at exactly the R1 commit,
  `make umpire-build-model` succeeds from a clean `.lake`, and `make umpire-check-regression` fails
  when `lake-manifest.json` resolves a different Veil commit. Errors: no error surface beyond that
  check.
- **R3:** `make lint-model` enforces `search-backend-isolation` as a direct-import rule: a `Veil.*`
  import outside `Umpire.Search.Backend.Veil`, or an import of that module outside
  `Umpire.Search.Selection`, fails with a diagnostic naming the module and the import, pinned as a
  controlled violation the way existing rules are; the semantic-root rule still passes for
  `Umpire.Search`. Errors: no error surface beyond the lint failure.
- **R4:** `Umpire.Search` exposes `BackendResult`, `Backend.reference`, and
  `finalizeBackendResult`; `search` equals `finalizeBackendResult` of `Backend.reference`, and
  every existing search golden and fixture is byte-identical after the refactor except the receipt
  goldens whose format string moved to v2 and whose `backendPulls` key became `enumeratorPulls`.
  Errors: any other golden diff fails the build.
- **R5:** A Property monitor lowering exists for `stateInvariant`, `transitionContract`,
  `identityRelation`, `inputOutput`, `ordered`, `eventuallyWithin`, and `neverWithin` with Limits
  in `steps` or `actions`, each answering `PropertyEndpointAnswer` for terminal and partial
  endings (closed and partial) with the state and verdicts the Architecture section states; every other clause kind
  and any `logicalTime` Limit returns a typed `Unsupported` naming the kind. Errors: `Unsupported`
  is the whole error surface; no silent weakening.
- **R6:** For every clause kind R5 supports, the repository holds either a theorem that the
  monitor's answer on a trace equals `clauseEndpointAnswer` for that clause under both endings, or
  an exhaustive differential test over every trace of every checked-in model within its Limits,
  and the receipt records `kernel` or `testing` per clause kind. Errors: a supported kind with
  neither fails the build. (Amended 2026-09-27 by fn-88.9: a Model over several interleaved
  instances has too many traces within its Limits to enumerate -- the Pair has 28513 within four of
  its six steps -- so on the Temporal feature Models the evaluator comparison reads every Model
  trace to the deepest depth within a stated cost, and the test prints that depth per Query; every
  trace within the Limits is compared for the Umpire Models and the synthetic alphabet.)
- **R7:** The Veil adapter proves, over the product, that adapter transitions and product
  transitions coincide in both directions and that initial states agree, with axiom inventories
  limited to `propext`, `Quot.sound`, `Classical.choice`, pinned by `#guard_msgs in #print
  axioms`. Errors: any other axiom fails the pin.
- **R8:** A differential test runs every Query reachable through `AdmittedQuery.search` in the
  checked-in models through both backends via `AdmittedQuery.searchWith`. Where `reference`
  terminates with `found`, `verified-within-limits`, `none-found`, or `unsatisfiable`, the
  comparison defined in API Contracts holds. Where `reference` reports `limit-reached`, a `veil`
  witness must pass R10 and a `veil` complete result must not contradict any trace `reference` did
  examine. Errors: any difference fails the build with both traces printed.
- **R9:** On the Nexus caller protocol machine, `veil` completes exhaustive `verify` and the exact
  product-state count is pinned; on the Pair model, `veil` completes exhaustively within its
  declared Limits and the count is pinned; and a synthetic three-instance fixture product in the
  Search tests reports `limit-reached` on `reference` at `search = 32768` and `complete` on
  `veil`. Wall time is recorded, never asserted. Errors: `veil` reporting `limit-reached` on any of
  the three fails the golden.
- **R10:** `finalizeBackendResult` accepts a witness only when it passes kernel replay as defined
  in Architecture; a constructed adapter fault returning a non-replayable trace produces `invalid`
  with `QueryErrorKind.unreplayableWitness` and the trace in `offendingValue`, pinned as a negative
  control. Errors: no error surface beyond `invalid`.
- **R11:** Running the same Query twice on one machine and once in CI yields identical Plan bytes,
  receipt JSON, and witness. Errors: a difference fails `make umpire-check-regression`.
- **R12:** `Backend.reference` gains no new capability: the `#check` surface pin in the Search
  visibility tests covers the new public names, and a line count of `Search.lean` pinned by the
  last task that edits it (task .6) makes later growth a visible diff. Errors: no error surface beyond the pins.
- **R13:** In either defer mode, every task other than the probe closes as not applicable citing
  the R1 receipt identity, no `Veil.*` import, Lake requirement, or Make target exists in the
  tree, and a test proves those surfaces absent. Errors: any of them present fails the test.
- **R14:** The Scenario progress automaton accepts exactly the traces `CheckedScenario.admits`
  accepts for the version-one constructs, by theorem or by an exhaustive differential test over
  every checked-in Scenario within its Limits; over a free schedule `ordering` and `adjacencies`
  return a typed `Unsupported` naming the construct, and under a schedule `traceExactly` or
  `actionsExactly` fixes they are decided at construction. Errors: `Unsupported` is the whole
  error surface.
- **R15:** For every Query where both backends report `found`, the witness is the lexicographically
  least shortest trace in the `pullCandidate` key order, and a fixture product in which two paths
  reach one product state at the same depth proves dedup keeps the lexicographically smaller one.
  Errors: a differing witness fails R8.
- **R16:** Selection routes `strategy = seeded` to `reference` with the reason recorded, and a
  `veil` `complete` result becomes `verified-within-limits`, `none-found`, or `unsatisfiable` only
  under the same completeness-evidence and admitted-endpoint gates `finalizePlanning` applies
  today. Errors: no error surface beyond the reason field.
- **R17:** The planning receipt is `umpire-planning-receipt/v2` with `searchBackend`,
  `backendReason`, `searchUnit`, `veilCommit`, and `enumeratorPulls`; the Plan artifact codec and
  the Go `artifactv2` decoder are unchanged, and `boundWasHit` keeps its comparison with
  `explored.traces`. Errors: no error surface beyond the receipt goldens.
- **R18:** One reviewed commit re-pins exactly the goldens the cutover flips, each listed in the
  task with its old and new outcome, the list rebuilt from the callers of `AdmittedQuery.search`
  and `searchWithIntent`; every golden not on the list stays byte-identical. Errors: an unlisted
  golden change fails review.
- **R19:** Exploration ledger credit and Replay violation keys compare by outcome and witness,
  never by `explored` or `artifactChecksum` alone, so a backend fallback on one Query changes no
  ledger status or Replay key. Errors: a fallback that changes either fails the bridge tests.
- **R20:** The cold `lake build` of the model with Veil fits the existing `umpire.yml` job
  timeouts, measured in one CI run, or the same task adds a `.lake` cache step to that workflow.
  Errors: a timeout is a finding for GOV-02, not a silent widening.
  Measured 2026-09-28 (CI run 36393502944 on `stephanos/umpire` at 271889d3d7, cold `.lake`, Veil
  required, the runner's default Node): portability 24m39s of its 40-minute timeout, canary 17m03s
  of its 30-minute timeout; both passed, so no `.lake` cache or Node pin was added.
- **R21:** A rollback drill on a scratch branch produces exactly the permitted diff listed under
  Edge Cases and every Query returns to `reference`; the diff stat is recorded in the task
  evidence. Errors: any other file in the diff fails the drill.
- **R22:** A second probe re-runs R1 in a temporary copy of `model/` whose `lean-toolchain` is
  Veil's declared toolchain and whose Lean requirements (Batteries, protobuf, binary) move to
  revisions for it, and appends a receipt naming: the toolchain and every requirement revision, the
  model-side changes needed for the whole model to build and `make umpire-check-goldens` to pass on
  it (as a file list and a line count), whether the checker closure builds unchanged, the facts R1
  lists, and the decision under the amended adopt conditions (Edge Cases). An `adopt` is followed by
  a toolchain-alignment task that lands those model-side changes before task .2. Errors: as R1; a
  model file that cannot build on the older toolchain without changing a Property's meaning, a
  fingerprint, or a golden is `defer-incompatible` naming the file.

## Boundaries
<!-- scope: business -->

- No Veil DSL, `veil_decl`, `action`, or relational authoring. Authors keep the commands.
- No SMT, `#check_invariants`, `bmc`, inductive-invariant proof, or parametric claim. That is
  fn-23, fn-24, and fn-25.
- No `#simulate`, seeded exploration, or change to the Exploration walker.
- No symmetry reduction or instance canonicalization.
- No monitors for `branches`, guarded clauses, correlated clauses, or logical-time Limits; no
  Scenario automaton for `ordering` or `adjacencies` over a free schedule.
- No change to the Case Producer's lowering, Case production, Testpilot, Contracts, Run evaluation,
  the Plan artifact format, or Go code.
- No `FiniteTable` to TLA+ exporter. If R1 defers, that exporter is the next spec.
- The fn-23 sandboxed gate is not run, amended, or closed here.
- No Veil source patching, vendoring, or fork.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

The team wants the best checking result while owning the least checker code, and wants the
compile step from author's code to checked rows to stay provable. Lean as the frontend and the
explicit table as the IR satisfy the second; delegating search to a library satisfies the first.
Veil was chosen over TLC and Stateright because it is the only mature engine in Lean: no second
toolchain, no serialization, counterexamples as Lean values, kernel replay as the trust boundary.
TLC is the fallback if Veil does not build. Stateright's last release was July 2025 with one
maintainer. A FizzBee-style checker of our own was rejected as the largest body of code we could
own.

### Implementation Tradeoffs
<!-- scope: technical -->

- **Seam inside `Umpire.Search`.** The traversal, `finalizePlanning`, `observeCandidate`, and the
  `PlanningResult` constructor are private to `Search.lean`. Putting `BackendResult`,
  `Backend.reference`, and `finalizeBackendResult` in that file is the only way to reuse them
  without opening the module, and it keeps `Umpire.Search` free of Veil.
- **Depth exhaustion is `complete`.** The reference returns `.complete` when iterative deepening
  reaches `maximumDepth`; mapping Veil's depth bound to `limit-reached` would make every such
  Query differ between backends. Only the state bound is a Limit hit.
- **Library, not DSL.** Veil carries no theorem that its DSL-extracted executable actions agree
  with its relational semantics. Feeding our own `SearchView` into Veil's
  `EnumerableTransitionSystem` sidesteps that gap; the adapter's equivalence theorems were proved
  once already in the probe.
- **Product with three-valued monitors.** Path enumeration is the single design choice that forced
  a bespoke checker. Monitors must answer `unresolved` because `stillPending` and `find` depend on
  it, and their shapes are read off the evaluator, not invented per clause name. Version one covers
  the seven kinds whose monitors are bits or a bounded counter.
- **Reference backend kept and frozen.** It is the oracle and the fallback; R12 makes growth
  visible.
- **Receipt fields, not Plan fields.** Backend fields in the Plan artifact would change every
  Plan's bytes, its checksum, the Go decoder, and fn-60's byte-identity requirement. The receipt
  is Lean-owned and versioned; `explored` keeps its names and changes value under `veil`, which is
  the one-time re-pin R18 owns.
- **Rules to amend under GOV-02**, drafted in their own task before the dependency lands, with the
  `*(drafted by fn-88; awaiting GOV-02 approval.)*` marker: `UMPIRE4_SPEC_MODEL_ARCH.md` §2
  principle 7, §3 module tree and MOD list, §9 (the two sentences, the diagram, the "Generic Veil
  mechanics" paragraph), §10 diagnostic, §11 build-gate sentence, §13 criterion 5, §14 non-goal;
  `UMPIRE4_DSL.md` "Optional Veil checking"; `UMPIRE4_SPEC_COMPS.md` §3 principle 9, the
  rejected-designs line, §6.4 row and paragraph on `Umpire.Search`, §10 non-goal;
  `UMPIRE4_SPEC.md` glossary entries for Search, SearchStats, PlanResult, Exhaustive Search,
  `Umpire.Verify.Veil`, MOD-05 and MOD-11 amendment adding the rule, VER-05 and VER-06
  co-attribution. VER-01 is satisfied as written.
- **fn-23 conflict.** fn-23 says `Umpire.lean` must remain free of Veil and the model is
  dependency-free; both are contradicted here and the second is already stale. R1 supersedes
  fn-23's purpose for concrete checking; the symbolic-path decision stays fn-23's. Re-scoping fn-23
  is a separate edit the user approves.
- **Rejected:** vendoring checker files (a fork we would own); exporting to TLC first (a second
  toolchain before the in-language option is tried); calling Veil's `#model_check` command
  (elaborator-owned); a second Lake project (default backend unavailable by default); backend
  fields in the Plan artifact; mapping depth exhaustion to `limit-reached`.

## Early proof point

Task fn-88-veil-concrete-checker-as-the-umpire.1 validates the core approach: Veil's checker
library builds unchanged inside the model's own dependency graph with a pure entry, exposed
definitions, and a closure free of Loom, lean-smt, mathlib, and widgets. If it returns either
defer, stop, close the remaining tasks under R13, and open the `FiniteTable` to TLA+ exporter spec
instead of retrying with patches.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
|-----|-------------|---------|-------------------|
| R1  | Compatibility probe and receipt | .1 | — |
| R2  | Lake requirement and manifest pin check | .5 | — |
| R3  | `search-backend-isolation` lint rule | .5 | — |
| R4  | Backend seam inside `Umpire.Search`, byte-identical goldens | .2 | — |
| R5  | Property monitor lowering v1 | .4 | — |
| R6  | Monitor agrees with `clauseEndpointAnswer` | .4 | — |
| R7  | Adapter equivalence theorems and axiom pins | .5 | — |
| R8  | Differential test across backends | .9 | — |
| R9  | Caller, Pair, and three-instance pins | .9 | — |
| R10 | Kernel replay gate and negative control | .6 | — |
| R11 | Determinism on one machine and in CI | .10 | — |
| R12 | Reference frozen: surface and line pins | .2, .6 | — |
| R13 | Defer-branch closeout and absence test | .7 | — |
| R14 | Scenario progress automaton agrees with `admits` | .3 | — |
| R15 | Witness order preserved | .9 | — |
| R16 | Seeded routes to reference; finalization gates kept | .6 | — |
| R17 | Receipt v2 fields, Plan codec and Go untouched | .2 | — |
| R18 | One-commit golden re-pin list | .10 | — |
| R19 | Exploration and Replay keys ignore `explored` | .10 | — |
| R20 | CI cold-build budget | .5 | — |
| R21 | Rollback drill | .7 | — |
| —   | GOV-02 amendment drafts before the dependency lands | .8 | supports R2 and R3 ordering |

## References

- `.plans/UMPIRE4_DIRECTION.md` sections 1 and 6; `.plans/VEIL_BACKEND_RESEARCH.md`
- `experiments/umpire-dsl/VEIL_RESULTS.md`, `experiments/umpire-dsl/veil/Main.lean`
- `model/Umpire/Search.lean`, `model/Umpire/Search/Admission.lean`, `model/Umpire/Scenario/Check.lean`
- `model/Umpire/Property/Check.lean`, `model/Umpire/Property/Evaluate.lean`
- `model/ModelLint/ImportGraph.lean`, `model/lakefile.lean`, `model/lake-manifest.json`
- https://github.com/verse-lab/veil
