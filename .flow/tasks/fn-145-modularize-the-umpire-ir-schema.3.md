---
satisfies: [R4]
---
# fn-145-modularize-the-umpire-ir-schema.3 Replace the local Empty marker with protobuf Empty

## Description
Apply R4 after the declaration-only split is proven. Replace only the empty payload type while retaining every oneof arm as the meaning-bearing discriminator.

**Size:** S
**Files:** `proto/internal/temporal/server/api/umpire/v1/common.proto`, `proto/internal/temporal/server/api/umpire/v1/*.proto`, `api/umpire/v1/*.pb.go`, Umpire generated callers and tests
**Touches:** [proto/internal/temporal/server/api/umpire/v1/*.proto, api/umpire/v1/*.pb.go, model/irgen/**, tools/umpire/**]

### Approach
- Replace surviving local marker fields with `google.protobuf.Empty`.
- Record the retired top-level declaration name in the descriptor ledger; protobuf reservations apply only to retired field coordinates inside surviving messages.
- Update generated API call sites mechanically, then compare oneof presence, wire bytes and behavior.

### Investigation targets
**Required** (read before coding):
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:45-110` - current marker declaration and value/type arms
- `tools/umpire/ir/schema_test.go:125-210` - descriptor and wire assertions
- `model/check/Gate.scala:160-181` - generated Scala schema inputs
- `proto/internal/temporal/server/api/taskqueue/v1/message.proto:1-12` - repository WKT Empty import pattern


### Quick commands

```bash
go test -tags test_dep ./tools/umpire/ir/... ./tools/umpire/realization/...
```

## Acceptance
- [ ] R4 is implemented as a separate, reviewable delta after the file split.
- [ ] Descriptor changes are limited to the standard marker type and expected file identity.
- [ ] Empty/default canonical fingerprints and wire bytes remain pinned.
- [ ] Generated callers and focused Umpire tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
