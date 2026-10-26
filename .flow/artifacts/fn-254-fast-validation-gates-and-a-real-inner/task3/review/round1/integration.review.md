I’ll read the task specification first, then trace the changed runner and gate code through its callers and recorded evidence.
The runner changes preserve failure propagation and run doctests after successful binary tests. I’m checking the JVM filters and standalone stage against existing tests and generated artifact paths for integration failures.
I found a concrete integration break: the unchanged Temporal gate wiring test still expects one `cargo test --workspace` call. The new runner makes nextest and doctest calls, so that test’s first success case fails and blocks the Temporal gate. I tried the focused test, but the read-only sandbox prevented Cargo from creating its build directory.
- **Severity**: P1
- **Confidence**: 100
- **Classification**: introduced
- **File:Line**: `model/dev/xtask/src/test_run.rs:9`
- **R-IDs**: [R4]
- **Problem**: The new runner breaks `gate::tests::actual_temporal_gate_wiring_and_named_failures`. Its unchanged expectation at `model/dev/xtask/src/gate.rs:188` requires one `cargo test --workspace --locked --quiet` invocation. The runner now produces separate nextest and doctest invocations, so the success-case assertion at line 240 fails. This test belongs to the workspace being run, making the Temporal test stage fail on every normal gate execution.
- **Suggestion**: Update the Temporal wiring control to expect both commands and exercise each command’s failure path, preserving stage-name and exit-status assertions.

The focused test command could not execute because the read-only sandbox prevented Cargo from creating its build directory. The command/expectation mismatch is definitive from code.

## Requirements coverage

| R-ID | Status | Evidence |
|---|---|---|
| R4 | partial | Recorded Cargo/nextest inventories match across 962 identities, including four ignored tests; doctest execution and bounded JVM groups are covered. The Temporal wiring regression prevents successful gate execution. Remaining consolidation and final timing work belongs to later tasks. |

Unaddressed R-IDs: []

Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>NEEDS_WORK</verdict> - Fix the Temporal gate wiring regression.