---
satisfies: [R1, R8]
---
# fn-18-versioned-umpire-artifact-boundary.2 Build the bounded strict JSON admission kernel

## Description
Implement the one bounded byte/parser/version kernel shared by every retained Artifact family.


**Size:** M
**Files:** `tools/umpire/artifact/artifact.go`, `json.go`, `errors.go`, `limits.go`, and focused tests
**Touches:** [tools/umpire/artifact/**]

### Approach
- Enforce byte, token, depth, collection, string, and payload Limits with N+1 failure before unbounded allocation.
- Reject duplicate and case-colliding keys, unknown keys, noncanonical values, trailing bytes, and wrong family/version.
- Expose stable typed errors and exact canonical-byte comparison hooks; perform no Artifact-specific semantics.

### Investigation targets
**Required:** fn-37's strict v2 Go decoder and the parent strict-admission contract.

## Acceptance
- [ ] Every malformed, canonicality, Limit, duplicate-key, family, and unsupported-version class has one stable error.
- [ ] Fuzz and boundary tests produce no panic, truncation, or permissive success.
- [ ] The kernel contains no model, Observation Evaluation, or Run Evaluation logic.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestStrictJSON`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
