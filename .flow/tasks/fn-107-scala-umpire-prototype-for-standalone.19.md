---
satisfies: [R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.19 Author the activity realization and the evidence declarations conformance needs

## Description
**Touches:** [model/scalav2/scala/umpire/realize/**, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/scala/temporal/nexuscaller/Realization.scala, model/scalav2/lifter/**, model/scalav2/ir/**, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/goir/load.go, model/scalav2/goir/testpilot/**, model/scalav2/goir/conformance/**, model/scalav2/goir/*_test.go, model/scalav2/SEMANTICS.md, model/scalav2/README.md, model/scalav2/specimens/**, model/scalav2/run.sh]

Give the Scala realization vocabulary what tasks 6, 7 and 13 found missing, author the activity realization, and lower it, so task 9 has an activity Case to run and the conformance adapter can reach conclusions. No Testpilot runtime or protocol change here: the protocol additions are task 13's and are consumed as they are.

**Size:** L

### Approach
- **Activity realization.** Author a realization for the standalone activity Model in Scala beside the Model (start, completion, retry through a failing first attempt, pause/unpause as far as the public API allows), with its worker script. A failing attempt lowers to the `activity_attempt_failure` instruction arm, never to a `Finish` whose result is a `Failure`. The lowered start assigns `namespace` and `task_queue.name` from the worker and queue role bindings and sets `activity_id` and `activity_type.name` (task 13 handover). Lift the "activity activation" gap in `goir/testpilot/lower.go`; what Testpilot still cannot run (hold-delivery, durable-commit observation: task 10; authored monitors as Contract rules) stays a located `unsupported` entry naming its owner.
- **Evidence roles.** Let a retained evidence field declare its role (operation, attempt, delivery), lift it, admit it, and carry it into the lowered Case; `goir/conformance/evidence.go` then reads task 13's typed `InstructionOutcome.activity_attempt { activity_run_id, sdk_attempt, delivery_id, response }` and the declared roles, at the places task 7's handover lists, and stops refusing a Case that retains a field.
- **Exhaustive sources.** Add the declaration task 7 specified: an evidence kind may be declared exhaustive and a read command may close it (`Realize.scala`, `ir.proto`, admission refusing `exhaustive` on a kind no command closes, the lowered Case carrying which kinds are exhaustive and which instruction is each one's closing read). The conformance adapter then infers absence only for an exhaustive kind whose closing read succeeded with dense source ordinals on a Run that closed complete; without the declaration it still infers nothing.
- **Discriminating Nexus evidence.** Apply the table in task 7's handover to the Nexus caller realization so the seven Properties can conclude on their witness Runs: the started event carried with its history source exhaustive, a failure-kind role or separate kinds, completion-result evidence, distinct timeout kinds, a typed attempt count. Where a Property still cannot conclude, say why and leave it inconclusive; do not narrow a claim to make it pass.
- Derive every expectation from the specimens and the spec. Byte parity with the old Case fixtures is not required; identical inputs still give identical bytes.

### Investigation targets
**Required:** `.flow/tmp/fn-107/handover/task6-summary.md`, `task7-summary.md` (exhaustive declaration spec; Nexus evidence table; where typed fields plug in), `task13-summary.md` (protocol diff; what task 9 must lower); `model/scalav2/scala/umpire/realize/Realize.scala`; `model/scalav2/goir/testpilot/{lower,realization,descriptor}.go`; `model/scalav2/goir/conformance/{evidence,candidates,conclude}.go`; `model/scalav2/specimens/{activity,nexus}.md`; `proto/internal/temporal/server/api/testpilot/v1/{instruction,run}.proto`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; `GOFLAGS=-tags=test_dep make umpire-gen-scala`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `make lint-scala`; `GOFLAGS=-tags=test_dep mise exec -- make protoc` when `ir.proto` changes; scoped `make lint-code` over `./model/scalav2/goir/...` with `GOLANGCI_LINT_FIX=false`. Nothing that calls `lake`.

## Acceptance
- [ ] The standalone activity Model has a Scala realization; its completion and retry Queries lower to Cases that ordinary `testpilot.Prepare` admits under a derived Profile, with the failing attempt as an `activity_attempt_failure` instruction and the start carrying namespace, task queue, activity id and type from the declared bindings. Identical inputs give identical bytes.
- [ ] A retained evidence field declares its role in Scala; a crossed attempt or delivery cannot explain a step in the conformance adapter, and a Case that retains fields is assessed, not refused. Task 13's typed attempt identity is read as typed data.
- [ ] An evidence kind can be declared exhaustive with a closing read; absence is inferred only under that declaration with a successful closing read, and a stale-design violation is shown from commit evidence on such a Run, live and replayed alike. Without the declaration nothing is inferred.
- [ ] Each of the seven Nexus caller Properties either concludes on its witness Run with the added evidence or is listed as still inconclusive with the reason; no claim was narrowed to pass.
- [ ] What Testpilot cannot run yet is a located `unsupported` entry naming its owning task; nothing declared is silently dropped (inventory tests still close).
- [ ] No Testpilot protocol file changes; legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
