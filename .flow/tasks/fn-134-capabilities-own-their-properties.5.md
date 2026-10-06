---
satisfies: [R6]
---
# fn-134-capabilities-own-their-properties.5 Remove the law sidecar reader, law table and law lint kinds from Go

## Description
Delete the Go side of laws: the sidecar reader, the law table view, the five law lint kinds, the `--update` forward that task 2's gate writer replaced, and the IR reader's sidecar skip.

**Size:** M
**Files:** `tools/umpire/check/laws.go`, `tools/umpire/lint/{laws,lawtable,holes,lint,kinds}.go`, `tools/umpire/lint/{laws_test,lawtable_test}.go`, `tools/umpire/lint/testdata/laws/*`, `tools/umpire/cmd/umpire-lint/{main,main_test}.go`, `tools/umpire/ir/load.go`, `tools/umpire/README.md`
**Touches:** [tools/umpire/check/laws.go, tools/umpire/lint/**, tools/umpire/cmd/umpire-lint/**, tools/umpire/ir/load.go, tools/umpire/README.md]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Runs after the batch's single regeneration, which deletes the `*.laws.json` sidecars; the batch's full gates follow it.

### Approach
- Delete `check/laws.go` (LawSidecar, LawClaim, LawWaiver, LawEntry, LawInstance, ReadLawSidecar, LawViolations), `lint/laws.go`, `lint/lawtable.go`, their tests and `lint/testdata/laws/`.
- `lint/lint.go`: remove the kinds at :74-87, `Options.Instances`, `Model.Laws`, `Report.Laws` and the runner registrations (:223-227). `lint/holes.go:71, 88, 819-834`: remove the law-table field and writer.
- `cmd/umpire-lint/main.go`: remove the forward (:135-175) and `instances` (:202-217), and update the header comment and the `--update` flag text. The waiver acceptances are now written by the model gate (task 2). Adapt `main_test.go:182-265`.
- `ir/load.go:19-36`: drop `LawSidecarSuffix` and the skip, so a leftover sidecar is read as IR and refused.
- Check `lint/testdata/coverage.golden` and the grouping testdata for law references.

## Acceptance
- [ ] No Go code reads a law sidecar; `umpire-lint --tables` prints no law table; the five law kinds are gone.
- [ ] A `*.laws.json` in an IR directory is refused by the reader.
- [ ] `go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/...` and `make umpire-check-lint` pass with `model/ir/*.lint.json` unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
