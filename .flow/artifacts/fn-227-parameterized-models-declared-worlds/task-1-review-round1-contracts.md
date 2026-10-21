I’ll read the task specification first, then review the diff for agreement between the implementation, tests, and documented promises.
The task covers native World support and suite ownership; source lifting and TLC agreement are explicitly assigned downstream. I’m checking generic bound propagation, callback signatures, and retained World ownership against those promises.
The Quick command cannot run because Cargo’s build cache is read-only in this sandbox. I’ll continue tracing the code and use focused compile checks if the cached artifacts allow them.
Two focused compile checks found gaps in the generic-support promise: merging actors can generate an illegal default-parameter order, and copying a Model’s `Self` bound changes its meaning to World. Both checks pass after removing the problematic default or spelling the Model type explicitly.
1. **Severity:** P2 · **Confidence:** 100 · **Classification:** introduced  
   **File:Line:** `model/umpire/dsl/macros/src/machine/declared.rs:54`  
   **R-IDs:** [R1]  
   **Problem:** Merging actor generics preserves defaults without preserving Rust’s required ordering. Valid declarations `User<T = u8>` and `Clock<U>` in a `Model<T, U>` generate `enum Action<T = u8, U>`, which fails compilation: “generic parameters with a default must be trailing.” The default-generics tests cover Model defaults but miss combined actor defaults.  
   **Suggestion:** Remove defaults from the generated carrier’s merged parameters while preserving the authored actor declarations. Add a two-actor regression control.

2. **Severity:** P2 · **Confidence:** 100 · **Classification:** introduced  
   **File:Line:** `model/umpire/dsl/macros/src/machine/declared.rs:338`  
   **R-IDs:** [R1]  
   **Problem:** Declaration bounds are copied without resolving their original `Self`. For `Model<T> where T: Config<Self>`, the generated Machine implementation interprets the constraint as `T: Config<World<T>>`. The Associated implementation still requires `T: Config<Model<T>>`, so the generated implementations disagree and compilation fails. This contradicts the promise to preserve authored generic bounds.  
   **Suggestion:** Resolve declaration-level `Self` to the originating Model, World, or actor type before merging bounds, as already done for implementation bounds. Add declaration-bound regression tests.

Both failures reproduced through focused compiler checks using cached DSL artifacts; corresponding controls passed. The Quick command was blocked by Cargo’s read-only cache lock.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Native World, adapters, starts and suite binding delivered; generic failures above remain. Shared TLC fixture explicitly deferred. |
| R2 | deferred | Action-input task. |
| R3 | partial | Native typed association delivered; source lifting and TLC binding explicitly assigned downstream. |
| R4 | deferred | Activity retry task. |
| R5 | deferred | Activity timeout task. |
| R6 | deferred | Realization and live-proof task. |
| R7 | deferred | Parent documentation and timing completion. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Fix the two introduced generic-compilation failures.
