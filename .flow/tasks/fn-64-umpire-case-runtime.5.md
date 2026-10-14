---
satisfies: [R6, R9]
---
# fn-64-umpire-case-runtime.5 Implement the Temporal server Host runtime

## Description
Implement the controller/server half of the Temporal Host (R6): authorized dynamic unary protobuf
transport and Nexus completion behind the root facade's public Host adapter contract. Composite Host
wiring belongs to Task 7 so this task stays disjoint from the worker task.

**Size:** M
**Files:** `tools/umpire/temporal/server/**`, server package documentation and focused tests
**Touches:** [tools/umpire/temporal/server/**]

## Approach
- Own the descriptor catalog, symbolic endpoint binding, channels, transport credentials/metadata,
  method authorization, and raw unary call results without worker imports.
- Accept a prepared method and constructed request from execution; return only raw typed response
  plus protocol status for execution-owned projection and recording.
- Wrap the existing Nexus completion HTTP client as a controller instruction consuming an opaque
  Host capability because completion is not a protobuf RPC.
- Return Host-owned effect handles promptly; keep wait/cancel/quarantine resources under Profile
  ceilings.

## Investigation targets
**Required** (read before coding):
- `tools/umpire/temporal/local/environment.go:44-81` — current mixed environment boundary to split
- `tools/umpire/executorgrpc/server.go:17-58` — generic bounded transport shape
- `proto/internal/temporal/server/api/umpire/v1/service.proto:14` — legacy RPC executor, not the new
  public boundary
- `common/nexus/nexusrpc/completion.go:15-113` — Nexus completion client to reuse
- `components/callbacks/nexus_invocation.go:49-72` — production completion-client construction
  pattern

**Optional** (reference as needed):
- `tools/umpire/executor/portable_projection_test.go` — current projection test cases

## Key context
Mechanical support for every unary method is distinct from authorization. The immutable Profile
permits roles/method patterns and owns all endpoint addresses and transport secrets.

## Acceptance
- [ ] In-process gRPC tests transport constructed requests through unrelated unary methods and
  return exact typed responses and statuses without scenario interpretation.
- [ ] Unknown, streaming, unauthorized, oversized, endpoint-failed, and cancellation cases fail at
  the correct preparation/runtime boundary.
- [ ] Non-OK statuses and transport failures remain typed instruction outcomes; execution tests own
  Slot/Observation/EmitEach assertions.
- [ ] Host-injected credentials and transport metadata are not added to Case, returned payload
  data, Run Events, or diagnostics. Declared projections of authorized RPC responses are ordinary
  data, including callback fields; no special polling restriction or response redaction is added.
- [ ] Nexus completion consumes only the opaque capability; success/failure/timeout and quarantine
  are bounded and controller-only.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/temporal/server/...` passes.

## Done summary
Implemented the Temporal controller server Host with immutable Profile snapshots, authorized dynamic unary gRPC transport, typed protocol/transport outcomes, bounded shared session/effect/quarantine resources, and controller-only opaque Nexus completion. Reuses the public Host/PreparedProgram boundary and existing Temporal Nexus PayloadSerializer; no private execution import, static rebinder, worker implementation or scenario semantics.

Baseline: green for existing root/execution contracts, formatting and scoped no-fix lint; the server package did not exist before this task. Baseline records: .flow/tmp/fn64-task5-baseline-results.json. Final3 tagged server tests, race tests, make fmt-imports and authorized scoped no-fix make lint-code all exited 0: .flow/tmp/fn64-task5-final3-results.json. Tests use CGO_ENABLED=0; race uses CGO_ENABLED=1 CC=/usr/bin/clang, with physical workspace TMPDIR recorded in every gate result. Global inherited lint issues and later worker/live/model/cutover gates are not claimed green. Gate classification was FULL; focused suites do not mint full-suite receipts.

Tests cover unrelated unary request/response shapes, preparation rejection of unknown/streaming/unauthorized methods, runtime size/endpoint/cancellation/status failures, response ownership and credential absence, concurrent sessions, identity collisions, actual Nexus success/failure/timeout/redirect/truncated-body handling, payload roundtrip through Temporal serializer and SDK converter, capability ownership/closure/capacity, and late quarantine release. A first fixture omitted Contract transitions; a later test run was aborted using SIGQUIT after its stack identified a timeout HTTP fixture with an unread request body. Both fixture faults and protobuf cache-sensitive equality were corrected; test3 passed. Final1 lint reported three local style issues, fixed before final2 passed.

Review1 returned NEEDS_WORK for cancellation during mutex contention, capability loss between bridge Consume and rejected completion acceptance, and nested Profile method allocation before validation. Focused red regressions reproduced the ownership/cancellation failures; the fixes add context-aware serialization, recoverable context-bound private completion claims, current-claim validation, permanent authority consumption only at accepted dispatch, and pre-clone per-role/aggregate counts. Final3 reran all gates after those changes. Review2 confirmed all findings fixed and returned SHIP. Its attempted tests were blocked by read-only sandbox; writer numeric gate exits are authoritative.

API/lifecycle handoff: tools/umpire/temporal/server/README.md. Task7 injects Session.NewCompletionCapability through trusted Host glue after worker reservation validation, exposes the server CapabilityBridge, and composes worker Reserve/quarantine/close. Consume returns a private claim; rejected or canceled claims are recoverable by fresh cleanup Consume, stale claims cannot accept, and accepted effects cannot restore authority. Worker SDK result paths preserve full umpire.Value via standard binary/protobuf Payload encoding. Task9 owns public Run, recorder integration, drain, cleanup, closure ordering and immutable late diagnostics. Server tests exercise the private session constructor until that public lifecycle exists; full PreparedCase/worker reuse integration remains assigned downstream.

Actual HEAD: 0fc128eb50a8c8bdc1db3eb1dfdf1445130910bf; task-start tree: 50b8859d2683c6fa62536437aefbc83783a8de2b; reviewed owned tree: bfb279d7d4650ed9eb80bf0b5b7450e91be816da (trees are not commits). Source equality verified after SHIP. No commits/pushes/worktrees authored; all source and receipts staged for the user. Review snapshot: .flow/tmp/fn64-task5-review-snapshot.json; receipt: /tmp/impl-review-receipt-fn-64-umpire-case-runtime.5.json. Review timing/logs: .flow/tmp/fn64-task5-review{1,2}-result.json and review{1,2}.log. Captured memory: bug/runtime-errors/retain-opaque-completion-authority-2026-09-05.

stage: impl-review - ran [2026-09-05T01:39:13.413706+00:00..2026-09-05T01:54:15.625331+00:00] (model: gpt-5.6-sol, effort: high; NEEDS_WORK -> SHIP)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: go test -count=1 -tags test_dep ./tools/umpire/temporal/server/..., go test -count=1 -race -tags test_dep ./tools/umpire/temporal/server/..., make fmt-imports, make lint-code GOLANGCI_LINT_BASE_REV=4c4e26ebdb15100387107f5d03daf5ce5fc01111 GOLANGCI_LINT_FIX=false
- PRs: