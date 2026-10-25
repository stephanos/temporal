I’ll read the task spec first, then trace the changed workspace, driver, and script boundaries against their callers.
The workspace split and dedicated guest driver are in place. I’m tracing the unchanged backend discovery calls, guest scripts, and target-path consumers for assumptions tied to the old layout.
I found two integration problems: timed framework stages change back into the Temporal workspace, and the root guest builder’s native transcript directory differs from the Go host test’s new default. I’m checking the exact call paths and whether existing controls cover those cases.
1. **Severity:** P1  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `Makefile:1297`  
   **R-IDs:** [R7]  
   **Problem:** The new framework stages use `$(call RUST_MODEL_STEP,tests-framework)` after `cd model/umpire`. When timing is enabled, that wrapper invokes `model/scripts/time-gate.sh`, which unconditionally changes directory to `model/`. Consequently, `make model-validate` runs the Temporal workspace’s fmt, clippy, docs and tests again under the framework stage names, silently omitting framework coverage. A read-only execution of the timer’s prologue confirmed the directory reset.  
   **Suggestion:** Preserve the caller’s directory in `--step` mode, or use the framework-local timer for framework stages. Add a focused control asserting each timed stage’s actual workspace.

2. **Severity:** P2  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/hosts/go/host/current_contract_test.go:21`  
   **R-IDs:** [R7]  
   **Problem:** With `CARGO_TARGET_DIR` unset, the root guest builder writes native transcripts to `model/target/current-contract-native`. This changed fallback reads `model/umpire/target/current-contract-native`. The root Makefile supplies explicit Wasm paths and `UMPIRE_PANIC_NATIVE`, but omits `UMPIRE_CURRENT_CONTRACT_NATIVE`; the builder’s exported target does not survive into subsequent recipe processes. Thus ordinary `make model-check` fails the native/Wasm contract test on a clean checkout, or consumes stale framework-local transcripts.  
   **Suggestion:** Pass `UMPIRE_CURRENT_CONTRACT_NATIVE="$$target/current-contract-native"` from the root Go-host stage, retaining the correct standalone fallback. Cover the root invocation with `CARGO_TARGET_DIR` unset.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R7 | partial | Workspace/tool/driver separation is implemented, but the intermediate gate has the two integration defects above. Final gate composition and standalone gate proof remain assigned to .10/.11. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>