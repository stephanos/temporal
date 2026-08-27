---
satisfies: [R2, R4, R6]
---
# fn-33-run-resumable-semantic-exploration.2 Implement deterministic leases and crash-safe campaign state

## Description
Build fake-worker lease, parallelism, cancellation, checkpoint, and resume behavior for R2/R4/R6.

### Review reconciliation (normative)

The coordinator holds a nonblocking exclusive `STATE/lock` from preflight/admission through final publication. It resolves or safely creates a non-symlink state root, uses 0700 directories/0600 files, performs relative no-follow opens, and excludes locks/temps from artifact sets. Resume validates the campaign-checkpoint parent/generation graph: one genesis, contiguous parent+1 generations, one child per parent, one maximal leaf. Missing parents, cycles, forks, multiple heads, invalid complete directories, symlink/root escape, or a competing process rejects before a lease.

**Size:** M
**Files:** `tools/umpire/campaign/**`
**Touches:** [tools/umpire/campaign/**]

### Approach
- Reserve by semantic trace identity under one deterministic lease table.
- Retain attempts and late/stale results without credit.
- Stop leasing on time exhaustion and publish a new child checkpoint; retain immutable ancestors and derive the unique head without a mutable pointer.

### Investigation targets
**Required** (read before coding):
- `tools/common/artifactio/set.go:475-645` — lock/recovery behavior
- `tools/umpire/artifact` — admitted checkpoint/publication API after fn-18
- `model/Umpire/Exploration` — opaque state identity after fn-17

### Acceptance
- [ ] At most one active lease exists per semantic trace identity.
- [ ] Crash, cancellation, expiry, duplicate delivery, time exhaustion, competing-process, symlink/permission, fork, generation-gap, and multiple-head cases are deterministic.
- [ ] Resume selects the unique valid leaf and preserves all accepted/rejected/stale attempt lineage; ambiguous lineage performs no work.
## Acceptance
- [ ] R2/R4/R6 fake-worker matrices pass.
- [ ] N/N+1 parallelism and state Limits are tested.
- [ ] No semantic selection/coverage code exists in Go.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
