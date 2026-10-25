I’ll read the task contract first, then trace the changed workspace boundaries, entrypoints, and build logic for correctness regressions.
The workspace split is wired into both sets of root Rust checks. I’m now tracing guest tool selection and cache identity through the moved driver, including relative paths and the Temporal bridge.
I’ve traced the backend cache roots and activity TLC callers to the framework directory. One potential regression remains in the guest driver: it changes Cargo’s working directory while forwarding path-valued environment variables unchanged. I’m checking that path before finalizing the findings.
The relative `CARGO_HOME` regression is confirmed: the same locked, offline metadata command passes from the caller workspace and fails from the framework directory. I also found a reanchored TLC test that writes to `build/evidence` without creating it, so a focused run on a clean framework tree depends on another test having run first.
1. **Severity:** P2  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `model/umpire/dev/guest-driver/src/guest.rs:290`  
   **R-IDs:** [R7]  
   **Problem:** Cargo now runs from the framework root while inheriting the caller’s relative `CARGO_HOME`. Although `context.cargo_home` resolves against the caller, that absolute value never reaches Cargo. This selects a different registry/configuration directory and breaks builds or source admission. Verified: the same locked, offline metadata command succeeds from `model/` and fails from `model/umpire` with an unchanged relative `CARGO_HOME`. Tool probes also inherit this mismatch.  
   **Suggestion:** Resolve Cargo home before switching directories and pass that absolute `CARGO_HOME` consistently to probes, metadata, builds, and bootstrap commands. Add a control with distinct caller/tool roots and relative Cargo home.

2. **Severity:** P2  
   **Confidence:** 75  
   **Classification:** introduced  
   **File:Line:** `model/umpire/backends/tla/tests/main/optional_ticks.rs:319`  
   **R-IDs:** [R7]  
   **Problem:** The relocated evidence write targets `build/evidence/fn2372-eager-safety-{site}.log` without creating `build/evidence`. TLC admission and execution create only `build/tlc`. With installed prerequisites and a clean evidence directory, running `optional_ticks::present_optional_keeps_eager_errors_in_guard_assignment_and_claim` alone with `UMPIRE_BACKENDS=require` therefore fails with `NotFound`. Success depends on another test creating the parent first.  
   **Suggestion:** Create the evidence directory before writing, and verify the focused test with that directory initially absent.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R1 | deferred | Historical evidence work belongs to other tasks. |
| R2 | deferred | Quick gate implementation is outside .13. |
| R3 | deferred | TLC batching is outside .13. |
| R4 | deferred | This task preserves migration inventories; parallel execution belongs elsewhere. |
| R5 | deferred | Final composition belongs to .10. |
| R6 | deferred | Final measurements belong to .8. |
| R7 | partial | Workspace separation, driver independence, and root migration are implemented; the two regressions above remain. Full standalone gate proof belongs to .11. |
| R8 | deferred | Budget enforcement belongs to .12. |

Unaddressed R-IDs: [R7]

Validation included metadata checks, changed shell-script syntax, artifact hashes, and inherited policy/profile comparisons. Runtime regression tests were not executed.

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":["R7"]}
```

<verdict>NEEDS_WORK</verdict>