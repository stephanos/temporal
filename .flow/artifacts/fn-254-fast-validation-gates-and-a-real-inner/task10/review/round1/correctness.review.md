I’ll read the task specification first, then trace the changed gate code and focused test evidence for correctness and requirement coverage.
The split puts framework stages in the framework CLI and leaves Temporal stages in the root gate. I’m checking the mixed-control migration and path handling, where the new orchestration could change coverage or artifact selection.
The focused evidence covers stage failures, cache bypass, backend modes, and guest-driver independence. I’m now checking that the coverage ledger matches the actual commands and that the new source scanner handles existing framework inputs correctly.
I found a concrete gap in the new source-path guard: a valid Rust double-quote character literal can make the lexer skip a following `include_str!`, allowing an include outside the framework root to pass this check. The committed source/proof hashes match the checkout, and the changed shell scripts pass syntax checks.
I also reproduced a caller-path regression in the new Make delegation. With `CARGO_TARGET_DIR=relative-target`, the framework stage passes when launched from the repository root, but fails with “output target overlaps framework source inventory” after the recipe changes into `model/umpire`. The relative path needs to be resolved before that directory change.
1. **Resolve caller-relative paths before changing directories**
   - **Severity:** P1
   - **Confidence:** 75
   - **Classification:** introduced
   - **File:Line:** `Makefile:1272`
   - **R-IDs:** [R7]
   - **Problem:** `CARGO_TARGET_DIR=relative-target make model-check` now resolves the target beneath `model/umpire`, because the recipe changes directories before bootstrap or gate admission. The new inventory guard then rejects it with `output target overlaps framework source inventory`. I reproduced the same compiled stage passing from the repository root and failing from the recipe’s working directory. Relative `CARGO_HOME` also loses its original caller base.
   - **Suggestion:** Resolve inherited caller-relative paths before either delegation changes directories, then pass those absolute paths to both gate halves.

2. **Character literals can hide escaping source includes**
   - **Severity:** P1
   - **Confidence:** 100
   - **Classification:** introduced
   - **File:Line:** `model/umpire/dev/xtask/src/gate/boundary.rs:281`
   - **R-IDs:** [R7]
   - **Problem:** The lexer treats the double quote inside the valid character literal `'"'` as a string opener. For example:

     ```rust
     const QUOTE: char = '"';
     const TEXT: &str = include_str!("../../../../external.txt");
     ```

     It consumes `include_str!(` inside that apparent string, so `source_paths()` never checks the include. An existing file outside the framework can therefore pass the new containment guard and remain a dependency that breaks directory-only operation.
   - **Suggestion:** Recognize character and byte-character literals before string literals. Add a control proving that an escaping include following either literal is rejected.

FYI — **Long Method:** `fn tokens(text: &str) -> Vec<Token>` implements roughly 90 lines of lexical rules; finding 2 identifies a concrete correctness consequence.

Verification: committed source/proof hashes matched, changed shell scripts passed syntax checks, and the caller-directory failure was reproduced without running the full suite.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R7 | partial | Framework delegation and driver independence are implemented; caller-path preservation and source containment have the failures above. Directory-only full execution remains assigned to task .11. |

Unaddressed R-IDs: [R7]

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R7"]}
```

<verdict>NEEDS_WORK</verdict>