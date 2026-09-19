# Nexus3 DSL and verification framework research

Research date: 2026-09-06. This note evaluates existing facilities against the proposed syntax in
[`Nexus.md`](../model/Temporal/Feature/Nexus3/Nexus.md) and its proposed Case boundary in
[`Integration.md`](../model/Temporal/Feature/Nexus3/Integration.md). The Nexus3 files are design
specimens: the proposed Markdown blocks do not parse as Lean and have no implemented compiler.
The separate `Nexus.lean` already contains a narrower executable success model. Nothing below
is an adoption decision. No external toolchain was installed or compatibility-tested.

## Required semantic baseline

The desired surface is more than state-machine notation. It must retain separate Target, Property,
Behavior, and Query meanings; let the Target choose outcomes for requested Actions; distinguish a
command from an observed result; correlate runtime Observations to the right operation; reject an
unsupported Property rather than weaken it; and make finite search, bounded progress, and
unbounded liveness distinct claims. In particular, Nexus3's `within 1 operation_transition` is a
finite-trace obligation in semantic steps. It is neither a millisecond deadline nor an unbounded
fairness claim.

## Lean's native extension and verification facilities

Lean already supplies the mechanisms needed to implement the proposed surface inside Lean. Its
extensible parser accepts custom syntax; hygienic macros translate syntax to existing syntax while
automatically propagating source positions; command elaborators can add declarations and report
source-located errors; and term elaborators produce core expressions in the current local context.
These facilities establish that `model`, `property`, `behavior`, and `query` blocks can be Lean
frontends over existing Umpire owners. They do not define the DSL's semantics, prove that lowering
is faithful, or establish that the result is usable by ordinary Temporal engineers.
([Lean notation and macro overview](https://lean-lang.org/doc/reference/latest/Notations-and-Macros/),
[elaborators](https://lean-lang.org/doc/reference/latest/Notations-and-Macros/Elaborators/))

Lean's built-in `mvcgen` is narrower than the proposed model DSL. It reinterprets monadic programs
through weakest-precondition instances, decomposes Hoare triples into verification conditions, and
uses registered `@[spec]` lemmas for operations such as `bind`, `pure`, and loop traversal. It is
extensible to custom monads through `WP` and `WPMonad`. This can help prove an imperative Producer,
interpreter, or other monadic implementation, but it does not provide Umpire's state-machine
vocabulary, target-owned nondeterministic outcomes, Behavior selection, bounded trace search,
runtime Observation correlation, or Case generation. It is proof infrastructure, not an existing
replacement authoring language.
([`mvcgen` overview](https://lean-lang.org/doc/reference/latest/The--mvcgen--tactic/Overview/),
[verification-condition algorithm](https://lean-lang.org/doc/reference/latest/The--mvcgen--tactic/Verification-Conditions/),
[official tutorial](https://lean-lang.org/doc/tutorials/latest/mvcgen/))

The practical conclusion is to build any focused syntax as a thin, typed frontend that elaborates
to the existing Umpire semantic declarations and checked path. Lean makes that technically
possible. A prototype still has to demonstrate one-to-one lowering, source diagnostics, editor
behavior, deterministic identity, proof/trust discipline, and human usability.

## Veil

[Veil](https://github.com/verse-lab/veil) is the closest existing Lean framework. Its implemented
DSL declares a transition-system module, mutable state and immutable theory, initial states,
imperative atomic `action`s, relational two-state `transition`s, safety properties/invariants, and
verification commands. `#check_invariants` attempts inductive-invariant proofs with SMT and supports
interactive Lean proofs for obligations automation does not discharge. `#model_check` exhaustively
enumerates reachable states for a supplied finite instantiation and reports concrete traces;
symbolic `sat trace` and `unsat trace` queries use SMT for a specified bounded number of actions.
([Veil DSL reference](https://github.com/verse-lab/veil/blob/main/docs/DSL-Reference.md),
[project README](https://github.com/verse-lab/veil/blob/main/README.md))

That is substantial reusable verification machinery, but it does not already implement the Nexus3
contract:

- Veil's public DSL combines module state, externally invoked actions, transitions, invariants, and
  checker commands in its own language. Umpire requires its existing Property, Behavior, and Query
  languages to remain the sole public owners of those meanings, so a Veil model cannot become a
  second behavioral authority. A checked canonical-view binding is still necessary.
- A Veil action is an environment-invoked atomic model transition. Nexus3 separately classifies an
  Action as a command or a wait and advances only from correlated observed evidence. Veil does not
  supply Testpilot's Driver, Program, Observation correlation, append-only Run Events, deterministic
  Contract monitor, or Case admission.
- Veil's explicit checker is exhaustive only for the concrete finite instantiation supplied to
  `#model_check`. Its symbolic trace queries are bounded by action count. Neither result establishes
  that a real Temporal command produced the corresponding observed outcome.
- The current README describes proving safety and says liveness is future work. Nexus3's bounded
  progress can be encoded as a finite trace/monitor question only after an exact, operation-scoped
  mapping is defined; it must not be reported as unbounded liveness.

Veil also preserves an important assurance distinction. With `set_option veil.smt.trust true`, its
own regression fixture expects a warning that SMT results are trusted; setting it to `false` enables
proof reconstruction. Umpire must therefore report trusted-SMT and reconstructed/kernel-checked
results as different Assurance Methods and still replay counterexamples through canonical Umpire
semantics.
([trust-mode regression fixture](https://github.com/verse-lab/veil/blob/main/VeilTest/WarnTrustingSmtSolver.lean),
[SMT result categories](https://github.com/verse-lab/veil/blob/main/Veil/Backend/SMT/Result.lean))

Adoption is not currently demonstrated. Veil's `main` identifies itself as a Veil 2 prerelease with
known rough edges, has no published GitHub release, and pins Lean 4.32.0; Umpire pins Lean 4.33.1.
Its build also includes Lean SMT/Loom dependencies and a Node-built UI widget. These are compatibility,
reproducibility, and maintenance questions for an isolated adoption spike, not evidence against the
approach.
([Veil toolchain](https://github.com/verse-lab/veil/blob/main/lean-toolchain),
[Lake configuration](https://github.com/verse-lab/veil/blob/main/lakefile.lean),
[Umpire toolchain](../model/lean-toolchain))

Veil is therefore a credible optional safety checker and a valuable DSL implementation reference.
It does not remove the need for the Umpire authoring frontend, checked view correspondence, bounded
Query semantics, Case compiler, or evidence-aware runtime.

## `lean-tla`

[`yihuang/lean-tla`](https://github.com/yihuang/lean-tla) is an active exploration workspace for a
deep embedding and a native TLA-flavored Lean DSL, not an established library release. Its README
reports a working prototype with infinite behaviors, `□`, `◇`, leads-to, stuttering, weak/strong
fairness, refinement examples, and kernel-checked liveness rules. Its finite model checker contains
soundness theorems that turn successful finite Boolean checks into Lean theorems about the embedded
semantics; this is materially different from merely trusting an external search result.
([README](https://github.com/yihuang/lean-tla/blob/main/README.md),
[finite model checker and soundness proofs](https://github.com/yihuang/lean-tla/blob/main/TlaDsl/ModelCheck.lean),
[temporal and fairness rules](https://github.com/yihuang/lean-tla/blob/main/TlaDsl/Rules.lean))

This work is useful as a source of ideas for stuttering, fairness, refinement, and a proof-producing
finite checker. It does not implement Umpire's separate DSLs, exact finite Limits and statuses,
command/Observation boundary, or Case runtime. Its unbounded `eventually` and fairness operators
also answer a different question from Nexus3's finite `within 1 operation_transition`; exposing them
as ordinary Umpire progress would conflict with the bounded-progress rule.

The repository currently pins a Lean 4.33.0 release candidate while Umpire uses 4.33.1, depends on
CSLib `main` and an unpinned Mathlib declaration, and has no GitHub release. Treat it as research code
until API, dependency, and toolchain stability are evaluated.
([toolchain](https://github.com/yihuang/lean-tla/blob/main/lean-toolchain),
[Lake configuration](https://github.com/yihuang/lean-tla/blob/main/lakefile.lean))

## P and TorXakis as model-based-testing references

[P](https://p-org.github.io/P/) provides a mature separate language for asynchronously communicating
state machines, nondeterministic test harnesses, and observer `spec` machines. P test cases define
finite scenarios and ask the P Checker to explore them; monitors synchronously observe declared
events and express safety assertions or hot/cold-state liveness. This validates several authoring
ideas: keep executable machines, nondeterministic environments, scenarios, and observers explicit,
and prevent observers from changing the system.
([P program structure](https://p-org.github.io/P/advanced/structureOfPProgram/),
[test cases](https://p-org.github.io/P/manual/testcases/),
[monitors](https://p-org.github.io/P/manual/monitors/))

P does not cover Umpire directly. Its model is a separate behavioral source rather than Lean data;
its monitors synchronously observe model `send`/`announce` events, while Nexus3 must reconstruct
facts from correlated Temporal runtime evidence; and hot-state eventuality is not Nexus3's explicit
numeric operation-transition bound. P can inspire scenario composition and monitor UX, but adopting
it as the model authority would violate Umpire's Lean-authority and single-authoring-path rules.

[TorXakis](https://github.com/TorXakis/TorXakis) is closer to black-box model-based execution. A
model declares the permitted input/output behavior, a connection binds the tool to a System Under
Test, and `test N` performs a requested number of test steps, reporting pass or a model mismatch.
That command-versus-output split is relevant to Nexus3: an input sent to the SUT and an output
observed from it are different events.
([official getting-started guide](https://torxakis.org/userdocs/stable/getting-started.html),
[repository README](https://github.com/TorXakis/TorXakis/blob/develop/README.md))

TorXakis still does not supply Lean-kernel proofs, Umpire Definition IDs/fingerprints, exact bounded
Query claims, Testpilot Case admission, or Temporal-specific correlation. It also brings another
behavioral language and an SMT-backed Haskell toolchain. The latest stable release is v0.9.0 from
2020 even though the development repository has newer commits, so present compatibility and support
would need evaluation.
([v0.9.0 release](https://github.com/TorXakis/TorXakis/releases/tag/v0.9.0))

## Conclusions for Nexus3

1. Lean already covers the frontend mechanics. Umpire still owns the semantic design and proof that
   friendly syntax lowers exactly to checked declarations.
2. Veil is the strongest existing candidate for optional safety verification and for studying a
   embedded DSL. Bind it to a canonical view; do not make it a second model or runtime.
3. Keep assurance labels exact: finite enumeration, bounded SMT trace search, trusted SMT,
   reconstructed/kernel proof, and concrete execution are different evidence.
4. `lean-tla` offers promising proof-level ideas for temporal semantics and unbounded liveness, but
   it is an exploratory project and its claims must remain separate from Umpire's finite bounded
   progress.
5. P and TorXakis offer useful model-based-testing patterns, especially nondeterministic scenarios,
   passive monitors, and commands versus observed outputs. Neither satisfies Umpire's Lean authority,
   canonical artifacts, or evidence correlation without a new checked integration layer.

## External language and framework comparisons

The following are design recommendations inferred from documented capabilities, not tested integrations.

| Tool | Existing capability | Most useful transfer to Nexus3 |
| --- | --- | --- |
| Quint | Typed state/action language, explicit nondeterminism, run composition, simulator, checker integration | Readable action scenarios and one model feeding exploration and regression. |
| Quint Connect | Rust library that replays generated Quint traces through a Driver and compares projected implementation state | A small explicit action/evidence bridge and understandable divergence reports. |
| Alloy 6 | Relational constraints, assertions and analysis commands, temporal traces, scopes, adjacent-state visualizer and trace forks | Separate assumptions from checks; make witness and counterexample exploration first-class. |
| TLA+/PlusCal | State/action and temporal reasoning, explicit fairness, refinement with stuttering | Define the relationship between low-level events and feature-level steps precisely. |
| Ivy | Interface specifications as monitors; generated constrained-random testers; assume/guarantee isolates | Capability Contracts with explicit input assumptions and output guarantees, independently testable components. |
| Dafny | Preconditions, postconditions, frames, ghost definitions, module refinement | Small semantic interfaces and explicit obligations for transformations between representations. |

### Quint and Quint Connect

Quint's language distinguishes actions from runs and temporal expressions; runs support sequencing
and expectations, while nondeterministic choice is explicit. That resembles Nexus3's authoring
problem, although a Quint run is not automatically the same as a declarative Umpire Behavior.
Its temporal operators describe infinite executions, so their presence does not directly implement
operation-scoped finite progress. [Quint language reference](https://quint.sh/docs/lang).

Quint Connect is a concrete model-based testing framework for Rust. Its Driver steps through model
actions, a State projection obtains implementation state, and the framework compares states.
It retains action names and nondeterministic picks and supports seeded trace reproduction.
This provides an implementation pattern to study, not a ready-made Go/Testpilot integration.
[Quint Connect API](https://docs.rs/quint-connect/latest/quint_connect/).

For Nexus, distinguish test-controlled choices from system-produced results before replay.
A sampled canceled model outcome cannot authorize a driver to force cancellation to win.
The integration needs to recognize either admitted observed resolution; a particular witness
might fail to materialize without implying a safety violation. Quint's broader documentation
also distinguishes model-based testing from validating implementation traces against a model.
[Quint model-based testing](https://quint.sh/docs/model-based-testing).

### Alloy 6

Alloy separates constraints/facts, predicates, assertions, and run/check commands. This is a useful
analogy for Model + Behavior versus Property + Query, not a direct translation of each keyword.
An Alloy fact is an assumption restricting the solution space; an Umpire Fact is a transition's
semantic output. Confusing those meanings could accidentally assume the requirement.
[Alloy language specification](https://alloytools.org/spec.html).

Alloy 6 supplies temporal operators, bounded lasso search and complete temporal checking over a
finite signature scope. Its visualizer shows consecutive states and can fork an existing trace.
Borrow those inspection interactions, but keep Umpire's finite-trace end semantics explicit:
a lasso represents an infinite trace, and Alloy's steps scope is not a per-obligation deadline.
[Alloy 6](https://alloytools.org/alloy6.html).

### TLA+/PlusCal

TLA+ distinguishes state predicates and temporal properties, and makes fairness assumptions explicit.
Its leads-to operator permits a response in the triggering state or a later state; it is not
a one-operation-transition bound. A finite observed prefix alone does not establish unrestricted
eventual completion.
[Lamport's liveness tutorial](https://lamport.azurewebsites.net/tla/tutorial/session9.html).

Refinement with stuttering explains how implementation steps can leave abstract state unchanged.
That is a strong design analogy for history rereads and unrelated Run Events contributing zero
Nexus operation transitions.
[Lamport, Computation and State Machines](https://lamport.azurewebsites.net/pubs/state-machine.pdf).

Umpire still needs an explicit labeled-step projection: equal abstract states do not necessarily
mean no semantic step, because Actions and Outcomes can distinguish self-loops. Operation-scoped
counting and observation deduplication must therefore use model-declared identity and step meaning,
not merely count state changes. Step-bounded requirements need preservation of this counting
coordinate in addition to any ordinary trace-refinement argument.

### Ivy

Ivy can generate a component's test environment from its interface assumptions and check its
outputs against interface guarantees. This is directly relevant to avoiding independently authored
test behavior.
[Ivy compositional testing](https://microsoft.github.io/ivy/examples/testing/intro.html).

Its specifications can be stateful monitors over interface events. Import/export actions separate
the environment's role from the component's role. Isolates permit component-wise reasoning, and
Ivy checks noninterference and assertion coverage for composition. Its example explicitly
distinguishes trusting randomized-tested isolates from proving them.
[Ivy interface specifications and monitors](https://microsoft.github.io/ivy/examples/testing/specification.html).

For Umpire, borrow explicit provider assumptions/guarantees and the requirement that every composed
obligation has an owner. Keep Property definitions authoritative and derive monitors through checked
lowering; do not introduce separately handwritten monitors as a second requirement language.

### Dafny

Dafny provides method contracts, read/write frames, ghost code and module refinement.
Refining modules can implement abstract declarations and strengthen suitable guarantees.
These are valuable precedents for keeping feature semantics abstract and implementation relations
explicit. [Dafny reference](https://dafny.org/dafny/DafnyRef/DafnyRef).

The closest use here is an interface-design pattern: transformations expose the guarantees they
preserve, rather than callers unfolding representations. A method postcondition alone is not a
runtime temporal monitor, and verifying Dafny code does not by itself verify Temporal's existing Go
implementation.

## Recommended direction for this repository

Keep the proposed Nexus3 surface as a small front end to Umpire's existing semantic languages.
Lean should own declarations and their checked meaning; ordinary authors should not have to supply
serialization plumbing, identity registries, or solver setup for this finite example.

The highest-value ideas to investigate are:

- Veil for generating transition-system checking obligations from concise Lean declarations.
- Quint for readable scenarios and repeatable exploration from the same model.
- Alloy for witness/counterexample commands, visible scopes and trace forks.
- Ivy for explicit component contracts, generated test environments and monitor composition.
- TLA+ refinement for the implementation-event to semantic-step relationship.

This assessment found substantial reusable ideas and some actual libraries, but did not establish an
off-the-shelf framework providing the complete combination of Lean-owned finite semantics,
operation-scoped bounds, deterministic portable Cases, Temporal evidence correlation and
fail-closed verdicts. The work distinctive to Umpire is especially the checked bridge across those
representations. This is a scoped research conclusion, not proof that no other framework exists.

A model query should establish scenario satisfiability separately from universal property checking,
and distinguish a nonempty scenario with an unexercised trigger from one that actually covers the
requirement. Search work budgets remain distinct from semantic trace bounds. A backend translation
must preserve these distinctions instead of interpreting timeout or absence of a sampled witness as
a negative proof.

No code, dependency or architecture change was made or proposed as an approved decision. No external
framework was installed, built or compatibility-tested. The existing executable success slice in
Nexus.lean does not implement the full cancellation and operation-scoped-progress draft in Nexus.md.
