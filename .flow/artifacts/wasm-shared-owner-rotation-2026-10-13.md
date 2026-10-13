# WASM shared-owner rotation batch — 2026-10-13

This handover records source acceptance as the fn153/fn154/fn166 batch proceeds.
It does not qualify the changed candidate with historical fn151/fn156 evidence.
Native runtime execution remains transferred to fn128/fn149.

## Source accepted

| Owner | Integrated source | Evidence |
| --- | --- | --- |
| fn154.1, R6/R11 configuration surface | `ea47fba672`, review `be8d538d27` | Explicit 4 MiB direction and 10-second stall defaults; strict configuration/Record/replay controls; portable package, vet and lint exit 0; three-axis source review SHIP; Flow status verified done. |
| fn109.14, R14 generated codecs | `0b0971431c` | Shared network, volume and process definitions; cross-consumer vectors, malformed-count and drift controls; integrated generator, simulator, projection and architecture checks exit 0; both supported-target compile-only checks exit 0; three-axis source review SHIP; Flow status verified done. |

fn109.14's exact generated ownership inventory is retained in
`/tmp/fn109-14-codec-summary.md` and its commands/results in
`/tmp/fn109-14-codec-evidence.json`. Network ownership is
`simulation/schema/networkwire.{json,go.tmpl}` plus
`backends/native/generation/protocol/network.go`, emitting the host
`tools/gomad3sim/runtime_network_wire.go` and native overlay
`internal/gomadio/simulation_wire.go`. Fn154.2 must edit this definition and
regenerate both consumers. Volume and process use their analogous single
definitions; no independent native process JSON decoder exists.

## Integrated cleanup corrections

- `f598b1f048` preserves standard-library fixed-byte-array JSON input in
  minimizer checkpoints. Strictjson/minimizer tests and the previously failing
  detached replay/repeat/resume selector pass.
- `fc20a2af0d` restores the historical full uint32 World string-capacity
  declaration domain. `6d7ab35d1d` checks remaining expanded-envelope bytes
  before string/payload allocation and collection iteration, and removes
  preallocation from untrusted delivery/blocked counts. Long-string roundtrips,
  malformed allocation claims, all World packages and Record callers pass.
- `a19bc025f5` checks public staged-file cleanup. `2ab8c3378f` extracts the
  existing WASM lint classifier without changing routing; `bd35d28aee` binds
  the panic-policy source test to the current statistics guard and narrows its
  exception. Full lintcode tests with actual root/nested policy fixtures pass.
- `e1d0791e89` binds the combined World/network compiled model to
  `sha256:246f21740b12a0fa680648c3a40778049e136e951d68ef0013731f16f32dc1bf`.
  That source passed its literal identity assertion; standalone worker pins are historical.
- `c8dbb97c11` preserves supplied stall timeout identity through local/process/net
  and selected descriptor adapters, with closed wire code 22 and pre-narrowing
  unknown-code rejection. `eda3f1b007` binds that integrated model to
  `sha256:04d18f1c559ba4dd7896e343332e0181d93657dc979e2f5bd43f485ec6fe2441`.
  Full protocol/network/projection, identity, all-platform architecture/purity,
  generated/compatibility validation and scoped lint passed (exit 0).
  Three-axis source review SHIP is finalized at receipt
  `/tmp/impl-review-receipt-657da2bc4466-fn-154-gomad-simpler-virtual-network-with.3.json`.
  Fn154.3 remains open for its explicitly required patched-runtime adapter
  execution; portable actual-source coverage does not satisfy that gate.
- `2cdde5cf18` integrates fn153.20's four WASM reader migrations, removes their
  handwritten syntax scanners and binds the decoder source plus pinned library
  version/content sum into the WASI identity
  `sha256:bb14f0ed4249d17e7474cda0864ce6f8ae521b169e7aa80c9d3844b7d1ba149c`.
  Root-zero empty-container boundaries have red/green regressions. Root reader,
  identity and actual-helper parity checks passed; the full simulation package
  passed in 114.297 s, and affected lint reported zero issues. The helper was
  the existing `c2e4ff2f...` binary, not rebuilt or requalified by this task.
  Full architecture found the new tokenizer/pure-source inventory omissions;
  those remain under repair before source review or completion.

## Verification and remaining work

Stock Go is 1.27.1 on linux/arm64. Tests use `-tags test_dep -count=1`.
Earlier integrated World/Record/strictjson/hostfs/minimizer, protocol generation,
network/model projections, portable simulator, both-platform architecture and
`make -C tools/gomad3 validate` pass. Changed-package fast lint passes with
fixes disabled; log `/tmp/fn153-18-integrated-fast-lint-20261013.log`.
Generator log: `/tmp/wasm-rotation-validate-20261013.log`.
Current reader/simulation/lint source results are recorded separately from the
still-red architecture candidate: `/tmp/fn153-20-root-simulation-20261013.log`,
`/tmp/fn153-20-root-integrated-lint-20261013.log`, and
`/tmp/fn153-20-root-architecture-red-20261013.log`.
Timeout candidate validation log: `/tmp/fn154-3-frozen-validate-20261013.log`;
lint log: `/tmp/fn154-3-integrated-lint-20261013.log`. The reviewed source was
frozen at `7e60601caf`; the first review dispatch was not accepted after a
tracker-only HEAD movement, and a fresh frozen dispatch supplied the receipt.

The standard `make test-wasm` run completed with exit 0, WASI package 385.042 s,
on the earlier `0f314174` World model before the allocation and codec changes.
Log: `/tmp/fn153-18-integrated-test-wasm-20261013.log`. Its opt-in prototype and
frozen workload gates were skipped by their existing entry conditions. This
run is regression evidence for that source, not current batch qualification.

Fn153.18 remains open. The audit found the four explicitly named WASM JSON
readers still using handwritten scanners; fn153.20 now owns their required
library migration and precedes the final cleanup audit. Fn154.3 supplies the
timeout error transport; later tasks still own connection mechanics, stalls,
expiry reservations, payload-free history, process/framed-stream behavior and
the final identity cutover. Fn154.2 is implementing the generated config and
vocabulary against the accepted source interface under the batch-first
strategy; its provisional admission waives no completion or native gate.
Fn166 structural/module renames and the integrated
WASM scheduling, retained replay, installed operator, workload, resource and
cost gates remain outstanding. No full native gate, PR, push or CI action is
claimed.
