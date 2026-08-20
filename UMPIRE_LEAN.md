# Umpire3: a Lean-first model and testing system

Status: approved architecture and implementation plan, 2026-08-19.

## Decision

Build Umpire3 as a new, independent system under `tests/umpire3`. Do not migrate Umpire2 into it,
layer it over `common/testing/umpire`, or compile Umpire2's verification IR into Lean.

Umpire3 uses Lean as its sole semantic source of truth. Lean defines product behavior, system
behavior, properties, executable bounded semantics, and refinement. A new Go runtime realizes
Lean-produced semantic experiments against Temporal, observes the implementation, and retains
qualified evidence. The first bridge between the two is versioned data, not shared code, generated
Go, or FFI.

The intended relationship is:

```text
Lean / Veil = semantic brain
Umpire3 Go  = execution hands and observation eyes
Temporal Go = implementation under test
```

Umpire2 remains intact and operational. Replacing or retiring it is a separate future decision.
Umpire3 may reuse ordinary Temporal client, server, and test infrastructure, but it must not import
Umpire1, Umpire2, or `common/testing/umpire`. Shared Umpire code may be extracted only after both
systems independently demonstrate the same stable seam.

## Why a new system

The current Umpire architecture has grown from lifecycle testing into a semantic IR containing
entities, relations, actions, guards, effects, properties, modules, interfaces, obligations,
composition, and refinement maps. That is useful engineering machinery, but extending it to cover
epochs, histories, recursive structures, partial orders, fairness, leases, causality, ghost state,
and arbitrary simulation relations would amount to designing a theorem-prover language in Go.

Using that IR as the canonical model and Lean as another backend would preserve the central risk:
the most important semantics would have to survive a potentially lossy lowering before Lean could
reason about them. More generated backends would cross-check implementations of the lowering, but
they could all inherit the same omission from the source IR.

Umpire3 reverses that relationship. Lean is the language in which meanings and refinement
obligations are stated. Executable test descriptions and backend inputs are derived projections.
This removes the need to predict every semantic primitive Temporal will need while retaining
Umpire's valuable operational role: realization, observation, faults, replay, minimization,
environments, evidence, and regression generation.

The existing [`model/NexusAutoClose.lean`](./model/NexusAutoClose.lean) demonstrates that a narrow
Temporal design question can be modeled and kernel-checked in readable Lean. It is a useful input
to Umpire3's design, but Umpire3 does not import it. Umpire3 will re-express the relevant behavior
inside the product/system/refinement structure described here.

## Goals

Umpire3 will:

- define user-visible Temporal behavior as product semantics in Lean;
- define selected distributed implementation mechanisms as system semantics in Lean;
- prove that adopted system models refine their product models;
- explore bounded executions from the same semantics used by proofs;
- turn formal counterexamples and selected traces into versioned semantic experiments;
- realize those experiments against real Temporal environments through independent Go adapters;
- compare normalized observations with the experiment's semantic checkpoints;
- distinguish kernel proofs, bounded exploration, and implementation conformance claims;
- preserve assumptions, bounds, omissions, tool versions, model hashes, observations, and cleanup
  results in replayable artifacts;
- turn stable failures into minimized, deterministic regression candidates; and
- support composition through proved provider guarantees and consumer assumptions.

## Non-goals

Umpire3 will not:

- generate or verify all Temporal Server Go code;
- prove that the complete Go implementation refines the complete system model;
- make protobuf wire structures the product domain model;
- infer how to realize or observe a semantic action from Lean alone;
- replace input fuzzing with theorem proving;
- claim unbounded liveness from finite execution;
- require every model checker to understand every Lean definition;
- build a general compiler from arbitrary Lean to Go;
- use Lean's native runtime or cgo in the initial architecture;
- share Umpire2 implementation code before a genuinely shared seam exists; or
- retire Umpire2 as part of Umpire3's implementation plan.

## The semantic levels

Temporal correctness has three principal levels and one orthogonal input boundary.

```text
                  PRODUCT SEMANTICS P
             "What does Temporal mean?"
                         ▲
                         │ system refinement: S ⊑ P
                         │
                   SYSTEM SEMANTICS S
          "How may Temporal implement that meaning?"
                         ▲
                         │ conformance or selected proof: I ⊑ S
                         │
                  TEMPORAL GO I

   API / WIRE VALUES A ── interpretation ──► PRODUCT COMMANDS P
```

### Product semantics

Product modules describe externally meaningful behavior such as Workflow, Update, Activity,
Schedule, Child Workflow, and Nexus Operation. They state what commands mean, which outcomes are
observable, and which properties users may rely on. They should know almost nothing about history
shards, transfer queues, matching partitions, persistence retries, task processors, leases, or RPC
retries.

### System semantics

System modules describe selected internal mechanisms such as tasks, attempts, durable history,
mutable state, persistence commits, ownership epochs, retries, failover, recovery, and matching.
Several system steps may implement one product step. Many system steps should project to product
stuttering.

### Refinement

For adopted modules, Umpire3 proves `S ⊑ P` in Lean. A refinement is not a table matching action
names. It is a relation between system and product states plus proofs that:

1. every relevant initial system state relates to an initial product state; and
2. every system step preserves the relation by corresponding to zero or more legal product steps.

The relation may be an abstraction function when that is sufficient, but the framework must allow
an arbitrary relation when abstraction is not functional.

### Implementation conformance

Umpire3 normally evaluates `I ⊑ S` through execution and independently normalized observation.
That is evidence about a particular build, environment, trace, and observation profile, not a
universal proof. Selected Go modules may later receive modular verification through Gobra or, where
crash/distributed reasoning justifies the cost, Goose/Grove. Those proofs strengthen a named seam;
they do not silently upgrade the claim for the rest of Temporal.

### API interpretation

Selected protobuf schema types may be generated into Lean from Temporal descriptors, but wire
types do not define product state. Handwritten interpretation functions convert them to semantic
commands or semantic errors. Separate theorems establish properties such as:

- an accepted request produces a valid semantic command;
- a rejected request produces no product transition; and
- semantically relevant policies, identifiers, timeouts, and idempotency keys are preserved.

Binary encoding, field numbers, unknown fields, and generated Go representation remain specialized
wire-compatibility concerns unless a model explicitly targets them.

## Target architecture

```text
tests/umpire3/
  model/                         independent Lean project
    Umpire3/
      Transition.lean            transition systems and traces
      Executable.lean            bounded executable semantics
      Property.lean              safety and explicitly qualified progress
      Refinement.lean            relations, stuttering, and simulation
      Experiment.lean            semantic experiment construction/export
    Temporal/
      Product/
        Nexus.lean
        Update.lean
      System/
        NexusTasks.lean
        UpdateTasks.lean
        Ownership.lean
      Refinement/
        NexusTasks.lean
        UpdateTasks.lean
      API/
        Generated/               selected descriptor-derived types
        Nexus.lean               interpretation into semantic commands
      Experiments/
        NexusCancellation.lean
    lakefile.toml
    lean-toolchain
    mise.toml

  protocol/                      versioned experiment/result representation
  runtime/                       decoding, validation, execution, and results
  environment/                   resource ownership and cleanup
  temporal/                      Temporal-specific realizers and observers
  artifact/                      replay, evidence, and redaction
  internal/negativecontrol/      test-only faulty adapters and fixtures
```

The exact number of files may change as the implementation reveals cohesive modules. The ownership
rules may not:

- `model` owns meaning;
- `protocol` owns the data seam;
- `runtime` owns orchestration, not semantics;
- `environment` owns resource lifecycle;
- `temporal` owns implementation interaction; and
- `artifact` owns retained evidence and replayability.

## Independence rules

Umpire3 starts with hard isolation:

- no imports from `tests/umpire1`, `tests/umpire2`, or `common/testing/umpire`;
- no aliases of their semantic types;
- no calls through facade packages that hide those dependencies;
- no generated artifacts sourced from their declarations;
- no compatibility layer that constrains Umpire3's semantic model; and
- no test that passes only by running an Umpire2 oracle beside Umpire3.

CI will enforce the dependency rule using Go dependency inspection and repository searches.
Ordinary Temporal packages remain available for clients, server setup, workers, RPCs, test hooks,
and test clusters. Independence applies to the Umpire implementations, not to the system being
tested.

Later extraction requires evidence of a real seam:

1. Umpire2 and Umpire3 both contain working adapters for the same responsibility.
2. Their callers need the same behavioral contract, including error and lifecycle semantics.
3. Removing the proposed module would reintroduce meaningful complexity in both callers.
4. Extraction does not import either semantic model into the other.
5. Both suites pass against the extracted module without compatibility conditionals.

One adapter is a hypothetical seam. Two independent adapters may reveal a real one.

## Lean semantic kernel

The initial Lean library should be deliberately small. It must provide ordinary definitions, not a
new surface language.

```lean
structure TransitionSystem where
  State   : Type
  Action  : Type
  Initial : State → Prop
  Step    : State → Action → State → Prop

structure ExecutableModel (M : TransitionSystem) where
  next     : M.State → M.Action → List M.State
  next_iff : ∀ s a s', s' ∈ next s a ↔ M.Step s a s'
```

Supporting definitions cover finite traces, reachability, reflexive/transitive stepping,
stuttering, bounded enumeration, safety properties, and refinement relations. Product and system
modules remain free to use unrestricted Lean internally. The kernel should not gain a primitive
merely because one model uses a particular map, epoch, history, lease, or ownership scheme.

An executable definition is never presumed equivalent to a relational definition. The module must
prove soundness and completeness, such as `next_iff`, before the executable view may drive
exploration or experiments.

Friendly macros may be added only after at least two product/system models reveal stable repeated
syntax. A future `Umpire3` Lean authoring library may elaborate concise declarations into ordinary
Lean terms, but arbitrary Lean must always remain available. The macro is syntax inside Lean's
elaboration and type-checking world, not a second semantic IR.

## Refinement interface

The refinement module should support the following mathematical shape:

```lean
R systemState productState

R s p ∧ System.Step s a s'
  ⇒ ∃ p', Product.StepStar p p' ∧ R s' p'
```

`StepStar` permits internal system activity to stutter or to implement a short sequence of product
steps. When action correspondence matters for trace export, the refinement may additionally return
an observation label or semantic product action. That label is evidence metadata; it is not a
replacement for the state relation and proof.

Fairness is separate. A safety refinement theorem must not acquire an implicit fairness assumption
just because a model also wants to state progress. Progress theorems name their scheduling,
delivery, retry, and resource assumptions explicitly. Bounded executable results retain those
assumptions in their manifest.

## The semantic experiment seam

The first Lean-to-Go bridge is a versioned semantic experiment. It contains enough information to
reproduce intent and qualify a result without embedding either a Lean runtime or a duplicate Go
state machine.

An experiment contains:

```text
format_version
experiment_id

model:
  module identifiers
  source revision
  semantic hash
  Lean/tool versions

property:
  stable identifier
  statement hash
  requested implementation-conformance claim

scope:
  bounds
  assumptions
  symmetry choices
  exploration strategy and seed, when applicable

resources:
  symbolic identities and relationships

actions:
  stable action kind
  typed semantic arguments
  symbolic bindings
  required capabilities
  pre/post checkpoint identifiers

checkpoints:
  semantic observations required
  ordering/causality requirements
  omission policy

provenance:
  proof, bounded exploration, counterexample, or curated trace

retention:
  redaction class
  artifact limits
```

The schema excludes raw payloads, headers, credentials, user metadata, and implementation-specific
object dumps. Semantic identifiers are symbolic or hashed. Unknown fields, versions, action kinds,
capabilities, and claim kinds are rejected unless the version explicitly defines forward-compatible
handling.

The Go result contains:

```text
experiment digest
Temporal build and configuration identity
environment and evidence profile
realized actions and grounded bindings
normalized observations and causal references
omissions and ambiguities
checkpoint outcomes
qualified claims
timeouts and budget exhaustion
cleanup outcome and recoverable resource metadata
redacted artifact locations
```

The input experiment describes semantic intent. The result records implementation evidence. Neither
is allowed to overwrite the other.

## Go runtime modules

The Go side is independent and intentionally does not implement `enabled`, `next`, or a product
state machine. It validates the experiment, prepares an environment, realizes actions, collects
normalized observations, evaluates checkpoint evidence, and returns a qualified result.

The external runtime seam should be one deep operation:

```go
result, err := runtime.Run(ctx, request)
```

The implementation may contain internal seams for testing, but callers should not assemble stores,
poll loops, binding maps, cleanup, or evidence qualifiers themselves.

### Environment

The environment owns every allocated resource and supplies capability-scoped realizers and
observers. Preparation either returns a complete environment or a recoverable cleanup handle.
Cleanup runs on every post-preparation path under its own bounded context. A transport or process
adapter must honor context cancellation; cooperative cancellation is stated honestly when a hard
process boundary does not exist.

### Realization

A realizer maps one stable semantic action kind to Temporal traffic or controlled environment
behavior. It may use gRPC, SDK workers, test hooks, faults, failover controls, or deployment
operations. It returns grounded identities and action-local evidence references. It does not decide
whether the action was semantically legal; that judgment belongs to the exported Lean experiment.

### Observation

An observer converts available implementation evidence into a small normalized vocabulary. Sources
may include public responses, history, telemetry, internal facts, or test-only state. Every
observation records its source and available causal/order information. Wall-clock timestamps from
different processes do not establish causality.

### Artifact retention

Artifacts retain the exact experiment, grounded bindings, normalized observations, environment
profile, action windows, omissions, qualified claims, and cleanup result. Retention is bounded and
redacted before persistence. An artifact is replay input, not a dump of the Temporal process.

## Claims and evidence

Umpire3 keeps logically different results separate.

| Claim | Meaning |
| --- | --- |
| `proved` | The pinned Lean kernel accepted a theorem with the recorded assumptions and source hash. |
| `bounded-safe` | No counterexample was found within the recorded finite scope. |
| `counterexample` | Exploration found a semantic trace violating the named property within the recorded scope. |
| `conforming` | The named Temporal build produced sufficient observations matching one experiment. |
| `violating` | Sufficient implementation evidence contradicts a required checkpoint or property. |
| `unsupported` | The environment lacks a required action or observation capability. |
| `inconclusive` | Evidence loss, ambiguity, timeout, cleanup failure, or incomparable ordering prevents a claim. |

Claims do not silently promote one another:

- a Lean proof about `S ⊑ P` is not proof that Temporal Go implements `S`;
- a bounded-safe result is not an unbounded proof;
- a successful real execution is not proof of all executions;
- coverage is not correctness; and
- a formal counterexample that cannot be realized is not evidence of a Go defect.

Every success is fail-closed. Hash drift, unknown vocabulary, missing required observations,
ambiguous identity, conflicting lineage, unsupported causality, budget exhaustion, and incomplete
cleanup prevent a successful conformance claim.

## First vertical slice: Nexus cancellation and stale completion

The first slice tests the architecture rather than maximizing feature coverage.

### Product question

State precisely when Nexus cancellation has semantically won. Prove that, after that point, an old
or retried task cannot make a successful outcome externally visible.

The product model includes only externally meaningful operation state, cancellation acceptance,
and terminal outcome. It excludes attempts, queues, ownership, RPCs, and persistence mechanics.

### System coordinates

The system model includes only what the target race needs:

- operation identity and durable outcome;
- cancellation attempted, accepted, and committed states;
- task identity, attempt, dispatch, completion, and acknowledgement;
- shard identity and owner epoch;
- success/cancellation persistence decisions;
- crash, recovery, retry, and stale-worker behavior; and
- enough history to distinguish attempted from committed effects.

### System actions

The first action set is:

```text
ScheduleOperation
DispatchTask
WorkerReturnsSuccess
RequestCancellation
CommitCancellation
PersistSuccess
RetryTask
AcquireOwnership
CrashOwner
RecoverOwner
AckTask
```

Actions that do not change the product projection stutter. `CommitCancellation` and
`PersistSuccess` may change it only when the product model permits the corresponding transition.

### Required negative trace

The model must be able to express this candidate failure:

```text
operation starts
task is dispatched
cancellation is accepted
owner crashes or changes epoch
task is retried
old worker reports success
stale success is persisted
successful completion becomes visible
```

The sound system model must reject or refine away the stale persistence step. A deliberate mutation
that permits it must break the refinement theorem or produce a counterexample. If both proofs and
exploration remain green, the model or property is too weak and the milestone fails.

## Semantic exploration and input exploration

Lean exploration and Go fuzzing cover different spaces and must remain complementary.

| Semantic exploration in Lean/Veil | Input exploration in Go |
| --- | --- |
| cancellation during retry | malformed or missing fields |
| failover during completion | boundary integers and durations |
| duplicate completion | invalid enum values |
| retry after ownership transfer | payload and encoding variants |
| stale response after timeout | request-shape and validator edge cases |

Lean searches state-aware combinations of meaningful actions and system conditions. Go starts from
a semantically valid experiment and varies wire values, timing, retry counts, worker responses, and
other realization inputs. A Lean trace may identify where input variation is valuable, but Lean
does not become a protobuf fuzzer and input fuzzing does not become a semantic proof.

## Milestone plan

Milestones are sequential evidence gates, not calendar estimates. Work stops at a failed gate until
the design is corrected or the experiment is explicitly abandoned.

### M0: charter and independent scaffold

Deliver:

- the `tests/umpire3` directory structure;
- pinned Lean, Lake, and mise configuration;
- independent Go packages and build targets;
- a versioned experiment/result schema skeleton;
- CI jobs for Lean build/lint and focused Go tests;
- dependency guards rejecting Umpire1/2/common Umpire imports; and
- proof hygiene checks rejecting `sorry`, `admit`, unsafe definitions, `native_decide`, and
  unreviewed axioms.

Verify:

- a clean checkout installs the pinned toolchain and builds both halves;
- an intentionally added Umpire2 import fails the dependency guard;
- an intentionally admitted Lean theorem fails proof hygiene; and
- tool and schema versions appear in a generated empty manifest.

Exit gate: the empty system is reproducible, independent, and incapable of reporting a semantic
success.

### M1: minimal Lean semantic kernel

Deliver:

- transition systems, finite traces, reachability, and reflexive/transitive stepping;
- executable bounded semantics separated from relational semantics;
- soundness and completeness proof obligations for executable stepping;
- safety-property evaluation;
- refinement relations, stuttering, and step simulation;
- explicit assumption and bound metadata; and
- small example models used only to test the library.

Do not deliver macros, Temporal-specific primitives, backend projections, or Go generation.

Verify:

- executable traces correspond exactly to relational traces at example bounds;
- unreachable states cannot appear in exported experiments;
- a deliberately incomplete `next` function fails equivalence; and
- a deliberately permissive transition breaks a safety theorem.

Exit gate: the kernel exposes a small interface while product and system models can use arbitrary
Lean internally.

### M2: Nexus product semantics

Deliver:

- product operation state and semantic commands;
- a precise definition of cancellation acceptance/winning;
- terminal outcome and stability properties;
- a relational `Step` and proved-equivalent executable bounded view; and
- human-readable examples for permitted and forbidden races.

Verify:

- no task, RPC, shard, persistence, or ownership type appears in the product interface;
- terminal stability and cancellation outcome properties are kernel-checked;
- at least one plausible but incorrect cancellation rule fails; and
- reviewers can state the user-visible contract without reading the system model.

Exit gate: the model answers the product question rather than mirroring today's implementation.

### M3: Nexus task-system semantics

Deliver:

- the minimal system coordinates and actions listed in the first vertical slice;
- explicit attempted, applied, committed, persisted, and observed stages;
- task attempts, owner epochs, crash/recovery, retry, and stale worker behavior;
- declared assumptions about persistence atomicity and task delivery; and
- bounded executable enumeration.

Verify:

- the candidate stale-completion trace is representable;
- stale and current owner commits are distinguishable;
- retry and ownership changes do not directly mutate product state; and
- no hidden fairness or reliable-delivery premise appears in a safety definition.

Exit gate: the model contains enough distributed machinery to threaten the product property, but
no mechanics irrelevant to that threat.

### M4: Nexus system-to-product refinement

Deliver:

- the state relation between Nexus task-system and product states;
- initial-state correspondence;
- a simulation proof for every system step;
- explicit classification of stuttering steps;
- named assumptions and proof dependencies; and
- a machine-readable proof manifest.

Verify:

- Lean proves `NexusTaskSystem ⊑ NexusProduct`;
- every action constructor is covered by the simulation proof;
- permitting a stale owner to persist success after cancellation breaks the theorem;
- removing cancellation evidence from the relation breaks an appropriate theorem; and
- the proof manifest changes when a theorem statement, assumption, or semantic definition changes.

Exit gate: Umpire3 establishes the top refinement relation for the first real slice, not merely a
structural mapping.

### M5: bounded exploration and trace export

Deliver:

- explicit bounded exploration from the proved executable semantics;
- a stable semantic trace vocabulary;
- counterexample-to-experiment export;
- model, property, bound, assumption, and tool hashes;
- deterministic ordering and seed handling; and
- optional pinned Veil integration if it consumes the same Lean semantics without creating a
  second hand-maintained model.

Start with a simple explicit explorer when feasible. Add Veil for symbolic search or invariant
automation only after its integration preserves source identity and headless reproducibility.

Verify:

- the deliberate stale-completion mutation emits the expected semantic counterexample;
- the sound model does not emit that trace at the declared bounds;
- two exports from the same input are byte-stable;
- changing bounds or assumptions changes the experiment digest; and
- malformed or semantically incomplete traces cannot be exported as valid experiments.

Exit gate: Lean can produce a self-describing experiment that is useful without Lean being present
at Go runtime.

### M6: independent Go runtime

Deliver:

- strict experiment decoding and validation;
- one deep `Run` interface;
- internal environment, realizer, observer, binding, checkpoint, and artifact modules;
- deadline, cancellation, and cleanup handling;
- qualified result types; and
- fakes owned by Umpire3 for isolated runtime tests.

The Go runtime does not implement product transitions or re-evaluate Lean proofs. It verifies
schema integrity, provenance hashes, required capabilities, action realization, and observation
evidence.

Verify:

- dependency inspection shows no existing Umpire implementation dependency;
- unknown versions/actions/capabilities fail before environment allocation;
- every post-preparation failure attempts cleanup;
- missing evidence is unsupported or inconclusive, never conforming;
- cancellation interrupts adapters that promise cooperative cancellation; and
- retained artifacts are bounded and redacted.

Exit gate: a fake independent environment can execute a semantic experiment and produce a
fail-closed, replayable result.

### M7: Temporal Nexus adapter

Deliver:

- a local Temporal environment owned by Umpire3;
- Nexus workflow/worker fixtures;
- realizers for the vertical-slice actions that can be safely controlled;
- normalized Nexus/task/ownership observations from explicit sources;
- capability declarations for actions requiring test hooks or failover control; and
- symbolic-to-concrete identity grounding.

Verify:

- each realizer has a positive and failure-mode test;
- each observer demonstrates the evidence needed for its checkpoint;
- server-minted identities are learned from observations rather than guessed;
- incomparable clocks do not establish ordering;
- unsupported failover control prevents execution before mutation; and
- one Lean-produced trace completes against a real Temporal test cluster.

Exit gate: the first formal experiment crosses the full Lean-data-Go-Temporal-observation path.

### M8: closed validation loop

Deliver:

- a controlled implementation mutation or test-only faulty Temporal behavior that permits stale
  success after cancellation;
- formal discovery of the violating trace;
- real-cluster reproduction;
- semantic minimization that preserves the same qualified violation;
- deterministic regression promotion; and
- a negative-control job showing the fixed implementation rejects the regression.

The negative control must affect implementation behavior, not merely make an observer lie or force
the Go runner to return failure.

Verify:

- the mutation causes a `violating` result with sufficient evidence;
- removing irrelevant actions preserves the same property violation;
- fixing the mutation makes the promoted regression conform;
- observation loss changes the result to inconclusive rather than hiding the defect; and
- replay records schedule drift separately from semantic or evidence drift.

Exit gate: Umpire3 has demonstrated its core value on a real distributed behavior, not only on a
formal model.

### M9: runtime hardening and portable profiles

Deliver:

- deterministic replay and corpus management;
- hard count/evidence/resource budgets and cooperative time budgets;
- robust cleanup with recovery-safe resource metadata;
- local and CI environment profiles;
- deployment profile design with capability negotiation;
- secret-safe artifact retention; and
- monotonic minimization across actions, resources, faults, and unused bindings.

Verify:

- process crash, context cancellation, action timeout, observation loss, and cleanup timeout paths;
- budget reservation before concurrent work;
- corpus deduplication independent of concrete runtime identities;
- the same semantic experiment runs locally and in CI without changing its meaning; and
- a 10× scenario corpus changes offline selection cost, not per-request production behavior.

Exit gate: Umpire3 can be operated repeatedly in CI without leaking resources, authority, secrets,
or unjustified claims.

### M10: semantic API boundary

Deliver:

- descriptor-driven generation of selected protobuf schema types into Lean;
- generated enums and message shapes isolated under `Temporal/API/Generated`;
- handwritten interpretation into Nexus product commands;
- validation/error semantics; and
- interpretation theorems for the selected fields.

Do not implement protobuf binary encoding in Lean unless a separate wire-compatibility target later
justifies it.

Verify:

- descriptor changes deterministically update generated Lean;
- accepted requests imply valid product commands;
- rejected requests have no product effect;
- semantically meaningful enum/policy changes cannot be silently ignored; and
- fuzzed Go request variants can be classified through the same semantic boundary without making
  Lean the input fuzzer.

Exit gate: the API and system refinement questions meet at product semantics without contaminating
the product model with wire representation.

### M11: generated Go testing interface

This milestone is conditional. Begin it only when M8 has succeeded, two Lean models use the trace
schema, and repeated handwritten Go schema boilerplate is a measured maintenance cost.

Deliver, if the gate is met:

- generation of ordinary Go action and semantic value types from an explicitly marked executable
  Lean subset;
- generated decoders/builders and optional pure executable oracle;
- source/proof/model hashes in generated headers;
- a handwritten stable Go facade; and
- differential tests between Lean execution and generated Go.

The supported subset is intentionally small: structures, enums, primitive values, options, lists,
finite sets/maps, pure pattern matching, and bounded recursion. Noncomputable or unsupported
definitions fail generation explicitly. Proofs and arbitrary Lean programs are never compiled to
Go by this generator.

Verify:

- generated Go agrees with Lean for exhaustive smoke bounds and randomized larger bounds;
- a seeded generator bug is detected by differential tests;
- generated files are deterministic and never hand-edited;
- generated code does not become the semantic authority; and
- no Lean runtime or cgo dependency enters Temporal tests.

Exit gate: code generation reduces integration toil without changing the trust model. If the entry
gate is not met, skip directly to M12 and ship Umpire3 1.0 without generated Go.

### M12: composition, second domain, and Umpire3 1.0

Deliver:

- a second product/system/refinement slice for Workflow Update via history/task machinery;
- a reusable task-delivery guarantee proved by the system module and consumed by Nexus and Update;
- composition checks for provider guarantees and consumer assumptions;
- authoring guidance for ordinary Lean definitions and proofs;
- evidence-based evaluation of a small Gobra implementation-verification pilot; and
- a versioned Umpire3 1.0 manifest and support policy.

Grove/Goose is considered only if the selected implementation seam needs crash/distributed
reasoning that Gobra cannot express and the property justifies substantially higher proof cost.

Verify:

- the second domain reuses semantic library modules without Nexus-specific conditionals;
- changing a provider guarantee invalidates dependent proofs or manifests;
- both domains export and execute experiments through the same versioned seam;
- the Go implementation-verifier pilot states exactly which code and assumptions it covers; and
- no Umpire2 dependency or retirement requirement has appeared.

Exit gate: Umpire3 demonstrates compositional value beyond one showcase model and satisfies the
1.0 definition below.

## Test and CI strategy

### Lean checks

- build every theorem with the pinned kernel;
- forbid proof admissions and unreviewed axioms;
- lint Lean sources;
- run executable examples and bounded enumeration;
- run negative semantic mutations that must break named theorems or find counterexamples;
- hash definitions, statements, assumptions, and tool versions into manifests; and
- retain proof artifacts separately from bounded exploration results.

### Protocol checks

- schema round trips;
- deterministic canonical encoding;
- version compatibility and explicit rejection tests;
- malformed, oversized, unknown, and hash-mismatched inputs;
- redaction and maximum-size enforcement; and
- golden experiments small enough for human review.

### Go checks

- unit tests through Umpire3-owned fakes;
- failure-path tests for every environment lifecycle stage;
- adapter contract tests for action and observation capabilities;
- focused real-cluster tests with `-tags test_dep`;
- race tests for the Umpire3 runner and adapters;
- fuzz tests for protocol decoding and semantic input variants;
- deterministic artifact/replay tests; and
- dependency tests proving isolation from previous Umpire implementations.

### End-to-end checks

- Lean counterexample to real execution;
- real observations to qualified claim;
- deliberate implementation fault to stable violation;
- fixed implementation to conforming regression;
- evidence removal to inconclusive result;
- bounded minimization preserving property identity; and
- replay distinguishing realization, scheduling, observation, and semantic drift.

## Failure modes and controls

### Wrong specification

A kernel proof can establish the wrong theorem perfectly. Require reviewable property statements,
positive examples, forbidden examples, deliberate semantic mutations, and traces that distinguish
plausible competing definitions. Product models must state user-visible meaning rather than mirror
current Go states.

### Abstraction gap

A system model can omit the data or interleaving that causes the real bug. Record model omissions,
make assumptions explicit, challenge refinement with negative traces, and replay formal traces
against independently observed Go behavior. Adding a successful backend does not close an omitted
dimension.

### Shared executable-semantics bug

Exploration uses executable definitions related to relational semantics by proof. Export is allowed
only from models satisfying that equivalence. Mutation tests separately challenge the relation, the
executable view, and the property.

### Exporter or parser bug

The experiment schema is canonical and hashed. Round-trip tests, human-readable goldens, strict
decoding, and end-to-end trace identity checks prevent silent reinterpretation. Generated Go later
receives differential tests against Lean.

### Realization drift

A semantic action may no longer correspond to the Temporal mechanism an adapter invokes. Realizers
declare capabilities and return action-local evidence. Checkpoints distinguish realization
failure, semantic mismatch, and observation insufficiency. Replay reports action drift separately.

### Observation ambiguity

Attempted, applied, committed, persisted, and observed are distinct. A destination state label is
not evidence of the path that produced it. Ordered claims require a causal reference or comparable
source sequence; cross-process timestamps alone do not qualify.

### Fairness and liveness

Safety proofs do not smuggle in scheduler fairness. Progress theorems list assumptions. Live tests
make bounded progress claims under explicit time and work budgets. Timeout is inconclusive or a
bounded violation only when the property defines it that way.

### State explosion

Compose small models, use finite identities for exploration, exploit symmetry, retain explicit
bounds, and separate smoke from deeper jobs. Prefer interface guarantees over flattening product,
task, history, matching, and persistence into one state. Unsupported scale is reported, not silently
truncated.

### Tool or solver instability

Pin and checksum toolchains. Cap CPU, memory, trace length, and solver time. A timeout is
inconclusive. Veil or another solver-backed tool is optional until its headless workflow, trace
format, and source identity are reproducible.

### Environment crash and cleanup failure

Preparation produces recovery-safe metadata before risky execution. Cleanup runs under an
independent bounded context on every post-preparation path. A process-isolated adapter is required
before claiming a hard timeout against potentially non-cooperative code. Cleanup failure prevents
a successful operational result.

### Tenfold load growth

Proof and model checking remain offline and depend on declared model bounds, not production request
rate. Live execution uses scenario selection, sampling, and reserved evidence budgets. A 10×
cluster load may reduce supported observation fidelity or scenario rate; the environment profile
must downgrade the claim rather than overrun the safety envelope.

## Trade-offs

### Correctness

Lean substantially strengthens statements about models and refinement, but it does not remove the
need to validate the model against Temporal. The architecture deliberately carries a visible gap
between `S ⊑ P` proof and `I ⊑ S` evidence.

### Performance

Formal work is offline. The Go runtime avoids a Lean FFI and executes only bounded experiments.
Normalization and artifact retention are bounded. Generated Go is postponed until it solves a
measured problem.

### Scalability

Compositional guarantees scale better than a universal model, but proof dependencies and
assumptions require disciplined manifests. Deep exploration still grows combinatorially and will
need symmetry, decomposition, and possibly Veil's symbolic facilities.

### Complexity

Umpire3 introduces Lean expertise and a cross-language schema. Full independence initially
duplicates some runtime plumbing. This cost is intentional: premature reuse would constrain the
new semantics and hide whether a shared module is actually deep.

### Developer experience

Early authors write a disciplined subset of Lean: structures, inductives, definitions, matches,
relations, and theorem statements. Proof specialists handle difficult automation. Ergonomic macros
and generated Go arrive only after the semantic patterns and integration seam are stable.

### Security and operations

Formal tools and metaprograms execute at build time and are supply-chain/resource risks. Pin,
checksum, sandbox where practical, and cap them. Experiments retain semantic identifiers rather
than customer data. Deployment and canary authority are capability-scoped and are not implied by a
local adapter.

## Explicitly deferred decisions

These decisions have gates rather than unresolved placeholders:

- **Veil adoption:** decide during M5 after pure Lean exploration works. Adopt only if it consumes
  the same semantics and produces reproducible, normalizable traces.
- **Lean-to-Go generation:** decide at M11 only after M8 succeeds, two models use the schema, and
  boilerplate cost is measured.
- **Authoring macros:** add only after two domains reveal stable repeated forms. Until then use
  ordinary Lean.
- **Gobra:** pilot one narrow Go seam during M12. Retain only if its proof covers a meaningful
  implementation-to-system contract at sustainable cost.
- **Grove/Goose:** evaluate only after Gobra demonstrates a concrete expressiveness gap involving
  crash or distributed reasoning.
- **Shared extraction with Umpire2:** consider only after two independent working adapters satisfy
  the extraction rules in this document.
- **Umpire2 retirement:** out of scope. Make a separate decision using Umpire3 adoption, coverage,
  maintenance cost, and migration evidence.
- **Deployment and canary execution:** Umpire3 1.0 proves the architecture in local and CI
  environments. Production authority requires a separately approved 1.x milestone with explicit
  allowlists, tenant isolation, hard resource budgets, process-level cancellation, redaction, and
  recovery-tested cleanup. A local capability never implies deployment or canary authority.

## Umpire3 1.0 end state

Umpire3 1.0 is complete when all of the following are true:

- `tests/umpire3` has no dependency on any prior Umpire implementation.
- Lean is the sole authoritative source for Umpire3 product, system, and refinement semantics.
- The Lean semantic kernel proves executable and relational semantics agree for exported models.
- Nexus product and task-system models have a checked system-to-product refinement theorem.
- Workflow Update supplies a second product/system/refinement slice.
- Nexus and Update consume at least one shared proved system guarantee.
- A formal stale-completion counterexample is exported as a versioned semantic experiment.
- The independent Go runtime realizes that experiment against a real Temporal test cluster.
- A deliberate implementation defect is discovered, observed, minimized, and promoted to a
  deterministic regression; the fixed implementation passes it.
- Results distinguish proof, bounded exploration, conformance, violation, unsupported, and
  inconclusive outcomes.
- Every retained result records hashes, bounds, assumptions, environment, evidence, omissions,
  and cleanup status.
- Local and CI profiles are reproducible and fail closed.
- The protobuf boundary for the Nexus slice is descriptor-derived and semantically interpreted.
- Toolchains and generated artifacts are pinned, checksummed, deterministic, and resource-bounded.
- Code generation, Veil, macros, and implementation verification are included only if their gates
  were met; none is required merely to claim architectural completeness.
- Umpire2 remains usable and unchanged except for separately approved shared extractions.

At that point Umpire3 has demonstrated the full chain:

```text
product meaning
      ▲ proved refinement
distributed system model
      ▲ qualified implementation evidence
Temporal Go

Lean counterexample
      │
      ▼
semantic experiment
      │
      ▼
Umpire3 realization and observation
      │
      ▼
minimized reproducible regression
```

The project should expand only where a new module has a stable semantic interface and a valuable
property that current testing misses. Success is not measured in converted files, modeled states,
or generated backends. It is measured in precise claims, proved refinement, useful discovered
traces, and defects prevented in the real Temporal implementation.
