# fn-114 task 11: gate evidence (darwin/arm64, 2026-10-02)

Host: darwin/arm64, stock go1.27.1 first on PATH
(`~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.darwin-arm64/bin`).
Base commit `d56cc809ae138c57b01b29fdd81c788fdcb1ba7b`. Another session loaded
the host during the gates (load average 5 to 19). linux/amd64 was not run: no
native host in this session.

## Toolchain identity

| | build key |
| --- | --- |
| before (base commit, task 5) | `245141dc288328269b6fc164034c6d74b87c525e5d3c4756d3b82a3fcde40d52` |
| after (this task) | `2008ea81459afbd1ee019beb4a231968c1af8a24f4b46d4a8426b3188b1a9b87` |

The choice wire schema rose to v3 (`gomad3-choice-trace/v3`, magics `\x03`), and
the choice implementation identity changed with the overlay and patch edits.
`protocol-generate` refreshed `choice/internal/wire/wire_generated*.go`, the
overlay `gomadchoicewire` mirrors, `gomad_choicewire_generated.go`, and the
live-capability identities in `target/internal/livecap/protocol_generated.go`
and overlay `gomadcap/protocol_generated.go`. Retained artifacts are not
rewritten; a v2 trace is refused by profile name, a v2 or v4 header, terminal
frame, or tape by version.

## Baseline

`baseline: green via handoff`: the host tier was verified at 25330890 by
fn-112.15 and the runtime tier at b98c5018 by fn-114.5 on build key 245141dc;
only `.flow` commits landed since, so the focused Quick commands were not rerun
before the edit.

## Red first

See `red-first.txt`: the choice fixtures that build a select result without
readiness fail on the v3 rule before they gain one; the runner golden fails on
the v2 profile before its refresh; and the fixture's readiness assertion fails
when `blocking-two-ready` is told to expect one ready case (mutation, reverted).

## After the edit

| command | exit | elapsed |
| --- | ---: | --- |
| `make -C tools/gomad3 toolchain` (final rebuild) | 0 | 3:01 |
| `go test -tags test_dep -count=1 -run TestRuntimeSearchFixtures ./internal/gomadtool/conformance` (retained to `search-reproduction.json`) | 0 | 1:05 |
| `go test -tags test_dep -count=1 ./choice/... ./record/... ./artifact/... ./internal/gomadtool/... ./cmd/gomadtool ./qualification/... ./target/internal/livecap/...` | 0 | 2:02 |
| `go test -tags test_dep -count=1 -run TestDiagnosticsOffPreservesExistingCanonicalIdentities ./runner` (after the golden refresh) | 0 | 0:06 |
| `make -C tools/gomad3 validate test-toolchain overlay-test` | 0 | 2:21 |
| `make -C tools/gomad3 test-runtime` | 0 | 16:13 (load average 13 to 20) |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` (first run) | 2 | 10:12 |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` (second run) | 2 | 3:39 |
| `.toolchain/bin/go test -tags test_dep -count=1 ./deterministicio/... ./internal/gomadtool/conformance` with the recipe's `GOMAD3_STOCK_GO` (focused rerun of the second run's failures) | 0 | 0:46 |
| `GOFLAGS='-tags=test_dep -count=1' make -C tools/gomad3 test-host` (third run, the gate) | 0 | 3:16 |

The first host run found three faults of this task, each fixed before the
second run: `choice` exported `Readiness` as a forwarding alias, which the root
architecture test forbids (now a distinct type over the wire word); the runner
inspect fixture built a select result without readiness, and its fake executor's
`t.Fatal` off the test goroutine hung the `runner` package to the 10-minute
timeout (the shared record normalizer now gives such fixtures a known
readiness); and the retained root segment under
`runner/internal/campaign/testdata/pre-start-ordinal-journal` carried a v2 tape,
so it was re-encoded through `CommitRound` (trace digest, segment identity, and
after-state identity changed through the children's prefix bytes; decisions and
candidates unchanged) and the runner characterization test's pinned error text
moved from `v2 evidence` to `v3 evidence`. The second run failed only in
`deterministicio` (three adapter consumer builds) and the conformance channel
fixtures on vanished host go-build cache entries (`cannot open file
~/Library/Caches/go-build/...`, `package ... is not in std`), the environment
fault task 5 also recorded; the focused rerun and the third full run passed.
| `gofmt -l` on every changed Go file; `go vet -tags test_dep` on `./choice/... ./internal/gomadtool/conformance ./record/... ./artifact/... ./qualification/... ./runner` | 0 | |

## What `search-reproduction.json` shows (build key `2008ea81`)

- The seven task-2 select shapes exhaust their frontiers after
  196/68/68/68/196/68/68 executions, as in task 2 and task 5. Every execution's
  projected plan carries the shape's readiness on its select-poll decision and
  none on any other decision:
  zero-ready `{ready 0}`, one-ready `{ready 1}`, two-ready `{ready 2}`,
  default `{ready 0, default}`, timer `{ready 0, timer}`, closed `{ready 1,
  closed}`, nil `{ready 1, nil}`.
- Two new shapes: `timer-channel-due` (the clock passed the timer's deadline
  while nothing waited on its channel, so the select's poll loop ran the timer
  and its send counts: `{ready 1, timer}`, 200 executions) and
  `repeated-channel` (one channel in two cases: `{ready 2, repeated}`, 68
  executions, outcomes `first` and `again`).
- Recording cost, from the first execution's diagnostic trace between the
  select's last poll decision and its result: 0 allocations and 0 seeded draws
  for every shape that takes a case in its first locked pass (one-ready,
  two-ready, default, closed, nil, timer-due, repeated). The two shapes that
  park in between (zero-ready, timer) show 2 allocations and 2 draws, which is
  the sudog acquisition and the scheduler, not the recording; the fixture
  reports them and asserts nothing about them.
- Timer sections: the 32 seeds' transcripts, the 24 creator and 8 reset
  transcripts, the cross-seed result, the set of prefix outcomes, and the prefix
  decision ordinals are identical to task 5's evidence on build 245141dc. The
  identities, alternative-set digests, callback site offsets, and prefix
  identities differ, as they do after every runtime text change, and the rank
  numbers of the 16 prefixes renumber with them because a rank is a position in
  the identity-sorted alternative set.

## Golden refresh

`runner/testdata/diagnostic-identity-choices.json` changed only in the choice
profile name and trace schema (`v2` to `v3`, in the artifact, campaign plan,
portable plan, and environment), `implementation_sha256` (three places), and
the digests derived from them: `tape_sha256`, `failure_signature`,
`record_hash`, `portable_plan_sha256`. The `plain` golden is unchanged.
