# Task 15 source handover

The source candidate adds 11 behavioral characterization tests covering the
simulation progress cases in task 15 and chooses closed typed semantic
transitions in [simulation-progress-design.md](../simulation-progress-design.md).
Production code is unchanged. R11 stays open until task 16 implements the
lifecycle owner and satisfies its process conformance requirements.

The valid-path tests in `simulation_progress_test.go` contain no removable
accounting helper calls or response-map access. Migration-sensitive transport
callback wiring lives in `simulation_progress_fixture_test.go`; task 16 may
rewire that fixture while preserving valid-path test bodies. The two explicitly
named `Preexisting` negative tests pin confirmed old defects. Task 16 must
strengthen those two tests to require rejection before waiter wake or arrival
credit consumption, demonstrate their red result against the old source and
record the correction. The conductor owns the acceptance wording reconciliation.

The conductor reconciled task 16's Approach and Acceptance to preserve valid
behavior assertions, allow fixture rewiring and strengthen only the two named
historical defect pins. Native and process requirements remain unchanged.

All source, tests and design are frozen at the hashes in evidence.json. No live
commands remain. The user owns commits, so `commits` is empty and HEAD remains
`0dd05b313acd0986312da7fd3159520e6a21f1bf`. The worker made no Flow lifecycle or
review mutation. Conductor source audits and bounded-delta rechecks found no
remaining source defects; the task is blocked on native acceptance, not done.
Pre-existing dirty task 13/14 sources and unrelated changes remain.

## Bounded fixture follow-up

Two independent audits of the prior frozen candidate reported no Critical or
Important source defects. Their record is [source-audit.md](source-audit.md),
and the prior candidate hashes remain in evidence.json. The conductor selected
one minor improvement before task 16. Seven direct frame I/O calls now use
bounded helpers in `simulation_progress_fixture_test.go`. Each helper runs its
I/O in a goroutine, sends to a buffered result channel, closes the pipe endpoint
on its five-second timeout and joins the goroutine before returning either a
result or a timeout error. Behavioral assertions remain unchanged.

The final candidate passes these checks, run serially from `tools/gomad3`.

| Exact command | Exit | Log |
| --- | --- | --- |
| `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep ./runner/internal/execution -run 'SimulationTime\|SimulationModel\|SimulationCoordinator\|ServeSimulation'` | 0 | `bounded-focused.log` |
| `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep -race ./runner/internal/execution -run 'SimulationTime\|SimulationModel'` | 0 | `bounded-race.log` |
| `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=100 -tags test_dep ./runner/internal/execution -run 'Simulation(Time\|Model)Progress'` | 0 | `bounded-repeat.log` |
| `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go vet -tags test_dep ./runner/internal/execution` | 0 | `bounded-vet.log` |

`gofmt -l` remains empty and `git diff --check` passes. Production hashes still
match the pre-edit values. Source and tests are frozen at the updated final
hashes in evidence.json, with no live commands. The conductor's separate
conductor-verification.md remains unchanged.

## Prior candidate verification

All commands ran from `tools/gomad3` with stock module-selected Go 1.27.1 on
linux/arm64. Suite exit codes came from one captured execution per observation.

| Observation | Exact command | Exit | Log |
| --- | --- | --- | --- |
| Pre-edit broad baseline | `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep ./runner/internal/execution -run 'Simulation'` | 1 | `baseline-unit.log` |
| Pre-edit focused baseline | `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep ./runner/internal/execution -run 'SimulationTime\|SimulationModel\|SimulationCoordinator\|ServeSimulation'` | 0 | `baseline-focused.log` |
| Pre-edit race baseline | `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep -race ./runner/internal/execution -run 'SimulationTime\|SimulationModel'` | 0 | `baseline-race.log` |
| Final focused suite | `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep ./runner/internal/execution -run 'SimulationTime\|SimulationModel\|SimulationCoordinator\|ServeSimulation'` | 0 | `final-focused.log` |
| Final race suite | `env -u GOMADSEED -u GOMAD3_CHILD_SEED GOWORK=off timeout 600 go test -count=1 -tags test_dep -race ./runner/internal/execution -run 'SimulationTime\|SimulationModel'` | 0 | `final-race.log` |

The broad baseline was red before any test edit. Its two failures were
`TestRunSupervisesSimulationNodeProcess` and
`TestRunHardCrashesAndReapsSimulationNodeProcess`, both with child exit 49.
The source-only scope continued under the conductor's instruction using the
focused state-machine and transport suites. Neither failure was suppressed or
claimed green, and the unchanged broad environment failure was not retried.

The prior candidate test listing selected all 11 new top-level tests. `gofmt -l` returned
no paths, `git diff --check` passed, and the four investigated production-file
hashes equal their pre-edit values. Tests and design add no package boundary,
protocol or generator changes, so architecture and generation gates were not
repeated from the reviewed task 14 candidate. The captured first/second focused
runs are intermediate characterization checks, not red/green production fixes.

The three canonical patched-toolchain Quick commands remain unavailable because
`.toolchain/bin/go` is absent. Required darwin/arm64 and linux/amd64 native checks
remain incomplete, including `TestRootProcessSimulationUsesRunnerTransport`.
The source scout and MILESTONES retain the existing model-delay watchdog finding;
no process skip, platform policy or D12/D14 disposition changed. The World commit
fixture observes a real handler mutation and pipe protocol but supplies no
cross-process or native timer proof.

Tier: session(jev-unavailable(no_key)); explicit pin wins.
stage: source-audit - ran (two fresh same-family audits plus bounded-delta rechecks; source only)
stage: impl-review - skipped(error: prescribed patched/native process gates unavailable; no green acceptance tree)
