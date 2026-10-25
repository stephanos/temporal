# fn-232.13 receiver driver development handover

Base: `86dcc2b1f2415b00dc14e5818c6dba963bf13828`, branch/workspace
`fn232-rust-driver` / `/home/agent/wt/fn232-rust-driver`. This work implements the
receiver framework and executable adapters; it does not accept or complete .13.
The conductor owns integration, independent review, final identity freeze,
qualification and lifecycle. No Flow mutation, retirement, Make/CI edit, push,
Go/Rust build or actual guest/qualification run was performed. The existing
integrated validate=0 baseline was reused as instructed; no broad baseline was
repeated. No additional agent was delegated.

The implementation is under `tools/gomad/rust/driver`; [README.md](README.md)
contains execution/configuration contracts and the new semantic mappings.
`semantic-map.json` preserves all 96 labels, original bodies/root selectors and
the grouped gate's ten repetitions. Both serial and parallel modes dispatch the
same actual adapter code. The unchanged original resource DAG/scheduler is read
from receiver custody. Missing real producers/readers stay UNEXECUTED. The
final reader must additionally authenticate all 145 named required proofs
(39 live roots, 21 tamper cases × four entrypoints, and 22 receiving obligations).
A manifest, fixture or historical receipt cannot earn those proofs.

Actual adapter routes execute public qualify/report/explore/replay/inspect and
.11 plan/preparation receipts; exact fixture/unchanged ./tests spec generation
is included. Runtime runs 32 shards ×25 fresh single-seed recordings plus one
fresh forced full replay per shard, with original hook/counter/terminal and
variation assertions. Actual runtime terminal retention was coordinated with
the live-corpus backend owner (`evidence/runtime-terminal`, manifest-only raw
16-byte payload; no change to Record.streams). The SQLite owner and this receiver
agreed on one deduplicated native catalogue and an exact full-state/context
Selection. The receiver generates four selected specs and authenticates all
five actual Results. Its catalogue uses the original Go oracle limits and counts
only four selected execution costs plus their publication tails once.

Identity binding uses native Rust 64-character digests, never normalization of
artifact fields. Native Go State/Context values keep PascalCase and `sha256:`
identities. Actual binary/helper/host/runtime/compiler/module identities were
**not measured** in this development window. The example's zeros are deliberately
unbound placeholders, not an identity claim. Source/HEAD/symlinks/config and
pinned tools are guarded, and actual CLI PIDs bind measured producers. A deadline
above 3600 refuses; a complete overrun exits 3 after final checks, with preamble
and suspension included through the original send epoch.

## Focused verification

| Command | Numeric exit / observation |
| --- | --- |
| `PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -v` | 0; 21 tests. Missing/duplicate/unexecuted/synthetic subgates, role/identity tamper, source mutation/symlink, refusal classification, send-to-end deadline, zero/skipped/duplicate roots, process timeout/reap/inconclusive receipt, exact workload set/argv/bounds, runtime diagnostic/terminal bytes, native catalogue/context and full-state selected spec. |
| `python3 tools/gomad/rust/driver/driver.py plan` | 0; 96 stages, identical serial/parallel shared dispatch argv. Static observation only. |
| `python3 tools/gomad/rust/driver/driver.py stage --config /not-present --config-sha256 a storage` | 2; expected setup refusal, no actual stage execution. |

Test-first RED observations preceded implementation: missing driver/adapter/config
and native-catalogue modules (exit 1); native Rust digest encoding regression
(exit 1, unwanted `sha256:` prefix); timeout receipt absence (exit 1); widened namespace/model limits (exit 1). GREEN
observations above establish framework behavior only. Synthetic tests earn no
actual gate. Quiet profile hold was respected throughout.

## Concrete pending endpoints and next integration

1. **Mandatory producer owner** `/root/mandatory_workloads_impl`: finish actual
   durable SQLite and independent Temporal replication scenarios/specs and
   release Result/report recomputation. Proposed schema is Scenario.workload
   `{kind:"sqlite-durable-recovery",selection:null}`
   or `{kind:"independent-temporal-replication-durable-restart",canary:bool}`;
   Result native_crash_catalogues and native_crash_selection remain typed and
   raw-authenticated. Feed its actual primary scenario/installation into the
   qualifier and selection routes. No operation-only SQLite or small scenario
   fixture is accepted.
2. **Independent native Go oracle**, not yet integrated: pinned reader argv plus
   `--input FILE`; request names the actual authenticated GMSTORE state path,
   hash and original 32/8/8/4MiB/2^16 limits. Return the actual full native
   enumeration and pending context. The catalogue is UNEXECUTED without it.
3. **Live corpus owner** `/root/live_corpus_impl`: integrate all 39 original
   root/variant fixtures and the 21×4 admission matrix, then supply a bound
   independent machine reader. Individual backend/cluster scripts exist in
   that owner's workspace, but no complete 39-root pass was claimed. Root35
   additionally needs the exact ignored Ordinary/default-feature linked unit
   and 16 actual live-map reinsertions, not its supplementary fixture alone.
4. **Source/build/retirement wiring**, conductor-owned: bind actual retained Go
   oracle/model/source commands, consolidated Cargo feature matrix and two
   regeneration passes to existing labels. Replace obsolete wasmmeasure/helper
   build/publication labels with actual linked build/admission/measurement
   producers/readers, retaining obligations. Do not run old Go host producers.
5. **Actual Temporal storage/retry/inspection producer/readers**, not supplied by
   the mandatory ten-cell producer alone: all committed-crash supported states,
   named reset workflow retry, and shard/workflow/onebox inspection with original
   source, detached inputs, capacities and assertions. Retained/operator routes
   still require the twenty fresh observed replays and installed detached proof.
6. **Terminal host admission and measured linked attribution**: original timing
   and host/capacity admission, exact ./tests module preparation equality,
   release-feature evidence, and publication-tail attribution. A generic plan
   receipt is not a terminal measurement pass.
7. **Stress owner** `/root/stress_impl`: integrate actual 41-case/four-axis
   event-order matrix/two-cluster Bound request, original repeats/capacities,
   real role binaries and retained first-divergence/stop evidence. No fixture
   manifest, missing prefix or wrong-role refusal replaces its positive path.
8. **Attribution owner** `/root/rust_attribution`: reconcile actual R4 runtime
   release-behavior and qualified-workload choice/WASI/network/stdout/stderr
   preservation, with original inputs and limits. The full activity recipe is
   unchanged ./tests, original ActivityTestSuite, 4GiB/2^50 bounds; early proof is
   not the final candidate qualification.
9. **Rotated inspection receipt**: actual three-node ringpop, 120 seconds, fresh
   retained exact replay and canonical World/network/choice preservation apart
   from declared rotations. Still required by the final reader.
10. **Final reader**, conductor integration required: pinned executable reader
    must authenticate the exact required-final.json set and every original
    test-wasm behavioral obligation. Missing endpoint refuses final completion,
    even with all 96 stage receipts. Run the full resource-scheduled cycle only
    after final source/identity freeze and prerequisite acceptance.

The assigned worktree is retained for review/integration and has no build/cache
or actual qualification scratch root. Temporary Python contract tests own and
remove their private roots. No command started by this worker is left running.

stage: impl-review - skipped(policy: conductor owns integrated review)


## Independent-review follow-up after 9af80795ee

Three confirmed findings were fixed together in receiver-owned files:

- The retained RealExecutor intentionally blocks TERM/HUP/INT around Popen,
  and a child inherits that mask. The receiver now installs handlers and
  explicitly unblocks all three before stage work. Busy workers receive TERM;
  their cleanup also has a bounded kill/reap fallback. The original scheduler,
  96-stage DAG, target command argv and resource policy remain unchanged.
- Detached command sessions now wait on a private admission pipe until their
  process group is atomically registered. The receiver exec bootstrap unblocks
  inherited signals and keeps the actual target argv/PID. EOF prevents an
  unregistered command exec after parent death. Receiver scheduler cleanup kills
  any registered command group left by a hard-killed stage. Normal interruption
  kills and reaps the command and leaves an inconclusive observation.
- Runtime terminal validation now rejects version-1 finish bytes with a zero
  peak witness, matching the original Go validateQualificationPayloads assertion.
  The earlier proposed nested runtime counter change was retracted by the
  reviewer: the actual public serializer flattens those fields. No counter
  schema change was made.
- SQLite selected producers must use the primary installation. Both individual
  selection and independent catalogue aggregation now compare compiler,
  runtime/helper/engine/model/profile, host/config/module, original matching
  fields and all raw preparation provenance inputs. The later post-preparation contract below
  supersedes the initial scenario mutation; no retained field is normalized.

Actual lightweight subprocess regressions use the real retained RealExecutor
and the receiver stage entry. RED observed `[1,2,15]` in the stage's blocked mask
(exit 1) before busy work. GREEN runs the actual twelve-worker pool to return 0,
interrupts an actual command with TERM and observes its numeric exit/inconclusive
receipt, and forcibly KILLs a stage before the receiver independently kills its
registered detached command group. These are Python framework fixtures only;
no guest, Go/Cargo build or qualification gate ran.

The focused runtime terminal test was RED (exit 1: valid version + zero peak
wrongly accepted). The selected compiler/model/config regression was RED (exit 1)
before the shared matching check existed. Both now pass.

Latest command:
`PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools/gomad/rust/driver/tests -v`
returned **0**, **25 tests**, including all process, terminal and SQLite matching
regressions. `git diff --check` returned 0. Independent integrated review and
actual candidate qualification remain conductor-owned and pending.

Root19 final-reader decision: require all applicable actual Rust simulation-model
lazy-store/only-requested/one-result/zero-callback assertions, plus the separately
executed identity-bound frozen Go BrokerReference hard-isolation observation.
The latter is explicitly a legacy reference architecture observation; no Rust
hard-isolation support, fake flag, callback port or skipped positive path is
claimed. The final reader still must authenticate both applicable parts.


## Second independent-review follow-up after d6228e884b

Producer completion now checks the registered process group after the direct
parent exits. Any remaining member triggers SIGKILL and an inconclusive refusal,
including parent exit 0 or nonzero. The original direct-parent numeric exit is
retained in both receipts, with cleanup_signal 9 in the process journal. A stage
cannot earn PASS while producer descendants continue running. Actual short
Python reproductions were RED: exit 0 incorrectly returned and exit 7 left the
child sleeping. Both statuses now reject and the children become gone/dead.
The actual four-test process suite returned **0**, four tests, 0.53 seconds;
attribution was notified before and after, and further busy-pool tests held.

The mandatory producer owner confirmed the exact post-preparation SQLite seam:
selected specs preserve the original Scenario bytes and add only
`RuntimeWorkload={kind:"sqlite-durable-selection",primary_result:ABSOLUTE_RESULT,
selection:{primary_result_sha256,state,context}}`. All manifest compiler/runtime/
module/HostConfig and complete preparation provenance remain identical. The
actual cluster record configuration changes to seal the selection and retained
primary capsule. Public report authenticates that capsule; this receiver then
independently requires provenance.runtime_workload to equal the exact expected
request and every other provenance field to equal primary byte values as parsed.
No HostConfig hash check is removed or normalized. Both individual selected
readers and the final catalogue apply this comparison. The old pre-preparation
scenario-selection description is superseded.

SQLite regressions were RED for missing/wrong intent, changed scenario, unknown
provenance fields, and changed Scenario bytes. Focused SQLite suite returned
**0**, three tests. These are receiver fixtures, not actual SQLite qualification.
Actual producer integration and the existing pending endpoint list remain open.
