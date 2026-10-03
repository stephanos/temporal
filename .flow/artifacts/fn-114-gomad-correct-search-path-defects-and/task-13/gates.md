# fn-114 task 13 (E4, R9): evidence summary

Host: **linux/arm64 with the development harness on** (Go 1.27.1,
2026-10-03), base `bce040728` on `gomad-next`. Development evidence only:
nothing here is darwin/arm64 or linux/amd64 evidence. TMPDIR=/tmp.
Toolchain build keys (harness on): before
`344ad460671b7504bb4a4c59a31f211dcc44e4461de0c0a6ef6866a8f6ff898b`, after
`eb30c73f29145bf1786c6af5badb04b875ef1d9ffd6d5fd683d73ce9e8d33967`.

## Change

`gomadChoiceRunqIndex` (overlay `runtime/gomad.go`) now owns the whole
local-run-queue pick: the first runtime-owned goroutine
(`isSystemGoroutine(gp, false)`) in queue order is taken without a draw or a
record; only an all-user queue draws from the seeded run-queue stream and,
with a Choice Trace, records one Runnable decision whose alternatives are
exactly those goroutines (queue order, so the replayed index is the queue
offset). The `runqget` hunk loses the caller-side draw (one line). The rule
applies whether or not tracing is on. Rule stated in SPEC `[RUNTIME.SCHEDULING]`.

## Cause (before the edit)

Confirmed with a throwaway stderr-instrumented build; see
`../control-probe.md` and `attribution.md`. Control probe: 26/26 (seed 11)
and 29/29 (seed 17) decisions had a runtime-owned alternative; two-user
probe: 6/10 and 8/12.

## Red first (new conformance check on the unmodified toolchain)

Focused `go test -count=1 -run TestRuntimeSearchFixtures ./internal/gomadtool/conformance`
with the old overlay and patch restored (build key 344ad…), exit 1:

- full check: `main-only seed 1 recorded 27 Runnable decisions with at most one runnable user goroutine`
- no-choice modes skipped (temporary edit, reverted): `two-users decisions selected 8 distinct goroutines, more than its 3 user goroutines`
- same, busy-runtime first: `busy-runtime decisions selected 10 distinct goroutines, more than its 3 user goroutines`

Green after the edit: `search-reproduction.json` (`runqueue_user_choice`):
two-users 5 decisions per seed over 8 seeds, 2 distinct selected identities,
8 distinct step orders, seed-1 tape reproduced under seed 2; busy-runtime
37-39 decisions per seed, 2 identities, every seed `a=32 b=32` with both
workers seeing finalizers; main-only and one-user 0 decisions and equal
output for seeds 1,1,2,3,11,17. An instrumented run of the new rule
(`attribution.md`, after) shows busy-runtime taking runtime-owned
goroutines ahead of up to two user goroutines dozens of times per run while
both classes complete.

Side effect in the same file: every select-readiness shape exploration fell
from 68-200 executions (task 12, darwin/arm64) to 3 (linux/arm64), because
runtime-owned Runnable decisions no longer enter the frontier.

## Tape recorded before the change is rejected by identity

`gomad replay <before/artifacts-twouser-11 success>` with the rebuilt Runner:
`incompatible replay artifact: artifact choice profile identity does not
match this Runner`, exit 2 (artifact implementation
`sha256:b66e9c32…`, Runner implementation for the new sources
`sha256:934677bc…`, both harness-on values).

## Gates after the edit

| command | exit | elapsed | note |
| --- | ---: | --- | --- |
| `make validate` (harness on) | 0 | 0:03 | |
| `make test-toolchain` | 2 | 0:05 | only `TestPatchedRuntimeHostClockReferencesAreReviewed` (baseline, no linux/arm64 clock inventory) |
| `make overlay-test` | 0 | 0:24 | |
| `make test-runtime` | 0 | 14:33 | first attempt exit 2 after 1:30 on vanished go-build cache entries (cache cleaned concurrently); rerun passed |
| `GOFLAGS='-tags=test_dep -count=1' make test-host` | 2 | 3:36 | failures are a subset of the baseline list (19 of 21; the two process-termination tests passed this time) |
| focused `go test ./choice/... ./internal/gomadtool/generation/... ./internal/gomadtool/conformance/` | 0 | | |
| focused runner `-run ChoiceExploration ./runner/` and the five choice tests in `./runner/internal/execution/` | 0 | | |
| `make validate` (harness off, before commit) | 2 | | only `undefined: syscall.Dup2` (harness limitation) |

The first `test-host` run after the runtime edit failed six tests that the
rule changes: five `runner/internal/execution` tests whose `choice-trace` and
`choice-marker` helper targets started no goroutine and recorded only
runtime-owned decisions (now none), and the pinned two-outcome benchmark,
whose explorer now exhausts its frontier in 2 executions instead of spending
16. The helpers now run two yielding workers before exiting or printing the
marker (assertions unchanged); the benchmark asserts 2 attempts, exhaustion,
both outcomes, and strictly more outcomes per execution than seed sampling,
and is renamed `...BeatsEqualBudgetSeedSampling`.

## Owed

- `make -C tools/gomad3 validate test-toolchain test-runtime overlay-test` and
  `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` on
  darwin/arm64 and linux/amd64.
- Control-probe counts (`../control-probe.md`) on darwin/arm64.
- Core, smoke, and representative qualification under the new controller
  identity (task 14).
