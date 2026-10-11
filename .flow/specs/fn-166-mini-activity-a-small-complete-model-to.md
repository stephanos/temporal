# Mini activity: a small, complete Model to iterate the DSL on

## Conversation Evidence

> user (2026-10-11): "our turnaround time is way too slow; since our model size is too large and validation time is too slow."
> user: "how about the put the current model aside for a moment and work with a simplified one for now."
> user: "it should still be realistic in that it touches on all these DSL aspects including product vs protocol vs realization and has some depth, but is small enough to iterate _much_ faster on"
> user: "propose speicfic actions" / "yes do it"
> agent measurement (pasted by the user): compile 20–40 s; syntax lint 7–30 s; focused suites 20–60 s each; steps-only equivalence 40–50 s; full gate about 5 min; complete source seal (production lift, 5.3M-row native comparison, ledgers, mutation controls, broad suites) 20–30 min; the canonical Make equivalence route is blocked by stale linked protobuf generation.

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 50% [user], 30% [paraphrase], 20% [inferred] -->

The Scala batch (fn-159 through fn-165) changes the DSL, and every change is validated against the full Temporal Models: `activity-standalone.json` alone is 4.7 MB of IR, the complete source seal compares 5.3 million rows and takes 20 to 30 minutes, and the time around it, deciding what proof a change needs and building proof machinery when the existing one does not cover it, is where fn-149.6 went [user, paraphrase]. The ordinary loop is not the problem: an edit costs about two minutes of tool time [inferred from the measurement].

The owner wants to put the current Models aside and iterate on a simplified one that is still realistic: product, protocol and realization levels, every DSL concept, some depth, but small enough that a change is validated by reading its IR diff rather than by a sealed comparison [user]. The DSL feedback (`model/SEMANTICS_FEEDBACK.md`) is the agenda. The parked Models are migrated at the end of each spec under the existing seal, so nothing is lost and the seal runs once per spec instead of once per change [inferred].

## Architecture & Data Models
<!-- scope: technical -->

**The subject.** [paraphrase] A cut-down standalone activity, `model/temporal/features/activity/mini/`, beside `standalone/`, by subtraction from it: `client.start` (inputs `scheduleToStart`, `startToClose`, `maxAttempts` bounded at two), `client.cancel`, one worker (`poll`, `respondCompleted`, `respondFailed`), the two timers, dispatch over the task queue foundation. No pause, no reset, no heartbeat, no describe, no by-ID client. Three levels: `product/` (what the client sees), `protocol/` (dispatch, attempts, deadlines over the queue), `realization/` (the kit that runs it). Target size about 100, 400 and 250 lines against today's 446, 2,809 and 1,338.

**The hard instances it must hold, one of each.** [inferred] These are where the real Models disagree with a naive DSL design; a DSL change proven on the mini Model transfers only if the mini Model contains them.

| Concept | Instance |
| --- | --- |
| Refinement | protocol refines product, with one `query verify` of a product Property through it |
| Shared input | `scheduleToStart` on both `start` and `schedule` (the queue's) |
| Shared status fact | a protocol phase (`backingOff`) that reports the same status as another (`scheduled`) |
| Facets | at least two (`Dispatching`, `Completion`), with one rule whose guard reads the other facet's phase |
| Capabilities | `Closable`, `Pollable`, `Retryable`, `Expirable`; one `except` waiver and one `overriding` with a reason |
| Composition and channel | the protocol over the task queue with one `sync`; the worker's answers on one bounded lossy channel |
| Derivations | one `FailureModel` with a `NegativeControl` (a dispatch that trusts a stale answer) |
| Progress | one `leadsTo` under `assume.fair` |
| Monitor, hole, assumption | one of each, the hole marking a path the mini Model leaves open |
| Realization | evidence for every fact, one `.live` witness with an `inconclusive` expectation and its reason, one `deadlines` use |
| Queries | one `find`, one `verify`, one witness, with bounds in `Bounds` |

**The build switch.** [inferred] The Makefile's `MODEL_SOURCES` lists directories, and `IrFile.discover("temporal")` finds only what was compiled. A variable selects the feature folders every Scala step sees:

```make
MODEL_FEATURES ?= activity nexus
MODEL_SOURCES := model/project.scala model/framework model/temporal/foundations model/temporal/actors \
                 model/temporal/capabilities $(foreach f,$(MODEL_FEATURES),model/temporal/features/$(f))
```

`MODEL_FEATURES=activity/mini` compiles, packages, lifts, checks and tests the mini Model alone. Parked features stay where they are, in git and on disk; they are not compiled, so a DSL change that breaks them does not block the loop.

**The loop.** [inferred] `make model-mini`: compile, lift `activity-mini.json`, the Go checks on that file, the syntax lint, then `git diff --stat model/ir/activity-mini.json`. Target: under 90 seconds. For the mini Model the reviewed IR diff is the equivalence proof; no ledger, no native comparison, no mutation controls. The lifter fixture suite and the Model test suites run at the end of a task, not per edit.

**The rules.** [inferred]
- During a DSL spec, the parked Models are not edited. Each DSL change lands in the framework and the mini Model with a one-line migration note on the task (what the parked Models must change, and whether IR identities move).
- A DSL spec is not closed until the parked Models are migrated per its notes and the existing seal passes on them: steps-only equivalence per spec, the complete seal once per batch as `MILESTONES.md` already schedules for fn-141.
- The mini Model is a proving ground, not a replacement. It is kept compiling and lifting under every DSL change; it is never parked.

## API Contracts
<!-- scope: technical -->

- `MODEL_FEATURES` selects feature folders for compile, jar, gate, lint and the Scala test suites; the default is every feature, so no existing command changes meaning. [inferred]
- `make model-mini` is `MODEL_FEATURES=activity/mini` plus compile, lift, Go checks, lint and the IR diff, in that order, stopping at the first failure. [inferred]
- The mini Model's IR file is `model/ir/activity-mini.json` with its lint acceptances beside it, discovered like any feature's. [inferred]
- The mini Model's realization lowers to Cases like any other and may be run live by the existing commands, outside the loop. [inferred]

## Edge Cases & Constraints
<!-- scope: technical -->

- The task queue foundation, the actors, the capabilities and `Bounds` are shared with the parked Models and are always compiled; a DSL change that touches them is validated on the mini Model and carries a migration note like any other. [inferred]
- The mini Model's IR identities are its own; nothing pins them, so a DSL change that renames identities is an IR diff to read, not a seal to run. [inferred]
- Generated Cases of the mini Model are regenerated with it and are not pinned by the functional tests or the canary until the owner decides to. [inferred]
- `IrFilesTest` and `MachineTablePins` compare the exports the compiled Models declare with the files on disk; with `MODEL_FEATURES` narrowed they see only the mini Model's, and that must be what the test expects under the switch. [inferred]
- The canonical Make route is blocked before execution by stale linked protobuf generation, which is owner-held; until it is restored, `make model-mini` and the per-spec seal run through the direct Scala route and record that. [paraphrase]

## Acceptance Criteria
<!-- scope: both -->

- **R1:** `MODEL_FEATURES` narrows every Scala step (compile, jar, gate, lint, Scala test suites) to the named feature folders, with every feature as the default; with `activity/mini` alone, the parked features are neither compiled nor lifted and remain untouched on disk. [inferred] Errors: a parked feature that is referenced from a compiled file fails the compile naming the reference.
- **R2:** `model/temporal/features/activity/mini/` holds product, protocol and realization levels within about 800 lines, lifts to `model/ir/activity-mini.json`, passes the Go checks and the lint, and every one of the hard instances in the table is present and named in the feature file's header comment. [user, inferred] Errors: a missing instance is a review finding on the task, not a gate failure.
- **R3:** The mini Model's realization lowers to Cases and at least one Case runs live to a verdict once, recorded on the task, so the realization level is real and not a stub. [inferred] Errors: a Case that cannot run names the kit gap in the task's done summary.
- **R4:** `make model-mini` runs compile, lift, Go checks, lint and the IR diff in under 90 seconds on the measured machine with warm caches, and stops at the first failure. [user, inferred] Errors: a run over 90 seconds records its stage times on the task.
- **R5:** `MILESTONES.md` states the rules: the DSL batch iterates on the mini Model with the parked Models untouched; each DSL task carries a migration note; a DSL spec closes only after the parked Models are migrated under the existing seal. The README's "Writing a Model" names the mini Model as the smallest complete example. [inferred] Errors: none (documentation).
- **R6:** The two IR-inventory tests (`IrFilesTest`, `MachineTablePins`) pass under both the default and the narrowed switch. [inferred] Errors: a test that reads a parked feature's file while narrowed is fixed to read the compiled set.

**Error cases (negative-cases discipline):** each criterion above states its error cases inline or records that it has none.

## Boundaries
<!-- scope: business -->

- No DSL change is made by this spec; it builds the ground the batch iterates on. [inferred]
- The parked Models are not edited, moved or deleted. [user]
- Restoring the canonical Make route (`make protoc`, owner-held linked-API generation) is recorded here as the first thing to do but is not a task of this spec. [paraphrase]
- Pinning the mini Model's Cases in the functional tests or the canary is out of scope. [inferred]

## Decision Context
<!-- scope: both -->

- A non-Temporal example (the guide's `job`) was weighed and rejected: the realization layer, the lowering and Testpilot are Temporal's by design, so a toy realization would prove nothing about the real one. The mini Model is a Temporal feature so its realization is real. [inferred]
- Shrinking the Models does not move the compile floor (20 to 40 seconds is the framework and the lifter); fn-141 is the lever for that. The mini Model's gain is in the gate, the suites and above all in replacing the seal with a readable diff. [paraphrase]
- Placement: first in the Scala batch, so fn-159 onward iterate on it. [user, "yes do it"]

## Early proof point

Task 2 (the build switch) and task 3 (the product level) together: if the narrowed build cannot lift one feature alone with the inventory tests passing, or `make model-mini` cannot get under 90 seconds with the product level alone, the approach is reconsidered before the protocol and realization levels are written.

## Execution

Order: build switch (1), product level (2), protocol level with composition, channel, derivations and claims (3), realization with witnesses (4), the loop target and the rules (5). The loop target can be used from task 2 on. Each task ends with the fixture suite and the Scala suites under the narrowed switch and once under the default switch, so the parked Models are known to still compile at this spec's close.

## Quick commands

```sh
MODEL_FEATURES=activity/mini mise exec -- scala-cli compile model/project.scala model/framework model/temporal/foundations model/temporal/actors model/temporal/capabilities model/temporal/features/activity/mini
make model-mini
make umpire-check-model                # default switch, at task end
mise exec -- scala-cli test model/irgen  # at task end
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Build switch | 1 | |
| R2 | Mini Model with the hard instances | 2, 3, 4 | |
| R3 | Real realization, one live run | 4 | |
| R4 | `make model-mini` under 90 s | 5 | |
| R5 | Rules in MILESTONES and README | 5 | |
| R6 | Inventory tests under both switches | 1 | |
