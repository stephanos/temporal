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
- [ ] the typed Nexus Model file replaces the hand-written module; its Case regenerates with the diff listed; artifact and live tests pass; both crossed-pairing arms are pinned
- [ ] the four superseded shapes are absent from the protocol, Lean, Go and docs; their names are in the retired-vocabulary gate and `make umpire-check-retired-vocabulary` passes; `protocol_test.go` asserts they do not resolve
- [ ] no `register_case` line remains for the typed examples; the inventory rows are marked migrated
- [ ] `make umpire-check-regression` exit 0; `make lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
