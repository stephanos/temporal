I’ll read the task specification first, then trace the changed runner and gate code against its acceptance criteria.
The runner leaves now run nextest followed by Cargo doctests, and the recorded inventories match. I found that the Temporal gate’s existing wiring control still expects the old Cargo command; I’m checking whether the runner change makes that control fail and whether the JVM filters cover all affected tests.
I confirmed two integration problems: the Temporal wiring control expects a command the runner no longer uses, and `make model-validate` reaches the framework through `gate`, where the new standalone stage is skipped. I couldn’t run the focused test because the sandbox refused a required directory creation, so I’m tracing both failures directly from the code.
1. **Severity:** P1  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/dev/xtask/src/test_run.rs:10`  
   **R-IDs:** [R4]  
   **Problem:** The runner change deterministically breaks `gate::tests::actual_temporal_gate_wiring_and_named_failures`. Its expected calls in `model/dev/xtask/src/gate.rs:188` still contain one `cargo test --workspace --locked --quiet` invocation. The changed leaf produces nextest plus doctests, so the existing control’s first successful-dispatch comparison fails. The normal workspace test stage therefore cannot pass.  
   **Suggestion:** Update that wiring control’s command inventory, failure cases and stage-name indexing for both runner phases, then execute the focused control.

2. **Severity:** P1  
   **Confidence:** 100  
   **Classification:** introduced  
   **File:Line:** `model/umpire/dev/xtask/src/gate.rs:243`  
   **R-IDs:** [R7]  
   **Problem:** `make model-validate` silently skips the new standalone stage. Its actual route is `model-time` → `model-check` → framework `gate`, with `UMPIRE_BACKENDS=require`. However, `self.required` is initialized only from `command == "validate"`, so this condition skips standalone even during the root validation command. A broken isolated framework can consequently escape the authoritative gate.  
   **Suggestion:** Make standalone selection recognize the validation mode used by the root caller, while preserving the child recursion exclusion. Cover that actual invocation path in the wiring controls.

FYI — **Long Method:** `standalone.rs:432` adds `fn finish(...)`, combining stage-receipt validation, artifact collection and result publication across roughly 90 lines. This heuristic does not affect the verdict.

Focused execution was blocked by the read-only sandbox. Both findings follow mechanically from the traced code.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R4 | partial | Recorded inventories match across 962 identities, including ignored status; doctests remain separate and JVM groups are bounded. Existing Temporal wiring control is broken. |
| R7 | partial | Isolated-copy machinery and negative controls exist, but root validation skips the registered stage. |

Unaddressed R-IDs: []

Classification counts: 2 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":2,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict>