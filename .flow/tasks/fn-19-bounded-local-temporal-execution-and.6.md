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
Implemented one bounded real-SDK Nexus caller and handler bound to the exact System-owned force-close program and closed local authority, including readiness, terminal history/cancellation receipts, isolation, exactly-once partial cleanup, and the zero-argument loopback HTTP prerequisite. Unsupported/drifted inputs reject before IO, single-attempt operation failures stay operational, and the canonical program version is guarded against Lean model drift.

Verification is green for the implemented runtime/local/Nexus/temporaltest and Lean LocalProfile Quick surfaces, focused live lifecycle, race, vet, and diff checks. The missing `umpire-local-run` package, `Temporal.Feature.Nexus.ExecutionTests` target, and `umpire-run-local` Make target were red before this task and remain inherited later-task surfaces.

baseline: green for implemented dependency surfaces; inherited red for three later-task Quick entries listed above

review: SHIP after one fixed P1 finding; memory capture skipped because flow memory is not initialized

stage: impl-review - ran [2026-08-29T18:04:05Z..2026-08-29T18:22:45Z]
## Evidence
- Commits: f4b2a4665e7d9c9ae5230bde2030589be300ef36, 58b512f6cc5f4ac7eeabeefa592858b5e5b0d904
- Tests: go test -count=1 ./tools/umpire/runtime/..., go test -count=1 ./tools/umpire/temporal/local/..., go test -count=1 ./tools/umpire/temporal/nexus/..., go test -count=1 ./temporaltest/..., cd model && mise exec -- lake build Temporal.System.Execution.LocalProfileTests, go test -race -count=1 ./tools/umpire/runtime/... ./tools/umpire/temporal/local/... ./tools/umpire/temporal/nexus/..., go vet ./tools/umpire/runtime/... ./tools/umpire/temporal/local/... ./tools/umpire/temporal/nexus/..., git diff --check, INHERITED_RED:go test -count=1 ./tools/umpire/cmd/umpire-local-run/... - package absent before edit, INHERITED_RED:cd model && mise exec -- lake build Temporal.Feature.Nexus.ExecutionTests - target absent before edit, INHERITED_RED:make umpire-run-local SET=tools/umpire/temporal/nexus/testdata/caller-closure-input-set OUTPUT_ROOT=/tmp/umpire-local-runs RUN_ID=umpire.local.caller-closure.run-1 - target absent before edit
- PRs: