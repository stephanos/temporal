---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.19 One compressed World snapshot profile with a binary envelope

## Description
Owner decision (2026-10-12). Retire the plain World snapshot profile and the JSON/base64 compressed envelope. Encode World evidence as a binary envelope (magic, lengths, SHA-256, raw compress/gzip stream) around a bounded structured encoding of the expanded snapshot, using the repository wire generator or github.com/fxamacker/cbor/v2 (decode limits, core deterministic encoding). Delete world/compressed_scan.go, the JSON envelope and base64 layer in world/compressed.go and compressed_decode.go, codec.go's bounded JSON preflight and the uint64-string MarshalJSON helpers in world/types.go where no longer needed. Decode must stream into the existing summary (counts and digests) without buffering the expanded snapshot, to stay inside the 2 GiB operator memory budget. Producers: backends/wasm/wasi/broker.go EncodeWorldEvidence and native World recording; consumers: backends/wasm/simulation/retained.go and replay/inspection.

Protobuf is excluded by owner decision (2026-10-12).

## Acceptance
One World snapshot profile remains; no JSON or base64 World envelope and no hand-written World token scanner remain; size, count, nesting and duplicate-key bounds are enforced while decoding; tampered, truncated and oversized inputs are refused with existing error classes; two-cluster WASM replay and native World replay pass; measured peak decode memory on the two-cluster corpus stays within the 2 GiB operator budget; the World identity rotation lands in the fn-153/fn-154/fn-166 batch.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
