I’ll read the task specification first, then review the diff and trace the changed code paths for correctness.

The differential test exempts `explored` from receipt comparisons, but the spec only exempts it from Plan bytes. I’m also checking whether the query sweep covers searches that return no selected plan and whether the limit-reached checks enforce the required comparison.

The sweep misses some real search callers: it skips the two-instance query that reaches its limit, and it does not include the Promotion or Variations queries created outside `query` declarations. The focused Lake build was blocked by the read-only sandbox, so I’m verifying these findings from the code.

1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:70`  
   **R-IDs:** [R8]  
   **Problem:** `"explored"` adds an unauthorized receipt exemption. The contract exempts `explored` from **Plan bytes**, but its exhaustive receipt exemption list excludes it. This masks actual differences: the new Caller pin compares 3,525 reference paths with 171 Veil states while declaring the receipts equivalent.  
   **Suggestion:** Remove this receipt exemption and surface the resulting differences against the specified contract.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:332`  
   **R-IDs:** [R8]  
   **Problem:** The differential skips queries that actually reached `AdmittedQuery.search`. `twoCompletionsCutShort` successfully admits and searches, then `checkAdmitted` returns `.notSelected`; `checkedLine` treats this as an admission failure and never runs the comparison. The new golden explicitly accepts that omission. Additionally, the registry-only sweep misses non-declaration callers such as `PromotionTests.nonFoundRun` and the Variations `searchWithIntent` queries. Thus “every Query” coverage is incomplete precisely around unsuccessful searches and transformed queries.  
   **Suggestion:** Retain or reconstruct admissions before result selection, including multi-instance queries, and explicitly register the remaining `search`/`searchWithIntent` callers for comparison.

3. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:256`  
   **R-IDs:** [R6]  
   **Problem:** The carried Temporal monitor check is weaker than the required model-wide differential. It tests only `query.form.properties` on `scenarioTraces`, which stops extending any rejected prefix. Unlike the existing `Monitor.modelAgrees`, it neither enumerates all bounded model traces nor exercises the generated table covering every supported clause kind. The additional `Monitor.productAgrees` call remains restricted to the same Scenario. Consequently, “monitors ok” does not establish the required exhaustive agreement on the Temporal models.  
   **Suggestion:** Apply the existing model-wide monitor oracle to the Temporal models within their Limits, including its generated clause coverage, or supply the required equivalence proofs.

The focused Lake build could not run: the sandbox returned `Operation not permitted`. Findings above are established from code inspection.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R6 | partial | Temporal checks omit model-wide trace and clause coverage. |
| R8 | partial | Differential exists, but expands exemptions and omits search callers. |
| R9 | met | Caller, Pair, and three-instance outcome/count pins implemented. |
| R14 | met | Scenario acceptance and decoded product paths compared for swept queries. |
| R15 | met | Same-depth fixture pins `a · c` on both backends. |

Unaddressed R-IDs: [R6, R8]

Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":["R6","R8"]}
```

<verdict>NEEDS_WORK</verdict>