I’ll read the task spec first, then trace how the process backend changes interact with existing callers and platform code.
The capture and limit code is preserved; the main integration change is launching the supervisor from the caller’s process. I’m checking inherited process state, executable lookup, and the Rust controls that replace the deleted Python tests.
Both focused Cargo commands are blocked by the read-only sandbox when opening the build lock, so I can’t rerun them here. I’m continuing with source checks and the committed test evidence, including the raw syscall assumptions across supported Linux architectures.
I found two launch regressions: PATH lookup stops at a candidate that passes the permission check even if it cannot execute, and the raw `clone` call uses the wrong argument order on Linux/s390x. A memory-only probe confirmed that the old Python launcher continues to a working later PATH entry when the first candidate has a missing interpreter.
1. **Severity**: P2  
   **Confidence**: 75  
   **Classification**: introduced  
   **File:Line**: `model/umpire/backends/process/src/syscall.rs:339`  
   **R-IDs**: [R1]  
   **Problem**: PATH resolution returns the first executable regular file before attempting execution. If that file has a missing shebang interpreter, `faccessat` succeeds but `execve` fails. The former Python launcher continues to later PATH entries. A memory-only probe confirmed baseline success with a broken first candidate and working second candidate; preselecting the first produces ENOENT. Existing callers resolving `mise` or `shasum` by name inherit this regression.  
   **Suggestion**: Prepare all PATH candidates in the parent and attempt them in the child using allocation-free `execve` calls, preserving the previous search/error behavior. Add a broken-interpreter fallback control.

2. **Severity**: P2  
   **Confidence**: 100  
   **Classification**: introduced  
   **File:Line**: `model/umpire/backends/process/src/syscall.rs:714`  
   **R-IDs**: [R1]  
   **Problem**: The raw `clone` invocation assumes flags precede the child stack on every Linux architecture. Linux/s390x reverses these arguments. This supplies `SIGCHLD` as the stack address and zero as flags, breaking supervisor creation and ordinary child reaping. The installed Go implementation explicitly handles this reversal in `src/syscall/exec_linux.go:338`. The comment claiming argument order is immaterial overlooks the nonzero flags argument.  
   **Suggestion**: Select the correct argument order with `cfg(target_arch = "s390x")` and verify the platform-specific fork path.

Focused tests and Clippy could not run: both failed opening Cargo’s build lock on the read-only filesystem. Committed passing logs were inspected; native Darwin verification remains unavailable.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | partial | Rust ownership and safe capture retained; launch/platform regressions above remain. |
| R2 | deferred | Helper migration belongs to other tasks. |
| R3 | partial | All 14 prior process controls have Rust counterparts; remaining Python migration belongs to other tasks. |
| R4 | deferred | Final documentation and validation belong to other tasks. |
| R5 | deferred | Validation orchestration is outside task .4. |

Unaddressed R-IDs: [R1]

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R1"]}
```

<verdict>NEEDS_WORK</verdict>