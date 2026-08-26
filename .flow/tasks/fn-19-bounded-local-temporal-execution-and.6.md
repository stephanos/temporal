---
satisfies: [R2, R5, R7]
---
# fn-19-bounded-local-temporal-execution-and.6 Implement the Nexus SDK participant and exact force-close binding

## Description
Implement R2/R5's one SDK participant and semantic-to-operational adapter without interpreting its outputs.

**Size:** M
**Files:** `tools/umpire/temporal/nexus/participant.go`, `tools/umpire/temporal/nexus/workflow.go`, `tools/umpire/temporal/nexus/binding.go`, `tools/umpire/temporal/nexus/participant_test.go`, `tools/umpire/temporal/nexus/binding_test.go`
**Touches:** [tools/umpire/temporal/nexus/participant.go, tools/umpire/temporal/nexus/workflow.go, tools/umpire/temporal/nexus/binding.go, tools/umpire/temporal/nexus/participant_test.go, tools/umpire/temporal/nexus/binding_test.go]

### Approach
- Register one deterministic SDK caller workflow and Nexus handler against the local environment, with explicit readiness/cancellation/terminal channels and no client call from workflow code.
- Prepare exactly one started operation, realize force-close once, observe command/handler terminal receipts, check run-owned isolation, and clean all worker/client resources through the generic protocol.
- Bind only the exact Task `.5` program/target/action/occurrence/capabilities; return `unsupported` before environment use for every other semantic input.
- Disable implicit retry and record operational request/ack/status without mapping it to model outcome, state, or observation meaning.
- Test duplicate command, wrong correlation, rejection, SDK failure, timeout/cancel, handler crash, and cleanup after each partial preparation point.

### Investigation targets
**Required** (read before coding):
- Tasks `.4` and `.5` local environment/config contracts
- `temporaltest/server_test.go:31` worker/ExecuteWorkflow pattern
- `tests/nexus_workflow_update_test.go` worker/history patterns
- official SDK v1.44.0 client/worker/workflow APIs

### Acceptance
- [ ] Valid input realizes exactly one force-close control attempt and every command has one identity-bound terminal receipt.
- [ ] Unsupported/drifted semantic inputs cannot invoke the server or participant.
- [ ] SDK failures and cancellation are operational statuses only; no model outcome or Property verdict is constructed.
- [ ] Every partial startup path cleans its acquired handles exactly once.

## Acceptance
- [ ] R2/R5 one-participant force-close binding is exact and bounded.
- [ ] Focused participant/binding tests pass.
- [ ] Participant protocol stays language-neutral and non-persisted.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
