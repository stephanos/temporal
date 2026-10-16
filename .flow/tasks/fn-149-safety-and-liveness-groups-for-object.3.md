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
- `tools/umpire/check/checking.go:133` - receipt subject and progress part.
- `tools/umpire/check/checking.go:675` - progress receipt construction.
- `tools/umpire/check/checking.go:240` - public Report contract; independent Query/Progress dispatch at :292 and receipt precedence at :184.
- `tools/umpire/check/types_test.go` - public contract checks.

### Key context
This Go report lane remains parallel with .1; it needs no new grouping-discovery API. Re-anchor to the accepted integrated source baseline and actual fn-140 vocabulary. Root owns Batch 2 entry/source gates; .5's final grouping seal precedes fn-123.1, and closure waits fn-123.8. Fn-141 runs later and supplies no reporting prerequisite.

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
Blocked:
OWNER HOLD (2026-10-16): Umpire framework DSL is the top priority. This integration task is not admitted while fn-188.3 completes final native public-client verification and formal batch review. Preserve its existing source checkpoints, evidence, dependencies and unfinished acceptance obligations; this scheduling hold claims no implementation or proof completion. Completion of fn-188.3 does not automatically release this task: select later IR/exporter/schema/Go/backend work explicitly. Any model/temporal work remains independently excluded entirely, including disabled source contents, until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
