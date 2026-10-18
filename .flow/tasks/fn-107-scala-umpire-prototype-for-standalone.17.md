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
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: