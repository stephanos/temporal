---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.8 Migrate qualification sets and generated manifest inputs

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** qualification/set/set.go, set/manifestgen/manifestgen.go and soak/manifest.go with tests (3 core callers).
**Touches:** [tools/gomad3/qualification/set/set.go, tools/gomad3/qualification/set/set_test.go, tools/gomad3/qualification/set/manifestgen/**, tools/gomad3/qualification/soak/manifest*.go]

### Approach

- Switch set manifests/run keys/checkpoints/reports and generator-spec/soak manifest inputs to stdlib and strictjson.
- Preserve set merge/shard behavior, pruning ownership, complete identity inputs, support comparison and private/shared publication relationships.
- Behavior pin: frozen manifest/run/checkpoint projections, current-build repeatability, field-by-field identity sensitivity and incomplete/corrupt/stale shard controls. Recompute derived summaries only from final inputs; see memory recompute-mapping-summaries-after-final-2026-10-08.
- Inventory affected committed manifests for task17; this task does not dispatch qualification workloads or native CI.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/qualification/set/set.go:283`
- `tools/gomad3/qualification/set/set_test.go`
- `tools/gomad3/qualification/set/manifestgen/manifestgen.go:297`
- `tools/gomad3/qualification/set/manifestgen/manifestgen_test.go`
- `tools/gomad3/qualification/soak/manifest.go:96`
- `tools/gomad3/qualification/set/merge.go`
- `tools/gomad3/qualification/set/prune.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./qualification/set ./qualification/set/manifestgen ./qualification/soak

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Set and soak manifest/current checkpoint semantics, complete run keys and deterministic same-build projections survive.
- [ ] Malformed/unknown/duplicate/trailing, incompatible/current schema, incomplete shards, invalid strings/limits and stale evidence retain owned classifications.
- [ ] Generated-spec inputs use current encoding without qualification dispatch, capability changes or ownership changes; final regenerated outputs are inventoried.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
