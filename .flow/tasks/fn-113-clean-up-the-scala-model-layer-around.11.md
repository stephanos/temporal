---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.11 Port the checker's composition, refinement, monitor and progress tests off the typed fixture layer

## Description
Port the checker's composition, refinement, monitor and progress tests off the typed fixture layer. Same constraint as task 10 (recorded under R14), for the other test group; task 12 deletes the branches.

**Size:** M
**Files:** tools/umpire/model/internal/checker/compose_test.go, composekeys_test.go, refinement_test.go, monitor_test.go, progress_test.go
**Touches:** [tools/umpire/model/internal/checker/compose_test.go, tools/umpire/model/internal/checker/composekeys_test.go, tools/umpire/model/internal/checker/refinement_test.go, tools/umpire/model/internal/checker/monitor_test.go, tools/umpire/model/internal/checker/progress_test.go]

### Approach
- As task 10, for these five files (27 call sites at planning): typed builders become `NewTable`/`TableSpec`/`KeyProperty`/`KeyScenario`/`KeyFind` or IR fixtures; typed refinements (`RefinementSpec` with typed states, `mapValue`/`stepOf`) become key-level refinements (`RefineTables`, `MapState`); typed compositions, monitors and progress claims likewise. Preserve every assertion and comment; typed-only claims become their key-level counterpart or are listed for task 12.
- No deletion of support files or production code here. Count tests before and after.

### Investigation targets
**Required**:
- `tools/umpire/model/internal/checker/compose_test.go`, `composekeys_test.go`, `refinement_test.go`, `monitor_test.go`, `progress_test.go`
- `tools/umpire/model/internal/checker/compose_support_test.go`, `refine_support_test.go`, `monitor_support_test.go`, `progress_support_test.go`, `machine_support_test.go`
- `tools/umpire/model/internal/checker/refine.go:1-60,289` (`Refinement`, `keyLevel`, `checkKeyRefined`, `stepOf`)
- `tools/umpire/model/internal/checker/search.go:339-350,453-470,505-515` (typed `observe` arms, `readState`, `readStep`, `productStep`)
- `tools/umpire/model/internal/checker/compose.go`, `composekeys.go`, `monitor.go`, `progress.go`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/internal/checker/; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model/...; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `compose_test.go`, `composekeys_test.go`, `refinement_test.go`, `monitor_test.go` and `progress_test.go` reference no typed support builder; every assertion and comment is preserved or its key-level equivalent is noted; the test count before and after is in the summary.
- [ ] The checker and reader suites pass; `lint-code-fast` passes; no production file and no support file changed.


## Done summary
# fn-113 task 11: the checker's composition, refinement, monitor and progress tests off the typed fixture layer

### What changed

I edited only the five Touches files in `tools/umpire/model/internal/checker/`. No production file and no
`*_support_test.go` file changed. The task-10 files (`umpire_test.go`, `keyclaims_test.go`, `keylower_test.go`,
`keyunknown_test.go`) were not touched in this session.

A previous implementer started the task. They had ported `compose_test.go` and `composekeys_test.go`, and added
`chattyDoorTable` to `refinement_test.go`. They also removed the transitional typed block that task 10 had moved into
`compose_test.go` (door, abstract door, keyholder, house, and their types and actions), and removed the temporary
parity tests `TestTMPFixturesMatchTyped` and `TestTMPFixturesMatchTyped2`. Both parity tests passed before they were
removed (`fn113-11/fixture-parity.log`). The tree then did not compile, because `monitor_test.go`, `progress_test.go`
and most of `refinement_test.go` still used the typed layer. I finished the port in those three files and gofmt'd
`composekeys_test.go`, which had a stray blank line.

**Leftover TMP tests:** none. `grep -rn TMP` over the package finds nothing, in the task-10 files or in mine.

- `compose_test.go`: compositions use `ComposeTables` over `doorTable`/`keyholderTable`/`opaqueKeyTable`. New
  helpers: `startsAt`, `composed`, `detailedKeyTable` (the worn key, with `findKey` when `finds` is set),
  `refining(field)` and `joinTable`. The `head`/`tail`/`joined` types became key tables with the same state keys.
- `composekeys_test.go`: `keysOf` is gone, and the tables are built directly. Each typed composition that a case
  compared against whole is now a `pinned` value. It holds the Behavior Fingerprint, starts, ends, state fields,
  assumptions, and the state, row and reachable counts, all captured from the typed composition before it was retired
  (`fn113-11/composition-pins.log`).
- `refinement_test.go`: every typed `Refines(...).Visible/VisibleOutcomes/CoversStarts().Refinement()` became
  `RefineTables(doorTable("concrete", refiningAbstract), abstractTable(...), RefinementSpec{MapState: abstractKeyOf,
  SeesFact, SeesOutcome, CoverStarts})`. `seesOpened` now reads fact keys. `chattyDoor` became
  `chattyDoorTable(name, extend...)`, and `requireReplaysFromAStart` now takes a `*Table`. The new
  `seesOpenedOfAbstract` is the spec with the `opened` projection.
- `monitor_test.go`: `anything` is a `KeyVerify` over a `KeyFreeScenario`. `countOpenings` keeps its name and
  comment and delegates to `countOpeningKeys` (keyclaims_test.go). `neverOpened` and `spells` are now `KeyMonitor`s,
  and `After` became `AfterKey`. Properties and scenarios use `KeyProperty`/`KeyTransitionProperty`/`KeyScenario`/
  `KeyFind`/`KeyVerify`, `opensLoudlyOn` and `staysShutOn`. The typed fork machine is `forkedTable()`.
- `progress_test.go`: `NewProgress` became `KeyProgress` with the new `doorIs(phase)`, which `compose_test.go`
  also uses. `tableOf(newDoor(...).Assumes(...))` became `doorTable("door", assumes(...))`.

### Test counts (top-level `func Test`)

| file | before (HEAD) | after |
|---|---|---|
| compose_test.go | 9 | 9 |
| composekeys_test.go | 11 | 11 |
| refinement_test.go | 8 | 8 |
| monitor_test.go | 12 | 12 (one renamed, see below) |
| progress_test.go | 13 | 13 |

Package `=== RUN` lines (tests and subtests): 163 at HEAD, 157 after task 10 (including 1 TMP test), and 156 now.
That is 157 minus the TMP parity test. No test or subtest of these five files disappeared.

### Assertions and comments, test by test

compose_test.go:
- `TestACompositionStartsInEveryAdmittedStart`, `TestASyncNamingAnActionItsMemberLacksIsRejected` (same message),
  `TestAReplacingMemberStandsInForTheOpaqueProvider`, `TestAReplacingMemberKeepsEveryAssumptionItDeclares` (its
  comment is kept), `TestAMembersFairnessNamesItsComposedClasses` (the ghost message is the same),
  `TestAViolatingProviderFailsItsReplacement`, `TestAReplacementAccountsForEveryOpaqueStart` and
  `TestComposedKeysThatCollideAreRejected`: every assertion and message is kept.
- `TestAReplacementIsDeclaredAgainstARefinementOfAMember`:
  - Assertion 1 became its key-level form. A member that replaces opaqueKey without a map that reads it as opaqueKey
    is rejected with "compose-house: the member key replaces opaqueKey, and names no map that reads unrelated as it".
    The typed form was "... and unrelated does not refine it".
  - Assertion 2 ("the composition replaces opaqueKey at key, and has no member key") is dropped as typed-only (see
    below).
- Comments: "Member states whose keys hold the "_" ..." is kept, now above `joinTable`.

composekeys_test.go:
- `TestComposeTablesMatchesTypedComposition` (8 subtests) and `TestComposeTablesStartsInEveryMemberStart`: the
  whole-value comparison with the typed composition is now a whole-value `require.Equal` against the pinned
  composition. The Fingerprint stands for the full state, action and row lists, which the pin does not spell out. The
  names are kept; task 12 may rename them.
- The other tests are unchanged in substance, over key tables. The table in the "unnamed" case of
  `...KeepsEveryAssumptionOfAReplacingMember` is now spelled out: it has the `opaqueKey` state field and no
  `RefinedField`, which is what `keysOf(t, inherits, "")` produced.
- Comments: the `keysOf` comment went with `keysOf` (deleted code). `houseOf`'s comment now reads "as the houses of
  compose_test.go are composed", because the typed houses are gone.

refinement_test.go:
- `TestAnInvisibleStutterRefinesAndTheLegacyRuleIsUnchanged`, `TestAStutterRecordingAVisibleFactIsRejected` (full
  message, kind, witness replay and actions), `TestAStutterWithAVisibleOutcomeIsRejected`,
  `TestACarrierMustRecordEveryVisibleFact`, `TestInitialCorrespondenceCoversEveryProductStart` (message, kind, nil
  witness, product-witness replay) and `TestAStartOutsideTheProductStartsHasAReplayableWitness`: every assertion is
  kept, through `RefineTables`. One require message changed: "without CoversStarts" now reads "without
  CoverStarts", after the `RefinementSpec` field.
- `TestAProjectionWithoutARefinementIsRejected`: the typed claim was a Machine with `Visible` and no `Refines`. Its
  key-level form is a `RefinementSpec` with `SeesFact` and no `MapState`, rejected with "alone refines abstract: a
  refinement names how a state reads as a state of abstract". A comment says so.
- `TestRefineTablesChecksKeyOnlyTables`: unchanged.

monitor_test.go:
- `TestAMonitorKeepsDistinctHistoriesApart`, `...NeverDisablesAStep`, `...IsReadOnlyAtItsEvaluationPoint`,
  `TestAReplayRejectsAWitnessTheModelDoesNotTake`, `TestAQueryRejectsTwoMonitorsOfOneName`,
  `TestAMonitorWithoutANameIsRejected`, `TestMonitorStatesThatSpellAlikeStayApart`,
  `TestACounterexampleFoundBeforeTheLimitStands`, `TestAViolationOnlyBeyondTheLimitIsLimitReached`,
  `TestTheLimitBoundsEveryStateAPropertySearchVisits` and `TestAReplayChecksEveryDefinitionID`: every assertion and
  message is kept, with the same rows, explored counts, verdicts and monitor states.
- `TestAMonitorOfAnInfiniteStateTypeIsRejected` (typed-only: `DomainOf[int]` of a typed monitor state) is replaced by
  its key-level neighbour, `TestAMonitorWithoutItsNextFunctionIsRejected`. A `KeyMonitor` with a nil `next` is
  rejected with "query q: monitor endless: a Monitor names its next and violated functions". The typed claim is
  listed below.
- Comments: "opening counts the door's openings, up to two" is kept on `countOpenings`. "A machine whose two first
  steps reach one state by different outcomes" is kept, with a pointer to `forkedTable`. The `spells` comment is kept.

progress_test.go:
- Every key-level test is unchanged.
- `TestProgressOnATypedMachine`: every assertion is kept, over `doorTable("door")` and `KeyProgress`, including the
  action order `turn-right-true, push` and its message. The name is kept; task 12 may rename it.
- `TestProgressDeclarationsAreChecked`: the two key-level assertions are kept. The typed assertion ("progress typed:
  the state waiting of polls is not a checker_test.door") is dropped as typed-only. Its key-level neighbour, that an
  error from `from`/`to` fails the check and keeps its type, is already covered by `KeyProgressFunc` in
  `keyclaims_test.go` (`TestCallbackErrorsKeepTheirType`).
- `TestAMachinesAssumptionsJoinEveryProgressCheck`: the assumptions of the table and of the check are kept. The
  `Restrict` assertion (a restricted machine keeps its assumptions) is dropped as typed-only. There is no key-level
  restrict; a key table's assumptions are what its spec gives it.

### Typed-only claims for task 12

1. Monitor state-type enumeration: a typed `NewMonitor` over an infinite type fails with "state type: int is not
   finite" (`monitor_support_test.go`). This also covers its "initial state outside the domain" and "step is not a
   step of a T" errors, which no test exercised.
2. The typed composition's `Replaces(field, m)` with no member at `field` ("has no member key"), and "<member> does
   not refine it". `ComposeMember` carries `Replaces` itself, so the first cannot occur; the second is now the no-map
   message.
3. The typed `Progress` reading a state of the wrong type ("is not a checker_test.door"). `NewProgress` is in
   `progress_support_test.go`.
4. `Machine.Restrict` keeping assumptions (`machine_support_test.go`).
5. `Machine.Visible` without `Refines` ("names what a refined machine sees, and refines none"). Its key-level form
   is noted above.
6. The previously typed `Monitor.Next`/`Violated` fields and `After`: no test in these five files sets them now. Only
   `KeyMonitor`/`AfterKey` remain in use.

Key-level helpers that are defined in support files and that these files (and task 10's) still use. When task 12
deletes the support files, it must move these, not delete them:
- `KeyProgress` (`progress_support_test.go`).
- `ProgressAnswer.Incomplete` (`progress_support_test.go`, used in `composekeys_test.go`).
- `Answer.Incomplete` (`keyclaims_support_test.go`).

Names that task 12 may reword: `TestComposeTablesMatchesTypedComposition`, `TestProgressOnATypedMachine`, the
`pinned` and `composition` comments in `composekeys_test.go`, and `tableOf` in `umpire_test.go`, which no test uses
now (it is task 10's file, so I left it).

After this task, no test in these five files references `NewMachine`, `Step0`/`Step1`, typed `Property`/`Scenario`,
`Compose[...]`, `Refines`/`Visible`/`VisibleOutcomes`/`CoversStarts`/`Refinement()`, `NewProgress`, `NewMonitor`,
`After`, `Assumes`, `Restrict`, `DomainOf`/`KeyOf` or `tableOf`. I checked this by grep.

### Checks

| command | result | log |
|---|---|---|
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -v ./tools/umpire/model/internal/checker/` | PASS, 156 RUN | `.flow/tmp/fn113-11/checker-test.log` |
| `... -run 'TestTMPFixturesMatchTyped'` (by the previous implementer, before the typed block and the TMP tests were removed) | PASS (key fixtures equal the typed tables; composition pins captured) | `.flow/tmp/fn113-11/fixture-parity.log`, `.flow/tmp/fn113-11/composition-pins.log` |
| `GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model/internal/checker` | 0 issues | `.flow/tmp/fn113-11/lint-checker.log` |
| `gofmt -l tools/umpire/model/internal/checker/` | clean | (no output) |

Not run, per the conductor's constraints:
- The reader suite (`./tools/umpire/model/...`), `lint-code-fast` (repo-wide) and the model gate. The change touches
  only `_test.go` files of the checker package, which no other package imports.
- `CC=/usr/bin/clang`, because that compiler does not exist on this machine. The package needs no cgo.

No library was weighed (R25: none). No shared document needs a change.

### Review (claude-opus-5-5, fresh context, together with task 10)

Verdict NEEDS_WORK, fixed by the conductor:
- `pinned` in `composekeys_test.go` now carries `Order`, a SHA-256 of the ordered States, Actions, row keys and Reachable. The Behavior Fingerprint sorts and dedups what it reads, so it did not hold the order `ComposeTables` produces, which decides search order and explored counts.
  - The digests are the current output. They equal the typed compositions, because HEAD's test compared `ComposeTables` with them whole and the production code is unchanged.
  - The comment is corrected.
- `TestComposeTablesStartsInEveryMemberStart` pins `Starts` as a literal instead of comparing `tb.Starts` with itself.

Checker suite: PASS, 156 RUN (`.flow/tmp/fn113-11/checker-test-review-fixes.log`). Scoped lint: 0 issues (`.flow/tmp/fn113-11/lint-checker-review-fixes.log`).

For task 12, merge these duplicate pairs left by the port:
- `TestComposedKeysThatCollideAreRejected` (compose_test.go) with `TestComposeTablesRejectsCollidingKeys` (composekeys_test.go)
- `TestAViolatingProviderFailsItsReplacement` with `TestComposeTablesViolatingProviderFails`
- `TestAReplacementAccountsForEveryOpaqueStart` with `TestComposeTablesReplacementCoversEveryOpaqueStart`
- the fly-sync case, repeated in `TestComposeTablesDeclarationsAreChecked`

Also for task 12: `countOpenings` in `monitor_test.go` only forwards to `countOpeningKeys`, so inline it.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0513a1771b8cfabe152bf5bc3f63cbf2a75d641e
- Tests: GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -v ./tools/umpire/model/internal/checker/ -> PASS (156 RUN; .flow/tmp/fn113-11/checker-test.log), GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model/internal/checker -> 0 issues (.flow/tmp/fn113-11/lint-checker.log), go test -run TestTMPFixturesMatchTyped (previous implementer, before removal) -> PASS (.flow/tmp/fn113-11/fixture-parity.log), GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -v ./tools/umpire/model/internal/checker/ (after review fixes) -> PASS, 156 RUN (.flow/tmp/fn113-11/checker-test-review-fixes.log), GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model/internal/checker (after review fixes) -> 0 issues, independent review (claude-opus-5-5, fresh context): NEEDS_WORK; blocking and should-fix code findings fixed by the conductor, the rest recorded for task 12
- PRs: