# Isolated WASM engine protocol

Build and test with `cargo +1.94.1 build --release --locked` and `cargo +1.94.1 test --locked`
in this directory. Rust 1.94.1 and Wasmtime 47.0.3 are pinned locally. No WASI
implementation or engine cache is linked. A helper handles one execution and exits.

Every frame is one UTF-8 JSON object followed by LF. Stdout carries protocol only.
Integers are JSON integers, not floating-point strings. Byte strings use standard
padded base64. Go owns environment, entropy, clocks, descriptors, files and models;
Rust owns ABI decoding, bounds, memory writes and execution. Guest pointers never
cross the protocol. Every import is an explicit callback, except `proc_exit`.
No host arguments, environment, clocks, files, random source or sockets are inherited.

The first input is:

```json
{"schema":"gomad3.wasm-host/v1","type":"execute","module_path":"/absolute/module.wasm","module_sha256":"64 lower-case hex digits","memory_bytes":268435456,"fuel":100000000,"output_bytes":8388608}
```

All fields are required; extra fields are rejected. Memory is positive, at most
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
`finish` and `idle`, each with signature `(i32 pointer, i32 length) -> i32 errno`.
Go requires all five imports before authorizing a cooperative profile. The guest
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

The final frame is `{"type":"result","termination":"exit","exit_code":0,"message":"","fuel_remaining":123,"peak_memory_bytes":65536,"output_bytes":2}`.
Terminations are `exit`, `capacity`, `unsupported`, `trap`, `invalid`, and
`infrastructure`. Exit code is present only for `exit`. Invalid input, unsupported
imports, initial/grown memory limits, fuel, buffer, callback and output limits have
separate explicit results. Host resource failures remain infrastructure outcomes;
limits do not assert memory backing availability or a host peak-RSS bound.
`peak_memory_bytes` is the maximum limiter-approved requested guest linear-memory
size. It may exceed committed memory after allocation failure; it supplies no
host RSS or backing-availability guarantee.
The helper exits 0 when it successfully emits a result; transport-write failure exits 1.

Result diagnostics are bounded to 64 KiB, with an explicit truncation marker.
Every frame is at most 8 MiB including LF. Each transferred/ABI buffer is at most
4 MiB, iovecs at most 4096, poll subscriptions at most 4096, and executions at most
1,000,000 callbacks. Guest stdout/stderr bytes (fd 1/2 writes) consume `output_bytes`
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
