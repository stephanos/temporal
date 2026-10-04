# Task 15 exact source checkpoint

The conductor can review the two retained task-15 test blobs in
`/home/agent/.cache/codex-build/tmp/fn109-task15-checkpoint.10zps06w`.
All 11 characterizations, the 47-test focused time/model/coordinator/ServeSimulation
selection and the nine inherited architecture checks passed against committed
pre-task16 production. Task 15 acceptance and R11 remain open.

The preparation archived only `tools/gomad3` from
`5350185a3601921c0a5f9ba07e1f05bdad7df81f`. The archive contained 874 regular
files. The helper validated every archive path and rejected links, special
entries, absolute paths and traversal before extraction. The archive SHA-256 is
`a646f13248e9a6d2530f738e70b597c4c4ea52e8517217691611ee56106239ea`.
The complete before/after inventory in `checkpoint-verification.json` proves
that these are the only two changed module paths.

| Exact module path | Retained SHA-256 |
| --- | --- |
| `tools/gomad3/runner/internal/execution/simulation_progress_test.go` | `4d91f01eb5e378e5aa4824c2af655862d9fe0fe57772bd74f2dfa648418c0880` |
| `tools/gomad3/runner/internal/execution/simulation_progress_fixture_test.go` | `35ed448549b3aa5d6ce959d86a631b37979056642144e31274e18bcbfb0e8e5c` |

The first blob comes from task-16's retained
`simulation_progress_test.before.txt`. The fixture comes from
`/tmp/fn109-task15-preimage.B9MU5kXH`. An independent reconstruction reproduced
the fixture byte-for-byte by reversing exactly two task-16 callback substitutions.
`checkpoint-fixture-diff.log` retains the complete delta. Dispatch uses
`deliverExternal` and abandoned arrival uses `acknowledgeExternal` in the
task-15 fixture. No assertion, cleanup, framing or helper changed.

The four investigated production files match task-15's retained production
hashes. `simulation_time.go` retains the accounting implementation and
`simulation_unix.go` retains its response barriers. Neither
`simulation_progress.go` nor the later private architecture checker is present
in scratch. The stock architecture gate ran the nine existing HEAD tests.
It supplies no claim about task-19's new purity/effect enforcement or the
working-tree pure-import initializer reproducer.

The design remains byte-identical at
`1f2fc94d417ad3ffe2d64a2b255787d3ad74e13701bc85a7294522a0629a60c9`.
It compares concrete operation handles and closed typed semantic transitions
against caller responsibilities, response-phase ownership, acknowledgement
validation, atomic transfer, simultaneous operations, extra state and all 25
measured accounting sites. It selects typed complete protocol transitions under
one arbiter mutex because aggregate wire credits cannot identify individual
handle consumption. Its reversal condition requires operation-specific wire
acknowledgements or evidence that complete transitions still require caller
phase/counter sequencing. Both alternatives exclude host IPC arrival order from
semantic replay identity.

The tests cover arrivals at installed quiescence, both reply orders for two
same-participant operations, partial consumption, unknown admission/forward/
transfer/discard acknowledgements, death/restart/stale frames, committed World
mutation followed by cancellation and a late discard, and unknown/duplicate/
abandoned duplicate response rejection before callbacks. Assertions observe
whole time responses, quiescence, error classification, records, callback
selection and World state. Installed-waiter and bounded pipe helpers preserve
their original cancellation, pipe-close and goroutine-join behavior.

The two `Preexisting` characterizations deliberately continue to pass on old
production. Malformed wait acceptance wakes a waiter before rejecting its
unknown acknowledgement. Duplicate admission consumes arrival credit before
rejecting duplication. Task 16 must strengthen those pins and supply old-source
RED evidence for validation before mutation.

All commands used stock Go 1.27.1 at
`/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`,
with `GOWORK=off`, `GOTOOLCHAIN=local`, `GOMAXPROCS=2`, seed variables unset
and `-tags test_dep`. List commands preceded execution, and execution used
`-count=1`, verbose output and explicit Go test timeouts. Exact argv,
environment, exit codes and log hashes are in `checkpoint-verification.json`.
`checkpoint-characterization-list.log` proves all 11 retained tests were selected.
`checkpoint-focused-list.log` proves the focused selection contains 47 tests.
`checkpoint-architecture-list.log` proves the inherited selection contains nine.
`gofmt -l` returned no paths. All 876 scratch module file hashes remained
unchanged after the checks.

The exact blobs retain earlier focused race, 100-repeat and vet evidence in the
original handover. This checkpoint introduced no source change or concrete race
concern, so it did not repeat those checks. The known pre-edit broad stock
Simulation child-exit-49 failures remain historical evidence and were not
retried. All three patched-toolchain Quick commands, including the root process
transport test, remain unavailable here. Native darwin/arm64 and linux/amd64
process/runtime qualification remain incomplete. Stock framed pipes and World
fixtures establish no native timing, backend interception, hard isolation or
exact replay qualification. R11 also requires task 16 implementation and
conformance. D12/D14 dispositions remain unchanged.

The preparation made no shared source/test/design, Git/index or Flow mutation.
It wrote only new task-15 checkpoint artifacts and isolated scratch. No commit,
push, stash, worktree or new agent ran. The prior assignment judgment remains
`no_key` with explicit `gpt-6.1-sol` at high, same-family routing and unknown
actual backend identity. The preparation did not rejudge. This source checkpoint
provides no formal SHIP verdict or task completion.

The preparation helper is `checkpoint-prepare.py`, SHA-256
`d6c289394108bd965c4471bb3e126459c22569786f2e09ef45fc916401b49659`.
The source manifest is `checkpoint-verification.json`, SHA-256
`51375fda20e8c2e4357a4a3467b0d70cbf785d34a84e4b7c19369a6c2baf885f`.
