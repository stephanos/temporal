# Model ABI v1

The generated protobuf messages in `umpire/model/v1/model.proto` are the wire contract.
Method enum numbers are stable dispatch IDs. `umpire_call` always returns a `Response` envelope.
Explore replaces the current exploration only on success; Rows needs a successful Explore.
Each rows string is one complete native golden line, including its newline. States and actions
use Display; domain outcomes and facts use Debug.

Reset (3), Judge (4) and Answer (5) carry the core run protocol. Judge accepts an opaque
observation Any and returns either a present Explanation (including zero steps) or
Unexplained with expected actions. Answer accepts an opaque callback Any and returns an
opaque reply Any. `umpire.realization.v1` supplies Observation, Error, Callback and Reply envelopes.
Observation retains optional request Any at tag 4 and unchanged absent-request bytes.
System payload registries own type-URL admission; the old namespace
is not an alias for the new one. The Kettle guest currently refuses these three operations
with Unsupported (8). The export-free `umpire-runtime` library supplies the transactional
runtime and ABI; system guests supply a `Machine` and `Realize` realization. Reset restores
the unique start and clears frozen evidence. Judge replays actual Model steps; the
Explanation lists Debug-formatted outcomes in parallel with its steps, including `()`
for unit outcomes. The realization matches the final replayed outcome; a zero-step current
check has no outcome. The first Unexplained freezes advancement until Reset. Answer validates
delivery and its enabled response action before committing. Failed response publication commits nothing.
`Any::pack`/`unpack` use generated `prost::Name` canonical URLs and typed Refusal
constructors; the registry remains system-owned. Existing method IDs and old wire bytes
remain pinned by host tests.

Annotate (7) carries an opaque input Any and returns an opaque output Any at
Response tag 8. Shared `umpire.annotations.v1` messages carry annotation requests,
results and probe metadata; system probe controls remain system-owned.
`AnnotationRuntime` dispatches this stateless operation independently of Judge's frozen run. Method 6 and Response tag 7 remain
reserved for Propose; every existing method and field number is unchanged.

The wasm32-wasip1 guest directly exports application functions
`umpire_buffer_alloc(u32) -> u32`, `umpire_buffer_release(u32, u32)` and
`umpire_call(u32, u32, u32) -> u64`, plus memory and toolchain lifecycle exports.
The packed result is `(length << 32) | pointer`. `umpire_buffer_alloc(0)` returns zero;
`umpire_buffer_release(0, 0)` succeeds. Other allocations are nonzero, initialized,
guest-owned buffers.
The guest bounds individual buffers to 1 MiB and total live storage to 16 MiB.
Allocation exhaustion returns zero; invalid frees trap without dereferencing the supplied pointer.
`umpire_call` accepts only an exact live allocation range, with `(0, 0)` representing an empty request.
It retains the request and returns a separately owned response allocation, or zero on exhaustion.
Unknown methods, malformed requests and invalid checker queries return typed Refusal envelopes.
`REFUSAL_CODE_SCOPE_ACTION` (7) covers invalid action admission, including unknown catalog
actions and explicit restrictions that admit no actions. The message distinguishes these refusals.
`REFUSAL_CODE_MALFORMED_REQUEST` (2) also covers invalid progress bounds and native registered
progress query misuse. Diagnostics distinguish a zero bound, an unknown or ambiguous retained
registration, and an action catalog whose ordered values differ from exploration. These are
invalid checker requests, distinct from action-admission refusals (7) and state budgets (3).
This mapping adds no wire method; registered progress queries remain a native checker operation.
The host copies response bytes before freeing either allocation, then frees both exactly once.
Access to one instance is serialized. An unexpected trap invalidates the host instance.

Rust generates into Cargo's OUT_DIR with prost-build 0.14 and mise's protoc 29.5.
`umpire_runtime::google::protobuf` owns the generated standard well-known types; realizations
reuse this owner through prost-build's `extern_path` with default WKT mapping disabled.
This owner compiles the standard schema inputs independently of any system descriptors.
Upstream WKT comments contain examples in other languages, so their generated Rust
comments are suppressed rather than compiled as Rust documentation examples.
Go uses the repository's pinned protoc-gen-go v1.36.6 and protobuf runtime v1.36.11.
The separate Go host module and generated Go messages live at `../hosts/go/`;
regenerate from the repository root with `bash model/scripts/proto-go.sh`.
Use `--check` for byte-for-byte freshness; `make model-check` provisions the pinned
generator and runs freshness, native controls, both guests and both Go modules.
The system extension and its reproduction commands are documented in `model/README.md`
relative to the server root; its upstream schemas never become generic protocol inputs.

The sibling `umpire.realization.v1` package owns Observation, Error, Callback and
Reply. Observation carries its RPC method, one response or error, and an optional
actual-request Any at tag 4; absent requests retain their original encoding.
Callback carries a task Any and Reply a request Any. `umpire.annotations.v1`
owns stateless annotation and generic probe metadata, results and recipe transport.
System-specific controls remain in their owner's protocol. Regenerate these Go
messages together with the core contract using `bash model/scripts/proto-go.sh`.
