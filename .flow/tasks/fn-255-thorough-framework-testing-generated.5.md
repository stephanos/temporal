---
satisfies: [R4, R7]
---
# fn-255-thorough-framework-testing-generated.5 Miri on the WASM ABI buffer logic; record the syscall exclusion

Touches: [model/umpire/protocol/runtime/src/abi.rs, model/umpire/protocol/runtime/tests/**, model/umpire/backends/process/src/syscall.rs, model/umpire/backends/process/src/syscall/**]

## Description
Check the safe-Rust ABI ownership/envelope paths and pure syscall helpers under Miri (spec R4), using .3's pinned nightly. ABI unsafe export attributes are distinct from the actual unsafe subprocess/syscall execution boundary, which Miri excludes.

**Size:** S
**Files:** `abi.rs` (testable seams), runtime tests, syscall module (pure-helper tests and the exclusion note).
**Touches:** [model/umpire/protocol/runtime/src/abi.rs, model/umpire/protocol/runtime/tests/**, model/umpire/backends/process/src/syscall.rs, model/umpire/backends/process/src/syscall/**]

### Approach
- Make shared production `abi.rs::Guest` buffer ownership and call-envelope logic testable through a native-sized internal seam without changing the exported wasm32 ABI. The extern-C wrappers compile natively too; invoke Rust methods for expected invalid-range panics.
- Add unit-test modules named `miri_abi` under the runtime ABI and `miri_pure` beside process pure helpers. Run only `cargo +<pinned-nightly> miri test -p umpire-runtime --lib miri_abi::` and `cargo +<pinned-nightly> miri test -p umpire-backend-process --lib miri_pure::`. Keep process-spawn, file-descriptor and FFI tests outside these filters. Record the nonempty selected test list so a filter matching zero tests fails. ABI cases include zero length, maximum length, double release and mismatched pointer/length.
- `backends/process/src/syscall/calls.rs` uses FFI and volatile function pointers that Miri cannot execute. Record the FFI execution exclusion and its reason in the module docs; Miri-test pure helpers in `backends/process/src/syscall.rs` (including platform policy) without invoking process/syscall setup.
- Pin a finite Miri test manifest, case/size bounds and per-test/total timeouts in the R7 budget table. A bounded outer supervisor enforces timeouts.
- Planted control: a separately invoked test-only out-of-bounds read in a copy of the buffer logic is reported by Miri; the positive suite stays free of deliberate UB.

### Investigation targets
**Required:**
- `model/umpire/protocol/runtime/src/abi.rs`
- `model/umpire/backends/process/src/syscall.rs` and `syscall/calls.rs`

### Native ABI coverage

The current `Guest::allocate` converts a native pointer to `u32` and returns zero if it does not fit. A 64-bit native/Miri run that only follows that refusal proves no successful allocation/call. Exercise shared production buffer ownership/envelope logic through a native-sized internal seam, preserving the public wasm32 ABI. Assert a nonzero successful allocation, dispatch/publication and release, plus zero/maximum length, duplicate-release and wrong-range behavior. Invalid release is an expected panic/trap, not a promised typed decoder refusal. Do not suppress these cases or call the aborting extern-C wrapper in a positive Miri test.

## Acceptance
- [ ] Miri passes on the ABI buffer and envelope tests, edge cases included.
- [ ] Syscall exclusion recorded with its reason; the exact nonempty `miri_pure::` selection passes without spawning processes or executing FFI; the `miri_abi::` selection is also recorded and nonempty.
- [ ] The planted out-of-bounds control is reported in its separate bounded run; the finite positive suite meets work/time bounds.
- [ ] Miri executes successful production buffer/envelope paths, with allocation/call/publication assertions; native pointer truncation cannot turn the suite into refusal-only coverage.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
