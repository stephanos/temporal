---
satisfies: [R3, R4]
---
# fn-166-mini-activity-a-small-complete-model-to.4 Held sweep: mini Cases and live realization proof

## Description
**Note 2026-10-18 (one-language series):** Depends on fn-203/fn-204: mini Cases come from the Scala Temporal adapter.

Completes R4 and the executable realization portion of R3 after source task 8 and Case-layout task 6. Source authoring alone grants no lowering or live-run authority.

**Size:** M
**Files:** mini generated Case tree, existing lowering/live entrypoints and focused consumer tests only where an actual kit gap requires an approved change.
**Touches:** [model/temporal/features/activity_mini/generated/**, tools/umpire/lower/**, tools/umpire/ir/**, tests/testpilot_generated_test.go]

### Approach
- Admit only when the owner opens Case generation, Go validation and live execution. Verify task 8's source closure and task 6's Case-layout closure in the current history.
- Use the existing generator on the mini feature alone. Keep sibling features, legacy payloads, lint companions, pins and equivalence baselines untouched.
- Prove the authored realization's bindings/evidence/deadlines lower into actual Cases and at least one Case reaches a recorded live Run/verdict. Authored `.live` expectations are not evidence of execution.
- If the kit lacks a required primitive, report the exact gap and obtain authority for that change; do not stub it or fit the Model to the runtime. Do not pin mini Cases in functional tests or canary.
## Acceptance
- [ ] Owner admission and the source/layout committed prerequisites are recorded.
- [ ] Mini Cases lower and executable evidence/bindings are checked through the actual consumer; unselected and protected outputs remain unchanged.
- [ ] One Case ran live to a verdict, with its exact Case and Run recorded; any kit gap is named rather than hidden.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: