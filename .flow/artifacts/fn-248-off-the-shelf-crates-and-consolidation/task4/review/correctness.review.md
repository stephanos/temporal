I’ll read the task specification first, then inspect the diff and trace the changed parser and code generation paths for correctness.

The focused-test evidence reports passing checks, and the new tests cover the main parser and mutation paths. I’m checking edge cases in path parsing and prost name conversion that those tests may miss.

Cargo cannot rerun the focused suite because this read-only sandbox prevents it from opening the build lock. I found a possible regression for generic type paths in attributes and am checking it against syn’s parser and the cached macro.

I confirmed two regressions: `context = Ctx<u32>` now panics during macro expansion, and a protobuf message named `Self_` now resolves to the invalid path `…::Self`. The latter comes from applying heck’s case conversion without prost’s identifier sanitization.

1. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/catalog/syntax/src/derive.rs:193`  
   **R-IDs:** [R1]  
   **Problem:** `value: syn::parse_quote!(#value)` reparses a type path as an expression. Previously accepted attributes such as `context = Ctx<u32>` now panic with “comparison operators cannot be chained.” Confirmed using the cached macro; the previous implementation parses this input successfully.  
   **Suggestion:** Construct `Expr::Path` directly from the parsed `Path`, preserving generic arguments. Add a generic-context regression control.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/catalog/syntax/src/derive.rs:675`  
   **R-IDs:** [R1]  
   **Problem:** `let message = message.to_upper_camel_case();` omits prost’s identifier sanitization. A valid protobuf message named `Self_` previously resolved correctly; it now becomes `…::Self`, which fails compilation. The changed realization code repeats this incomplete conversion at `realization.rs:477`. Its field conversion also turns `self_` into forbidden `r#self`. The build generator already handles these escapes.  
   **Suggestion:** Apply consistent prost-compatible sanitization after case conversion, with compile controls for `Self_` and `self_`.

Verification: five cached parser controls passed. Cargo verification was blocked by the read-only build lock; targeted cached-macro probes confirmed both regressions.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | partial | Crate adoption implemented; generic-path parsing and generated-name correctness need fixes. Remaining portions belong to tasks .5/.6. |
| R2 | deferred | Assigned to task .2. |
| R3 | deferred | Assigned to task .3. |
| R4 | deferred | Assigned to task .1. |
| R5 | deferred | Joined full gate belongs to task .7. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>