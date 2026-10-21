---
satisfies: [R2]
---
# fn-188-gomad-wasm-determinism-boundary-audit.3 Fail-closed request, reply and engine-setting boundaries on both sides

## Description
Audit the Go-to-helper execute request and the helper-to-Go callback path as boundaries. Helper: strict decoding of the execute request refuses unknown fields and kinds before `started` with the existing `unsupported` class and a message naming the field or kind. Go: strict decoding of helper replies and callback requests (namespace_call, storage transport, network, the fn-176 split-mode remote path) refuses unknown reply kinds or fields and classifies the terminal as unsupported; admission refusals precede guest execution; runtime callback/reply validation precedes affected-operation dispatch or reply consumption and reports the actual execution state. Engine settings: the Go side pins the expected engine configuration revision and refuses a helper whose `started` frame reports a setting outside the pin. Add tests in Rust (`tests/engine.rs` or the fn-187.1 transcript corpus once it exists) and Go (`wasi`) for each refusal, including the split-mode remote reply. Keep message shapes for the existing refusals (`unknown import m::n`, `operation outside stock WASI profile: <op>`) unchanged. Each new refusal is an inventory row with disposition `rejected`.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasmhost/src/{protocol.rs,engine.rs,namespace_call.rs,namespace_transport.rs}, tools/gomad/internal/backends/wasm/wasmhost/tests/engine.rs, tools/gomad/internal/backends/wasm/wasi/{process.go,helper_model.go,cluster.go,*_test.go}, tools/gomad/internal/backends/wasm/determinism/inventory.json

## Acceptance
- [ ] Unknown execute-request field or kind refused before `started` with `unsupported`, tested in Rust and Go
- [ ] Unknown callback reply kind or field refused on the Go side, including the split-mode remote path, with a classified terminal
- [ ] Off-pin engine setting in `started` refused by Go
- [ ] Existing refusal messages unchanged; every new refusal is an inventory row

## Done summary
Implemented strict execute-request, helper-frame and callback/remote boundaries, including split-mode replies, and pinned standalone/cluster started-frame engine configurations before authorization. Unknown fields/kinds are classified unsupported with their names; existing unknown-import and stock-WASI messages remain literal corpus expectations. Rejected inventory rows name the enforcing tests. Current integrated Rust engine refusal/corpus tests and Go transport/engine-pin tests pass. This is boundary source acceptance, not a stress or full qualification claim.

Final scoped repair 696a6ecc3e classifies valid tagged unknown kinds as unsupported naming the kind, with malformed/stale/terminal guards preserved; real ten-case regression and both helper-backed execute refusal subtests pass.

Evidence handover: `.flow/artifacts/fn188-source-acceptance-2026-10-20.md`. Integrated source `5dcd37ceaea62e95ba7e33703e247094fea6d158`: full default validate exit 0 and affected lint exit 0, zero issues.

stage: impl-review - SHIP (official SINGLE re-review, 2026-10-20T18:15:58Z; same session 01a15a74-37ad-7020-b5d4-830052a6a1a3; codex:gpt-6.1-sol:high, same-family; .flow/tmp/impl-review-fn188-source-batch.json; both findings fixed, zero open).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 5dcd37ceaea62e95ba7e33703e247094fea6d158
- Tests: Rust determinism_refuses engine tests: 3/3 pass in integrated default validation, Rust helper corpus: 26/26 pass, including preserved literal refusals, Go wasi TestDeterminism* and TestTransportInputValidation: exit 0, Integrated make -C tools/gomad validate d2410c5e: exit 0, make -C tools/gomad validate on integrated5dcd37ce: exit0 (inventory12/12, corpus26/26, engine3/3, Go determinism/architecture/WASI); /home/agent/wt/evidence/integration-2026-10-20/validate-fn188-controller-tagged-fixes.log, make lint-code-gomad with Linux tools, FIX=false, basee0cd113841: exit0, zeroissues; /home/agent/wt/evidence/integration-2026-10-20/lint-fn188-controller-tagged-fixes.log
- PRs:
