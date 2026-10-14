I’ll read the task spec first, then review the diff for agreement between the implementation, tests, and documented contracts.
The diff adds a shared differential checker and registers it for both Umpire and Temporal queries. I’m checking whether its comparisons enforce the stated exceptions and whether the query sweep covers all search callers.
The focused Lean build is blocked: the checkout needs Lean 4.32.0, but only 4.33.1 is installed, and the sandbox prevents fetching it. The new monitor tests also explicitly reduce coverage below the queries’ declared limits; I’m checking that against the task’s coverage requirement.
Two introduced coverage gaps need fixing.

1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:407`  
   **R-IDs:** [R8]  
   **Problem:** The sweep enumerates `Registry.queries`, but queries created programmatically also reach `AdmittedQuery.search`. For example, `ReplayTests.dropped?` searches `restrictSource softAfterHard.source [1]` through `admitKept → QuerySource.admit → checkAdmitted`. Only the original `softAfterHard` declaration gets compared. Exploration campaign queries and the property-only query in `Search/Tests/Admission.lean:99` are also absent from the supplemental comparisons. Consequently, the claimed differential over every reachable query misses existing callers.  
   **Suggestion:** Include the replay edits, campaign-generated admissions, and property-only admission in the differential corpus, running each through both backends.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:288`  
   **R-IDs:** [R6]  
   **Problem:** `deepestWithin depth 5000` and `deepestWithin ownDepth 100000` reduce monitor coverage below the declared Limits. The new Pair golden explicitly reports query depth 3 and clause-table depth 1, while its Limits permit six steps. R6 requires evaluator agreement for every supported clause kind over every model trace within its Limits, unless proved. `Monitor.productAgrees` does not close this gap: it checks only the query’s own properties on scenario-filtered paths under its selected ending. The existing monitor proofs explicitly do not prove evaluator agreement.  
   **Suggestion:** Complete the comparisons through the declared depth, using streaming or partitioned enumeration if necessary, or supply the permitted agreement theorem.

The focused command `lake build Umpire.Search.Tests Umpire.Search.VisibilityTests` could not run: Lean 4.32.0 is unavailable locally, and the sandbox prevents fetching it.

## Requirements coverage

Task requirements and explicitly carried requirements:

| R-ID | Status | Evidence |
|---|---|---|
| R6 | partial | Temporal monitor comparisons added, but depth caps omit required traces. |
| R8 | partial | Declared queries compared; programmatically generated queries remain uncovered. |
| R9 | met | Caller 171-state, Pair 7-state, and three-instance 32768-path/8-state assertions added; execution unverified here. |
| R14 | met | Scenario automaton comparisons registered for Temporal queries. |
| R15 | met | Same-depth fixture pins `a → c` on both backends. |

Unaddressed R-IDs: [R6, R8]

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R6","R8"]}
```

<verdict>NEEDS_WORK</verdict>