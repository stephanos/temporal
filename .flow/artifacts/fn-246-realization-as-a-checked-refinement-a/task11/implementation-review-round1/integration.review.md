I’ll read the task specification first, then trace the changed code’s integration with existing callers and generated code.
The registry generates typed request dispatch and evidence matching from the Model’s action inventory. I’m checking its interaction with capability traits, enum admission, and renamed facade support, where assumptions cross module boundaries.
The focused syntax test could not start: Cargo needs to write to a build directory, and this sandbox is read-only. I’m continuing with source tracing and the committed test evidence, particularly action coverage, descriptor type checks, and macro visibility.
I found an integration issue with opaque metadata: the existing Machine scanner searches every token for `reject!` and `stay!`, including the newly accepted payloads. A metadata payload containing `reject!(...)` therefore changes the generated outcome type even when no handler rejects. I’m checking the remaining changes for similar boundary failures.
The registry also conflicts with Reach’s existing admission contract. It always returns `Reject` for an empty witness path, while Reach requires an admitted root cursor before exploring any actions. A consumer delegating to this matcher therefore cannot explain a Reach observation. The new tests call it with prebuilt nonempty paths, so they miss this integration failure.
1. **Severity:** P1  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/temporal/annotate/syntax/src/registry.rs:429`  
   **R-IDs:** [R2, R3]  
   **Problem:** `check_registered` always returns `Reject` for an empty path. The unchanged `Runtime::reach` first calls `reach_evidence(observation, &[])` and immediately abandons exploration when that returns `Reject`. Consequently, consumers delegating to this registry—as the synthetic binding does—cannot explain any `Projection::Reach` observation. The tests supply prebuilt nonempty paths and bypass this admission step.  
   **Suggestion:** Provide explicit root admission for supported evidence, returning an appropriate continuation cursor. Add a synthetic test exercising the registry through `Runtime` with `Projection::Reach`.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/front/src/declare/action.rs:75`  
   **R-IDs:** [R2, R3]  
   **Problem:** Newly accepted opaque metadata still reaches the existing whole-module semantic scanner. For example, `#[metadata(example(reject!(NotFound)))]` sets `Machine.rejecting` because `declare::machine` recursively searches all tokens for `reject!`. This changes the generated outcome type from `()` to `umpire_dsl::Outcome`, its catalog, and transition expansion despite unchanged handlers. Metadata therefore affects Model semantics and can break callers requiring `Outcome = ()`.  
   **Suggestion:** Exclude metadata payloads from semantic macro detection, or inspect executable bodies. Add a regression proving opaque `reject!` and `stay!` tokens preserve native and lifted semantics.

Focused test execution was blocked by the read-only filesystem; these findings follow directly from the traced code paths.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Reach/frontier implementation belongs to other tasks. |
| R2 | partial | Typed Fill and synthetic consumers exist; root admission prevents Reach integration. |
| R3 | partial | Descriptor and entity/read seams exist; opaque metadata changes semantics. Remaining lens work is separately assigned. |
| R4 | deferred | Activity migration belongs to downstream tasks. |
| R5 | deferred | Protocol/refinement work belongs to other tasks. |
| R6 | deferred | Worker strategies belong to other tasks. |
| R7 | deferred | Mapping measurements and documentation belong to downstream tasks. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Introduced issues must be fixed.