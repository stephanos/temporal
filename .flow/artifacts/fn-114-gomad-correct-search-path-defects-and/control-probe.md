# E4 control probe: Runnable decisions before and after the run-queue rule

Task 13 (R9). Host: **linux/arm64 with the development harness on** (Go 1.27.1,
2026-10-03). This is development evidence only, not darwin/arm64 or
linux/amd64 evidence. The historical counts (26 for seed 11, 29 for seed 17)
were taken on darwin/arm64 (fn-105 D21).

Probe: `.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d21-probe-control.go.txt`,
unchanged, as `./cmd/control` of a scratch module. It starts no goroutine, so
main is its only user goroutine. `twouser` is a scratch copy of the
`two-users` mode of the new `runq_user_choice` fixture (two workers that yield
four times each while main waits).

Command, for each probe and seed:

```
gomad qualify --seed SEED --repeat 2 --choices --replay-successes \
  --success-limit 4 --success-bytes 64MiB --artifacts <scratch> \
  --working-dir <scratch>/probe go-run ./cmd/PROBE
```

| probe | seed | before: Runnable decisions | after: Runnable decisions | qualified, exact replay (both) |
| --- | ---: | ---: | ---: | --- |
| control | 11 | 26 | 0 | yes |
| control | 17 | 29 | 0 | yes |
| twouser | 11 | 10 | 5 | yes |
| twouser | 17 | 12 | 5 | yes |

- Before: base `bce040728` (`gomad-next`), toolchain build key
  `344ad460671b7504bb4a4c59a31f211dcc44e4461de0c0a6ef6866a8f6ff898b`,
  choice implementation `sha256:b66e9c32…56c6`. Every decision was Runnable;
  no select records. The linux/arm64 counts equal the darwin/arm64 historical
  counts.
- After: the task-13 rule, build key
  `eb30c73f29145bf1786c6af5badb04b875ef1d9ffd6d5fd683d73ce9e8d33967`,
  choice implementation `sha256:934677bc…94b4` (both harness-on values; the
  committed harness-off identities differ). The control probe's stdout is the
  same before and after for both seeds (`virtual_now_unix_nanos
  946684800000000000`, `num_gc 4`, `last_gc_is_host_time true`,
  `memstats_pause_total_ns 0`).

## Cause attribution (before)

A throwaway build of the unmodified rule printed, for every recorded Runnable
decision, each alternative's start function and `isSystemGoroutine(gp, false)`
to stderr (not committed; it did not change the counts). Decisions with a
runtime-owned alternative:

| probe | seed | decisions | with a runtime-owned alternative | user-only |
| --- | ---: | ---: | ---: | ---: |
| control | 11 | 26 | 26 | 0 |
| control | 17 | 29 | 29 | 0 |
| twouser | 11 | 10 | 6 | 4 |
| twouser | 17 | 12 | 8 | 4 |

The runtime-owned alternatives were `gcenable.gowrap1`/`gowrap2` (sweeper and
scavenger), `forcegchelper`, `updateMaxProcsGoroutine`, `runFinalizers` (the
finalizer goroutine while not running a finalizer), and
`gcBgMarkStartWorkers.gowrap1` (a mark worker on its first start). The cause
is confirmed: every control-probe decision, and every extra two-user decision,
offered a runtime-owned goroutine. Full listing: `task-13/attribution.md`.
