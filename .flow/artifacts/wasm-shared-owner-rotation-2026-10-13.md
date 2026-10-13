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
  The literal identity assertion passes; standalone worker pins are historical.

## Verification and remaining work

Stock Go is 1.27.1 on linux/arm64. Tests use `-tags test_dep -count=1`.
Integrated World/Record/strictjson/hostfs/minimizer, protocol generation,
network/model projections, portable simulator, both-platform architecture and
`make -C tools/gomad3 validate` pass. Changed-package fast lint passes with
fixes disabled; log `/tmp/fn153-18-integrated-fast-lint-20261013.log`.
Generator log: `/tmp/wasm-rotation-validate-20261013.log`.

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
the final identity cutover. Fn166 structural/module renames and the integrated
WASM scheduling, retained replay, installed operator, workload, resource and
cost gates remain outstanding. No full native gate, PR, push or CI action is
claimed.
