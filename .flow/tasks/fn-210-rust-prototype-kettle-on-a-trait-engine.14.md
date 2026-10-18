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
Unknown scope actions now have the canonical `Refusal::ScopeAction { action }` error and stable protobuf code 7. The wasm adapter preserves the code and message through the generated response envelope, and the pinned generator updated the Go bindings without changing existing codes.

Tier: project implementer gpt-6.1-sol at medium (judge unavailable no_key)
stage: impl-review - ran [2026-10-18T21:39:49Z..2026-10-18T21:41:12Z] (SHIP; three first-round lenses; codex:gpt-6.1-sol:high; same-family review)

baseline: green. The pre-edit full gate, focused Cargo tests, release wasm build and timing command all passed at 746545cd87df51a46385c0da329070da66d0d863. Logs are `.flow/tmp/fn210-14-baseline-gate.log`, `fn210-14-baseline-focused.log`, `fn210-14-baseline-build.log` and `fn210-14-baseline-time.log`. No gate was skipped. Baseline warm timing was 0.779 s for the inner loop and 2.527 s for the gate.

The protocol regression `scope_action_refusal_survives_generated_protocol` uses the actual production mapping source and generated Rust encoding/decoding. It pins code 7 and `unknown scope action: User::Missing`. It first failed because the checker and generated protocol lacked ScopeAction. A subsequent deliberate wrong-code mutation failed with actual code 3 versus expected 7; restoring the correct adapter passed. Logs are `.flow/tmp/fn210-14-red.log`, `fn210-14-mutation-red.log` and `fn210-14-final-focused.log`. The adapter lives in a private source module so the integration test can exercise it without adding exports or dependencies. Runtime scope validation and its Model fixture remain task .3.

All three review lenses returned SHIP with no findings. The canonical receipt is `/tmp/impl-review-receipt-8f37faba39e2-fn-210-rust-prototype-kettle-on-a-trait-engine.14.json`; draw evidence is `.flow/review-fanout/e084026e5a224da392a9aa2e9d7c9e32/`. Reviewer Cargo attempts were inconclusive because the read-only sandbox denied build-lock writes. The worker-owned checks below establish green verification. No NEEDS_WORK-to-SHIP transition occurred, so memory capture did not run.

Final verification passed with `CARGO_TARGET_DIR=/home/agent/.cache/codex-build/cargo-target`:

- `make model2-check` ran fmt, Clippy, docs, native fixtures, capability/export controls, protobuf freshness, wasm builds and CGO-disabled Go parity. The native dispatch binary ran four tests and the Go oracle tests passed. Log `.flow/tmp/fn210-14-final-gate.log`; receipt `.flow/tmp/green-receipts/3c056ea5-unittest.json`.
- From model2, `mise exec -- cargo test -p umpire-wasm -p umpire-checker` passed the checker fixtures, brute-force oracle and all four dispatch tests. Log `.flow/tmp/fn210-14-final-focused.log`.
- From model2, `mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked` passed using the existing export-name linker wrapper. Log `.flow/tmp/fn210-14-final-build.log`.
- `make model2-time` passed with warm timing within both R30 budgets. Log `.flow/tmp/fn210-14-final-time.log`.

Existing Kettle goldens and the native/wasm/Go parity checks stayed byte-identical. This checkpoint adds no progress implementation, exports, dependencies, manifest or lockfile changes.

Final `make model2-time` output (exit 0):

```text
Time the model2 feedback loop (fn-210 R30)...
step (warm)                         time
gate command: UMPIRE_TIME_STEPS=1 make -C .. model2-check
make[1]: Entering directory '/Users/stephan/Workspace/skunkworks/umpire/temporal'
Check model2 (fmt, clippy, docs, tests)...
timing-controls                   0.009s
capability-controls               0.083s
export-controls                   0.073s
protobuf-freshness                0.064s
fmt                               0.072s
clippy                            0.076s
docs                              0.094s
tests                             0.724s
wasm32-wasip1                     0.049s
core-wasm-clippy                  0.070s
wasm-guest                        0.062s
go-host                           0.898s
make[1]: Leaving directory '/Users/stephan/Workspace/skunkworks/umpire/temporal'
inner source: umpire/checker/src/lib.rs
inner command: mise exec -- cargo test -p umpire-checker --quiet
inner loop (edit+test)            0.821s
gate total                        2.518s  (budget 30s)
inner loop                        0.821s  (budget 3s)
```

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: e472b47ae0b696aeb8239888aa7a64a91fe1980d, 3c056ea5e1cfb997daf110e578ce63d78c6b74ac
- Tests: make model2-check, cd model2 && mise exec -- cargo test -p umpire-wasm -p umpire-checker, cd model2 && mise exec -- cargo build -p umpire-wasm --target wasm32-wasip1 --release --locked, make model2-time
- PRs: