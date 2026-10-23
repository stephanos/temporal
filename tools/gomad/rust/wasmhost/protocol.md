# Isolated WASM engine protocol

Build and test with `cargo +1.99.1 build --release --locked` and `cargo +1.99.1 test --locked`
in this directory. Rust 1.99.1 and Wasmtime 47.0.3 are pinned locally. No WASI
implementation or engine cache is linked. A helper handles one `execute` and exits,
or hosts a `cluster_execute` session until its explicit `finish` control.

The helper transcript corpus in `tests/testdata/helper_transcript` (schema
`gomad.helper-transcript-corpus/v1`) pins this protocol end to end. A case holds
`intent`, `replaces`, a `guest` (`wat`, `path` under the corpus directory, or
`wat_parts` with `{"repeat":N,"text":...}`), the first `request` frame, initial
`inputs`, `replies` (each sends its frames once the next emitted frame matches
its `after` subset pattern), a `reply_policy` (`fail`, the default, or `deny`)
for other `call`/`sync` frames, an optional `close_input` (`true` or
`"after_replies"`), and `expected`. Sent strings `$guest_path`, `$guest_sha256`
and `$trigger.<path>` resolve to the guest and the triggering frame; tagged
frames put `instance` first. `expected` is the frame stream, termination,
per-stream evidence sizes and SHA-256 (plus the text of the storage log and
guest stdout/stderr), helper stderr and exit status. Host
telemetry (`metrics`, cluster `scheduler`) keeps only zero, `nonzero` or
`measured`, and `engine.host_os`/`host_arch` become `$host_os`/`$host_arch`.
The guest path always has the same length, because refusals quote request
columns. `tests/helper_corpus.rs` and the Go `wasi` test `TestHelperCorpus`
run each case twice (ten times when regenerating) and refuse output that
varies between runs. A case listing `builds` (`diagnostic-profile`,
`diagnostic-fuel-off`, `guest-profile`) also runs on those builds
(`make test-helper`); `expected_builds.<build>` replaces the top-level
expected fields that build changes, such as its projected `diagnostic_profile`
stderr report. A `derived` case embeds inputs taken from another case's
expected evidence (`storage_image`, or the storage state's `crash_context`
with JSON-pointer `set` edits and optional `select`); the Rust loader refuses
stale copies and rebuilds them on regeneration. A Rust test checks the
relations between committed cases that single cases cannot state (equal
split-revocation storage, the image identity bound in storage state, the
recovered `/db`, the namespace transcript records). After a reviewed behavior
change, rerun `GOMAD_HELPER_CORPUS_REGENERATE=1 cargo +1.99.1 test --release
--locked --test helper_corpus`, and the same with `--features diagnostic-profile`,
`--features diagnostic-fuel-off` and `--features guest-profile` in a separate
target directory; the corpus diff is the approval.

Every frame is one UTF-8 JSON object followed by LF. Stdout carries protocol only.
Integers are JSON integers, not floating-point strings. Byte strings use standard
padded base64. Go supplies the sealed environment. Go owns descriptors and files
except in helper-owned namespace runs (see "Helper-owned namespace").
The cooperative helper owns runtime choices, clocks, captured entropy, arguments
and environment; Rust also owns ABI decoding, bounds, memory writes and execution.
Guest pointers never cross the protocol. Stock imports use explicit callbacks,
except `proc_exit`.
No host arguments, environment, clocks, files, random source or sockets are inherited.

The first input is:

```json
{"schema":"gomad.wasm-host/v1","type":"execute","module_path":"/absolute/module.wasm","module_sha256":"64 lower-case hex digits","memory_bytes":268435456,"fuel":100000000,"output_bytes":8388608}
```

The fields shown are required; optional `compile_cache` names an absolute
host-local compiled-module cache directory. Optional `local_model` selects
cooperative model execution. Extra fields are rejected. Memory is positive, at most
4 GiB and a multiple of 64 KiB. Fuel is positive. Output is positive and at most
1 GiB. Modules are bounded to 512 MiB. On a cache miss or uncached run, input
bytes are read and SHA-256 checked before compilation. A cache hit loads the
artifact for the claimed digest without opening the module path. Binary WASM
and textual WAT are admitted; the digest binds the exact
input bytes. Only one exported memory32 named `memory` and `_start: () -> ()`
are admitted. Start sections execute only after validation and `started`.

The helper emits `started` after compilation/validation, before instantiation:

```json
{"schema":"gomad.wasm-host/v1","type":"started","engine":{"name":"wasmtime","version":"47.0.3","configuration":"cranelift-fuel-nan-canonical-nounwind-v2","host_os":"macos","host_arch":"aarch64"},"module_sha256":"...","imports":[{"module":"wasi_snapshot_preview1","name":"fd_write","params":["i32","i32","i32","i32"],"results":["i32"]}],"exports":[{"name":"memory","kind":"memory"},{"name":"_start","kind":"func"}],"initial_memory_pages":1}
```

Import entries preserve module order and duplicate entries. Signatures must match
WASI Preview1 or the cooperative runtime ABI exactly. Unknown namespaces,
operations or signatures are unsupported before `started`. The 33 WASI operations
below and five runtime operations described below are admitted. Threaded/shared,
memory64 and multiple memories are unsupported by the engine configuration.

The host validates the engine/module/import identity and then sends exactly
`{"type":"authorize","module_sha256":"...","configuration":"cranelift-fuel-nan-canonical-nounwind-v2"}`.
Both identity fields must match `started`; unknown fields, mismatches and EOF
terminate as `invalid` before instantiation or any guest start-section code.

`local_model` contains `seed`, numeric choice `mode` (record 1, replay 2, prefix 3),
`limit`, `diagnostics`, `tape_chunks`, `epoch_nanos`, `read_step_nanos`, `now`,
`digest_only`, and `limits` with `calls`, `transcript_bytes`, `pending_events`.
It also requires `entropy_key` (32 unsigned bytes),
`working_directory_sha256` (64 lowercase hexadecimal digits), and `arguments`
and `environment` declarations, each containing `bytes`, `count`, and `chunks`.
Each stream is the captured UTF-8 strings in order, each followed by NUL, with
the existing 4 MiB aggregate string-byte bound. `chunks` must equal
`ceil(bytes / 65536)` and `count` must match the reconstructed entries.
Before compilation or `started`, Go sends arguments followed by environment:
`{"type":"model_input","stream":"arguments","sequence":0,"data_base64":"..."}`.
Sequences start at zero independently for each stream. Raw chunks are exactly
64 KiB except the declared final remainder; entries may span chunks.
`{"type":"model_inputs_end"}` must follow, including for empty streams.
Unknown fields, missing/reordered/extra chunks, invalid base64/UTF-8/framing,
duplicate or empty environment names, and mismatched totals fail before `started`.
The reconstructed `PWD` must be clean and absolute and its SHA-256 must match
the declared working-directory identity. No ambient directory is consulted.
Fragmentation preserves valid single large control-character strings without
raising the 8 MiB frame bound or introducing another transcript format.
After authorization, Go sends exactly `tape_chunks` frames of
`{"type":"tape","decisions":[...]}` using the canonical Go choice decision JSON
field names. Each frame remains bounded to 8 MiB and the aggregate decision count
must fit the choice limit. Replay with zero chunks has an empty tape.

The helper handles all runtime controls, clock reads, clock-only polls, yields,
`random_get`, and the four captured argument/environment imports
locally (and, in helper-owned namespace runs, every filesystem import and poll), maintaining the global callback ID sequence. Remote callback frames add
`local_state` with `now`, `advances`, and the total binary WASI `transcript_bytes`.
Go adopts this state before performing a remote operation. Replies add
`local_state` with `now` and `advances`, plus `transcript_base64` containing the
exact binary WASI records, without state metadata. The helper
hashes these bytes alongside its local WASI records; runtime controls remain
excluded from the WASI transcript.
For binary remote transcript records exceeding 64 KiB, Go first sends
`{"type":"remote_transcript","data_base64":"..."}` chunks of at most 64 KiB,
then a reply with empty `transcript_base64`. The aggregate is bounded to 8 MiB
and enters the helper transcript only after the matching reply is validated.
While the helper waits for a remote reply, Go may instead send
`{"type":"interrupt","termination":"storage_crash","message":"modeled storage crash: fault"}`.
Only typed `storage_crash`, `crash` and `stop` reasons are admitted; unknown
reasons, unknown fields and messages exceeding 64 KiB
are invalid. Go sends `storage_crash` only in runs that keep Go's volume; the
helper originates it when it owns the volume (see "Helper-owned storage"). The helper stops before returning the syscall to the guest, flushes
partial evidence, and reports unfinished cooperative state. The interrupted
operation does not count as a completed call. This modeled outcome is distinct
from host watchdog termination.

Evidence is sent without replies in bounded frames
`{"type":"evidence","stream":"choice","data_base64":"..."}`. Streams are
`choice` (raw 96-byte canonical records), `diagnostic` (raw 96-byte samples),
`transcript` (binary WASI records described below), and, in helper-owned
namespace runs only, `stdout` and `stderr` (raw guest fd 1/2 bytes).
Chunks contain at most 64 KiB decoded bytes and may split a record; each stream
is concatenated in frame order. The helper retains at most one partial chunk per
stream between imports and flushes every remaining chunk before the result.
Digest-only execution hashes the complete WASI transcript without sending its
bytes; choice capacity bounds remain active.

Each callback emits `{"type":"call","id":1,"op":"fd_write","input":{...}}`.
IDs increase by one, starting at one. The host sends exactly one corresponding
`{"type":"reply","id":1,"errno":0,"output":{...}}`. Optional `unsupported`
(boolean, default false) and `message` (string, default empty) may be added.
Unknown frame fields, mismatched IDs, invalid output layouts, EOF, and malformed
JSON terminate as `invalid`. An unsupported reply terminates as `unsupported`;
its message identifies the rejected boundary. Nonzero errno (0..65535) writes no
output memory. Successful output objects must have exactly the listed fields.

## WASI transcript v1

Every profile uses the same binary stream. An empty execution has an empty
stream; the first recorded operation emits the eight bytes `GMWT 01 00 00 00`.
All integers below are little endian. Each record has a 16-byte header:
`tag:u8, flags:u8, errno:u16, call_id:u64, payload_bytes:u32`. Flag 1 means
unsupported; other flag bits are invalid. IDs are positive and strictly
increasing; runtime controls consume IDs but emit no WASI record.

Payloads concatenate typed input fields followed by typed output fields in the
order shown in the operation table below. Nonzero errno omits output. An
unsupported record instead appends `message_bytes:u32` and UTF-8 message bytes
(at most 64 KiB). Byte fields use `length:u32` followed by raw bytes, bounded to
4 MiB. Arrays use `count:u32`, bounded to 4096, followed by typed elements.
Subscriptions contain userdata:u64, type:u8, then clock_id:u32, timeout:u64,
precision:u64, flags:u16 for clocks or fd:u32 for descriptors. Events contain
userdata:u64, errno:u16, type:u8, nbytes:u64, flags:u16. There is no alignment
padding. A record including its header is bounded to 8 MiB.

Tags 1–32, in order, are clock_time_get, sched_yield, args_sizes_get, args_get,
environ_sizes_get, environ_get, random_get, fd_write, fd_pwrite, fd_read, fd_pread,
fd_close, fd_sync, fd_filestat_get, fd_fdstat_get, fd_prestat_get,
fd_prestat_dir_name, fd_fdstat_set_flags, fd_filestat_set_size, fd_seek, fd_readdir,
path_create_directory, path_remove_directory, path_unlink_file, path_filestat_get,
path_open, path_readlink, path_rename, path_symlink, sock_accept, sock_shutdown,
poll_oneoff. Tag 1 records unsupported CPU-clock reads. Successful reads use
tag 33: clock_id:u32, precision:u64, starting model now:u64, count:u64. Its header
ID is the first call; count is positive and at most 2^28. Consecutive reads fold
only with equal clock ID/precision, consecutive call IDs and the expected model
position under the identity-bound read step. Epoch/read-step configuration
derives each returned timestamp; a frozen clock still binds count. Runtime
controls, other records, remote callbacks/splices, interrupts and terminal
flush close the run. Pending records count toward retained capacity once.
Inspection projects pending bytes/digests without ending the run. The binary path_open input orders
fd, dirflags, path, oflags, rights_base, rights_inheriting, fdflags; path_filestat_get
orders fd, flags, path. Derived args/environ records have no payload; random_get
has length:u32 and starting entropy offset:u64, without output bytes. Those
results follow from identity-bound request inputs.

Tag 34 binds a rejected operation accepted by the Go callback owner: operation
name length:u32 and UTF-8 name bytes (zero to 64 KiB, preserving the Go owner's
empty-operation unsupported result), raw input length:u32 and
the exact already-validated JSON input (at most 8 MiB), then the unsupported
message. It is always unsupported. The strict decoder applies the existing
32-level strict JSON syntax check to that raw input and rejects use of this tag
for a known typed operation. This preserves direct Go unknown-operation and JSON
validation fixtures; unknown guest imports still fail before `started`. Rust's
local path never emits this tag. It is part of this binary stream, with no
format detection or legacy transcript fallback.

Each language has one record writer. Retained capacity reserves the header when
needed, the exact input size and the operation's largest supported output before
model mutation; digest-only runs retain none. The Go decoder rejects unknown
tags/flags, truncation, payload trailing bytes, invalid array elements, oversized
lengths/counts and non-increasing IDs. Remote records splice unchanged, with a
stream header only when no record preceded them. The record bytes, including the
header, are the SHA-256 digest input.

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
When Go owns the namespace it validates event correlation/readiness and implements
descriptor rights/path policy; in helper-owned namespace runs the Rust namespace does.
Rust validates the complete ABI input and output range before calling Go or writing
reply data, including zero-length pointers, iovec counts and checked aggregate sizes.

The `gomad_wasm_v1` namespace admits only `config`, `decision`, `observation`,
`finish` and `idle`, each with signature `(i32 pointer, i32 length) -> i32 errno`.
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
virtual time, raw runtime finish bytes, the transcript byte count, and
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
by host clocks and resource counters. `prepare_nanos` covers module read/hash on misses,
engine configuration, compilation/cache and import/export validation.
`compile_nanos` covers Wasmtime's binary/text builder and Cranelift compilation.
`cache_load_nanos` measures cache identity, locking, bounded envelope read,
verification and `Module::deserialize` from the verified private buffer;
`cache_hit` distinguishes a successful load from compilation after a miss.
`cache_write_nanos` covers serialization and cache publication on a miss.
`instantiate_nanos` measures linker instantiation, including any Wasm start
section; `execute_nanos` measures only the exported `_start` invocation.
`process_user_nanos` and `process_system_nanos` come from
`getrusage(RUSAGE_SELF)` on 64-bit x86-64/ARM64 Linux and macOS. Linux
`peak_rss_bytes` uses `/proc/self/status`'s `VmHWM`, converted from KiB, to
exclude the pre-exec image's inherited high water; macOS uses the byte-valued
`getrusage` peak. If Linux cannot read a valid `VmHWM`, telemetry falls back to
the `getrusage` peak, which can include inherited high water. Cluster results report these process counters once for all
stores and boot templates, rather than attributing them to each incarnation.
`resource_usage_available` is false
with zero resource fields when the platform or observation is unsupported.
These measurements never enter modeled time, control decisions or canonical
evidence and do not establish cross-host timing or memory guarantees.

Compiled-code cache v3 is host-owned, guest-inaccessible storage: the configured
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
`.gomad-cache-v3`. Its single-file envelope is `GOMADC03`, u32 little-endian
manifest length, u64 little-endian artifact length, u32 little-endian artifact CRC-32,
exact JSON manifest and serialized artifact. Manifest bytes are limited to
16 KiB and artifact bytes to 1 GiB, with checked fallible reservation before
reading. Bad identities, corruption, unsafe file identities or stale envelopes
terminate before `started`; they never silently compile around a poisoned entry.
Only a missing v3 entry reads and verifies module bytes, then compiles.
A miss with a digest mismatch terminates as `invalid` before publication;
a missing module on a miss is `infrastructure`. Module files are unused on hits.
Earlier `.gomad-cache-v2` and legacy `.cwasm` files are ignored and preserved.

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
The counter charges offered bytes even when the callback returns an errno or a
short write; the returned written count and output stream retain actual bytes.
Fuel counts engine work and supplies no goroutine preemption or virtual-time advance.
A stalled callback is bounded by the Go adapter's process watchdog/cancellation.
The configuration revision binds Cranelift compilation, fuel enabled, canonical
NaNs, native unwind info disabled, relaxed SIMD/memory64/multiple memories disabled, no WASI/cache/threads
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
Tables are bounded to eight and 1,000,000 elements per table. One Wasmtime instance
is admitted per store; a cluster creates separate stores under those same limits.
Updating engine settings requires a new configuration revision.
The helper transports bounded scheduling hooks; the Go runtime/profile owns their
semantics. It does not claim server startup, host-platform qualification or portable
exact replay beyond the retained cooperative qualification profile.

## Pure namespace image admission

Optional execute field `namespace` contains `chunks`, `files`, `bytes`, and
`sha256` (64 lowercase hexadecimal digits). It requires `local_model`; Go sends
it only for helper-owned or split-owned execution. Its presence makes the
helper the namespace owner for the whole run (see "Helper-owned namespace").

After authorization and all tape frames, Go sends `namespace_input` frames with
`sequence` starting at zero and `data_base64`, then `namespace_inputs_end`.
Chunks are exactly 64 KiB except the declared final remainder. `bytes` and SHA-256
bind the complete binary image; `files` includes the implicit root, excludes
standard descriptor nodes, and matches the encoded entry count plus one.
Malformed descriptors, chunks, digests or image contents return `invalid`
before Store creation and guest instantiation, with zero modeled activity.

Image v1 starts with `GMWN 01 00 00 00`. All scalar integers are little-endian
u64; blobs are u64 length followed by exact bytes. The header contains file,
descriptor, filesystem-byte and output-byte limits, epoch nanoseconds, then
entry, writable-directory, captured-root and preopen counts. Captured roots,
writable directories and preopens follow as path blobs, then working-directory
and stdin blobs. Each entry contains kind u8 (directory 1, regular file 2),
readonly u8 (0 or 1), path blob and data blob. Entries preserve Go inode creation
order and omit the implicit root. A zero byte ends the image, with exact EOF.

Captured roots and preopens are sorted and unique; writable directories retain
configuration order. Captured roots cannot overlap. Captured descendants remain
readonly; writable roots may contain captured roots but cannot lie beneath them.
Preopens equal the sorted mount union. Parents precede children, directory data
is empty, and the working directory exists. Data plus stdin obey the original
filesystem-byte budget; metadata has no additional path or aggregate-byte cap.
Decoding allocates incrementally from actual bytes, not declared blob lengths.

## Helper-owned namespace

The helper owns the namespace exactly when it owns the cooperative model
(cooperative profile, no network v2, no broker, not the test-only Go-owner
reference), with or without a storage model, and in broker storage runs (see
"Broker split ownership"). Stock-profile, non-storage broker and network-v2 runs
keep Go's namespace. Ownership never changes during a run.

In owned runs the helper builds a Rust namespace from the image and answers every
`fd_*` and `path_*` import and every `poll_oneoff` (clock, descriptor or mixed)
locally; none of these becomes a callback. Go refuses such a callback as
`invalid` before adopting helper state. Local operations consume global call IDs
and admit in Go's order before mutation: call limit, the callback and reply frame
reservations, then transcript capacity. They write the same binary records the Go
owner writes, through the same recorder as other local imports. Capacity and
invalid outcomes keep Go's `WASI <kind>: <dimension>` messages. Unsupported
operations or flags terminate `unsupported` with `<op>: <Go message>`, and their
record is retained. File times come from the model clock.

Guest fd 1/2 bytes return in the `stdout` and `stderr` evidence streams. The
helper flushes them when 64 KiB accumulate, before every remote callback and at
termination, including interrupts, traps and capacity ends. Go fills the result's
stdout/stderr from these streams, mirrors stdout chunks as they arrive (best
effort), and bounds the combined stream bytes by `output_bytes`
(`offered-output-bytes`). The terminal `output_bytes` is the namespace's offered
output, which charges rejected fd 1/2 writes as Go's namespace does; Go requires
it to cover the received stream bytes. After a watchdog kill or helper crash the
result holds the chunks received so far. Go reports per-operation counts of the
filesystem calls its own namespace served; owned runs report an empty map.

## Helper-owned storage

With a storage model the helper also owns the volume model (fn-176), a port of
the Go volume model `internal/model/volume` plus the WASI fault engine.
Optional execute field `storage` requires `namespace` and contains `model` (Go's
`StorageModel` JSON), `model_identity` (`sha256:` plus the hexadecimal SHA-256
of Go's `json.Marshal(StorageModel)`; the helper recomputes it and refuses a
mismatch), optional `crash_limits` (Go `CrashEnumerationLimits`) and
`crash_selection` (volume ID to operation IDs; both owners refuse a selection
naming an unconfigured volume as `invalid` before execution), optional
`crash_context`, `return_image`,
and an optional
recovery `image` descriptor with `chunks`, `bytes` and `sha256`. A recovered run
sends `storage_image_input` frames (`sequence`, `data_base64`, exactly 64 KiB
except the final remainder) after `namespace_inputs_end`, then
`storage_image_inputs_end`. Before Store creation the helper re-validates the
model with Go's rules (bounds, writable mounts, capacities, the fault
action/operation matrix including the refused `crash-after`, ambiguity and lock
paths), configures the volumes or recovers them from the image, and mounts them
on their empty writable directories. Any refusal is `invalid` (`capacity` for an
oversized image) before guest start, with zero modeled activity.

`crash_context` has the same `chunks`, `bytes`, `sha256` descriptor as `image`.
Its `storage_context_input` frames and `storage_context_inputs_end` follow the
namespace inputs and any storage image inputs. The decoded JSON maps configured
volume IDs to `PendingSHA256` (`sha256:` plus 64 lowercase hex digits) and
`Operations` (only the previously observed pending operations). The helper checks
chunk framing, lengths, digest, strict JSON fields, ordered positive operation IDs,
dependencies positive and smaller than their operation, and the model's pending
operation count limit before Store creation. Context bytes are bounded by 18 times
the image byte allowance. Nonempty selections require context; zero, duplicate or
unknown operation IDs and selections missing pending dependencies refuse before
guest start. Null and empty selections retain the default durable state.

Every `fd_*`/`path_*` operation on a volume node runs Go's storage decision:
`resource\0operation` occurrence counting, first-match faults (`io-error` and
`sync-error` errno 29, `no-space` 51, `short-write`/`torn-write` shorten the
write and torn also gives 29, a non-shortening match is
`invalid: storage write fault must shorten the matched write`), and one storage
event line rendered byte-identically to Go's `json.Marshal(StorageEvent)`; the
storage log is bounded by `transcript_bytes` (`capacity: storage-transcript`).
Volume byte exhaustion is errno 51; other volume capacities are
`capacity: storage-<resource>`, distinct from namespace `filesystem-bytes`.
SQLite lock-database artifacts stay namespace-only. Volume timestamps use the
model clock of the last storage decision. `crash-before` stops before the
syscall returns: the helper flushes evidence and reports the `storage_crash`
terminal with message `modeled storage crash: <fault>` and an interrupted,
unfinished choice prefix. Volume paths must be UTF-8 (`unsupported` otherwise).

At every terminal after attachment the helper performs Go's post-run storage
collection in order: transcript, snapshots and operations, each configured
volume's enumeration (if `crash_limits`; an error is the `invalid` terminal),
then the crash image for `crash_selection` (an error is `infrastructure`). It
sends evidence streams `storage` (the event lines), `storage_state` and, when
`return_image`, `storage_image` (the blob). `storage_state` envelope v1 is
`GMSTORE\x01`, u32 version 1, u32 section count 5, then tag (u32 1..5) and
u64-length-prefixed sections: snapshots JSON, operations JSON (object by volume
ID), enumerations JSON (object by volume ID, or `null` without limits), the
64-hex image identity, and the `storage` stream's u64 length plus SHA-256; a
u64 length and the SHA-256 of all preceding bytes end it. Go decodes it with the
strict JSON decoder into the existing volume types and refuses trailing data,
truncation, unknown versions, inconsistent volume sets, a storage stream that
does not match its binding, or an image whose SHA-256 is not the identity, as
`invalid`. An oversized image is `capacity: storage-image-bytes` on both owners.
Go admits a helper-originated `storage_crash` only with the message of a
configured `crash-before` fault.

With `crash_limits`, complete evidence uses version 3 and six sections; section 6
is the pending crash-context JSON map for every configured volume. Without limits,
complete evidence retains version 1. Go refuses complete evidence whose version
does not match the request (v1 with limits, or v3 without) as `invalid`. Before creating the selected crash image, the
helper compares every supplied context's hash and operations with the actual
pending state. A mismatch reports `invalid` with the exact message
`storage crash context diverged from prior observation` and partial version 2,
stage 2 evidence. It returns no image.

Collection failures retain the prefix Go collected before the error. They use
the same magic and framing with version 2 and six sections. Sections 1–5 have
the v1 types, but the image identity is empty; section 6 is exactly one byte:
1 for enumeration failure or 2 for image creation failure. Enumeration failure
requires an `invalid` terminal and an enumeration object containing exactly the
completed prefix in configured volume order. Image failure requires all requested
enumerations and either `infrastructure` or `capacity` with the exact message
`WASI capacity: storage-image-bytes`. Image failure also permits `invalid` only
for the exact crash-context divergence message above. All partial failures
require a nonempty terminal message
and forbid image bytes. Snapshots, operations and the bound storage log remain
mandatory and strict. A partial envelope cannot accompany a successful terminal;
ordinary v1 evidence still requires its image identity and any requested image.
Malformed, truncated, corrupt or trailing partial evidence is refused as `invalid`.

Recovery image v1 is `GMVOLIMG`, u32 version 1, u32 section count 3, then tagged
u64-length sections: the model identity, sorted fault occurrence counters, and
the complete durable state (next inode, generation, and per volume the mount,
capacity, persisted objects/paths/owned inodes, pending and history operations
with full effects, next operation and last-store keys); integers are u64
little-endian. Its identity is its SHA-256. The Go owner encodes the same bytes
(`storage_image.go`), so Go-owner and helper images of one state are identical.
A recovered run continues the image's occurrence counters; an image with another
model identity, lock-database paths, inconsistent state, a digest mismatch or
truncation is refused. Images are bounded by the filesystem byte limit plus
1 MiB (measured: Temporal crash/reopen images at most 7.5 MB with a 512 MiB
limit, SQLite at most 85 KB with 8 MiB).

## Broker split ownership

Broker storage runs (cooperative profile, including network v2, whose streams
are `runtime_net_stream` calls served by the broker's Go network owner) use split ownership:
`local_model.split` is true, the helper owns the namespace, volume, clock reads,
yields, polls, captured inputs and the WASI transcript, and only `runtime_*`
controls and guest writes to the still-open standard fd 1/2 descriptors are
callbacks, answered by the Go broker with
its runtime session and shared clock. Clock reads and polls stay local because
broker time changes only while the incarnation is quiescent in `runtime_idle`,
whose reply carries it in `local_state`; this keeps timestamps and clock-run
folding identical to the Go-owned broker path. Local calls consume global call
IDs; the broker admits remote IDs that advance by at least one.

Split callback `local_state` also carries `offered_output_bytes`, the namespace's
total before this remote call. Go requires it to be monotonic, at most the
configured output limit, and present only for the split-owned namespace before
adopting it. This reconciles local rejected fd 1/2 writes with earlier and later
remote writes. The helper shares one counter across both routes; closing a
standard descriptor changes routing but does not reset its output budget.
The split terminal `output_bytes` must cover the last total observed by Go and
stay within the configured limit, including on capacity outcomes. Ordinary
remote-owner terminals retain exact equality with Go's offered-output total.

Remote replies carry `"revocable":true` because the broker evaluates delivery
after sending them. Before its first local call and before the first local call
after a revocable reply, the helper sends `{"type":"sync","id":<last completed
call>}` and waits for `{"type":"continue","now":<broker-raised clock>}` or the
typed `crash`/`stop` interrupt of a revocation that crossed the reply, so its
durable effects at a broker crash equal the Go-owned path's. When a revocation
lands after Go completed a remote call (an output boundary, or a concurrent
crash/stop between the handled call and its broker completion), the interrupt
carries `completed` with the resulting `now` and `advances` and, for a WASI
call, its canonical record as `transcript_base64` (records over 64 KiB arrive
first as `remote_transcript` chunks with an empty inline value); the helper
adopts the state, appends the record and counts the call, as the Go owner does.
A record without `completed` is invalid. A crash or stop revocation from outside
the incarnation is delivered in band at the helper's next broker call, so it
flushes its storage evidence and image; the provider cancels the helper only
after a bounded grace, and such a cancellation is the `infrastructure` terminal
(no recovery image), never a modeled crash. Broker crash, stop and
storage interrupts flush storage evidence as above, and split runs always
return the recovery image because the broker may restart any incarnation.

## Conformance driver

The non-default Cargo feature `conformance` replaces the engine entry point with
a test-only model driver. From `tools/gomad`, `make validate-model-corpora`
builds it through the single Rust workspace and copies it to
`rust/target/release/gomad-model-conformance` (or the configured
`CARGO_TARGET_DIR`) before restoring the ordinary reference transport. It is
never installed or used as a qualification producer.
Default executables omit the driver. Plain `cargo test`
includes their unit tests and replays the checked-in namespace corpus.

Each UTF-8 JSON object ends in LF and is bounded to 8 MiB including LF. The first
record is `{"model":"namespace","image":{...}}`; unknown models are refused.
The image contains `entries` with `path_base64`, `directory`, `readonly` and
`data_base64`; `writable_directories` and `preopens` arrays of base64 names;
`working_directory` and `stdin_base64`; `epoch_nanos`, `read_step_nanos`; and
`limits` with `files`, `descriptors`, `filesystem_bytes`, `output_bytes`, `calls`,
`transcript_bytes` and `pending_events`. Entries retain inode creation order,
exclude the implicit root and include parents before children. Empty file/stdin
bytes may be `null`. Initialization returns `{}` or
`{"boundary":"invalid:dimension"}` / `{"boundary":"capacity:dimension"}`.

Subsequent records are `{"op":"fd_read","input":{"fd":0,"length":8}}`, using
the callback input shapes above without `type` or `id`. Replies retain Go's
field order: `{"errno":0,"output":{...}}`, optionally followed by `unsupported`
and `message`; boundary replies contain only `boundary`. Unsupported and
boundary replies leave the session available for a digest or further probes.
Admission reserves the full Go callback/reply envelopes and transcript capacity
before modeled mutations. Clock reads and mixed clock/fd polls use the image's
modeled epoch/read step; a zero read step starts at monotonic position 1.
`{"op":"digest","input":{}}` returns `{"digest":"lowercase SHA-256"}` without
consuming a callback or transcript reservation.

Digest v1 hashes the prefix `gomad.namespace-conformance/v1` plus NUL, followed
by `N` records for a depth-first tree walk from `/` with byte-sorted child names,
`O` records for descriptor-reachable detached nodes and descendants sorted by
inode, `D` records sorted by fd, and one `S` record. Integers are little-endian
u64; booleans are u8; byte strings have a u64 length prefix. `N` contains path,
then inode/directory/readonly/size/data/modified. `O` contains the same node
fields without path, then child count and sorted child name/inode pairs. `D`
contains fd/node inode/offset/rights/inheriting/flags/preopen/std tag.
`S` contains next fd/next inode/files/bytes/offered output/stdout/stderr.
This digest is a harness assertion, not runtime evidence.

`make test-conformance` builds the isolated helper unless
`GOMAD_WASM_CONFORMANCE_HELPER` explicitly selects an absolute executable; a
missing supplied helper fails. It compares raw replies and digests for the
scripted/recorded corpus and 100 seeded scripts of 128 operations over a depth-4
path alphabet; every fourth seed lowers the file, descriptor or filesystem/output
byte limit so generated runs reach those capacities. The driver admits, records
and answers typed filesystem operations, clock reads and polls through the same
Rust code as the helper-owned model (`namespace_call`, `transcript`,
`local_model`); only Go's JSON-surface checks are driver-local. `test-wasm`
runs this differential once and includes it as a drift guard. Corpus updates require the Go test's explicit
`-update-conformance` flag.

The first record `{"model":"internal/model/volume","image":{...},"storage":{...}}`
selects the volume session: the same namespace image plus Go's `StorageModel`.
Its records are `{"op":...,"input":...,"now":N}`; `now` sets the model clock
before the operation. Besides filesystem operations, controls are `enumerate`
(`volume`, `limits`, optional saved `cursor`, `save`), `select` (`seed`, `save`),
`crash` (`selections`, a saved `selection`, or a saved `page` and `state`, and
the `image` name) and `recover` (`image`; a fresh environment from the saved
image). Each reply is one Go `volumeConformanceObservation`: reply bytes, the
appended storage event lines, snapshots, operations, enumeration, selections,
image digest and bytes, and the namespace digest. `TestVolumeConformance`
compares it with the Go model over the checked-in corpus
(`wasi/testdata/volume-conformance`); plain `cargo test` replays the corpus too.

## Cluster execution

The simulation provider selects this path by default. The per-process Go broker
remains a differential reference. One helper serves either `execute` or
`cluster_execute`, never both. Diagnostic-profile builds refuse clusters; matched
release measurements must use release helpers for both baseline and candidate.

**Execute.** The first frame may instead be:

```json
{"schema":"gomad.wasm-host/v1","type":"cluster_execute","module_path":"/absolute/module.wasm","module_sha256":"...","compile_cache":"/absolute/cache","instances":[{"id":"node-a","memory_bytes":268435456,"fuel":100000000,"output_bytes":8388608,"local_model":{...}}]}
```

The module identity, compile cache and module bounds are shared and validated
once, exactly as for `execute`. `instances` holds 1–64 entries with unique ids of
1–128 bytes from `[A-Za-z0-9._-]`, in any order; the helper orders instances by
byte-wise id, like the Go broker's sorted node order. Each entry carries the
single-execution `memory_bytes`, `fuel`, `output_bytes` and optional
`local_model`, validated with the same rules. An optional `broker` object
(fn-178.7) carries Go `BrokerConfig`'s World fields: `{"seed":7,"limits":
{"max_requests":…,"max_events":…,"max_queued_events":…,"max_transitions":…,
"max_payload_bytes":…,"max_string_bytes":…},"pause_on_settle":false}`, refused
as `invalid` unless it fits the compressed World profile, as `NewBroker`
requires. A cluster `network` requires a `broker`, and every instance under a
`broker` requires a `local_model` (`broker node requires cooperative runtime`),
since the broker serves `runtime_idle` on the cooperative runtime. Manual sessions
also admit `namespace`, `storage`, and optional module path/digest overrides per
boot. `boots` seals additional configurations for later incarnations. Engines
and prepared modules are shared; every incarnation gets a fresh store. Initial
recovery images are refused: restart images belong to the Rust lifecycle.
Unknown fields, duplicate ids, an empty or oversized list and any
invalid instance limit end the run with one `cluster_result` of termination
`invalid` before any compilation output; there is no partial cluster.

**Network (fn-178.12).** An optional cluster `network` object carries the Go
broker's network configuration, and then every instance must carry its own
`network_v2` profile and a `local_model`:

```json
"network":{"model":{"profile":"gomad.simulation-network-stream/v3","limits":{"listeners":128,"connections":128,"bytes":8388608,"connection_bytes":4194304,"stall_nanos":10000000000,"transitions":8192}},
  "addresses":{"node-a":"10.0.0.9"},"links":[{"from":"node-a","to":"node-b","enabled":false,"delay_nanos":0}],
  "strict_topology":true,"faults":[{"name":"cut","action":"partition","from":"node-a","to":"node-b","at_nanos":5,"delay_nanos":0,"symmetric":true}]}
```

`model` and each instance's `network_v2` (`profile`, `limits`, optional
`faults`) follow Go's `NetworkModelV2` rules. The rest mirrors
`newBrokerNetwork`: a node-local fault schedule (in `model` or an instance),
a non-canonical IPv4 address, a self, unknown or duplicate link endpoint, an
unknown, unordered or duplicate central fault, or an unknown fault action is
`invalid` before `started`, with Go's `WASI invalid: <dimension>` message. A
cluster also refuses up front an instance without a stream profile or
cooperative runtime, and an instance profile without a cluster network (the Go
broker refuses the first two at the node's first call). One Rust network owner
serves every instance: nodes `<id>.server` (the configured address, else
`10.0.x.y`) and `<id>.client` (`10.1.x.y`), links in sorted node order under the
strict-topology rule, both domains registered per incarnation, request IDs
based at `slot << 48` (slot = instance order from 1), and graceful revocation
of a finished incarnation's domains after its evidence. Every broker-visible
call attaches its endpoint and advances the shared owner to broker time, as Go's
`brokerNetwork.attach` does; the broker advances it again (applying due central
faults at their own times) before resuming a round of idlers, and folds the
owner's next deadline, the next fault and a non-empty ready set into each idle.
Faults and live link changes apply to all four client/server pairs of the two
nodes. An instance's
`gomad_wasm_v2::net_stream` is served in the helper with Go's buffer layout,
validation, errno table and per-instance network World; it never calls back to
Go. Single executions still forward `runtime_net_stream` to the Go owner.

Each network instance streams its binary `network` evidence
(`{"instance":"<id>","type":"evidence","stream":"network","data_base64":"..."}`,
64 KiB chunks) at every terminal, after its model evidence and before its
result. The `GMNETEV1` layout is documented in `src/cluster/network/evidence.rs`:
the operations log (request/result and World-quiescence entries, with the
helper's Go-JSON byte count of the log), the shared owner's Record at that
moment and the instance's expanded `GMWSNAP1` network World. The log is bounded
per append by the instance's `transcript_bytes` as Go's
`network-v2-transcript-bytes`; a stream longer than `transcript_bytes` replaces
the outcome with `network-v2-evidence-bytes` capacity and is not sent. The
binary stream is never longer than the JSON document it encodes, and Go
re-applies the exact JSON bound when it rebuilds the document (fn-178.13).

**Model inputs.** Right after `cluster_execute`, before compilation, each
instance with a `local_model` receives its single-execution model-input frames
(`model_input` chunks, then `model_inputs_end`), instance-tagged and in
instance order. Any failure, including an unattributable frame, ends the run
with one pre-start `cluster_result`. With `manual_start:true`, each initial
instance and then each additional boot receives its model inputs followed by
namespace frames, tagged by its boot id, before compilation. Namespace and
storage validation finish before authorization. Sealing configurations does
not execute guests.

**Admission.** The helper compiles or cache-loads the module once, sends one
`started` frame (the single-execution shape; `initial_memory_pages` is checked
per instance at instantiation, not against the cluster), and reads one
`authorize` frame with the single-execution identity check. Manual sessions first
send `boot_started` inventories (`boot` names the sealed configuration) for Go
to validate every module. Without manual start, each cooperative
instance's tape follows as `tape_chunks` instance-tagged `tape` frames, in
instance order. The helper reads each instance's declared frames in full before
validating them, so a correctly tagged but invalid tape becomes that instance's
result at its turn (it never runs) and the next instance's frames stay in step. One shared `Engine`, `Module` and `InstancePre` serve every
store.

**Instance tagging.** Every helper-to-Go frame produced by an instance is the
exact single-execution frame with the prefix `{"instance":"<id>",` in place of
its opening `{`: `call`, `evidence` and the per-instance `result`. Every
Go-to-helper frame addressed to an instance (`reply`, `remote_transcript`,
`interrupt`, `tape`) must begin with the prefix of the instance that is waiting.
EOF, a missing prefix or a prefix naming any other instance cannot be
attributed and ends the whole run as `invalid`; a correctly tagged frame that
fails single-execution validation is that instance's `invalid` result. Stripping the prefix yields
byte-for-byte the frames that instance's single execution produces, so existing
Go consumers decode per-instance streams unchanged, with one stated exception:
legacy automatic per-instance `result` frames omit `metrics`. Manual results
carry instantiation/execution metrics; process CPU/RSS belong only to the final
cluster metrics. Incarnation 1 keeps the original prefix; later frames begin
`{"instance":"<id>","incarnation":N,`. The expected incarnation must match,
so stale replies and tapes are refused. A tagged frame may exceed the
8 MiB frame limit by at most its prefix. An instance whose initial memory
exceeds its own `memory_bytes` fails at instantiation with that instance's
`capacity` result; module admission uses the largest instance limit.

**Scheduling and threading.** Exactly one instance runs at a time. Frame order
across instances follows the baton: an automatic cluster streams an instance's
`result` before the baton passes on, and a manual session marks the incarnation
finished first, because Go answers that result with its next control. Each store
lives on its own OS thread (8 MiB host stack) and runs only while it holds the
cluster baton; synchronous typed imports are unchanged. Without a `broker`
the baton runs each instance to its terminal in instance order. With one, the
in-process broker (`src/cluster/handoff/broker.rs`) is the Go broker's policy: the active
instance hands off only at `runtime_idle`; the next is the first runnable
instance in id order; an idle asks the Rust clock to quiesce with
`{generation, current, deadline, arrivals: 0}`; when every live instance is idle
and answered, the network advances to the current time and they all become
runnable. `runtime_idle` and `runtime_net_stream` are World-visible: each
registers a World request (`wasm-broker`/`incarnation`/`<id>/<inc>`, payload the
SHA-256 of the callback input JSON), becomes ready at the clock's current time
and must quiesce to exactly that delivery (`broker delivery order`). An idle
resumes the instance's local clock at the broker's time (`broker clock moved
backwards` if earlier), settles its network World and reports status 1 when
the instance has a network. A clock deadlock answer is `WASM broker global
deadlock`, and a broker that no instance can advance is a whole-run
`infrastructure` stall. The Go broker also books every helper callback with the
clock's progress credits; serialized local calls cannot overlap an idle, so
the in-process broker settles exactly when every live participant is idle and
sends no arrivals.

**Automatic-mode live link control.** With `"pause_on_settle":true`, whenever the
clock advances (every instance idle, due faults applied) the helper sends
`{"at":<virtual nanos>,"reason":"settle","type":"pause"}` and reads
`{"type":"control","command":"link","action":"partition|heal|delay","from":"<id>","to":"<id>","delay_nanos":0,"symmetric":true}`
frames until `{"type":"resume"}`; any other frame or an invalid link ends the
run as `invalid`. A failure the broker raises itself (a refused control frame,
a stall, a World or clock refusal) is the run's classification and message;
the instances it aborts report only the abort and never override it. Go's `ChangeLink` applies at whatever moment the scenario
calls it; the helper applies it at the next clock advance, deterministically.

**Manual lifecycle.** After authorization, one reader accepts monotonically
numbered controls. `{"type":"start","id":1,"node":"a","incarnation":1,"boot":"a"}`
registers a sealed boot; `restart` requires the preceding incarnation finished and
registers its successor. Manual scheduling starts controller-owned and paused,
so registration alone cannot advance a guest. Optional `runtime:{"mode":2,"tape_chunks":N}` selects
replay; tagged tape frames follow the control. The acknowledgement is
`{"type":"control_result","id":1,"termination":"","message":""}`; refused
controls retain their classification and message. Restart adds the node count
to its slot, refusing slots >=2^16 as capacity. The recovery image stays in Rust
even when `return_image:false`.

`{"type":"run","id":2,"node":"a","incarnation":1}` grants a driving turn
until that incarnation finishes. Its acknowledgement precedes streamed results;
the helper reacquires the controller pause after the target's terminal evidence
and before scheduling another guest. A matching already-terminal target is an
idempotent acknowledgement, while stale or never-started targets and overlapping
live driver targets are invalid. The Go client serializes driving `Wait` calls;
passive observation sends no `run`. Cancellation releases the caller but keeps
its driver gate until the target terminates or the helper is reaped. Live
revocation and link controls bypass that gate. Automatic mode has no such pause.

`output_crashes` and `output_stops` contain `{node,incarnation,marker_base64}`;
one optional `output_crash_group:{nodes,marker_base64}` covers at least two
initial incarnations. Markers are 1–4096 decoded bytes. They match cumulative
stdout, including markers split across writes, with individual crash, completed
group, then stop precedence. Storage crashes retain the existing crash-before
semantics. Revocation cancels pending World work, revokes network endpoints
(stop gracefully, crash abruptly), and prevents the next local or network
effect. Finished releases the restart hold so survivors can progress without
a restart. A storage crash follows Go's graceful Finished network cleanup.

Unbound `control` commands `crash`, `stop` (exact node/incarnation), and `link`
(`action`, `from`, `to`, `delay_nanos`, `symmetric`) are queued and applied at
host-call or scheduler handoffs. They remain non-replayable. An optional boot
`stdout_progress:true` emits tagged `progress` stdout observations for host
observers, without replies; final stdout evidence remains unchanged. `finish`
closes the session after terminal results, removes never-started participants,
and emits final shared evidence. EOF before finish refuses the run.

Each incarnation's network evidence precedes its Finished cleanup. After every
incarnation finishes, untagged `cluster_network` chunks (`source` boot id,
`data_base64`) carry the final shared Record in a `GMNETEV1` envelope with empty
operations and a fresh empty network World. Go decodes it strictly and uses its
Record; it does not replace any incarnation's network document. World chunks
then carry the expanded shared snapshot. The final `cluster_result` contains
every completed id/incarnation, `virtual_elapsed_nanos`, process metrics, and
bounded `scheduler` counters outside semantic evidence. Per-store memory peaks
remain in each result; process CPU and RSS cover all resident stores and boot
templates once. Frame and callback limits apply per incarnation; World and
control frames belong to the cluster.

**Terminal taxonomy.** Per instance (streamed as that instance's `result`):
`exit`; `trap`; `capacity` (fuel, linear memory, tables, output, callbacks,
transcript, network evidence); `unsupported`; `invalid` raised by that
instance's ABI, model or reply validation; `divergence`; and the modeled
`storage_crash`, `crash` and `stop`. A failing instance's store is dropped and
never reused; the baton passes to the next instance, whose evidence is
unaffected. Whole run (one `cluster_result`, no further instance results):
unattributable Go frames (`invalid`), stdout write failure
(helper exits 1), any engine or host error classified `infrastructure` in any
store, and every Rust panic. A panic is always whole-run `infrastructure`,
because shared transport, scheduler, World or clock state may be torn.

**Diagnostic builds.** Builds with `diagnostic-profile` or `guest-profile`
refuse `cluster_execute` as `unsupported`: their attribution state is
per-thread and cannot see store threads.

**Watchdog.** The Go process watchdog bounds the whole cluster run, including
time blocked at pause points. The default build adds no epoch ticker: epoch
interruption changes generated code and the configuration revision for every
run, and a stuck serialized guest blocks the cluster either way. A watchdog
kill is a whole-run `infrastructure` outcome; results already streamed stand.

**Memory.** Each store has its own `ResourceLimiter` enforcing that instance's
`memory_bytes`; growth beyond it is that instance's `capacity`, and the
per-instance `peak_memory_bytes` is the limiter-recorded peak. Process CPU and
peak RSS are sampled once, in `cluster_result.metrics`, and replace
per-incarnation helper RSS in qualification budgets.
The qualification limit is 8 GiB for the shared process, including retained boot
templates; the single-helper 4 GiB warm and 6 GiB cold limits do not apply to a
cluster. Compile-cache hits distinguish warm and cold observations without
changing that resident limit. Missing process-image peaks cannot pass the
cluster budget using a wait4 upper bound instead.

**Evidence and results.** Per-instance evidence streams (`choice`,
`diagnostic`, `transcript`, and `network` from fn-178.12) flush before that
instance's result, so a later whole-run failure never loses a finished
instance. With a broker, World evidence follows the last instance as cluster-scope
`{"type":"evidence","stream":"world","data_base64":"..."}` frames of at most
64 KiB decoded bytes, carrying the expanded `GMWSNAP1` wire bounded by the
profile's 384 MiB expanded limit while it is written (over it, the run ends as
`capacity` `compressed-snapshot-expanded-bytes`, as in Go); Go wraps it in the `GMWORLD2` envelope and
validates it with `world.Restore`. The final frame is:

```json
{"type":"cluster_result","termination":"exit","message":"","instances":[{"id":"node-a","termination":"exit"}],"metrics":{"prepare_nanos":1,"compile_nanos":0,"cache_load_nanos":0,"cache_write_nanos":0,"cache_hit":false,"instantiate_nanos":2,"execute_nanos":3,"process_user_nanos":4,"process_system_nanos":5,"peak_rss_bytes":6,"resource_usage_available":true}}
```

`termination` is `exit` when every instance produced a result, otherwise the
whole-run classification. `instances` lists every instance result, in result
order, with an `incarnation` field on lifecycle results and restarted instances'
frames. `instantiate_nanos` and `execute_nanos` sum the
instances.

## Separate diagnostic builds

The non-default Cargo feature `diagnostic-profile` compiles bounded attribution
hooks. Normal builds contain no profiler module, clock sampling hooks, counters
or diagnostic reporter. A diagnostic build must use a separate target directory
and immutable helper path; never install it over a qualification helper. Source
metadata can change a rebuilt default binary's digest even with hooks excluded;
hash each actual artifact rather than assuming historical byte identity.

`guest-profile` implies `diagnostic-profile` and enables Wasmtime's pinned
`profiling` and `call-hook` features only in that diagnostic build. Build it with
`cargo +1.99.1 build --release --locked --features guest-profile --target-dir /tmp/gomad-wasmhost-guestprof`.
It adds the optional execute field
`"guest_profile":{"path":"/absolute/profile.json","interval_micros":1000,"max_samples":10000}`.
The interval must be 1–1,000,000 microseconds and the sample bound 1–100,000.
Default helpers reject this unknown field before loading the module. The
profile file is created exclusively, with mode 0600 on Unix; existing files
are preserved. Wasmtime's GuestProfiler writes Firefox processed-profile JSON,
with `meta.gomadGuestProfile` recording `intervalMicros`, `maxSamples`, `samples`,
`truncated` and `markersTruncated`. Reaching the sample limit stops the ticker;
host markers are independently bounded to twice the sample limit. JSON remains
complete when either bound is reached. Profile output failure emits one
`gomad-wasm-guest-profile warning:` stderr line after the terminal frame and
does not change its outcome or the helper exit status. The phase report retains
its separate 16 KiB contract below.

The sampler uses epoch interruption and Wasmtime guest-stack symbolization;
call hooks identify host work. Returning from a host call resets the epoch
deadline and sample time, discarding ticks and elapsed time inside that call.
Guest clocks and evidence contain no sample inputs. Profiled timings are not
baseline or A/B measurements.

`diagnostic-fuel-off` also implies `diagnostic-profile`; build it with
`--features diagnostic-fuel-off --target-dir /tmp/gomad-wasmhost-nofuel`.
It disables engine fuel instrumentation and reports the offered fuel unchanged
in `fuel_remaining`. The configuration revision gains `-guest-profile`,
`-fuel-off`, or `-guest-profile-fuel-off` when both features are enabled. These
revisions bind `started`, authorization and the compile-cache identity. Go
accepts them only through a private test request opt-in; normal requests and
release installations keep the pinned no-unwind configuration.

A diagnostic helper writes exactly one LF-terminated stderr line prefixed
`gomad-wasm-profile-v1 `, followed by JSON schema `gomad.wasm-profile/v1`. The
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

### World transition byte bounds

New host preparations retain `configuration.world_transition_bytes`, derived from
`--world-transition-limit`. Cluster declarations retain the same value as
`broker.transition_bytes`; standalone network owners receive it during admission.
Each World independently budgets the sum of its accepted canonical native
transition encodings. The bound excludes snapshot headers, request/event tables,
compression, inspection metadata and the number of transitions. The original Go
recorder counted its JSON lines; this Rust lineage uses its retained binary
transition encoding and does not claim byte equivalence with that lineage.

Register, readiness, cancellation and quiescence all check the prospective
transition before changing state or history. Rejection reports `capacity` with
`transition-bytes` and its limit, used bytes and attempted delta, retaining the
accepted prefix. Retained World validation replays that prefix under the same
bound. Re-execution uses the retained bound, including the qualification values
of 1 MiB and 32 MiB.

Both new configuration fields are optional and omitted when absent. Earlier Rust
artifacts retain their original configuration bytes and hashes and their previous
World count/size bounds; replay does not insert a new default into them. New
production preparations record the explicitly selected byte bound.
