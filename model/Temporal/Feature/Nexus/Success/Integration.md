/-!
# Nexus success Case integration draft

Proposed binding syntax. Read `Nexus.md` first. This file keeps identity conventions, execution
choices, and evidence interpretation out of the feature-facing model. Both Markdown files remain
design specimens; the checked success-only completion Producer lives in `Testpilot.lean`.
Nexus operation cancellation in this draft is explicitly deferred to fn-79. The generic correlated
qualification delivers no scheduled-only cancellation Target, evidence adapter, operation
capability, or Case; *Current support and reuse* below says why the historical already-started
Target is not one either.

## Admission and identity

IDs derive from `feature namespace + declaration kind + relative declaration name`. This feature's
namespace is `temporal.nexus.success`; for example, the progress Property gets the ID
`temporal.nexus.success.property.cancellationResolves`. Ordinary references still resolve to declarations,
not strings. A future checker derives IDs after resolving those references; authors keep no registry.

Nested declarations use their named owner to disambiguate them. Examples:
* state `lifecycle.scheduled` → `temporal.nexus.success.state.lifecycle.scheduled`
* relation `lifecycle.start` → `temporal.nexus.success.relation.lifecycle.start`
* clause `cancellationResolves.terminalResponse` →
  `temporal.nexus.success.property.cancellationResolves.terminalResponse`
* occurrence `cancellationRace.start` → `temporal.nexus.success.occurrence.cancellationRace.start`
* setup role `cancellationRace.operation` →
  `temporal.nexus.success.setup.cancellationRace.operation`.

Here "stable" means unchanged across builds, comment edits, and declaration reordering. A rename
changes the derived ID, including the IDs of declarations whose relative names contain that owner.
The checked success demonstration — `Nexus.lean` and the Producer in `Testpilot.lean` — has no
compatibility overrides or aliases at all: derivation is its only identity source, and generated
consumers regenerate after a rename.

The broader draft keeps one optional escape from that default, so a rename need not break an
existing consumer: an individual declaration may carry an explicit compatibility ID instead of its
derived one. It is described here, not offered — no override spelling, alias table, or registry is
proposed, and the success syntax above accepts no such input. Its intended meaning is narrow:

* Declaration-local. An override names exactly one declaration and does not cascade to the
  declarations it owns. A clause, occurrence, state, or relation under a renamed owner keeps
  deriving from that owner's current name unless it carries an override of its own.
* Identity only. An override preserves the ID a consumer addresses. It does not freeze that
  declaration's Behavior Fingerprint, its source-bound provenance, or any other checked meaning.
* Ordinary admission. An override is an authored ID like any other, judged on the same grounds.

Admission must reject malformed IDs, duplicate or conflicting declarations, ambiguous references,
and wrong-kind references before planning or Case production — overridden and derived IDs alike,
so an override can neither introduce a colliding identity nor rescue a rejected one. Derivation
does not use source positions, declaration order, or enum numeric ordinals. Behavior Fingerprints
still come from checked meaning, including outcome alternatives, action roles, scope, and bounds;
changed meaning changes the fingerprint whether or not the ID was overridden. Identity never
supplies missing transitions or makes a Property pass.

Authors provide no identity or version input in the delivered success syntax; the override above is
proposed for the broader draft only. Generated `DefinitionMetadata` retains Umpire's format
version `1` because Target admission and artifact provenance consume that core field;
`Temporal.Shared.definitionMetadata` supplies it outside the Nexus success syntax.

The produced Testpilot Case has a separate wire version. The checked completion selection emits one
Case 1.0 Program whose symbolic namespace, task-queue, and Nexus endpoint IDs are stable resource
references, not physical names or transport addresses. Preparing those exact Case bytes against a
Profile resolves environment-owned values without changing the checked definitions, Behavior
Fingerprints, Contract, or Umpire provenance.

## Commands, waits, and observation correlation

There is one logical `operation` role. Setup must establish a scheduled Nexus operation before
the first modeled acknowledgment. The concrete setup needs workflow, worker, endpoint, and task
queue roles; those are Case/Host concerns rather than additional product states.

A command initiates work; a wait recognizes a correlated observed event. A successful command
return is not evidence that the operation reached its modeled result. For example, cancellation
must be confirmed by the matching history event before emitting `cancellationRequested`.
`cancelRequested` describes the logical state immediately after that event, not a claim that a
later history read will still find the operation pending cancellation.

Evidence must bind namespace, workflow/run identity, and scheduled-event identity to this role.
Match event-type-specific scheduled-event references and request IDs where the schema supplies
them. History order within that workflow is the causal order; do not correlate by timestamps.
Repeated history reads must not count the same event twice. Missing, contradictory, or unrelated
evidence cannot advance the model or prove satisfaction.

The binding table describes the required meaning, not existing function names. Waits generate
bounded observation work, never a command named "resolve" or an instruction choosing its result.
A fixture may arrange asynchronous handler completion, as the existing Case example does, but
only its recorded completion evidence can establish the model's completed outcome.

## Model steps and runtime bounds

An operation transition is one admitted semantic step for the bound operation, reconstructed
from correlated evidence. Several Run Events can support one such step; duplicate reads and
events for other operations contribute zero steps. An event cannot satisfy another operation's
obligation. Each triggered obligation retains its own operation identity and starting coordinate.

The one-step progress Property permits the response on the trigger step or the next step of the
same operation. This requires operation-correlated counting before evaluation; merely filtering the
response while continuing to count global steps is incorrect. The Nexus race tree's global step unit
cannot be reused unchanged for this meaning.

A classic Contract rule declares exactly one deadline bound: elapsed milliseconds, or a count of the
Run Events the rule itself evaluated since its last transition. The generic version-one correlated
Contract capability counts admitted operation transitions instead, with checked lowering through
`Umpire.Case.Correlated.lower`. The three units are separate: there is no implicit conversion between
them, and a correlated rule never falls back to either of the classic bounds. That generic capability is delivered, and non-cancellation typed fixtures
qualify it through public Prepare/Run. Only its cancellation-specific use is still unsupported:
cancellation progress rejects at Case production because the scheduled-only cancellation Target,
its evidence adapter, and the operation capability remain deferred to fn-79. A Known Gap must not
turn that requested Property into a weaker timed Property.
Separately configured execution timeouts close unresolved evaluation as inconclusive, not as
proof that the model's one-step requirement was violated.

## Current support and reuse

The successful-completion Query has checked mapping and lowering through `produce` and
`completionCase`. Cancellation Queries remain unsupported until the Program can retain and
address the matching operation handle. The mappings below state that executable boundary and its
rejection requirements.

`Temporal.Testpilot.asyncNexusCase` is a useful example of setup, asynchronous handler response,
completion capability, and history correlation. It has independently authored Program/Contract
meaning; returning it under a Nexus success Query ID would not establish Nexus success lowering correctness.
The Nexus success Producer, `produce`, carries the checked values into generated Program and Contract
values. It compares no checked value against an expected model: a different Target, Scenario, Query
or Property produces different Case bytes. The Contract carries no monitor rule at all. Each
`require` clause becomes one operation-correlated bounded-response clause, placed by the Action order
the Scenario fixes: from the operation's first Action, the required value is due within as many
semantic steps as the Scenario puts between them, and `Umpire.Case.Correlated.lower` certifies the
correspondence between the checked clauses and the emitted capability. The evidence those clauses
read is lifted out of the same history read the Case already performs, keyed by the scheduled event
a started or a completed Nexus event names, which is the only identity either records.

Four claims reject by name rather than lowering something weaker: a Query that selected no trace,
a clause form with no trigger and response to carry, a value constraint the portable predicate
vocabulary has no spelling for, and a required value the selected trace already reaches before the
Action the clause names — the last because the window includes its trigger step, so such a clause
could be answered without that Action ever being observed.
`Umpire.Case.Compiler.compile` then validates the source-bound rule rows, preserves unsupported
construct errors, attaches exact Umpire provenance, and performs final generated Case assembly.
The generated fixture is prepared and run in the live integration test against two distinct
namespace, task-queue, and Nexus endpoint bindings. Both environments retain the same symbolic Case
and satisfied Contract meaning; only their Profile binding fingerprints and Driver identities differ.

The Nexus race tree's finite admission and typed authoring helpers are candidates for reuse. Its baseline
allows a started setup and immediate cancellation; its race starts already running. Neither
Target can substitute for the scheduled-only, request-then-resolution model here unchanged.

`Cancellation.lean` is one such reuse already taken: it derives a separate Target from
`Nexus.Race.Race` and adds the explicit terminal closure that Race lacks. It is a historical
already-started slice, not this draft's Target — it begins at a running operation rather than a
scheduled one, and it carries the race tree's states and Actions under its own Target identity.
`Temporal/System/Nexus/ImplementationLink.lean` adds an offline Lean evidence projection over it,
distinguishing submission from confirmation and admitting either resolution. Nothing lowers that
projection: it reaches no Testpilot evidence path, no operation-level cancellation capability, and
no Case, and no test exercises it. It therefore qualifies no runtime behavior and satisfies none of
the scheduled-only cancellation contract deferred to fn-79. Its presence is not cancellation
support here.

The selected cancellation mechanism is the workflow SDK cancellation function from the dedicated
`workflow.WithCancel` context used to start this operation. `tests/nexus_workflow_test.go` uses
this pattern. Cancel only that operation's context, not the workflow or worker activation.
The current closed Case Program does not expose this operation-level cancellation capability;
its future instruction must retain and address the matching handle. Until that extension exists,
cancellation Cases must reject before Host I/O. A cancellation Query cannot silently fall back
to the successful-completion fixture or use a standalone-operation RPC for a workflow operation.

Runtime execution checks a selected Case against its Contract. It does not prove an exhaustive
model Query, and failure to observe a witness in one Run does not establish bounded model absence.
Unsupported Properties reject the whole requested Case; do not drop a rule or publish partial
success. Supported producers must also carry the scope omissions from Nexus.md as checked
Known Gaps in Case metadata. Gaps disclose limitations; they never waive a requested Property.
-/

import Temporal.Feature.Nexus.Success.Model

namespace Temporal.Feature.Nexus.Success

integration lifecycleCases on lifecycle
  identity namespace "temporal.nexus.success"
    derive from kind and qualifiedName

  actions for operation
    awaitStart      => wait history NexusOperationStarted
    awaitSuccess    => wait history NexusOperationCompleted
    requestCancel   => command cancelSDKOperationContext
                      confirm history NexusOperationCancelRequested
    awaitResolution => wait history oneOf [NexusOperationCanceled, NexusOperationCompleted]

  admission
    require checkedTargetAndQueries
    require uniqueWellFormedIdsAndTypedReferences
    require terminalStatesHaveNoOutgoingRows
    require completeActionBindingsAndCorrelatedObservations
    require exactPropertyLowering
    require checkedKnownGaps

  unsupported
    cancellationSafety  => operationCancellationInstructionUnavailable
    completionCanWin    => operationCancellationInstructionUnavailable
    cancellationProgress => operationCancellationInstructionUnavailable, cancellationEvidenceAdapterUnavailable

end Temporal.Feature.Nexus.Success
