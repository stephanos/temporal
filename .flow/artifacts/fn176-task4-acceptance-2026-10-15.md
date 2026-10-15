# fn-176.4 storage fault and namespace acceptance

The implementation inherited at lifecycle baseline `ab39e8ec3c406c4181c293388929b0e06ce23168` is merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d`, including original fault/config/routing commit `6318cad368` and review corrections `629e735ca6`. This task adds four missing validator predicate cases; it changes no production behavior.

## R1 / R2: admission

`volume/config.rs::validation_mirrors_every_go_rule_including_crash_after` mirrors `wasi/volume.go::validateStorageModel`. Its cases cover:

| Rule | Focused cases |
| --- | --- |
| Model bounds | Empty and 65 volumes; 4097 faults |
| SQLite adapter bounds | 65 database paths |
| SQLite database paths | Outside mount, unclean path, overlong path, duplicate and unsorted paths |
| SQLite path collision | Database nested beneath another database; `.lock`, `-shm`, and `-shm/` overlap |
| Writable preopens and capacity | Missing writable preopen, per-volume excess, summed-volume excess |
| Fault identity | Empty, overlong, NUL and duplicate names; outside-mount, relative, unclean, trailing-slash, overlong and NUL resources; zero occurrence |
| Process-owned locks | Fault targeting configured database `.lock` |
| Operation | Unknown operation |
| Error payload | Nonzero bytes for `io-error` and `crash-before` |
| Sync action | Wrong operation and nonzero payload for `sync-error` |
| Space action | Wrong operation and nonzero payload for `no-space` |
| Bounded write action | Wrong operation, zero bytes and over-buffer bytes |
| Action | `crash-after` and unknown action |
| Ambiguity | Same resource/operation/occurrence with distinct names |

Rust strings and serde decoding enforce UTF-8 before this validator; the Go validator checks UTF-8 explicitly. `VolumeSession::namespace` maps each validator refusal to `invalid:<dimension>` before attaching storage. `Storage::new` additionally maps volume-core configuration refusal to `invalid:storage volume configuration`.

## R1 / R2: routed operations and fault evidence

The 46-script corpus enters `conformance.rs::VolumeSession::namespace`, creates `Namespace`, validates the model and calls `Namespace::attach_storage`. Ordinary records dispatch through the namespace; they do not call the volume core in place of WASI operations. The namespace routes close/sync, read, write, truncate, open, mkdir, remove and rename, refreshing storage size/mtime and checking namespace byte capacity separately.

| Contract | Existing differential coverage |
| --- | --- |
| Every fault action | `fault-{io-error,sync-error,no-space,short-write,torn-write,crash-before}.json` |
| Nonshortening writes refuse at runtime | `invalid-short-write.json`, `invalid-torn-write.json` |
| Admission refuses `crash-after` | `invalid-crash-after.json` |
| Exact event lines and transcript capacity | Exact `events_base64` comparison in every observation; `capacity-storage-transcript.json`; Rust `event_lines_match_go_marshal_field_order_and_omitempty` |
| Resource/operation occurrence and recovery continuation | Fault scripts and `recover-occurrence-continuation.json` |
| SQLite exception | `sqlite-lock-opt-in-{true,false}.json`: `.lock`/`-shm` ephemeral only with opt-in; database, WAL, journal and unrelated lock files remain durable |
| Rights and filesystem operations | `filesystem-operations-rights.json`, generated scripts and recorded SQLite prefix |
| Distinct capacities | `capacity-filesystem-bytes.json` versus `capacity-volume-bytes.json`, pending-operations and transitions scripts |

The fault engine increments `resource\0operation`, selects the first match, emits Go field order/escaping/omitempty plus newline, and returns errno 29/51, shortened writes or typed crash as appropriate. A nonshortening write returns `invalid` before event append. A transcript overflow returns `capacity:storage-transcript` before event append. SQLite `.lock`, `-shm` and `-shm/…` nodes have no storage path and stay outside the volume.

## Verification and scope

The task-3 handover `.flow/tmp/handovers/fn-176.3-evidence.json` records the exact canonical conformance and focused Go Quick commands passing on production source `6476da1cd1`; `git diff 6476da1cd1 ab39e8ec3c -- tools/gomad` is empty. Those source-identical results are reused under the conductor's instruction: 46 corpus scripts plus 4 malformed-image controls and 6 cross-directory recovery cases, and focused Go storage tests (64.153s). They are inherited observations, not commands re-executed by this task.

This task's pre-edit and post-edit Rust Quick command both exit 0 with 54 unit tests and 1 integration test. The four added cases exercise empty fault name, NUL resource, sync-error payload and no-space payload. Rust fmt, conformance-feature all-target clippy and configured fast lint pass. Logs and exact commands are in the task handover evidence. No production defect was found, so there is no claimed red-to-green defect reproduction.

Whole-spec frozen qualification, Temporal crash/reopen, workloads, and WASM SQLite selected-state/catalogue gates remain pending with the conductor. Native qualification remains deferred with its owners.
