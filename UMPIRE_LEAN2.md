# Umpire3 verification architecture

Status: approved implementation blueprint, 2026-08-20.

This document defines the next architecture for Umpire3 after the 1.2 candidate implemented under
`tests/umpire3`. It complements `UMPIRE_LEAN.md`: that document records the 1.0-to-1.2 roadmap and
operational framework; this document concentrates on the semantic architecture required to turn the
framework into a trustworthy, multi-modal verification workbench.

The ambition is deliberately large:

> A Temporal behavior is modeled once as a Lean model family. The same family supports feature
> contracts, distributed-system mechanisms, refinement proofs, exact finite exploration, symbolic
> bug finding, generated regressions, live conformance, evidence qualification, campaigns, and
> eventually proof-producing safety and liveness checking.

The central decision is not to make Umpire a wrapper around one checker. Umpire owns a small Lean
semantic kernel and explicit, proved views of each model. Veil is the primary advanced Lean checking
backend. A collision-safe Umpire explorer is the proof-grade finite-state foundation. TLC and
Apalache are independent external backends for the areas where their mature algorithms add value.
Ivy is prior art, not an integration target.

## 1. Decisions

These decisions are normative.

| ID | Decision |
| --- | --- |
| D1 | Lean remains the sole semantic authority. Go owns orchestration and evidence transport, not Temporal behavior. |
| D2 | “One model” means one compositional `ModelFamily`, not one monolithic state machine. |
| D3 | Feature models and system models are independently defined. A refinement proof relates them after both exist. |
| D4 | No system transition may contain a proof that it already performs a legal feature transition. That makes refinement circular. |
| D5 | Every executable, finite, first-order, observation, and backend representation is an explicit view with a soundness or equivalence obligation. Arbitrary Lean is never silently translated. |
| D6 | Umpire will build an exact, collision-safe finite explorer and a small Lean certificate checker. Fast native and external search engines are replaceable certificate or witness producers. |
| D7 | Integrate **Veil**, spelled exactly that way. Use Veil's `#model_check`, symbolic trace queries, and invariant workflow. Do not add Ivy as a required tool. |
| D8 | Veil `#model_check` is a high-value testing engine, not proof-grade completeness evidence until its state-identity boundary is collision-safe for the selected mode. |
| D9 | A live adapter emits typed raw evidence. It does not emit the Boolean truth of the property it is meant to check. Generated, Lean-defined observation programs interpret evidence. |
| D10 | Composition contracts contain propositions and proofs. String identifiers, ranks, hashes, and `status := "complete"` are exported metadata, never the reason composition is valid. |
| D11 | Safety, bounded safety, finite exhaustive checking, inductive proof, liveness proof, and live conformance remain distinct result classes. There is no generic `verified` Boolean. |
| D12 | Lean and model checkers remain offline build/test tools. No Lean FFI, solver, or checker enters a Temporal production request path. |
| D13 | The first end-to-end pilot is stale Nexus completion after cancellation/ownership change. It already has a Lean model, a mutation, a trace, a runtime adapter, and a real-cluster path. |

## 2. Terminology and scope

### 2.1 Feature model

A **feature model** states user-visible Temporal meaning. It answers questions such as:

- When is a Nexus Operation terminal?
- What does accepting cancellation permit or forbid?
- What lineage must Connect-as-New or Reset preserve?
- What outcome may an Update report after its acceptance is durable?

This is the role currently named `Temporal.Product`. The new architecture uses “feature” in design
discussion because it is clearer to Temporal developers; existing package names need not be renamed
immediately.

A feature model excludes shards, task queues, persistence attempts, ownership epochs, processor
loops, and retry machinery unless one of those is itself part of the user-visible contract.

### 2.2 System model

A **system model** states a selected mechanism by which Temporal may implement one or more feature
contracts. It may contain tasks, attempts, ownership, durable records, failover, matching, clocks,
retries, crashes, and recovery.

A system model is not the Go implementation. It is an independently reviewable abstraction of the
mechanism most likely to violate the feature contract.

### 2.3 Implementation

The Go server is the implementation under test. Umpire normally relates it to a model through
qualified execution evidence. A passing live run is a conformance result for one build, profile,
experiment, and evidence set. It is not a universal refinement theorem.

### 2.4 Model family

A **model family** is the authoritative graph containing:

- feature models;
- system models;
- shared mechanism contracts;
- refinement relations;
- observation interpretations;
- safety and temporal properties;
- executable, finite, and first-order views;
- target projections and declared omissions; and
- theorem-backed source identities for generated artifacts.

It is a graph because a Temporal feature can have several system realizations, one system mechanism
can support several features, and a focused checking target should not pull the entire server into
one state space.

### 2.5 Configuration world

Each check runs in a **world**: a choice of finite identities, topology, enabled features, retry and
fault bounds, time abstraction, and environment assumptions. A world is immutable during an
execution. Runtime state evolves within it.

This also supports software-configuration feature models if Umpire needs them later. Configuration
constraints select valid worlds; the feature and system transition models execute inside a selected
world. The two use cases therefore share a kernel without pretending constraint solving and
distributed execution are the same algorithm.

## 3. Current baseline: Umpire3 1.2

The analysis in this document uses commit `ed8b85399` (`Expand Umpire3 into a model-driven test
framework`) as observed on 2026-08-20. As a baseline verification of that revision,
`make umpire3-check` passed: generated drift checks, 87 Lean build jobs, and every focused Go
package succeeded.

The generated catalog currently reports:

| Kind | Count |
| --- | ---: |
| Semantic types | 9 |
| Capabilities | 12 |
| Actions | 33 |
| Entities | 11 |
| Relations | 11 |
| Observations | 18 |
| Properties | 15 |
| Faults | 13 |
| Modules | 39 |
| Targets | 15 |
| Monitor programs | 15 |

The migration ledger describes 28 root behaviors: 21 exact, five semantically equivalent, and two
partial. The checked 1.2 release remains correctly labeled `candidate`; only two of its 13 vision
goals are `passed`, while 11 remain `partial`. It requires external qualification for remote,
gRPC-only, and production-canary profiles.

That is substantial implementation, not a toy. The next step is to make the assurance depth match
the operational breadth.

## 4. Good patterns to keep and deepen

### 4.1 Lean/Go separation

The versioned-data seam is correct. Lean emits catalogs, experiments, monitor programs, composition
metadata, parity metadata, proof manifests, and coverage data. Go strictly decodes and executes
those artifacts. Lean is not linked into Temporal.

Keep this seam. Make the artifacts more theorem-backed; do not replace the seam with cgo or a live
Lean process.

### 4.2 Relational and executable semantics

`Umpire3.Transition` defines relational behavior. `Umpire3.Executable` requires `next_iff`, proving
that executable successors neither invent nor omit relational transitions. This is the most
important formal pattern in the current kernel.

Generalize it to initial-state enumeration, labeled successors, finite worlds, observations, and
checker views. Never accept an executable projection without both soundness and completeness.

### 4.3 Feature/system source layout

The `Temporal/Product`, `Temporal/System`, and `Temporal/Refinement` split communicates the intended
layers. The product files mostly avoid system mechanics, and system models name threats such as stale
ownership and duplicate delivery.

Keep the layout while making the models genuinely independent.

### 4.4 Strict, generated protocol vocabulary

The catalog-generated Go identifier types removed the original handwritten action and capability
allowlists. Strict JSON decoding, canonical encoding, content digests, schema drift checks, and
generated author constructors are strong foundations.

Keep generated vocabulary. Move from string-backed declarations to typed, theorem-backed
declarations before export.

### 4.5 Selected protobuf projection

`Temporal/API/selection.json` selects a bounded wire surface, closes descriptors recursively, and
records field dispositions and fixtures. Product meaning remains in handwritten Lean interpretation
modules rather than being guessed from field names.

This is the right direction. Extend the proof connection between wire interpretation, action
meaning, and live transport fixtures; do not import all Temporal protobufs.

### 4.6 Fail-closed runtime behavior

The compiler and runtime reject unknown vocabulary, incomplete bindings, insufficient capabilities,
budget exhaustion, missing evidence, ambiguous ordering, and incomplete cleanup. Environment
profiles separate driving, observation, and fault authority. The canary path uses process isolation
for enforceable execution limits.

These are production-quality instincts. Preserve them as formal results become stronger.

### 4.7 Causal evidence rather than clock faith

The evidence graph carries source identity, clock domain, source sequence, causal references,
entity identity, and lineage. Cross-source timestamps are not silently treated as a total order.

Keep this data model. Replace Boolean adapter judgments with generated evidence interpretation.

### 4.8 Negative controls and replay

The models contain invalid states or weakened transitions, the runtime has controlled faulty
adapters, and campaigns minimize and replay failures. A proof or monitor with no demonstrated power
to reject a nearby wrong behavior is weak assurance.

Make mutation adequacy a release metric: every important property should reject both a semantic
mutation and an implementation/evidence mutation.

### 4.9 Honest release qualification

The 1.2 manifest does not fabricate remote or production receipts. Partial migration fidelity is
recorded rather than hidden behind skips.

Keep the candidate/qualified distinction and apply the same honesty to formal checking claims.

## 5. Patterns to replace

The following are architectural findings, not criticisms of the bootstrap. Several were reasonable
ways to prove a vertical slice. They are unsafe foundations for the moonshot version.

### 5.1 P0: refinement is proof by construction

All 12 current system models with feature counterparts define a `TransitionResult` containing:

```lean
productActions : List Product.Action
productRun : Runs Product.model state.visible productActions nextState.visible
```

The system transition exists only after the author constructs a legal product run. The refinement
proof then opens the system transition and returns the stored `productRun`. Because system state also
embeds `visible : Product.State`, the abstraction relation is normally equality on that field.

This proves that the author used the constructor correctly. It does not independently test whether a
mechanism model implements the feature model. A modeling mistake that should make refinement fail is
often unrepresentable as a system step.

Replace this pattern with:

1. a system `Step` defined only in system vocabulary;
2. an independently defined relation `Relates : System.State → Feature.State → Prop`;
3. an action/event mapping defined in the refinement module; and
4. a proof that every system step simulates zero or more feature steps.

The stale-completion mutation must be expressible as a normal mutated system model and must make the
refinement theorem fail. That is the acceptance test for genuine independence.

### 5.2 P0: monitor equivalence ends before the real observation seam

For many properties, `Temporal.Monitors` defines one Boolean normalized observation whose value is
the feature predicate itself, then proves that checking that Boolean is equivalent to the predicate.
For example, the Nexus closure observation is defined as `closureB state`, so the equivalence proof
is necessarily simple.

The live SDK adapter separately contains a large switch that computes observations such as
`nexus-operation-closed`, `callback-reference-valid`, and `workflow-ownership-fenced` from Go state
and history. The Lean theorem does not prove that this Go computation corresponds to the Lean
observation function.

Replace the Boolean seam with a typed evidence program:

```text
raw public/internal facts
        |
        v
generated typed interpretation program
        |
        v
abstract observations + identity/ordering proof obligations
        |
        v
three/four-valued monitor
        |
        v
established / violated / unknown / conflict
```

Adapters should report event kinds, typed fields, positions, references, and omissions. They should
not report that the target property is satisfied.

### 5.3 P0: proof and composition manifests are self-attestations

Current proof manifests store theorem names and statement hashes as strings. Composition modules
store guarantee identifiers, hashes, prose, and `status := "complete"`. Their `WellFormed` theorems
show that the strings are present and internally consistent; they do not resolve a theorem name,
derive its statement hash, establish that a provider proves a proposition, or prove that an exported
target closes its obligations.

This metadata is useful, but it is an index, not proof evidence.

Replace author-supplied attestations with typed declarations:

```lean
structure TheoremRef (claim : Prop) where
  name : Name
  proof : claim

structure Guarantee {World : Type} (model : Behavior World) where
  identifier : Name
  claim : (world : World) → Execution model world → Prop
  holds : ∀ world execution, ValidExecution model world execution → claim world execution

structure Requirement {World : Type} (model : Behavior World) where
  claim : (world : World) → Execution model world → Prop
```

Export names, hashes, and statuses after Lean has checked the values. A Lean metaprogram should
resolve declaration names, inventory axioms, and derive dependency/source digests. Humans should
never type a theorem statement hash.

### 5.4 P0: path enumeration is not state-space model checking

`BoundedModel.frontier` expands complete execution paths recursively. It has no visited-state set,
so cycles and converging paths duplicate work exponentially. `take maxResults` truncates a list but
does not produce completeness evidence. `Executable.follow` checks a supplied trace; it does not
search for one.

The Go `explore` package enumerates template assignments and calls the scenario compiler. Its
`Build` and `Observe` closures live in Go. A Boolean field says whether symmetry or partial-order
preservation was “checked.” This is guided test generation, not a Lean model checker.

Keep the campaign machinery, but name it accurately. Add a real reachability engine with:

- exact state equality;
- a visited set and work queue;
- shortest predecessor traces;
- action coverage and deadlock reporting;
- explicit finite-world or depth completeness;
- collision-safe state identity;
- deterministic resource termination; and
- a checkable closure/safety certificate.

### 5.5 P0: broad assurance can be vacuous

Several feature actions set the property they are meant to preserve directly to `true`. The
aggregate `Temporal.Product.Assurance` model is not imported by the main `Temporal` library and its
actions only move toward good Boolean flags. Negative controls show that predicates can be false on
manually constructed states, but not necessarily that realistic threatening transitions can reach
those states.

Every safety property needs four non-vacuity checks:

1. the antecedent is reachable;
2. a good outcome is reachable;
3. a nearby bad mechanism is expressible and rejected; and
4. the property is not merely an assignment performed by every enabled action.

Delete dead assurance models or connect them through proved abstraction. Do not count an unimported
or unreachable negative state as model-checking strength.

### 5.6 P1: the semantic catalog is still mostly metadata

The catalog is generated from Lean and well validated, which is valuable. However, declarations use
strings for types, actions, dependencies, properties, modules, and targets. `catalogWellFormed`
checks identifier references, not that an action declaration denotes a constructor of a particular
model, that a dependency is semantically necessary, or that a target's property is proved by its
modules.

Make declarations dependent on the model values they describe. Export strings only from registered
typed handles. The compiler may continue consuming the generated catalog, but the catalog must be a
projection of semantic declarations rather than a parallel inventory.

### 5.7 P1: feature models are numerous but not compositionally executed

Umpire3 now has focused models for closure, timeout, links, callbacks, lineage, routing, ownership,
speculative tasks, and progress. They are mostly independent finite universes. Composition metadata
declares ownership, ranks, requirements, interference action names, and omissions, but no composed
transition system demonstrates how two features share state or interfere.

Introduce theorem-backed composition for selected cross-feature slices. Do not immediately compose
all 15 targets. Start with two modules sharing one mechanism, prove non-interference or a rely/
guarantee rule, and model-check the composition.

### 5.8 P1: only state safety is formalized

The kernel defines `Safety` over reachable states. It has no explicit behavior stream, fairness,
enabledness, leads-to, termination-sensitive refinement, or divergence condition. Properties named
“progress” or “starvation” are currently finite-state invariants over explicit Boolean state, not
unbounded liveness theorems.

Retain these bounded safety contracts, but label them precisely. Add temporal semantics before
claiming eventual progress under retries, fair task delivery, or recovery.

### 5.9 P1: coverage is narrower than the catalog

The generated coverage denominator currently contains 17 Nexus lifecycle edges for one target,
while the catalog exposes 15 targets and 15 properties. Parity and migration ledgers use theorem and
monitor names as strings, so “complete” means declared completeness plus separate tests, not
machine-resolved proof coverage.

Coverage should be derived per model from typed initial states, action constructors, transition
classes, properties, relation schemas, observation alternatives, faults, and refinement cases. A
target without a denominator is `coverage-undefined`, not complete.

### 5.10 P1: source identity is manually curated

Export commands hash manually maintained file lists. A newly imported semantic file can affect a
Lean artifact without necessarily being added to the Go list. Statement hashes and assumptions are
also manually copied.

Hash the transitive Lean environment or a generated dependency manifest rooted at resolved
declarations. Include toolchain, options, axioms, generated descriptor digests, and backend adapter
versions. A source list can remain in the human-readable manifest, but it must be generated.

### 5.11 P2: one massive toolchain would be brittle

The main model currently pins Lean 4.33 with no Lake dependencies. The surveyed Veil 2 revision pins
a different Lean version and is explicitly pre-release. Pulling Veil directly into the main model
would couple every Umpire proof and generator to Veil churn.

Use an isolated backend project and versioned input/output seam first. Merge toolchains only when
Veil's version and interface are stable enough that doing so reduces, rather than increases, risk.

## 6. Non-negotiable semantic invariants

The implementation must enforce these invariants.

1. **One meaning.** Feature behavior is defined once in Lean. Generated catalogs, traces, monitors,
   and checker views point back to that definition.
2. **Independent layers.** A system model can be read without importing a feature state or proof.
3. **Explicit abstraction.** Every loss of detail has a named relation and an adequacy obligation.
4. **Executable exactness.** Enumerators are proved equivalent to relational semantics.
5. **No silent bounds.** Identity cardinalities, depth, fairness, retries, faults, time, memory, and
   state limits are part of the result.
6. **No proof laundering.** External UNSAT, finite exhaustion, and live conformance never become a
   Lean theorem without a checked certificate or proof.
7. **No oracle in the driver.** Action success and property truth come from independent evidence.
8. **No success from absence without closure.** An absence observation requires a closed evidence
   interval or authoritative terminal cut.
9. **No timestamp causality across clocks.** Cross-domain order requires a causal edge or explicit
   ordering contract.
10. **No self-attested obligation.** A qualifying `complete` status is computed from checked
    evidence, never authored.
11. **No unqualified reduction.** Symmetry, partial-order, and abstraction reductions need a theorem
    or are labeled heuristic.
12. **No cleanup amnesty.** Incomplete cleanup can invalidate a conformance result and always remains
    visible.

## 7. Target architecture

```text
                       LEAN MODEL FAMILY

   selected wire types ──> interpretation ──> FEATURE MODEL F
                                                   ^
                                                   | refinement theorem
                                                   |
                         shared contracts ──> SYSTEM MODEL S
                                                   |
                            +----------------------+------------------+
                            |                      |                  |
                      ExecutableView          FiniteView       FirstOrderView
                            |                      |                  |
                    trace interpreter      exact explorer       Veil adapter
                            |                + certificate       + #model_check
                            |                      |              + symbolic trace
                            |                      |              + invariants
                            |                      |
                            +---------- normalized trace --------+
                                                   |
                                        experiment/scenario view
                                                   |
                                            GO REALIZATION
                                                   |
                                           typed raw evidence
                                                   |
                                   generated observation interpreter
                                                   |
                                     qualified conformance result

  Independent portfolio: TLA+/TLC and Apalache views consume the same named model target,
  return normalized traces, and are differentially checked against the Lean executable view.
```

There are three semantic relationships:

```text
system model S  refines  feature model F       -- Lean theorem
implementation I conforms to selected S/F      -- qualified execution evidence
checker view V represents selected S/F          -- equivalence, refinement, or scoped evidence
```

Never collapse them into one arrow.

## 8. Semantic kernel 2.0

Keep the new Lean types internal through A1 and A2. Stabilize the public registration API only after
it expresses the Nexus pilot and a compile-only spike of a second existing feature/system pair. Its
responsibilities are fixed even while the dependent-type shape is being tested.

### 8.1 Behavior

Conceptually:

```lean
structure Behavior (World : Type u) where
  State : World → Type v
  Action : World → Type w
  Initial : (world : World) → State world → Prop
  Step : (world : World) → State world → Action world → State world → Prop
```

`World` separates immutable configuration from mutable state. It prevents topology, feature flags,
and finite cardinalities from being smuggled into globals or duplicated per backend.

The kernel also needs labeled executions, reachability, finite prefixes, infinite behaviors,
enabledness, and stuttering. Keep these definitions small and ordinary. Do not build a general
theorem-prover language inside Umpire.

### 8.2 ExecutableView

```lean
structure ExecutableView {World : Type u} (model : Behavior World) where
  initials : (world : World) → List (model.State world)
  successors : (world : World) → model.State world →
    List (model.Action world × model.State world)
  initials_exact : ...
  successors_exact : ...
```

The successor list includes the action label. The current split between an action list and
`next state action` may remain as an internal adapter, but checkers should consume labeled
successors directly.

### 8.3 FiniteView

A finite view adds decidable equality, canonical serialization, and finiteness evidence for one
world. It must distinguish:

- finite entire-state completeness;
- complete exploration through depth `k`;
- a bounded abstraction of an infinite model; and
- heuristic sampling.

An encoding used for deduplication must be injective or collision-resolving. A 64-bit hash alone is
never state identity.

### 8.4 ModelFamily

Conceptually:

```lean
structure ModelFamily where
  World : Type
  FeatureId : Type
  SystemId : Type
  feature : FeatureId → Behavior World
  system : SystemId → Behavior World
  realizes : SystemId → FeatureId → Prop
  relates : {systemId : SystemId} → {featureId : FeatureId} →
    realizes systemId featureId → (world : World) →
    (system systemId).State world → (feature featureId).State world → Prop
  refines : {systemId : SystemId} → {featureId : FeatureId} →
    (edge : realizes systemId featureId) →
    Refinement (system systemId) (feature featureId) (relates edge)
  observations : (featureId : FeatureId) → ObservationModel (feature featureId)
```

This is a shape, not promised compiling syntax. The final types may use dependent records or
explicit parameters to keep elaboration manageable. The important parts are that a family can
represent many-to-many realization edges and that `system` does not contain `feature.State`,
`feature.Action`, or a `feature.Run` witness.

### 8.5 Action mapping

Refinement needs more than unlabelled `StepStar`. Add an explicit mapping:

```lean
System.Action → Feature.ActionEmission

Feature.ActionEmission =
  | stutter
  | one Feature.Action
  | many (List Feature.Action)
```

The proof shows that the mapped emission is legal. The mapping is useful for counterexample source
maps, coverage, and generated experiments. It must not be stored as proof inside the system step.

### 8.6 Refinement strengths

Provide named refinement interfaces rather than one overly permissive relation:

- **Safety simulation** — every finite system prefix maps to a legal feature prefix.
- **Observation refinement** — the system and feature agree on declared externally visible facts.
- **Failure refinement** — system failures map only to permitted feature outcomes.
- **Liveness refinement** — under explicit fairness and non-divergence conditions, selected feature
  progress is preserved.

Most models will start with safety simulation. A manifest reports exactly which strengths exist.

### 8.7 Typed declarations

Catalog registration should bind metadata to values:

```lean
structure PropertyDeclaration {World : Type} (model : Behavior World) where
  identifier : Name
  predicate : (world : World) → model.State world → Prop
  safety : Option (∀ world, Safety (model.at world) (predicate world))
  observation : Option (ObservationSpec model predicate)
```

This prevents the catalog from claiming that a string-named theorem proves an unrelated string-
named property. The exporter erases proofs and emits stable names, hashes, source maps, and claim
strengths.

## 9. Feature and system model design

### 9.1 Feature modules

A feature module owns a coherent user-visible contract, not one Boolean per property. A good module
contains:

- typed identities and values;
- state with explicit unknown/absent distinctions where relevant;
- semantic commands and externally visible outcomes;
- relational and executable transitions;
- safety and temporal properties;
- positive reachability witnesses;
- threatening mutations; and
- observation requirements.

For example, Nexus cancellation should model the race among cancellation acceptance, cancellation
winning, operation completion, and visible terminal outcome. It should not mention owner epochs.

### 9.2 System modules

A system module owns the smallest mechanism that can threaten the feature. For the same pilot:

- task dispatch and attempts;
- owner availability and ownership epoch;
- worker epoch and returned completion epoch;
- cancellation commit/persistence;
- completion validation/persistence; and
- crash, recovery, retry, and duplicate delivery.

Its state should not contain `visible : Product.State`. If the model needs a system-level durable
outcome, define a system outcome type and map it in the refinement relation.

### 9.3 Shared mechanism modules

Task delivery, persistence atomicity, ownership fencing, and history ordering should become deep
modules with actual predicates and theorems. A consumer imports a theorem-backed guarantee or states
an assumption explicitly.

The current `TaskDelivery.Guarantee` identifier/hash pair becomes export metadata for something like:

```lean
def CurrentCompletionOnly (trace : DeliveryTrace) : Prop := ...

theorem taskDeliveryGuarantee :
  ∀ trace, DeliverySystem.Accepts trace → CurrentCompletionOnly trace := ...
```

### 9.4 Composition

Composition is a proof problem, not a manifest-validation problem. A composed target must establish:

- state ownership or shared-state protocol;
- compatible initial states;
- action synchronization rules;
- rely/guarantee compatibility;
- interference preservation for every invariant;
- absence of circular assumptions;
- fairness allocation for liveness; and
- a projection/refinement to its feature contract.

Ranks can remain a human-readable DAG diagnostic. They are not the proof of acyclicity or
compatibility.

### 9.5 Target projections

A target is a deliberate view of a model family:

```text
target = world constraints
       + selected modules
       + selected properties
       + retained actions/faults
       + abstraction relation
       + declared omissions
       + checker capabilities
```

An omission must state what is abstracted and why the selected property is preserved. A numeric
`maxCount` without a preservation argument is an execution bound, not a sound abstraction.

## 10. The model-checker portfolio

### 10.1 What Lean is and is not

Lean is a theorem prover and a pure functional programming language capable of building executable
explorers. It is not automatically a model checker because a transition relation is written in Lean.
Lake can build normal executables, making Lean a suitable host for a reference interpreter,
certificate checker, and small exact explorer ([Lean programming introduction](https://lean-lang.org/functional_programming_in_lean/Introduction/),
[Lake reference](https://lean-lang.org/doc/reference/latest/Build-Tools-and-Distribution/Lake/)).

### 10.2 Veil, not “Vail”; Lace is not a separate dependency

The relevant project is [Veil](https://github.com/verse-lab/veil). Veil 2 is a Lean-embedded
framework for distributed transition systems. Its public workflow includes:

- `#model_check` for concrete finite explicit-state exploration;
- `sat trace` and `unsat trace` for SMT-backed bounded traces;
- `#check_invariants` for inductive-invariant obligations; and
- ordinary Lean proofs as the escape hatch.

One current example comment calls the concrete checker “Lace,” but the public package and command
surface do not present Lace as a separate tool. Umpire should consistently say **Veil's
`#model_check`**. Veil 2 is explicitly pre-release, so pin an exact revision and isolate it
([Veil README](https://github.com/verse-lab/veil/blob/300c305e945750ab3fb62de4a79c23161b24da39/README.md),
[DSL reference](https://github.com/verse-lab/veil/blob/300c305e945750ab3fb62de4a79c23161b24da39/docs/DSL-Reference.md)).

### 10.3 Why Veil fits

Veil follows the workflow Umpire needs:

1. find short bugs in small instances;
2. inspect action coverage and counterexamples;
3. use counterexamples to induction to strengthen invariants;
4. discharge first-order obligations with SMT; and
5. fall back to interactive Lean for the hard remainder.

Its verification-condition generator has a Lean soundness story, and Veil 2 is rebuilt on Loom's
executable effect semantics ([Veil CAV 2025 paper](https://verse-lab.org/papers/veil-cav25.pdf),
[Loom POPL 2026 paper](https://verse-lab.org/papers/loom-popl26.pdf)). This is much closer to Umpire's
goal than maintaining a separate source language.

### 10.4 Veil trust boundary

Umpire must expose three important qualifications.

1. Veil's concrete checker exhausts a selected finite instance; it does not prove arbitrary system
   size or background theories. Veil's own command source says it should be treated as testing.
2. The surveyed concrete checker stores 64-bit state fingerprints, while its completeness theorem
   assumes an injective state view. A general 64-bit hash is not injective. Use this mode for fast
   bug discovery, not proof-grade finite exhaustion
   ([fingerprint source](https://github.com/verse-lab/veil/blob/300c305e945750ab3fb62de4a79c23161b24da39/Veil/Core/Tools/ModelChecker/Concrete/Core.lean),
   [completeness lemma](https://github.com/verse-lab/veil/blob/300c305e945750ab3fb62de4a79c23161b24da39/Veil/Core/Tools/ModelChecker/Concrete/SequentialLemmas.lean)).
3. Veil defaults to trusting SMT UNSAT. Proof-grade jobs should request reconstruction and record any
   remaining assumptions. Lean-SMT itself is beta and may return residual Lean goals
   ([Veil SMT option](https://github.com/verse-lab/veil/blob/300c305e945750ab3fb62de4a79c23161b24da39/Veil/Base.lean),
   [Lean-SMT](https://github.com/ufmg-smite/lean-smt)).

These are reasons to integrate Veil carefully, not reasons to avoid it.

### 10.5 Veil integration design

Create an isolated project:

```text
tests/umpire3/model-checkers/veil/
  lean-toolchain
  lakefile.lean
  Umpire3Veil/
    Schema.lean
    Generated/
  testdata/
```

The main Lean project exports a versioned `FirstOrderView` artifact for one target. The adapter
generates a reviewable Veil module with the same state, action, invariant, and source identifiers.
The Veil runner returns normalized results and traces.

Do not attempt to translate arbitrary Lean functions or propositions. A target opts into the
first-order view by providing:

- finite or uninterpreted sorts;
- state relations/functions;
- action formulas;
- initial formula;
- invariant formula;
- a relation to the canonical feature/system model; and
- a theorem or explicit evidence classification for that relation.

The first integration supports four jobs:

| Job | Veil command/mode | Umpire classification |
| --- | --- | --- |
| Fast small-instance check | `#model_check` | `tested-instance`, collision-qualified |
| Symbolic depth check | `sat/unsat trace` | `bounded-safe(k)` or counterexample |
| Invariant discovery | CTIs plus `#check_invariants` | development evidence |
| Reconstructed invariant proof | `#check_invariants`, SMT trust disabled, all goals closed | `invariant-proved` with axiom inventory |

Every Veil counterexample must replay through the canonical Lean `ExecutableView`. Failure to replay
is a translator defect and fails the backend job.

### 10.6 Differential checking

For tiny worlds, compare:

- canonical reachable states from Umpire's exact explorer;
- Veil concrete states/traces where export permits;
- TLC reachable states; and
- selected hand-written golden traces.

Compare normalized semantic states, not printer output. The differential suite catches an exporter
that consistently omits the same transition from all generated properties.

### 10.7 Ivy decision

Ivy is a standalone protocol language and verifier, not a Lean model checker. Veil's language is
substantially derived from Ivy's RML and preserves its strengths: atomic actions, first-order
invariants, decidability-aware modeling, and CTI-driven proof development. The original Microsoft
repository is archived, with development continuing in another fork
([Ivy language guide](https://microsoft.github.io/ivy/language.html),
[Ivy CAV 2020 paper](https://www.wisdom.weizmann.ac.il/~padon/ivy-cav2020.pdf),
[archived repository](https://github.com/microsoft/ivy)).

Do not integrate Ivy into Umpire3. Use its examples as benchmark and modeling prior art. Adding Ivy
would introduce another source language and translation boundary while duplicating much of the Veil
workflow.

### 10.8 TLC and Apalache

Veil's documented checker focus is safety. TLC has mature finite-instance safety, liveness, and
fairness checking for TLA+; Apalache adds symbolic bounded and inductiveness checking
([TLA+ tools](https://lamport.azurewebsites.net/tla/tools.html?unhideBut=hide-tlc&unhideDiv=tlc),
[Apalache running guide](https://apalache-mc.org/docs/apalache/running.html),
[Apalache supported features](https://apalache-mc.org/docs/apalache/features.html)).

Generate a readable TLA+ view only after the canonical/executable model and Veil pilot are stable.
Treat TLC/Apalache success as external tool evidence. Import and replay counterexamples. A candidate
invariant becomes a Lean theorem only after Lean checks initialization, preservation, and implication
of the property.

### 10.9 Temporal logic in Lean

Veil's public README still lists liveness as future work, although its authors have demonstrated an
emerging interactive liveness workflow using Lentil. LeanLTL supports finite and infinite linear
traces and proof automation; Lentil formalizes a useful portion of TLA
([LeanLTL](https://github.com/UCSCFormalMethods/LeanLTL),
[Lentil](https://github.com/verse-lab/Lentil)).

Define Umpire behavior and fairness independently behind a small interface. Prototype adapters to
Lentil or LeanLTL, but do not let a young library own Umpire's core semantics. Near term:

- Lean/Veil inductive proofs for unbounded safety;
- TLC for finite-instance liveness and fairness;
- Apalache for supported bounded lasso checks;
- Lean temporal proofs for selected high-value progress arguments; and
- no unbounded liveness claim from a finite trace budget.

## 11. Exact exploration and proof-producing checking

### 11.1 Reference explorer

Replace path-tree enumeration with an exact BFS module:

```text
Explore(world, initials, successors, property, limits)
  -> Counterexample(shortest trace)
   | Exhausted(reachable set + closure certificate)
   | DepthComplete(k, frontier certificate)
   | ResourceLimited(reason, checkpoint)
   | InternalError
```

Use full state equality or collision buckets keyed by a hash and resolved by equality. The reference
version can favor clarity over maximum performance.

### 11.2 Certificate boundary

The moonshot design separates fast search from trusted checking.

An untrusted native/parallel/external explorer produces:

- canonical initial-state indexes;
- visited states or a compact state dictionary;
- predecessor/action edges for witnesses;
- a closed-successor certificate for exhaustive safety;
- the property result for every visited state;
- bounds, reduction metadata, and termination reason; and
- optional symmetry/POR witnesses.

A small Lean checker validates the certificate. If it validates, Lean derives a theorem for the
stated finite world or bounded depth. The search producer can then be optimized, parallelized, or
replaced without expanding the proof-critical code.

### 11.3 Counterexamples

A counterexample is easier to trust than a no-bug result. Replay it by checking:

1. the first state is initial;
2. every labeled successor is legal under canonical semantics;
3. the final state violates the named property; and
4. all world choices and assumptions match the artifact.

This produces a Lean-checked witness even when Veil, TLC, Apalache, fuzzing, or a live run found it.

### 11.4 Reductions

Add reductions in this order:

1. exact deduplication;
2. canonical identity/symmetry with a preservation theorem;
3. property-preserving abstraction;
4. partial-order reduction with an independence relation and ample/persistent-set conditions;
5. compositional assume/guarantee checking; and
6. distributed/parallel search with deterministic certificate merge.

A Boolean `PreservationChecked` is insufficient. Until a theorem or certificate exists, label the
reduction heuristic and forbid proof-grade completeness.

## 12. Observation and live conformance

### 12.1 Four layers

Separate observation into four modules:

1. **Source adapter** — obtains public history, RPC responses, telemetry, or in-process facts.
2. **Normalizer** — converts source-specific data into a stable typed evidence vocabulary.
3. **Interpreter** — generated from Lean, binds identities and derives abstract observations.
4. **Monitor/qualifier** — evaluates the property with evidence and profile requirements.

Only the first two are hand-written per source. The interpreter and monitor are semantic artifacts.

### 12.2 Typed raw facts

Replace `Observation{Satisfied bool}` as the primary semantic input with facts such as:

```text
HistoryEvent {
  event_type,
  event_id,
  workflow_id,
  run_id,
  attributes_view,
  source_identity,
  clock_domain,
  causal_references
}

MechanismReceipt {
  action,
  resource,
  attempt,
  owner_epoch,
  outcome,
  source_identity,
  source_sequence
}
```

Sensitive or unbounded fields remain opaque hashes with explicit disposition. The fact vocabulary
must preserve absence versus unknown.

### 12.3 Observation programs

Define a small, typed, total program language in Lean for:

- selecting facts by kind and identity;
- binding projected identities;
- following lineage and causal edges;
- comparing enums, hashes, and bounded values;
- checking source-local order;
- opening and closing evidence windows;
- deriving abstract state deltas; and
- returning `true`, `false`, `unknown`, or `conflict` with supporting fact IDs.

Prove the interpreter sound against a Lean evidence semantics. Generate a generic Go interpreter and
cross-language vectors. This is a deep module: a small interface hides the complexity currently
spread across observation switches.

### 12.4 Absence and quiescence

`stale-success-absent` cannot be established by not seeing a success at an arbitrary instant. Its
monitor must identify an authoritative cut, such as terminal history, a closed dispatch response, or
a bounded window whose closure is part of the property. Otherwise the result is `unknown`.

### 12.5 Implementation relation

Eventually, live evidence should constrain a set of compatible abstract traces rather than directly
set property Booleans:

```text
compatible(evidence, abstract_trace)
```

- `established`: every compatible trace satisfies the property and evidence is complete enough;
- `violated`: a qualified observed trace contradicts the property;
- `unknown`: both satisfying and violating traces remain compatible;
- `conflict`: the evidence is internally inconsistent.

The first implementation can use a deterministic interpreter, but its interface should leave room
for this set-of-traces semantics.

## 13. The scenario, campaign, and runtime relationship

The existing Go compiler, runtime, campaign, replay, profiles, participant, and canary modules remain
valuable. The model-checking architecture feeds them; it does not replace them.

### 13.1 One normalized trace

All search engines return one canonical trace representation:

```text
Trace {
  world,
  initial_state_digest,
  steps [{ action_id, arguments, choices, state_digest }],
  property,
  violation,
  assumptions,
  bounds,
  source_map
}
```

The trace compiler turns a semantic trace into a sparse or completed experiment. Runtime-learned
identities remain symbolic until evidence binds them.

### 13.2 Scenario compiler

The compiler should use generated declarations for syntax, dependencies, and type checking, but it
must not infer that catalog membership proves semantic reachability. Add optional model validation:

- replay each compiled path in the canonical executable view;
- reject a path that has no abstract execution;
- record when a live-only action is outside the executable view; and
- retain the abstract state/action source map.

### 13.3 Campaigns

Campaigns remain the orchestrator for templates, novelty, execution, minimization, and promotion.
Their candidate sources expand to:

- exact/Veil/TLC/Apalache counterexamples;
- satisfying traces selected for coverage;
- typed protobuf and parameter mutations;
- schedule/fault/topology holes;
- production evidence anomalies; and
- proof CTIs that are executable as finite scenarios.

Coverage novelty is derived from model declarations, not only author-declared labels.

### 13.4 Promotion

A promoted failure must preserve:

- the same property and model target;
- a Lean-replayed semantic counterexample when one exists;
- the same evidence predicate, not merely a status string;
- grounded identity/lineage constraints;
- a deterministic environment profile or declared uncontrollable schedule;
- complete cleanup; and
- source maps back to the model and discovered trace.

## 14. Deep modules and stable interfaces

The target architecture has six important deep modules.

### 14.1 `ModelFamily`

**Interface:** select a world and target; obtain feature/system semantics, properties, refinements,
and supported views.

**Implementation:** dependent Lean types, module composition, theorem registry, source mapping.

Callers do not learn backend syntax or internal model state layout.

### 14.2 `Explorer`

**Interface:** check one finite view with explicit property and limits; receive a structured result
and optional certificate/witness.

**Implementation:** BFS, state storage, predecessor graph, coverage, certificate production,
parallelization.

### 14.3 `Backend`

**Interface:** advertise capabilities, check a versioned view, return normalized results and traces.

**Adapters:** Veil, TLC, Apalache, future SAT/SMT engines.

The interface exposes trust mode, exactness, temporal support, and certificate support. It is not a
lowest-common-denominator `Run() bool`.

### 14.4 `ObservationInterpreter`

**Interface:** evaluate generated typed programs over an evidence graph.

**Implementation:** identity binding, order, causal traversal, closed-world windows, derived facts,
four-valued logic, supporting evidence.

### 14.5 `Verifier`

**Interface:** run a target through selected engines, replay traces, check certificates, and emit one
verification bundle.

**Implementation:** orchestration, cache, resource isolation, deterministic merging, trust ladder,
artifact retention.

### 14.6 `Runtime`

**Interface:** realize one compiled experiment against an environment and return raw evidence,
cleanup, and profile identity.

**Implementation:** existing environment/session, participant, Temporal, process, fault, and canary
machinery.

`Runtime` must get shallower semantically as the observation interpreter gets deeper.

## 15. Proposed repository layout

```text
tests/umpire3/
  model/
    Umpire3/
      Behavior.lean
      Execution.lean
      ExecutableView.lean
      FiniteView.lean
      Certificate.lean
      Safety.lean
      Temporal.lean
      Refinement.lean
      Composition.lean
      Observation.lean
      Declaration.lean
    Temporal/
      Feature/                 user-visible feature models
      System/                  independent mechanism models
      Refinement/              relations and simulation proofs
      Contracts/               theorem-backed shared mechanisms
      Observation/             typed fact interpretations
      Targets/                 finite/first-order/checker views
      API/                     selected wire projections

  model-checkers/
    veil/                      isolated pinned Veil project
    tla/                       generated TLA+, TLC configs, Apalache config

  verifier/                    checker portfolio orchestration
  certificate/                 native producer and Lean-checkable schema
  protocol/                    generated versioned artifacts
  compiler/                    sparse intent and trace compilation
  evidence/                    raw fact graph
  observation/                 generic generated-program interpreter
  runtime/                     realization orchestration
  temporal/                    Temporal source/driver adapters
  campaign/                    discovery and promotion
  canary/                      production authority controller
```

Do not move all current files at once. Create new modules for the pilot, then migrate model families
when the new interface demonstrates more leverage and stronger failures.

## 16. Result and trust model

### 16.1 Result ladder

| Result | Meaning | Main trust boundary |
| --- | --- | --- |
| `trace-witness` | This checked initial state and legal trace reaches a violation | Lean definitions/kernel; execution mode if native |
| `sampled-no-counterexample` | Selected randomized/simulated traces did not fail | sampler, seed, coverage; never complete |
| `bounded-safe(k)` | No encoded execution through depth `k` violates the property | encoding and solver/certificate mode |
| `finite-exhaustive(world)` | Every reachable state in one finite world was checked | exact state identity and checked closure certificate |
| `external-no-counterexample` | Named external engine found none in its scope | exporter and external engine |
| `invariant-proved` | An inductive invariant implies safety for the quantified scope | Lean kernel plus disclosed axioms/solver trust |
| `temporal-proved` | A temporal property holds under named fairness assumptions | behavior/fairness semantics and Lean proof |
| `refinement-proved` | Selected system behaviors are included in feature behavior | relation and Lean proof |
| `implementation-conforming` | One implementation run meets a property under a profile and evidence set | driver, observation sources/interpreter, model, bounds |
| `unknown` | Scope, evidence, solver, or resources cannot decide | explicit reason |

### 16.2 Trust badges

Every proof/check result records one of:

- `kernel`;
- `kernel-with-declared-axioms`;
- `reconstructed-solver-proof`;
- `trusted-solver`;
- `checked-certificate`;
- `external-tool`;
- `tested-instance`;
- `sampled`; or
- `heuristic`.

The badge is data, not prose.

### 16.3 Verification bundle

One immutable bundle contains:

- model-family, target, world, and property identities;
- transitive semantic/toolchain/descriptor digests;
- axiom inventory;
- checker name, version, options, and trust badge;
- bounds, fairness, reductions, and omissions;
- result and termination reason;
- normalized trace or certificate digest;
- trace replay result;
- generated experiment digests;
- implementation/profile/evidence results where run;
- cleanup and retention results; and
- source locations for declarations and properties.

The existing replay bundle and qualification receipt can evolve toward this envelope.

## 17. Developer workflow

### 17.1 Model author

The happy path should be:

```text
umpire3 model new nexus-cancellation-fencing
umpire3 model check nexus-cancellation-fencing --world smoke
umpire3 model veil nexus-cancellation-fencing --mode invariant
umpire3 model trace <counterexample> --emit-regression
```

Editor feedback should follow Veil's excellent pattern: start interpreted checking immediately, then
switch to compiled search when ready. Diagnostics name the feature action, system action, property,
world choice, and source line.

### 17.2 Test author

The existing generated domain facade remains the right surface:

```go
umpire3test.RequireRegression(t, scenario,
    umpire3test.WithEnvironment(factory),
)
```

The test author should not select theorem hashes, monitor programs, checker encodings, or evidence
rules. `Explain` adds the model trace and verification bundle identity.

### 17.3 CI tiers

| Tier | Trigger | Work |
| --- | --- | --- |
| Editor | save/focused command | Lean build, trace replay, tiny exact/Veil check |
| PR smoke | affected target | proofs, exact tiny worlds, Veil BMC/invariant, generated drift, focused Go tests |
| PR integration | selected changes | generated traces against local Temporal; negative controls |
| Nightly | all targets | larger worlds, TLC/Apalache, campaigns, mutation score, certificate checks |
| Release | candidate | all proofs, portfolio agreement, root parity, remote/black-box receipts |
| Canary | separately authorized | approved digest, bounded live conformance only; no proof tools in production |

Affected targets should be computed from the generated transitive declaration graph.

## 18. Implementation roadmap

The sequence starts from 1.2 and deliberately proves one vertical slice before broad migration.

### A0 — Claim and provenance hardening

**Goal:** make current artifacts say exactly what they establish.

**Work**

- Add trust badges and result classes to proof/check manifests.
- Inventory Lean axioms for every exported theorem.
- Generate transitive source/dependency digests.
- Rename current path/template exploration results so they cannot imply finite-state completeness.
- Reclassify composition and parity `WellFormed` results as metadata validation.
- Remove or connect dead assurance modules.
- Make coverage undefined for targets without generated denominators.

**Exit gate:** no manifest can obtain `proved`, `finite-exhaustive`, or `complete` from authored
strings or Booleans.

### A1 — Independent Nexus feature/system pilot

**Goal:** remove proof-by-construction refinement for the stale-completion slice.

**Work**

- Add `World`, `Behavior`, labeled executions, and `ExecutableView` without breaking old models.
- Re-express Nexus cancellation as an independent feature model.
- Re-express task/ownership/persistence mechanics without importing feature state.
- Define the relation and action mapping in the refinement module.
- Prove safety simulation.
- Define a guard-removal mutated system model.
- Compile a second existing feature/system pair against the API before stabilizing registration.

**Exit gate:** the sound system proves refinement; the mutation is executable but cannot satisfy the
refinement theorem and yields the known stale-success counterexample.

### A2 — Exact finite explorer

**Goal:** add honest, reusable state-space checking.

**Work**

- Implement deterministic exact BFS with collision resolution.
- Return shortest counterexamples, coverage, deadlocks, and explicit termination.
- Prove frontier/visited soundness and bounded completeness.
- Define the first closure certificate and Lean checker.
- Check the sound and mutated Nexus worlds.

**Exit gate:** the mutation's shortest trace is discovered automatically; the sound finite world
returns `finite-exhaustive` only after its certificate checks; injected hash collisions lose no
states.

### A3 — Veil backend

**Goal:** add symbolic and invariant-driven checking without changing semantic authority.

**Work**

- Pin Veil in an isolated project.
- Define `FirstOrderView/v1` and generate the Nexus pilot.
- Run `#model_check`, symbolic trace queries, and invariant checking.
- Expose trusted-SMT versus reconstructed mode.
- Import and replay Veil traces in canonical Lean.
- Differential-check tiny worlds against the exact explorer.

**Exit gate:** Veil rediscovers the mutation; its trace replays; the sound invariant closes with a
recorded trust mode; translation mutations are caught by differential tests.

### A4 — Typed observation semantics

**Goal:** remove property truth from Temporal adapters.

**Work**

- Define raw Nexus history/mechanism fact types.
- Define the Lean evidence interpreter and four-valued monitor.
- Generate the Go interpreter program and cross-language fixtures.
- Change the Nexus adapter to emit facts, not `Satisfied`.
- Model closed evidence windows for absence.

**Exit gate:** a deliberately wrong Go observation mapping fails fixtures; missing closure yields
`unknown`; stale success yields `violated`; the sound real-cluster path yields `established`.

### A5 — Theorem-backed catalog and composition

**Goal:** make semantic metadata a derived artifact.

**Work**

- Introduce typed property, action, theorem, guarantee, and requirement registrations.
- Resolve theorem names and axioms during Lean export.
- Replace author-supplied statement hashes.
- Compose task delivery with Nexus and Workflow ownership through actual predicates.
- Prove one interference-preservation result.

**Exit gate:** deleting or changing a provider theorem fails the consumer at Lean elaboration, not a
later hash comparison; the generated catalog retains the stable Go-facing IDs.

### A6 — Model-derived compiler and coverage

**Goal:** connect authoring, exploration, and model semantics.

**Work**

- Replay compiled scenario paths through executable views.
- Generate denominators for every qualifying target.
- Derive transition, relation, property, fault, observation, and refinement coverage.
- Replace symmetry/POR attestation Booleans with checked evidence.
- Feed exact/Veil traces into campaign and promotion.

**Exit gate:** every qualifying target has a nonempty derived denominator; a semantically impossible
Go scenario fails before allocation; promoted model-checker traces use normal `RequireRegression`.

### A7 — Temporal properties and independent portfolio

**Goal:** support real progress claims.

**Work**

- Add infinite behavior, enabledness, fairness, and leads-to semantics.
- Specify task delivery and recovery fairness explicitly.
- Generate a TLA+ view for the pilot and run TLC liveness checks.
- Add Apalache for supported bounded/inductiveness jobs.
- Prototype a Lean temporal proof with Lentil or LeanLTL behind an adapter.
- Add liveness-preserving refinement or explicitly limit the claim.

**Exit gate:** one bounded progress bug produces a replayable lasso; one selected progress theorem is
proved under named fairness assumptions; no finite result is reported as unbounded liveness.

### A8 — Proof-producing scale

**Goal:** make large search fast without growing the trusted base.

**Work**

- Implement a parallel native certificate producer.
- Add compact closure and symmetry certificates.
- Add checkpoint/resume with transactional artifact publication.
- Make deterministic merges independent of worker count.
- Benchmark state storage, certificate size, and Lean checking.

**Exit gate:** a 10x larger search completes through parallel production, the same small Lean checker
validates the result, and corrupted/partial certificates fail closed.

### A9 — Model-family migration and qualification

**Goal:** apply the architecture across Umpire3 without repeating the bootstrap's breadth-first leap.

**Order**

1. Workflow ownership fencing;
2. Workflow lineage;
3. routing and speculative delivery;
4. callback reference/response;
5. Nexus timeout and Nexus/Activity linking;
6. Update lifecycle;
7. cross-feature compositions; and
8. remaining parity targets.

Each migrated family must have independent feature/system semantics, a real refinement test, a
checker view, an evidence interpreter, non-vacuity, and a mutation. Old models remain until their
replacement meets the gate.

## 19. Verification plan

### 19.1 Lean kernel tests

- initial enumerator soundness and completeness;
- successor enumerator soundness and completeness;
- trace replay accepts every explorer/backend trace;
- illegal action/state mutation fails replay;
- independent system mutation breaks refinement;
- finite certificate checker rejects missing states, edges, property checks, and wrong worlds;
- theorem export contains the expected axiom inventory;
- composition fails for missing, circular, or interference-breaking guarantees; and
- temporal proofs name every fairness assumption.

### 19.2 Explorer tests

- acyclic, cyclic, nondeterministic, converging, and deadlocked graphs;
- multiple initial states;
- shortest counterexample stability;
- exact exhaustion versus depth completion;
- injected hash collisions;
- deterministic results across iteration and parallelism;
- timeout, memory, state, and cancellation termination; and
- corrupt/resumed checkpoint handling.

### 19.3 Veil adapter tests

- pinned toolchain and checksum/revision gate;
- generated source stability and source mapping;
- all canonical actions represented;
- trace replay both directions;
- known model mutation found;
- exporter mutation detected by differential reachable-state comparison;
- solver trusted/reconstructed classification; and
- timeout/unknown never converted to success.

### 19.4 Observation tests

- each raw fact variant and field disposition;
- absent, unknown, contradictory, duplicate, and out-of-order evidence;
- runtime-learned identity and lineage conflicts;
- cross-clock timestamps with no causal edge;
- authoritative source-local order;
- closed and unclosed absence windows;
- Lean/Go interpreter differential fixtures;
- evidence loss and byte-limit exhaustion; and
- property/evidence mutations.

### 19.5 End-to-end pilot tests

- sound model: refinement theorem, exact finite exhaustion, Veil invariant, conforming live run;
- mutated model: shortest counterexample in exact explorer and Veil, replay into Lean;
- faulty implementation: same semantic violation from real-cluster evidence;
- missing evidence: `unknown`, never conforming;
- model/compiler trace: emitted regression compiles and replays; and
- cleanup failure: retained as an orthogonal failure and disqualifies conformance where required.

### 19.6 Repository gates

Every implementation phase runs the smallest focused tests, then:

```sh
make umpire3-check
go test -tags test_dep <affected packages>
make fmt-imports
make lint-code
```

Use `integration` only for integration tests. Repository-wide `make unit-test` is the final broad
gate when feasible.

## 20. Performance and 10x scale

State explosion, not Lean syntax, is the governing scalability risk. Increasing several independent
domains by 10 can increase the state space by orders of magnitude.

Use a portfolio:

- exact explicit checking for small, high-value finite worlds;
- symbolic BMC for shallow bugs in larger domains;
- inductive invariants for unbounded safety;
- compositional refinement to avoid global products;
- symmetry and POR only with preservation evidence;
- TLC distributed/parallel modes for suitable finite TLA+ targets;
- sampled campaigns for implementation schedule/input diversity; and
- live monitoring bounded independently from offline checking.

At 10x candidates or evidence:

- cache by semantic target/world/property/backend digest;
- separate model compilation from search;
- shard search deterministically;
- merge certificates and corpus entries independent of completion order;
- apply backpressure before evidence or cleanup falls behind;
- cap state bytes as well as state count;
- retain frontier/termination metadata; and
- return `resource-limited`, never a weaker success definition.

## 21. Crash, cancellation, and recovery

- An interrupted explorer never emits exhaustive success.
- Publish verification bundles transactionally after their trace/certificate checks.
- A resumable checkpoint records search progress, not proof evidence.
- Solver timeout, `unknown`, crash, malformed output, or killed process maps to `unknown` or
  infrastructure failure.
- Backend subprocesses run with bounded CPU, memory, time, and output and without production
  credentials.
- Live cleanup keeps its independent bounded authority and recovery metadata.
- A verifier crash after live allocation cannot lose the resource ledger.
- Canary approval binds the semantic experiment, not a checker executable or arbitrary model input.

## 22. Security

- Do not send Temporal payloads, headers, credentials, customer metadata, or raw failures to solvers
  or model checkers.
- Generate checker inputs only from reviewed semantic views.
- Treat Veil, cvc5, TLC, Apalache, generated binaries, and proof metaprograms as build-time supply-
  chain inputs: pin, checksum, sandbox, and cap them.
- Record solver options and trust mode.
- Validate imported traces and certificates as hostile input.
- Keep concrete production identities in redacted realization records, outside semantic digests.
- A proved model says nothing by itself about authorization, protobuf compatibility, persistence
  schema, rate limits, performance, or side channels; retain specialized tests.

## 23. Trade-offs

### 23.1 Complexity

This architecture adds a real semantic kernel, checker views, and certificate protocol. That is
substantial. It removes more dangerous complexity: repeated state machines, self-attested manifests,
property-specific Go observers, backend-specific traces, and ambiguous claims.

The discipline is to add a view only when a second implementation exists or a checker needs it. Do
not build a universal IR capable of representing arbitrary Lean.

### 23.2 Performance

Exact equality and checkable certificates cost memory and output. Use them for proof-grade jobs.
Allow fingerprinted fast search for development, but label it. Search speed is not worth an
overstated completeness claim.

### 23.3 Proof maintenance

Independent feature/system models make refinement harder than carrying a product proof in each
transition. That difficulty is the value: it exposes abstraction mistakes. Keep models small,
compose them, and use Veil's CTI workflow to control proof cost.

### 23.4 Toolchain breadth

Veil plus TLC/Apalache adds operational overhead. Veil is required after A3; TLC/Apalache remain
target-specific until A7. Ivy is excluded. Lentil/LeanLTL remain adapter experiments until one earns
a stable role.

### 23.5 Source-of-truth tension

A reified first-order view can look like a second model. It is acceptable only when its relation to
the canonical model is explicit and checked or its result is classified as translation-dependent
external evidence. A generated backend file is never authoritative.

## 24. Risk register

| Risk | Failure | Mitigation |
| --- | --- | --- |
| Wrong feature contract | Proof establishes the wrong product meaning | domain review, reachable examples, real regression corpus, semantic mutations |
| Circular refinement | System is legal by construction | independent system state/step; mutation must break theorem |
| Vacuous invariant | Antecedent or threat unreachable | non-vacuity witnesses and model mutation gate |
| Exporter omission | All generated checks miss an action | canonical replay and differential tiny-state comparison |
| Hash collision | Distinct states merged | exact equality/collision buckets; certificate validation |
| Trusted SMT bug | False UNSAT accepted | trust badge, reconstruction, independent finite checks |
| Veil churn | Main Lean build destabilized | isolated pinned project and versioned seam |
| State explosion | Search cannot finish | composition, abstraction, BMC, invariants, honest resource results |
| Liveness overclaim | Finite check presented as eventuality | temporal result classes and fairness inventory |
| Observer restates oracle | Go produces desired Boolean | typed raw facts and generated interpreter |
| Missing-event “success” | Absence mistaken for proof | closed evidence windows and four-valued result |
| Manifest laundering | String says theorem/complete | typed registrations, name resolution, axiom inventory |
| Production leakage | Sensitive data reaches checker/artifact | selected views, field dispositions, redaction, sandbox |

## 25. Definition of done

The architecture is realized when all of these hold.

### Semantic authority

- Every qualifying target resolves to typed feature/system/property declarations.
- No system step stores a feature run or feature-state witness.
- Every system/feature pair has an independent mutation that breaks refinement.
- Exported theorem names resolve, their axioms are inventoried, and hashes are derived.

### Checking

- Every qualifying finite target has an exact executable view.
- Proof-grade finite success comes from a checked certificate with collision-safe state identity.
- Veil runs at least concrete, symbolic trace, and invariant jobs for supported targets.
- Every backend counterexample replays through canonical Lean semantics.
- Temporal claims distinguish finite lasso evidence from Lean liveness proof.

### Feature/system support

- Multiple system realizations can refine one feature contract.
- A shared system contract supports at least two feature families through actual theorems.
- At least one composed target proves interference preservation.
- Target omissions have property-preservation evidence or are explicitly heuristic.

### Live conformance

- Temporal adapters emit typed raw evidence rather than property truth.
- Generated observation programs evaluate identically in Lean and Go.
- Absence, missing evidence, contradiction, and clock ambiguity cannot establish conformance.
- The same normalized semantic counterexample can originate from a checker or a live run and enter
  the same minimization/replay/promotion path.

### Developer and operational quality

- A model author can run a tiny exact/Veil check from one command with source diagnostics.
- A test author continues using the generated domain facade.
- PR checks select affected targets from semantic dependencies.
- Nightly checks scale independently and produce deterministic bundles.
- Canary execution remains digest-bound, process-isolated, redacted, and separately authorized.

## 26. The moonshot end state

The best version of Umpire is a continuously running assurance system for Temporal:

1. A developer changes a feature or mechanism.
2. Umpire identifies affected model families from typed dependencies.
3. Lean checks semantic, refinement, composition, and observation theorems.
4. The exact explorer and Veil search small worlds immediately.
5. TLC/Apalache and larger native certificate producers run in the portfolio when useful.
6. Every discovered trace is normalized and replayed in canonical Lean.
7. Selected traces become real Temporal experiments with symbolic identities and first-class faults.
8. Independent evidence either establishes, violates, or leaves the claim unknown.
9. Stable violations are minimized and emitted as ordinary regression source.
10. Production evidence can challenge the model and seed new worlds without running proof tools in
    production.
11. The release assurance graph shows which contracts are proved, finitely exhausted, externally
    checked, live-conforming, partial, or unknown.

The outcome is not “Temporal proved correct.” It is more useful and more honest:

> Temporal's important behavioral contracts have explicit semantics; selected mechanisms are proved
> to refine them; multiple engines aggressively search those semantics; every counterexample can
> become a real test; every live claim is evidence-qualified; and the trust boundary of every green
> result is mechanically visible.

## 27. Primary sources

The checker decisions are grounded in primary project documentation and papers:

- [Veil 2 README and status](https://github.com/verse-lab/veil/blob/300c305e945750ab3fb62de4a79c23161b24da39/README.md)
- [Veil DSL and checking modes](https://github.com/verse-lab/veil/blob/300c305e945750ab3fb62de4a79c23161b24da39/docs/DSL-Reference.md)
- [Veil CAV 2025 paper](https://verse-lab.org/papers/veil-cav25.pdf)
- [Loom POPL 2026 paper](https://verse-lab.org/papers/loom-popl26.pdf)
- [Lean-SMT](https://github.com/ufmg-smite/lean-smt)
- [Ivy language documentation](https://microsoft.github.io/ivy/language.html)
- [Ivy CAV 2020 paper](https://www.wisdom.weizmann.ac.il/~padon/ivy-cav2020.pdf)
- [TLA+ tools and TLC](https://lamport.azurewebsites.net/tla/tools.html?unhideBut=hide-tlc&unhideDiv=tlc)
- [Apalache](https://apalache-mc.org/)
- [LeanLTL](https://github.com/UCSCFormalMethods/LeanLTL)
- [Lentil](https://github.com/verse-lab/Lentil)

The local architecture findings are grounded in:

- `tests/umpire3/model/Umpire3/{Transition,Executable,Refinement,Catalog,Composition,Monitor}.lean`;
- `tests/umpire3/model/Temporal/{Monitors,Composition,Coverage,Parity}.lean`;
- `tests/umpire3/model/Temporal/{Product,System,Refinement}`;
- `tests/umpire3/{compiler,explore,evidence,runtime,temporal,campaign,canary}`;
- `tests/umpire3/protocol/generated`;
- `tests/umpire3/migration/ledger.json`; and
- `tests/umpire3/testdata/umpire3-1.2.json`.
