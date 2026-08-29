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
Implemented the bounded strict JSON admission kernel under `tools/umpire/artifact/**`: exact canonical-pretty bytes, typed stable errors and precedence, zero-allocation lexical accounting, capped key scanning, strict schema/version handling, and caller hooks without Artifact-family semantics. Encoded N/N+1, combined-precedence, allocation, canonicality, fuzz, and truncation coverage exercises every declared ceiling; Codex review reached SHIP after five bounded rounds.

Baseline: red as expected because the task-owned package did not exist. Final Quick/full/fuzz/race/vet/scoped-lint/artifactv2 and `make umpire-check-regression` gates passed; the broader `./tools/umpire/...` run remained inherited-red only in the two case-insensitive dynamic-config import-collision tests. Gate receipts were not written because the protected inherited `config/development.yaml` false symlink stat keeps the worktree dirty. The required review-fix memory capture was attempted but memory is not initialized.

stage: impl-review - ran [2026-08-29T00:54:12Z..2026-08-29T01:35:33Z]
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 93554eb44354c13cf8d5d7b06866f63b6e408691, c1b2f2e2f87b8d453c922e1d8d65c9cd5353d877, 5d1ee54ef6b88a6b5c389a739cb7e160026fd1d2, d2f2d265e76b52a4b8f891d6305e798e5223575b, 7101de00573c483044b70c33697547e991ba96dc
- Tests: baseline: red (mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestStrictJSON failed before edit because the task-owned package did not exist), mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestStrictJSON, mise exec -- go test -count=1 ./tools/umpire/artifact/..., mise exec -- go test -count=1 ./tools/umpire/artifact/... -run '^$' -fuzz '^FuzzStrictJSONNoPanicOrPermissiveSuccess$' -fuzztime=5s, mise exec -- go test -race -count=1 ./tools/umpire/artifact/..., mise exec -- go vet ./tools/umpire/artifact/..., .bin/golangci-lint-v2.13.1 run --config .github/.golangci.yml ./tools/umpire/artifact/..., mise exec -- go test -count=1 ./tools/umpire/internal/artifactv2/..., make umpire-check-regression, BROADER_INHERITED_RED: mise exec -- go test -count=1 ./tools/umpire/... (only two pre-existing case-insensitive dynamic-config import-collision tests failed), GATE_RECEIPT_NOT_WRITTEN:unittest - protected inherited config/development.yaml false symlink stat kept worktree dirty, GATE_RECEIPT_NOT_WRITTEN:smoke - protected inherited config/development.yaml false symlink stat kept worktree dirty
- PRs:
