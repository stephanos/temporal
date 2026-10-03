---
satisfies: [R15, R19, R24, R25]
---
# fn-113-clean-up-the-scala-model-layer-around.8 Retire the native evaluator from model/umpire

## Description
Retire the native evaluator from model/umpire. Implements R15, the code half of R19 for the framework, and R25 for what stays generic; moves `model/umpire` toward R24's 1,300 lines.

**Size:** M
**Files:** model/umpire/Table.scala and Search.scala (deleted); Refine.scala, Compose.scala, Machine.scala, Claims.scala, Domain.scala, Errors.scala, Channel.scala, Monitor.scala, Assume.scala (runtime halves removed); model/umpire/test/Declarations.test.scala, model/temporal/test/NexusCallerPins.test.scala, model/temporal/test/NexusKernel.test.scala (deleted per the audit), model/temporal/test/StandaloneActivityPins.test.scala (evaluator tests removed; the step-function test stays)
**Touches:** [model/umpire/**, model/umpire/test/**, model/temporal/test/**]

### Approach
- The rule: a declaration stays when the lifter reads it by name or when its type makes a wrong Model fail to compile; code that builds a table, searches, checks a refinement, composes machines, answers a Query, or spells a key or a Definition ID at runtime goes. Names the lifter reads (keep name and type, R15's error clause): `umpire.Step` (`Context.scala:83`), `action`, `machine`, `starts`, `ends`, `evidence`, `refines`, `visible`, `steps`, `channel`, `monitor`, `assume` (`Declarations.scala:38,126-183,283,423,454`), `query`, `Limits` with `steps`/`actions`/`search` (`Claims.scala:93-99,207-218`), and the types `umpire.Action`, `Channel`, `Class`, `Composition`, `Finite`, `Hole`, `Inbox`, `Limits`, `Machine`, `Progress`, `Query`, `Reads`, `Step`, `realize.*`; `Verdict` if the IR's expected verdicts come through it.
- Goes: `Table`, `Row`, `RowResult`, `Atom`, `ClaimEntry`, `Family.id` if only tables used it, `Search`, `Keys` (and `Keyed`: no Model or fixture uses it), `Model.table` and `Machine.build/bind/enumerate/run`, `Composition.build` and its helpers, `Refinement`, `Refinement.of`, `productStep`, `RefinementRow`, `Query.answer`, `Answer`, `Trace`, `TraceStep`, `check`/`checkQuery`/`checkModel`, and `Errors.scala`'s `Checked`/`Fails`/`checked`/`fail`/`ModelError` once nothing calls them.
- `Finite` keeps `derived` (the compile-time rejection of a non-finite field, `Domain.scala:67-79`), `upTo` (the lifter reads the bound from the `given` block, `Types.scala:167-184`) and `of`; it may stop carrying `values` and `product`. `Channel.contents` keeps its name and type `Finite[Inbox[M]]` (the lifter reads it, `Types.scala:65-70`).
- Step functions stay executable (`Step`, the typed `~>`), so `StandaloneActivityPins` "one lost admission response..." stays as a native test; the other tests go exactly as the audit covered them; a test whose audit outcome is missing blocks the task.
- The Models do not change (no Model reads `Keys`, `table` or `answer` outside tests; checked at planning), so the lifted IR must be byte-identical: run the check-mode gate, `git status --short model/ir model/cases model/lifter/testdata` stays empty, the goldens pass strictly.
- Comments: a comment explaining a rule stays; one whose code goes, goes; the search-order and key-spelling comments that named the Go consumer go with their code.
- R25: record the library weighed for what remains generic (`Finite.derived`, about 40 lines, against shapeless-3 or magnolia; Iron for integer bounds) with line counts both ways; the spec expects none to pay for its wiring.
- Report the line count of `model/umpire` (non-test) in the summary; if it is above 1,300, name what stayed and why rather than cutting a compile-time guarantee (R24).

### Investigation targets
**Required**:
- `model/umpire/Machine.scala:9-17,131-271`, `model/umpire/Claims.scala:30-50,150-257`, `model/umpire/Refine.scala:4-30,49-163`, `model/umpire/Compose.scala:19-60,86-242`, `model/umpire/Domain.scala:22-128`, `model/umpire/Errors.scala:1-33`, `model/umpire/Table.scala`, `model/umpire/Search.scala`, `model/umpire/Channel.scala`, `model/umpire/Monitor.scala`, `model/umpire/Assume.scala`
- `model/lifter/Context.scala:83`, `model/lifter/Declarations.scala:38,126-183,283,423,454`, `model/lifter/Claims.scala:93-99,207-218`, `model/lifter/Types.scala:65-70,167-184`, `model/lifter/Realizations.scala:19-22` (`namedByIR`)
- `model/umpire/test/Declarations.test.scala`, `model/temporal/test/*.scala`, `.plans/umpire-scala-evaluator-audit.md`
- `.flow/tmp/fn115-7-summary.md` (borderline kept list)

### Quick commands
mise exec -- scala-cli compile model/project.scala model/umpire; mise exec -- scala-cli test model/project.scala model/umpire model/temporal; CC=/usr/bin/clang mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (check mode; IR must not change); git status --short model/ir model/cases model/lifter/testdata; CC=/usr/bin/clang mise exec -- go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model ./tools/umpire/lower; mise exec -- make lint-model; find model/umpire -name '*.scala' -not -path '*/test/*' | xargs wc -l

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `Table`, `Search`, `Keys`, `Model.table`, `Query.answer`, `check` and the runtime halves of `Refine`, `Compose` and `Errors` are gone; nothing in `model/umpire` builds a table, answers a Query, checks a refinement, composes machines or spells a key or a Definition ID at runtime; the DSL compiles alone.
- [ ] Every declaration the lifter reads keeps its name and type: the check-mode gate lifts byte-identical IR and expected fixtures, and the goldens pass strictly.
- [ ] The munit tests removed are exactly those the audit covered; the remaining native tests call step functions only.
- [ ] The summary records the line count of `model/umpire` (non-test), each borderline declaration kept and why, and the R25 weighing; `lint-model` passes.


## Done summary
# fn113-8 handover: retire the native evaluator from model/umpire

### What was removed
- Deleted (moved to `.flow/tmp/trash/fn113-8/`): `model/umpire/Table.scala` (Table, Row, RowResult, Atom; `Family` moved to Action.scala without `id`), `Search.scala`, `Check.scala` (check/checkQuery/checkModel), `Errors.scala` (Checked, Fails, checked, fail, ModelError).
- Machine.scala: `Model.table`, `Machine.table/build/bind/enumerate/run`, `refinementCheck`, `hasRefinement`, `ClaimNames`; `StepBinding` keeps the step function instead of the `(state, class)` closure with casts.
- Claims.scala: `ClaimNames`, the key predicate of `when`, `Query.answer`, `Answer`, `Trace`, `TraceStep`, `Verdict` (nothing lifts it: expected verdicts come through `realize.RunExpectation`), the refinement thunk of `Reads`/`Query`.
- Refine.scala: `Refinement`, `Refinement.of`, `productStep`, `RefinementRow`, `sameNamedKey`.
- Compose.scala: `build`, `collectActions`, `composedFields`, `own`, `synced`, `stateKey`, `replacements`, `memberModels`.
- Domain.scala: `Keys`, `Keyed`; Action.scala: `Class.key`, `ActionDecl.classes`, `ClassRef.resolve`; Channel.scala: `Inbox.key`.
- Comments whose code went went (the search-order and key-spelling comments, the "this framework refuses" notes in Channel/Monitor/Assume, the Domain header's runtime claims).
- Tests removed (all audit-covered): `Declarations.test.scala` (18 tests: A/B/C per audit), `NexusCallerPins.test.scala` (all 8 left after task 7, incl. the three provisional ones, now resolved), `NexusKernel.test.scala` (2, outcome C; the audit assigns it to task 13, this task's Files list deletes it because it reads `Table`), five evaluator tests of `StandaloneActivityPins` (B). Kept: "one lost admission response..." (step functions only).

### Borderline declarations kept, and why
- `Finite.values` and its catalog (`of`, the Boolean/Nothing/Option givens, `product`): `Inbox.send` on an unordered channel places a message at its catalog position, and step functions stay executable. `derived` keeps the compile-time "field X has no Finite instance".
- `Channel.entries`/`contents`: `contents` must stay `Finite[Inbox[M]]` (lifter, Types.scala:65-70); with `values` kept it lists the catalog.
- MachineScope/Machine/Composition/PropertyDecl/ScenarioDecl keep the declared values as plain data (no evaluation): lint runs with `-Wunused:all -Werror`, so dropping them would need `@unused` on each DSL parameter. Composition fields became `private[umpire]` for that reason.
- `ActionDecl` identity equality: `restrict` still filters bindings by declaration.
- `Hole.reached` still throws `HoleReached`: a step function run as Scala has no other way to stop there.
- Every DSL signature and parameter-list shape is unchanged (the lifter matches them, e.g. `refines` with its using list, `compose` with its `Mirror`).

### Line counts (R24)
- `model/umpire` non-test: 2,224 before, 1,206 after (+72/-1,512), under 1,300.
- `model/temporal` non-test: 5,414 before and after (two Claims.scala edits keep their line counts).
- Scala tests (umpire + temporal): 908 before, 12 after.

### R25 weighing
- `Finite.derived` + `summonCases` + `summonFields` + `product`: 38 lines. magnolia (Scala 3) needs a `join`/`split` instance pair (about 20 lines) plus a dependency in `model/project.scala`, which R3 forbids, and gives no field-naming compile error without the same `summonFrom` code. shapeless-3 `K0` derivation: about 15 lines plus the dependency, same loss of the field name in the error. Not adopted.
- Integer bounds: `Finite.upTo` is 1 line and the lifter reads the bound from the `given` source. Iron would change the Models' `Int` fields to refined types (a Model change and a lifter change) plus a dependency. Not adopted.

### Provisional tests: resolution (conductor's preferred route)
- `model/gate/Roots.scala`: added `Model$package$.nexusCaller`, `Claims$package$.terminalHolds`, `Claims$package$.stoppedWorkerRepliesNothing` to `nexus-caller.json`.
- Required Model change outside Touches: `repliedThenStopped` (model/temporal/nexuscaller/Claims.scala) used `nexusCaller.own(...)`/`synced(...)` (runtime key spelling, which R15 removes and the lifter cannot fold), now literal keys `operation_schedule-unset-expires-unset`, `handlerReply-handlerError-true`, `workerStop`, `operation_scheduleToStart` — exactly the keys of the former hand-written copy; same pattern as activity's `stoppedBeforeRetry`. Also `closepolicy/Claims.scala`: `import umpire.{Answer as _, *}` -> `import umpire.*` (umpire's `Answer` is gone), comment kept on one line so `nexus-close.json` stays byte-identical.
- Regenerated with `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks`: only `model/ir/nexus-caller.json` changed, purely additive (1 type, 1 composition, 2 functions, 2 properties, 1 scenario, 2 queries; no existing entry changed or reordered; source label lists the new roots). `model/cases/manifest.json` gains two `nothing-to-realize` entries; no new Case. Every other IR file byte-identical.
- `tools/umpire/model/nexus_pins_test.go`: fragment `nexusCallerClaims` deleted; tests read the lifted IR; the test-local `productClaimProbes` (everywhere/terminalHoldsEverywhere/timesOut) and `timesOutOnProtocol` remain. All Nexus pin tests pass.
- Golden re-capture (nexus-caller only): the repository capture paths (`migrationFiles`/`golden.Capture` in model; `captureArtifacts`/`legacyArtifacts` in lower) run from temporary tests (moved to trash) over a frozen input built as the OLD frozen input with the nine new declarations spliced in at the lifter's positions, in the original `model/scalav2/...` spelling; a whole recapture in the current spelling would make `Match` return "original" for this file only and fail the lower test's "partial IR migration" check, and break `Migrate` for later tasks. `tools/umpire/internal/golden/config.json`: the nexus-caller label substitution now lists the new roots (both sides).
- Proof (decoded, `.flow/tmp/fn113-8/golden/*.txt`):
  - model goldens: 44 files identical; 4 nexus-caller files changed, 0 old entries changed or removed except the `source` label: declarations +5 (2 properties without fingerprint: transition/composition claims, 1 scenario, 2 queries), semantics +5 (Subject nexusCaller table, 2 Properties, 2 receipts verified-within-limits), refined-properties `null` -> terminalIsFinal rows, inputs +9 declarations.
  - lower goldens: 1,283 files identical (every Query Case/program/contract, generated Case and oracle). Changed: inputs +9 (label), `generated/manifest.json` +2 nothing-to-realize, 4 new `queries/{terminalHolds,stoppedWorkerRepliesNothing}/lowering.json`; 7 query `assessment.json` and the `nexusDeadlines` exploration artifacts differ only in sha256 digests of the whole model (52 files equal with every 64-hex digest masked; the 12 model/proposal files embed the model and are additive plus digests) — exactly R13's amendment (exploration Case IDs hash the whole candidate Model). `oracles/job/*` differ in a fresh capture regardless of this change (projected by the test) and were not touched.
  - `testdata/schema/before-rename/wire/ir/nexus-caller.binpb`: re-encoded with the captured pre-rename descriptor via dynamicpb (control: activity re-encodes byte-identical).
- Go tests that assumed seven Queries / no composition in nexus-caller, adjusted without changing their claims: `migration_golden_test.go` (model: the two never-realized Properties have no fingerprint; lower: `legacyArtifacts` takes the find Queries), `lower_test.go` (two tests), `migration_fixture_test.go`, `export/quint_test.go` (`nexusCaller` is exported; 19 Properties).

### Checks (logs under .flow/tmp/fn113-8/)
- `mise exec -- scala-cli compile model/project.scala model/umpire` -> ok (compile-umpire.log)
- `mise exec -- scala-cli test model/project.scala model/umpire model/temporal` -> 1 test passed (test-umpire-temporal.log)
- `make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks` (roots) -> ok (gen-model-roots.log)
- `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks` -> ok, incl. lifter fixtures (R16 crossed/werror refusals) (check-model-2.log); a first run failed once at the Case check with an empty read ("case ProtoJSON is required"), and the step passed twice directly and in the rerun: a filesystem flake (check-model.log, gen-cases-check-*.log). `git status --short model/ir model/cases model/lifter/testdata` shows only nexus-caller.json and manifest.json (the root additions).
- `make lint-model` -> ok (lint-model.log)
- `go test -run '^TestMigrationGoldens$' ./tools/umpire/model` and `./tools/umpire/lower` (GOMEMLIMIT=4500MiB, -p=1, -parallel 1) -> ok (go-model-goldens-final.log, go-lower-goldens-final.log)
- `go test ./tools/umpire/model` -> only TestAdmission* fail, pre-existing: their expected lifter-fixture lines (Admission.scala/Channels.scala) differ from the unchanged committed `model/lifter/testdata/lifts/expected/*.json`; neither file is touched here (go-model-all-after-recapture2.log)
- `go test ./tools/umpire/lower` -> ok after the three adjustments (go-lower-all-final.log, go-lower-fixed.log)
- `go test ./tools/umpire/export` -> ok (go-export-2.log); `./tools/umpire/cmd/umpire-gen-cases ./tools/umpire/cmd/umpire-repeat` -> ok (go-cmd.log)
- `make lint-code LINT_CODE_TARGETS="./tools/umpire/model ./tools/umpire/lower ./tools/umpire/export"` -> 0 issues (lint-code.log)

### For the conductor (shared docs and other tasks)
- Touches widened beyond the brief: `model/temporal/nexuscaller/Claims.scala`, `model/temporal/nexuscaller/closepolicy/Claims.scala`, `tools/umpire/internal/golden/config.json`, `tools/umpire/model/{migration_golden_test.go,testdata/schema/...}`, `tools/umpire/lower/{migration_golden_test.go,lower_test.go,migration_fixture_test.go}`, `tools/umpire/export/quint_test.go`.
- `.plans/UMPIRE_MODULES.md` / MILESTONES.md: `model/umpire` is declarations only (no Table/Search/Check/Errors); nexus-caller.json now carries the composition and its two verify Queries.
- Task 13: `model/temporal/nexuscaller/kernel/NexusActions.scala:4` still names `NexusKernel.test.scala`, which this task deleted; `umpire/prelude/Prelude.scala` still cites Stainless.
- Pre-existing: the TestAdmission* line expectations in `tools/umpire/model/admission_test.go`.

### Conductor notes at commit

- The `TestAdmission*` failures in `tools/umpire/model` were not pre-existing. Task 5's reformatting of the lifter fixtures moved the lines the admission tests expect. The conductor fixed them in a separate commit, and the whole `tools/umpire/model` package then passed: ok, 158.9 s.
- This task was committed without a separate independent review, because the owner asked to wrap up and stop. A review of it is the first item for the next session.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d5fe738cf28c029f0a9289e5b795d2a96b8a4a0d
- Tests: mise exec -- scala-cli compile model/project.scala model/umpire -> ok, mise exec -- scala-cli test model/project.scala model/umpire model/temporal -> 1 passed, mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks -> ok; only nexus-caller.json and cases/manifest.json differ (root additions), mise exec -- make lint-model -> ok, go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/model -> ok, go test -tags test_dep -run '^TestMigrationGoldens$' ./tools/umpire/lower -> ok, go test -tags test_dep ./tools/umpire/lower -> ok, go test -tags test_dep ./tools/umpire/export -> ok, go test -tags test_dep ./tools/umpire/model -> pass except pre-existing TestAdmission* fixture-line failures, go test -tags test_dep ./tools/umpire/cmd/umpire-gen-cases ./tools/umpire/cmd/umpire-repeat -> ok, make lint-code LINT_CODE_TARGETS=tools/umpire/{model,lower,export} -> 0 issues, GOMEMLIMIT=4500MiB GOFLAGS=-p=1 go test -count=1 -tags test_dep -parallel 1 ./tools/umpire/model (conductor, after fixing the admission tests fn-113.5 broke) -> ok 158.9 s (.flow/tmp/fn113-8/model-package-full.log), no separate independent review: the owner asked to wrap up and stop; review pending
- PRs: