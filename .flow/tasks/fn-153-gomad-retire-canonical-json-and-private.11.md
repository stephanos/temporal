---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.11 Migrate simulation identities while preserving paired target formulas

## Description
Implements R2 for this owner; see the parent Architecture and Delivery sections.

**Size:** M
**Files:** runner/internal/exploration/simulation/frontier.go; simulationrecord/{record,wire}.go; runner/simulation_exploration_campaign.go and paired fixtures/tests (4 core callers).
**Touches:** [tools/gomad3/runner/internal/exploration/simulation/**, tools/gomad3/runner/internal/exploration/simulationrecord/**, tools/gomad3/runner/simulation_exploration_campaign*.go]

### Approach

- Use stdlib host-only projections and retain paired plan/alternative/candidate/forced identity formulas against tools/gomad3sim's existing map encoders.
- Replace combinedAlternativeSetIdentity's unsorted struct through a wire-specific map/ordered projection matching the target; preserve nil/empty override and collection semantics.
- Keep target cluster records intentionally projected as map[string]RawMessage with extra producer fields allowed; strictly parse owned fields/decision arrays, preserve raw plan equality, size limits and identity checks.
- Behavior pin: paired host/target formula vectors, same-build frontier/decision repeatability, optional/nil/empty permutations and controller/round outcomes; mutate every hash input and malicious owned record field independently.
- No root-module simulation wire redesign or target source rewrite. If a paired formula cannot match without a target change, return the concrete scope conflict to the conductor.

### Investigation targets

**Required** (read before coding):

- `tools/gomad3/runner/internal/exploration/simulation/frontier.go:370`
- `tools/gomad3/runner/internal/exploration/simulationrecord/record.go:205`
- `tools/gomad3/runner/internal/exploration/simulationrecord/wire.go:114`
- `tools/gomad3/runner/internal/exploration/simulationrecord/wire_test.go`
- `tools/gomad3/runner/simulation_exploration_campaign.go:461`
- `tools/gomad3/runner/internal/exploration/simulationrecord/record_test.go`

### Verification

Focused command: go -C tools/gomad3 test -tags test_dep -count=1 ./runner/internal/exploration/simulation ./runner/internal/exploration/simulationrecord

Capture the frozen behavior pin named above, run focused negative controls and follow the parent Delivery and verification section. Declare scope growth to the conductor before implementation. Shared gates run serially on the integrated frozen candidate.

## Acceptance
- [ ] Host and target plan/alternative/candidate/forced formulas match for current vectors, including combined alternatives and nil/empty overrides.
- [ ] Projected target records explicitly retain producer-owned extra fields while owned malformed/unknown/duplicate/trailing/invalid numeric/string/decision inputs fail.
- [ ] Semantic exploration outcomes and full host identity projections match the frozen pin; raw plan authentication and runtime binary wire contracts remain intact.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
