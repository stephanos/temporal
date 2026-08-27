# UMPIRe4 model architecture

Status: guiding architecture contract for the Lean model under `model/`. This document defines the
intended module seams, authoring experience, and optional verification structure. It describes the
target architecture; implementation status remains in [UMPIRE4_COMPONENTS.md](UMPIRE4_COMPONENTS.md).

This document refines [UMPIRE4_DSL.md](UMPIRE4_DSL.md), especially its package-architecture and
optional-Veil sections. Family ownership of a Veil binding means semantic ownership by the family;
the expert adapter itself lives under `Temporal.Verify`, not in the ordinary Feature or System
authoring surface.

## 1. Purpose

The Lean model exists to solve Temporal's modeling, regression, exploration, conformance, and
verification problems. It is not intended to become a general-purpose framework shaped by every
possible downstream user.

The reusable `Umpire` library nevertheless remains Temporal-agnostic:

- its public vocabulary contains no Temporal products, entities, actions, configuration keys, or
  evidence sources;
- its interfaces are selected and deepened according to problems demonstrated by Temporal;
- it concentrates sophisticated Lean, checking, planning, artifact, observation, refinement, and
  optional formal-verification machinery behind small authoring interfaces; and
- it can be tested with domain-neutral examples without importing `Temporal`.

`Temporal` is the approachable domain layer. A Temporal engineer who understands Lean basics should
be able to model product behavior and implementation mechanisms without learning Umpire internals,
proof plumbing, or Veil.

## 2. Architectural principles

1. **Temporal-focused, Temporal-agnostic Umpire.** Umpire is reusable because its vocabulary and
   interfaces are domain-neutral, not because it attempts to solve unobserved use cases.
2. **Deep Umpire modules.** Umpire should provide substantial checking and generation behavior
   through small interfaces. Removing Umpire should force its complexity to reappear across many
   Temporal models; otherwise it is only a pass-through.
3. **Approachable Temporal modules.** Feature and System authors state domain meaning, choices,
   bounds, and evidence requirements. They do not assemble Umpire internals.
4. **One semantic authority.** Canonical Feature and System declarations own Temporal meaning.
   Artifacts, Go projections, runtime adapters, observation mappings, and checker adapters do not
   independently redefine it.
5. **Semantic altitude, not expertise.** Feature and System distinguish product meaning from
   implementation meaning. Both are normal authoring surfaces for regular Temporal engineers.
6. **Explicit composition.** Product meaning and implementation meaning meet in refinement modules.
   Neither declaration order nor implicit type-class selection creates that relationship.
7. **Optional expert verification.** Veil is isolated behind generic Umpire machinery and
   family-specific adapters under `Temporal.Verify`. Ordinary Temporal imports never expose Veil.
8. **Honest outcomes.** Invalid, unsatisfiable, exhausted, divergent, unknown, conflicting,
   unsupported, and violated remain distinct outcomes. None silently becomes success.

## 3. Module topology

The intended source organization is:

```text
model/
├── Tools/
│   ├── LeanImportGraph.lean      # reusable pure qualified-import traversal
│   └── LeanSourceInventory.lean  # reusable canonical source/metadata reconciliation
│
├── Umpire/
│   ├── Core, Property, Behavior, Query, Planning, ...
│   ├── Observation and Refinement
│   └── Verify/
│       └── Veil/                 # generic optional Veil machinery
│
├── Temporal/
│   ├── API/                      # generated structural input
│   ├── DynamicConfig/            # generated structural input
│   ├── Feature/                  # canonical product meaning
│   ├── System/                   # canonical implementation meaning
│   ├── Tool/                     # ordinary developer tools
│   └── Verify/                   # expert-only Temporal checker adapters
│       └── <Family>/...
│
├── Umpire.lean                   # reusable public facade
├── Temporal.lean                 # ordinary facade; excludes Temporal.Verify
└── TemporalVerify.lean           # opt-in expert verification aggregate
```

The exact internal filenames may evolve. Normative ownership of import boundaries remains in
MOD-01, MOD-03, MOD-05, MOD-09, MOD-10, and MOD-11. The import-graph phase of `make lint-model` is
their single enforcement mechanism: it checks transitive reachability over the complete first-party
module inventory rather than scanning import text.

The current qualified policy keeps `Shared.*` independent of `Umpire.*` and `Temporal.*`, and
`Umpire.*` independent of `Temporal.*`. It isolates `Temporal.Feature.*` from
`Temporal.System.*`, `Temporal.Verify.*`, and `Umpire.Verify.Veil`, with the exact verification-test
exception `Temporal.Feature.Nexus.Experimental.CallerClosure.VeilTests`. In the reverse direction,
only the exact reviewed refinement consumer `Temporal.System.Nexus.Refinement` composes
`Temporal.System.*` with `Temporal.Feature.*`; refinement-shaped names receive no exception.

Ordinary aggregates, tools, and tests remain isolated from `Temporal.Verify.*` and
`Umpire.Verify.Veil`. The complete opt-in consumer set is `TemporalVerify`, `TemporalVeilTests`,
`Temporal.Tool.VerifyVeil`, and `Temporal.Feature.Nexus.Experimental.CallerClosure.VeilTests`; it
is an exact set, not a wildcard convention (MOD-05).

Physical placement under `Temporal/Verify/` keeps expert bindings discoverable beside their owning
Temporal families. Import isolation, not physical distance, protects the ordinary authoring path.

## 4. Module responsibilities

| Module | Responsibility | Normal author |
| --- | --- | --- |
| `Umpire` | Domain-neutral semantic authoring, checking, planning, observation, refinement, artifacts, and verification interfaces | Umpire maintainer |
| `Temporal.API` | Generated Protobuf and gRPC structure without product meaning | Generator |
| `Temporal.DynamicConfig` | Generated configuration structure without product meaning | Generator |
| `Temporal.Feature` | Product-visible states, actions, outcomes, relations, properties, and scenarios | Temporal engineer |
| `Temporal.System` | Concrete mechanisms, configuration interpretations, evidence mappings, execution semantics, and refinements | Temporal engineer |
| `Temporal.Tool` | Inspection and developer workflow without semantic authority | Temporal engineer |
| `Temporal.Verify` | Optional checker views, Veil declarations, bindings, correspondence proofs, and verification entry points | Verification expert |

`Temporal.Verify` is an adapter at the optional verification seam. Its implementation may be large;
its interface to the rest of the model should remain small: select a checked target and property,
verify them under declared assumptions and bounds, replay any counterexample, and return a receipt.

## 5. Umpire as a deep authoring module

Umpire owns the complexity shared across Temporal models:

- stable identity and source capture;
- capability, provider, connector, and target checking;
- canonicalization and semantic digests;
- authored-to-checked transitions;
- finite-domain enumeration and completeness plumbing;
- behavior admission and contradiction detection;
- property evaluation and agreement theorems;
- query validation, planning, and exploration;
- artifact construction and versioning;
- observation qualification and refinement interfaces;
- source-located diagnostics; and
- optional checker integration, replay, trust classification, and receipts.

Ordinary Temporal authoring should not require direct manipulation of:

- `CapabilityProvider` or `CapabilityConnector` records;
- raw proof-carrying planner kernels;
- canonical metadata or semantic digest strings;
- manual `SemanticSource` records when source location can be captured;
- repetitive string identities when a stable declaration identity can be derived or declared once;
- `Except.toOption`, proofs that a checked result is `some`, or `native_decide` merely to extract a
  valid checked declaration; or
- planner, artifact, or checker backend implementation types.

The lower-level typed interfaces remain valuable for Umpire implementation and expert extension.
The ordinary authoring interface should validate declarations during elaboration and translate
typed failures into precise source-located diagnostics.

The interface MUST NOT hide meaning-bearing choices. Authors still state:

- domain states, actions, outcomes, observations, and relations;
- product properties and mechanism invariants;
- allowed, required, and forbidden behavior;
- variations, requested faults, and coverage goals;
- explicit bounds or named profiles whose exact bounds are inspectable; and
- omissions and unsupported capabilities.

## 6. Authoring roles

Expertise is attached to work, not directories.

### 6.1 Regular Temporal engineer

A regular Temporal engineer may work in either Feature or System using approachable Umpire
interfaces.

In Feature they define product-visible behavior, properties, scenarios, and regressions. In System
they define concrete mechanisms, configuration meaning, evidence mappings, mechanism invariants,
and refinements. Neither path requires Veil or routine proof plumbing.

Illustrative authoring syntax may eventually resemble:

```lean
property cancellationIsUnique on Nexus.Experimental.CallerClosure where
  ...

behavior closeCallerWithPendingOperation on Nexus.Experimental.CallerClosure where
  ...

regression callerClosureCancellation where
  behavior closeCallerWithPendingOperation
  checks cancellationIsUnique
  bounds perCommit
```

This document does not fix macro syntax. It fixes the semantic interface: concise declarations in,
checked declarations and source-located errors out.

### 6.2 Semantic-model maintainer

A semantic-model maintainer extends a reusable Feature or System target with typed state, actions,
transitions, observations, finite domains, and laws. This work may require stronger Lean knowledge.
Umpire should derive routine enumeration, metadata, canonicalization, and decidable checking where
possible, while keeping required semantic assumptions and proof obligations explicit.

One maintainer pays this cost for a family; many Temporal engineers reuse the checked target.

### 6.3 Verification expert

A verification expert works under `Temporal.Verify`. They define or maintain an explicit checker
view, optional handwritten Veil declarations, and a checked correspondence with an existing
canonical Feature or System model. They do not create a second ordinary regression interface or a
second source of product meaning.

## 7. Feature and System seam

The distinction is semantic:

- **Feature defines what Temporal means.**
- **System defines how a particular implementation realizes or reveals that meaning.**

The primary classification test is whether a statement survives a complete rewrite of Temporal's
internals while externally observable behavior remains the same. If it does, it belongs in Feature.
If it may change with the implementation, it belongs in System.

| Concern | Feature | System |
| --- | --- | --- |
| Workflow or Nexus lifecycle | Product-visible lifecycle meaning | Concrete state-machine mechanism |
| Caller closure | Owned operations must be cancelled | Task, event, or handler that performs cancellation |
| Cancellation uniqueness | At-most-once semantic claim | Deduplication key, callback record, or task-attempt mechanism |
| API operation | Semantic request and outcome | RPC handler, persistence transaction, and queue routing |
| Configuration | Abstract semantic choice when product behavior changes | Key, resolution, precedence, sampling, and refresh |
| Ordering | User-visible causal or lifecycle guarantee | Storage, history, task, network, or scheduler ordering |
| Observation | Semantic fact a property may consume | Evidence sources and rules that establish the fact |
| Verification | Product property or mechanism invariant | Mechanism conformance and evidence obligations |

Mixed concerns MUST be split rather than placed wholesale in one layer:

```text
Feature property
      ▲
      │ established by refinement
      │
System mechanism ── observation mapping ──▶ runtime evidence
```

For example:

- Feature states that closing a caller eventually cancels its owned Nexus operation.
- System states which history event, task, handler, SDK participant action, and resolved
  configuration realize cancellation.
- A refinement relates the System transition to the Feature cancellation transition.
- An observation mapping explains which runtime evidence establishes that the System events
  occurred.

Feature and base System models should remain independently understandable and testable. Their
relationship becomes explicit in a `Temporal.System.<Family>.Refinement` leaf. Refinement is normal
Temporal modeling work, not a Veil-specific concern.

### 7.1 Structural API and configuration inputs

Generated inputs remain below semantic ownership:

- `Temporal.API` records RPC and message structure. Feature may interpret an API operation's product
  meaning; System may model the concrete handler or evidence source.
- `Temporal.DynamicConfig` records generated key structure. System owns resolution, precedence,
  sampling, and concrete key interpretation.
- Feature may expose an abstract semantic choice such as a retry or cancellation policy only when
  that choice changes product-visible meaning.
- A refinement maps the resolved System configuration to the abstract Feature choice.

Neither descriptor presence nor a generated configuration default creates product semantics by
itself.

## 8. Ordinary semantic flow

The ordinary model, execution, and conformance path is independent of Veil:

```text
Temporal.Feature declarations
  properties + behaviors + product target
                     │
Temporal.System model│
  mechanisms         │
  configuration      │
  observations       │
          │          │
          ▼          ▼
       refinement proof
          │
          ▼
      checked Umpire query ──▶ planning or exploration
                                      │
                                      ▼
                               ExperimentSpec
                                      │
                                      ▼
                              runtime execution
                                      │
                                      ▼
                       raw evidence → observation
                                      │
                                      ▼
                               qualified Result
```

The existing separation of Property, Behavior, Query, Observation, execution, and Result remains:

- Property states what must hold.
- Behavior constrains admissible semantic traces.
- Query states what bounded planning or execution must establish.
- Observation interprets raw implementation evidence.
- Refinement relates System meaning to Feature meaning.
- `ExperimentSpec` records environment-independent execution intent.
- Result reports qualified execution, evidence, and property outcomes.

No stage acquires semantic authority merely because it is downstream.

## 9. Optional formal-verification flow

Formal verification branches from checked semantics rather than from runtime artifacts:

```text
checked target + checked property
                │
                ▼
       Temporal.Verify adapter
                │
                ▼
         Veil verification
                │
       proof or counterexample
                │
                ▼
    canonical-model replay gate
                │
                ▼
       verification receipt
```

Generic Veil mechanics belong under `Umpire.Verify.Veil`. Temporal-specific checker views,
handwritten declarations, field/action mappings, and correspondence proofs belong under
`Temporal.Verify.<Family>`.

The verification path MUST satisfy these rules:

- ordinary `Temporal.Feature`, `Temporal.System`, `Temporal.Tool`, and `Temporal.lean` imports do not
  expose Veil;
- a Veil adapter references an existing checked target and property rather than defining an
  independent property identity;
- any repeated state or action representation is related to the canonical model by an explicit,
  checked correspondence;
- stale source, semantic, view, or binding digests fail closed;
- every counterexample replays through the canonical Umpire transition kernel, Behavior, and pure
  Property evaluator before supporting a semantic violation or promotion;
- kernel proof, reconstructed solver proof, trusted solver, bounded search, testing, and concrete
  replay remain distinct trust classes;
- Veil is not part of `ExperimentSpec`, runtime execution, evidence interpretation, production
  binaries, or the normal Temporal model build; and
- Umpire does not generate Veil source or introduce a checker-neutral semantic IR.

`TemporalVerify.lean` is the opt-in aggregate for these adapters. A focused verification command or
test target may build it without changing the ordinary Temporal developer workflow.

## 10. Failure model and diagnostics

The low-level Umpire implementation should retain typed errors for programmatic checking. The
ordinary authoring interface should convert them into source-located Lean diagnostics without
requiring authors to manually unwrap checked results.

Authoring diagnostics include at least:

- unknown or wrong-kind declarations;
- missing capabilities or incompatible providers;
- ambiguous composition or missing connectors;
- contradictory or unsatisfiable behavior;
- missing, invalid, or unit-incompatible bounds;
- incomplete target enumeration for a complete query;
- invalid or incomplete refinement;
- ambiguous, conflicting, or incomplete observation mapping; and
- use of Verify or Veil from a forbidden import path.

Phase outcomes remain separate:

| Phase | Representative outcomes |
| --- | --- |
| Authoring/checking | valid, invalid, unsatisfiable |
| Planning | selected, verified within complete bounds, absent, exhausted |
| Execution | realized, diverged, unsupported, infrastructure failed |
| Observation | established, missing, ambiguous, conflicting, unsupported |
| Property | satisfied, violated, unknown, conflict, unsupported |
| Verification | established, violated, unknown, unsupported, invalid |

Missing evidence never establishes absence. Budget exhaustion never establishes verification.
Requested actions or faults never establish realization. A checker timeout, unavailable solver,
stale binding, incomplete correspondence, or replay disagreement never establishes success.

## 11. Test and build strategy

Tests follow the same module seams used by callers:

| Scope | Required evidence |
| --- | --- |
| `Umpire` | Domain-neutral authoring, checking, planning, refinement, observation, artifacts, and diagnostics |
| `Temporal.Feature` | Pure product semantics, properties, admitted/rejected traces, and semantic mutations |
| `Temporal.System` | Mechanisms, configuration, evidence mappings, omissions, corruption, ambiguity, and conflicts |
| `Temporal.System.*.Refinement` | Positive correspondence plus independent mutations that fail refinement |
| `Temporal.Verify` | Opt-in binding, stale digest, correspondence, trust-class, counterexample, and replay tests |
| Architecture | Import direction, domain purity, forbidden Veil exposure, and aggregate isolation |

The domain-neutral Switch example remains the minimum Umpire reference. Temporal should retain a
small teaching progression from one Feature target through a System refinement and observation
mapping before presenting an advanced composed family.

The normal model build and regression gate MUST validate Umpire and ordinary Temporal declarations
without compiling or running `Temporal.Verify`. A separate focused verification gate owns Veil's
dependency, toolchain compatibility, execution cost, retained evidence, and trust reporting.

## 12. Current-state implications

The current model already has the correct high-level dependency direction: `Umpire` is independent
of Temporal, while Temporal examples use Umpire's checked authoring and planning types. It does not
yet fully achieve the intended module depth or author experience.

Current Temporal examples still expose substantial implementation detail, including manual semantic
identities and sources, capability providers, transition kernels, completeness proofs, checked
result extraction, `native_decide`, query contexts, and planner assembly. Those are evidence that
Umpire's ordinary authoring interface needs to deepen; they are not the desired long-term authoring
style.

Observation, refinement, and current-model Veil integration also remain incomplete or separate as
tracked in `UMPIRE4_COMPONENTS.md`. This document does not authorize a wholesale refactor or claim
that those modules are built. Each implementation slice still requires bounded Flow-Next planning,
acceptance criteria, and approval.

## 13. Architectural acceptance criteria

The target architecture is realized when:

1. A Temporal engineer with Lean basics can author a normal Feature regression without constructing
   providers, connectors, kernels, checked-result extraction proofs, semantic digests, or planner
   machinery.
2. The same engineer can author a System mechanism, configuration interpretation, observation
   mapping, and Feature refinement through cohesive Umpire interfaces without learning Veil.
3. Feature and base System models are independently understandable and testable, and mixed claims
   meet only through explicit refinement.
4. Umpire remains free of Temporal vocabulary and is fully testable through domain-neutral
   fixtures.
5. Ordinary `import Temporal`, its model tests, and its developer tools expose no Veil declarations
   or types.
6. `Temporal.Verify` can opt one family into Veil without changing that family's ordinary authoring
   interface or making Veil a second semantic authority.
7. Every accepted checker counterexample replays through canonical semantics, and every receipt
   exposes bounds, trust, provenance, and omissions.
8. `make lint-model` enforces the import seams owned by MOD-01, MOD-03, MOD-05, MOD-09, and MOD-10
   over the complete first-party module graph; semantic altitude, module depth, and isolated
   testability remain design-review judgments.
9. Authoring, planning, execution, evidence, property, and verification failures remain explicit and
   fail closed.

## 14. Non-goals and rejected designs

- Designing Umpire around hypothetical non-Temporal consumers.
- Putting Temporal vocabulary or family-specific Veil bindings under `Umpire`.
- Treating Feature as the easy layer and System as an expert-only layer.
- Letting Feature import concrete System mechanisms.
- Collapsing product properties, mechanism models, refinements, and evidence mappings into one
  declaration language.
- Hiding meaning-bearing bounds, assumptions, omissions, requested faults, or trust classes for the
  sake of concise syntax.
- Requiring ordinary Temporal engineers to learn Veil.
- Importing `Temporal.Verify` from the ordinary Temporal facade or normal developer tools.
- Generating Veil source, shipping Veil in production paths, or treating `ExperimentSpec` as a
  checker-neutral intermediate representation.
- Accepting a Veil proof or counterexample without a checked binding to canonical semantics.
- Duplicating Temporal semantic authority in Go, generated projections, runtime adapters, evidence
  mappings, or formal-checker declarations.
