---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.19 One compressed World snapshot profile with a binary envelope

## Description
Owner decision (2026-10-12). Retire the plain World snapshot profile and the JSON/base64 compressed envelope. Encode World evidence as a binary envelope (magic, lengths, SHA-256, raw compress/gzip stream) around a bounded structured encoding of the expanded snapshot, using the repository wire generator or github.com/fxamacker/cbor/v2 (decode limits, core deterministic encoding). Delete world/compressed_scan.go, the JSON envelope and base64 layer in world/compressed.go and compressed_decode.go, codec.go's bounded JSON preflight and the uint64-string MarshalJSON helpers in world/types.go where no longer needed. Decode must stream into the existing summary (counts and digests) without buffering the expanded snapshot, to stay inside the 2 GiB operator memory budget. Producers: backends/wasm/wasi/broker.go EncodeWorldEvidence and native World recording; consumers: backends/wasm/simulation/retained.go and replay/inspection.

Protobuf is excluded by owner decision (2026-10-12).

**Size:** L
**Touches:** [tools/gomad3/world/**, tools/gomad3/backends/native/replay*.go, tools/gomad3/backends/wasm/wasi/broker*.go, tools/gomad3/backends/wasm/wasi/integration_test.go, tools/gomad3/backends/wasm/simulation/retained*.go, tools/gomad3/backends/wasm/simulation/world_compressed_probe_test.go, tools/gomad3/internal/archcheck/**]

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./world ./backends/native ./backends/wasm/wasi ./backends/wasm/simulation ./internal/archcheck

Capture exact current round trips and refusal classes before replacing the envelope; add malformed/truncated/duplicate/oversized/tampered controls and a streaming memory-bound measurement. Run Native replay plus the two-cluster WASM replay/retention selectors that exercise this evidence profile. Shared gates run serially on the integrated frozen candidate.

## Acceptance
One World snapshot profile remains; no JSON or base64 World envelope and no hand-written World token scanner remain; size, count, nesting and duplicate-key bounds are enforced while decoding; tampered, truncated and oversized inputs are refused with existing error classes; two-cluster WASM replay and native World replay pass; measured peak decode memory on the two-cluster corpus stays within the 2 GiB operator budget; the World identity rotation lands in the fn-153/fn-154/fn-166 batch.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Revival trigger (2026-10-13)

Revived when fn153.16's mandatory zero-reference inventory proved that this task is a missing predecessor, not an optional tail: every remaining production import of `internal/canonicaljson` is in this task's declared World-compression, World-identity, or Native replay/inspection ownership, with matching World tests and live archcheck owner assertions. Implement the owner-approved single binary envelope, migrate Native replay comparison and implementation identity off the generic package, preserve bounded streaming decode and existing public error classes, and leave package deletion/owner-table removal to fn153.16 after a new zero-reference proof. Final approvals, compatibility packs, and generated identity outputs remain owned by fn153.17. The forced local claim is required only because Flow chain admission still sees completed parent work on an unpushed remote branch; pushing remains outside the requested scope.
