# Environment-bound developmental baseline

All four serialized baseline campaigns and the relevant companion controls passed. The source path sets, hashes and modes remained identical before and after every campaign. The original reconstructed baseline and every historical measurement artifact and historical scratch file remained unchanged. This is preparation for a future matched comparison, not that comparison, native qualification, or completion of R19/task 21.

The executing driver session was `79949`; its terminal exit was 0. No campaign processes remain live. The four test invocations were separate processes, ordered discard-10, discard-100, novel-10, novel-100. No production source, generated code, Flow lifecycle state or Git state was changed.

## Evidence and identities

The input baseline is `/tmp/fn109-baseline-reconstruction.lDSSw8Gx/tools/gomad3`, containing 670 original files. Its retained historical source manifest is SHA256 `d78601b3176195f8cc06860f5499e757a2d04333b9211f0ed13a92976b017845`. The historical input manifest remains unchanged, SHA256 `5b1904a15a5e11dd93926465d97c5467ab2a008f48b20610d9b0ffcfa4e8d0e6`. The earlier independent reconstruction established exact Git blob/mode equivalence to nested commit `38957053f1ce342a8797af1803f5f8f6bb53fcad` for the original base plus dirty fn108 inputs; this rerun used the retained reconstruction and rechecked its source manifest.

There is precisely one historical input metadata drift: `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.21.md`, historical SHA256 `4aba7e49a41f5567dceed8c81c038351af45ddb258cc577e753c341eaa30176b`, observed current SHA256 `ab5b964fde268884338678cd88da78d9acc6cd78e8e22027c837cba6719aab72`. It is the conductor's baseline-pointer update, not reconstruction source drift. The original input manifest was not rewritten.

The new isolated module is `/tmp/fn109-r19-bound-baseline.7hehYl3f/gomad3`. All five supplementary source files were declared before its initial inventory. Each of four campaigns has complete sorted 675-file path/hash/mode inventories before and after execution, compared to the initial inventory. The final inventory also includes the companion-control interval. Added paths, removed paths, changed hashes, changed modes and symlinks are rejected, not just hash failures for known paths.

- [execution-evidence.json](runs/execution-evidence.json) retains 98 child commands, launch timestamps, exact argv, environment subsets, exits, elapsed times and source/test/overlay/profile/binary hashes.
- [initial-source-inventory.json](runs/initial-source-inventory.json) is SHA256 `c419aee14eeb8e53aef78db3fbfa43b4f7dafb4e619402250a92a20ff9b1ee0e`; the matching before/after inventories are alongside it.
- [profile-attribution.json](runs/profile-attribution.json) retains every site's alloc_space, alloc_objects, inuse_space and inuse_objects, allocation-size buckets, disjoint stack categories and raw-profile identities. All four metrics reconcile exactly to every raw profile's samples.
- [verify_bound_evidence.py](verify_bound_evidence.py) independently passed; [verification-result.json](verification-result.json) records 171 retained output hashes checked and preservation of the original 670 files, historical artifact 337 files and historical scratch 675 files.
- [retained-output.sha256](runs/retained-output.sha256) is the driver's completion output manifest. A separate final handoff manifest covers later reporting and verifier artifacts without rewriting this completion manifest.

The bound measurement test is SHA256 `6e0204f41d1b00f382f4892622c339f0d2233f1bfe4f69aaddd09f61ec26991c`. Its only fixture change from the historical corrected harness is reading/asserting actual runtime metrics after the completed profile, then recording them. The logical companion is `c8288a1a4f57e613625501d309515ff85497828aa69b74cc27e9dfea7664c1fe`, artifact companion `aeda700f7abd9e9535566840856ba86ad64e1ad11bec51cfb1f8245ab841f223`, platform overlay `4005d79cdfe8bca2419fd233613fb647008e4d35f8c204a65cee8b80b805caa6`, guarded baseline call overlay `94ef16e36bd112775c17575ea72dbdb71b223a0bd26b05acacc89f721ab05bee`, transport guard `a3b1b6a44f4b6b90c5c496583f425c27fdf76b4f2df8ee291be9476f9b0dd59c`, exact reused Linux dup helper `16905765dc1ebb180d5695e347b85a6c611edb471ceebfb0978ca4cacbaf9674`.

The executing driver retained its completion hash `7ddc0ed0f0169bdc11751142fd8919246f341db491292b86ca145daa23f63432`. The parent inspected that script during execution. [The separate driver snapshot](runs/executed-driver-post-run-snapshot.py) was captured after the process had finished and matches this hash. It is not represented as a pre-launch source capture. No driver, harness or scratch edits occurred during the run. The analyzer hash is `3fec5fd795891adbe68b47886d01489e21881d236f1d7599b0cd8d77c6c26c69`.

All four compiled runner binaries have SHA256 `ac68f6c135acea44de8da96cec08e48e9bcbcac641c5009564492809fd8a8b4b`. The pinned stock Go binary has SHA256 `1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64`.

## Explicit environment contract

Before any Go build or test, the driver bound and persisted:

```text
GOENV=off GOFLAGS='' GOEXPERIMENT='' GODEBUG=''
GOGC=100 GOMEMLIMIT=off GOMAXPROCS=2
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 GOARM64=v8.0
GOWORK=off GOTOOLCHAIN=local TZ=UTC
```

The executable is `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/bin/go`. Every child command records these 14 exact entries, plus its case variables where applicable. [Effective safe Go settings](runs/effective-go-env-before-build.json), Go version, exact command/build flags and source inventory were retained before the first build. `go env GOENV` reports an empty path when the process binding is `GOENV=off`; that is the expected effective result.

The first driver preflight incorrectly expected that effective GOENV path to contain literal `off`. It aborted before any build or campaign. Its complete evidence and failed driver are retained under `failed-preflight-goenv/` and `failed-preflight-run_bound.py`. Only that assertion was corrected; the explicit environment contract did not change.

Each completed campaign independently observed `/gc/gogc:percent = 100`, `/gc/gomemlimit:bytes = 9223372036854775807`, and `/sched/gomaxprocs:threads = 2`. The memory-limit value is the runtime representation of disabled soft cap. These metrics were read after the completed heap profile to keep their measurement allocation outside that profile.

Actual embedded binary build metadata was obtained after each campaign. It records Go1.27.1, compiler `gc`, buildmode `exe`, `-tags=test_dep`, CGO_ENABLED=1, linux/arm64/v8.0 and empty embedded CGO_CFLAGS/CPPFLAGS/CXXFLAGS/LDFLAGS. This after-run metadata is evidence about the retained binary; it is not a claim that its embedded record was inspected before execution.

[Safe compiler settings](post-run-compiler-settings.json) were additionally observed after all campaigns under the same 14 bindings: CC=gcc, CXX=g++, AR=ar, PKG_CONFIG=pkg-config; effective CGO_CFLAGS/CXXFLAGS/FFLAGS/LDFLAGS `-O2 -g`, CPPFLAGS empty. These extra compiler variables were not explicitly bound or captured before the campaigns. Do not infer historical effective values from that later observation. A future comparison must preserve the explicit 14-setting contract, compare embedded build settings, and bind any additional compiler controls it relies upon, rerunning a matched baseline if those additional historical values cannot be established. The unrelated older unbound measurements remain historical evidence, not a matched environment baseline.

## Fixture and commands

The retained same-package baseline harness calls baseline `Explore(ctx, CampaignSpec)` with injected preparer/executor policy seams. It uses StrategySeed, Seeds 1-10 or 1-100, fixed Parallel=2, PolicyAll, CoverageSemantic, Guide=false, no resume/sharding/choice/guidance/diagnostics. The executor retains no request/result history. Each execution produces separate 1MiB stdout and stderr buffers, an actually observed 1MiB encoded semantic transcript, and the existing seed-specific completion World record. Each transcript contains 8192 operations: one `boundary.probe` for `stdlib.os.openfile` and 8191 `host.hostname` operations. Probe vocabulary is constant, not expanded per record or seed. Stream content is seed-distinct; World assessment follows the seed fixture.

Discard uses KeepSuccessesNone, success count and byte capacities zero. Novel uses KeepSuccessesNovel, count capacity 3 and byte capacity 64MiB. Output limit is 1MiB; IO transcript limit 64MiB; World transition limit 1MiB; execution and context timeout five minutes. No arbitrary sleep implements concurrency. The rejected historical 2MiB transcript-bound run remains unchanged in historical evidence; 64MiB is the accepted baseline contract, not a weakened test expectation.

For each case the exact command pattern was:

```text
<pinned-go> test -tags test_dep -count=1 -timeout=6m
  -run ^TestR19BoundedSeedCampaignMeasurement$ -v -memprofilerate=1
  -memprofile=<case>/process-end.pprof -o <case>/runner.test ./runner
```

Each child receives R19_JOBS=10 or 100, R19_MODE=discard or novel, R19_RESULT_DIR=<case>, and the explicit environment above. MemProfileRate=1 is also set in the test. The driver produces raw and top pprof outputs for all four metrics with no node/edge fraction pruning. Runner constructor aliases, artifact data payload aliases, both-role logical-policy accounting, and the unchanged original `TestRunBoundsActiveExecutionsAndRetentionAtTenAndOneHundredJobs` capacity characterization also ran separately with `-tags test_dep -count=1`. Their elapsed seconds were 0.315126, 0.410843, 0.240162 and 2.410558; each exited 0. Exact flags, inputs and outputs are in execution evidence.

## Campaign and persisted-evidence results

| Case | Attempted/succeeded | Maximum active | Retained count/bytes | Journal records/bytes/segments | Campaign files/bytes | Command seconds/exit |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| discard-10 | 10/10 | 2 | 0/0 | 10/3365/1 | 3/4650 | 9.454405/0 |
| discard-100 | 100/100 | 2 | 0/0 | 100/33835/1 | 3/35136 | 74.501350/0 |
| novel-10 | 10/10 | 2 | 1/3155326 | 10/3520/1 | 11/3160198 | 12.992839/0 |
| novel-100 | 100/100 | 2 | 1/3155326 | 100/33985/1 | 11/3190679 | 69.374841/0 |

Journal bytes, retained artifact bytes, campaign file totals and allocator memory are separate measurements. Cumulative produced stdout+stderr+transcript bytes are 31,457,280 for 10 jobs and 314,572,800 for 100 jobs. Cumulative World record bytes are 13,682 and 136,984. These cumulative totals are not live retention. The observed semantic probe vocabulary count is 1 in every case. The journal novelty-feature union is semantic 0 for discard and 1 for novel, choice 0 throughout. The private internal novelty-map cardinality is not directly observed and is not replaced by that feature union count.

Journal capacity is derived from the selection: maximum executions N, maximum bytes N*1MiB, segment bytes 1MiB, segment records 1024, maximum segments N, maximum partial executions 2. Artifact failure capacity N and failure byte capacity N*1,148,518,440; novel adds at most three successes/64MiB to that declared aggregate byte bound. These are capacity declarations, not allocated storage or evidence of policy growth. The original all-count/all-byte controls still enforce capacity, and are not substitutes for the full 10/100 completion cases.

## Logical policy, live heap and allocation sites

The logical companion counts both execution-controller and ordering job sources/iterators. The total is 4120 bytes for both N values: controller128, selectionheader32, two jobsource40 each, two iterator48 each, shared seed-range backing16, four completion slots3744, two channel handles16 and done handle8. The campaign JSON's older one-pair named tally4032 is intentionally retained and is not the complete both-role total. These unsafe.Sizeof totals exclude allocator headers, channel runtime headers, map/closure/runtime metadata and payload storage. Pprof captures actual allocations separately.

Paired gates preserve both local results with KeepAlive through snapshots after two GCs. Early captures Returned/Committed/Active=0/0/2; late=N-2/N-2/2; completed=N/N/0. The explicit completed profile is captured before opening the journal for report extraction. Ordinary process-end profiling includes that later reporting and is not a substitute for the gated late snapshot.

| Case | Early inuse bytes | Late inuse bytes | Completed inuse bytes | Completed alloc bytes/objects | Completed inuse objects |
| --- | ---: | ---: | ---: | ---: | ---: |
| discard-10 | 6414968 | 6459888 | 169320 | 169420496/1489188 | 1959 |
| discard-100 | 6457320 | 6514216 | 224144 | 1535959480/13494094 | 2002 |
| novel-10 | 6415288 | 6530752 | 238856 | 172656232/1588406 | 2782 |
| novel-100 | 6416440 | 6538032 | 245032 | 1547181984/14071562 | 2806 |

These pprof sample totals are not runtime HeapAlloc or process RSS. Exact runtime counters and all four metric totals at early/late/completed/process-end are retained per case and in profile attribution.

All early and late profiles contain four 1MiB stream allocation-bucket objects and two 1MiB encoded-transcript objects at their known producer sites; completed has zero live objects in those 1MiB buckets. Stream-producer metadata can still be live at112 bytes. Named campaign-policy site live bytes are368 during both pairs for every case; direct runLocal allocation-site live bytes are6808 discard and6824 novel at the late pair, independent of N. Named ordering insertion at campaign.go:13 allocates1024 bytes per committed result cumulatively, but has zero live bytes at gated snapshots; its constructor site has112 live bytes. Channel allocation sites are distinct from slot-only logical accounting.

The following completed-profile values use the correct denominator. Producer and assessment values are per completed execution. Publication values are per publication, whose observed count is exactly one in each novel case; none occurred in discard.

| Site/category | discard-10 | discard-100 | novel-10 | novel-100 |
| --- | ---: | ---: | ---: | ---: |
| Transcript encoder alloc bytes/execution | 1048704 | 1048704 | 1048704 | 1048704 |
| Transcript fixture other alloc bytes/execution | 983232 | 983233.12 | 983232 | 983232 |
| Streams stack alloc bytes/execution | 2097177.6 | 2097154.56 | 2097177.6 | 2097154.56 |
| Coverage/outcome assessment alloc bytes/execution | 1147881.6 | 1147924.08 | 1147922.4 | 1147951.6 |
| World assessment alloc bytes/execution | 442389.6 | 440480.88 | 442616.8 | 440887.6 |
| Publication stack alloc bytes/publication | n/a | n/a | 1401640 | 1413328 |
| Publication stack alloc objects/publication | n/a | n/a | 11258 | 11264 |

Exact baseline sites clarify the transient costs: deterministicio.EncodeTranscript:22 produces one observed1MiB size-bucket allocation per execution plus128 bytes for the probe fixture; deterministicio.DecodeTranscript:56 allocates983040-byte typed-operation storage per execution, with zero live bytes at completion; wire.DecodeTranscriptRecord:192 allocates16-byte record payloads. artifact.copyWithContext:347 allocates eight64KiB buffers,524288 cumulative bytes per one publication, zero live bytes at completion. No integer full-payload copy count is inferred from total profile bytes. Constructor alias controls show all nine nonempty supported runner payload fields alias producer storage, and the artifact payload companion inventories all input kinds. Baseline ArtifactInput has no diagnostic payload field; diagnostics are excluded. Synthetic constructor-only choice/simulation/mount inputs do not imply execution of those paths.

Attribution remains limited: broad runtime/library/fixture categories include substantial JSON/World costs and do not prove ownership or eliminate all transient copying. Private novelty storage is not directly counted. Profiles provide exact allocation sites and live samples, not a universal payload-copy proof or a memory-complexity proof outside this fixture. Early denominator is zero; late has two produced-but-uncommitted payloads, so completed profiles supply the final per-execution figures above.

## Portability and qualification exclusions

The scratch-only deterministicio host exception permits the injected fixture on linux/arm64 while the production native platform manifest is unchanged. Future matched current work must apply the exact same developmental platform overlay and fixture contract. It is not native platform validation.

The baseline contains two old syscall.Dup2 references that do not compile on pinned linux/arm64. A separate compilation-only overlay routes them to a fail-fast counter guard; the exact existing Linux helper was reused, including equal-fd semantics before Dup3. Any call to either excluded transport path would panic. Every completed campaign reports zero calls. This reports a baseline/current portability difference, not equivalent transport behavior. Real preparer, supervisor, executor process transport and runtime/toolchain qualification were not executed. No new dependency or patched toolchain was introduced.

The current source comparison has not run. Dependencies19/20 and task21's qualification admission remain conductor-owned. This evidence does not mark task21 or R19 complete, make a formal review/SHIP verdict, or establish native CI qualification. Requested routing was gpt-6.1-sol/high, Tier session (jev-unavailable(no_key)); actual model metadata is not asserted.
