---
satisfies: [R1, R6, R8]
---
# fn-18-versioned-umpire-artifact-boundary.9 Reject unsupported formats and mixed Artifact sets

## Description
Close the hard-cut error surface without implementing a migration engine.


**Size:** S
**Files:** `tools/umpire/artifact/version_test.go`, `tools/umpire/artifact/set_test.go`, and negative fixtures
**Touches:** [tools/umpire/artifact/version_test.go, tools/umpire/artifact/set_test.go, tools/umpire/artifact/testdata/unsupported/**]

### Approach
- Verify each earlier or unknown major fails with one stable unsupported-format result before field validation.
- Verify a set mixing v2 with any other version fails before relationship closure or publication.
- Reserve named migrations for a real reviewed post-v2 successor; add no production or fixture migration registry.

### Investigation targets
**Required:** task `.3`, task `.8`, and fn-37's unsupported-format classification tests.

## Acceptance
- [ ] Earlier, unknown, and mixed versions fail deterministically without returned values or writes.
- [ ] Error precedence is format classification before legacy-field or checksum validation.
- [ ] No migration, alias, downgrade, fallback, or repair path exists.

### Quick command

`mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestUnsupportedFormat`

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
