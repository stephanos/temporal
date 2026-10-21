I’ll read the task spec first, then trace the changed DSL and suite code through their callers and focused tests.
The changes separate state-to-World association from implicit World construction and retain explicit Worlds through the suite APIs. I’m checking generic bounds and the existing path and claim adapters for assumptions that may still depend on unit Worlds.
The Quick test command cannot run in this sandbox: Cargo cannot acquire its build lock on the read-only filesystem. I’m continuing with source tracing and checking whether the available compiled artifacts can support focused compiler probes.
Focused compiler probes found two failures in the new generic machine path: the action carrier’s `Debug` implementation lacks a required bound, and readonly checks copy Model bounds into nested modules where their names may not resolve. I’m verifying the locations and narrowing the fixes.
1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/dsl/macros/src/machine/declared.rs:630`  
   **R-IDs:** [R1]  
   **Problem:** The generated action carrier calls `.field(value)` under only `Actor` bounds, but `Actor` does not require `Debug`. A valid generic machine with `Model<T>` storing `PhantomData<T>` and `User<T: Default>` storing an input `T` fails with E0277—even when instantiated with `u8`. The existing Action derive correctly makes its own Debug implementation conditional; the carrier loses that condition.  
   **Suggestion:** Give the carrier’s Debug implementation separate conditional bounds requiring each actor type to implement Debug. Add a passing generic fixture whose actor stores a type parameter absent from the state’s value fields.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/dsl/macros/src/machine.rs:269`  
   **R-IDs:** [R1]  
   **Problem:** `impl #state_implementation __Readonly ... #state_constraints` copies Model’s generic bounds into every nested module containing a mutable method. Those bounds resolve in the nested module’s scope. For `Model<T: Token>`, adding an unrelated nested `Counter::inc(&mut self)` causes E0405 because `Token` exists only in the parent module. Valid helper code therefore breaks generic machine expansion.  
   **Suggestion:** Keep the Model-specific probe implementation in Model’s lexical scope and reference it from nested guards. Add a passing fixture with a locally named generic bound and an unrelated mutable helper in a child module.

The Quick suite could not run because Cargo’s build lock is on a read-only filesystem. Focused compiler type-check probes using available compiled artifacts reproduced both failures; corresponding controls passed.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Native World, dispatch, starts and suite controls added; generic failures above remain. Source/TLC proof deferred to task .7. |
| R2 | deferred | Assigned to task .2. |
| R3 | partial | Typed native association and owned suite binding delivered; source/backend work assigned downstream. |
| R4 | deferred | Assigned to task .3. |
| R5 | deferred | Assigned to task .4. |
| R6 | deferred | Assigned to task .5. |
| R7 | deferred | Assigned to task .6. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>
