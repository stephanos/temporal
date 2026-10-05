---
satisfies: [R11]
---
# fn-125-represent-dynamic-configuration-in-the.10 Record a disposition for every implicit assumption and state request fields in the realization

## Description
Implements R11. Every implicit assumption in `.plans/DYNAMIC_CONFIG.md` section 3 gets a recorded disposition: request field, precondition, bound assumption, Model setting, behavior-neutral, or out of scope with a reason. Where a request field can state the value, the realization sets it, so the value is per Case and visible in its bytes.

**Cross-spec entry gate:** coordinate with fn-124.3 (activity attempts, timeouts in the realization); not concurrent with fn-124.8. Depends on task 7 and task 9.

**Size:** M
**Files:** `model/temporal/features/standaloneactivity/Realization.scala` (e.g. `retry_policy` in every start, replacing reliance on `history.defaultActivityRetryPolicy`); `model/temporal/features/nexuscaller/Realization.scala` (inert `timeoutMs`); the disposition table (in `model/README.md` or beside the kit declarations, decided here); regenerated `model/cases/**`.
**Touches:** [model/temporal/features/**/Realization.scala, model/temporal/realize/**, model/ir/**, model/cases/**, model/README.md]

### Approach
- One row per section-3 assumption with its disposition and the declaration or reason that carries it. The parked unknowns (`recordCancelRequestCompletionEvents`, `defaultActivityRetryPolicy`) are decided per Case here.
- Request fields replace server defaults where they can (activity retry policy, deadlines); `limit.scheduleToCloseTimeout` and the callback URL template are dispositioned explicitly.
- An assumption without a disposition is listed for the owner, not dropped.
- Record the Case-byte delta of each request-field change.

### Investigation targets
**Required:**
- `.plans/DYNAMIC_CONFIG.md` section 3
- `model/temporal/features/standaloneactivity/{Model,Realization}.scala` (attempt bound, backoff); `nexuscaller/Realization.scala:370-410`
**Optional:**
- `tests/testcore/dynamic_config_overrides.go` (behavior-neutral by assumption; no audit beyond section 3)

### Quick commands
```bash
make umpire-gen-model && git diff --stat model/ir model/cases
make umpire-check-cases
go test -count=1 -tags 'test_dep integration' ./tests -run TestTestpilotGeneratedCases
```

### Execution constraints
- Case bytes change only by recorded request-field deltas.

## Acceptance
- [ ] Every section-3 assumption has a recorded disposition with its carrying declaration or reason; any without one is listed for the owner.
- [ ] Where a request field can state the value, the realization sets it; the Case-byte delta is recorded.
- [ ] `make umpire-check-cases`, the live generated Cases and `make lint-code-fast` pass.


## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-05 together with the whole of fn-125 (dynamic configuration in the Models). Task 1 (the HSM/CHASM switch fixes) is done and merged; revive the spec to continue.
## Evidence
- Commits:
- Tests:
- PRs:
