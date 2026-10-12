# Isolated WASM engine protocol

Build and test with `cargo +1.94.1 build --release --locked` and `cargo +1.94.1 test --locked`
in this directory. Rust 1.94.1 and Wasmtime 47.0.3 are pinned locally. No WASI
implementation or engine cache is linked. A helper handles one execution and exits.

Every frame is one UTF-8 JSON object followed by LF. Stdout carries protocol only.
Integers are JSON integers, not floating-point strings. Byte strings use standard
padded base64. Go supplies the sealed environment and owns entropy, descriptors
and files. The cooperative helper owns runtime choices, clocks and network
decisions; Rust also owns ABI decoding, bounds, memory writes and execution.
Guest pointers never cross the protocol. Stock imports use explicit callbacks,
except `proc_exit`.
No host arguments, environment, clocks, files, random source or sockets are inherited.

The first input is:

```json
{"schema":"gomad3.wasm-host/v1","type":"execute","module_path":"/absolute/module.wasm","module_sha256":"64 lower-case hex digits","memory_bytes":268435456,"fuel":100000000,"output_bytes":8388608}
```

The fields shown are required; optional `compile_cache` names an absolute
host-local compiled-module cache directory. Optional `local_model` selects
cooperative model execution. Extra fields are rejected. Memory is positive, at most
4 GiB and a multiple of 64 KiB. Fuel is positive. Output is positive and at most
1 GiB. Modules are bounded to 512 MiB. Input bytes are SHA-256 checked before
compilation. Binary WASM and textual WAT are admitted; the digest binds the exact
input bytes. Only one exported memory32 named `memory` and `_start: () -> ()`
are admitted. Start sections execute only after validation and `started`.

The helper emits `started` after compilation/validation, before instantiation:

```json
{"schema":"gomad3.wasm-host/v1","type":"started","engine":{"name":"wasmtime","version":"47.0.3","configuration":"cranelift-fuel-nan-canonical-v1","host_os":"macos","host_arch":"aarch64"},"module_sha256":"...","imports":[{"module":"wasi_snapshot_preview1","name":"fd_write","params":["i32","i32","i32","i32"],"results":["i32"]}],"exports":[{"name":"memory","kind":"memory"},{"name":"_start","kind":"func"}],"initial_memory_pages":1}
```

Import entries preserve module order and duplicate entries. Signatures must match
WASI Preview1 or the cooperative runtime ABI exactly. Unknown namespaces,
operations or signatures are unsupported before `started`. The 33 WASI operations
below and five runtime operations described below are admitted. Threaded/shared,
memory64 and multiple memories are unsupported by the engine configuration.

The host validates the engine/module/import identity and then sends exactly
`{"type":"authorize","module_sha256":"...","configuration":"cranelift-fuel-nan-canonical-v1"}`.
Both identity fields must match `started`; unknown fields, mismatches and EOF
terminate as `invalid` before instantiation or any guest start-section code.

`local_model` contains `seed`, numeric choice `mode` (record 1, replay 2, prefix 3),
`limit`, `diagnostics`, `tape_chunks`, `epoch_nanos`, `read_step_nanos`, `now`,
`digest_only`, and `limits` with `calls`, `transcript_bytes`, `pending_events`.
Optional `network` carries the seeded delay and fault schedule from the Go model.
After authorization, Go sends exactly `tape_chunks` frames of
`{"type":"tape","decisions":[...]}` using the canonical Go choice decision JSON
field names. Each frame remains bounded to 8 MiB and the aggregate decision count
must fit the choice limit. Replay with zero chunks has an empty tape.

The helper handles all runtime controls, clock reads, clock-only polls and yields
locally, maintaining the global callback ID sequence. Remote callback frames add
`local_state` with `now`, `advances`, and the total canonical `transcript_bytes`.
Go adopts this state before performing a remote operation. Replies add
`local_state` with `now` and `advances`, plus `transcript_base64` containing the
exact canonical Go call and reply JSON lines, without state metadata. The helper
hashes these bytes alongside its local WASI lines; runtime controls remain
excluded from the WASI transcript.
For canonical remote transcripts exceeding 64 KiB, Go first sends
`{"type":"remote_transcript","data_base64":"..."}` chunks of at most 64 KiB,
then a reply with empty `transcript_base64`. The aggregate is bounded to 8 MiB
and enters the helper transcript only after the matching reply is validated.
While the helper waits for a remote reply, Go may instead send
`{"type":"interrupt","termination":"storage_crash","message":"modeled storage crash: fault"}`.
Only typed `storage_crash`, `crash` and `stop` reasons are admitted; unknown
reasons, unknown fields and messages exceeding 64 KiB
are invalid. The helper stops before returning the syscall to the guest, flushes
partial evidence, and reports unfinished cooperative state. The interrupted
operation does not count as a completed call. This modeled outcome is distinct
from host watchdog termination.

Evidence is sent without replies in bounded frames
`{"type":"evidence","stream":"choice","data_base64":"..."}`. Streams are
`choice` (raw 96-byte canonical records), `diagnostic` (raw 96-byte samples),
`network` (canonical JSON lines), and `transcript` (canonical WASI JSON lines).
Chunks contain at most 64 KiB decoded bytes and may split a record; each stream
is concatenated in frame order. The helper retains at most one partial chunk per
stream between imports and flushes every remaining chunk before the result.
Digest-only execution hashes the complete WASI transcript without sending its
bytes; choice and network capacity bounds remain active.

Each callback emits `{"type":"call","id":1,"op":"fd_write","input":{...}}`.
IDs increase by one, starting at one. The host sends exactly one corresponding
`{"type":"reply","id":1,"errno":0,"output":{...}}`. Optional `unsupported`
(boolean, default false) and `message` (string, default empty) may be added.
Unknown frame fields, mismatched IDs, invalid output layouts, EOF, and malformed
JSON terminate as `invalid`. An unsupported reply terminates as `unsupported`;
its message identifies the rejected boundary. Nonzero errno (0..65535) writes no
output memory. Successful output objects must have exactly the listed fields.

| Operation | Call input | Successful reply output |
| --- | --- | --- |
| args_sizes_get, environ_sizes_get | `{}` | `{"count":u32,"bytes":u32}` |
| args_get, environ_get | `{}` | `{"values":[base64]}`; each byte string excludes NUL |
| random_get | `{"length":u32}` | `{"data_base64":base64}` of exactly length |
| clock_time_get | `{"clock_id":u32,"precision":u64}` | `{"timestamp":u64}` |
| sched_yield | `{}` | `{}` |
| fd_write | `{"fd":u32,"data_base64":base64}` | `{"written":u32}` no larger than data |
| fd_pwrite | previous plus `"offset":u64` | same |
| fd_read | `{"fd":u32,"length":u32}` | `{"data_base64":base64}` no larger than length |
| fd_pread | previous plus `"offset":u64` | same |
| fd_close, fd_sync | `{"fd":u32}` | `{}` |
| fd_filestat_set_size | `{"fd":u32,"size":u64}` | `{}` |
| fd_fdstat_set_flags | `{"fd":u32,"flags":u16}` | `{}` |
| fd_seek | `{"fd":u32,"offset":i64,"whence":u8}` | `{"offset":u64}` |
| fd_fdstat_get | `{"fd":u32}` | `{"filetype":u8,"flags":u16,"rights_base":u64,"rights_inheriting":u64}` |
| fd_filestat_get | `{"fd":u32}` | filestat below |
| fd_prestat_get | `{"fd":u32}` | `{"name_length":u32}` (directory preopen) |
| fd_prestat_dir_name | `{"fd":u32,"length":u32}` | `{"data_base64":base64}` of exactly length |
| fd_readdir | `{"fd":u32,"length":u32,"cookie":u64}` | `{"data_base64":base64}` no larger than length; encoded Preview1 dirents |
| path_create_directory, path_remove_directory, path_unlink_file | `{"fd":u32,"path_base64":base64}` | `{}` |
| path_filestat_get | previous plus `"flags":u32` | filestat below |
| path_open | previous plus `"dirflags":u32,"oflags":u16,"rights_base":u64,"rights_inheriting":u64,"fdflags":u16` | `{"fd":u32}` |
| path_readlink | path input plus `"length":u32` | `{"data_base64":base64}` no larger than length |
| path_rename | `{"fd":u32,"path_base64":base64,"new_fd":u32,"new_path_base64":base64}` | `{}` |
| path_symlink | `{"old_path_base64":base64,"fd":u32,"path_base64":base64}` | `{}` |
| sock_accept | `{"fd":u32,"flags":u16}` | `{"fd":u32}` |
| sock_shutdown | `{"fd":u32,"how":u8}` | `{}` |
| poll_oneoff | `{"subscriptions":[subscription]}` | `{"events":[event]}`; at most input count |
| proc_exit | handled locally, no callback | result `exit`, guest exit code |

Filestat is `{"dev":u64,"ino":u64,"filetype":u8,"nlink":u64,"size":u64,"atim":u64,"mtim":u64,"ctim":u64}`.
Subscription is `{"userdata":u64,"type":u8,"clock_id":u32,"timeout":u64,"precision":u64,"flags":u16}` for type 0, or
`{"userdata":u64,"type":u8,"fd":u32}` for type 1/2.
Event is `{"userdata":u64,"errno":u16,"type":u8,"nbytes":u64,"flags":u16}`.
Go must validate event correlation/readiness and implement descriptor rights/path policy.
Rust validates the complete ABI input and output range before calling Go or writing
reply data, including zero-length pointers, iovec counts and checked aggregate sizes.

The `gomad_wasm_v1` namespace admits only `config`, `decision`, `observation`,
`finish`, `idle` and `net`, each with signature `(i32 pointer, i32 length) -> i32 errno`.
Go requires the profile's imports before authorizing a cooperative profile. The guest
uses bounded static buffers, including during runtime startup and nosplit paths.
Rust validates the entire memory range and operation-specific size before copying.
The protocol callback names are `runtime_config`, `runtime_decision`,
`runtime_observation`, `runtime_finish` and `runtime_idle`; input and successful
reply are both `{"data_base64":base64}` of exactly the admitted length. Rust
checks encoded reply size before decoding and writes no memory on nonzero errno.

Config, finish and idle buffers contain 16 bytes. Config bootstraps ABI version 1
and the runtime seed. Idle carries version, reserved zero bytes and the next
absolute deadline; the host returns the modeled clock position. Decision buffers
contain a 32-byte header and 2..256 stable 32-byte alternatives, optionally followed
by a 96-byte diagnostic sample. The host returns the selected physical rank after
canonical choice/replay validation. Observation buffers contain one 96-byte choice
record and optionally one 96-byte diagnostic sample. Oversized, truncated, unknown
version or malformed semantic payloads fail closed. Finish marks complete tape
consumption. Runtime control does not enter the application WASI evidence bytes.
The `gomad_wasm_v2` namespace admits `net_stream` with the same function
signature. It forwards `runtime_net_stream` to the Go owner using an in-place
buffer of 64 through 65,600 bytes; the v2 network profile uses remote model
execution.

The final frame is `{"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":123,"peak_memory_bytes":65536,"output_bytes":2}`.
Terminations are `exit`, `capacity`, `unsupported`, `trap`, `invalid`, and
`infrastructure`, plus `divergence` for cooperative forced-choice mismatch.
`storage_crash` reports a validated modeled-storage interrupt; `crash` and `stop`
report validated host-Broker interruptions. Graceful interrupts retain actual
measured phases, fuel, peak guest memory and bounded partial owner evidence;
they do not fabricate a completed trace or turn watchdog cancellation into an
owner-observed terminal outcome.
Cooperative results add `local_model` with the final clock, advances, global calls,
record count, tape cursor, configured/finished state, peak goroutines, elapsed
virtual time, raw runtime finish bytes, transcript and network byte counts, and
the lowercase hexadecimal full-transcript SHA-256. Divergence results add
`divergence`, preserving the Go choice `Ordinal`, `Reason`, `Expected`, `Observed`
and `TapeRecords` fields. Exit code is present only for `exit`. Invalid input, unsupported
imports, initial/grown memory limits, fuel, buffer, callback and output limits have
separate explicit results. Host resource failures remain infrastructure outcomes;
limits do not assert memory backing availability or a host peak-RSS bound.
`peak_memory_bytes` is the maximum limiter-approved requested guest linear-memory
size. It may exceed committed memory after allocation failure; it supplies no
host RSS or backing-availability guarantee.
The helper exits 0 when it successfully emits a result; transport-write failure exits 1.

Results after module preparation include informational `metrics`, measured only
by host clocks and resource counters. `prepare_nanos` covers module read/hash,
engine configuration, compilation/cache and import/export validation.
`compile_nanos` covers Wasmtime's binary/text builder and Cranelift compilation.
`cache_load_nanos` measures cache identity, locking, bounded envelope read,
verification and `Module::deserialize` from the verified private buffer;
`cache_hit` distinguishes a successful load from compilation after a miss.
`cache_write_nanos` covers serialization and cache publication on a miss.
`instantiate_nanos` measures linker instantiation, including any Wasm start
section; `execute_nanos` measures only the exported `_start` invocation.
`process_user_nanos`, `process_system_nanos` and `peak_rss_bytes` come from
`getrusage(RUSAGE_SELF)` on 64-bit x86-64/ARM64 Linux and macOS. Linux's KiB RSS
is converted to bytes; macOS supplies bytes. `resource_usage_available` is false
with zero resource fields when the platform or observation is unsupported.
These measurements never enter modeled time, control decisions or canonical
evidence and do not establish cross-host timing or memory guarantees.

Compiled-code cache v2 is host-owned, guest-inaccessible storage: the configured
directory must be owned by the helper's effective UID and mode 0700; it cannot
itself be a symlink. Entries and per-key locks must be single-link regular files
owned by that UID and mode 0600. A directory descriptor anchors all no-follow
entry opens and publication renames; a per-key OS file lock serializes compilers.
The same-UID host owner is trusted. Checksums detect corruption, not malicious
artifacts forged by that owner, who already controls the host executable.

The key is SHA-256 of exact JSON identity bytes binding guest source SHA-256,
full helper executable SHA-256, Wasmtime version, configuration revision,
host OS/architecture and engine precompile-compatibility hash. The helper digest
binds its compiled Rust runtime model; the guest digest binds the embedded guest
profile. Go callback model state/profile and execution limits are not immutable
compilation inputs and cannot affect this artifact. Each filename ends in
`.gomad-cache-v2`. Its single-file envelope is `GOMADC02`, u32 little-endian
manifest length, u64 little-endian artifact length, 32-byte artifact SHA-256,
exact JSON manifest and serialized artifact. Manifest bytes are limited to
16 KiB and artifact bytes to 1 GiB, with checked fallible reservation before
reading. Bad identities, corruption, unsafe file identities or stale envelopes
terminate before `started`; they never silently compile around a poisoned entry.
Only a missing v2 entry compiles. Legacy `.cwasm` files are ignored and preserved.

Loading uses an exact verified private buffer, not a reopened path or mutable
file-backed executable mapping. This temporarily holds the entire artifact in
addition to Wasmtime's deserialized code, increasing peak RSS versus the old
mapped cache (Temporal artifacts can exceed 500 MiB). The allocation bound is
not a promise that the host can back that allocation. Stores and guest heaps
are always newly instantiated, never cached.

Result diagnostics are bounded to 64 KiB, with an explicit truncation marker.
Every frame is at most 8 MiB including LF. Each transferred/ABI buffer is at most
4 MiB, iovecs at most 4096, poll subscriptions at most 4096, and executions at most
2^28 callbacks (the cooperative environment may impose a lower bound). Guest stdout/stderr bytes (fd 1/2 writes) consume `output_bytes`
before callback delivery (including positional writes); overrun terminates without silently truncating output.
Fuel counts engine work and supplies no goroutine preemption or virtual-time advance.
A stalled callback is bounded by the Go adapter's process watchdog/cancellation.
The configuration revision binds Cranelift compilation, fuel enabled, canonical
NaNs, relaxed SIMD/memory64/multiple memories disabled, no WASI/cache/threads
features, and the fixed frame/buffer/iovec/subscription/callback/module limits.
Wasmtime exposes its existing wasmparser 0.252.0 API for validation-error
classification. Exact pinned disabled-feature messages classify memory64, shared
memory and atomic instructions, multiple memories and relaxed SIMD as unsupported. Malformed text, binary
and function bodies remain invalid. Function translation erases the parser type;
the classifier accepts only its anchored `Invalid input WebAssembly code at
offset <digits>: <message>` root-cause form. Engine/config/compiler/mapping and
module read failures remain infrastructure outcomes. Protocol frame capacity
(including `started` inventories) remains capacity; transport write failures are
infrastructure failures. Only typed Wasmtime traps are guest traps; untyped
instantiation/execution errors are infrastructure failures. Submitted fd 1/2 write bytes consume the output bound,
including attempts receiving an errno or a short write.
Tables are bounded to eight and 1,000,000 elements per table; one store instance
is admitted. Updating these settings requires a new configuration revision.
The helper transports bounded scheduling hooks; the Go runtime/profile owns their
semantics. It does not claim server startup, native qualification or portable
exact replay beyond the retained cooperative qualification profile.

## Separate diagnostic builds

The non-default Cargo feature `diagnostic-profile` compiles bounded attribution
hooks. Normal builds contain no profiler module, clock sampling hooks, counters
or diagnostic reporter. A diagnostic build must use a separate target directory
and immutable helper path; never install it over a qualification helper. Source
metadata can change a rebuilt default binary's digest even with hooks excluded;
hash each actual artifact rather than assuming historical byte identity.

A diagnostic helper writes exactly one LF-terminated stderr line prefixed
`gomad-wasm-profile-v1 `, followed by JSON schema `gomad3.wasm-profile/v1`. The
whole line is strictly smaller than 16 KiB. It adds no stdout protocol frame,
terminal fields, guest fd2 data, configuration input or canonical evidence.
Missing, duplicate, malformed or additional stderr remains an inconclusive
diagnostic at the consuming driver; an abruptly killed process may emit none.
The reporter samples actual `Instant` wall time and the existing platform
`getrusage(RUSAGE_SELF)` implementation, without new dependencies or modeled-time
feedback. Its engine attribution is `cranelift-jit`, not an interpreter.

The fixed `phases` map contains `module_read`, `module_hash`, `engine_init`,
`cache_identity`, `cache_read`, `envelope_verify`, `deserialize`, `compile`,
`cache_write`, `instantiate` and `execute`. Each entry has `count`, `wall_nanos`,
`user_nanos`, `system_nanos`, `cpu_samples` and `peak_rss_bytes_high_water`.
CPU values are paired before/after process CPU deltas, not whole-process totals
relabelled as a phase. Missing or regressing CPU samples do not count as observed
CPU; RSS is the cumulative process high-water mark at the phase boundary, not
memory allocated solely by that phase. Cache read includes validation and buffer
allocation/read; envelope verification includes exact identity/length checks and
artifact checksum; deserialization starts only after those checks succeed.
Cache identity includes hashing the actual helper executable. Lock/setup costs
outside those phase scopes are not silently attributed to deserialization.

The disjoint full-import `abi` buckets are `clock`, `scheduler`, `select`,
`observation`, `idle`, `other_local` and `remote_callback`; failed attempts are
included. A remote-routed import is charged to the remote bucket for its whole
ABI span. Entries contain `count`, `wall_nanos`, `instantiate_wall_nanos` and
`execute_wall_nanos`; nested ABI attempts are rejected from double accounting
and counted in `abi_overlap_errors`. Runtime decision kind 1 denotes scheduler,
kind 2 ready-select; `poll_oneoff` also uses select. The `residual` map subtracts
only disjoint ABI wall spans from the corresponding instantiate/execute wall
phase. That residual includes unmeasured Wasmtime/trampoline work and profiling
overhead. It is not guest CPU, nor does subtraction identify a cause.

The separately nested `alternative_hash`, `encoding`, `transcript_hash`, `flush`,
`remote_write` and `remote_read_wait` entries contain only `count` and
`wall_nanos`. They overlap ABI and phase scopes and must never be summed as
independent work. Encoding covers sampled codec, bounded copy and ABI memory
projection spans, not every instruction that handles bytes. Remote write covers
only frame write/flush after JSON serialization; read wait covers bounded frame
reading before JSON decoding and may include several transcript chunks per
callback. Flush includes evidence encoding/writing. Fixed counters saturate;
there are no per-import diagnostic logs or unbounded operation labels.

The ignored `diagnostic_calibration_preserves_model_bytes_without_timing_feedback`
test compares three paired direct-model loops with an active profiler against the
same diagnostic build with no active session. Inactive cfg hooks still check for
a session, so this is not a feature-off binary timing control. It asserts exact
concatenated evidence and metadata while retaining actual loop wall/CPU samples.
It excludes Wasmtime guest execution, import trampolines, IPC, final flush and
byte comparison. Calibration and any actual workload diagnostic require an
exclusive owner-approved measurement window and are never qualification gates.
