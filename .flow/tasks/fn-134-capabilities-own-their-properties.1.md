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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
