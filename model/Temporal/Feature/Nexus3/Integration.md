/-!
# Nexus3 Case integration draft

Proposed binding syntax. Read `Nexus.md` first. This file keeps identity conventions, execution
choices, and evidence interpretation out of the feature-facing model. Both Markdown files remain
design specimens; the checked success-only completion Producer lives in `Testpilot.lean`.
Nexus operation cancellation in this draft is explicitly deferred to fn-79. No cancellation Target,
evidence adapter, operation capability, or Case is delivered by the generic scoped qualification.

## Admission and identity

IDs derive from `feature namespace + declaration kind + relative declaration name`. This feature's
namespace is `temporal.nexus3`; for example, the progress Property gets the ID
`temporal.nexus3.property.cancellationResolves`. Ordinary references still resolve to declarations,
not strings. A future checker derives IDs after resolving those references; authors keep no registry.

Nested declarations use their named owner to disambiguate them. Examples:
* state `lifecycle.scheduled` → `temporal.nexus3.state.lifecycle.scheduled`
* relation `lifecycle.start` → `temporal.nexus3.relation.lifecycle.start`
* clause `cancellationResolves.terminalResponse` →
  `temporal.nexus3.property.cancellationResolves.terminalResponse`
* occurrence `cancellationRace.start` → `temporal.nexus3.occurrence.cancellationRace.start`
* setup role `cancellationRace.operation` →
  `temporal.nexus3.setup.cancellationRace.operation`.

Here "stable" means unchanged across builds, comment edits, and declaration reordering. A rename
changes the derived ID, including the IDs of declarations whose relative names contain that owner.
The Nexus3 demonstration has no compatibility overrides or aliases; generated consumers regenerate
after a rename.

Admission must reject malformed IDs, duplicate or conflicting declarations, ambiguous references,
and wrong-kind references before planning or Case production. Derivation does not use
source positions, declaration order, or enum numeric ordinals. Semantic fingerprints still come
from checked meaning, including outcome alternatives, action roles, scope, and bounds. Identity
never supplies missing transitions or makes a Property pass.

Authors provide no identity or version input for these declarations. Generated
`DefinitionMetadata` retains Umpire's format version `1` because Target admission and artifact
provenance consume that core field; `Temporal.Shared.definitionMetadata` supplies it outside the
Nexus3 syntax.

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
same operation. This requires operation-scoped counting before evaluation; merely filtering the
response while continuing to count global steps is incorrect. Nexus2's global transition unit
cannot be reused unchanged for this meaning.

Ordinary timed Contract rules use elapsed milliseconds. The generic version-one scoped Contract
capability now counts admitted operation transitions, with checked lowering through
`Umpire.Case.Scoped.lower`; there is no implicit conversion to milliseconds, instruction counts,
or raw Run Event counts. This is qualified by non-cancellation typed fixtures through public
Prepare/Run. Cancellation progress still rejects at Case production because the cancellation
Target, evidence adapter, and operation capability remain deferred to fn-79. A Known Gap must not turn that requested Property into a weaker timed Property.
Separately configured execution timeouts close unresolved evaluation as inconclusive, not as
proof that the model's one-step requirement was violated.

## Current support and reuse

The successful-completion Query has checked mapping and lowering through `produceCompletionCase`
and `completionCase`. Cancellation Queries remain unsupported until the Program can retain and
address the matching operation handle. The mappings below state that executable boundary and its
rejection requirements.

`Temporal.Testpilot.asyncNexusCase` is a useful example of setup, asynchronous handler response,
completion capability, and history correlation. It has independently authored Program/Contract
meaning; returning it under a Nexus3 Query ID would not establish Nexus3 lowering correctness.
The Nexus3 Producer validates the checked mapping and lowers generated Program and Contract values.
`Umpire.Case.Compiler.compile` then validates the source-bound rule rows, preserves unsupported
construct errors, attaches exact Umpire provenance, and performs final generated Case assembly.
The generated fixture is prepared and run in the live integration test against two distinct
namespace, task-queue, and Nexus endpoint bindings. Both environments retain the same symbolic Case
and satisfied Contract meaning; only their Profile binding fingerprints and Driver identities differ.

Nexus2's finite admission and typed authoring helpers are candidates for reuse. Its baseline
allows a started setup and immediate cancellation; its race starts already running. Neither
Target can substitute for the scheduled-only, request-then-resolution model here unchanged.

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

import Temporal.Feature.Nexus3.Nexus

namespace Temporal.Feature.Nexus3

integration lifecycleCases on lifecycle
  identity namespace "temporal.nexus3"
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

end Temporal.Feature.Nexus3
