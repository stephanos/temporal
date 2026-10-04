---
status: draft
---

# Product-facing Models and selective refinement

Design direction agreed in discussion, 2026-10-03. Concrete authoring and checker contracts remain
proposals. This note supports #AUTHORING, #ZOOM and #PORTABILITY in the
[vision](UMPIRE4_VISION.md). It changes neither the authoritative
[shared specification](UMPIRE4_SPEC.md) nor today's [model semantics](../model/SEMANTICS.md).

## Decision

Technically comfortable product owners review the actual executable product/interface Model.
That Model knows public RPCs, their meaningful options and outcomes, and user-visible commitments.
Readable shorthands name constrained request shapes. There is no mandatory translation into a
second Model that restates the same public behavior.

Authors add detail through checked refinements where a concrete question requires it. Refinement
depth has no architectural cap. Each module can stay abstract or gain further detail independently
of unrelated modules. Every check still has explicit finite domains and resource limits.

Environment access is a separate choice. Realizations and Profiles govern execution, evidence and
authorization; they do not define a parallel lifecycle for each deployment type.

## What belongs in the product/interface Model

Include a distinction when it changes what a user may rely on. Public RPC names, request options,
identity, rejection, duplication, cancellation, concurrent operations and durability can all belong
here. Readability means exposing these decisions clearly, without requiring the reviewer to know
Temporal's persistence or routing implementation.

An RPC invocation does not have to be one atomic transition. Separate invocation, meaningful system
commitments and response where their ordering changes allowed behavior. Otherwise a lost response
or a concurrent operation could disappear inside an overly coarse step.

Keep product promises as Properties separate from the behavior being checked. A faulty detailed
design must be able to violate the same promise without redefining it. Internal refinements may
add their own local invariants, but must not restate inherited product promises independently.

### Update example

The following distinctions illustrate the intended authoring, rather than declare a complete
Workflow Update contract or require one universal lifecycle:

| Distinction | Product meaning | What does not establish it |
| --- | --- | --- |
| Received | The system has received this request | Merely issuing the RPC |
| Durable request | The commitment to retain the request survives named failures | Receipt alone, or a requested wait stage |
| Completed | The update has produced its outcome | Acceptance alone |
| Retained result | The outcome remains obtainable under the declared retention and failure conditions | Completion without that additional promise |

A received-but-not-durable request may be lost under failures the Model explicitly permits.
A durable request must survive the failures its Property names. Durability needs a subject, failure
scope and retention conditions; request durability and result durability need not be the same claim.
Durability alone also does not establish eventual execution without the stated progress premises.

Keep actual state separate from knowledge. After a response is lost, the request may already be
durable even though the caller cannot establish that. Evidence interpretation retains the possible
executions consistent with the records. Unknown durability must never be classified as definitely
not durable.

## Named request shapes

A request shape is a proposed named constraint over a typed public RPC request. For example,
`SubmitUpdateAwaitingAcceptance` can name `UpdateWorkflowExecution` with the request's
`wait_policy.lifecycle_stage` constrained to `ACCEPTED`. This is illustrative vocabulary, not new
Scala syntax or an implemented declaration form.

The same definition should support construction of generated requests, validation of supplied
requests and recognition of recorded requests. Binding an Action to a shape must not create a
second lifecycle, independent Property catalog or handwritten test oracle. Preserve source identity
so a generated request or rejected input can point back to its shape declaration.

Proposed constraints on the design:

- Required options are constraints, not defaults a caller can override silently. Conflicting
  requirements produce a located error before execution.
- Presence, equality, allowed values and relationships between fields are explicit. An absent
  field and a present default value are distinguished where the interface gives them different
  meaning. The first implementation should support a bounded, declared subset of predicates.
- Other fields have explicit parameters, input classes, defaults or scope decisions. Omission from
  the shorthand does not establish that a field is behaviorally irrelevant.
- Shapes may intentionally overlap. Where a binding needs unique classification, ambiguity is
  diagnosed. No first-match rule silently discards another interpretation. A family intended to
  cover a declared input domain gets a scoped exhaustiveness check; unmatched requests remain
  explicit rather than disappearing from conformance.
- A shorthand cannot hide a retry loop, polling protocol or extra lifecycle. Such behavior belongs
  in the Model or an explicit realization with its existing ownership and checking rules.

The constraint describes the request, not its result. Temporal's published `WaitPolicy` says a
server timeout can return before the requested lifecycle stage is reached. Requesting acceptance
therefore cannot serve as evidence that acceptance or durability occurred.
[Temporal API definition](https://raw.githubusercontent.com/temporalio/api/master/temporal/api/update/v1/message.proto).

Typed descriptor support establishes that methods, fields and values exist. The Model still owns
their meaning. A new API field or enum member must be accounted for rather than silently entering
an existing abstraction class.

## Selectively deeper Models

A feature might refine its public behavior into distributed coordination, refine its queue into a
retention and recovery protocol, then refine storage behavior further. These are possible subjects,
not mandatory tiers. Another feature may bind its product/interface Model directly to execution.

Each refinement names its abstract machine and interface, state map, visible outcomes and facts,
additional assumptions, and supported claims. Internal steps may leave the abstract state unchanged.
They cannot hide an externally relevant outcome or fact under the label of an internal step.

Checking detailed behavior against allowed abstract behavior is only part of assurance. Preserve
explicitly required behavior and recheck progress under the detailed machine's assumptions. Removing
an optional alternative need not be a defect; removing a required admission or allowing endless
internal activity can be. A bound in abstract steps cannot become the same numeric bound in concrete
steps without justification. Record units and any translation.

For a chain of refinements, a reused claim records every mapping, selected variant, assumption and
limit it depends on. Unsupported transitive claim transport stays explicit. A successful local
mapping check must not silently advertise preservation of arbitrary Properties or end-to-end timing.

The existing reader has a state-map and row/stutter refinement check. It is a foundation, not proof
that all of these proposed obligations or arbitrary refinement chains work today. The
[modalities study](MODALITIES.md) identifies related required-behavior and progress questions.

## Separate refinement, execution and evidence

| Relation | Responsibility |
| --- | --- |
| Refinement | Relate a detailed Model's behavior to an abstract Model and check the claimed preservation obligations |
| Realization | Bind model actions and observations to runtime instructions, sources and controls |
| Evidence interpretation | Determine which executions fit a Run and what their evidence settles |
| Profile admission | Check the environment's permissions, resources, controls and supported execution before target I/O |

A detailed Model may be useful for design checking even when no canary can observe its internals.
Canaries use supported public observations. A controlled environment can expose additional internal
evidence and fault cut points. Less evidence can make a claim inconclusive; it cannot weaken the
promise. Unsupported required execution capabilities reject a Case before contacting the target.

Black-box access is relative to the subject being checked. Storage calls are internal when testing
Temporal's public behavior, but can be the public interface when testing a storage module. Neither
classification fixes the abstraction depth of that module's Model.

## Alternatives considered

- **Mandatory product, public-protocol and internal tiers.** Rejected because the product Model
  already knows public RPCs. A second public description would often duplicate actions and promises.
  Introduce a separate protocol Model only when it adds behavior worth checking.
- **A maximum of two levels.** Rejected because a queue or recovery question can require further
  refinement without adding detail to the whole feature.
- **One fully detailed Model with filtered displays.** A readable display remains useful, but
  hiding fields does not reduce the checked state space or establish refinement correctness.
- **A behavior Model per environment.** Rejected because environment access should change available
  execution and evidence, without silently changing the feature's commitments.

## Assurance and acceptance examples

The [assurance plan](MODEL_ASSURANCE.md) owns the checks rather than a second assurance framework.

| Example | Required evidence |
| --- | --- |
| Product owner reviews the update Model | They can explain receipt, durability, completion and ambiguous responses without reading storage mechanics |
| Generate and recognize a named request shape | Both use the same constraints; a conflicting wait stage fails at the declaration or binding |
| Omit or overlap an input case | Report the scope gap or ambiguous binding without silently discarding the request |
| Mutate a shape's acceptance-stage constraint | A pinned control detects the changed request semantics; merely regenerating expected values is insufficient |
| Response disappears after commitment | Retain the committed explanation until evidence settles it; a timeout alone cannot prove loss |
| Refine a queue, then its recovery protocol | Expose the selected interruption while unrelated modules remain abstract |
| Detailed behavior drops required work or never progresses | The unchanged product obligation fails, even if the state map and safety-only checks pass |
| Hide a visible fact in a mapping | A mapping control rejects it and identifies the lost product fact |
| Run with fewer observation sources | Preserve the promise and report the remaining uncertainty or reject unsupported required capabilities |

## Handoffs and decisions still needed

Keep the direction in the vision and this rationale here. Add concrete contracts to the shared
specification only after design review; add behavior to `model/SEMANTICS.md` only when implemented.
No schema change, task creation, code change or relaxation of an existing spec freeze follows from
this note alone.

Use fn-117's typed API surface and fn-114's declaration ownership when designing shapes. Coordinate
with fn-118's API behavior hints, while preserving its distinction between runtime waiting and Model
semantics. Reuse fn-120's lint/report surfaces, fn-122's shared laws and fn-123's fault and durability
work where their contracts apply.

Before implementation, settle the smallest request-predicate language and its IR representation,
overlap handling at each use site, ownership of response/evidence bindings, and the exact Properties
and bounds supported through multiple refinements. Validate those choices on one update-shaped
example and one selectively refined provider before proposing a general abstraction framework.
