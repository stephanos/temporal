---
satisfies: [R7, R9]
---
# fn-64-umpire-case-runtime.6 Implement the Temporal worker Host runtime

## Description
Implement the worker half of the Temporal Host (R7): generic workflow and Nexus-handler
interpreters, worker lifecycle, and opaque per-Run completion-capability publication. Composite Host
wiring belongs to Task 7 so this task stays disjoint from the server task.

**Size:** M
**Files:** `tools/umpire/temporal/worker/**`, worker package documentation and focused tests
**Touches:** [tools/umpire/temporal/worker/**]

## Approach
- Register stable generic workflow and Nexus service entrypoints that select prepared Program
  entrypoints through Host-owned symbolic bindings.
- Interpret workflow nodes only through replay-safe SDK APIs; implement `StartNexusOperation`,
  `Await`, and `Finish` without direct server clients or verification events.
- Keep `RespondNexus` and Nexus service registration in worker lifecycle. Store callback URL,
  headers, and token behind an opaque Run/activation capability Slot; expose only readiness and
  authorized completion consumption.
- Share long-lived workers where permitted while isolating registration/activation failures and
  capability state per Run.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/temporal/nexus/workflow.go:13-124` — current SDK workflow and Nexus registration to
  generalize
- `tools/umpire/temporal/nexus/participant.go:46-122` — scenario-specific worker lifecycle to replace
- `tools/umpire/temporal/nexus/binding.go:9-118` — current closed scenario binding to remove
- `tests/nexus_workflow_test.go:321-415` — existing async Nexus SDK/completion behavior
- `tests/umpire4_testenv_test.go:1-44` — attached Temporal test environment convention

**Optional** (reference as needed):
- `tests/umpire2_probe_test.go:57-70` — minimal workflow Nexus client usage

## Key context
The worker runtime executes Program instructions but does not self-report a verification stream.
Contracts consume authoritative server-side observations collected by controller instructions.

## Acceptance
- [ ] SDK tests interpret the approved workflow and Nexus-handler opcode sets and reject every
  controller/context mismatch during preparation.
- [ ] Workflow execution remains deterministic/replay-safe and opens no direct server or arbitrary
  network client.
- [ ] Async response stores completion authority opaquely; expressions/projections cannot obtain URL,
  headers, or token, and crossed/conflicting/late publications are rejected and scrubbed.
- [ ] Worker startup/activation/shared-worker failure affects only dependent Runs and yields bounded
  incomplete closure without leaking another Run's state.
- [ ] No SDK-side evidence channel is introduced, and Nexus remains part of worker lifecycle.
- [ ] `go test -race -count=1 -tags test_dep ./tools/umpire/temporal/worker/...` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
