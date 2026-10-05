---
satisfies: [R13]
---
# fn-125-represent-dynamic-configuration-in-the.11 Document settings, preconditions and bound assumptions and close fn-125

## Description
Implements R13 and closes fn-125. Document how to declare, bind and encode a setting, a precondition and a bound assumption; state QLF-01 for Profiles in the Testpilot README; run every gate.

**Cross-spec entry gate:** all fn-125 tasks done (task 8 implies the owner answered Q2).

**Size:** S
**Files:** `model/README.md` (settings, `under`, `encode`, preconditions, assumptions, the four kinds table); `common/testing/testpilot/README.md` (QLF-01: a Profile value must be behavior-neutral; fail-closed required settings); `model/SEMANTICS.md` cross-check; the spec's Requirement coverage.
**Touches:** [model/README.md, model/SEMANTICS.md, common/testing/testpilot/README.md, .flow/specs/fn-125-represent-dynamic-configuration-in-the.md]

### Approach
- Examples come from the real declarations (caller `implementation`, Pause/Unpause and `workerStop` preconditions, timer assumption).
- Confirm the `workerStop` fan-out precondition's removal condition (upstream fix) is stated where the precondition is declared.
- Collect the recorded Case-byte deltas of tasks 6-10 in the done summary.

### Investigation targets
**Required:**
- `model/README.md:319-323,786-792` (current `RequiredSetting` text)
- `UMPIRE4_SPEC.md:516-517` (QLF-01)

### Quick commands
```bash
make umpire-check-model
make lint-model
go test -tags test_dep -p 2 -timeout 30m ./tools/umpire/... ./common/testing/testpilot/...
make umpire-check-cases
make umpire-check-live-tests
GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast
```

### Execution constraints
- Docs only, plus fixes a gate demands.

## Acceptance
- [ ] `model/README.md` explains declaring, binding and encoding a setting, a precondition and a bound assumption; the Testpilot README states QLF-01 for Profiles and fail-closed required settings.
- [ ] The model gate, `make lint-model`, Go tooling and Testpilot suites, `make umpire-check-cases` with the recorded deltas, the live generated Cases and `make lint-code-fast` pass.


## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-05 together with the whole of fn-125 (dynamic configuration in the Models). Task 1 (the HSM/CHASM switch fixes) is done and merged; revive the spec to continue.
## Evidence
- Commits:
- Tests:
- PRs:
