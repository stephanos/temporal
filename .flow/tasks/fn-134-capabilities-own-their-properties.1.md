---
satisfies: [R4, R5]
---
# fn-134-capabilities-own-their-properties.1 IR Property.origin: schema, Go reader, inertness

## Description
Add the inert `origin` to the IR `Property` before anything emits it, so R5 is pinned first. Nothing produces `origin` yet; this task proves that adding it changes no identity.

**Size:** S
**Files:** `proto/internal/temporal/server/api/umpire/v1/ir.proto`, the generated Go stubs, `tools/umpire/ir/identity_test.go`, `tools/umpire/ir/load.go` (doc comment only)
**Touches:** [proto/internal/temporal/server/api/umpire/v1/ir.proto, api/umpire/**, tools/umpire/ir/**, tools/umpire/internal/engine/**]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `message Property` (ir.proto:514-531): add `PropertyOrigin origin = 8;` with exactly `string name = 1; Position position = 2;` (reuse `Position`, ir.proto:45). Document it inert, as `Query.total` is (ir.proto:571-577).
- Regenerate the Go stubs with `make proto`. The Scala stubs regenerate from the schema through the model build (Makefile:699).
- Check `tools/umpire/internal/engine/canonical.go` and `tools/umpire/lower` for whole-message serialization of `Property`. If any exists, strip `origin` there the way `ir.WithoutChoiceNames` (ir/load.go:55-80) and `ir.WithoutTotals` (ir/totals.go:84-91) strip theirs.
- Add an identity test beside the positions-do-not-matter tests in `ir/identity_test.go`: the same Model with and without `origin` on a Property has an equal fingerprint, answer, lowering and exploration identity.

### Investigation targets
**Required:**
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:514-578`
- `tools/umpire/ir/identity_test.go`
- `tools/umpire/ir/load.go:55-80`, `tools/umpire/ir/totals.go:84-91`
- `tools/umpire/internal/engine/canonical.go`

### Key context
Memory `track-schema-inputs-before-reusing-2026-09-07`: a schema change must be a tracked build input, or the generated code goes stale. Check that the model build picks up the new field.

## Acceptance
- [ ] `Property` has `origin` with exactly a name and a position, documented inert.
- [ ] Go and Scala stubs regenerated and checked in; the gate's schema staleness check passes.
- [ ] Identity test: adding `origin` changes no fingerprint, answer, lowering or exploration identity.
- [ ] `make umpire-check-model` and `go test -tags test_dep -p 2 ./tools/umpire/ir/...` pass; `model/ir` is byte-identical.

## Done summary
The IR `Property` now has an inert `origin` (`PropertyOrigin { string name = 1; Position position = 2; }`, field 8). The Go stubs are regenerated. `ir.WithoutOrigins` strips the field at every identity that hashes the whole Model, and a cross-layer test shows that setting `origin` on every Property changes no answer, Definition ID, fingerprint, lowering, Model identity or exploration identity. Nothing emits `origin` yet.

stage: impl-review - skipped(config: REVIEW_MODE=none, DSL batch reviews at the batch end)

Tier: IMPLEMENTER claude-opus-5-5 at high (actual_model: claude-opus-5-5)

### Touches deviation (conductor: please accept or redirect)
The task named `internal/engine/canonical.go` and `lower` as the places to check for whole-message serialization. Neither serializes a Property whole. Three identity sites outside the declared Touches do hash the whole Model, and with `origin` set they changed (confirmed red-first):
- `tools/umpire/explore/explore.go` (candidate digest)
- `tools/umpire/explore/proposal.go` (promotion-source recipe and its SHA-256)
- `tools/umpire/conformance/conformance.go` (`modelIdentity`: it clears positions but would keep `origin.name`)

Each now wraps its Model in `ir.WithoutOrigins`, beside the existing `WithoutTotals` and `WithoutChoiceNames`. That is one call and a comment per site. The approach step says to strip origin wherever whole-message serialization exists. The alternative was to fold origins into `WithoutTotals`, which the callers already use, and that would have made the function's name and its closed test lie.

The cross-layer identity test is `tools/umpire/lower/origin_identity_test.go`, not `ir/identity_test.go`. The ir package may not import check, lower, explore or conformance (`TestLiveModelDependencyGraph`), so the first draft placed in ir failed that gate. External `lower` tests are the approved place where those layers meet. `tools/umpire/ir/origins_test.go` holds the ir-local part: the reader admits Properties with and without origin, and `WithoutOrigins` clears only the origins.

`tools/umpire/ir/schema_test.go` (inside Touches) lists the new field and message in `schemaAddedFields` / `schemaAddedMessages`, and `schemaAddedSupplement` sets them. That table is closed by design.

### Acceptance
- `origin` has exactly a name and a position and is documented inert (ir.proto, beside `Query.total`'s precedent).
- Go stubs: regenerated with `make protoc` using darwin tool builds (`.bin` holds linux binaries), and only `api/umpire/v1/ir.pb.go` and `ir.go-helpers.pb.go` changed. The Scala stubs are not checked in: they rebuild from the schema (`model/build/ir-scalapb.jar` depends on ir.proto in the Makefile, so the schema is a tracked build input). The gate's staleness check runs at the batch's `make umpire-check-model`, which this task did not run, per the batch rules.
- Identity: `TestAnOriginMovesNoAnswerFingerprintLoweringOrModelIdentity` uses activity-standalone, whose capabilities generate Properties. It compares every check receipt, including its Target (Definition ID) and Fingerprint, every Query's lowering and Case, and the conformance assessment binding of every lowered Query. `TestAnOriginMovesNoExplorationIdentity` uses nexus-workflow-control/nexusControl and compares every candidate's key, priority, digest, Case identity, Case bytes and proposal. Both failed with the stripping removed, on the Model identity and the digest.
- `go test ./tools/umpire/ir/...` passes. `make umpire-check-model` and the byte-identical `model/ir` check are deferred to the batch. This task changes no lifter or Model, so `model/ir` cannot move from it.

### Declared IR delta
None. Nothing emits `origin` in this task, so regeneration shows no `model/ir`, fixture, Case or lift-expected change from fn-134.1. The Scala IR jar repackages because ir.proto changed.

### Follow-ups
- model/SEMANTICS.md and model/README.md do not mention `Property.origin` yet. Task .6 (docs) should add it.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3255a1b4a6fac69039ca4cf2519b66d0d95a1421, 4fd4be4bde2fe19534bee4a82b3dd210b19abf1f
- Tests: go test -tags test_dep -count=1 -p 2 ./tools/umpire/ir/..., go test -tags test_dep -count=1 -p 2 -run Origin ./tools/umpire/lower/, go test -tags test_dep -count=1 -p 2 ./tools/umpire/explore/... ./tools/umpire/conformance/..., GATE_SKIPPED:umpire-check-model:dsl-batch - deferred to the batch's single regeneration
- PRs: