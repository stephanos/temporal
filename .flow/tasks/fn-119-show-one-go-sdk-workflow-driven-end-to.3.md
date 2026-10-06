---
satisfies: [R2, R9]
---
# fn-119-show-one-go-sdk-workflow-driven-end-to.3 Express workflow-scheduled activities and awaited outcomes in the realization DSL, lifter and lowering

## Description
Give realizations the generic way to say "the workflow schedules this activity, awaits it and finishes with its outcome", lift it and lower it to the instructions of tasks 1-2, including the withhold-answer attempt. Answers the question formerly parked in the spec: can the lowering place a workflow's activity attempts with the existing realization declarations, or does it need a new one?

**Cross-spec entry gate:** start after fn-114 and fn-118 are closed: the realization DSL, lifter, IR schema and `tools/umpire/lower/realization.go` stop moving, and the new lowering must sit under fn-118's read-after-write visibility refusal. Does not need fn-120.

**Size:** M
**Files:** `model/umpire/realize/Realize.scala` (`Activation` :329, `Instruction` :424-462, `Operand` :481), `model/irgen/Realizations.scala`, `tools/umpire/lower/realization.go` (activity :483, WorkflowCommand :601), `tools/umpire/model/validate_realization.go`, lifter and lowering fixtures, spec Architecture section.
**Touches:** [model/umpire/realize/**, model/irgen/**, tools/umpire/lower/**, tools/umpire/model/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/v1/**, .flow/specs/fn-119-show-one-go-sdk-workflow-driven-end-to.md]

### Approach
- Add an `Operand` for the outcome of an earlier command and an `Activation.Activity` whose starts come from a workflow command, reusing existing realization declarations if they suffice; add IR fields only if the existing ones cannot carry it (then regenerate bindings/jar and extend `schema_test.go` coverage as fn-112.11 did).
- Lower to the Testpilot instructions of tasks 1-2; fixtures in `model/irgen/testdata` and `tools/umpire/lower` use generic fixture names.
- Record the answer in the spec's Architecture section; every missing capability found goes to an R9 findings list in `.flow/tmp/fn119-3/findings.md` for task 6's done summary.

### Investigation targets
**Required:**
- `model/umpire/realize/Realize.scala:320-490`
- `model/temporal/features/nexus/workflow/Realization.scala` - workflow-command realization example
- `model/temporal/features/activity/standalone/system/Realization.scala` - attempt answers
- `tools/umpire/lower/realization.go:470-620`

### Quick commands
```bash
scala-cli test model/irgen
go test -count=1 -tags test_dep ./tools/umpire/lower/... ./tools/umpire/model/...
make umpire-check-model
```

### Execution constraints
- Existing IR and Case bytes unchanged (new forms are additive).

### Carried from fn-119.2 review (P3, deferred)
- The scheduled-activity route index duplicates the Nexus route-index code; fold them into one generic index if this task touches either.
- Scheduled-activity dispatch has no activation bound like the Nexus path (route capacity still limits it); add the same bound.
- A standalone activity start already carrying the reserved scheduled-attempt header is not refused, so its Run fails without a clear message; refuse it at preparation with a named error.
## Acceptance
- [ ] A realization can declare a workflow-scheduled activity, await it and finish with its outcome; the lifter lifts it and lowering emits the instructions of tasks 1-2 (positive and refusal fixtures).
- [ ] The realization-declaration question is answered in the spec's Architecture section.
- [ ] Gaps found are recorded for R9; existing IR and Cases unchanged; model gate passes.
## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-04 as not needed for the code deliverable (the DSL and its execution). fn-119 (the Go SDK workflow showcase) waits until it is revived; its generic Driver primitives (tasks 1-2) are done.
## Evidence
- Commits:
- Tests:
- PRs:
