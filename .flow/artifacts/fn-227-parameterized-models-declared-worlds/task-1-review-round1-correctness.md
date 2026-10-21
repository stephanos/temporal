I’ll read the task specification first, then trace the changed code and focused tests for correctness and edge cases.
The task separates native World support from the later source/TLC work. I’m checking native dispatch, generic parameter handling, and whether suite searches and retained runs use the supplied World throughout.
The Quick test command stopped at Cargo’s build lock because the sandbox is read-only. I’ll continue with code inspection and focused compiler checks using the existing build artifacts.
Focused compiler checks reproduced two failures in the new generic code: the action carrier’s `Debug` implementation lacks a required bound, and merging actors with defaulted parameters can produce invalid Rust. I’m checking the exact locations and confirming that the reproductions use otherwise valid declarations.
Two introduced defects remain in generic expansion.

**1. Missing bound on the action carrier’s Debug implementation**

- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/umpire/dsl/macros/src/machine/declared.rs:630`
- **R-IDs:** [R1]
- **Problem:** `.field(value)` requires each actor to implement `Debug`, but the generated constraints provide only `Actor`, which has no `Debug` supertrait. A valid generic actor such as `enum User<T: Default> { Replace(#[inputs([T::default()])] T) }` therefore fails with E0277 inside `#[machine]`. The actor derives successfully by itself; its conditional Debug bound is lost at the carrier.
- **Suggestion:** Add per-actor Debug bounds specifically to the carrier’s Debug implementation. Add a control with generic payload fields whose value traits are supplied conditionally by the derive.

**2. Merging actor defaults can generate invalid generic declarations**

- **Severity:** P2
- **Confidence:** 100
- **Classification:** introduced
- **File:Line:** `model/umpire/dsl/macros/src/machine/declared.rs:54`
- **R-IDs:** [R1]
- **Problem:** `target.params.push(parameter)` retains defaults while combining actor parameters. Valid declarations `Model<T, U>`, `World<T, U>`, `User<T = u8>` and `Clock<U>` generate `Action<T = u8, U>`, which Rust rejects because defaulted parameters must be trailing. A focused compiler reproduction fails; removing the actor default makes the same model compile.
- **Suggestion:** Normalize defaults when synthesizing combined generic lists, preserving the original authored declarations. Cover actors using different subsets of Model parameters.

Verification: both failures were reproduced with compiler checks against existing build artifacts. Ten cached focused tests passed. The Quick command could not rebuild because Cargo’s build lock is on a read-only filesystem.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Native World dispatch, computed starts and refusals are covered; generic expansion has the defects above. Shared TLC evidence is assigned downstream. |
| R2 | deferred | Action domains belong to subsequent work. |
| R3 | partial | Typed association and owned suite binding are implemented; source lifting is explicitly deferred. |
| R4 | deferred | Activity retry changes are outside this task. |
| R5 | deferred | Activity timeout changes are outside this task. |
| R6 | deferred | Live realization changes are outside this task. |
| R7 | deferred | Final DESIGN.md and milestone evidence belong to later completion. |

Unaddressed R-IDs: [R1]

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R1"]}
```

<verdict>NEEDS_WORK</verdict>
