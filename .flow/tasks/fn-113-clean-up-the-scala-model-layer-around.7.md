---
satisfies: [R1, R2, R15, R17, R19]
---
# fn-113-clean-up-the-scala-model-layer-around.7 Remove the sets, Coverage and the rules with no Go consumer, and verify Part A

## Description
Remove the sets, Coverage and the rules with no Go consumer, and verify Part A. Implements R1 and R2 (verification), R17, the no-consumer clause of R15 and part of R19. First task that edits `model/umpire` and the Models.

**Size:** M
**Files:** model/umpire/Sets.scala and Coverage.scala (deleted), model/umpire/Domain.scala (`Keys.spelling`), model/temporal/nexuscaller/Claims.scala, model/temporal/standaloneactivity/Claims.scala, model/temporal/test/NexusCallerPins.test.scala, model/temporal/test/StandaloneActivityPins.test.scala, model/ir/*.json (regenerated), model/README.md (only if it names a set)
**Touches:** [model/umpire/Sets.scala, model/umpire/Coverage.scala, model/umpire/Domain.scala, model/temporal/nexuscaller/Claims.scala, model/temporal/standaloneactivity/Claims.scala, model/temporal/test/NexusCallerPins.test.scala, model/temporal/test/StandaloneActivityPins.test.scala, model/ir/**, model/README.md (set mentions only)]

### Approach
- Part A first (R1, R2): confirm `Canonical.scala`, `Lower.scala`, `Alterer`, `Table.alter` and `Machine.alterer` are absent (`grep -rn 'Alterer\|alterer\|Canonical' model/`), restate fn-115.7's borderline-kept list, and cite its passing check-mode gate with unchanged IR (`.flow/tmp/fn115-7-summary.md` section 1 and `.flow/tmp/fn115-7/check-model*.log`; `model/umpire` non-test is 2,415 lines). Rerun the check-mode gate only if that evidence is invalidated (MILESTONES.md: reuse a passing baseline). Record the outcome in the summary.
- Record before deleting (R17): each set with name, purpose, bindings, repeat, Queries, machine, coverage goals and budget: `nexusCallerTests`, `nexusCallerCanary`, `nexusCallerExploration` (`nexuscaller/Claims.scala:273-310`), `standaloneActivityTests`, `standaloneActivityCanary`, `standaloneActivityExploration` (`standaloneactivity/Claims.scala:242-260`).
- R17's stop clause: the set names appear in Go and testdata only as Case and pin labels (`tools/canary/assessment/testdata/nexusCallerCanary-*`, `tools/umpire/lower/{lower,activity,activity_cases}_test.go`, `tools/umpire/conformance/{nexus,activity}_test.go`, `tools/umpire/cmd/umpire-fuzz/run_test.go`). Confirm they are strings fixed in Go or in the Case manifest and that no consumer reads a set from Scala or the IR (the IR has no set message). If one does, stop and report for the owner's decision.
- Delete `UmpireSet`, `Purpose`, `Binding`, the set member of `Declaration`, `checkSet`, `CoverageGoal`, `CoverageTarget`, `Coverage.targets` and `Coverage.within`, and `Keys.spelling` (its only caller was Coverage); these three rules were the ones fn-115.12 marked for removal here (R19). `check`, `checkQuery` and `checkModel` stay for task 8.
- Tests, by the audit's reference: delete "the sets" (`NexusCallerPins.test.scala:212-255`) and the set arguments of the `check(...)` calls (`NexusCallerPins.test.scala:300-315`, `StandaloneActivityPins.test.scala:146-162`); the rest of those suites stays for task 8.
- Declarations follow the sets in both `Claims.scala` files, so their positions shift: regenerate with `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`; the diff of `model/ir` must show position changes only. The goldens pass under task 5's projection; Case bytes, the functional fixtures and the canary Case do not change.
- The DSL must still compile alone (gate step "compile the framework alone"). R25: nothing generic is written here.

### Investigation targets
**Required**:
- `model/umpire/Sets.scala:1-128`, `model/umpire/Coverage.scala:1-85`, `model/umpire/Domain.scala:117-126`
- `model/temporal/nexuscaller/Claims.scala:273-310`, `model/temporal/standaloneactivity/Claims.scala:242-260`
- `model/temporal/test/NexusCallerPins.test.scala:212-255,300-315`, `model/temporal/test/StandaloneActivityPins.test.scala:146-162`
- `tools/umpire/lower/lower_test.go` and `tools/canary/assessment/testdata/` (set names as labels)
- `.flow/tmp/fn115-7-summary.md` (section 1), `.flow/tmp/fn115-12-summary.md` (section 6 table), `.plans/umpire-scala-evaluator-audit.md`

### Quick commands
mise exec -- scala-cli compile model/project.scala model/umpire; mise exec -- scala-cli test model/project.scala model/umpire model/temporal; CC=/usr/bin/clang mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks; git diff --stat model/ir (positions only); CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model ./tools/umpire/lower; make umpire-check-cases umpire-check-fixtures canary-check-case; mise exec -- make lint-model

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] Part A verified: the named files and declarations are absent, the borderline list is restated, and the summary cites the evidence (or a fresh check-mode gate) that Part A changed no lifted output.
- [ ] `UmpireSet`, `Purpose`, `Binding`, `Coverage`, `CoverageGoal`, `CoverageTarget` and `Keys.spelling` are gone and nothing under `model/` names them; the summary lists each of the six sets with purpose, bindings, repeat, Queries, machine, goals and budget.
- [ ] No Go consumer needed a set (checked and recorded); the regenerated IR differs in positions only; the goldens pass under the projection; `umpire-check-cases`, `umpire-check-fixtures` and `canary-check-case` report no change; the DSL compiles alone and `lint-model` passes.


## Done summary
# fn-113.7 handover: remove the sets, Coverage and the rules with no Go consumer; verify Part A

Worktree: `temporal-milestones` (branch `umpire-milestones`, HEAD 9d5e42eb7). Nothing staged or committed.

### 1. Part A (R1, R2): verified

- Absence (`.flow/tmp/fn113-7/part-a-absence.log`): `grep -rn 'Alterer\|alterer\|Canonical' model/` finds nothing (exit 1); `model/umpire/Canonical.scala` and `model/umpire/Lower.scala` do not exist; `grep -rn alter model/umpire` finds nothing, so `Table.alter` and `Machine.alterer` are gone.
- `model/umpire` non-test was 2,415 lines before this task (`.flow/tmp/fn113-7/lines-before.log`), the figure the spec records after fn-115.7.
- R2 evidence: `.flow/tmp/fn115-7-summary.md` and its logs do not exist on this machine. Instead: every gate run since fn-113.2 passed in check mode, without `--update`, and left the IR unchanged (`.flow/tmp/fn113-2/check-model-skip-go.log`, `.flow/tmp/fn113-4/check-model.log`). Before this task's edits, `git status --short model/ir model/cases model/lifter/testdata` was empty. No fresh Part A gate was needed.
- Borderline-kept list, restated from `model/umpire` as it stands. These declarations have no reader in the lifter, the gate or the Models; they stay only because a test or the evaluator (Search, Refine, Compose, `check`) reads them. All of them go with task 8:
  - Read only by tests: `Table.rowKeyOf`, `Table.stateValueOf`, `Search.unwatched`, `Refine.RefinementRow`.
  - Read by `check` (now in `Check.scala`) and tests: `Machine.refinementCheck`, `Machine.hasRefinement`, `Machine.visibleFacts`, `Machine.visibleOutcomeSet`.
  - Read only by Search: `Table.stateAtom/actionAtom/outcomeAtom/factAtom`, `Atom`, `Trace`, `TraceStep`, `Claims` `holds2`, `isTransition`, `triggers` and `whenLabel`, `Machine.monitorList`, `Compose.memberModels`, `Refine.mapValue`.
  - Read by Search, Refine and Compose: `Table.rowsFrom`, `Row`, `RowResult`, `Table.stateValue`, `Machine.stateValue`, `Table.refinedField`, `Table.rowKey`.
  - Gone in this task: `Table.claimEntries` and `ClaimEntry`, which only Coverage read (see section 4).

### 2. The six sets, recorded before deletion

The source text before deletion is in `.flow/tmp/fn113-7/sets-before.txt`. All six had `repeat = ""` unless stated otherwise. "Bindings" lists every party except system.

| Set | Purpose | Bindings | Repeat | Queries | Machine | Goals | Budget |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `nexusCallerTests` | functional | caller driven, handler driven, network observed, worker driven | `implementation` (each Case runs under HSM and under CHASM) | `functionalQueries`: syncCompletion, asyncCompletion, asyncFailure, handlerError, retry, scheduleToStartTimeout, startToCloseTimeout | none | none | none |
| `nexusCallerCanary` | canary | caller driven, handler **observed**, network observed, worker driven | none | syncCompletion, asyncCompletion | none | none | none |
| `nexusCallerExploration` | exploratory | caller driven, handler driven, network observed, worker driven | none | none | `nexusProtocol` | rows, results, classMembers | `four` (steps 4, actions 4, search 32768) |
| `standaloneActivityTests` | functional | caller driven, worker driven | none (standalone activities exist only under CHASM) | `functionalQueries`: completion, nonRetryableFailure, retry, cancel, terminate, pauseResume, scheduleToStartTimeout, startToCloseTimeout | none | none | none |
| `standaloneActivityCanary` | canary | caller driven, worker **observed** | none | completion, cancel | none | none | none |
| `standaloneActivityExploration` | exploratory | caller driven, worker driven | none | none | `activityProtocol` | rows, results, classMembers | `four` (steps 4, actions 4, search 32768) |

Other authored facts that go with the sets:
- The canary rule was "a canary Query's path takes no silent step (a step that records no fact): no deployment can close that gap". The pin's negative case was `retry`, which takes the silent step `backoff`. The nexus canary comment named the backoff and the worker stop as silent steps.
- The nexus exploration's targets under `four` were 889: 885 rows; 2 results (`...outcome.nexusProtocol.accepted` and `notFound`); and 2 class members. The second-to-last target was `handlerReply-handlerError-false` (`className` "handlerError (retryable := false)", example "BadRequest"). The 1,152-row count of `nexusProtocol` is still asserted by `TestNexusProtocolTable`.
- Exploration targets were enumerated in the machine's catalog order: rows in table order, the outcomes those rows reach in catalog order, and class members in claim order, each goal in the order given. The list was cut at the budget's search count, so the enumeration was the same on every reading.

### 3. Go-consumer check (R17 stop clause): no consumer, so no stop

Log: `.flow/tmp/fn113-7/go-consumer-check.log`.
- The IR schema (`ir.proto`) has no set, Purpose, Binding or coverage message. `model/ir` and `model/cases` do not contain any set name.
- In live Go (`tools/`, `tests/`), each set name appears only in `_test.go` files and testdata, as fixed strings:
  - Case IDs and labels: `cp.IdentityFor("temporal.case", "nexusCallerTests", q)` in `tools/umpire/lower/{lower,activity,activity_cases}_test.go` and `tools/umpire/conformance/{nexus,activity}_test.go`, `played_test.go` and `tests/testcore/testpilot/model_fixture_test.go`.
  - Canary testdata file names: `tools/canary/assessment/testdata/nexusCallerCanary-*`, `admission_test.go` and `controller_test.go`.
  - The `--set nexusCallerExploration` flag string in `tools/umpire/cmd/umpire-fuzz/run_test.go`.
- The live `umpire-fuzz --set` value is resolved by `tools/umpire/explore/bridge.go` (`server.model`) against an IR `Query.exploration.name` (for example `nexusDeadlines`), not against a Scala set.
- Non-test Go that names the sets is only in the archives (`model0/go/*/claims.go` and `.plans/archive`), which define their own Go `umpire.Set` values and never read Scala or the IR.

### 4. What was deleted or changed

- `model/umpire/Sets.scala` and `model/umpire/Coverage.scala` were moved to `.flow/tmp/trash/fn113-7/`. This removes `UmpireSet` and `UmpireSet.targets`, `Purpose`, `Binding`, `checkSet`, the `UmpireSet` member of `Declaration`, `CoverageGoal`, `CoverageTarget`, `Coverage.targets` and `Coverage.within`.
- **New file `model/umpire/Check.scala` (outside Touches).** `check`, `checkQuery` and `checkModel` stay for task 8, so they moved out of the deleted `Sets.scala` unchanged. `Declaration` became `Query | Model`; the scaladoc lost "and set rules". Diff against the original: `.flow/tmp/fn113-7/Sets.scala.before`.
- `model/umpire/Domain.scala`: `Keys.spelling` was removed together with its scaladoc.
- **`model/umpire/Table.scala` (outside Touches).** `Keys.spelling` had one more caller, `Table.claimEntries`, which built `ClaimEntry.spelling`. The task file's "its only caller was Coverage" holds only indirectly. `claimEntries` and `ClaimEntry` were read only by `Coverage.targets`, so both were removed (26 lines). Task 8 lists `ClaimEntry` among the declarations that go, and deletes the whole of `Table.scala`.
- In both `Claims.scala` files, the three sets were removed, along with `drivenAll` and the activity's "### The sets" section and its CHASM comment. Comments that pointed at the sets were reworded:
  - "The functional set's Queries" became "The functional Queries".
  - In the nexus Queries header, "realized by the set below / outside the set" became "realized as a Case / outside `functionalQueries`".
  - In the activity's `cancelRequest` scaladoc, "no Query of the sets / in no set" became "no functional Query / not one of them". The new wording is one line longer, which is why `terminalHolds` and `pauseHolds` move by +1.
- `functionalQueries` and the `four` budgets stay: `functionalQueries` is a gate root (`model/gate/Roots.scala`).
- Tests:
  - `NexusCallerPins.test.scala`: `test("the sets")` (old lines 212-255) was deleted, and the three set arguments were removed from the `check(...)` call (old line 300). The rest of `:300`, and `:156` and `:257`, are untouched (provisional, task 8).
  - `StandaloneActivityPins.test.scala:146`: the three set arguments were removed, along with the whole two-line comment above that test: its first half repeated the test name, and its second half ("and the canary admits only paths whose every step records evidence") described the deleted canary rule. `:164` is untouched.
- `model/README.md` names no set, so it is unchanged. No reference to the retired proof front end was written. R25: no library was weighed, because nothing generic was written.
- Lines of `model/umpire` non-test: 2,415 before, 2,224 after (-191; logs `lines-before.log` and `lines-after.log`). `model/temporal` non-test: 5,414 after.

### 5. Checks (logs under `.flow/tmp/fn113-7/`)

| Command | Result | Log |
| --- | --- | --- |
| `mise exec -- scala-cli compile model/project.scala model/umpire` | pass (the DSL compiles alone) | `compile-umpire.log` |
| `mise exec -- scala-cli test model/project.scala model/umpire model/temporal` | pass, 34 tests (35 minus "the sets") | `munit.log` |
| `mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` | pass; only `model/ir/activity.json` changed (16 lines); `model/cases` and `model/lifter/testdata` unchanged | `gen-model.log` |
| `python3 .flow/tmp/fn113-7/positions-only.py` (parses HEAD and the working copy, compares every changed leaf, and checks equality with `position.line` masked) | 16 changed leaves, all `position/line`; equal when masked: "positions only" | `ir-positions-only.log` |
| `mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | pass in check mode, including the vocabulary step and "compile the framework alone" | `check-model.log` |
| `GOFLAGS=-p=1 go test -p 1 -timeout 30m -tags test_dep -run '^(TestMigrationGoldens\|TestMigrationProjectionKeepsLoweredCases)$' ./tools/umpire/model` | pass | `goldens-model.log` |
| same for `./tools/umpire/lower` | pass (TestMigrationGoldens and TestMigrationProjectionKeepsLoweredCases) | `goldens-lower.log` |
| `mise exec -- make umpire-check-cases` | pass, no change | `umpire-check-cases.log` |
| `mise exec -- make umpire-check-fixtures` | pass, no change | `umpire-check-fixtures.log` |
| `mise exec -- make canary-check-case` | pass, no change | `canary-check-case.log` |
| `mise exec -- make lint-model` | pass (exit 0). The `NoSuchFieldException: path` traces from scalafix are pre-existing and appear in the fn113-2 to fn113-5 lint logs too | `lint-model.log` |

Not run, as instructed: Go checks in the gate, `umpire-check-backends` and `lint-code-fast`. No Go file changed.

### 6. For the conductor

- **Touches deviations to accept or redirect:** the new `model/umpire/Check.scala` and the edit to `model/umpire/Table.scala`, both explained in section 4. Task 8 should delete `check` from `Check.scala`; its Touches `model/umpire/**` already covers this.
- **Shared docs:**
  - `.plans/umpire-migration-manifest.json` names only the archived `model/scala{,v2}/umpire/{Sets,Coverage}.scala` paths. No current path changes, so an update is needed only if the manifest tracks live `model/umpire` files.
  - MILESTONES.md: record that fn-113.7 removed the sets and Coverage, and the new `model/umpire` baseline of 2,224 lines.
  - `.plans/umpire-scala-evaluator-audit.md`: the C entry at NexusCallerPins `:212` and the set parts of `:300` and StandaloneActivityPins `:146` are done. The remaining line numbers in NexusCallerPins shifted: "the composition" is now at about :212 and the checks test at about :255.
- The set names stay in Go tests and testdata as Case-ID and file labels (section 3). Renaming them is a separate decision and is not needed.

### 7. Review

The independent review returned SHIP and accepted both out-of-Touches edits (`Check.scala`, `claimEntries`). I applied its three points:

1. **Stale set comments, rewritten without changing any comment's line count:**
   - `model/temporal/nexuscaller/Model.scala:3-5`: "the functional set runs one Query per side effect ... once per value of the implementation switch" now reads "the functional Queries are one per side effect that settles the operation". The implementation-switch clause is dropped; it is recorded in section 2.
   - `nexuscaller/Model.scala:9`: "what the set asks" became "what the Queries ask".
   - `nexuscaller/Model.scala:200-201`: "No set names the composition" became "No functional Query reads the composition".
   - `model/temporal/standaloneactivity/Model.scala:4`: "the functional set runs one Query" became "the functional Queries are one".
   - `standaloneactivity/Model.scala:15`: "what the set asks" became "what the Queries ask".
   - `model/temporal/standaloneactivity/Realization.scala:224`: "every path of the functional set" became "every functional Query's path".
   - `grep -rn 'functional set\|the set\b\|No set' model/temporal model/umpire model/README.md model/SEMANTICS.md` now finds nothing.
2. Section 4 now says the whole two-line StandaloneActivityPins comment was removed.
3. Section 2 now records how exploration targets were ordered and cut.

Checks after the review edits (logs under `.flow/tmp/fn113-7/`):

| Command | Result | Log |
| --- | --- | --- |
| `mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` | pass. `model/ir/activity.json` has the same 16 position-line changes as before, with identical values, so the comment edits moved no lifted position | `gen-model-review.log` |
| `python3 .flow/tmp/fn113-7/positions-only.py` | positions only | `ir-positions-only-review.log` |
| `mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` | failed once, then passed on rerun (see below) | `check-model-review.log` (failed), `check-model-review-rerun.log` (pass) |
| `mise exec -- make lint-model` | pass | `lint-model-review.log` |

The failed run broke in the gate's own suite: `GateSuite` "the IR's classes are packaged when the schema changed, and with --if-stale only then" (`model/gate/test/Gate.test.scala:111`). That test rewrites the schema right after packaging and expects the jar to be stale. The likely cause is timestamp resolution; this task changed no gate file.

The goldens were not rerun, because `model/ir` is byte-identical to the state they passed against.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 734ba4acb107b077f6d512cca1f60233588c09d6
- Tests: mise exec -- scala-cli compile model/project.scala model/umpire -> pass, mise exec -- scala-cli test model/project.scala model/umpire model/temporal -> pass (34 tests), mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks -> pass; model/ir/activity.json positions only, python3 .flow/tmp/fn113-7/positions-only.py -> positions only (16 position.line leaves), mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> pass, GOFLAGS=-p=1 go test -p 1 -tags test_dep -run '^(TestMigrationGoldens|TestMigrationProjectionKeepsLoweredCases)$' ./tools/umpire/model -> pass, GOFLAGS=-p=1 go test -p 1 -tags test_dep -run '^(TestMigrationGoldens|TestMigrationProjectionKeepsLoweredCases)$' ./tools/umpire/lower -> pass, mise exec -- make umpire-check-cases -> pass, no change, mise exec -- make umpire-check-fixtures -> pass, no change, mise exec -- make canary-check-case -> pass, no change, mise exec -- make lint-model -> pass, review: mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks -> pass; model/ir unchanged from first regeneration (positions only), review: mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> fail once (GateSuite --if-stale timestamp flake, Gate.test.scala:111), rerun pass, review: mise exec -- make lint-model -> pass, independent review (claude-opus-5-5, fresh context): SHIP; stale set comments reworded, handover nits applied
- PRs: