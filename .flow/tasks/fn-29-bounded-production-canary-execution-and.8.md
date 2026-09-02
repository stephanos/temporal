---
satisfies: [R3, R4, R5, R6, R7, R8]
---
# fn-29-bounded-production-canary-execution-and.8 Compose the canary Claim Assessment controller and closed Run mode

## Description
Compose R3-R8 behind one production-fixed canary controller. The controller acquires and validates the pinned Lean-generated PortableTestPlan, executes it through fn-52's UmpireExecutor gRPC interface, then performs canary-owned Claim Assessment and publication.

**Size:** M
**Files:** `tools/canary/controller/**`, `tools/canary/cmd/umpire-assess-production-canary/**`, `tools/canary/evaluation/**`, `model/lakefile.toml`
**Touches:** [tools/canary/controller/**, tools/canary/cmd/umpire-assess-production-canary/**, tools/canary/evaluation/**, model/lakefile.toml]

### Approach
- Compose ordered input/pilot/profile/workflow-context admission, protected authority and scope preflight, lease acquisition, pinned plan/provenance admission, gRPC execution, cleanup/reconciliation/postflight, Claim Assessment, v6 construction, and exactly one publication behind a narrow interface.
- Supply executor connectivity and provenance trust from protected host configuration; accept no plan bytes, executor address, target, action, Property, or trust anchor from command inputs.
- Treat the fn-52 typed ExecutionResult as the complete semantic execution output; canary code adds operational Claim Assessment but never interprets or overrides semantic facts.
- Preserve the exact run/reconcile arguments, status 0/1/2 behavior, RPC ledger and cleanup reserve, post-dispatch evidence, and no-redispatch/no-republication guarantees.
- Register only required sibling executables and keep every canary-specific policy, credential, lease, recovery, workflow, and command under `tools/canary`.

### Investigation targets
**Required** (read before coding):
- `.flow/specs/fn-52-caller-neutral-grpc-portable-test-plans.md` — gRPC execution and provenance interface
- `.flow/specs/fn-19-bounded-local-temporal-execution-and.md` — runtime and cleanup dominance
- `.flow/specs/fn-20-local-execution-semantic-conformance.md` — semantic status authority
- `.flow/specs/fn-26-local-qualification-receipts-and-staged.md` — Claim Assessment and receipt contract
- `.flow/tasks/fn-29-bounded-production-canary-execution-and.5.md` — exact portable result admission

### Key context
The controller is deep operational composition around a stable plan/executor interface. Public Temporal gRPC is downstream target access; UmpireExecutor gRPC is the caller-neutral plan ingress. Neither is a new canary execution language.
## Acceptance
- [ ] Malformed input or missing/external/stale/crossed plan provenance performs no remote or publication I/O.
- [ ] One valid pinned plan runs through UmpireExecutor gRPC, closes cleanup/postflight, preserves the exact typed result, and publishes one admitted v6 set.
- [ ] Run mode exposes no plan bytes, executor endpoint, trust anchor, target/action/fault/Property/claim selector, or semantic override.
- [ ] Cancellation, transport ambiguity, failure, cleanup, reporting, and publication rows preserve exact facts without automatic gRPC redispatch.
- [ ] Recovery spends only the persisted reserve and cannot submit a plan or start another execution.
- [ ] R3-R8 controller, command, stage-order, status, provenance, and publication tests pass.
- [ ] Existing orchestration comments are preserved.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
