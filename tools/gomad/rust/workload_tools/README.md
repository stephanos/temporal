# Actual mandatory workload producers

`producer.py` runs the original durable SQLite or independent Temporal
replication cell through public `install`, `qualify --spec`, and installation-free
`report --result`. Each invocation requires a new root, prepares its own target
in its own process, retains two fresh executions and their exact replays, and
checks the original success/canary, bug-count and measured-budget conditions.
Raw nested command exits and stdout/stderr are retained beside the producer
receipt. A canary's actual `qualify`/`report` exit is 1; only the wrapper's validated
expected canary outcome is successful.

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
  substitutes the exact typed `{primary_result_sha256,state,context}` into
  `selection`; only the append-crash boot receives it.
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
Go catalogue comparison and complete coverage aggregation.

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
