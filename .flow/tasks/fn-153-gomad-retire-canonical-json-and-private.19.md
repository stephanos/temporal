---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.19 One compressed World snapshot profile with a binary envelope

## Description
Owner decision (2026-10-12). Retire the plain World snapshot profile and the JSON/base64 compressed envelope. Encode World evidence as a binary envelope (magic, lengths, SHA-256, raw compress/gzip stream) around a bounded structured encoding of the expanded snapshot, using the repository wire generator or github.com/fxamacker/cbor/v2 (decode limits, core deterministic encoding). Delete world/compressed_scan.go, the JSON envelope and base64 layer in world/compressed.go and compressed_decode.go, codec.go's bounded JSON preflight and the uint64-string MarshalJSON helpers in world/types.go where no longer needed. Decode must stream into the existing summary (counts and digests) without buffering the expanded snapshot, to stay inside the 2 GiB operator memory budget. Producers: backends/wasm/wasi/broker.go EncodeWorldEvidence and native World recording; consumers: backends/wasm/simulation/retained.go and replay/inspection.

Protobuf is excluded by owner decision (2026-10-12).

**Size:** L
**Touches:** [tools/gomad3/world/**, tools/gomad3/backends/native/generation/protocol/protocol.go, tools/gomad3/backends/native/generation/protocol/protocol_test.go, tools/gomad3/backends/native/replay*.go, tools/gomad3/backends/wasm/wasi/broker*.go, tools/gomad3/backends/wasm/wasi/identity_test.go, tools/gomad3/backends/wasm/wasi/integration_test.go, tools/gomad3/backends/wasm/simulation/provider_test.go, tools/gomad3/backends/wasm/simulation/retained*.go, tools/gomad3/backends/wasm/simulation/world_codec_test.go, tools/gomad3/backends/wasm/simulation/world_compressed_probe_test.go, tools/gomad3/backends/wasm/simulation/world_profile_test.go, tools/gomad3/internal/archcheck/**, tools/gomad3sim/wasm.go, tools/gomad3sim/wasm_test.go]

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./world ./backends/native ./backends/wasm/wasi ./backends/wasm/simulation ./internal/archcheck

Capture exact current round trips and refusal classes before replacing the envelope; add malformed/truncated/duplicate/oversized/tampered controls and a streaming memory-bound measurement. Run Native replay plus the two-cluster WASM replay/retention selectors that exercise this evidence profile. Shared gates run serially on the integrated frozen candidate.

## Acceptance
One World snapshot profile remains; no JSON or base64 World envelope and no hand-written World token scanner remain; size, count, nesting and duplicate-key bounds are enforced while decoding; tampered, truncated and oversized inputs are refused with existing error classes; two-cluster WASM replay and native World replay pass; measured peak decode memory on the two-cluster corpus stays within the 2 GiB operator budget; the World identity rotation lands in the fn-153/fn-154/fn-166 batch.

## Done summary
Replaced the handwritten World structured-payload codec with a repository-generated endpoint whose checked-in schema owns field ordering, transition ordering, tags, and bounds; generator drift and schema-order tests cover the ownership contract. Raised the sole compressed World profile to admit the unchanged implicit `gomad3sim.DefaultLimits` mapping and added a broker-path regression, while preserving the binary gzip envelope, bounded streaming and semantic replay validation, refusal classes, and producer/consumer identities.

Focused World, Native, gomad3sim, WASM owner/profile, generator, arch/purity, race, vet, compressed two-guest record/replay, and 200k probe gates pass at commit `f53809edb7ab8ac2d538a59d379f690a10fd2ff9`. The 200,000-request probe measured 639,631,360 bytes peak RSS, below 2 GiB. `make -C tools/gomad3 validate` again stopped only at the authorized inherited stale `golang-x-sys-v047-darwin-arm64` approval owned by fn153.17, after generator validation passed.

Baseline remains green via the conductor handoff at `660ad8ca35f4ed540a1e698cc90eefe3fd219e18`. The retired module-root architecture command and unconfigured `GOMAD3_STOCK_GO` guest gates remain inherited environment/baseline limitations; their live arch/purity and pinned-toolchain replay successors pass.

fn153.16 can perform the repository-wide zero-reference proof and delete `internal/canonicaljson`; this task intentionally left that package and its ownership registration intact. Generator integration scope was authorized by conductor commit `9ea544bf4f`, in addition to the previously authorized matching consumer/test surfaces.

stage: impl-review - skipped(policy: parallel-wave - conductor owns the gate)
Tier: session (jev-unavailable(no_key))

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 2e6b73ad3d4b5e3851c245d79870fb6f78243179, c7f264fd296a25249084f77c34c62eb2c6f5dde3, fc7eb01666d6943a44ebcd756adfbb2e56d7acc8, 2467af3bfb4ecd8e911e020252bac6151c7bf750, ea77a2832588279dbb3b4c2db693e32eda026a26, 9ea544bf4f1628e1c1ecf874cc86a9f53f342c3f, 00c087e2859649e4c16f428a3b874328cac6832f, 15c23e42d569e17b49b88d7d5c5937f54d9f1676
- Tests:
- PRs:
## Revival trigger (2026-10-13)

Revived when fn153.16's mandatory zero-reference inventory proved that this task is a missing predecessor, not an optional tail: every remaining production import of `internal/canonicaljson` is in this task's declared World-compression, World-identity, or Native replay/inspection ownership, with matching World tests and live archcheck owner assertions. Implement the owner-approved single binary envelope, migrate Native replay comparison and implementation identity off the generic package, preserve bounded streaming decode and existing public error classes, and leave package deletion/owner-table removal to fn153.16 after a new zero-reference proof. Final approvals, compatibility packs, and generated identity outputs remain owned by fn153.17. The forced local claim is required only because Flow chain admission still sees completed parent work on an unpushed remote branch; pushing remains outside the requested scope.
