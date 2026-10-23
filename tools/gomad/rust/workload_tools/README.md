# Actual mandatory workload producers

`producer.py` runs the original durable SQLite or independent Temporal
replication cell through public `install`, `qualify --spec`, and installation-free
`report --result`. Each invocation requires a new root, prepares its own target
in its own process, retains two fresh executions and their exact replays, and
checks the original success/canary, bug-count and measured-budget conditions.
Raw nested command exits and stdout/stderr are retained beside the producer
receipt. A canary's actual `qualify` exit is 1. Authenticated `report` returns 0 for
either cell; the wrapper verifies the retained classification and original verdict.

`workloads.py` constructs the original requests and copies original Temporal
fixture bytes without rewriting them. Temporal installation takes the captured
schema preimage from a receiver-owned configuration input. `producer.captured`
authenticates its descriptor/payloads and compares every file and directory to
this candidate's `schema/sqlite` before reuse. Old module, helper, runtime,
preparation measurement and Result identities are never adopted.

The scenario's optional `workload` is a closed typed registry, currently:

- `{"kind":"sqlite-durable-recovery","selection":null}`. The original boots
  are `setup-live`, `append-crash` (or real omitted-sync `unsynced` canary), and
  `recover`. SQLite alone owns the 4 MiB volume and its exact 32-state,
  8-operation, depth-8, 4 MiB, 65536-work enumeration. A selected-state request
  leaves this compiled base scenario unchanged. Its top-level spec `RuntimeWorkload`
  is `{kind:"sqlite-durable-selection",primary_result:absolute_path,selection:{primary_result_sha256,state,context}}`;
  only the append-crash boot receives the authenticated storage intent.
- `{"kind":"independent-temporal-replication-durable-restart","canary":false}`.
  The four actual boots bind active/standby commit and reopen, independently
  seeded entropy, all three peer service addresses, strict bidirectional links,
  partition/heal, grouped output acknowledgments, durable recovery and history
  convergence. The canary changes only expected event intent.

The registry intent is sealed in the preparation scenario and actual cluster
configuration. Workload observations are recomputed from actual stdout both at
publication and fresh replay, retained under `evidence/workload-oracles.json`,
and recomputed independently by `report` from authenticated artifact payloads.
Missing/malformed observations cannot become target canaries.

Rust Results optionally retain `native_crash_catalogues` and
`native_crash_selection`. A catalogue has `handle`, `volume`,
`storage_state_sha256`, full PascalCase `enumeration` and pending `context`.
The catalogue is projected from the actual volume owner's authenticated
GMSTORE v3 storage state and log binding, deduplicated only after exact equality
across fresh records. Four-state completeness, state/context binding,
dependency closure, append-only selection and recovered marker presence,
mode/time/size/hash are checked. The qualification driver's `sqlite.py` owns
primary Result authentication, the four fresh selected producers, independent
Go catalogue comparison and complete coverage aggregation. The closed SQLite
and committed-workflow adapters also retain the actual `storage_precrash_image`
GMVOLIMG bytes before lifecycle advancement. Ordinary workloads emit no extra
image. Public report requires and authenticates that stream; exact replay binds
it in the cluster summary. `internal/model/cmd/sqlite-native --input FILE`
authenticates `storage_state_path`/`storage_state_sha256`,
`storage_log_path`/`storage_log_sha256`, `precrash_image_path`/
`precrash_image_sha256`, and the actual descriptor's `model_identity`, with the
unchanged original SQLite limits. The retained Go model strictly imports the
full persisted objects and pending effects, independently enumerates them using
the original algorithm, and compares the complete catalogue and context. It
does not accept the supplied catalogue as an independent oracle.

`temporal-inspection.py` executes each original `shard`, `workflow`, or `onebox`
selection with its original bounds. It detaches staged source, makes installation,
compiler and runtime lookup unavailable, runs fresh replay and stopped inspection,
and checks useful stacks/heap, registered snapshot/probe data, prefix/witness
verification and unchanged artifacts. Onebox additionally requires completed
workflow timers/retries and selects `history/1`.

Unit/plan tests exercise request construction and guards. Their success is not a
live workload or frozen qualification pass. Actual producers, selected-state
runs, Temporal inspection and the final 3600-second cycle remain independently
scheduled by the conductor.

`producer.py --workload recovery --mode success` prepares and executes the original
committed-workflow fixture. `temporal-recovery.py --primary ROOT --root NEW_ROOT`
opens that Result through public report, reads its complete native catalogue,
and qualifies every state in a fresh producer with exact original primary tape
forcing, durable reopen assertions, and normal independent retained replay.
The base Scenario, module BuildKey and ChoiceIdentity remain fixed. A typed
`RuntimeWorkload` is added only after preparation and retained in provenance;
execution configuration and own-artifact primary capsules bind its exact
state, pending context, raw prefix and original history.

`producer.py --workload retry --mode success` uses the unchanged named-reset
fixture, original 1 GiB choice bound and 40-second rpc-reset. Public report
recomputes the original network witness from authenticated stdout/network:
established SDK stream failure, fresh connection and in-flight Unavailable/OK
RPC retry, plus sequential complete workflow history and exact normal replay.

`producer.py --workload smoke|frontend --mode success|canary` supplies the exact
fn222 selections, argument order, qualification policy (nil network), original
clock/entropy, runtime/storage/output bounds and fresh measured Result pairs.
The smoke is only one row; both rows also require preparation and detached
inspection receipts before the receiving support gate can pass.

`temporal-support.py` runs the complete receiving sequence through public
install/plan/qualify/report and detached inspection. It requires a frozen source
manifest, retains authenticated plan receipts with `guest_executions=0`, and
delegates to the bounded authenticated parity runner, preserving the original
private source, cache, policy and PID observation context. It emits all eight
ordered row gates only after both exact smoke and frontend pairs
and the original onebox inspection have passed. `--support` selects the sealed
`fn222-support` release example (default: `examples/fn222-support` beside the
release executable); the final list has Rust schema
`gomad.rust-wasm-support-list/v1`. The final full driver remains
the owner of serial/parallel ordering and aggregate acceptance.

The independent SQLite oracle builds from the retained Go reference only:
`go build -tags=test_dep -o /ABS/sqlite-native ./tools/gomad/internal/model/cmd/sqlite-native`.
Its sealed command is `/ABS/sqlite-native --input /ABS/oracle-input.json`.
Input contains `storage_state_path`, `storage_state_sha256`,
`storage_log_path`, `storage_log_sha256`, and exact lowercase limits
`{states:32,operations:8,depth:8,bytes:4194304,work:65536}`. It authenticates
both actual files, calls the original Go GMSTORE decoder and storage evidence
admission, and outputs full PascalCase `enumeration`/`context`. It executes no
guest and supplies no Rust Result identity. The conductor's source relocation
places the shared reference implementation under `internal/model/wasi` before
this endpoint is built.

The original committed-workflow primary retains commit `temporal/1` and two
fresh reopens `temporal/2` and `/3` from the identical actual primary image.
The internal runtime pins those bytes before either reopen and retains
`recovery_image_primary` plus each `recovery_image_input` source handle/digest
receipt. Reopen writes cannot replace the primary pin. Domain report/replay
recomputes the image binding against native storage state. Selected-state
producers retain the original single-reopen variant and authenticate their
source primary image, complete catalogue, context and interrupted tape.

Known recovery and onebox-inspection workloads resolve their original 1 GiB
choice bound before preparation, including the checked aggregate evidence
bound; the base request and preparation identity record that actual limit.
