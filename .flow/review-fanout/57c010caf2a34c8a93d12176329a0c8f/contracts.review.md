I’ll read the task specification first, then review the diff and related code for contract and consistency issues.
The new comparison adds a receipt exemption beyond the spec’s list. I’m also tracing whether the query sweep covers multi-instance searches that hit their limit before selecting a Plan.
The sweep skips `twoCompletionsCutShort` even though it reaches `AdmittedQuery.search` and returns `limit-reached`. That leaves a required boundary case untested. I also attempted the focused Lean build, but the sandbox rejected it with `Operation not permitted`; verification is limited to source inspection.
1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:332`  
   **R-IDs:** [R8]  
   **Problem:** `| .error error => "not admitted: " ++ admissionFailure error` skips multi-instance Queries whose search returned `.notSelected`. These Queries were admitted and searched. For example, `twoCompletionsCutShort` reaches `AdmittedQuery.search`, returns `limit-reached`, and is explicitly pinned as skipped in the new Temporal sweep. R8 requires comparing this case across backends.  
   **Suggestion:** Retain or reconstruct the product admission before searching, then compare both backends even when no Plan is selected.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:70`  
   **R-IDs:** [R8]  
   **Problem:** The receipt exemption list adds `"explored"`. The specified comparison exempts `explored` from **Plan bytes**, but does not exempt it from receipt JSON; the task explicitly prohibits adding exemptions. Consequently, the oracle silently accepts an additional difference while claiming the exact contractual comparison.  
   **Suggestion:** Match the specified receipt exemptions and expose the resulting discrepancy. If receipt exploration counts also require exemption, reconcile that explicitly in the contract rather than silently broadening the test.

3. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:256`  
   **R-IDs:** [R6]  
   **Problem:** Temporal monitor agreement uses `scenarioTraces`, which stops extending a trace as soon as `behavior.admitsPrefix` rejects it. `Monitor.productAgrees` also follows the constrained Scenario. Thus these checks omit Model traces outside the authored schedules, whereas R6 requires exhaustive agreement over every trace within the Model’s Limits. The new whole-machine Caller search pin does not run monitor agreement on those omitted traces.  
   **Suggestion:** Run the monitor/evaluator comparison over unpruned Model traces within the declared Limits, using the existing `allTraces`/`modelAgrees` helpers, or provide the required equivalence theorem.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R6 | partial | Temporal agreement added, but trace enumeration is Scenario-pruned. |
| R8 | partial | Differential implemented; skips an admitted bounded Query and broadens receipt exemptions. |
| R9 | met | Caller 171-state, Pair 7-state, and three-instance 8-state completion pins added. |
| R15 | met | Same-depth fixture checks the `a · c` witness on both backends. |

Unaddressed R-IDs: [R6, R8]

The focused Lean build could not run: the sandbox returned `Operation not permitted`. Findings are based on source inspection.

Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":["R6","R8"]}
```

<verdict>NEEDS_WORK</verdict>