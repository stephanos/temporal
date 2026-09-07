# Umpire public API

Umpire is the reusable, Temporal-independent Lean library for semantic modeling, finite planning,
reviewed promotion, and Case production. For the cross-library map and Go runtime boundary, see the
[model architecture](../ARCHITECTURE.md).

## Imports and modules

Most consumers import the umbrella facade:

```lean
import Umpire
```

Focused public imports are available by responsibility:

| Import | Public responsibility |
| --- | --- |
| `Umpire.Core` | Stable definitions, traces, capabilities, laws, and finite kernels. |
| `Umpire.Target` | Finite-machine and expert Target authoring plus checked composition. |
| `Umpire.Property` | Property authoring, validation, and pure trace evaluation. |
| `Umpire.Behavior` | Setup and trace-shape authoring and validation. |
| `Umpire.Query` | Bounded questions over a checked Target, Properties, and Behavior. |
| `Umpire.Space` | Checked finite axes, request-only faults, and atomic point compilation. |
| `Umpire.Exploration` | Bounded finite selection, pinned precedence, and process-local sessions. |
| `Umpire.Observation` | Offline evidence mappings and accepted semantic traces. |
| `Umpire.ImplementationLink` | Checked correspondence between independent semantic Targets. |
| `Umpire.Planning` | Deterministic incremental planning over checked Queries. |
| `Umpire.Promotion` | Exact review-only source compilation from an unchanged planned Query. |
| `Umpire.Artifact` | Retained model-planning and offline-analysis artifact codecs. |
| `Umpire.Json` | Ordered JSON construction for codec owners. |
| `Umpire.Case` | Umpire provenance and temporary aliases for generated Testpilot protocol types. |
| `Umpire.Case.Compiler` | Generated Case assembly, source-bound producer diagnostics, and Umpire provenance. |

Implementation modules remain behind these facades. Reusable Umpire modules cannot import the
domain-specific Temporal modules; the complete import graph is enforced by `make lint-model`.

## Semantic model lifecycle

A model maintainer defines a checked Target once. Ordinary authors then define independent
Properties and Behaviors, combine them in a bounded Query, and plan or explore only through that
checked Target. Target-owned transitions decide outcomes; authoring order and instance search do
not select behavior.

```text
AuthoredTarget ── checkTarget ──▶ CheckedTarget
                                     ├── Property
                                     ├── Behavior
                                     └── Query ──▶ Planning / Space / Exploration
```

The finite-machine adapter is the ordinary route for fully enumerable Targets. Direct
`TransitionKernel` construction remains the expert route when authoritative propositions are
specified independently. Both routes converge before Property, Behavior, or Query checking.

`FiniteMachine.targetDefinition` and `FiniteMachine.authoredTarget` assemble the ordinary finite
Target without deriving its evidence. The author still owns every ordered domain, encoder,
enumerator, closure proof, Action-executability proof, provider, connector, source, and stable ID.
`checkTarget` remains the semantic admission boundary. Property, Behavior, Query, and Observation
constructors follow the same raw input → typed `check` result → explicitly proof-backed `checked`
shape; no constructor infers Target outcomes or checker success.

`FiniteTable` keeps ordered typed catalogs, setup alternatives, transition alternatives, Model
Outcomes, and Model Facts explicit, then validates domain closure before constructing the ordinary
finite Target. `DefinitionFamily`, `PropertySpec`, `ExactSequenceSpec`, `QuerySpec`, and
`QueryLimitSpec` reduce repeated structure while delegating to the existing language-owned checkers.
Their `checked` operations require explicit proof of checker success; the ordinary `check`
operations return the existing typed `Except` results.

Version-two Property data adds typed Boolean predicates, same-step case groups, and guarded bounded
temporal clauses. Boolean composition is limited to `atom`, `all`, `any`, `not`, and `oneOf` over
the field contexts admitted by the Property checker. Every applicable obligation is conjoined.
Exceptions are trigger-time applicability conditions and do not select a winning case or retract a
pending temporal obligation. Case analysis reports coverage, overlap, logical conflict, modeled
incompatibility, exhaustive completion, and limit exhaustion as separate bounded results.

All public declarations carry stable Definition IDs, source locations, and behavior fingerprints.
Limits are stage-specific. Exhaustion and limit-reached outcomes remain distinct, and a planning
artifact never proves that runtime work occurred.

Checked Queries may carry a default-empty `KnownGapSet`. Planning composes authored and phase-owned
sets once before traversal and returns a typed conflict before artifact publication. Known Gaps are
nonbehavioral: they are excluded from Query behavior fingerprints, cannot establish success, and do
not imply that absent limitations were discovered. Case lowering copies the exact composed rows;
runtime interpretation remains outside the semantic authoring path.

## Promotion API

`Umpire.Promotion` is scenario-neutral. `compilePromotionSource` accepts an unchanged checked Query,
its target-indexed planner kernel, a complete planning anchor, fresh source identities, and exact
expected bytes. It replans and rechecks the target-owned trace before returning one opaque
`CompiledPromotionSource`.

The module performs no runtime reproduction, reduction, replay, publication, or installation. Its
source template imports only generic Umpire modules and cannot receive caller-selected imports or a
namespace. The focused `Umpire.PromotionTests` build protects that boundary.

## Testpilot protocol and Umpire provenance

The checked-in `.proto` closure rooted at
`proto/internal/temporal/server/api/testpilot/v1/case.proto` owns the closed wire vocabulary.
`Testpilot.Protocol` exposes its generated Lean declarations, including Case, Program, Contract,
Run, values, paths, expressions, instructions, monitors, and Verdict data. Producers construct
those generated values through the context-safe `Testpilot.Authoring` facade.

`Umpire.Case` owns only Umpire-specific definition bindings, Behavior Fingerprints, sources, and
Known Gaps. `Umpire.Case.Provenance` encodes that data into opaque `producerData`; Testpilot and Go
do not interpret its contents. Temporary `Umpire.Case` type aliases delegate directly to the
generated declarations and are removed when downstream imports use `Testpilot.Protocol` directly.
The compatibility codec delegates directly to `Testpilot.ProtoJSON` and is covered by an
equivalent-output regression under `Testpilot.Tests` until its callers migrate.

The generated protocol contains no client, credential, worker, callback, endpoint, filesystem
access, executable hook, or runtime registry.

### Slots and Observations

A Slot is immutable single-assignment data used by later instructions. Slot storage is private to
one execution and is never exposed as a public scheduler API or recorded automatically.

An Observation is a declared typed projection on a Run Event. Contracts may inspect Observations
and selected Run Event fields. They cannot inspect arbitrary Slots or raw request and response
payloads. This opacity limits evidence authority; it does not classify every response field as
secret.

### Program instructions

Version one supports a closed set of generic instructions: authorized unary RPC invocation, Slot
await, Nexus completion, SDK Nexus start and await, workflow/activity finish, and Nexus response.
Every node declares its context, dependencies, optional guard, typed outcome schema, optional
response projections, activation reservations, and exact bounds. Unsupported context/opcode pairs
reject during preparation.

Programs are acyclic. Descriptor paths have a bounded grammar with explicit presence, oneof,
repeated fanout, and literal map-key selection. There is no general JSONPath or arbitrary expression
language.

### Contract semantics

Each rule has one initial state, finite transitions, and terminal satisfied or violated states.
Bounded-liveness rules have explicit horizons. The Evaluator checks expiry before transitions for
every Run Event kind, so a matching event at the deadline cannot revive an expired rule.

Captures copy only declared values, are bounded by count, bytes, and work, and are isolated per rule
and per Run. A transition may cite the matching event as support; Verdicts retain exact supporting
event sequences. Pending rules close inconclusive. A proved violation has precedence over later
operational and cleanup failures.

## Case production

Each Lean Producer owns its checked semantic lowering into generated protocol values through
`Testpilot.Authoring`. Umpire-backed Producers pass those values to `Umpire.Case.Compiler`, which
validates source-bound property rows, preserves typed unsupported-lowering errors, converts Known
Gaps in order, attaches exact opaque provenance, and performs final generated Case assembly. Its
input contains generated protocol values and introduces no parallel wire representation. A
Testpilot-only synthetic Producer can assemble a generated Case directly.

`Testpilot.ProtoJSON.canonical` delegates canonical Case encoding to `Protobuf.Json`;
`Umpire.Case.ProtoJSON.canonical` is only a temporary forwarding compatibility name. Testpilot's
`Prepare` owns static admission, including Program and Contract closure, types, paths, instruction
contexts, limits, identity, scope, and environment policy, before Driver I/O. Temporal-specific
Producer declarations live outside Umpire.

## Runtime handoff

The Lean package stops at canonical Case data. It does not open a Driver, schedule instructions,
create workers, collect runtime credentials, or choose a Monitor. Testpilot performs:

```text
testpilot.Prepare(case, profile)
PreparedCase.Run(ctx, driver)
```

Static preparation snapshots the admitted Case and Profile without Driver I/O. The prepared Contract
creates the private Run-local Monitor. The internal Executor owns scheduling, recording, Slots,
effect handles, cancellation, and cleanup. Alternate Drivers are the environment extension seam.

Exact Case 1.0 is the sole format. Resource-bearing Programs declare symbolic text resources; their
IDs identify namespace, task-queue, and named Nexus endpoint relationships but contain no
physical names or transport addresses. The Profile owns physical binding values. `Prepare` resolves
them into immutable private prepared data and includes the complete binding fingerprint in Prepared
Case identity while preserving the source Case bytes. `Run` checks the Driver's matching identity,
calls its no-I/O `Validate` hook before Monitor creation, and calls `Open` only after validation.
This environment identity does not alter Behavior Fingerprints, Contract meaning, or producer-owned
provenance.

## Artifact and generated-view boundaries

Planning artifacts, the semantic inventory, and Generated Views remain deterministic projections
owned by their existing modules and generators. They do not execute Cases or define Contract
results. The Case conformance tree is separately owned by the Case renderer and transactional Go
publisher.

Lean Case fixtures compare byte-for-byte. Runtime results use a named closed stable projection only
for stable fields, while excluded Run IDs, event timing and identities, causal links, activation
identities, support references, and diagnostics are structurally validated. There is no generic
normalization or ignored-field registry.

## API invariants

- Reusable Umpire code is Temporal-independent.
- Public semantic declarations are checked before planning or Case lowering.
- The generated Testpilot Case, Program, Contract, and Run vocabularies are finite, versioned, and
  bounded by their `.proto` schema.
- A Program contains no clients, credentials, callbacks, or arbitrary executable code.
- A Contract is the sole authority for live and offline Verdict semantics.
- Slots are private execution state; Observations are the declared evidence surface.
- Preparation is static and immutable; one Prepared Case supports isolated concurrent Runs.
- Resource-bearing Case 1.0 Programs use complete symbolic bindings; resource-free Programs may have an empty environment.
- Run disposition, cleanup status, and Verdict remain independent.
- Generated data and views cannot create behavior.
- Promotion remains generic and review-only.

## Superseded runtime history

The pre-fn-64 portable plan, resident executor, caller-specific adapter, and separate Run Evaluation
path were removed. Historical documents label those interfaces explicitly as superseded. They are
not compatibility surfaces of `Umpire.Case`.
