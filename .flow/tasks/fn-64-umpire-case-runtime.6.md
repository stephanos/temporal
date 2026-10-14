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
- Reuse task16's prepared outcome/value boundary and task18's reservation delivery ledger and
  header codec. Task17 already freezes carrier policy and source-node/ordinal topology; do not
  implement a second binder or correlate deliveries through a FIFO.
- Register complete compatible workflow/Nexus signatures before starting shared workers. Admit
  exact history-carried routes once, retain immutable activation data for replay, and pin Nexus
  request identities before capability publication.
- Register stable generic workflow and Nexus service entrypoints that select prepared Program
  entrypoints through Host-owned symbolic bindings.
- Interpret workflow nodes only through replay-safe SDK APIs; implement `StartNexusOperation`,
  `Await`, and `Finish` without direct server clients or verification events. Scheduling, outcomes,
  and ordinary Slots are replay-local to an activation. Use the activation reservations and
  cancellation semantics defined in the spec; do not add per-instruction controller gating.
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
- [ ] Concurrent identical prepared Runs with reordered workflow/Nexus delivery remain isolated;
  stale/crossed routes reject, replay consumes no new reservation, and registration incompatibility
  rejects before competing workers start on the same queue.
- [ ] Real SDK tests use the task16 validator and task18 ledger through workflow/Nexus interceptors;
  failed triggers, unused reservations, cancellation races and terminal release follow the spec.
- [ ] SDK tests interpret the approved workflow and Nexus-handler opcode sets and reject every
  controller/context mismatch during preparation.
- [ ] Workflow execution remains deterministic/replay-safe and opens no direct server or arbitrary
  network client.
- [ ] Async response stores completion authority opaquely; expressions/projections cannot obtain
  URL, headers, or token from that private Slot. This does not restrict ordinary fields returned by
  an authorized RPC. Crossed/conflicting/late bridge publications reject.
- [ ] Worker startup/activation/shared-worker failure affects only dependent Runs and yields bounded
  incomplete closure without leaking another Run's state.
- [ ] SDK tests exercise Stop racing an already-reserved activation and its next command: commands
  before cancellation takes effect count as in-flight work, cancellation/drain stays bounded, and
  no new activation reservation is admitted after Stop. Delayed reserved delivery and replay use
  stable activation identity; unreserved/closed-session delivery cannot start a new DAG.
- [ ] Late publications after Run return use bounded Host diagnostics without changing the returned
  Run/Verdict or another Run.
- [ ] No SDK-side evidence channel is introduced, and Nexus remains part of worker lifecycle.
- [ ] `go test -race -count=1 -tags test_dep ./tools/umpire/temporal/worker/...` passes.

## Done summary
Implemented the Temporal SDK worker Host runtime over task16 prepared programs, task17 frozen carrier topology, and task18's delivery ledger. The package now validates and shares only compatible complete worker registrations, routes exact workflow/Nexus deliveries through bounded indexes and immutable admission caches, interprets the approved workflow and Nexus opcodes, exposes lifecycle carriers/reservations for task7, and publishes asynchronous completion details only through a Run-local opaque capability and bridge.

Worker/session lifecycle is context-aware and bounded: Run IDs are globally reserved during worker creation, partial starts roll back only workers that started, duplicate acquisition cannot detach another Run, fatal worker state reaches only dependent sessions, Stop rejects new work while admitted commands retain replay-stable dispatch metadata, cancellation waits for exact admitted identities and remains retryable, and retained admissions/results/diagnostics have explicit capacity limits. Workflow replay may complete an unfinished admission exactly once through synchronized terminal state; repeated terminal delivery remains idempotent.

Focused tests cover registration ambiguity/incompatibility and startup order, concurrent identical Runs with reversed delivery, stale/crossed/replayed routes, every approved opcode and context mismatch, carrier physical binding, exact cancellation and Stop/Consume/bind races, retryable close, worker failure isolation, opaque async completion conflicts/late publication, bounded diagnostics, and absence of SDK evidence. SDK tests use the full workflow interceptor with arbitrary `converter.EncodedValues`, reject foreign workflow types, drive Start/Await/Finish, execute Nexus through a registered `nexus.ServiceRegistry` handler, and use `worker.WorkflowReplayer` with partial then completed recorded history to prove reconstruction completes the original reservation without another admission.

Baseline was green under the focused tagged normal/race commands in `.flow/tmp/fn64-task6-resume-baseline-{normal,race}.{log,rc}`. Final post-review-fix verification passed:
- `TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/temporal/worker/... ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire`
- `TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -race -count=1 -tags test_dep ./tools/umpire/temporal/worker/... ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire`
- `make fmt-imports`
- `make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false` (0 scoped issues; no global-main green claim)

Exact final gate logs and exits are recorded in `.flow/tmp/fn64-task6-resume2-final-results.json`. Gate classification was FULL. No HEAD-bound receipt is claimed because source remains intentionally uncommitted in the shared user-owned staging area.

Official `codex:gpt-5.6-sol:high` implementation review found lifecycle/replay, capacity/indexing, carrier-validation, UTF-8, and SDK-coverage issues across its fix loop. All were addressed. After the materially changed tree hit the deterministic `same-not-fixed-lineage` pre-dispatch guard, standing user authorization relayed by the conductor approved the task-scoped review-round reset; rationale and reset evidence are in `.flow/tmp/fn64-task6-resume2-review-reset-*`. The next official read-only review returned SHIP with zero introduced/pre-existing findings and R7/R9 met. Receipt: `/tmp/impl-review-receipt-fn-64-umpire-case-runtime.6.json` (SHA-256 `dfe24ac7cfae1ca0b0400adeb781451c6faa168e1a3dde1bfc7f23ef9222dac3`). The reviewer could not run Go in its read-only sandbox; the writer's recorded final gates are authoritative.

The non-trivial replay fix was captured as memory entry `bug/runtime-errors/workflow-replay-must-complete-2026-09-05`.

stage: impl-review - ran [2026-09-05T05:38:47Z..2026-09-05T06:43:35Z] | codex:gpt-5.6-sol:high | NEEDS_WORK fix loop, authorized task-scoped reset, SHIP round 4
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: concurrent-wave - skipped(policy: shared checkout; one writer)
Tracker sync: n/a (bridge inactive)

Commits: `[]` (user-owned commits; HEAD remained `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`).
Corrected resume start tree: `a1010fd9dfcfb664277d301b3c31105fd1dde844`.
Reviewed worker-owned tree: `5fb27f5c159e7948ffcab02c06dbf1310774a51a`.
Actual full staged tree at final review: `ffdec4868c86a29c7bbcb68fd3a05a2b92eaaa74`.
Worker-owned working and staged paths matched the reviewed tree exactly before lifecycle completion.
## Evidence
- Commits:
- Tests: baseline: green (.flow/tmp/fn64-task6-resume-baseline-{normal,race}.{log,rc}), TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/temporal/worker/... ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -race -count=1 -tags test_dep ./tools/umpire/temporal/worker/... ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire, make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, All final gate command exits and logs: .flow/tmp/fn64-task6-resume2-final-results.json, OWNED_TREE_MATCH: worker working/index paths equal reviewed tree 5fb27f5c159e7948ffcab02c06dbf1310774a51a, GATE_CLASSIFICATION:full - executable Umpire source changed, NO_RECEIPT: user-owned shared staged source is uncommitted; no HEAD-bound receipt is warrantable
- PRs: