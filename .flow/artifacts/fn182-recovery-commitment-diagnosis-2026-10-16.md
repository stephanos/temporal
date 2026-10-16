# fn-182 residual recovery commitment diagnosis

The repaired helper's cold matched multi-instance gate failed. A warm retained
diagnostic completed the unchanged scenario, but is not cold acceptance.
The next repair is limited to bounded exact-byte recovery commitment reuse.
fn-182 remains in progress; no full performance comparison or adoption follows.

## Retained runs

The failed matched run is `/tmp/fn182-repaired-r4-20261016`, session `93357`,
exit 1. Six onebox success cells passed; the first multi-instance cell failed
`overall_timeout`, iteration 1, helper wait 900.054789429 s. Its qualification
contains no completed executions or replays, and lacks terminal helper metrics.
All source inventory, link/gitlink, helper and qualification-binary postchecks
match. Details are in [the current batch report](fn182-current-batch-2026-10-16.md).

The separate diagnostic, session `32562`, is terminal with test exit 0. Its
actual `RunWASM` returned nil after 751.6575654 s. Both original incarnations
terminated by the intended crash controls; both fresh incarnations exited.
The adapter discarded the returned simulation result. A nil RunWASM error does
not prove a completed outcome or passing recovery/convergence oracles, because
scenario and oracle failures can be represented in that result. Those outcomes
are unverified in this diagnostic and must not be claimed as passed.
Explicit cluster result flags: Cancelled=false, WatchdogTimeout=false,
GroupGone=true; four terminal results, virtual time 33000000001 ns.

Root deliberately reused the failed cell's engine cache. Terminal metrics show
cache_hit=true, compile_nanos=0, cache_load_nanos=563476292, helper wall
751529439192 ns, actual process peak RSS 1134133248 bytes. These warm diagnostic
metrics cannot replace the failed cold gate or matched repetitions.

Raw evidence: `/tmp/fn182-repaired-retained-multi-diagnostic-20261016` and its
sibling `.log`. Cluster-result SHA-256:
`dcf18816f1aafbb1715310dfec7456f5a02a489f1834290473e9ab49cf60bf02`.
Both roles retain mirrored stdout, and all four incarnations retain raw network,
stdout/stderr and inspection payload files. Empty inspection payloads remain
empty; they are not fabricated authenticated inspection receipts.

The private adapter/build packet is
`/tmp/fn182-repaired-stack-tools-20261016-p9dgeq`. Root read its complete source
and diff: only the retained producer/input paths and expected helper hash changed
from the earlier adapter. Module SHA remains
`35f2d28c55c9de335a60d88c9011f58d83bcf8c731464786b02f83e4744f4c74`,
production helper SHA remains
`9ec2d4fce9be8f96337e0e7a7f67f1e5268bb4686f5272430fd4e9ab3db704f1`.
Seed, actual boot configurations, memory/fuel, scenario, crash markers, oracles
and 15-minute deadline were unchanged. Stdout mirrors assert unchanged boot
configuration identities. Adapter binary SHA:
`83738c2966993d546b5e9536154f269a77a74584b738a66f3c530183595dfb15`;
source SHA `8de0fcdae65646809546605bec40dce8dc2701203c4afbe880432ced49b61fa0`.
After terminal, all 2,012 component sources, helper and adapter/binary hashes
matched; HEAD stayed `55dc9175c1d9f9ec6d9de82e5454113f5dff42e7`, worktree clean.

## Localization

Four bounded debugger attaches revalidated the exact helper executable and
parent PID, disabled auto-load/debuginfod, used a five-second external timeout,
explicitly detached, exited 0 and left the helper running (`Sl`). No security
settings or host-installed tools changed. Existing private debugger hashes match.

| Helper elapsed | Running standby thread |
| --- | --- |
| 42 s | SHA-256 → inspection commitment → Recorder snapshot → debug exchange |
| 131 s | SHA-256 → volume byte digest → namespace inspection material → broker call |
| 334 s | SHA-256 → World state digest → expanded World serialization → debug exchange |
| 509 s | Same Recorder snapshot commitment call as the first sample |

The peer thread was broker-idle and the main helper awaited controller input in
all four samples. These are stack observations, not statistical CPU shares.

The repeated Recorder return address was `0xaaaacca555bc`; `/proc` maps bind
the helper load base `0xaaaacc900000`. Offline disassembly identifies the call
at static `0x1555b8`, returning to `0x1555bc`, with an eight-byte domain argument
at `0x8e5928`. Reading that exact literal yields `recovery`. This identifies
recovery-image commitment hashing, not the network or World commitment call.
The packet retains callsite disassembly, literal bytes and the load address.

A read-only volume ownership audit found another possible cache boundary:
`PersistenceOperation` fields are finalized before insertion and not subsequently
mutated by audited production callers. It also identified raw pending-effect
hash streams that cannot be replaced by standalone digests without changing
canonical framing. That audit is supporting evidence, not authority to bundle
operation, pending-state or World changes into this repair.

## Selected repair and gates

The updated Flow task fn-182.3 owns this follow-up. Reuse a recovery digest only
after comparing freshly encoded current recovery bytes exactly with retained
bytes for that handle/incarnation. Keep cache allocation and entry ownership
bounded, remove stale material before replacement, and use uncached hashing or
eviction on cache pressure rather than adding a semantic refusal. All other
state, metadata, namespace, recovery encoding, network, log and witness inputs
remain fresh. No freshness epoch, schema, policy, profile, dependency or budget
change is allowed.

The existing Rust worker continues in its original isolated workspace. Evidence
packet: `/tmp/fn182-recovery-cache-repair-evidence-20261016-BPdn2a`.
It must observe a real repeated-hash RED before implementation and retain GREEN,
independent literal-framing hash/evidence checks, same-length mutations, multiple
handles/incarnations, cache allocation/eviction/oversize tests, capacity and
sealed/exhausted controls, full Rust/conformance tests, fmt and both clippy gates.
It makes no Flow/Git lifecycle writes and no actual Temporal/helper campaigns;
root owns commit, integration and acceptance.

Before another full acceptance batch, root will run an explicitly nonacceptance
cold first-record diagnostic with an empty private engine cache and the original
15-minute bound. Retain the full simulation result and explicitly require its
completed outcome and passing original oracles, not merely a nil Go error.
This only checks critical-path feasibility. Current-helper
source review, runtime/Temporal qualification, a new full matched 16-cell run,
spread reporting, documentation and task/spec completion gates remain mandatory.
Native execution remains deferred. Delivery order remains fn-182 → fn-184 →
fn-187 → fn-185 last.

## Integrated source preparation

Root read the worker's complete handover and exact source/differential tests,
verified the one-file scope and frozen hash, committed it as `1826a8a8a3` in the
private workspace and integrated it as `6ca3554487`. Only `model/inspection.rs`
changed. Its SHA-256 is
`d3ecfa333ec9d30152cee6e65035aca9f782d0df9ceaa85094ba267d819d569d`.

Actual RED observed six recovery SHA operations instead of two; minimal cache
GREEN passed. A deliberate length-only comparison mutation produced the wrong
digest on same-length changed bytes; restoring exact comparison passed all
eight focused tests. The default Rust suite passed 399 tests, with two existing
diagnostic tests ignored; conformance passed 139. Both all-target clippy gates
deny warnings and exited 0, as did formatting and whitespace checks. Raw logs,
commands, intermediate failed test setups and final source hashes remain in
`/tmp/fn182-recovery-cache-repair-evidence-20261016-BPdn2a`.

The cache has 64 fixed optional slots. Its 64 MiB owned-storage bound charges
the complete fixed layout and exact boxed key/material lengths, with inline
digests and no retained spare Vec capacity. It compares complete freshly encoded
bytes per node/incarnation; replacement drops stale bytes first. Pressure evicts
and oversized material uses the original uncached hash, adding no semantic
refusal. Independent literal-framing and original-snapshot differential tests
compare complete Cuts and final evidence, plus original refusal precedence.
Other hash/model paths remain unchanged.

Root built the integrated production helper with canonical `make helper`, pinned
Rust 1.94.1 and locked dependencies: session `27125`, exit 0. The private helper
SHA-256 is `897178c7dc480598fe226715657ec04a5f8cfdfacc7d7e22d73910082662df64`.
The fresh 2,012-path component manifest and all post-build source checks match
under `/tmp/gomad-fn182-recovery-cache-qualification-20261016-S7ASd2`.
This is source preparation, not workload acceptance or independent review.

The new cold diagnostic adapter is retained separately at
`/tmp/fn182-recovery-cold-diagnostic-tools-20261016-p8V8Wi`. Unlike the earlier
adapter, it retains the full returned simulation result and requires completed
outcome, all seven exact passing original oracles, four terminal incarnations,
reaped group, no cancellation/watchdog and actual cold compilation telemetry.
It preserves the original retained module/inputs/scenario/15-minute bound.
Its feasibility result remains unproven until actual execution; it supplies no
full qualification or task/spec completion.

## Owner-requested stop

The cold diagnostic was admitted on clean `19ba9c67fe` and ran in session
`4957`. The owner explicitly requested stopping it. Root verified the helper's
exact private executable and parent before sending SIGTERM. The diagnostic
then exited 1 after 592.22 s, retained its infrastructure error and empty
simulation outcome with zero oracles, and reaped the helper. Driver and helper
PIDs are gone. This was an owner-interrupted run, not an observed watchdog
timeout or successful cold feasibility result. No retry or further gate ran.

Raw evidence remains at `/tmp/fn182-recovery-cold-diagnostic-20261016` and its
sibling `.log`; the adapter and binary remain in the separate tools packet.
All 2,012 component sources, the helper and diagnostic binary matched their
frozen hashes afterward. fn-182 remains in progress. Current-helper full
qualification, independent review and matched measurements remain unverified.
Work is paused at the owner's request; no active task-owned process remains.
