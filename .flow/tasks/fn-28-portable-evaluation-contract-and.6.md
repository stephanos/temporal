---
satisfies: [R4, R5]
---

# fn-28-portable-evaluation-contract-and.6 Deepen the resident executor and Evidence-closure lifecycle
## Description

Compose contract admission, the existing runner, explicit Evidence closure, portable evaluation, cleanup, and local decision mapping behind one small resident executor interface.

**Size:** L
**Files:** `tools/umpire/executor/**`, `tools/umpire/runner/**`, `tools/umpire/temporal/local/**`
**Touches:** [`tools/umpire/executor/**`, `tools/umpire/runner/**`, `tools/umpire/temporal/local/**`]

### Approach
- Expose one request/result seam; keep phases, adapters, resource accounting, and status mapping internal to the module.
- Reuse an attached authority across bounded requests while assigning fresh run correlations and owning only per-run workers/endpoints/workflows; never close the enclosing cluster/client.
- Wait for contract-declared terminal receipts and source closure within explicit Limits. Mark the executor poisoned after uncertain cleanup and reject further work.
- Guard `idle`/`active`/`poisoned` atomically: reject overlap as typed pre-I/O `busy`/`inconclusive`, return to idle only after complete cleanup, and never queue requests internally.

### Investigation targets

**Required** (read before coding):
- Existing `runner.Run`, runtime engine phases, `nexus.Binding`, and local authority/resource ownership.
- Parent contract/evaluator tasks and current cleanup/source-closure validation.
- Existing cancellation and failure classification tests.

## Acceptance
- [ ] A caller can execute a complete contract through one small interface without orchestrating admission, execution, evaluation, or cleanup phases.
- [ ] Multiple closed runs reuse the resident process/authority safely; run identity or resource leakage and post-uncertain-cleanup reuse fail closed.
- [ ] Eventual closure, deadline, cancellation, cleanup, race, and N/N+1 tests preserve independent statuses and never infer absence from quiet time.
- [ ] Overlap loses atomically before runtime I/O, active cancellation cannot expose idle early, and poisoned state permanently rejects reuse.

## Done summary
Finalized the previously implemented resident executor after auditing implementation commit `dd631aa861346487e3328f8a8a660e6789db4c3d`, task receipt `aa68a66b665921a8ffe9de12935160c87dbcc3e5`, plan-sync record `372acd4eb40e8dda10473a3b2dc163f30b2b73fc`, and the authoritative Codex SHIP review over `562b77b2b151e3c2903708485cde0163e9ed6c7b..dd631aa861346487e3328f8a8a660e6789db4c3d`. The reviewed executor/runner implementation remains unchanged and focused unit, race, fuzz, vet, task-range lint, local lifecycle/resource-ownership, and tagged integration verification is green; no product edit was warranted.

The finalization-only review inspected the intentionally empty `06c51d428a0c9bfa1dd3c75e2af9b3aaabd19c56..HEAD` range and returned scope-only NEEDS_WORK with zero introduced or pre-existing code findings, so it does not supersede the authoritative implementation review. The unrelated user modifications in `config/development.yaml` and `schema/elasticsearch/visibility/index_template_v7.json` remain untouched.

baseline: green (focused evaluation-contract, portable-evaluation, executor, and runner packages)

GATE_CLASSIFY_FULL: unrelated user-owned `config/development.yaml` working-tree modification

stage: impl-review - ran [2026-09-02T02:41:33Z..2026-09-02T02:42:11Z] (scope-only NEEDS_WORK on an empty finalization range; authoritative implementation SHIP retained)
## Evidence
- Commits:
- Tests: baseline: green (go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract/... ./tools/umpire/portableevaluation/... ./tools/umpire/executor/... ./tools/umpire/runner/...), go test -race -count=1 -tags test_dep ./tools/umpire/executor/..., go test -count=1 -tags test_dep ./tools/umpire/evaluationcontract -run '^$' -fuzz '^FuzzAdmitRejectsSingleByteContractMutations$' -fuzztime=1s, go test -count=1 -tags test_dep ./tools/umpire/portableevaluation -run '^$' -fuzz '^FuzzEvaluateFailsClosed$' -fuzztime=1s, go vet -tags test_dep ./tools/umpire/executor/... ./tools/umpire/runner/..., .bin/golangci-lint-v2.13.1 run --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=562b77b2b151e3c2903708485cde0163e9ed6c7b --config=.github/.golangci.yml ./tools/umpire/executor/... ./tools/umpire/runner/... (0 issues), go test -count=1 -tags test_dep ./tools/umpire/temporal/local/..., go test -count=1 -tags 'test_dep integration' ./tests -run '^TestUmpirePortableCanaryExecutor$' (green; no tests to run until the later integration task), GATE_CLASSIFY_FULL: unrelated user-owned config/development.yaml working-tree modification, NO_RECEIPT: gate receipt was not warrantable while unrelated user-owned config/development.yaml remained dirty, AUTHORITATIVE_REVIEW_SHIP: 562b77b2b151e3c2903708485cde0163e9ed6c7b..dd631aa861346487e3328f8a8a660e6789db4c3d, FINALIZATION_REVIEW_SCOPE_ONLY: 06c51d428a0c9bfa1dd3c75e2af9b3aaabd19c56..HEAD returned NEEDS_WORK with 0 introduced and 0 pre-existing findings because the range was empty
- PRs: