# Veil concrete checker as the Umpire search engine

## Umpire4 architecture reconciliation

This spec makes Veil's concrete model-checker library a search engine behind `Umpire.Search`,
consuming the explicit `FiniteTable` the `machine` command already produces. It adopts no Veil
DSL, no SMT path, and no Veil-authored semantics. The authoring commands, `CheckedModel`, the
Property evaluator and its denotation proof, Case production, Testpilot, and every artifact format
are unchanged. `Umpire.Search` stays the only module that names the engine; a `ModelLint` rule
confines the Veil import to one adapter module. It is separate from the optional symbolic
checker slot (`Umpire.Verify.Veil`, fn-23, fn-24, fn-25), which remains as specified and is
neither started nor closed by this spec. The direction note
`.plans/UMPIRE4_DIRECTION.md`, sections 1 and 6, records the reasoning this spec executes.

Two rules need a GOV-02 decision and are listed under Decision Context: the sentence in
`UMPIRE4_SPEC_MODEL_ARCH.md` section 9 that keeps Veil out of the normal model build, and the
sentence that forbids a checker-neutral semantic IR, which `FiniteTable` already is.

## Goal & Context
<!-- scope: business -->

`Umpire.Search` is a 1.2k-line hand-written checker: iterative-deepening depth-first search over
traces with no visited-state set, capped by a candidate count. It revisits every state once per
path, so the multi-instance models fn-85 introduces (two interleaved operations today, five
planned) reach `limit-reached` on questions a visited-set search answers in milliseconds. The
roadmap's next features, symmetry canonicalization, implicit fault placement, seeded walks, are a
model checker's feature list, and building them into `Search.lean` means owning a fourth explicit
checker beside TLC, Stateright, and Veil.

The team wants to own as little checker code as possible while keeping Lean as the frontend,
because the `machine` command's enumeration into `FiniteTable` and the `Machine` sound/complete
proofs are what make the compile step from author's code to checked rows trustworthy. Veil's
concrete checker is the only mature explicit-state engine written in Lean. Consuming it as a
library keeps counterexamples as Lean values, needs no serialization, and lets the existing
kernel replay gate (VER-05) stand as the trust boundary.

The 2026-09-06 probe under `experiments/umpire-dsl/veil` already showed the data adapter works:
it imported Veil's `EnumerableTransitionSystem` and `RelationalTransitionSystem` unchanged, proved
transition and initial-state equivalence in both directions with only `propext` and `Quot.sound`,
and replayed 335 paths. What it never ran is Veil's breadth-first checker, blocked by the Lean
version skew and a full disk. This spec finishes that run and, if the checker builds, makes it
the default engine for the Query forms it supports.

Who benefits: a model author whose exhaustive `verify` on a two-instance model completes instead
of reporting `limit-reached`, and reviewers who diff one adapter instead of a growing checker.

## Architecture & Data Models
<!-- scope: technical -->

```text
machine command ──enumerates──▶ CheckedTable (FiniteTable + Machine proofs)      unchanged
Property, Scenario ──lower──▶ Monitor automata (shared with Contract lowering)   new lowering reuse
                                        │
                                        ▼
                    Product = model state × scenario progress × monitor states
                                        │
                        ┌───────────────┴───────────────┐
                        ▼                               ▼
              Engine.reference                  Engine.veil
              (today's Search.lean, frozen)     (adapter → Veil concrete BFS)
                        │                               │
                        └───────────────┬───────────────┘
                                        ▼
                     EngineResult → Exact Replay through the kernel → PlanResult
                                        │
                                        ▼
                    Producer, Cases, fixtures, receipts                         unchanged
```

Components:

- **`Umpire.Search.Engine`** (new, deep module). One small interface: given a `Product` and
  `Limits`, return an `EngineResult`. Two implementations, `reference` and `veil`. Nothing else in
  `Umpire` sees which engine ran except through the receipt field. Testable with fixture products
  and no Temporal model (MOD-08).
- **`Umpire.Search.Product`** (new). Builds the product state space from a `CheckedTable`, the
  Scenario's admitted-prefix automaton (today's `Scenario/Check.lean` logic, reused), and the
  Property monitors. Product transitions are labeled with the model `Step` so a product path
  decodes to a `Umpire.Scenario.Trace` without loss.
- **Monitor lowering.** The lowering from a `CheckedProperty` to per-rule monitors that the Case
  Producer already performs for Contracts is extracted so Search and the Producer call one
  function. A clause kind the lowering does not support is a typed `Unsupported` value, never a
  silently weakened monitor.
- **`Umpire.Search.Engine.Veil`** (new). The only module that imports Veil. Wraps the product as a
  Veil `EnumerableTransitionSystem`, runs the concrete checker with depth and state bounds derived
  from `Limits`, and decodes the result. Carries the equivalence theorems from the probe, restated
  over the product: every adapter transition is a product transition and vice versa, and initial
  states agree.
- **Veil dependency.** A Lake `require` on `verse-lab/veil` at one pinned commit, importing only
  the `Veil.Core.Tools.ModelChecker` module tree. The commit, its Lean toolchain declaration, and
  the transitive closure of the imported modules are recorded in the compatibility receipt (R1).
  The primary Lake project gains this one dependency; no widget, Loom, or lean-smt module is
  imported.
- **`ModelLint` rule `search-engine-isolation`.** Only `Umpire.Search.Engine.Veil` may import a
  `Veil.*` module. Enforced by `make lint-model` beside the existing reachability rules (MOD-11).
- **`Search.lean` frozen.** The current traversal becomes `Engine.reference`. It gains no features.
  Its role is differential testing and the fallback for unsupported Property clauses.

Data shapes:

- `EngineResult`: one of `violationFound (trace : List Step) (depth : Nat)`, `complete`
  (every reachable product state visited, no violation), `depthBound (reached : Nat)`,
  `stateBound (visited : Nat)`, `cancelled`. Plus `SearchStats` with states visited, transitions
  taken, and the engine name.
- `Engine` receipt fields added to `PlanResult` and every Artifact that carries a `PlanResult`:
  `engine ∈ {reference, veil}`, `engineReason ∈ {default, unsupported-clause:<kind>,
  unsupported-form:<form>}`, `veilCommit` (present only when `engine = veil`).
- Mapping to `Umpire.PlanningOutcome`: `violationFound` → `found` (for `find`, `pick`) or
  `findViolation`'s found; `complete` → `verified-within-limits` or `none-found`; `depthBound` and
  `stateBound` → `limit-reached`; a product with no admitted initial state → `unsatisfiable`;
  `cancelled` → `limit-reached` with the cancellation recorded in stats. `never-triggered` and
  `still-pending` are computed from the monitor terminal states exactly as today.

## API Contracts
<!-- scope: technical -->

- `Umpire.Search.answer (query : CheckedQuery) : PlanResult` keeps its signature. Engine
  selection is internal: `veil` when the Property lowers completely and the Query form is `verify`,
  `find`, `findViolation`, or `pick`; `reference` otherwise, with `engineReason` set. No CLI flag,
  Query field, or environment variable selects the engine (CLI-02).
- `Limits` mapping: `steps` → maximum product depth; `search` → maximum product states visited;
  `actions` → enforced by the Scenario automaton as today; `logicalTime` and `plans` unchanged.
  Which Limit stopped the engine is named in the receipt (PLN-01, PLN-04).
- Every `violationFound` trace is decoded to `Umpire.Scenario.Trace` and replayed through
  `Machine` and the Property evaluator before `PlanResult` is built. A trace that fails replay is
  an `invalid` outcome carrying the engine's trace and the replay diagnostic; it is never reported
  as `found` (VER-05).
- Determinism: identical `CheckedQuery` and `Limits` produce identical `PlanResult` bytes and
  identical witness traces (PLN-02). The adapter fixes the order in which the product's successors
  are handed to the engine to the table order.
- The Veil dependency is a normal Lake requirement of `model/`. `make umpire-build-model`,
  `make lint-model`, and `make umpire-check-regression` build it. No opt-in target.

## Edge Cases & Constraints
<!-- scope: technical -->

- **Veil does not build unchanged under the repository toolchain.** Task 1 records
  `defer-incompatible` with the exact commit, toolchain, and the first compiler error, and every
  later task closes as not applicable with that receipt. No Veil source is patched, vendored, or
  forked. The follow-up is a separate spec for a `FiniteTable` to TLA+ exporter, named in
  Boundaries.
- **Veil's checker visits states in an order that differs between runs.** The adapter must still
  return the same witness. If the probe shows nondeterministic order, the adapter selects among
  shortest violating traces by table order; how is a task decision, but the R-ID holds
  regardless.
- **A Property clause has no monitor lowering.** The Query runs on `reference` with
  `engineReason = unsupported-clause:<kind>`; the outcome vocabulary is unchanged. This is not an
  error.
- **The product is larger than the model.** A monitor with k states multiplies the space by at
  most k per rule. Limits apply to the product, and the receipt reports product states, not model
  states, so a `limit-reached` is honest about what was searched.
- **A monitor lowering disagrees with the evaluator.** Caught by R6: for every lowered clause kind
  there is a theorem or an exhaustive differential test over the checked-in models within their
  Limits, and the receipt records which (`kernel` or `testing`, VER-06 vocabulary).
- **Two engines disagree.** The differential test in R8 fails the build. There is no
  majority vote and no preference for the faster answer.
- **Cancellation.** A cancelled engine run is `limit-reached`, never `none-found`.
- **Build cost.** The Veil module tree adds to the cold `lake build`. R1 records cold and warm
  build time of the imported closure; a cold cost above five minutes on the reference developer
  machine is a finding for GOV-02, not a silent acceptance.
- **Proof trust.** Adapter theorems must print only `propext`, `Quot.sound`, and `Classical.choice`
  in their axiom inventories; `native_decide` is allowed where the repository already allows it
  at the checked seam and nowhere new.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** A compatibility probe builds `Veil.Core.Tools.ModelChecker.Concrete.Checker` and its
  transitive imports, at one pinned commit, unchanged, under `model/lean-toolchain`, in a
  temporary Lake project on a developer machine, and writes a receipt naming the commit, Veil's
  declared toolchain, the imported module closure, whether any Loom, lean-smt, mathlib, or widget
  module is in that closure, cold and warm build seconds, and the decision `adopt` or
  `defer-incompatible`. Errors: a compiler or Lake error is `defer-incompatible` with the first
  error message; a network or disk failure is an infrastructure failure, distinct from both
  decisions and retried, never recorded as a decision.
- **R2:** In `adopt` mode, `model/lakefile.lean` requires Veil at exactly that commit and
  `make umpire-build-model` succeeds from a clean `.lake`. Errors: a manifest that resolves a
  different commit fails `make umpire-check-regression`.
- **R3:** `make lint-model` enforces `search-engine-isolation`: any `Veil.*` import outside
  `Umpire.Search.Engine.Veil` fails with a diagnostic naming the module and the import. Errors:
  no error surface beyond the lint failure.
- **R4:** `Umpire.Search.Engine` exposes `reference` and `veil`, and the current `Search.lean`
  traversal is `reference` with no behavior change: every existing search golden and fixture is
  byte-identical after the refactor. Errors: a golden diff fails the build.
- **R5:** Property and Scenario lower to a product automaton through the same monitor lowering
  the Case Producer uses; the Producer calls the shared function and its Case fixtures are
  byte-identical. Errors: a clause kind without lowering returns a typed `Unsupported` value
  naming the kind; no silent weakening.
- **R6:** For every clause kind the lowering supports, the repository holds either a theorem that
  the monitor's terminal verdict on a trace equals the Property evaluator's verdict, or an
  exhaustive differential test over every trace of every checked-in model within its Limits, and
  the engine receipt records `kernel` or `testing` per clause kind. Errors: a supported clause kind
  with neither fails the build.
- **R7:** The Veil adapter proves, over the product, that adapter transitions and product
  transitions coincide in both directions and that initial states agree, with axiom inventories
  limited to `propext`, `Quot.sound`, `Classical.choice`. Errors: any other axiom fails a pinned
  `#guard_msgs` test.
- **R8:** A differential test runs every checked-in Query through both engines and requires
  identical `PlanningOutcome`, identical witness trace, and identical `PlanResult` bytes except
  the `engine`, `engineReason`, `veilCommit`, and `SearchStats` fields. Errors: any difference
  fails the build with both traces printed.
- **R9:** On the Nexus caller protocol machine, `veil` completes exhaustive `verify` visiting at
  most the model's 158 reachable states times the product factor, and on the Pair model it
  completes exhaustively where `reference` reports `limit-reached` at the largest checked-in
  `search` Limit; both facts are pinned as goldens with wall time recorded but not asserted.
  Errors: `veil` reporting `limit-reached` on either fails the golden.
- **R10:** Every `violationFound` trace passes Exact Replay before it becomes `found`; a
  constructed adapter fault that returns a non-replayable trace produces `invalid` with the
  replay diagnostic, pinned as a negative control. Errors: no error surface beyond `invalid`.
- **R11:** Running the same Query twice, and on two machines in CI, yields identical `PlanResult`
  bytes and witness traces. Errors: a difference fails `make umpire-check-regression`.
- **R12:** `Engine.reference` gains no new capability in this spec; a lint or test pins its module
  line count and public surface so a later addition is a visible diff. Errors: no error surface
  beyond the pin.
- **R13:** In `defer-incompatible` mode, R2 through R12 close as not applicable citing the R1
  receipt identity, no `Veil.*` import, Lake requirement, or Make target exists in the tree, and
  a test proves those surfaces absent. Errors: any of them present fails the test.

## Boundaries
<!-- scope: business -->

- No Veil DSL, `veil_decl`, `action`, or relational authoring. Authors keep the `machine`,
  `property`, `scenario`, `limits`, `query`, and `set` commands.
- No SMT, `#check_invariants`, bounded-model-checking `bmc`, inductive-invariant proof, or
  parametric claim. That is fn-23, fn-24, and fn-25, unchanged.
- No `#simulate` or seeded exploration. Exploration (fn-33) keeps its walker.
- No symmetry reduction or instance canonicalization. That is a follow-up once fn-85's instances
  land, and it may live in the engine or the product.
- No change to Case production, Testpilot, Contracts, Run evaluation, receipts other than the new
  engine fields, or any Go code.
- No `FiniteTable` to TLA+ exporter. If R1 is `defer-incompatible`, that exporter is the next spec
  and TLC becomes the engine candidate.
- The fn-23 sandboxed compatibility gate is not run, amended, or closed. R1 is a narrower probe
  for a narrower purpose.
- No Veil source patching, vendoring, or fork.

## Decision Context
<!-- scope: both -->

### Motivation
<!-- scope: business -->

The team wants the best checking result while owning the least checker code, and wants the
compile step from author's code to checked rows to stay provable. Keeping Lean as the frontend
and the explicit table as the IR satisfies the second; delegating search to a library satisfies
the first. Veil was chosen over TLC and Stateright because it is the only mature engine in Lean:
no second toolchain in CI, no serialization of the table, counterexamples as Lean values, and the
existing kernel replay gate as the trust boundary. TLC scores higher on support and expressiveness
and is the fallback if Veil does not build. Stateright's last release was July 2025 with one
maintainer. Building a FizzBee-style checker of our own was rejected as the largest body of code
we could own.

### Implementation Tradeoffs
<!-- scope: technical -->

- **Library, not DSL.** Veil carries no theorem that its DSL-extracted executable actions agree
  with its relational semantics. Feeding our own `FiniteTable`, which has `Machine` sound and
  complete proofs, into Veil's `EnumerableTransitionSystem` sidesteps that gap; the adapter's
  own equivalence theorems (R7) are small and were already proved once in the probe.
- **Product with monitors, not trace evaluation.** Path enumeration is the single design choice
  that forced a bespoke checker. Lowering Properties to monitors lets any visited-set engine
  check them and reuses the lowering the Producer already performs. The cost is R6: proving or
  exhaustively testing that each monitor agrees with the evaluator.
- **Reference engine kept.** Deleting `Search.lean` would remove the differential oracle and the
  fallback for clauses without a lowering. Freezing it costs nothing and R12 makes growth visible.
- **One dependency in the primary Lake project.** The alternative, an opt-in second Lake project,
  would keep the default build Veil-free but make the default engine unavailable by default,
  which defeats the purpose. The concession is the `search-engine-isolation` lint and the closure
  inventory in R1, so the dependency stays one import wide.
- **Rules to amend under GOV-02.** `UMPIRE4_SPEC_MODEL_ARCH.md` section 9: "Veil is not part of
  `Plan`, runtime execution, evidence interpretation, production binaries, or the normal Temporal
  model build" becomes "Veil's symbolic path is not part of ..." with the concrete checker
  library named as a Search engine; and "Umpire does not generate Veil source or introduce a
  checker-neutral semantic IR" is restated to name `FiniteTable` as the IR every engine consumes.
  VER-01 (Lean-native default) is satisfied as written: the engine is executable Lean.
- **Rejected:** vendoring the two or three checker files to dodge the version skew (a fork we
  would own); exporting the table to TLC first (a second toolchain before the in-language option
  is tried); adopting Veil's `#model_check` command (elaborator-owned, binds us to frontend
  internals).

## Parked unknowns

- Whether Veil's concrete BFS visits successors in a deterministic order. R1's probe records it;
  if not, the adapter's witness selection carries the determinism.
- Whether `Veil.Core.Tools.ModelChecker.Concrete.Checker`'s transitive closure at the chosen
  commit is free of Loom, lean-smt, and widget modules after Veil's 2026-09-22 move to the Lean
  module system. R1 answers it.
- Whether the module names above survived that migration. R1 pins whatever the commit calls them.
