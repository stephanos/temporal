# Gomad D17: the Nexus operation test that needs two clusters

Assessment date: 2026-10-01.

`TestNexusOTELSuite/TestOperation` has never run with two clusters under Gomad. The test needs two
dedicated cluster slots at once. The dedicated pool has one slot, because Gomad pins `GOMAXPROCS`
to 1. The setting meant to raise it, `TEMPORAL_TEST_DEDICATED_CLUSTERS=2`, does not reach the test:
the Runner records a supplied `--env` entry and hands it to the process, and the patched runtime
then replaces the environment that Go code can read with `TZ=UTC` alone. The test takes the only
slot and waits for a second one on a channel receive that has no deadline. The cluster it already
holds keeps firing periodic timers, so virtual time runs on for hours and the target never exits.
The wall watchdog kills it after two minutes.

Everything recorded as nondeterminism follows from that kill. Two repetitions are stopped at
different virtual instants, the elapsed virtual time is the first evidence field that differs, and
the set report says `nondeterministic`. A killed target writes no terminal choice frame, so its artifact keeps no
Choice Trace and is replayed without tracing. A traced recording and an untraced replay differ in
one heap address that the server prints in a log line, which is the seed-11 replay's stderr
difference. No diverging runtime choice was observed. The killed runs keep no Choice Trace, so
this rests on their outputs and on the controls, not on a trace comparison. No evidence of a cause
shared with D12 or D14 was found, and the D14 fix does not change the outcome.

Two clusters in one process are deterministic in the evidence gathered here. With the pool forced
to two slots in a throwaway copy of the source tree, the suite with `TestOperation` enabled is
`qualified` on seeds 11 and 17 with exact choice-tape replay, in 12 executions, and passes on seeds
1 to 17.

Nothing in this report is implemented or fixed. The skip stays. The recommended correction, owned
by the Gomad runtime patch and Runner, is to deliver supplied environment entries to the target.
It is open work in fn-105 until a decision selects it and its verification passes.

This is the receipt for
[D17](../../../.flow/tasks/fn-105-gomad-follow-ups-deferred-scope.17.md) (fn-105 R17).

## Scope and identities

| Item | Value |
| --- | --- |
| Repository revision | `7ef3800a2c` plus the shared uncommitted working tree for the first session of runs, then `6782b55f49`, a commit of that working tree, for the second |
| Test source | `tests/nexus_otel_test.go`, SHA-256 `ece9451a...07cd9f`; `tests/testcore/test_cluster_pool.go`, SHA-256 `feca6003...fd6d5` |
| Toolchain | go1.27.1, build key `8d28bd4486f0b6300e8d25efd4caf8cb6ccbf000e96dbd26b1d8f53bf5f251bc`, which contains the D14 fix |
| Runner | `tools/gomad3/.bin/gomad`, SHA-256 `4dce3aed...db5222` in the first session and `0e00a1be...e42a2f` in the second |
| Native Go | go1.27.0 darwin/arm64 |
| Host | darwin/arm64, macOS 26.6.2 (25G83), 8 cores. Another project's Go tests ran throughout; the 1-minute load average was between 4 and 21 when runs ended. |
| linux/amd64 | Not available. No linux run exists for this report. |

The runtime patch, the runtime overlay, both qualification manifests, and `go.mod` have the same
SHA-256 in both sessions. A leaf run repeated on the second checkout reproduced the first
session's stdout and stderr digests on both seeds.

Source references are `file:line` at `6782b55f49`. Evidence files are under
`.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/` with the prefix `fn105-d17-`;
[`fn105-d17-evidence.json`](../../../.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d17-evidence.json)
indexes them and records the commands. No tracked product, test, runtime, or manifest file was
edited. The skip was lifted only through manifest copies in a scratch directory, and the two-slot
variant ran in a throwaway copy of the source tree that was deleted afterwards.

## Reproduction

Suite runs use the pattern `^TestNexusOTELSuite$` and leaf runs add `/^TestOperation$`. All run
with `-test.parallel=8`, the manifest's build tags, closure mode, and the schema mount. "env=2"
means `--env=TEMPORAL_TEST_DEDICATED_CLUSTERS=2`, or the `environment` field of a `qualify-set`
workload, which becomes that flag (`tools/gomad3/qualification/set/set.go:804`).

| Run | Pool | Tick | Seeds | Outcome |
| --- | --- | --- | --- | --- |
| `qualify-set`, skip lifted, env=2, traced, repeat 2 | one slot | forward | 11, 17 | All four executions end in `watchdog_timeout` after 118 s wall. Both seeds `nondeterministic`, first divergence `virtual_time_elapsed_nanos`. Seed 11 replay diverges in `stderr.full_sha256`; seed 17 replay matches. |
| Leaf, env=2 | one slot | forward | 11, 17 | `watchdog_timeout` on both. The seed-11 log reaches virtual 13:27:30. |
| Leaf, env=2, `-test.timeout=2m`, three runs | one slot | forward | 11, 17 | Target fails at virtual 2 m in 16 to 31 s wall. The test waits at `test_cluster_pool.go:109`. Byte-identical stdout and stderr across the three runs of each seed. |
| Leaf, no env, `-test.timeout=2m`, two runs | one slot | forward | 11, 17 | The same wait. Byte-identical across the two runs of each seed. |
| Suite, env=2, `-test.timeout=2m` | one slot | forward | 11, 17 | Seed 11: three tests wait for a slot after `TestCallback` finished. Seed 17: all four. |
| Suite, env=2, traced, then `gomad replay` twice | one slot | forward | 11 | `watchdog_timeout`. Both replays: `reproduced=false divergence=stderr.full_sha256 choice-replay=none`, each after 118 to 119 s wall. |
| Leaf, pool forced to two slots | two slots | forward | 11, 17 | Pass at 5.10 s virtual. Two clusters created. |
| Suite, pool forced to two slots, verbose | two slots | forward | 1 to 17 | 17 of 17 pass, four subtests each |
| `qualify-set`, skip lifted, pool forced to two slots, traced, repeat 2 | two slots | forward | 11, 17 | `qualified`, exact choice-tape replay |
| The same at repeat 4 | two slots | forward | 11, 17 | `qualified`, eight executions, exact choice-tape replay |
| The same at repeat 2 | two slots | strict | 11, 17 | `target_failure`, deterministic, exact replay of the failure. `TestWorkerOperation` and `TestOperation` fail the span comparison. |
| `qualify-set`, `TestOperation` skipped, traced, repeat 2 | one slot | forward | 11, 17 | `qualified`, exact choice-tape replay |
| Native suite, `-count=3` | 8 slots | wall clock | none | 12 of 12 subtest runs pass |
| Native suite, `-count=3`, `TEMPORAL_TEST_DEDICATED_CLUSTERS=2` | two slots | wall clock | none | 12 of 12 subtest runs pass |
| Native leaf, `TEMPORAL_TEST_DEDICATED_CLUSTERS=1`, `-timeout 60s` | one slot | wall clock | none | Times out with the test at `test_cluster_pool.go:109` |

`fn105-d17-qualify-set-runs.txt` holds the per-execution digests, wall times, and replay results of
the five `qualify-set` runs. The first row reproduces the recorded classification on the toolchain
that contains the D14 fix: seed 17 nondeterministic, a seed-11 replay that differs in stderr, and
no terminal frame.

## The environment entry does not reach the test

A probe program reads `TEMPORAL_TEST_DEDICATED_CLUSTERS` in a package `init` and in `main` and
prints `os.Environ()` (`fn105-d17-envprobe-main.go.txt`). With zero, one, or two supplied entries
its output under Gomad is the same (`fn105-d17-envprobe-results.txt`):

```
init="" main="" ok=false gomaxprocs=1 environ=1
environ=["TZ=UTC"] args=["gomad3-target"]
```

The artifact manifest of the same execution records the supplied entries next to `GOMADSEED` and
`TZ`. The path through the code is as follows.

1. The CLI accepts `--env` as "target NAME=VALUE"
   (`tools/gomad3/cmd/gomad/internal/cli/cli.go:456`, `qualify.go:72`). The Runner validates each
   entry and rejects reserved names (`tools/gomad3/runner/runner.go:1425-1448`).
2. The Runner adds the seed and `TZ=UTC`, records the list, and passes it to the target process
   (`runner.go:1622-1638`, `runner.go:1497`). The bootstrap calls `syscall.Exec` with it
   (`tools/gomad3/runner/internal/execution/bootstrap_unix.go:130`).
3. The runtime copies the process environment and drops only the control variables with the
   prefixes `GOMAD3_` and `GOMADSEED=`
   (`tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:857-890`). A supplied entry
   survives this step.
4. The patched `syscall.copyenv` discards that list in deterministic mode and substitutes
   `[]string{"TZ=UTC"}` (`tools/gomad3/toolchain/runtime/go1.27.1.patch:892-896`, in the
   `src/syscall/env_unix.go` hunk that starts at line 873). `os.Getenv`, `os.LookupEnv`, and
   `os.Environ` read through `copyenv`, so Go code sees `TZ=UTC` only.

Step 4 is deliberate in the runtime. A conformance fixture asserts that `os.Environ()` equals
`["TZ=UTC"]` and calls the scrubbed environment "the activation signal"
(`tools/gomad3/internal/gomadtool/conformance/testdata/gotest/gotest_test.go:13-18`).

The documentation states the opposite for supplied entries. The tutorial says the target gets
"only values explicitly supplied with `--env`" (`tools/gomad3/TUTORIAL.md:221-222`). The
architecture says the Runner adds "explicitly supplied validated entries"
(`tools/gomad3/ARCHITECTURE.md:300-301`). The generator documents its `environment` field as
"target variables for the test, such as a pool size the test harness reads"
(`tools/gomad3/qualification/set/manifestgen/manifestgen.go:99-101`). No test was found that
supplies an entry and reads it inside a target.

The evidence therefore records an environment the target's code never observed. A supplied entry
is part of Campaign, Artifact, and evidence identity while having no semantic effect. It is not
inert either: the runtime copies its bytes onto the heap in step 3, and on seed 11 the leaf run's
output differs between env=2 and no entry in which goroutines are runnable at the timeout instant
(`fn105-d17-self-wait-goroutine.txt`). On seed 17 the two differ only in hex addresses.

The earlier statement that two pool slots resolve the initial wait is not supported. Every run on
the tracked tree in this investigation, with or without the entry, ends in the same wait.

## The wait

`TestOperation` creates two environments with a span exporter
(`tests/nexus_otel_test.go:142-143`). A span exporter requires a fresh dedicated cluster
(`tests/testcore/test_env.go:122-127`), and each one reserves a slot of the dedicated pool
(`tests/testcore/test_cluster_pool.go:342`). The pool size is `runtime.GOMAXPROCS(0)` unless the
environment variable overrides it (`test_cluster_pool.go:28-35`). The runtime patch sets one
processor in deterministic mode (`go1.27.1.patch:172-176`). Reserving a slot is a bare channel
receive, released by the test's own cleanup (`test_cluster_pool.go:107-113`).

With `-test.timeout=2m`, which Gomad treats as a virtual deadline, the timeout panic prints every
goroutine. On both seeds the test goroutine is in

```
goroutine 12 [chan receive, 2 minutes]:
testcore.(*clusterPool).reserveSlot      tests/testcore/test_cluster_pool.go:109
testcore.(*clusterRouter).getDedicated   tests/testcore/test_cluster_pool.go:342
testcore.NewEnv                          tests/testcore/test_env.go:302
tests.(*NexusOTELSuite).TestOperation    tests/nexus_otel_test.go:143
```

and the log holds one cluster creation event for the test, not two. The test holds the only slot
and waits for it. In the suite the other tests queue behind it: on seed 17 all four tests wait in
`reserveSlot`, and on seed 11 three do after `TestCallback` has finished
(`fn105-d17-suite-self-wait.txt`).

Nothing bounds the wait in virtual time. The test context does not cover a channel receive, and
the Runner starts the test binary without `-test.timeout`. The first cluster stays up and logs
about 42 lines per virtual minute from its periodic checks
(`fn105-d17-hang-timeline-seed11.txt`). Each timer is the next event when the process goes idle,
so the virtual clock advances by hours in two wall minutes. A native sample of the hung process
shows it executing that server work and not spinning (`fn105-d17-native-sample-top.txt`).

## What the recorded classification was

The acceptance asks to separate target termination, watchdog or cancellation, runtime evidence
failure, and Runner failure from ordinary replay divergence.

- **Target termination.** None. The target never exits on its own. Its outcome has no exit code.
- **Watchdog.** Every execution on the tracked tree that ran without a virtual test timeout ends
  as domain `watchdog`, reason `watchdog_timeout`, deadline `execution_timeout`, at 118 s wall of
  the manifest's 2 m `run_timeout`. The supervisor kills the process group. The runs with
  `-test.timeout=2m` end as target failures instead.
- **Runtime evidence failure.** None was reported. The I/O transcript is complete with 17
  records. The terminal choice frame is absent because the target was killed before it could
  write one. The Runner treats that as part of the outcome: it does not validate a killed run's
  trace, a watchdog artifact records no choice profile (`runner.go:1515-1520`,
  `runner.go:1653-1656`), and such an artifact must use diagnostic replay
  (`tools/gomad3/record/validation.go:383-384`). A trace overflow in a killed run would therefore
  not be reported, and these runs cannot exclude one.
- **Runner failure.** None in these runs; the set reports show `infrastructure_errors: 0`. The
  check that produced one earlier still exists for targets that exit: a traced run without a
  complete terminal frame fails with "enabled choice profile did not produce the required
  terminal trace" (`runner.go:1669-1670`). The task that added the two-slot setting recorded that
  it stopped the Runner from applying this to killed targets. This investigation did not rerun the
  older Runner.
- **Replay divergence.** Not shown to be a diverging schedule. The replay runs without a tape,
  produces stderr of the same length up to the same wait, and is killed by the same watchdog. Its
  one differing byte is accounted for below.

The set report still classifies the workload `nondeterministic` with `timed_out: 0`. Two
repetitions that were both killed by the watchdog are compared field by field, and the comparison
stops at the first mismatch (`tools/gomad3/qualification/qualification.go:394`). That mismatch is
the virtual time at which each was killed: 37,061 s on seed 11 and 14,044 s on seed 17 in the
first repetition. The value is host-timed for a killed target. Whether later fields also differ
was not retained, apart from stderr, whose digest is equal across repetitions.

## First divergent event

No Choice Trace exists for the failing runs, so `inspect --choices` has nothing to compare and a
choice divergence cannot be excluded from traces. Three observations locate the differences that
were recorded, and none of them indicates a diverging choice.

- **Between repetitions.** Stderr is byte-identical in all three traced recordings of the seed-11
  suite and in both of seed 17. The first differing evidence field is
  `virtual_time_elapsed_nanos`.
- **With a virtual deadline.** The failing leaf run repeats byte for byte, including the goroutine
  dump with its addresses, in three runs per seed across both Runner builds.
- **Between a recording and its replay.** The recorded stderr and the replay's stderr have the
  same length and differ in one byte, in line 401 (`fn105-d17-seed11-replay-stderr.txt`):

  ```
  < ... "address": "membership://frontend~0x4923ba3045c0" ...
  > ... "address": "membership://frontend~0x4923ba3045d0" ...
  ```

  The server builds that address by formatting a pointer with `%p`
  (`common/membership/grpc_resolver.go:79`). Both replays produce the same stderr as each other,
  and a fresh untraced run of the same seed produces exactly the replays' stderr. The recording
  ran with tracing and the diagnostic replay runs without it
  (`tools/gomad3/runner/replay_operation.go:355-360`: an artifact with no choice profile gets no
  choice capability). One heap object sits 16 bytes apart between the two modes. The seed-17
  recordings contain no `membership://` line and no hex address before the wait, and their
  replays match (scan at the end of `fn105-d17-seed11-replay-stderr.txt`).

Why tracing moves that object was not traced. The retained evidence is the equality of the
untraced run and the replays.

## Native comparison

Natively the pool has eight slots on this host and the suite passes, 12 of 12 subtest runs. With
`TEMPORAL_TEST_DEDICATED_CLUSTERS=2` it also passes 12 of 12; `TestOperation` then takes about 15 s
because it waits for the other tests to release the second slot. With one slot the native test
blocks at the same line as under Gomad until `go test` stops it
(`fn105-d17-native-results.txt`). The wait is the test harness's behavior with one slot. Gomad
differs from native Go in two respects: its pool has one slot by default, and the documented way
to change that has no effect.

## Two real clusters

The variant adds one line to the throwaway copy that sets the dedicated pool to two slots
(`fn105-d17-variant-pool2.diff.txt`). It stands in for a delivered
`TEMPORAL_TEST_DEDICATED_CLUSTERS=2` and changes nothing else. The test file and its assertions are
untouched.

- The leaf passes on seeds 11 and 17 and creates two clusters.
- The suite passes on seeds 1 to 17 with all four subtests. On seeds 11 and 17 the run creates
  five clusters, and `TestOperation` finishes last at 15 to 20 s virtual
  (`fn105-d17-pool2-sweep-seeds1-17.txt`).
- `qualify-set` with the skip lifted qualifies seeds 11 and 17 at repeat 2 and again at repeat 4,
  traced, with exact choice-tape replay of every retained success: 12 executions in total, each
  2 to 4 s wall. Per seed the virtual time and stderr size are equal across both runs.
- Under the strict tick the same manifest fails deterministically on both seeds, with exact
  replay of the failure. Two tests fail the span list comparison at
  `tests/nexus_otel_test.go:530`. In the first compared pair of each, the actual list holds the
  expected spans in another order once the order-assigned trace, span, and parent IDs are removed
  (`fn105-d17-strict-failures.txt`). `TestWorkerOperation`, which uses one cluster, lists the
  frontend server span before the history client span. `TestOperation` lists both client spans
  before both server spans. The comparison runs on spans sorted by start time
  (`tests/nexus_otel_test.go:473-475`). Whether equal start times under the strict tick produce
  these orders was not established. The tracked manifest already runs this suite under `forward`,
  and a single-cluster test fails as well, so the strict result does not show a two-cluster
  defect.
- The tracked configuration, with `TestOperation` skipped, qualifies with exact replay.

These runs support removing the skip only together with a correction that gives the pool two
slots. They were made in a throwaway copy, not from the tracked tree.

## D12 and D14

The task asked whether the D14 fix, which removed a host-timed draw from the seeded stream at
runtime-lock contention, also removed this divergence. It did not. No evidence of a shared cause
was found, and the wait described above accounts for the recorded results without one.

- The recorded classification reproduces on toolchain `8d28bd44`, which contains the fix.
- The first differing field is the virtual time at a wall-clock kill. D14's signature was a
  shifted seeded stream that moved later choices; here stderr is byte-identical across
  repetitions up to the kill.
- With a virtual deadline the failing run repeats exactly, and with two real slots a process that
  runs two clusters at once qualifies with exact choice-tape replay in 12 executions.

This does not prove the absence of a D14-like channel in the killed runs. They keep no Choice
Trace, and the unfixed toolchain no longer exists on this host, so no run compares the two
toolchains. D12 is a linux/amd64 channel and is unassessed, because no linux run exists.

## Corrections and owners

| Option | Change | Owner | Evidence here | Cost and risk |
| --- | --- | --- | --- | --- |
| A | Deliver supplied entries: in deterministic mode `syscall.copyenv` keeps the runtime's filtered list instead of replacing it. The Runner already starts the target from an empty environment and rejects reserved names. Then set `"environment": ["TEMPORAL_TEST_DEDICATED_CLUSTERS=2"]` on the suite in `tests.generator.json`. | Gomad runtime patch and Runner (`tools/gomad3`, owner `gomad3`) | The probe shows the defect. The two-slot variant shows what a delivered entry would produce. | Changes the runtime patch, so the toolchain build key changes and everything bound to the key is requalified. A target started outside the Runner with a seed would see the caller's environment, which the current hunk prevents; the fixture that uses the scrubbed environment as its activation signal needs a new signal. Not built or verified. |
| A2 | Keep the environment hidden and make that the contract: reject `--env` and the manifest `environment` field, and correct `TUTORIAL.md`, `ARCHITECTURE.md`, and the generator comment. | Gomad Runner and documentation (`tools/gomad3`, owner `gomad3`) | The probe | Small. Removes the false evidence. Leaves this test without a way to get two slots, so it needs option B as well. |
| B | Size the dedicated pool for Gomad in `tests/testcore` behind the `gomad` build tag, as `flag_sql_gomad.go` does for another setting. The default build is unchanged. | Gomad integration in `tests/testcore` (owner `gomad3`) | The variant qualifies seeds 11 and 17 with exact replay and passes seeds 1 to 17. The variant set the size unconditionally; a build-tagged file was not written. | No toolchain change. Within the milestones' bound for server source changes. Does not fix the environment defect. |

Recommendation: option A. The cause is that a documented Runner input is recorded and then hidden
from the target, and A removes it at the source and serves any other test that reads a setting
from its environment. If the decision prefers a hidden environment, A2 with B reaches the same
result for this suite. Whichever is selected, either A or A2 is needed, because the tracked tools
currently accept and record an input that has no effect.

Two further observations are recorded for the decision. Neither is needed to enable the test.

- A workload whose repetitions are all killed by the watchdog is reported as `nondeterministic`
  with `timed_out: 0`. The owner is the Gomad Runner's qualification assessment. Reporting the
  watchdog outcome ahead of the comparison would have named this problem on the first run.
- A diagnostic replay of a traced watchdog artifact runs untraced, and an application that prints
  heap addresses then differs from the recording. The owner is the Gomad Runner. Passing a virtual
  `-test.timeout` to `go-test` workloads would turn a wait like this one into a repeatable target
  failure with a goroutine dump, in 16 s instead of two wall minutes per execution.

## Proposed next action

Open work in fn-105 for a subsequent decision. The skip stays in the tracked manifest through
step 6.

1. Decide between A and A2 with B.
2. For A or A2, add a conformance fixture that supplies an entry and reads it in a package `init`
   and in `main`. Under A it must see the value, and with no entry it must see `TZ=UTC` only.
   Under A2 the Runner must refuse the entry. The probe in this report fails the first form today.
3. Give the suite two slots through the chosen route, keep the `skip_subtests` entry, regenerate
   `tests.json`, and validate it with `qualify-set --check`.
4. Run the leaf on seeds 11 and 17 from the tracked tree. Require a pass and two cluster creation
   events for `TestOperation` on each seed.
5. Copy the suite entry into a scratch manifest, remove its skip there, and run `qualify-set`.
   Require `qualified` on seeds 11 and 17 at the manifest's repeat count, and a traced run with
   exact choice-tape replay. Run the suite verbose on seeds 1 to 17 and require four passing
   subtests per seed. Run the native suite and require it green.
6. Keep a one-slot reproducer. The native control is available under every option:
   `TEMPORAL_TEST_DEDICATED_CLUSTERS=1 go test -tags test_dep ./tests -run
   '^TestNexusOTELSuite$/^TestOperation$' -timeout 60s` must time out with the test goroutine at
   `test_cluster_pool.go:109`. Under option A the Gomad leaf with no entry and `-test.timeout=2m`
   must still fail the same way, since the pool then keeps one slot. Under option B the tagged
   build always has two slots, so the Gomad form needs a throwaway one-slot variant and the native
   control is the retained reproducer.
7. When steps 2 to 6 pass, remove the `skip_subtests` entry, regenerate `tests.json`, and qualify
   the suite from the tracked manifest.
8. Run the suite on linux/amd64 when a host is available. The decision must state whether that
   run gates step 7.

`TestOperation`'s application-tracing assertions are unchanged by every option: four HTTP spans in
two traces with their parents, services, kinds, URL paths, and Nexus attributes
(`tests/nexus_otel_test.go:206-256`). No option edits the test.

Whichever option the decision selects, including keeping the skip, the skip's recorded reason
should be replaced with the cause stated here. It currently says that with the setting the test
runs and that two clusters in one process are not yet deterministic.

## Limits

- No linux/amd64 run.
- Options A, A2, and B are not implemented. No toolchain was built with a changed patch. The
  two-slot evidence comes from an unconditional one-line change in a throwaway copy, not from a
  delivered environment entry or a build-tagged file.
- The reason the environment hunk hides all entries is taken from the conformance fixture's
  comment. No commit message that explains it was found.
- No diverging runtime choice was observed. Whether choices diverged in the killed runs is
  unknown, because they keep no Choice Trace, and a trace overflow in a killed run would not have
  been reported.
- No test was found that reads a supplied entry inside a target. The search covered the Runner,
  CLI, and conformance tests by name and was not exhaustive.
- The address difference between a traced recording and an untraced run is established by
  comparing outputs. The allocation that moves was not identified, and one pair of modes on one
  seed was compared.
- The cause of the seed-11 difference between env=2 and no entry was not traced beyond the fact
  that the runtime copies the entry.
- The cause of the strict-tick span orders was not established. Span timestamps were not printed.
- The first session's runs were made on an uncommitted working tree and the second session's on
  its commit. Equality is shown for five files and one repeated run, not for the whole tree.
- The earlier Runner failure on seed 11 was not reproduced, because the Runner that produced it
  was not rebuilt.
- All runs shared the host with another project's tests. No run was made on an idle host, and no
  dedicated load generator was used.
