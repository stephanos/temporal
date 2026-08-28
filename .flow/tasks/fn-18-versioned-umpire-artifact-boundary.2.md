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
- Enforce the parent constants exactly: 32 MiB/document, 1,048,576 tokens, depth 32, 4,096 array
  items, 256 object members, 1 MiB/string, 1--512-byte identities/codes/digests, 4,096-byte
  diagnostics, and the per-family source/fact/field/payload/set ceilings, with N+1 failure before
  the N+1 allocation or append.
- Count UTF-8 bytes including the LF, JSON punctuation/scalars as tokens, the root at depth one, and
  collection entries before allocation.
- Reject duplicate and case-colliding keys, unknown keys, noncanonical values, trailing bytes, and wrong family/version.
- Expose the exact stable precedence `byte-limit`, `syntax`, `token-limit`, `depth-limit`,
  `duplicate-key`, `case-collision`, `unsupported-format`, `wrong-family`, `unknown-field`,
  `collection-limit`, `string-limit`, `payload-limit`, `malformed-value`, `noncanonical`,
  `provenance-checksum`, `artifact-checksum`, `closure`, plus exact canonical-byte comparison hooks;
  perform no Artifact-specific semantics.

### Investigation targets
**Required:** fn-37's strict v2 Go decoder and the parent strict-admission contract.

## Acceptance
- [ ] Every malformed, canonicality, Limit, duplicate-key, family, and unsupported-version class has one stable error.
- [ ] Table-driven N/N+1 tests prove every numeric ceiling, accounting rule, and error-precedence row.
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
