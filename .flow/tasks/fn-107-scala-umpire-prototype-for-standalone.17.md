---
satisfies: [R3, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.17 Key-level claim, query and composition seams over IR-built tables

## Description
**Touches:** [model/go/umpire/**]

Add additive key-level constructors so IR-built `*Table`s reach the reviewed search, monitor, through-refinement, replay, progress and composition algorithms. Carry holes as explicit unknown pairs that are never read as disabled. Preserve error types across callbacks. Bound composition by a mandatory ceiling. Change no typed constructor, Lean-parity table, Definition ID or fingerprint. Add no feature policy and no IR reading (goir stays with task 3). This is generic support only; task 3 carries the R4, R5 and R9 binding.

The design is the two reviewed proposals under `.flow/artifacts/fn-107-scala-umpire-prototype-for-standalone/continuation/`, consolidated by `orchestration-recommendation.md` into one serial task because both phases edit `table.go` and share the unknown-pair and error contracts. Read all three before starting. Their `source-manifest.json` hashes and `.flow/tmp/fn-107/` prep files no longer exist; verify each cited line against the current source instead.

**Size:** L

### Approach
- **Phase A** (`key-level-query-proposal.md`, items 1-10): the table-backed `Model`, `Unknown` pairs on `TableSpec`/`Table`, `Error.Unwrap`, key-level Property, Scenario, Query and through-refinement, `KeyMonitor`/`AfterKey`, the claim-error classifier, explored-unknown accounting with `Incomplete()`, and the progress unknown rules. Go test-first per item; key-level twins of the existing typed tests are the parity oracle.
- **Phase B** (`key-level-composition-proposal.md`): the shared non-generic composition core, `ComposeTables`, `ComposedStep`, `RefinedField`, unknown propagation, ceilings before allocation, and replacement via `RefineTables(…, CoverStarts:true)`. `compose_test.go` stays unmodified; parity twins are added.
- Phase A lands with its tests green before Phase B starts. Preserve existing comments. Do not edit typed tests. Keep every existing message byte-identical.
- The `ends`/`visible` Kind guard (`end_visible_fix.md`) lives in `model/scalav2/goir/**` and belongs to task 3, not here.

### Investigation targets
**Required:** `model/go/umpire/{table,claims,search,compose,refine,progress,canonical,lower}.go` and their tests; the three continuation documents above.
**Optional:** `model/scalav2/goir/machine.go` (the consumer task 3 will write), read-only.

### Quick commands
`CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/...`; `make lint-code-fast` (or the scoped `make lint-code LINT_CODE_TARGETS=./model/go/umpire` when the global one is red for unrelated reasons). The Lean-dump parity tests skip in this checkout; that is expected.

## Acceptance
- [ ] Every existing `model/go/umpire` test passes unmodified. Lean-parity tables, `Table.IDs()`, `TargetFingerprint()`, `PropertyID`/`ScenarioID` (`canonical.go:231-234`), `QueryCanonical` and typed-constructor behavior are unchanged. No existing exported identifier changes signature.
- [ ] Key-level twins of the typed monitor, find/verify, through and replay tests reproduce the typed answers exactly: outcome, `Explored`, `Rows`, witness atoms, `Exercised`, `Monitors`, and `Query.Replay` success, over a `NewTable` copy of the typed table.
- [ ] Monitor history stays in product identity. A key-level Query with a monitor whose states differ across commuting paths explores strictly more product states than without it and returns the same table rows. A monitor never removes a counterexample.
- [ ] Callback errors keep their type. A test error type returned from `holds`, `next`, `violated`, `after`, progress `from` or `to` is recovered with `errors.As` from `Answer`/`CheckProgress`, with no `%v` flattening.
- [ ] Hole ≠ disabled:
  - An explored scheduled unknown pair makes a verify `Incomplete()`.
  - The same pair at depth ≥ `Steps`, unscheduled by a pinned Scenario, or behind a found witness produces no entry and leaves the answer complete.
  - A counterexample found on another branch stays `CounterexampleFound` with the unknown listed.
  - A disabled pair (absent row and not unknown) never appears in `Unknown`.
- [ ] A classified claim error is an unknown edge, not an abort. An unclassified one aborts with its cause.
- [ ] Progress: a deadlock whose only pairs are unknown is not reported. A deadline violation next to an unrelated unknown stands. An unfair-by-unknown cycle is not a counterexample.
- [ ] `NewTable` rejects malformed unknown pairs as `*Error`. `Lower` on a key-level Property is a located error.
- [ ] Every reported key-level witness and unknown `Prefix` replays with `Table.Replay`, and every key-level answer replays with `Query.Replay`.
- [ ] Every existing `compose_test.go` test and every typed-composition answer is unchanged: table, IDs, `TargetFingerprint`, starts, assumptions, and collision and replacement errors.
- [ ] Parity twins: for each typed composition in `compose_test.go`, `ComposeTables` over `NewTable` copies of its members yields equal:
  - `States`, `Actions`, `Outcomes`, `Facts`, `StateFields`, `Starts`, `Ends`, `Rows` (keys, order, results and facts), `Assumptions`;
  - `IDs()` and `TargetFingerprint()`.
  The twins cover a refining member with `RefinedField` set.
- [ ] Composition starts are the full Cartesian product of every member `Table.Starts`, with the last member fastest. The replacement check covers every opaque start (twin of `compose_test.go:162`), and a violating provider is rejected with a replayable witness (twin of `:149`).
- [ ] With a tight `States` or `Evaluations` ceiling, a `*ComposeLimitError` carries the ceiling and count, and a start product above the ceiling is refused before its list is allocated. The tenfold probe completes under a sufficient ceiling and is limit-refused under an insufficient one, never truncated.
- [ ] Unknown propagation: a member unknown pair yields a composed unknown pair. A sync whose other move is disabled stays disabled. Exploration does not pass through unknown pairs. A Phase A key-level verify over the composed table becomes `Incomplete()` only when that composed pair is explored.
- [ ] `Ends` callback errors propagate with their type (`errors.As`).
- [ ] Separator collisions in member keys are rejected as today (twin of `compose_test.go:200`).
- [ ] `go test -tags test_dep ./model/go/...` passes with every pre-existing test unmodified.
- [ ] No exported identifier changes signature or meaning, and no file outside `model/go/umpire/**` changes.


## Done summary
Tables built outside the package now reach the reviewed search, monitor, through-refinement, replay, progress and composition algorithms through additive key-level constructors, with holes carried as unknown pairs that no check reads as disabled. Typed constructors, existing tests, messages, Definition IDs and fingerprints are unchanged; nothing is committed, and the work sits uncommitted in the tree for the owner.

Files changed (before-copies under `.flow/tmp/fn-107/task17-before/model/go/umpire/`): `table.go`, `claims.go`, `search.go`, `monitor.go`, `refine.go`, `progress.go`, `lower.go`, `replay.go`, `names.go`, `compose.go`.
Files added: `keyclaims.go`, `composekeys.go`, `keyclaims_test.go`, `composekeys_test.go` (all in `model/go/umpire/`).

Decided differently from the proposals:
- `NewTable` keeps its signature, so it cannot return the malformed-unknown-pair error. The table carries it: `(*Table).Err()` exposes the `*Error`, and `Model().Table()`, every key-level Query, `CheckProgress`, `RefineTables` and `ComposeTables` return it instead of reading the table.
- Progress lists the unknown pairs of every state whose steps the check read, not only region states. A reachable hole outside the region can hide a From state, so reporting such a check as complete would read the hole as disabled.
- A key-level Monitor leaves the exported `Next` and `Violated` nil; its functions are unexported and `Query.Watch` dispatches on them.
- A duplicate key-level claim name fails the Queries that name it (`Query.check`) and `Check(table.Model())`. A caller must declare each claim once per table.
- `TableSpec.RefinedField` must name a state field, `RefineTables` rejects a nil `MapState`, and `ComposeTables` rejects a member with no start and a refinement that replaces nothing. These were panics or silent no-ops.
- Tests were written after the implementation, not before. Each new behavior was then checked by mutation: 22 mutants fail a new test.

Not done, for task 3 or a follow-up:
- `Table.Stuck` and `Check` still read a state whose only pairs are unknown as stuck, as the proposal specifies (no identity or reachability reads `Unknown`).
- `RefineTables` checks rows only. A refining table's unknown pairs do not qualify a refinement or a composition replacement; the caller must consult `Table.Unknown`.
- The run's gate receipt was not written: `flowctl gate receipt` refuses a tree dirty outside the ignore set (`mise.toml`).



### Review round 1

The Codex review returned NEEDS_WORK with three P1 findings. All three were valid: each reproduced as a failing test before any fix. `set.go` now changes too (before-copy saved); the new tests are in the added file `model/go/umpire/keyunknown_test.go`. No pre-existing test changed.

1. **A reachable hole passed refinement (valid).** `RefineTables` now returns a `*RefinementError` of the new kind `RefinementIncomplete` when a start of the refining table reaches an unknown pair. It carries the shortest replayable path to the pair's state as `Witness` and keeps the pair's cause for `errors.As`. `ComposeTables` returns it unchanged. A row that refines nothing is still reported first, and an unreachable unknown pair changes nothing. This supersedes the "RefineTables checks rows only" item under "Not done" above.
   - Red before the fix: `TestRefineTablesStopsAtAReachableUnknownPair` (`keyunknown_test.go:52`, "Expected nil, but got: &umpire.Refinement{...}") and `TestComposeTablesReplacementStopsAtAReachableUnknownPair` (`keyunknown_test.go:82`, "Expected nil, but got: &umpire.Table{...}").
2. **An unknown Monitor erased a violation on its own step (valid).** A step on which a Monitor function fails with a classified error is now kept as a terminal node when what was read of it already answers the Query: the Property failed, an earlier Monitor was violated, or a find's claim was realized. The unknown is still listed and the search does not continue past the node. The unread Monitor reports the new verdict `MonitorUnknown` with its state before the step, and `Query.Replay` accepts such a witness.
   - Red before the fix: `TestAnUnknownMonitorDoesNotEraseAViolationOnItsStep`, all six subtests for `next`, `violated` and `after` (`keyunknown_test.go:152`, actual `verified-within-limits`, expected `counterexample-found`) and the find case (`keyunknown_test.go:169`, actual `not-found`, expected `found`).
3. **`Check(q)` accepted an incomplete verify (valid).** `checkQuery` now rejects an answer whose `Incomplete()` is true, with `query <name>: <answer> is incomplete: the search explored <n> unknown, the first the <kind> '<row>'`. Typed Queries have no unknowns, so their results and messages are unchanged.
   - Red before the fix: `TestCheckRejectsAVerifyThatExploredAnUnknown` (`keyunknown_test.go:181`, "An error is expected but got nil").

One test fixture of mine was wrong on the first green run (a "stray" row that was in fact a stutter); I corrected the fixture, not the code. Reverting each fix on its own turns its test red again (six mutants; a seventh did not compile).

Gates after the fixes: full `go test` rc=0, `go vet` rc=0, scoped `make lint-code` rc=0 (0 issues).

### Review round 2

The Codex re-review confirmed the round-one fixes and returned two P2 findings. Both were valid and reproduced as failing tests before any fix. No pre-existing test changed; tests this task added were updated where the fixes changed their inputs (the ceiling now has three bounds, and one malformed-pair case now reports the key mismatch).

1. **Composition products were built before the ceiling was checked (valid).** `ComposeCeiling` gains a third required bound, `Results`, beside `States` and `Evaluations`; this differs from the proposal's two-field ceiling. Three products are now counted by size before any of them is built: the members' starts (already), two members' synchronized classes, and the results of one composed step. Composed actions count against `Evaluations`, since every action is evaluated at a start. A refusal is a `*ComposeLimitError` with the ceiling and the count, and no table. Typed `Compose` has no ceiling and takes none of these checks; its tables are byte-identical to the pre-task sources.
   - Red before the fix: `TestComposeTablesRefusesAProductBeforeBuildingIt` (`keyunknown_test.go:242`): the class case built all 90000 actions and then reported `Needed: 2`, and the results case built the table and returned no error (`:252` likewise).
   - The test counts allocations (under 1000 for a product of 90000), so it cannot pass by building the product.
2. **`NewTable` accepted an unknown pair keyed as something other than its row (valid).** A pair whose `Row` is not `<source>-<action>` is now the table's `*Error`. The field is validated, not derived: the proposals name it, and `goir` already supplies exactly this key.
   - Red before the fix: `TestAnUnknownPairIsKeyedAsItsRow` (`keyunknown_test.go:194`, "An error is expected but got nil").

Reverting each fix on its own turns its test red again (four mutants). Lint flagged one complexity finding after the first fix pass (`collectActions`), resolved by splitting out the own-class loop.

Gates after the fixes: full `go test` rc=0, `go vet` rc=0, scoped `make lint-code` rc=0 (0 issues).

### Review round 3

Codex pass three accepted the `Results` bound and both round-two fixes, and returned one P2 finding. It was valid and reproduced as a failing test before the fix. No pre-existing test changed.

1. **A step that does not happen still expanded an earlier member's results (valid).** `stepFrom` now looks up every move's row before expanding any result (`movesFrom`), and `admitsResults` uses the same lookup. A disabled or unknown later move builds no part of the product; an unknown one still becomes a composed unknown pair by the existing rule.
   - Red before the fix: `TestComposeTablesExpandsNoResultOfAStepThatDoesNotHappen` (`keyunknown_test.go:293`): "30086 is not less than 1000" with the second move disabled, "15081" with it unknown.
   - The test asserts fewer than 1000 allocations beside a 5000-result move, under `Results: 1`.

**Audit of the composition core** for anything sized by member data and built before a ceiling or existence check:
- **Found and fixed, same test:** a later move whose row exists but has no result. Its product is zero, so the bound admitted it, and the earlier move's results were expanded and then discarded ("30090 is not less than 1000" before the fix). `movesFrom` now treats a result-less row as a step with no result.
- **Checked, already bounded:** the start product (counted before listing); synchronized class products in either member order and across several syncs (counted cumulatively before listing); own classes (counted one by one); result products when every move has a row (counted before expansion); the state set and queue (counted before kept); the split map (one entry per admitted result at most); composed unknown pairs (one per admitted evaluation at most). The two first-move cases of the new test pin the reverse member order.
- **Checked, linear in one member's own data, no product:** class lists, row and unknown lookups, catalogs, assumptions, and the replacement check.
- **Tidied, no behavior change:** the round-one reachable-unknown check ran one path search per unknown pair; it now reads the table's reachable set first and searches once. The existing reachable and unreachable tests cover it.
- **Not bounded by the ceiling, by design:** typed `Compose` (no ceiling; its tables are byte-identical to the pre-task sources) and the size of the member tables themselves, which the caller builds.

Gates after the fix: full `go test` rc=0, `go vet` rc=0, scoped `make lint-code` rc=0 (0 issues).

### Review round 4

Codex pass four confirmed the allocation fix and the `refine.go` tidy, and returned one P2 finding. It was valid and reproduced as a failing test before the fix. No pre-existing test changed.

1. **A result-less move paired with an unknown one was reported unknown (valid).** A synchronized step one of whose moves has a row with no result is now disabled in either member order, and a key-level verify over that composition stays complete.
   - Red before the fix: `TestAResultlessMoveDisablesAStepWhoseOtherMoveIsUnknown`, both orders (`keyunknown_test.go:323`, "Should be empty, but was [{s_s-meet-0-0 ...}]").

**Audit of the rule** "absent row = disabled; row with no result = disabled; unknown pair = unknown; disabled dominates unknown; unknown dominates enabled". It was decided in six places with separate logic. It is now one helper in `table.go`: `(*Table).pair(state, action)` returns disabled, unknown or enabled, the three ordered so that a step needing every move takes the least and a state needing any pair takes the greatest (`(*Table).steps(state)`).

| Place | Before | Now |
|---|---|---|
| Composition, unknown propagation (`unknownMove`) | result-less row read as available (the finding) | removed; `movesFrom` reads every move through `pair` |
| Composition, results and ceiling (`movesFrom`, `admitsResults`, `stepFrom`) | own check, already correct | same `movesFrom` |
| Progress deadlock | result-less row read as a step | `steps`; such a state is a deadlock |
| Progress fairness (`enabled`, `disabled`) | result-less row read as enabled | `pair` |
| Progress deadlock replay (`deadlockAt`) | result-less row read as a step | `steps` |
| `Table.Stuck` | result-less row read as a step | `steps`; still ignores unknown pairs, the deferral the reviewer accepted |
| `Table.Replay` | a step by a result-less row reported "no result of that row" | reported "not enabled", through `pair` |

   - Red before these: `TestARowWithNoResultIsADisabledPair`, four subtests (`keyunknown_test.go:351` Stuck was "done"; `:357` deadlock was verified; `:369` fair cycle was verified; `:382` replay message).
   - **Not changed, already consistent:** search (`expand` iterates results, so a result-less row yields no successor, and lists unknown pairs separately); `RefineTables` and `carrierOf` (iterate results); the reachable-unknown check; composition own classes and syncs in both orders, which all go through `movesFrom`; the replacement check, which is `RefineTables`. `CoverageTargets` lists rows as targets and decides nothing about enablement; I left it.
   - **Meaning change to note:** for a key table holding a row with no result, `CheckProgress`, `Progress.Replay`, `Table.Stuck` and `Table.Replay` now read that row as disabled. No typed table, typed composition or `goir` table contains such a row (each skips an empty step list), so typed behavior is byte-identical; only hand-built specs are affected.

Reverting each part of the rule on its own turns a test red (six mutants). Lint flagged one finding after the first pass (identical switch branches in `deadlockAt`), resolved.

Gates after the fix: full `go test` rc=0, `go vet` rc=0, scoped `make lint-code` rc=0 (0 issues).

Review round 5 (conductor): the reviewer returned SHIP with one P3, fixed by the conductor: `CoverageTargets` no longer lists a row with no result as a target (`coverage.go`; test `TestARowWithNoResultIsADisabledPair/an_exploration_has_no_target_in_it`, red without the fix).

For task 3: `ComposeCeiling` has three required bounds (`States`, `Evaluations`, `Results`); a reachable unknown pair in a refining table makes `RefineTables` return a `*RefinementError` of kind `RefinementIncomplete`; `Check(q)` rejects an incomplete verify; build each key-level claim once per table; `Table.Stuck` still ignores unknown pairs.

The work is uncommitted; the owner makes the commits. The task diff and the five review outputs are under `.flow/tmp/fn-107/task17/`.

stage: implement - ran (worker subagent, session model claude-opus-5-5; four fix rounds)
stage: impl-review - ran (codex:gpt-5.6-sol:high, one session 01a0f520-3ccc-7410-8424-60c13497bfb0 over the uncommitted task diff; rounds 1-4 NEEDS_WORK with 3, 2, 1 and 1 introduced findings, all fixed red-then-green; round 5 SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: green (CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... rc=0, pre-edit), CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc=0; model/go/umpire 114 passing tests and subtests, none failed or skipped), CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/go/... (rc=0), CC=/usr/bin/clang mise exec -- make lint-code-fast GOLANGCI_LINT_FIX=false (rc=0, 0 issues, 5m03s; auto-fix disabled so the linter cannot rewrite files outside model/go/umpire), CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/go/umpire GOLANGCI_LINT_FIX=false (rc=0, 0 issues), differential: typed compositions dumped (JSON table, TargetSemantic, IDs, StateFields, Assumptions, Starts, typed steps) under the saved before-sources and the new sources are byte-identical (108609 bytes each), mutation: 22 mutants of the new behavior each fail at least one new test; two were not scored (removing the start-product check makes the test allocate 2^40 start vectors; dropping the member moves does not compile), flowctl gate receipt --gate unittest: NO_RECEIPT (worktree dirty outside the ignore set: mise.toml, not this task's) - no receipt written, Review round 1 red (before the fixes): CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/umpire/ rc=1 - TestRefineTablesStopsAtAReachableUnknownPair (keyunknown_test.go:52), TestComposeTablesReplacementStopsAtAReachableUnknownPair (keyunknown_test.go:82), TestAnUnknownMonitorDoesNotEraseAViolationOnItsStep (keyunknown_test.go:152 x6, :169), TestCheckRejectsAVerifyThatExploredAnUnknown (keyunknown_test.go:181), Review round 1 green: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc=0; model/go/umpire 126 passing tests and subtests), Review round 1: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/go/... (rc=0), Review round 1: CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/go/umpire GOLANGCI_LINT_FIX=false (rc=0, 0 issues), Review round 1 mutation: reverting each of the three fixes individually fails its test (6 mutants; 1 not scored, did not compile), Review round 2 red (before the fixes): CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -timeout 120s ./model/go/umpire/ rc=1 - TestComposeTablesRefusesAProductBeforeBuildingIt (keyunknown_test.go:242, :252), TestAnUnknownPairIsKeyedAsItsRow (keyunknown_test.go:194), Review round 2 green: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc=0; model/go/umpire 130 passing tests and subtests), Review round 2: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/go/... (rc=0), Review round 2: CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/go/umpire GOLANGCI_LINT_FIX=false (rc=0, 0 issues), Review round 2 differential: typed composition dump under the final sources is byte-identical to the dump under the saved before-sources, Review round 2 mutation: reverting each fix individually fails its test (4 mutants), Review round 3 red (before the fix): CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -timeout 120s ./model/go/umpire/ rc=1 - TestComposeTablesExpandsNoResultOfAStepThatDoesNotHappen (keyunknown_test.go:293; 30086, 15081 and 30090 allocations against a bound of 1000 for the disabled, unknown and result-less second move), Review round 3 green: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc=0; model/go/umpire 136 passing tests and subtests), Review round 3: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/go/... (rc=0), Review round 3: CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/go/umpire GOLANGCI_LINT_FIX=false (rc=0, 0 issues), Review round 3 differential: typed composition dump under the final sources is byte-identical to the dump under the saved before-sources, Review round 3 mutation: 2 mutants (result-less row expands; unreachable unknown stops refinement) each fail a test, Review round 4 red (before the fix): CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 -timeout 120s ./model/go/umpire/ rc=1 - TestAResultlessMoveDisablesAStepWhoseOtherMoveIsUnknown both orders (keyunknown_test.go:323), TestARowWithNoResultIsADisabledPair four subtests (keyunknown_test.go:351, :357, :369, :382), Review round 4 green: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc=0; model/go/umpire 144 passing tests and subtests), Review round 4: CC=/usr/bin/clang mise exec -- go vet -tags test_dep ./model/go/... (rc=0), Review round 4: CC=/usr/bin/clang mise exec -- make lint-code LINT_CODE_TARGETS=./model/go/umpire GOLANGCI_LINT_FIX=false (rc=0, 0 issues), Review round 4 differential: typed composition dump under the final sources is byte-identical to the dump under the saved before-sources, Review round 4 mutation: 6 mutants of the pair rule each fail a test, conductor final: CC=/usr/bin/clang mise exec -- go test -tags test_dep -count=1 ./model/go/... ./model/scalav2/... (rc 0), conductor final: go vet -tags test_dep ./model/go/... (rc 0), conductor final: make lint-code LINT_CODE_TARGETS=./model/go/umpire GOLANGCI_LINT_FIX=false (rc 0, 0 issues), codex impl-review rounds: .flow/tmp/fn-107/task17/t17-r1..r5.md (final SHIP)
- PRs: