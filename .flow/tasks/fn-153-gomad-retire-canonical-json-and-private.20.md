---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.20 Replace remaining WASM strict JSON scanners with the shared decoder

## Description
The integrated requirements audit found that the explicit 2026-10-12 amendment still has handwritten duplicate-key and UTF-8/trailing scanners in backends/wasm/wasi/protocol.go, backends/wasm/qualification/spec.go, backends/wasm/installation/installation.go and backends/wasm/simulation/oracles.go. Migrate these named surfaces to the pinned shared strict JSON owner; retain field ownership, protocol framing, transport buffers and mutable caller-buffer ownership, numeric semantics, capacity/depth limits, refusal classes, error precedence, and model identity inputs. Update permitted architecture edges and tests before using the shared owner. No encoder, generic canonical comparison, compatibility fallback or new handwritten strictness scanner. This is a missing explicit spec requirement, not a new feature.

**Size:** M
**Touches:** [tools/gomad3/backends/wasm/wasi/protocol*, tools/gomad3/backends/wasm/wasi/helper_model*, tools/gomad3/backends/wasm/wasi/identity*, tools/gomad3/backends/wasm/qualification/spec*, tools/gomad3/backends/wasm/qualification/*_test.go, tools/gomad3/backends/wasm/installation/**, tools/gomad3/backends/wasm/simulation/oracles*, tools/gomad3/internal/strictjson/**, tools/gomad3/internal/archcheck/**]

## Revival trigger (2026-10-13)

The final fn153 requirements audit demonstrated that the named WASM readers still contain the owner-amended handwritten JSON scanners. Finish this missing migration within the authorized shared-owner rotation batch before fn153.18's final audit and completion review. Preserve existing transport/model semantics and resource/error contracts; update generated identities from the integrated candidate and keep historical qualification bound to its original sources. Native deferrals and the no-CI/PR/push boundary remain in force.


### Identity owner binding

The caller migration also binds the actual strictjson implementation sources and pinned library version/content sum into the WASI composite identity. `internal/strictjson/identity.go` embeds source without runtime host reads; tests verify its pin against go.mod/go.sum and reject missing-source/drift. `wasi/identity.go` consumes that identity and fails closed when absent. This exact source/identity edge was authorized before implementation. Caller depth enforcement uses only the pinned tokenizer API, preserving legacy root-zero empty-container boundaries; duplicate/UTF-8/schema syntax remains library-owned.

## Verification

Run focused strictjson and all four WASM reader packages with `-tags test_dep -count=1`, existing strict transport/mutable-buffer/error-order controls and actual helper parity; run both-platform architecture/purity and generated validation. The conductor runs the standard WASM suite and final integrated rotation-batch qualification.

## Acceptance
All four named readers use library-owned strict syntax and their handwritten duplicate/UTF-8/trailing scanners are removed. Recursive duplicate keys, invalid UTF-8, trailing data and unknown owned typed fields are refused; intentionally open maps/raw producer fields retain their declared ownership. Existing transport depth/byte and model/installation/qualification/oracle bounds, mutation and precedence negatives pass. Current-source architecture and generated validation pass; changed identities are regenerated and tested. Focused portable readers and actual helper/model parity tests pass; no historical qualification is claimed for the rotated model.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
