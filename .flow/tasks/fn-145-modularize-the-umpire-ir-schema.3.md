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
Replaced the Umpire IR schema's local `Empty` marker with `google.protobuf.Empty` across all nine oneof arms, regenerated the declared Go/Scala APIs, and migrated every affected Go and Scala constructor while preserving field numbers, presence, deterministic wire bytes, ProtoJSON, and fingerprints. The descriptor ledger now retires the local name across the actual message-and-enum declaration union; production Model IR/Cases and the resulting inherited batch failures remain assigned to fn-145.4.

Tier: session (jev-unavailable(no_key))

baseline: red (`go test -tags test_dep ./tools/umpire/ir/... ./tools/umpire/realization/...` failed pre-edit with 385 normalized stale `umpire.Step`/list-of-step errors assigned to fn-145.4; the final run has the identical 385-error set, with zero added or removed)

Verification:
- PASS: `mise exec -- make protoc`
- PASS: `mise exec -- make model/build/ir-scalapb.jar`
- PASS: `go test -count=1 -tags test_dep ./tools/umpire/ir -run '^TestSchema'`
- PASS: `mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen`
- PASS: `go test -count=1 -tags test_dep ./tools/umpire/realization -run '^(TestActivityExternalSettlement|TestPayloadFields|TestGuardProblem|TestTypeOf|TestSeveralValues)'`
- EXPECTED BATCH RED: `go test -tags test_dep ./tools/umpire/ir/... ./tools/umpire/realization/...` — identical inherited pre/post error signature; fn-145.4 owns the production Model IR/Cases update.

stage: impl-review - ran [2026-10-09T04:42:41Z..2026-10-09T04:57:25Z] (codex; NEEDS_WORK -> SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 913fe909482702ec140969c7b4f597f1c7cdeafe, ae0fa805d215fa4b51285cf9a58cc6c7339002d4
- Tests: mise exec -- make protoc, mise exec -- make model/build/ir-scalapb.jar, go test -count=1 -tags test_dep ./tools/umpire/ir -run '^TestSchema', mise exec -- scala-cli test --suppress-outdated-dependency-warning model/irgen, go test -count=1 -tags test_dep ./tools/umpire/realization -run '^(TestActivityExternalSettlement|TestPayloadFields|TestGuardProblem|TestTypeOf|TestSeveralValues)', EXPECTED BATCH RED (fn-145.4 production Model IR/Cases deferred; identical 385 normalized stale umpire.Step/list-of-step errors pre/post): go test -tags test_dep ./tools/umpire/ir/... ./tools/umpire/realization/...
- PRs: