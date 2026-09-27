---
satisfies: [R7]
---
# fn-94-simplify-the-testpilot-go-runtime.15 One scripted session in testcore and one repeated-run live helper

## Description
Lane F for `tests/testcore/testpilot` and the live tests: one scripted session with hooks replaces three, the unchanged-bytes checks share a helper, and one repeated-run assertion helper serves the start, pair and caller live tests. Disjoint from fn-94.14.

**Size:** M
**Files:** `tests/testcore/testpilot/{workflow_start_artifact_test,artifact_test,nexus_pair_artifact_test}.go`, `tests/testpilot_{workflow_start,nexus_pair,nexus_caller,live}_case_test.go`, `tests/testpilot_signature_test.go`
**Touches:** [tests/testcore/testpilot/*_test.go, tests/testpilot_workflow_start_case_test.go, tests/testpilot_nexus_pair_case_test.go, tests/testpilot_nexus_caller_case_test.go, tests/testpilot_live_case_test.go, tests/testpilot_signature_test.go]

### Approach
- Merge `workflowStartSession` (`workflow_start_artifact_test.go:364`), `artifactSession` (`artifact_test.go:520`) and `nexusPairGapSession` (`nexus_pair_artifact_test.go:133`) into one scripted session with namespace, queue and history hooks; one helper for the unchanged-bytes checks.
- One repeated-run helper (distinct Run IDs, the workflow in its own namespace, Case bytes and frozen bindings unchanged) replaces the boilerplate in `testpilot_workflow_start_case_test.go:40-59`, `testpilot_nexus_pair_case_test.go:43-74`, `testpilot_nexus_caller_case_test.go:140-215`.
- `nexusEvidenceKind` (`testpilot_live_case_test.go:272`) becomes a map; `nexusOperationCoordinates` (`:291`) reads through a getter interface.
- No live test renamed, merged or removed; the live identity count equals fn-94.2's.

### Investigation targets
**Required:**
- `tests/testcore/testpilot/workflow_start_artifact_test.go:350-420`
- `tests/testcore/testpilot/artifact_test.go:500-600`
- `tests/testpilot_nexus_caller_case_test.go:130-220`
- `tests/testpilot_signature_test.go` — fn-90's signature helpers

### Quick commands
```sh
go test -tags test_dep ./tests/testcore/testpilot/...
make umpire-check-live-tests
make lint-code-fast
```

## Acceptance
- [ ] One scripted session and one unchanged-bytes helper in testcore.
- [ ] One repeated-run helper serves the start, pair and caller live tests; no live identity renamed or removed, and the count matches fn-94.2's.
- [ ] Tests and lint pass (live tests reported not run with the reason if no cluster).


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
