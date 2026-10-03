---
satisfies: [R14]
---
# fn-113-clean-up-the-scala-model-layer-around.12 Delete the checker branches only typed fixtures reached

## Description
Delete the checker branches only typed fixtures reached. Completes the "Go checker's typed fixture layer" constraint (recorded under R14) after tasks 10 and 11; no reader or lowering semantics change (tables built from IR never set the typed fields).

**Size:** S
**Files:** tools/umpire/model/internal/checker/{claims.go, search.go, refine.go, lower.go, table.go, keyclaims.go} (production), the nine *_support_test.go files (deleted)
**Touches:** [tools/umpire/model/internal/checker/*.go, tools/umpire/model/internal/checker/*_support_test.go]

### Approach
- The list from fn-115.7's decision 1: `PropertyDecl.holds`/`holds2` (`claims.go:14-22`); the typed arms of `searcher.observe`, `readState`/`readStep`, `Refinement.productStep` (`search.go:339-350,453-470,505-515`); `Refinement.mapValue`, `stepOfFn`, `stepOf`, the typed side of `keyLevel` and the typed cases of `checkKeyRefined` (`refine.go:16-45,289`); the typed branch of `asking.accepts`, `Table.alterer()` and the `alter` field (`lower.go:94-104,168-180`, `table.go:72`); `keyAlterer` stays. Delete them and the nine `*_support_test.go` files (1,342 lines) to `.flow/tmp/trash/fn113-12/`.
- Prove deadness before deleting: `grep` that no production code assigns `holds`, `mapValue`, `stepOfFn`, `alter` or `stateValue` from an IR-built table (only the deleted support did). Simplify what remains (`keyLevel` becomes constant, `checkKeyRefined` loses its typed cases); keep diagnostics and error texts unchanged.
- The reader's public aliases (`tools/umpire/model/types.go`) must still build; run the full `./tools/umpire/...` once (goldens included) since the checker is in every consumer's closure.
- Record any branch that stays and why; count checker production lines before and after for the handover (module map's checker row).

### Investigation targets
**Required**:
- `tools/umpire/model/internal/checker/claims.go:10-25`
- `tools/umpire/model/internal/checker/search.go:339-350,453-470,505-515`
- `tools/umpire/model/internal/checker/refine.go:1-60,285-295`
- `tools/umpire/model/internal/checker/lower.go:85-200`
- `tools/umpire/model/internal/checker/table.go:60-80`
- `tools/umpire/model/types.go` (reader aliases over checker types)
- `.flow/tmp/fn115-7-summary.md` (decision 1), `.plans/UMPIRE_MODULES.md:76-85`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/...; CC=/usr/bin/clang mise exec -- go test -count=1 -json -tags test_dep ./tools/umpire/... > .flow/tmp/fn113-12/full-go.jsonl; GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast

### Execution constraints
No staging, commits, worktrees or recursive deletion: move a removed file to `.flow/tmp/trash/fn113-N/` (N = this task's number). Preserve existing comments, except where the spec's Comments rule applies to code this task deletes and to Stainless references. Never install or invoke the retired proof toolchain, and write none of its vocabulary under `model/`: the gate's first step (`TestModelNamesNoRetiredFrontEnd`) fails on a mention. `make umpire-check-backends` is deferred by the owner; do not run it. No IR schema change; no change to the semantics of `tools/umpire/model`, `tools/umpire/lower` or `tools/umpire/export`; no dependency in `model/project.scala` (R3). Other workers share this tree: edit only the files in Touches, and do not run the model gate (it writes `model/gen`, and `--update` writes `model/ir`) while another Scala task is running it; ask the conductor. Verification follows MILESTONES.md: the smallest checks while editing, the task's required gates once when ready for review, and the fn-115 goldens (`TestMigrationGoldens` in `tools/umpire/model` and `tools/umpire/lower`) as the unchanged baseline. Keep logs under `.flow/tmp/fn113-N/`, the handover at `.flow/tmp/fn113-N-summary.md` and the evidence at `.flow/tmp/fn113-N-evidence.json`. Record every library weighed against generic code with the line counts both ways (R25). Shared documents (`MILESTONES.md`, `.plans/UMPIRE_MODULES.md`, the migration manifest) belong to the conductor: list the needed changes in the handover instead of editing them.

## Acceptance
- [ ] The nine `*_support_test.go` files are gone and the listed production branches are deleted or each remaining one is recorded with its reason; no diagnostic or error text changed.
- [ ] `./tools/umpire/...` passes in full (goldens included) and `lint-code-fast` passes; checker production line counts before and after are in the summary.


## Done summary
# fn-113 task 12: delete the checker branches only typed fixtures reached

### What changed

All edits are in `tools/umpire/model/internal/checker/`. No reader, lowering or export file changed.

### Support files

The nine `*_support_test.go` files (1,342 lines) were moved to `.flow/tmp/trash/fn113-12/`. These files were
`action`, `claims`, `compose`, `domain`, `keyclaims`, `machine`, `monitor`, `progress` and `refine`. The key-level
helpers that the remaining tests use moved verbatim to a new `export_test.go` (package `checker`, 29 lines). They
are `KeyProgress`, `ProgressAnswer.Incomplete`, `Answer.Incomplete`, `Table.Model` and `Table.UnknownFrom`. The
handover named the first three. The tests also need `Model` and `UnknownFrom`, which lived in
`keyclaims_support_test.go`. Production needs none of them: the reader has its own forms. I reworded only the
`KeyProgress` doc comment, to drop "typed".

### Production branches deleted (each one from the Approach list)

Deadness proof: `grep` over `tools/umpire`, `common/testing/testpilot` and `tools/canary`, excluding the support
files. No production code assigns `holds`, `holds2`, `mapValue`, `stepOfFn` or `alter`. Every production
`stateValue` assignment was `stateValue[s] = s`, for each state in `NewTable` and `assembleKeys`. That makes it a
membership set of `States`. No production code reads or sets `Monitor.Next`, `Monitor.Violated`, `Monitor.err` or
`Evaluation.after`. The reader builds monitors through `KeyMonitor`/`AfterKey` only (`model/claims.go:571-579`).
`Model` has one production implementation, `tableModel`.

- `claims.go`: removed the `PropertyDecl.holds`/`holds2` fields. `isTransition` now reads `keyHolds2` only.
- `search.go`:
  - Removed the typed arms of `searcher.observe`. `observe` is gone, and `step` calls `observeKeys`.
  - Removed `readState`, `readStep` and `Refinement.productStep`.
  - `watch` lost its `before any` parameter, and `checkScenario` checks membership with `slices.Contains(t.States, …)`.
- `refine.go`:
  - Removed `Refinement.mapValue`, `stepOfFn`, `keyLevel` and `stepOf`.
  - `checkKeyRefined` lost its two typed cross-checks and its typed `return nil`. Only the check that the refinement
    belongs to the Query's tables remains. Its error text is unchanged.
- `lower.go`:
  - Removed the typed branch of `asking.accepts`, `Table.alterer()` and the typed wording of the `asking`/`alterer`
    docs.
  - `fixedRequirements` calls `keyAlterer()` directly. `keyAlterer` stays.
  - `Lower` used to detect a composition by `t.alter.state == nil`. It now uses `t.parts != nil`, with a comment.
    `catalogs` is the only constructor that sets `parts`, and it was the only production table without an alterer,
    so the test is equivalent.
- `table.go`:
  - Removed the `alter` and `stateValue` fields and their setup in `NewTable`.
  - The doc comments of `Result.Step` and `Table` no longer say that typed values are kept. `Result.Step` stays,
    because `ComposeTables` puts a `ComposedStep` there and the reader reads it (`model/compose.go:80`).
- `compose.go`, `composekeys.go`: removed the `stateValue` setup.
- `replay.go`: membership is checked by `slices.Contains(t.States, …)` instead of `stateValue`. The message is the
  same.
- `monitor.go`: this branch was not in the Approach list but is the same kind of branch. Handover 11 item 6 flagged
  it, and `monitor.go` is within Touches.
  - Removed the typed `Monitor.Next`, `Monitor.Violated`, `Monitor.err`, `Evaluation.after`, and the
    `next`/`violated` dispatchers.
  - `check` now rejects a Monitor with a missing `keyNext` or `keyViolated`, with the same message.
  - `model.Monitor` is a public alias, so its exported `Next`/`Violated` fields are gone. No package set them.

No diagnostic or error text changed. The one expectation that changed (below) comes from a new check, not a reworded
one.

### Branches that stay

- `PropertyDecl.keyLevel` (`keyclaims.go`) stays. It now means "names a function", and `checkKeyClaims` still rejects
  a Property that names none ("… names no function that says whether it holds", tested). Its doc comment was reworded
  in review.
- `Result.Step` stays. It is a live carrier for `ComposedStep`.
- (Deleted in review) `ComposeTables`' "the member %s has no start" guard: it can no longer fire; see Review.

### `NewTable.checkSpec`: the five key-level checks (conductor decision)

All five are added, in a new `checkRows` that runs after every existing check, so existing errors keep precedence:

| check | message |
|---|---|
| no start | `<machine>: the table has no start` |
| row source not a state | `<machine>: the row '<key>' is at '<src>', which is not a state` |
| row action not an action class | `<machine>: the row '<key>' takes <a>, which is not an action class` |
| two rows with one key | `<machine>: the row '<key>' is listed twice` |
| result state not a state | `<machine>: the row '<key>' leads to '<s>', which is not a state` |

Why the IR-built tables pass: `Interpreter.machine` rejects:
- a start outside the domain, and a machine with no start (`machine.go:611,616`; a hole in the starts fails the
  machine through `unread.err()`);
- a result outside the domain (`machine.go:520`);
- colliding row keys (`rowKeys`).

It builds rows only over its own states × classes. `binding.view`/`claimed` copy that table. The suites confirm this:
every package under `./tools/umpire/...` passes, including the goldens.

Every test fixture passes too, except one deliberately malformed one: the `startless` member in
`TestComposeTablesDeclarationsAreChecked`. Its expectation changed from `compose-house: the member key has no start`
to `keyholder: the table has no start`. That is the same claim, now rejected one level earlier. No production path
builds such a table, so the check was added.

Negative test: `TestARowThatDoesNotFitItsTableIsRejected` (keyclaims_test.go). It has one case per check, and
asserts `*umpire.Error` and that a Query over the table reports `Err`.

The lookups use a state set, not `slices.Contains`, so large reader tables stay linear.

### Test merges and renames

- Merged the duplicate pairs. The compose_test.go test is kept, and the composekeys_test.go twin is deleted:
  - `TestComposedKeysThatCollideAreRejected` kept; `TestComposeTablesRejectsCollidingKeys` deleted (identical).
  - `TestAViolatingProviderFailsItsReplacement` kept (its `requireReplaysFromAStart` covers the twin's replay and
    start assertions); `TestComposeTablesViolatingProviderFails` deleted.
  - `TestAReplacementAccountsForEveryOpaqueStart` kept and gained the twin's two extra assertions (the
    `ProductWitness` replays on the opaque table, and the table is nil); `TestComposeTablesReplacementCoversEveryOpaqueStart`
    deleted.
  - Fly-sync: `TestASyncNamingAnActionItsMemberLacksIsRejected` deleted. The case in
    `TestComposeTablesDeclarationsAreChecked` is kept, since it also asserts `*umpire.Error` and a nil table.
- `countOpenings` is inlined into `countOpeningKeys`, and its comment moved onto `countOpeningKeys`.
- Renames:
  - `TestTableOrdersActionsByKeyAndRowsStatesMajor` → `TestATableKeepsItsSpecOrderAndSweepsWhatItReaches`. The
    name now matches its existing comment ("keeps the order its spec gives … sweeping the rows"). The assertions
    are kept.
  - `TestComposeTablesMatchesTypedComposition` → `TestComposeTablesMatchesPinnedComposition`.
  - `TestProgressOnATypedMachine` → `TestProgressOnADoor`.
  - `TestKeyLevelQueryMatchesTypedQuery` → `TestKeyLevelQueryMatchesPinnedAnswer`.
  - `TestAKeyLevelPropertyIsRefusedAsATypedOneIs` → `TestAKeyLevelPropertyThatNoClausesCarryIsRefused`.
- Reworded the test comments that compared with "a typed one". These are the three in keylower_test.go that
  handover 10 listed, plus `keyCopy` and the `composition` comment. The `answered` and `pinned` comments keep their
  provenance note.

Package `=== RUN` count: 156 before, 153 after. That is 4 deleted duplicates and 1 new negative test.

### Typed-only claims removed with the typed layer (from handovers 10 and 11)

- Domain enumeration: `DomainOf` catalog order (last field fastest), a `Sum` of a non-interface, an unenumerable
  type.
- `Machine` building errors:
  - a result outside the state domain, two steps binding one class, no start, and an example of the wrong type;
  - three of these now have key-level forms in the new `checkRows`: no start, result outside the state domain, and
    two rows with one key.
- The `checkKeyRefined` typed cross-checks ("reads typed states … reads keys" and the reverse).
- The typed `Composition.Sync` malformed reference (cannot occur with `ComposeSync`).
- Typed `NewMonitor` errors: an infinite state type, an initial state outside the domain, and a step of the wrong
  type.
- The typed composition's `Replaces` with no member at the field, and "does not refine it".
- The typed `Progress` reading a state of the wrong type.
- `Machine.Restrict` keeping assumptions.
- `Machine.Visible` without `Refines`.
- The typed `Monitor.Next`/`Violated`/`After`.

### Line counts (checker production, non-test `.go`)

| | lines |
|---|---|
| before | 4,161 |
| after (with the review fixes) | 4,064 |
| of which the new key-level checks (`checkRows`, 32 lines, plus a blank and a doc line) | +34 |
| typed branches, the dead no-start guard and stale comments removed (net, excluding the new checks) | -131 |

Per file (before → after): claims 155→153, compose 493→491, composekeys 217→211, keyclaims 204→203, lower 203→197,
monitor 175→147, refine 325→303, search 513→457, table 428→454; the others are unchanged (`.flow/tmp/fn113-12/prod-lines-{before,after}.txt`).
Checker test lines: 2,881 now. The 1,342 support lines are gone, and `export_test.go` adds 29.

### For the conductor (shared documents)

- `.plans/UMPIRE_MODULES.md:82-85`, the checker paragraph:
  - "the typed declaration layer of the checker … is test support in `*_support_test.go`" and "Production branches
    that only typed fixtures reach remain in the checker; fn-113 Part C lists their removal" are now false.
  - Suggested text: "fn-113 Part C deleted the typed declaration layer and the production branches only it reached;
    the checker reads keys only."
  - The checker row's line count is 4,064.
- `model.Monitor` no longer has exported `Next`/`Violated` fields. The module map's alias list is unchanged.

### Checks

| command | result | log |
|---|---|---|
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -v ./tools/umpire/model/internal/checker/` (baseline, before any edit) | PASS, 156 RUN | `.flow/tmp/fn113-12/baseline-checker.log` |
| same, after | PASS, 153 RUN | `.flow/tmp/fn113-12/checker-test.log` |
| every package of `go list -tags test_dep ./tools/umpire/...` (16), one at a time, `GOFLAGS=-p=1 -timeout 30m` | all 16 PASS (export 187s, lower 451s incl. TestMigrationGoldens) | `.flow/tmp/fn113-12/full-results.txt`, `go-<pkg>.log` |
| `tools/umpire/model`: first run killed by the OS (`signal: killed`, memory pressure from other agents; no test output); re-run alone with `-v` | PASS, 799 RUN, `TestMigrationGoldens` PASS | `go-model.log`, `go-model-rerun.log` |
| `./common/testing/testpilot/...`, `./tools/canary/...` | not run: `go list -deps -test` shows neither imports `tools/umpire/model` or the checker | - |
| `GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS="./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/..."` (substitute for `lint-code-fast`, which is unusable on this branch: base rev far behind, ~718 unrelated findings) | 0 issues | `.flow/tmp/fn113-12/lint-code.log` |
| `gofmt -l tools/umpire/model/internal/checker` | clean | - |

`CC=/usr/bin/clang` does not exist here; runs used no CC prefix (no cgo).

No library was weighed (R25: none). The model gate was not run, as instructed.

### Review (independent reviewer: SHIP with two should-fix items; all applied)

The coordinator confirmed that "every test fixture" means valid tables.

1. Stale doc comments that described the deleted typed layer are reworded or dropped:
   - `compose.go` (`composing`);
   - `composekeys.go` (`ComposeSpec`, `ComposeTables`);
   - `keyclaims.go` (file header, `keyLevel`);
   - `table.go` (`TableSpec`, `FieldValues`, `Claims`, and the `NewTable` doc, which compared with "a declared
     machine's table" and named only unknown pairs and the refined field as what makes Err);
   - `claims.go` ("untyped part" on `PropertyDecl`/`ScenarioDecl`).

   `grep -i typed` over the checker's production files now finds nothing.
2. Deleted the ComposeTables no-start guard (`composekeys.go`, 3 lines). It cannot fire:
   - a `NewTable` member with no start fails earlier, at `member.model.Table()`;
   - a composed member's starts are a non-empty product;
   - a struct-literal table panics before reaching it.
3. `checkRows` now uses an actions set like the states set, so it is linear in rows.
4. Recorded, no change:
   - `ComposedStep.Moves` and `MemberMove` (`composekeys.go`) have no production reader. Only
     `TestAComposedStepNamesItsMemberMoves` reads them, which makes them a later cleanup candidate.
   - `checkRows` does not check `Key == rowKey(Source, Action)`. That is beyond the five agreed checks.

Re-run after the fixes (comments, one dead guard and a set; no behaviour change, so no heavy suite):

| command | result | log |
|---|---|---|
| `GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/internal/checker/` | PASS | `.flow/tmp/fn113-12/checker-test-review.log` |
| `GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model/internal/checker` | 0 issues | `.flow/tmp/fn113-12/lint-checker-review.log` |
| `gofmt -l tools/umpire/model/internal/checker` | clean | - |

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 0d0b72df04ab3dc20b519918e898fb14ae531aa4
- Tests: GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -v ./tools/umpire/model/internal/checker/ -> PASS (156 RUN before, 153 after), GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep -timeout 30m <each of 16 packages in ./tools/umpire/...> -> all PASS (model first run OOM-killed, re-run PASS incl. TestMigrationGoldens; lower PASS incl. goldens), GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS="./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/..." -> 0 issues (lint-code-fast unusable on this branch), (review fixes) GOFLAGS=-p=1 mise exec -- go test -count=1 -tags test_dep ./tools/umpire/model/internal/checker/ -> PASS, (review fixes) GOLANGCI_LINT_FIX=false mise exec -- make lint-code LINT_CODE_TARGETS=./tools/umpire/model/internal/checker -> 0 issues, independent review (claude-opus-5-5, fresh context): SHIP; should-fix items (stale comments, dead guard) and nits applied
- PRs: