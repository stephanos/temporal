---
satisfies: [R1]
---
# fn-123-declare-faults-as-the-environments.1 Fault kinds and modelOnly declared on the fault actor's actions, lifted to an IR Fault record

## Description
Fault declarations (R1): a kind (`crash`, `responseLoss`, `storageLoss`) and an optional `modelOnly(because = …)` on each action of the `fault` actor, an IR `Fault` record on `Action`, the Go reader's admission of it, and the conversion of every `fault` action the Models and lifter fixtures declare. It goes first because every later task reads the kind. It also settles the spellings for the whole spec and records them in the spec's API Contracts.

**Size:** M
**Files:** `model/umpire/Action.scala` (fault metadata on `ActionDecl`), `model/irgen/Declarations.scala` (lift the record), `model/irgen/Markers.scala` (fault test keyed on the record), `proto/internal/temporal/server/api/umpire/v1/ir.proto` + regenerated Go, `tools/umpire/ir/validate.go`, `model/temporal/shared/taskqueue/TaskQueue.scala` (the `fault` actor), `model/irgen/testdata/lifts/{Declarations,Captured}.scala` + `expected/*.json`, refusal fixtures under `model/irgen/testdata/`, `model/irgen/test/Fixtures.test.scala`, `.flow/specs/fn-123-declare-faults-as-the-environments.md` (API Contracts)
**Touches:** [model/umpire/Action.scala, model/irgen/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/**, tools/umpire/ir/**, model/temporal/shared/taskqueue/TaskQueue.scala, model/ir/**, model/cases/**, .flow/specs/fn-123-declare-faults-as-the-environments.md]
**Order:** Starts after the DSL batch and fn-140 close (spec Edge Cases, Order). The `path:line` targets below were verified on 2026-10-06 against the tree before the batch, so re-read each file first. Before starting, check whether fn-141 has started; if it has, add the record in its exporter instead of the tree lifter and tell the owner.

### Approach
- Settle the spelling first, within fn-112's operator rules. Keep the Model's `object fault extends Actor` and its three actions with their Definition IDs (`temporal.shared.taskqueue.fault.{crash,ackLoss,storageLoss}`), and attach the kind and reason to each action declaration (spec Open Questions, "The `fault` spelling"). Check the name against fn-133.3's lower-case `fault(…)` instruction; if they collide in a Realization file, record how the author disambiguates. Rewrite the spec's API Contracts sketch in the object-and-`rules` style the batch leaves (`when` blocks).
- Lifter: extend the action lifting at `Declarations.scala:22-70` the way channel deliver/lose actions are handled (:47-50). Refuse at the Scala position: an action of the `fault` actor with no kind, `modelOnly` with an empty reason, and a second declaration of one fault.
- Markers: `faultActions` (`Markers.scala:162-164`) keys on `actor == "fault"`. Key it on the `Fault` record and keep the FailureModel and NegativeControl refusals at `:140-158` equivalent. Add one fixture that shows each refusal still fires.
- Proto: add the `Fault` message and an optional `Action` field with the next free number after `loses = 14` (`ir.proto:322-343`). Emit it only when present, so actions without faults keep their bytes and fingerprints (memory `default-empty-extensions-must-preserve-2026-09-05`). Run the descriptor structure check (memory `validate-protobuf-descriptor-structure-2026-09-05`).
- Go: `validator.action` (`validate.go:207`) admits the record. An unknown kind or a reason on a non-model-only fault is an admission error.
- Regenerate with `make umpire-gen-model`, then classify the `model/ir`, `model/cases` and lift-golden diff against the spec's recorded deltas (new fault metadata, fixture goldens, positions). Any other changed line stops the task.

### Investigation targets
**Required** (read before coding):
- `model/umpire/Action.scala:9-57` - actors, `ActionDecl`, `internal`/`delivers`/`loses`
- `model/irgen/Declarations.scala:22-70` - action lifting and channel actions
- `model/irgen/Markers.scala:130-165` - fault-keyed marker checks
- `model/temporal/shared/taskqueue/TaskQueue.scala:104-115` - the `fault` actor and `storageLossAssumed`
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:322-343` - `Action`

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
