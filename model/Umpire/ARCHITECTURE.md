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
| `Umpire.Case` | Case, Program, Contract, Run, typed value, and Verdict data. |
| `Umpire.Case.Compiler` | Checked Producer input and deterministic Case lowering. |

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

All public declarations carry stable Definition IDs, source locations, and behavior fingerprints.
Limits are stage-specific. Exhaustion and limit-reached outcomes remain distinct, and a planning
artifact never proves that runtime work occurred.

## Promotion API

`Umpire.Promotion` is scenario-neutral. `compilePromotionSource` accepts an unchanged checked Query,
its target-indexed planner kernel, a complete planning anchor, fresh source identities, and exact
expected bytes. It replans and rechecks the target-owned trace before returning one opaque
`CompiledPromotionSource`.

The module performs no runtime reproduction, reduction, replay, publication, or installation. Its
source template imports only generic Umpire modules and cannot receive caller-selected imports or a
namespace. The focused `Umpire.PromotionTests` build protects that boundary.

## Case Runtime IR

The `Umpire.Case` facade exposes a closed data vocabulary:

- `Value`, `ValueType`, `ValueExpression`, and `FieldPath` describe typed data and bounded access;
- `Program` contains roles, private Slot schemas, Observation schemas, typed entrypoint DAGs,
  cleanup, and limits;
- `Contract` contains deterministic safety and bounded-liveness rules, bounded captures, horizons,
  and work limits;
- `Run` contains immutable sequenced Run Events, disposition, cleanup status, diagnostics, and an
  embedded Verdict;
- `Case` binds one Program and one Contract to version, provenance, definitions, and Known Gaps.

The reusable IR contains no Temporal method name, client, credential, worker, callback, endpoint,
filesystem access, executable hook, or runtime registry.

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

## Case compiler

`Umpire.Case.Compiler.compile` accepts complete checked Producer data and either returns one Case or
a source-bound lowering error. Lowering validates stable bindings, Program and Contract closure,
types, paths, instruction contexts, limits, Known Gaps, and provenance. Unsupported input is rejected
instead of omitted.

The compiler is deterministic. `Umpire.Case.ProtoJSON.canonical` emits the canonical Case bytes used
at the Go boundary. Temporal-specific producer declarations live outside Umpire.

## Runtime handoff

The Lean package stops at canonical Case data. It does not open a Host, schedule instructions,
create workers, collect runtime credentials, or choose a Monitor. The Go root facade performs:

```text
PrepareCase(case, profile)
PreparedCase.Run(ctx, host)
```

Static preparation snapshots the admitted Case and Profile without Host I/O. The prepared Contract
creates the private Run-local Monitor. The internal Executor owns scheduling, recording, Slots,
effect handles, cancellation, and cleanup. Alternate Hosts are the environment extension seam.

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
- Case, Program, Contract, and Run vocabularies are finite, versioned, and bounded.
- A Program contains no clients, credentials, callbacks, or arbitrary executable code.
- A Contract is the sole authority for live and offline Verdict semantics.
- Slots are private execution state; Observations are the declared evidence surface.
- Preparation is static and immutable; one Prepared Case supports isolated concurrent Runs.
- Run disposition, cleanup status, and Verdict remain independent.
- Generated data and views cannot create behavior.
- Promotion remains generic and review-only.

## Superseded runtime history

The pre-fn-64 portable plan, resident executor, caller-specific adapter, and separate Run Evaluation
path were removed. Historical documents label those interfaces explicitly as superseded. They are
not compatibility surfaces of `Umpire.Case`.
