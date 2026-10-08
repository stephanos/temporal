---
satisfies: [R4]
---
# fn-149-safety-and-liveness-groups-for-object.3 Expose derived claim classification in existing reports

## Description
Expose derived claim classification in existing reports. Advances R4 of the parent spec.

**Size:** M
**Files:** `tools/umpire/check/checking.go`, `tools/umpire/check/*test.go`
**Touches:** [tools/umpire/check/checking.go, tools/umpire/check/*test.go]

### Approach
- Derive classification centrally from existing subject/declaration kinds; keep Query form and verdict separate. No new classification field in the model IR.
- Expose the derived classification in the existing public Check Report/Receipt API. Preserve existing Subject/Part meanings, source positions and assumption names; do not assume the CLI has a claim-report renderer.
- Report each progress claim's declared `within` bound separately from Receipt.Limits, which describes exploration/search limits. Add a successful-receipt regression with deliberately different claim and search bounds, asserting both values remain visible and distinct.
- Keep Found as an existential witness result even when its predicate is a safety Property. Preserve unsupported/incomplete dispositions and independent safety results beside failed progress prerequisites.
- Update focused report goldens only for the deliberate classification addition; do not change checker outcomes or add a new report command.

### Investigation targets
**Required:**
- `tools/umpire/check/checking.go:132` - receipt subject and progress part.
- `tools/umpire/check/checking.go:675` - progress receipt construction.
- `tools/umpire/check/checking.go:244` - public Report contract and receipt consumers.
- `tools/umpire/check/types_test.go` - public contract checks.

### Key context
Re-anchor paths and interfaces against completed fn-140/fn-141 and the approved schema/package moves before editing. Keep this work outside the activity batch and serialize shared regeneration with it and the schema chain; no new spec-close dependency is implied.

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/check
```

## Acceptance
- [ ] One derived classifier serves the existing affected output surfaces without duplicate IR state.
- [ ] Successful bounded-progress receipts expose the declared `within` bound separately from exploration limits; a regression uses unequal values and asserts both.
- [ ] Focused tests distinguish safety verification, Found witness, bounded progress, unsupported and incomplete outputs.
- [ ] A mixed report retains a safety counterexample when progress prerequisites fail; attribution and existing verdict meanings are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
