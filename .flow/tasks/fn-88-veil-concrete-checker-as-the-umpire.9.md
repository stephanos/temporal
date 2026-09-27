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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
