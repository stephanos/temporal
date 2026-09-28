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
- [ ] R1 valid/empty/truncated/wrong header/seed boundary vectors cover actual runtime consumer.
- [ ] Later checksum/identity rejection and disabled/direct-seed activation are unchanged.
- [ ] New overlay files are registered in version.json overlay_allowlist and descriptor outputs regenerated.
- [ ] Generation/check is deterministic; no runtime allocations or forbidden imports are introduced.
- [ ] Wire bytes unchanged for fixed inputs; implementation identities regenerate normally, with all new inputs bound.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
