I’ll read the task specification first, then review the diff and trace how the new gates interact with existing build and validation commands.
The root Makefile now delegates to separate framework and Temporal gates. I’m checking path handling, guest output wiring, and the split controls against the old recipes to see whether existing callers retain the same behavior.
I found a path-handling risk in the new delegation: Make changes into `model/umpire` before the gate resolves a relative `CARGO_TARGET_DIR`. That can place the output under the framework source tree, where the new boundary guard rejects it. I’m verifying that path and checking for other regressions in the boundary policy.
1. **Severity:** P1  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `Makefile:1272`  
   **R-IDs:** [R7]  
   **Problem:** `CARGO_TARGET_DIR=relative-target make model-check` now resolves the target beneath `model/umpire` because delegation changes directory first. The boundary guard then fails with `output target overlaps framework source inventory`. Previously, guest builders and host consumers resolved this override from the repository root. The compiled boundary stage passes from the repository root and reproduces this failure from `model/umpire`.  
   **Suggestion:** Resolve caller-relative paths before changing directory, or invoke the framework bootstrap and executable from the repository root.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/dev/xtask/src/gate/boundary.rs:134`  
   **R-IDs:** [R7]  
   **Problem:** The organization exception accepts `other/github.com/temporalio/umpire`, since `/` passes the preceding-character check. The existing policy explicitly rejects that token. Placing it in `README.md` therefore escapes the independent framework guard; the unchanged Temporal scanner excludes Markdown.  
   **Suggestion:** Preserve the existing occurrence-level token-boundary rule when recognizing the organization exception.

FYI — **Long Method:** `Gate::stage` begins with `fn stage(&self, name: &str) -> Result<i32>` and contains the complete command dispatch. This heuristic does not affect the verdict.

Normal boundary and retired-name stages passed using the existing compiled gate. Full gates were not run.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R7 | partial | Framework ownership and root delegation are implemented, but caller-relative target handling and the exact name-policy exception have defects. Directory-only full proof remains assigned to task .11. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>