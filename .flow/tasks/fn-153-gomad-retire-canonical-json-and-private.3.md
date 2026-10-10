---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.3 Migrate execution records and I/O identity validation

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** record/{record,identity,validation}.go; deterministicio/adapter_regenerate.go and readonlymount/persistence.go; associated tests (5 core callers).
**Touches:** [tools/gomad3/record/**, tools/gomad3/deterministicio/adapter_regenerate*.go, tools/gomad3/deterministicio/readonlymount/persistence*.go]

### Approach

- Replace ordinary encodes with stdlib and typed decodes with strictjson; keep Finalize validation/hash ordering, failure versus execution projections and artifact path exclusions.
- Preserve I/O inventory as producer-owned opaque structured JSON with exact raw-byte authentication. Keep structural/string/integer-token checks from its existing accepted value contract; do not invent a closed inventory schema or claim map unknown-field rejection.
- Retain typed decimal-number rules, original-string validation, mount bounds/path sorting and all full projection inputs. The independent deterministicio/domain encoder and runtime inventory bytes remain unchanged.
- Behavior pin: split record_test.go's whitespace refusal from hash-tamper refusal; compare semantic manifests/mount descriptors and vary each identity input independently. Keep global SchemaVersion=1 and fn152 envelope lineage.
- Retain adapter review semantics and capture profile/pin regeneration impact for the final generated-input owner; see memory profile-adapter-changes-leave-libc-2026-10-01.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/record/record.go:82`
- `tools/gomad3/record/identity.go:320`
- `tools/gomad3/record/validation.go:329`
- `tools/gomad3/record/record_test.go:304`
- `tools/gomad3/deterministicio/adapter_regenerate.go:280`
- `tools/gomad3/deterministicio/readonlymount/persistence.go:148`
- `tools/gomad3/deterministicio/profile.go:204`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./record ./deterministicio/readonlymount ./deterministicio

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Record round trips preserve complete identity projections, failure/execution separation, field/pointer/nil semantics and tamper refusal while ordinary encoding bytes may change.
- [ ] Malformed, duplicate, trailing and invalid-string inputs plus inventory integer-token violations, mount/path/limit errors and decimal overflow retain refusal; opaque inventory ownership is explicit.
- [ ] Global record schema, independent I/O inventory/bootstrap bytes and reviewed adapter scope remain unchanged; affected derived values are inventoried for regeneration.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
