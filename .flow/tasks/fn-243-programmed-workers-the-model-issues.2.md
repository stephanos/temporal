---
satisfies: [R2, R3]
---
# fn-243-programmed-workers-the-model-issues.2 Raw and SDK executors follow the program; Driver hold fault; judgment from server history

## Description
After .1. Executors under `model/temporal/hosts/go/reference/worker`: `raw` (plain gRPC poll, send the response message as is) first, then `sdk` (one generic activity handler). Both implement `Follow(ctx, conn, program)`, import only the Temporal API or SDK and `umpire.temporal.v1`, refuse unknown key types at start, and fail an unmatched task non-retryably with type `umpire.unprogrammed`. The Driver loses the gate, the hold-unanswered logic and the callback path; it starts the configured executor with the program bytes, implements the hold instruction on the connection it owns (unsupported hold is a DriverFault), and records the executor kind and version in the Run. The Describe projection sets `include_last_failure`, reads `outcome` and `info.last_failure`, projects multi-step paths, and projects nothing when the observed result or failure disagrees with the program row. Live slice through `umpire explore` with the raw executor: complete, retry-then-complete, fail, cancel-before-start under a hold, cancel-after-completion, host refusal; then the same slice with the sdk executor and identical judgments. Controls: a mutant raw executor completing where the program says fail and one completing with a different result are Unexplained at that observation; an empty program ends the Run Refused as unprogrammed; replay of a recorded Run without a server.
## Acceptance
- [ ] Raw executor passes the live slice through `umpire explore`; Run is the only artifact and records the executor kind
- [ ] SDK executor passes the same slice with identical judgments
- [ ] Deviation controls Unexplained; empty-program control Refused; unsupported hold is a DriverFault
- [ ] Gate, `worker_tasks` and worker callback path deleted from the Driver
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
