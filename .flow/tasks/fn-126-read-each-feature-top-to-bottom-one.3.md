---
satisfies: [R5, R6, R14]
---
# fn-126-read-each-feature-top-to-bottom-one.3 Group actions by actor in transparent section objects

## Description
Group every feature's actions by actor (R14) and keep every Definition ID through transparent section objects (R6). This is the first lifter change of the spec.

**Cross-spec entry gate:**
- Task 2 is done.
- Never alongside fn-124.8.
- Before fn-124.7.

**Size:** M
**Files:**
- `model/umpire` (the markers `Section` and `Actor`; `Party` opened so an object can be one);
- `model/irgen/Context.scala` (`definitionId`, `pinOf`) and the party naming, with fixtures;
- every feature file's signature, its Scenarios and realizations (call sites);
- `tools/umpire/internal/golden/config.json`.

**Touches:** [model/umpire/**, model/irgen/**, model/temporal/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**]

### Approach
- `Section` is a marker trait. `Actor` is a `Party` and a `Section`, named by its object's name. Today's `val caller = Party()` becomes `object caller extends Actor`, and its members are the actions it takes. The party name is unchanged.
- Lifter rule: when computing a Definition ID, a section object is skipped. A member takes the ID it would take as a direct member of the section's enclosing owner, or of the file's package object at the top level, so the file's pin applies. Refuse, at their line:
  - a section inside a section;
  - a section anywhere other than the top level of a Model file or directly in a machine, composition or derived-machine object (task 4 adds the last three);
  - two members that would share an ID (`idTakenBy`, as today).
- Groups:
  - parties `caller`, `handler`, `network` and the shared `worker`;
  - sections `timers`, `deadline`, `history` (internal steps), `queue`, `faults`;
  - the standalone activity's `object worker extends Section`, whose actions are taken by the shared party, imported as `process` (R14).

  Actions keep their val names (`worker.attemptStart`), so every ID stays. Task 6 renames them.
- Move the deadline inputs out of `object Inputs`. Keep only inputs named like an action of their actor object.
- Rewrite call sites in Scenarios, Properties, capabilities, compositions and realizations. Record function-symbol and position deltas.

### Investigation targets
**Required:**
- `model/irgen/Context.scala:225-335`
- `model/umpire/Action.scala` (Party, action naming), `model/irgen/Declarations.scala:20-40` (action lifting)
- `.plans/DSL_SIMPLIFICATION.md` section 4b (the per-feature table)
**Optional:**
- `model/temporal/shared/worker/Model.scala:44-60` (why the prefix stayed)

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower
```

### Execution constraints
- Zero Definition-ID, party-name, action-class or Case-content change beyond positions and function symbols.

## Acceptance
- [ ] Every action of every feature is declared in an actor or section object, and every call site shows its actor (`caller.start()`, `worker.attemptStart`).
- [ ] `Section` and `Actor` exist in `model/umpire`. Section objects are transparent to Definition IDs, with a lifting fixture and refusal fixtures for a nested section, a misplaced section and a duplicate ID.
- [ ] Definition IDs, party names, action classes, tables, answers and Case bytes are unchanged apart from recorded positions and function symbols.
- [ ] `object Inputs` keeps only action-named inputs.
- [ ] All gates of the spec's Verification pass.


## Done summary
# fn-126.3 done summary

Branch `umpire-fn126-3` in wt/lane-a, on `umpire` 8fb7426add. `git merge umpire` before the final gates reported "Already up to date". Commits:
- ac9905f2c5 feat(umpire): transparent section and actor objects for Definition IDs (framework, lifter, fixtures);
- 23c77ffb56 style(umpire): wrap the section and actor comments at the column limit (no message or behavior change);
- b4c7016000 refactor(model): group every feature's actions by actor (Models, realizations, dependent fixtures, IR, Cases, Go positions);
- a4b412a123 docs(model): actor and section objects in the README and the DSL study.

After review (SHIP, four P3s):
- ad9a0315ab fix(irgen): find a file's package object rather than spell it from its name. It adds the unpinned fixture, and the refusal message drops the trailing `$`.
- c78ef51717 style(model): wrap three signature comments at the column limit. Line counts are kept, so no position moved.
- bce41cbd4b refactor(model): name the Nexus caller's request deadline as the activity's.
- 82078f8eff test(golden): inventory the lifter's new sections fixture (`later_inventory`). Without it, ad9a0315ab alone fails the migration goldens.

### Framework markers (model/umpire)
- `model/umpire/Section.scala`:
  - `trait Section` is the marker.
  - `abstract class Actor extends Party(), Section` is a party named after its object, with the first letter lowered.
- `Party` is opened: `final case class` became `case class`. Its runtime name stays `""`, as `Party()` gave it; the lifter names it.

### Lifter transparency rule and its fixtures (model/irgen)
- `Context.definitionId`: when a val's owner is a section, it takes `<sectionOwner>.<name>`.
  - At a file's top level, `sectionOwner` is the file's package object: the file's `DefinitionScope` pin if it has one, else the object's full name.
  - The object is found as the owner of one of that file's own top-level definitions among the lifted sources, not spelled from the file name.
  - A section in a file that declares nothing at its top level has no package object, and it is refused.
  - Directly in a machine's object (an object at a file's top level that declares a `Machine`/`Composition` val), it is that object's pin or full name.
- The lifter refuses, each at its line:
  - a section inside a section;
  - a section anywhere else;
  - a section that pins;
  - two members that would share an ID (`idTakenBy`, with a section-specific reason).
- `Declarations.partyName`: an action names an actor object as its party by `this` or by reference (`action(process)`, `action(caller)`). Any other party falls back to `constString` as before.
- Lifting fixture: `lifts/Captured.scala` (pinned to `fixture.spelled.Spelled$package$`).
  - It now holds `object client extends Actor` (put), sections `background` (flush, expire), `faults` (crash) and `relaying` (send = `action(client)`).
  - `expected/captured.json` changes in positions only.
  - The captured test now asserts every action ID (`fixture.spelled.Spelled$package$.<name>`) and party.
  - R6's "two owners pin one former owner" fixture is task 1's `object Watched` in the same file, unchanged.
- Unpinned lifting fixture, after review: `lifts/Sections.scala`, root `fixture.sections.Switch$.switch`, `expected/sections.json`.
  - The file pins nothing. A top-level `object panel extends Section`'s `flip` takes `fixture.sections.Sections$package$.flip`, as the file's own top-level `reset` does.
  - `Switch.operator extends Actor`, directly in the machine's object, gives `press` the ID `fixture.sections.Switch$.press` and the party `operator`.
  - The test "a section's member takes its owner's ID in a file that pins nothing" asserts the three IDs and parties.
  - A file name holding a dot (the review's `X.test.scala`) cannot carry top-level definitions under `-Werror`: scalac warns that the package name `X.test$package` "will be encoded on the classpath". So the fixture uses a plain name.
- Refusal fixtures, appended to `lifts/Rejects.scala` and listed in `expected/rejects.txt`:
  - `sectionNested` (:1162);
  - `sectionMisplaced` (:1173, a section in an object of no machine). Its message now names `fixture.rejects.Holder`, without the `$`;
  - `sectionTwins` (:1187, `leftHand.clap`/`rightHand.clap` both `fixture.rejects.Rejects$package$.clap`);
  - `sectionPinned` (:1197).

### Groups per feature
| Feature | Actor objects | Sections |
| --- | --- | --- |
| standaloneactivity | `caller` (start, control) | `worker` (attemptStart, attemptResult; party = the shared worker, imported `shared.worker.{worker as process}`), `timers` (timeout, backoff), `deadline` (scheduleToClose, scheduleToStart, startToClose) |
| record | — | `history` (dispatch, answerDelivery) |
| nexuscaller | `caller` (schedule), `handler` (handlerReply, complete), `network` (transportFault) | `timers` (timeout, backoff), `deadline` (three), `Control.caller` (inspect, under `Control`'s pin `…Control$`; party = the feature's `caller`) |
| closepolicy | — | `callerSide` (callerClose, reset, requestCancel), `handlerSide` (handlerFinish), `history` (deliverCancel) |
| nexusoperation | `caller` (start, requestCancel, terminate), `handler` (handlerReply, complete) | — |
| shared/taskqueue | — (`val fault = Party()` stays the party) | `queue` (enqueue, deliver, acknowledge, addActivityTask, persistTask, syncMatch), `faults` (storageLoss, crash, ackLoss) |
| shared/worker | `worker` (workerStop, workerResume, serve), replacing `val party = Party("worker")` | — |

- Every top-level `val … = action(…)/timer/internal` under model/temporal is gone (grep empty).
- Call sites in Scenarios, Properties, capabilities, compositions and realizations all show the actor.

### Moved inputs
- standaloneactivity: `scheduleToClose`, `scheduleToStart`, `startToClose` and `result` move to the top level. `object Inputs` keeps only `control`.
- nexuscaller: `scheduleToClose`, `scheduleToStart`, `startToClose`, `reply` and `resolution` move to the top level. `Inputs` is removed.
- nexusoperation: `reply` and `resolution` move to the top level. `Inputs` is removed.
- Input names come from the token vals, so the IR is unchanged.

### R5 deltas and where they are recorded
- **Projection.** `.flow/tmp/fn-126/fn126-3/project3.py`, strict: lines are dropped, the binding texts are mapped, and everything else must be equal. The before side is `before/` = 8fb7426add (`before/REV`).
  - `ir-deltas.json`: OK. 10 IR/law files and 2 Cases are equal under the deltas; everything else is identical. That includes all 7 `*.lint.json`, `manifest.json` and 19 of 21 Cases.
  - `lifts-deltas.json`: OK. captured, hints, hintsRefused and taskqueue are equal under the deltas.
- **Positions.** Lines only; no path changes.
  - Cases: one `waitHints[].source.line` each in `activity-retry` (228→235) and `activity-scheduleToStartTimeout` (229→236), both in the activity's `Realization.scala`.
- **Law-sidecar binding text** (display only; no key, ID or fingerprint reads it):
  - `…StandaloneActivity$package.control.` → `…standaloneactivity.caller.control.`;
  - `…NexusOperation$package.requestCancel|terminate` → `…nexusoperation.caller.requestCancel|terminate`.
  - Recorded in `project3.py` `BINDINGS`, as tasks 1 and 2 recorded theirs.
- **Function symbols and source root strings:** none moved. No action is a function, and no root moved.
- **`tools/umpire/internal/golden/config.json`: no R5 entry needed.** The projection has `positions_by_file`, and there is no symbol or root delta. The original-baseline and migration tests pass. Its one change, after review, adds the new lifter fixture `expected/sections.json` to `later_inventory`.
- **Unchanged:**
  - Definition IDs, party names, action classes and IR type names;
  - machine, Property, Scenario, Query, Limits and law names;
  - tables, answers, lint findings, coverage and fingerprints;
  - the generated testpilot Cases and the canary Case (`umpire-gen-fixtures` and `canary-gen-case` rewrote nothing).
- No pinned Run needed re-recording.
- `rejects.txt` also changes two ScriptRejects messages because the fixture's input moved: `…Inputs.scheduleToStart` → `…StandaloneActivity$package.scheduleToStart`.

### Gates after review (logs in .flow/tmp/fn-126/fn126-3/)
- `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`: `gen6.log`, exit 0. `gen5.log` failed on the dotted fixture name before it was renamed.
  - `model/ir`, `model/cases` and the existing lifter expected files are byte-identical to before the P3s.
  - `rejects.txt` changes only in the `Holder` message, and `expected/sections.json` is new.
- Projection rerun against the same `before/`: `ir-deltas.json` OK, `lifts-deltas.json` OK, the same deltas as below.
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`: `model-gate2.log`, exit 0.
- `make lint-model`: `lint-model3.log`, exit 0.
- `make umpire-check-cases`: `umpire-check-cases2.log`, exit 0.
- `go test -p 1` under the flock of `./tools/umpire/lint`, `./tools/umpire/model`, `./tools/umpire/internal/golden` and `./tools/umpire/lower`: `go-lint-model3.json`, all ok.
  - `go-lint-model2.json` is the run before the inventory entry; it failed with "unknown IR inventory entry …/sections.json".
- Full Go suite, `-p 2`: `go-suite3.json`. 17 of 18 packages pass; `model` was OOM-killed and passes alone at `-p 1` (above).
- `lint-code-fast`: `lint-code-fast2.log`, 0 issues.
- `git merge umpire`: already up to date.

### Gates before review (logs in .flow/tmp/fn-126/fn126-3/)
- `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`: `gen1.log` failed (lifts/TaskQueue.scala consumer, then fixed); `gen2.log`, `gen3.log` and `gen4.log` all exit 0.
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` on HEAD: `model-gate.log`, exit 0. This includes the irgen munit fixtures and the Models' munit tests.
- `make lint-model`: `lint-model2.log`, exit 0. Its scalafix warnings are pre-existing.
- `umpire-gen-cases`, `umpire-gen-fixtures` and `canary-gen-case` regenerated nothing new. `umpire-check-cases`, `umpire-check-fixtures` and `canary-check-case` exit 0; each is logged by target name.
- Full Go suite, `-json -p 2`, on HEAD: `go-suite2.json`, 3m28s wall. 16 of 18 packages pass.
  - `model` and `export` were OOM-killed ("signal: killed"). Rerun alone at `-p 1`, both pass (`go-rerun.json`).
  - Slowest: lower TestOriginalBaselineCases 53.8s, model TestMigrationProjectionPreservesSemantics 43.7s.
- The first suite run (`go-suite1.json`, before commit) found two position-only expectations, fixed in b4c7016000:
  - `model/diagnostics_test.go` NexusCaller.scala:460→466;
  - `model/hints_fixture_test.go` Hints.scala:126→127, a wrapped line.
- `tools/umpire/lint` passes with no `-update-coverage` needed.
- `make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast`: `lint-code-fast.log`, 0 issues.

### Decisions
1. **Where a section may sit before task 4.**
   - At a file's top level, or directly in a top-level object that declares a machine or composition val. That is today's machine object, and it is needed for `Control.inspect`, which must keep `…Control$.inspect` and the control family.
   - Task 4 replaces the test with `Machine`/`Derived`/`Composition` objects.
2. **The close policy's sections are `callerSide`/`handlerSide`.**
   - `object caller`/`object handler` in `closepolicy` compile to `caller$.class`/`handler.class`. Those differ only in case from the enums `Caller`/`Handler`, whose IR type names are frozen.
   - scalac refuses the pair, and this file system is case-insensitive: the first attempt corrupted the bloop state, which `make umpire-clean-scratch` cleared.
   - The parent's actions keep their own objects (`handler.complete`, `deadline.scheduleToClose`).
3. **Inputs follow today's action names.** A Nexus `reply` is not named like an action today (`handlerReply`), so it moved to the top level with the rest. See "For the owner" on R18.
4. **`fault` stays a plain party,** `val fault = Party()`, with `object faults extends Section`. The task lists `faults` as a section and lists `caller`/`handler`/`network`/`worker` as the parties, and the party name `fault` must not change.
5. **Placement of the queue's matching steps.** The six queue-internal steps all sit in `queue` (all `on taskQueueEntity`), and `history` is used for the record's and the close policy's server steps.
6. **Nexus caller imports the shared worker as `worker`.** It has no worker section of its own, so it reads `worker.workerStop`. The activity imports it as `process`, as R14 says.
7. **Name collisions in realizations.**
   - The kit's wildcard-imported `temporal.realize.deadline` operand beats the package's `deadline` section from another file. Scala ranks wildcard imports above other-file package members.
   - The activity's realization therefore imports `{deadline as requestDeadline, *}`.
   - The Nexus caller's realization never imports the kit's `deadline`. Its collision was its own private `deadline` value, a request Duration, which shadowed the section. After review that value is named `requestDeadline` too, so both realizations use one name, and the server steps read `deadline.scheduleToStart`.
   - That is a rename of a realization-private value that no IR names, a small step past R8's "imports and qualified references"; no position moved.
8. **The transparency check is in the lifter, not the declaration-order lint.** So a misplaced section is refused when a member is lifted, as the task specifies. A misplaced section holding nothing ID-bearing is not refused. R17's section lint (task 4) can cover it.

### For the owner
- **R18 conflicts with the close policy's types.** R18's table renames `caller.callerClose` → `caller.close` and `handler.handlerFinish` → `handler.finish`. In `closepolicy` those objects cannot be named `caller`/`handler` (decision 2), so task 6 should rename `callerSide.callerClose` → `callerSide.close` and so on, or pick other names.
- **R14's "one place where two objects name one party" no longer holds.** There are now three places:
  - the activity's `worker` section;
  - the close policy's `callerSide`/`handlerSide` (parties `caller`/`handler`);
  - the Nexus caller's `Control.caller` (party `caller`).
  - Each is a Section that names a party declared elsewhere, the same pattern R14 gives the activity.
- **R18 needs a Nexus input moved.** R18's `handler.handlerReply` → `handler.reply` will make the top-level input `reply` unreadable inside `object handler`, where `reply` would name the action. Task 6 must move it into `object Inputs` (`Inputs.reply`), as R14's rule says. This applies to both nexuscaller and nexusoperation.
- Commit ad9a0315ab does not pass the Go migration goldens on its own; 82078f8eff right after it adds the inventory entry. History was not rewritten.
- `Party` is now open, so an `Actor` is a `Party("")` at runtime, as `Party()` always was. Nothing at runtime reads or compares parties.

Subagents used: 0.

Review: claude-opus-5-5 at high, fresh context (host-dispatched subagent). Writer and reviewer are the same family (Opus). Round 1: SHIP.
- The reviewer:
  - verified `before/` byte for byte;
  - reran `project3.py` (OK);
  - compared every action's ID, party, name, flags and `on` raw;
  - confirmed `LawClaim.Bindings` is read only by messages, not keys or fingerprints.
- One P2, deferred by the host to task 4's R17 lint (`.flow/tmp/fn-126/carry-forward.md`): a whole-index check for twin section IDs and misplaced sections. Today each IR file's lift has its own `Context`. No current instance exists.
- P3s applied in ad9a0315ab..82078f8eff:
  - the real package object is looked up, with positive unpinned fixtures;
  - line wraps;
  - one `requestDeadline` spelling;
  - the `$`-free message.
- The Nexus caller's realization renamed its own private `deadline` value. That is slightly beyond R8's "imports and qualified references", has no IR effect, and is accepted by the host.
- Deferred to task 6: `Control.caller` shadowing, and the sketch's `Inputs.result`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: ac9905f2c5, 23c77ffb56, b4c7016000, a4b412a123, ad9a0315ab, c78ef51717, bce41cbd4b, 82078f8eff
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (gen2-4.log), make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (model-gate.log), make lint-model (lint-model2.log), make umpire-gen-cases umpire-gen-fixtures canary-gen-case, make umpire-check-cases umpire-check-fixtures canary-check-case, go test -count=1 -json -tags test_dep -p 2 -timeout 30m ./tools/umpire/... (go-suite2.json; model, export OOM-killed), go test -count=1 -json -tags test_dep -p 1 -timeout 30m ./tools/umpire/model ./tools/umpire/export (go-rerun.json, ok), make GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main lint-code-fast (0 issues), python3 .flow/tmp/fn-126/fn126-3/project3.py before/ir model/ir before/cases model/cases (OK), python3 .flow/tmp/fn-126/fn126-3/project3.py lifts-a lifts-b (OK), review reruns: umpire-gen-model (gen6.log), projection OK, umpire-check-model (model-gate2.log), lint-model (lint-model3.log), umpire-check-cases, go test -p 1 lint/model/internal/golden/lower (go-lint-model3.json, ok), go suite -p 2 (go-suite3.json; model OOM-killed, ok alone), lint-code-fast (0 issues)
- PRs: