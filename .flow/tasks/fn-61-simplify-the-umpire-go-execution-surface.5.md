---
satisfies: [R2, R3, R5, R6]
---
# fn-61-simplify-the-umpire-go-execution-surface.5 Hide Temporal and Nexus mechanics behind the facade

## Description
Internalize the attached Temporal lifecycle and sole Nexus participant/validation implementation now that external callers use the root facade and the execution contracts are private (R2-R3).

**Size:** M
**Files:** `tools/umpire/temporal/local/**`, `tools/umpire/temporal/nexus/**`, `tools/umpire/internal/temporal/**`, `tools/umpire/umpire.go`
**Touches:** [tools/umpire/temporal/local/**, tools/umpire/temporal/nexus/**, tools/umpire/internal/temporal/**, tools/umpire/umpire.go]

### Approach
- Move attached authority ownership, fresh worker/task-queue lifecycle, isolation verification, Nexus program construction, participant execution, and output closure validation behind one internal Temporal implementation.
- Keep the small caller-owned attached-authority capability at the root construction boundary; hide `NewAttachedFactory`, `nexus.Binding`, `CheckRequest`, `NewParticipant`, environment interfaces, and runtime output types.
- Collapse pass-through local/Nexus constructors that only adapt private execution types, retaining deep submodules where SDK lifecycle or Evidence projection can be tested independently.
- Preserve borrowed-client ownership, per-run resource identity, eventual history/source draining, cleanup retry semantics, and every stable error classification.

### Investigation targets
**Required** (read before coding):
- `tools/umpire/temporal/local/attached.go:18-168` — borrowed authority and lifecycle ownership
- `tools/umpire/temporal/local/environment.go:36-135` — Temporal-specific environment boundary
- `tools/umpire/temporal/local/profile.go:30-69` — sole closed authority profile
- `tools/umpire/temporal/nexus/runner.go:11-79` — shallow binding adapter
- `tools/umpire/temporal/nexus/participant.go:48-120` — sole participant construction

**Optional** (reference as needed):
- `tools/umpire/temporal/nexus/evidence_test.go` — closure and causal-Evidence cases
- `tools/umpire/temporal/local/attached_test.go` — cleanup and ownership cases

### Key context
The external authority remains caller-owned: Umpire may create and stop run-owned workers/resources but never closes the supplied SDK client or cluster. Eventual consistency is handled by existing bounded polling/closure logic, not sleeps in callers.

### Acceptance
- [ ] External execution callers see only the root attached-authority capability and plan executor; Temporal environment, Nexus binding, participant, and output mechanics are private.
- [ ] Borrowed client/cluster ownership, fresh per-run routing, one-active-run isolation, cleanup retry/poisoning, and eventual history/source closure retain exact behavior.
- [ ] Detailed local and Nexus tests remain adjacent to their internal deep modules and use `require`, including Eventually-style assertions without sleeps.
- [ ] No compatibility aliases preserve the old local/Nexus execution constructors.
- [ ] Focused internal Temporal tests and the tagged `testcore.NewEnv` integration pass.

## Acceptance
- [ ] Temporal/Nexus execution mechanics are private and the root facade owns composition.
- [ ] Cluster ownership, isolation, eventual closure, and cleanup contracts remain exact.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
