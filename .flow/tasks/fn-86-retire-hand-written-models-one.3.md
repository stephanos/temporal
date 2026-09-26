---
satisfies: [R3]
---
# fn-86-retire-hand-written-models-one.3 The typed Nexus example migrated; the superseded Nexus instructions removed

## Description
Re-author the typed Nexus example (two workflow-owned Nexus operations correlated by identity, a correlated capture Property, crossed-pairing mutations) as a Model file with the commands, field relations and the Nexus realization, regenerate its Case, and delete the hand-written module, its tests and its `register_case` line (R3). It was the last Producer emitting `StartNexusOperation`, `RespondNexus`, `NexusResponseKind` and the untyped `CompleteNexusOperation` result, so remove those from the protocol, the Lean declarations, the Driver and the Profile, and add their names to the retired-vocabulary gate in the same change.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/TwoOperations/Model.lean` (new; final path a task decision), `TwoOperations/Tests.lean` (crossed-pairing pins as realization or observation mutations), `model/Temporal/Case/Realization/Nexus.lean` (two operation instances; correlated capture binding), `model/Temporal/Feature/Nexus/Success/TypedNexus.lean`, `Success/Tests/TypedNexus.lean`, `Success/Nexus.md`, `Success/Integration.md` (deleted), `model/Temporal/Tool/Testpilot.lean` (`register_case` removed), `proto/internal/temporal/server/api/testpilot/v1/instruction.proto` (four shapes removed), `api/testpilot/v1/*`, `model/Testpilot/Authoring.lean`, `common/testing/testpilot/contract/profile.go` (opcodes), `common/testing/testpilot/temporal/worker/{interpreter,routing,driver}.go`, `common/testing/testpilot/internal/execution/dataflow.go`, `common/testing/testpilot/internal/execution/README.md:164-170`, `common/testing/testpilot/protocol_test.go`, `tools/umpire/internal/retiredvocabulary/check.go` + test, `tests/testcore/testpilot/testdata/typed-nexus-case.json` (replaced by the derived fixture), `tests/testcore/testpilot/typed_nexus_artifact_test.go`, `tests/testpilot_typed_nexus_case_test.go`, `tools/umpire/cmd/umpire-run/run_test.go` (fixture names)
**Touches:** [model/Temporal/Feature/Nexus/**, model/Temporal/Case/**, model/Temporal/Tool/Testpilot.lean, proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, common/testing/testpilot/**, tools/umpire/**, tests/testcore/testpilot/**, tests/testpilot_typed_nexus_case_test.go]

### Approach
- Two operation instances of the fn-85 `operation` entity in one workflow; the correlated capture Property becomes a field relation over the two operations' observation fields keyed by `scheduled_event_id`; the Known Gaps the typed Nexus Case records today (bounded-response window evaluated in the model only; a completion referencing another scheduled event leaves its rule pending) are re-recorded through `query` gaps or `unobservable` rows and named in the receipt.
- Crossed pairing (`completionSatisfies firstCase secondCase == some false`, `(withCompletion := false) == none`): keep both arms through a realization mutation helper outside production, as `Umpire/Command/Tests/Authoring.lean` does.
- Removal: follow fn-87's extension checklist in reverse for the four shapes; the retired tokens are compound (`StartNexusOperation`, `RespondNexus`, `NexusResponseKind`, the `invokeRPC`-style lowerCamel variants the gate derives); the gate scans open Flow task files, so this task's own text must not spell the retired names once they are added, and the two README lines are rewritten to the typed instructions.
- Sizes: `typed-nexus-case.json` was 2,025 lines before fn-87; record before and after.
- Adjusted 2026-09-19 after fn-85 .4 and .7 landed. **Two operations:** fn-85 .4 gives a Scenario
  `instances: N` with numbered actions (`awaitStart 2`), a Search over the product of N copies of
  the machine, and a Producer that reads the first instance back with every instance's actions as
  the Program's path -- every instance performs the same sequence, and the count is at most nine.
  The typed Nexus example's two operations map onto `instances: 2` on the fn-85 `operation` entity
  rather than onto two entities. **Claims:** `CaseProvenance.abstraction_claims` (the ninth field,
  fn-85 .7) carries a row for every class with an `examples:` line the Program performs, so the
  regenerated fixture gains claim rows if the Model writes examples; list them in the diff.
  **Identity:** the Case is produced by the Temporal `case … realizes <set>` block (fn-85 .7) under
  `temporal.case.<set>.<query>` and `<set>-<query>-case.json`, so `typed-nexus-case.json` and the
  `temporal.case.typed-nexus` id both move; `tools/umpire/cmd/umpire-run/run_test.go:175-188`
  reads `typed-nexus-case.json` by name. **Readers of the removed shapes** include
  `internal/execution/dataflow.go` (opcode dispatch and slot typing) besides the worker package, and
  the two README lines are `common/testing/testpilot/internal/execution/README.md:164-170`, not
  `:119,144-146`.
- Adjusted 2026-09-20 by fn-86 .1 after fn-85 closed. `Temporal.Case.Realization.Nexus` is the
  realization to extend (bindings keyed by classed member, timer bindings, the `await-scheduled`
  read and the handler task-queue role, fn-85 .10 to .12); `NexusHandlerReply` and the schedule
  command's attributes are the typed instructions (fn-85 .8), and the four shapes this task removes
  are still emitted only by `Success/TypedNexus.lean` (`Testpilot.Authoring`'s
  `Program.startNexusOperation`, `Program.respondNexus`, `Program.completeNexusOperation`).
  `register_case` has four lines in `model/Temporal/Tool/Testpilot.lean`; `typed-nexus-case.json`
  is read by name in `tests/testcore/testpilot/typed_nexus_{fixture,artifact_test}.go`,
  `derive_profile_test.go` and `tools/umpire/cmd/umpire-run/run_test.go`. The `HANDWRITTEN_INVENTORY.md`
  row lists every reader.

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/Success/TypedNexus.lean:700-830` — the two operations, `historyLimits`, the response and completion nodes
- `model/Temporal/Feature/Nexus/Success/Tests/TypedNexus.lean:238-300` — the crossed-pairing and derived-read-path pins
- `common/testing/testpilot/temporal/worker/interpreter.go:19-32,64-129,180-264` and `routing.go:189`, `driver.go:365-385`, `internal/execution/dataflow.go:26-48,92-130,169,211,608-620` — every reader of the removed shapes
- `model/Testpilot/Authoring.lean` (`Program.startNexusOperation`, `Program.respondNexus`, `Program.completeNexusOperation`) and `model/Temporal/Case/Realization/Nexus.lean` — the Lean emitters, as fn-85 .8 and .11 leave them
- `common/testing/testpilot/contract/profile.go:8-16` — the opcode table
- `common/testing/testpilot/README.md` — the extension checklist

**Optional:**
- `model/README.md:187-211` — the typed authoring walkthrough task .9 rewrites

### Key context
- fn-85 R10 deferred this removal here; fn-87 documented and renumbered the four messages without renaming them.

## Acceptance
- [x] the typed Nexus Model file replaces the hand-written module; its Case regenerates with the diff listed; artifact and live tests pass; both crossed-pairing arms are pinned
- [x] the four superseded shapes are absent from the protocol, Lean, Go and docs; their names are in the retired-vocabulary gate and `make umpire-check-retired-vocabulary` passes; `protocol_test.go` asserts they do not resolve
- [x] no `register_case` line remains for the typed examples; the inventory rows are marked migrated
- [x] `make umpire-check-regression` exit 0; `make lint-model` green


## Done summary

Done 2026-09-20; self-review. Commit 5a5158f.

### The pair Model

`Temporal/Feature/Nexus/Pair/Model.lean` (`Temporal.Feature.Nexus.Pair`; the path is this task's
decision: the example is two instances of one operation, not a second Nexus feature) imports the
caller Model and reuses its `operation` entity and action declarations. Its machine `pair` keeps
the operation without deadlines or retries -- `unscheduled`, `scheduled`, `started`, `succeeded`,
`failed`, `canceled`; every class of `schedule`, `handlerReply` and `complete` has a row because
admission requires one (`actionWithoutRow`), and a retryable handler error keeps the state and
records nothing -- with `evidence:` lines for the five recorded event kinds. `property completed`
holds of `complete (succeeded)`; `property completionReferencesSchedule … relates:
nexusOperationCompleted.scheduled_event_id = nexusOperationScheduled.event_id` is the hand-written
captured field Property as one line. `scenario twoAsync` runs `instances: 2` (both schedules, both
asynchronous replies, both completions); `limits six` (6/6/64); `query bothComplete` carries the two
interpretation Known Gaps the hand-written Case recorded (`completion-identity-is-unrecorded`,
`crossed-completion-is-inconclusive`, each against the Property it limits); `set nexusPairTests`
(caller and handler `driven`) and `case nexusPairCases realizes nexusPairTests as
(Temporal.Case.Realization.asyncNexus "umpire.case.service" "complete")`. The Case is
`temporal.case.nexusPairTests.bothComplete`, fixture `nexusPairTests-bothComplete-case.json`.

### Three Umpire changes it needed

**Captured operands.** A `relates:` operand `<kind>.<path>` is the step's own event only where the
`when:` action's rows record the fact the kind confirms; otherwise it is an earlier step's event,
read at the state the step starts from (`FieldOperand.root := .priorState`, `member` := the first
state a row of the action starts from, in table order) and captured. The `property` command reads
the rows off the machine's table while the file compiles (`DeclaredModel.recordedFacts`,
`sourceStates`, `rowOutcomes`, evaluated the way the stuck-state check is), which also fixed the
result operand: `<action>.result.<path>` is now read under the step's outcome (`.outcome`, the
first outcome the rows produce) rather than under the action, which `Property.check` would have
refused. `FieldOperand` carries the kind it observes (`observed`). The Producer lowers a relation
with a captured operand through `Umpire.Case.Projection.lower`'s `.crossEvent` policy: the selector
is the field the realization's evidence source names for the captured kind
(`EvidenceSource.selector : Option FieldOperand`, resolved by `Temporal.Case.FieldPath.resolve` at
definition time -- the scheduled event's `operation`), read under the operand's reference against
the literal the binding of the action that kind confirms assigns under the selector's spelling
(`literals`, now per placement); the rule's states are named after the two kinds. Rejections:
`relation.capture-unrecorded`, `relation.capture-unselectable`, `relation.capture-unbound`,
`relation.capture-literal-unassigned`, `relation.capture-shape`. `Temporal.Case.FieldPath` reads a
field of the history event itself (`event_id`) off the event before the arm is selected.

**Placement.** `Umpire.Case.Producer.Placement` is where a node lands: the Case, the instance
performing it (from one) and the count. `ActionBinding.node` and `literals`, `EntrypointItem.fixed`
and `whenOnPath` and `EntrypointPlan.activate` take it instead of the identity; the Program path
(`Input.program`, `Realizable.program`, `checkInstances`) carries each action's instance; an
instance's node ids carry `-<n>` on a Case over several instances and nothing on one over one, so
every existing fixture is byte-identical; `whenOnPath` emits once per instance that performs a
keyed class; `EntrypointPlan.perInstance` emits an entrypoint once per instance with that
instance's actions; `ProgramPlan.instanceSlots` declares slots per instance; a relation lowers to
one rule per instance (`relation-<n>`). The Nexus realization addresses `<operation>-<n>` per
instance (`Nexus.operationOf`), with a slot, an authority wait, an await and a handler entrypoint
per instance, and states the operation name as the schedule binding's literal and the scheduled
source's selector. The pair Case's Program: controller `start-workflow`, `await-scheduled`,
`await-completion-authority-1/2`, `complete-nexus-operation-1/2`, `await-close`, `history`;
workflow `start-nexus-operation-1/2`, `await-nexus-operation-1/2`, `finish-workflow`; `handler-1`
and `handler-2` answering `complete-1` and `complete-2`; slots `completion-authority-1/2`. Its
Contract: `relation-1` and `relation-2`, each `capture-nexusOperationScheduled-…` (the scheduled
event whose `operation` is the instance's) then `match-nexusOperationCompleted-…` (the completion's
`scheduled_event_id` against the retained `event_id`), beside the correlated capability.

**Search prefix pruning.** A six-step exact sequence over two instances of a 6-state, 17-class
machine was not found within 131,072 traces: the Search enumerated every trace of every depth and
filtered at the end. `CheckedScenario.admitsPrefix` says whether a prefix can still be admitted
(allowed and forbidden actions, occurrence maxima, an exact sequence's order), and `Search`'s
candidate pull does not extend a prefix it rejects. The admitted traces and their order are
unchanged; the explored counts are not, so the two Operations compatibility artifacts that record
them (`Temporal/Feature/Nexus/Fixtures/OperationsCancellationArtifact.json`,
`OperationsSuccessfulCompletionArtifact.json`) were regenerated by `make umpire-gen-goldens`
(traces 4→3 and transitions 2→1 in each, and their checksums; the async-start artifact is
unchanged), and one `#guard_msgs` in `Success/Tests.lean` now reads "explored 2 traces". The pair
Query runs under `search: 64`.

### Removal

The four untyped Nexus instruction shapes (the start, the completion whose result was an
expression, the handler response and the enum naming its kind) are gone from
`instruction.proto` (arms 3, 4 and 7 removed outright; their numbers are reused by the arms that followed, and nothing is reserved, since the protocol is internal and every fixture regenerates), `api/testpilot/v1`,
`Testpilot.Authoring` and its tests, the Go contract (`Opcode` is now the arm's position in the
oneof; `MaxOpcode` stays `ReadEvidence`), admission, the scheduler, the worker interpreter and
driver, the profile tables and both READMEs; the Go unit tests that built them build the typed
successors, and `protocol_test.go` asserts the four descriptor names no longer resolve. The
retired-vocabulary gate holds the response and kind names bare, and the start and completion names
by their protocol-qualified Go spellings (the oneof arm type and its accessor), because both are
also HistoryService method names the generated `Temporal.API` spells; `check_test.go` pins the
four and the live look-alikes. `make proto` fails at the api-linter on a pre-existing `case.proto`
field name (`class_name`, `core::0122::name-suffix`) unrelated to this task; the generated code was
produced with `make protoc proto-codegen`, and `instruction.proto` lints clean. Deleted:
`Success/TypedNexus.lean`, `Success/Tests/TypedNexus.lean`, `Success/Nexus.md`,
`Success/Integration.md`, `typed-nexus-case.json`, `typed_nexus_{fixture,artifact_test}.go`,
`tests/testpilot_typed_nexus_case_test.go`; the `register_case` line; the inventory rows are
marked migrated. The Go readers of the pair Case are `nexus_pair_{fixture,artifact_test}.go` and
`tests/testpilot_nexus_pair_case_test.go`.

### Pins

`Pair/Tests.lean`: the machine; both operands as resolved (roots, members, kinds, steps, presence,
type, schema); an arm-field rejection; the instanced program path; the produced Case's instruction
ids per entrypoint, slots, handler operations, the two capture rules and their literals, the two
Known Gaps; and the field requirement evaluated over real admitted payloads with references from
the vocabulary: `completionSatisfies first first`/`second second` `some true`, crossed `some
false` both ways, missing completion `none`. Go: `TestNexusPairCaseCarriesOneCaptureRulePerInstance`
prepares the fixture offline under a hand-written Profile with two handler reservations;
`TestTestpilotNexusPairCase` runs it live twice.

### Fixture diff

`typed-nexus-case.json` deleted; `nexusPairTests-bothComplete-case.json` added; the seven
`nexusCallerTests-*` and `workflowStartTests-started` fixtures unchanged.

### Gates

`lake build` green (633 targets); `make umpire-gen-goldens` (the two artifacts above) then
`make umpire-check-goldens` exit 0; conformance fixtures regenerated (the diff above); inventory,
retired vocabulary, protocol, authoring and regression-view checks exit 0; `LEAN_NUM_THREADS=1 make
lint-model` at the `.1` baseline (41 warnings, none new); `GOLANGCI_LINT_BASE_REV=dc80ea7
make lint-code-fast` clean; `go test` over `common/testing/testpilot`, `tests/testcore/testpilot`
and `tools/umpire` ok; `TestTestpilotNexusPairCase` passes live; `make umpire-check-regression`
exit 0 with 29 passing live identities (29 before; the typed-nexus identity is
replaced by the pair one).

## Evidence
- Commits: 5a5158f
- Tests: `lake build`; `make umpire-gen-goldens && make umpire-check-goldens`; `make umpire-gen-case-runtime-conformance && make umpire-check-case-runtime-conformance`; `make umpire-gen-inventory && make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-regression-views`; `LEAN_NUM_THREADS=1 make lint-model`; `make lint-code-fast`; `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...`; `go test -count=1 -tags test_dep,integration ./tests -run TestTestpilotNexusPairCase`; `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression`
- PRs:
