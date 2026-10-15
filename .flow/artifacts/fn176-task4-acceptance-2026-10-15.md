# fn-176.4 storage fault and namespace acceptance

The implementation inherited at lifecycle baseline `ab39e8ec3c406c4181c293388929b0e06ce23168` is merge `cd2cf0c9f644345a0cdf5807dd5d8f4d52a5474d`, including original fault/config/routing commit `6318cad368` and review corrections `629e735ca6`. This task adds four missing validator predicate cases and fixes split-output accounting identified during integration review, with conductor authorization to perform this task .6 correction early.

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

This task's initial pre-edit and post-edit Rust Quick command both exit 0 with 54 unit tests and 1 integration test. The four added cases exercise empty fault name, NUL resource, sync-error payload and no-space payload. After the review correction, canonical source gates are re-executed; logs and exact commands are in the task handover evidence.

## Early task .6 correction: offered-output accounting

Integration review found that remote stdout output and subsequent writes to locally closed stdout used independent budgets. `TestStorageSplitOutputBudgetAfterClose` at failing-test commit `53320ca608` reproduces the missed capacity and a related Go-owner terminal-validation failure: Go counts rejected writes as offered bytes, while the remote ABI refunded their reservation.

The corrected ABI charges offered bytes even on errno or partial replies, preserving actual written counts and stream bytes separately. Split namespace and remote operations synchronize the same cumulative offered counter. A subsequent callback carries the pre-call total, which Go admits only for split-owned namespaces, within the output limit and without regression. Terminal totals cannot regress below the last Go observation or exceed the configured limit. Ordinary remote terminals retain exact equality with Go's counter. Descriptor routing remains based on descriptor identity; fd allocation is monotonic, and `fd_renumber` is unsupported, so no descriptor-reuse support is added.

The real-Run regression compares both owners for a closed write crossing the limit, a below-limit EBADF followed by stderr, and a later remote write crossing a budget already charged locally. It checks capacity/exit, offered totals, actual stdout/stderr, call counts, transcript bytes, storage event bytes, and the split remote callback count. Focused admission/terminal controls reject regressing and over-limit counters and counters outside split namespace ownership. Rust's former actual-written/refund expectations are intentionally updated to Go's documented offered-output contract, not weakened: partial/failed replies still assert their actual count/errno and buffer state.

Prior-fix checks found no matching memory entry or local recent correction; GitHub PR/issue lookups were unavailable (HTTP 401). There is no known-good split revision to bisect. The cheap failing regression was committed before the fix, and the same real-Run table passes after it. This is a library/helper surface with no live app route.

`Namespace::adopt_offered_output` asserts the internal monotonic/bounded invariant. Direct helper requests cannot inject a different namespace output limit: `engine.rs` passes execute `output_bytes` to `namespace_transport::read`, and `namespace_image::decode` rejects a different image output limit before constructing the namespace. The full Rust run includes `rejects_invalid_header_limits_and_expected_identity` and `rejects_digest_cwd_and_image_identity_mismatches`, which exercise this refusal through raw image and framed transport inputs respectively.

Whole-spec frozen qualification, Temporal crash/reopen, workloads, and WASM SQLite selected-state/catalogue gates remain pending with the conductor. Native qualification remains deferred with its owners.
