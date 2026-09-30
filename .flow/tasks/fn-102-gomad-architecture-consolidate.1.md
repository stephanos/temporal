---
satisfies: [R1]
---
# fn-102-gomad-architecture-consolidate.1 Generate the runtime bootstrap consumer from the existing I/O schema

## Description
R1. Current runtime gomadIOConfigFrame/gomadReadConfig/gomadConfigSeed repeat the 212-byte frame and seed offset 172. Follow the generated choice runtime endpoint registered in protocol.go:351. Generate constants/header recognition/seed decoding from the same wire-definition owner used by the host; extend schema metadata only if needed without changing protocol bytes. Keep descriptor reads, zero-frame fallback, short-frame rejection and startup termination in runtime. Do not move full checksum/identity validation earlier. Characterize the actual runtime consumer, not only a second generated host helper. Any added schema field must have one source of truth. Register every new runtime overlay file in toolchain/version/version.json overlay_allowlist (descriptor validation compares the entire overlay tree), regenerate the descriptor outputs, and audit profile/build identity inputs when registering the generated file. Preserve existing comments. No new dependencies. Run focused commands from tools/gomad3 with GOWORK=off and -tags test_dep; use the patched toolchain for runtime-consumer tests.

**Size:** M

**Touches:** [tools/gomad3/internal/gomadtool/generation/protocol/**, tools/gomad3/deterministicio/**, tools/gomad3/toolchain/runtime/overlay/**, tools/gomad3/toolchain/version/**, tools/gomad3/version_generated.mk, tools/gomad3/choice/**, tools/gomad3/target/**, tools/gomad3/simulation/**, tools/gomad3/internal/gomadtool/conformance/**] — WIDER: generation may refresh dependent endpoint/identity outputs; no runner edits.

**Files:** `tools/gomad3/internal/gomadtool/generation/protocol/{protocol.go,protocol_test.go}`; `tools/gomad3/deterministicio/schema/` (new runtime template); `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go` and generated bootstrap file; conformance bootstrap fixtures; `tools/gomad3/toolchain/version/{version.json,generated.go}` and generated version outputs.

### Quick commands

`go test -tags test_dep ./internal/gomadtool/generation/protocol`
`make validate-toolchain`
`make test-runtime` (includes activation and disabled-mode conformance)

## Acceptance
- The runtime bootstrap consumer passes valid, empty, truncated, malformed-header, and seed-boundary cases on both qualified platforms; later checksum/identity rejection and disabled/direct-seed behavior unchanged.
- Generated-source checks, focused tests, the full Gomad gates on darwin/arm64 and linux/amd64 (CI), and project lint pass (moved here from fn-102.6 on 2026-09-29).
## Done summary
The runtime's early bootstrap consumer is generated from `deterministicio/schema/iowire.json`: `protocol-generate` emits `toolchain/runtime/overlay/src/runtime/gomad_iowire_generated.go` (frame size, version, kind, magic, header recognition, seed projection) next to the host and overlay codecs, and the file is registered in the version descriptor's overlay allowlist. The seed offset derives from the schema's checksum offset (the seed is the last field before it), so no schema field was added and protocol bytes are unchanged. Descriptor reads, the zero-frame fallback, short-frame rejection, and startup termination stay in the runtime; full checksum and identity validation stays in internal/gomadio before any workload.

The runtime tier feeds real frames to the consumer on descriptor 5: truncated, header-only, wrong magic, wrong magic terminator, wrong version (both bytes), and wrong kind exit 2 with the early diagnostic before user initialization; valid-header frames at seed 0 and 2^64-1 pass the early phase and stop in a later Gomad phase. A generated wire test pins the encoder's seed to the runtime offset for 0, 1, a byte-order vector, and 2^64-1. Disabled and direct-seed activation cases are unchanged and pass.

Scope: F8 was cut to this task on 2026-09-29; R2-R6 moved to fn-105 D1-D5, and this task carries the full-gate qualification R6 used to own. Gates: make validate, test-toolchain, test-runtime on darwin/arm64; fork run 36669836359 at df0fa048f: darwin core (upgrade dossier, all gates) and linux conformance, host, pack, and core tiers passed. Project lint: server packages 0 issues; the nested gomad3 module is outside the root lint config (857 pre-existing findings), and the files this work added are clean under it.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: c506f3f39
- Tests: make validate, make test-toolchain, make test-runtime (darwin/arm64), go test ./deterministicio/internal/wire ./internal/gomadtool/generation/protocol, fork run 36669836359: core success, core-linux steps 5-10 success, make lint-code-fast (server packages: 0 issues)
- PRs: