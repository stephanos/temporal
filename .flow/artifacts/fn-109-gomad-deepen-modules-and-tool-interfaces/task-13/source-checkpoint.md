# Task 13 source checkpoint

The task-13-only source boundary is reconstructed and verified, independently
audited for ownership, and checked in an isolated module. Native acceptance
remains incomplete. This is verified progress under the user's current commit
policy, not a completion or formal SHIP receipt.

## Exact source separation

The source is `/tmp/fn109-task13-checkpoint.FAl6gGl0/tools/gomad3`: base
`0dd05b313acd0986312da7fd3159520e6a21f1bf` plus task 13's 22 retained final
source entries, all matching the full SHA-256 values in `evidence.json`.
Twenty entries still match current source. The two shared files were recovered
with the existing exact-hunk helper, not copied from later task 16/18 source:

- `simulation_time.go`: only the three historical codec-removal hunks.
- `version.json`: only the generated time-wire allowlist hunk; no domain additions.

`reconstruct-checkpoint.py` and `checkpoint-reconstruction.json` retain the
base, historical diff/helper hashes, recovery origins and all 867 module files.
The first attempt stopped before scratch creation because the module-only
parser received Flow diff sections. Restricting its input to the two requested
module sections corrected that call boundary; all original exact checks remain.

The independent `task-checkpoint-boundaries.md` audit rehashed all 22 task
entries and all 867 scratch files, and compared the whole listed source set
to the base. No changed/new source path is outside the task-13 map. The four
identity mirrors are identical in later task manifests, not later deltas.
The scratch architecture file remains base bytes; the active task-19 writer
and all later working-tree source are preserved. The historical source audits
in `source-audit.md` cover these same task-13 identities.

Staging used the verified scratch blobs directly at their Git index paths,
with mode 100644. All 22 staged blob hashes were checked against the retained
task manifest; later worktree versions were not overwritten. A transient index
lock interrupted the first per-file attempt. No visible process owned it and
its bytes exactly matched the live index, SHA-256
`0f313da12b79608949b64a10929af2918b330e6c22f0ace6ce5c2781d69d9565`.
It was moved, not deleted, to the recoverable local backup
`.git/index.lock.task13-checkpoint-20261004T0613`. One batched index update then
succeeded. Raw compiler/test logs retain their original bytes and are excluded
from formatting checks; hand-written source and reports retain normal checks.

## Fresh developmental checks

All commands used pinned stock Go 1.27.1 on linux/arm64, GOWORK=off,
GOTOOLCHAIN=local, GOMAXPROCS=2 and unset GOMADSEED/GOMAD3_CHILD_SEED.
The Go executable was
`/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`.
Commands ran from the isolated module, not the later working tree:

| Command after environment prefix | Exit | Result |
| --- | ---: | --- |
| `go test -count=1 -tags test_dep ./internal/gomadtool/generation/protocol ./internal/gomadtool/conformance ./runner/internal/execution -run 'SimulationTime\|Protocol\|RuntimeCampaignSimulationTimeVectors'` | 0 | Packages passed in 0.001, 0.001 and 0.002 seconds |
| `go run ./cmd/gomadtool version-generate -check` | 0 | No output; generated identities current |
| `go run ./cmd/gomadtool protocol-generate -check` | 0 | No output; protocol outputs current |
| `go test -count=1 -tags test_dep . -run '^TestPackageArchitecture$'` | 0 | Root package passed in 0.085 seconds |

The generator checks and architecture check were one serial fail-fast batch.
Earlier exact-source generation, validation, vet, runtime-vector stand-in and
nosplit evidence remains in `evidence.json` and its retained logs. None of these
stock-host checks qualifies supported-platform patched execution.

## Inherited pin gap and native acceptance

Task 19's broader diagnostic found an inherited first-party simulation bridge
pin mismatch. At the task-13 base and current HEAD,
`tools/gomad3sim/runtime_time_toolchain.go` hashes to
`211c01f57125ba62115b1ffce5d2479d3c22116d51a41aefcfb1a576e8b393a9`.
Base policy still pins `e6402e8fbfc848c7360d19a1b77de93e841d64870ab625433fac8a47de83d23d`
and omits the existing Current directive. Git history identifies the integrated
`ad90b462e0` source change; its parent source matches that old pin exactly.
Task 13 changes neither the root bridge nor its capability-policy owner.
This non-native gap needs its own owning-task repair; do not waive it or
attribute it to time-wire extraction.

Required supported-platform rebuild, patched runtime vectors, real Runner
transport, gomad3sim execution and full quiescence/nosplit checks remain open
on both darwin/arm64 and linux/amd64. See `evidence.json`'s `native_commands`
and the task's Quick section for exact commands.
The source checkpoint may be committed, but task 13 remains blocked and
R7 acceptance remains incomplete. The user now authorizes per-task commits;
older handovers' user-only commit wording is historical. Pushes remain unauthorized.
