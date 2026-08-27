---
satisfies: [R4, R5]
---
# fn-33-run-resumable-semantic-exploration.4 Expose umpire-fuzz list explain run and resume

## Description
Implement the thin public CLI and exact state-directory behavior for R4/R5.

### Review reconciliation (normative)

List/explain project the checked runnable bindings through fn-5's catalog; they do not create a second registry. Run/resume acquire the whole-session safe state lock and apply the unique-head algorithm before work. Stderr emits bounded `umpire-fuzz-progress/v2` NDJSON events (`started|leased|result|checkpoint|stopping|cleanup|finished`) with identity, generation, counts, remaining time, checkpoint, cleanup, and termination fields; at most 4 KiB per line and one periodic event per second. Progress is noncanonical and excluded from state identities/stdout comparisons.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-fuzz/**`, `tools/umpire/campaign/**`, `Makefile`
**Touches:** [tools/umpire/cmd/umpire-fuzz/**, tools/umpire/campaign/**, Makefile]

### Approach
- Reuse fn-5 model-owned catalog/list/explain Generated View plus the checked `RunnableExplorationBinding` from Task `.6`.
- Parse only declared environment/time/parallelism/state/seed inputs.
- Keep final canonical stdout separate from bounded progress and sanitized tooling diagnostics on stderr.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/Tool/Inspect.lean:17-88` — thin canonical command pattern
- `tools/umpire/cmd` — command layout after fn-18–20
- `Makefile:988-1032` — root-only Umpire command conventions

### Acceptance
- [ ] List/explain/run/resume grammar, final summary/status, progress event/schema/rate/size, and tooling diagnostic contracts are exact.
- [ ] Invalid or broadened Limits fail before execution.
- [ ] Existing state is exclusively locked and lineage-validated rather than reset, overwritten, or selected through a mutable pointer.
## Acceptance
- [ ] R4/R5 CLI and filesystem mutation matrices pass.
- [ ] No `temporal-model-explore` or second fuzz command remains.
- [ ] Progress never enters canonical state identity.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
