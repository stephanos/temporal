---
satisfies: [R1]
---
# fn-123-declare-faults-as-the-environments.1 Fault kinds and modelOnly declared on the fault actor's actions, lifted to an IR Fault record

## Description
Fault declarations (R1): a kind (`crash`, `responseLoss`, `storageLoss`) and an optional `modelOnly(because = …)` on each action of the `fault` actor, an IR `Fault` record on `Action`, the Go reader's admission of it, and the conversion of every `fault` action the Models and lifter fixtures declare. It goes first because every later task reads the kind. It also settles the spellings for the whole spec and records them in the spec's API Contracts.

**Size:** M
**Files:** `model/framework/Action.scala` (fault metadata on `ActionDecl`), `model/irgen/Declarations.scala` (lift the record), `model/irgen/Markers.scala` (fault test keyed on the record), `proto/internal/temporal/server/api/umpire/v1/machine.proto` + regenerated Go, `tools/umpire/ir/validate.go`, `model/temporal/foundations/taskqueue/TaskQueue.scala` (the `fault` actor), `model/irgen/testdata/lifts/{Declarations,Captured}.scala` + `expected/*.json`, refusal fixtures under `model/irgen/testdata/`, `model/irgen/test/Fixtures.test.scala`, `.flow/specs/fn-123-declare-faults-as-the-environments.md` (API Contracts)
**Touches:** [model/framework/Action.scala, model/irgen/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/**, tools/umpire/ir/**, model/temporal/foundations/taskqueue/TaskQueue.scala, model/ir/**, model/cases/**, .flow/specs/fn-123-declare-faults-as-the-environments.md]
**Order:** Starts after fn-156's Flow closure, fn-140.6's independent witness seal and fn-149.5's final source/docs/layout grouping seal are committed in the integrated local branch (spec Edge Cases, Order). Freeze fn-149.5's final grouped scratch IR, Cases and lift goldens with its authorized mapping composed through fn-140 and fn-155's actually committed identity mapping; preserve fn-140.6's separate witness oracle and do not infer pending effect mappings. Fn-123 shares the selected authoring batch with fn-140 and fn-149; neither earlier whole-spec/task closure is an entry prerequisite, before the format batch fn-146 through fn-148 and lifter fn-141. The fn-145 schema split is already integrated. The `path:line` targets below were verified on 2026-10-06 against the earlier tree; re-read each file first.

### Approach
- Settle the spelling first, within fn-112's operator rules. Keep the Model's `object fault extends Actor` and its three actions with their Definition IDs (`temporal.foundations.taskqueue.fault.{crash,ackLoss,storageLoss}`), and attach the kind and reason to each action declaration (spec Open Questions, "The `fault` spelling"). Check the name against fn-133.3's lower-case `fault(…)` instruction; if they collide in a Realization file, record how the author disambiguates. Rewrite the spec's API Contracts sketch in the object-and-`rules` style the batch leaves (`when` blocks).
- Lifter: extend the action lifting at `Declarations.scala:22-70` the way channel deliver/lose actions are handled (:47-50). Refuse at the Scala position: an action of the `fault` actor with no kind, `modelOnly` with an empty reason, and a second declaration of one fault.
- Markers: `faultActions` (`Markers.scala:162-164`) keys on `actor == "fault"`. Key it on the `Fault` record and keep the FailureModel and NegativeControl refusals at `:140-158` equivalent. Add one fixture that shows each refusal still fires.
- Proto: add the planned fault metadata and an optional `Action` field with the next free number after `loses = 14` in `machine.proto`. The existing package-level `Fault` in `script.proto:189` owns realization worker/response-loss instructions. Do not declare a second package-level `Fault`, repurpose that message, or add a machine-to-script schema dependency. Settle the new metadata type's spelling within this task's existing API-contract spelling work. Emit it only when present, so actions without faults keep their bytes and fingerprints (memory `default-empty-extensions-must-preserve-2026-09-05`). Run the descriptor structure check (memory `validate-protobuf-descriptor-structure-2026-09-05`).
- Go: `validator.action` (`validate.go:207`) admits the record. An unknown kind or a reason on a non-model-only fault is an admission error.
- At the shared batch regeneration owned by fn-123.8, run `make umpire-gen-model`, then classify the `model/ir`, `model/cases` and lift-golden diff against the final fn-149.5 grouped baseline and the spec's recorded deltas (new fault metadata, fixture goldens, positions). Any other changed line stops the task.

### Investigation targets
**Required** (read before coding):
- `model/framework/Action.scala:9-57` - actors, `ActionDecl`, `internal`/`delivers`/`loses`
- `model/irgen/Declarations.scala:22-70` - action lifting and channel actions
- `model/irgen/Markers.scala:130-165` - fault-keyed marker checks
- `model/temporal/foundations/taskqueue/TaskQueue.scala:104-115` - the `fault` actor and `storageLossAssumed`
- `proto/internal/temporal/server/api/umpire/v1/machine.proto:18-39` - `Action`

**Optional** (reference as needed):
- `tools/umpire/ir/validate.go:207-260` - action admission
- `model/irgen/testdata/lifts/Declarations.scala:23`, `Captured.scala:48-50` - fixture crashes
- `model/irgen/test/Fixtures.test.scala:1008` - the `"faults.crash" -> "fault"` mapping

### Key context
- Memory `channel-catalogs-and-visible-results-2026-09-30`: an unrecognized expression must be refused at lifting, never defaulted. Apply it to an unknown fault kind.
- The Nexus `network.fault` actions and `worker.stop`/`resume` are out of scope (spec Boundaries). They are other actors, so the R1 refusal does not reach them.

## Acceptance
- [ ] The spec's API Contracts records the settled spellings for fault kinds, `modelOnly`, durability forms, `crashes(…)`, `budgetedBy` and `choosing`, in the current object-and-`rules` style.
- [ ] `crash`, `ackLoss` and `storageLoss` in the shared task queue, and the crashes in `Declarations.scala` and `Captured.scala`, carry a kind. Their action Definition IDs and the composition keys `queue_crash` and `queue_ackLoss` are unchanged.
- [ ] One refusal fixture each for an undeclared `fault` action, `modelOnly` with an empty reason and a second declaration. Each message names the action and its position.
- [ ] The FailureModel and NegativeControl refusals key on the `Fault` record and still fire on their fixtures.
- [ ] The Go reader admits the record and refuses an unknown kind, with a test for each.
- [ ] The regenerated diff holds only the recorded deltas, listed in the done summary.
- [ ] `make umpire-check-model`, the irgen munit tests and `go test -tags test_dep ./tools/umpire/ir/...` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
