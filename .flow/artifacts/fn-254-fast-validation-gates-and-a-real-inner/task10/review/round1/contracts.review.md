I’ll read the task specification first, then review the diff and focused evidence for contract and consistency issues.
The diff moves framework stages into the framework CLI and leaves Temporal stages in the outer workspace. I’m checking the coverage mapping and command dependencies against the previous gate to see whether the split preserves the promised checks and timing behavior.
The evidence correctly leaves full combined validation and the directory-only proof to later tasks. I’m now checking a few narrower contracts: stage dependency output, split control coverage, and whether boundary checks accept valid Rust source references.
I found two nonblocking contract mismatches: the framework name check accepts prefixed organization paths that the recorded policy rejects, and `gate --list` names `proto-tools` as a prerequisite without listing it as a stage. Source and proof hashes match the index, and changed shell and Python files pass syntax checks. I have not rerun suites that create temporary files in this read-only session.
1. **Severity**: P2  
   **Confidence**: 100  
   **Classification**: introduced  
   **File:Line**: `model/umpire/dev/xtask/src/gate/boundary.rs:134`  
   **R-IDs**: [R7]  
   **Problem**: The organization exception accepts `other/github.com/temporalio/umpire` because `/` passes `!is_ascii_alphanumeric()`. The recorded policy explicitly rejects this embedded organization path. Placing it in README content passes the new scanner, violating R7’s required name rejection.
   **Suggestion**: Preserve the existing token-boundary rule from `model/temporal/testing/boundary.rs`, including its embedded-path negative controls.

2. **Severity**: P2  
   **Confidence**: 100  
   **Classification**: introduced  
   **File:Line**: `model/umpire/dev/xtask/src/gate.rs:22`  
   **R-IDs**: []  
   **Problem**: `gate --list` reports three stages depending on `proto-tools`, but never lists `proto-tools` itself. The advertised dependency graph therefore contains an undefined prerequisite and cannot describe the full gate for scheduling.
   **Suggestion**: Register and list `proto-tools`, executing it exactly once through that registry.

Changed shell/Python syntax checks passed, and source/proof hashes match the committed index. Runtime proof logs were inspected; suites requiring writes were not rerun.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Historical/final evidence belongs to .1/.8. |
| R2 | deferred | Quick command belongs to .6. |
| R3 | deferred | TLC batching belongs to .2. |
| R4 | deferred | Runner/consolidation belongs to .3/.4/.7. |
| R5 | deferred | Parallel composition belongs to .5. |
| R6 | deferred | Final measurements belong to .8. |
| R7 | partial | Gate split and CLI independence delivered; required name rejection has finding 1. Standalone proof remains assigned to .11. |
| R8 | deferred | Budgets belong to .12. |

Unaddressed R-IDs: [R7]

Both findings are P2; the verdict follows the explicit exception for an unmet R-ID rejection requirement.

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R7"]}
```

<verdict>NEEDS_WORK</verdict>