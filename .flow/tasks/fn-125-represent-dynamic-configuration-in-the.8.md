---
satisfies: [R12]
---
# fn-125-represent-dynamic-configuration-in-the.8 Model the HSM/CHASM attempt semantics the owner chooses for the Nexus caller

## Description
**Recut 2026-10-18 (one-language series):** Owner-deferred. At revival: lowering goes to fn-203, export to fn-206/207 (fn-205 deferred), Go model valuations to the Scala checker.

Implements the Model half of R12. Where task 1's runs show the observable differs between HSM and CHASM (CHASM counts `attempt` at schedule, HSM on failure; `.plans/DYNAMIC_CONFIG.md` section 1), the caller Model reads `implementation` or the CHASM Case is recorded as an expected failure, as the owner decides.

**Owner gate:** do not start until the owner has answered Q2 in the spec (open as of 2026-10-04, pending task 1's evidence): is the attempt-counting difference intended and modeled, or a CHASM defect to report with the Model kept on HSM semantics? The conductor records the answer in the spec before claiming this task.

**Cross-spec entry gate:** not concurrent with fn-124.8. Depends on task 7.

**Size:** M
**Files:** `model/temporal/features/nexus/workflow/{Workflow.scala,system/System.scala,Realization.scala}` (and properties if an answer moves); regenerated `model/ir/**`, `model/cases/**`; plain-Scala step tests with a `given Valuation`.
**Touches:** [model/temporal/features/nexus/workflow/**, model/ir/**, model/cases/**]

### Approach
- If modeled: step functions read `implementation.value` where attempt counting differs (e.g. `failAttempt`, the spec's API sketch); the realization's `attempt == 1` poll (`nexus/workflow/Realization.scala`) follows the Model, not a constant.
- If a CHASM defect: the Model stays on HSM semantics; the CHASM retry Case is expected to fail, recorded with the upstream report drafted for the owner.
- Each other divergence task 1 or task 7 listed gets the same treatment or an owner-recorded reason.

### Investigation targets
**Required:**
- task 1's and task 7's done summaries (divergences); the owner's Q2 answer in the spec
- `model/temporal/features/nexus/workflow/system/System.scala`, `model/temporal/features/nexus/workflow/Realization.scala`
- `chasm/lib/nexusoperation/` and `service/history/hsm/nexusoperations/executors.go` attempt handling
**Optional:**
- `.plans/DYNAMIC_CONFIG.md` sections 1 and 3

### Quick commands
```bash
make umpire-check-model
make umpire-check-cases
go test -count=1 -tags 'test_dep integration' ./tests -run TestTestpilotGeneratedCases
```

### Execution constraints
- Case-byte changes only in caller Cases, recorded per valuation.
## Acceptance
- [ ] Waits for the owner's answer to Q2 (recorded in the spec); work follows that answer.
- [ ] Where the observable differs, the caller Model reads `implementation` (or, per the owner, the CHASM Case is an expected failure with an upstream report drafted); every listed divergence has a disposition.
- [ ] Each caller Case passes under its own valuation, or fails as the owner decided, naming the valuation.
- [ ] Model gate, `make umpire-check-cases`, the live generated Cases and `make lint-code-fast` pass.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: