---
satisfies: [R3, R4]
---
# fn-107-scala-umpire-prototype-for-standalone.18 Keep a replacing member's assumptions and admit action-scoped composition Properties

## Description
**Touches:** [model/go/umpire/compose.go, model/go/umpire/composekeys.go, model/go/umpire/*_test.go, model/scalav2/goir/load.go, model/scalav2/goir/claims.go, model/scalav2/goir/*_test.go, model/scalav2/scala/temporal/standaloneactivity/**, model/scalav2/ir/activity*.json, model/scalav2/run.sh, model/scalav2/SEMANTICS.md]

Close the two pre-existing defects that the task 4 review found outside its Touches, then restore the parity they blocked. Review text: `.flow/tmp/fn-107/task4/t4-r1.md` (findings 2 and 3).

**Size:** M

### Approach
- **Replacement keeps the member's own assumptions (P1).** In `model/go/umpire/compose.go` (~:236) replacement removes any assumption of the replacing member whose name also appears on the replaced interface. `SEMANTICS.md` says the replacing member's assumptions stay and only the replaced machine's are discharged, so a receipt can claim an assumption-gated fault was checked unconditionally. Keep every assumption the replacing member declares; do not import the replaced non-member's. The fix applies to the typed and the key-level composition, which share the core; typed tables that have no such name clash stay byte-identical.
- **A `when` on a composition's Property (P2).** `model/scalav2/goir/load.go` (~:969) refuses `when_action`/`when_class` on a Property of a composition although the semantics allow it and the Go baseline answers it. Admit it against composed action keys and bind it through the key-level Property; keep the located refusal for a key that names no composed class.
- **Restore what the defects blocked.** Lift `stoppedWorkerStartsNothing`, move it from the parity test's exclusion list into the compared domain (the list is then empty), and rename `dispatchQueue.storageLoss` back to the shared assumption name with the receipts still listing it.
- Test-first for each. No feature policy in Go.

### Investigation targets
**Required:** `model/go/umpire/compose.go`, `composekeys.go`, `compose_test.go`, `composekeys_test.go`; `model/scalav2/goir/load.go` (~:960-975), `claims.go`, `admission_test.go` (~:650); `model/scalav2/goir/activity_parity_test.go` (`TestActivityCrossEntityClaimIsOutsideTheDomain`), `activity_system_test.go`; `model/scalav2/scala/temporal/standaloneactivity/{Claims,System}.scala`; `model/go/standaloneactivity/claims.go`.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/...`; `GOFLAGS=-tags=test_dep make umpire-check-scala`; `make lint-scala`; `make lint-code LINT_CODE_TARGETS='./model/go/umpire ./model/scalav2/goir' GOLANGCI_LINT_FIX=false`.

## Acceptance
- [ ] A replacing member keeps every assumption it declares, also one the replaced interface names; the replaced machine's own assumptions are not imported. A receipt for a check gated by such an assumption names it. Typed compositions without a name clash are byte-identical (tables, IDs, fingerprints, existing tests unmodified).
- [ ] goir admits a Property of a composition with `when_action` or `when_class` over composed class keys and answers it through the generic key-level search; a key that names no composed class is a located admission error.
- [ ] `stoppedWorkerStartsNothing` is lifted and compared with the Go baseline; the activity parity test's exclusion list is empty and the test still fails on any uncompared baseline declaration.
- [ ] The storage-loss assumption has one name on the interface and the provider, and every receipt that relies on it lists it.
- [ ] Legacy model/scala files remain unchanged; v2 generation, lifting, native tests and lint keep passing.


## Done summary
A replacing member now keeps every assumption it declares, and goir admits a composition Property with `when_action` or `when_class` over composed class keys. `stoppedWorkerStartsNothing` is lifted and compared with Go on its path and on every row of the composition; the parity exclusion list is empty and the storage-loss assumption has one name. Nothing is committed and the task stays `in_progress`.

### Files

Changed (before-copies under `.flow/tmp/fn-107/task18-before/`):
- `model/go/umpire/compose.go`: `assumptions()` reads every member's own assumptions; `dischargedBy` is deleted. The replaced machine was never a member, so nothing of it is imported.
- `model/go/umpire/compose_test.go`, `composekeys_test.go`: the two tests that pinned the old rule now pin the new one (decision 1).
- `model/scalav2/goir/load.go`: the refusal is gone. `selectors` checks a composition Property's `when` against the composed class keys once the Model admits, beside `schedules`; both read the keys through `readable`.
- `model/scalav2/goir/claims.go`: `actionOf` and `classKey` are free functions that admission and the binding share. The binding itself is unchanged: `umpire.KeyProperty` over the composed table already read composed keys.
- `model/scalav2/goir/admission_test.go`, `bounds_test.go`, `checking_test.go`, `activity_parity_test.go`, `activity_properties_test.go`, `activity_system_test.go`.
- `model/scalav2/scala/temporal/standaloneactivity/Claims.scala`: `stoppedBeforeRetry` schedules literal keys (decision 3).
- `model/scalav2/scala/temporal/standaloneactivity/System.scala`: `queueLosesStorage` is gone; the interface assumes `storageLossAssumed` (`storageLoss`).
- `model/scalav2/run.sh`: `stoppedWorkerStartsNothing` is a root of `ir/activity.json`.
- `model/scalav2/ir/activity.json`, `ir/activity-system.json`: regenerated by `make umpire-gen-scala`.
- `model/scalav2/SEMANTICS.md`, `model/scalav2/README.md`.

No file added. `model/go/umpire/composekeys.go` is byte-unchanged: the key-level composition takes the fix through the shared core. The before-copy of `bounds_test.go` was rebuilt after the edit by removing the one appended test (83 lines, the size read before the edit).

### Acceptance

| # | Item | Result | Proving tests |
|---|---|---|---|
| 1 | A replacing member keeps every assumption it declares; the replaced machine's are not imported; a receipt names it; no-clash compositions byte-identical | pass | `TestAReplacingMemberKeepsEveryAssumptionItDeclares` (umpire, typed, with the progress answer's assumptions), `TestComposeTablesKeepsEveryAssumptionOfAReplacingMember`, `TestComposeTablesMatchesTypedComposition` (unmodified), `TestAReplacingMemberStandsInForTheOpaqueProvider` (unmodified), goir `TestAReplacingMemberKeepsEveryAssumptionItDeclares`, `TestActivityStorageLossIsAssumed` |
| 2 | goir admits `when_action` and `when_class` over composed class keys and answers through the key-level search; a key of no composed class is a located error | pass | `TestAdmissionAdmitsAWhenOverComposedClasses`, `TestAdmissionRejectsMisaddressedSelectors` (five composition cases), `TestComposedKeysAreCountedBeforeAPropertyReadsThem`, `TestActivityCrossEntityClaimByClassAnswersAlike` |
| 3 | `stoppedWorkerStartsNothing` lifted and compared; exclusion list empty; inventory still fails on an uncompared declaration | pass | `TestActivityClaimsEqualTheGoModel` (12 Queries), `TestActivityCrossEntityClaimIsCompared`, `TestActivityPropertiesAgreeOnEveryRow`, `TestActivityPropertyRowsCatchWhatThePathsMiss` (two new mutants), `TestActivityClaimDomainIsWholeOrExcluded` (logic unchanged) |
| 4 | One storage-loss name on interface and provider; every receipt relying on it lists it | pass | `TestActivityStorageLossIsAssumed` |
| 5 | Legacy `model/scala` unchanged; v2 generation, lifting, native tests and lint pass | pass | gates below; `git status` shows only the `Lean.test.scala` change that predates the task |

### What changed in behavior

- **Assumptions.** `currentOverLossyMatching` and its seven Queries list `storageLoss` with the interface and the provider sharing the name. With the old `compose.go` and the renamed Model those receipts list nothing.
- **Admission messages.** `<owner>: <composition> has no class <key>` for a `when_class`, the wording a Scenario key already had, and `<owner>: <composition> has no class of the action <name>` for a `when_action`. A `when_action` names a sync, or `<field>_<action>` for a member's own action. A member's action a sync takes is an error under either spelling (`put`, `front_put`).
- **Ceilings.** A composition whose classes pass the ceiling is refused at the Property with the same `LimitError` a Scenario gets. The keys are counted once for both.
- **Parity.** 12 Queries compared (9 found, 3 verified, plus the refinement). The per-row comparison adds the 2,518 rows of `standaloneActivity`: `startedByPollingWorker` is about 24 of them and holds on all 24.

### Test-first record

Logs under `.flow/tmp/fn-107/task18-logs/`.
- `red-p1-umpire.log`: both umpire tests fail, `expected [doorIsOiled keyIsOpaque]`, `actual [doorIsOiled]`.
- `p1-goir-after-umpire.log`: goir's `TestOnlyTheReplacingMembersAssumptionsAreDischarged` fails after the fix, which is the test that pinned the old rule there.
- `red-p1-receipts.log`: with the before-copy of `compose.go` swapped in over the final tree, `TestActivityStorageLossIsAssumed` fails with `[]string(nil) does not contain "storageLoss"` for `composition currentOverLossyMatching`. The file was restored and compared.
- `red-p2-admission.log`: five composition cases fail on `a Property of a composition is about every step`; the admit test and the ceiling test fail.
- `red-task4-tripwire.log`: task 4's `TestActivityCrossEntityClaimIsOutsideTheDomain` fails once admission takes the Property, as it was written to.
- `gen-01.log`: the first lift after the Scala edits admitted. The Go run inside it fails the four tests not yet updated, and `stoppedWorkerStartsNothing` is already `verified-within-limits`.
- `mutants.log`: five mutants of the new admission code, in the foreground with the file restored after each; all five fail a test.

### Decisions that differ from the task text

1. **Three existing tests were edited, not left byte-identical.** `TestAReplacementDischargesOnlyTheReplacingMembersAssumptions`, `TestComposeTablesDischargesOnlyTheReplacingMembersAssumptions` and goir's `TestOnlyTheReplacingMembersAssumptionsAreDischarged` asserted the defect itself. Each keeps its setup and its second half; the first expectation and the name changed. The two admission cases that pinned the refusal changed the same way. Every other pre-existing test is unmodified in what it asserts.
2. **`claims.go` got no new binding logic.** The existing `when` already matched composed keys through `umpire.KeyProperty`; only admission stood in the way. The change there is the two shared helpers.
3. **`stoppedBeforeRetry` uses literal keys.** The lifter does not fold `Composition.own(...)` (task 4, gap 2) and is outside the Touches. `TestActivityCrossEntityClaimIsCompared` requires the lifted keys to equal Go's `Own(...)` keys, and admission checks each is a class.
4. **`TestActivityCrossEntityClaimIsOutsideTheDomain` is replaced** by `TestActivityCrossEntityClaimIsCompared`.
5. **`excludedClaims` stays as an empty map** so the inventory test keeps its exclusion branch.
6. **The tally in `TestActivityPropertiesAgreeOnEveryRow`** counts each Property over the table its Query runs on, since one Property now belongs to the composition.

### Not done, and gaps outside the Touches

1. `model/scalav2/goir/compose.go` and the `Replaces` doc comments in `model/go/umpire` still say the composition "relies on none of the replaced machine's assumptions". That is true of what is imported and says nothing of a shared name. Left as is.
2. `make umpire-gen-scala` also rewrote `ir/nexus-close.json` and `ir/nexus-caller.json`. Their sources are unchanged, `nexus-caller.json` shows no diff in git, and `nexus-close.json` is untracked with no earlier copy to compare; the later `umpire-check-scala` found it current.
3. The snapshot run without `model/scala` was not repeated. The task added no path and no build input; `TestNoInputNamesTheLegacyTree` and `TestIRSourcePositionsResolveInsideScalav2` pass.
4. No Scala-native test reads the literal keys against `own(...)`; `scala/temporal/test` is outside the Touches.
5. No `flowctl gate` receipt: the tree is dirty and the owner commits.

### Gates

| Command | rc |
|---|---|
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/...` (baseline, before any edit) | 0 |
| `CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/...` (goir: 437 passing tests and subtests, 0 failed, 5 Lean-dump skips) | 0 |
| same, in the overrides' order | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala` (after the last edit) | 0 |
| `GOFLAGS=-tags=test_dep make lint-scala` (0 `[error]` lines; the `NoSuchFieldException` traces are the same four as in task 4's accepted run) | 0 |
| `GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/go/umpire ./model/scalav2/goir' GOLANGCI_LINT_FIX=false` (0 issues) | 0 |
| `CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/umpire/...` | 0 |

The work is uncommitted; the owner makes the commits. The task diff and the review output are under `.flow/tmp/fn-107/task18/`.

stage: implement - ran (worker subagent, session model claude-opus-5-5)
stage: impl-review - ran (codex:gpt-5.6-sol:high, session 01a0f68d-4e6b-7352-a047-692c4b4d7e12 over the uncommitted task diff; round 1 SHIP, no findings)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... rc=0, before any edit), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc=0), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc=0), GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-gen-scala (IR regenerated; rc=2 only from the Go tests not yet updated at that point), GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-check-scala (rc=0, run after the last edit), GOFLAGS=-tags=test_dep make lint-scala (rc=0, 0 [error] lines), GOFLAGS=-tags=test_dep CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS='./model/go/umpire ./model/scalav2/goir' GOLANGCI_LINT_FIX=false (rc=0, 0 issues), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/scalav2/... ./model/go/umpire/... (rc=0), conductor final: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc 0), conductor final: GOFLAGS=-tags=test_dep make umpire-check-scala (rc 0), conductor final: make lint-scala (rc 0, 0 [error] lines), conductor final: make lint-code LINT_CODE_TARGETS='./model/scalav2/goir ./model/go/umpire' GOLANGCI_LINT_FIX=false (0 issues), codex impl-review: .flow/tmp/fn-107/task18/t18-r1.md (SHIP)
- PRs: