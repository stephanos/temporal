---
satisfies: [R1, R3, R4, R7]
---
# fn-152-gomad-runner-storage-on-one-append-only.5 Commit choice and simulation rounds through one strategy-parameterized owner

## Description
Implement R4 and the round-specific R3/R7 controls.

**Size:** M
**Files:** Core files (4-5): a shared campaign round owner, runner/choice_exploration_campaign.go, simulation_exploration_campaign.go, exploration_round.go and a scripted round matrix. Wider touches: delete campaign/choice_exploration_journal.go and simulation_exploration_journal.go; migrate their semantic round/replay/fault tests and affected inspect status helpers.
**Touches:** [tools/gomad3/runner/internal/campaign/**, tools/gomad3/runner/choice_exploration_campaign.go, tools/gomad3/runner/simulation_exploration_campaign.go, tools/gomad3/runner/exploration_round*.go, tools/gomad3/runner/*exploration*_test.go]

### Approach

Replace the two storage-specific exploration journal types with one shared round owner on campaign records. Keep the pure choice and simulation engines and their distinct strategy identities. A single validated round record includes all executions, candidate ordering, frontier/controller transition, counters, replay evidence and referenced payloads. Apply a whole round after durable success and reconstruct it atomically on replay. Uncommitted work may run again; no committed candidate/round does. Remove the second campaign execution append after the round commit. Frame bounds must accommodate the frozen round envelope or fail visibly before an oversized transaction is written.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/campaign/choice_exploration_journal.go:364`
- `tools/gomad3/runner/internal/campaign/simulation_exploration_journal.go:540`
- `tools/gomad3/runner/choice_exploration_campaign.go:146`
- `tools/gomad3/runner/simulation_exploration_campaign.go:161`
- `tools/gomad3/runner/exploration_round.go`
- `tools/gomad3/runner/exploration_round_test.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/campaign ./runner/internal/exploration/...; run the scripted Runner round matrix separately.

Follow the parent spec's Delivery and verification section. Retain exact selectors and current command scope; do not substitute portable coverage for supported-host evidence. If deletion/fixture migration expands the surviving implementation beyond this cohesive owner, stop for conductor scope splitting before implementation.

## Acceptance
- [ ] R4: both strategies use one shared round journal owner and neither retains its own journal type or round-directory store.
- [ ] R3/R4: one record atomically binds all round executions/frontier/counters/controller progress, and campaign replay never applies a subset or duplicate execution records.
- [ ] R3: scripted uninterrupted/resumed comparisons enumerate every round append/sync failpoint, cancellation/watchdog/policy transitions and parallel completion orders for both strategies.
- [ ] R3/R4: test identities distinguish logical round/candidate/ordinal even when every exploration candidate uses the same numeric base seed; committed rounds never rerun and interrupted rounds rerun in full.
- [ ] R7: referenced round artifacts are durably published before commit and uncommitted round files are removable only under campaign ownership.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
