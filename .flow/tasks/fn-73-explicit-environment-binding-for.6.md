---
satisfies: [R5, R7]
---
# fn-73-explicit-environment-binding-for.6 Prove one Nexus3 Case in two live environments

## Description
Extend the live async Nexus regression to run the exact same decoded Case fixture in two provisioned environments (R5, R7). Prove physical routing and isolation without changing the Case or its checked Contract.

**Size:** M
**Files:** `tests/testpilot_async_nexus_case_test.go`, `tests/testpilot_testenv_test.go`, shared test fixture helper from task 5, focused live-test support
**Touches:** [tests/testpilot_async_nexus_case_test.go, tests/testpilot_testenv_test.go, tests/testcore/testpilot/*.go]

### Approach
- Provision two namespaces, task queues and named Nexus endpoints, and construct each SDK client from the same frozen Profile binding used to prepare its environment.
- Decode the fixture once, prepare it twice without mutation, and construct symbolic-mode Drivers with explicit transport/lifecycle inputs only.
- Run both prepared Cases and assert satisfied Verdicts, identical checked Contract/provenance/model identity, intended namespace/queue/Nexus routing and isolation between environments.
- Add a remote-absence/misconfiguration path that reaches the existing execution outcome instead of being misclassified as static validation success.

### Investigation targets
**Required** (read before coding):
- `tests/testpilot_async_nexus_case_test.go:26-117` — current one-environment live proof
- `tests/testpilot_testenv_test.go:11-14` — shared functional environment setup
- `tests/testcore/testpilot/testdata/async-nexus-case.json` — exact shared Case artifact
- `common/testing/temporaltestpilot/driver.go` — symbolic Driver construction from task 4
- `common/testing/temporaltestpilot/sdkworker/interpreter.go:79-95` — Nexus route use

**Optional** (reference as needed):
- `tests/testcore/testpilot/artifact_test.go` — offline assertions to mirror

### Key context
The SDK client does not expose reliable namespace introspection, so construction and observed routing provide the proof. Use `integration` only for this integration test.

## Acceptance
- [ ] One decoded Case fixture is prepared unchanged against two disjoint binding snapshots and run with two separately configured SDK clients/Drivers.
- [ ] Both Runs satisfy the same Contract and preserve identical Case, Contract, definition, Behavior Fingerprint and provenance bytes/meaning across environments.
- [ ] Start/history requests, worker registrations and named Nexus routing reach their intended namespace/queue/endpoint with no cross-environment effects.
- [ ] Local inconsistent/missing bindings still reject before Open; unavailable remote resources surface through existing execution outcomes.
- [ ] Concurrent/repeated Runs retain immutable snapshots and Run-derived identities.
- [ ] The focused integration command with `-tags 'test_dep integration'` and `make umpire-check-live-tests` pass.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: