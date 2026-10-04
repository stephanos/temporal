---
satisfies: [R4]
---
# fn-121-shard-generated-cases-per-case-in-ci.2 Pin the generated Case names against the manifest with a golden

## Description
A cluster-free test in `tests/testcore/testpilot` that holds the name contract from spec Architecture: every lowered manifest entry's derived name equals its file stem, names are unique and carry no `/` or whitespace, and the sorted list equals a committed golden rewritten only through the goldens convention. A renamed, added or removed lowered Query fails here until the golden is updated in the same change.

**Cross-spec entry gate:** after fn-112.10 (structural Case freeze); reads `model/cases/manifest.json` and changes no byte under `model/`.

**Size:** S
**Files:** new `tests/testcore/testpilot/generated_names_test.go`; new golden `tests/testcore/testpilot/testdata/generated-case-names.txt` (one name per line, sorted); a sentence in `tests/testcore/testpilot/README.md` on the name contract and how to rewrite the golden.
**Touches:** [tests/testcore/testpilot/**]

### Approach
- Load entries with `GeneratedCases(filepath.Join("..", "..", "..", "model", "cases"))` (path style of `model_fixture_test.go:27`), keep `Standing == lower.Lowered`, derive names with fn-121.1's exported function.
- Assert per entry: name == `strings.TrimSuffix(entry.File, "-case.json")`; no `/`, no `unicode.IsSpace` rune; uniqueness over the set.
- Golden: compare the sorted names joined by newline against the file; rewrite when an environment variable is `write`, following `common/testing/testpilot/evaluation/receipt_test.go:15-17` (`UMPIRE_RECEIPT_GOLDENS=write`), e.g. `UMPIRE_CASE_NAME_GOLDENS=write`. The failure message says a renamed Case moves a shard unit and names the variable.
- A manifest that cannot be read or decoded fails with `require.NoError`.

### Investigation targets
**Required:**
- `tests/testcore/testpilot/model_fixture.go:107-118`
- `tests/testcore/testpilot/model_fixture_test.go:17-30`
- `common/testing/testpilot/evaluation/receipt_test.go:1-40` - goldens convention
**Optional:**
- `model/cases/manifest.json`

### Quick commands
```bash
go test -count=1 -tags test_dep ./tests/testcore/testpilot/ -run 'GeneratedCaseName'
UMPIRE_CASE_NAME_GOLDENS=write go test -count=1 -tags test_dep ./tests/testcore/testpilot/ -run 'GeneratedCaseName'
```

### Execution constraints
- Package `tests/testcore/testpilot` only; no cluster, no `tests/testpilot_*.go` edit.

## Acceptance
- [ ] The test asserts stem equality, uniqueness and the no-`/`/no-whitespace rule for every lowered manifest entry and passes on the current manifest.
- [ ] The sorted name list equals the committed golden; a renamed entry (probe by editing a copy of the manifest or the golden) fails with a message naming the rewrite variable; `=write` regenerates the golden.
- [ ] An unreadable or undecodable manifest fails the test.
- [ ] README names the contract and the rewrite variable; no byte under `model/` changed.

## Done summary
# fn-121.2 done summary

**What changed** (commits 339ed9c0b0, a75829721d; base a61bcee9ec): `tests/testcore/testpilot/generated_names_test.go` (`TestGeneratedCaseNames`, no cluster) reads `model/cases/manifest.json`, keeps lowered entries, and holds each `GeneratedCaseName` to the file stem (file must end in `-case.json`) and to `<model>-<query>` from the entry's Model and Query, unique, no `/`, no whitespace; the sorted list must equal `testdata/generated-case-names.txt` (16 names), rewritten with `UMPIRE_CASE_NAME_GOLDENS=write` (receipt_test.go convention). The mismatch message says a renamed Case moves a shard unit and names the variable. README gains a paragraph on the contract and the variable.

**Evidence** (`.flow/tmp/fn121-2/`): name test pass; probe renaming `activity-retry` in the golden fails with the shard-unit message naming `UMPIRE_CASE_NAME_GOLDENS=write` (exit 1), `=write` restores it byte-identical (exit 0); whole package pass; `make umpire-check-cases` exit 0; no byte under `model/` changed; lint-code-fast exit 0.

**Decisions:** README paragraph placed after the intro, not in the `testdata/generated` fixture paragraph, because the names come from `model/cases`, not the functional fixture tree.

**Review:** `flowctl claude impl-review --spec claude:claude-opus-5-5:high`, round 1 SHIP with one P3 (the stem check repeated the derivation, so it could not fail); applied in a75829721d by requiring the `-case.json` suffix and checking `<model>-<query>` from independent fields. Writer and reviewer are the same family (Opus 5.5).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 339ed9c0b0, a75829721d
- Tests: go test -count=1 -tags test_dep ./tests/testcore/testpilot/ -run GeneratedCaseName (exit 0), golden rename probe (exit 1, message names UMPIRE_CASE_NAME_GOLDENS=write), UMPIRE_CASE_NAME_GOLDENS=write go test ... -run GeneratedCaseName (exit 0, golden byte-identical), go test -count=1 -tags test_dep ./tests/testcore/testpilot/ (exit 0), make umpire-check-cases (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: