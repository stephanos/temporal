---
satisfies: [R4, R7]
---
# fn-255-thorough-framework-testing-generated.5 Miri on the WASM ABI buffer logic; record the syscall exclusion

Touches: [model/umpire/protocol/runtime/src/abi.rs, model/umpire/protocol/runtime/src/abi/**, model/umpire/protocol/runtime/src/lib.rs, model/umpire/protocol/runtime/src/testing.rs, model/umpire/protocol/runtime/tests/**, model/umpire/backends/process/src/syscall.rs, model/umpire/backends/process/src/syscall/**, model/umpire/dev/miri-control/**, model/umpire/dev/xtask/src/main.rs, model/umpire/dev/xtask/src/lib.rs, model/umpire/dev/xtask/src/gate.rs, model/umpire/dev/xtask/src/gate/**, model/umpire/dev/xtask/tests/**, model/umpire/README.md, model/umpire/dev/xtask/README.md]

## Description
Check shared production safe-Rust ABI ownership/envelope paths and pure syscall helpers under Miri (spec R4), using .3's isolated pinned nightly. Preserve production unsafe-code prohibitions. Prove that the Miri command detects actual undefined behavior with a separately invoked isolated control crate; a safe bounds-check panic is insufficient.

**Size:** S
**Files:** `abi.rs` and ABI child test modules (native-sized production seams and positive tests), runtime source test support/integration tests, syscall pure-helper tests/exclusion documentation, isolated `dev/miri-control/` manifest/lockfile/control test, focused bounded command registration and documentation.
**Dependencies:** .3's committed nightly/components and canonical fn-254-fast-validation-gates-and-a-real-inner.12. Source preparation may precede committed budget closure; real execution and task closure require the single committed R7 authority. The existing .3 same-spec dependency remains; the conductor records the budget cross-spec gate with Flow blocked status until its committed closure/admitted bounds are verified.

### Approach
- Make shared production `abi.rs::Guest` buffer ownership and call-envelope logic testable through a native-sized internal seam, preserving the exported wasm32 ABI. Test the same allocation, publication, envelope and release implementation used in production. Preserve `forbid(unsafe_code)` and existing stronger unsafe restrictions in production/runtime crates; do not weaken them to accommodate a planted control.
- Add positive unit-test modules named `miri_abi` under the runtime ABI and `miri_pure` beside process pure helpers. Use .3's dated nightly with `miri` and `rust-src` components. Run only `cargo +<pinned-nightly> miri test -p umpire-runtime --lib miri_abi::` and `cargo +<pinned-nightly> miri test -p umpire-backend-process --lib miri_pure::`, under bounded framework supervision. Record the exact nonempty selected test lists; zero matches fail. Keep process-spawn, file-descriptor and FFI tests outside these filters.
- Positive ABI cases assert nonzero native-sized allocation, dispatch, response publication and release; include zero length, maximum length, live-buffer/size caps, double release, mismatched pointer/length and invalid ranges. Invoke Rust methods for expected invalid-release panics rather than unwinding across aborting `extern "C"` wrappers. A native pointer that cannot fit `u32` must not turn this suite into refusal-only coverage.
- `backends/process/src/syscall/calls.rs` uses FFI and volatile function pointers that Miri cannot execute. Record that execution exclusion and its reason in module docs; Miri-test pure helpers in `syscall.rs`, including platform policy, without invoking process/syscall setup. Keep the real native process-supervision controls separately maintained.
- Put the deliberately unsafe out-of-bounds read only in `model/umpire/dev/miri-control`, an isolated non-workspace crate with its own manifest and committed lockfile. Use `cfg(miri)` so the UB test is unavailable natively, narrow the unsafe permission to that test-only control, and mark the test ignored. Do not put deliberate UB or unsafe allowances in runtime ABI child modules, the production workspace or the positive native/Miri suite.
- Invoke that one ignored control explicitly and separately, for example `cargo +<pinned-nightly> miri test --manifest-path dev/miri-control/Cargo.toml --locked --test ub -- --ignored --exact planted_out_of_bounds_read`. Admit and record exactly that test. Require nonzero exit plus the expected Miri undefined-behavior/out-of-bounds memory-read diagnostic. A safe Rust bounds panic, missing tool, compile failure, timeout or zero tests cannot satisfy the control. The test-only unsafe read may use a bounded copy of buffer logic; it is a tool-detection control and supplies no production-path positive coverage.
- Consume finite positive/control test manifests, case/size bounds and per-test/total timeouts from committed fn-254.12 tables. A bounded outer supervisor enforces the whole stage and separates expected UB failure from infrastructure failure. Reports identify completed/remaining selected work when capped; a capped positive suite does not pass.
- Keep scratch/artifacts under framework-root `target/` or `build/`, or an admitted external directory, including the isolated control crate's target directory. Register bounded Miri execution/control commands explicitly in the framework entry point, with focused inventory controls. No Miri execution enters `model-quick`; .7 owns final tier wiring. Coordinate shared xtask files serially; reuse .3 pins without modifying stable Rust selection or frozen provider work.

### Investigation targets
**Required:**
- `protocol/runtime/src/abi.rs`, ABI child test modules, runtime source test support and integration tests
- `backends/process/src/syscall.rs` and `syscall/calls.rs`
- `dev/miri-control/` isolation, ignored-test selection and diagnostic classification
- .3 nightly/components, committed fn-254.12 bounds, current command inventory, supervisor and standalone containment

### Native ABI coverage

Current `Guest::allocate` converts a native pointer to `u32` and returns zero when it does not fit. A 64-bit native/Miri run that only follows that refusal proves no successful allocation/call. Exercise shared production ownership/envelope logic through the native-sized internal seam while preserving the public wasm32 ABI. Invalid release remains an expected panic/trap, not a promised typed decoder refusal.

## Acceptance
- [ ] The exact nonempty `miri_abi::` selection passes on shared production buffer/envelope logic, including positive allocation/dispatch/publication/release assertions and edge cases. Native pointer truncation cannot satisfy positive-path coverage.
- [ ] The syscall FFI exclusion and reason are recorded; the exact nonempty `miri_pure::` selection passes without processes or FFI execution.
- [ ] Production/runtime unsafe-code prohibitions remain intact. Deliberate UB exists only in the isolated non-workspace `dev/miri-control` crate, under `cfg(miri)`, with a narrow test-only unsafe permission and ignored test selected explicitly in a separate bounded invocation.
- [ ] The planted out-of-bounds read produces the required Miri UB memory diagnostic. Safe bounds panics, infrastructure failures, timeouts and empty selections do not count. The native positive suite cannot execute the UB control.
- [ ] Positive suite and separate control enforce committed fn-254.12 work/time limits, retain exact selection/version/source/diagnostic evidence and use admitted artifact roots. No Miri work enters `model-quick`; .7 tier wiring remains separate.

## Done summary
Blocked:
Execution and task closure wait for the committed fn-254-fast-validation-gates-and-a-real-inner.12 closure in umpire and admission of this task's work/time limits to its single authoritative budget table. Disjoint source preparation does not satisfy this gate. Flow task dependencies only accept same-spec IDs; after verifying that closure and admitted entries, the conductor may reset this block and admit the task normally.
## Evidence
- Commits:
- Tests:
- PRs:
