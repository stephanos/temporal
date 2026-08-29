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
Added one bounded format-classification preflight shared with strict Artifact admission, so all expected set members and the manifest are structurally scanned and classified before any typed field, checksum, or relationship validation. Earlier v1 and unknown v3 negative fixtures cover every retained family; mixed sets return only a zero admitted value, leave caller bytes unchanged, and expose no migration, alias, downgrade, fallback, repair, or publication path.

The configured Codex review reached SHIP twice on the same receipt: once for the implementation and again after the repository legacy-vocabulary gate required the exact literal ExperimentSpec v1 negative bytes to move to the established non-scanned `.jsonfixture` shape. Focused, full, race, fuzz, static, Lean/model, and exact regression gates pass on the final reviewed HEAD.

stage: impl-review - ran [2026-08-29T09:18:24Z..2026-08-29T09:30:40Z] (model: gpt-5.6-sol)
## Evidence
- Commits: 52414e03e430f988cef08c183a49e676f8a7d317, ecf90fd85a36a80f9252685c85e5f2dba27b0efc, b527189e47d2126f43c8a2e013f78cc22a98cc96
- Tests: baseline: green (mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestUnsupportedFormat; no matching tests before task implementation), TDD RED: mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestUnsupportedFormat (mixed later members and manifest returned artifact-checksum before unsupported-format), mise exec -- go test -count=1 ./tools/umpire/artifact/... -run TestUnsupportedFormat, mise exec -- go test -count=1 ./tools/umpire/artifact/..., mise exec -- go test -race -count=1 ./tools/umpire/artifact/..., mise exec -- go test -count=1 ./tools/umpire/artifact/... -run '^$' -fuzz '^FuzzStrictJSONNoPanicOrPermissiveSuccess$' -fuzztime=5s, mise exec -- go vet ./tools/umpire/artifact/..., ./.bin/golangci-lint-v2.13.1 run --timeout 10m --new-from-rev=fa3772892b5d0c6e0f5b29c8c3188494320a5d92 --config=.github/.golangci.yml ./tools/umpire/artifact/... (0 issues), make lint-model, make umpire-check-legacy-vocabulary, policy RED: make umpire-check-regression (literal umpire-experiment/v1 negative JSON entered the retired-vocabulary scan), make umpire-check-regression, gofmt -d tools/umpire/artifact/artifact.go tools/umpire/artifact/set.go tools/umpire/artifact/set_test.go tools/umpire/artifact/version_test.go, git diff --check, impl-review: SHIP at b527189e47d2126f43c8a2e013f78cc22a98cc96 (codex:gpt-5.6-sol:high; /tmp/impl-review-receipt-fn-18-versioned-umpire-artifact-boundary.9.json), GATE_RECEIPT_NOT_WRITTEN:unittest - protected inherited config/development.yaml false-symlink status kept worktree dirty
- PRs: