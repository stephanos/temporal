---
satisfies: [R1, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.1 Bind generated operation references to checked schema identities

## Description
Bind generated operation references to checked schema identities for the referenced parent requirements.

**Size:** M
**Files:** tools/umpire/cmd/umpire-gen-lean-api/**; model/Temporal/API.lean; model/Temporal/API/**; model/Umpire/Operation.lean; model/Umpire/Operation/**; model/UmpireTests.lean; Makefile
**Touches:** [tools/umpire/cmd/umpire-gen-lean-api/**, model/Temporal/API.lean, model/Temporal/API/**, model/Umpire/Operation.lean, model/Umpire/Operation/**, model/UmpireTests.lean, Makefile]

### Approach
- Before edits, freeze original source/fixture hashes and complete raw transitive axiom inventories for affected Target/Property/Query/scoped/Case declarations; retain multiline inventories and generated-name mappings for final comparison.
- Extend the generator-owned descriptor projection with the exact method/request/response/schema-closure identity needed for checked binding; retain descriptive Method/Bytes/MessageRef compatibility. Ordinary use references Temporal.Api.Workflowservice.V1.WorkflowService.startWorkflowExecution directly.
- Implement one checked operation declaration boundary for generated unary methods and separately typed SDK-command/event declarations. Validate structural provenance and method identity, not request-type equality or caller flags; keep product transition alternatives with Target.
- Add forged-name/schema, same-request-different-method, streaming and deterministic descriptor-reorder controls; track complete schema input closure and regenerate the complete owned surface through existing generator routes.

### Investigation targets
**Required:**
- model/Temporal/API/Proto.lean:25 — existing forgeable phantom Method shape.
- model/Temporal/API.lean:43 — real generated WorkflowService namespace.
- tools/umpire/cmd/umpire-gen-lean-api/model.go:219 — method schema/streaming metadata.
- tools/umpire/cmd/umpire-gen-lean-api/generate.go:56 — complete artifact validation.
- model/Umpire/Target/Semantics.lean:15 — checked semantic boundary.

### Quick commands
`make umpire-check-lean-api`
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api`
`cd model && mise exec -- lake build Umpire.TargetTests Umpire.Target.ImportTests Temporal.Feature.Nexus3.Tests`

`cd model && mise exec -- lake build Umpire.Operation.Tests`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

Create and wire the proposed test module Umpire.Operation.Tests into its existing aggregate test root; if an existing equivalent is reused, record its exact name and run it explicitly. The new-module Quick command applies after creation; baseline existing roots before editing instead of treating an absent module as an environmental failure.

The permanent Lean API gate must execute basic/empty-service generated fixture checks and the real Temporal binding checks, without rewriting checked-in fixtures or silently skipping; it participates in the existing regression target. Checked RPC declarations retain the generated owner and payload-indexed witness through conversion, so a fabricated same-name owner cannot satisfy a generated-owner consumer.
## Acceptance
- [ ] Original raw trust and compatibility substrate is recorded before the first source edit and parses without missing/truncated entries.
- [ ] Generated unary binding and distinct SDK/event kinds admit through checked declarations; forged method/schema and streaming controls reject with attributable diagnostics.
- [ ] Method references, schema provenance and field identity are independent of generated spelling/source order; unchanged generated semantics retain compatibility.
- [ ] Focused generator and downstream tests pass; intentional generation uses the owner and no test silently rewrites fixtures.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: