---
satisfies: [R27, R31]
---
# fn-210-rust-prototype-kettle-on-a-trait-engine.14 Pre-wire scope refusals through the generated host protocol

## Description
The .9 host adapter exhaustively matches the checker's Budget/State refusals. R3/R27 require an unknown scope-action refusal next; adding that variant during .3 would break the adapter and require shared protobuf generation, defeating the declared disjoint .3/.4 lanes. Pre-wire only that existing requirement here, before either lane begins.

Add `Refusal::ScopeAction { action: String }`, spelling `unknown scope action: {action}`. Add stable schema enum value `REFUSAL_CODE_SCOPE_ACTION = 7` without renumbering existing values; regenerate Go with the existing pinned generator and Rust through prost-build. Extend the wasm checker-refusal adapter to preserve the new code and message. Add a protocol-boundary regression for actual mapping/generated decoding; runtime scope validation and its Model fixture remain .3, not this checkpoint.

No progress algorithm, macro changes, new exports, new message structs, dependencies, lockfiles, or golden rewrites. Preserve existing byte-for-byte Kettle parity. This is a serial shared-protocol checkpoint, after .9 and before parallel .3/.4.

**Touches:** `model2/umpire/checker/src/refusal.rs`, `model2/umpire/proto/umpire/oracle/v1/oracle.proto`, `model2/umpire/wasm/src/**`, `model2/umpire/wasm/tests/**`, `model2/hosts/go/umpire/oracle/v1/oracle.pb.go`

**Quick commands:** from model2, `mise exec -- cargo test -p umpire-wasm -p umpire-checker` and `mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked`; from repository root, `make model2-check` and `make model2-time`.

**R30:** warm inner loop <=3s and full gate <=30s; paste actual timing output into done summary. One lead, current umpire checkout, no parallel source writers needed for this tiny coordination change. Fresh implementation review and done/show/closure are required.

## Acceptance
ScopeAction has its canonical named error and a stable generated protobuf code; actual mapping/decode regression passes; existing Kettle native/wasm/Go parity and full gate/timing remain green. No progress implementation.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
