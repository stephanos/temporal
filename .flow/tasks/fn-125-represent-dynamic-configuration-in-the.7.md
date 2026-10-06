---
satisfies: [R8, R9, R12]
---
# fn-125-represent-dynamic-configuration-in-the.7 Encode the Nexus implementation, lower one Case per valuation and retire the switch

## Description
Implements R8, the rest of R9, and the binding half of R12. The caller declares the `implementation` setting, its realization encodes each value as a complete assignment of kit keys, its Queries bind `implementation.each`, and lowering emits one Case per valuation (owner's Q4, decided 2026-10-04). The switch is then deleted: agreement across implementations becomes each Case's own expectation under its own valuation.

**Cross-spec entry gate:** fn-118.5 done (the caller realization migrated to derived waits). Not concurrent with fn-124.8. Depends on task 1 (fixed switch and its evidence), task 3 (`under`) and task 6 (required settings with origins, registry lookup).

**Size:** L
**Files:** `model/temporal/features/nexus/workflow/{Workflow.scala,system/System.scala,Realization.scala}` (setting declared, not yet read; `encode(implementation)(...)`); `model/temporal/realize/**` (`encode`); `model/irgen/**`; `ir.proto` (`Realization.encodings`: `SettingEncoding {setting, repeated ValueEncoding {value, repeated RequiredSetting}}`) and generated Go; Testpilot `CaseProvenance.valuation`; `tools/umpire/lower/**` (one Case per valuation, manifest keyed by Query and valuation); `tests/testcore/testpilot/switch.go` (deleted), `tests/testpilot_generated_test.go`, `tests/testpilot_nexus_caller_case_test.go`, `tests/testpilot_run_case_test.go`; `tests/testcore/testpilot/testdata/generated-case-names.txt` (fn-121's golden, regenerated); `model/cases/**`.
**Touches:** [model/temporal/features/nexus/workflow/**, model/temporal/realize/**, model/irgen/**, model/ir/**, model/cases/**, proto/internal/temporal/server/api/umpire/v1/**, proto/internal/temporal/server/api/testpilot/v1/**, api/umpire/v1/**, api/testpilot/v1/**, tools/umpire/lower/**, tools/umpire/model/**, common/testing/testpilot/**, tests/testcore/testpilot/**, tests/testpilot_*_test.go]

### Approach
- `enum Implementation derives Finite { hsm, chasm }`; `val implementation = setting[Implementation]` in the caller Model. The Model does not read it yet (task 8 decides, after the owner's Q2); a Query may bind a setting its machine does not read.
- The caller realization encodes every value with every key that selects the path: `hsm` matches what the upstream HSM suites set, `chasm` matches `tests/nexus_workflow_test.go:82-94` (rollout 100). Refuse a value with no encoding, an encoding naming an undeclared key, and two values with equal encodings, at their line.
- Lowering: one Case per valuation, the valuation in its name, manifest entry and provenance, its encoding in the required settings with origin `implementation=<value>`, merged with the preconditions of task 6 under the same conflict rule.
- Delete `switch.go`'s switch and `CheckSwitchAgreement`, the per-value subtests and their callers; the harness runs each Case once from its required settings.
- Record the Case-byte delta (caller Case files double and are renamed) and regenerate fn-121's Case-name golden in the same change.
- A Case that fails under one valuation is a failing Case naming its valuation (R12's error clause). Do not hide it; list it for the owner's Q2 and task 8.

### Investigation targets
**Required:**
- `model/temporal/features/nexus/workflow/{Workflow.scala,system/System.scala,Realization.scala}`
- `tools/umpire/lower/lower.go` (Case naming, manifest); `tests/testcore/testpilot/generated_names_test.go`
- `tests/testcore/testpilot/switch.go` and every caller (`grep -rn NexusImplementationSwitch tests`)
- task 1's done summary (divergences)
**Optional:**
- `.flow/specs/fn-121-shard-generated-cases-per-case-in-ci.md`

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-cases
UMPIRE_CASE_NAME_GOLDENS=write go test -tags test_dep ./tests/testcore/testpilot -run GeneratedCaseNames
go test -count=1 -tags 'test_dep integration' ./tests -run 'TestTestpilotGeneratedCases|TestTestpilotNexusCaller'
```

### Execution constraints
- Case-byte changes are only the recorded per-valuation caller Cases and their required settings; non-caller Cases are byte-identical.
- No Model reads `implementation` here.

## Acceptance
- [ ] The caller declares `implementation` and its realization encodes `hsm` and `chasm` as complete, distinct assignments of kit keys matching the upstream suites; a missing encoding, an undeclared key and equal encodings are refused at their line.
- [ ] The caller Queries bind `implementation.each`; lowering emits one Case per valuation with the valuation in name, manifest entry and provenance and the encoding in its required settings.
- [ ] `switch.go`'s switch and `CheckSwitchAgreement`, the per-value subtests and their callers are deleted; each Case runs once from its required settings.
- [ ] The Case-byte delta is recorded and fn-121's Case-name golden regenerated; a Case failing under one valuation is reported naming the valuation.
- [ ] `make umpire-check-cases`, tooling and Testpilot tests, the live generated Cases and `make lint-code-fast` pass (or a valuation's failure is listed for task 8).


## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-05 together with the whole of fn-125 (dynamic configuration in the Models). Task 1 (the HSM/CHASM switch fixes) is done and merged; revive the spec to continue.
## Evidence
- Commits:
- Tests:
- PRs:
