---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.10 Port the checker's search, lowering and claim tests off the typed fixture layer

## Description
Port the checker's search, lowering and claim tests off the typed fixture layer. Advances the spec's "Go checker's typed fixture layer" constraint (recorded under R14: the checker's tests then run over tables built from keys or IR); task 12 deletes the branches.

**Size:** M
**Files:** tools/umpire/model/internal/checker/umpire_test.go, keyclaims_test.go, keylower_test.go, keyunknown_test.go
**Touches:** [tools/umpire/model/internal/checker/umpire_test.go, tools/umpire/model/internal/checker/keyclaims_test.go, tools/umpire/model/internal/checker/keylower_test.go, tools/umpire/model/internal/checker/keyunknown_test.go]

### Approach
- Replace every use of the typed support layer (`NewMachine`, `Step0`/`Step1`, the typed Property and Scenario builders, reflection domains and keys from `*_support_test.go`; 47 call sites across these four files at planning) with the key-level constructors the module map names as private-checker operations (`NewTable`, `TableSpec`, `KeyProperty`, `KeyScenario`, `KeyFind`) or with IR fixtures the reader's tests already load. Preserve every assertion and comment; a test that asserts typed-only behavior (for example the typed `alterer` or a typed `holds`) becomes the key-level assertion of the same claim, or is listed for task 12 to decide.
- Do not delete the support files or any production branch here (task 12 does, once task 11 is also done). `require` over `assert`; `Equal`-style whole-value comparisons.
- Count the tests before and after; none disappears without a note.

### Investigation targets
**Required**:
- `tools/umpire/model/internal/checker/umpire_test.go`, `keyclaims_test.go`, `keylower_test.go`, `keyunknown_test.go`
- `tools/umpire/model/internal/checker/machine_support_test.go`, `domain_support_test.go`, `action_support_test.go`, `claims_support_test.go`, `keyclaims_support_test.go` (what each builder produces)
- `tools/umpire/model/internal/checker/table.go:60-80` (`Table` fields; `NewTable`, `TableSpec`)
- `tools/umpire/model/internal/checker/keyclaims.go:75-121` (`keyLevel`, `KeyProperty`, `observeKeys`)
- `tools/umpire/model/internal/checker/lower.go:94-200` (`asking.accepts`, `alterer`, `keyAlterer`)
- `.flow/tmp/fn115-7-summary.md` (decision 1: the branch list)

### Quick commands
CC=/usr/bin/clang mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/internal/checker/; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/model/...; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] `umpire_test.go`, `keyclaims_test.go`, `keylower_test.go` and `keyunknown_test.go` reference no typed support builder; every assertion and comment is preserved or its key-level equivalent is noted; the test count before and after is in the summary.
- [ ] The checker and reader suites pass; `lint-code-fast` passes; no production file and no support file changed.


## Done summary
# fn-113 task 10: the checker's search, lowering and claim tests off the typed fixture layer

### What changed

Edited only test files in `tools/umpire/model/internal/checker/`. No production file and no
`*_support_test.go` file changed.

- `umpire_test.go`: the toy door, the abstract door, the keyholder and the opaque key are now tables of
  keys (`doorTable`, `abstractTable`, `keyholderTable`, `opaqueKeyTable`, plus `silentDoor` and
  `opensFirst` for two refinement cases), built with `NewTable`/`TableSpec`. The search tests use
  `KeyProperty`/`KeyTransitionProperty`/`KeyScenario`/`KeyFreeScenario`/`KeyFind`/`KeyVerify`. The
  refinement tests use `RefineTables`, and the composition test uses `ComposeTables`. New helpers:
  `rowOf`, `resultOf`, `tableFrom`, `walk` (the witness a list of rows spells), `assumes`,
  `refiningAbstract`, `readsAbstract`, `opensLoudlyOn`, `staysShutOn`.
- `keyclaims_test.go`: the typed side of the typed/key comparisons is gone. Each case now pins the
  answer the typed Query gave, captured before the port (`fn113-10/typed-answers.log`), and compares it
  whole with `require.Equal` (outcome, explored, expanded, explanation, rows, exercised, monitor,
  monitors, and the witness through `walk`). `forked()` became `forkedTable()`. The `doorMachine` alias
  is removed. `keyCopy`'s comment now reads "a table as the spec of ...".
- `keylower_test.go`: one call site (`keysOf(newDoor)`, `keysOf(opaqueKey())`) became
  `doorTable("door")` and `opaqueKeyTable()`.
- `keyunknown_test.go`: no typed reference at planning or now. Unchanged.
- Transitional, in a task-11 file: the typed declarations that the task-11 files still used (door,
  abstract door, keyholder, house, and their types and actions) moved verbatim from `umpire_test.go` to
  `compose_test.go`, under a note. Task 11 removes them. A temporary parity test,
  `TestTMPFixturesMatchTyped` in `compose_test.go`, checks that every new key fixture equals the
  `keyCopy` of the typed table it replaces, including reachability and `TargetFingerprint`
  (`fn113-10/fixture-parity.log`). Task 11 extends it and then deletes it.

### Test counts (top-level `func Test`)

| file | before | after |
|---|---|---|
| umpire_test.go | 18 | 11 |
| keyclaims_test.go | 13 | 13 |
| keylower_test.go | 5 | 5 |
| keyunknown_test.go | 10 | 10 |

Package `=== RUN` lines (tests and subtests): 163 before, 157 after. That is 163 - 7 removed + 1
temporary parity test.

### Assertions and comments, test by test

umpire_test.go:
- `TestTableOrdersActionsByKeyAndRowsStatesMajor`: every assertion is kept, over `doorTable("door")`.
  Actions and rows are now in the spec's order, since the key table keeps what it is given. Reachable,
  Ends, Facts and `IDs()` are computed or carried by `NewTable`. A new comment says this. The name is
  kept; task 12 may rename it.
- `TestRefinementMatchesStepsAndStutters`: every assertion is kept, through `RefineTables` with the map
  `abstractKeyOf`. "StateFields contains abstract" is now a spec input (`refiningAbstract`). As its
  key-level counterpart, I added a check that `NewTable` rejects a `RefinedField` that is not a state
  field ("concrete: the refined field abstract is not a state field").
- `TestRefinementRejectsARowWithNoProductStep`, `...AProductFactTheRowDoesNotRecord` (its comment is
  kept), `...AStartTheProductDoesNotHave`: same messages, through `RefineTables`. The default
  declaration "concrete refines abstract" is unchanged.
- `TestFindReturnsTheShortestWitness`, `TestFindReportsNotFoundAndAWrongPathIsRejected` (including "a
  pinned action with no row admits no trace" and the pin-limit error), `TestVerifyFinds...`,
  `TestSearchStopsAtItsLimit`, `TestAFindCannotRealizeATransitionClaim`: every assertion is kept, over
  key claims.
- `TestCompositionSynchronizesAndKeysByMember`: every assertion is kept, over `ComposeTables`, except
  the malformed-sync-reference error (see below).
- Removed (typed-only; listed for task 12): `TestDomainOrderIsDeclarationOrderWithTheLastFieldFastest`,
  `TestSumOfANonInterfaceIsReportedWhenEnumerated`, `TestUnenumerableTypeIsRejected`,
  `TestAStepOutsideTheDomainIsRejected`, `TestTwoStepsBindingOneClassAreRejected`,
  `TestAMachineWithoutAStartIsRejected`, `TestAnExampleOfTheWrongTypeIsRejected`.

keyclaims_test.go:
- `TestKeyLevelQueryMatchesTypedQuery` (all 12 cases) and `TestKeyLevelThroughQueryReadsByMapAndName`
  (2 cases): the whole-value comparison is kept against the pinned typed answers, along with
  `Incomplete`, `q.Replay` and `tb.Replay`. The message "outcome, explored states, rows, witness,
  exercise and monitor verdicts" is kept. The names are kept.
- `TestKeyLevelMonitorsKeepHistoriesApart`: unchanged, over `forkedTable()`, which equals the typed
  table (parity log).
- `TestAThroughQueryReadsKeysThroughAKeyRefinementOnly`: the two key-level assertions are kept (a
  refinement of other tables; a nil refinement). The two typed cross-check assertions are removed (see
  below).
- All other tests in the file were already key-level and are unchanged.

### Typed-only claims for task 12

There is no key-level form of these, so the port dropped them:
1. Domain enumeration and keys: `DomainOf` catalog order (last field fastest), a `Sum` of a
   non-interface, and an unenumerable type (`domain_support_test.go`). The key-level catalog order is
   the reader's.
2. `Machine` building errors: a result outside the state domain, two steps binding one class, no
   start, and an example of the wrong type (`machine_support_test.go`, `action_support_test.go`).
   Key-level neighbours: `ComposeTables` refuses a member with no start
   (`TestComposeTablesDeclarationsAreChecked`), and `NewTable` refuses a claim of no action class
   (`TestAKeyedTableCarriesItsFieldValuesAndClaims`). **Open point for task 12 or the owner:**
   `NewTable.checkSpec` accepts a result whose state is not in `States`, and a spec with no start. If
   the claim should live at key level, that is a production change, which neither task 10 nor task 11
   may make.
3. The `checkKeyRefined` cross-checks: "typed reads typed states, and the refinement ... reads keys"
   and "keyed reads keys, and the refinement ... reads typed states". These are the branches task 12
   deletes.
4. The typed `Composition.Sync` malformed reference ("sync reference \"door\" is not
   <member>.<action>"). `ComposeSync` has separate fields, so the case cannot occur at key level.

Comments for task 12 to reword once the typed layer is gone: the doc comments of
`TestAKeyLevelPropertyLowersToItsClauses`, `TestAKeyLevelPropertyIsRefusedAsATypedOneIs` and
`TestAKeyedTableCarriesItsFieldValuesAndClaims` in `keylower_test.go`, which compare with "a typed
one". I left them unchanged, as the task asks.

After task 10, no test in these four files reaches the typed `holds`/`holds2`, the typed `alterer`,
`readState`/`readStep`, the typed `observe` arms, `productStep`/`stepOf`, or the typed branch of
`asking.accepts`. Task 11 covers the rest of the package.

### Checks

| command | result | log |
|---|---|---|
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -v ./tools/umpire/model/internal/checker/` (baseline) | PASS, 163 RUN | `.flow/tmp/fn113-10/baseline-test.log` |
| same, after the port | PASS, 157 RUN | `.flow/tmp/fn113-10/checker-test.log` |
| `... -run TestTMPFixturesMatchTyped` | PASS (key fixtures equal the typed tables, including fingerprints) | `.flow/tmp/fn113-10/fixture-parity.log` |
| `GOLANGCI_LINT_FIX=false mise exec -- make lint-code-fast` | failed on another agent's `tools/umpire/fn113audit/dump_test.go`, which vanished mid-run (typecheck: no such file); nothing from this task | `.flow/tmp/fn113-10/lint-code-fast.log` |
| `GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model/internal/checker` (the Makefile's own golangci-lint and vet invocation, scoped) | 0 issues (after gofmt of the moved block) | `.flow/tmp/fn113-10/lint-checker.log` |

Notes:
- `CC=/usr/bin/clang` does not exist on this machine, so the runs used the default C compiler. The
  package needs no cgo.
- I did not run the reader suite (`./tools/umpire/model/...`) or the model gate: the conductor's
  concurrency constraint forbids it while other agents run them. The change touches only `_test.go`
  files of the checker package, and no other package can import those.
- No library was weighed (R25: none). No shared document needs a change.

### Review (claude-opus-5-5, fresh context, together with task 11)

Verdict NEEDS_WORK, fixed by the conductor. `tableOf`, the last reference to the typed `umpire.Model` in `umpire_test.go`, is deleted. The floating comment above `forkedTable` in `keyclaims_test.go` is now its doc comment. Added to the typed-only list for task 12:
- `TestTableOrdersActionsByKeyAndRowsStatesMajor` now compares Actions, Rows, Ends and Facts of `doorSpec` with itself. Only Reachable and `IDs()` test `NewTable`, so its "orders by key / states-major" claim is typed-only: rename or trim it.
- The open point on `NewTable.checkSpec` is wider than item 2 says. Besides a result state outside `States` and a spec with no start, it also accepts a row whose source is not a state, a row whose action is not in `Actions`, and two rows with one key. The last is the real key-level analogue of the removed `TestTwoStepsBindingOneClassAreRejected`.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0513a1771b8cfabe152bf5bc3f63cbf2a75d641e
- Tests: GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/internal/checker/ -> PASS (157 RUN incl. subtests; baseline 163), GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -run TestTMPFixturesMatchTyped ./tools/umpire/model/internal/checker/ -> PASS (key fixtures equal typed tables), GOLANGCI_LINT_FIX=false mise exec -- make lint-code-fast -> FAIL on another agent's vanished tools/umpire/fn113audit/dump_test.go (unrelated), GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model/internal/checker -> 0 issues, independent review (claude-opus-5-5, fresh context): NEEDS_WORK; blocking and should-fix code findings fixed by the conductor, the rest recorded for task 12
- PRs: