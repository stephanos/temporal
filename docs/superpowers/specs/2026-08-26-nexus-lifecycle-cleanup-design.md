# Nexus lifecycle cleanup design

## Goal

Make the ordinary `Temporal.Feature.Nexus` surface explain the three basic operation lifecycles
first:

1. start a scheduled operation;
2. cancel a started operation;
3. complete a started operation successfully.

The detailed AutoClose and caller-closure models remain available, but are explicitly experimental
and are not re-exported by the ordinary `Temporal.Feature` facade.

## Current problem

The current module hierarchy communicates the opposite priority:

- `Temporal.Feature.Nexus.AutoClose` is a large experimental design proof and tutorial, but it owns
  the operation state and transition relation used by the introductory lifecycle target.
- `Temporal.Feature.Nexus.Examples.BasicLifecycle` adapts that AutoClose relation into Umpire.
- `Temporal.Feature.Nexus.Examples.BasicOperations` demonstrates start and successful completion,
  but does not demonstrate cancellation.
- `Temporal.Feature` imports AutoClose and caller closure, while the simple operations live under
  `Examples` and receive a separate import from `Temporal`.

`Examples/` was introduced by `fn-11-basic-nexus-umpire-dsl-showcases` as a progressive teaching
path for Property, Behavior, Query, and planning. It was not intended to mean that the lifecycle is
noncanonical. In practice, however, the directory makes the basic product behavior appear less
important than the experimental AutoClose design.

## Chosen approach

Use a clean module break with no compatibility aliases:

```text
model/Temporal/Feature/Nexus/
  Lifecycle.lean
  LifecycleTests.lean
  Operations.lean
  OperationsTests.lean
  Experimental/
    AutoClose.lean
    CallerClosure.lean
    CallerClosureTests.lean
    testdata/
      nexus-caller-closure-experiment-spec.json
```

The old `Examples/` directory and old module names disappear. Call sites are updated directly so
the repository has one obvious import path for the simple Nexus model.

### Rejected alternatives

- **Keep `Examples/` as a tutorial facade.** This preserves the ambiguity about whether the simple
  lifecycle is product meaning or optional sample code.
- **Leave import-only compatibility modules at the old paths.** This reduces immediate import
  churn, but leaves two ways to discover and import the same model. The code has no external Lean
  compatibility requirement that justifies that extra interface.
- **Only move AutoClose.** This changes the directory tree without fixing the core model's
  dependency on experimental semantics or adding the missing cancellation lifecycle.

## Core lifecycle module

`Temporal.Feature.Nexus.Lifecycle` becomes the small semantic module for the focused lifecycle.
It must not import any module below `Temporal.Feature.Nexus.Experimental`.

The module defines its own focused state/event vocabulary and transition relation. The richer
AutoClose state graph remains internal to the experimental module; the ordinary lifecycle neither
aliases it nor treats it as an authority.

Its exposed transition surface is deliberately limited to:

| Initial state | Event | Resulting state | Meaning |
|---|---|---|---|
| `scheduled` | `start` | `started` | The handler acknowledges an asynchronous start. |
| `started` | `cancel` | `canceled` | Cancellation is accepted and the operation settles as canceled. |
| `started` | `succeed` | `succeeded` | The handler reports successful completion. |

Unsupported pairs produce no transition. In particular, a terminal operation cannot restart,
cancel again, or complete again. The focused model does not claim to cover retries, failures,
timeouts, termination, rejection, AutoClose policy, cancellation initiators, or caller ownership.

The module also owns the shared Umpire target machinery currently in `BasicLifecycle`: one
lifecycle capability, one provider, checked target composition, finite setup/action domains,
bounded planning, and a deterministic incremental kernel. This is the deep module: the mechanics
remain local while operation walkthroughs consume a small checked interface.

Existing start and completion declaration identities should remain stable where their meaning is
unchanged. Cancellation receives new identities. Semantic digests are versioned where the target's
action/state domain changes.

## Basic operations module

`Temporal.Feature.Nexus.Operations` replaces `Examples.BasicOperations`. Its top-level module
documentation presents the three paths in order: start, cancel, complete.

Each operation retains the existing authored-to-checked progression:

1. a portable Property states the expected target-owned state, outcome, and observation;
2. an exact one-action Behavior selects the action without choosing its result;
3. a checked Query combines the Property, Behavior, target, bounds, and policy;
4. deterministic planning produces the expected artifact.

The start and completion walkthroughs retain their current meaning and comments. Cancellation is
added with the same shape and test coverage. Successful completion remains explicitly described as
handler-reported progress, not a Nexus caller command.

## Experimental modules

The following modules move together because caller closure is an Umpire scenario built directly on
the AutoClose model:

- `Temporal.Feature.Nexus.Experimental.AutoClose`
- `Temporal.Feature.Nexus.Experimental.CallerClosure`
- `Temporal.Feature.Nexus.Experimental.CallerClosureTests`
- the caller-closure experiment fixture

The move preserves existing comments and proofs. Namespace, import, source-provenance, fixture,
inspector, generated projection, Makefile, and exact verification-consumer references are updated
to the experimental path.

The experimental models retain their existing semantic declaration identities unless a value's
meaning changes. Moving source provenance is not itself a reason to rename stable scenario IDs.
Generated artifacts that record source paths are regenerated rather than hand-edited.

## Facades and build surfaces

- `Temporal.Feature` imports `Nexus.Lifecycle` and `Nexus.Operations`; it does not import
  `Nexus.Experimental.*`.
- `Temporal` obtains the basic operations through `Temporal.Feature` and no longer needs a special
  `Examples.BasicOperations` import.
- `TemporalModelTests` imports the core lifecycle and operations tests.
- A separate `TemporalExperimentalTests` Lake target imports the experimental caller-closure and
  inspector tests. The full regression gate builds it, but `TemporalModelTests` does not import it.
- `Temporal.Tool.Inspect` may explicitly import the experimental caller-closure scenario while it
  remains registered; this opt-in dependency does not re-export the experiment through the normal
  feature facade.

Documentation leads with `Lifecycle` and `Operations`, then lists `Experimental/AutoClose` and
`Experimental/CallerClosure` as advanced historical/design material. References to the old
`Examples` paths are removed from live documentation.

## Failure behavior

This refactor introduces no runtime behavior or new runtime error path.

- Unsupported lifecycle state/event pairs return no transition.
- Invalid Umpire target, Property, Behavior, and Query declarations retain their existing typed
  errors.
- Missing or conflicting capability providers remain covered by the lifecycle tests.
- Stale experimental fixtures or generated projections fail their existing deterministic
  regression checks.

## Verification

Focused checks cover:

- all three supported lifecycle transitions;
- representative unsupported and terminal transitions;
- the exact lifecycle target setup and action domains;
- missing and conflicting provider errors;
- positive and negative Property/Behavior checks for start, cancellation, and completion;
- deterministic repeated planning and target-owned artifact contents for all three operations;
- experimental AutoClose and caller-closure proofs after the namespace move;
- inspector output, checked fixture, and generated regression projections after path changes.

Repository gates:

```sh
cd model && mise exec -- lake build Temporal.Feature.Nexus.LifecycleTests
cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests
cd model && mise exec -- lake build Temporal.Feature.Nexus.Experimental.CallerClosureTests
cd model && mise exec -- lake build TemporalModelTests
cd model && mise exec -- lake build TemporalExperimentalTests
make umpire-check-regression
make lint-model
```

The implementation must preserve existing comments in moved or refactored files and must avoid
overwriting the unrelated model-lint and architecture work already present in the working tree.
