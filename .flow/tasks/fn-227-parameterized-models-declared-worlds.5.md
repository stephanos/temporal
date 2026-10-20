---
satisfies: [R6]
---
# fn-227-parameterized-models-declared-worlds.5 After .4: realization reads activity options into the World; Answer selects retryable; live SDK retry proof

## Description
In temporal/realization, build activity::World from the observed StartActivityExecutionRequest activity options (retry policy maximum attempts, the four timeouts mapped to tick budgets through a declared domain, cancellation type); options outside the admitted domains are Unsupported naming the field and freeze the run. Answer maps the callback payload to RespondCompleted or RespondFailed { retryable }. Extend the live SDK slice (make umpire-check-model-slice) with a worker that fails retryably once and completes on the second attempt, judged through real Describe observations showing attempt advance; keep Complete, Fail, HostRefusal and Cancellation passing. Use fn-217's catalog fill/validate for the start request if fn-217 is closed, otherwise construct the request directly and note it.

## Acceptance
- [ ] World built from observed start options; out-of-domain options Unsupported with the field named
- [ ] Live retry-then-complete run passes with attempt advancing
- [ ] Existing slice controls pass; umbrella unchanged beyond the slice

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
