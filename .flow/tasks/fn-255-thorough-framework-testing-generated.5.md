---
satisfies: [R4]
---
# fn-255-thorough-framework-testing-generated.5 Miri on the WASM ABI buffer logic; record the syscall exclusion

Touches: [model/umpire/protocol/runtime/src/abi.rs, model/umpire/protocol/runtime/tests/**, model/umpire/backends/process/src/syscall/**]

## Description
Check the framework's `unsafe` code for undefined behaviour (spec R4), using .3's pinned nightly.

**Size:** S
**Files:** `abi.rs` (testable seams), runtime tests, syscall module (pure-helper tests and the exclusion note).
**Touches:** [model/umpire/protocol/runtime/src/abi.rs, model/umpire/protocol/runtime/tests/**, model/umpire/backends/process/src/syscall/**]

### Approach
- Make the buffer allocation, release and call-envelope logic behind `abi.rs:139-160` testable natively (the exports are wasm-only), without changing the exported ABI.
- Run `cargo +<pinned-nightly> miri test` on those tests, including edge cases: zero length, maximum length, double release, mismatched pointer and length.
- `backends/process/src/syscall/calls.rs` uses FFI and volatile function pointers that Miri cannot execute. Record the exclusion and its reason in the module docs; Miri-test its pure helpers.
- Planted control: a test-only out-of-bounds read in a copy of the buffer logic is reported by Miri.

### Investigation targets
**Required:**
- `model/umpire/protocol/runtime/src/abi.rs`
- `model/umpire/backends/process/src/syscall/calls.rs`

### Acceptance
- [ ] Miri passes on the ABI buffer and envelope tests, edge cases included.
- [ ] Syscall exclusion recorded with its reason; pure helpers Miri-tested.
- [ ] The planted out-of-bounds control is reported.

## Acceptance
- [ ] TBD

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
