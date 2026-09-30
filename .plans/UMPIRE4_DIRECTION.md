# Umpire 4 direction: Veil, Specula, tracing, and FizzBee

Assessment note, 2026-09-26. It answers six questions about the `umpire4` prototype under `model/`
and the surrounding Go runtime: whether Veil should be its symbolic core, how much of it Veil can
replace, how it relates to Specula, whether the tracing meets the vision, whether it beats
FizzBee on value and usability, and whether it is reimplementing a model checker. It is descriptive and proposes an order of work; it
changes no rule and approves no design. Repository facts come from reading the tree on the date
above. External facts come from the public Veil, Specula, and FizzBee repositories and papers on
the same date and are cited inline. Nothing was installed, built, or run except the plan-index
check.

## Answers in one place

1. **Veil as a symbolic core for the current finite-table path: no.** Umpire materializes the
   transition table before Search, so adding SMT after that point avoids no enumeration. Take Veil's
   concrete BFS checker and simulator, and keep SMT as an opt-in for parametric invariants or for a
   future model representation that does not require complete materialization.
2. **What Veil can replace:** the checker core, about 8 to 10k of Umpire's 56k Lean lines. None of
   the Case, Evidence, Testpilot, or Go runtime half. The larger saving is work not yet built.
3. **Specula** runs the opposite direction on both axes. It derives the spec from the code and
   validates recorded traces against it. Borrow its trace-validation harness and agent-drafted
   instrumentation plan, never its authority model.
4. **Tracing is sufficient for canary and functional regression and insufficient for the white-box,
   fault, and exploration goals** in [UMPIRE4_VISION](UMPIRE4_VISION.md).
5. **FizzBee wins on usability today.** Umpire's value proposition is a regression and canary gate
   against a real Temporal with auditable claims, which FizzBee is not. Five fixes below.
6. **Umpire is partly reimplementing a model checker**, and the roadmap adds the rest of one.
   Freeze the search, lower Properties to monitors, and hand the product to an existing engine.

## 1. Veil under the hood

### What the checker does today

[`Umpire/Search.lean`](../model/lean/Umpire/Search.lean) runs iterative-deepening depth-first search
over paths of a finite table. It keeps a cursor path and no visited-state set, prunes by the
Scenario's admitted prefixes, and stops at the `search` Limit, a candidate count. The `seeded`
strategy rotates enumeration order by a fixed offset. The table comes from
[`Umpire/Model/Table.lean`](../model/lean/Umpire/Model/Table.lean): the `machine` command runs every
step function on every state and action pair and stores the rows, and
[`Umpire/Core.lean`](../model/lean/Umpire/Core.lean) carries the `Machine` proofs that the rows are
exactly the declared relation. States are structures of enums, `Bool`, and saturating `Fin`
counters ([`Umpire/Command/Finite.lean`](../model/lean/Umpire/Command/Finite.lean)).

The largest model is the Nexus caller protocol machine
([`Caller/Model.lean`](../model/lean/Temporal/Feature/Nexus/Caller/Model.lean)). Its numbers, pinned in
[`Caller/Tests.lean`](../model/lean/Temporal/Feature/Nexus/Caller/Tests.lean):

| Quantity | Value |
| --- | ---: |
| States | 192 |
| Transitions | 1152 |
| Reachable states | 158 |
| Exploration coverage targets | 889 |
| Search caps | 512, 4096, 32768 candidates |

### Why symbolic checking is not the default here

There are three different mechanisms in play. Umpire's current Search enumerates concrete paths.
Veil's concrete checker enumerates concrete states with breadth-first search and a visited set.
Veil's symbolic path instead represents states and transitions as formulas: bounded model checking
asks an SMT solver for a satisfying trace of a fixed depth, while inductive verification asks it to
discharge initiation and preservation obligations for an invariant. The latter two are the
"symbolic" mechanisms discussed in this section; Veil's concrete checker is still a model checker,
not a Run against Temporal.

The reason symbolic checking does not pay on Umpire's current path is architectural, not merely that
the models are finite. Finite systems can be large enough for symbolic encodings to help. Umpire,
however, admits a Model by enumerating its complete state and action domains and materializing the
transition rows used by Behavior Fingerprints, Cases, coverage targets, and exhaustive claims.
Target admission therefore incurs the enumeration before Search chooses a backend
([VEIL_BACKEND_RESEARCH](VEIL_BACKEND_RESEARCH.md#the-exact-umpire-seam)). An SMT backend attached
after admission would encode a graph the pipeline has already built; it would not avoid building
that graph. Avoiding materialization would require changing the admission and fingerprinting
contract, not replacing only the planner.

The checker's actual weakness is path enumeration without state dedup. Multi-instance models
([fn-85](../.flow/specs/fn-85-model-side-effects-as-typed-actions-and.md) plans a five-operation
Query) multiply the state space and the DFS revisits every state once per path. Veil's concrete
checker is a BFS over an `EnumerableTransitionSystem` with counterexample trace reconstruction,
and Veil now compiles model checks and random `#simulate` walks through emitted C
([verse-lab/veil](https://github.com/verse-lab/veil), commits of 2026-09-13 and 2026-09-18). Those
are the pieces to take under the hood. SMT bounded model checking and inductive-invariant proof
belong in the opt-in `Umpire.Verify.Veil` slot that [UMPIRE4_SPEC](UMPIRE4_SPEC.md) reserves under
VER-02 and VER-06, for claims the finite table cannot make, such as "for any number of operations".

### Benefit by benefit

Symbolic checking earns most when it avoids explicit enumeration or proves a claim for a family of
systems at once. The `machine` command currently enumerates the full table at elaboration, so the
first saving is gone before checking starts. The second remains valuable for opted-in parametric
claims that the finite table does not express.

| Benefit of SMT-based checking | Umpire's situation |
| --- | --- |
| Reason about unbounded sorts and counters for all sizes at once | Current state is a structure of enums and saturating `Fin` counters. A claim over an arbitrary number of entities needs a separate parametric representation and proof. |
| Avoid materializing a state space too large to enumerate | Current admission has already materialized the table. Obtaining this benefit requires a different admission and fingerprinting path that keeps the relation symbolic. |
| Bounded model checking finds shallow bugs by unrolling k steps | On the current 158-reachable-state model, explicit reachability covers every depth and gives a stronger completeness result. Bounded checking can still be a useful bug finder for a much larger model that is not first expanded into a table. |
| Inductive invariants give unbounded-depth guarantees | This is useful for parametric claims, but the author must supply or discover a sufficiently strong invariant. That is the bottleneck in Ivy and Veil, which is why Veil's own Raft port is model-checked with 3 servers rather than proved. |

Three further mismatches:

- **Model shape.** Symbolic methods shine on relational models with quantifiers inside the EPR
  fragment, the Ivy style. Umpire's steps are executable functions over records that return
  successor lists, a TLC-shaped model. Letting a solver do useful work means rewriting the models
  relationally and keeping them inside EPR, the expert skill AUT-01 says an ordinary engineer should
  not need.
- **Property class.** `ordered`, `eventuallyWithin`, and correlated per-operation obligations are
  history-sensitive. This is not a fundamental limit of symbolic checking: monitor state can make
  the relevant history part of the checked state. It is a mismatch with Veil's current symbolic
  surface, which supports state invariants and fixed bounded trace patterns but not Umpire's full
  Property language. A Veil symbolic backend would therefore check a subset of the language unless
  Umpire first lowered those Properties to monitors.
- **Determinism and trust.** PLN-02 requires identical inputs to yield identical plans and checksums.
  Solver versions, seeds, and machine-dependent timeouts can change which SAT witness is returned or
  whether a bounded run finishes. Pinning the toolchain, canonicalizing witness selection, and
  replaying SAT counterexamples can control that engineering problem. An unsat result is the harder
  assurance boundary: it remains a trusted-solver claim unless proof reconstruction runs at three
  to five times the cost. VER-06 correctly records those outcomes in distinct trust classes.

The one real explosion is multi-instance models. Five operations over the 192-state protocol
machine is about 2.6 times ten to the eleventh raw states, and symmetry over five interchangeable
instances divides by only 120. Neither an explicit product table nor bounded model checking applied
after that table is materialized is comfortable there. A symbolic encoding that avoids constructing
the product table could help, but that is the architectural change the current admission path does
not permit; a solver also does not by itself supply the needed symmetry argument or inductive
invariant. Pairwise races need two instances, which the Pair model covers, and per-operation
properties are correlated by key, so one or two instances plus symmetry answers the present finite
question. A claim about any number of operations is a parametric proof per property and belongs in
the opt-in slot. If models acquire real data, payload identifiers compared for equality, integer
timestamps, unbounded queues, or an arbitrary number of workers, the complete finite table may stop
being viable. Umpire would then need a symbolic representation before admission rather than merely
an SMT backend after it.

### Adoption caveats

- Veil is a 2.0 pre-release with no tagged release. It moved to Lean's module system on
  2026-09-22 and pins Lean 4.32.0; `model/lean-toolchain` moved from 4.33.1 to match it (fn-88.12).
- Its SMT automation runs more than ten times slower than Ivy on the authors' own benchmarks
  ([Veil, Dafny 2026 paper](https://verse-lab.org/papers/veil-dafny26.pdf)); Lean-SMT proof
  reconstruction costs another three to five times.
- Veil dropped Mathlib on 2026-09-18, which makes the
  [fn-23](../.flow/specs/fn-23-veil-toolchain-compatibility-and.md) compatibility gate more likely
  to pass than when that spec was written. But fn-23 as specified reaches a conclusive result only
  inside a sandboxed `linux/aarch64` rootfs and returns `inconclusive` on the macOS machines the
  team develops on. Cut it down to a temporary Lake project with the probe on a developer machine.
- Veil carries no theorem that its extracted executable actions agree with its relational
  semantics ([VEIL_BACKEND_RESEARCH](VEIL_BACKEND_RESEARCH.md#what-veil-actually-exposes)).
  `Machine` carries sound and complete proofs. Adopting Veil beneath `CheckedModel` trades a proven
  boundary for an unproven one until that theorem exists.

## 2. How much of the prototype Veil can replace

Line counts are from the tree on 2026-09-26. Umpire's reusable library is about 56k Lean lines;
handwritten Temporal behavior models total about 1.5k.

| Umpire module | Lines | Veil can own it |
| --- | ---: | --- |
| Search, Model table, Machine kernel, Command/Finite | ~8k | Yes, through `RelationalTransitionSystem`, `EnumerableTransitionSystem`, and the concrete checker |
| Exploration | 1.6k | Partly, the walker over coverage targets |
| Property ([`Property.lean`](../model/lean/Umpire/Property.lean), [`Evaluate.lean`](../model/lean/Umpire/Property/Evaluate.lean)) | 8.3k | No. Veil has state invariants and fixed bounded trace formulas; no `ordered`, `eventuallyWithin`, correlated per-operation rules, or protobuf field relations |
| Evidence, Case, Artifact, Provenance, KnownGap, Replay, Promotion, Fingerprint | ~24k | No. Veil has nothing in this area |
| Command DSL | 6.1k | Only by replacing the authoring surface with Veil's, a GOV-02 change to AUT-07 |
| ImplementationLink and [refinement](../model/lean/Umpire/ImplementationLink/Refinement.lean) | 4.1k | No. Veil has no refinement command |
| Testpilot Lean and the Go runtime | 2.6k plus Go | No |

Two facts decide whether path 3 of [VEIL_BACKEND_RESEARCH](VEIL_BACKEND_RESEARCH.md#three-adoption-paths)
is worth it. Veil actions are ordinary Lean `do` code with `require`, which is the "logic as code"
shape [UMPIRE_CMP_FIZZBEE](UMPIRE_CMP_FIZZBEE.md#41-logic-as-lean-code-declarations-as-commands)
recommends over fn-85's row grammar. And the missing extraction-correspondence theorem above means
Veil's DSL would weaken the admission boundary today.

**Recommendation.** Write step functions as plain Lean code enumerated by the `machine` command
now; that needs no Veil. Spike the data adapter to Veil's concrete checker that the research note
proposes, with its four required outputs. Decide on the DSL after the spike.

## 3. Relation to Specula

[Specula](https://github.com/specula-org/Specula) is agent-driven
([paper, arXiv:2607.25333](https://arxiv.org/abs/2607.25333)). It reads a system's code, drafts a
TLA+ spec and invariants from docs, issues, and commit history, writes an instrumentation plan that
names for each TLA+ action where in the code it fires and which state to snapshot, replays the
recorded trace through TLC until it diverges, model-checks the spec, and reproduces violations as
integration tests. It claims 249 bugs across 48 systems, 68 confirmed, at a median cost of 57
dollars and about 1.5 hours of review per system. Its critics note that a spec derived from the
code under test is circular and that about one fifth of its invariants are protocol level
([Demirbas, 2026-08](https://muratbuffalo.blogspot.com/2026/08/specula-scaling-formal-specifications.html)).

Umpire runs the opposite direction on both axes.

- **Authority.** SEM-01 and the Feature/System split exist to prevent implementation-shaped models.
  Specula's spec is implementation-shaped by construction.
- **Bridge.** Specula validates recorded traces. Umpire generates Cases and drives the system. Umpire
  has no working trace-validation path: `Umpire.Evidence` is exercised only by tests, and
  [`ExplorationBridge.lean`](../model/lean/Temporal/Tool/ExplorationBridge.lean) states that it reads no
  Run Event.

What to borrow:

- The per-action instrumentation plan is the white-box half the vision names and the tree never
  built, and Specula shows an agent can draft it.
- Its scenario projections, which disable actions and coarsen steps, are Umpire's Scenarios and
  Known Gaps without Umpire's discipline; keep the discipline.
- The bugs Specula finds are interleaving bugs in consensus code. Umpire's tables cannot express
  that class, because goroutine and network scheduling are non-goals. That class remains outside
  Umpire's scope.

## 4. Is the tracing sufficient

### What reaches Lean

The only record is the Testpilot Run
([`run.proto`](../proto/internal/temporal/server/api/testpilot/v1/run.proto),
[runtime README](../common/testing/testpilot/README.md)): harness events such as instruction
started or completed, one `FAULT_INJECTED` kind for worker stop and resume, and public-API reads,
namely history events and `DescribeWorkflowExecution` pending-operation attempts. The Go Monitor
evaluates the Contract. Lean reads only disposition, cleanup, and Verdict. The server Session
([`session.go`](../common/testing/testpilot/temporal/server/session.go)) refuses every fault. No
server state, persistence, matching, timer, or clock data is recorded.

### What is emitted and never read

[fn-81](../.flow/specs/fn-81-delete-the-pre-testpilot-go-generations.md) deleted the white-box
OTEL observer. The server still emits enriched span events for CHASM transitions
([`chasm/statemachine.go`](../chasm/statemachine.go)), workflow lineage, updates, and matching
store or discard, under the vocabulary in [`common/telemetry/tags.go`](../common/telemetry/tags.go),
whose header cites a plan document that does not exist. Nothing consumes them. A persistence
interceptor seam ([`interceptor.go`](../common/persistence/intercept/interceptor.go)) is wired
through fx, and every caller passes `nil`. The test hooks in
[`hooks.go`](../common/testing/testhooks/hooks.go) change behavior and record nothing.

### Against the vision

| Vision item | State |
| --- | --- |
| Black-box mode | Exists; honest by design |
| White-box mode | Absent since fn-81 |
| Faults as first-class citizens | One kind, worker stop; server side none |
| Exploration to find unknown bugs | Walks coverage targets of the finite table; credits from the Verdict alone |
| Guided fuzzing | Absent |
| Canary | [fn-29](../.flow/specs/fn-29-bounded-production-canary-execution-and.md) in progress; black-box, which is correct |

The Nexus caller's [COVERAGE.md](../model/lean/Temporal/Feature/Nexus/Caller/COVERAGE.md) already lists
transport faults, mutable state, pending timeouts, duplicate completion, and two unobserved silent
steps as Known Gaps. The bugs Temporal ships, task-queue races, speculative workflow task handling,
update races, and duplicate delivery, live below the public API. History reads detect some after
the fact and steer toward none.

## 5. Value proposition and usability against FizzBee

[UMPIRE_CMP_FIZZBEE](UMPIRE_CMP_FIZZBEE.md) already holds the side-by-side. The state on
2026-09-26: FizzBee 0.5.3 gets an engineer to a first checked model in hours with Starlark, a
playground, state graphs, implicit crash and loss at every yield, liveness under fairness, refinement
checking, model-based testing scaffolds in four languages, and agent skills. Umpire's own
[FEATURE_AUTHORING_ASSESSMENT](FEATURE_AUTHORING_ASSESSMENT.md) says the Feature surface does not
yet meet the intended experience for developers without deep Lean knowledge, the FizzBee note puts
time to first model at days, and no tool renders a diagram. About 130k Lean lines and 19k lines of
plans cover one feature family.

Umpire's differentiator is a different product. One deterministic Case runs against a real Temporal
server and SDK worker with typed API catalogs, returns a three-valued Verdict, fails closed on
missing evidence, keeps byte-identical fixtures with provenance and Known Gaps, and proves refinement
and evaluator agreement in Lean. FizzBee's model-based testing is a hand-written adapter in which an
unimplemented action passes a thousand runs. That makes Umpire a regression and canary gate with
auditable claims. State that as the value proposition and stop competing on design-exploration
ergonomics, where SCP-04 already says to complement rather than replace.

Fixes, in value order:

1. **Logic as Lean code.** Step functions over structures, enumerated by the `machine` command into
   the same table. Drop the row grammar planned in fn-85. This is section 4.1 of the FizzBee note.
2. **Render.** Mermaid state diagrams from the checked table with the witness path highlighted, and
   sequence diagrams from the Case Program, embedded in the generated coverage documents.
3. **Implicit fault placement in Search** with `ephemeral` fields, realized at runtime as declared
   instructions under EVD-20.
4. **An authoring skill for coding agents.** FizzBee and Specula both ship one, and agents will
   write most of these models.
5. **Run the human pilot** the authoring assessment asks for: five to eight engineers with no Umpire
   background, measuring time to a live Verdict.
   [fn-14](../.flow/specs/fn-14-milestone-a-pilot-baseline-and-lean.md) was superseded and never
   replaced.

## 6. Are we reimplementing a model checker

Partly, and the current roadmap adds the rest of one.

**What exists.** [`Umpire/Search.lean`](../model/lean/Umpire/Search.lean) is a 1.2k-line hand-rolled
checker: iterative-deepening depth-first search over traces, candidate caps, deterministic ordering,
and a seeded rotation. [`Scenario/Check.lean`](../model/lean/Umpire/Scenario/Check.lean) is an 800-line
admitted-prefix automaton and [`Exploration.lean`](../model/lean/Umpire/Exploration.lean) with its
directory is a 1.6k-line coverage walker. Together they are the `Checker` of
[Stateright](https://github.com/stateright/stateright) written in Lean, minus the parts Stateright
gets right: no visited set, no symmetry reduction, no breadth-first search, no explorer. The
property side is different in kind. [`Property/Evaluate.lean`](../model/lean/Umpire/Property/Evaluate.lean)
is a trace oracle with a proven agreement to its denotation, and that oracle stays whichever engine
runs the search.

**Why it grew.** Properties are evaluated over whole traces, so the search enumerates paths rather
than states. That single choice forces a bespoke checker. TLC, Ivy's checker, Stateright, and Veil's
concrete checker all check a state predicate over a product of model state and monitor state.
[VEIL_BACKEND_RESEARCH](VEIL_BACKEND_RESEARCH.md#the-exact-umpire-seam) names the fix: the checked
state must be the product of model state, Scenario progress, and Property-monitor state. The monitor
form already exists. A Contract Rule is a finite state machine with satisfied and violated terminals,
and [`Case/Compiler.lean`](../model/lean/Umpire/Case/Compiler.lean) assembles Properties into it for the
Go runtime. Lowering the same way for search turns the problem into safety reachability on a finite
labeled graph.

**Where the roadmap heads.** fn-85's multi-instance interleaving and recommendations 4.4 through
4.6 of [UMPIRE_CMP_FIZZBEE](UMPIRE_CMP_FIZZBEE.md#4-what-to-take-from-fizzbee), implicit fault
placement, symmetry canonicalization of entity instances, and seeded random walks, are Stateright's
feature list item by item. Building them into `Search.lean` is the reimplementation to avoid.

**The rule that causes it.** [UMPIRE4_SPEC_MODEL_ARCH](UMPIRE4_SPEC_MODEL_ARCH.md#9-optional-formal-verification-flow)
says Umpire does not generate checker source or introduce a checker-neutral semantic IR. But
`FiniteTable` already is that IR, canonical, serializable, and fingerprinted. Keeping the rule means
the only checker that can consume it is one written in this repository.

**Recommendation.**

1. Freeze `Search.lean` as the reference checker used in tests. Add no symmetry, instances, or fault
   placement to it.
2. Lower Property and Scenario to monitor automata once, shared with the Contract lowering, and
   define the check as reachability over the product.
3. Feed the product to one existing engine. Veil's concrete checker has the least friction: same
   language, no serialization, counterexamples arrive as Lean values, and `#simulate` covers seeded
   walks. Stateright needs Rust and JSON glue but is mature and has the explorer. TLC needs Java and
   TLA+ generation, and its strengths are wasted on an explicit table. Any of the three beats a
   fourth checker in Lean.
4. Amend section 9 of the model architecture to allow generating checker input from `FiniteTable`.
   VER-05 already requires every counterexample to replay through the Lean kernel, which is the
   safeguard that makes an external engine safe to trust.

The checker is not where most of the 56k lines went. Search, Scenario checking, and Exploration
total about 3.6k. The sprawl is in artifacts, provenance, evidence, and Case plumbing, which no model
checker provides. The reimplementation risk is about the next 10k lines.

## Recommended order

1. Step functions as Lean code in the `machine` command; Mermaid rendering; an authoring skill.
   No new dependency.
2. Freeze `Search.lean`; lower Property and Scenario to monitor automata shared with the Contract
   lowering; spike the Veil concrete-checker adapter on the Nexus caller model with the four
   outputs the Veil research note requires. Simplify fn-23 to a developer-machine probe first.
3. Human authoring pilot, five to eight engineers, time to a live Verdict.
4. Consume the existing span events as Observations, and revisit exploration guidance
   with the semantic-coverage experiment
   ([UMPIRE4_RESEARCH](UMPIRE4_RESEARCH.md#1-lean-native-semantic-coverage-for-implementation-fuzzing)).

## Verification performed

Source reading covered the plan set, the spec, the open Veil, canary, and pilot specs, the Umpire
and Temporal Lean modules, the Testpilot runtime and protocol, the telemetry and persistence seams,
and the public Veil, Specula, and FizzBee
repositories and papers. `go run ./tools/planindex` was run after registering this document. No
Lean or Go build, lint, or test was run, and no implementation file was changed.
