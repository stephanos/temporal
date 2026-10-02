I’ll read the task specification first, then inspect the diff and affected callers for correctness and logic issues.
The main change adds differential tests for the two search backends. I’m checking whether the comparisons enforce the task’s contract, especially for reference searches that hit their limit, and whether the query sweep covers every caller.
The focused Lean build is blocked: the sandbox is read-only, and the pinned Lean 4.32.0 toolchain is not installed. I’m continuing with source inspection, focusing on comparison gaps and the caller paths omitted from the sweep.
I found three gaps in the new test oracle: it omits some admitted queries, truncates monitor checks below the declared limits, and cannot detect a false “verified” result after the reference observed an unresolved endpoint. The state-count and witness-order pins are present.
1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:135`  
   **R-IDs:** [R8]  
   **Problem:** `limitDifference` checks satisfiability, coverage, and fired clauses, but loses unresolved endpoint evidence. A partial `verify` can hit the reference limit after observing an unresolved endpoint. An erroneous Veil `verified-within-limits` result with matching coverage and triggers passes this comparison, despite contradicting that endpoint. `PlanResult` does not retain the reference’s `PlanningObservations.unresolved`, so this helper cannot enforce the required check.  
   **Suggestion:** Retain reference observations or evaluate its examined candidates when validating a complete Veil result. Add a negative control that rejects verification after an unresolved reference endpoint.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:288`  
   **R-IDs:** [R6]  
   **Problem:** The new monitor comparison reduces the tested depth using `deepestWithin depth 5000` and a second 100000-work budget. The Pair golden explicitly records Query depth 3 and clause-table depth 1, although its declared limit is 6. The subsequent `Monitor.productAgrees query view depth` covers only Scenario-permitted paths and the Query’s selected ending; it does not restore exhaustive model-trace coverage under both endings. This leaves the carried monitor-agreement requirement incomplete.  
   **Suggestion:** Check the required traces through their declared limits under both endings, or supply monitor-equivalence proofs that cover the omitted cases.

3. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:452`  
   **R-IDs:** [R8]  
   **Problem:** The registry sweep covers `query` declarations, while the manual additions omit other existing `AdmittedQuery.search` callers. For example, `Search/Tests/Admission.lean:99` searches the unconstrained `switch.query.property-only` query, and `Tests/MigrationCompatibility.lean:207` and `:211` search separately admitted targets. None is included in this differential. The property-only query also exercises a different Scenario from the three listed Switch queries.  
   **Suggestion:** Inventory the direct admission/search callers alongside registry declarations and run each omitted query through both `AdmittedQuery.searchWith` backends.

## Requirements coverage

Coverage below concerns this task and its carried obligations.

| R-ID | Status | Evidence |
|---|---|---|
| R6 | partial | Temporal monitor checks added, but depth caps omit required traces/endings. |
| R8 | partial | Differential implemented; incomplete caller inventory and limit-result validation. |
| R9 | met | Caller 171-state, Pair 7-state, and three-instance 32768-path/8-state pins added. |
| R14 | met | Scenario automaton and decoded accepted-path comparisons added for Temporal queries. |
| R15 | met | Same-depth fixture pins `a · c` on both backends. |

Unaddressed R-IDs: [R6, R8]

Verification: `lake build Umpire.Search.Tests Umpire.Search.VisibilityTests` could not run: the pinned toolchain is unavailable and the read-only sandbox returned `Operation not permitted`. Findings are based on source inspection.

Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":["R6","R8"]}
```

<verdict>NEEDS_WORK</verdict>