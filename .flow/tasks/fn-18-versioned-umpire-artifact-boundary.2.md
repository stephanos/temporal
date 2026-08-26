---
satisfies: [R1, R8]
---
# fn-18-versioned-umpire-artifact-boundary.2 Build the bounded recursive strict JSON admission kernel

## Description
Implement R1's one reusable streaming byte/parser/version core and the common Go artifact API skeleton.

**Size:** M
**Files:** `tools/umpire/artifact/artifact.go`, `tools/umpire/artifact/json.go`, `tools/umpire/artifact/errors.go`, `tools/umpire/artifact/limits.go`, `tools/umpire/artifact/binding.go`, `tools/umpire/artifact/json_test.go`, `tools/umpire/artifact/fuzz_test.go`
**Touches:** [tools/umpire/artifact/**]

### Approach
- Add N+1 bounded streaming reads, token/depth/object/string/number checks, duplicate and case-fold collision detection before object materialization.
- Enforce one object, exact terminal LF, canonical numeric/string/token spelling, no BOM/whitespace/trailing value, and exact closed format dispatch.
- Implement common `ArtifactBinding`, family/limit tables, stable error classifications, and exact-byte canonical comparison hooks used by typed decoders.
- Reject unknown nested fields through family-provided schemas rather than Go's permissive default unmarshal behavior.
- Fuzz malformed/truncated/concatenated/deep/large/key-collision inputs and assert bounded typed outcomes without panic.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/regression/projection.go:201-292` — duplicate/trailing JSON precedent
- `tools/umpire/regression/projection_test.go:116-132` — negative fixture pattern
- `tools/common/artifactio/artifact.go:10-40` — byte/path error style
- parent spec `Strict Byte Contract` — exact limits and canonical requirements

### Acceptance
- [ ] Every R1 malformed/canonicality/limit class has one stable structured error and N+1 evidence.
- [ ] Duplicate/case-colliding and unknown nested keys fail before typed admission.
- [ ] Reader memory is bounded by the family ceiling and fuzz tests produce no panic or permissive success.
- [ ] No artifact-specific semantic interpretation exists in the kernel.

## Acceptance
- [ ] The shared strict kernel implements the exact parent byte/version contract.
- [ ] All valid/invalid token classes and family ceilings are covered.
- [ ] Focused Go tests and fuzz seed corpus pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
