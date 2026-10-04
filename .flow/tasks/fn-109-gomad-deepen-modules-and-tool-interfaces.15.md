---
satisfies: [R11]
---
# fn-109-gomad-deepen-modules-and-tool-interfaces.15 Characterize simulation progress ordering and choose the lifecycle interface from two designs

## Description
Stage 5, first half of R11 (F10). Before changing the arbiter, pin its current guarantees with tests written against behaviour, then compare at least two lifecycle interfaces against those guarantees and record the choice. This task alone leaves R11 unmet; the next task implements the chosen design.

**Size:** M
**Files:** `tools/gomad3/runner/internal/execution/simulation_time_test.go`, `simulation_unix_test.go`, `simulation_test.go`, `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md`.
**Touches:** [tools/gomad3/runner/internal/execution/*_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/simulation-progress-design.md]

### Approach
- Current protocol (`runner/internal/execution/simulation_time.go`): `beginExternal` `:312`, `beginHandledExternal` `:323`, `beginExternalAfterArrivals` `:335`, `forwardExternalAfterArrivals` `:352`, `transferExternalArrival` `:372`, `acknowledgeExternal` `:395`, `deliverExternal` `:410`, `endExternal` `:423`, `remove` `:436`, `settleLocked` `:449`. 24 call sites in `simulation_unix.go` (`:173-557`) and 3 in `process_unix.go` (`:101`, `:216`, `:220`).
- Parallel state machine: `simulationResponseBarrier` and `coordinator.responses` (`simulation_unix.go:34-49`) with `beginResponseBarrier` `:501`, `beginNodeResponseBarrier` `:151`, `retainCompletionUntilResponse` `:480`, `beginForwardedResponseBarrier` `:516`, `releaseWaitResponseBarrier` `:532`, `reserveWaitResponseBarrier` `:544`. Transport callbacks `delivered` / `arrived` / `discarded` (`simulation_model.go:30-50`).
- Characterization tests (state-machine level, no production change): forwarding, acknowledged arrivals, delivered-but-unconsumed work blocking time advance, arrival exactly at quiescence, cancellation followed by a late committed response, participant death, restart with a new incarnation, unknown acknowledgement, abandoned response, stale incarnation, and two concurrent blocking operations on one participant. Assert observable results (time advances or not, error or not, which participant is quiescent), not helper names, so the tests survive the refactor.
- Design comparison in `simulation-progress-design.md`: at minimum (a) a token/handle interface (begin returns an operation handle; terminal disposition and arrival consumption are methods) and (b) a typed transition function over private arbiter state (one `apply(event)` entry). For each: what the caller still has to know, where the response-barrier state goes, how acknowledgement-before-admission and transfer atomicity are kept, how concurrent blocking operations are represented, extra state introduced, and which of the 27 call sites collapse. State the selected design and why, plus what would make you reverse it.
- Host IPC arrival order must not become replay identity in either design.

### Investigation targets
**Required:**
- `tools/gomad3/runner/internal/execution/simulation_time.go:167-508`
- `tools/gomad3/runner/internal/execution/simulation_unix.go:34-560`
- `tools/gomad3/runner/internal/execution/simulation_model.go:20-304`
- `tools/gomad3/runner/internal/execution/process_unix.go:90-225`
- `tools/gomad3/runner/internal/execution/simulation_time_test.go` (14 existing tests)
**Optional:**
- `tools/gomad3/ARCHITECTURE.md` section "Process arbitration and model evidence"

### Quick commands
```bash
cd tools/gomad3
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep ./runner/internal/execution -run 'Simulation'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep -race ./runner/internal/execution -run 'SimulationTime|SimulationModel'
env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off .toolchain/bin/go test -count=1 -tags test_dep,integration ./runner/internal/execution -run 'TestRootProcessSimulationUsesRunnerTransport'
```

### Constraints
- The conductor commits each verified task separately under MILESTONES item 5, preserving unrelated changes. Older user-only commit instructions are superseded; unavailable native gates remain incomplete and Flow acceptance stays open. Push only when authorized.
- No new third-party dependency. `tools/gomad3/go.mod` requires only `golang.org/x/mod`, so testify is unavailable inside `tools/gomad3`: follow the existing `t.Fatalf` style with whole-value comparisons there. In the root module (`tools/gomad3sim`, `tools/gomad3integration`) use `require` with `Equal`/`EqualValues`.
- Preserve existing comments with their owning code, CLI grammar/defaults, canonical bytes for fixed supplied identities, and error precedence/classification.
- Actual host is linux/arm64 with stock Go. Native darwin/arm64 and linux/amd64 qualification remain incomplete; source checks and framed-pipe fixtures do not qualify either platform.
- fn-105 D12/D14 replay-divergence dispositions stay unchanged. Attribute a failure to those owners with retained evidence instead of relaxing an expectation.
- Run tests with `-tags test_dep`. Baseline the Quick commands before editing so a pre-existing failure is not attributed to this task.
- Evidence and decision records go under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/`.
## Acceptance
- [ ] Behavioural characterization tests cover forwarding, acknowledged arrivals, delivered-but-unconsumed work, arrival at quiescence, cancellation with a late committed response, death, restart, unknown acknowledgement, abandoned response, stale incarnation and concurrent blocking operations, and pass against the unchanged implementation.
- [ ] `simulation-progress-design.md` compares at least two concrete interfaces against those guarantees and records the selected design, the rejected one and the reasons.
- [ ] No production code changes in this task; the summary states that R11 stays open until the implementation task is done.

## Done summary
Blocked:
# Task 15 native gates remain open

The final source candidate adds eleven behavior characterizations and the
required concrete two-interface lifecycle design comparison without production
changes. Fresh conductor focused/race/100-repeat/vet checks and two independent
same-family source audits plus delta rechecks found no remaining source defects.
Exact final identities are retained in evidence.json and source-audit.md.

Acceptance remains incomplete. This host is linux/arm64 with stock Go 1.27.1;
`.toolchain/bin/go` is absent. The prescribed patched-toolchain Simulation,
SimulationTime/SimulationModel race and root process transport Quick commands
remain unrun. The retained pre-edit broad stock Simulation baseline failed
TestRunSupervisesSimulationNodeProcess and
TestRunHardCrashesAndReapsSimulationNodeProcess with child exit 49. No expectation,
selector, platform policy or D12/D14 disposition was weakened.

Native darwin/arm64 and linux/amd64 process/runtime qualification remain open.
Stock framed-pipe/World fixture checks do not prove native timing, process hard
isolation, backend interception or exact replay. R11 also requires task 16's
implementation and conformance; a design cannot close it.

MILESTONES permits subsequent source work after the integrated predecessor
candidate is reviewed while acceptance stays open. Task 16 may therefore start
source implementation against this candidate without marking task 15 done.
User owns commits; commits remain empty.

Blocked:
# Task 15 acceptance remains open after characterization checkpoint

The exact two retained characterization test blobs and unchanged two-interface
design are verified for a separate task progress commit. See conductor-checkpoint.md,
checkpoint-report.md and the original source-audit.md. Production is unchanged.

All 11 characterizations, the focused state-machine/transport selection and
inherited architecture checks passed on stock linux/arm64. This does not supply
the prescribed patched-toolchain Simulation, SimulationTime/SimulationModel race
or root process transport Quick commands. Native darwin/arm64 and linux/amd64
process/runtime gates remain incomplete; no assertion or disposition was relaxed.

R11 also requires task 16's implementation and conformance. Keep task 15
acceptance open until required gates pass. The conductor commits verified progress
under MILESTONES item 5, superseding older user-only commit instructions, without
overwriting successor working-tree source or pushing.
## Evidence
- Commits:
- Tests:
- PRs:
