---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.1 Introduce narrow strict JSON decoding without a shared encoder

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** S
**Files:** New internal/strictjson decoder and tests; initial pure owner/edge registration (4-5 core files).
**Touches:** [tools/gomad3/internal/strictjson/**, tools/gomad3/internal/archcheck/edges.go, tools/gomad3/internal/archcheck/architecture_test.go, tools/gomad3/go.mod, tools/gomad3/go.sum]

### Approach

- Re-anchor the post-fn152 surviving import inventory against the research's 44-file projection; assign any new consumer before migration.
- Add a narrow typed decoder around github.com/go-json-experiment/json. Preserve caller-wrapped errors while relying on the library defaults for duplicate-key, invalid-UTF-8 and trailing-data refusal; enable unknown-member rejection for owned typed schemas. Add no encoder, canonical comparison or generic reflection walk.
- Keep untyped number projection outside this helper; typed destinations retain their declared numeric decoding and bounds. Caller byte bounds and JSONL framing remain outside this helper.
- Register the new pure owner and permitted edges before consumers import it. Keep the old package until retirement.
- Behavior pin: separate old StrictDecode rejection controls from byte-spelling refusals in DecodeCanonicalJSON; retain recursive duplicate/unknown/trailing/raw-UTF8 tests and document map/case/surrogate dispositions.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/internal/canonicaljson/canonical.go:44`
- `tools/gomad3/internal/canonicaljson/canonical_test.go`
- `tools/gomad3/toolchain/installation.go:131`
- `tools/gomad3/toolchain/version/descriptor.go:68`
- `tools/gomad3/target/capability_collection.go:206`
- `tools/gomad3/internal/archcheck/architecture_test.go:225`
- `tools/gomad3/internal/archcheck/edges.go:23`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./internal/strictjson ./internal/archcheck

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.


### Revival trigger (2026-10-13)

Revived after fn-152 completed with completion review SHIP and final supported-host WASM qualification green. The owner amendment now requires github.com/go-json-experiment/json; Touches includes the module files that this dependency changes.
## Acceptance
- [ ] Typed unknown fields, malformed JSON, recursive duplicate keys, trailing values and raw invalid UTF-8 fail; legal typed input decodes without canonical spelling equality.
- [ ] Tests distinguish typed fields from maps and raw projected producer payloads, plus stdlib case/Unicode/surrogate behavior; the helper advertises only its actual guarantees.
- [ ] No encoder, recursive sorter, canonical equality or reflection-based string walk exists in the new owner; purity and declared edges pass.

## Done summary
Added the narrow strictjson decoder owner on go-json-experiment/json, registered its architecture edges, preserved encoding/json-style case aliases without delimiter folding, and pinned strict syntax and typed-schema behavior with focused regressions.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: dbda5d7399c7f10d1043f769aeb2d87f35335ef1, 53c7b62b7d67feac4d2ee7fa79676f633eb381b9, b27d345b16cc85235fc409959c7854de4885d469
- Tests: go -C tools/gomad3 test -tags test_dep -count=1 ./internal/strictjson ./internal/archcheck, go -C tools/gomad3 test -tags test_dep -count=1 ./internal/hostfs ./world, make -C tools/gomad3 validate
- PRs:
## Strict JSON library amendment (2026-10-12)

Owner decision: implement owned strict decoding with `github.com/go-json-experiment/json` and delete the hand-written strictness checks listed in the spec's owner amendment. Acceptance adds: duplicate keys, invalid UTF-8 and trailing data are rejected by the library's default options in every migrated decoder, with existing refusal tests passing.
