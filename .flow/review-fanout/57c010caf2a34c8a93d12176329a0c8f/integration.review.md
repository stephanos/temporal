I’ll read the task specification first, then review the diff and its integration with unchanged callers and consumers.
The new comparison exempts receipt `explored` counts, which the task contract does not list as an exemption. I’m checking the query inventory and the protocol change’s consumers for other integration gaps.
The query sweep skips `twoCompletionsCutShort` because its declared result is an error, although that query reaches search and returns `limit-reached`. It also omits manually constructed queries used by Promotion and Variations. Verification is blocked here: Lake cannot initialize, and Go cannot create its build directory in the read-only sandbox.
1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:332`  
   **R-IDs:** [R8]  
   **Problem:** The differential skips queries that successfully reach search but produce `.notSelected`. For example, `twoCompletionsCutShort` runs through `checkInstances → check → checkAdmitted → admitted.search`, returns `limit-reached`, and is then merely printed as “not admitted.” The registry-only sweep also omits manually constructed queries, including Promotion’s `nonFoundQuery` and Variations’ lowered query passed to `searchWithIntent`. These existing integration paths therefore never receive the required backend comparison.  
   **Suggestion:** Reconstruct admissions before search outcomes become `.notSelected`, and supplement the registry sweep with the queries constructed by existing `search` and `searchWithIntent` callers.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/Umpire/Search/Tests/Differential.lean:70`  
   **R-IDs:** [R8]  
   **Problem:** Adding `"explored"` to `exemptReceiptFields` silently broadens the comparison contract. The task exempts `explored` from **Plan bytes**, but explicitly limits receipt exemptions to the listed backend fields, `SearchStats` counters, triggers, and conditional verification fields. Receipt `explored` comes from `PlanningMetadata`, not `SearchStats`. This masks an actual difference: the new Caller pin compares 3,525 paths against 171 states.  
   **Suggestion:** Enforce the specified receipt allowlist and expose the mismatch. If receipt exploration counts should also be exempt, reconcile that explicitly in the contract before treating this comparison as satisfying R8.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R8 | partial | Differential infrastructure exists, but skips reachable queries and adds an unauthorized exemption. |
| R9 | met | Caller and Pair count assertions and the three-instance 32,768-path/eight-state fixture are present. |
| R15 | met | Same-depth fixture asserts the identical `a → c` witness on both backends. |

Unaddressed R-IDs: [R8]

Verification could not run: Lake initialization and Go’s temporary build-directory creation failed under the read-only sandbox. Assertion results remain unverified.

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R8"]}
```

<verdict>NEEDS_WORK</verdict> - Introduced differential coverage and comparison-contract gaps must be fixed.