# Matched developmental campaigns

The frozen current source completed discard and novel-success campaigns at 10
and 100 jobs with parallelism 2. Both baseline and current retain four live
1MiB stream objects and two live 1MiB encoded transcripts at their paired
checkpoints, and zero of those live objects after completion. Named logical
policy storage remains independent of job count. This evidence covers the
injected stock-linux/arm64 fixture; both native qualifications remain incomplete.

## Source and execution contract

The comparator is the reviewed [bound baseline](../bound-baseline-measurement/measurement.md),
derived from actual first-task revision `6782b55f49a0317b230e827ea2a63a37d116d502`
plus retained dirty fn-108 changes. It has 670 original paths and 675 frozen
scratch paths. Nested Git tree `38957053f1ce342a8797af1803f5f8f6bb53fcad`
corroborates the reconstruction only inside `tools/gomad3`.

Current shipped source is revision `8604c07def0f97b63cbca3864b4c286d6803c4b1`,
copied from all 978 tracked nested-module paths with original file modes to
`/tmp/fn109-r19-current.RyYsbzUr/gomad3`. The five predeclared supplementary
paths include four added harness/guard files and the already shipped
`runner/internal/execution/descriptor_dup_linux.go`, reused byte-identically.
The [initial inventory](runs/initial-source-inventory.json) freezes all 982
paths, hashes and modes before any Go build. Every case's complete before/after
inventory and the final inventory match it. The one-line developmental host
overlay is identical to the baseline condition, with a 14-line location offset
in the current file; no production platform or boundary manifest was changed.

Driver SHA256 `8252784ee6b75db8038ff3616ba4d12e4ec543a0b6eebe891c688a70c21ec031`
was recorded before invocation and again in its persisted pre-launch evidence.
It still matches after all commands. Driver session 38361 exited 0. Its 98
children all exited 0 and retain argv, cwd, UTC launch, duration, environment
and output hashes in [execution evidence](runs/execution-evidence.json).

The initial session 70846 stopped before any Go build or campaign because GNU
patch created a `.orig` backup after its location-offset match. The complete
failed preflight and driver are retained in `failed-inventory-preflight/` and
`failed-inventory-run_current.py`. A fresh scratch suppressed patch backups;
the source check still requires the exact complete inventory. No failed run
was treated as green and no source expectation was relaxed.

The exact pinned stock Go binary is
`/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`,
SHA256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`.
The driver binds exactly the baseline's 14 values before each child.

```text
GOENV=off GOFLAGS='' GOEXPERIMENT='' GODEBUG=''
GOGC=100 GOMEMLIMIT=off GOMAXPROCS=2
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 GOARM64=v8.0
GOWORK=off GOTOOLCHAIN=local TZ=UTC
```

All four actual binary build-setting records match the baseline except their
binary path. Runtime metrics confirm GOGC 100, disabled memory limit and two Ps.
No new explicit compiler/C controls enter this comparison. Baseline's later
compiler observations are retained as observations, not historical prebindings.

The paired executor retains no result/request history. Payload sizes, one-probe
vocabulary, 8192-operation encoded transcript, seed-specific connected World
record, declared capacities and timeouts are identical. The only harness
adaptations are the existing private `testConfig` dependency tuple,
`exploreWith`, atomic `Complete(CompletedSuccess())`, and the diagnostic-field
presence description. Each profile follows two GCs; `KeepAlive` retains both
paired results through their snapshots. All cases report `(returned, committed,
active)` of `(0,0,2)`, `(N-2,N-2,2)`, then `(N,N,0)`.

Baseline's two old Dup2 call sites are compilation-routed to fail-fast guards;
all cases report zero guard calls. Current production uses its shipped Linux helper and
does not call the supplemental guard. The current zero counter alone proves
nothing about transport. The injected preparer and executor define the common
exclusion of actual child transport, real target preparation and replay.

## Measured state and allocation sites

The four campaign commands each use `go test -tags test_dep -count=1
-timeout=6m -run '^TestR19BoundedSeedCampaignMeasurement$' -v
-memprofilerate=1 -memprofile=<case>/process-end.pprof -o <case>/runner.test
./runner`. Case variables select discard/novel and 10/100; exact absolute
commands are retained in execution evidence.

| Case | Current seconds/exit | Attempted/succeeded | Max active | Retained count/bytes | Current journal records/bytes |
| --- | --- | --- | --- | --- | --- |
| discard-10 | 11.043931 / 0 | 10 / 10 | 2 | 0 / 0 | 10 / 3367 |
| discard-100 | 71.930231 / 0 | 100 / 100 | 2 | 0 / 0 | 100 / 33837 |
| novel-10 | 9.246927 / 0 | 10 / 10 | 2 | 1 / 3155327 | 10 / 3520 |
| novel-100 | 64.124884 / 0 | 100 / 100 | 2 | 1 / 3155327 | 100 / 33988 |

The both-role logical total grows from baseline 4120 bytes to current 4472
bytes, a 352-byte increase. Both totals are identical at 10 and 100 jobs.
All components match except four completion slots, whose total grows from
3744 to 4096 bytes as the retained completion record grows. This accounting
includes both controller and ordering sources/iterators with one shared range
backing; it excludes allocator metadata, map/closure backing, runtime channel
headers and payloads. It is not a heap-size tally. The initial comparison
verifier incorrectly asserted byte-for-byte policy equality; that failed
assertion and script remain in `failed-policy-equality*`. The final verifier
reports the measured delta instead of converting it into an unchanged claim.

Named campaign-policy sites retain 368 live bytes at both pair checkpoints
for all cases, matching baseline. Current direct `runLocal` late-site live
samples are 7432 bytes for discard-10 and 7448 bytes for the other three;
baseline values are 6808 discard and 6824 novel. Exact invariance of that
current aggregate is unsupported by the 16-byte observed difference.
Journal capacity grows with N as explicitly declared evidence capacity;
journal record bytes and retained evidence are not live policy storage.
Private novelty-map cardinality is not directly measured.

Completed-profile per-execution allocation uses N as its denominator. Novel
publication allocation uses its actual count of one, never N.

| Site/category | discard-10 baseline/current | discard-100 baseline/current | novel-10 baseline/current | novel-100 baseline/current |
| --- | --- | --- | --- | --- |
| Encoded transcript bytes/execution | 1048704 / 1048704 | 1048704 / 1048704 | 1048704 / 1048704 | 1048704 / 1048704 |
| Stream producer stack bytes/execution | 2097177.6 / 2097177.6 | 2097154.56 / 2097154.56 | 2097177.6 / 2097177.6 | 2097154.56 / 2097154.56 |
| Coverage/outcome assessment bytes/execution | 1147881.6 / 1148005.6 | 1147924.08 / 1147952.32 | 1147922.4 / 1147989.6 | 1147951.6 / 1147922.48 |
| World assessment bytes/execution | 442389.6 / 441318.4 | 440480.88 / 440603.68 | 442616.8 / 441556.8 | 440887.6 / 440542.08 |
| Publication stack bytes/publication | absent / absent | absent / absent | 1401640 / 1476008 | 1413328 / 1480488 |

At exact producer sites current allocates two 1MiB streams and one 1MiB
encoded transcript per execution, matching baseline. DecodeTranscript's
983040-byte typed-operation storage is also one allocation per execution,
with zero live bytes after completion. No new 1MiB extraction allocation site
appears in these profiles. Current `artifact.copyWithContext` allocates nine
64KiB buffers per novel publication (589824 cumulative bytes), versus
baseline eight (524288 bytes); all are dead at completion. Exact raw stacks
attribute six buffers to the six inline payload writes, one to the manifest
write and one to the target copy on both sides. The added current buffer is
`verifySharedPayload` (`artifact/target_pool.go:226`) through
`placeSharedPayload` and `Store.PublishArtifact`. It opens the shared target and
streams its bytes into a SHA256 hasher. The fixture target is 20 bytes,
`fake prepared target`, with a shared pool inode and link count 2; the three
large payloads remain stdout, stderr and transcript. Commit
`bc2e970b5306aa09f594657a8d42c159cf4a1270` assigns that path to fn-114.9's
target sharing, independently of fn-109's extraction. This is one additional
filesystem target verification read with a bounded buffer, not an observed
1MiB heap clone. The retained sample attribution does not infer integer
whole-payload copies from aggregate totals or generalize this fixture.

All sites, allocation-size buckets and disjoint categories reconcile the four
metrics of each of the 16 current profiles. [comparison.json](comparison.json)
retains exact matched categories, all runtime snapshots, binaries and deltas.
Broad runtime/library/fixture categories, JSON/World allocations, allocator
rounding and profile overhead prevent a universal absence-of-copy or private
storage proof. Early snapshots have denominator zero; late snapshots include
two produced but uncommitted payloads. The completed profiles supply the final
per-execution comparison. No throughput or memory-reduction claim follows.

## Independent integrity and remaining gates

[verify_current.py](verify_current.py) exited 0 and retained its result in
[verification.log](verification.log). It checked all 982 current source paths,
978 shipped paths, ten complete source inventories, 98 successful commands,
174 current output hashes and all 201 immutable baseline handoff hashes.
Original reconstruction, bound-baseline artifacts and bound scratch remain
unchanged. The known task-description-only historical input drift is disclosed
in execution evidence; no historical manifest was rewritten.

A separate read-only measurement scout independently recalculated all four
metrics for all 32 baseline/current raw profiles, checked binary settings,
and compared actual complete path/hash/mode inventories. It identified the
guard asymmetry and the measured increases described above. This check is
source-evidence analysis and supplies no formal review verdict.

Keep binaries, raw profiles, raw/top pprof logs, campaign payloads and duplicated
inventory snapshots local under `runs/`, identified by the retained manifests.
Checkpoint the harnesses, driver, analyzer, verifier, structured measurement/
comparison/command metadata and lightweight reports. A new checkout reproduces
local bulk outputs before rerunning the verifier. Neither native `darwin/arm64`
nor native `linux/amd64` ran. R19 and task-21 acceptance remain incomplete.
