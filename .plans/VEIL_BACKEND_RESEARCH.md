# Veil as Umpire's semantic or checking backend

> The Umpire3 sources this note cites were removed by fn-81. They are named below as provenance
> rather than as links.

## Scope and source pin

This note evaluates code reuse, not only surface syntax. It inspects
`verse-lab/veil` at commit
[`be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd`](https://github.com/verse-lab/veil/commit/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd)
(2026-09-01). The source was read through `gh`; Veil was not installed or built. Its current `main`
is explicitly a Veil 2.0 prerelease, pins Lean 4.32.0, and depends on pinned `lean-smt` and Loom plus
a Node-built widget, while Umpire pins Lean 4.33.1. Compatibility is therefore unknown, not known to
fail. ([Veil README](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/README.md),
[Veil toolchain](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/lean-toolchain),
[Veil Lake configuration](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/lakefile.lean),
[Umpire toolchain](../model/lean-toolchain))

The conclusion changes materially if Umpire's authoring rules are open to revision: Veil can own
more than an optional checker. It still cannot replace the distinctive runtime/evidence half of
Umpire, and adopting it as semantic authority requires a new proof boundary that Veil does not
currently supply.

## What Veil actually exposes

Veil has three separable layers.

1. Its small semantic kernel is real Lean data. `RelationalTransitionSystem ρ σ l` contains
   `assumptions`, `init`, and labeled `tr`; it defines `next` and inductive reachability.
   `EnumerableTransitionSystem` contains enumerable initial states and labeled execution outcomes
   (`success`, assertion failure, or divergence), and Veil proves that its successful-transition
   reachability agrees with the relational view produced by `toRelational`.
   ([transition-system types and theorem](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/TransitionSystem.lean))
2. Its imperative action semantics are reusable below the DSL. `VeilM` is a nondeterministic
   computation over immutable theory, mutable state, exceptions, and possible divergence. Loom
   supplies the nondeterminism and weakest-precondition algebras. `VeilM.toTransition` gives an
   angelic relational meaning, while demonic weakest-precondition interpretations support invariant
   and assertion obligations. This is semantic machinery, not merely commands.
   ([action semantics](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Frontend/DSL/Action/Semantics/Definitions.lean),
   [Loom dependency](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/lakefile.lean))
3. The module/action DSL is elaborator-owned. It keeps an in-memory `Module` representation, then
   generates named Lean declarations including `relationalTransitionSystem` and
   `enumerableTransitionSystem`. Symbolic trace commands look up the current module and construct
   formulas directly against the generated `relationalTransitionSystem`; they are not a function
   accepting an arbitrary transition-system value. The concrete checker has a more reusable
   function API over `EnumerableTransitionSystem`.
   ([module representation](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Frontend/DSL/Module/Representation.lean),
   [relational assembly](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Frontend/DSL/Module/Util/Assemble.lean),
   [enumerable extraction](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Frontend/DSL/Action/Extract.lean),
   [symbolic trace elaborator](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/Symbolic/TraceLang.lean))

The concrete breadth-first checker is useful and reasonably modular. Its result distinguishes a
found safety/deadlock/assertion violation, complete reachable-state exploration, depth-bound exit,
and cancellation; a violation can carry the full labeled state trace reconstructed from the search
log. The trace type separately defines `Trace.isValid`, but `findReachable` returns an ordinary
`ModelCheckingResult`, not a dependent result carrying a validity or completeness proof. The search
implementation maintains proof-indexed internal invariants, but the public result is still an
executable checker receipt rather than a kernel theorem.
([checker entry point](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/Concrete/Checker.lean),
[result vocabulary](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/Interface.lean),
[trace validity](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/Trace.lean))

The symbolic and invariant pipelines have different assurance. `veil.smt.trust` defaults to true;
then unsatisfiability is trusted. With it false, Veil asks `lean-smt` to reconstruct a Lean proof.
SAT results retain a structured SMT model/JSON counterexample. An inductive-invariant failure can
be a counterexample to induction whose pre-state is not reachable, so it must not be promoted as a
reachable Umpire violation. A bounded `sat/unsat trace` formula includes initialization and each
transition and is the suitable symbolic source of reachable bounded traces.
([trust option](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Base.lean),
[SMT invocation](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Frontend/DSL/Tactic.lean),
[SMT result and structured models](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Backend/SMT/Result.lean),
[induction VC generation](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Frontend/DSL/Module/VCGen/Induction.lean),
[bounded trace VC generation](https://github.com/verse-lab/veil/blob/be6a1ceebd103d05e1e0f0863e8bf73db1ea9ccd/Veil/Core/Tools/ModelChecker/Symbolic/TraceLang.lean))

Veil currently generates the enumerable system by extracting executable actions and independently
generates the relational system from action transition meanings. The inspected transition-system,
action-extraction, assembly, and model-checker paths contain a theorem from an arbitrary enumerable
system to that system's own relational view. I did not find a theorem in those paths that the
generated enumerable system equals or refines the independently generated
`relationalTransitionSystem`. Establishing that extraction correspondence is the largest trust gap
if Veil becomes Umpire's authority.

## The exact Umpire seam

Umpire's `TransitionKernel` carries more semantic coordinates than a basic Veil transition: Setup,
State, Action, Model Outcome, and ordered Observations, along with authoritative relations and
sound/complete finite lists. Its `TargetBehaviorDomain` separately enumerates all five domains and
provides stable encoders. Target admission requires a complete behavior domain and materializes the
state-by-action transition rows for the behavior fingerprint. A symbolic backend therefore cannot
simply replace the planner; admission and fingerprinting currently require complete finite
materialization. ([Umpire core](../model/Umpire/Core.lean),
[target admission and fingerprinting](../model/Umpire/Target/Language.lean))

A faithful adapter must use a label such as
`{ action : Action, outcome : Outcome, observations : List Observation }`, rather than only Action,
and establish these equivalences:

```text
veil.assumptions setup              <-> Umpire setup is admitted
veil.init setup state               <-> kernel.authoritativeInitial setup state
veil.tr setup s {a, o, obs} s'      <->
  kernel.authoritativeStep s a { modelOutcome := o,
                                 resultingState := s',
                                 observations := obs }
```

For concrete checking it must additionally prove list membership agrees with those propositions,
that state fingerprints are injective on the admitted finite domain, and that Veil's depth is the
same semantic transition coordinate as the Umpire Query limit. For each supported Property it must
prove that the Veil invariant or bounded-trace formula is equivalent to
`CheckedProperty.denote` on encoded traces. Umpire already proves its executable Property evaluator
agrees with that denotation, so this theorem composes the two meanings instead of reimplementing
the Property interpreter. ([Property evaluator agreement](../model/Umpire/Property/Evaluation.lean))

A visited-state checker cannot evaluate a history-sensitive Umpire Property from model state alone.
The adapted state must be a product of Target state, Behavior/scenario progress, Property monitor
state, and every bound coordinate that affects acceptance. Veil's `ExecutionOutcome` means
successful execution versus assertion failure/divergence; it is not Umpire's domain-level Model
Outcome. That semantic Outcome and ordered Observations must remain in the enriched transition label
or product state.

Every decoded counterexample must then pass Umpire trace admission and Exact Replay. Unknown,
timeout, depth-bound, cancellation, trusted-unsat, reconstructed-unsat, complete finite search, and
counterexample replay need separate receipt statuses/assurance methods. Veil's UI JSON is useful for
display, but stable Definition IDs, Behavior Fingerprints, Limits, Known Gaps, and source bindings
must come from Umpire's receipt layer. These obligations match the current optional-verification
rules, but remain necessary even if those rules are rewritten because they prevent semantic drift.
([Umpire verification rules](UMPIRE4_SPEC.md#verification-cli-and-claims))

## Three adoption paths

### 1. Veil as an optional Umpire backend

This is immediately plausible for opted-in safety Properties and bounded trace queries. Keep
`CheckedTarget`, Property/Behavior/Query, current planning, fingerprints, and all Case/runtime code.
Add the labeled transition adapter, a small supported-Property compiler, result/receipt mapping, and
Exact Replay. The concrete checker can consume an enumerable adapter directly. Symbolic trace and
invariant checking either need a Veil contribution that exposes function APIs, or an Umpire module
that generates the expected declarations/VCs; calling today's commands programmatically would bind
Umpire to Veil's frontend internals.

No current Umpire module can honestly be deleted on this path. For supported exhaustive safety
queries, Veil can be selected instead of the traversal inside `Planning/Engine.lean`, but that file
also owns Query forms, deterministic candidate ordering, work limits, instrumentation, statuses,
and artifacts. The saving is avoided future work: Veil's concrete checker/search implementation and
symbolic trace frontend need not be recreated. Umpire still adds adapters, receipts, and translation
proofs. This is a capability gain rather than demonstrated source reduction.

### 2. Veil as Umpire's semantic core

Make `RelationalTransitionSystem`/`VeilM` the representation beneath Umpire, while Umpire retains
its public concepts and serializable views. This could replace the proposition half of
`TransitionKernel` and much of `FiniteMachine`/`FiniteTable`; Veil's action language and WP
machinery could also replace a future custom imperative action AST. The concrete checker could
replace the mechanics of finite breadth-first reachability.

The directly overlapping Umpire code is concentrated in `Target/Language.lean`,
`Target/FiniteMachine.lean`, and `Target/FiniteTable.lean`, plus part of `Core.lean` and the
traversal mechanics in `Planning/Engine.lean`.
Stable IDs, canonical encoders, capability contracts/laws, outcomes, ordered facts, explicit finite
coverage, deterministic fingerprinting, Behavior narrowing, Query forms/limits, and result artifacts
still need Umpire-owned wrappers. Veil's fixed-theory enumerable checker also does not enumerate
Umpire Setup values. Net saving cannot be claimed until a spike shows which current admission
structures can be deleted rather than duplicated.

This path does not replace `Property/Check.lean` plus `Property/Evaluation.lean`: Veil invariants
cover state safety and its trace formulas cover fixed
bounded patterns, while Umpire has identity, ordering, same-step cases, guarded bounded temporal
clauses, exact diagnostics, and a proven evaluator/denotation agreement used outside model checking.
It also does not replace Behavior and Query semantics, Case production, Observation correlation, or
Testpilot.

The remaining central proof is an isomorphism/refinement between Umpire's enriched transition label
and Veil's relation, plus a theorem that every enumerable action execution denotes exactly that
relation. Without the latter, adopting Veil below `CheckedTarget` weakens Umpire's current
sound/complete enumerator boundary.

The retired Umpire3 integration is concrete evidence of this cost. Its generic
`SemanticRelation` required `initial_iff`, `next_iff`, `property_iff`, injectivity, and completeness,
and the Nexus cancellation target then supplied lengthy per-model symbolic/concrete equivalence
proofs. Maintaining both Umpire and Veil representations can consume most reuse savings. This is a
warning about the adapter burden, not a recommendation to restore the retired runtime.
(the Umpire3 Veil semantic relation `Umpire3/Veil/Semantics.lean` and the Nexus concrete
semantics proof `Temporal/Families/NexusCancellation/Targets/Veil/SoundConcreteSemantics.lean`,
both under the `tools/umpire3` tree fn-81 removed)

### 3. Veil source authority with Umpire runtime extensions

Let engineers author state, theory, init, actions/transitions, and safety invariants in Veil. Derive
the Umpire canonical model view, Behavior/Query data, Case Programs/Contracts, and runtime evidence
mapping from that authority. Given the user's willingness to change Umpire DSLs, this is the path
with the largest potential simplification: it can retire most Umpire Target authoring and its
finite-machine adapter, avoid building a separate action AST, and replace the state-safety subset of
Property authoring/checking. It also requires new metadata, extraction, receipt, temporal-property,
and runtime-lowering code. The inspection establishes overlapping responsibilities, not a measured
net reduction.

This choice removes the need to prove that two independently authored transition relations agree:
the Umpire view is derived from Veil. It does not remove the derivation proof. Umpire must prove or
check that the derived stable IDs, encodings, outcomes/facts, finite domains, and serialized traces
faithfully represent Veil declarations. Most importantly, it needs the missing equivalence between
Veil's extracted enumerable actions and its relational semantics. Replaying every emitted trace
against the relational system can validate counterexample soundness, but cannot establish exhaustive
search completeness or an unsatisfiability claim. Until Veil provides the correspondence theorem,
Umpire must supply it for complete results or label enumeration as a lower-trust executable-checker
assurance; replay remains an additional gate for emitted counterexamples.

Umpire runtime extensions remain substantial and product-specific: Action classification as command
or bounded wait, target-owned nondeterministic outcomes, Behavior scenario selection, per-stage
Limits, stable artifact identity, Program compilation, deterministic Contract monitors, Observation
projection/correlation, append-only Run Events, Driver authorization, Verdicts, and Known Gaps. The
current Case compiler is only 97 lines and merely assembles already-lowered monitor rules; the major
future cost is exact Property-to-Contract lowering, which Veil does not provide.
([Case compiler](../model/Umpire/Case/Compiler.lean),
[Nexus command/observation boundary](../model/Temporal/Feature/Nexus3/Integration.md))

## Recommendation

Use a two-stage decision rather than treating optional integration as the final architecture.
First, prototype a direct data adapter from one Nexus race model to Veil's
`RelationalTransitionSystem` and `EnumerableTransitionSystem`, with the enriched label above. Require
four outputs: transition equivalence, property equivalence for one safety Property, a decoded
counterexample that passes Exact Replay, and distinct complete/depth/unknown/trust receipts. This
tests the reusable core without committing to Veil's frontend internals.

If that works, evaluate path 3 as a deliberate DSL replacement. It offers the only material source
reduction and lets Umpire avoid owning another imperative action language. Make adoption contingent
on an extraction-correctness theorem or an accepted weaker assurance boundary, a stable callable VC
API, Lean 4.33 compatibility, and a demonstrated encoding of Outcome plus ordered Observation facts.
Otherwise retain path 1. Path 2 has the highest migration risk relative to likely code savings: it
changes Umpire's core while preserving nearly all of Umpire's public semantic and runtime layers.

Veil can therefore save real implementation work, especially action/WP infrastructure and model
checking. It does not collapse Umpire into a thin runtime adapter. The irreducible work is the exact
bridge from model transition coordinates and Properties to canonical artifacts and evidence-aware
runtime monitoring.
