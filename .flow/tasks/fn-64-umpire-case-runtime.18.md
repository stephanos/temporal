---
satisfies: [R5, R6, R7, R9]
---
# fn-64-umpire-case-runtime.18 Implement bounded Host reservation delivery and carrier encoding

## Description
Build the Host-local delivery ledger and Temporal start/header adapter that task6 and task7 compose with actual SDK workers.

**Size:** M
**Files:** `tools/umpire/temporal/internal/delivery/**`, focused tests and README, server README
**Touches:** [tools/umpire/temporal/internal/delivery/**, tools/umpire/temporal/server/README.md]

### Approach
- Implement one bounded, context-cooperative ledger for explicit reservation identities, trigger binding, first delivery, replay/redelivery, cancellation, terminal release and quarantine ownership. Use fake activation handles to prove lifecycle independently of SDK registration.
- Use immutable prepared carrier topology from the preceding task. Preserve source-node/ordinal routing under out-of-order workflow and Nexus arrivals; never use a cross-Run FIFO.
- Define the private versioned canonical route codec and reserved workflow/Nexus header keys. Validate bounds and exact identities without echoing route data in diagnostics.
- Add the concrete StartWorkflowExecution request carrier: clone the constructed request, reject header collisions, verify namespace/type/queue bindings, inject only delivery metadata and enforce transmitted size. Return the request for ordinary server InvokeRPC; do not wrap or alter application Value payloads.
- Pin actual Temporal workflow RunID on admission/start result and stable Nexus RequestID on first handler delivery. Reject crossed/stale retries. Expose admission data immutably to the future SDK adapter.
- Retain every accepted handle. Retire unconsumed routes after failed triggers, cancellation or parent workflow terminal state; distinguish unused reservation release from activation success and never cancel an unpinned foreign workflow ID.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/host.go` — public handle/session contracts
- `tools/umpire/internal/execution/contracts.go` — identity/ownership semantics
- `tools/umpire/temporal/server/session.go` — bounded effects and typed unary outcomes
- `tools/umpire/temporal/server/nexus.go` — context-aware serialization and recoverable claims
- `tools/umpire/temporal/server/README.md` — completion payload/bridge handoff
- `.flow/tmp/fn64-task6-investigation.md` — pinned SDK header feasibility

### Key context
This task supplies real routing/lifecycle primitives tested with fake transports/activations. Actual SDK interpreters and registrations remain task6; public composite/live integration remains task7.

## Acceptance
- [ ] Deterministic tests cover two identical concurrent Runs, reordered arrivals, absent/crossed/stale/duplicate routes, replay without second consumption, cancellation before/during admission, trigger rejection/uncertainty and unused reservation release.
- [ ] Carrier tests prove authored request immutability, reserved-key collision rejection, namespace/type/queue mismatch rejection, final byte bounds and exact round-trip route decoding; unrelated unary calls and application payloads remain unchanged.
- [ ] Race tests prove bounded lock waits, atomic Stop/admission, one terminal release, capacity retention until actual completion, and no cross-session cancellation or closed-result mutation.
- [ ] The ledger exposes a complete task6/task7 lifecycle handoff without SDK evidence or placeholder public Run APIs. Focused tagged tests/race, formatting and scoped lint pass.

## Done summary
Implemented the bounded Host-local reservation delivery ledger and private Temporal workflow/Nexus carrier seam, including exact task17 topology binding, immutable replay admission, trigger/parent/Stop lifecycle, context-aware cancellation, and quarantine completion ownership. Added deterministic and race coverage for carrier strictness, concurrent Run isolation, terminal orderings, bounded unresolved-trigger retention, cancellation retry, and late completion.

The reopened correction fixes a production-consumer compile defect exposed by task6: `startWorkflowPath` now lives in private production delivery code instead of a same-package test file. The new external-package `production_test.go` imports and constructs the delivery package so test-only declarations cannot mask production dependencies again. Task6 worker WIP remained byte-identical before and after the correction.

Baseline:
- Original task18 baseline: green under focused tagged server/root normal and race tests.
- Correction baseline: red as expected before the fix; `go build -tags test_dep ./tools/umpire/temporal/internal/delivery` reported `undefined: startWorkflowPath` in production (`.flow/tmp/fn64-task18-correction-baseline-build.log`).

Original task18 verification:
- `make fmt-imports`
- `TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire`
- `TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -count=1 -race -tags test_dep ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire`
- `make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false` (0 scoped issues at the original final gate; no global-main green claim)

Correction verification:
- `TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go build -tags test_dep ./tools/umpire/temporal/internal/delivery` (`.flow/tmp/fn64-task18-correction-build.log`)
- `make fmt-imports` (`.flow/tmp/fn64-task18-correction-fmt-imports.log`)
- `TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/temporal/internal/delivery` (`.flow/tmp/fn64-task18-correction-normal.log`)
- `TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -count=1 -race -tags test_dep ./tools/umpire/temporal/internal/delivery` (`.flow/tmp/fn64-task18-correction-race.log`)
- `.bin/golangci-lint-v2.13.1 run --verbose --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=4c4e26ebdb15100387107f5d03daf5ce5fc01111 --config=.github/.golangci.yml ./tools/umpire/temporal/internal/delivery/...` (`.flow/tmp/fn64-task18-correction-lint-delivery.log`, 0 issues)
- `go vet -tags disable_grpc_modules,test_dep -vettool=.bin/errortype -style-check=false ./tools/umpire/temporal/internal/delivery` (`.flow/tmp/fn64-task18-correction-errortype.log`)
- The required aggregate `make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false` was attempted and is red solely because paused task6 `tools/umpire/temporal/worker/**` WIP does not compile (`.flow/tmp/fn64-task18-correction-lint.log`). It reports no delivery finding; the repo-pinned delivery-only lint above passed. Worker hashes match exactly in `.flow/tmp/fn64-task18-correction-worker-before.sha256` and `.flow/tmp/fn64-task18-correction-worker-after.sha256`.

The bounded bundle-capacity regression is `TestCompletedBeforeResponseBundlesRemainBoundedUntilFinalization`; it proves completion-before-response tombstones consume a construction ceiling, reject overflow, and reopen capacity after trigger finalization.

Official implementation reviews:
- Original review: `codex:gpt-5.6-sol:high` returned NEEDS_WORK in round 1 for terminal-before-start-response retention and post-parent-terminal admitted Nexus replay. Both were fixed and all gates rerun; round 2 returned SHIP with zero findings. Receipt: `/tmp/impl-review-receipt-fn-64-umpire-case-runtime.18.json` (SHA-256 `1e429f13e73a7a4d5202ed2bcb32cb5a5a47f4557ca855fa84a1c7a83455ed9c`).
- Correction review: `codex:gpt-5.6-sol:high` returned SHIP with zero findings on the correction-only tree. Receipt: `/tmp/impl-review-receipt-fn-64-umpire-case-runtime.18-correction.json` (SHA-256 `c4abb15523d966467a3f793b47cad388f8fd313f8d7a72d200c7a8d7ed3aa7fc`).

stage: impl-review - ran [2026-09-05T04:16:15Z..2026-09-05T05:02:04Z] | codex:gpt-5.6-sol:high | original SHIP round 2; correction SHIP round 1
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: concurrent-wave - skipped(policy: shared checkout; one writer)
Tracker sync: n/a (bridge inactive)

Commits: `[]` (user-owned commits; HEAD remained `0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf`).
Start tree: `5e022eb7015fd1e8faeb65147a5a4117afb8b247`.
Prior reviewed owned tree: `c830e7838fa667d0c9df312f0b7691aa59aea229`.
Reviewed correction tree: `fa22ab1b5edee5dc1ac71eeab4a165c61e48a460`.
Actual full staged tree at correction review: `8b2344d0fbf922e9bf4d2be847116972b7327d3f`.
Task18-owned working and staged paths matched the reviewed correction tree exactly (`.flow/tmp/fn64-task18-correction-owned-tree-verify.log`).
## Evidence
- Commits:
- Tests: ORIGINAL_GREEN: make fmt-imports, ORIGINAL_GREEN: TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire, ORIGINAL_GREEN: TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -count=1 -race -tags test_dep ./tools/umpire/temporal/internal/delivery ./tools/umpire/temporal/server/... ./tools/umpire, ORIGINAL_GREEN: make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false, CORRECTION_BASELINE_RED: TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go build -tags test_dep ./tools/umpire/temporal/internal/delivery - undefined production startWorkflowPath, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp go build -tags test_dep ./tools/umpire/temporal/internal/delivery, make fmt-imports, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=0 go test -count=1 -tags test_dep ./tools/umpire/temporal/internal/delivery, TMPDIR=/Users/stephan/Workspace/temporal/umpire/.flow/tmp/go-test-tmp CGO_ENABLED=1 CC=/usr/bin/clang go test -count=1 -race -tags test_dep ./tools/umpire/temporal/internal/delivery, INHERITED_RED: make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false - paused task6 tools/umpire/temporal/worker WIP only; no task18 delivery finding, .bin/golangci-lint-v2.13.1 run --verbose --build-tags disable_grpc_modules,test_dep --timeout 10m --fix=false --new-from-rev=4c4e26ebdb15100387107f5d03daf5ce5fc01111 --config=.github/.golangci.yml ./tools/umpire/temporal/internal/delivery/..., go vet -tags disable_grpc_modules,test_dep -vettool=.bin/errortype -style-check=false ./tools/umpire/temporal/internal/delivery, OWNED_TREE_MATCH: task18 delivery and server README working/staged paths equal reviewed correction tree fa22ab1b5edee5dc1ac71eeab4a165c61e48a460
- PRs: