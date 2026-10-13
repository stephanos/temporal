# WASM shared-owner rotation batch — 2026-10-13

This handover records source acceptance as the fn153/fn154/fn166 batch proceeds.
It does not qualify the changed candidate with historical fn151/fn156 evidence.
Previously transferred native execution remains with fn128/fn149. New explicit
fn154 native execution requirements remain open in their owning tasks.

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
  Full architecture found the new tokenizer/pure-source inventory omissions.
  `046ff9ecb8` fixes the exact import/source classification, removes a dynamic
  identity closure and reconciles the caller contract without a brittle count.
- `c467490c78` adds fn154.2's NC2 config, explicit shared limits, closed
  held/timeout/reset vocabulary, matching generated host/overlay validation,
  and selected WASM boot/spec equality. `42574e0753` binds the combined model
  to `sha256:c27f0ae4168586879fc623eab2e34f97a349d2055538d8e966ac327b3d06b020`.
  Integrated readers, protocol/projection, helper parity, portable simulator,
  full WASM simulation (140.134 s), full all-platform architecture/purity,
  generated/compatibility validation and scoped lint passed (exit 0).
  Combined source review on frozen `a23880029f` finalized NEEDS_WORK at
  `/tmp/wasm-json-network-config-source-review-20261013.json` (gpt-6.1-sol/high,
  same-family three-axis review). Four distinct introduced findings were recorded:
  effective broker limits bypass, timeout-dial source-port overflow,
  malformed-overdepth refusal precedence, and installation typed/unknown-field
  versus trailing-data precedence. The existing JSON and network workers own
  the corresponding fixes; no acceptance or qualification is claimed.
  `f44ccd6726` integrates the two network fixes (worker `ed6a7966c1`, frozen
  review `439b0e2eb2`). Byte/stall override and source-port-overflow negatives
  were red before and green after; matching broker and zero/65535 source-port
  controls remain valid without overwriting captured inputs. Root's focused
  broker selector and full protocol/network/projection generator checks pass.
  `494ae20d30..d3eb7edeab` integrates the JSON red/green fixes: tokenizer
  navigation checks child depth before consuming malformed values; installation
  validates/decodes the first value before classifying a suffix. Full strictjson,
  qualification and installation suites pass; log
  `/tmp/wasm-json-precedence-integrated-20261013.log`. The frozen source-review
  candidate `6ff6a117b2` pins `sha256:0b9d0091a80b036be2eec7a2d271bd4ca1ce9bc134965621c5e582a9d25ff444`;
  its depth/reader/identity controls and all-platform architecture/purity pass.
  Final single-session review returned SHIP on `6ff6a117b2`, with all four
  findings fixed and no new blockers. Its receipt retains the scoped source
  requirement coverage; full native and final WASM qualification stay open.
  All-platform architecture/purity, generated validation and affected lint
  exit 0; logs `/tmp/wasm-json-network-review-fixes-architecture-20261013.log`,
  `/tmp/wasm-json-network-review-fixes-validate-20261013.log`, and
  `/tmp/fn153-20-reviewfix-lint.log`. Main's `955fdd2f20` pin is
  historical after the subsequent expiry source checkpoint, not final identity.
- `49e4f12d5b` integrates fn154.4's shared byte-bounded local connection owner.
  Exact-source projected public-operation equivalence controls and full
  protocol/network/projection suites pass (exit 0), including 65 tiny writes,
  byte-capacity/short-read accounting, half-close, descriptor primitives and
  persistent timeout identity. This checkpoint changes model source after
  the `c27f0ae4...` pin; a fresh combined pin and validation remain required.
  Native execution acceptance remains open. Holding/reservation development
  proceeds in the same connection worker's isolated worktree.
  Root architecture checks pass in 63.571 s at the connection checkpoint;
  purity and protocol/network/projection checks pass after the network fixes.
  Logs: `/tmp/wasm-shared-connection-architecture-20261013.log` and
  `/tmp/wasm-shared-connection-review-fix-source-20261013.log`. The first
  command also named the absent `internal/purity` package and exited 1 for
  setup; the corrected `internal/puritycheck` run exits 0. The frozen review
  projection command similarly used an absent generator path; its corrected
  `simulation/network/generate` run exits 0. Neither setup error is a semantic
  regression or a full qualification result.
- `df06ec9c73` binds tasks fn154.5/.6/.7 to current Runner registration and
  portable gate-selection owners, replacing absent historical paths without
  changing their actual-run acceptance requirements.
- `13a2d649a4` integrates partition-held FIFO bytes and atomic topology/history
  reservations. Full protocol/network/projection suites pass; log
  `/tmp/wasm-partition-holding-integrated-20261013.log`.
- `074b6c0e2f` integrates autonomous connection expiry through native timers
  and existing projected NextDeadline/Advance ownership. Exact-source native
  timer harness and projected no-I/O/full-reserved-history, threshold, epoch,
  multi-pair replay, overflow and close controls pass in full protocol/network/
  projection suites; log `/tmp/wasm-connection-expiry-integrated-20261013.log`.
  Patched native public/process execution is not supplied by those harnesses.
  The complete unchanged shared clock suite passes (exit 0); log
  `/tmp/wasm-connection-expiry-clock-preservation-20261013.log`.
  Pending-dial, registration and history-removal work continues in the same
  worker. The combined source pin and final batch gates remain pending.
  On unchanged main `02ac7ec5a7`, the full WASM simulation suite exits 0 in
  220.855 s; log `/tmp/wasm-connection-expiry-simulation-20261013.log`.
  Full all-platform architecture/purity exits 0 (58.631 s/15.693 s), log
  `/tmp/wasm-connection-expiry-all-platform-20261013.log`; generated validation
  exits 0, log `/tmp/wasm-connection-expiry-generated-validate-20261013.log`.
  The existing helper is regression evidence only, not rebuilt/current final
  qualification. Main was held unchanged until that run terminated.
- `105da909f4` integrates pending-dial independent strict intervals, caller
  precedence, reservation cleanup, epoch/overflow controls, process Runner
  registrations and structural timer-projection drift guards. Root's focused
  protocol/model/projection/selection checks exit 0; log
  `/tmp/wasm-pending-dial-integrated-20261013.log`. Exact native execution,
  complete batch review and final qualification remain open. Further timer
  guard hardening is queued with the worker; no clock interface was changed.
- `ddee2cd0a1` fixes remaining conformance paths and admits the disjoint
  contract documentation surface, now ready at worker `a790cacd9b` but not yet
  integrated pending source/history alignment. That metadata commit also
  inadvertently included concurrently created fn172-fn178 Flow specs;
  their bytes were preserved unchanged and they are not executed in this batch.
- `02ac7ec5a7` admits the two disjoint framed/composed replay test files.
  `f6a649f9db` adds a source-reviewed standalone replay fixture because the
  full gomad3sim test variant is intentionally refused for syscall imports.
  Preparation/identity/capability refusal stays intact; no fabricated Prepared
  metadata is allowed. New source development is not actual native execution.
  Task8 now owns the affected network-only NE2 error layout as well as NC3/NR2;
  embedded Expected/Actual transition field removal makes NE1 retention invalid.
  Generic RPC/modelwire, volume and unrelated error formats remain unchanged.
- `852d7cd652` reconciles the next structural rotation's exact six groups:
  target stays a package beneath backends to avoid Provider/artifact cycles;
  runtime sources and strict JSON live beneath model; cluster codecs acquire
  lower record/identity ownership before extraction. Flow validation passes
  with retained pre-existing orphan-commit warnings. No package move is claimed.

## Verification and remaining work

Stock Go is 1.27.1 on linux/arm64. Tests use `-tags test_dep -count=1`.
Earlier integrated World/Record/strictjson/hostfs/minimizer, protocol generation,
network/model projections, portable simulator, both-platform architecture and
`make -C tools/gomad3 validate` pass. Changed-package fast lint passes with
fixes disabled; log `/tmp/fn153-18-integrated-fast-lint-20261013.log`.
Generator log: `/tmp/wasm-rotation-validate-20261013.log`.
Earlier reader/simulation/lint source results are recorded separately from the
then-red architecture candidate: `/tmp/fn153-20-root-simulation-20261013.log`,
`/tmp/fn153-20-root-integrated-lint-20261013.log`, and
`/tmp/fn153-20-root-architecture-red-20261013.log`.
The combined correction/config logs are
`/tmp/wasm-json-network-config-{architecture,validate,lint,simulator,simulation}-20261013.log`.
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
the final identity cutover. Fn154.2's generated config and vocabulary are
integrated, with the review fixes above pending. Its provisional admission
waives no completion or native gate. Fn154.4-.8 are admitted as one
connection/holding/expiry/dial/history source batch; final review, integrated
validation and each explicit execution requirement remain open.
Fn166 structural/module renames and the integrated
WASM scheduling, retained replay, installed operator, workload, resource and
cost gates remain outstanding. No full native gate, PR, push or CI action is
claimed.
