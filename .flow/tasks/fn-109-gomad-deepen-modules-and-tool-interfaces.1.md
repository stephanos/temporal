---
satisfies: [R1]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.1 Carry simulation bounds across the isolated coordinator and prove it through real isolated execution

## Description
Stage 1, first half of R1 (F1 immediate correction). The isolated coordinator drops three `CampaignSpec` fields, so a valid `--strategy=simulation-exploration` request fails validation in the child. Fix the transport and add the regression that the existing fake-coordinator tests cannot provide. The options-owner refactor is the next task; keep this one minimal so it can land before fn-108.

**Size:** M
**Files:** `tools/gomad3/runner/coordinator.go`, new `tools/gomad3/runner/coordinator_transport_test.go` (new file on purpose: fn-108 edits `runner_test.go`).
**Touches:** [tools/gomad3/runner/coordinator.go, tools/gomad3/runner/coordinator_transport_test.go, tools/gomad3/runner/testdata/**]

### Approach
- Confirmed omission (HEAD `d4d800fb47`): `coordinatorConfig` (`runner/coordinator.go:23-60`) has `MaxExecutions`, `MaxChoiceDepth`, `MaxExplorationBytes` but not `MaxForcedDecisions`, `MaxExplorationResultBytes`, `SimulationDimensionLimits` (declared `runner/runner.go:148-151`, type at `:266`). The outbound literal (`coordinator.go:105-119`) and the inbound literal in `CoordinatorMain` (`:295-307`) omit all three. `CoordinatorMain` calls `runLocal` (`:312`), whose `validateConfig` rejects the zero bound (`runner.go:1276`, then `:1282`, `:1285`). The CLI supplies them at `cmd/gomad/internal/cli/cli.go:620-625` and enables isolation at `:626`.
- Add the three fields to the wire struct and both conversions. Move the two literals into two private functions (spec to wire with the adjusted child timeout; wire to spec) so a test can call them; do not otherwise restructure.
- Transport test: a `CampaignSpec` with a distinct nonzero value in every transported field, including all six `SimulationDimensionLimits` fields (`Runtime`, `Scenario`, `Network`, `Storage`, `Fault`, `Crash`), survives encode, JSON, decode as one whole-value comparison. Add a reflection guard that fails when an exported `CampaignSpec` field is neither transported nor on an explicit non-transported list (`CoordinatorCommand`, `Progress`, `Preparer`, `Executor`, `Replayer`).
- Real isolated execution: drive `Explore` -> `runIsolated` -> child process -> `CoordinatorMain` -> `runLocal` with a real prepared target for all three strategies. `runSeed` derives the bootstrap command from `SupervisorCommand[0]` (`runner.go:1512`), so the child must be a binary that serves `DispatchPrivateMode` (`runner/private.go:15`): either a `TestMain` private-mode dispatch in the runner test binary or the built `.bin/gomad`. Existing helpers such as `TestChoiceTraceCoordinatorHelper` (`runner_test.go:1793`) decode the wire themselves and never reach `runLocal`; they do not count.
- Assert the simulation summary echoes the supplied bounds (`projectSimulationExplorationSummary`, `runner.go:1839-1850`: `MaxForcedDecisions`, `MaxResultBytes`, `Limits`). Seed and choice strategies assert their own distinguishable limits.
- Error cases through the same seam: zero/missing simulation bound, simulation bounds with the choice strategy (`runner.go:1239`), choice depth with the simulation strategy (`:1274`), malformed JSON, trailing data and an unknown field (`DisallowUnknownFields`, `coordinator.go:287-293`).

### Investigation targets
**Required:**
- `tools/gomad3/runner/coordinator.go:23-119,285-334`
- `tools/gomad3/runner/runner.go:126-177,362-377,1194-1300`
- `tools/gomad3/runner/runner_test.go:1614-1860` (isolated tests and helper pattern)
- `tools/gomad3/runner/replay_operation_test.go:789` (`toolchainRoot`), `runner/replay_io_integration_test.go:252-270` (supervisor/bootstrap helpers)
**Optional:**
- `tools/gomad3sim/scenario_control_toolchain_test.go:75-130` and `tools/gomad3/runner/internal/execution/simulation_root_integration_test.go` (a simulation-exploration-capable target built with `-tags test_dep,gomad3_toolchain`)
- `tools/gomad3/internal/gomadtool/conformance/testdata/{gotest,choice_exploration}` (seed and choice fixtures)

### Key context
If no existing fixture can complete a simulation-exploration campaign through the coordinator on this host, stop and report it: R1 is not met by a local fake-executor test, and the gap must not be papered over.

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner -run 'Coordinator|IsolatedRunner|ValidateConfig'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/... ./cmd/gomad/...
GOWORK=off go vet -tags test_dep ./runner/...
```

### Constraints
- No `git add`, commit, stash or worktree: the user owns commits. Record `"commits": []` in the `flowctl done` evidence and say so in the summary.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- This host is `darwin/arm64`. `linux/amd64` gates cannot run here: list them as incomplete in the done summary, never claim them.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] `coordinatorConfig` and both conversions carry `MaxForcedDecisions`, `MaxExplorationResultBytes` and every `SimulationDimensionLimits` field; a whole-value round-trip test with distinct nonzero values passes.
- [ ] A guard test fails when a new exported `CampaignSpec` field is neither transported nor explicitly listed as non-transported.
- [ ] Seed, choice and simulation strategies each run through `Explore` -> child process -> `CoordinatorMain` -> `runLocal` with a real target, and the result reports the supplied distinguishable limits.
- [ ] Zero/missing bounds, incompatible strategy settings, malformed requests, trailing data and unknown fields are still rejected with their existing messages and `HostError` reasons.
- [ ] Before/after evidence shows the simulation request failed validation before the fix and reaches execution after it; CLI flags, defaults and exit statuses are unchanged.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
