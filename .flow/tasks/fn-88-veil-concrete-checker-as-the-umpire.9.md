---
satisfies: [R8, R9, R15]
---
# fn-88-veil-concrete-checker-as-the-umpire.9 Differential test, witness order, and the Caller, Pair, and three-instance pins

## Description
Add the differential test that runs every Query reachable through `AdmittedQuery.search` on both backends via `AdmittedQuery.searchWith`, with the comparison the spec's API Contracts define; prove witness order; pin the Caller and Pair product-state counts and the three-instance fixture. Adopt mode only.

**Size:** M
**Files:** `model/Umpire/Search/Tests/Differential.lean` (new), `model/Umpire/Search/Tests/Fixtures.lean` (three-instance and same-depth fixtures), `model/Umpire/Search/Tests.lean` (register), `model/Temporal/Feature/Nexus/Caller/Tests.lean`, `model/Temporal/Feature/Nexus/Pair/Tests.lean`
**Touches:** [model/Umpire/Search/Tests/Differential.lean, model/Umpire/Search/Tests/Fixtures.lean, model/Umpire/Search/Tests.lean, model/Temporal/Feature/Nexus/Caller/Tests.lean, model/Temporal/Feature/Nexus/Pair/Tests.lean]

### Approach
- Enumerate Queries from the callers of `AdmittedQuery.search` and `searchWithIntent` (`model/Umpire/Search/Admission.lean:137-145`), not from Search fixture tests, which call `Umpire.search` directly and stay on `reference`.
- Comparison exactly as the spec's API Contracts state: equal outcome; equal witness `Scenario.Trace`; equal Plan bytes except `explored`; equal receipt JSON except the four backend fields, every `SearchStats` counter, `triggers`, and for a `verify` with a counterexample `validity.searchComplete`, `searchTermination`, and `coverage`. Where `reference` reports `limit-reached`, check only that a `veil` witness replays and a `veil` complete result contradicts no trace `reference` examined.
- R15: a fixture product where two paths reach one product state at the same depth; assert the lexicographically smaller path is the witness on both backends.
- R9: `#guard` the exact product-state counts for the Caller protocol machine (beside the 158-state pin, `Caller/Tests.lean:109`) and the Pair model; a synthetic three-instance fixture where `reference` reports `limit-reached` at `search = 32768` and `veil` reports `complete`; record wall times in evidence.

### Investigation targets
**Required:**
- `model/Umpire/Search/Admission.lean:137-145`; `model/Umpire/Search.lean:493-501, 849, 1116-1142`
- `model/Temporal/Feature/Nexus/Caller/Tests.lean:109`; `model/Temporal/Feature/Nexus/Pair/Tests.lean`

**Optional:**
- `model/Umpire/Search/Tests/Fixtures.lean`

### Key context
- Exempt only the listed fields; an unlisted difference is a finding, not a new exemption.

### Carried from fn-88.3 review (2026-09-27)
- R14's differential in `Umpire/Search/Tests/Product.lean` reaches only the Umpire test roots (Switch, Search fixture, parameterized Model). Apply its public `Umpire.SearchTests.Product.productAgrees` to the Caller and Pair Scenarios (and any other Temporal feature Scenario this task's modules reach) within their Limits, beside the product-state pins, so `ScenarioAutomaton.admits = CheckedScenario.admits` is checked trace-by-trace on them.
- The automaton lowers `ordering`/`adjacencies` when `actionsExactly`/`traceExactly` pins the schedule (every `scenario`-command Scenario does); only free-schedule `ordering`/`adjacencies` is `Unsupported`. Plan the selection reasons and R18 flip list against that.

### Carried from fn-88.3 and fn-88.4 (2026-09-27)
- Run the monitor agreement check (`Umpire/Search/Tests/Monitor.lean`, evaluator vs monitor under both endings) and the product check (`productAgrees`, fn-88.3) over the Temporal feature Models too: Nexus Caller, Pair, Control, Success, Workflow Start/Outage, which Umpire tests cannot import.
- Replace the three copies of the Property ordering with one shared helper.
- Rename `Product.Product` and re-enable `linter.extra.dupNamespace` in `Umpire/Search/Product.lean`.

## Acceptance
- [ ] Differential test over every `AdmittedQuery.search` Query passes with exactly the stated exemptions
- [ ] Same-depth fixture proves witness order on both backends
- [ ] Caller, Pair, and three-instance pins pass with exact product-state counts; wall times in evidence
- [ ] New test modules registered; `make lint-model` passes
- [ ] Defer mode: closed as not applicable citing the R1 receipt identity, nothing added

## Done summary
Added the fn-88 R8 differential: every Query reachable through `AdmittedQuery.search` and `searchWithIntent` now runs on both backends through `AdmittedQuery.searchWith`, compared with exactly the API Contract exemptions. Also added the R15 witness-order fixture, the R9 Caller, Pair and three-instance pins, and the carried items.

- **Differential** (`model/Umpire/Search/Tests/Differential.lean`, registered in `Umpire/Search/Tests.lean`). Where the reference terminates, the outcome, witness, Plan bytes except `explored`, and receipt JSON except the exempt fields must all be equal. Where the reference reports `limit-reached`, `veil` must not be `invalid` and must not contradict an endpoint the reference examined. `examinedEndpoints` re-decides each such endpoint, and a negative control rejects a `verified` answer after an unresolved endpoint.
- **Query corpus.** `sweep` covers every `query` declaration. Multi-instance Queries are re-admitted from the arguments their declaration passes to `checkInstances`, so a search that selected nothing is still compared. Also covered: the Switch's hand-admitted and Property-only Queries, every Variations point, the Promotion variant, a Replay re-admission, the migration tests' relocated admissions, and every campaign candidate of every exploratory set. The Caller's 889 targets are split over `TemporalModelTests/SearchDifferential/CallerCampaign1-4`. All 35 Temporal declarations and every candidate agree. Two Caller class-member candidates and two Success instance Queries are rejected before search, and the pins list them.
- **Scenario and monitor checks** (R14, R6) on every swept Query. The Temporal Models are read to cost-bounded depths, which the pins print.
- **R15.** Two paths reach one product state at depth one, and both backends report `a · c`: reference 4 paths, `veil` 3 states.
- **R9.** Caller `terminalHolds`: 4 states. Caller whole protocol machine within `four`: `veil` 171 states against 3525 reference paths. Pair `twoAsync` verify: 7 states. Three-instance fixture: reference `limit-reached` at 32768, `veil` verified with 8 states.
- **Carried items.**
  - `Product.Product` is renamed `StateSpace`, and `linter.extra.dupNamespace` is re-enabled.
  - The three Property orderings are now one helper, `CheckedProperty.sortedById`. The `Search.lean` line pin moved from 1428 to 1427.
  - `Monitor.productAgrees` was generalized off the Switch `LawStatement`.

Deviations and follow-ups:
- **Spec amendments.** The API Contracts now name the receipt `explored` exemption, the same counts as the Plan's exempt `explored`; each run's Plan and receipt must still agree. R6 now records the cost-bounded depths on interleaved-instance Models.
- **Files outside Touches:** `Umpire/Query.lean`, `Search.lean`, `Search/Product.lean`, `Product/Monitor.lean`, `Backend/Veil.lean`, `Tests/Product.lean`, `Tests/Monitor.lean`, `Tests/Replay.lean` (all carried items), and the new `TemporalModelTests/SearchDifferential*` modules. Umpire tests cannot import Temporal.
- **Vocabulary rename after SHIP.** The regression gate caught retired vocabulary. Commit 11663988d6 renames the helpers only; regression is green after it.
- **lint-model.** The whole-model builtin `lake lint` step was OOM-killed. The per-module builtin lint over every touched module is clean.

Defer mode: not applicable (R22 adopt).

stage: impl-review - ran [2026-09-27..2026-09-28] (codex fan-out NEEDS_WORK x3 draws; re-review NEEDS_WORK on caller corpus; SHIP at b0867578a2)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6df703a26bcf928f0750cb437494a662c43f1e6c, e37b7f9aba41cb8bf7e9d821f8fb72eba2ce5fb5, c8e7ae4e00317701a23d77cacec5b1f74905dec6, b0867578a2ebe4a51d9babe30a43bace83711323, 11663988d6c4a7d41f01b8b613e6360b7b1cf005
- Tests: baseline: green (cd model && mise exec -- lake build Umpire.Search Umpire.Search.Product Umpire.Search.Selection Umpire.Search.Tests Umpire.Search.VisibilityTests, pre-edit), cd model && mise exec -- lake build Umpire.Search.Tests TemporalModelTests Temporal.Feature.Nexus.Pair.Tests Temporal.Feature.Nexus.Caller.Tests (green at b0867578a2 and 11663988d6), make umpire-check-goldens (green, byte-identical), make umpire-check-regression (green at 11663988d6, 1735s; the prior run caught retired vocabulary in Differential.lean, fixed in 11663988d6), LEAN_NUM_THREADS=1 make lint-model: import-graph, module-index, Batteries lint passed; the whole-model builtin lake lint step INCONCLUSIVE (OOM-killed, signal 9, peak footprint about 20 GB); lake --wfail lint --builtin-only run per module over every touched module (Umpire.Query, Umpire.Search, Search.Product, Product.Monitor, Backend.Veil, Search.Tests and its Differential/Fixtures/Monitor/Product/Replay, TemporalModelTests and SearchDifferential + CallerCampaign1-4, Caller.Tests, Pair.Tests): all clean, wall times (recorded, not asserted): Umpire.Search.Tests.Differential build 6-8s incl. the three-instance fixture (reference 32768 paths limit-reached, veil 8 states verified); TemporalModelTests.SearchDifferential 225-430s; CallerCampaign1-4 74-311s each in parallel; Caller.Tests 437-532s; Pair.Tests 54-79s, impl-review codex: fan-out NEEDS_WORK (3 findings) -> re-review NEEDS_WORK (1) -> SHIP at b0867578a2
- PRs: