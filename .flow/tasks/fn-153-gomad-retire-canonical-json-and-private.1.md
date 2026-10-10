---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.1 Introduce narrow strict JSON decoding without a shared encoder

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** S
**Files:** New internal/strictjson decoder and tests; initial pure owner/edge registration (4-5 core files).
**Touches:** [tools/gomad3/internal/strictjson/**, tools/gomad3/internal/gomadtool/architecture/architecture.go, tools/gomad3/internal/gomadtool/architecture/edges.go]

### Approach

- Re-anchor the post-fn152 surviving import inventory against the research's 44-file projection; assign any new consumer before migration.
- Extract only UTF-8, duplicate-key token validation, typed stdlib decoding and EOF refusal from the old decoder. Preserve caller-wrapped errors. No encode, canonical comparison, generic reflection walk or new dependency.
- Use explicit UseNumber where a caller projects untyped numbers; typed destinations retain their own numeric decoding. Caller bounds and JSONL framing remain outside this helper.
- Register the new pure owner and permitted edges before consumers import it. Keep the old package until retirement.
- Behavior pin: separate old StrictDecode rejection controls from byte-spelling refusals in DecodeCanonicalJSON; retain recursive duplicate/unknown/trailing/raw-UTF8 tests and document map/case/surrogate dispositions.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/canonicaljson/canonical.go:44`
- `tools/gomad3/internal/canonicaljson/canonical_test.go`
- `tools/gomad3/toolchain/installation.go:131`
- `tools/gomad3/toolchain/version/descriptor.go:68`
- `tools/gomad3/target/capability_collection.go:206`
- `tools/gomad3/internal/gomadtool/architecture/architecture.go:296`
- `tools/gomad3/internal/gomadtool/architecture/edges.go:14`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./internal/strictjson ./internal/gomadtool/architecture

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Typed unknown fields, malformed JSON, recursive duplicate keys, trailing values and raw invalid UTF-8 fail; legal typed input decodes without canonical spelling equality.
- [ ] Tests distinguish typed fields from maps and raw projected producer payloads, plus stdlib case/Unicode/surrogate behavior; the helper advertises only its actual guarantees.
- [ ] No encoder, recursive sorter, canonical equality or reflection-based string walk exists in the new owner; purity and declared edges pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
