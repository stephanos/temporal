I’ll read the task specification first, then trace the changed code and evidence for contract and consistency issues.
The change establishes separate workspaces and a dedicated guest driver. I’m checking root and path handling, plus whether the docs accurately distinguish this checkpoint from the gate work deferred to later tasks.
I found a caller-context mismatch: Cargo runs from the framework directory, while a relative `CARGO_HOME` is normalized only for compiler identity tracking. Cargo still receives the original relative value. I’m checking the remaining documentation and evidence before finalizing the findings.
1. **Severity:** P2  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `model/umpire/dev/guest-driver/src/guest.rs:236`  
   **R-IDs:** [R7]  
   **Problem:** Relative `CARGO_HOME` is resolved against the caller workspace for identity tracking, but Cargo inherits the original relative value while metadata/build run from the framework root. For a Temporal caller with `CARGO_HOME=.cargo-home`, Cargo and the compiler remaps therefore use different registry roots, causing cache lookup or source-admission failures.  
   **Suggestion:** Pass the resolved absolute `CARGO_HOME` to every tool/Cargo command, including metadata and bootstrap. Add a control with distinct caller/framework roots and a relative Cargo home.

2. **Severity:** P2  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `model/umpire/backends/tla/tests/main/optional_ticks.rs:319`  
   **R-IDs:** [R7]  
   **Problem:** The relocated evidence write targets `build/evidence`, but this test never creates that directory. TLC creates `build/tlc` only. Running this focused test with required backends on a fresh framework checkout therefore fails with a missing-directory error after successful checks. Other tests can mask the failure by creating the directory first. The previous `.flow/tmp` parent existed in the tracked base tree.  
   **Suggestion:** Create the evidence directory before writing, so the test runs independently.

3. **Severity:** P3  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/dev/guest-driver/README.md:8`  
   **R-IDs:** [R7]  
   **Problem:** The documented `cargo run -p umpire-guest-driver -- guest-lock ...` command fails because the package has two binaries and no `default-run`. Executing it reproduced Cargo’s “could not determine which binary to run” error.  
   **Suggestion:** Specify `--bin umpire-guest-driver` or declare that binary as `default-run`.

The 136 indexed proof-file hashes match. Runtime suites were not rerun in the read-only sandbox.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R7 | partial | Separate workspaces, driver executable, local tools and preservation evidence are present. Caller-context and isolated-test defects remain above; final gate composition and standalone gate controls belong to .10/.11. |

Unaddressed R-IDs: []

Classification counts: 3 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":3,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>