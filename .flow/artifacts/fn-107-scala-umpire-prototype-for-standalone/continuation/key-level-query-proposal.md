# Proposal: key-level query surface over `*umpire.Table` (Phase A of one serial prerequisite)

Status: proposed and unrun. No Flow ID is minted. This is **Phase A** of the consolidated L prerequisite in `orchestration-recommendation.md`. It is not a task that can run in parallel with the composition proposal: Phase B depends on the unknown-pair and error contracts defined here.

## Title

Key-level Property/Scenario/Query and unknown-pair accounting in `model/go/umpire`

## Description

Task 3 must bind IR Properties, Scenarios, Queries (including `through`) and passive monitors to the reviewed generic search. The reviewed API admits only typed claims, so an IR-built `*Table` cannot reach it. Phase A adds additive key-level constructors that produce the existing `*PropertyDecl`, `*ScenarioDecl` and `*Query`, so `Query.Answer`, `Query.Watch` and `Query.Replay` run unchanged over a `NewTable` table. It also adds an explicit unknown-pair input on tables and precise explored-unknown accounting in search and progress. With that input, a hole is never read as a disabled pair, an explored hole makes an otherwise-verified answer incomplete, an unexplored hole changes nothing, and a violation is never downgraded.

Phase A adds no feature policy, adds no IR reading (goir stays with task 3), and changes no canonical Definition ID or fingerprint.

### Grounding (reviewed source, hashes per `source-manifest.json`)

- Claim internals are private. `PropertyDecl.when/holds/holds2` are at `model/go/umpire/claims.go:19-22`, `ScenarioDecl.free` at `:90`, and `Query.refinement/monitors` at `:164-165`. The only constructors are typed: `Machine.Property` `:37`, `Composition.Property` `:44`, `Scenario[S].Find/Verify/VerifyRefined` `:170-198`.
- A literal `&PropertyDecl{}` panics in the search at `search.go:277` (and `:270` for a transition claim) because `holds`/`holds2` are nil. It would also panic in `PropertyDecl.Lower` at `lower.go:76`.
- Callback errors are flattened. Monitor `Next` errors are rewrapped with `%v` at `search.go:245-246`, and `umpire.Error` (`table.go:228-237`) has no `Unwrap`, so a `*goir.Hole` loses its type. `Monitor.Violated` (`monitor.go:40`) and `After` (`:29`) return a bare `bool`.
- The search is hole-blind. `expand` reads only `Table.RowsFrom` (`search.go:188`), and `Row` documents "An absent pair is disabled" (`table.go:31`). goir keeps holes outside the table: `HoleRow` (`goir/machine.go:157-165`) and `Disabled` (`:202-209`). `TableSpec` (`table.go:246-261`) has no field to carry them.
- Progress is hole-blind. A deadlock is a region state with no row (`progress.go:317`), and `enabled` reads rows only (`:492-494`). `KeyProgress` from/to return a bare `bool` (`:51-56`), although the internal callback already returns `(bool, error)` (`:32`) and `discover` propagates errors unflattened (`:279-285`).
- A through-read needs `Refinement.MapValue` and `stepOfFn` (`refine.go:26-27`, read at `search.go:370-385` and `:424-430`). `RefineTables` sets only `MapState` (`refine.go:230`). A key-level through Query over a `RefineTables` refinement would therefore hit a nil `MapValue`, or a nil `stepOfFn` at `refine.go:371-373`.
- The product identity already includes monitor history through `productKey.mons` = `identity(n.mons)` (`search.go:99-108`, `monitor.go:139-145`). `Query.Replay` rebuilds through `searcher()` (`replay.go:157-199`), so a key-level Query inherits replay once it is constructible.
- `Query.check` requires `Property.Machine == Scenario.Machine` by interface equality (`claims.go:260`). A table-backed `Model` must therefore be one stable instance per table. `Model` is the public interface `Name()/Table()` (`machine.go:20-23`).

## Touches

`[model/go/umpire/**]`

Expected files: `table.go`, `claims.go`, `search.go`, `monitor.go`, `refine.go`, `progress.go` and `lower.go`, plus new `keyclaims.go` and `keyclaims_test.go` (names are suggestions). Existing test files are read, not rewritten.

## Proposed additive API (names indicative; the implementer may rename within the package's idiom)

1. **Table-backed Model.** `(*Table).Model() Model` returns one cached instance per table (an unexported field on `Table`, not serialized). `Name()` is `t.Machine` and `Table()` returns `t`.
2. **Unknown pairs.** `TableSpec.Unknown []UnknownPair` and `Table.Unknown`. `UnknownPair{Row, Source, Action string; Cause error}` is a state and class whose steps are unknown. `NewTable` indexes it as `UnknownFrom(state)`. `Reachable`, `Stuck`, `Rows`, `IDs()`, `TargetSemantic()` and fingerprints are **unchanged**: they never read `Unknown` (`canonical.go` behaviorJSON reads rows, starts and catalogs only). `NewTable` rejects the following with an `*Error`, never a panic, as a declaration error:
   - an unknown pair whose key is also a row;
   - a duplicate unknown pair;
   - a source outside `States`;
   - an action outside `Actions`.
3. **Error preservation.** `Error` gains an unexported cause and `Unwrap() error`. Every place a callback error is rewrapped (`search.go:246`, and new key-level paths) keeps the existing `Message` text and wraps the cause, so `errors.As(err, *goir.Hole)` works and existing message assertions stay byte-identical.
4. **Key-level claims.** Each constructor declares its name on a claim-name registry kept by the table-backed Model (the same `claimNames` type, `names.go:10-27`). A duplicate kind and name on one table is reported as a declaration error, surfaced by `Query.check` (or by constructors that return `(decl, error)`; the implementer chooses one). The same name on two tables is admitted:
   - `KeyProperty(t *Table, name string, when func(action string) bool, whenLabel string, holds func(step Result) (bool, error)) *PropertyDecl`. A nil `when` means every step.
   - `KeyTransitionProperty(t, name, holds func(before string, step Result) (bool, error)) *PropertyDecl`.
   - `KeyScenario(t, name, start string, actions ...string) *ScenarioDecl`, which is pinned by composed or class keys.
   - `KeyFreeScenario(t, name, start string) *ScenarioDecl`.
   - `KeyFind` / `KeyVerify(name string, p *PropertyDecl, s *ScenarioDecl, limits Limits) *Query`.
   - `KeyVerifyRefined(name, p, s, ref *Refinement, limits) *Query`. Here `ref` comes from `RefineTables(scenarioTable, propertyTable, spec)`, and `checkRefined` (`search.go:63-81`) applies unchanged.

   `PropertyDecl` stores the key-level callbacks in new unexported fields. `observe` (`search.go:258-282`) dispatches typed or key-level. `IsTransition`/`Triggers` cover both. A key-level Property passed to `Lower` returns a located `*Error` ("a key-level Property is searched and verified, never realized"), never a nil call.
5. **Key-level through reading.** `RefineTables` records the product catalogs it already reads (unexported) and marks the refinement key-level. For a key-level transition or same-step Property read through it, `before` is `MapState(beforeKey)`. The step is `Result{Outcome: res.Outcome, State: MapState(res.State), Facts: mapFacts(res.Facts, dst.Facts), Because: res.Because}`, which is SEMANTICS Claims' "a state by its map, an outcome and facts by name" and the same fact rule `refinementDecl.stepOf` applies (`refine.go:63-72`). Typed `VerifyRefined` behavior is unchanged. A key-level Property through a typed refinement, or the reverse, is a declaration error.
6. **Key-level monitors.** `KeyMonitor(name, initial string, next func(mon, before string, step Result) (string, error), violated func(mon string) (bool, error), at Evaluation) *Monitor` and `AfterKey(func(Result) (bool, error)) Evaluation`. Existing `NewMonitor`, `After` and the exported `Monitor` fields keep their meaning. `Query.Watch` accepts both kinds.
7. **Unknown-evidence classifier.** `Query.Unknown func(error) bool`. It is exported and optional, and nil means every callback error aborts. A callback error the classifier accepts (task 3 passes `errors.As(err, *goir.Hole)`) does not abort. It records an unknown **edge** (kind `claim`), and that successor is not enqueued, because its claim or monitor state is unknown. Any other callback error aborts `Answer` with a wrapped cause.
8. **Explored-unknown accounting on `Answer`.** Answer gains two things:
   - `Unknown []UnknownReach{Kind ("row"|"claim"), Row, Source, Action string; Depth int; Prefix *Trace; Cause error}`. It is deduplicated by (kind, row) in discovery order, and `Prefix` is the shortest product path to `Source` that `Table.Replay` accepts.
   - `Expanded int`, the product states whose successors were read.

   A `row` unknown is recorded only when `expand` reads a node with `pos < depth()` whose Scenario schedules that pair (`scheduled`, `search.go:229-231`). The following never produce an entry: pairs at nodes cut by the depth bound (`:184`), nodes never expanded because the search stopped (found, counterexample-precedence or `exceeded`, `:152-177`), and unscheduled classes.
9. **Answer semantics under unknowns (additive `Answer.Incomplete() bool`).**
   - `CounterexampleFound` and `Found` are authoritative whatever `Unknown` holds. Unknowns are listed and the outcome is not downgraded.
   - `VerifiedWithinLimits` or `NotFound` with a non-empty `Unknown` is incomplete (`Incomplete()==true`). The outcome string is unchanged, so the Lean outcome spellings (`claims.go:200-209`) stay the same.
   - `LimitReached` stays `LimitReached` and lists what it explored.
   - With an empty `Unknown`, every existing outcome, witness, `Explored`, `Rows`, `Exercised` and monitor verdict is byte-identical to today.
10. **Progress under unknowns.** Both constructors are additive:
    - `KeyProgressFunc(name, from, to func(state string) (bool, error), within, assumptions...)`.
    - `ProgressAnswer.Unknown []UnknownReach`, the explored region states' unknown pairs.

    The rules:
    - A region state with an unknown pair is not a deadlock. The deadlock verdict stays unestablished there. It becomes incomplete if no other deadlock is found.
    - A deadline witness is a concrete path of rows and stays authoritative.
    - A fair-cycle witness is not reported as `CounterexampleFound` when some fair class has an unknown pair at a cycle state and the cycle does not take that class. That verdict is incomplete instead.
    - Verified verdicts with explored unknowns are incomplete. Unknowns outside the explored region, or beyond `Steps`, change nothing.
    - `CheckProgress` over a table with no unknown pairs is byte-identical to today.

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

## Focused tests (proposed, in `model/go/umpire`, `require` only, hand-built key tables; no IR fixtures)

- `TestKeyLevelQueryMatchesTypedQuery` (table-driven over the existing typed machines in `umpire_test.go`/`monitor_test.go`: find, verify, transition, pinned, free)
- `TestKeyLevelMonitorsKeepHistoriesApart` and `TestKeyLevelMonitorNeverSuppressesACounterexample` (twins of `monitor_test.go:33` and `:51`)
- `TestKeyLevelThroughQueryReadsByMapAndName` (twin of the typed `VerifyRefined` test over `RefineTables`)
- `TestCallbackErrorsKeepTheirType`
- `TestAnExploredUnknownPairMakesAVerifyIncomplete`, `TestUnexploredUnknownPairsChangeNothing` (depth-cut, unscheduled, after-found), `TestAnUnrelatedUnknownDoesNotDowngradeACounterexample`, `TestADisabledPairIsNeverUnknown`
- `TestAClassifiedClaimErrorIsAnUnknownEdge`
- `TestProgressReadsUnknownPairs` (deadlock, deadline, fair-cycle cases)
- `TestMalformedUnknownPairsAreRejected`, `TestKeyLevelPropertyIsNotLowered`
- `TestKeyLevelClaimNamesAreScopedPerTable` (duplicate on one table rejected; the same name on two tables admitted, and `PropertyID` unchanged, documenting S5)

## Prerequisites and ordering

- Depends on task 14, whose reviewed APIs this extends. It does not need task 15 or 16 source. Its Touches are disjoint from task 16's (`model/scalav2/**`, `Makefile`, `model/scala/umpire/*`).
- Phase B (composition) depends on this phase's `UnknownPair`, `(*Table).Model()` and error wrapping.
- Task 3 depends on the whole prerequisite.

## Out of scope

- IR reading, hole classification and the receipt formats (task 3).
- SEMANTICS text.
- Canonical claim IDs (S5 stays documented and unresolved).
- Lowering key-level claims to Testpilot.
- Any change to typed constructors.
