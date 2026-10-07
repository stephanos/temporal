---
satisfies: [R17]
---
# fn-133-lean-typed-realizations.8 Derive per-class carrier schemas from typed realizations

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
Replace the unpaired action-level .schema[T] lists with carrier mappings derived from typed realization bindings. The authoritative association is perform(actionClass -> instruction): for example control(pause) -> pauseActivity -> METHOD_PAUSE_ACTIVITY_EXECUTION -> PauseActivityExecutionRequest. Derive metadata per realization and concrete action class, preserving the binding's pattern semantics and supporting withFields variants, derived realizations, multiple carriers, protobuf command payloads, and typed worker/handler messages. Actions without a realization have no inferred carrier: keep model declarations independent and let R6 report missing realizations where required. Retire the duplicate .schema authoring form and teach the mapping in README. Model and Testpilot behavior remains unchanged; record deliberate Umpire metadata changes.

## Acceptance
- [ ] The four standalone activity control classes expose separate derived mappings to PauseActivityExecutionRequest, UnpauseActivityExecutionRequest, RequestCancelActivityExecutionRequest, and TerminateActivityExecutionRequest; order in an old schema list never determines association.
- [ ] RPC request carriers come from generated method descriptors. Non-RPC typed command payloads and worker/handler protobuf carriers are covered by the same mechanism; internal/timer/fault steps do not receive invented carrier schemas.
- [ ] Mappings are scoped to realization and concrete action class. Partial/exact patterns, combined inputs, withFields variants, multiple instructions, and derived binding additions/replacements retain their actual association. Auxiliary onPath evidence/read calls do not become the performed action's carrier.
- [ ] Unrealized classes remain explicitly unmapped, and R6 distinguishes absent realization coverage from an internal action that needs no transport. No additional schema declaration is required to author an abstract Model.
- [ ] Remove live .schema[T] declarations and retire their DSL/lifter form with a clear migration diagnostic. Update affected IR metadata, fixtures, consumers, and README; preserve any aggregate compatibility view only if derived from the per-class mapping.
- [ ] Tests pin the concrete class-to-method-to-message mapping, non-RPC carriers, partial/combined class coverage, derived overrides, auxiliary calls, and unmapped cases. Negative fixtures cover ambiguous or unsupported derivation without fabricating a semantic correctness guarantee from message names.
- [ ] Projection evidence accounts for intentional metadata and source-position deltas and proves unchanged machine tables, Query answers, existing Case execution, and verdicts. Run focused checks during implementation and the required full gates once at the fn-133 batch boundary.

## Done summary
What each class carries is now derived per realization and concrete class from the class's binding (Go: `tools/umpire/realization.Carriers`). The action-level `.schema[T]` form is retired with a migration diagnostic, no Model or fixture names a schema any more, and the README teaches the derived mapping.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: IMPLEMENTER claude-opus-5-5 at high

### What changed
- **`tools/umpire/realization/carriers.go`:** `Carriers(r, *protoregistry.Files) ([]Mapping, error)` returns one `Mapping{Class, Carriers}` per class the realization performs, in binding order. A Carrier is `{Kind, Method, Message}`. What each binding carries:

  | binding | carrier kind | message |
  |---|---|---|
  | an rpc (perform) | rpc | the method's request, resolved from the generated method descriptor (e.g. `control(pause)` → PauseActivityExecutionRequest) |
  | a workflow command | workflow-command | the `*_command_attributes` message it sets (ScheduleNexusOperationCommandAttributes) |
  | a Nexus reply | nexus-reply | the reply message (StartOperationResponse or HandlerError) |
  | a Nexus completion | nexus-completion | the result message |
  | `finish` / `attemptFailure` / `attemptCanceled` in an activity script | activity-answer | RespondActivityTask{Completed,Failed,Canceled}Request |
  | an activity script's `starts` | activity-delivery | PollActivityTaskQueueResponse |
  | fault, hold, release, await, poll | none | — |

  - A class no binding performs is absent, so it stays unmapped.
  - Carriers are scoped to the realization and the class: the derived ForgedControl's inspect maps only in that realization.
  - Refused: two different carriers of one class in one script (ambiguous), and a method the registry does not hold.
  - Two different scripts may both carry a class (multiple carriers).
  - No carrier is inferred from a message name.
- **Tests** (`carriers_test.go`, over model/ir; they hold after the regeneration):
  - the activity's four controls map to Pause/Unpause/RequestCancel/Terminate ActivityExecutionRequest;
  - the bound start classes (partial patterns, exact) carry StartActivityExecutionRequest, and an unbound start class is unmapped;
  - worker answers, the delivery, the fault-performed stop (no carrier) and the onPath await (no carrier);
  - the Nexus schedule (unset and partial classes), replies and completion, and the derived ForgedControl's inspect versus asyncNexus;
  - negative cases: ambiguous and unknown-method.
- **Retired `.schema[T]`:**
  - `Action.schema` is an `inline def` raising `compiletime.error` with the migration text. `ActionDecl.schemas` is removed.
  - The lifter's `schema` case fails with the same text.
  - The `.schema[…]` lines are removed from Activity.scala, activity Standalone.scala, Nexus.scala, nexus standalone Standalone.scala and Workflow.scala (outside the realization files, a needed exception), together with their now-unused imports. The Nexus.scala comment that mentioned the schema is reworded.
  - Fixtures:
    - lifts/Typed.scala and binding/Binding.scala no longer name schemas;
    - typedInvalid line 22 becomes a comment line, so no other line moves, and its expected `Invalid.scala:22:63` is dropped;
    - the new `testdata/retiredSchema` fixture is refused at `Retired.scala:9:3` with the diagnostic;
    - the Fixtures tests that asserted schemas are updated;
    - lifts/expected hints.json and hintsRefused.json are regenerated (`UMPIRE_LIFTER_UPDATE=1 scala-cli test model/irgen`). Their diff is only the removed `schemas` lists and line numbers.
- **README:** the action vocabulary drops `schema`, and the realization section describes the derived carriers and the retirement.

### Decisions (owner unavailable)
- **No new IR field.** Every carrier is derivable from the Performances already in the IR (rpc method, the protos the commands carry, the activation kind). The derivation is therefore a Go function over the IR rather than duplicated metadata, which also avoids a protoc regeneration in the batch.
  - The proto field `Action.schemas` stays in ir.proto but is now always empty. No aggregate view is kept: no Go code read it (grep of tools/common/tests).
  - fn-139.8 notes that a rejection table may need a new IR field. It can either add one, or carry its table beside these Go-derived carriers.
- **Owner of a carrier mapping: the realization.** Coverage stays with umpire-lint's `uncovered-class` (fn-133.4). It counts only performed non-system actions, so an internal or system action that needs no transport never counts as uncovered, and a class without a realization shows there.

### Declared IR delta (batch regeneration)
- Every model/ir file: each Action's `"schemas": [...]` is removed. Checked: actions are equal with `schemas` and positions stripped in all seven files (scratch lift).
- Positions move in the five feature files where the schema lines and imports were removed.
- No Model table, Query answer or Case changes. Testpilot is untouched.
- Lift fixtures: hints.json and hintsRefused.json (regenerated as above).

### Tests
- `go test -count=1 -tags test_dep ./tools/umpire/realization/ ./tools/umpire/lint/` passes. gofmt and go vet are clean.
- `UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test model/irgen`: 100 passed. This rewrote only hints*.json.
- The model unit tests pass. `--check-syntax` and `--check-comments`: clean. scalafmt: clean.
- Not run: golangci-lint. There is no darwin build in this clone, so the batch's lint-code-fast covers it.

### For the batch regeneration / later tasks
- After the regeneration, rerun `go test ./tools/umpire/realization` and `./tools/umpire/lint` (with `-update-coverage` for the lint golden, as noted in fn-133.4).
- fn-133.7 (close): line counts after .8 are activity system/Realization.scala 303, nexus workflow 305, nexus standalone 83. Baseline was 333 / 514 / 101.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 55b5cc6070
- Tests: go test -count=1 -tags test_dep ./tools/umpire/realization/ ./tools/umpire/lint/, UMPIRE_LIFTER_UPDATE=1 mise exec -- scala-cli test model/irgen, mise exec -- scala-cli test model/project.scala model/umpire model/temporal, scala-cli run model/check -- --check-syntax, scala-cli run model/check -- --check-comments, scratch lift --ir: actions equal without schemas in every file
- PRs: