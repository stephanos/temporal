---
satisfies: [R11]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.16 Implement the simulation progress lifecycle owner and remove caller-side accounting

## Description
Stage 5, second half of R11. Implement the design selected in `simulation-progress-design.md` so one private module owns admission, forwarding, delivered-but-unconsumed work, abandonment and participant removal. Callers stop sequencing several accounting calls and the parallel response-barrier state machine disappears.

**Size:** M (if the migration exceeds one session, migrate the coordinator-owned paths first and report the remaining call sites; do not leave two accounting schemes active for the same operation)
**Files:** `tools/gomad3/runner/internal/execution/{simulation_time.go,simulation_unix.go,simulation_model.go,process_unix.go}`, a new private lifecycle file, tests.
**Touches:** [tools/gomad3/runner/internal/execution/**]

### Approach
- Re-read the task 15 design record and characterization tests first; they are the behaviour pin and must pass unchanged.
- Replace the eight external-work methods (`simulation_time.go:299-434`) with the selected lifecycle interface. Callers to migrate: 24 sites in `simulation_unix.go` (`:173-557`) and 3 in `process_unix.go` (`:101`, `:216`, `:220`), plus the `delivered` / `arrived` / `discarded` callbacks wired into `newSimulationModelTransport` (`simulation_model.go:45`).
- Remove `simulationResponseBarrier`, `coordinator.responses` and their helpers (`simulation_unix.go:34-49,151-200,480-560`): the lifecycle owner holds that state. A wrapper that forwards to the old methods one-to-one does not meet R11.
- Unchanged owners: blocking transport and cancellation (`simulationModelTransport`), process lifetime (`process_unix.go`, `supervisor_unix.go`), domain mutation (the model handlers), native timers (runtime). Time-wire encoding from the generated codec stays as delivered by the simulation-time task.
- Invariants to keep: acknowledgement before admission, atomic transfer between participants, rejection of unknown acknowledgement, abandoned response and stale incarnation before any progress or model mutation, committed-versus-uncommitted classification on death, and more than one concurrent blocking operation per participant.
- Process conformance is required in addition to state-machine tests: detached model agreement cannot prove cross-process timing.

### Investigation targets
**Required:**
- `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`
- `tools/gomad3/runner/internal/execution/simulation_time.go:167-508`
- `tools/gomad3/runner/internal/execution/simulation_unix.go` (655 lines)
- `tools/gomad3/runner/internal/execution/process_unix.go:90-225`
- `tools/gomad3/runner/internal/execution/simulation_model.go`

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution -run 'Simulation'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep -race ./runner/internal/execution -run 'Simulation'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep,integration ./runner/internal/execution -run 'TestRootProcessSimulationUsesRunnerTransport'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/...
cd ../.. && tools/gomad3/.toolchain/bin/go test -count=1 -tags test_dep,gomad3_toolchain ./tools/gomad3sim
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
- [ ] One private lifecycle owner handles admission, forwarding, delivered-but-unconsumed work, abandonment and participant removal; callers no longer sequence multiple accounting updates.
- [ ] The response-barrier map and its helpers are removed, with no second bookkeeping structure replacing them outside the lifecycle owner.
- [ ] The task 15 characterization tests pass unchanged, including concurrent blocking operations and cancellation with a late committed response.
- [ ] Unknown acknowledgement, abandoned response and stale incarnation fail before invalid progress or model mutation.
- [ ] Process simulation conformance (`TestRootProcessSimulationUsesRunnerTransport` and the gomad3sim toolchain tests) passes on darwin/arm64 with `-race` clean for the execution package; linux/amd64 is recorded as incomplete.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
