---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.4 Migrate preparation identities and preserve live-capability payloads

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** backends/native/build/{prepared_cache,target}.go; build/internal/{provenance,livecap}; focused tests and generated livecap protocol consumers. Six capability goldens belong to task17.
**Touches:** [tools/gomad3/backends/native/build/**, tools/gomad3/backends/native/generation/protocol/**, tools/gomad3/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/**]

### Approach

- Use stdlib ordinary cache/provenance identities and strictjson at typed host boundaries. Keep the complete cache key/source closure, binary size/hash, capacity/eviction and safe miss/removal behavior.
- At livecap.Decode, retain header size/count/reserved fields, expected producer and raw payload authentication before typed parsing. Reencode with the established producer field order and HTML-unescaped stdlib seam for existing wire refusal checks.
- Regenerate the two livecap protocol identity outputs immediately after validator input changes using existing protocol-generate/check commands. Inspect all twelve generator inputs; producer identity values may change together, while payload bytes and header layout stay fixed.
- Behavior pin: prepared-cache field-by-field invalidation and corrupted binary controls; provenance semantic round trip; producer/header/payload fixtures with unknown/duplicate/trailing strings and mismatched hashes.
- Do not convert provenance's direct truncation into a new atomic publisher. Final capability goldens and complete generated validation are assigned to task17.
- Migrate root target test dependencies here, including capability_source_test.go, capability_projection_test.go, coverage_test.go and the golden-test encoder. Preserve their source-closure/projection/admission assertions; task17 updates the six golden data files after final source retirement.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/backends/native/build/prepared_cache.go:45`
- `tools/gomad3/backends/native/build/prepared_cache_test.go`
- `tools/gomad3/backends/native/build/prepared_cache_digest_test.go`
- `tools/gomad3/backends/native/build/internal/provenance/store.go:14`
- `tools/gomad3/backends/native/build/internal/livecap/livecap.go:38`
- `tools/gomad3/backends/native/toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go:68`
- `tools/gomad3/backends/native/generation/protocol/protocol.go:384`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./backends/native/build/internal/provenance ./backends/native/build/internal/livecap ./backends/native/build

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.
## Acceptance
- [ ] All prepared identity inputs and retained binary verification/invalidation behavior survive; stale entries never become accepted evidence through a smaller key.
- [ ] Livecap payload bytes/header layout and authentication order match the producer fixtures; both source-derived producer identities regenerate together with their exact inputs bound.
- [ ] Required UTF-8/duplicate/unknown/trailing, sorted fact, size/count and governance failures retain refusal; ordinary cache/provenance spelling checks alone retire.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:

## Revival trigger (2026-10-13)

Revived after fn153.1 completed with a SHIP implementation review and its integrated strict-decoder, architecture and validation gates green. Paths were reconciled with fn166's completed package move into backends/native/build before dispatch. This consumer migration is disjoint from the in-flight fn153.2 and fn153.3 owners. The forced local claim is only needed because Flow chain admission requires the completed fn152 parent on an unpushed remote branch; pushing remains outside the user's requested scope.
