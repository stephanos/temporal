I’ll read the task spec first, then review the changed code and focused tests against its acceptance criteria.
The diff adds explicit bounded results, derives Kettle’s bound as 63, and moves process cleanup into the shared TLC helper. I’m checking the failure paths now, especially cancellation and descendant cleanup.
The read-only sandbox blocked Cargo’s build lock and the tests’ temporary directories. The pure Darwin policy test passed. I’ve also found a possible flaw in output-file handling and am tracing whether the shared runner can follow a tool-created symlink.
I confirmed two defects: the runner reopens output paths after detecting unsafe output, allowing a tool-created symlink to redirect the write; and its new drain-thread joins can hang on the Darwin helper path if a descendant starts a separate session. I reproduced the latter branch in memory on Linux and cleaned up all test processes. Actual macOS execution remains unverified.
1. **Severity:** P1  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/backends/tlc/src/process.rs:209`  
   **R-IDs:** [R2]  
   **Problem:** The runner closes its initial output-file handles, then reopens tool-accessible paths with `fs::write(stdout, &output)?`. A tool can replace `check.stdout` with a symlink, causing the runner to truncate/write a file outside scratch. This happens even when `within_budget` detects the symlink: output publication precedes `result?`. Replacing the file with a FIFO can instead block indefinitely.  
   **Suggestion:** Retain securely opened file handles and publish through them. Reject substituted output paths before reopening them, including in Apalache’s error-output reader. Add symlink/FIFO controls.

2. **Severity:** P1  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `model/umpire/backends/tlc/src/process.rs:207`  
   **R-IDs:** [R2]  
   **Problem:** The new unconditional pipe-reader joins can defeat timeout/cancellation on Darwin. For non-JVM commands, the supervisor only kills the original process group; a descendant that starts another session survives and retains stdout/stderr. The supervisor exits, but `out.join()` waits indefinitely. Selecting the Darwin branch in memory on Linux reproduced both the surviving descendant and missing pipe EOF; all reproducer processes were cleaned up. The previous file-based output collection did not introduce this indefinite join.  
   **Suggestion:** Complete descendant ownership for Darwin helper commands and bound cleanup/drain waits. Add a separate-session descendant control for that path.

Verification was limited by the read-only sandbox: Cargo could not acquire its build lock, and filesystem-based Python controls could not create temporary directories. The pure Darwin policy test passed. Actual macOS execution remains unverified.

## Requirements coverage

Coverage below is scoped to task .2.

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Emitter work belongs to task .1. |
| R2 | partial | Explicit bounded results, checked native bound, and pin admission are present; output containment and Darwin timeout defects remain. |
| R3 | deferred | Typed decoding/replay belongs to task .3. |
| R4 | deferred | Fixture agreement/goldens belong to task .4. |
| R5 | deferred | Fixture-consumer integration belongs to task .4. |
| R6 | deferred | Symbolic Worlds belong to task .5. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>