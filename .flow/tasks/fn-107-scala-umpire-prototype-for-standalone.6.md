---
satisfies: [R2, R6, R7, R8]
---
# fn-107-scala-umpire-prototype-for-standalone.6 Lower Scala scenarios and observations through Testpilot

Touches: [model/scalav2/scala/umpire/caseproducer/**, model/scalav2/scala/temporal/nexuscaller/Realization.scala, model/scalav2/scala/temporal/test/CaseBytes.test.scala, model/scalav2/scala/umpire/**, model/scalav2/lifter/**, model/scalav2/ir/**, model/scalav2/goir/testpilot/**, model/scalav2/goir/load.go, model/scalav2/goir/*_test.go, proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/SEMANTICS.md, model/scalav2/gen.sh, model/scalav2/README.md, model/scalav2/run.sh, Makefile, proto/internal/temporal/server/api/testpilot/v1/**, api/testpilot/v1/**, model/scalav2/gen.sh]

## Description
Add the producer adapter from admitted Scala IR scenarios to existing Testpilot Cases. Identify narrowly necessary protocol changes explicitly.

Cases are produced in Go from the IR, in `goir/testpilot`. Scala declares the scenario, observations, learned values and worker script; it does not build Cases. `model/scalav2/scala/umpire/caseproducer` is a Scala-side Case producer that task 16 carried over from the legacy `model/scala` experiment; only `nexuscaller/Realization.scala` and `CaseBytes.test.scala` use it, and nothing lifts it. It is a second producer beside the Go one, which the spec rules out, so this task retires it.

**Size:** M
**Files:** lifter scenario support; goir/testpilot lowering and admission fixtures; the Scala realization declarations, re-expressed as lifted declarations; removal of model/scalav2/scala/umpire/caseproducer and the v2 Testpilot proto jar; Testpilot protocol/generated outputs only when required.

### Approach
- Reuse the current Case/Program/Contract abstraction and producer-shaped declarations. Import relevant existing descriptors; generate source annotations for field identity/projection roles.
- Lower DAG dependencies, typed learned values, scripts, observations, and monitors mechanically. Reject unsupported lowering rather than synthesize feature semantics.
- Inventory every Testpilot gap from the reviewed sketches. Use existing primitives where possible; document and test the smallest schema/admission evolution where necessary.
- Exercise ordinary Prepare with existing fixtures and Cases lowered from the Scala IR; retain existing consumer compatibility and canonical artifact identity.
- Retire the Scala Case producer: delete `scala/umpire/caseproducer`, the producer calls in `Realization.scala`, `CaseBytes.test.scala`, and the `testpilot` jar from `gen.sh`, the Makefile's Scala jar prerequisites and `run.sh`. Byte parity with the existing `nexusCallerTests-*-case.json` fixtures is not required (owner decision, 2026-09-30): a Case lowered from the IR must be admitted by ordinary Prepare and carry the behaviour the Scala declares, not match those bytes. Reuse `model/go/caseproducer` where it fits rather than duplicating it.

### Investigation targets
**Required:** model/scalav2/scala/umpire/caseproducer/Producer.scala; model/scalav2/scala/umpire/caseproducer/Program.scala; model/scalav2/scala/temporal/nexuscaller/Realization.scala; model/go/caseproducer/; model/go/nexuscaller/realization.go and case_test.go; common/testing/testpilot/prepare.go:27; common/testing/testpilot/profile.go:84; proto/internal/temporal/server/api/testpilot/v1/program.proto:131.
**Optional:** common/testing/testpilot/internal/ir/descriptor.go; tests/testcore/testpilot/protobuf_lean_authoring_test.go.

### Quick commands
`make umpire-check-scala`; `make lint-scala`; `mise exec -- go test -tags test_dep ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...`; run existing protocol generation gates and keep `model/scalav2/gen.sh` building only the jars that remain.

## Acceptance
- [ ] An admitted Scala scenario produces a Testpilot Case with typed learned values and independent branches.
- [ ] Valid public/internal descriptor projections preserve presence/validation, while crossed types/IDs/descriptors and cycles reject.
- [ ] Existing Testpilot consumers admit and evaluate after any documented narrow protocol tweak.
- [ ] Generated monitor/script behavior is traceable to Scala and no feature policy is introduced in Go.
- [ ] Cases are built only in Go from the IR, and `model/scalav2/scala/umpire/caseproducer`, its byte-parity test and the v2 Testpilot proto jar are gone. No test requires byte parity with the existing Nexus caller Case fixtures.

- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing with that tree absent.

## Done summary
Scala now declares a realization and builds no Case. `goir/testpilot` lowers each of the seven Nexus caller functional Queries from the lifted IR into a Testpilot Case that ordinary `testpilot.Prepare` admits, the same bytes for the same inputs, and names every declaration Testpilot cannot run yet with the task that owns it. Nothing is committed: the work sits uncommitted for the owner and the task stays `in_progress`. No Testpilot protocol file changed.

### Files

Changed (before-copies under `.flow/tmp/fn-107/task6-before/`):
- `proto/internal/temporal/server/api/modelir/v1/ir.proto`: `Model.realizations` and 36 messages for a realization (379 added lines, none removed). `api/modelir/v1/ir.pb.go` and `ir.go-helpers.pb.go` regenerated by `make protoc`; nothing else under `api/` or `proto/` changed.
- `model/scalav2/lifter/Lift.scala`: realizations are emitted by name against the IR's descriptors (2 lines replaced, 269 added).
- `model/scalav2/goir/load.go`: admission of realizations (612 added lines, none removed).
- `model/scalav2/goir/fixtures_test.go`: the `realizations` fixture joins the admitted list.
- `model/scalav2/scala/temporal/nexuscaller/Realization.scala`: rewritten as declarations.
- `model/scalav2/gen.sh`, `model/scalav2/run.sh`, `Makefile`: the Testpilot jar is gone; `run.sh` lifts `functionalQueries`, `NexusRealization.asyncNexus` and the new fixture.
- `model/scalav2/ir/nexus-caller.json`: now carries 7 Properties, 7 Scenarios, 7 Queries and 1 realization. The other three `ir/*.json` and the five older `expected/*.json` are byte-identical to their before-copies.
- `model/scalav2/SEMANTICS.md` (Realizations section, Admission list, what goir implements), `model/scalav2/README.md`.

Added:
- `model/scalav2/scala/umpire/realize/Realize.scala` (237 lines): the declaration vocabulary.
- `model/scalav2/goir/testpilot/`: `lower.go` (804), `realization.go` (634), `descriptor.go` (281), `produce.go` (749), `localize.go` (211), `lower_test.go` (418), `fixture_test.go` (298).
- `model/scalav2/goir/realization_test.go` (admission).
- `model/scalav2/lifter/testdata/lifts/Realizations.scala.fixture` and `expected/realizations.json`.

Deleted (whole directory and file copied to the before-copies first): `model/scalav2/scala/umpire/caseproducer/` (6 files), `model/scalav2/scala/temporal/test/CaseBytes.test.scala`. The stale build outputs `gen/testpilot-proto.jar` and `gen/testpilot.stamp` (gitignored) were removed.

### How it is built

1. **Scala** (`umpire.realize`): roles, learned values (text or handle), observations, kinds of evidence with the field that keys the operation and a commitment, a correlation, controls, and scripts. A script item is a command, a command with `when` classes, or performances (class to command). A command has `after`, a deadline and `regardless`. A written protobuf message names its type and only the fields it sets.
2. **Lifter**: each constructor becomes the IR message of its name and each argument the field of its parameter's name, checked against the IR's own descriptors, so a declaration the IR has no field for is a lift error at its line. Helper functions and vals of the feature file are followed.
3. **`goir.Validate`**: absent and empty ids, duplicates, dangling references, crossed role and learned-value kinds, a learned value bound twice or never, a class performed twice, crossed correlations, `after` cycles, malformed items, operands and messages. Every problem is reported, each at its Scala line.
4. **`goir/testpilot`**: `NewProducer(model)` admits, builds and checks once; `Lower(query, identity)` returns a standing (`lowered`, `nothing-to-realize`, `no-realization`, `unsupported`), the Case, the gaps, and an inventory of every declaration. It checks what the realization writes and reads against protobuf descriptors, translates it into a `caseproducer.Realization`, and produces the Case.

### The gap in model/go, and what stands in for it

`caseproducer.Produce` cannot be called for an IR Query. Exactly:
1. `umpire.PropertyDecl.Lower` (`model/go/umpire/lower.go:40`) refuses a Property declared over a table's keys, and `Table.alter` is only set by a typed machine. Over keys the alteration is a `Result` with one key replaced, about 25 lines.
2. `umpire.TableSpec` has no state field values, so `Table.FieldValues` is empty for a keyed table and the Contract would lose its state fields.
3. `Table.Claims()` reads typed action declarations, so a keyed table has no Abstraction Claims.
4. `caseproducer` exports only `Produce(*umpire.Query, ...)`; its evidence, clause, projection, program and local-name steps are unexported.
5. `caseproducer.WhenOnPath` reads its keys through the bindings only. The stand-in reads the schedule, which is the same wherever the key is bound.

`goir/testpilot/produce.go` and `localize.go` (960 lines) are therefore `caseproducer`'s unexported steps over a keyed table, taking the same `caseproducer.Realization`. `lower.go` reads the clauses, state fields and claims from the IR. `TestTheStandInAgreesWithTheGenericProducer` feeds one IR-derived realization to both producers: `cp.Produce` over the comparative Go Model's Query and the stand-in over the IR's Query give the same Case, whole, for all seven Queries. Once gaps 1 to 3 close, `Lower` becomes `cp.Produce(keyQuery, realization)` and both files are deleted.

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | An admitted Scala scenario produces a Case with typed learned values and independent branches | pass | `TestALearnedTextIsBoundOnceAndReadByIndependentBranches` (a text slot bound by the start call's `run_id`, two commands each after the start call alone, both reading it under a presence guard, a join after both, `Prepare` admits), `TestALearnedHandleIsBoundOnceAndReadByItsDependents`, `TestLoweredCasesPrepareUnderTheirDerivedProfile` (7) |
| 2 | Valid public and internal descriptor projections keep presence and validation; crossed types, ids, descriptors and cycles reject | pass | `TestAWrittenMessageKeepsItsPresence`, `TestADescriptorARealizationCrossesIsRejectedWhereItIsWritten` (18 cases, each a located `*goir.Error`), `TestARealizationIsAdmittedBeforeItIsLowered` (47 cases), `TestEveryProblemOfARealizationIsReported` |
| 3 | Existing Testpilot consumers admit and evaluate after any protocol tweak | pass, no tweak | `./common/testing/testpilot/...` and `./tests/testcore/testpilot/...` rc 0 before and after; no file under `proto/internal/.../testpilot` or `api/testpilot` changed |
| 4 | Monitor and script behavior is traceable to Scala; no feature policy in Go | pass, with one limit | `TestALoweredCaseCarriesWhatTheScalaDeclares` (instruction ids, rule ids and bounds per Query written from `Claims.scala` and `Realization.scala`), `TestEveryDeclarationIsAccountedForInEveryCase`, `TestAnInventoryThatDoesNotCloseIsAnError`. Limit: authored monitors are not lowered; they are a named gap (task 12) |
| 5 | Cases are built only in Go; the Scala producer, its byte-parity test and the v2 Testpilot jar are gone; no test requires fixture byte parity | pass, with one leftover outside the Touches | `make umpire-check-scala` rc 0 with no `gen/testpilot-proto.jar` on disk; see "Outside the Touches" 1 |
| 6 | Legacy `model/scala` unchanged; v2 generation, lifting, tests and lint pass | pass | no file under `model/scala` touched by this task; `make umpire-check-scala`, `make lint-scala` rc 0; `goir/isolation_test.go` passes |

### What cannot be lowered, and where it is named

`TestWhatTestpilotCannotRunIsNamedWithItsOwner` lowers the activity specimen's held race (`pauseRace`, written from `specimens/activity.md` E8 and E10 over the `staleAdmission` fixture). The result has no Case and seven located entries:

| Construct | Owner |
|---|---|
| authored monitor `atMostOneActiveAttempt`, `terminalFinality` | fn-107.12 |
| durable-commit observation | fn-107.10 |
| hold-delivery control, and its hold and release commands | fn-107.10 |
| activity activation (the activity's worker script) | fn-107.13 |

`TestEveryQueryHasAStanding` pins the three other Models: `activity` 3 verify and 9 find, `activity-system` 61 and 23, `nexus-close` 84 and 72. Every verify is `nothing-to-realize` and every find is `no-realization`.

### Decisions that differ from the task text

1. **The real activity and close/reset Models lower nothing.** They declare no realization, and their Scala (`scala/temporal/standaloneactivity`, `closepolicy`) is outside the Touches, so I could not author one. The gap inventory runs on a lifter fixture instead. A realization for the activity Model is task 9 or 13's to write.
2. **A realization is producer-shaped, not one DAG per Scenario.** One realization serves every find Query of its machine: the path selects among its commands. `after` gives the dependency DAG.
3. **A command that reads a learned text is lowered with a presence guard.** Testpilot refuses a slot read without one (`internal/ir/expression.go:335`). A command that is `regardless` and reads a learned value is an admission error.
4. **The Nexus caller Cases keep their sequential controller and learn no run id.** Changing a live Case's behaviour could not be checked without a cluster. The typed learned value and the independent branches are proven on the `learnedRun` fixture.
5. **Descriptor checks are in `goir/testpilot`, not `goir.Validate`.** `goir` stays free of the Temporal API descriptors. Both run before any Case exists.
6. **A gap stops the whole Case.** Nothing is lowered around an unsupported declaration.
7. **A path step a party takes and no script performs is an error** (`TestAStepNoScriptPerformsIsRefused`). `caseproducer` has no such check.
8. **The comparison with the generic producer skips, with a message, if the Scala Model's Behavior Fingerprint moves away from the comparative Go Model's.** It runs today (7 of 7).
9. **The vocabulary has no attempt, delivery or causal correlation key.** The specimen's E8 names them; nothing can lower them before task 10.
10. **Path selectors checked: `field`, `field[*]`, `oneof<member>`.** A map key selector or a trailing `?` is a located error.

### Test-first record

- Admission: `TestARealizationIsAdmittedBeforeItIsLowered` failed in 46 subtests at `realization_test.go:226` ("An error is expected but got nil") before `load.go` changed (`task6-logs/red-01-admission.log`).
- Lowering: against a stub producer, 22 tests and subtests failed (`red-02-lower.log`), for example `TestLoweredCasesPrepareUnderTheirDerivedProfile/syncCompletion` at `lower_test.go:46`.
- Fixture: `TestALearnedTextIsBoundOnceAndReadByIndependentBranches` failed at `fixture_test.go:104` ("reference or path read requires an explicit presence guard") and `TestEveryDeclarationIsAccountedForInEveryCase` at `:252` (`green-03-fixture.log`). The first led to decision 3, the second to keying a performance's inventory entry by its class.
- Never red, written after the code: the gap, unperformed-step, no-witness, inventory self-check, lift and workflow-type tests, and the `regardless` admission case. Each is covered by a mutant.
- Mutation, in the foreground with restore (`task6-logs/mutants.log`): 24 mutants. 21 failed a test at once, one did not build, and two survived (a lift with no rule, the Case-scoped name). I added `TestALiftCarriesTheHistoryKindsOfThePathOrIsLeftOut` and the workflow-type assertions; both now fail.

### Incident

Formatting three Scala files by path made scala-cli take the first path as its root, miss `.scalafmt.conf`, and reformat `Lift.scala` and my two new files with scalafmt's defaults. I restored `Lift.scala` from its before-copy and reapplied my edits: its diff against the before-copy is 2 replaced lines and 269 added. My restore script then truncated my own two new files (`Realize.scala`, `nexuscaller/Realization.scala`); I rewrote both from this session and regenerated the IR. No other file was touched: the three files are the only `*.scala` newer than the task start, and all gates below ran after the repair.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (baseline, pre-edit) | 0 |
| `... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (baseline, pre-edit) | 0 |
| `GOFLAGS=-tags=test_dep mise exec -- make protoc` | 0 |
| `GOFLAGS=-tags=test_dep make umpire-gen-scala` | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (goir/testpilot: 72 passing tests and subtests; goir: 5 Lean-dump skips) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/...` | 0 |
| `GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot" GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

Skipped: `make umpire-check-testpilot-protocol` and `make umpire-check-testpilot-authoring` both run `lake`, which is not installed. No protocol file changed. No live-cluster test was run. No `flowctl gate` receipt was attempted: the tree is dirty outside the ignore set.

### Outside the Touches

1. **`model/scalav2/scala/project.scala` still names the retired jar** (comment lines 5 to 6 and `//> using jar ../gen/testpilot-proto.jar` on line 13). scala-cli ignores a jar that is not there, which I checked on a scratch project and which the gates confirm, so the build is unaffected. Removing those lines is a one-line follow-up for the owner. The two `protobuf-java` dependencies stay: `Parity.test.scala` and `Lean.test.scala` take gson from them.
2. **`model/scalav2/specimens/README.md`** cites `Realization.scala:173-205` and `:59-78`, and a 291-line realization. Those lines moved.
3. **`model/go/umpire` and `model/go/caseproducer`**: the five gaps above.

### Left undone

- No Case was run or evaluated against a server, and `Evaluate` was not exercised on a lowered Case.
- A Case is over one operation, as in `caseproducer`.
- A machine with two realizations is refused by `Lower`; it takes no realization name.
- A repeated protobuf field cannot be written out; none was needed.

### Review round 1

The Codex review returned NEEDS_WORK with four P2 findings and one P3. All five are valid and fixed. Each code fix was red before it. The sections above describe round 0; where they speak of `produce.go`, `localize.go`, the stand-in or its comparison test, this round replaces them.

1. **P2, the forked producer (valid).** `goir/testpilot` now calls ordinary `caseproducer.Produce`. `produce.go` and `localize.go` are deleted. The package is 1,869 non-test lines, down from 2,679; with tests it is 2,958, down from 3,395 (tests grew from 716 to 1,089).
   - `model/go/umpire/lower.go` (+59, -6): `PropertyDecl.Lower` lowers a Property over a table's keys the way it lowers a typed one. A predicate that fails on any step it is asked about, a table row or a row with a value changed, stops the lowering with the predicate's own error, wrapped as `property <name>: ...` with its type kept. A Property of a composed table is refused rather than read over whole composed states.
   - `model/go/umpire/table.go` (+25): `TableSpec.FieldValues` and `TableSpec.Claims`, checked against the table's states and action classes, returned by `Table.FieldValues` and `Table.Claims` for a keyed table.
   - `model/go/caseproducer/program.go` (+5, -1): `WhenOnPath` is also carried for a class of the Model that the path takes and no binding performs. The existing condition is unchanged and comes first.
   - Typed constructors, Definition IDs, fingerprints and the fixture byte-identity test are unchanged: `./model/go/...` passes, including `TestCasesAreByteIdenticalToTheFixtures`.
   - **One existing test changed, on purpose.** `TestKeyLevelPropertyIsNotLowered` in `keyclaims_test.go` pinned the refusal this finding removes, so it is deleted (12 lines, nothing else in that file). `keylower_test.go` replaces it. The message "a key-level Property is searched and verified, never realized" no longer exists; a key-level transition claim now gets the typed transition message.
   - Red first (`task6-logs/r1-red-01-modelgo.log`): 5 umpire tests and 2 caseproducer tests, 21 failures with subtests. Proving tests: `TestAKeyLevelPropertyLowersToItsClauses`, `TestAKeyLevelPropertyThatCannotBeReadIsNotLowered`, `TestAKeyLevelPropertyIsRefusedAsATypedOneIs`, `TestAPropertyOfAComposedTableIsNotLowered`, `TestAKeyedTableCarriesItsFieldValuesAndClaims`, `TestAKeyLevelQueryProducesTheCaseOfTheTypedOne` (7 Queries, whole Case), `TestANodeIsPlacedForAClassOfThePathNoBindingPerforms`.
   - The comparison test is now `TestALoweredCaseIsTheComparativeGoModelsCase`, with no skip: the IR-lowered Case equals the Case `cp.Produce` gives the comparative Go Model with `nexuscaller.AsyncNexus`, for all seven Queries. They differ in one thing, which the test states and normalizes: the provenance source locations name `Claims.scala` with provenance `scala-model` where the Go Model names its Lean counterpart.
   - `machine.go` is outside the Touches, so `goir.Build` does not fill the two new spec fields. `goir/testpilot` builds the producer's table from the machine's keys, hole rows as unknown pairs, with field values and claims read from the IR (`TestAHoleRowIsAnUnknownPairOfTheProducersTable`).
2. **P2, a gap masked an error (valid).** `Producer.check` reads the realization and the path whole and emits nothing: descriptors of every observation, kind of evidence and command, the pinned classes, unperformed party steps, the witness, and the Property's clauses. It collects every problem. `standingOf` is the one place the standing is decided: errors, then gaps, then what the Query is.
   - Red first (`r1-red-02-precedence.log`): `TestAGapHidesNoError` (4 cases returned `unsupported`) and `TestAStandingIsAnErrorThenAGapThenWhatTheQueryIs` (8 cases).
   - Also `TestEveryDescriptorARealizationCrossesIsReported` and `TestAPropertyThatReachesAHoleIsNotLowered`. The second was red against my own first version, which wrapped the error and lost the hole's type.
   - The fixture's durable-commit evidence read a path that is no repeated message, which the old order hid. It now reads `ListActivityExecutions.executions`, with a comment that no public read reports the commit.
   - Evidence rules are resolved by `caseproducer` when it produces, so an unknown or ambiguous evidence kind on the path is still found only for a realization with no gap.
3. **P2, admission of the envelope (valid).** `Realization.id` is required and unique, as the name is. A realization with no name, no id or an undeclared machine is read whole; only the class checks, which need the machine, are skipped.
   - Red first (`r1-red-04-envelope.log`): `TestARealizationWithABrokenEnvelopeIsStillReadWhole` (4 cases) and `TestARealizationIsNamedByItsIdAndItsName`.
4. **P2, the inventory (valid).** The inventory walks the `Realization` and `Correlation` descriptors. Each field has an accounting rule, and a field with none fails every lowering. It now records the id and name (as `names`), machine, producer, producer version, cleanup and each correlation field including the six window bounds, and checks each against the provenance, Contract or Program field that carries it. The reverse check walks the Case's own descriptors: every populated field of the Case, provenance, Program, Contract and correlated Contract is owned by a declaration or listed in `derived` with what it comes from.
   - Red first (`r1-red-03-inventory.log`): `TestTheInventoryIsEveryDeclarationOfTheRealization`, 7 subtests. Its expected set is read off the realization's message tree, with no list in the test.
   - Also `TestEveryFieldOfARealizationHasAPlaceInTheInventory` (fails when the IR gains a field), `TestTheInventoryAgreesWithTheCase`, `TestAnInventoryThatDoesNotCloseIsAnError` (19 cases across Program, Contract and provenance), `TestEveryBoundOfTheWindowIsInTheProjectionFingerprint` (6).
   - Limit: a window bound reaches the Case only inside the projection fingerprint, so the inventory checks that the fingerprint is there and the test checks that each bound changes it. A control has no part of a Case; a realization that declares one is never lowered.
   - `Entry.Kind` is now the realization's field name (`roles`, `learned`, `observations`, `evidence`, `scripts`, `correlation`, `realization`, `command`) and `Entry.As` names Case parts by path.
5. **P3, the specimens README (valid).** The reference is now `Realization.scala:286-306`, and the size row reads 496 / 400, remeasured 2026-10-01, with the earlier 291 / 202 kept for comparison. No test covers a documentation line.

Mutation after the fixes, in the foreground with restore (`task6-logs/r1-mutants.log`): 28 mutants across the four packages. 24 failed at once, one did not build and failed when rewritten, and three survived (only the first descriptor error reported, a missing target definition accepted, hole rows dropped from the producer's table). I added a test for each and all 28 now fail.

Still outside the Touches:
- `model/scalav2/goir/machine.go` and `claims.go` do not pass `FieldValues` and `Claims` to `NewTable`. If they did, `goir/testpilot` would not rebuild the table.
- `model/scalav2/specimens/nexus.md` lines 511 to 512 cite the same old `Realization.scala` ranges.
- `umpire.Table.Coverage` still reads typed claims only, so a keyed table reports no class-member targets.

Gates after the fixes:

| Command | rc |
|---|---|
| `GOFLAGS=-tags=test_dep make umpire-gen-scala` | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (goir/testpilot: 124 passing tests and subtests, no skip) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/...` | 0 |
| `GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot ./model/go/umpire ./model/go/caseproducer" GOLANGCI_LINT_FIX=false` (0 issues; the first run reported 2 in `umpire/lower.go`, both fixed) | 0 |

The two Lean protocol gates were skipped again (`lake`). Files this round: `model/go/umpire/{lower.go,table.go,keyclaims_test.go}`, new `model/go/umpire/keylower_test.go` and `model/go/caseproducer/keyed_test.go`, `model/go/caseproducer/program.go`, `goir/load.go`, `goir/realization_test.go`, `goir/testpilot/` (two files deleted, `inventory_test.go` added), `lifter/testdata/lifts/Realizations.scala.fixture` and `expected/realizations.json`, `SEMANTICS.md`, `specimens/README.md`. Before-copies for the widened Touches are under `task6-before/model/go/` and `task6-before/model/scalav2/specimens/`; the round-0 state of `goir/testpilot` and `load.go` is under `task6-before/r1/`.

### Review round 2

The re-review left three P2 findings and one P3, and asked me to settle my own limit (iii). All are fixed. The round-1 limits (i) and (iii) no longer hold.

1. **P2, lowering read another step than the check (valid).** `goir/testpilot` no longer builds a step value or a Property predicate at all. `goir.Realizer` (new, in `claims.go`) binds the Model as `Check` does, and `Realizer.Find` returns the Query the checker answers: the same `PropertyDecl`, whose predicate reads the checker's own step record, over the same table. `caseproducer.Produce` lowers that. The one constructor is `binding.keyedStep`, which already carried `res.Because`; class keys now come from `binding.classKey` too, through `Realizer.ClassKey`.
   - Red first (`task6-logs/r2-red-01.log`): `TestAPropertyIsLoweredOverTheStepItIsCheckedOn` at `fixture_test.go:342`, "the predicate holds on no step of this machine at `push`", while `Check` found the Query. The fixture's `door` machine has a Scala Property that reads `s.because`. The test also rewrites the explanation in the Property and requires `Check` to answer `not-found` and lowering to refuse.
   - Field by field, the step records now cannot differ: there is one.
2. **P2, a gap hid the producer's evidence errors (valid).** `caseproducer.Preflight(q, identity, r)` runs `production.decide`, which `produce` itself calls first: witness, evidence rules, alternatives, clauses, projection and Program assembly, with no Case written. `check` runs it before gaps are classified.
   - Red first: `TestAGapHidesNoError/a_fact_of_the_path_no_kind_of_evidence_records` returned `unsupported`. It now reports `statusPaused: evidence.kind-unknown`.
   - Also `TestPreflightRefusesWhatProduceRefuses` in `model/go/caseproducer`.
   - The preflight runs only for a realization with no descriptor error, a witness and a Property that lowers; each of those is already an error, and a realization that is not all there would add a spurious one (`TestEveryDescriptorARealizationCrossesIsReported` pins that).
   - **It found two real errors in my own fixture.** The race realization declared one of the four kinds of evidence its path records, and its three-step path ends in an admission whose two results (message consumed, message kept for redelivery) record the same facts, which the producer refuses as `evidence.kind-ambiguous`. The fixture now declares all four kinds and pins the path up to the pause. The ambiguity is a finding for task 10: the specimen's race needs a delivery key before its admission step can be realized.
3. **P2, an admitted Model could panic (valid).** `goir.Validate` rejects an example on an action with other than one input, an example with no value, and one that is no member of the input's type. The read in `claims.go` checks the arity again.
   - Red first: `TestAnExampleIsOfAClassOfAOneInputAction`, 4 cases, "An error is expected but got nil".
   - Random-change tests (`random_test.go`): 2,000 changed Models per fixture through `Validate`, 2,000 changed realizations per fixture through `Lower`, and 40 per fixture through the whole of `NewProducer` and `Lower`, each with one to three schema-valid changes to the realizations and actions, fixed seed. None panics. With the two example guards removed, the admission test panics, so the tests can fail.
   - What I read for unchecked indexes, map lookups and nil dereferences on IR data, and what changed:
     - `lower.go`: a Query naming an undeclared Property or Scenario is a located error; `gaps` no longer dereferences a machine that was not interpreted; `performed` reads classes through goir.
     - `realization.go`: an observation whose message is unknown no longer makes a later read dereference it (round 1); a poll of evidence with no read source gets the method error.
     - `claims.go` (new code): state field count against the record, example arity and value, case field count in a class's spelling.
     - `load.go` realization admission: class keys are computed only after the class is admitted; a missing command, script or correlation is read through nil-safe getters; cycles over undeclared ids.
     - `descriptor.go`: no index on IR data.
   - The random tests change realizations and actions only. Changing machines, types and functions exercises interpreter code outside this task's Touches; I did not try it.
4. **P3 (valid).** The `TableSpec` comment now says a keyed table serves claims declared over its keys and takes its state fields and Abstraction Claims from the spec.
5. **Limit (iii), fixed by moving, not left.** The rebuild did duplicate `binding.view`. `claims.go` now has one `spec` for a machine's check table; a realizing binding adds the state fields and Abstraction Claims there (`claimed`). `goir/testpilot`'s table, field, claim and predicate code is deleted. `machine.go` did not need to change. `TestTheRealizersTableIsTheCheckTableWithFieldsAndClaims` checks rows, hole rows and fingerprint against the check table and pins the Nexus field values and one claim.

`goir/testpilot` is now 1,657 non-test lines (1,869 after round 1, 2,679 before it) and 1,369 test lines. Changes this round: `goir/claims.go` +141, -3; `goir/load.go` +23; `caseproducer/producer.go` +38, -1 (the body of `produce` moved into `decide`); `umpire/table.go` comment only.

Mutation, foreground with restore (`task6-logs/r2-mutants.log`): 9 mutants. Six failed at once, two did not build and failed when rewritten, and one survived (the preflight run on a realization with descriptor errors); I added the assertion that pins it and it now fails.

Gates:

| Command | rc |
|---|---|
| `GOFLAGS=-tags=test_dep make umpire-gen-scala` | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (goir/testpilot: 134 passing tests and subtests, no skip, about 50 s, of which the random tests are about 35) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/...` | 0 |
| `GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot ./model/go/umpire ./model/go/caseproducer" GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

The two Lean protocol gates were skipped (`lake`). Round-1 state of the files this round changed is under `task6-before/r2/`.

Still open, all stated before: no Case has run against a server; `specimens/nexus.md` lines 511 to 512 cite old line ranges; `umpire.Table.Coverage` reads typed claims only.

### Review round 3

Codex pass three confirmed the round-two fixes and the shortened race fixture, and left one P2 and one P3. Both are valid and fixed. No mutation run this round, as instructed.

1. **P2, the Realizer bound Queries the Model does not declare (valid).** `Realizer.Find` now takes a `ClaimKey`, the key `Check` gives a Query's receipt (family, machine or composition, name), and resolves the Query from the bound Model itself. A key the Model declares no Query under is a located error; a right name under another machine or family is one too. `NewRealizer` runs `Validate` and returns an error for a Model admission rejects.
   - Red first (`task6-logs/r3-red-01.log`): a temporary test gave the old `Find` a declared Query cloned and renamed and got a bound Query back ("An error is expected but got nil"). The old signature no longer exists, so the permanent test is `TestARealizerGivesOnlyTheQueriesOfAnAdmittedModel`: an unadmitted Model is refused, every Query receipt of the Nexus Model resolves under its own key with the receipt's position and limits, and four keys the Model does not declare are refused with a located error, by `Find` and by `Declared`.
   - The seam is `NewRealizer`, `Declared(key)`, `Find(key)`, `Realizations()`, `ClassKey(class)` and `Machine(name)`: what `goir/testpilot` reads, and nothing else of the binding.
   - `goir/testpilot` no longer holds the Model. Its lookups of Queries, Properties, Scenarios, monitors and actions are removed: a Query comes from the Realizer by its receipt's key, a machine's monitors and the party of each class from the interpreted `Machine`. `NewProducer` no longer calls `Validate` itself.
   - `claims.go` +44, -7. `goir/testpilot` is 1,645 non-test lines.
2. **P3, the random tests under `-short` (valid).** `go test -short` tries a twentieth of the changed Models (100, 100 and 2), with the same fixed seed, so the same first changes. A failure still prints the seed, the iteration and the changes. The package takes about 16 s under `-short` and about 49 s otherwise.

Gates:

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/...` (21 packages ok) | 0 |
| `GOFLAGS=-tags=test_dep make umpire-check-scala` | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/...` | 0 |
| `GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot ./model/go/umpire ./model/go/caseproducer" GOLANGCI_LINT_FIX=false` (0 issues) | 0 |

No Scala file changed this round, so `make lint-scala` and `make umpire-gen-scala` were not rerun. The round-2 state of the files this round changed is under `task6-before/r3/`.

Review round 4 (conductor): SHIP with one P3, fixed by the conductor: an unknown Query returns the Realizer's located error (`goir/testpilot/lower.go`, pinned whole in `lower_test.go`). The conductor also removed the retired jar directive from `model/scalav2/scala/project.scala`.

Carried to later tasks: no realization exists for the activity or close/reset Models (tasks 9, 13); the activity race's admission step needs a delivery key before it can be realized (task 10); authored monitors are not lowered (task 12); no lowered Case has run against a server or been evaluated; `specimens/nexus.md` lines 511-512 cite old `Realization.scala` line ranges; `umpire.Table.Coverage` reads typed claims only. The two Testpilot protocol gates need `lake` and were not run; no protocol file changed.

The work is uncommitted; the owner makes the commits. The task diff and the four review outputs are under `.flow/tmp/fn-107/task6/`. For the producer reuse the conductor widened the Touches to `model/go/umpire/**` and `model/go/caseproducer/**`.

stage: implement - ran (worker subagent, session model claude-opus-5-5; three fix rounds)
stage: impl-review - ran (codex:gpt-5.6-sol:high, one session 01a0f6d1-7b41-7c91-8042-03f41db40f50 over the uncommitted task diff; rounds 1-3 NEEDS_WORK with 4, 3 and 1 introduced P2 findings, all fixed; round 4 SHIP with one P3, fixed)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... rc=0; ./common/testing/testpilot/... ./tests/testcore/testpilot/... rc=0, pre-edit), GOFLAGS=-tags=test_dep mise exec -- make protoc (rc=0), GOFLAGS=-tags=test_dep make umpire-gen-scala (rc=0), GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0), GOFLAGS=-tags=test_dep make lint-scala (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... (rc=0), GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot" GOLANGCI_LINT_FIX=false (rc=0, 0 issues), SKIPPED (needs Lean lake, not installed): make umpire-check-testpilot-protocol; make umpire-check-testpilot-authoring, no flowctl gate receipt attempted: tree dirty outside the ignore set, Review round 1: GOFLAGS=-tags=test_dep make umpire-gen-scala (rc=0), Review round 1: GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0), Review round 1: GOFLAGS=-tags=test_dep make lint-scala (rc=0), Review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0), Review round 1: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), Review round 1: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/... (rc=0), Review round 1: GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot ./model/go/umpire ./model/go/caseproducer" GOLANGCI_LINT_FIX=false (rc=0, 0 issues), Review round 1: SKIPPED (needs Lean lake): make umpire-check-testpilot-protocol; make umpire-check-testpilot-authoring, Review round 2: GOFLAGS=-tags=test_dep make umpire-gen-scala (rc=0), Review round 2: GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0), Review round 2: GOFLAGS=-tags=test_dep make lint-scala (rc=0), Review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0), Review round 2: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), Review round 2: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/... (rc=0), Review round 2: GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot ./model/go/umpire ./model/go/caseproducer" GOLANGCI_LINT_FIX=false (rc=0, 0 issues), Review round 2: SKIPPED (needs Lean lake): make umpire-check-testpilot-protocol; make umpire-check-testpilot-authoring, Review round 3: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc=0), Review round 3: GOFLAGS=-tags=test_dep make umpire-check-scala (rc=0), Review round 3: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/... (rc=0), Review round 3: GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS="./model/scalav2/goir ./model/scalav2/goir/testpilot ./model/go/umpire ./model/go/caseproducer" GOLANGCI_LINT_FIX=false (rc=0, 0 issues), conductor final: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... ./common/testing/testpilot/... ./tests/testcore/testpilot/... (rc 0, before the P3 fix), conductor final: go test -short ./model/scalav2/goir/testpilot/ (rc 0, after the P3 fix), conductor final: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0), conductor final: make lint-scala (rc 0, 0 [error] lines), conductor final: make lint-code over goir, goir/testpilot, model/go/umpire, model/go/caseproducer, GOLANGCI_LINT_FIX=false (0 issues), skipped: make umpire-check-testpilot-protocol, make umpire-check-testpilot-authoring (need lake), codex impl-review rounds: .flow/tmp/fn-107/task6/t6-r1..r4.md (final SHIP)
- PRs: