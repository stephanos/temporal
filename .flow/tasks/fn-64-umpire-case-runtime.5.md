---
satisfies: [R6, R9]
---
# fn-64-umpire-case-runtime.5 Implement the Temporal server Host runtime

## Description
Implement the controller/server half of the Temporal Host (R6): authorized dynamic unary protobuf
transport and Nexus completion behind execution's Host interface. Composite Host wiring belongs to
Task 7 so this task stays disjoint from the worker task.

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
- [ ] Credentials and injected metadata are absent from Case, returned payload data, Run Events, and
  diagnostics.
- [ ] Nexus completion consumes only the opaque capability; success/failure/timeout and quarantine
  are bounded and controller-only.
- [ ] `go test -count=1 -tags test_dep ./tools/umpire/temporal/server/...` passes.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
